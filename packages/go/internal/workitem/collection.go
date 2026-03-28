package workitem

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shakestzd/htmlgraph/internal/graph"
	"github.com/shakestzd/htmlgraph/internal/htmlparse"
	"github.com/shakestzd/htmlgraph/internal/models"

	dbpkg "github.com/shakestzd/htmlgraph/internal/db"
)

// FilterFunc is a predicate applied to nodes during queries.
type FilterFunc func(*models.Node) bool

// FilterOption configures listing/query behaviour.
type FilterOption func(*filterConfig)

type filterConfig struct {
	status   string
	priority string
	trackID  string
	agent    string
}

// WithStatus filters by node status.
func WithStatus(s string) FilterOption {
	return func(c *filterConfig) { c.status = s }
}

// WithPriority filters by node priority.
func WithPriority(p string) FilterOption {
	return func(c *filterConfig) { c.priority = p }
}

// WithTrackID filters by track ID.
func WithTrackID(id string) FilterOption {
	return func(c *filterConfig) { c.trackID = id }
}

// WithAgent filters by agent assignment.
func WithAgent(a string) FilterOption {
	return func(c *filterConfig) { c.agent = a }
}

// Collection is a generic, type-aware collection of work item nodes.
// It manages a single subdirectory of .htmlgraph/ (features, bugs, spikes,
// tracks, or sessions) and provides CRUD, filtering, and lifecycle methods.
type Collection struct {
	base           *Base
	collectionName string // e.g. "features"
	nodeType       string // e.g. "feature"
}

func newCollection(base *Base, name, nodeType string) *Collection {
	return &Collection{base: base, collectionName: name, nodeType: nodeType}
}

// Dir returns the absolute path to this collection's directory.
func (c *Collection) Dir() string {
	return filepath.Join(c.base.ProjectDir, c.collectionName)
}

// Get retrieves a single node by ID from the HTML file on disk.
func (c *Collection) Get(id string) (*models.Node, error) {
	path := filepath.Join(c.Dir(), id+".html")
	node, err := htmlparse.ParseFile(path)
	if err != nil {
		return nil, fmt.Errorf("get %s/%s: %w", c.collectionName, id, err)
	}
	return node, nil
}

// List returns all nodes in this collection, optionally filtered.
func (c *Collection) List(opts ...FilterOption) ([]*models.Node, error) {
	cfg := &filterConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	nodes, err := graph.LoadDir(c.Dir())
	if err != nil {
		// Directory might not exist yet — return empty list.
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("list %s: %w", c.collectionName, err)
	}

	var filtered []*models.Node
	for _, n := range nodes {
		if n.Type != c.nodeType {
			continue
		}
		if cfg.status != "" && string(n.Status) != cfg.status {
			continue
		}
		if cfg.priority != "" && string(n.Priority) != cfg.priority {
			continue
		}
		if cfg.trackID != "" && n.TrackID != cfg.trackID {
			continue
		}
		if cfg.agent != "" && n.AgentAssigned != cfg.agent {
			continue
		}
		filtered = append(filtered, n)
	}
	return filtered, nil
}

// Filter returns nodes matching a custom predicate.
func (c *Collection) Filter(fn FilterFunc) ([]*models.Node, error) {
	nodes, err := graph.LoadDir(c.Dir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("filter %s: %w", c.collectionName, err)
	}

	var out []*models.Node
	for _, n := range nodes {
		if n.Type != c.nodeType {
			continue
		}
		if fn(n) {
			out = append(out, n)
		}
	}
	return out, nil
}

// Delete removes a node's HTML file from disk.
func (c *Collection) Delete(id string) error {
	path := filepath.Join(c.Dir(), id+".html")
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("delete %s/%s: %w", c.collectionName, id, err)
	}
	return nil
}

// writeNode writes a node to disk and returns its path.
func (c *Collection) writeNode(node *models.Node) (string, error) {
	return WriteNodeHTML(c.Dir(), node)
}

// Start marks a node as in-progress and dual-writes status to SQLite.
func (c *Collection) Start(id string) (*models.Node, error) {
	node, err := c.Get(id)
	if err != nil {
		return nil, err
	}
	node.Status = models.StatusInProgress
	node.AgentAssigned = c.base.Agent
	node.UpdatedAt = time.Now().UTC()
	if _, err := c.writeNode(node); err != nil {
		return nil, err
	}
	if c.base.DB != nil {
		_ = dbpkg.UpdateFeatureStatus(c.base.DB, id, "in-progress")
	}
	return node, nil
}

// Complete marks a node as done and auto-completes all steps.
func (c *Collection) Complete(id string) (*models.Node, error) {
	node, err := c.Get(id)
	if err != nil {
		return nil, err
	}
	for i := range node.Steps {
		if !node.Steps[i].Completed {
			node.Steps[i].Completed = true
			node.Steps[i].Agent = c.base.Agent
			node.Steps[i].Timestamp = time.Now().UTC()
		}
	}
	node.Status = models.StatusDone
	node.UpdatedAt = time.Now().UTC()
	if _, err := c.writeNode(node); err != nil {
		return nil, err
	}
	if c.base.DB != nil {
		_ = dbpkg.UpdateFeatureStatus(c.base.DB, id, "done")
	}
	return node, nil
}

// --- Edge operations ---------------------------------------------------------

