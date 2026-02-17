package web

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os/exec"
	"runtime"

	"github.com/alg/crev/internal/diff"
	"github.com/alg/crev/internal/review"
)

//go:embed static/*
var staticFiles embed.FS

// Result holds the outcome of the web review session
type Result struct {
	Review    *review.Review
	Submitted bool
}

// Start launches the web review server and blocks until submission or cancellation
func Start(d *diff.Diff, baseCommit string) (*Result, error) {
	// Pick a random available port
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	url := fmt.Sprintf("http://localhost:%d", port)

	// Channel to receive the submitted review
	submitCh := make(chan *review.Review, 1)

	// Set up routes
	mux := http.NewServeMux()

	h := &handlers{
		diff:       d,
		baseCommit: baseCommit,
		submitCh:   submitCh,
	}

	mux.HandleFunc("GET /api/diff", h.handleDiff)
	mux.HandleFunc("GET /api/file", h.handleFile)
	mux.HandleFunc("POST /api/submit", h.handleSubmit)

	// Serve static files
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return nil, fmt.Errorf("failed to create static fs: %w", err)
	}
	mux.Handle("GET /", http.FileServer(http.FS(staticFS)))

	server := &http.Server{Handler: mux}

	// Start server in background
	go func() {
		if err := server.Serve(listener); err != http.ErrServerClosed {
			fmt.Printf("Server error: %v\n", err)
		}
	}()

	fmt.Printf("Review server started at %s\n", url)
	openBrowser(url)

	// Wait for submission
	rev := <-submitCh

	// Shutdown server
	server.Shutdown(context.Background())

	if rev == nil {
		return &Result{Submitted: false}, nil
	}

	return &Result{Review: rev, Submitted: true}, nil
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		fmt.Printf("Open %s in your browser\n", url)
		return
	}
	cmd.Start()
}
