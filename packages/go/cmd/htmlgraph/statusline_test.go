package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	dbpkg "github.com/shakestzd/htmlgraph/packages/go/internal/db"
)

func TestStatuslineCmd(t *testing.T) {
	// Create temporary directory for test database
	tmpDir := t.TempDir()
	htmlgraphDir := filepath.Join(tmpDir, ".htmlgraph")
	if err := os.MkdirAll(htmlgraphDir, 0o755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	// Set up project directory
	oldCwd, _ := os.Getwd()
	defer os.Chdir(oldCwd)
	os.Chdir(tmpDir)

	// Create and populate test database
	dbPath := filepath.Join(htmlgraphDir, "htmlgraph.db")
	db, err := dbpkg.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Insert test data
	tests := []struct {
		name        string
		data        []testWorkItem
		expectID    string
		expectTitle string
	}{
		{
			name: "no_active_items",
			data: []testWorkItem{
				{ID: "feat-123", Type: "feature", Status: "todo", Title: "Test Feature"},
			},
			expectID:    "",
			expectTitle: "",
		},
		{
			name: "single_active_feature",
			data: []testWorkItem{
				{ID: "feat-456", Type: "feature", Status: "in-progress", Title: "Active Feature"},
			},
			expectID:    "feat-456",
			expectTitle: "Active Feature",
		},
		{
			name: "single_active_bug",
			data: []testWorkItem{
				{ID: "bug-789", Type: "bug", Status: "in-progress", Title: "Critical Bug"},
			},
			expectID:    "bug-789",
			expectTitle: "Critical Bug",
		},
		{
			name: "bug_prioritized_over_feature",
			data: []testWorkItem{
				{ID: "feat-111", Type: "feature", Status: "in-progress", Title: "Feature"},
				{ID: "bug-222", Type: "bug", Status: "in-progress", Title: "Bug Fix"},
			},
			expectID:    "bug-222",
			expectTitle: "Bug Fix",
		},
		{
			name: "truncates_long_title",
			data: []testWorkItem{
				{ID: "feat-333", Type: "feature", Status: "in-progress", Title: "This is a very long feature title that should be truncated"},
			},
			expectID:    "feat-333",
			expectTitle: "This is a very long feat…",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up previous test data
			db.Exec("DELETE FROM features")

			// Insert test data
			for _, item := range tt.data {
				_, err := db.Exec(`
					INSERT INTO features (id, type, title, status)
					VALUES (?, ?, ?, ?)
				`, item.ID, item.Type, item.Title, item.Status)
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
				}
			}

			// Query for active item
			var workItemID, title string
			err := db.QueryRow(`
				SELECT id, title
				FROM features
				WHERE status = 'in-progress'
				ORDER BY CASE type WHEN 'bug' THEN 0 WHEN 'feature' THEN 1 ELSE 2 END
				LIMIT 1
			`).Scan(&workItemID, &title)

			if tt.expectID == "" {
				if err != sql.ErrNoRows {
					t.Errorf("expected no rows, got %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("query failed: %v", err)
			}

			if workItemID != tt.expectID {
				t.Errorf("expected ID %q, got %q", tt.expectID, workItemID)
			}

			truncatedTitle := truncate(title, 25)
			if truncatedTitle != tt.expectTitle {
				t.Errorf("expected truncated title %q, got %q", tt.expectTitle, truncatedTitle)
			}
		})
	}
}

type testWorkItem struct {
	ID     string
	Type   string
	Status string
	Title  string
}
