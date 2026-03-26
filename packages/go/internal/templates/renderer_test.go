package templates_test

import (
	"strings"
	"testing"
	"time"

	"github.com/shakestzd/htmlgraph/internal/htmlparse"
	"github.com/shakestzd/htmlgraph/internal/models"
	"github.com/shakestzd/htmlgraph/internal/templates"
)

func mustParseTime(s string) time.Time {
	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02T15:04:05.999999",
		"2006-01-02T15:04:05",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t
		}
	}
	panic("cannot parse time: " + s)
}

// --- Feature tests ---

func TestRenderFeature_Basic(t *testing.T) {
	node := &models.Node{
		ID:            "feat-abc123",
		Title:         "Add user authentication",
		Type:          "feature",
		Status:        models.StatusInProgress,
		Priority:      models.PriorityHigh,
		CreatedAt:     mustParseTime("2026-03-26T09:47:14.165536"),
		UpdatedAt:     mustParseTime("2026-03-26T10:12:06.794715"),
		AgentAssigned: "claude-code",
		TrackID:       "trk-696ae199",
	}

	html, err := templates.RenderFeature(node)
	if err != nil {
		t.Fatalf("RenderFeature: %v", err)
	}

	assertContains(t, html, `<!DOCTYPE html>`)
	assertContains(t, html, `<meta name="htmlgraph-version" content="1.0">`)
	assertContains(t, html, `<title>Add user authentication</title>`)
	assertContains(t, html, `id="feat-abc123"`)
	assertContains(t, html, `data-type="feature"`)
	assertContains(t, html, `data-status="in-progress"`)
	assertContains(t, html, `data-priority="high"`)
	assertContains(t, html, `data-agent-assigned="claude-code"`)
	assertContains(t, html, `data-track-id="trk-696ae199"`)
	assertContains(t, html, `<h1>Add user authentication</h1>`)
}

func TestRenderFeature_WithSteps(t *testing.T) {
	node := &models.Node{
		ID:        "feat-steps01",
		Title:     "Feature with steps",
		Type:      "feature",
		Status:    models.StatusTodo,
		Priority:  models.PriorityMedium,
		CreatedAt: mustParseTime("2026-03-26T09:00:00"),
		UpdatedAt: mustParseTime("2026-03-26T09:00:00"),
		Steps: []models.Step{
			{StepID: "step-0", Description: "First step", Completed: true},
			{StepID: "step-1", Description: "Second step", Completed: false, Agent: "haiku"},
			{StepID: "step-2", Description: "Third step", Completed: false, DependsOn: []string{"step-1"}},
		},
	}

	html, err := templates.RenderFeature(node)
	if err != nil {
		t.Fatalf("RenderFeature: %v", err)
	}

	assertContains(t, html, `data-steps`)
	assertContains(t, html, `data-completed="true"`)
	assertContains(t, html, `data-step-id="step-0"`)
	assertContains(t, html, `data-step-id="step-1"`)
	assertContains(t, html, `data-agent="haiku"`)
	assertContains(t, html, `data-depends-on="step-1"`)
	assertContains(t, html, "\u2705") // check emoji for completed
	assertContains(t, html, "\u23F3") // hourglass for pending
}

func TestRenderFeature_WithEdges(t *testing.T) {
	node := &models.Node{
		ID:        "feat-edges01",
		Title:     "Feature with edges",
		Type:      "feature",
		Status:    models.StatusTodo,
		Priority:  models.PriorityMedium,
		CreatedAt: mustParseTime("2026-03-26T09:00:00"),
		UpdatedAt: mustParseTime("2026-03-26T09:00:00"),
		Edges: map[string][]models.Edge{
			"implemented-in": {
				{
					TargetID:     "sess-001",
					Relationship: models.RelImplementedIn,
					Title:        "sess-001",
					Since:        mustParseTime("2026-03-26T08:51:02.622637"),
				},
			},
		},
	}

	html, err := templates.RenderFeature(node)
	if err != nil {
		t.Fatalf("RenderFeature: %v", err)
	}

	assertContains(t, html, `<nav data-graph-edges>`)
	assertContains(t, html, `data-edge-type="implemented-in"`)
	assertContains(t, html, `href="sess-001.html"`)
	assertContains(t, html, `data-relationship="implemented-in"`)
	assertContains(t, html, `data-since="2026-03-26T08:51:02.622637"`)
}

