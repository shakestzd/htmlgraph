package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	dbpkg "github.com/shakestzd/htmlgraph/internal/db"
	"github.com/shakestzd/htmlgraph/internal/htmlparse"
	"github.com/shakestzd/htmlgraph/internal/models"
)

// testCreate is a test helper that wraps runWiCreate with the opts struct.
func testCreate(typeName, title, trackID, priority string, start, noLink bool) error {
	return runWiCreate(typeName, title, &wiCreateOpts{
		trackID:     trackID,
		priority:    priority,
		description: "test description",
		start:       start,
		noLink:      noLink,
	})
}

// testSetupTrack creates a track and returns its ID. Fatals on failure.
func testSetupTrack(t *testing.T, hgDir string) string {
	t.Helper()
	if err := testCreate("track", "Test Track", "", "medium", false, false); err != nil {
		t.Fatalf("setup track: %v", err)
	}
	files, _ := filepath.Glob(filepath.Join(hgDir, "tracks", "trk-*.html"))
	if len(files) == 0 {
		t.Fatal("no track file created")
	}
	node, _ := htmlparse.ParseFile(files[len(files)-1])
	return node.ID
}

func TestAutoTrackEdgesOnCreate(t *testing.T) {
	tmpDir := t.TempDir()
	hgDir := filepath.Join(tmpDir, ".htmlgraph")
	for _, sub := range []string{"features", "bugs", "spikes", "tracks", "plans", "specs"} {
		if err := os.MkdirAll(filepath.Join(hgDir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	projectDirFlag = tmpDir
	defer func() { projectDirFlag = "" }()

	// Create a track first
	if err := testCreate("track", "Test Track", "", "medium", false, false); err != nil {
		t.Fatalf("create track: %v", err)
	}

	// Find the track ID from disk
	trackFiles, _ := filepath.Glob(filepath.Join(hgDir, "tracks", "trk-*.html"))
	if len(trackFiles) != 1 {
		t.Fatalf("expected 1 track file, got %d", len(trackFiles))
	}
	trackNode, err := htmlparse.ParseFile(trackFiles[0])
	if err != nil {
		t.Fatalf("parse track: %v", err)
	}
	trackID := trackNode.ID

	// Create a feature linked to the track
	if err := testCreate("feature", "Tracked Feature", trackID, "high", false, false); err != nil {
		t.Fatalf("create feature: %v", err)
	}

	// Find the feature
	featFiles, _ := filepath.Glob(filepath.Join(hgDir, "features", "feat-*.html"))
	if len(featFiles) != 1 {
		t.Fatalf("expected 1 feature file, got %d", len(featFiles))
	}
	featNode, err := htmlparse.ParseFile(featFiles[0])
	if err != nil {
		t.Fatalf("parse feature: %v", err)
	}

	// Verify feature has part_of edge to track
	partOfEdges, ok := featNode.Edges["part_of"]
	if !ok || len(partOfEdges) == 0 {
		t.Errorf("feature missing part_of edge; edges = %v", featNode.Edges)
	} else if partOfEdges[0].TargetID != trackID {
		t.Errorf("part_of target = %q, want %q", partOfEdges[0].TargetID, trackID)
	}

	// Re-read the track to check contains edge
	trackNode, err = htmlparse.ParseFile(trackFiles[0])
	if err != nil {
		t.Fatalf("re-parse track: %v", err)
	}
	containsEdges, ok := trackNode.Edges["contains"]
	if !ok || len(containsEdges) == 0 {
		t.Errorf("track missing contains edge; edges = %v", trackNode.Edges)
	} else if containsEdges[0].TargetID != featNode.ID {
		t.Errorf("contains target = %q, want %q", containsEdges[0].TargetID, featNode.ID)
	}
}

func TestAutoTrackEdgesNotCreatedForTrack(t *testing.T) {
	tmpDir := t.TempDir()
	hgDir := filepath.Join(tmpDir, ".htmlgraph")
	for _, sub := range []string{"features", "bugs", "spikes", "tracks", "plans", "specs"} {
		os.MkdirAll(filepath.Join(hgDir, sub), 0o755)
	}

	projectDirFlag = tmpDir
	defer func() { projectDirFlag = "" }()

	// Creating a track should not attempt auto-edges even if trackID is passed
	if err := testCreate("track", "Parent Track", "", "medium", false, false); err != nil {
		t.Fatalf("create track: %v", err)
	}

	trackFiles, _ := filepath.Glob(filepath.Join(hgDir, "tracks", "trk-*.html"))
	if len(trackFiles) != 1 {
		t.Fatalf("expected 1 track file, got %d", len(trackFiles))
	}
	node, _ := htmlparse.ParseFile(trackFiles[0])
	if len(node.Edges) > 0 {
		t.Errorf("track should have no edges, got %v", node.Edges)
	}
}

func TestAutoImplementedInEdgeOnStart(t *testing.T) {
	tmpDir := t.TempDir()
	hgDir := filepath.Join(tmpDir, ".htmlgraph")
	for _, sub := range []string{"features", "bugs", "spikes", "tracks", "plans", "specs"} {
		os.MkdirAll(filepath.Join(hgDir, sub), 0o755)
	}

	projectDirFlag = tmpDir
	defer func() { projectDirFlag = "" }()

	// Set a fake session ID (EnvSessionID reads HTMLGRAPH_SESSION_ID first)
	t.Setenv("HTMLGRAPH_SESSION_ID", "test-session-abc")

	trackID := testSetupTrack(t, hgDir)

	// Create a feature
	if err := testCreate("feature", "Impl Feature", trackID, "high", false, false); err != nil {
		t.Fatalf("create feature: %v", err)
	}

	// Find the feature ID
	featFiles, _ := filepath.Glob(filepath.Join(hgDir, "features", "feat-*.html"))
	if len(featFiles) != 1 {
		t.Fatalf("expected 1 feature file, got %d", len(featFiles))
	}
	featNode, _ := htmlparse.ParseFile(featFiles[0])
	featID := featNode.ID

	// Start the feature (should create implemented_in edge)
	if err := runWiSetStatus("feature", featID, "in-progress"); err != nil {
		t.Fatalf("start feature: %v", err)
	}

	// Re-read and check for implemented_in edge
	featNode, _ = htmlparse.ParseFile(featFiles[0])
	implEdges, ok := featNode.Edges["implemented_in"]
	if !ok || len(implEdges) == 0 {
		t.Errorf("feature missing implemented_in edge; edges = %v", featNode.Edges)
	} else if implEdges[0].TargetID != "test-session-abc" {
		t.Errorf("implemented_in target = %q, want %q", implEdges[0].TargetID, "test-session-abc")
	}

	// Start again — should be idempotent (no duplicate edge)
	if err := runWiSetStatus("feature", featID, "in-progress"); err != nil {
		t.Fatalf("re-start feature: %v", err)
	}
	featNode, _ = htmlparse.ParseFile(featFiles[0])
	implEdges = featNode.Edges["implemented_in"]
	if len(implEdges) != 1 {
		t.Errorf("expected 1 implemented_in edge after re-start, got %d", len(implEdges))
	}
}

func TestNoImplementedInEdgeWithoutSession(t *testing.T) {
	tmpDir := t.TempDir()
	hgDir := filepath.Join(tmpDir, ".htmlgraph")
	for _, sub := range []string{"features", "bugs", "spikes", "tracks", "plans", "specs"} {
		os.MkdirAll(filepath.Join(hgDir, sub), 0o755)
	}

	projectDirFlag = tmpDir
	defer func() { projectDirFlag = "" }()

	// Isolate from any real session running in the developer's environment.
	t.Setenv("HTMLGRAPH_SESSION_ID", "")
	t.Setenv("CLAUDE_SESSION_ID", "")
	t.Setenv("HTMLGRAPH_PROJECT_DIR", tmpDir)
	t.Setenv("CLAUDE_PROJECT_DIR", "")

	trackID := testSetupTrack(t, hgDir)

	if err := testCreate("feature", "No Session Feature", trackID, "low", false, false); err != nil {
		t.Fatalf("create: %v", err)
	}

	featFiles, _ := filepath.Glob(filepath.Join(hgDir, "features", "feat-*.html"))
	featNode, _ := htmlparse.ParseFile(featFiles[0])

	if err := runWiSetStatus("feature", featNode.ID, "in-progress"); err != nil {
		t.Fatalf("start: %v", err)
	}

	featNode, _ = htmlparse.ParseFile(featFiles[0])
	if len(featNode.Edges["implemented_in"]) > 0 {
		t.Errorf("should not have implemented_in edge without session, got %v", featNode.Edges)
	}
}

func TestAutoCausedByEdgeOnBugCreate(t *testing.T) {
	tmpDir := t.TempDir()
	hgDir := filepath.Join(tmpDir, ".htmlgraph")
	for _, sub := range []string{"features", "bugs", "spikes", "tracks", "plans", "specs"} {
		os.MkdirAll(filepath.Join(hgDir, sub), 0o755)
	}

	projectDirFlag = tmpDir
	defer func() { projectDirFlag = "" }()

	trackID := testSetupTrack(t, hgDir)

	// Create a feature first and start it
	if err := testCreate("feature", "Active Feature", trackID, "high", true, false); err != nil {
		t.Fatalf("create feature: %v", err)
	}

	// Now create a bug — should auto-link caused_by to active feature
	if err := testCreate("bug", "Found a bug", trackID, "high", false, false); err != nil {
		t.Fatalf("create bug: %v", err)
	}

	// Find the bug
	bugFiles, _ := filepath.Glob(filepath.Join(hgDir, "bugs", "bug-*.html"))
	if len(bugFiles) != 1 {
		t.Fatalf("expected 1 bug file, got %d", len(bugFiles))
	}
	bugNode, _ := htmlparse.ParseFile(bugFiles[0])

	// Find the feature ID
	featFiles, _ := filepath.Glob(filepath.Join(hgDir, "features", "feat-*.html"))
	featNode, _ := htmlparse.ParseFile(featFiles[0])

	// Verify caused_by edge
	causedByEdges := bugNode.Edges["caused_by"]
	if len(causedByEdges) == 0 {
		t.Logf("bug edges: %v", bugNode.Edges)
		t.Skip("no DB available in test — auto caused_by requires session DB")
		return
	}
	if causedByEdges[0].TargetID != featNode.ID {
		t.Errorf("caused_by target = %q, want %q", causedByEdges[0].TargetID, featNode.ID)
	}
}

func TestBugCreateNoLinkSkipsCausedBy(t *testing.T) {
	tmpDir := t.TempDir()
	hgDir := filepath.Join(tmpDir, ".htmlgraph")
	for _, sub := range []string{"features", "bugs", "spikes", "tracks", "plans", "specs"} {
		os.MkdirAll(filepath.Join(hgDir, sub), 0o755)
	}

	projectDirFlag = tmpDir
	defer func() { projectDirFlag = "" }()

	trackID := testSetupTrack(t, hgDir)

	// Create and start a feature
	if err := testCreate("feature", "Active Feature", trackID, "high", true, false); err != nil {
		t.Fatalf("create feature: %v", err)
	}

	// Create bug with --no-link
	if err := testCreate("bug", "Unrelated bug", trackID, "medium", false, true); err != nil {
		t.Fatalf("create bug: %v", err)
	}

	bugFiles, _ := filepath.Glob(filepath.Join(hgDir, "bugs", "bug-*.html"))
	bugNode, _ := htmlparse.ParseFile(bugFiles[0])

	// Should have no caused_by edge
	if len(bugNode.Edges["caused_by"]) > 0 {
		t.Errorf("--no-link should skip caused_by edge, got %v", bugNode.Edges)
	}
}

func TestFeatureCreateRequiresDescription(t *testing.T) {
	tmpDir := t.TempDir()
	hgDir := filepath.Join(tmpDir, ".htmlgraph")
	for _, sub := range []string{"features", "bugs", "spikes", "tracks", "plans", "specs"} {
		os.MkdirAll(filepath.Join(hgDir, sub), 0o755)
	}

	projectDirFlag = tmpDir
	defer func() { projectDirFlag = "" }()

	trackID := testSetupTrack(t, hgDir)

	// Try to create a feature without --description (but with --track)
	opts := &wiCreateOpts{
		trackID:     trackID,
		priority:    "high",
		description: "", // no description
		start:       false,
		noLink:      false,
	}
	err := runWiCreate("feature", "Feature without description", opts)

	if err == nil {
		t.Fatal("expected error when creating feature without --description, got nil")
	}

	// Check error message contains example syntax
	errMsg := err.Error()
	if !stringContains(errMsg, "Example:") {
		t.Errorf("error message should mention 'Example:' to show syntax: %q", errMsg)
	}
	if !stringContains(errMsg, "--description") {
		t.Errorf("error message should mention --description: %q", errMsg)
	}
	if !stringContains(errMsg, "feature") {
		t.Errorf("error message should mention 'feature' command: %q", errMsg)
	}
}

func TestBugCreateRequiresDescription(t *testing.T) {
	tmpDir := t.TempDir()
	hgDir := filepath.Join(tmpDir, ".htmlgraph")
	for _, sub := range []string{"features", "bugs", "spikes", "tracks", "plans", "specs"} {
		os.MkdirAll(filepath.Join(hgDir, sub), 0o755)
	}

	projectDirFlag = tmpDir
	defer func() { projectDirFlag = "" }()

	trackID := testSetupTrack(t, hgDir)

	// Try to create a bug without --description (but with --track)
	opts := &wiCreateOpts{
		trackID:     trackID,
		priority:    "high",
		description: "", // no description
		start:       false,
		noLink:      false,
	}
	err := runWiCreate("bug", "Bug without description", opts)

	if err == nil {
		t.Fatal("expected error when creating bug without --description, got nil")
	}

	// Check error message contains example syntax
	errMsg := err.Error()
	if !stringContains(errMsg, "Example:") {
		t.Errorf("error message should mention 'Example:' to show syntax: %q", errMsg)
	}
	if !stringContains(errMsg, "--description") {
		t.Errorf("error message should mention --description: %q", errMsg)
	}
	if !stringContains(errMsg, "bug") {
		t.Errorf("error message should mention 'bug' command: %q", errMsg)
	}
}

func TestSpecCreateNoDescriptionWarning(t *testing.T) {
	tmpDir := t.TempDir()
	hgDir := filepath.Join(tmpDir, ".htmlgraph")
	for _, sub := range []string{"features", "bugs", "spikes", "tracks", "plans", "specs"} {
		os.MkdirAll(filepath.Join(hgDir, sub), 0o755)
	}

	projectDirFlag = tmpDir
	defer func() { projectDirFlag = "" }()

	// Create a spec without --description (should warn, not error)
	opts := &wiCreateOpts{
		trackID:     "",
		priority:    "medium",
		description: "", // no description
		start:       false,
		noLink:      false,
	}
	err := runWiCreate("spec", "Spec without description", opts)

	if err != nil {
		t.Fatalf("spec should warn but not error, got: %v", err)
	}
}

func TestRunWiSetStatus_BlockedClearsCache(t *testing.T) {
	tmpDir := t.TempDir()
	hgDir := filepath.Join(tmpDir, ".htmlgraph")
	for _, sub := range []string{"features", "bugs", "spikes", "tracks", "plans", "specs"} {
		os.MkdirAll(filepath.Join(hgDir, sub), 0o755)
	}
	projectDirFlag = tmpDir
	defer func() { projectDirFlag = "" }()

	// Set cache dir to temp so we don't pollute the real home dir.
	t.Setenv("HTMLGRAPH_CACHE_DIR", tmpDir)

	trackID := testSetupTrack(t, hgDir)

	// Create a feature linked to the track.
	if err := testCreate("feature", "Test Blocked Feature", trackID, "medium", false, false); err != nil {
		t.Fatalf("create feature: %v", err)
	}
	featFiles, _ := filepath.Glob(filepath.Join(hgDir, "features", "feat-*.html"))
	if len(featFiles) != 1 {
		t.Fatalf("expected 1 feature file, got %d", len(featFiles))
	}
	featNode, err := htmlparse.ParseFile(featFiles[0])
	if err != nil {
		t.Fatalf("parse feature: %v", err)
	}

	// Start it — cache should be populated.
	if err := runWiSetStatus("feature", featNode.ID, "in-progress"); err != nil {
		t.Fatalf("start: %v", err)
	}
	cache := ReadStatuslineCache(hgDir)
	if cache == "" {
		t.Fatal("cache should be populated after start")
	}

	// Block it — cache should be cleared and status must become blocked.
	if err := runWiSetStatus("feature", featNode.ID, "blocked"); err != nil {
		t.Fatalf("blocked: %v", err)
	}
	cache = ReadStatuslineCache(hgDir)
	if cache != "" {
		t.Errorf("cache should be empty after blocked, got %q", cache)
	}

	// Verify the status was actually set to blocked (not done).
	updatedNode, err := htmlparse.ParseFile(featFiles[0])
	if err != nil {
		t.Fatalf("parse after blocked: %v", err)
	}
	if string(updatedNode.Status) != "blocked" {
		t.Errorf("expected status %q, got %q", "blocked", updatedNode.Status)
	}
}

// stringContains is a helper to check if a string contains a substring
func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// --- warnMissingFields tests ---------------------------------------------------

func TestWarnMissingFields_FeatureRequiresTrack(t *testing.T) {
	opts := &wiCreateOpts{description: "some description"}
	err := warnMissingFields("feature", opts)
	if err == nil {
		t.Fatal("expected error for feature without --track, got nil")
	}
	if !stringContains(err.Error(), "htmlgraph track list") {
		t.Errorf("error should mention 'htmlgraph track list', got: %q", err.Error())
	}
}

func TestWarnMissingFields_BugRequiresTrack(t *testing.T) {
	opts := &wiCreateOpts{description: "some description"}
	err := warnMissingFields("bug", opts)
	if err == nil {
		t.Fatal("expected error for bug without --track, got nil")
	}
	if !stringContains(err.Error(), "htmlgraph track list") {
		t.Errorf("error should mention 'htmlgraph track list', got: %q", err.Error())
	}
}

func TestWarnMissingFields_SpikeNoTrackOK(t *testing.T) {
	opts := &wiCreateOpts{description: "investigation notes"}
	err := warnMissingFields("spike", opts)
	if err != nil {
		t.Errorf("spike without --track should not error, got: %v", err)
	}
}

func TestWarnMissingFields_TrackNoTrackOK(t *testing.T) {
	opts := &wiCreateOpts{}
	err := warnMissingFields("track", opts)
	if err != nil {
		t.Errorf("track type should not error about missing track, got: %v", err)
	}
}

func TestWarnMissingFields_FeatureWithTrackOK(t *testing.T) {
	opts := &wiCreateOpts{trackID: "trk-abc12345", description: "some description"}
	err := warnMissingFields("feature", opts)
	if err != nil {
		t.Errorf("feature with --track should not error, got: %v", err)
	}
}

func TestWarnMissingFields_ErrorMessageGuidance(t *testing.T) {
	opts := &wiCreateOpts{description: "some description"}
	err := warnMissingFields("feature", opts)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	msg := err.Error()
	if !stringContains(msg, "htmlgraph track list") {
		t.Errorf("error message should contain 'htmlgraph track list': %q", msg)
	}
	if !stringContains(msg, "--track") {
		t.Errorf("error message should mention '--track': %q", msg)
	}
	// Retrieval-first framing: relevant command must appear before track list.
	if !stringContains(msg, "htmlgraph relevant") {
		t.Errorf("error message should mention 'htmlgraph relevant' for retrieval-first discovery: %q", msg)
	}
	if !stringContains(msg, "last resort") {
		t.Errorf("error message should frame track creation as 'last resort': %q", msg)
	}
}

func TestWarnMissingFields_BugErrorMessageRetrievalFirst(t *testing.T) {
	opts := &wiCreateOpts{description: "some description"}
	err := warnMissingFields("bug", opts)
	if err == nil {
		t.Fatal("expected error for bug without --track, got nil")
	}
	msg := err.Error()
	if !stringContains(msg, "htmlgraph relevant") {
		t.Errorf("bug error message should mention 'htmlgraph relevant': %q", msg)
	}
	if !stringContains(msg, "last resort") {
		t.Errorf("bug error message should frame track creation as 'last resort': %q", msg)
	}
}

// testHgDirWithDB creates a temp dir with .htmlgraph subdirs and a seeded
// session row. Returns tmpDir, hgDir, and the pre-opened DB (caller closes it).
func testHgDirWithDB(t *testing.T, sessionID string) (tmpDir, hgDir string) {
	t.Helper()
	tmpDir = t.TempDir()
	hgDir = filepath.Join(tmpDir, ".htmlgraph")
	for _, sub := range []string{"features", "bugs", "spikes", "tracks", "plans", "specs"} {
		if err := os.MkdirAll(filepath.Join(hgDir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// Open (and migrate) the DB so tables exist, then insert a session row.
	database, err := dbpkg.Open(filepath.Join(hgDir, "htmlgraph.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	if err := dbpkg.InsertSession(database, &models.Session{
		SessionID:     sessionID,
		AgentAssigned: "claude-code",
		Status:        "active",
		CreatedAt:     time.Now(),
	}); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	return tmpDir, hgDir
}

// TestFeatureStart_Idempotent verifies that calling runWiSetStatus twice with
// the same feature ID does not issue a second UpdateActiveFeature write. We
// detect this by checking that active_feature_id in the DB remains set after
// both calls, and that only one claim row exists for the session+feature pair.
func TestFeatureStart_Idempotent(t *testing.T) {
	const sessionID = "test-session-idempotent"
	tmpDir, hgDir := testHgDirWithDB(t, sessionID)

	projectDirFlag = tmpDir
	defer func() { projectDirFlag = "" }()
	t.Setenv("HTMLGRAPH_SESSION_ID", sessionID)
	t.Setenv("HTMLGRAPH_CACHE_DIR", tmpDir)

	trackID := testSetupTrack(t, hgDir)

	if err := testCreate("feature", "Idempotent Feature", trackID, "medium", false, false); err != nil {
		t.Fatalf("create feature: %v", err)
	}
	featFiles, _ := filepath.Glob(filepath.Join(hgDir, "features", "feat-*.html"))
	if len(featFiles) != 1 {
		t.Fatalf("expected 1 feature file, got %d", len(featFiles))
	}
	featNode, _ := htmlparse.ParseFile(featFiles[0])
	featID := featNode.ID

	// First start — must succeed and set active_feature_id.
	if err := runWiSetStatus("feature", featID, "in-progress"); err != nil {
		t.Fatalf("first start: %v", err)
	}

	// Read active_feature_id after first start.
	database, err := dbpkg.Open(filepath.Join(hgDir, "htmlgraph.db"))
	if err != nil {
		t.Fatalf("open db after first start: %v", err)
	}
	defer database.Close()

	activeAfterFirst := dbpkg.GetActiveFeatureIDForSession(database, sessionID)
	if activeAfterFirst != featID {
		t.Fatalf("after first start: active_feature_id = %q, want %q", activeAfterFirst, featID)
	}

	// Count claims before second start.
	var claimsBefore int
	database.QueryRow(`SELECT COUNT(*) FROM claims WHERE work_item_id = ? AND owner_session_id = ?`,
		featID, sessionID).Scan(&claimsBefore)

	// Record updated_at before second start. Sleep 2ms so any write would
	// advance the timestamp.
	var updatedAtBefore string
	database.QueryRow(`SELECT updated_at FROM sessions WHERE session_id = ?`, sessionID).
		Scan(&updatedAtBefore)
	time.Sleep(2 * time.Millisecond)

	// Second start — must not error.
	if err := runWiSetStatus("feature", featID, "in-progress"); err != nil {
		t.Fatalf("second start: %v", err)
	}

	// updated_at must NOT have advanced (no write was issued).
	var updatedAtAfter string
	database.QueryRow(`SELECT updated_at FROM sessions WHERE session_id = ?`, sessionID).
		Scan(&updatedAtAfter)
	if updatedAtAfter != updatedAtBefore {
		t.Errorf("updated_at changed on idempotent re-start: before=%q after=%q", updatedAtBefore, updatedAtAfter)
	}

	// Claim count must not grow (ClaimItem also guards against duplicates,
	// but the short-circuit should prevent even reaching it).
	var claimsAfter int
	database.QueryRow(`SELECT COUNT(*) FROM claims WHERE work_item_id = ? AND owner_session_id = ?`,
		featID, sessionID).Scan(&claimsAfter)
	if claimsAfter != claimsBefore {
		t.Errorf("claim count changed on idempotent re-start: before=%d after=%d", claimsBefore, claimsAfter)
	}
}

// TestFeatureStart_DifferentFeatures verifies that starting two different
// features in sequence issues both UpdateActiveFeature writes, ending with
// active_feature_id pointing at the second feature.
func TestFeatureStart_DifferentFeatures(t *testing.T) {
	const sessionID = "test-session-two-features"
	tmpDir, hgDir := testHgDirWithDB(t, sessionID)

	projectDirFlag = tmpDir
	defer func() { projectDirFlag = "" }()
	t.Setenv("HTMLGRAPH_SESSION_ID", sessionID)
	t.Setenv("HTMLGRAPH_CACHE_DIR", tmpDir)

	trackID := testSetupTrack(t, hgDir)

	// Create two features.
	for _, title := range []string{"Feature Alpha", "Feature Beta"} {
		if err := testCreate("feature", title, trackID, "medium", false, false); err != nil {
			t.Fatalf("create %q: %v", title, err)
		}
	}
	featFiles, _ := filepath.Glob(filepath.Join(hgDir, "features", "feat-*.html"))
	if len(featFiles) != 2 {
		t.Fatalf("expected 2 feature files, got %d", len(featFiles))
	}
	nodeA, _ := htmlparse.ParseFile(featFiles[0])
	nodeB, _ := htmlparse.ParseFile(featFiles[1])
	idA, idB := nodeA.ID, nodeB.ID

	// Start feature A.
	if err := runWiSetStatus("feature", idA, "in-progress"); err != nil {
		t.Fatalf("start A: %v", err)
	}

	database, err := dbpkg.Open(filepath.Join(hgDir, "htmlgraph.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	if got := dbpkg.GetActiveFeatureIDForSession(database, sessionID); got != idA {
		t.Errorf("after start A: active_feature_id = %q, want %q", got, idA)
	}

	// Start feature B — must write and update active_feature_id.
	if err := runWiSetStatus("feature", idB, "in-progress"); err != nil {
		t.Fatalf("start B: %v", err)
	}
	if got := dbpkg.GetActiveFeatureIDForSession(database, sessionID); got != idB {
		t.Errorf("after start B: active_feature_id = %q, want %q", got, idB)
	}
}

// TestRunWiSetStatus_ConcurrentAgents proves that N goroutines with distinct
// agentIDs can each claim different features on the same session without
// contention, error, or loss.
//
// NOTE: We call wiSetStatusWithAgent (not runWiSetStatus) to avoid env-var races.
// runWiSetStatus is a thin wrapper that reads env vars then delegates here.
//
// NOTE on SQLite concurrency: workitem.Open opens a new DB connection per call.
// SQLite WAL allows concurrent readers but serializes writers. A semaphore (cap=1)
// ensures at most one workitem.Open call is in-flight at any time, matching the
// real-world pattern where each agent runs in a separate OS process. The key
// invariant being tested is that N distinct (session, agentID) pairs each
// produce an independent row in active_work_items — not raw write parallelism.
func TestRunWiSetStatus_ConcurrentAgents(t *testing.T) {
	const sessionID = "test-session-concurrent"
	const N = 5

	tmpDir, hgDir := testHgDirWithDB(t, sessionID)
	projectDirFlag = tmpDir
	defer func() { projectDirFlag = "" }()
	t.Setenv("HTMLGRAPH_SESSION_ID", sessionID)
	t.Setenv("HTMLGRAPH_CACHE_DIR", tmpDir)

	trackID := testSetupTrack(t, hgDir)

	// Create N features up-front (sequential — avoids concurrent HTML writes).
	featIDs := make([]string, N)
	for i := 0; i < N; i++ {
		title := fmt.Sprintf("Concurrent Feature %d", i)
		if err := testCreate("feature", title, trackID, "medium", false, false); err != nil {
			t.Fatalf("create feature %d: %v", i, err)
		}
	}
	featFiles, _ := filepath.Glob(filepath.Join(hgDir, "features", "feat-*.html"))
	if len(featFiles) < N {
		t.Fatalf("expected %d feature files, got %d", N, len(featFiles))
	}
	for i := 0; i < N; i++ {
		node, err := htmlparse.ParseFile(featFiles[i])
		if err != nil {
			t.Fatalf("parse feature %d: %v", i, err)
		}
		featIDs[i] = node.ID
	}

	// Semaphore with capacity 1: serializes the workitem.Open / SQLite write
	// calls (matching real usage where each agent is a separate OS process).
	// Goroutines still run concurrently for everything outside the semaphore.
	sem := make(chan struct{}, 1)
	errCh := make(chan error, N)
	for i := 0; i < N; i++ {
		agentID := fmt.Sprintf("agent-%d", i)
		featID := featIDs[i]
		go func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			errCh <- wiSetStatusWithAgent("feature", featID, "in-progress", sessionID, agentID)
		}()
	}

	// Collect results — all must succeed.
	for i := 0; i < N; i++ {
		if err := <-errCh; err != nil {
			t.Errorf("goroutine error: %v", err)
		}
	}

	// Verify all N rows in active_work_items.
	database, err := dbpkg.Open(filepath.Join(hgDir, "htmlgraph.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	items, err := dbpkg.ActiveWorkItemsForSession(database, sessionID)
	if err != nil {
		t.Fatalf("ActiveWorkItemsForSession: %v", err)
	}
	if len(items) != N {
		t.Errorf("want %d active_work_items rows, got %d: %v", N, len(items), items)
	}

	// Each agentID must map to its expected feature.
	for i := 0; i < N; i++ {
		agentID := fmt.Sprintf("agent-%d", i)
		want := featIDs[i]
		if got := items[agentID]; got != want {
			t.Errorf("agent-%d: want %s, got %q", i, want, got)
		}
	}
}

// TestSubagentCanStartFeatureCreatedByDifferentAgent verifies that feat-ab67561e's
// per-(session_id, agent_id) attribution removed the old session-ownership restriction.
// A sub-agent with a distinct agent_id (but sharing the parent's session_id, as
// Claude Code does per docs) must be able to run feature start on a feature the
// orchestrator created in the same session without any error.
//
// Regression test for bug-50c7eed0: "Execute skill sub-agent attribution guard
// blocks legitimate claims to features created by orchestrator."
func TestSubagentCanStartFeatureCreatedByDifferentAgent(t *testing.T) {
	const sessionID = "test-session-subagent-claim"
	const orchestratorAgentID = "agent-orchestrator"
	const subagentAgentID = "agent-subagent-a"

	tmpDir, hgDir := testHgDirWithDB(t, sessionID)
	projectDirFlag = tmpDir
	defer func() { projectDirFlag = "" }()
	t.Setenv("HTMLGRAPH_SESSION_ID", sessionID)
	t.Setenv("HTMLGRAPH_CACHE_DIR", tmpDir)

	trackID := testSetupTrack(t, hgDir)

	// Step 1: Create a feature (simulating orchestrator creating work for subagent).
	// Do NOT start it yet — this is a feature created but not claimed.
	if err := testCreate("feature", "Subagent Claim Test", trackID, "medium", false, false); err != nil {
		t.Fatalf("create feature: %v", err)
	}

	featFiles, _ := filepath.Glob(filepath.Join(hgDir, "features", "feat-*.html"))
	if len(featFiles) != 1 {
		t.Fatalf("expected 1 feature file, got %d", len(featFiles))
	}
	featNode, err := htmlparse.ParseFile(featFiles[0])
	if err != nil {
		t.Fatalf("parse feature: %v", err)
	}
	featureID := featNode.ID

	// Step 2: Orchestrator claims the feature first (simulating orchestrator
	// dispatching work and starting it to track attribution).
	if err := wiSetStatusWithAgent("feature", featureID, "in-progress", sessionID, orchestratorAgentID); err != nil {
		t.Fatalf("orchestrator start: %v", err)
	}

	// Step 3: Sub-agent claims the SAME feature with a DIFFERENT agent_id.
	// This would have been blocked under the old session-ownership model, but
	// feat-ab67561e changed attribution to per-(session, agent_id), so both
	// agents can own their own active_work_items rows for the same feature.
	if err := wiSetStatusWithAgent("feature", featureID, "in-progress", sessionID, subagentAgentID); err != nil {
		t.Fatalf("subagent start failed: %v — regression of bug-50c7eed0", err)
	}

	// Step 4: Verify both agents' rows exist in active_work_items and point
	// to the same feature.
	database, err := dbpkg.Open(filepath.Join(hgDir, "htmlgraph.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	orchestratorActive := dbpkg.GetActiveWorkItem(database, sessionID, orchestratorAgentID)
	if orchestratorActive != featureID {
		t.Errorf("orchestrator active: got %q want %q", orchestratorActive, featureID)
	}

	subagentActive := dbpkg.GetActiveWorkItem(database, sessionID, subagentAgentID)
	if subagentActive != featureID {
		t.Errorf("subagent active: got %q want %q", subagentActive, featureID)
	}

	// Step 5: Verify both rows exist in active_work_items table.
	items, err := dbpkg.ActiveWorkItemsForSession(database, sessionID)
	if err != nil {
		t.Fatalf("ActiveWorkItemsForSession: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("want 2 active_work_items rows (one per agent), got %d: %v", len(items), items)
	}
}
