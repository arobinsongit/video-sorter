package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DropboxStorage implements StorageProvider for Dropbox.
type DropboxStorage struct {
	accessToken string
}

// dropboxCredsPath returns the path to the stored Dropbox credentials.
func dropboxCredsPath() string {
	return filepath.Join(credentialsDir(), "dropbox-credentials.json")
}

// dropboxTokenPath returns the path to the stored Dropbox token.
func dropboxTokenPath() string {
	return filepath.Join(credentialsDir(), "dropbox-token.json")
}

// DropboxCredentials holds OAuth client ID/secret for Dropbox.
type DropboxCredentials struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
}

// DropboxToken holds the Dropbox access/refresh tokens.
type DropboxToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

// loadDropboxCreds reads saved Dropbox OAuth credentials from disk.
func loadDropboxCreds() (*DropboxCredentials, error) {
	data, err := os.ReadFile(dropboxCredsPath())
	if err != nil {
		return nil, err
	}
	var creds DropboxCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}
	return &creds, nil
}

// loadDropboxToken reads a saved Dropbox token from disk.
func loadDropboxToken() (*DropboxToken, error) {
	data, err := os.ReadFile(dropboxTokenPath())
	if err != nil {
		return nil, err
	}
	var token DropboxToken
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}
	return &token, nil
}

// saveDropboxToken saves a Dropbox token to disk.
func saveDropboxToken(token *DropboxToken) error {
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(dropboxTokenPath(), data, 0600)
}

// newDropboxStorage creates a Dropbox storage provider from a saved token.
func newDropboxStorage() (*DropboxStorage, error) {
	token, err := loadDropboxToken()
	if err != nil {
		return nil, fmt.Errorf("no saved Dropbox token: %w", err)
	}
	return &DropboxStorage{accessToken: token.AccessToken}, nil
}

// dropboxAuthURL returns the OAuth2 authorization URL.
func dropboxAuthURL(clientID, redirectURL, state string) string {
	return fmt.Sprintf(
		"https://www.dropbox.com/oauth2/authorize?client_id=%s&redirect_uri=%s&response_type=code&state=%s&token_access_type=offline",
		clientID, redirectURL, state,
	)
}

// exchangeDropboxCode exchanges an OAuth code for tokens.
func exchangeDropboxCode(clientID, clientSecret, code, redirectURL string) (*DropboxToken, error) {
	body := fmt.Sprintf("code=%s&grant_type=authorization_code&redirect_uri=%s&client_id=%s&client_secret=%s",
		code, redirectURL, clientID, clientSecret)

	resp, err := http.Post("https://api.dropboxapi.com/oauth2/token",
		"application/x-www-form-urlencoded", strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		Error        string `json:"error_description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Error != "" {
		return nil, fmt.Errorf("dropbox oauth error: %s", result.Error)
	}

	return &DropboxToken{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(result.ExpiresIn) * time.Second).Unix(),
	}, nil
}

// dropboxAPI makes an authenticated Dropbox API call.
func (d *DropboxStorage) dropboxAPI(url, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+d.accessToken)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return http.DefaultClient.Do(req)
}

// parseDropboxPath converts "dropbox:///path/to/folder" to "/path/to/folder".
func parseDropboxPath(fullPath string) string {
	p := strings.TrimPrefix(fullPath, "dropbox://")
	if p == "" || p == "/" {
		return ""
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return p
}

func (d *DropboxStorage) ListFiles(path string) ([]FileInfo, error) {
	dbxPath := parseDropboxPath(path)

	payload := map[string]interface{}{
		"path":                            dbxPath,
		"recursive":                       false,
		"include_media_info":              false,
		"include_deleted":                 false,
		"include_has_explicit_shared_members": false,
	}
	if dbxPath == "" {
		payload["path"] = ""
	}

	body, _ := json.Marshal(payload)
	resp, err := d.dropboxAPI("https://api.dropboxapi.com/2/files/list_folder",
		"application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Entries []struct {
			Tag            string `json:".tag"`
			Name           string `json:"name"`
			Size           int64  `json:"size"`
			ClientModified string `json:"client_modified"`
		} `json:"entries"`
		HasMore bool   `json:"has_more"`
		Cursor  string `json:"cursor"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var files []FileInfo
	for _, e := range result.Entries {
		if e.Tag != "file" {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name))
		if !mediaExts[ext] {
			continue
		}
		modified := ""
		if t, err := time.Parse(time.RFC3339, e.ClientModified); err == nil {
			modified = t.Format("2006-01-02 15:04")
		}
		files = append(files, FileInfo{
			Name:     e.Name,
			Size:     e.Size,
			Modified: modified,
		})
	}

	// Handle pagination
	for result.HasMore {
		continueBody, _ := json.Marshal(map[string]string{"cursor": result.Cursor})
		resp2, err := d.dropboxAPI("https://api.dropboxapi.com/2/files/list_folder/continue",
			"application/json", bytes.NewReader(continueBody))
		if err != nil {
			break
		}
		if err := json.NewDecoder(resp2.Body).Decode(&result); err != nil {
			resp2.Body.Close()
			break
		}
		resp2.Body.Close()

		for _, e := range result.Entries {
			if e.Tag != "file" {
				continue
			}
			ext := strings.ToLower(filepath.Ext(e.Name))
			if !mediaExts[ext] {
				continue
			}
			modified := ""
			if t, err := time.Parse(time.RFC3339, e.ClientModified); err == nil {
				modified = t.Format("2006-01-02 15:04")
			}
			files = append(files, FileInfo{
				Name:     e.Name,
				Size:     e.Size,
				Modified: modified,
			})
		}
	}

	return files, nil
}

