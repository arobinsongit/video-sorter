package server

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func newTestHandlerWithServer(s *Server) http.Handler {
	staticFS := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html></html>")},
	}
	sub, _ := fs.Sub(staticFS, ".")
	return s.Handler(sub)
}

func TestCloudCredentialsGetNotAllowed(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest("GET", "/api/cloud/credentials", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 405 {
		t.Errorf("status = %d, want 405", w.Code)
	}
}

func TestCloudCredentialsInvalidJSON(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest("POST", "/api/cloud/credentials", strings.NewReader("not json"))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestCloudCredentialsUnknownProvider(t *testing.T) {
	handler := newTestHandler()
	body := `{"provider":"unknown","credentials":{}}`
	req := httptest.NewRequest("POST", "/api/cloud/credentials", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestCloudCredentialsS3MissingFields(t *testing.T) {
	handler := newTestHandler()
	body := `{"provider":"s3","credentials":{"accessKeyId":"key"}}`
	req := httptest.NewRequest("POST", "/api/cloud/credentials", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !strings.Contains(resp["error"], "required") {
		t.Errorf("error = %q, want message about required fields", resp["error"])
	}
}

func TestCloudCredentialsDropboxMissingFields(t *testing.T) {
	handler := newTestHandler()
	body := `{"provider":"dropbox","credentials":{"clientId":"id"}}`
	req := httptest.NewRequest("POST", "/api/cloud/credentials", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestCloudCredentialsOneDriveMissingFields(t *testing.T) {
	handler := newTestHandler()
	body := `{"provider":"onedrive","credentials":{"clientId":"id"}}`
	req := httptest.NewRequest("POST", "/api/cloud/credentials", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestCloudConnectGetNotAllowed(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest("GET", "/api/cloud/connect", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 405 {
		t.Errorf("status = %d, want 405", w.Code)
	}
}

func TestCloudConnectInvalidJSON(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest("POST", "/api/cloud/connect", strings.NewReader("bad"))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestCloudConnectUnknownProvider(t *testing.T) {
	handler := newTestHandler()
	body := `{"provider":"unknown"}`
	req := httptest.NewRequest("POST", "/api/cloud/connect", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestCloudConnectS3NoCreds(t *testing.T) {
	handler := newTestHandler()
	body := `{"provider":"s3"}`
	req := httptest.NewRequest("POST", "/api/cloud/connect", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestCloudConnectDropboxNoCreds(t *testing.T) {
	handler := newTestHandler()
	body := `{"provider":"dropbox"}`
	req := httptest.NewRequest("POST", "/api/cloud/connect", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestCloudConnectOneDriveNoCreds(t *testing.T) {
	handler := newTestHandler()
	body := `{"provider":"onedrive"}`
	req := httptest.NewRequest("POST", "/api/cloud/connect", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestCloudDisconnectGetNotAllowed(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest("GET", "/api/cloud/disconnect", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 405 {
		t.Errorf("status = %d, want 405", w.Code)
	}
}

func TestCloudDisconnectInvalidJSON(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest("POST", "/api/cloud/disconnect", strings.NewReader("bad"))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestCloudDisconnectAllProviders(t *testing.T) {
	for _, provider := range []string{"gdrive", "s3", "dropbox", "onedrive"} {
		handler := newTestHandler()
		body := `{"provider":"` + provider + `"}`
		req := httptest.NewRequest("POST", "/api/cloud/disconnect", strings.NewReader(body))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("disconnect %s: status = %d, want 200", provider, w.Code)
		}
	}
}

func TestCloudCallbackInvalidState(t *testing.T) {
	s := newTestServer()
	s.oauthState = "expected-state"
	handler := newTestHandlerWithServer(s)

	req := httptest.NewRequest("GET", "/api/cloud/callback?state=wrong&code=abc", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestCloudCallbackMissingCode(t *testing.T) {
	s := newTestServer()
	s.oauthState = "test-state"
	handler := newTestHandlerWithServer(s)

	req := httptest.NewRequest("GET", "/api/cloud/callback?state=test-state", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestCloudCallbackUnknownProvider(t *testing.T) {
	s := newTestServer()
	s.oauthState = "test-state"
	s.oauthProvider = "unknown"
	handler := newTestHandlerWithServer(s)

	req := httptest.NewRequest("GET", "/api/cloud/callback?state=test-state&code=abc", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestCloudBrowseNotConnected(t *testing.T) {
	handler := newTestHandler()

	providers := []struct {
		id      string
		wantMsg string
	}{
		{"gdrive", "Google Drive not connected"},
		{"s3", "S3 not connected"},
		{"dropbox", "Dropbox not connected"},
		{"onedrive", "OneDrive not connected"},
	}

	for _, p := range providers {
		req := httptest.NewRequest("GET", "/api/cloud/browse?provider="+p.id, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != 400 {
			t.Errorf("browse %s: status = %d, want 400", p.id, w.Code)
		}
		var resp map[string]string
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["error"] != p.wantMsg {
			t.Errorf("browse %s: error = %q, want %q", p.id, resp["error"], p.wantMsg)
		}
	}
}

func TestCloudCredentialsS3InvalidJSON(t *testing.T) {
	handler := newTestHandler()
	body := `{"provider":"s3","credentials":"not an object"}`
	req := httptest.NewRequest("POST", "/api/cloud/credentials", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestCloudCredentialsDropboxInvalidJSON(t *testing.T) {
	handler := newTestHandler()
	body := `{"provider":"dropbox","credentials":"bad"}`
	req := httptest.NewRequest("POST", "/api/cloud/credentials", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestCloudCredentialsOneDriveInvalidJSON(t *testing.T) {
	handler := newTestHandler()
	body := `{"provider":"onedrive","credentials":"bad"}`
	req := httptest.NewRequest("POST", "/api/cloud/credentials", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestCloudBrowseUnsupportedProvider(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest("GET", "/api/cloud/browse?provider=ftp", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}
