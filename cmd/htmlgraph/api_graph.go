package main

import (
	"database/sql"
	"net/http"
)

// graphNode represents a work item node in the graph response.
type graphNode struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Edges    int    `json:"edges"`
	Activity int    `json:"activity"` // agent_events count for this node
}

// graphEdge represents a directed edge between two nodes.
type graphEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"`
}

// graphData is the full response shape for /api/graph.
type graphData struct {
	Nodes []graphNode `json:"nodes"`
	Edges []graphEdge `json:"edges"`
}

// graphAPIHandler returns a force-directed graph payload for the dashboard.
// By default it filters to nodes that have at least one edge; pass ?all=true
// to include orphan nodes as well.
func graphAPIHandler(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		includeAll := r.URL.Query().Get("all") == "true"

		// Load all nodes with their track_id for implicit edge derivation.
		nodes, trackIDs, err := loadGraphNodes(database)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Collect explicit edges from graph_edges table.
		edges, err := loadGraphEdges(database)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Build known-node set to avoid dangling edge references.
		nodeSet := make(map[string]struct{}, len(nodes))
		for _, n := range nodes {
			nodeSet[n.ID] = struct{}{}
		}

		// Derive implicit part_of edges from track_id column.
		for i, n := range nodes {
			tid := trackIDs[i]
			if tid == "" {
				continue
			}
			if _, ok := nodeSet[tid]; !ok {
				continue // target track not in node set
			}
			edges = append(edges, graphEdge{
				Source: n.ID,
				Target: tid,
				Type:   "part_of",
			})
		}

		// Derive session→feature edges from agent_events.
		edges = append(edges, loadSessionFeatureEdges(database)...)

		// Derive track-to-track edges from shared sessions: if a session
		// worked on features from two different tracks, those tracks are related.
		edges = append(edges, loadTrackCooccurrenceEdges(database)...)

		// Deduplicate edges (explicit DB edges may duplicate implicit ones).
		edges = deduplicateEdges(edges)

		// Build edge-count index.
		edgeCounts := make(map[string]int, len(nodes))
		for _, e := range edges {
			edgeCounts[e.Source]++
			edgeCounts[e.Target]++
		}

		// Annotate nodes with their edge counts.
		for i := range nodes {
			nodes[i].Edges = edgeCounts[nodes[i].ID]
		}

		// Load activity counts per node from agent_events.
		activityCounts := loadActivityCounts(database)
		for i := range nodes {
			nodes[i].Activity = activityCounts[nodes[i].ID]
		}

		// Filter orphans unless ?all=true.
		if !includeAll {
			filtered := make([]graphNode, 0, len(nodes))
			for _, n := range nodes {
				if n.Edges > 0 {
					filtered = append(filtered, n)
				}
			}
			nodes = filtered

			// Rebuild node set after filtering.
			nodeSet = make(map[string]struct{}, len(nodes))
			for _, n := range nodes {
				nodeSet[n.ID] = struct{}{}
			}

			// Drop edges whose endpoints are no longer present.
			filteredEdges := make([]graphEdge, 0, len(edges))
			for _, e := range edges {
				if _, ok := nodeSet[e.Source]; !ok {
					continue
				}
				if _, ok := nodeSet[e.Target]; !ok {
					continue
				}
				filteredEdges = append(filteredEdges, e)
			}
			edges = filteredEdges
		}

		if nodes == nil {
			nodes = []graphNode{}
		}
		if edges == nil {
			edges = []graphEdge{}
		}

		respondJSON(w, graphData{Nodes: nodes, Edges: edges})
	}
}

