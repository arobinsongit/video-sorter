package main

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

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// GoogleDriveStorage implements StorageProvider for Google Drive.
type GoogleDriveStorage struct {
	service *drive.Service
	token   *oauth2.Token
}

// gdriveOAuthConfig returns the OAuth2 config for Google Drive.
// Uses embedded client credentials by default. If a credentials JSON file
// exists at ~/.media-sorter/gdrive-credentials.json, it overrides the defaults.
func gdriveOAuthConfig() (*oauth2.Config, error) {
	// Check for override credentials file first
	credsPath := gdriveClientCredsPath()
	if data, err := os.ReadFile(credsPath); err == nil {
		config, err := google.ConfigFromJSON(data,
			drive.DriveScope,
		)
		if err == nil {
			return config, nil
		}
	}

	// Use embedded credentials
	if embeddedGDriveClientID == "" || embeddedGDriveClientSecret == "" {
		return nil, fmt.Errorf("Google Drive not configured: set embedded credentials in gdrive_client.go and rebuild, or place credentials JSON at %s", credsPath)
	}

	return &oauth2.Config{
		ClientID:     embeddedGDriveClientID,
		ClientSecret: embeddedGDriveClientSecret,
		Scopes:       []string{drive.DriveScope},
		Endpoint:     google.Endpoint,
	}, nil
}

// credentialsDir returns the directory for storing credentials.
func credentialsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	dir := filepath.Join(home, ".media-sorter")
	os.MkdirAll(dir, 0700)
	return dir
}

// gdriveTokenPath returns the path to the stored Google Drive token.
func gdriveTokenPath() string {
	return filepath.Join(credentialsDir(), "gdrive-token.json")
}

// gdriveClientCredsPath returns the path to the Google OAuth client credentials.
func gdriveClientCredsPath() string {
	return filepath.Join(credentialsDir(), "gdrive-credentials.json")
}

// newGoogleDriveStorage creates a Google Drive storage provider from a saved token.
func newGoogleDriveStorage() (*GoogleDriveStorage, error) {
	config, err := gdriveOAuthConfig()
	if err != nil {
		return nil, err
	}

	tokenData, err := os.ReadFile(gdriveTokenPath())
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

	return &GoogleDriveStorage{service: srv, token: &token}, nil
}

// saveToken saves an OAuth token to disk.
func saveGdriveToken(token *oauth2.Token) error {
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(gdriveTokenPath(), data, 0600)
}

// parseDrivePath splits "gdrive://folder/path" into the path portion.
func parseDrivePath(fullPath string) string {
	return strings.TrimPrefix(fullPath, "gdrive://")
}

