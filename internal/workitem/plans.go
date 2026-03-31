package workitem

import (
	"fmt"
	"time"

	dbpkg "github.com/shakestzd/htmlgraph/internal/db"
	"github.com/shakestzd/htmlgraph/internal/models"
)

// PlanOption configures a new plan during creation.
type PlanOption func(*planConfig)

type planConfig struct {
	priority string
	status   string
	trackID  string
	steps    []string
	content  string
}

// PlanWithPriority sets the plan's priority.
func PlanWithPriority(p string) PlanOption {
	return func(c *planConfig) { c.priority = p }
}

// PlanWithTrack links the plan to a track.
func PlanWithTrack(trackID string) PlanOption {
	return func(c *planConfig) { c.trackID = trackID }
}

// PlanWithSteps adds implementation steps.
func PlanWithSteps(steps ...string) PlanOption {
	return func(c *planConfig) { c.steps = steps }
}

// PlanWithContent sets the description body.
func PlanWithContent(content string) PlanOption {
	return func(c *planConfig) { c.content = content }
}

// PlanCollection provides CRUD operations for plans.
type PlanCollection struct {
	*Collection
}

// NewPlanCollection creates a PlanCollection bound to the given Base.
func NewPlanCollection(base *Base) *PlanCollection {
	return &PlanCollection{Collection: newCollection(base, "plans", "plan")}
}

// Create builds a new plan node, writes HTML, and optionally inserts into SQLite.
func (pc *PlanCollection) Create(title string, opts ...PlanOption) (*models.Node, error) {
	if title == "" {
		return nil, fmt.Errorf("plan title must not be empty")
	}

	cfg := &planConfig{priority: "medium", status: "todo"}
	for _, opt := range opts {
		opt(cfg)
	}

	now := time.Now().UTC()
	id := generateID("plan", title)

	var steps []models.Step
	for i, desc := range cfg.steps {
		steps = append(steps, models.Step{
			StepID:      fmt.Sprintf("step-%s-%d", id, i),
			Description: desc,
		})
	}

	node := &models.Node{
		ID:            id,
		Title:         title,
		Type:          "plan",
		Status:        models.NodeStatus(cfg.status),
		Priority:      models.Priority(cfg.priority),
		CreatedAt:     now,
		UpdatedAt:     now,
		AgentAssigned: pc.base.Agent,
		TrackID:       cfg.trackID,
		Steps:         steps,
		Content:       cfg.content,
	}

	if _, err := pc.writeNode(node); err != nil {
		return nil, fmt.Errorf("create plan: %w", err)
	}

	if pc.base.DB != nil {
		dbFeat := &dbpkg.Feature{
			ID:         id,
			Type:       "plan",
			Title:      title,
			Status:     cfg.status,
			Priority:   cfg.priority,
			AssignedTo: pc.base.Agent,
			TrackID:    cfg.trackID,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		_ = dbpkg.InsertFeature(pc.base.DB, dbFeat)
	}

	return node, nil
}
