package licensestudio

import (
	"context"
	"crypto/ed25519"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/compnew2006/whatomate/internal/license"
	"github.com/compnew2006/whatomate/internal/licenseissuer"
)

const (
	maxIssueRequestBytes   = 1 << 20
	maxVerifyRequestBytes  = 1 << 20
	maxPrivateKeyBytes     = 8 << 10
	maxFormFieldBytes      = 16 << 10
	defaultShutdownTimeout = 5 * time.Second
)

//go:embed all:frontend/dist
var studioFS embed.FS

type Config struct {
	Addr         string
	DataDir      string
	RegistryPath string
	KeyRingPath  string
	OpenBrowser  bool
	Logger       *log.Logger
}

type Server struct {
	addr        string
	log         *log.Logger
	openBrowser bool
	registry    *licenseissuer.RegistryStore
	keyRing     *licenseissuer.KeyRingStore
	mux         *http.ServeMux
}

type bootstrapResponse struct {
	Defaults     bootstrapDefaults             `json:"defaults"`
	Summary      licenseissuer.RegistrySummary `json:"summary"`
	KnownKeyIDs  []string                      `json:"known_key_ids"`
	RegistryPath string                        `json:"registry_path"`
	KeyRingPath  string                        `json:"keyring_path"`
}

type bootstrapDefaults struct {
	KID           string `json:"kid"`
	Duration      string `json:"duration"`
	Tier          string `json:"tier"`
	Organizations int    `json:"orgs"`
	Users         int    `json:"users"`
	WAEndpoints   int    `json:"wa_endpoints"`
	Workers       int    `json:"workers"`
}

type issueResponse struct {
	Token       string                        `json:"token"`
	Entry       registryEntryView             `json:"entry"`
	Summary     licenseissuer.RegistrySummary `json:"summary"`
	KnownKeyIDs []string                      `json:"known_key_ids"`
}

type verifyRequest struct {
	Token string `json:"token"`
}

type licensesResponse struct {
	Items []registryEntryView `json:"items"`
}

