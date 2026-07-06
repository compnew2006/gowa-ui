package licensestudio

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/license"
	"github.com/compnew2006/whatomate/internal/licenseissuer"
)

func TestIssueEndpointSuccess(t *testing.T) {
	server := newTestServer(t)
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	_, privateKey, err := license.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	status, body := postIssueRequest(t, ts.URL+"/api/issue", privateKey, map[string]string{
		"hwid":         strings.Repeat("a", 64),
		"trial":        "7d",
		"duration":     "365d",
		"tier":         "starter",
		"orgs":         "1",
		"users":        "5",
		"wa_endpoints": "5",
		"workers":      "2",
	})
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", status, body)
	}
	if !strings.Contains(body, `"token"`) {
		t.Fatalf("response missing token: %s", body)
	}

	listResp, err := http.Get(ts.URL + "/api/licenses")
	if err != nil {
		t.Fatalf("GET /api/licenses error = %v", err)
	}
	defer func() { _ = listResp.Body.Close() }()

	var licenses struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&licenses); err != nil {
		t.Fatalf("decode licenses: %v", err)
	}
	if len(licenses.Items) != 1 {
		t.Fatalf("registry items = %d, want 1", len(licenses.Items))
	}
}

func TestIssueEndpointRejectsMissingHWID(t *testing.T) {
	server := newTestServer(t)
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	_, privateKey, err := license.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	status, _ := postIssueRequest(t, ts.URL+"/api/issue", privateKey, map[string]string{
		"trial":        "7d",
		"duration":     "365d",
		"orgs":         "1",
		"users":        "5",
		"wa_endpoints": "5",
		"workers":      "2",
	})
	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", status)
	}
}

func TestIssueEndpointRejectsInvalidDurationWhenTrialProvided(t *testing.T) {
	server := newTestServer(t)
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	_, privateKey, err := license.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	status, body := postIssueRequest(t, ts.URL+"/api/issue", privateKey, map[string]string{
		"hwid":         strings.Repeat("c", 64),
		"trial":        "7d",
		"duration":     "bogus",
		"orgs":         "1",
		"users":        "5",
		"wa_endpoints": "5",
		"workers":      "2",
	})
	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (%s)", status, body)
	}
}

func TestIssueEndpointAcceptsCustomPaidDuration(t *testing.T) {
	server := newTestServer(t)
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	_, privateKey, err := license.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	status, body := postIssueRequest(t, ts.URL+"/api/issue", privateKey, map[string]string{
		"hwid":         strings.Repeat("f", 64),
		"duration":     "55 days",
		"tier":         "starter",
		"orgs":         "1",
		"users":        "5",
		"wa_endpoints": "5",
		"workers":      "2",
	})
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", status, body)
	}
	if !strings.Contains(body, `"duration_preset":"55d"`) {
		t.Fatalf("response missing normalized custom duration: %s", body)
	}
}

func TestVerifyEndpointTrackedAndUntracked(t *testing.T) {
	server := newTestServer(t)
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	_, privateKey, err := license.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	status, body := postIssueRequest(t, ts.URL+"/api/issue", privateKey, map[string]string{
		"hwid":         strings.Repeat("d", 64),
		"trial":        "7d",
		"duration":     "365d",
		"tier":         "starter",
		"orgs":         "1",
		"users":        "5",
		"wa_endpoints": "5",
		"workers":      "2",
	})
	if status != http.StatusOK {
		t.Fatalf("issue status = %d, want 200 (%s)", status, body)
	}

	var issuePayload struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal([]byte(body), &issuePayload); err != nil {
		t.Fatalf("Unmarshal issue response: %v", err)
	}

	trackedStatus := postVerifyRequest(t, ts.URL+"/api/verify", issuePayload.Token)
	if trackedStatus.Status != licenseissuer.StatusValidTracked {
		t.Fatalf("tracked verify status = %s, want %s", trackedStatus.Status, licenseissuer.StatusValidTracked)
	}

	untracked, err := licenseissuer.IssueLicenseFromPrivateKeyText(licenseissuer.IssueOptions{
		HWID:                    strings.Repeat("e", 64),
		Duration:                "365d",
		Tier:                    "starter",
		Organizations:           1,
		UsersPerOrg:             5,
		WhatsAppEndpointsPerOrg: 5,
		Workers:                 2,
	}, privateKey, time.Now().UTC())
	if err != nil {
		t.Fatalf("IssueLicenseFromPrivateKeyText() error = %v", err)
	}

	untrackedStatus := postVerifyRequest(t, ts.URL+"/api/verify", untracked.Token)
	if untrackedStatus.Status != licenseissuer.StatusValidUntracked {
		t.Fatalf("untracked verify status = %s, want %s", untrackedStatus.Status, licenseissuer.StatusValidUntracked)
	}

	invalidStatus := postVerifyRequest(t, ts.URL+"/api/verify", "fake-token")
	if invalidStatus.Status != licenseissuer.StatusInvalid {
		t.Fatalf("invalid verify status = %s, want %s", invalidStatus.Status, licenseissuer.StatusInvalid)
	}
}

