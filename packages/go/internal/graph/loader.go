// Package graph loads and queries HtmlGraph work item files.
package graph

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shakestzd/htmlgraph/internal/htmlparse"
	"github.com/shakestzd/htmlgraph/internal/models"
)

// LoadDir reads all HTML work item files from a directory and returns Nodes.
// Non-HTML files are silently skipped.
func LoadDir(dir string) ([]*models.Node, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", dir, err)
	}

	var nodes []*models.Node
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".html") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		node, err := htmlparse.ParseFile(path)
		if err != nil {
			// Skip unparseable files (matches Python's lenient behaviour).
			continue
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

// LoadAll reads features, bugs, spikes, and tracks from a .htmlgraph root.
func LoadAll(htmlgraphDir string) ([]*models.Node, error) {
	subdirs := []string{"features", "bugs", "spikes", "tracks"}
	var all []*models.Node

	for _, sub := range subdirs {
		dir := filepath.Join(htmlgraphDir, sub)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}
		nodes, err := LoadDir(dir)
		if err != nil {
			return nil, fmt.Errorf("loading %s: %w", sub, err)
		}
		all = append(all, nodes...)
	}
	return all, nil
}
