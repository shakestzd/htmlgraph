package graph

import (
	"github.com/shakestzd/htmlgraph/internal/models"
)

// ByStatus filters nodes by status.
func ByStatus(nodes []*models.Node, status models.NodeStatus) []*models.Node {
	var out []*models.Node
	for _, n := range nodes {
		if n.Status == status {
			out = append(out, n)
		}
	}
	return out
}

// ByType filters nodes by type string (e.g. "feature", "spike", "bug").
func ByType(nodes []*models.Node, nodeType string) []*models.Node {
	var out []*models.Node
	for _, n := range nodes {
		if n.Type == nodeType {
			out = append(out, n)
		}
	}
	return out
}

// ByTrack filters nodes belonging to a specific track.
func ByTrack(nodes []*models.Node, trackID string) []*models.Node {
	var out []*models.Node
	for _, n := range nodes {
		if n.TrackID == trackID {
			out = append(out, n)
		}
	}
	return out
}

// FindByID returns the first node with the given ID, or nil.
func FindByID(nodes []*models.Node, id string) *models.Node {
	for _, n := range nodes {
		if n.ID == id {
			return n
		}
	}
	return nil
}
