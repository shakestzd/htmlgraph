package db

import (
	"database/sql"
	"fmt"
	"strings"
)

// SemanticEntry holds the searchable content for a single feature in the FTS5 index.
type SemanticEntry struct {
	FeatureID      string
	Title          string
	Description    string
	Content        string
	Tags           string // space-separated tags/keywords
	TrackTitle     string
	RelatedContext string // titles of linked features via graph_edges
}

// SemanticResult is a ranked search hit from the semantic index.
type SemanticResult struct {
	FeatureID string  `json:"feature_id"`
	Title     string  `json:"title"`
	Type      string  `json:"type"`
	Status    string  `json:"status"`
	Priority  string  `json:"priority"`
	TrackID   string  `json:"track_id"`
	Rank      float64 `json:"rank"`
	Snippet   string  `json:"snippet"`
}

// CreateSemanticIndex creates the FTS5 virtual table for semantic search.
// Porter stemming enables "cache" to match "caching", "cached", etc.
// Column weights are applied at query time via bm25().
func CreateSemanticIndex(db *sql.DB) error {
	_, err := db.Exec(`CREATE VIRTUAL TABLE IF NOT EXISTS semantic_index USING fts5(
		feature_id UNINDEXED,
		title,
		description,
		content,
		tags,
		track_title,
		related_context,
		tokenize='porter unicode61'
	)`)
	if err != nil {
		return fmt.Errorf("create semantic_index: %w", err)
	}
	return nil
}

