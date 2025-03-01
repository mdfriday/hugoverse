package sse

import (
	"fmt"
	"net/http"
	"path/filepath"
	"runtime"
	"time"
)

// Command represents the SSE test command
type Command struct{}

// Name returns the name of the command
func (c *Command) Name() string {
	return "sse"
}

// Description returns the description of the command
func (c *Command) Description() string {
	return "Start a web server for testing SSE implementation"
}

// Run executes the command
func (c *Command) Run() error {
	return runServer()
}

func runServer() error {
	// Get the current file's directory
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("failed to get current file path")
	}
	dir := filepath.Dir(filename)

	// Serve static files
	http.Handle("/", http.FileServer(http.Dir(filepath.Join(dir, "static"))))

	// Handle SSE endpoint
	http.HandleFunc("/deploy", handleDeploy)

	fmt.Println("Server started at http://localhost:8080")
	return http.ListenAndServe(":8080", nil)
}

func handleDeploy(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Create a channel to simulate progress updates
	done := make(chan bool)
	totalSize := int64(1024 * 1024 * 100) // 100MB for testing
	currentSize := int64(0)

	// Send progress updates
	go func() {
		for currentSize < totalSize {
			// Simulate progress
			currentSize += int64(1024 * 1024 * 5) // 5MB chunks
			if currentSize > totalSize {
				currentSize = totalSize
			}

			// Send progress event
			fmt.Fprintf(w, "event: progress\ndata: {\"current\": %d, \"total\": %d}\n\n", currentSize, totalSize)
			w.(http.Flusher).Flush()

			time.Sleep(500 * time.Millisecond)
		}

		// Send completion event
		fmt.Fprintf(w, "event: complete\ndata: {\"message\": \"Upload completed\"}\n\n")
		w.(http.Flusher).Flush()
		done <- true
	}()

	// Wait for completion or client disconnect
	select {
	case <-done:
		return
	case <-r.Context().Done():
		return
	}
}
