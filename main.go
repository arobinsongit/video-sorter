package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
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

func migrateOrLoadConfig(dir string) (ProjectConfig, error) {
	jsonPath := filepath.Join(dir, "media-sorter-config.json")
	legacyJsonPath := filepath.Join(dir, "video-sorter-config.json")
	txtPath := filepath.Join(dir, "video-sorter-config.txt")

	// 1. Try new JSON config name first
	if data, err := os.ReadFile(jsonPath); err == nil {
		var cfg ProjectConfig
		if err := json.Unmarshal(data, &cfg); err == nil {
			return cfg, nil
		}
	}

	// 2. Try legacy JSON config name (video-sorter-config.json)
	if data, err := os.ReadFile(legacyJsonPath); err == nil {
		var cfg ProjectConfig
		if err := json.Unmarshal(data, &cfg); err == nil {
			// Migrate to new name
			writeProjectConfig(jsonPath, cfg)
			return cfg, nil
		}
	}

	// 3. Try migrating from .txt
	if data, err := os.ReadFile(txtPath); err == nil {
		subjects, tags := parseConfigText(string(data))
		cfg := defaultConfig()
		if len(subjects) > 0 {
			cfg.Groups[0].Options = subjects
		}
		if len(tags) > 0 {
			cfg.Groups[1].Options = tags
		}
		// Write the new JSON config
		writeProjectConfig(jsonPath, cfg)
		return cfg, nil
	}

	// 4. No config exists — return defaults
	cfg := defaultConfig()
	writeProjectConfig(jsonPath, cfg)
	return cfg, nil
}

func writeProjectConfig(path string, cfg ProjectConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

//go:embed all:static
var staticFS embed.FS

func main() {
	mux := http.NewServeMux()

	// Serve embedded frontend (static/index.html, static/app.min.js, static/favicon.svg)
	staticContent, _ := fs.Sub(staticFS, "static")
	mux.Handle("/", http.FileServer(http.FS(staticContent)))

	// List media files in a directory
	mux.HandleFunc("/api/list", func(w http.ResponseWriter, r *http.Request) {
		dir := r.URL.Query().Get("dir")
		if dir == "" {
			jsonError(w, "dir parameter required", 400)
			return
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			jsonError(w, err.Error(), 500)
			return
		}

		type FileInfo struct {
			Name     string `json:"name"`
			Size     int64  `json:"size"`
			Modified string `json:"modified"`
		}

		mediaExts := map[string]bool{
			// Video
			".mp4": true, ".mov": true, ".avi": true, ".mkv": true, ".webm": true,
			// Photo
			".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
			".bmp": true, ".tiff": true, ".tif": true, ".heic": true, ".heif": true,
		}
		var files []FileInfo
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			ext := strings.ToLower(filepath.Ext(e.Name()))
			if mediaExts[ext] {
				info, err := e.Info()
				if err != nil {
					continue
				}
				files = append(files, FileInfo{
					Name:     e.Name(),
					Size:     info.Size(),
					Modified: info.ModTime().Format("2006-01-02 15:04"),
				})
			}
		}
		sort.Slice(files, func(i, j int) bool {
			return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
		})

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

		// Prevent path traversal
		clean := filepath.Clean(file)
		if strings.Contains(clean, string(filepath.Separator)) || strings.Contains(clean, "..") {
			jsonError(w, "invalid filename", 400)
			return
		}

		fullPath := filepath.Join(dir, clean)
		http.ServeFile(w, r, fullPath)
	})

	// Read config (JSON config with migration from .txt)
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		dir := r.URL.Query().Get("dir")
		if dir == "" {
			jsonError(w, "dir parameter required", 400)
			return
		}

		cfg, err := migrateOrLoadConfig(dir)
		if err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		jsonOK(w, cfg)
	})

	// Save config (receives full ProjectConfig JSON)
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

		configPath := filepath.Join(req.Dir, "media-sorter-config.json")
		if err := writeProjectConfig(configPath, req.Config); err != nil {
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

		// Prevent path traversal on filenames
		for _, name := range []string{req.OldName, req.NewName} {
			clean := filepath.Clean(name)
			if strings.Contains(clean, string(filepath.Separator)) || strings.Contains(clean, "..") {
				jsonError(w, "invalid filename", 400)
				return
			}
		}

		// Validate outputFolder if provided
		if req.OutputFolder != "" {
			clean := filepath.Clean(req.OutputFolder)
			if strings.Contains(clean, "..") {
				jsonError(w, "invalid output folder", 400)
				return
			}
		}

		oldPath := filepath.Join(req.Dir, req.OldName)
		if _, err := os.Stat(oldPath); os.IsNotExist(err) {
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
			if !filepath.IsAbs(destDir) {
				destDir = filepath.Join(req.Dir, destDir)
			}
			if err := os.MkdirAll(destDir, 0755); err != nil {
				jsonError(w, "failed to create output folder: "+err.Error(), 500)
				return
			}
			newPath = filepath.Join(destDir, req.NewName)
		}

		if _, err := os.Stat(newPath); err == nil {
			jsonError(w, "target already exists: "+req.NewName, 409)
			return
		}

		switch mode {
		case "rename", "move":
			if err := os.Rename(oldPath, newPath); err != nil {
				// Cross-filesystem move: fall back to copy+delete
				if mode == "move" {
					if err := copyFile(oldPath, newPath); err != nil {
						jsonError(w, err.Error(), 500)
						return
					}
					os.Remove(oldPath)
				} else {
					jsonError(w, err.Error(), 500)
					return
				}
			}
		case "copy":
			if err := copyFile(oldPath, newPath); err != nil {
				jsonError(w, err.Error(), 500)
				return
			}
		default:
			jsonError(w, "invalid outputMode: "+mode, 400)
			return
		}

		jsonOK(w, "ok")
	})

	// Open folder in OS file explorer
	mux.HandleFunc("/api/open-folder", func(w http.ResponseWriter, r *http.Request) {
		dir := r.URL.Query().Get("dir")
		if dir == "" {
			jsonError(w, "dir parameter required", 400)
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

	// Session file: ~/.media-sorter-session.json
	// User settings file: ~/.media-sorter-settings.json
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

	// Load session
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

	// Save session
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

	// Load user settings
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
	port := listener.Addr().(*net.TCPAddr).Port
	url := fmt.Sprintf("http://127.0.0.1:%d", port)

	fmt.Printf("Media Sorter running at %s\n", url)

	// Open browser
	go openBrowser(url)

	// Serve
	http.Serve(listener, mux)
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

func buildConfigText(subjects []string, tags []string) string {
	var b strings.Builder
	b.WriteString("# Subjects\n")
	b.WriteString("# One per line\n")
	for _, p := range subjects {
		b.WriteString(p)
		b.WriteString("\n")
	}
	b.WriteString("\n# Tags\n")
	b.WriteString("# One per line (use dashes instead of spaces, e.g. home-run)\n")
	for _, t := range tags {
		b.WriteString(t)
		b.WriteString("\n")
	}
	return b.String()
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
