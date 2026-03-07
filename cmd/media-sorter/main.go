package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"media-sorter/internal/server"
)

//go:embed all:static
var staticFS embed.FS

var version = "dev"

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("media-sorter %s\n", version)
		return
	}

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

	httpServer := &http.Server{Handler: handler}

	fmt.Printf("Media Sorter %s running at %s\n", version, url)

	go openBrowser(url)

	// Graceful shutdown on SIGINT/SIGTERM
	done := make(chan struct{})
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		fmt.Println("\nShutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		httpServer.Shutdown(ctx)
		close(done)
	}()

	if err := httpServer.Serve(listener); err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
	<-done
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
