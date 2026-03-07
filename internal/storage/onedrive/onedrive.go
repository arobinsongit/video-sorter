package onedrive

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

	"video-sorter/internal/storage"
)

// Storage implements storage.Provider for Microsoft OneDrive.
type Storage struct {
	accessToken string
}

// Credentials holds OAuth client ID/secret for OneDrive.
type Credentials struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
}

// Token holds the OneDrive access/refresh tokens.
type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

// CredsPath returns the path to the stored OneDrive credentials.
func CredsPath() string {
	return filepath.Join(storage.CredentialsDir(), "onedrive-credentials.json")
}

// TokenPath returns the path to the stored OneDrive token.
func TokenPath() string {
	return filepath.Join(storage.CredentialsDir(), "onedrive-token.json")
}

// LoadCreds reads saved OneDrive OAuth credentials from disk.
func LoadCreds() (*Credentials, error) {
	data, err := os.ReadFile(CredsPath())
	if err != nil {
		return nil, err
	}
	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}
	return &creds, nil
}

// LoadToken reads a saved OneDrive token from disk.
func LoadToken() (*Token, error) {
	data, err := os.ReadFile(TokenPath())
	if err != nil {
		return nil, err
	}
	var token Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}
	return &token, nil
}

// SaveToken saves a OneDrive token to disk.
func SaveToken(token *Token) error {
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(TokenPath(), data, 0600)
}

// New creates a OneDrive storage provider from a saved token.
func New() (*Storage, error) {
	token, err := LoadToken()
	if err != nil {
		return nil, fmt.Errorf("no saved OneDrive token: %w", err)
	}
	return &Storage{accessToken: token.AccessToken}, nil
}

// NewFromToken creates a OneDrive storage provider from a token directly.
func NewFromToken(accessToken string) *Storage {
	return &Storage{accessToken: accessToken}
}

// AuthURL returns the OAuth2 authorization URL.
func AuthURL(clientID, redirectURL, state string) string {
	return fmt.Sprintf(
		"https://login.microsoftonline.com/consumers/oauth2/v2.0/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=Files.ReadWrite.All+offline_access&state=%s",
		clientID, redirectURL, state,
	)
}

// ExchangeCode exchanges an OAuth code for tokens.
func ExchangeCode(clientID, clientSecret, code, redirectURL string) (*Token, error) {
	body := fmt.Sprintf("client_id=%s&client_secret=%s&code=%s&redirect_uri=%s&grant_type=authorization_code&scope=Files.ReadWrite.All+offline_access",
		clientID, clientSecret, code, redirectURL)

	resp, err := http.Post("https://login.microsoftonline.com/consumers/oauth2/v2.0/token",
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
		return nil, fmt.Errorf("onedrive oauth error: %s", result.Error)
	}

	return &Token{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(result.ExpiresIn) * time.Second).Unix(),
	}, nil
}

func (o *Storage) graphAPI(method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+o.accessToken)
	if body != nil && method != "GET" {
		req.Header.Set("Content-Type", "application/json")
	}
	return http.DefaultClient.Do(req)
}

