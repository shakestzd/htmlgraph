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
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"

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

// runGlobalServer starts the multi-project dashboard on the given port.
// All project DB access is read-only; no foreign DBs are mutated.
func runGlobalServer(port int) error {
	mux := buildGlobalMux()
	addr := fmt.Sprintf("localhost:%d", port)
	fmt.Printf("HtmlGraph Global Dashboard:  http://%s/\n", addr)
	fmt.Printf("API Projects:                http://%s/api/projects\n", addr)
	fmt.Println("Press Ctrl+C to stop.")
	return http.ListenAndServe(addr, mux)
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

	// Per-project routes: reuse the existing single-project handlers by
	// dispatching with the DB matched from ?project=<id>.
	mux.Handle("/api/sessions", corsMiddleware(dispatchByProject(cache, func(db *sql.DB, _ string) http.HandlerFunc {
		return sessionsHandler(db)
	})))
	mux.Handle("/api/features", corsMiddleware(dispatchByProject(cache, func(db *sql.DB, dir string) http.HandlerFunc {
		return featuresHandler(db, dir)
	})))
	mux.Handle("/api/stats", corsMiddleware(dispatchByProject(cache, func(db *sql.DB, dir string) http.HandlerFunc {
		return statsHandler(db, dir)
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
