package server

import (
	"encoding/json"
	"fmt"
	"os"

	"media-sorter/internal/storage"
	"media-sorter/internal/storage/dropbox"
)

type dropboxProvider struct {
	store *dropbox.Storage
}

func newDropboxProvider() *dropboxProvider {
	p := &dropboxProvider{}
	if db, err := dropbox.New(); err == nil {
		p.store = db
	}
	return p
}

func (p *dropboxProvider) ID() string          { return "dropbox" }
func (p *dropboxProvider) DisplayName() string { return "Dropbox" }
func (p *dropboxProvider) PathPrefix() string  { return "dropbox://" }
func (p *dropboxProvider) Connected() bool     { return p.store != nil }

func (p *dropboxProvider) HasCredentials() bool {
	_, err := os.Stat(dropbox.CredsPath())
	return err == nil
}

func (p *dropboxProvider) SaveCredentials(data json.RawMessage) error {
	var creds dropbox.Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return fmt.Errorf("Invalid Dropbox credentials: %w", err)
	}
	if creds.ClientID == "" || creds.ClientSecret == "" {
		return fmt.Errorf("clientId and clientSecret are required")
	}
	out, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}
	return os.WriteFile(dropbox.CredsPath(), out, 0600)
}

func (p *dropboxProvider) Connect(callbackURL, state string) (string, error) {
	creds, err := dropbox.LoadCreds()
	if err != nil {
		return "", fmt.Errorf("Dropbox credentials not configured. Place your credentials JSON file at: %s", dropbox.CredsPath())
	}
	authURL := dropbox.AuthURL(creds.ClientID, callbackURL, state)
	return authURL, nil
}

func (p *dropboxProvider) HandleCallback(code, callbackURL string) error {
	creds, err := dropbox.LoadCreds()
	if err != nil {
		return fmt.Errorf("failed to load credentials: %w", err)
	}
	token, err := dropbox.ExchangeCode(creds.ClientID, creds.ClientSecret, code, callbackURL)
	if err != nil {
		return fmt.Errorf("failed to exchange token: %w", err)
	}
	if err := dropbox.SaveToken(token); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}
	p.store = dropbox.NewFromToken(token.AccessToken)
	return nil
}

func (p *dropboxProvider) Disconnect() {
	p.store = nil
	os.Remove(dropbox.TokenPath())
}

func (p *dropboxProvider) BrowseFolders(path string) ([]storage.FolderEntry, error) {
	return p.store.ListFolders(storage.NormalizeBrowsePath(path))
}

func (p *dropboxProvider) StorageProvider() storage.Provider {
	if p.store == nil {
		return nil
	}
	return p.store
}
