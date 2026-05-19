package whatsmeow

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"path"
	"strings"
	"sync/atomic"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	objectstorage "github.com/compnew2006/whatomate/internal/storage"
	"github.com/google/uuid"
	"github.com/zerodha/logf"
	waClient "go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
	"go.mau.fi/whatsmeow/util/hkdfutil"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const mediaHMACLength = 10

type mediaContextInstanceIDKey struct{}

type mediaStreamFunc func(ctx context.Context, client *waClient.Client, media waClient.DownloadableMessage, dst io.Writer) (int64, error)

type downloadableMessageWithLength interface {
	waClient.DownloadableMessage
	GetFileLength() uint64
}

type downloadableMessageWithSizeBytes interface {
	waClient.DownloadableMessage
	GetFileSizeBytes() uint64
}

type downloadableMessageWithURL interface {
	waClient.DownloadableMessage
	GetURL() string
}

// HandledMedia describes the object-store result for an inbound media message.
type HandledMedia struct {
	MediaAssetID uuid.UUID
	MimeType     string
	Filename     string
	Size         int64
	WasDedupHit  bool
}

// MediaService streams inbound WhatsApp media into object storage.
type MediaService struct {
	db             *gorm.DB
	storage        objectstorage.ObjectStorage
	logger         logf.Logger
	clientResolver func(uuid.UUID) *waClient.Client
	streamDownload mediaStreamFunc
}

// NewMediaService builds a MediaService bound to the current connection manager.
func NewMediaService(
	db *gorm.DB,
	storage objectstorage.ObjectStorage,
	logger logf.Logger,
	clientResolver func(uuid.UUID) *waClient.Client,
) *MediaService {
	return &MediaService{
		db:             db,
		storage:        storage,
		logger:         logger,
		clientResolver: clientResolver,
		streamDownload: streamDownloadableToWriter,
	}
}

// WithMediaInstanceID binds the whatsmeow instance ID needed by HandleIncomingMedia.
func WithMediaInstanceID(ctx context.Context, instanceID uuid.UUID) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, mediaContextInstanceIDKey{}, instanceID)
}

func mediaInstanceIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	if ctx == nil {
		return uuid.Nil, false
	}
	instanceID, ok := ctx.Value(mediaContextInstanceIDKey{}).(uuid.UUID)
	if !ok || instanceID == uuid.Nil {
		return uuid.Nil, false
	}
	return instanceID, true
}

// HandleIncomingMedia extracts media from an inbound event, deduplicates using the native file hash,
// and streams the content into object storage.
func (s *MediaService) HandleIncomingMedia(ctx context.Context, evt *events.Message) (*HandledMedia, error) {
	if evt == nil {
		return nil, fmt.Errorf("event is nil")
	}
	instanceID, ok := mediaInstanceIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("instance id missing from context")
	}

	baseMessage := evt.Message
	if baseMessage == nil {
		baseMessage = evt.RawMessage
	}
	descriptor, ok := describeIncomingDownloadable(unwrapIncomingMessage(baseMessage))
	if !ok {
		return nil, fmt.Errorf("event does not contain downloadable media")
	}
	return s.handleDownloadable(ctx, instanceID, descriptor)
}

