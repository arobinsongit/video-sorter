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

		videoExts := map[string]bool{".mp4": true, ".mov": true, ".avi": true, ".mkv": true, ".webm": true}
		var videos []string
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			ext := strings.ToLower(filepath.Ext(e.Name()))
			if videoExts[ext] {
				videos = append(videos, e.Name())
			}
		}
		sort.Slice(videos, func(i, j int) bool {
			return strings.ToLower(videos[i]) < strings.ToLower(videos[j])
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

	// Read config
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		dir := r.URL.Query().Get("dir")
		if dir == "" {
			jsonError(w, "dir parameter required", 400)
			return
		}

		configPath := filepath.Join(dir, "video-sorter-config.txt")
		data, err := os.ReadFile(configPath)
		if err != nil {
			// No config yet - return empty
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("{}"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	})

	// Save config
	mux.HandleFunc("/api/config/save", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			jsonError(w, "POST required", 405)
			return
		}

		var req struct {
			Dir     string          `json:"dir"`
			Content json.RawMessage `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error(), 400)
			return
		}

		configPath := filepath.Join(req.Dir, "video-sorter-config.txt")
		if err := os.WriteFile(configPath, req.Content, 0644); err != nil {
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
