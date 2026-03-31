package main

import (
	"database/sql"
	"encoding/json"
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
	mux.Handle("/api/events/subagent", corsMiddleware(subagentEventsHandler(database)))
	mux.Handle("/api/sessions", corsMiddleware(sessionsHandler(database)))
	mux.Handle("/api/features", corsMiddleware(featuresHandler(database, htmlgraphDir)))
	mux.Handle("/api/stats", corsMiddleware(statsHandler(database, htmlgraphDir)))
	mux.Handle("/api/initial-stats", corsMiddleware(initialStatsHandler(database)))
	mux.Handle("/api/timeline", corsMiddleware(timelineHandler(database)))
	mux.Handle("/api/transcript", corsMiddleware(transcriptHandler(database)))
	mux.Handle("/api/sessions/", corsMiddleware(sessionIngestHandler(database)))
	mux.Handle("/api/features/detail", corsMiddleware(featureDetailHandler(htmlgraphDir)))
	mux.Handle("/api/features/related", corsMiddleware(relatedFeaturesHandler(database)))

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

// resolvePluginDir finds the go-plugin directory using a priority-ordered
// search strategy. This decouples plugin discovery from the binary's
// filesystem location, supporting Homebrew, go install, curl install, and
// dev-mode symlink workflows.
//
// Search order:
//  1. CLAUDE_PLUGIN_ROOT env var (always set by Claude Code in hook/plugin context)
//  2. HTMLGRAPH_PLUGIN_DIR env var (explicit user override)
//  3. installed_plugins.json installPath (marketplace: ~/.claude/plugins/cache/...)
//  4. Symlink walk-up from binary (dev mode: binary lives inside plugin tree)
//  5. Project-root detection (CWD walk-up: find .htmlgraph/ + plugin/)
func resolvePluginDir() string {
	// 1. CLAUDE_PLUGIN_ROOT — set by Claude Code whenever a hook runs.
	//    This is the authoritative source in hook and plugin context, and
	//    works correctly for both dev-mode symlinks and marketplace installs.
	if root := os.Getenv("CLAUDE_PLUGIN_ROOT"); root != "" {
		if _, err := os.Stat(filepath.Join(root, ".claude-plugin", "plugin.json")); err == nil {
			return root
		}
	}

	// 2. Explicit user override — useful for non-standard installs or testing.
	if dir := os.Getenv("HTMLGRAPH_PLUGIN_DIR"); dir != "" {
		if _, err := os.Stat(filepath.Join(dir, ".claude-plugin", "plugin.json")); err == nil {
			return dir
		}
	}

	// 3. Read installed_plugins.json to find the marketplace install path.
	//    Claude Code stores plugins at ~/.claude/plugins/cache/<marketplace>/<name>/<version>/
	//    and records the exact path in installed_plugins.json.  Iterating the
	//    file is more robust than hard-coding a path that varies by version.
	if dir := resolveMarketplacePluginDir(); dir != "" {
		return dir
	}

	// 4. Symlink walk-up from binary — works for dev mode where the binary
	//    lives at plugin/hooks/bin/htmlgraph (two levels up is
	//    the plugin root).  Fails gracefully when the binary is at
	//    ~/.local/bin/htmlgraph (standalone) or inside the marketplace cache
	//    (already handled above), because those paths have no plugin.json.
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

	// 5. Project-root detection — walk up from CWD to find .htmlgraph/,
	//    then check for plugin/ relative to the project root.
	//    This makes dev mode work from a fresh clone or fork without
	//    needing a marketplace install first.
	if projectPlugin := resolveProjectPluginDir(); projectPlugin != "" {
		return projectPlugin
	}

	return ""
}

// resolveProjectPluginDir walks up from CWD looking for a directory containing
// .htmlgraph/ and plugin/.claude-plugin/plugin.json. Returns the
// plugin dir path or "" if not found.
func resolveProjectPluginDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Walk up at most 5 levels looking for the project root.
	dir := cwd
	for i := 0; i < 5; i++ {
		// Check if this directory has both .htmlgraph/ and plugin/
		pluginDir := filepath.Join(dir, "plugin")
		if _, err := os.Stat(filepath.Join(dir, ".htmlgraph")); err == nil {
			if _, err := os.Stat(filepath.Join(pluginDir, ".claude-plugin", "plugin.json")); err == nil {
				return pluginDir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached filesystem root
		}
		dir = parent
	}
	return ""
}

// resolveMarketplacePluginDir reads ~/.claude/plugins/installed_plugins.json
// and returns the first installPath that has a valid .claude-plugin/plugin.json
// and whose key contains "htmlgraph". Returns "" on any error or miss.
func resolveMarketplacePluginDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(home, ".claude", "plugins", "installed_plugins.json"))
	if err != nil {
		return ""
	}

	var registry struct {
		Plugins map[string][]struct {
			InstallPath string `json:"installPath"`
		} `json:"plugins"`
	}
	if err := json.Unmarshal(data, &registry); err != nil {
		return ""
	}

	for key, entries := range registry.Plugins {
		if !strings.Contains(key, "htmlgraph") {
			continue
		}
		for _, e := range entries {
			if e.InstallPath == "" {
				continue
			}
			candidate := e.InstallPath
			// Resolve symlinks (dev-mode swap replaces the cache dir with a
			// symlink to the source tree — we want the real plugin root).
			if resolved, err := filepath.EvalSymlinks(candidate); err == nil {
				candidate = resolved
			}
			if _, err := os.Stat(filepath.Join(candidate, ".claude-plugin", "plugin.json")); err == nil {
				return candidate
			}
		}
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
	// Filter to current project only — use full CWD path for exact match
	projectFilter := ""
	if cwd, err := os.Getwd(); err == nil {
		projectFilter = cwd
	}
	files, err := ingest.DiscoverSessions(projectFilter)
	if err != nil {
		return
	}
	var newSessions []string
	for _, sf := range files {
		// Check if re-ingest is needed: skip if file hasn't changed since last sync.
		needsIngest := false
		count, _ := dbpkg.CountMessages(database, sf.SessionID)
		if count == 0 {
			needsIngest = true
		} else {
			// Re-ingest if JSONL file modified after last sync.
			var syncedAt string
			database.QueryRow(`SELECT COALESCE(transcript_synced, '') FROM sessions WHERE session_id = ?`,
				sf.SessionID).Scan(&syncedAt)
			if syncedAt != "" {
				if info, err := os.Stat(sf.Path); err == nil {
					synced, _ := time.Parse(time.RFC3339, syncedAt)
					if info.ModTime().After(synced) {
						needsIngest = true
					}
				}
			}
		}
		if !needsIngest {
			continue
		}

		result, err := ingest.ParseFile(sf.Path)
		if err != nil || len(result.Messages) == 0 {
			continue
		}
		if isHeadlessSession(result) {
			continue
		}

		// Clear old messages before re-ingest to avoid duplicates.
		if count > 0 {
			_ = dbpkg.DeleteSessionMessages(database, sf.SessionID)
		}

		ensureSession(database, sf.SessionID, result)
		msgCount, toolCount := storeParseResult(database, sf.SessionID, "", result)
		_ = dbpkg.UpdateTranscriptSync(database, sf.SessionID, sf.Path)
		if msgCount > 0 {
			log.Printf("auto-ingest: %s — %d msgs, %d tools\n",
				truncate(sf.SessionID, 14), msgCount, toolCount)
			newSessions = append(newSessions, sf.SessionID)
		}
	}

	// Update session status from JSONL file mtime (source of truth).
	// "active" if file modified < 5 min ago, "completed" otherwise.
	// Also tag active sessions with launch_mode from .launch-mode file.
	launchMode := ""
	// Find .htmlgraph/.launch-mode relative to CWD
	if cwd, err := os.Getwd(); err == nil {
		if data, err := os.ReadFile(filepath.Join(cwd, ".htmlgraph", ".launch-mode")); err == nil {
			if strings.Contains(string(data), `"yolo`) {
				launchMode = "yolo"
			}
		}
	}

	for _, sf := range files {
		info, err := os.Stat(sf.Path)
		if err != nil {
			continue
		}
		status := "completed"
		if time.Since(info.ModTime()) < 5*time.Minute {
			status = "active"
			// Tag active sessions with the current launch mode
			if launchMode != "" {
				database.Exec(`UPDATE sessions SET metadata = json_set(COALESCE(metadata, '{}'), '$.launch_mode', ?) WHERE session_id = ?`,
					launchMode, sf.SessionID)
			}
		}
		database.Exec(`UPDATE sessions SET status = ? WHERE session_id = ?`, status, sf.SessionID)
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
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
