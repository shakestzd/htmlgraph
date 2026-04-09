// Package main — `htmlgraph serve --global` cross-project dashboard backend.
//
// The global server loads the registry at startup and on every /api/projects
// request (re-reading the ~1 ms JSON file so newly-registered projects appear
// without a restart). It opens each project DB lazily and read-only via
// registry.OpenReadOnly, caching *sql.DB handles keyed by the stable 8-char
// project ID. Existing single-project handlers (sessionsHandler,
// featuresHandler, statsHandler) are reused by extracting `?project=<id>`
// from the request and dispatching with the matched DB. All cross-project
// access is strictly read-only — no migrations, no schema mutations.
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/shakestzd/htmlgraph/internal/registry"
)

// projectCache holds lazily-opened, read-only *sql.DB handles keyed by the
// stable 8-char project ID from the registry entry.
type projectCache struct {
	mu  sync.Mutex
	dbs map[string]*sql.DB
	// dirs tracks the filesystem path per project ID so handlers that need
	// htmlgraphDir (featuresHandler, statsHandler, etc.) can resolve it.
	dirs map[string]string
}

func newProjectCache() *projectCache {
	return &projectCache{
		dbs:  make(map[string]*sql.DB),
		dirs: make(map[string]string),
	}
}

// get returns a cached *sql.DB for the given entry, opening it lazily on
// first access. Returns nil if the project .htmlgraph directory is missing
// or the DB cannot be opened read-only.
func (c *projectCache) get(e registry.Entry) (*sql.DB, string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if db, ok := c.dbs[e.ID]; ok {
		return db, c.dirs[e.ID]
	}
	hgDir := filepath.Join(e.ProjectDir, ".htmlgraph")
	if _, err := os.Stat(hgDir); err != nil {
		return nil, ""
	}
	dbPath := filepath.Join(hgDir, "htmlgraph.db")
	db, err := registry.OpenReadOnly(dbPath)
	if err != nil {
		return nil, ""
	}
	c.dbs[e.ID] = db
	c.dirs[e.ID] = hgDir
	return db, hgDir
}

// closeAll closes every cached DB handle. Best-effort — errors are ignored.
func (c *projectCache) closeAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, db := range c.dbs {
		_ = db.Close()
	}
	c.dbs = nil
	c.dirs = nil
}

// projectSummary is the JSON shape returned by /api/projects entries.
type projectSummary struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Dir          string `json:"dir"`
	LastSeen     string `json:"lastSeen"`
	GitRemoteURL string `json:"gitRemoteURL,omitempty"`
	FeatureCount int    `json:"featureCount"`
	BugCount     int    `json:"bugCount"`
	SpikeCount   int    `json:"spikeCount"`
}

