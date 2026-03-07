package main

import (
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// splitScheme splits "scheme://rest" into ("scheme://", "rest").
// If no scheme, returns ("", p).
func splitScheme(p string) (scheme, rest string) {
	if idx := strings.Index(p, "://"); idx >= 0 {
		return p[:idx+3], p[idx+3:]
	}
	return "", p
}

// cloudJoin joins cloud path segments using forward slashes, preserving the URL scheme.
func cloudJoin(parts ...string) string {
	if len(parts) == 0 {
		return ""
	}
	scheme, first := splitScheme(parts[0])
	parts[0] = first
	return scheme + path.Join(parts...)
}

// cloudDir returns the directory portion of a cloud path, preserving the URL scheme.
func cloudDir(p string) string {
	scheme, rest := splitScheme(p)
	return scheme + path.Dir(rest)
}

// cloudBase returns the filename portion of a cloud path.
func cloudBase(p string) string {
	_, rest := splitScheme(p)
	return path.Base(rest)
}

// FileInfo represents a media file in any storage backend.
type FileInfo struct {
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	Modified string `json:"modified"`
}

// StorageProvider abstracts file operations across local and cloud storage.
type StorageProvider interface {
	// ListFiles returns media files at the given path.
	ListFiles(path string) ([]FileInfo, error)

	// ServeFile writes the file content to the HTTP response (supports range requests for local).
	ServeFile(w http.ResponseWriter, r *http.Request, dir, file string)

	// ReadFile reads a file's contents.
	ReadFile(path string) ([]byte, error)

	// WriteFile writes data to a file.
	WriteFile(path string, data []byte) error

	// Rename renames a file within the same directory.
	Rename(dir, oldName, newName string) error

	// MoveFile moves a file to a new path (potentially cross-directory).
	MoveFile(oldPath, newPath string) error

	// CopyFile copies a file to a new path.
	CopyFile(oldPath, newPath string) error

	// FileExists checks if a file exists.
	FileExists(path string) bool

	// MkdirAll creates a directory tree.
	MkdirAll(path string) error

	// IsLocal returns true if this is a local filesystem provider.
	IsLocal() bool
}

// mediaExts defines supported media file extensions.
var mediaExts = map[string]bool{
	// Video
	".mp4": true, ".mov": true, ".avi": true, ".mkv": true, ".webm": true,
	// Photo
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
	".bmp": true, ".tiff": true, ".tif": true, ".heic": true, ".heif": true,
}

// LocalStorage implements StorageProvider for the local filesystem.
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
		// Cross-filesystem: fall back to copy+delete
		if err := copyFileLocal(oldPath, newPath); err != nil {
			return err
		}
		return os.Remove(oldPath)
	}
	return nil
}

func (l *LocalStorage) CopyFile(oldPath, newPath string) error {
	return copyFileLocal(oldPath, newPath)
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

func copyFileLocal(src, dst string) error {
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

// getStorageProvider returns the appropriate storage provider for a path.
// Cloud paths use prefixes like "gdrive://", "s3://", "dropbox://", "onedrive://".
// All other paths use local filesystem.
func getStorageProvider(path string) StorageProvider {
	switch {
	case strings.HasPrefix(path, "gdrive://"):
		if gdrive != nil {
			return gdrive
		}
		return &LocalStorage{}
	case strings.HasPrefix(path, "s3://"):
		if s3store != nil {
			return s3store
		}
		return &LocalStorage{}
	case strings.HasPrefix(path, "dropbox://"):
		if dropbox != nil {
			return dropbox
		}
		return &LocalStorage{}
	case strings.HasPrefix(path, "onedrive://"):
		if onedrive != nil {
			return onedrive
		}
		return &LocalStorage{}
	default:
		return &LocalStorage{}
	}
}

// Global cloud storage providers (nil if not connected).
var gdrive *GoogleDriveStorage
var s3store *S3Storage
var dropbox *DropboxStorage
var onedrive *OneDriveStorage