func TestRenderFeature_WithContent(t *testing.T) {
	node := &models.Node{
		ID:        "feat-content01",
		Title:     "Feature with description",
		Type:      "feature",
		Status:    models.StatusTodo,
		Priority:  models.PriorityLow,
		CreatedAt: mustParseTime("2026-03-26T09:00:00"),
		UpdatedAt: mustParseTime("2026-03-26T09:00:00"),
		Content:   "This is the feature description.",
	}

	html, err := templates.RenderFeature(node)
	if err != nil {
		t.Fatalf("RenderFeature: %v", err)
	}

	assertContains(t, html, `<section data-content>`)
	assertContains(t, html, `This is the feature description.`)
}

// --- Round-trip tests ---

func TestRenderFeature_RoundTrip(t *testing.T) {
	original := &models.Node{
		ID:            "feat-roundtrip",
		Title:         "Round trip test",
		Type:          "feature",
		Status:        models.StatusInProgress,
		Priority:      models.PriorityHigh,
		CreatedAt:     mustParseTime("2026-03-26T09:47:14.165536"),
		UpdatedAt:     mustParseTime("2026-03-26T10:12:06.794715"),
		AgentAssigned: "claude-code",
		TrackID:       "trk-test",
		Steps: []models.Step{
			{StepID: "step-0", Description: "Build Go binary", Completed: true},
			{StepID: "step-1", Description: "Create plugin directory", Completed: false},
		},
		Edges: map[string][]models.Edge{
			"implemented-in": {
				{TargetID: "sess-001", Relationship: models.RelImplementedIn, Title: "sess-001"},
			},
		},
		Content: "Round trip test content",
	}

	html, err := templates.RenderFeature(original)
	if err != nil {
		t.Fatalf("RenderFeature: %v", err)
	}

	// Parse the rendered HTML back into a Node
	parsed, err := htmlparse.ParseString(html)
	if err != nil {
		t.Fatalf("ParseString (round-trip): %v", err)
	}

	// Verify all data attributes survived
	assertEqual(t, "ID", parsed.ID, original.ID)
	assertEqual(t, "Title", parsed.Title, original.Title)
	assertEqual(t, "Type", parsed.Type, original.Type)
	assertEqual(t, "Status", string(parsed.Status), string(original.Status))
	assertEqual(t, "Priority", string(parsed.Priority), string(original.Priority))
	assertEqual(t, "AgentAssigned", parsed.AgentAssigned, original.AgentAssigned)
	assertEqual(t, "TrackID", parsed.TrackID, original.TrackID)

	// Timestamps should round-trip (within format precision)
	if parsed.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero after round-trip")
	}
	if parsed.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should not be zero after round-trip")
	}

	// Steps
	if len(parsed.Steps) != len(original.Steps) {
		t.Fatalf("Steps count: got %d, want %d", len(parsed.Steps), len(original.Steps))
	}
	for i, s := range parsed.Steps {
		assertEqual(t, "Step.StepID", s.StepID, original.Steps[i].StepID)
		if s.Completed != original.Steps[i].Completed {
			t.Errorf("Step[%d].Completed: got %v, want %v", i, s.Completed, original.Steps[i].Completed)
		}
	}

	// Edges
	edges := parsed.Edges["implemented-in"]
	if len(edges) != 1 {
		t.Fatalf("Edges count: got %d, want 1", len(edges))
	}
	assertEqual(t, "Edge.TargetID", edges[0].TargetID, "sess-001")

	// Content
	if parsed.Content == "" {
		t.Error("Content should survive round-trip")
	}
}

// --- Bug tests ---

func TestRenderBug_Basic(t *testing.T) {
	node := &models.Node{
		ID:            "bug-xyz789",
		Title:         "Login fails on Safari",
		Type:          "bug",
		Status:        models.StatusDone,
		Priority:      models.PriorityCritical,
		CreatedAt:     mustParseTime("2026-03-23T20:31:47.038707"),
		UpdatedAt:     mustParseTime("2026-03-23T20:51:46.887096"),
		AgentAssigned: "claude-code",
	}

	html, err := templates.RenderBug(node)
	if err != nil {
		t.Fatalf("RenderBug: %v", err)
	}

	assertContains(t, html, `data-type="bug"`)
	assertContains(t, html, `data-status="done"`)
	assertContains(t, html, `data-priority="critical"`)
	assertContains(t, html, `<h1>Login fails on Safari</h1>`)
}