// loadGraphNodes fetches all work items (features, bugs, spikes from the
// features table) plus tracks from the tracks table. Returns nodes and a
// parallel slice of track IDs for implicit edge derivation.
func loadGraphNodes(database *sql.DB) ([]graphNode, []string, error) {
	var nodes []graphNode
	var trackIDs []string

	// Features, bugs, spikes (all stored in features table).
	rows, err := database.Query(`
		SELECT id, COALESCE(type,'feature'), title, COALESCE(status,'todo'),
		       COALESCE(track_id,'')
		FROM features
		ORDER BY created_at DESC`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var n graphNode
		var tid string
		if err := rows.Scan(&n.ID, &n.Type, &n.Title, &n.Status, &tid); err != nil {
			continue
		}
		nodes = append(nodes, n)
		trackIDs = append(trackIDs, tid)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	// Tracks (separate table).
	trows, err := database.Query(`
		SELECT id, 'track', title, COALESCE(status,'todo')
		FROM tracks
		ORDER BY created_at DESC`)
	if err != nil {
		return nodes, trackIDs, nil // non-fatal, tracks table may not exist
	}
	defer trows.Close()
	for trows.Next() {
		var n graphNode
		if err := trows.Scan(&n.ID, &n.Type, &n.Title, &n.Status); err != nil {
			continue
		}
		nodes = append(nodes, n)
		trackIDs = append(trackIDs, "") // tracks don't have a parent track
	}

	// Sessions that worked on features — only include sessions with
	// meaningful activity (>5 events) to avoid noise.
	// Only include sessions that have actual transcript content —
	// both agent_events (proves attribution) AND messages (proves ingest).
	// Without the messages check, hook-only sessions surface as empty transcripts.
	srows, serr := database.Query(`
		SELECT s.session_id,
		       COALESCE(s.agent_assigned, 'session'),
		       COALESCE(s.status, 'completed')
		FROM sessions s
		WHERE EXISTS (
		    SELECT 1 FROM agent_events e
		    WHERE e.session_id = s.session_id AND e.feature_id != ''
		    GROUP BY e.session_id HAVING COUNT(*) > 5
		)
		AND EXISTS (
		    SELECT 1 FROM messages m WHERE m.session_id = s.session_id
		)
		LIMIT 200`)
	if serr == nil {
		defer srows.Close()
		for srows.Next() {
			var n graphNode
			var agent string
			if err := srows.Scan(&n.ID, &agent, &n.Status); err != nil {
				continue
			}
			n.Type = "session"
			n.Title = agent + " · " + n.ID[:8]
			nodes = append(nodes, n)
			trackIDs = append(trackIDs, "")
		}
	}

	return nodes, trackIDs, nil
}

// loadSessionFeatureEdges derives edges from agent_events — sessions that
// worked on features create a "worked_on" relationship.
func loadSessionFeatureEdges(database *sql.DB) []graphEdge {
	rows, err := database.Query(`
		SELECT DISTINCT session_id, feature_id
		FROM agent_events
		WHERE feature_id != '' AND session_id != ''
		LIMIT 500`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var edges []graphEdge
	for rows.Next() {
		var sid, fid string
		if err := rows.Scan(&sid, &fid); err != nil {
			continue
		}
		edges = append(edges, graphEdge{
			Source: fid,
			Target: sid,
			Type:   "worked_on",
		})
	}
	return edges
}

// loadActivityCounts returns agent_event counts per feature_id.
// Used for node sizing — more activity = bigger node.
func loadActivityCounts(database *sql.DB) map[string]int {
	counts := make(map[string]int)
	rows, err := database.Query(`
		SELECT feature_id, COUNT(*) FROM agent_events
		WHERE feature_id != ''
		GROUP BY feature_id`)
	if err != nil {
		return counts
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var n int
		if err := rows.Scan(&id, &n); err == nil {
			counts[id] = n
		}
	}
	return counts
}

// loadTrackCooccurrenceEdges derives track-to-track relationships from
// shared sessions: if a single session worked on features belonging to
// two different tracks, those tracks are related ("co_session").
func loadTrackCooccurrenceEdges(database *sql.DB) []graphEdge {
	// Find pairs of tracks that share at least one session via agent_events.
	rows, err := database.Query(`
		SELECT DISTINCT t1.track_id, t2.track_id
		FROM agent_events e1
		JOIN features t1 ON t1.id = e1.feature_id
		JOIN agent_events e2 ON e2.session_id = e1.session_id AND e2.feature_id != e1.feature_id
		JOIN features t2 ON t2.id = e2.feature_id
		WHERE t1.track_id != '' AND t2.track_id != ''
		  AND t1.track_id != t2.track_id
		  AND t1.track_id < t2.track_id
		LIMIT 200`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var edges []graphEdge
	for rows.Next() {
		var src, tgt string
		if err := rows.Scan(&src, &tgt); err != nil {
			continue
		}
		edges = append(edges, graphEdge{
			Source: src,
			Target: tgt,
			Type:   "co_session",
		})
	}
	return edges
}

// loadGraphEdges fetches all rows from graph_edges.
func loadGraphEdges(database *sql.DB) ([]graphEdge, error) {
	rows, err := database.Query(`
		SELECT from_node_id, to_node_id, relationship_type
		FROM graph_edges`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []graphEdge
	for rows.Next() {
		var e graphEdge
		if err := rows.Scan(&e.Source, &e.Target, &e.Type); err != nil {
			continue
		}
		edges = append(edges, e)
	}
	return edges, rows.Err()
}

// deduplicateEdges removes duplicate (source, target, type) triples.
func deduplicateEdges(edges []graphEdge) []graphEdge {
	seen := make(map[string]struct{}, len(edges))
	result := make([]graphEdge, 0, len(edges))
	for _, e := range edges {
		key := e.Source + "|" + e.Target + "|" + e.Type
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, e)
	}
	return result
}
