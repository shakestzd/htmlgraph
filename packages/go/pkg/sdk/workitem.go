package sdk

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/shakestzd/htmlgraph/internal/htmlparse"
	"github.com/shakestzd/htmlgraph/internal/models"
)

// WorkItem is a type-agnostic representation of an active work item.
// Returned by GetActiveWorkItem to avoid callers needing to know which
// collection the item belongs to.
type WorkItem struct {
	ID     string
	Type   string // "feature", "bug", or "spike"
	Title  string
	Status string
}

// GetActiveWorkItem scans features, bugs, and spikes for the first
// work item with status "in-progress" and returns it.
// Returns (nil, nil) if no active work item is found.
func (s *SDK) GetActiveWorkItem() (*WorkItem, error) {
	dirs := []struct {
		path     string
		nodeType string
	}{
		{s.FeaturesDir(), "feature"},
		{s.BugsDir(), "bug"},
		{s.SpikesDir(), "spike"},
	}

	for _, d := range dirs {
		item, err := findActiveInDir(d.path, d.nodeType)
		if err != nil {
			return nil, err
		}
		if item != nil {
			return item, nil
		}
	}
	return nil, nil
}

// findActiveInDir scans a directory for the first in-progress node
// of the given type.
func findActiveInDir(dir, nodeType string) (*WorkItem, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".html") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		node, err := htmlparse.ParseFile(path)
		if err != nil {
			continue // skip unparseable files
		}
		if node.Status != models.StatusInProgress || node.Type != nodeType {
			continue
		}
		return &WorkItem{
			ID:     node.ID,
			Type:   node.Type,
			Title:  node.Title,
			Status: string(node.Status),
		}, nil
	}
	return nil, nil
}
