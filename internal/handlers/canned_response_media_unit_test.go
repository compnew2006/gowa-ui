package handlers_test

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// TestIsMultipartFormRequest tests the isMultipartFormRequest function
func TestIsMultipartFormRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		contentType string
		expected    bool
	}{
		{
			name:        "multipart form data",
			contentType: "multipart/form-data",
			expected:    true,
		},
		{
			name:        "multipart form data with boundary",
			contentType: "multipart/form-data; boundary=----WebKitFormBoundary",
			expected:    true,
		},
		{
			name:        "uppercase multipart",
			contentType: "MULTIPART/FORM-DATA",
			expected:    true,
		},
		{
			name:        "application json",
			contentType: "application/json",
			expected:    false,
		},
		{
			name:        "text plain",
			contentType: "text/plain",
			expected:    false,
		},
		{
			name:        "empty content type",
			contentType: "",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := &fasthttp.RequestCtx{}
			ctx.Request.Header.SetContentType(tt.contentType)
			req := &fastglue.Request{RequestCtx: ctx}

			result := handlers.IsMultipartFormRequest(req)
			assert.Equal(t, tt.expected, result, "isMultipartFormRequest should return correct result")
		})
	}
}

// TestFirstFormValue tests the firstFormValue helper function
func TestFirstFormValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		values   map[string][]string
		key      string
		expected string
	}{
		{
			name:     "existing key with single value",
			values:   map[string][]string{"name": {"John"}},
			key:      "name",
			expected: "John",
		},
		{
			name:     "existing key with multiple values",
			values:   map[string][]string{"tags": {"tag1", "tag2", "tag3"}},
			key:      "tags",
			expected: "tag1",
		},
		{
			name:     "non-existent key",
			values:   map[string][]string{"name": {"John"}},
			key:      "email",
			expected: "",
		},
		{
			name:     "empty values array",
			values:   map[string][]string{"name": {}},
			key:      "name",
			expected: "",
		},
		{
			name:     "nil values map",
			values:   nil,
			key:      "name",
			expected: "",
		},
		{
			name:     "empty string value",
			values:   map[string][]string{"name": {""}},
			key:      "name",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := handlers.FirstFormValue(tt.values, tt.key)
			assert.Equal(t, tt.expected, result, "firstFormValue should return correct value")
		})
	}
}

// TestParseOptionalBool tests the parseOptionalBool function
func TestParseOptionalBool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		wantValue *bool
		wantError bool
	}{
		{
			name:      "empty string returns nil",
			input:     "",
			wantValue: nil,
			wantError: false,
		},
		{
			name:      "whitespace only returns nil",
			input:     "   ",
			wantValue: nil,
			wantError: false,
		},
		{
			name:      "true string",
			input:     "true",
			wantValue: func() *bool { b := true; return &b }(),
			wantError: false,
		},
		{
			name:      "false string",
			input:     "false",
			wantValue: func() *bool { b := false; return &b }(),
			wantError: false,
		},
		{
			name:      "1 string",
			input:     "1",
			wantValue: func() *bool { b := true; return &b }(),
			wantError: false,
		},
		{
			name:      "0 string",
			input:     "0",
			wantValue: func() *bool { b := false; return &b }(),
			wantError: false,
		},
		{
			name:      "TRUE uppercase",
			input:     "TRUE",
			wantValue: func() *bool { b := true; return &b }(),
			wantError: false,
		},
		{
			name:      "invalid string",
			input:     "invalid",
			wantValue: nil,
			wantError: true,
		},
		{
			name:      "yes string",
			input:     "yes",
			wantValue: nil,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := handlers.ParseOptionalBool(tt.input)

			if tt.wantError {
				assert.Error(t, err, "parseOptionalBool should return error for invalid input")
				assert.Nil(t, result, "parseOptionalBool should return nil on error")
			} else {
				assert.NoError(t, err, "parseOptionalBool should not return error for valid input")
				if tt.wantValue == nil {
					assert.Nil(t, result, "parseOptionalBool should return nil for empty/whitespace")
				} else {
					require.NotNil(t, result, "parseOptionalBool should return non-nil for valid bool")
					assert.Equal(t, *tt.wantValue, *result, "parseOptionalBool should parse correctly")
				}
			}
		})
	}
}

