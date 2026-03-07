package main

// Embedded Google OAuth client credentials for Google Drive.
// Set at build time via ldflags:
//
//	go build -ldflags "-X main.embeddedGDriveClientID=YOUR_ID -X main.embeddedGDriveClientSecret=YOUR_SECRET" .
//
// Or set env vars GDRIVE_CLIENT_ID / GDRIVE_CLIENT_SECRET before building:
//
//	go build -ldflags "-X main.embeddedGDriveClientID=$GDRIVE_CLIENT_ID -X main.embeddedGDriveClientSecret=$GDRIVE_CLIENT_SECRET" .
//
// Can also be overridden at runtime by placing a credentials JSON at ~/.media-sorter/gdrive-credentials.json
var (
	embeddedGDriveClientID     string
	embeddedGDriveClientSecret string
)