func (s *MediaService) handleDownloadable(
	ctx context.Context,
	instanceID uuid.UUID,
	descriptor *incomingDownloadableDescriptor,
) (*HandledMedia, error) {
	if s == nil {
		return nil, fmt.Errorf("media service is nil")
	}
	if s.db == nil {
		return nil, fmt.Errorf("database is not configured")
	}
	if s.storage == nil {
		return nil, fmt.Errorf("object storage is not configured")
	}
	if descriptor == nil || descriptor.Downloadable == nil {
		return nil, fmt.Errorf("downloadable media descriptor is nil")
	}

	fileHash, err := nativeMediaFileHash(descriptor.Downloadable)
	if err != nil {
		return nil, err
	}

	var existing models.MediaAsset
	err = s.db.WithContext(ctx).
		Where("file_hash = ?", fileHash).
		First(&existing).Error
	var restorable *models.MediaAsset
	if err == nil {
		if s.mediaAssetObjectExists(ctx, &existing) {
			return &HandledMedia{
				MediaAssetID: existing.ID,
				MimeType:     coalesceMediaValue(existing.MimeType, descriptor.MimeType, "application/octet-stream"),
				Filename:     descriptor.Filename,
				Size:         coalesceMediaSize(existing.Size, downloadableSize(descriptor.Downloadable)),
				WasDedupHit:  true,
			}, nil
		}
		s.logger.Warn("Deduplicated media asset is missing from object storage; restoring from inbound payload", "asset_id", existing.ID, "file_hash", fileHash, "s3_key", existing.S3Key)
		restorable = &existing
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("lookup media asset by hash: %w", err)
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		var deleted models.MediaAsset
		if err := s.db.WithContext(ctx).
			Unscoped().
			Where("file_hash = ? AND deleted_at IS NOT NULL", fileHash).
			First(&deleted).Error; err == nil {
			restorable = &deleted
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("lookup deleted media asset by hash: %w", err)
		}
	}

	if s.clientResolver == nil {
		return nil, fmt.Errorf("client resolver is not configured")
	}
	client := s.clientResolver(instanceID)
	if client == nil {
		return nil, waClient.ErrClientIsNil
	}

	expectedSize := downloadableSize(descriptor.Downloadable)
	objectKey := buildMediaObjectKey(fileHash)
	resolvedMimeType := coalesceMediaValue(descriptor.MimeType, "application/octet-stream")

	uploadCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	pr, pw := io.Pipe()
	counter := &countingWriter{writer: pw}
	streamErrCh := make(chan error, 1)

	go func() {
		_, streamErr := s.streamDownload(uploadCtx, client, descriptor.Downloadable, counter)
		_ = pw.CloseWithError(streamErr)
		streamErrCh <- streamErr
		close(streamErrCh)
	}()

	uploadErr := s.storage.PutObject(uploadCtx, objectKey, pr, expectedSize, resolvedMimeType)
	if uploadErr != nil {
		cancel()
		_ = pr.CloseWithError(uploadErr)
	}

	streamErr := <-streamErrCh
	if uploadErr != nil {
		if streamErr != nil && !errors.Is(streamErr, context.Canceled) && !errors.Is(streamErr, io.ErrClosedPipe) {
			return nil, fmt.Errorf("stream whatsapp media: %w", streamErr)
		}
		return nil, fmt.Errorf("upload media object: %w", uploadErr)
	}
	if streamErr != nil {
		return nil, fmt.Errorf("stream whatsapp media: %w", streamErr)
	}

	actualSize := counter.Count()
	if expectedSize >= 0 && actualSize != expectedSize {
		return nil, fmt.Errorf("%w: expected %d, got %d", waClient.ErrFileLengthMismatch, expectedSize, actualSize)
	}

	if restorable != nil {
		updates := map[string]any{
			"s3_key":     objectKey,
			"mime_type":  resolvedMimeType,
			"size":       actualSize,
			"deleted_at": nil,
			"updated_at": time.Now().UTC(),
		}
		if err := s.db.WithContext(ctx).
			Unscoped().
			Model(&models.MediaAsset{}).
			Where("id = ?", restorable.ID).
			Updates(updates).Error; err != nil {
			return nil, fmt.Errorf("restore media asset: %w", err)
		}
		restorable.S3Key = objectKey
		restorable.MimeType = resolvedMimeType
		restorable.Size = actualSize
		return &HandledMedia{
			MediaAssetID: restorable.ID,
			MimeType:     resolvedMimeType,
			Filename:     descriptor.Filename,
			Size:         actualSize,
		}, nil
	}

	asset := models.MediaAsset{
		BaseModel: models.BaseModel{ID: uuid.New()},
		FileHash:  fileHash,
		S3Key:     objectKey,
		MimeType:  resolvedMimeType,
		Size:      actualSize,
	}
	createResult := s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "file_hash"}},
			DoNothing: true,
		}).
		Create(&asset)
	if createResult.Error != nil {
		return nil, fmt.Errorf("create media asset: %w", createResult.Error)
	}

	if createResult.RowsAffected == 0 {
		var concurrentAsset models.MediaAsset
		if err := s.db.WithContext(ctx).
			Where("file_hash = ?", fileHash).
			First(&concurrentAsset).Error; err != nil {
			return nil, fmt.Errorf("reload concurrent media asset: %w", err)
		}
		asset = concurrentAsset
	}

	return &HandledMedia{
		MediaAssetID: asset.ID,
		MimeType:     resolvedMimeType,
		Filename:     descriptor.Filename,
		Size:         asset.Size,
	}, nil
}