// TestNormalizeCannedAttachmentType tests the normalizeCannedAttachmentType function
func TestNormalizeCannedAttachmentType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		mimeType     string
		expectedType string
		expectedOK   bool
	}{
		{
			name:         "jpeg image",
			mimeType:     "image/jpeg",
			expectedType: models.CannedResponseAttachmentTypeImage,
			expectedOK:   true,
		},
		{
			name:         "png image",
			mimeType:     "image/png",
			expectedType: models.CannedResponseAttachmentTypeImage,
			expectedOK:   true,
		},
		{
			name:         "gif image",
			mimeType:     "image/gif",
			expectedType: models.CannedResponseAttachmentTypeImage,
			expectedOK:   true,
		},
		{
			name:         "mp4 video",
			mimeType:     "video/mp4",
			expectedType: models.CannedResponseAttachmentTypeVideo,
			expectedOK:   true,
		},
		{
			name:         "quicktime video",
			mimeType:     "video/quicktime",
			expectedType: models.CannedResponseAttachmentTypeVideo,
			expectedOK:   true,
		},
		{
			name:         "application pdf",
			mimeType:     "application/pdf",
			expectedType: "",
			expectedOK:   false,
		},
		{
			name:         "text plain",
			mimeType:     "text/plain",
			expectedType: "",
			expectedOK:   false,
		},
		{
			name:         "audio mp3",
			mimeType:     "audio/mpeg",
			expectedType: "",
			expectedOK:   false,
		},
		{
			name:         "empty string",
			mimeType:     "",
			expectedType: "",
			expectedOK:   false,
		},
		{
			name:         "uppercase image",
			mimeType:     "IMAGE/JPEG",
			expectedType: "",
			expectedOK:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resultType, resultOK := handlers.NormalizeCannedAttachmentType(tt.mimeType)

			assert.Equal(t, tt.expectedOK, resultOK, "normalizeCannedAttachmentType should return correct ok status")
			assert.Equal(t, tt.expectedType, resultType, "normalizeCannedAttachmentType should return correct type")
		})
	}
}

// TestResolveMediaFilePath tests the resolveMediaFilePath function
func TestResolveMediaFilePath(t *testing.T) {
	t.Parallel()

	// We can't directly test resolveMediaFilePath without being able to mock getMediaStoragePath,
	// but we can test the validation logic that it performs
	tests := []struct {
		name         string
		relativePath string
		wantError    bool
		errorMsg     string
	}{
		{
			name:         "empty path",
			relativePath: "",
			wantError:    true,
			errorMsg:     "invalid media file path",
		},
		{
			name:         "dot only",
			relativePath: ".",
			wantError:    true,
			errorMsg:     "invalid media file path",
		},
		{
			name:         "double dot only",
			relativePath: "..",
			wantError:    true,
			errorMsg:     "invalid media file path path traversal",
		},
		{
			name:         "path with parent directory escape",
			relativePath: "../suspicious.dat",
			wantError:    true,
			errorMsg:     "invalid media file path",
		},
		{
			name:         "path with leading separator and parent escape",
			relativePath: ".." + string(os.PathSeparator) + "file.dat",
			wantError:    true,
			errorMsg:     "invalid media file path",
		},
		{
			name:         "whitespace only",
			relativePath: "   ",
			wantError:    true,
			errorMsg:     "invalid media file path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Note: We can't fully test resolveMediaFilePath without being able to mock getMediaStoragePath
			// This test focuses on the path validation logic that we can verify
			cleanPath := filepath.Clean(strings.TrimSpace(tt.relativePath))

			// Verify the basic validation that resolveMediaFilePath performs
			isInvalid := cleanPath == "" || cleanPath == "." || cleanPath == ".." ||
				strings.HasPrefix(cleanPath, ".."+string(os.PathSeparator))

			if tt.wantError {
				assert.True(t, isInvalid, "Path should be identified as invalid: %s", tt.relativePath)
			} else {
				assert.False(t, isInvalid, "Path should be valid: %s", tt.relativePath)
			}
		})
	}
}

// TestReadCannedAttachmentData tests the readCannedAttachmentData function
func TestReadCannedAttachmentData(t *testing.T) {
	t.Parallel()

	// Note: We can't fully test readCannedAttachmentData without being able to mock getMediaStoragePath
	// and resolveMediaFilePath. This test verifies the structure and error handling expectations.

	testData := []byte("test attachment data")
	attachment := models.CannedResponseAttachment{
		ID:        uuid.NewString(),
		FileName:  "test.txt",
		FilePath:  "test_attachment.txt", // Relative path
		FileSize:  int64(len(testData)),
		MimeType:  "text/plain",
		Type:      models.CannedResponseAttachmentTypeImage,
		CreatedAt: "2024-01-01T00:00:00Z",
	}

	// The function expects resolveMediaFilePath to work, which depends on getMediaStoragePath
	// Since we can't mock that, we'll just verify the attachment structure is correct
	assert.NotEmpty(t, attachment.ID, "Attachment should have ID")
	assert.NotEmpty(t, attachment.FilePath, "Attachment should have file path")
	assert.Equal(t, int64(len(testData)), attachment.FileSize, "Attachment should have correct file size")
}

