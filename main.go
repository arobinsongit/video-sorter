package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// GroupDef defines a metadata group (e.g. Subject, Tags, Quality)
type GroupDef struct {
	Name        string   `json:"name"`
	Key         string   `json:"key"`
	Type        string   `json:"type"`        // "multi-select" or "single-select"
	InputType   string   `json:"inputType"`   // "number", "text", or "slider"
	Options     []string `json:"options"`
	AllowCustom bool     `json:"allowCustom"`
	Separator   string   `json:"separator"`
	Prefix      string   `json:"prefix"`
}

// ProjectConfig is the JSON config stored per media folder
type ProjectConfig struct {
	Version      int               `json:"version"`
	OutputFormat string            `json:"outputFormat"`
	OutputFolder string            `json:"outputFolder"`
	OutputMode   string            `json:"outputMode"`
	Groups       []GroupDef        `json:"groups"`
	Keybindings  map[string]string `json:"keybindings,omitempty"`
}

func defaultConfig() ProjectConfig {
	return ProjectConfig{
		Version:      1,
		OutputFormat: "{basename}__{S}__{tags}__{quality}.{ext}",
		OutputFolder: "",
		OutputMode:   "rename",
		Groups: []GroupDef{
			{
				Name:        "Subject",
				Key:         "S",
				Type:        "multi-select",
				InputType:   "number",
				Options:     []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12", "13", "14", "15", "16", "17", "18", "19", "20", "21", "22", "23", "24", "25"},
				AllowCustom: true,
				Separator:   "__",
				Prefix:      "S",
			},
			{
				Name:        "Tags",
				Key:         "tags",
				Type:        "multi-select",
				InputType:   "text",
				Options:     []string{"single", "double", "triple", "home-run", "strikeout", "walk", "steal", "catch", "dive", "throw", "slide", "bunt", "sac-fly", "error", "celebration", "pitching", "hitting", "fielding", "running", "warmup"},
				AllowCustom: true,
				Separator:   "_",
				Prefix:      "",
			},
			{
				Name:        "Quality",
				Key:         "quality",
				Type:        "single-select",
				InputType:   "slider",
				Options:     []string{"bad", "ok", "good", "great"},
				AllowCustom: false,
				Separator:   "",
				Prefix:      "",
			},
		},
	}
}

const configFileName = "media-sorter-config.json"

func loadConfig(store StorageProvider, dir string) (ProjectConfig, error) {
	configPath := filepath.Join(dir, configFileName)

	// 1. Try current config name
	if data, err := store.ReadFile(configPath); err == nil {
		var cfg ProjectConfig
		if err := json.Unmarshal(data, &cfg); err == nil {
			return cfg, nil
		}
	}

	// 2. Try legacy config names (local only)
	if store.IsLocal() {
		legacyJsonPath := filepath.Join(dir, "video-sorter-config.json")
		if data, err := store.ReadFile(legacyJsonPath); err == nil {
			var cfg ProjectConfig
			if err := json.Unmarshal(data, &cfg); err == nil {
				saveConfig(store, dir, cfg)
				return cfg, nil
			}
		}

		txtPath := filepath.Join(dir, "video-sorter-config.txt")
		if data, err := store.ReadFile(txtPath); err == nil {
			subjects, tags := parseConfigText(string(data))
			cfg := defaultConfig()
			if len(subjects) > 0 {
				cfg.Groups[0].Options = subjects
			}
			if len(tags) > 0 {
				cfg.Groups[1].Options = tags
			}
			saveConfig(store, dir, cfg)
			return cfg, nil
		}
	}

	// 3. No config — return defaults and save
	cfg := defaultConfig()
	saveConfig(store, dir, cfg)
	return cfg, nil
}

func saveConfig(store StorageProvider, dir string, cfg ProjectConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return store.WriteFile(filepath.Join(dir, configFileName), data)
}

//go:embed all:static
var staticFS embed.FS

