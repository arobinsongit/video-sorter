package server

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"media-sorter/internal/storage"
)

func newTestServer() *Server {
	return &Server{
		clouds: []CloudProvider{
			&mockCloudProvider{id: "gdrive", name: "Google Drive", prefix: "gdrive://"},
			&mockCloudProvider{id: "s3", name: "Amazon S3", prefix: "s3://"},
			&mockCloudProvider{id: "dropbox", name: "Dropbox", prefix: "dropbox://"},
			&mockCloudProvider{id: "onedrive", name: "OneDrive", prefix: "onedrive://"},
		},
		Port: 9999,
	}
}

func newTestHandler() http.Handler {
	s := newTestServer()
	staticFS := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html></html>")},
	}
	sub, _ := fs.Sub(staticFS, ".")
	return s.Handler(sub)
}

func TestAPIListMissingDir(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest("GET", "/api/list", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] == "" {
		t.Error("expected error message")
	}
}

func TestAPIListLocalDir(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.mp4"), []byte("video"), 0644)
	os.WriteFile(filepath.Join(dir, "photo.jpg"), []byte("img"), 0644)
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("text"), 0644)

	handler := newTestHandler()
	req := httptest.NewRequest("GET", "/api/list?dir="+dir, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var files []storage.FileInfo
	json.Unmarshal(w.Body.Bytes(), &files)
	if len(files) != 2 {
		t.Errorf("got %d files, want 2 (mp4 + jpg)", len(files))
	}
}

func TestAPIMediaMissingParams(t *testing.T) {
	handler := newTestHandler()

	// Missing both
	req := httptest.NewRequest("GET", "/api/media", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}

	// Missing file
	req = httptest.NewRequest("GET", "/api/media?dir=/tmp", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestAPIMediaServeFile(t *testing.T) {
	dir := t.TempDir()
	content := []byte("fake video content")
	os.WriteFile(filepath.Join(dir, "test.mp4"), content, 0644)

	handler := newTestHandler()
	req := httptest.NewRequest("GET", "/api/media?dir="+dir+"&file=test.mp4", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestAPIConfigRead(t *testing.T) {
	dir := t.TempDir()
	handler := newTestHandler()

	req := httptest.NewRequest("GET", "/api/config?dir="+dir, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var cfg ProjectConfig
	json.Unmarshal(w.Body.Bytes(), &cfg)
	if cfg.Version != 1 {
		t.Errorf("version = %d, want 1", cfg.Version)
	}
	if len(cfg.Groups) != 3 {
		t.Errorf("groups = %d, want 3", len(cfg.Groups))
	}
}

func TestAPIConfigReadMissingDir(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest("GET", "/api/config", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestAPIConfigSave(t *testing.T) {
	dir := t.TempDir()
	handler := newTestHandler()

	cfg := ProjectConfig{
		Version:      1,
		OutputFormat: "{basename}.{ext}",
		OutputMode:   "rename",
		Groups:       []GroupDef{{Name: "Test", Key: "test"}},
	}
	body, _ := json.Marshal(map[string]interface{}{
		"dir":    dir,
		"config": cfg,
	})

	req := httptest.NewRequest("POST", "/api/config/save", strings.NewReader(string(body)))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}

	// Verify it was saved
	data, _ := os.ReadFile(filepath.Join(dir, configFileName))
	var saved ProjectConfig
	json.Unmarshal(data, &saved)
	if saved.OutputFormat != "{basename}.{ext}" {
		t.Errorf("saved format = %q", saved.OutputFormat)
	}
}

func TestAPIConfigSaveGetNotAllowed(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest("GET", "/api/config/save", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 405 {
		t.Errorf("status = %d, want 405", w.Code)
	}
}

func TestAPIRename(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "old.mp4"), []byte("video"), 0644)

	handler := newTestHandler()
	body, _ := json.Marshal(map[string]string{
		"dir":     dir,
		"oldName": "old.mp4",
		"newName": "new.mp4",
	})

	req := httptest.NewRequest("POST", "/api/rename", strings.NewReader(string(body)))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}

	if _, err := os.Stat(filepath.Join(dir, "old.mp4")); !os.IsNotExist(err) {
		t.Error("old file should not exist")
	}
	if _, err := os.Stat(filepath.Join(dir, "new.mp4")); err != nil {
		t.Error("new file should exist")
	}
}

func TestAPIRenameFileNotFound(t *testing.T) {
	dir := t.TempDir()
	handler := newTestHandler()

	body, _ := json.Marshal(map[string]string{
		"dir":     dir,
		"oldName": "nonexistent.mp4",
		"newName": "new.mp4",
	})

	req := httptest.NewRequest("POST", "/api/rename", strings.NewReader(string(body)))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestAPIRenameTargetExists(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "old.mp4"), []byte("old"), 0644)
	os.WriteFile(filepath.Join(dir, "new.mp4"), []byte("new"), 0644)

	handler := newTestHandler()
	body, _ := json.Marshal(map[string]string{
		"dir":     dir,
		"oldName": "old.mp4",
		"newName": "new.mp4",
	})

	req := httptest.NewRequest("POST", "/api/rename", strings.NewReader(string(body)))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 409 {
		t.Errorf("status = %d, want 409", w.Code)
	}
}

func TestAPIRenamePathTraversal(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.mp4"), []byte("video"), 0644)

	handler := newTestHandler()
	body, _ := json.Marshal(map[string]string{
		"dir":     dir,
		"oldName": "../../../etc/passwd",
		"newName": "hacked.mp4",
	})

	req := httptest.NewRequest("POST", "/api/rename", strings.NewReader(string(body)))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400 for path traversal", w.Code)
	}
}

func TestAPIRenameCopy(t *testing.T) {
	dir := t.TempDir()
	content := []byte("video content")
	os.WriteFile(filepath.Join(dir, "original.mp4"), content, 0644)

	handler := newTestHandler()
	body, _ := json.Marshal(map[string]interface{}{
		"dir":        dir,
		"oldName":    "original.mp4",
		"newName":    "copy.mp4",
		"outputMode": "copy",
	})

	req := httptest.NewRequest("POST", "/api/rename", strings.NewReader(string(body)))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}

	// Both files should exist
	if _, err := os.Stat(filepath.Join(dir, "original.mp4")); err != nil {
		t.Error("original should still exist after copy")
	}
	if _, err := os.Stat(filepath.Join(dir, "copy.mp4")); err != nil {
		t.Error("copy should exist")
	}
}

func TestAPIRenameMove(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "output")
	os.WriteFile(filepath.Join(dir, "source.mp4"), []byte("data"), 0644)

	handler := newTestHandler()
	body, _ := json.Marshal(map[string]interface{}{
		"dir":          dir,
		"oldName":      "source.mp4",
		"newName":      "moved.mp4",
		"outputMode":   "move",
		"outputFolder": outDir,
	})

	req := httptest.NewRequest("POST", "/api/rename", strings.NewReader(string(body)))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}

	if _, err := os.Stat(filepath.Join(dir, "source.mp4")); !os.IsNotExist(err) {
		t.Error("source should not exist after move")
	}
	if _, err := os.Stat(filepath.Join(outDir, "moved.mp4")); err != nil {
		t.Error("moved file should exist in output folder")
	}
}

