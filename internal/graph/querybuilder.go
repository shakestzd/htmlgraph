package graph

import (
	"database/sql"
	"fmt"
	"strings"
)

// NodeResult holds a node ID with optional metadata from a query.
type NodeResult struct {
	ID     string
	Type   string
	Title  string
	Status string
}

// queryStep represents a single operation in the traversal pipeline.
type queryStep interface {
	kind() string
}

type fromStep struct{ id string }
type followStep struct{ relType string }
type whereStep struct{ field, value string }
type depthStep struct{ n int }

func (fromStep) kind() string   { return "from" }
func (followStep) kind() string { return "follow" }
func (whereStep) kind() string  { return "where" }
func (depthStep) kind() string  { return "depth" }

// QueryBuilder chains graph traversal operations into a fluent API.
// It reads from the graph_edges table and resolves node metadata from
// features and tracks tables.
type QueryBuilder struct {
	db       *sql.DB
	steps    []queryStep
	maxDepth int
}

// NewQuery creates a new QueryBuilder bound to the given database.
func NewQuery(db *sql.DB) *QueryBuilder {
	return &QueryBuilder{db: db, maxDepth: 10}
}

// From sets the starting node for the traversal.
func (q *QueryBuilder) From(id string) *QueryBuilder {
	q.steps = append(q.steps, fromStep{id: id})
	return q
}

// Follow traverses edges of the given relationship type.
func (q *QueryBuilder) Follow(relType string) *QueryBuilder {
	q.steps = append(q.steps, followStep{relType: relType})
	return q
}

// Where filters the current result set by a node metadata field.
// Supported fields: status, type, priority, track_id.
func (q *QueryBuilder) Where(field, value string) *QueryBuilder {
	q.steps = append(q.steps, whereStep{field: field, value: value})
	return q
}

// Depth sets the maximum traversal depth for follow operations.
func (q *QueryBuilder) Depth(n int) *QueryBuilder {
	q.maxDepth = n
	q.steps = append(q.steps, depthStep{n: n})
	return q
}

// Execute runs the query and returns matching nodes.
func (q *QueryBuilder) Execute() ([]NodeResult, error) {
	if q.db == nil {
		return nil, fmt.Errorf("querybuilder: database is nil")
	}

	// Parse the step pipeline into a starting ID and sequence of operations.
	var startID string
	var ops []queryStep

	for _, s := range q.steps {
		switch v := s.(type) {
		case fromStep:
			startID = v.id
		case depthStep:
			// Already captured in q.maxDepth during build.
		default:
			ops = append(ops, s)
		}
	}

	if startID == "" {
		return nil, fmt.Errorf("querybuilder: From() is required")
	}

	// Start with the seed node.
	currentIDs := []string{startID}

	for _, op := range ops {
		switch v := op.(type) {
		case followStep:
			next, err := q.followEdges(currentIDs, v.relType)
			if err != nil {
				return nil, err
			}
			currentIDs = next
		case whereStep:
			filtered, err := q.filterByField(currentIDs, v.field, v.value)
			if err != nil {
				return nil, err
			}
			currentIDs = filtered
		}
		if len(currentIDs) == 0 {
			return nil, nil
		}
	}

	return q.resolveNodes(currentIDs)
}

