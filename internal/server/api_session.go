package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
)

func (s *Server) registerSessionAPI(mux *http.ServeMux) {
	sessionPath := ""
	userSettingsPath := ""
	if home, err := os.UserHomeDir(); err == nil {
		sessionPath = filepath.Join(home, ".media-sorter-session.json")
		userSettingsPath = filepath.Join(home, ".media-sorter-settings.json")
	}

	mux.HandleFunc("/api/session", func(w http.ResponseWriter, r *http.Request) {
		if sessionPath == "" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("{}"))
			return
		}
		data, err := os.ReadFile(sessionPath)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("{}"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	})

	mux.HandleFunc("/api/session/save", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			jsonError(w, "POST required", 405)
			return
		}
		if sessionPath == "" {
			jsonError(w, "no home directory", 500)
			return
		}
		var body json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			jsonError(w, err.Error(), 400)
			return
		}
		if err := os.WriteFile(sessionPath, body, 0644); err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		jsonOK(w, "ok")
	})

	mux.HandleFunc("/api/user-settings", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			if userSettingsPath == "" {
				jsonError(w, "no home directory", 500)
				return
			}
			var body json.RawMessage
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				jsonError(w, err.Error(), 400)
				return
			}
			if err := os.WriteFile(userSettingsPath, body, 0644); err != nil {
				jsonError(w, err.Error(), 500)
				return
			}
			jsonOK(w, "ok")
			return
		}
		if userSettingsPath == "" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("{}"))
			return
		}
		data, err := os.ReadFile(userSettingsPath)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("{}"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	})
}
