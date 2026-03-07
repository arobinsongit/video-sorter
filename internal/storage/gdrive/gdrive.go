package gdrive

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"video-sorter/internal/storage"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// Storage implements storage.Provider for Google Drive.
type Storage struct {
	Service *drive.Service
	Token   *oauth2.Token
}

// OAuthConfig returns the OAuth2 config for Google Drive.
func OAuthConfig() (*oauth2.Config, error) {
	credsPath := ClientCredsPath()
	if data, err := os.ReadFile(credsPath); err == nil {
		config, err := google.ConfigFromJSON(data, drive.DriveScope)
		if err == nil {
			return config, nil
		}
	}

	if embeddedClientID == "" || embeddedClientSecret == "" {
		return nil, fmt.Errorf("Google Drive not configured: set embedded credentials and rebuild, or place credentials JSON at %s", credsPath)
	}

	return &oauth2.Config{
		ClientID:     embeddedClientID,
		ClientSecret: embeddedClientSecret,
		Scopes:       []string{drive.DriveScope},
		Endpoint:     google.Endpoint,
	}, nil
}

// HasEmbeddedCreds returns true if embedded OAuth credentials are set.
func HasEmbeddedCreds() bool {
	return embeddedClientID != "" && embeddedClientSecret != ""
}

// TokenPath returns the path to the stored Google Drive token.
func TokenPath() string {
	return filepath.Join(storage.CredentialsDir(), "gdrive-token.json")
}

// ClientCredsPath returns the path to the Google OAuth client credentials.
func ClientCredsPath() string {
	return filepath.Join(storage.CredentialsDir(), "gdrive-credentials.json")
}

// New creates a Google Drive storage provider from a saved token.
func New() (*Storage, error) {
	config, err := OAuthConfig()
	if err != nil {
		return nil, err
	}

	tokenData, err := os.ReadFile(TokenPath())
	if err != nil {
		return nil, fmt.Errorf("no saved token: %w", err)
	}

	var token oauth2.Token
	if err := json.Unmarshal(tokenData, &token); err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	client := config.Client(context.Background(), &token)
	srv, err := drive.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to create Drive service: %w", err)
	}

	return &Storage{Service: srv, Token: &token}, nil
}

// SaveToken saves an OAuth token to disk.
func SaveToken(token *oauth2.Token) error {
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(TokenPath(), data, 0600)
}

func parseDrivePath(fullPath string) string {
	return strings.TrimPrefix(fullPath, "gdrive://")
}

// ResolveFolder finds a Drive folder by path and returns its ID.
func (g *Storage) ResolveFolder(path string) (string, error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	parentID := "root"

	for _, name := range parts {
		if name == "" {
			continue
		}
		q := fmt.Sprintf("name='%s' and '%s' in parents and mimeType='application/vnd.google-apps.folder' and trashed=false",
			strings.ReplaceAll(name, "'", "\\'"), parentID)
		result, err := g.Service.Files.List().Q(q).Fields("files(id, name)").Do()
		if err != nil {
			return "", fmt.Errorf("searching for folder %q: %w", name, err)
		}
		if len(result.Files) == 0 {
			return "", fmt.Errorf("folder not found: %s", name)
		}
		parentID = result.Files[0].Id
	}
	return parentID, nil
}

func (g *Storage) ListFiles(path string) ([]storage.FileInfo, error) {
	drivePath := parseDrivePath(path)
	folderID, err := g.ResolveFolder(drivePath)
	if err != nil {
		return nil, err
	}

	var files []storage.FileInfo
	pageToken := ""
	for {
		q := fmt.Sprintf("'%s' in parents and trashed=false", folderID)
		call := g.Service.Files.List().Q(q).
			Fields("nextPageToken, files(id, name, size, modifiedTime, mimeType)").
			PageSize(1000)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}
		result, err := call.Do()
		if err != nil {
			return nil, err
		}

		for _, f := range result.Files {
			if f.MimeType == "application/vnd.google-apps.folder" {
				continue
			}
			ext := strings.ToLower(filepath.Ext(f.Name))
			if !storage.MediaExts[ext] {
				continue
			}
			modified := ""
			if t, err := time.Parse(time.RFC3339, f.ModifiedTime); err == nil {
				modified = t.Format("2006-01-02 15:04")
			}
			files = append(files, storage.FileInfo{
				Name:     f.Name,
				Size:     f.Size,
				Modified: modified,
			})
		}

		pageToken = result.NextPageToken
		if pageToken == "" {
			break
		}
	}

	return files, nil
}