func TestAPIRenameInvalidMode(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.mp4"), []byte("data"), 0644)

	handler := newTestHandler()
	body, _ := json.Marshal(map[string]interface{}{
		"dir":        dir,
		"oldName":    "test.mp4",
		"newName":    "out.mp4",
		"outputMode": "invalid",
	})

	req := httptest.NewRequest("POST", "/api/rename", strings.NewReader(string(body)))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestAPIRenameGetNotAllowed(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest("GET", "/api/rename", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 405 {
		t.Errorf("status = %d, want 405", w.Code)
	}
}

func TestAPISessionRoundtrip(t *testing.T) {
	handler := newTestHandler()

	// Load empty session
	req := httptest.NewRequest("GET", "/api/session", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("GET /api/session status = %d", w.Code)
	}

	// Save session
	sessionData := `{"dir":"/test","file":"clip.mp4"}`
	req = httptest.NewRequest("POST", "/api/session/save", strings.NewReader(sessionData))
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("POST /api/session/save status = %d, body: %s", w.Code, w.Body.String())
	}

	// Load it back
	req = httptest.NewRequest("GET", "/api/session", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("GET /api/session status = %d", w.Code)
	}

	var session map[string]string
	json.Unmarshal(w.Body.Bytes(), &session)
	if session["dir"] != "/test" {
		t.Errorf("session dir = %q, want '/test'", session["dir"])
	}
}

func TestAPIUserSettingsRoundtrip(t *testing.T) {
	handler := newTestHandler()

	// Load empty settings
	req := httptest.NewRequest("GET", "/api/user-settings", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("GET /api/user-settings status = %d", w.Code)
	}

	// Save settings
	settings := `{"theme":"dark"}`
	req = httptest.NewRequest("POST", "/api/user-settings", strings.NewReader(settings))
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("POST /api/user-settings status = %d, body: %s", w.Code, w.Body.String())
	}
}

func TestAPIListInvalidDir(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest("GET", "/api/list?dir=/nonexistent/path/12345", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestAPIListPathTraversal(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest("GET", "/api/list?dir=../../etc", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestAPIMediaInvalidDir(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest("GET", "/api/media?dir=/nonexistent/path/12345&file=test.mp4", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestAPIConfigReadInvalidDir(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest("GET", "/api/config?dir=/nonexistent/path/12345", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestOpenFolderMissingDir(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest("GET", "/api/open-folder", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestOpenFolderInvalidDir(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest("GET", "/api/open-folder?dir=/nonexistent/path/12345", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}
