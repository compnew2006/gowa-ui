package handlers

import (
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/compnew2006/whatomate/internal/models"
)

const (
	bytesPerMB int64 = 1024 * 1024

	whatsappImageMaxBytes    int64 = 5 * bytesPerMB
	whatsappVideoMaxBytes    int64 = 16 * bytesPerMB
	whatsappAudioMaxBytes    int64 = 16 * bytesPerMB
	whatsappDocumentMaxBytes int64 = 100 * bytesPerMB

	applicationOctetStreamMIME = "application/octet-stream"
)

var (
	whatsappImageMIMEs = map[string]struct{}{
		"image/jpeg": {},
		"image/png":  {},
		"image/webp": {},
	}
	whatsappVideoMIMEs = map[string]struct{}{
		"video/mp4":  {},
		"video/3gpp": {},
	}
	whatsappAudioMIMEs = map[string]struct{}{
		"audio/aac":  {},
		"audio/amr":  {},
		"audio/mpeg": {},
		"audio/mp4":  {},
		"audio/ogg":  {},
	}
)

func normalizeWhatsAppMediaMIME(raw string) string {
	value := strings.TrimSpace(strings.ToLower(raw))
	if value == "" {
		return ""
	}

	parsedType, _, err := mime.ParseMediaType(value)
	if err == nil {
		return strings.TrimSpace(strings.ToLower(parsedType))
	}

	if delimiter := strings.Index(value, ";"); delimiter >= 0 {
		value = value[:delimiter]
	}

	return strings.TrimSpace(strings.ToLower(value))
}

func mimeTypeFromFilenameExtension(filename string) string {
	ext := strings.ToLower(strings.TrimSpace(filepath.Ext(filename)))
	if ext == "" {
		return ""
	}

	return normalizeWhatsAppMediaMIME(mime.TypeByExtension(ext))
}

func resolveWhatsAppMediaMIME(partContentType, filename string, fileData []byte) string {
	detectedMIME := ""
	if len(fileData) > 0 {
		sniffBytes := fileData
		if len(sniffBytes) > 512 {
			sniffBytes = sniffBytes[:512]
		}
		detectedMIME = normalizeWhatsAppMediaMIME(http.DetectContentType(sniffBytes))
	}
	if detectedMIME != "" && detectedMIME != applicationOctetStreamMIME {
		return detectedMIME
	}

	if partMIME := normalizeWhatsAppMediaMIME(partContentType); partMIME != "" && partMIME != applicationOctetStreamMIME {
		return partMIME
	}

	if extensionMIME := mimeTypeFromFilenameExtension(filename); extensionMIME != "" {
		return extensionMIME
	}

	if detectedMIME != "" {
		return detectedMIME
	}

	return applicationOctetStreamMIME
}

func deriveWhatsAppMediaMessageType(mimeType string) models.MessageType {
	normalized := normalizeWhatsAppMediaMIME(mimeType)

	if _, ok := whatsappImageMIMEs[normalized]; ok {
		return models.MessageTypeImage
	}
	if _, ok := whatsappVideoMIMEs[normalized]; ok {
		return models.MessageTypeVideo
	}
	if _, ok := whatsappAudioMIMEs[normalized]; ok {
		return models.MessageTypeAudio
	}

	return models.MessageTypeDocument
}

func whatsappMediaMaxSizeBytes(messageType models.MessageType) int64 {
	switch messageType {
	case models.MessageTypeImage:
		return whatsappImageMaxBytes
	case models.MessageTypeVideo:
		return whatsappVideoMaxBytes
	case models.MessageTypeAudio:
		return whatsappAudioMaxBytes
	default:
		return whatsappDocumentMaxBytes
	}
}

func whatsappMediaMaxSizeMB(messageType models.MessageType) int64 {
	return whatsappMediaMaxSizeBytes(messageType) / bytesPerMB
}
