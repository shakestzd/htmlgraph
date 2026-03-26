package sdk

import (
	"fmt"
	"sort"

	"github.com/shakestzd/htmlgraph/internal/models"
)

// SessionCollection provides read operations for sessions.
// Sessions are primarily created by hooks, not by the SDK directly.
type SessionCollection struct {
	*Collection
}

// NewSessionCollection creates a SessionCollection bound to the SDK.
func NewSessionCollection(s *SDK) *SessionCollection {
	return &SessionCollection{Collection: newCollection(s, "sessions", "session")}
}

// GetLatest returns the N most recent sessions.
func (sc *SessionCollection) GetLatest(limit int) ([]*models.Node, error) {
	if limit <= 0 {
		limit = 1
	}

	nodes, err := sc.List()
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].CreatedAt.After(nodes[j].CreatedAt)
	})

	if len(nodes) > limit {
		nodes = nodes[:limit]
	}
	return nodes, nil
}
