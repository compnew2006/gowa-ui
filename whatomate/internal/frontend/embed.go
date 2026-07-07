package frontend

import (
	"crypto/rand"
	"embed"
	"encoding/base64"
	"fmt"
	"html"
	"io/fs"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/compnew2006/whatomate/internal/middleware"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

// mimeTypes maps file extensions to MIME types
var mimeTypes = map[string]string{
	".js":    "application/javascript",
	".mjs":   "application/javascript",
	".css":   "text/css",
	".html":  "text/html",
	".json":  "application/json",
	".png":   "image/png",
	".jpg":   "image/jpeg",
	".jpeg":  "image/jpeg",
	".gif":   "image/gif",
	".svg":   "image/svg+xml",
	".ico":   "image/x-icon",
	".woff":  "font/woff",
	".woff2": "font/woff2",
	".ttf":   "font/ttf",
	".eot":   "application/vnd.ms-fontobject",
}

//go:embed all:dist
var distFS embed.FS

// cachedIndexHTMLTemplate stores the modified index.html with injected base path and CSP nonce placeholder.
var cachedIndexHTMLTemplate []byte

const basePathBootstrapScriptName = "__whatomate_base_path__.js"
const cspNoncePlaceholder = "__CSP_NONCE__"

// Handler returns a fasthttp handler that serves the embedded frontend files
// basePath should be empty string for root deployment or "/subpath" for subdirectory
// If frontend is not embedded, returns a handler that shows a helpful message
func Handler(basePath string) fasthttp.RequestHandler {
	// Normalize base path
	basePath = strings.TrimSuffix(basePath, "/")

	// Get the dist subdirectory
	distSubFS, err := fs.Sub(distFS, "dist")
	if err != nil {
		return notEmbeddedHandler("Frontend not embedded: " + err.Error())
	}

	// Read and modify index.html to inject base path
	indexContent, err := fs.ReadFile(distSubFS, "index.html")
	if err != nil {
		return notEmbeddedHandler("Frontend not embedded: index.html not found. Run 'make build-prod' to embed frontend.")
	}

	// Inject base tag right after <head> so it's processed before any relative URLs
	// Base tag ensures relative URLs (./assets/...) resolve from basePath, not current page path
	baseHref := basePath + "/"
	if basePath == "" {
		baseHref = "/"
	}
	baseTag := fmt.Sprintf(`<head><base href="%s">`, html.EscapeString(baseHref))
	modifiedHTML := strings.Replace(string(indexContent), "<head>", baseTag, 1)

	// Add a nonce placeholder to the inline theme-init script so CSP can allow it.
	modifiedHTML = strings.Replace(modifiedHTML, "<script>", fmt.Sprintf(`<script nonce="%s">`, cspNoncePlaceholder), 1)

	// Inject external base path bootstrap script to avoid inline script CSP violations.
	basePathScriptTag := fmt.Sprintf(`<script src="./%s"></script></head>`, basePathBootstrapScriptName)
	cachedIndexHTMLTemplate = []byte(strings.Replace(modifiedHTML, "</head>", basePathScriptTag, 1))

	// Create file server
	fileServer := http.FileServer(http.FS(distSubFS))

	// Wrap with SPA fallback and proper MIME types
	spaHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Serve base path bootstrap script.
		if path == "/"+basePathBootstrapScriptName {
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			_, _ = w.Write([]byte("window.__BASE_PATH__ = " + strconv.Quote(basePath) + ";"))
			return
		}

		// Try to serve the file
		if path != "/" && !strings.HasPrefix(path, "/api") {
			// Check if file exists
			filePath := strings.TrimPrefix(path, "/")
			file, err := distSubFS.Open(filePath)
			if err == nil {
				defer func() { _ = file.Close() }()

				// Get file info for size
				stat, err := file.Stat()
				if err != nil {
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}

				// Skip directories
				if stat.IsDir() {
					fileServer.ServeHTTP(w, r)
					return
				}

				// Set correct Content-Type based on file extension
				ext := strings.ToLower(filepath.Ext(filePath))
				if mimeType, ok := mimeTypes[ext]; ok {
					w.Header().Set("Content-Type", mimeType)
				} else {
					w.Header().Set("Content-Type", "application/octet-stream")
				}

				// Check Accept-Encoding and serve pre-compressed if available
				acceptEncoding := r.Header.Get("Accept-Encoding")
				var content []byte
				var contentEncoding string

				// Try Brotli first (better compression)
				if strings.Contains(acceptEncoding, "br") {
					if brContent, err := fs.ReadFile(distSubFS, filePath+".br"); err == nil {
						content = brContent
						contentEncoding = "br"
					}
				}

				// Fall back to gzip
				if content == nil && strings.Contains(acceptEncoding, "gzip") {
					if gzContent, err := fs.ReadFile(distSubFS, filePath+".gz"); err == nil {
						content = gzContent
						contentEncoding = "gzip"
					}
				}

				// Fall back to uncompressed
				if content == nil {
					content, err = fs.ReadFile(distSubFS, filePath)
					if err != nil {
						http.Error(w, "Internal Server Error", http.StatusInternalServerError)
						return
					}
				}

				if contentEncoding != "" {
					w.Header().Set("Content-Encoding", contentEncoding)
				}
				w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
				// #nosec G705 -- content originates from trusted embedded static assets.
				_, _ = w.Write(content)
				return
			}
		}

		// For root or non-existent files (SPA routes), serve modified index.html
		if path == "/" || (!strings.HasPrefix(path, "/api") && !strings.Contains(path, ".")) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			nonce, _ := generateCSPNonce()
			w.Header().Del("Content-Security-Policy")
			w.Header().Set("Content-Security-Policy", middleware.ContentSecurityPolicyWithNonce(nonce))
			html := strings.ReplaceAll(string(cachedIndexHTMLTemplate), cspNoncePlaceholder, nonce)
			_, _ = w.Write([]byte(html))
			return
		}

		// Serve the actual file
		fileServer.ServeHTTP(w, r)
	})

	// Convert to fasthttp handler
	return fasthttpadaptor.NewFastHTTPHandler(spaHandler)
}

// IsEmbedded returns true if the frontend dist folder is embedded
func IsEmbedded() bool {
	entries, err := distFS.ReadDir("dist")
	if err != nil {
		return false
	}
	return len(entries) > 0
}

func generateCSPNonce() (string, error) {
	const nonceSize = 16
	buf := make([]byte, nonceSize)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(buf), nil
}

// notEmbeddedHandler returns a handler that displays a message when frontend is not embedded
func notEmbeddedHandler(message string) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		ctx.SetContentType("text/plain; charset=utf-8")
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		_, _ = ctx.WriteString(message)
	}
}
