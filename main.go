package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

//go:embed index.html
var indexHTML embed.FS

func main() {
	mux := http.NewServeMux()

	// Serve the embedded HTML
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data, _ := indexHTML.ReadFile("index.html")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	})

	// List video files in a directory
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

		type VideoInfo struct {
			Name     string `json:"name"`
			Size     int64  `json:"size"`
			Modified string `json:"modified"`
		}

		videoExts := map[string]bool{".mp4": true, ".mov": true, ".avi": true, ".mkv": true, ".webm": true}
		var videos []VideoInfo
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			ext := strings.ToLower(filepath.Ext(e.Name()))
			if videoExts[ext] {
				info, err := e.Info()
				if err != nil {
					continue
				}
				videos = append(videos, VideoInfo{
					Name:     e.Name(),
					Size:     info.Size(),
					Modified: info.ModTime().Format("2006-01-02 15:04"),
				})
			}
		}
		sort.Slice(videos, func(i, j int) bool {
			return strings.ToLower(videos[i].Name) < strings.ToLower(videos[j].Name)
		})

		jsonOK(w, videos)
	})

	// Serve video files
	mux.HandleFunc("/api/video", func(w http.ResponseWriter, r *http.Request) {
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

	// Read config (parses text file, returns JSON to frontend)
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		dir := r.URL.Query().Get("dir")
		if dir == "" {
			jsonError(w, "dir parameter required", 400)
			return
		}

		configPath := filepath.Join(dir, "video-sorter-config.txt")
		data, err := os.ReadFile(configPath)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("{}"))
			return
		}
		subjects, tags := parseConfigText(string(data))
		result := map[string][]string{}
		if len(subjects) > 0 {
			result["subjects"] = subjects
		}
		if len(tags) > 0 {
			result["tags"] = tags
		}
		jsonOK(w, result)
	})

	// Save config (receives JSON from frontend, writes text file)
	mux.HandleFunc("/api/config/save", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			jsonError(w, "POST required", 405)
			return
		}

		var req struct {
			Dir     string   `json:"dir"`
			Subjects []string `json:"subjects"`
			Tags    []string `json:"tags"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error(), 400)
			return
		}

		configPath := filepath.Join(req.Dir, "video-sorter-config.txt")
		text := buildConfigText(req.Subjects, req.Tags)
		if err := os.WriteFile(configPath, []byte(text), 0644); err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		jsonOK(w, "ok")
	})

	// Rename video
	mux.HandleFunc("/api/rename", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			jsonError(w, "POST required", 405)
			return
		}

		var req struct {
			Dir     string `json:"dir"`
			OldName string `json:"oldName"`
			NewName string `json:"newName"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error(), 400)
			return
		}

		// Prevent path traversal
		for _, name := range []string{req.OldName, req.NewName} {
			clean := filepath.Clean(name)
			if strings.Contains(clean, string(filepath.Separator)) || strings.Contains(clean, "..") {
				jsonError(w, "invalid filename", 400)
				return
			}
		}

		oldPath := filepath.Join(req.Dir, req.OldName)
		newPath := filepath.Join(req.Dir, req.NewName)

		if _, err := os.Stat(oldPath); os.IsNotExist(err) {
			jsonError(w, "file not found: "+req.OldName, 404)
			return
		}
		if _, err := os.Stat(newPath); err == nil {
			jsonError(w, "target already exists: "+req.NewName, 409)
			return
		}

		if err := os.Rename(oldPath, newPath); err != nil {
			jsonError(w, err.Error(), 500)
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

	// Session file: ~/.video-sorter-session.json
	sessionPath := ""
	if home, err := os.UserHomeDir(); err == nil {
		sessionPath = filepath.Join(home, ".video-sorter-session.json")
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

	// Find a free port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to find free port: %v\n", err)
		os.Exit(1)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	url := fmt.Sprintf("http://127.0.0.1:%d", port)

	fmt.Printf("Video Sorter running at %s\n", url)

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
