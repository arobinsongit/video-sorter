package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"media-sorter/internal/storage"
	"media-sorter/internal/storage/gdrive"

	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type gdriveProvider struct {
	store *gdrive.Storage
}

func newGDriveProvider() *gdriveProvider {
	p := &gdriveProvider{}
	if gd, err := gdrive.New(); err == nil {
		p.store = gd
	}
	return p
}

func (p *gdriveProvider) ID() string          { return "gdrive" }
func (p *gdriveProvider) DisplayName() string { return "Google Drive" }
func (p *gdriveProvider) PathPrefix() string  { return "gdrive://" }
func (p *gdriveProvider) Connected() bool     { return p.store != nil }

func (p *gdriveProvider) HasCredentials() bool {
	if gdrive.HasEmbeddedCreds() {
		return true
	}
	_, err := os.Stat(gdrive.ClientCredsPath())
	return err == nil
}

func (p *gdriveProvider) SaveCredentials(data json.RawMessage) error {
	return os.WriteFile(gdrive.ClientCredsPath(), data, 0600)
}

func (p *gdriveProvider) Connect(callbackURL, state string) (string, error) {
	config, err := gdrive.OAuthConfig()
	if err != nil {
		return "", err
	}
	config.RedirectURL = callbackURL
	authURL := config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	return authURL, nil
}

func (p *gdriveProvider) HandleCallback(code, callbackURL string) error {
	config, err := gdrive.OAuthConfig()
	if err != nil {
		return fmt.Errorf("failed to load credentials: %w", err)
	}
	config.RedirectURL = callbackURL

	token, err := config.Exchange(context.Background(), code)
	if err != nil {
		return fmt.Errorf("failed to exchange token: %w", err)
	}
	if err := gdrive.SaveToken(token); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	client := config.Client(context.Background(), token)
	srv, err := drive.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("failed to create Drive service: %w", err)
	}
	p.store = &gdrive.Storage{Service: srv, Token: token}
	return nil
}

func (p *gdriveProvider) Disconnect() {
	p.store = nil
	os.Remove(gdrive.TokenPath())
}

func (p *gdriveProvider) BrowseFolders(path string) ([]storage.FolderEntry, error) {
	folderID := "root"
	if path != "" && path != "/" {
		var err error
		folderID, err = p.store.ResolveFolder(path)
		if err != nil {
			return nil, err
		}
	}

	q := fmt.Sprintf("'%s' in parents and mimeType='application/vnd.google-apps.folder' and trashed=false", folderID)
	result, err := p.store.Service.Files.List().Q(q).
		Fields("files(id, name)").OrderBy("name").Do()
	if err != nil {
		return nil, err
	}

	var folders []storage.FolderEntry
	for _, f := range result.Files {
		entryPath := f.Name
		if path != "" && path != "/" {
			entryPath = strings.TrimRight(path, "/") + "/" + f.Name
		}
		folders = append(folders, storage.FolderEntry{
			Name: f.Name,
			ID:   f.Id,
			Path: entryPath,
		})
	}
	return folders, nil
}

func (p *gdriveProvider) StorageProvider() storage.Provider {
	if p.store == nil {
		return nil
	}
	return p.store
}
