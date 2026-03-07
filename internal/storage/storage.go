package storage

import (
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// SplitScheme splits "scheme://rest" into ("scheme://", "rest").
// If no scheme, returns ("", p).
func SplitScheme(p string) (scheme, rest string) {
	if idx := strings.Index(p, "://"); idx >= 0 {
		return p[:idx+3], p[idx+3:]
	}
	return "", p
}

// CloudJoin joins cloud path segments using forward slashes, preserving the URL scheme.
func CloudJoin(parts ...string) string {
	if len(parts) == 0 {
		return ""
	}
	scheme, first := SplitScheme(parts[0])
	parts[0] = first
	return scheme + path.Join(parts...)
}

// CloudDir returns the directory portion of a cloud path, preserving the URL scheme.
func CloudDir(p string) string {
	scheme, rest := SplitScheme(p)
	return scheme + path.Dir(rest)
}

// CloudBase returns the filename portion of a cloud path.
func CloudBase(p string) string {
	_, rest := SplitScheme(p)
	return path.Base(rest)
}

// FileInfo represents a media file in any storage backend.
type FileInfo struct {
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	Modified string `json:"modified"`
}

// Provider abstracts file operations across local and cloud storage.
type Provider interface {
	ListFiles(path string) ([]FileInfo, error)
	ServeFile(w http.ResponseWriter, r *http.Request, dir, file string)
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte) error
	Rename(dir, oldName, newName string) error
	MoveFile(oldPath, newPath string) error
	CopyFile(oldPath, newPath string) error
	FileExists(path string) bool
	MkdirAll(path string) error
	IsLocal() bool
}

// MediaExts defines supported media file extensions.
var MediaExts = map[string]bool{
	// Video
	".mp4": true, ".mov": true, ".avi": true, ".mkv": true, ".webm": true,
	// Photo
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
	".bmp": true, ".tiff": true, ".tif": true, ".heic": true, ".heif": true,
}

// CredentialsDir returns the directory for storing credentials.
func CredentialsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	dir := filepath.Join(home, ".media-sorter")
	os.MkdirAll(dir, 0700)
	return dir
}

// LocalStorage implements Provider for the local filesystem.
type LocalStorage struct{}

func (l *LocalStorage) ListFiles(dir string) ([]FileInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []FileInfo
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if MediaExts[ext] {
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
	return files, nil
}

func (l *LocalStorage) ServeFile(w http.ResponseWriter, r *http.Request, dir, file string) {
	clean := filepath.Clean(file)
	if strings.Contains(clean, string(filepath.Separator)) || strings.Contains(clean, "..") {
		http.Error(w, `{"error":"invalid filename"}`, http.StatusBadRequest)
		return
	}
	fullPath := filepath.Join(dir, clean)
	http.ServeFile(w, r, fullPath)
}

func (l *LocalStorage) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (l *LocalStorage) WriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

func (l *LocalStorage) Rename(dir, oldName, newName string) error {
	return os.Rename(filepath.Join(dir, oldName), filepath.Join(dir, newName))
}

func (l *LocalStorage) MoveFile(oldPath, newPath string) error {
	if err := os.Rename(oldPath, newPath); err != nil {
		if err := CopyFileLocal(oldPath, newPath); err != nil {
			return err
		}
		return os.Remove(oldPath)
	}
	return nil
}

func (l *LocalStorage) CopyFile(oldPath, newPath string) error {
	return CopyFileLocal(oldPath, newPath)
}

func (l *LocalStorage) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (l *LocalStorage) MkdirAll(path string) error {
	return os.MkdirAll(path, 0755)
}

func (l *LocalStorage) IsLocal() bool {
	return true
}

// CopyFileLocal copies a file on the local filesystem.
func CopyFileLocal(src, dst string) error {
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