// buildGlobalMux constructs the http.ServeMux for the global server. Split
// out so tests can drive it via httptest.NewServer without binding a port.
func buildGlobalMux() *http.ServeMux {
	cache := newProjectCache()
	mux := http.NewServeMux()

	// /api/mode — dashboard calls this on startup to detect global mode.
	mux.Handle("/api/mode", corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		summaries := loadProjectSummaries(cache)
		respondJSON(w, map[string]any{
			"mode":     "global",
			"projects": summaries,
		})
	})))

	// /api/projects — re-reads the registry every call so newly-registered
	// projects appear without a server restart.
	mux.Handle("/api/projects", corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		respondJSON(w, loadProjectSummaries(cache))
	})))

	// /api/projects/all/stats — aggregated stats across every loaded project.
	mux.Handle("/api/projects/all/stats", corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		respondJSON(w, aggregateStats(cache))
	})))

	// /api/stats — aggregate when no ?project, dispatch when ?project=<id>.
	mux.Handle("/api/stats", corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("project") == "" {
			respondJSON(w, aggregateStats(cache))
			return
		}
		dispatchByProject(cache, func(db *sql.DB, dir string) http.HandlerFunc {
			return statsHandler(db, dir)
		}).ServeHTTP(w, r)
	})))

	// Per-project routes: reuse the existing single-project handlers by
	// dispatching with the DB matched from ?project=<id>.
	mux.Handle("/api/sessions", corsMiddleware(dispatchByProject(cache, func(db *sql.DB, _ string) http.HandlerFunc {
		return sessionsHandler(db)
	})))
	mux.Handle("/api/features/detail", corsMiddleware(dispatchByProject(cache, func(_ *sql.DB, dir string) http.HandlerFunc {
		return featureDetailHandler(dir)
	})))
	mux.Handle("/api/features/related", corsMiddleware(dispatchByProject(cache, func(db *sql.DB, _ string) http.HandlerFunc {
		return relatedFeaturesHandler(db)
	})))
	// /api/features/ must come after the more-specific /api/features/* prefixes.
	mux.Handle("/api/features/", corsMiddleware(dispatchByProject(cache, func(db *sql.DB, _ string) http.HandlerFunc {
		return featureActivityHandler(db)
	})))
	mux.Handle("/api/features", corsMiddleware(dispatchByProject(cache, func(db *sql.DB, dir string) http.HandlerFunc {
		return featuresHandler(db, dir)
	})))
	mux.Handle("/api/graph", corsMiddleware(dispatchByProject(cache, func(db *sql.DB, _ string) http.HandlerFunc {
		return graphAPIHandler(db)
	})))
	mux.Handle("/api/events/tree", corsMiddleware(dispatchByProject(cache, func(db *sql.DB, _ string) http.HandlerFunc {
		return treeHandler(db)
	})))
	mux.Handle("/api/events/subagent", corsMiddleware(dispatchByProject(cache, func(db *sql.DB, _ string) http.HandlerFunc {
		return subagentEventsHandler(db)
	})))
	// /api/events/recent — aggregate across all projects when no ?project,
	// otherwise dispatch to the single-project handler.
	mux.Handle("/api/events/recent", corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("project") == "" {
			globalRecentEventsHandler(cache, w, r)
			return
		}
		dispatchByProject(cache, func(db *sql.DB, _ string) http.HandlerFunc {
			return recentEventsHandler(db)
		}).ServeHTTP(w, r)
	})))
	// /api/events/stream — SSE poll-fallback. When ?project=<id> is present,
	// stream only that project. Without ?project, fan across all loaded DBs.
	mux.Handle("/api/events/stream", corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("project") == "" {
			globalSSEHandler(cache, w, r)
			return
		}
		dispatchByProject(cache, func(db *sql.DB, _ string) http.HandlerFunc {
			return sseHandler(db)
		}).ServeHTTP(w, r)
	})))
	// CRISPI plan routes — list must precede the per-plan catch-all.
	mux.Handle("/api/plans", corsMiddleware(dispatchByProject(cache, func(db *sql.DB, dir string) http.HandlerFunc {
		return plansListHandler(dir, db)
	})))
	mux.Handle("/api/plans/", corsMiddleware(dispatchByProject(cache, func(db *sql.DB, dir string) http.HandlerFunc {
		return planRouter(db, dir)
	})))
	mux.Handle("/api/transcript", corsMiddleware(dispatchByProject(cache, func(db *sql.DB, dir string) http.HandlerFunc {
		return transcriptHandler(db, dir)
	})))
	mux.Handle("/api/sessions/", corsMiddleware(dispatchByProject(cache, func(db *sql.DB, _ string) http.HandlerFunc {
		return sessionIngestHandler(db)
	})))

	// Serve the embedded dashboard SPA (same assets as single-project mode).
	// The frontend calls /api/mode on startup to detect global mode and
	// render the project switcher.
	mux.Handle("/", corsMiddleware(http.FileServer(http.FS(dashboardSub()))))

	return mux
}

// loadProjectSummaries re-reads the registry on each call and returns one
// summary per entry whose .htmlgraph directory exists on disk. DB handles
// are cached across calls via the projectCache.
func loadProjectSummaries(cache *projectCache) []projectSummary {
	reg, err := registry.Load(registry.DefaultPath())
	if err != nil {
		return nil
	}
	var out []projectSummary
	for _, e := range reg.List() {
		db, _ := cache.get(e)
		if db == nil {
			continue
		}
		s := projectSummary{
			ID:           e.ID,
			Name:         e.Name,
			Dir:          e.ProjectDir,
			LastSeen:     e.LastSeen,
			GitRemoteURL: e.GitRemoteURL,
		}
		s.FeatureCount, s.BugCount, s.SpikeCount = countByType(db)
		out = append(out, s)
	}
	return out
}

// countByType runs a single COUNT query against a read-only project DB and
// returns feature/bug/spike counts. Returns zeros on any query error.
func countByType(db *sql.DB) (int, int, int) {
	var f, b, s int
	row := db.QueryRow(`SELECT
		COALESCE(SUM(CASE WHEN type = 'feature' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN type = 'bug'     THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN type = 'spike'   THEN 1 ELSE 0 END), 0)
		FROM features`)
	if err := row.Scan(&f, &b, &s); err != nil {
		return 0, 0, 0
	}
	return f, b, s
}

// aggregateStats sums counts across every cached project DB.
func aggregateStats(cache *projectCache) map[string]any {
	summaries := loadProjectSummaries(cache)
	var f, b, s int
	for _, p := range summaries {
		f += p.FeatureCount
		b += p.BugCount
		s += p.SpikeCount
	}
	return map[string]any{
		"project_count": len(summaries),
		"features":      f,
		"bugs":          b,
		"spikes":        s,
	}
}

