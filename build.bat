@echo off
echo Building Media Sorter...

echo Building frontend...
call npm run build

if not exist dist mkdir dist

for /f "tokens=*" %%i in ('git describe --tags --always --dirty 2^>nul') do set VERSION=%%i
if "%VERSION%"=="" set VERSION=dev
set LDFLAGS=-s -w -X main.version=%VERSION%

echo [1/4] Windows (amd64)...
set GOOS=windows
set GOARCH=amd64
go build -ldflags="%LDFLAGS%" -o dist\media-sorter-windows.exe .\cmd\media-sorter

echo [2/4] Mac Intel (amd64)...
set GOOS=darwin
set GOARCH=amd64
go build -ldflags="%LDFLAGS%" -o dist\media-sorter-mac-intel .\cmd\media-sorter

echo [3/4] Mac Apple Silicon (arm64)...
set GOOS=darwin
set GOARCH=arm64
go build -ldflags="%LDFLAGS%" -o dist\media-sorter-mac-arm .\cmd\media-sorter

echo [4/4] Linux (amd64)...
set GOOS=linux
set GOARCH=amd64
go build -ldflags="%LDFLAGS%" -o dist\media-sorter-linux .\cmd\media-sorter

echo.
echo Done! Builds are in the dist/ folder:
dir /b dist\
