package gowa

import "encoding/json"

// sendResponse is the envelope returned by all GOWA send operations.
// The code field is polymorphic: the string "SUCCESS" on success and an
// integer HTTP status code on error, so it is typed as json.RawMessage.
type sendResponse struct {
	Code    json.RawMessage `json:"code"`
	Message string          `json:"message"`
	Results struct {
		MessageID string `json:"message_id"`
		Status    string `json:"status"`
	} `json:"results"`
}

// downloadResponse is the envelope returned by GET /message/{id}/download.
type downloadResponse struct {
	Status  int             `json:"status"`
	Code    json.RawMessage `json:"code"`
	Message string          `json:"message"`
	Results struct {
		MessageID string `json:"message_id"`
		MediaType string `json:"media_type"`
		Filename  string `json:"filename"`
		FilePath  string `json:"file_path"`
		FileURL   string `json:"file_url"`
		FileSize  int64  `json:"file_size"`
	} `json:"results"`
}

// mediaCacheItem holds raw bytes for the UploadMedia → send pattern.
// GOWA has no standalone upload endpoint, so we stash bytes keyed by a
// generated ID and the send methods consume them inline.
type mediaCacheItem struct {
	Data     []byte
	MimeType string
	Filename string
}