func (d *DropboxStorage) ServeFile(w http.ResponseWriter, r *http.Request, dir, file string) {
	dbxPath := parseDropboxPath(dir)
	fullPath := dbxPath + "/" + file

	arg, _ := json.Marshal(map[string]string{"path": fullPath})
	req, err := http.NewRequest("POST", "https://content.dropboxapi.com/2/files/download", nil)
	if err != nil {
		http.Error(w, `{"error":"request failed"}`, http.StatusInternalServerError)
		return
	}
	req.Header.Set("Authorization", "Bearer "+d.accessToken)
	req.Header.Set("Dropbox-API-Arg", string(arg))

	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		http.Error(w, `{"error":"file not found"}`, http.StatusNotFound)
		return
	}
	defer resp.Body.Close()

	ext := strings.ToLower(filepath.Ext(file))
	switch ext {
	case ".mp4":
		w.Header().Set("Content-Type", "video/mp4")
	case ".mov":
		w.Header().Set("Content-Type", "video/quicktime")
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	}
	io.Copy(w, resp.Body)
}

func (d *DropboxStorage) ReadFile(path string) ([]byte, error) {
	dbxPath := parseDropboxPath(path)

	arg, _ := json.Marshal(map[string]string{"path": dbxPath})
	req, err := http.NewRequest("POST", "https://content.dropboxapi.com/2/files/download", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+d.accessToken)
	req.Header.Set("Dropbox-API-Arg", string(arg))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("dropbox download failed: %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func (d *DropboxStorage) WriteFile(path string, data []byte) error {
	dbxPath := parseDropboxPath(path)

	arg, _ := json.Marshal(map[string]interface{}{
		"path":       dbxPath,
		"mode":       "overwrite",
		"autorename": false,
		"mute":       true,
	})

	req, err := http.NewRequest("POST", "https://content.dropboxapi.com/2/files/upload", bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+d.accessToken)
	req.Header.Set("Dropbox-API-Arg", string(arg))
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("dropbox upload failed: %d - %s", resp.StatusCode, string(body))
	}
	return nil
}

func (d *DropboxStorage) Rename(dir, oldName, newName string) error {
	dbxPath := parseDropboxPath(dir)
	fromPath := dbxPath + "/" + oldName
	toPath := dbxPath + "/" + newName

	payload, _ := json.Marshal(map[string]interface{}{
		"from_path":           fromPath,
		"to_path":             toPath,
		"allow_shared_folder": false,
		"autorename":          false,
	})

	resp, err := d.dropboxAPI("https://api.dropboxapi.com/2/files/move_v2",
		"application/json", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("dropbox rename failed: %s", string(body))
	}
	return nil
}

func (d *DropboxStorage) MoveFile(oldPath, newPath string) error {
	from := parseDropboxPath(oldPath)
	to := parseDropboxPath(newPath)

	payload, _ := json.Marshal(map[string]interface{}{
		"from_path":           from,
		"to_path":             to,
		"allow_shared_folder": false,
		"autorename":          false,
	})

	resp, err := d.dropboxAPI("https://api.dropboxapi.com/2/files/move_v2",
		"application/json", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("dropbox move failed: %s", string(body))
	}
	return nil
}

func (d *DropboxStorage) CopyFile(oldPath, newPath string) error {
	from := parseDropboxPath(oldPath)
	to := parseDropboxPath(newPath)

	payload, _ := json.Marshal(map[string]interface{}{
		"from_path":           from,
		"to_path":             to,
		"allow_shared_folder": false,
		"autorename":          false,
	})

	resp, err := d.dropboxAPI("https://api.dropboxapi.com/2/files/copy_v2",
		"application/json", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("dropbox copy failed: %s", string(body))
	}
	return nil
}

func (d *DropboxStorage) FileExists(path string) bool {
	dbxPath := parseDropboxPath(path)
	payload, _ := json.Marshal(map[string]string{"path": dbxPath})

	resp, err := d.dropboxAPI("https://api.dropboxapi.com/2/files/get_metadata",
		"application/json", bytes.NewReader(payload))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func (d *DropboxStorage) MkdirAll(path string) error {
	dbxPath := parseDropboxPath(path)
	if dbxPath == "" {
		return nil
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"path":       dbxPath,
		"autorename": false,
	})

	resp, err := d.dropboxAPI("https://api.dropboxapi.com/2/files/create_folder_v2",
		"application/json", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// Ignore 409 (folder already exists)
	return nil
}

func (d *DropboxStorage) IsLocal() bool {
	return false
}

// listDropboxFolders lists folders at a given path for the browse UI.
func (d *DropboxStorage) listFolders(path string) ([]struct {
	Name string
	Path string
}, error) {
	payload := map[string]interface{}{
		"path":                            path,
		"recursive":                       false,
		"include_deleted":                 false,
		"include_has_explicit_shared_members": false,
	}
	if path == "" {
		payload["path"] = ""
	}
	body, _ := json.Marshal(payload)

	resp, err := d.dropboxAPI("https://api.dropboxapi.com/2/files/list_folder",
		"application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Entries []struct {
			Tag  string `json:".tag"`
			Name string `json:"name"`
			Path string `json:"path_display"`
		} `json:"entries"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var folders []struct {
		Name string
		Path string
	}
	for _, e := range result.Entries {
		if e.Tag == "folder" {
			folders = append(folders, struct {
				Name string
				Path string
			}{Name: e.Name, Path: e.Path})
		}
	}
	return folders, nil
}