// globalRecentEventsHandler returns a merged list of recent events from all
// loaded project DBs, ordered by timestamp DESC and limited to N rows.
func globalRecentEventsHandler(cache *projectCache, w http.ResponseWriter, r *http.Request) {
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		var n int
		if _, err := fmt.Sscanf(l, "%d", &n); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	summaries := loadProjectSummaries(cache)
	type evtRow struct {
		EventID       string `json:"event_id"`
		AgentID       string `json:"agent_id"`
		EventType     string `json:"event_type"`
		Timestamp     string `json:"timestamp"`
		ToolName      string `json:"tool_name"`
		InputSummary  string `json:"input_summary"`
		OutputSummary string `json:"output_summary"`
		SessionID     string `json:"session_id"`
		FeatureID     string `json:"feature_id"`
		FeatureTitle  string `json:"feature_title"`
		ParentEventID string `json:"parent_event_id"`
		Status        string `json:"status"`
	}
	var all []evtRow

	for _, s := range summaries {
		cache.mu.Lock()
		db := cache.dbs[s.ID]
		cache.mu.Unlock()
		if db == nil {
			continue
		}
		rows, err := db.Query(`
			SELECT e.event_id, e.agent_id, e.event_type, e.timestamp, e.tool_name,
			       COALESCE(e.input_summary, ''), COALESCE(e.output_summary, ''),
			       e.session_id, COALESCE(e.feature_id, ''),
			       COALESCE(e.parent_event_id, ''), e.status,
			       COALESCE((SELECT f.title FROM features f WHERE f.id = e.feature_id LIMIT 1), '')
			FROM agent_events e
			ORDER BY e.timestamp DESC
			LIMIT ?`, limit)
		if err != nil {
			continue
		}
		for rows.Next() {
			var ev evtRow
			_ = rows.Scan(&ev.EventID, &ev.AgentID, &ev.EventType, &ev.Timestamp,
				&ev.ToolName, &ev.InputSummary, &ev.OutputSummary,
				&ev.SessionID, &ev.FeatureID, &ev.ParentEventID, &ev.Status, &ev.FeatureTitle)
			all = append(all, ev)
		}
		rows.Close()
	}

	// Sort merged slice by timestamp DESC (lexicographic on RFC3339 works).
	for i := 1; i < len(all); i++ {
		for j := i; j > 0 && all[j].Timestamp > all[j-1].Timestamp; j-- {
			all[j], all[j-1] = all[j-1], all[j]
		}
	}
	if len(all) > limit {
		all = all[:limit]
	}
	if all == nil {
		all = []evtRow{}
	}
	respondJSON(w, all)
}

// globalSSEHandler opens an SSE stream and pushes merged events from all
// loaded project DBs every 2 seconds. This is the poll-fallback approach —
// it keeps the browser's EventSource alive (no "Reconnecting…" loop) while
// aggregating across every cached project DB.
func globalSSEHandler(cache *projectCache, w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Send an initial "connected" comment so the client transitions to open state.
	fmt.Fprintf(w, ": connected\n\n")
	flusher.Flush()

	// Track the highest rowid seen per project DB to avoid re-sending rows.
	lastRowIDs := make(map[string]int64)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			summaries := loadProjectSummaries(cache)
			for _, s := range summaries {
				cache.mu.Lock()
				db := cache.dbs[s.ID]
				cache.mu.Unlock()
				if db == nil {
					continue
				}
				db.Exec("PRAGMA wal_checkpoint(PASSIVE)")
				rows, err := db.Query(`
					SELECT rowid, event_id, agent_id, event_type, timestamp,
					       tool_name, COALESCE(input_summary, ''),
					       COALESCE(output_summary, ''), session_id,
					       COALESCE(feature_id, ''), status
					FROM agent_events
					WHERE rowid > ?
					ORDER BY rowid ASC
					LIMIT 20`, lastRowIDs[s.ID])
				if err != nil {
					continue
				}
				for rows.Next() {
					var rowid int64
					var eid, aid, etype, ts, tool, inputSum, outputSum, sid, fid, status string
					if err := rows.Scan(&rowid, &eid, &aid, &etype, &ts,
						&tool, &inputSum, &outputSum, &sid, &fid, &status); err != nil {
						continue
					}
					payload, _ := json.Marshal(map[string]string{
						"event_id":       eid,
						"agent_id":       aid,
						"event_type":     etype,
						"timestamp":      ts,
						"tool_name":      tool,
						"input_summary":  inputSum,
						"output_summary": outputSum,
						"session_id":     sid,
						"feature_id":     fid,
						"status":         status,
					})
					fmt.Fprintf(w, "data: %s\n\n", payload)
					lastRowIDs[s.ID] = rowid
				}
				rows.Close()
			}
			flusher.Flush()
		}
	}
}

// dispatchByProject returns an http.Handler that extracts the `project`
// query parameter, looks up the matching DB in the cache, and dispatches
// to the per-project handler constructed via factory.
func dispatchByProject(cache *projectCache, factory func(*sql.DB, string) http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("project")
		if id == "" {
			http.Error(w, "project query parameter required in global mode", http.StatusBadRequest)
			return
		}
		// Re-read the registry so a freshly-registered project can be
		// resolved without a server restart. The DB handle itself is
		// still cached and reused.
		reg, err := registry.Load(registry.DefaultPath())
		if err != nil {
			http.Error(w, "load registry: "+err.Error(), http.StatusInternalServerError)
			return
		}
		for _, e := range reg.List() {
			if e.ID != id {
				continue
			}
			db, dir := cache.get(e)
			if db == nil {
				http.Error(w, "project .htmlgraph not accessible", http.StatusNotFound)
				return
			}
			factory(db, dir).ServeHTTP(w, r)
			return
		}
		http.Error(w, "unknown project id", http.StatusNotFound)
	})
}