func TestRenderBug_RoundTrip(t *testing.T) {
	original := &models.Node{
		ID:            "bug-rt001",
		Title:         "Bug round trip",
		Type:          "bug",
		Status:        models.StatusTodo,
		Priority:      models.PriorityMedium,
		CreatedAt:     mustParseTime("2026-03-23T20:31:47.038707"),
		UpdatedAt:     mustParseTime("2026-03-23T20:51:46.887096"),
		AgentAssigned: "test-agent",
	}

	html, err := templates.RenderBug(original)
	if err != nil {
		t.Fatalf("RenderBug: %v", err)
	}

	parsed, err := htmlparse.ParseString(html)
	if err != nil {
		t.Fatalf("ParseString (round-trip): %v", err)
	}

	assertEqual(t, "ID", parsed.ID, original.ID)
	assertEqual(t, "Type", parsed.Type, "bug")
	assertEqual(t, "Status", string(parsed.Status), string(original.Status))
	assertEqual(t, "Priority", string(parsed.Priority), string(original.Priority))
	assertEqual(t, "AgentAssigned", parsed.AgentAssigned, original.AgentAssigned)
}

// --- Spike tests ---

func TestRenderSpike_Basic(t *testing.T) {
	node := &models.Node{
		ID:            "spk-spike01",
		Title:         "Investigate caching strategy",
		Type:          "spike",
		Status:        models.StatusTodo,
		Priority:      models.PriorityMedium,
		CreatedAt:     mustParseTime("2026-03-08T06:34:51.619820"),
		UpdatedAt:     mustParseTime("2026-03-08T06:34:51.619823"),
		AgentAssigned: "researcher",
	}

	html, err := templates.RenderSpike(node, "Redis is faster for this use case", "Use Redis", 4)
	if err != nil {
		t.Fatalf("RenderSpike: %v", err)
	}

	assertContains(t, html, `data-type="spike"`)
	assertContains(t, html, `data-spike-type="general"`)
	assertContains(t, html, `data-timebox-hours="4"`)
	assertContains(t, html, `<section data-spike-metadata>`)
	assertContains(t, html, `<section data-findings>`)
	assertContains(t, html, `Redis is faster for this use case`)
	assertContains(t, html, `<section data-decision>`)
	assertContains(t, html, `Use Redis`)
}

func TestRenderSpike_RoundTrip(t *testing.T) {
	original := &models.Node{
		ID:            "spk-rt002",
		Title:         "Spike round trip",
		Type:          "spike",
		Status:        models.StatusDone,
		Priority:      models.PriorityHigh,
		CreatedAt:     mustParseTime("2026-03-08T06:34:51.619820"),
		UpdatedAt:     mustParseTime("2026-03-08T06:34:51.619823"),
		AgentAssigned: "researcher",
	}

	html, err := templates.RenderSpike(original, "", "", 0)
	if err != nil {
		t.Fatalf("RenderSpike: %v", err)
	}

	parsed, err := htmlparse.ParseString(html)
	if err != nil {
		t.Fatalf("ParseString (round-trip): %v", err)
	}

	assertEqual(t, "ID", parsed.ID, original.ID)
	assertEqual(t, "Type", parsed.Type, "spike")
	assertEqual(t, "Status", string(parsed.Status), string(original.Status))
}

// --- Track tests ---

func TestRenderTrack_Basic(t *testing.T) {
	node := &models.Node{
		ID:        "trk-track01",
		Title:     "Go Runtime Migration",
		Type:      "track",
		Status:    models.StatusTodo,
		Priority:  models.PriorityHigh,
		CreatedAt: mustParseTime("2026-03-26T09:00:00"),
		UpdatedAt: mustParseTime("2026-03-26T09:00:00"),
		Steps: []models.Step{
			{StepID: "step-0", Description: "Scaffold Go module", Completed: true},
			{StepID: "step-1", Description: "Port core models", Completed: false},
		},
	}

	html, err := templates.RenderTrack(node)
	if err != nil {
		t.Fatalf("RenderTrack: %v", err)
	}

	assertContains(t, html, `data-type="track"`)
	assertContains(t, html, `<title>Track: Go Runtime Migration</title>`)
	assertContains(t, html, `<h1>Track: Go Runtime Migration</h1>`)
	assertContains(t, html, `data-steps`)
}