// AddEdge reads a node, appends an edge, and writes it back to disk.
// It also dual-writes to graph_edges in SQLite when a DB connection is available.
// HTML is canonical; SQLite errors are non-fatal.
func (c *Collection) AddEdge(id string, e models.Edge) (*models.Node, error) {
	node, err := c.Get(id)
	if err != nil {
		return nil, fmt.Errorf("add edge %s: %w", id, err)
	}
	node.AddEdge(e)
	if _, err := c.writeNode(node); err != nil {
		return nil, fmt.Errorf("add edge %s: %w", id, err)
	}

	// Dual-write to SQLite read index.
	if c.base.DB != nil {
		edgeID := fmt.Sprintf("%s-%s-%s", id, string(e.Relationship), e.TargetID)
		_ = dbpkg.InsertEdge(
			c.base.DB,
			edgeID, id, c.nodeType,
			e.TargetID, inferNodeType(e.TargetID),
			string(e.Relationship),
			e.Properties,
		)
	}

	return node, nil
}

// RemoveEdge reads a node, removes the matching edge, and writes it back.
// Returns the updated node and whether an edge was actually removed.
// It also removes the corresponding row from graph_edges in SQLite.
func (c *Collection) RemoveEdge(id, targetID string, relType models.RelationshipType) (*models.Node, bool, error) {
	node, err := c.Get(id)
	if err != nil {
		return nil, false, fmt.Errorf("remove edge %s: %w", id, err)
	}
	removed := node.RemoveEdge(targetID, relType)
	if !removed {
		return node, false, nil
	}
	if _, err := c.writeNode(node); err != nil {
		return nil, false, fmt.Errorf("remove edge %s: %w", id, err)
	}

	// Dual-write: remove from SQLite read index.
	if c.base.DB != nil {
		_ = dbpkg.DeleteEdge(c.base.DB, id, targetID, string(relType))
	}

	return node, true, nil
}

// inferNodeType derives the node type string from an ID prefix.
// feat-* → "feature", bug-* → "bug", spk-* → "spike",
// trk-* → "track", plan-* → "plan", spec-* → "spec".
// Falls back to "unknown" for unrecognised prefixes.
func inferNodeType(id string) string {
	switch {
	case strings.HasPrefix(id, "feat-"):
		return "feature"
	case strings.HasPrefix(id, "bug-"):
		return "bug"
	case strings.HasPrefix(id, "spk-"):
		return "spike"
	case strings.HasPrefix(id, "trk-"):
		return "track"
	case strings.HasPrefix(id, "plan-"):
		return "plan"
	case strings.HasPrefix(id, "spec-"):
		return "spec"
	default:
		return "unknown"
	}
}

// --- Claim / release operations ----------------------------------------------

// Claim marks a work item as claimed by the current agent.
// It sets AgentAssigned, ClaimedAt, and ClaimedBySession.
func (c *Collection) Claim(id, sessionID string) error {
	node, err := c.Get(id)
	if err != nil {
		return fmt.Errorf("claim %s/%s: %w", c.collectionName, id, err)
	}

	now := time.Now().UTC()
	node.AgentAssigned = c.base.Agent
	node.ClaimedAt = fmtTime(now)
	node.ClaimedBySession = sessionID
	node.UpdatedAt = now

	if _, err := c.writeNode(node); err != nil {
		return fmt.Errorf("claim %s/%s: %w", c.collectionName, id, err)
	}
	return nil
}

// Release clears the claim on a work item, removing agent assignment
// and claim metadata.
func (c *Collection) Release(id string) error {
	node, err := c.Get(id)
	if err != nil {
		return fmt.Errorf("release %s/%s: %w", c.collectionName, id, err)
	}

	node.AgentAssigned = ""
	node.ClaimedAt = ""
	node.ClaimedBySession = ""
	node.UpdatedAt = time.Now().UTC()

	if _, err := c.writeNode(node); err != nil {
		return fmt.Errorf("release %s/%s: %w", c.collectionName, id, err)
	}
	return nil
}

// AtomicClaim claims a work item only if it is not already claimed
// by another agent. Returns an error if already claimed.
func (c *Collection) AtomicClaim(id, sessionID string) error {
	node, err := c.Get(id)
	if err != nil {
		return fmt.Errorf("atomic claim %s/%s: %w", c.collectionName, id, err)
	}

	if node.ClaimedBySession != "" && node.ClaimedBySession != sessionID {
		return fmt.Errorf(
			"atomic claim %s/%s: already claimed by session %s",
			c.collectionName, id, node.ClaimedBySession,
		)
	}
	if node.AgentAssigned != "" && node.AgentAssigned != c.base.Agent {
		return fmt.Errorf(
			"atomic claim %s/%s: already claimed by agent %s",
			c.collectionName, id, node.AgentAssigned,
		)
	}

	now := time.Now().UTC()
	node.AgentAssigned = c.base.Agent
	node.ClaimedAt = fmtTime(now)
	node.ClaimedBySession = sessionID
	node.UpdatedAt = now

	if _, err := c.writeNode(node); err != nil {
		return fmt.Errorf("atomic claim %s/%s: %w", c.collectionName, id, err)
	}
	return nil
}

// Unclaim removes the claim metadata without changing the node's status.
// Unlike Release, Unclaim only clears ClaimedAt and ClaimedBySession
// but preserves AgentAssigned.
func (c *Collection) Unclaim(id string) error {
	node, err := c.Get(id)
	if err != nil {
		return fmt.Errorf("unclaim %s/%s: %w", c.collectionName, id, err)
	}

	node.ClaimedAt = ""
	node.ClaimedBySession = ""
	node.UpdatedAt = time.Now().UTC()

	if _, err := c.writeNode(node); err != nil {
		return fmt.Errorf("unclaim %s/%s: %w", c.collectionName, id, err)
	}
	return nil
}
