package gdrive

// Embedded Google OAuth client credentials for Google Drive.
// Set at build time via ldflags:
//
//	go build -ldflags "-X media-sorter/internal/storage/gdrive.embeddedClientID=YOUR_ID -X media-sorter/internal/storage/gdrive.embeddedClientSecret=YOUR_SECRET" .
var (
	embeddedClientID     string
	embeddedClientSecret string
)