// resolveFolder finds or creates a Drive folder by path and returns its ID.
func (g *GoogleDriveStorage) resolveFolder(path string) (string, error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	parentID := "root"

	for _, name := range parts {
		if name == "" {
			continue
		}
		q := fmt.Sprintf("name='%s' and '%s' in parents and mimeType='application/vnd.google-apps.folder' and trashed=false",
			strings.ReplaceAll(name, "'", "\\'"), parentID)
		result, err := g.service.Files.List().Q(q).Fields("files(id, name)").Do()
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

func (g *GoogleDriveStorage) ListFiles(path string) ([]FileInfo, error) {
	drivePath := parseDrivePath(path)
	folderID, err := g.resolveFolder(drivePath)
	if err != nil {
		return nil, err
	}

	var files []FileInfo
	pageToken := ""
	for {
		q := fmt.Sprintf("'%s' in parents and trashed=false", folderID)
		call := g.service.Files.List().Q(q).
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
			if !mediaExts[ext] {
				continue
			}
			modified := ""
			if t, err := time.Parse(time.RFC3339, f.ModifiedTime); err == nil {
				modified = t.Format("2006-01-02 15:04")
			}
			files = append(files, FileInfo{
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

func (g *GoogleDriveStorage) ServeFile(w http.ResponseWriter, r *http.Request, dir, file string) {
	drivePath := parseDrivePath(dir)
	folderID, err := g.resolveFolder(drivePath)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Find the file
	q := fmt.Sprintf("name='%s' and '%s' in parents and trashed=false",
		strings.ReplaceAll(file, "'", "\\'"), folderID)
	result, err := g.service.Files.List().Q(q).Fields("files(id, name, mimeType)").Do()
	if err != nil || len(result.Files) == 0 {
		http.Error(w, `{"error":"file not found"}`, http.StatusNotFound)
		return
	}

	resp, err := g.service.Files.Get(result.Files[0].Id).Download()
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

func (g *GoogleDriveStorage) ReadFile(p string) ([]byte, error) {
	dir := cloudDir(p)
	name := cloudBase(p)
	drivePath := parseDrivePath(dir)

	folderID, err := g.resolveFolder(drivePath)
	if err != nil {
		return nil, err
	}

	q := fmt.Sprintf("name='%s' and '%s' in parents and trashed=false",
		strings.ReplaceAll(name, "'", "\\'"), folderID)
	result, err := g.service.Files.List().Q(q).Fields("files(id)").Do()
	if err != nil {
		return nil, err
	}
	if len(result.Files) == 0 {
		return nil, fmt.Errorf("file not found: %s", name)
	}

	resp, err := g.service.Files.Get(result.Files[0].Id).Download()
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (g *GoogleDriveStorage) WriteFile(p string, data []byte) error {
	dir := cloudDir(p)
	name := cloudBase(p)
	drivePath := parseDrivePath(dir)

	folderID, err := g.resolveFolder(drivePath)
	if err != nil {
		return err
	}

	// Check if file exists — update it; otherwise create
	q := fmt.Sprintf("name='%s' and '%s' in parents and trashed=false",
		strings.ReplaceAll(name, "'", "\\'"), folderID)
	result, err := g.service.Files.List().Q(q).Fields("files(id)").Do()
	if err != nil {
		return err
	}

	reader := strings.NewReader(string(data))
	if len(result.Files) > 0 {
		_, err = g.service.Files.Update(result.Files[0].Id, &drive.File{}).
			Media(reader).Do()
	} else {
		_, err = g.service.Files.Create(&drive.File{
			Name:    name,
			Parents: []string{folderID},
		}).Media(reader).Do()
	}
	return err
}

func (g *GoogleDriveStorage) Rename(dir, oldName, newName string) error {
	drivePath := parseDrivePath(dir)
	folderID, err := g.resolveFolder(drivePath)
	if err != nil {
		return err
	}

	q := fmt.Sprintf("name='%s' and '%s' in parents and trashed=false",
		strings.ReplaceAll(oldName, "'", "\\'"), folderID)
	result, err := g.service.Files.List().Q(q).Fields("files(id)").Do()
	if err != nil {
		return err
	}
	if len(result.Files) == 0 {
		return fmt.Errorf("file not found: %s", oldName)
	}

	_, err = g.service.Files.Update(result.Files[0].Id, &drive.File{Name: newName}).Do()
	return err
}

func (g *GoogleDriveStorage) MoveFile(oldPath, newPath string) error {
	// For Google Drive, move is rename + parent change
	// Simplified: only support rename within same folder for now
	dir := cloudDir(oldPath)
	oldName := cloudBase(oldPath)
	newName := cloudBase(newPath)
	return g.Rename(dir, oldName, newName)
}

func (g *GoogleDriveStorage) CopyFile(oldPath, newPath string) error {
	dir := cloudDir(oldPath)
	oldName := cloudBase(oldPath)
	newName := cloudBase(newPath)
	drivePath := parseDrivePath(dir)

	folderID, err := g.resolveFolder(drivePath)
	if err != nil {
		return err
	}

	q := fmt.Sprintf("name='%s' and '%s' in parents and trashed=false",
		strings.ReplaceAll(oldName, "'", "\\'"), folderID)
	result, err := g.service.Files.List().Q(q).Fields("files(id)").Do()
	if err != nil {
		return err
	}
	if len(result.Files) == 0 {
		return fmt.Errorf("file not found: %s", oldName)
	}

	_, err = g.service.Files.Copy(result.Files[0].Id, &drive.File{
		Name:    newName,
		Parents: []string{folderID},
	}).Do()
	return err
}

func (g *GoogleDriveStorage) FileExists(p string) bool {
	dir := cloudDir(p)
	name := cloudBase(p)
	drivePath := parseDrivePath(dir)

	folderID, err := g.resolveFolder(drivePath)
	if err != nil {
		return false
	}

	q := fmt.Sprintf("name='%s' and '%s' in parents and trashed=false",
		strings.ReplaceAll(name, "'", "\\'"), folderID)
	result, err := g.service.Files.List().Q(q).Fields("files(id)").Do()
	return err == nil && len(result.Files) > 0
}

func (g *GoogleDriveStorage) MkdirAll(path string) error {
	// For Drive, folders are created on demand during write
	return nil
}

func (g *GoogleDriveStorage) IsLocal() bool {
	return false
}