func parseOneDrivePath(fullPath string) string {
	p := strings.TrimPrefix(fullPath, "onedrive://")
	if p == "" || p == "/" {
		return ""
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return p
}

func graphDriveItemURL(path string) string {
	if path == "" || path == "/" {
		return "https://graph.microsoft.com/v1.0/me/drive/root"
	}
	return fmt.Sprintf("https://graph.microsoft.com/v1.0/me/drive/root:%s:", path)
}

func (o *Storage) ListFiles(path string) ([]storage.FileInfo, error) {
	odPath := parseOneDrivePath(path)

	var url string
	if odPath == "" || odPath == "/" {
		url = "https://graph.microsoft.com/v1.0/me/drive/root/children"
	} else {
		url = fmt.Sprintf("https://graph.microsoft.com/v1.0/me/drive/root:%s:/children", odPath)
	}

	var files []storage.FileInfo
	for url != "" {
		resp, err := o.graphAPI("GET", url, nil)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		var result struct {
			Value []struct {
				Name         string `json:"name"`
				Size         int64  `json:"size"`
				LastModified struct {
					DateTime string `json:"dateTime"`
				} `json:"lastModifiedDateTime"`
				Folder *struct{} `json:"folder"`
			} `json:"value"`
			NextLink string `json:"@odata.nextLink"`
		}

		respBody, _ := io.ReadAll(resp.Body)
		if err := json.Unmarshal(respBody, &result); err != nil {
			return nil, err
		}

		for _, item := range result.Value {
			if item.Folder != nil {
				continue
			}
			ext := strings.ToLower(filepath.Ext(item.Name))
			if !storage.MediaExts[ext] {
				continue
			}
			modified := ""
			if item.LastModified.DateTime != "" {
				if t, err := time.Parse(time.RFC3339, item.LastModified.DateTime); err == nil {
					modified = t.Format("2006-01-02 15:04")
				}
			}
			files = append(files, storage.FileInfo{
				Name:     item.Name,
				Size:     item.Size,
				Modified: modified,
			})
		}
		url = result.NextLink
	}

	return files, nil
}

func (o *Storage) ServeFile(w http.ResponseWriter, r *http.Request, dir, file string) {
	odPath := parseOneDrivePath(dir)
	fullPath := odPath + "/" + file

	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/me/drive/root:%s:/content", fullPath)
	resp, err := o.graphAPI("GET", url, nil)
	if err != nil || resp.StatusCode >= 400 {
		http.Error(w, `{"error":"file not found"}`, http.StatusNotFound)
		return
	}
	defer resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	io.Copy(w, resp.Body)
}

func (o *Storage) ReadFile(path string) ([]byte, error) {
	odPath := parseOneDrivePath(path)
	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/me/drive/root:%s:/content", odPath)

	resp, err := o.graphAPI("GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("onedrive download failed: %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func (o *Storage) WriteFile(path string, data []byte) error {
	odPath := parseOneDrivePath(path)
	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/me/drive/root:%s:/content", odPath)

	req, err := http.NewRequest("PUT", url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+o.accessToken)
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("onedrive upload failed: %d - %s", resp.StatusCode, string(body))
	}
	return nil
}

func (o *Storage) Rename(dir, oldName, newName string) error {
	odPath := parseOneDrivePath(dir)
	itemPath := odPath + "/" + oldName

	payload, _ := json.Marshal(map[string]string{"name": newName})
	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/me/drive/root:%s:", itemPath)

	req, err := http.NewRequest("PATCH", url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+o.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("onedrive rename failed: %d - %s", resp.StatusCode, string(body))
	}
	return nil
}

func (o *Storage) MoveFile(oldPath, newPath string) error {
	odOld := parseOneDrivePath(oldPath)
	newName := storage.CloudBase(newPath)
	newDir := storage.CloudDir(parseOneDrivePath(newPath))

	dirURL := graphDriveItemURL(newDir)
	resp, err := o.graphAPI("GET", dirURL, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var dirItem struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&dirItem); err != nil {
		return err
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"parentReference": map[string]string{"id": dirItem.ID},
		"name":            newName,
	})

	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/me/drive/root:%s:", odOld)
	req, err := http.NewRequest("PATCH", url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+o.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp2.Body.Close()
	if resp2.StatusCode >= 400 {
		body, _ := io.ReadAll(resp2.Body)
		return fmt.Errorf("onedrive move failed: %d - %s", resp2.StatusCode, string(body))
	}
	return nil
}

func (o *Storage) CopyFile(oldPath, newPath string) error {
	odOld := parseOneDrivePath(oldPath)
	newName := storage.CloudBase(newPath)
	newDir := storage.CloudDir(parseOneDrivePath(newPath))

	dirURL := graphDriveItemURL(newDir)
	resp, err := o.graphAPI("GET", dirURL, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var dirFull struct {
		ID              string `json:"id"`
		ParentReference struct {
			DriveID string `json:"driveId"`
		} `json:"parentReference"`
	}
	json.Unmarshal(respBody, &dirFull)

	payload, _ := json.Marshal(map[string]interface{}{
		"parentReference": map[string]string{
			"driveId": dirFull.ParentReference.DriveID,
			"id":      dirFull.ID,
		},
		"name": newName,
	})

	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/me/drive/root:%s:/copy", odOld)
	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+o.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp2.Body.Close()
	if resp2.StatusCode >= 400 {
		body, _ := io.ReadAll(resp2.Body)
		return fmt.Errorf("onedrive copy failed: %d - %s", resp2.StatusCode, string(body))
	}
	return nil
}

func (o *Storage) FileExists(path string) bool {
	odPath := parseOneDrivePath(path)
	url := graphDriveItemURL(odPath)
	resp, err := o.graphAPI("GET", url, nil)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func (o *Storage) MkdirAll(path string) error {
	odPath := parseOneDrivePath(path)
	if odPath == "" {
		return nil
	}

	parts := strings.Split(strings.Trim(odPath, "/"), "/")
	parentURL := "https://graph.microsoft.com/v1.0/me/drive/root"

	for _, name := range parts {
		payload, _ := json.Marshal(map[string]interface{}{
			"name":                              name,
			"folder":                            map[string]interface{}{},
			"@microsoft.graph.conflictBehavior": "rename",
		})

		resp, err := o.graphAPI("POST", parentURL+"/children", bytes.NewReader(payload))
		if err != nil {
			return err
		}
		var item struct {
			ID string `json:"id"`
		}
		json.NewDecoder(resp.Body).Decode(&item)
		resp.Body.Close()

		parentURL = fmt.Sprintf("https://graph.microsoft.com/v1.0/me/drive/items/%s", item.ID)
	}
	return nil
}

func (o *Storage) IsLocal() bool {
	return false
}

// FolderEntry represents a folder in the browse UI.
type FolderEntry struct {
	Name string
	Path string
}

// ListFolders lists folders at a given path for the browse UI.
func (o *Storage) ListFolders(path string) ([]FolderEntry, error) {
	var url string
	if path == "" || path == "/" {
		url = "https://graph.microsoft.com/v1.0/me/drive/root/children?$filter=folder ne null"
	} else {
		url = fmt.Sprintf("https://graph.microsoft.com/v1.0/me/drive/root:%s:/children?$filter=folder ne null", path)
	}

	resp, err := o.graphAPI("GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Value []struct {
			Name   string    `json:"name"`
			Folder *struct{} `json:"folder"`
		} `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var folders []FolderEntry
	for _, item := range result.Value {
		if item.Folder == nil {
			continue
		}
		itemPath := "/" + item.Name
		if path != "" && path != "/" {
			itemPath = strings.TrimRight(path, "/") + "/" + item.Name
		}
		folders = append(folders, FolderEntry{Name: item.Name, Path: itemPath})
	}
	return folders, nil
}