// followEdges returns destination node IDs reachable from sourceIDs
// via edges of the given relationship type.
func (q *QueryBuilder) followEdges(sourceIDs []string, relType string) ([]string, error) {
	if len(sourceIDs) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(sourceIDs))
	args := make([]any, len(sourceIDs)+1)
	for i, id := range sourceIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	args[len(sourceIDs)] = relType

	query := fmt.Sprintf(`
		SELECT DISTINCT to_node_id FROM graph_edges
		WHERE from_node_id IN (%s) AND relationship_type = ?`,
		strings.Join(placeholders, ","))

	rows, err := q.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("follow edges: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan edge target: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// tableColumns lists which columns each table exposes for filterByField.
// Only tables that have a given column are included in the UNION arm.
var tableColumns = map[string]map[string]bool{
	"features": {
		"status": true, "type": true, "priority": true, "track_id": true,
	},
	"tracks": {
		"status": true, "type": true, "priority": true,
	},
	"git_commits": {
		"commit_hash": true, "message": true, "session_id": true,
	},
	"feature_files": {
		"file_path": true, "session_id": true,
	},
	"sessions": {
		"session_id": true, "status": true,
	},
}

// tableIDCols maps table name to its primary ID column for filterByField SELECTs.
var tableIDCols = map[string]string{
	"features":      "id",
	"tracks":        "id",
	"git_commits":   "commit_hash",
	"feature_files": "file_path",
	"sessions":      "session_id",
}

// filterByField keeps only IDs whose node metadata matches field=value.
// Searches all node tables that have the requested column.
func (q *QueryBuilder) filterByField(ids []string, field, value string) ([]string, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	// Whitelist fields to prevent SQL injection.
	col, ok := allowedFilterColumns[field]
	if !ok {
		return nil, fmt.Errorf("unsupported filter field %q; allowed: status, type, priority, track_id, commit_hash, message, file_path, session_id", field)
	}

	placeholders := make([]string, len(ids))
	idArgs := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		idArgs[i] = id
	}
	inClause := strings.Join(placeholders, ",")

	// Build UNION arms only for tables that have the requested column.
	tableOrder := []string{"features", "tracks", "git_commits", "feature_files", "sessions"}
	var arms []string
	var fullArgs []any

	for _, table := range tableOrder {
		if !tableColumns[table][col] {
			continue
		}
		idCol := tableIDCols[table]
		distinct := ""
		if table == "feature_files" {
			distinct = "DISTINCT "
		}
		arms = append(arms, fmt.Sprintf(
			`SELECT %s%s AS id FROM %s WHERE %s IN (%s) AND %s = ?`,
			distinct, idCol, table, idCol, inClause, col))
		fullArgs = append(fullArgs, idArgs...)
		fullArgs = append(fullArgs, value)
	}

	if len(arms) == 0 {
		// Column not found in any table; return empty.
		return nil, nil
	}

	query := strings.Join(arms, "\nUNION\n")
	rows, err := q.db.Query(query, fullArgs...)
	if err != nil {
		return nil, fmt.Errorf("filter by %s: %w", field, err)
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan filter result: %w", err)
		}
		result = append(result, id)
	}
	return result, rows.Err()
}

// allowedFilterColumns maps user-facing field names to SQL column names.
var allowedFilterColumns = map[string]string{
	"status":      "status",
	"type":        "type",
	"priority":    "priority",
	"track_id":    "track_id",
	"commit_hash": "commit_hash",
	"message":     "message",
	"file_path":   "file_path",
	"session_id":  "session_id",
}

// resolveNodes looks up metadata for a set of node IDs.
func (q *QueryBuilder) resolveNodes(ids []string) ([]NodeResult, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	inClause := strings.Join(placeholders, ",")

	// Each UNION arm needs its own copy of the id args.
	query := fmt.Sprintf(`
		SELECT id, type, title, status FROM features WHERE id IN (%s)
		UNION ALL
		SELECT id, type, title, status FROM tracks WHERE id IN (%s)
		UNION ALL
		SELECT commit_hash AS id, 'commit' AS type, SUBSTR(COALESCE(message,''),1,80) AS title, 'done' AS status FROM git_commits WHERE commit_hash IN (%s)
		UNION ALL
		SELECT DISTINCT file_path AS id, 'file' AS type, file_path AS title, '' AS status FROM feature_files WHERE file_path IN (%s)
		UNION ALL
		SELECT session_id AS id, 'session' AS type, COALESCE(title,'') AS title, COALESCE(status,'') AS status FROM sessions WHERE session_id IN (%s)`,
		inClause, inClause, inClause, inClause, inClause)

	fullArgs := make([]any, 0, len(args)*5)
	fullArgs = append(fullArgs, args...)
	fullArgs = append(fullArgs, args...)
	fullArgs = append(fullArgs, args...)
	fullArgs = append(fullArgs, args...)
	fullArgs = append(fullArgs, args...)

	rows, err := q.db.Query(query, fullArgs...)
	if err != nil {
		return nil, fmt.Errorf("resolve nodes: %w", err)
	}
	defer rows.Close()

	resolved := make(map[string]NodeResult, len(ids))
	for rows.Next() {
		var r NodeResult
		if err := rows.Scan(&r.ID, &r.Type, &r.Title, &r.Status); err != nil {
			return nil, fmt.Errorf("scan node: %w", err)
		}
		resolved[r.ID] = r
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Return in input order, including unresolved IDs with minimal info.
	results := make([]NodeResult, 0, len(ids))
	for _, id := range ids {
		if r, ok := resolved[id]; ok {
			results = append(results, r)
		} else {
			results = append(results, NodeResult{ID: id})
		}
	}
	return results, nil
}
