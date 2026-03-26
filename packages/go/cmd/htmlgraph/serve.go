package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func serveCmd() *cobra.Command {
	var port int

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start HTTP file server for .htmlgraph/ and project root",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runServe(port)
		},
	}
	cmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to listen on")
	return cmd
}

func runServe(port int) error {
	root, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	addr := fmt.Sprintf("localhost:%d", port)

	// Serve the project root so graph-viewer.html can fetch .htmlgraph/ files.
	mux := http.NewServeMux()
	mux.Handle("/", corsMiddleware(http.FileServer(http.Dir(root))))

	// Convenience: also expose .htmlgraph/ directly under /htmlgraph/
	htmlgraphDir := filepath.Join(root, ".htmlgraph")
	if _, err := os.Stat(htmlgraphDir); err == nil {
		mux.Handle("/htmlgraph/", corsMiddleware(
			http.StripPrefix("/htmlgraph/", http.FileServer(http.Dir(htmlgraphDir))),
		))
	}

	fmt.Printf("Serving %s on http://%s\n", root, addr)
	fmt.Printf("  Project root  →  http://%s/\n", addr)
	fmt.Printf("  Work items    →  http://%s/htmlgraph/\n", addr)
	fmt.Println("\nPress Ctrl+C to stop.")

	return http.ListenAndServe(addr, mux)
}

// corsMiddleware adds permissive CORS headers so in-browser HTML files can
// fetch() local work item files without a same-origin error.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
