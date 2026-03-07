package server

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"media-sorter/internal/storage"
	s3store "media-sorter/internal/storage/s3"
)

type s3Provider struct {
	store *s3store.Storage
}

func newS3Provider() *s3Provider {
	p := &s3Provider{}
	if s, err := s3store.New(); err == nil {
		p.store = s
	}
	return p
}

func (p *s3Provider) ID() string          { return "s3" }
func (p *s3Provider) DisplayName() string { return "Amazon S3" }
func (p *s3Provider) PathPrefix() string  { return "s3://" }
func (p *s3Provider) Connected() bool     { return p.store != nil }

func (p *s3Provider) HasCredentials() bool {
	_, err := os.Stat(s3store.CredsPath())
	return err == nil
}

func (p *s3Provider) SaveCredentials(data json.RawMessage) error {
	var creds s3store.Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return fmt.Errorf("Invalid S3 credentials: %w", err)
	}
	if creds.AccessKeyID == "" || creds.SecretAccessKey == "" || creds.Region == "" {
		return fmt.Errorf("accessKeyId, secretAccessKey, and region are required")
	}
	out, _ := json.MarshalIndent(creds, "", "  ")
	return os.WriteFile(s3store.CredsPath(), out, 0600)
}

func (p *s3Provider) Connect(callbackURL, state string) (string, error) {
	store, err := s3store.New()
	if err != nil {
		return "", fmt.Errorf("S3 credentials not configured. Place your credentials JSON file at: %s", s3store.CredsPath())
	}
	p.store = store
	return "", nil // direct connect, no OAuth
}

func (p *s3Provider) HandleCallback(code, callbackURL string) error {
	return fmt.Errorf("S3 does not use OAuth")
}

func (p *s3Provider) Disconnect() {
	p.store = nil
}

func (p *s3Provider) BrowseFolders(path string) ([]FolderEntry, error) {
	if path == "" {
		buckets, err := p.store.ListBuckets()
		if err != nil {
			return nil, err
		}
		var folders []FolderEntry
		for _, b := range buckets {
			folders = append(folders, FolderEntry{Name: b, Path: b})
		}
		return folders, nil
	}

	parts := strings.SplitN(path, "/", 2)
	bucket := parts[0]
	prefix := ""
	if len(parts) > 1 {
		prefix = parts[1]
	}
	subfolders, err := p.store.ListFolders(bucket, prefix)
	if err != nil {
		return nil, err
	}
	var folders []FolderEntry
	for _, f := range subfolders {
		folderPath := bucket + "/"
		if prefix != "" {
			folderPath += prefix + "/"
		}
		folderPath += f
		folders = append(folders, FolderEntry{Name: f, Path: folderPath})
	}
	return folders, nil
}

func (p *s3Provider) StorageProvider() storage.Provider {
	if p.store == nil {
		return nil
	}
	return p.store
}
