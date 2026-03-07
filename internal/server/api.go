package server

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"media-sorter/internal/storage"
)

func (s *Server) registerCoreAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/list", s.handleList)
	mux.HandleFunc("/api/media", s.handleMedia)
	mux.HandleFunc("/api/config", s.handleConfigRead)
	mux.HandleFunc("/api/config/save", s.handleConfigSave)
	mux.HandleFunc("/api/rename", s.handleRename)
	mux.HandleFunc("/api/open-folder", s.handleOpenFolder)
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	dir := r.URL.Query().Get("dir")
	if dir == "" {
		jsonError(w, "dir parameter required", 400)
		return
	}
	store := s.getProvider(dir)
	files, err := store.ListFiles(dir)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonOK(w, files)
}

func (s *Server) handleMedia(w http.ResponseWriter, r *http.Request) {
	dir := r.URL.Query().Get("dir")
	file := r.URL.Query().Get("file")
	if dir == "" || file == "" {
		jsonError(w, "dir and file parameters required", 400)
		return
	}
	store := s.getProvider(dir)
	store.ServeFile(w, r, dir, file)
}

func (s *Server) handleConfigRead(w http.ResponseWriter, r *http.Request) {
	dir := r.URL.Query().Get("dir")
	if dir == "" {
		jsonError(w, "dir parameter required", 400)
		return
	}
	store := s.getProvider(dir)
	cfg, err := loadConfig(store, dir)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonOK(w, cfg)
}

func (s *Server) handleConfigSave(w http.ResponseWriter, r *http.Request) {
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
	store := s.getProvider(req.Dir)
	if err := saveConfig(store, req.Dir, req.Config); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonOK(w, "ok")
}

func (s *Server) handleRename(w http.ResponseWriter, r *http.Request) {
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

	store := s.getProvider(req.Dir)

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

	joinPath := filepath.Join
	if !store.IsLocal() {
		joinPath = storage.CloudJoin
	}

	oldPath := joinPath(req.Dir, req.OldName)
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
		newPath = joinPath(req.Dir, req.NewName)
	} else {
		destDir := req.OutputFolder
		if store.IsLocal() && !filepath.IsAbs(destDir) {
			destDir = filepath.Join(req.Dir, destDir)
		}
		if err := store.MkdirAll(destDir); err != nil {
			jsonError(w, "failed to create output folder: "+err.Error(), 500)
			return
		}
		newPath = joinPath(destDir, req.NewName)
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
}

func (s *Server) handleOpenFolder(w http.ResponseWriter, r *http.Request) {
	dir := r.URL.Query().Get("dir")
	if dir == "" {
		jsonError(w, "dir parameter required", 400)
		return
	}
	store := s.getProvider(dir)
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
}
