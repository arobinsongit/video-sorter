package server

import (
	"encoding/json"
	"fmt"
	"os"

	"media-sorter/internal/storage"
	"media-sorter/internal/storage/onedrive"
)

type onedriveProvider struct {
	store *onedrive.Storage
}

func newOneDriveProvider() *onedriveProvider {
	p := &onedriveProvider{}
	if od, err := onedrive.New(); err == nil {
		p.store = od
	}
	return p
}

func (p *onedriveProvider) ID() string          { return "onedrive" }
func (p *onedriveProvider) DisplayName() string { return "OneDrive" }
func (p *onedriveProvider) PathPrefix() string  { return "onedrive://" }
func (p *onedriveProvider) Connected() bool     { return p.store != nil }

func (p *onedriveProvider) HasCredentials() bool {
	_, err := os.Stat(onedrive.CredsPath())
	return err == nil
}

func (p *onedriveProvider) SaveCredentials(data json.RawMessage) error {
	var creds onedrive.Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return fmt.Errorf("Invalid OneDrive credentials: %w", err)
	}
	if creds.ClientID == "" || creds.ClientSecret == "" {
		return fmt.Errorf("clientId and clientSecret are required")
	}
	out, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}
	return os.WriteFile(onedrive.CredsPath(), out, 0600)
}

func (p *onedriveProvider) Connect(callbackURL, state string) (string, error) {
	creds, err := onedrive.LoadCreds()
	if err != nil {
		return "", fmt.Errorf("OneDrive credentials not configured. Place your credentials JSON file at: %s", onedrive.CredsPath())
	}
	authURL := onedrive.AuthURL(creds.ClientID, callbackURL, state)
	return authURL, nil
}

func (p *onedriveProvider) HandleCallback(code, callbackURL string) error {
	creds, err := onedrive.LoadCreds()
	if err != nil {
		return fmt.Errorf("failed to load credentials: %w", err)
	}
	token, err := onedrive.ExchangeCode(creds.ClientID, creds.ClientSecret, code, callbackURL)
	if err != nil {
		return fmt.Errorf("failed to exchange token: %w", err)
	}
	if err := onedrive.SaveToken(token); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}
	p.store = onedrive.NewFromToken(token.AccessToken)
	return nil
}

func (p *onedriveProvider) Disconnect() {
	p.store = nil
	os.Remove(onedrive.TokenPath())
}

func (p *onedriveProvider) BrowseFolders(path string) ([]storage.FolderEntry, error) {
	return p.store.ListFolders(storage.NormalizeBrowsePath(path))
}

func (p *onedriveProvider) StorageProvider() storage.Provider {
	if p.store == nil {
		return nil
	}
	return p.store
}
