package server

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"media-sorter/internal/storage"
)

// mockCloudProvider implements CloudProvider for testing.
type mockCloudProvider struct {
	id           string
	name         string
	prefix       string
	connected    bool
	hasCreds     bool
	folders      []storage.FolderEntry
	connectURL   string
	connectErr   error
	callbackErr  error
	browseErr    error
	saveCredsErr error
	savedCreds   json.RawMessage
	disconnected bool
}

func (m *mockCloudProvider) ID() string          { return m.id }
func (m *mockCloudProvider) DisplayName() string { return m.name }
func (m *mockCloudProvider) PathPrefix() string  { return m.prefix }
func (m *mockCloudProvider) Connected() bool     { return m.connected }
func (m *mockCloudProvider) HasCredentials() bool { return m.hasCreds }

func (m *mockCloudProvider) SaveCredentials(data json.RawMessage) error {
	if m.saveCredsErr != nil {
		return m.saveCredsErr
	}
	m.savedCreds = data
	return nil
}

func (m *mockCloudProvider) Connect(callbackURL, state string) (string, error) {
	if m.connectErr != nil {
		return "", m.connectErr
	}
	m.connected = true
	return m.connectURL, nil
}

func (m *mockCloudProvider) HandleCallback(code, callbackURL string) error {
	if m.callbackErr != nil {
		return m.callbackErr
	}
	m.connected = true
	return nil
}

func (m *mockCloudProvider) Disconnect() {
	m.connected = false
	m.disconnected = true
}

func (m *mockCloudProvider) BrowseFolders(path string) ([]storage.FolderEntry, error) {
	if m.browseErr != nil {
		return nil, m.browseErr
	}
	return m.folders, nil
}

func (m *mockCloudProvider) StorageProvider() storage.Provider {
	return nil
}

func newTestHandlerWithServer(s *Server) http.Handler {
	staticFS := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html></html>")},
	}
	sub, _ := fs.Sub(staticFS, ".")
	return s.Handler(sub)
}

// --- Provider list tests ---

func TestCloudProvidersReturnsAll(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest("GET", "/api/cloud/providers", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var providers []struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Connected bool   `json:"connected"`
	}
	json.Unmarshal(w.Body.Bytes(), &providers)

	if len(providers) != 4 {
		t.Fatalf("providers count = %d, want 4", len(providers))
	}
	for _, p := range providers {
		if p.Connected {
			t.Errorf("provider %q should not be connected in test", p.ID)
		}
	}
}

