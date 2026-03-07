package server

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"

	"video-sorter/internal/storage"
	"video-sorter/internal/storage/dropbox"
	"video-sorter/internal/storage/gdrive"
	"video-sorter/internal/storage/onedrive"
	s3store "video-sorter/internal/storage/s3"
)

// Server holds the HTTP server state including cloud provider connections.
type Server struct {
	GDrive   *gdrive.Storage
	S3       *s3store.Storage
	Dropbox  *dropbox.Storage
	OneDrive *onedrive.Storage

	oauthState    string
	oauthProvider string
	Port          int
}

// New creates a new Server, restoring any saved cloud provider connections.
func New() *Server {
	s := &Server{}

	if gd, err := gdrive.New(); err == nil {
		s.GDrive = gd
	}
	if s3s, err := s3store.New(); err == nil {
		s.S3 = s3s
	}
	if db, err := dropbox.New(); err == nil {
		s.Dropbox = db
	}
	if od, err := onedrive.New(); err == nil {
		s.OneDrive = od
	}

	return s
}

// Handler returns the HTTP handler with all routes registered.
func (s *Server) Handler(staticFS fs.FS) http.Handler {
	mux := http.NewServeMux()

	// Serve embedded frontend
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	// Core API
	s.registerCoreAPI(mux)

	// Cloud API
	s.registerCloudAPI(mux)

	// Session and user settings
	s.registerSessionAPI(mux)

	return mux
}

// getProvider returns the appropriate storage provider for a path.
func (s *Server) getProvider(path string) storage.Provider {
	switch {
	case strings.HasPrefix(path, "gdrive://"):
		if s.GDrive != nil {
			return s.GDrive
		}
	case strings.HasPrefix(path, "s3://"):
		if s.S3 != nil {
			return s.S3
		}
	case strings.HasPrefix(path, "dropbox://"):
		if s.Dropbox != nil {
			return s.Dropbox
		}
	case strings.HasPrefix(path, "onedrive://"):
		if s.OneDrive != nil {
			return s.OneDrive
		}
	}
	return &storage.LocalStorage{}
}

func jsonOK(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
