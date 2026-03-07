package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"media-sorter/internal/storage/dropbox"
	"media-sorter/internal/storage/gdrive"
	"media-sorter/internal/storage/onedrive"
	s3store "media-sorter/internal/storage/s3"

	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
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

	gdriveHasCreds := gdrive.HasEmbeddedCreds()
	if !gdriveHasCreds {
		_, err := os.Stat(gdrive.ClientCredsPath())
		gdriveHasCreds = err == nil
	}
	_, s3CredsErr := os.Stat(s3store.CredsPath())
	_, dropboxCredsErr := os.Stat(dropbox.CredsPath())
	_, onedriveCredsErr := os.Stat(onedrive.CredsPath())

	providers := []ProviderStatus{
		{ID: "gdrive", Name: "Google Drive", Connected: s.GDrive != nil, HasCreds: gdriveHasCreds},
		{ID: "s3", Name: "Amazon S3", Connected: s.S3 != nil, HasCreds: s3CredsErr == nil},
		{ID: "dropbox", Name: "Dropbox", Connected: s.Dropbox != nil, HasCreds: dropboxCredsErr == nil},
		{ID: "onedrive", Name: "OneDrive", Connected: s.OneDrive != nil, HasCreds: onedriveCredsErr == nil},
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

	switch req.Provider {
	case "gdrive":
		if err := os.WriteFile(gdrive.ClientCredsPath(), req.Creds, 0600); err != nil {
			jsonError(w, "Failed to save credentials: "+err.Error(), 500)
			return
		}
		jsonOK(w, "ok")

	case "s3":
		var creds s3store.Credentials
		if err := json.Unmarshal(req.Creds, &creds); err != nil {
			jsonError(w, "Invalid S3 credentials: "+err.Error(), 400)
			return
		}
		if creds.AccessKeyID == "" || creds.SecretAccessKey == "" || creds.Region == "" {
			jsonError(w, "accessKeyId, secretAccessKey, and region are required", 400)
			return
		}
		data, _ := json.MarshalIndent(creds, "", "  ")
		if err := os.WriteFile(s3store.CredsPath(), data, 0600); err != nil {
			jsonError(w, "Failed to save credentials: "+err.Error(), 500)
			return
		}
		jsonOK(w, "ok")

	case "dropbox":
		var creds dropbox.Credentials
		if err := json.Unmarshal(req.Creds, &creds); err != nil {
			jsonError(w, "Invalid Dropbox credentials: "+err.Error(), 400)
			return
		}
		if creds.ClientID == "" || creds.ClientSecret == "" {
			jsonError(w, "clientId and clientSecret are required", 400)
			return
		}
		data, _ := json.MarshalIndent(creds, "", "  ")
		if err := os.WriteFile(dropbox.CredsPath(), data, 0600); err != nil {
			jsonError(w, "Failed to save credentials: "+err.Error(), 500)
			return
		}
		jsonOK(w, "ok")

	case "onedrive":
		var creds onedrive.Credentials
		if err := json.Unmarshal(req.Creds, &creds); err != nil {
			jsonError(w, "Invalid OneDrive credentials: "+err.Error(), 400)
			return
		}
		if creds.ClientID == "" || creds.ClientSecret == "" {
			jsonError(w, "clientId and clientSecret are required", 400)
			return
		}
		data, _ := json.MarshalIndent(creds, "", "  ")
		if err := os.WriteFile(onedrive.CredsPath(), data, 0600); err != nil {
			jsonError(w, "Failed to save credentials: "+err.Error(), 500)
			return
		}
		jsonOK(w, "ok")

	default:
		jsonError(w, "unknown provider: "+req.Provider, 400)
	}
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

	callbackURL := fmt.Sprintf("http://127.0.0.1:%d/api/cloud/callback", s.Port)
	stateBytes := make([]byte, 16)
	rand.Read(stateBytes)
	s.oauthState = hex.EncodeToString(stateBytes)

	switch req.Provider {
	case "gdrive":
		config, err := gdrive.OAuthConfig()
		if err != nil {
			jsonError(w, err.Error(), 400)
			return
		}
		s.oauthProvider = "gdrive"
		config.RedirectURL = callbackURL
		authURL := config.AuthCodeURL(s.oauthState, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
		jsonOK(w, map[string]string{"authURL": authURL})

	case "s3":
		store, err := s3store.New()
		if err != nil {
			jsonError(w, "S3 credentials not configured. Place your credentials JSON file at: "+s3store.CredsPath(), 400)
			return
		}
		s.S3 = store
		jsonOK(w, map[string]string{"status": "connected"})

	case "dropbox":
		creds, err := dropbox.LoadCreds()
		if err != nil {
			jsonError(w, "Dropbox credentials not configured. Place your credentials JSON file at: "+dropbox.CredsPath(), 400)
			return
		}
		s.oauthProvider = "dropbox"
		authURL := dropbox.AuthURL(creds.ClientID, callbackURL, s.oauthState)
		jsonOK(w, map[string]string{"authURL": authURL})

	case "onedrive":
		creds, err := onedrive.LoadCreds()
		if err != nil {
			jsonError(w, "OneDrive credentials not configured. Place your credentials JSON file at: "+onedrive.CredsPath(), 400)
			return
		}
		s.oauthProvider = "onedrive"
		authURL := onedrive.AuthURL(creds.ClientID, callbackURL, s.oauthState)
		jsonOK(w, map[string]string{"authURL": authURL})

	default:
		jsonError(w, "unknown provider: "+req.Provider, 400)
	}
}

func (s *Server) handleCloudCallback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	if state != s.oauthState {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "No code provided", http.StatusBadRequest)
		return
	}

	callbackURL := fmt.Sprintf("http://127.0.0.1:%d/api/cloud/callback", s.Port)
	providerName := ""

	switch s.oauthProvider {
	case "gdrive":
		providerName = "Google Drive"
		config, err := gdrive.OAuthConfig()
		if err != nil {
			http.Error(w, "Failed to load credentials", http.StatusInternalServerError)
			return
		}
		config.RedirectURL = callbackURL

		token, err := config.Exchange(context.Background(), code)
		if err != nil {
			http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if err := gdrive.SaveToken(token); err != nil {
			http.Error(w, "Failed to save token: "+err.Error(), http.StatusInternalServerError)
			return
		}

		client := config.Client(context.Background(), token)
		srv, err := drive.NewService(context.Background(), option.WithHTTPClient(client))
		if err != nil {
			http.Error(w, "Failed to create Drive service: "+err.Error(), http.StatusInternalServerError)
			return
		}
		s.GDrive = &gdrive.Storage{Service: srv, Token: token}

	case "dropbox":
		providerName = "Dropbox"
		creds, err := dropbox.LoadCreds()
		if err != nil {
			http.Error(w, "Failed to load credentials", http.StatusInternalServerError)
			return
		}
		token, err := dropbox.ExchangeCode(creds.ClientID, creds.ClientSecret, code, callbackURL)
		if err != nil {
			http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if err := dropbox.SaveToken(token); err != nil {
			http.Error(w, "Failed to save token: "+err.Error(), http.StatusInternalServerError)
			return
		}
		s.Dropbox = dropbox.NewFromToken(token.AccessToken)

	case "onedrive":
		providerName = "OneDrive"
		creds, err := onedrive.LoadCreds()
		if err != nil {
			http.Error(w, "Failed to load credentials", http.StatusInternalServerError)
			return
		}
		token, err := onedrive.ExchangeCode(creds.ClientID, creds.ClientSecret, code, callbackURL)
		if err != nil {
			http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if err := onedrive.SaveToken(token); err != nil {
			http.Error(w, "Failed to save token: "+err.Error(), http.StatusInternalServerError)
			return
		}
		s.OneDrive = onedrive.NewFromToken(token.AccessToken)

	default:
		http.Error(w, "Unknown OAuth provider", http.StatusBadRequest)
		return
	}

	s.oauthState = ""
	s.oauthProvider = ""

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(fmt.Sprintf(`<!DOCTYPE html><html><body>
		<h2>%s connected successfully!</h2>
		<p>You can close this tab and return to Media Sorter.</p>
		<script>setTimeout(function(){window.close()},2000)</script>
	</body></html>`, providerName)))
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

	switch req.Provider {
	case "gdrive":
		s.GDrive = nil
		os.Remove(gdrive.TokenPath())
		jsonOK(w, "ok")
	case "s3":
		s.S3 = nil
		jsonOK(w, "ok")
	case "dropbox":
		s.Dropbox = nil
		os.Remove(dropbox.TokenPath())
		jsonOK(w, "ok")
	case "onedrive":
		s.OneDrive = nil
		os.Remove(onedrive.TokenPath())
		jsonOK(w, "ok")
	default:
		jsonError(w, "unknown provider: "+req.Provider, 400)
	}
}

func (s *Server) handleCloudBrowse(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	path := r.URL.Query().Get("path")

	switch provider {
	case "gdrive":
		if s.GDrive == nil {
			jsonError(w, "Google Drive not connected", 400)
			return
		}

		folderID := "root"
		if path != "" && path != "/" {
			var err error
			folderID, err = s.GDrive.ResolveFolder(path)
			if err != nil {
				jsonError(w, err.Error(), 500)
				return
			}
		}

		type FolderEntry struct {
			Name string `json:"name"`
			ID   string `json:"id"`
			Path string `json:"path"`
		}

		q := fmt.Sprintf("'%s' in parents and mimeType='application/vnd.google-apps.folder' and trashed=false", folderID)
		result, err := s.GDrive.Service.Files.List().Q(q).
			Fields("files(id, name)").OrderBy("name").Do()
		if err != nil {
			jsonError(w, err.Error(), 500)
			return
		}

		var folders []FolderEntry
		for _, f := range result.Files {
			entryPath := f.Name
			if path != "" && path != "/" {
				entryPath = strings.TrimRight(path, "/") + "/" + f.Name
			}
			folders = append(folders, FolderEntry{
				Name: f.Name,
				ID:   f.Id,
				Path: entryPath,
			})
		}
		jsonOK(w, folders)

	case "s3":
		if s.S3 == nil {
			jsonError(w, "S3 not connected", 400)
			return
		}
		type FolderEntry struct {
			Name string `json:"name"`
			Path string `json:"path"`
		}
		if path == "" {
			buckets, err := s.S3.ListBuckets()
			if err != nil {
				jsonError(w, err.Error(), 500)
				return
			}
			var folders []FolderEntry
			for _, b := range buckets {
				folders = append(folders, FolderEntry{Name: b, Path: b})
			}
			jsonOK(w, folders)
		} else {
			parts := strings.SplitN(path, "/", 2)
			bucket := parts[0]
			prefix := ""
			if len(parts) > 1 {
				prefix = parts[1]
			}
			subfolders, err := s.S3.ListFolders(bucket, prefix)
			if err != nil {
				jsonError(w, err.Error(), 500)
				return
			}
			var folders []FolderEntry
			for _, f := range subfolders {
				folderPath := bucket + "/"
				if prefix != "" {
					folderPath += prefix + "/"
				}
				folderPath += f
				folders = append(folders, FolderEntry{Name: f, Path: folderPath})
			}
			jsonOK(w, folders)
		}

	case "dropbox":
		if s.Dropbox == nil {
			jsonError(w, "Dropbox not connected", 400)
			return
		}
		dbxPath := ""
		if path != "" && path != "/" {
			dbxPath = path
			if !strings.HasPrefix(dbxPath, "/") {
				dbxPath = "/" + dbxPath
			}
		}
		entries, err := s.Dropbox.ListFolders(dbxPath)
		if err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		type FolderEntry struct {
			Name string `json:"name"`
			Path string `json:"path"`
		}
		var folders []FolderEntry
		for _, e := range entries {
			folders = append(folders, FolderEntry{Name: e.Name, Path: e.Path})
		}
		jsonOK(w, folders)

	case "onedrive":
		if s.OneDrive == nil {
			jsonError(w, "OneDrive not connected", 400)
			return
		}
		odPath := ""
		if path != "" && path != "/" {
			odPath = path
			if !strings.HasPrefix(odPath, "/") {
				odPath = "/" + odPath
			}
		}
		entries, err := s.OneDrive.ListFolders(odPath)
		if err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		type FolderEntry struct {
			Name string `json:"name"`
			Path string `json:"path"`
		}
		var folders []FolderEntry
		for _, e := range entries {
			folders = append(folders, FolderEntry{Name: e.Name, Path: e.Path})
		}
		jsonOK(w, folders)

	default:
		jsonError(w, "provider not supported: "+provider, 400)
	}
}
