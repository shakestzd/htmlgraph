package main

import (
	"database/sql"
	"net/http"
	"strings"
)

// provenanceResponse is the JSON shape for /api/provenance/{id}.
type provenanceResponse struct {
	Node       provenanceNode   `json:"node"`
	Upstream   []provenanceLink `json:"upstream"`
	Downstream []provenanceLink `json:"downstream"`
}

type provenanceNode struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

type provenanceLink struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Title        string `json:"title"`
	Relationship string `json:"relationship"`
}

// commitResult is the JSON shape for /api/graph/commits items.
type commitResult struct {
	CommitHash string `json:"commit_hash"`
	Message    string `json:"message"`
	SessionID  string `json:"session_id"`
	Timestamp  string `json:"timestamp"`
}

// fileResult is the JSON shape for /api/graph/files items.
type fileResult struct {
	FilePath   string `json:"file_path"`
	SessionID  string `json:"session_id"`
	ChangeType string `json:"change_type"`
}

// sessionResult is the JSON shape for /api/graph/sessions items.
type sessionResult struct {
	SessionID string `json:"session_id"`
	Agent     string `json:"agent"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// provenanceHandler handles GET /api/provenance/{id}.
// It returns the node's metadata plus upstream and downstream causal links.
func provenanceHandler(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/provenance/")
		if id == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			respondJSON(w, map[string]string{"error": "not found"})
			return
		}

		node, ok := resolveProvenanceNode(database, id)
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			respondJSON(w, map[string]string{"error": "not found"})
			return
		}

		upstream := loadUpstreamLinks(database, id)
		downstream := loadDownstreamLinks(database, id)

		respondJSON(w, provenanceResponse{
			Node:       node,
			Upstream:   upstream,
			Downstream: downstream,
		})
	}
}

// resolveProvenanceNode looks up node metadata from features, sessions, or tracks tables.
func resolveProvenanceNode(database *sql.DB, id string) (provenanceNode, bool) {
	// Try features table (includes features, bugs, spikes).
	var node provenanceNode
	err := database.QueryRow(
		`SELECT id, COALESCE(type,'feature'), COALESCE(title,''), COALESCE(status,'todo')
		 FROM features WHERE id = ?`, id,
	).Scan(&node.ID, &node.Type, &node.Title, &node.Status)
	if err == nil {
		return node, true
	}

	// Try tracks table.
	err = database.QueryRow(
		`SELECT id, 'track', COALESCE(title,''), COALESCE(status,'todo')
		 FROM tracks WHERE id = ?`, id,
	).Scan(&node.ID, &node.Type, &node.Title, &node.Status)
	if err == nil {
		return node, true
	}

	// Try sessions table.
	err = database.QueryRow(
		`SELECT session_id, 'session', COALESCE(title,''), COALESCE(status,'')
		 FROM sessions WHERE session_id = ?`, id,
	).Scan(&node.ID, &node.Type, &node.Title, &node.Status)
	if err == nil {
		return node, true
	}

	// Try commit nodes.
	var hash, msg string
	err = database.QueryRow(
		`SELECT commit_hash, COALESCE(message,'') FROM git_commits WHERE commit_hash = ? LIMIT 1`, id,
	).Scan(&hash, &msg)
	if err == nil {
		return provenanceNode{ID: hash, Type: "commit", Title: msg, Status: "done"}, true
	}

	// Try file nodes.
	var filePath string
	err = database.QueryRow(
		`SELECT file_path FROM feature_files WHERE file_path = ? LIMIT 1`, id,
	).Scan(&filePath)
	if err == nil {
		return provenanceNode{ID: filePath, Type: "file", Title: filePath, Status: ""}, true
	}

	return provenanceNode{}, false
}

// loadUpstreamLinks returns nodes that point TO the given id via graph_edges.
func loadUpstreamLinks(database *sql.DB, id string) []provenanceLink {
	rows, err := database.Query(
		`SELECT from_node_id, relationship_type FROM graph_edges WHERE to_node_id = ?`, id,
	)
	if err != nil {
		return []provenanceLink{}
	}
	defer rows.Close()

	var links []provenanceLink
	for rows.Next() {
		var fromID, rel string
		if err := rows.Scan(&fromID, &rel); err != nil {
			continue
		}
		node, ok := resolveProvenanceNode(database, fromID)
		link := provenanceLink{ID: fromID, Relationship: rel}
		if ok {
			link.Type = node.Type
			link.Title = node.Title
		}
		links = append(links, link)
	}
	if links == nil {
		links = []provenanceLink{}
	}
	return links
}

// loadDownstreamLinks returns nodes that the given id points TO via graph_edges.
func loadDownstreamLinks(database *sql.DB, id string) []provenanceLink {
	rows, err := database.Query(
		`SELECT to_node_id, relationship_type FROM graph_edges WHERE from_node_id = ?`, id,
	)
	if err != nil {
		return []provenanceLink{}
	}
	defer rows.Close()

	var links []provenanceLink
	for rows.Next() {
		var toID, rel string
		if err := rows.Scan(&toID, &rel); err != nil {
			continue
		}
		node, ok := resolveProvenanceNode(database, toID)
		link := provenanceLink{ID: toID, Relationship: rel}
		if ok {
			link.Type = node.Type
			link.Title = node.Title
		}
		links = append(links, link)
	}
	if links == nil {
		links = []provenanceLink{}
	}
	return links
}

// commitsForFeatureHandler handles GET /api/graph/commits?feature=X.
func commitsForFeatureHandler(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		featureID := r.URL.Query().Get("feature")
		rows, err := database.Query(
			`SELECT commit_hash, COALESCE(message,''), COALESCE(session_id,''), COALESCE(timestamp,'')
			 FROM git_commits WHERE feature_id = ?
			 ORDER BY timestamp DESC`, featureID,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		results := []commitResult{}
		for rows.Next() {
			var c commitResult
			if err := rows.Scan(&c.CommitHash, &c.Message, &c.SessionID, &c.Timestamp); err != nil {
				continue
			}
			results = append(results, c)
		}
		respondJSON(w, results)
	}
}

// filesForFeatureHandler handles GET /api/graph/files?feature=X.
func filesForFeatureHandler(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		featureID := r.URL.Query().Get("feature")
		rows, err := database.Query(
			`SELECT file_path, COALESCE(session_id,''), COALESCE(operation,'')
			 FROM feature_files WHERE feature_id = ?
			 ORDER BY file_path`, featureID,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		results := []fileResult{}
		for rows.Next() {
			var f fileResult
			if err := rows.Scan(&f.FilePath, &f.SessionID, &f.ChangeType); err != nil {
				continue
			}
			results = append(results, f)
		}
		respondJSON(w, results)
	}
}

// sessionsForFeatureHandler handles GET /api/graph/sessions?feature=X.
func sessionsForFeatureHandler(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		featureID := r.URL.Query().Get("feature")
		rows, err := database.Query(
			`SELECT DISTINCT s.session_id, COALESCE(s.agent_assigned,''), COALESCE(s.status,''),
			        COALESCE(s.created_at,'')
			 FROM sessions s
			 JOIN agent_events e ON e.session_id = s.session_id
			 WHERE e.feature_id = ?`, featureID,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		results := []sessionResult{}
		for rows.Next() {
			var s sessionResult
			if err := rows.Scan(&s.SessionID, &s.Agent, &s.Status, &s.CreatedAt); err != nil {
				continue
			}
			results = append(results, s)
		}
		respondJSON(w, results)
	}
}