func (s *MediaService) mediaAssetObjectExists(ctx context.Context, asset *models.MediaAsset) bool {
	if s == nil || s.storage == nil || asset == nil || strings.TrimSpace(asset.S3Key) == "" {
		return false
	}
	reader, _, err := s.storage.GetObject(ctx, asset.S3Key)
	if err != nil {
		return false
	}
	if reader != nil {
		_ = reader.Close()
	}
	return true
}

func nativeMediaFileHash(media waClient.DownloadableMessage) (string, error) {
	if media == nil {
		return "", fmt.Errorf("downloadable media is nil")
	}
	hash := media.GetFileSHA256()
	if len(hash) == 0 {
		return "", fmt.Errorf("downloadable media is missing FileSha256")
	}
	return strings.ToLower(hex.EncodeToString(hash)), nil
}

func downloadableSize(media waClient.DownloadableMessage) int64 {
	switch sized := media.(type) {
	case downloadableMessageWithLength:
		return safeInt64FromUint64(sized.GetFileLength())
	case downloadableMessageWithSizeBytes:
		return safeInt64FromUint64(sized.GetFileSizeBytes())
	default:
		return -1
	}
}

func safeInt64FromUint64(value uint64) int64 {
	if value > math.MaxInt64 {
		return -1
	}
	return int64(value)
}

func buildMediaObjectKey(fileHash string) string {
	if len(fileHash) < 4 {
		return path.Join("whatsmeow", "media", fileHash)
	}
	return path.Join("whatsmeow", "media", fileHash[:2], fileHash[2:4], fileHash)
}

func streamDownloadableToWriter(
	ctx context.Context,
	client *waClient.Client,
	media waClient.DownloadableMessage,
	dst io.Writer,
) (int64, error) {
	if client == nil {
		return 0, waClient.ErrClientIsNil
	}
	mediaType := waClient.GetMediaType(media)
	if mediaType == "" {
		return 0, fmt.Errorf("%w %T", waClient.ErrUnknownMediaType, media)
	}

	if urlable, ok := media.(downloadableMessageWithURL); ok {
		url := strings.TrimSpace(urlable.GetURL())
		if url != "" && !strings.HasPrefix(url, "https://web.whatsapp.net") {
			return streamMediaURLToWriter(ctx, client, url, media.GetMediaKey(), mediaType, downloadableSize(media), media.GetFileEncSHA256(), media.GetFileSHA256(), dst)
		}
	}

	directPath := strings.TrimSpace(media.GetDirectPath())
	if directPath == "" {
		return 0, waClient.ErrNoURLPresent
	}

	return streamMediaWithPathToWriter(
		ctx,
		client,
		directPath,
		media.GetMediaKey(),
		mediaType,
		downloadableSize(media),
		media.GetFileEncSHA256(),
		media.GetFileSHA256(),
		dst,
	)
}

func streamMediaWithPathToWriter(
	ctx context.Context,
	client *waClient.Client,
	directPath string,
	mediaKey []byte,
	mediaType waClient.MediaType,
	fileLength int64,
	fileEncSHA256 []byte,
	fileSHA256 []byte,
	dst io.Writer,
) (int64, error) {
	if !strings.HasPrefix(directPath, "/") {
		return 0, fmt.Errorf("media download path does not start with slash: %s", directPath)
	}

	mediaConn, err := client.DangerousInternals().RefreshMediaConn(ctx, false)
	if err != nil {
		return 0, fmt.Errorf("refresh media connections: %w", err)
	}
	if mediaConn == nil || len(mediaConn.Hosts) == 0 {
		return 0, fmt.Errorf("no media hosts returned by whatsmeow")
	}

	var lastErr error
	for idx, host := range mediaConn.Hosts {
		mediaURL := fmt.Sprintf(
			"https://%s%s&hash=%s&mms-type=%s&__wa-mms=",
			host.Hostname,
			directPath,
			base64.URLEncoding.EncodeToString(fileEncSHA256),
			mediaTypeToMMSType(mediaType),
		)
		written, err := streamMediaURLToWriter(ctx, client, mediaURL, mediaKey, mediaType, fileLength, fileEncSHA256, fileSHA256, dst)
		if err == nil {
			return written, nil
		}
		lastErr = err
		if written > 0 ||
			errors.Is(err, waClient.ErrFileLengthMismatch) ||
			errors.Is(err, waClient.ErrInvalidMediaSHA256) ||
			errors.Is(err, waClient.ErrMediaDownloadFailedWith403) ||
			errors.Is(err, waClient.ErrMediaDownloadFailedWith404) ||
			errors.Is(err, waClient.ErrMediaDownloadFailedWith410) ||
			errors.Is(err, context.Canceled) ||
			idx >= len(mediaConn.Hosts)-1 {
			break
		}
	}

	if lastErr == nil {
		lastErr = waClient.ErrNoURLPresent
	}
	return 0, lastErr
}

