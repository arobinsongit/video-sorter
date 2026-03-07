package main

import (
	"embed"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"

	"video-sorter/internal/server"
)

//go:embed all:static
var staticFS embed.FS

func main() {
	srv := server.New()

	staticContent, _ := fs.Sub(staticFS, "static")
	handler := srv.Handler(staticContent)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to find free port: %v\n", err)
		os.Exit(1)
	}
	srv.Port = listener.Addr().(*net.TCPAddr).Port
	url := fmt.Sprintf("http://127.0.0.1:%d", srv.Port)

	fmt.Printf("Media Sorter running at %s\n", url)

	go openBrowser(url)

	http.Serve(listener, handler)
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	cmd.Run()
}
