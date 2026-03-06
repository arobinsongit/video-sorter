#!/bin/bash
echo "Building Video Sorter..."

echo "Building frontend..."
npm run build

mkdir -p dist

echo "[1/4] Windows (amd64)..."
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o dist/video-sorter-windows.exe .

echo "[2/4] Mac Intel (amd64)..."
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o dist/video-sorter-mac-intel .

echo "[3/4] Mac Apple Silicon (arm64)..."
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o dist/video-sorter-mac-arm .

echo "[4/4] Linux (amd64)..."
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/video-sorter-linux .

echo ""
echo "Done! Builds are in the dist/ folder:"
ls -lh dist/
