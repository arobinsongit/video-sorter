package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
)

func (s *Server) registerCloudAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/cloud/providers", s.handleCloudProviders)
	mux.HandleFunc("/api/cloud/credentials", s.handleCloudCredentials)
	mux.HandleFunc("/api/cloud/connect", s.handleCloudConnect)
	mux.HandleFunc("/api/cloud/callback", s.handleCloudCallback)
	mux.HandleFunc("/api/cloud/disconnect", s.handleCloudDisconnect)
	mux.HandleFunc("/api/cloud/browse", s.handleCloudBrowse)
}

func (s *Server) handleCloudProviders(w http.ResponseWriter, r *http.Request) {
	type ProviderStatus struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Connected bool   `json:"connected"`
		HasCreds  bool   `json:"hasCreds"`
	}

	providers := make([]ProviderStatus, 0, len(s.clouds))
	for _, c := range s.clouds {
		providers = append(providers, ProviderStatus{
			ID:        c.ID(),
			Name:      c.DisplayName(),
			Connected: c.Connected(),
			HasCreds:  c.HasCredentials(),
		})
	}
	jsonOK(w, providers)
}

func (s *Server) handleCloudCredentials(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonError(w, "POST required", 405)
		return
	}

	var req struct {
		Provider string          `json:"provider"`
		Creds    json.RawMessage `json:"credentials"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, err.Error(), 400)
		return
	}

	cloud := s.cloudByID(req.Provider)
	if cloud == nil {
		jsonError(w, "unknown provider: "+req.Provider, 400)
		return
	}

	if err := cloud.SaveCredentials(req.Creds); err != nil {
		jsonError(w, err.Error(), 400)
		return
	}
	jsonOK(w, "ok")
}

func (s *Server) handleCloudConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonError(w, "POST required", 405)
		return
	}

	var req struct {
		Provider string `json:"provider"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, err.Error(), 400)
		return
	}

	cloud := s.cloudByID(req.Provider)
	if cloud == nil {
		jsonError(w, "unknown provider: "+req.Provider, 400)
		return
	}

	callbackURL := fmt.Sprintf("http://127.0.0.1:%d/api/cloud/callback", s.Port)
	stateBytes := make([]byte, 16)
	_, _ = rand.Read(stateBytes)
	state := hex.EncodeToString(stateBytes)

	authURL, err := cloud.Connect(callbackURL, state)
	if err != nil {
		jsonError(w, err.Error(), 400)
		return
	}

	if authURL == "" {
		// Direct connect (e.g., S3)
		jsonOK(w, map[string]string{"status": "connected"})
	} else {
		s.mu.Lock()
		s.oauthState = state
		s.oauthProvider = req.Provider
		s.mu.Unlock()
		jsonOK(w, map[string]string{"authURL": authURL})
	}
}

func (s *Server) handleCloudCallback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")

	s.mu.Lock()
	expectedState := s.oauthState
	providerID := s.oauthProvider
	s.mu.Unlock()

	if state != expectedState {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}
	if code == "" {
		http.Error(w, "No code provided", http.StatusBadRequest)
		return
	}

	cloud := s.cloudByID(providerID)
	if cloud == nil {
		http.Error(w, "Unknown OAuth provider", http.StatusBadRequest)
		return
	}

	callbackURL := fmt.Sprintf("http://127.0.0.1:%d/api/cloud/callback", s.Port)
	if err := cloud.HandleCallback(code, callbackURL); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.mu.Lock()
	s.oauthState = ""
	s.oauthProvider = ""
	s.mu.Unlock()

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(fmt.Sprintf(`<!DOCTYPE html><html><body>
		<h2>%s connected successfully!</h2>
		<p>You can close this tab and return to Media Sorter.</p>
		<script>setTimeout(function(){window.close()},2000)</script>
	</body></html>`, cloud.DisplayName())))
}

func (s *Server) handleCloudDisconnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonError(w, "POST required", 405)
		return
	}

	var req struct {
		Provider string `json:"provider"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, err.Error(), 400)
		return
	}

	cloud := s.cloudByID(req.Provider)
	if cloud == nil {
		jsonError(w, "unknown provider: "+req.Provider, 400)
		return
	}

	cloud.Disconnect()
	jsonOK(w, "ok")
}

func (s *Server) handleCloudBrowse(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	path := r.URL.Query().Get("path")

	cloud := s.cloudByID(provider)
	if cloud == nil {
		jsonError(w, "provider not supported: "+provider, 400)
		return
	}

	if !cloud.Connected() {
		jsonError(w, cloud.DisplayName()+" not connected", 400)
		return
	}

	folders, err := cloud.BrowseFolders(path)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonOK(w, folders)
}
