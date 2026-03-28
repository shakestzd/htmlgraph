package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	dbpkg "github.com/shakestzd/htmlgraph/internal/db"
	"github.com/shakestzd/htmlgraph/internal/ingest"
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

	dbPath := filepath.Join(htmlgraphDir, "htmlgraph.db")
	database, err := dbpkg.Open(dbPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer database.Close()

	// Auto-ingest transcripts on startup and every 60s.
	go autoIngestLoop(database)

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
	mux.Handle("/api/transcript", corsMiddleware(transcriptHandler(database)))

	// .htmlgraph/ files accessible under /htmlgraph/
	mux.Handle("/htmlgraph/", corsMiddleware(
		http.StripPrefix("/htmlgraph/", http.FileServer(http.Dir(htmlgraphDir))),
	))

	// Serve embedded dashboard (index.html, css/, js/, components/)
	mux.Handle("/", corsMiddleware(http.FileServer(http.FS(dashboardSub()))))

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
	if resolved, err := filepath.EvalSymlinks(binPath); err == nil {
		binPath = resolved
	}
	pluginDir := filepath.Join(filepath.Dir(binPath), "..", "..")
	pluginDir, _ = filepath.Abs(pluginDir)
	if _, err := os.Stat(filepath.Join(pluginDir, ".claude-plugin", "plugin.json")); err == nil {
		return pluginDir
	}
	return ""
}

// autoIngestLoop runs transcript ingestion immediately, then every 60 seconds.
func autoIngestLoop(database *sql.DB) {
	for {
		autoIngestOnce(database)
		time.Sleep(60 * time.Second)
	}
}

// autoIngestOnce discovers session files and ingests any that are new.
func autoIngestOnce(database *sql.DB) {
	files, err := ingest.DiscoverSessions("")
	if err != nil {
		return
	}
	var newSessions []string
	for _, sf := range files {
		count, _ := dbpkg.CountMessages(database, sf.SessionID)
		if count > 0 {
			continue
		}
		result, err := ingest.ParseFile(sf.Path)
		if err != nil || len(result.Messages) == 0 {
			continue
		}
		// Skip headless claude -p sessions (e.g. titler calls) — they have
		// very few messages and a haiku model.
		if isHeadlessSession(result) {
			continue
		}
		ensureSession(database, sf.SessionID, result)
		msgCount, toolCount := storeParseResult(database, sf.SessionID, result)
		_ = dbpkg.UpdateTranscriptSync(database, sf.SessionID, sf.Path)
		if msgCount > 0 {
			log.Printf("auto-ingest: %s — %d msgs, %d tools\n",
				truncate(sf.SessionID, 14), msgCount, toolCount)
			newSessions = append(newSessions, sf.SessionID)
		}
	}

	// Generate titles for newly ingested sessions (runs sequentially to
	// avoid hammering claude CLI).
	for _, sid := range newSessions {
		generateTitle(database, sid)
	}

	// Also title any older sessions that are missing titles.
	rows, err := database.Query(`
		SELECT s.session_id FROM sessions s
		WHERE (s.title IS NULL OR s.title = '')
		  AND EXISTS (SELECT 1 FROM messages m WHERE m.session_id = s.session_id)
		LIMIT 5`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var sid string
			if rows.Scan(&sid) == nil {
				generateTitle(database, sid)
			}
		}
	}
}

// isHeadlessSession returns true if the session was created by the
// htmlgraph titler (claude -p calls). Detected by the [htmlgraph-titler]
// marker in the first user message.
func isHeadlessSession(result *ingest.ParseResult) bool {
	for _, m := range result.Messages {
		if m.Role == "user" {
			return strings.Contains(m.Content, "[htmlgraph-titler]") ||
				strings.Contains(m.Content, "Generate a concise 4-8 word title for this AI coding session")
		}
	}
	return false
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
