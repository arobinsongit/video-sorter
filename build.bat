@echo off
echo Building Video Sorter...
if not exist dist mkdir dist

echo [1/4] Windows (amd64)...
set GOOS=windows
set GOARCH=amd64
go build -ldflags="-s -w" -o dist\video-sorter-windows.exe .

echo [2/4] Mac Intel (amd64)...
set GOOS=darwin
set GOARCH=amd64
go build -ldflags="-s -w" -o dist\video-sorter-mac-intel .

echo [3/4] Mac Apple Silicon (arm64)...
set GOOS=darwin
set GOARCH=arm64
go build -ldflags="-s -w" -o dist\video-sorter-mac-arm .

echo [4/4] Linux (amd64)...
set GOOS=linux
set GOARCH=amd64
go build -ldflags="-s -w" -o dist\video-sorter-linux .

echo.
echo Done! Builds are in the dist/ folder:
dir /b dist\
