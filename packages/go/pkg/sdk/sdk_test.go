package sdk_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shakestzd/htmlgraph/internal/htmlparse"
	"github.com/shakestzd/htmlgraph/internal/models"
	"github.com/shakestzd/htmlgraph/pkg/sdk"
)

// newTestSDK creates an SDK rooted in a temp dir with the required subdirectories.
func newTestSDK(t *testing.T) *sdk.SDK {
	t.Helper()
	dir := t.TempDir()
	for _, sub := range []string{"features", "bugs", "spikes", "tracks", "sessions"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", sub, err)
		}
	}
	s, err := sdk.New(dir, "test-agent")
	if err != nil {
		t.Fatalf("sdk.New: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// ---------------------------------------------------------------------------
// Feature CRUD
// ---------------------------------------------------------------------------

func TestFeatureCreate(t *testing.T) {
	s := newTestSDK(t)

	feat, err := s.Features.Create("User Authentication",
		sdk.FeatWithPriority("high"),
		sdk.FeatWithTrack("trk-test"),
		sdk.FeatWithSteps("Design schema", "Implement API", "Add tests"),
		sdk.FeatWithContent("<p>Auth feature for multi-tenant</p>"),
	)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Verify returned node
	if !strings.HasPrefix(feat.ID, "feat-") {
		t.Errorf("ID prefix: got %q, want feat-*", feat.ID)
	}
	if feat.Title != "User Authentication" {
		t.Errorf("Title: got %q", feat.Title)
	}
	if feat.Type != "feature" {
		t.Errorf("Type: got %q", feat.Type)
	}
	if string(feat.Priority) != "high" {
		t.Errorf("Priority: got %q", feat.Priority)
	}
	if string(feat.Status) != "todo" {
		t.Errorf("Status: got %q", feat.Status)
	}
	if feat.TrackID != "trk-test" {
		t.Errorf("TrackID: got %q", feat.TrackID)
	}
	if feat.AgentAssigned != "test-agent" {
		t.Errorf("AgentAssigned: got %q", feat.AgentAssigned)
	}
	if len(feat.Steps) != 3 {
		t.Fatalf("Steps count: got %d, want 3", len(feat.Steps))
	}
	if feat.Steps[0].Description != "Design schema" {
		t.Errorf("Step[0]: got %q", feat.Steps[0].Description)
	}

	// Verify HTML file exists on disk
	htmlPath := filepath.Join(s.FeaturesDir(), feat.ID+".html")
	if _, err := os.Stat(htmlPath); err != nil {
		t.Fatalf("HTML file not found: %v", err)
	}
}

func TestFeatureCreateEmptyTitle(t *testing.T) {
	s := newTestSDK(t)
	_, err := s.Features.Create("")
	if err == nil {
		t.Error("expected error for empty title")
	}
}

func TestFeatureGet(t *testing.T) {
	s := newTestSDK(t)

	created, err := s.Features.Create("Get Test Feature",
		sdk.FeatWithPriority("low"),
	)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := s.Features.Get(created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got.ID != created.ID {
		t.Errorf("ID mismatch: got %q, want %q", got.ID, created.ID)
	}
	if got.Title != "Get Test Feature" {
		t.Errorf("Title: got %q", got.Title)
	}
	if string(got.Priority) != "low" {
		t.Errorf("Priority: got %q", got.Priority)
	}
}

func TestFeatureList(t *testing.T) {
	s := newTestSDK(t)

	_, _ = s.Features.Create("Feat A", sdk.FeatWithPriority("high"))
	_, _ = s.Features.Create("Feat B", sdk.FeatWithPriority("low"))
	_, _ = s.Features.Create("Feat C", sdk.FeatWithPriority("high"))

	// List all
	all, err := s.Features.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("List all: got %d, want 3", len(all))
	}

	// Filter by priority
	high, err := s.Features.List(sdk.WithPriority("high"))
	if err != nil {
		t.Fatalf("List high: %v", err)
	}
	if len(high) != 2 {
		t.Errorf("List high: got %d, want 2", len(high))
	}
}

func TestFeatureListWithStatus(t *testing.T) {
	s := newTestSDK(t)

	f1, _ := s.Features.Create("Active Feature")
	_, _ = s.Features.Create("Todo Feature")
	_, _ = s.Features.Start(f1.ID)

	inProg, err := s.Features.List(sdk.WithStatus("in-progress"))
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(inProg) != 1 {
		t.Errorf("in-progress count: got %d, want 1", len(inProg))
	}
}

func TestFeatureDelete(t *testing.T) {
	s := newTestSDK(t)

	feat, _ := s.Features.Create("Delete Me")
	if err := s.Features.Delete(feat.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := s.Features.Get(feat.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

// ---------------------------------------------------------------------------
// Feature Lifecycle
// ---------------------------------------------------------------------------

func TestFeatureStartComplete(t *testing.T) {
	s := newTestSDK(t)

	feat, _ := s.Features.Create("Lifecycle Test",
		sdk.FeatWithSteps("Step 1", "Step 2"),
	)

	// Start
	started, err := s.Features.Start(feat.ID)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if string(started.Status) != "in-progress" {
		t.Errorf("after Start: status = %q", started.Status)
	}

	// Complete
	done, err := s.Features.Complete(feat.ID)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if string(done.Status) != "done" {
		t.Errorf("after Complete: status = %q", done.Status)
	}
	for i, step := range done.Steps {
		if !step.Completed {
			t.Errorf("step %d not completed after Complete", i)
		}
	}
}

// ---------------------------------------------------------------------------
// Round-trip: create -> write HTML -> parse -> verify
// ---------------------------------------------------------------------------

func TestFeatureRoundTrip(t *testing.T) {
	s := newTestSDK(t)

	feat, err := s.Features.Create("Round Trip Feature",
		sdk.FeatWithPriority("critical"),
		sdk.FeatWithTrack("trk-roundtrip"),
		sdk.FeatWithSteps("Alpha", "Beta"),
		sdk.FeatWithContent("<p>Round trip test</p>"),
	)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Re-parse the HTML file with the internal parser
	path := filepath.Join(s.FeaturesDir(), feat.ID+".html")
	parsed, err := htmlparse.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	assertEqual(t, "ID", parsed.ID, feat.ID)
	assertEqual(t, "Title", parsed.Title, "Round Trip Feature")
	assertEqual(t, "Type", parsed.Type, "feature")
	assertEqual(t, "Status", string(parsed.Status), "todo")
	assertEqual(t, "Priority", string(parsed.Priority), "critical")
	assertEqual(t, "TrackID", parsed.TrackID, "trk-roundtrip")
	assertEqual(t, "AgentAssigned", parsed.AgentAssigned, "test-agent")

	if len(parsed.Steps) != 2 {
		t.Fatalf("Steps count: got %d, want 2", len(parsed.Steps))
	}
	assertEqual(t, "Step[0]", parsed.Steps[0].Description, "Alpha")
	assertEqual(t, "Step[1]", parsed.Steps[1].Description, "Beta")

	if !strings.Contains(parsed.Content, "Round trip test") {
		t.Errorf("Content missing expected text: %q", parsed.Content)
	}
}

func TestFeatureWithEdgesRoundTrip(t *testing.T) {
	s := newTestSDK(t)

	feat, err := s.Features.Create("Edge Feature",
		sdk.FeatWithEdge("blocks", "feat-other"),
	)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	path := filepath.Join(s.FeaturesDir(), feat.ID+".html")
	parsed, err := htmlparse.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	edges, ok := parsed.Edges["blocks"]
	if !ok || len(edges) == 0 {
		t.Fatal("no 'blocks' edges found after round-trip")
	}
	assertEqual(t, "edge target", edges[0].TargetID, "feat-other")
}

// ---------------------------------------------------------------------------
// Bug CRUD
// ---------------------------------------------------------------------------

func TestBugCreate(t *testing.T) {
	s := newTestSDK(t)

	bug, err := s.Bugs.Create("Login broken on Safari",
		sdk.BugWithPriority("critical"),
		sdk.BugWithReproSteps("Open Safari", "Click login"),
	)
	if err != nil {
		t.Fatalf("Create bug: %v", err)
	}
	if !strings.HasPrefix(bug.ID, "bug-") {
		t.Errorf("ID prefix: got %q, want bug-*", bug.ID)
	}
	if bug.Type != "bug" {
		t.Errorf("Type: got %q", bug.Type)
	}

	// Verify file
	path := filepath.Join(s.BugsDir(), bug.ID+".html")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("HTML file not found: %v", err)
	}
}

func TestBugRoundTrip(t *testing.T) {
	s := newTestSDK(t)
	bug, _ := s.Bugs.Create("Bug RT", sdk.BugWithPriority("high"))

	parsed, err := htmlparse.ParseFile(filepath.Join(s.BugsDir(), bug.ID+".html"))
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	assertEqual(t, "Type", parsed.Type, "bug")
	assertEqual(t, "Priority", string(parsed.Priority), "high")
}

// ---------------------------------------------------------------------------
// Spike CRUD
// ---------------------------------------------------------------------------

func TestSpikeCreate(t *testing.T) {
	s := newTestSDK(t)

	spike, err := s.Spikes.Create("Investigate caching",
		sdk.SpikeWithType("technical"),
		sdk.SpikeWithFindings("Redis is the best option"),
	)
	if err != nil {
		t.Fatalf("Create spike: %v", err)
	}
	if !strings.HasPrefix(spike.ID, "spk-") {
		t.Errorf("ID prefix: got %q, want spk-*", spike.ID)
	}
	if spike.Type != "spike" {
		t.Errorf("Type: got %q", spike.Type)
	}
}

func TestSpikeSetFindings(t *testing.T) {
	s := newTestSDK(t)

	spike, _ := s.Spikes.Create("Investigation")
	updated, err := s.Spikes.SetFindings(spike.ID, "Found the root cause")
	if err != nil {
		t.Fatalf("SetFindings: %v", err)
	}
	if !strings.Contains(updated.Content, "Found the root cause") {
		t.Errorf("Content: got %q", updated.Content)
	}

	// Verify round-trip
	parsed, _ := htmlparse.ParseFile(filepath.Join(s.SpikesDir(), spike.ID+".html"))
	if !strings.Contains(parsed.Content, "Found the root cause") {
		t.Errorf("Parsed content missing findings: %q", parsed.Content)
	}
}

func TestSpikeGetLatest(t *testing.T) {
	s := newTestSDK(t)

	_, _ = s.Spikes.Create("Spike A")
	_, _ = s.Spikes.Create("Spike B")
	_, _ = s.Spikes.Create("Spike C")

	latest, err := s.Spikes.GetLatest("", 2)
	if err != nil {
		t.Fatalf("GetLatest: %v", err)
	}
	if len(latest) != 2 {
		t.Errorf("GetLatest count: got %d, want 2", len(latest))
	}
}

// ---------------------------------------------------------------------------
// Track CRUD
// ---------------------------------------------------------------------------

func TestTrackCreate(t *testing.T) {
	s := newTestSDK(t)

	track, err := s.Tracks.Create("Go SDK Port",
		sdk.TrackWithPriority("high"),
		sdk.TrackWithSpec("Port Python SDK to Go"),
		sdk.TrackWithPlanPhases("Phase 1: Models", "Phase 2: Collections"),
	)
	if err != nil {
		t.Fatalf("Create track: %v", err)
	}
	if !strings.HasPrefix(track.ID, "trk-") {
		t.Errorf("ID prefix: got %q, want trk-*", track.ID)
	}
	if len(track.Steps) != 2 {
		t.Errorf("Steps: got %d, want 2", len(track.Steps))
	}
}

func TestTrackRoundTrip(t *testing.T) {
	s := newTestSDK(t)
	track, _ := s.Tracks.Create("Track RT", sdk.TrackWithPriority("medium"))

	parsed, err := htmlparse.ParseFile(filepath.Join(s.TracksDir(), track.ID+".html"))
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	assertEqual(t, "Type", parsed.Type, "track")
	assertEqual(t, "Title", parsed.Title, "Track RT")
}

// ---------------------------------------------------------------------------
// Collection.Filter
// ---------------------------------------------------------------------------

func TestFeatureFilter(t *testing.T) {
	s := newTestSDK(t)

	_, _ = s.Features.Create("AAA Feature")
	_, _ = s.Features.Create("BBB Feature")
	_, _ = s.Features.Create("AAA Other")

	filtered, err := s.Features.Filter(func(n *models.Node) bool {
		return strings.Contains(n.Title, "AAA")
	})
	if err != nil {
		t.Fatalf("Filter: %v", err)
	}
	if len(filtered) != 2 {
		t.Errorf("Filter AAA: got %d, want 2", len(filtered))
	}
}

// ---------------------------------------------------------------------------
// ID Generation
// ---------------------------------------------------------------------------

func TestIDGeneration(t *testing.T) {
	s := newTestSDK(t)

	f1, _ := s.Features.Create("Feature One")
	f2, _ := s.Features.Create("Feature Two")

	if f1.ID == f2.ID {
		t.Error("two features should have different IDs")
	}
	if !strings.HasPrefix(f1.ID, "feat-") {
		t.Errorf("f1 ID prefix: got %q", f1.ID)
	}
	if !strings.HasPrefix(f2.ID, "feat-") {
		t.Errorf("f2 ID prefix: got %q", f2.ID)
	}
	// IDs should be prefix + 8 hex chars
	parts := strings.SplitN(f1.ID, "-", 2)
	if len(parts) != 2 || len(parts[1]) != 8 {
		t.Errorf("ID format: got %q, want feat-XXXXXXXX", f1.ID)
	}
}

// ---------------------------------------------------------------------------
// SDK init validation
// ---------------------------------------------------------------------------

func TestNewSDKRequiresAgent(t *testing.T) {
	dir := t.TempDir()
	_, err := sdk.New(dir, "")
	if err == nil {
		t.Error("expected error for empty agent")
	}
}

func TestNewSDKRequiresProjectDir(t *testing.T) {
	_, err := sdk.New("", "agent")
	if err == nil {
		t.Error("expected error for empty projectDir")
	}
}

// ---------------------------------------------------------------------------
// Claim / Release / AtomicClaim / Unclaim
// ---------------------------------------------------------------------------

func TestClaimReleaseCycle(t *testing.T) {
	s := newTestSDK(t)
	feat, err := s.Features.Create("Claim Test")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Claim
	if err := s.Features.Claim(feat.ID, "sess-001"); err != nil {
		t.Fatalf("Claim: %v", err)
	}

	got, err := s.Features.Get(feat.ID)
	if err != nil {
		t.Fatalf("Get after claim: %v", err)
	}
	assertEqual(t, "AgentAssigned", got.AgentAssigned, "test-agent")
	assertEqual(t, "ClaimedBySession", got.ClaimedBySession, "sess-001")
	if got.ClaimedAt == "" {
		t.Error("ClaimedAt should be set after Claim")
	}

	// Release
	if err := s.Features.Release(feat.ID); err != nil {
		t.Fatalf("Release: %v", err)
	}

	got, err = s.Features.Get(feat.ID)
	if err != nil {
		t.Fatalf("Get after release: %v", err)
	}
	assertEqual(t, "AgentAssigned after release", got.AgentAssigned, "")
	assertEqual(t, "ClaimedBySession after release", got.ClaimedBySession, "")
	assertEqual(t, "ClaimedAt after release", got.ClaimedAt, "")
}

func TestAtomicClaimSuccess(t *testing.T) {
	s := newTestSDK(t)
	feat, _ := s.Features.Create("Atomic Claim Test")

	// First claim should succeed
	if err := s.Features.AtomicClaim(feat.ID, "sess-001"); err != nil {
		t.Fatalf("AtomicClaim: %v", err)
	}

	got, _ := s.Features.Get(feat.ID)
	assertEqual(t, "AgentAssigned", got.AgentAssigned, "test-agent")
	assertEqual(t, "ClaimedBySession", got.ClaimedBySession, "sess-001")
}

func TestAtomicClaimSameAgentSameSession(t *testing.T) {
	s := newTestSDK(t)
	feat, _ := s.Features.Create("Atomic Reclaim Test")

	// Claim once
	_ = s.Features.AtomicClaim(feat.ID, "sess-001")

	// Same agent, same session should succeed (idempotent)
	if err := s.Features.AtomicClaim(feat.ID, "sess-001"); err != nil {
		t.Fatalf("AtomicClaim same agent/session should succeed: %v", err)
	}
}

func TestAtomicClaimDifferentSessionFails(t *testing.T) {
	s := newTestSDK(t)
	feat, _ := s.Features.Create("Atomic Conflict Test")

	// Claim with session-001
	_ = s.Features.AtomicClaim(feat.ID, "sess-001")

	// Different session should fail
	err := s.Features.AtomicClaim(feat.ID, "sess-002")
	if err == nil {
		t.Error("expected error when claiming with different session")
	}
	if !strings.Contains(err.Error(), "already claimed") {
		t.Errorf("error should mention 'already claimed': %v", err)
	}
}

func TestAtomicClaimDifferentAgentFails(t *testing.T) {
	s := newTestSDK(t)
	feat, _ := s.Features.Create("Agent Conflict Test")

	// Claim with test-agent
	_ = s.Features.Claim(feat.ID, "sess-001")

	// Create second SDK with different agent, same project dir
	s2, err := sdk.New(s.ProjectDir, "other-agent")
	if err != nil {
		t.Fatalf("sdk.New for other-agent: %v", err)
	}
	defer s2.Close()

	// Different agent should fail
	err = s2.Features.AtomicClaim(feat.ID, "sess-002")
	if err == nil {
		t.Error("expected error when claiming with different agent")
	}
	if !strings.Contains(err.Error(), "already claimed") {
		t.Errorf("error should mention 'already claimed': %v", err)
	}
}

func TestUnclaim(t *testing.T) {
	s := newTestSDK(t)
	feat, _ := s.Features.Create("Unclaim Test")

	// Claim then unclaim
	_ = s.Features.Claim(feat.ID, "sess-001")

	if err := s.Features.Unclaim(feat.ID); err != nil {
		t.Fatalf("Unclaim: %v", err)
	}

	got, _ := s.Features.Get(feat.ID)
	// Unclaim preserves AgentAssigned but clears claim metadata
	assertEqual(t, "AgentAssigned preserved", got.AgentAssigned, "test-agent")
	assertEqual(t, "ClaimedBySession cleared", got.ClaimedBySession, "")
	assertEqual(t, "ClaimedAt cleared", got.ClaimedAt, "")
}

func TestClaimNonexistentFails(t *testing.T) {
	s := newTestSDK(t)

	if err := s.Features.Claim("feat-nonexistent", "sess-001"); err == nil {
		t.Error("expected error claiming nonexistent feature")
	}
}

func TestClaimOnBugs(t *testing.T) {
	s := newTestSDK(t)
	bug, _ := s.Bugs.Create("Bug Claim Test")

	if err := s.Bugs.Claim(bug.ID, "sess-001"); err != nil {
		t.Fatalf("Claim bug: %v", err)
	}

	got, _ := s.Bugs.Get(bug.ID)
	assertEqual(t, "Bug AgentAssigned", got.AgentAssigned, "test-agent")
	assertEqual(t, "Bug ClaimedBySession", got.ClaimedBySession, "sess-001")
}

func TestClaimOnSpikes(t *testing.T) {
	s := newTestSDK(t)
	spike, _ := s.Spikes.Create("Spike Claim Test")

	if err := s.Spikes.Claim(spike.ID, "sess-001"); err != nil {
		t.Fatalf("Claim spike: %v", err)
	}

	got, _ := s.Spikes.Get(spike.ID)
	assertEqual(t, "Spike AgentAssigned", got.AgentAssigned, "test-agent")
}

// ---------------------------------------------------------------------------
// GetActiveWorkItem
// ---------------------------------------------------------------------------

func TestGetActiveWorkItemNoActive(t *testing.T) {
	s := newTestSDK(t)

	// Create items but leave them in todo status
	_, _ = s.Features.Create("Todo Feature")
	_, _ = s.Bugs.Create("Todo Bug")

	item, err := s.GetActiveWorkItem()
	if err != nil {
		t.Fatalf("GetActiveWorkItem: %v", err)
	}
	if item != nil {
		t.Errorf("expected nil, got %+v", item)
	}
}

func TestGetActiveWorkItemOneActive(t *testing.T) {
	s := newTestSDK(t)

	feat, _ := s.Features.Create("Active Feature")
	_, _ = s.Features.Start(feat.ID)

	item, err := s.GetActiveWorkItem()
	if err != nil {
		t.Fatalf("GetActiveWorkItem: %v", err)
	}
	if item == nil {
		t.Fatal("expected active work item, got nil")
	}
	assertEqual(t, "WorkItem.ID", item.ID, feat.ID)
	assertEqual(t, "WorkItem.Type", item.Type, "feature")
	assertEqual(t, "WorkItem.Title", item.Title, "Active Feature")
	assertEqual(t, "WorkItem.Status", item.Status, "in-progress")
}

func TestGetActiveWorkItemPrefersFeatures(t *testing.T) {
	s := newTestSDK(t)

	// Start a feature and a bug
	feat, _ := s.Features.Create("Active Feature")
	bug, _ := s.Bugs.Create("Active Bug")
	_, _ = s.Features.Start(feat.ID)
	_, _ = s.Bugs.Start(bug.ID)

	// Features are scanned first, so the feature should be returned
	item, err := s.GetActiveWorkItem()
	if err != nil {
		t.Fatalf("GetActiveWorkItem: %v", err)
	}
	if item == nil {
		t.Fatal("expected active work item, got nil")
	}
	assertEqual(t, "WorkItem.Type", item.Type, "feature")
}

func TestGetActiveWorkItemFindsBug(t *testing.T) {
	s := newTestSDK(t)

	// Only a bug is active
	bug, _ := s.Bugs.Create("Active Bug")
	_, _ = s.Bugs.Start(bug.ID)

	item, err := s.GetActiveWorkItem()
	if err != nil {
		t.Fatalf("GetActiveWorkItem: %v", err)
	}
	if item == nil {
		t.Fatal("expected active work item, got nil")
	}
	assertEqual(t, "WorkItem.Type", item.Type, "bug")
	assertEqual(t, "WorkItem.ID", item.ID, bug.ID)
}

func TestGetActiveWorkItemFindsSpike(t *testing.T) {
	s := newTestSDK(t)

	// Only a spike is active
	spike, _ := s.Spikes.Create("Active Spike")
	_, _ = s.Spikes.Start(spike.ID)

	item, err := s.GetActiveWorkItem()
	if err != nil {
		t.Fatalf("GetActiveWorkItem: %v", err)
	}
	if item == nil {
		t.Fatal("expected active work item, got nil")
	}
	assertEqual(t, "WorkItem.Type", item.Type, "spike")
	assertEqual(t, "WorkItem.ID", item.ID, spike.ID)
}

func TestGetActiveWorkItemEmptyProject(t *testing.T) {
	s := newTestSDK(t)

	item, err := s.GetActiveWorkItem()
	if err != nil {
		t.Fatalf("GetActiveWorkItem: %v", err)
	}
	if item != nil {
		t.Errorf("expected nil for empty project, got %+v", item)
	}
}

// ---------------------------------------------------------------------------
// EditBuilder
// ---------------------------------------------------------------------------

func TestEditSetStatus(t *testing.T) {
	s := newTestSDK(t)
	feat, _ := s.Features.Create("Edit Status Test")

	err := s.Features.Edit(feat.ID).
		SetStatus("in-progress").
		Save()
	if err != nil {
		t.Fatalf("Edit.SetStatus.Save: %v", err)
	}

	got, _ := s.Features.Get(feat.ID)
	assertEqual(t, "Status", string(got.Status), "in-progress")
}

func TestEditSetDescription(t *testing.T) {
	s := newTestSDK(t)
	feat, _ := s.Features.Create("Edit Desc Test")

	// Content must be in an element (e.g. <p>) to survive HTML round-trip,
	// because the parser reads child elements, not raw text nodes.
	err := s.Features.Edit(feat.ID).
		SetDescription("<p>New description body</p>").
		Save()
	if err != nil {
		t.Fatalf("Edit.SetDescription.Save: %v", err)
	}

	got, _ := s.Features.Get(feat.ID)
	if !strings.Contains(got.Content, "New description body") {
		t.Errorf("Content: got %q", got.Content)
	}
}

func TestEditSetFindings(t *testing.T) {
	s := newTestSDK(t)
	spike, _ := s.Spikes.Create("Edit Findings Test")

	err := s.Spikes.Edit(spike.ID).
		SetFindings("Investigation complete: use Redis").
		Save()
	if err != nil {
		t.Fatalf("Edit.SetFindings.Save: %v", err)
	}

	got, _ := s.Spikes.Get(spike.ID)
	if !strings.Contains(got.Content, "Investigation complete: use Redis") {
		t.Errorf("Content: got %q", got.Content)
	}
}

func TestEditAddNote(t *testing.T) {
	s := newTestSDK(t)
	feat, _ := s.Features.Create("Edit Note Test",
		sdk.FeatWithContent("<p>Original content</p>"),
	)

	err := s.Features.Edit(feat.ID).
		AddNote("First note").
		AddNote("Second note").
		Save()
	if err != nil {
		t.Fatalf("Edit.AddNote.Save: %v", err)
	}

	got, _ := s.Features.Get(feat.ID)
	if !strings.Contains(got.Content, "First note") {
		t.Errorf("Content missing first note: %q", got.Content)
	}
	if !strings.Contains(got.Content, "Second note") {
		t.Errorf("Content missing second note: %q", got.Content)
	}
	if !strings.Contains(got.Content, "Original content") {
		t.Errorf("Content missing original: %q", got.Content)
	}
}

func TestEditChainMultipleFields(t *testing.T) {
	s := newTestSDK(t)
	feat, _ := s.Features.Create("Multi Edit Test")

	err := s.Features.Edit(feat.ID).
		SetStatus("in-progress").
		SetPriority("critical").
		SetAgent("new-agent").
		SetTrack("trk-new").
		AddNote("Started work").
		Save()
	if err != nil {
		t.Fatalf("Edit chain Save: %v", err)
	}

	got, _ := s.Features.Get(feat.ID)
	assertEqual(t, "Status", string(got.Status), "in-progress")
	assertEqual(t, "Priority", string(got.Priority), "critical")
	assertEqual(t, "AgentAssigned", got.AgentAssigned, "new-agent")
	assertEqual(t, "TrackID", got.TrackID, "trk-new")
	if !strings.Contains(got.Content, "Started work") {
		t.Errorf("Content missing note: %q", got.Content)
	}
}

func TestEditNonexistentNode(t *testing.T) {
	s := newTestSDK(t)

	err := s.Features.Edit("feat-nonexistent").
		SetStatus("done").
		Save()
	if err == nil {
		t.Error("expected error editing nonexistent node")
	}
}

func TestEditDeferredError(t *testing.T) {
	s := newTestSDK(t)

	// All chained methods should be no-ops when initial load fails
	err := s.Features.Edit("feat-nonexistent").
		SetStatus("done").
		SetPriority("high").
		AddNote("note").
		SetDescription("desc").
		SetFindings("findings").
		SetAgent("agent").
		SetTrack("trk-x").
		Save()
	if err == nil {
		t.Error("expected error from deferred load failure")
	}
}

func TestEditOnBug(t *testing.T) {
	s := newTestSDK(t)
	bug, _ := s.Bugs.Create("Bug Edit Test")

	err := s.Bugs.Edit(bug.ID).
		SetStatus("in-progress").
		AddNote("Investigating").
		Save()
	if err != nil {
		t.Fatalf("Edit bug: %v", err)
	}

	got, _ := s.Bugs.Get(bug.ID)
	assertEqual(t, "Bug status", string(got.Status), "in-progress")
	if !strings.Contains(got.Content, "Investigating") {
		t.Errorf("Bug content: %q", got.Content)
	}
}

// ---------------------------------------------------------------------------
// AddNote (standalone Collection method)
// ---------------------------------------------------------------------------

func TestAddNote(t *testing.T) {
	s := newTestSDK(t)
	feat, _ := s.Features.Create("Note Test",
		sdk.FeatWithContent("<p>Initial</p>"),
	)

	if err := s.Features.AddNote(feat.ID, "First observation"); err != nil {
		t.Fatalf("AddNote: %v", err)
	}

	got, _ := s.Features.Get(feat.ID)
	if !strings.Contains(got.Content, "First observation") {
		t.Errorf("Content missing note: %q", got.Content)
	}
	if !strings.Contains(got.Content, "Initial") {
		t.Errorf("Content lost original: %q", got.Content)
	}
	// Note should include agent name
	if !strings.Contains(got.Content, "test-agent") {
		t.Errorf("Note missing agent: %q", got.Content)
	}
}

func TestAddNoteAppendsMultiple(t *testing.T) {
	s := newTestSDK(t)
	feat, _ := s.Features.Create("Multi Note Test")

	_ = s.Features.AddNote(feat.ID, "Note one")
	_ = s.Features.AddNote(feat.ID, "Note two")
	_ = s.Features.AddNote(feat.ID, "Note three")

	got, _ := s.Features.Get(feat.ID)
	if !strings.Contains(got.Content, "Note one") {
		t.Errorf("Missing note one: %q", got.Content)
	}
	if !strings.Contains(got.Content, "Note two") {
		t.Errorf("Missing note two: %q", got.Content)
	}
	if !strings.Contains(got.Content, "Note three") {
		t.Errorf("Missing note three: %q", got.Content)
	}
}

func TestAddNoteNonexistent(t *testing.T) {
	s := newTestSDK(t)
	err := s.Features.AddNote("feat-nonexistent", "note")
	if err == nil {
		t.Error("expected error for nonexistent feature")
	}
}

func TestAddNoteOnBug(t *testing.T) {
	s := newTestSDK(t)
	bug, _ := s.Bugs.Create("Bug Note Test")

	if err := s.Bugs.AddNote(bug.ID, "Bug observation"); err != nil {
		t.Fatalf("AddNote bug: %v", err)
	}

	got, _ := s.Bugs.Get(bug.ID)
	if !strings.Contains(got.Content, "Bug observation") {
		t.Errorf("Bug content: %q", got.Content)
	}
}

func TestAddNoteOnSpike(t *testing.T) {
	s := newTestSDK(t)
	spike, _ := s.Spikes.Create("Spike Note Test")

	if err := s.Spikes.AddNote(spike.ID, "Spike finding"); err != nil {
		t.Fatalf("AddNote spike: %v", err)
	}

	got, _ := s.Spikes.Get(spike.ID)
	if !strings.Contains(got.Content, "Spike finding") {
		t.Errorf("Spike content: %q", got.Content)
	}
}

// ---------------------------------------------------------------------------
// SetFindings (standalone Collection method)
// ---------------------------------------------------------------------------

func TestSetFindingsReplacesContent(t *testing.T) {
	s := newTestSDK(t)
	spike, _ := s.Spikes.Create("Findings Test",
		sdk.SpikeWithFindings("Old findings"),
	)

	if _, err := s.Spikes.SetFindings(spike.ID, "New findings replace old"); err != nil {
		t.Fatalf("SetFindings: %v", err)
	}

	got, _ := s.Spikes.Get(spike.ID)
	if !strings.Contains(got.Content, "New findings replace old") {
		t.Errorf("Content missing new findings: %q", got.Content)
	}
	// Old findings should be gone (replaced, not appended)
	if strings.Contains(got.Content, "Old findings") {
		t.Errorf("Content still has old findings: %q", got.Content)
	}
}

func TestSetFindingsOnFeature(t *testing.T) {
	s := newTestSDK(t)
	feat, _ := s.Features.Create("Feature Findings Test")

	if err := s.Features.SetFindings(feat.ID, "Feature analysis complete"); err != nil {
		t.Fatalf("SetFindings: %v", err)
	}

	got, _ := s.Features.Get(feat.ID)
	if !strings.Contains(got.Content, "Feature analysis complete") {
		t.Errorf("Content: %q", got.Content)
	}
}

func TestSetFindingsNonexistent(t *testing.T) {
	s := newTestSDK(t)
	_, err := s.Spikes.SetFindings("spk-nonexistent", "findings")
	if err == nil {
		t.Error("expected error for nonexistent spike")
	}
}

// ---------------------------------------------------------------------------
// Claim round-trip with HTML parsing
// ---------------------------------------------------------------------------

func TestClaimRoundTrip(t *testing.T) {
	s := newTestSDK(t)
	feat, _ := s.Features.Create("Claim RT Test")

	_ = s.Features.Claim(feat.ID, "sess-rt-001")

	// Parse the HTML file directly
	path := filepath.Join(s.FeaturesDir(), feat.ID+".html")
	parsed, err := htmlparse.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	assertEqual(t, "parsed.AgentAssigned", parsed.AgentAssigned, "test-agent")
	assertEqual(t, "parsed.ClaimedBySession", parsed.ClaimedBySession, "sess-rt-001")
	if parsed.ClaimedAt == "" {
		t.Error("parsed.ClaimedAt should be set")
	}
}

// ---------------------------------------------------------------------------
// Edit round-trip with HTML parsing
// ---------------------------------------------------------------------------

func TestEditRoundTrip(t *testing.T) {
	s := newTestSDK(t)
	feat, _ := s.Features.Create("Edit RT Test")

	_ = s.Features.Edit(feat.ID).
		SetStatus("in-progress").
		SetPriority("high").
		AddNote("Implementation started").
		Save()

	path := filepath.Join(s.FeaturesDir(), feat.ID+".html")
	parsed, err := htmlparse.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	assertEqual(t, "parsed.Status", string(parsed.Status), "in-progress")
	assertEqual(t, "parsed.Priority", string(parsed.Priority), "high")
	if !strings.Contains(parsed.Content, "Implementation started") {
		t.Errorf("parsed.Content missing note: %q", parsed.Content)
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func assertEqual(t *testing.T, field, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %q, want %q", field, got, want)
	}
}
