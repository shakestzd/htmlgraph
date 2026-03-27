package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/shakestzd/htmlgraph/internal/db"
	"github.com/spf13/cobra"
)

func serveCmd() *cobra.Command {
	var port int

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start HTTP dashboard server with SSE event stream",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runServer(port)
		},
	}
	cmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to listen on")
	return cmd
}

func runServer(port int) error {
	htmlgraphDir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}

	// Project root is the parent of .htmlgraph/
	root := filepath.Dir(htmlgraphDir)

	dbPath := filepath.Join(htmlgraphDir, "htmlgraph.db")
	database, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer database.Close()

	mux := http.NewServeMux()

	// API endpoints registered before file server so they take precedence.
	mux.Handle("/api/events/recent", corsMiddleware(recentEventsHandler(database)))
	mux.Handle("/api/events/tree", corsMiddleware(treeHandler(database)))
	mux.Handle("/api/events/stream", corsMiddleware(sseHandler(database)))
	mux.Handle("/api/sessions", corsMiddleware(sessionsHandler(database)))
	mux.Handle("/api/features", corsMiddleware(featuresHandler(database, htmlgraphDir)))
	mux.Handle("/api/stats", corsMiddleware(statsHandler(database, htmlgraphDir)))
	mux.Handle("/api/initial-stats", corsMiddleware(initialStatsHandler(database)))
	mux.Handle("/api/timeline", corsMiddleware(timelineHandler(database)))

	// Serve dashboard from Go plugin directory (primary) or project root (fallback).
	// Also serve static assets (components.js, CSS) from the Go plugin.
	pluginDir := resolvePluginDir()
	if pluginDir != "" {
		staticDir := filepath.Join(pluginDir, "static")
		if info, err := os.Stat(staticDir); err == nil && info.IsDir() {
			mux.Handle("/static/", corsMiddleware(
				http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))),
			))
		}
	}

	// .htmlgraph/ files accessible under /htmlgraph/
	mux.Handle("/htmlgraph/", corsMiddleware(
		http.StripPrefix("/htmlgraph/", http.FileServer(http.Dir(htmlgraphDir))),
	))

	// Project root file server — serves graph-viewer.html, index.html, etc.
	mux.Handle("/", corsMiddleware(http.FileServer(http.Dir(root))))

	addr := fmt.Sprintf("localhost:%d", port)
	fmt.Printf("HtmlGraph Dashboard:  http://%s/\n", addr)
	fmt.Printf("Graph Viewer:         http://%s/graph-viewer.html\n", addr)
	fmt.Printf("API Stats:            http://%s/api/stats\n", addr)
	fmt.Printf("SSE Stream:           http://%s/api/events/stream\n", addr)
	fmt.Println("Press Ctrl+C to stop.")

	return http.ListenAndServe(addr, mux)
}

// resolvePluginDir finds the go-plugin directory relative to the binary.
func resolvePluginDir() string {
	binPath, err := os.Executable()
	if err != nil {
		return ""
	}
	// Resolve symlinks (e.g., .venv/bin/htmlgraph → packages/go-plugin/hooks/bin/htmlgraph)
	if resolved, err := filepath.EvalSymlinks(binPath); err == nil {
		binPath = resolved
	}
	// Binary at packages/go-plugin/hooks/bin/htmlgraph → plugin at ../..
	pluginDir := filepath.Join(filepath.Dir(binPath), "..", "..")
	pluginDir, _ = filepath.Abs(pluginDir)
	if _, err := os.Stat(filepath.Join(pluginDir, ".claude-plugin", "plugin.json")); err == nil {
		return pluginDir
	}
	return ""
}

// corsMiddleware adds permissive CORS headers so in-browser HTML files can
// fetch() local work-item files without a same-origin error.
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