// UpsertSemanticEntry inserts or replaces a feature's searchable content.
// FTS5 tables don't support ON CONFLICT, so we delete-then-insert.
func UpsertSemanticEntry(db *sql.DB, e *SemanticEntry) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	tx.Exec(`DELETE FROM semantic_index WHERE feature_id = ?`, e.FeatureID)

	_, err = tx.Exec(`INSERT INTO semantic_index
		(feature_id, title, description, content, tags, track_title, related_context)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		e.FeatureID, e.Title, e.Description, e.Content,
		e.Tags, e.TrackTitle, e.RelatedContext,
	)
	if err != nil {
		return fmt.Errorf("insert semantic entry %s: %w", e.FeatureID, err)
	}

	return tx.Commit()
}

// DeleteSemanticEntry removes a feature from the semantic index.
func DeleteSemanticEntry(db *sql.DB, featureID string) error {
	_, err := db.Exec(`DELETE FROM semantic_index WHERE feature_id = ?`, featureID)
	return err
}

// SemanticSearch performs a BM25-ranked full-text search across all indexed features.
// Column weights (bm25 args): title=10, description=5, content=2, tags=8, track_title=3, related_context=4.
// The query is sanitized to prevent FTS5 syntax errors from user input.
func SemanticSearch(db *sql.DB, query string, limit int) ([]SemanticResult, error) {
	if limit <= 0 {
		limit = 20
	}

	ftsQuery := sanitizeFTSQuery(query)
	if ftsQuery == "" {
		return nil, nil
	}

	rows, err := db.Query(`
		SELECT
			si.feature_id,
			si.title,
			COALESCE(f.type, t.type, ''),
			COALESCE(f.status, t.status, ''),
			COALESCE(f.priority, t.priority, ''),
			COALESCE(f.track_id, ''),
			bm25(semantic_index, 0.0, 10.0, 5.0, 2.0, 8.0, 3.0, 4.0) AS rank,
			snippet(semantic_index, 2, '<b>', '</b>', '...', 32) AS snippet
		FROM semantic_index si
		LEFT JOIN features f ON f.id = si.feature_id
		LEFT JOIN tracks t ON t.id = si.feature_id
		WHERE semantic_index MATCH ?
		ORDER BY rank
		LIMIT ?`, ftsQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("semantic search: %w", err)
	}
	defer rows.Close()

	var results []SemanticResult
	for rows.Next() {
		var r SemanticResult
		if err := rows.Scan(&r.FeatureID, &r.Title, &r.Type, &r.Status,
			&r.Priority, &r.TrackID, &r.Rank, &r.Snippet); err != nil {
			continue
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// SemanticRelated finds features semantically similar to a given feature.
// It extracts the feature's indexed content and uses it as a search query,
// excluding the feature itself from results.
func SemanticRelated(db *sql.DB, featureID string, limit int) ([]SemanticResult, error) {
	if limit <= 0 {
		limit = 10
	}

	// Extract the feature's title and tags to use as the similarity query.
	var title, tags string
	err := db.QueryRow(`SELECT title, tags FROM semantic_index WHERE feature_id = ?`,
		featureID).Scan(&title, &tags)
	if err != nil {
		return nil, fmt.Errorf("feature %s not in semantic index: %w", featureID, err)
	}

	// Combine title + tags as the similarity query.
	combined := title
	if tags != "" {
		combined += " " + tags
	}

	ftsQuery := sanitizeFTSQuery(combined)
	if ftsQuery == "" {
		return nil, nil
	}

	rows, err := db.Query(`
		SELECT
			si.feature_id,
			si.title,
			COALESCE(f.type, t.type, ''),
			COALESCE(f.status, t.status, ''),
			COALESCE(f.priority, t.priority, ''),
			COALESCE(f.track_id, ''),
			bm25(semantic_index, 0.0, 10.0, 5.0, 2.0, 8.0, 3.0, 4.0) AS rank,
			snippet(semantic_index, 2, '<b>', '</b>', '...', 32) AS snippet
		FROM semantic_index si
		LEFT JOIN features f ON f.id = si.feature_id
		LEFT JOIN tracks t ON t.id = si.feature_id
		WHERE semantic_index MATCH ?
		  AND si.feature_id != ?
		ORDER BY rank
		LIMIT ?`, ftsQuery, featureID, limit)
	if err != nil {
		return nil, fmt.Errorf("semantic related: %w", err)
	}
	defer rows.Close()

	var results []SemanticResult
	for rows.Next() {
		var r SemanticResult
		if err := rows.Scan(&r.FeatureID, &r.Title, &r.Type, &r.Status,
			&r.Priority, &r.TrackID, &r.Rank, &r.Snippet); err != nil {
			continue
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// RebuildSemanticIndex drops and recreates the FTS5 table, then repopulates
// it from the features table enriched with graph_edges and tracks.
func RebuildSemanticIndex(db *sql.DB) (int, error) {
	db.Exec(`DROP TABLE IF EXISTS semantic_index`)

	if err := CreateSemanticIndex(db); err != nil {
		return 0, err
	}

	// Load all features with their descriptions.
	rows, err := db.Query(`
		SELECT f.id, f.title, COALESCE(f.description, ''),
		       COALESCE(f.tags, ''), COALESCE(f.track_id, ''),
		       COALESCE(t.title, '') AS track_title
		FROM features f
		LEFT JOIN tracks t ON t.id = f.track_id`)
	if err != nil {
		return 0, fmt.Errorf("load features for semantic index: %w", err)
	}
	defer rows.Close()

	type featureRow struct {
		id          string
		title       string
		description string
		tags        string
		trackID     string
		trackTitle  string
	}

	var features []featureRow
	for rows.Next() {
		var fr featureRow
		if err := rows.Scan(&fr.id, &fr.title, &fr.description,
			&fr.tags, &fr.trackID, &fr.trackTitle); err != nil {
			continue
		}
		features = append(features, fr)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	// Build related context from graph_edges: for each feature,
	// collect titles of features it's connected to.
	relatedCtx := buildRelatedContext(db)

	count := 0
	for _, fr := range features {
		tags := normalizeJSONTags(fr.tags)
		entry := &SemanticEntry{
			FeatureID:      fr.id,
			Title:          fr.title,
			Description:    fr.description,
			Content:        fr.description, // description is already the truncated content
			Tags:           tags,
			TrackTitle:     fr.trackTitle,
			RelatedContext: relatedCtx[fr.id],
		}
		if err := UpsertSemanticEntry(db, entry); err != nil {
			continue
		}
		count++
	}

	// Also index tracks (stored in separate table).
	trackCount, err := indexTracks(db, relatedCtx)
	if err == nil {
		count += trackCount
	}

	return count, nil
}

// indexTracks adds tracks from the tracks table into the semantic index.
func indexTracks(db *sql.DB, relatedCtx map[string]string) (int, error) {
	rows, err := db.Query(`
		SELECT id, title, COALESCE(description, ''), COALESCE(metadata, '')
		FROM tracks`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, title, description, metadata string
		if err := rows.Scan(&id, &title, &description, &metadata); err != nil {
			continue
		}
		// For tracks, include titles of child features as related context.
		childCtx := relatedCtx[id]
		entry := &SemanticEntry{
			FeatureID:      id,
			Title:          title,
			Description:    description,
			Content:        description,
			Tags:           normalizeJSONTags(metadata),
			TrackTitle:     "", // tracks don't have a parent track
			RelatedContext: childCtx,
		}
		if err := UpsertSemanticEntry(db, entry); err != nil {
			continue
		}
		count++
	}
	return count, rows.Err()
}

// buildRelatedContext collects titles of features linked via graph_edges
// for each feature, building a map of featureID -> space-separated related titles.
func buildRelatedContext(db *sql.DB) map[string]string {
	ctx := make(map[string]string)

	rows, err := db.Query(`
		SELECT ge.from_node_id,
		       GROUP_CONCAT(COALESCE(f.title, t.title, ge.to_node_id), ' | ')
		FROM graph_edges ge
		LEFT JOIN features f ON f.id = ge.to_node_id
		LEFT JOIN tracks t ON t.id = ge.to_node_id
		GROUP BY ge.from_node_id`)
	if err != nil {
		return ctx
	}
	defer rows.Close()

	for rows.Next() {
		var fromID, titles string
		if rows.Scan(&fromID, &titles) == nil {
			ctx[fromID] = titles
		}
	}

	// Also add reverse direction (to_node_id -> from_node titles).
	rows2, err := db.Query(`
		SELECT ge.to_node_id,
		       GROUP_CONCAT(COALESCE(f.title, t.title, ge.from_node_id), ' | ')
		FROM graph_edges ge
		LEFT JOIN features f ON f.id = ge.from_node_id
		LEFT JOIN tracks t ON t.id = ge.from_node_id
		GROUP BY ge.to_node_id`)
	if err != nil {
		return ctx
	}
	defer rows2.Close()

	for rows2.Next() {
		var toID, titles string
		if rows2.Scan(&toID, &titles) == nil {
			if existing, ok := ctx[toID]; ok {
				ctx[toID] = existing + " | " + titles
			} else {
				ctx[toID] = titles
			}
		}
	}

	return ctx
}

// normalizeJSONTags extracts tag strings from a JSON array like ["tag1","tag2"]
// and returns them space-separated for FTS5 indexing.
func normalizeJSONTags(jsonTags string) string {
	jsonTags = strings.TrimSpace(jsonTags)
	if jsonTags == "" || jsonTags == "null" || jsonTags == "[]" {
		return ""
	}

	// Simple extraction: strip brackets and quotes.
	jsonTags = strings.Trim(jsonTags, "[]")
	var tags []string
	for _, part := range strings.Split(jsonTags, ",") {
		tag := strings.Trim(strings.TrimSpace(part), `"'`)
		if tag != "" {
			tags = append(tags, tag)
		}
	}
	return strings.Join(tags, " ")
}

// sanitizeFTSQuery converts user input into a safe FTS5 query.
// It splits on whitespace and joins with implicit AND,
// stripping FTS5 operators that could cause syntax errors.
func sanitizeFTSQuery(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}

	// FTS5 special characters that need to be removed from user input.
	replacer := strings.NewReplacer(
		"(", " ",
		")", " ",
		"*", " ",
		"\"", " ",
		":", " ",
		"^", " ",
		"{", " ",
		"}", " ",
	)
	cleaned := replacer.Replace(input)

	// Split into words, filter out FTS5 operators.
	ftsOps := map[string]bool{
		"AND": true, "OR": true, "NOT": true, "NEAR": true,
	}

	var terms []string
	for _, word := range strings.Fields(cleaned) {
		word = strings.TrimSpace(word)
		if word == "" || ftsOps[strings.ToUpper(word)] {
			continue
		}
		// Use prefix matching for better recall: each term matches as a prefix.
		terms = append(terms, word+"*")
	}

	if len(terms) == 0 {
		return ""
	}

	return strings.Join(terms, " ")
}