func main() {
	mux := http.NewServeMux()

	// Try to restore Google Drive connection
	if gd, err := newGoogleDriveStorage(); err == nil {
		gdrive = gd
	}

	// Serve embedded frontend
	staticContent, _ := fs.Sub(staticFS, "static")
	mux.Handle("/", http.FileServer(http.FS(staticContent)))

	// List media files
	mux.HandleFunc("/api/list", func(w http.ResponseWriter, r *http.Request) {
		dir := r.URL.Query().Get("dir")
		if dir == "" {
			jsonError(w, "dir parameter required", 400)
			return
		}
		store := getStorageProvider(dir)
		files, err := store.ListFiles(dir)
		if err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		jsonOK(w, files)
	})

	// Serve media files
	mux.HandleFunc("/api/media", func(w http.ResponseWriter, r *http.Request) {
		dir := r.URL.Query().Get("dir")
		file := r.URL.Query().Get("file")
		if dir == "" || file == "" {
			jsonError(w, "dir and file parameters required", 400)
			return
		}
		store := getStorageProvider(dir)
		store.ServeFile(w, r, dir, file)
	})

	// Read config
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		dir := r.URL.Query().Get("dir")
		if dir == "" {
			jsonError(w, "dir parameter required", 400)
			return
		}
		store := getStorageProvider(dir)
		cfg, err := loadConfig(store, dir)
		if err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		jsonOK(w, cfg)
	})

	// Save config
	mux.HandleFunc("/api/config/save", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			jsonError(w, "POST required", 405)
			return
		}
		var req struct {
			Dir    string        `json:"dir"`
			Config ProjectConfig `json:"config"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error(), 400)
			return
		}
		store := getStorageProvider(req.Dir)
		if err := saveConfig(store, req.Dir, req.Config); err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		jsonOK(w, "ok")
	})

	// Rename/move/copy file
	mux.HandleFunc("/api/rename", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			jsonError(w, "POST required", 405)
			return
		}

		var req struct {
			Dir          string `json:"dir"`
			OldName      string `json:"oldName"`
			NewName      string `json:"newName"`
			OutputMode   string `json:"outputMode"`
			OutputFolder string `json:"outputFolder"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error(), 400)
			return
		}

		store := getStorageProvider(req.Dir)

		// Validate filenames (local only — cloud providers handle their own paths)
		if store.IsLocal() {
			for _, name := range []string{req.OldName, req.NewName} {
				clean := filepath.Clean(name)
				if strings.Contains(clean, string(filepath.Separator)) || strings.Contains(clean, "..") {
					jsonError(w, "invalid filename", 400)
					return
				}
			}
			if req.OutputFolder != "" {
				clean := filepath.Clean(req.OutputFolder)
				if strings.Contains(clean, "..") {
					jsonError(w, "invalid output folder", 400)
					return
				}
			}
		}

		oldPath := filepath.Join(req.Dir, req.OldName)
		if !store.FileExists(oldPath) {
			jsonError(w, "file not found: "+req.OldName, 404)
			return
		}

		mode := req.OutputMode
		if mode == "" {
			mode = "rename"
		}

		var newPath string
		if mode == "rename" || req.OutputFolder == "" {
			newPath = filepath.Join(req.Dir, req.NewName)
		} else {
			destDir := req.OutputFolder
			if store.IsLocal() && !filepath.IsAbs(destDir) {
				destDir = filepath.Join(req.Dir, destDir)
			}
			if err := store.MkdirAll(destDir); err != nil {
				jsonError(w, "failed to create output folder: "+err.Error(), 500)
				return
			}
			newPath = filepath.Join(destDir, req.NewName)
		}

		if store.FileExists(newPath) {
			jsonError(w, "target already exists: "+req.NewName, 409)
			return
		}

		var err error
		switch mode {
		case "rename":
			err = store.Rename(req.Dir, req.OldName, req.NewName)
		case "move":
			err = store.MoveFile(oldPath, newPath)
		case "copy":
			err = store.CopyFile(oldPath, newPath)
		default:
			jsonError(w, "invalid outputMode: "+mode, 400)
			return
		}
		if err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		jsonOK(w, "ok")
	})

	// Open folder in OS file explorer (local only)
	mux.HandleFunc("/api/open-folder", func(w http.ResponseWriter, r *http.Request) {
		dir := r.URL.Query().Get("dir")
		if dir == "" {
			jsonError(w, "dir parameter required", 400)
			return
		}
		store := getStorageProvider(dir)
		if !store.IsLocal() {
			jsonError(w, "cannot open cloud folders in file explorer", 400)
			return
		}
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			jsonError(w, "not a valid directory", 400)
			return
		}
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "windows":
			cmd = exec.Command("explorer", dir)
		case "darwin":
			cmd = exec.Command("open", dir)
		default:
			cmd = exec.Command("xdg-open", dir)
		}
		cmd.Start()
		jsonOK(w, "ok")
	})

	// --- Cloud provider endpoints ---

	// List cloud providers and connection status
	mux.HandleFunc("/api/cloud/providers", func(w http.ResponseWriter, r *http.Request) {
		type ProviderStatus struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			Connected bool   `json:"connected"`
			HasCreds  bool   `json:"hasCreds"`
		}

		_, credsErr := os.Stat(gdriveClientCredsPath())
		providers := []ProviderStatus{
			{ID: "gdrive", Name: "Google Drive", Connected: gdrive != nil, HasCreds: credsErr == nil},
			{ID: "s3", Name: "Amazon S3", Connected: false, HasCreds: false},
			{ID: "dropbox", Name: "Dropbox", Connected: false, HasCreds: false},
			{ID: "onedrive", Name: "OneDrive", Connected: false, HasCreds: false},
		}
		jsonOK(w, providers)
	})

	// Store for OAuth state parameter
	var oauthState string

	// Initiate OAuth flow or save credentials
	mux.HandleFunc("/api/cloud/connect", func(w http.ResponseWriter, r *http.Request) {
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
			config, err := gdriveOAuthConfig(gdriveClientCredsPath())
			if err != nil {
				jsonError(w, "Google Drive credentials not configured. Place your OAuth client credentials JSON file at: "+gdriveClientCredsPath(), 400)
				return
			}
			// Find the server's port for the callback URL
			oauthState = fmt.Sprintf("media-sorter-%d", time.Now().UnixNano())
			config.RedirectURL = fmt.Sprintf("http://127.0.0.1:%d/api/cloud/callback", serverPort)
			authURL := config.AuthCodeURL(oauthState, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
			jsonOK(w, map[string]string{"authURL": authURL})
		default:
			jsonError(w, "provider not yet supported: "+req.Provider, 400)
		}
	})

	// OAuth callback handler
	mux.HandleFunc("/api/cloud/callback", func(w http.ResponseWriter, r *http.Request) {
		state := r.URL.Query().Get("state")
		if state != oauthState {
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "No code provided", http.StatusBadRequest)
			return
		}

		config, err := gdriveOAuthConfig(gdriveClientCredsPath())
		if err != nil {
			http.Error(w, "Failed to load credentials", http.StatusInternalServerError)
			return
		}
		config.RedirectURL = fmt.Sprintf("http://127.0.0.1:%d/api/cloud/callback", serverPort)

		token, err := config.Exchange(context.Background(), code)
		if err != nil {
			http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if err := saveGdriveToken(token); err != nil {
			http.Error(w, "Failed to save token: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Create the Drive service
		client := config.Client(context.Background(), token)
		srv, err := drive.NewService(context.Background(), option.WithHTTPClient(client))
		if err != nil {
			http.Error(w, "Failed to create Drive service: "+err.Error(), http.StatusInternalServerError)
			return
		}

		gdrive = &GoogleDriveStorage{service: srv, token: token}
		oauthState = ""

		// Show success page that closes itself
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html><html><body>
			<h2>Google Drive connected successfully!</h2>
			<p>You can close this tab and return to Media Sorter.</p>
			<script>setTimeout(function(){window.close()},2000)</script>
		</body></html>`))
	})

	// Disconnect a cloud provider
	mux.HandleFunc("/api/cloud/disconnect", func(w http.ResponseWriter, r *http.Request) {
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
			gdrive = nil
			os.Remove(gdriveTokenPath())
			jsonOK(w, "ok")
		default:
			jsonError(w, "unknown provider: "+req.Provider, 400)
		}
	})

	// Browse cloud folder structure
	mux.HandleFunc("/api/cloud/browse", func(w http.ResponseWriter, r *http.Request) {
		provider := r.URL.Query().Get("provider")
		path := r.URL.Query().Get("path")

		switch provider {
		case "gdrive":
			if gdrive == nil {
				jsonError(w, "Google Drive not connected", 400)
				return
			}

			folderID := "root"
			if path != "" && path != "/" {
				var err error
				folderID, err = gdrive.resolveFolder(path)
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
			result, err := gdrive.service.Files.List().Q(q).
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

		default:
			jsonError(w, "provider not supported: "+provider, 400)
		}
	})

	// Session and user settings (local only — unchanged)
	sessionPath := ""
	userSettingsPath := ""
	if home, err := os.UserHomeDir(); err == nil {
		sessionPath = filepath.Join(home, ".media-sorter-session.json")
		userSettingsPath = filepath.Join(home, ".media-sorter-settings.json")
		// Migrate legacy filenames
		legacySession := filepath.Join(home, ".video-sorter-session.json")
		legacySettings := filepath.Join(home, ".video-sorter-settings.json")
		if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
			if _, err := os.Stat(legacySession); err == nil {
				os.Rename(legacySession, sessionPath)
			}
		}
		if _, err := os.Stat(userSettingsPath); os.IsNotExist(err) {
			if _, err := os.Stat(legacySettings); err == nil {
				os.Rename(legacySettings, userSettingsPath)
			}
		}
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

	// Find a free port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to find free port: %v\n", err)
		os.Exit(1)
	}
	serverPort = listener.Addr().(*net.TCPAddr).Port
	url := fmt.Sprintf("http://127.0.0.1:%d", serverPort)

	fmt.Printf("Media Sorter running at %s\n", url)

	go openBrowser(url)

	http.Serve(listener, mux)
}

// serverPort is stored globally so OAuth callback URL can reference it.
var serverPort int

func jsonOK(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func parseConfigText(text string) (subjects []string, tags []string) {
	section := ""
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "# subjects") || strings.HasPrefix(lower, "# players") {
			section = "subjects"
			continue
		}
		if strings.HasPrefix(lower, "# tags") {
			section = "tags"
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		switch section {
		case "subjects":
			subjects = append(subjects, line)
		case "tags":
			tags = append(tags, line)
		}
	}
	return
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	cmd.Run()
}
