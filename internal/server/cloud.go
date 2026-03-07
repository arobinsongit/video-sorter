package server

import (
	"encoding/json"

	"media-sorter/internal/storage"
)

// CloudProvider abstracts cloud storage operations for testability.
type CloudProvider interface {
	ID() string
	DisplayName() string
	PathPrefix() string
	Connected() bool
	HasCredentials() bool
	SaveCredentials(data json.RawMessage) error
	// Connect initiates a connection. Returns an auth URL for OAuth providers,
	// or empty string for direct-connect providers (e.g., S3).
	Connect(callbackURL, state string) (authURL string, err error)
	HandleCallback(code, callbackURL string) error
	Disconnect()
	BrowseFolders(path string) ([]storage.FolderEntry, error)
	StorageProvider() storage.Provider
}
