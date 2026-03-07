package server

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"

	"media-sorter/internal/storage"
)

// Server holds the HTTP server state including cloud provider connections.
type Server struct {
	clouds        []CloudProvider
	oauthState    string
	oauthProvider string
	Port          int
}

// New creates a new Server, restoring any saved cloud provider connections.
func New() *Server {
	return &Server{
		clouds: []CloudProvider{
			newGDriveProvider(),
			newS3Provider(),
			newDropboxProvider(),
			newOneDriveProvider(),
		},
	}
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
	for _, c := range s.clouds {
		if strings.HasPrefix(path, c.PathPrefix()) {
			if sp := c.StorageProvider(); sp != nil {
				return sp
			}
		}
	}
	return &storage.LocalStorage{}
}

// cloudByID returns the cloud provider with the given ID, or nil.
func (s *Server) cloudByID(id string) CloudProvider {
	for _, c := range s.clouds {
		if c.ID() == id {
			return c
		}
	}
	return nil
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