func streamMediaURLToWriter(
	ctx context.Context,
	client *waClient.Client,
	url string,
	mediaKey []byte,
	mediaType waClient.MediaType,
	fileLength int64,
	fileEncSHA256 []byte,
	fileSHA256 []byte,
	dst io.Writer,
) (int64, error) {
	resp, err := client.DangerousInternals().DoMediaDownloadRequest(ctx, url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if len(mediaKey) == 0 && len(fileEncSHA256) == 0 {
		return streamPlainMediaToWriter(resp.Body, fileLength, fileSHA256, dst)
	}
	return streamEncryptedMediaToWriter(resp.Body, mediaKey, mediaType, fileLength, fileEncSHA256, fileSHA256, dst)
}

func streamPlainMediaToWriter(body io.Reader, fileLength int64, fileSHA256 []byte, dst io.Writer) (int64, error) {
	hasher := sha256.New()
	writer := io.MultiWriter(dst, hasher)
	written, err := io.CopyBuffer(writer, body, make([]byte, 32*1024))
	if err != nil {
		return written, err
	}
	if fileLength >= 0 && written != fileLength {
		return written, fmt.Errorf("%w: expected %d, got %d", waClient.ErrFileLengthMismatch, fileLength, written)
	}
	if len(fileSHA256) == 32 && !hmac.Equal(fileSHA256, hasher.Sum(nil)) {
		return written, waClient.ErrInvalidMediaSHA256
	}
	return written, nil
}

func streamEncryptedMediaToWriter(
	body io.Reader,
	mediaKey []byte,
	mediaType waClient.MediaType,
	fileLength int64,
	fileEncSHA256 []byte,
	fileSHA256 []byte,
	dst io.Writer,
) (int64, error) {
	iv, cipherKey, macKey := deriveMediaKeys(mediaKey, mediaType)
	block, err := aes.NewCipher(cipherKey)
	if err != nil {
		return 0, fmt.Errorf("create media cipher: %w", err)
	}

	decrypter := cipher.NewCBCDecrypter(block, iv)
	encryptedHasher := sha256.New()
	plaintextHasher := sha256.New()
	hmacHasher := hmac.New(sha256.New, macKey)
	_, _ = hmacHasher.Write(iv)

	tail := make([]byte, 0, mediaHMACLength)
	ciphertextBuf := make([]byte, 0, 64*1024)
	var pendingPlain []byte
	var written int64
	readBuf := make([]byte, 32*1024)

	for {
		n, readErr := body.Read(readBuf)
		if n > 0 {
			chunk := readBuf[:n]
			_, _ = encryptedHasher.Write(chunk)
			tail = append(tail, chunk...)

			processLen := len(tail) - mediaHMACLength
			if processLen > 0 {
				ciphertextPart := append([]byte(nil), tail[:processLen]...)
				tail = append(tail[:0], tail[processLen:]...)
				_, _ = hmacHasher.Write(ciphertextPart)
				ciphertextBuf = append(ciphertextBuf, ciphertextPart...)

				for len(ciphertextBuf) >= aes.BlockSize {
					blockCiphertext := append([]byte(nil), ciphertextBuf[:aes.BlockSize]...)
					ciphertextBuf = ciphertextBuf[aes.BlockSize:]

					plainBlock := make([]byte, aes.BlockSize)
					decrypter.CryptBlocks(plainBlock, blockCiphertext)

					if pendingPlain != nil {
						blockWritten, err := writePlaintext(dst, plaintextHasher, pendingPlain)
						written += blockWritten
						if err != nil {
							return written, err
						}
					}
					pendingPlain = plainBlock
				}
			}
		}

		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return written, readErr
		}
	}

	if len(tail) <= mediaHMACLength {
		if pendingPlain == nil && len(tail) <= mediaHMACLength {
			return written, waClient.ErrTooShortFile
		}
	}
	if len(tail) != mediaHMACLength {
		return written, waClient.ErrTooShortFile
	}
	if len(ciphertextBuf) != 0 || pendingPlain == nil {
		return written, fmt.Errorf("invalid encrypted media block length")
	}
	if len(fileEncSHA256) == 32 && !hmac.Equal(fileEncSHA256, encryptedHasher.Sum(nil)) {
		return written, waClient.ErrInvalidMediaEncSHA256
	}
	if !hmac.Equal(hmacHasher.Sum(nil)[:mediaHMACLength], tail) {
		return written, waClient.ErrInvalidMediaHMAC
	}

	finalPlain, err := removePKCS7Padding(pendingPlain, aes.BlockSize)
	if err != nil {
		return written, err
	}
	blockWritten, err := writePlaintext(dst, plaintextHasher, finalPlain)
	written += blockWritten
	if err != nil {
		return written, err
	}

	if fileLength >= 0 && written != fileLength {
		return written, fmt.Errorf("%w: expected %d, got %d", waClient.ErrFileLengthMismatch, fileLength, written)
	}
	if len(fileSHA256) == 32 && !hmac.Equal(fileSHA256, plaintextHasher.Sum(nil)) {
		return written, waClient.ErrInvalidMediaSHA256
	}
	return written, nil
}