func TestCloudProvidersShowsConnected(t *testing.T) {
	s := &Server{
		clouds: []CloudProvider{
			&mockCloudProvider{id: "gdrive", name: "Google Drive", prefix: "gdrive://", connected: true, hasCreds: true},
			&mockCloudProvider{id: "s3", name: "Amazon S3", prefix: "s3://"},
		},
		Port: 9999,
	}
	handler := newTestHandlerWithServer(s)

	req := httptest.NewRequest("GET", "/api/cloud/providers", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var providers []struct {
		ID        string `json:"id"`
		Connected bool   `json:"connected"`
		HasCreds  bool   `json:"hasCreds"`
	}
	json.Unmarshal(w.Body.Bytes(), &providers)

	if !providers[0].Connected || !providers[0].HasCreds {
		t.Errorf("gdrive should be connected with creds, got connected=%v hasCreds=%v",
			providers[0].Connected, providers[0].HasCreds)
	}
	if providers[1].Connected {
		t.Error("s3 should not be connected")
	}
}

// --- Credentials tests ---

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

func TestCloudCredentialsSaveSuccess(t *testing.T) {
	mock := &mockCloudProvider{id: "testprov", name: "Test", prefix: "test://"}
	s := &Server{clouds: []CloudProvider{mock}, Port: 9999}
	handler := newTestHandlerWithServer(s)

	body := `{"provider":"testprov","credentials":{"key":"value"}}`
	req := httptest.NewRequest("POST", "/api/cloud/credentials", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if mock.savedCreds == nil {
		t.Error("credentials should have been saved")
	}
}

func TestCloudCredentialsSaveError(t *testing.T) {
	mock := &mockCloudProvider{
		id: "testprov", name: "Test", prefix: "test://",
		saveCredsErr: fmt.Errorf("validation failed"),
	}
	s := &Server{clouds: []CloudProvider{mock}, Port: 9999}
	handler := newTestHandlerWithServer(s)

	body := `{"provider":"testprov","credentials":{}}`
	req := httptest.NewRequest("POST", "/api/cloud/credentials", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

// --- Connect tests ---

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

func TestCloudConnectOAuthFlow(t *testing.T) {
	mock := &mockCloudProvider{
		id: "testprov", name: "Test", prefix: "test://",
		connectURL: "https://auth.example.com/authorize",
	}
	s := &Server{clouds: []CloudProvider{mock}, Port: 9999}
	handler := newTestHandlerWithServer(s)

	body := `{"provider":"testprov"}`
	req := httptest.NewRequest("POST", "/api/cloud/connect", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["authURL"] != "https://auth.example.com/authorize" {
		t.Errorf("authURL = %q, want auth URL", resp["authURL"])
	}
	if s.oauthProvider != "testprov" {
		t.Errorf("oauthProvider = %q, want 'testprov'", s.oauthProvider)
	}
}

func TestCloudConnectDirectConnect(t *testing.T) {
	mock := &mockCloudProvider{
		id: "testprov", name: "Test", prefix: "test://",
		connectURL: "", // empty = direct connect
	}
	s := &Server{clouds: []CloudProvider{mock}, Port: 9999}
	handler := newTestHandlerWithServer(s)

	body := `{"provider":"testprov"}`
	req := httptest.NewRequest("POST", "/api/cloud/connect", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "connected" {
		t.Errorf("status = %q, want 'connected'", resp["status"])
	}
	if !mock.connected {
		t.Error("provider should be connected after direct connect")
	}
}

func TestCloudConnectError(t *testing.T) {
	mock := &mockCloudProvider{
		id: "testprov", name: "Test", prefix: "test://",
		connectErr: fmt.Errorf("no credentials"),
	}
	s := &Server{clouds: []CloudProvider{mock}, Port: 9999}
	handler := newTestHandlerWithServer(s)

	body := `{"provider":"testprov"}`
	req := httptest.NewRequest("POST", "/api/cloud/connect", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

// --- Callback tests ---

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
	s.oauthProvider = "nonexistent"
	handler := newTestHandlerWithServer(s)

	req := httptest.NewRequest("GET", "/api/cloud/callback?state=test-state&code=abc", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestCloudCallbackSuccess(t *testing.T) {
	mock := &mockCloudProvider{id: "testprov", name: "Test Provider", prefix: "test://"}
	s := &Server{
		clouds:        []CloudProvider{mock},
		Port:          9999,
		oauthState:    "valid-state",
		oauthProvider: "testprov",
	}
	handler := newTestHandlerWithServer(s)

	req := httptest.NewRequest("GET", "/api/cloud/callback?state=valid-state&code=authcode123", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}
	if !mock.connected {
		t.Error("provider should be connected after callback")
	}
	if s.oauthState != "" {
		t.Error("oauthState should be cleared after callback")
	}
	if !strings.Contains(w.Body.String(), "Test Provider connected successfully") {
		t.Errorf("response should contain provider name, got: %s", w.Body.String())
	}
}

func TestCloudCallbackError(t *testing.T) {
	mock := &mockCloudProvider{
		id: "testprov", name: "Test", prefix: "test://",
		callbackErr: fmt.Errorf("token exchange failed"),
	}
	s := &Server{
		clouds:        []CloudProvider{mock},
		Port:          9999,
		oauthState:    "valid-state",
		oauthProvider: "testprov",
	}
	handler := newTestHandlerWithServer(s)

	req := httptest.NewRequest("GET", "/api/cloud/callback?state=valid-state&code=badcode", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 500 {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

// --- Disconnect tests ---

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

func TestCloudDisconnectUnknownProvider(t *testing.T) {
	handler := newTestHandler()
	body := `{"provider":"unknown"}`
	req := httptest.NewRequest("POST", "/api/cloud/disconnect", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestCloudDisconnectSuccess(t *testing.T) {
	mock := &mockCloudProvider{id: "testprov", name: "Test", prefix: "test://", connected: true}
	s := &Server{clouds: []CloudProvider{mock}, Port: 9999}
	handler := newTestHandlerWithServer(s)

	body := `{"provider":"testprov"}`
	req := httptest.NewRequest("POST", "/api/cloud/disconnect", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if mock.connected {
		t.Error("provider should be disconnected")
	}
	if !mock.disconnected {
		t.Error("Disconnect() should have been called")
	}
}

// --- Browse tests ---

func TestCloudBrowseNotConnected(t *testing.T) {
	mock := &mockCloudProvider{id: "testprov", name: "Test Provider", prefix: "test://", connected: false}
	s := &Server{clouds: []CloudProvider{mock}, Port: 9999}
	handler := newTestHandlerWithServer(s)

	req := httptest.NewRequest("GET", "/api/cloud/browse?provider=testprov", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] != "Test Provider not connected" {
		t.Errorf("error = %q, want 'Test Provider not connected'", resp["error"])
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

func TestCloudBrowseSuccess(t *testing.T) {
	mock := &mockCloudProvider{
		id: "testprov", name: "Test", prefix: "test://",
		connected: true,
		folders: []storage.FolderEntry{
			{Name: "Photos", Path: "Photos"},
			{Name: "Videos", Path: "Videos"},
		},
	}
	s := &Server{clouds: []CloudProvider{mock}, Port: 9999}
	handler := newTestHandlerWithServer(s)

	req := httptest.NewRequest("GET", "/api/cloud/browse?provider=testprov", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var folders []storage.FolderEntry
	json.Unmarshal(w.Body.Bytes(), &folders)
	if len(folders) != 2 {
		t.Fatalf("folders count = %d, want 2", len(folders))
	}
	if folders[0].Name != "Photos" || folders[1].Name != "Videos" {
		t.Errorf("folders = %v, want Photos and Videos", folders)
	}
}

func TestCloudBrowseWithPath(t *testing.T) {
	mock := &mockCloudProvider{
		id: "testprov", name: "Test", prefix: "test://",
		connected: true,
		folders: []storage.FolderEntry{
			{Name: "Subfolder", Path: "Photos/Subfolder"},
		},
	}
	s := &Server{clouds: []CloudProvider{mock}, Port: 9999}
	handler := newTestHandlerWithServer(s)

	req := httptest.NewRequest("GET", "/api/cloud/browse?provider=testprov&path=Photos", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var folders []storage.FolderEntry
	json.Unmarshal(w.Body.Bytes(), &folders)
	if len(folders) != 1 || folders[0].Name != "Subfolder" {
		t.Errorf("folders = %v, want [Subfolder]", folders)
	}
}

func TestCloudBrowseError(t *testing.T) {
	mock := &mockCloudProvider{
		id: "testprov", name: "Test", prefix: "test://",
		connected: true,
		browseErr: fmt.Errorf("API error"),
	}
	s := &Server{clouds: []CloudProvider{mock}, Port: 9999}
	handler := newTestHandlerWithServer(s)

	req := httptest.NewRequest("GET", "/api/cloud/browse?provider=testprov", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 500 {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

// --- getProvider routing tests ---

func TestGetProviderRouting(t *testing.T) {
	s := newTestServer()

	// All mock providers return nil StorageProvider, so everything falls back to local
	tests := []struct {
		path    string
		isLocal bool
	}{
		{"/local/path", true},
		{"C:\\Windows\\Path", true},
		{"gdrive://folder", true},   // mock returns nil StorageProvider
		{"s3://bucket", true},       // mock returns nil StorageProvider
		{"dropbox://path", true},    // mock returns nil StorageProvider
		{"onedrive://path", true},   // mock returns nil StorageProvider
	}

	for _, tt := range tests {
		provider := s.getProvider(tt.path)
		if provider.IsLocal() != tt.isLocal {
			t.Errorf("getProvider(%q).IsLocal() = %v, want %v", tt.path, provider.IsLocal(), tt.isLocal)
		}
	}
}