func TestRenderTrack_RoundTrip(t *testing.T) {
	original := &models.Node{
		ID:        "trk-rt003",
		Title:     "Track round trip",
		Type:      "track",
		Status:    models.StatusInProgress,
		Priority:  models.PriorityMedium,
		CreatedAt: mustParseTime("2026-03-26T09:00:00"),
		UpdatedAt: mustParseTime("2026-03-26T09:00:00"),
	}

	html, err := templates.RenderTrack(original)
	if err != nil {
		t.Fatalf("RenderTrack: %v", err)
	}

	parsed, err := htmlparse.ParseString(html)
	if err != nil {
		t.Fatalf("ParseString (round-trip): %v", err)
	}

	assertEqual(t, "ID", parsed.ID, original.ID)
	assertEqual(t, "Type", parsed.Type, "track")
	assertEqual(t, "Status", string(parsed.Status), string(original.Status))
}

// --- Session tests ---

func TestRenderSession_Basic(t *testing.T) {
	sess := &models.Session{
		SessionID:     "sess-abc123",
		AgentAssigned: "claude-code",
		CreatedAt:     mustParseTime("2026-03-26T09:00:00"),
		TotalEvents:   42,
		Status:        "active",
		IsSubagent:    false,
	}

	html, err := templates.RenderSession(sess)
	if err != nil {
		t.Fatalf("RenderSession: %v", err)
	}

	assertContains(t, html, `data-type="session"`)
	assertContains(t, html, `id="sess-abc123"`)
	assertContains(t, html, `data-agent="claude-code"`)
	assertContains(t, html, `data-event-count="42"`)
	assertContains(t, html, `data-is-subagent="false"`)
	assertContains(t, html, `data-status="active"`)
}

// --- RenderNode dispatch test ---

func TestRenderNode_Dispatch(t *testing.T) {
	types := []string{"feature", "bug", "spike", "track"}
	for _, typ := range types {
		node := &models.Node{
			ID:        typ + "-dispatch01",
			Title:     "Dispatch test for " + typ,
			Type:      typ,
			Status:    models.StatusTodo,
			Priority:  models.PriorityMedium,
			CreatedAt: mustParseTime("2026-03-26T09:00:00"),
			UpdatedAt: mustParseTime("2026-03-26T09:00:00"),
		}

		html, err := templates.RenderNode(node)
		if err != nil {
			t.Fatalf("RenderNode(%s): %v", typ, err)
		}

		assertContains(t, html, `data-type="`+typ+`"`)
	}
}

// --- Valid HTML structure tests ---

func TestRenderedHTML_IsWellFormed(t *testing.T) {
	node := &models.Node{
		ID:            "feat-wellformed",
		Title:         "Well-formed HTML test",
		Type:          "feature",
		Status:        models.StatusTodo,
		Priority:      models.PriorityMedium,
		CreatedAt:     mustParseTime("2026-03-26T09:00:00"),
		UpdatedAt:     mustParseTime("2026-03-26T09:00:00"),
		AgentAssigned: "test",
		TrackID:       "trk-test",
		Content:       "Some content",
		Steps: []models.Step{
			{StepID: "s1", Description: "Step 1", Completed: true},
		},
		Edges: map[string][]models.Edge{
			"blocks": {{TargetID: "feat-other", Relationship: models.RelBlocks, Title: "Other"}},
		},
	}

	html, err := templates.RenderFeature(node)
	if err != nil {
		t.Fatalf("RenderFeature: %v", err)
	}

	// Should start with DOCTYPE
	if !strings.HasPrefix(html, "<!DOCTYPE html>") {
		t.Error("HTML should start with <!DOCTYPE html>")
	}

	// Should contain opening and closing tags
	assertContains(t, html, `<html lang="en">`)
	assertContains(t, html, `</html>`)
	assertContains(t, html, `<head>`)
	assertContains(t, html, `</head>`)
	assertContains(t, html, `<body>`)
	assertContains(t, html, `</body>`)
	assertContains(t, html, `<article`)
	assertContains(t, html, `</article>`)
}

// --- Helpers ---

func assertContains(t *testing.T, html, substr string) {
	t.Helper()
	if !strings.Contains(html, substr) {
		t.Errorf("HTML missing %q\n(first 500 chars: %s)", substr, truncate(html, 500))
	}
}

func assertEqual(t *testing.T, field, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %q, want %q", field, got, want)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