type registryEntryView struct {
	licenseissuer.RegistryEntry
	Status string `json:"status"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func NewServer(cfg Config) (*Server, error) {
	addr := strings.TrimSpace(cfg.Addr)
	if addr == "" {
		addr = licenseissuer.DefaultStudioBindAddr
	}
	logger := cfg.Logger
	if logger == nil {
		logger = log.New(os.Stdout, "", log.LstdFlags)
	}

	dataDir := strings.TrimSpace(cfg.DataDir)
	if dataDir == "" {
		defaultDir, err := licenseissuer.DefaultStudioDataDir()
		if err != nil {
			return nil, err
		}
		dataDir = defaultDir
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	registryPath := strings.TrimSpace(cfg.RegistryPath)
	if registryPath == "" {
		registryPath = filepath.Join(dataDir, licenseissuer.DefaultRegistryName)
	}
	keyRingPath := strings.TrimSpace(cfg.KeyRingPath)
	if keyRingPath == "" {
		keyRingPath = filepath.Join(dataDir, licenseissuer.DefaultKeyRingName)
	}

	server := &Server{
		addr:        addr,
		log:         logger,
		openBrowser: cfg.OpenBrowser,
		registry:    licenseissuer.NewRegistryStore(registryPath),
		keyRing:     licenseissuer.NewKeyRingStore(keyRingPath),
		mux:         http.NewServeMux(),
	}
	server.routes(registryPath, keyRingPath)
	return server, nil
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	defer func() { _ = listener.Close() }()

	actualAddr := listener.Addr().String()
	s.log.Printf("License studio listening on http://%s", actualAddr)

	if s.openBrowser {
		go func() {
			time.Sleep(200 * time.Millisecond)
			if err := openBrowser("http://" + actualAddr); err != nil {
				s.log.Printf("open browser: %v", err)
			}
		}()
	}

	httpServer := &http.Server{
		Handler: s.Handler(),
	}

	errCh := make(chan error, 1)
	go func() {
		if serveErr := httpServer.Serve(listener); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			errCh <- serveErr
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
		return nil
	case serveErr := <-errCh:
		return serveErr
	}
}

func (s *Server) RunUntilSignal() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return s.ListenAndServe(ctx)
}

func (s *Server) routes(registryPath, keyRingPath string) {
	s.mux.HandleFunc("/api/bootstrap", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
			return
		}

		defaults := licenseissuer.DefaultIssueOptions()
		summary, err := s.registry.Summary()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
			return
		}
		kids, err := s.keyRing.KnownKeyIDs()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, bootstrapResponse{
			Defaults: bootstrapDefaults{
				KID:           defaults.KeyID,
				Duration:      defaults.Duration,
				Tier:          defaults.Tier,
				Organizations: defaults.Organizations,
				Users:         defaults.UsersPerOrg,
				WAEndpoints:   defaults.WhatsAppEndpointsPerOrg,
				Workers:       defaults.Workers,
			},
			Summary:      summary,
			KnownKeyIDs:  kids,
			RegistryPath: registryPath,
			KeyRingPath:  keyRingPath,
		})
	})

	s.mux.HandleFunc("/api/issue", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxIssueRequestBytes)
		defer func() { _ = r.Body.Close() }()

		opts, privateKeyText, err := parseIssueMultipart(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}

		issued, err := licenseissuer.IssueLicenseFromPrivateKeyText(opts, privateKeyText, time.Now().UTC())
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}

		if err := s.keyRing.UpsertPublicKey(issued.KeyID, issued.PublicKey); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
			return
		}

		entry, err := s.registry.Save(licenseissuer.BuildRegistryEntry(issued))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
			return
		}

		summary, _ := s.registry.Summary()
		kids, _ := s.keyRing.KnownKeyIDs()
		writeJSON(w, http.StatusOK, issueResponse{
			Token:       issued.Token,
			Entry:       toRegistryEntryView(entry, time.Now()),
			Summary:     summary,
			KnownKeyIDs: kids,
		})
	})

	s.mux.HandleFunc("/api/verify", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxVerifyRequestBytes)
		defer func() { _ = r.Body.Close() }()

		var request verifyRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid verify request"})
			return
		}

		result := licenseissuer.VerifyToken(request.Token, s.keyRing, s.registry, time.Now().UTC())
		writeJSON(w, http.StatusOK, result)
	})

	s.mux.HandleFunc("/api/licenses", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
			return
		}

		filter := licenseissuer.RegistryFilter{
			HWID:   r.URL.Query().Get("hwid"),
			Tier:   r.URL.Query().Get("tier"),
			Kind:   r.URL.Query().Get("kind"),
			Status: r.URL.Query().Get("status"),
		}
		entries, err := s.registry.List(filter)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
			return
		}
		items := make([]registryEntryView, 0, len(entries))
		now := time.Now().UTC()
		for _, entry := range entries {
			items = append(items, toRegistryEntryView(entry, now))
		}
		writeJSON(w, http.StatusOK, licensesResponse{Items: items})
	})

	s.mux.HandleFunc("/api/licenses/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/api/licenses/")
		if !strings.HasSuffix(path, "/token") {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "not found"})
			return
		}
		id := strings.TrimSuffix(path, "/token")
		id = strings.Trim(id, "/")
		if id == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "license id is required"})
			return
		}

		entry, err := s.registry.FindByID(id)
		if err != nil {
			if os.IsNotExist(err) {
				writeJSON(w, http.StatusNotFound, errorResponse{Error: "license not found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"id":    entry.ID,
			"token": entry.Token,
		})
	})

	s.mux.HandleFunc("/", s.serveFrontend)
}

func (s *Server) serveFrontend(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "not found"})
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "frontend/dist/index.html"
	} else {
		path = filepath.ToSlash(filepath.Join("frontend/dist", path))
	}

	data, err := studioFS.ReadFile(path)
	if err == nil {
		w.Header().Set("Content-Type", contentType(path))
		_, _ = w.Write(data)
		return
	}

	index, readErr := studioFS.ReadFile("frontend/dist/index.html")
	if readErr != nil {
		http.Error(w, "embedded studio frontend missing", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(index)
}

func toRegistryEntryView(entry licenseissuer.RegistryEntry, now time.Time) registryEntryView {
	return registryEntryView{
		RegistryEntry: entry,
		Status:        entry.Status(now),
	}
}

func contentType(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	default:
		return "text/html; charset=utf-8"
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func parseIssueMultipart(r *http.Request) (licenseissuer.IssueOptions, string, error) {
	reader, err := r.MultipartReader()
	if err != nil {
		return licenseissuer.IssueOptions{}, "", fmt.Errorf("expected multipart form data")
	}

	opts := licenseissuer.DefaultIssueOptions()
	privateKeyText := ""

	for {
		part, err := reader.NextPart()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return licenseissuer.IssueOptions{}, "", err
		}

		if part.FormName() == "" {
			_ = part.Close()
			continue
		}

		if part.FormName() == "private_key_file" {
			data, readErr := readPartString(part, maxPrivateKeyBytes)
			_ = part.Close()
			if readErr != nil {
				return licenseissuer.IssueOptions{}, "", readErr
			}
			privateKeyText = data
			continue
		}

		value, readErr := readPartString(part, maxFormFieldBytes)
		_ = part.Close()
		if readErr != nil {
			return licenseissuer.IssueOptions{}, "", readErr
		}

		switch part.FormName() {
		case "kid":
			opts.KeyID = value
		case "hwid":
			opts.HWID = value
		case "duration":
			opts.Duration = value
		case "trial":
			opts.Trial = value
		case "tier":
			opts.Tier = value
		case "license_id":
			opts.LicenseID = value
		case "family_id":
			opts.FamilyID = value
		case "issued_at":
			opts.IssuedAt = value
		case "not_before":
			opts.NotBefore = value
		case "revision":
			parsed, parseErr := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
			if parseErr != nil {
				return licenseissuer.IssueOptions{}, "", fmt.Errorf("invalid revision")
			}
			opts.Revision = parsed
		case "orgs":
			parsed, parseErr := parsePositiveInt(value, "orgs")
			if parseErr != nil {
				return licenseissuer.IssueOptions{}, "", parseErr
			}
			opts.Organizations = parsed
		case "users":
			parsed, parseErr := parsePositiveInt(value, "users")
			if parseErr != nil {
				return licenseissuer.IssueOptions{}, "", parseErr
			}
			opts.UsersPerOrg = parsed
		case "wa_endpoints":
			parsed, parseErr := parsePositiveInt(value, "wa_endpoints")
			if parseErr != nil {
				return licenseissuer.IssueOptions{}, "", parseErr
			}
			opts.WhatsAppEndpointsPerOrg = parsed
		case "workers":
			parsed, parseErr := parsePositiveInt(value, "workers")
			if parseErr != nil {
				return licenseissuer.IssueOptions{}, "", parseErr
			}
			opts.Workers = parsed
		}
	}

	if strings.TrimSpace(opts.HWID) == "" {
		return licenseissuer.IssueOptions{}, "", fmt.Errorf("hwid is required")
	}
	if strings.TrimSpace(privateKeyText) == "" {
		return licenseissuer.IssueOptions{}, "", fmt.Errorf("private key file is required")
	}
	if opts.Trial != "" && opts.Duration != "" {
		if !isValidPaidDuration(opts.Duration) {
			return licenseissuer.IssueOptions{}, "", fmt.Errorf("duration must be a positive number of days like 55d or lifetime")
		}
	}

	return opts, privateKeyText, nil
}

func parsePositiveInt(value, field string) (int, error) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed < 0 {
		return 0, fmt.Errorf("invalid %s", field)
	}
	return parsed, nil
}

func isValidPaidDuration(value string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "lifetime" {
		return true
	}
	trimmed = strings.TrimSuffix(trimmed, "days")
	trimmed = strings.TrimSuffix(trimmed, "day")
	trimmed = strings.TrimSuffix(trimmed, "d")
	trimmed = strings.TrimSpace(trimmed)
	if trimmed == "" {
		return false
	}
	parsed, err := strconv.Atoi(trimmed)
	return err == nil && parsed > 0
}

func readPartString(part *multipart.Part, maxBytes int64) (string, error) {
	data, err := io.ReadAll(io.LimitReader(part, maxBytes+1))
	if err != nil {
		return "", err
	}
	if int64(len(data)) > maxBytes {
		return "", fmt.Errorf("%s is too large", part.FormName())
	}
	return strings.TrimSpace(string(data)), nil
}

func PublicKeyFromPrivateKeyText(raw string) (ed25519.PublicKey, error) {
	privateKey, err := license.DecodePrivateKey(strings.TrimSpace(raw))
	if err != nil {
		return nil, err
	}
	return privateKey.Public().(ed25519.PublicKey), nil
}