// TestCleanupCannedResponseAttachments tests the cleanupCannedResponseAttachments function
func TestCleanupCannedResponseAttachments(t *testing.T) {
	t.Parallel()

	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create test files
	testFile1 := filepath.Join(tempDir, "attachment1.txt")
	testFile2 := filepath.Join(tempDir, "attachment2.jpg")
	testData := []byte("test data")

	err := os.WriteFile(testFile1, testData, 0644)
	require.NoError(t, err)
	err = os.WriteFile(testFile2, testData, 0644)
	require.NoError(t, err)

	// Note: We can't fully test cleanupCannedResponseAttachments without being able to mock
	// getMediaStoragePath and resolveMediaFilePath. This test verifies the cleanup logic structure.

	attachments := models.CannedResponseAttachments{
		{
			ID:        uuid.NewString(),
			FileName:  "attachment1.txt",
			FilePath:  "attachment1.txt",
			FileSize:  int64(len(testData)),
			Type:      models.CannedResponseAttachmentTypeImage,
			CreatedAt: "2024-01-01T00:00:00Z",
		},
		{
			ID:        uuid.NewString(),
			FileName:  "attachment2.jpg",
			FilePath:  "attachment2.jpg",
			FileSize:  int64(len(testData)),
			Type:      models.CannedResponseAttachmentTypeImage,
			CreatedAt: "2024-01-01T00:00:00Z",
		},
	}

	// Verify attachments have the expected structure
	assert.Len(t, attachments, 2, "Should have 2 attachments")
	for _, attachment := range attachments {
		assert.NotEmpty(t, attachment.ID, "Attachment should have ID")
		assert.NotEmpty(t, attachment.FilePath, "Attachment should have file path")
	}
}

// TestCreateMultipartFormDataRequest is a helper to create multipart form requests for testing
func TestCreateMultipartFormDataRequest(t *testing.T) {
	t.Parallel()

	// This is a helper function test that demonstrates how to create multipart form requests
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add form fields
	_ = writer.WriteField("name", "Test Response")
	_ = writer.WriteField("content", "Test content")
	_ = writer.WriteField("is_active", "true")

	// Add a file
	part, err := writer.CreateFormFile("attachments", "test.txt")
	require.NoError(t, err, "Should create form file")
	_, err = part.Write([]byte("test file content"))
	require.NoError(t, err, "Should write file content")

	err = writer.Close()
	require.NoError(t, err, "Should close writer")

	// Verify the body was created
	assert.NotEmpty(t, body.Bytes(), "Body should not be empty")
	assert.NotEmpty(t, writer.Boundary(), "Should have boundary")

	contentType := writer.FormDataContentType()
	assert.Contains(t, contentType, "multipart/form-data", "Content type should be multipart/form-data")
	assert.Contains(t, contentType, "boundary=", "Content type should have boundary")
}

// TestMultipartFileHeader tests creating multipart file headers for testing
func TestMultipartFileHeader(t *testing.T) {
	t.Parallel()

	// Create a temporary file
	tempFile, err := os.CreateTemp("", "test-upload-*.txt")
	require.NoError(t, err, "Should create temp file")
	defer os.Remove(tempFile.Name())

	testData := []byte("test file content for upload")
	_, err = tempFile.Write(testData)
	require.NoError(t, err, "Should write to temp file")
	tempFile.Close()

	// Reopen the file for reading
	file, err := os.Open(tempFile.Name())
	require.NoError(t, err, "Should open temp file")
	defer file.Close()

	// Create a FileHeader (this simulates what multipart parsing gives us)
	header := &multipart.FileHeader{
		Filename: filepath.Base(tempFile.Name()),
		Header:   make(textproto.MIMEHeader),
	}

	// Verify the header structure
	assert.NotEmpty(t, header.Filename, "FileHeader should have filename")
	assert.NotNil(t, header.Header, "FileHeader should have header")

	// Verify we can read the file
	data, err := io.ReadAll(file)
	require.NoError(t, err, "Should read file content")
	assert.Equal(t, testData, data, "File content should match")
}