func (g *Storage) ServeFile(w http.ResponseWriter, r *http.Request, dir, file string) {
	drivePath := parseDrivePath(dir)
	folderID, err := g.ResolveFolder(drivePath)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	q := fmt.Sprintf("name='%s' and '%s' in parents and trashed=false",
		strings.ReplaceAll(file, "'", "\\'"), folderID)
	result, err := g.Service.Files.List().Q(q).Fields("files(id, name, mimeType)").Do()
	if err != nil || len(result.Files) == 0 {
		http.Error(w, `{"error":"file not found"}`, http.StatusNotFound)
		return
	}

	resp, err := g.Service.Files.Get(result.Files[0].Id).Download()
	if err != nil {
		http.Error(w, `{"error":"download failed: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if result.Files[0].MimeType != "" {
		w.Header().Set("Content-Type", result.Files[0].MimeType)
	}
	io.Copy(w, resp.Body)
}

func (g *Storage) ReadFile(p string) ([]byte, error) {
	dir := storage.CloudDir(p)
	name := storage.CloudBase(p)
	drivePath := parseDrivePath(dir)

	folderID, err := g.ResolveFolder(drivePath)
	if err != nil {
		return nil, err
	}

	q := fmt.Sprintf("name='%s' and '%s' in parents and trashed=false",
		strings.ReplaceAll(name, "'", "\\'"), folderID)
	result, err := g.Service.Files.List().Q(q).Fields("files(id)").Do()
	if err != nil {
		return nil, err
	}
	if len(result.Files) == 0 {
		return nil, fmt.Errorf("file not found: %s", name)
	}

	resp, err := g.Service.Files.Get(result.Files[0].Id).Download()
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (g *Storage) WriteFile(p string, data []byte) error {
	dir := storage.CloudDir(p)
	name := storage.CloudBase(p)
	drivePath := parseDrivePath(dir)

	folderID, err := g.ResolveFolder(drivePath)
	if err != nil {
		return err
	}

	q := fmt.Sprintf("name='%s' and '%s' in parents and trashed=false",
		strings.ReplaceAll(name, "'", "\\'"), folderID)
	result, err := g.Service.Files.List().Q(q).Fields("files(id)").Do()
	if err != nil {
		return err
	}

	reader := strings.NewReader(string(data))
	if len(result.Files) > 0 {
		_, err = g.Service.Files.Update(result.Files[0].Id, &drive.File{}).
			Media(reader).Do()
	} else {
		_, err = g.Service.Files.Create(&drive.File{
			Name:    name,
			Parents: []string{folderID},
		}).Media(reader).Do()
	}
	return err
}

func (g *Storage) Rename(dir, oldName, newName string) error {
	drivePath := parseDrivePath(dir)
	folderID, err := g.ResolveFolder(drivePath)
	if err != nil {
		return err
	}

	q := fmt.Sprintf("name='%s' and '%s' in parents and trashed=false",
		strings.ReplaceAll(oldName, "'", "\\'"), folderID)
	result, err := g.Service.Files.List().Q(q).Fields("files(id)").Do()
	if err != nil {
		return err
	}
	if len(result.Files) == 0 {
		return fmt.Errorf("file not found: %s", oldName)
	}

	_, err = g.Service.Files.Update(result.Files[0].Id, &drive.File{Name: newName}).Do()
	return err
}

func (g *Storage) MoveFile(oldPath, newPath string) error {
	dir := storage.CloudDir(oldPath)
	oldName := storage.CloudBase(oldPath)
	newName := storage.CloudBase(newPath)
	return g.Rename(dir, oldName, newName)
}

func (g *Storage) CopyFile(oldPath, newPath string) error {
	dir := storage.CloudDir(oldPath)
	oldName := storage.CloudBase(oldPath)
	newName := storage.CloudBase(newPath)
	drivePath := parseDrivePath(dir)

	folderID, err := g.ResolveFolder(drivePath)
	if err != nil {
		return err
	}

	q := fmt.Sprintf("name='%s' and '%s' in parents and trashed=false",
		strings.ReplaceAll(oldName, "'", "\\'"), folderID)
	result, err := g.Service.Files.List().Q(q).Fields("files(id)").Do()
	if err != nil {
		return err
	}
	if len(result.Files) == 0 {
		return fmt.Errorf("file not found: %s", oldName)
	}

	_, err = g.Service.Files.Copy(result.Files[0].Id, &drive.File{
		Name:    newName,
		Parents: []string{folderID},
	}).Do()
	return err
}

func (g *Storage) FileExists(p string) bool {
	dir := storage.CloudDir(p)
	name := storage.CloudBase(p)
	drivePath := parseDrivePath(dir)

	folderID, err := g.ResolveFolder(drivePath)
	if err != nil {
		return false
	}

	q := fmt.Sprintf("name='%s' and '%s' in parents and trashed=false",
		strings.ReplaceAll(name, "'", "\\'"), folderID)
	result, err := g.Service.Files.List().Q(q).Fields("files(id)").Do()
	return err == nil && len(result.Files) > 0
}

func (g *Storage) MkdirAll(path string) error {
	return nil
}

func (g *Storage) IsLocal() bool {
	return false
}
