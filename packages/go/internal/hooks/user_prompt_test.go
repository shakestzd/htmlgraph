package hooks

import (
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/shakestzd/htmlgraph/internal/db"
	"github.com/shakestzd/htmlgraph/internal/models"
)

// setupTestDB creates an in-memory DB with schema, a session, and an
// optional feature. Returns the database and a cleanup function.
func setupTestDB(t *testing.T) *testDB {
	t.Helper()
	database, err := db.Open("file::memory:?cache=shared&_busy_timeout=5000")
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	now := time.Now().UTC()

	sess := &models.Session{
		SessionID:     "test-sess",
		AgentAssigned: "claude-code",
		CreatedAt:     now,
		Status:        "active",
		Model:         "sonnet-4",
	}
	if err := db.InsertSession(database, sess); err != nil {
		t.Fatalf("InsertSession: %v", err)
	}

	return &testDB{DB: database, now: now, t: t}
}

type testDB struct {
	DB  *sql.DB
	now time.Time
	t   *testing.T
}

func (td *testDB) addTrack(id, title string) {
	td.t.Helper()
	now := td.now.Format(time.RFC3339)
	_, err := td.DB.Exec(
		`INSERT INTO tracks (id, title, status, created_at, updated_at) VALUES (?,?,?,?,?)`,
		id, title, "active", now, now,
	)
	if err != nil {
		td.t.Fatalf("insert track: %v", err)
	}
}

func (td *testDB) addFeature(id, ftype, title, status string) {
	td.t.Helper()
	feat := &db.Feature{
		ID:        id,
		Type:      ftype,
		Title:     title,
		Status:    status,
		Priority:  "medium",
		CreatedAt: td.now,
		UpdatedAt: td.now,
	}
	if err := db.InsertFeature(td.DB, feat); err != nil {
		td.t.Fatalf("InsertFeature(%s): %v", id, err)
	}
}

func (td *testDB) setActiveFeature(sessionID, featureID string) {
	td.t.Helper()
	_, err := td.DB.Exec(
		`UPDATE sessions SET active_feature_id = ? WHERE session_id = ?`,
		featureID, sessionID,
	)
	if err != nil {
		td.t.Fatalf("setActiveFeature: %v", err)
	}
}

func TestUserPrompt_EmptyPrompt(t *testing.T) {
	td := setupTestDB(t)
	defer td.DB.Close()

	os.Setenv("HTMLGRAPH_SESSION_ID", "test-sess")
	defer os.Unsetenv("HTMLGRAPH_SESSION_ID")

	event := &CloudEvent{SessionID: "test-sess", Prompt: ""}
	result, err := UserPrompt(event, td.DB)
	if err != nil {
		t.Fatalf("UserPrompt: %v", err)
	}
	if !result.Continue {
		t.Error("expected Continue=true for empty prompt")
	}
}

func TestUserPrompt_InsertsUserQuery(t *testing.T) {
	td := setupTestDB(t)
	defer td.DB.Close()

	os.Setenv("HTMLGRAPH_SESSION_ID", "test-sess")
	defer os.Unsetenv("HTMLGRAPH_SESSION_ID")

	event := &CloudEvent{SessionID: "test-sess", Prompt: "implement a new API endpoint"}
	_, err := UserPrompt(event, td.DB)
	if err != nil {
		t.Fatalf("UserPrompt: %v", err)
	}

	// Verify a UserQuery event was inserted.
	var count int
	if err := td.DB.QueryRow(
		`SELECT COUNT(*) FROM agent_events WHERE session_id = 'test-sess' AND tool_name = 'UserQuery'`,
	).Scan(&count); err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 UserQuery event, got %d", count)
	}
}

