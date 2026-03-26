// Package sdk provides the public HtmlGraph SDK for Go consumers.
//
// This is a placeholder that will be fleshed out in Wave 4 (feat-a42e1ef3).
// It mirrors the Python SDK from htmlgraph/__init__.py.
package sdk

import (
	"fmt"
	"path/filepath"

	"github.com/shakestzd/htmlgraph/internal/graph"
	"github.com/shakestzd/htmlgraph/internal/models"
)

// SDK is the main entry point for interacting with an HtmlGraph project.
type SDK struct {
	// ProjectDir is the path to the .htmlgraph/ directory.
	ProjectDir string
	Agent      string
}

// New creates a new SDK instance.
func New(projectDir, agent string) *SDK {
	return &SDK{ProjectDir: projectDir, Agent: agent}
}

// LoadNodes reads all work item nodes from the project.
func (s *SDK) LoadNodes() ([]*models.Node, error) {
	nodes, err := graph.LoadAll(s.ProjectDir)
	if err != nil {
		return nil, fmt.Errorf("sdk load nodes: %w", err)
	}
	return nodes, nil
}

// FeaturesDir returns the path to the features subdirectory.
func (s *SDK) FeaturesDir() string {
	return filepath.Join(s.ProjectDir, "features")
}

// BugsDir returns the path to the bugs subdirectory.
func (s *SDK) BugsDir() string {
	return filepath.Join(s.ProjectDir, "bugs")
}

// SpikesDir returns the path to the spikes subdirectory.
func (s *SDK) SpikesDir() string {
	return filepath.Join(s.ProjectDir, "spikes")
}

// TracksDir returns the path to the tracks subdirectory.
func (s *SDK) TracksDir() string {
	return filepath.Join(s.ProjectDir, "tracks")
}