func deriveMediaKeys(mediaKey []byte, mediaType waClient.MediaType) ([]byte, []byte, []byte) {
	expanded := hkdfutil.SHA256(mediaKey, nil, []byte(mediaType), 112)
	return expanded[:16], expanded[16:48], expanded[48:80]
}

func mediaTypeToMMSType(mediaType waClient.MediaType) string {
	switch mediaType {
	case waClient.MediaImage:
		return "image"
	case waClient.MediaAudio:
		return "audio"
	case waClient.MediaVideo:
		return "video"
	case waClient.MediaDocument:
		return "document"
	case waClient.MediaHistory:
		return "md-msg-hist"
	case waClient.MediaAppState:
		return "md-app-state"
	case waClient.MediaStickerPack:
		return "sticker-pack"
	case waClient.MediaLinkThumbnail:
		return "thumbnail-link"
	default:
		return "document"
	}
}

func removePKCS7Padding(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 || len(data)%blockSize != 0 {
		return nil, fmt.Errorf("invalid padded payload length")
	}
	paddingLen := int(data[len(data)-1])
	if paddingLen == 0 || paddingLen > blockSize || paddingLen > len(data) {
		return nil, fmt.Errorf("invalid media padding")
	}
	for _, value := range data[len(data)-paddingLen:] {
		if int(value) != paddingLen {
			return nil, fmt.Errorf("invalid media padding")
		}
	}
	return data[:len(data)-paddingLen], nil
}

func writePlaintext(dst io.Writer, hasher io.Writer, chunk []byte) (int64, error) {
	if len(chunk) == 0 {
		return 0, nil
	}
	n, err := dst.Write(chunk)
	if n > 0 {
		_, _ = hasher.Write(chunk[:n])
	}
	if err == nil && n != len(chunk) {
		err = io.ErrShortWrite
	}
	return int64(n), err
}

func coalesceMediaValue(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func coalesceMediaSize(values ...int64) int64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

type countingWriter struct {
	writer io.Writer
	count  atomic.Int64
}

func (cw *countingWriter) Write(p []byte) (int, error) {
	n, err := cw.writer.Write(p)
	if n > 0 {
		cw.count.Add(int64(n))
	}
	return n, err
}

func (cw *countingWriter) Count() int64 {
	return cw.count.Load()
}