func TestLicensesFilterByHWIDAndTier(t *testing.T) {
	server := newTestServer(t)
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	_, privateKey, err := license.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	postIssueRequest(t, ts.URL+"/api/issue", privateKey, map[string]string{
		"hwid":         "alpha-hwid",
		"duration":     "365d",
		"tier":         "starter",
		"orgs":         "1",
		"users":        "5",
		"wa_endpoints": "5",
		"workers":      "2",
	})
	postIssueRequest(t, ts.URL+"/api/issue", privateKey, map[string]string{
		"hwid":         "beta-hwid",
		"duration":     "365d",
		"tier":         "pro",
		"orgs":         "1",
		"users":        "5",
		"wa_endpoints": "5",
		"workers":      "2",
	})

	resp, err := http.Get(ts.URL + "/api/licenses?hwid=beta&tier=pro")
	if err != nil {
		t.Fatalf("GET /api/licenses error = %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var payload struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("filtered items = %d, want 1", len(payload.Items))
	}
}

func TestStudioFrontendContainsTabsAndActions(t *testing.T) {
	server := newTestServer(t)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()

	server.Handler().ServeHTTP(response, request)

	body := response.Body.String()
	for _, want := range []string{"Generate", "Verify", "Registry", "Copy", "55d", "120 days", "lifetime"} {
		if !strings.Contains(body, want) {
			t.Fatalf("frontend missing %q", want)
		}
	}
}

func newTestServer(t *testing.T) *Server {
	t.Helper()

	tempDir := t.TempDir()
	server, err := NewServer(Config{
		Addr:         "127.0.0.1:0",
		DataDir:      tempDir,
		RegistryPath: filepath.Join(tempDir, "registry.json"),
		KeyRingPath:  filepath.Join(tempDir, "keyring.json"),
		OpenBrowser:  false,
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	return server
}

func postIssueRequest(t *testing.T, url string, privateKey string, fields map[string]string) (int, string) {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("WriteField(%s): %v", key, err)
		}
	}
	part, err := writer.CreateFormFile("private_key_file", "private.key")
	if err != nil {
		t.Fatalf("CreateFormFile(): %v", err)
	}
	if _, err := part.Write([]byte(privateKey)); err != nil {
		t.Fatalf("Write(): %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close(): %v", err)
	}

	request, err := http.NewRequest(http.MethodPost, url, &body)
	if err != nil {
		t.Fatalf("NewRequest(): %v", err)
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("Do(): %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("ReadAll(): %v", err)
	}
	return response.StatusCode, string(responseBody)
}

func postVerifyRequest(t *testing.T, url, token string) licenseissuer.VerifyResult {
	t.Helper()

	requestBody, err := json.Marshal(map[string]string{"token": token})
	if err != nil {
		t.Fatalf("Marshal(): %v", err)
	}
	response, err := http.Post(url, "application/json", bytes.NewReader(requestBody))
	if err != nil {
		t.Fatalf("POST verify: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	var result licenseissuer.VerifyResult
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("Decode(): %v", err)
	}
	return result
}