func TestUserPrompt_WithOpenItems_ReturnsAttribution(t *testing.T) {
	td := setupTestDB(t)
	defer td.DB.Close()

	os.Setenv("HTMLGRAPH_SESSION_ID", "test-sess")
	defer os.Unsetenv("HTMLGRAPH_SESSION_ID")

	// Add features so attribution block is generated.
	td.addFeature("feat-aaa", "feature", "Auth System", "in-progress")
	td.addFeature("feat-bbb", "feature", "Dashboard", "todo")
	td.setActiveFeature("test-sess", "feat-aaa")

	event := &CloudEvent{SessionID: "test-sess", Prompt: "show me the current status"}
	result, err := UserPrompt(event, td.DB)
	if err != nil {
		t.Fatalf("UserPrompt: %v", err)
	}

	if result.AdditionalContext == "" {
		t.Fatal("expected AdditionalContext with attribution guidance")
	}
	if !strings.Contains(result.AdditionalContext, "feat-aaa") {
		t.Errorf("guidance should reference active feature, got: %s", result.AdditionalContext)
	}
}

func TestUserPrompt_ImplementationWithSpike_WarnsAboutSpike(t *testing.T) {
	td := setupTestDB(t)
	defer td.DB.Close()

	os.Setenv("HTMLGRAPH_SESSION_ID", "test-sess")
	defer os.Unsetenv("HTMLGRAPH_SESSION_ID")

	td.addFeature("spk-001", "spike", "Research caching", "in-progress")
	td.setActiveFeature("test-sess", "spk-001")

	event := &CloudEvent{SessionID: "test-sess", Prompt: "implement the caching layer"}
	result, err := UserPrompt(event, td.DB)
	if err != nil {
		t.Fatalf("UserPrompt: %v", err)
	}

	if result.AdditionalContext == "" {
		t.Fatal("expected AdditionalContext with spike warning")
	}
	if !strings.Contains(result.AdditionalContext, "spike") {
		t.Errorf("guidance should warn about spike, got: %s", result.AdditionalContext)
	}
}

func TestUserPrompt_Dedup(t *testing.T) {
	td := setupTestDB(t)
	defer td.DB.Close()

	os.Setenv("HTMLGRAPH_SESSION_ID", "test-sess")
	defer os.Unsetenv("HTMLGRAPH_SESSION_ID")

	event := &CloudEvent{SessionID: "test-sess", Prompt: "hello world"}
	_, err := UserPrompt(event, td.DB)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}

	// Second identical call within 5s should be deduped.
	result, err := UserPrompt(event, td.DB)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if !result.Continue {
		t.Error("expected Continue=true for deduped prompt")
	}
}

func TestUserPrompt_SanitizesXMLBlocks(t *testing.T) {
	td := setupTestDB(t)
	defer td.DB.Close()

	os.Setenv("HTMLGRAPH_SESSION_ID", "test-sess")
	defer os.Unsetenv("HTMLGRAPH_SESSION_ID")

	prompt := "<system-reminder>internal stuff</system-reminder>implement auth"
	event := &CloudEvent{SessionID: "test-sess", Prompt: prompt}
	_, err := UserPrompt(event, td.DB)
	if err != nil {
		t.Fatalf("UserPrompt: %v", err)
	}

	// Verify the stored summary does not contain the XML block.
	var summary string
	if err := td.DB.QueryRow(
		`SELECT input_summary FROM agent_events WHERE session_id = 'test-sess' AND tool_name = 'UserQuery'`,
	).Scan(&summary); err != nil {
		t.Fatalf("query: %v", err)
	}
	if strings.Contains(summary, "system-reminder") {
		t.Errorf("stored summary should not contain XML block, got: %s", summary)
	}
	if !strings.Contains(summary, "implement auth") {
		t.Errorf("stored summary should contain actual prompt, got: %s", summary)
	}
}

func TestGetActiveWorkItemType(t *testing.T) {
	td := setupTestDB(t)
	defer td.DB.Close()

	td.addFeature("feat-001", "feature", "Auth", "in-progress")
	td.addFeature("spk-001", "spike", "Research", "in-progress")

	if got := getActiveWorkItemType(td.DB, "feat-001"); got != "feature" {
		t.Errorf("expected 'feature', got %q", got)
	}
	if got := getActiveWorkItemType(td.DB, "spk-001"); got != "spike" {
		t.Errorf("expected 'spike', got %q", got)
	}
	if got := getActiveWorkItemType(td.DB, "nonexistent"); got != "" {
		t.Errorf("expected empty for nonexistent, got %q", got)
	}
	if got := getActiveWorkItemType(td.DB, ""); got != "" {
		t.Errorf("expected empty for empty ID, got %q", got)
	}
}
