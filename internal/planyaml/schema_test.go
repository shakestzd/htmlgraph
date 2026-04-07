package planyaml

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestNewPlan_Defaults(t *testing.T) {
	p := NewPlan("plan-abcd1234", "Test Plan", "A description")

	if p.Meta.ID != "plan-abcd1234" {
		t.Errorf("Meta.ID = %q, want %q", p.Meta.ID, "plan-abcd1234")
	}
	if p.Meta.Title != "Test Plan" {
		t.Errorf("Meta.Title = %q, want %q", p.Meta.Title, "Test Plan")
	}
	if p.Meta.Description != "A description" {
		t.Errorf("Meta.Description = %q, want %q", p.Meta.Description, "A description")
	}
	if p.Meta.Status != "draft" {
		t.Errorf("Meta.Status = %q, want %q", p.Meta.Status, "draft")
	}
	if p.Meta.CreatedAt == "" {
		t.Error("Meta.CreatedAt should be set")
	}
	if p.Design.Problem != "" {
		t.Errorf("Design.Problem should be empty, got %q", p.Design.Problem)
	}
	if len(p.Design.Goals) != 0 {
		t.Errorf("Design.Goals should be empty, got %v", p.Design.Goals)
	}
	if len(p.Slices) != 0 {
		t.Errorf("Slices should be empty, got %d items", len(p.Slices))
	}
	if len(p.Questions) != 0 {
		t.Errorf("Questions should be empty, got %d items", len(p.Questions))
	}
	if p.Critique != nil {
		t.Error("Critique should be nil for a new plan")
	}
}

func TestNewPlan_WithTrack(t *testing.T) {
	p := NewPlan("plan-abcd1234", "Test Plan", "desc")
	p.Meta.TrackID = "trk-12345678"

	if p.Meta.TrackID != "trk-12345678" {
		t.Errorf("Meta.TrackID = %q, want %q", p.Meta.TrackID, "trk-12345678")
	}
}

func TestSaveLoad_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plan-test.yaml")

	original := &PlanYAML{
		Meta: PlanMeta{
			ID:          "plan-bc1e8c2c",
			TrackID:     "trk-9ae9e043",
			Title:       "Hooks System Optimization",
			Description: "Reduce hook latency.",
			CreatedAt:   "2026-04-05",
			Status:      "draft",
			CreatedBy:   "claude-opus",
		},
		Design: PlanDesign{
			Problem: "The hook system is slow.",
			Goals:   []string{"SessionStart <1s", "PreToolUse <100ms"},
			Constraints: []string{
				"Hooks must never block Claude",
				"No external dependencies",
			},
			Approved: false,
			Comment:  "",
		},
		Slices: []PlanSlice{
			{
				ID:       "feat-058c4074",
				Num:      1,
				Title:    "Deduplicate helper functions",
				What:     "Merge resolveAgentID and resolveEventAgentID.",
				Why:      "3 duplicate function pairs identified.",
				Files:    []string{"internal/hooks/tooluse_shared.go", "internal/hooks/log.go"},
				Deps:     []int{},
				DoneWhen: []string{"resolveEventAgentID removed"},
				Effort:   "S",
				Risk:     "Low",
				Tests:    "Unit: resolveAgentID returns event.AgentID when set",
				Approved: false,
				Comment:  "",
			},
			{
				ID:       "bug-9778dff9",
				Num:      2,
				Title:    "Fix PreToolUse subagent blocking",
				What:     "Fix the guard logic.",
				Why:      "Subagents get blocked.",
				Files:    []string{"internal/hooks/pretooluse.go"},
				Deps:     []int{1},
				DoneWhen: []string{"Subagent with no claim but session feature can write"},
				Effort:   "S",
				Risk:     "Med",
				Tests:    "Unit: PreToolUse allows subagent",
				Approved: false,
				Comment:  "",
			},
		},
		Questions: []PlanQuestion{
			{
				ID:          "q-migration",
				Text:        "Migration caching strategy?",
				Description: "db.Open() runs 9 ALTER TABLE migrations per hook invocation.",
				Recommended: "schema-version",
				Options: []QuestionOption{
					{Key: "schema-version", Label: "A: Schema version"},
					{Key: "pragma-only", Label: "B: Pragma only"},
				},
				Answer: nil,
			},
		},
		Critique: &PlanCritique{
			ReviewedAt: "2026-04-05",
			Reviewers:  []string{"Haiku (design review)"},
			Assumptions: []CritiqueAssumption{
				{
					ID:       "A1",
					Status:   "verified",
					Text:     "resolveAgentID and resolveEventAgentID are identical",
					Evidence: "tooluse_shared.go:103-120",
				},
			},
			Critics: []CriticSection{
				{
					Title: "HAIKU -- DESIGN REVIEW",
					Sections: []CriticSubsection{
						{
							Heading: "Slice Assessment",
							Items: []CriticItem{
								{
									Badge: "S1",
									Kind:  "success",
									Text:  "Dedup scope correct.",
								},
							},
						},
					},
				},
			},
			Risks: []CritiqueRisk{
				{
					Risk:       "Second GetGitRemoteURL call unaddressed",
					Severity:   "High",
					Mitigation: "Goroutine both calls in S3",
				},
			},
			Synthesis: "The plan's structure is sound.",
		},
	}

	if err := Save(path, original); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify meta
	if loaded.Meta.ID != original.Meta.ID {
		t.Errorf("Meta.ID = %q, want %q", loaded.Meta.ID, original.Meta.ID)
	}
	if loaded.Meta.TrackID != original.Meta.TrackID {
		t.Errorf("Meta.TrackID = %q, want %q", loaded.Meta.TrackID, original.Meta.TrackID)
	}
	if loaded.Meta.Title != original.Meta.Title {
		t.Errorf("Meta.Title = %q, want %q", loaded.Meta.Title, original.Meta.Title)
	}
	if loaded.Meta.Status != original.Meta.Status {
		t.Errorf("Meta.Status = %q, want %q", loaded.Meta.Status, original.Meta.Status)
	}
	if loaded.Meta.CreatedBy != original.Meta.CreatedBy {
		t.Errorf("Meta.CreatedBy = %q, want %q", loaded.Meta.CreatedBy, original.Meta.CreatedBy)
	}

	// Verify design
	if loaded.Design.Problem != original.Design.Problem {
		t.Errorf("Design.Problem mismatch")
	}
	if len(loaded.Design.Goals) != len(original.Design.Goals) {
		t.Errorf("Design.Goals len = %d, want %d", len(loaded.Design.Goals), len(original.Design.Goals))
	}
	if loaded.Design.Approved != original.Design.Approved {
		t.Errorf("Design.Approved = %v, want %v", loaded.Design.Approved, original.Design.Approved)
	}

	// Verify slices
	if len(loaded.Slices) != len(original.Slices) {
		t.Fatalf("Slices len = %d, want %d", len(loaded.Slices), len(original.Slices))
	}
	s0 := loaded.Slices[0]
	if s0.ID != "feat-058c4074" {
		t.Errorf("Slice[0].ID = %q, want %q", s0.ID, "feat-058c4074")
	}
	if s0.Num != 1 {
		t.Errorf("Slice[0].Num = %d, want 1", s0.Num)
	}
	if len(s0.Files) != 2 {
		t.Errorf("Slice[0].Files len = %d, want 2", len(s0.Files))
	}
	if len(s0.Deps) != 0 {
		t.Errorf("Slice[0].Deps len = %d, want 0", len(s0.Deps))
	}
	s1 := loaded.Slices[1]
	if len(s1.Deps) != 1 || s1.Deps[0] != 1 {
		t.Errorf("Slice[1].Deps = %v, want [1]", s1.Deps)
	}

	// Verify questions
	if len(loaded.Questions) != 1 {
		t.Fatalf("Questions len = %d, want 1", len(loaded.Questions))
	}
	q0 := loaded.Questions[0]
	if q0.ID != "q-migration" {
		t.Errorf("Question[0].ID = %q, want %q", q0.ID, "q-migration")
	}
	if q0.Answer != nil {
		t.Errorf("Question[0].Answer should be nil, got %v", q0.Answer)
	}
	if len(q0.Options) != 2 {
		t.Errorf("Question[0].Options len = %d, want 2", len(q0.Options))
	}

	// Verify critique
	if loaded.Critique == nil {
		t.Fatal("Critique should not be nil")
	}
	if loaded.Critique.ReviewedAt != "2026-04-05" {
		t.Errorf("Critique.ReviewedAt = %q, want %q", loaded.Critique.ReviewedAt, "2026-04-05")
	}
	if len(loaded.Critique.Assumptions) != 1 {
		t.Errorf("Critique.Assumptions len = %d, want 1", len(loaded.Critique.Assumptions))
	}
	if len(loaded.Critique.Critics) != 1 {
		t.Errorf("Critique.Critics len = %d, want 1", len(loaded.Critique.Critics))
	}
	if len(loaded.Critique.Risks) != 1 {
		t.Errorf("Critique.Risks len = %d, want 1", len(loaded.Critique.Risks))
	}
	if loaded.Critique.Synthesis != "The plan's structure is sound." {
		t.Errorf("Critique.Synthesis mismatch")
	}
}

func TestNilCritique_MarshalAsNull(t *testing.T) {
	p := NewPlan("plan-test1234", "Nil Critique Test", "desc")

	data, err := yaml.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	content := string(data)
	// With omitempty, the critique key should be absent entirely.
	if strings.Contains(content, "critique:") {
		t.Errorf("Expected critique key to be omitted (omitempty), but found it in:\n%s", content)
	}
}

func TestNilAnswer_MarshalAsNull(t *testing.T) {
	q := PlanQuestion{
		ID:   "q-test",
		Text: "Test question?",
		Options: []QuestionOption{
			{Key: "a", Label: "Option A"},
		},
		Answer: nil,
	}

	data, err := yaml.Marshal(q)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "answer: null") {
		t.Errorf("Expected 'answer: null' in output, got:\n%s", content)
	}
}

func TestAnsweredQuestion_RoundTrip(t *testing.T) {
	answer := "schema-version"
	q := PlanQuestion{
		ID:     "q-test",
		Text:   "Test?",
		Answer: &answer,
	}

	data, err := yaml.Marshal(q)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var loaded PlanQuestion
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if loaded.Answer == nil {
		t.Fatal("Answer should not be nil")
	}
	if *loaded.Answer != answer {
		t.Errorf("Answer = %q, want %q", *loaded.Answer, answer)
	}
}

func TestYAMLStructure_MatchesSampleSchema(t *testing.T) {
	// Verify that the YAML output has the expected top-level keys
	// in the order matching the sample_plan.yaml schema.
	p := &PlanYAML{
		Meta: PlanMeta{
			ID:        "plan-test",
			Title:     "Test",
			CreatedAt: "2026-04-05",
			Status:    "draft",
		},
		Design: PlanDesign{
			Problem: "test problem",
			Goals:   []string{"goal1"},
		},
		Slices: []PlanSlice{
			{
				ID:    "feat-test",
				Num:   1,
				Title: "First slice",
				Effort: "S",
				Risk:   "Low",
			},
		},
		Questions: []PlanQuestion{
			{
				ID:     "q-test",
				Text:   "Test?",
				Answer: nil,
			},
		},
		Critique: &PlanCritique{
			ReviewedAt: "2026-04-05",
			Reviewers:  []string{"test"},
			Synthesis:  "test synthesis",
		},
	}

	data, err := yaml.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	content := string(data)

	// Check top-level keys exist
	expectedKeys := []string{"meta:", "design:", "slices:", "questions:", "critique:"}
	for _, key := range expectedKeys {
		if !strings.Contains(content, key) {
			t.Errorf("Expected top-level key %q in YAML output:\n%s", key, content)
		}
	}

	// Check key ordering: meta before design before slices before questions before critique
	metaIdx := strings.Index(content, "meta:")
	designIdx := strings.Index(content, "design:")
	slicesIdx := strings.Index(content, "slices:")
	questionsIdx := strings.Index(content, "questions:")
	critiqueIdx := strings.Index(content, "critique:")

	if metaIdx >= designIdx {
		t.Error("meta should come before design")
	}
	if designIdx >= slicesIdx {
		t.Error("design should come before slices")
	}
	if slicesIdx >= questionsIdx {
		t.Error("slices should come before questions")
	}
	if questionsIdx >= critiqueIdx {
		t.Error("questions should come before critique")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/plan.yaml")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestSave_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plan-new.yaml")

	p := NewPlan("plan-new12345", "New Plan", "desc")
	if err := Save(path, p); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("File was not created: %v", err)
	}

	// Verify content is valid YAML
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	var loaded PlanYAML
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Output is not valid YAML: %v", err)
	}
	if loaded.Meta.ID != "plan-new12345" {
		t.Errorf("Loaded ID = %q, want %q", loaded.Meta.ID, "plan-new12345")
	}
}

func TestNewPlan_VersionDefault(t *testing.T) {
	p := NewPlan("plan-ver12345", "Version Test", "desc")
	if p.Meta.Version != 1 {
		t.Errorf("Meta.Version = %d, want 1", p.Meta.Version)
	}
}

func TestSave_IncrementsVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plan-ver.yaml")

	p := NewPlan("plan-ver12345", "Version Test", "desc")
	// Version starts at 1 from NewPlan

	if err := Save(path, p); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	// After first Save, version should be 2 (1 + 1 increment)
	if p.Meta.Version != 2 {
		t.Errorf("After first save: Meta.Version = %d, want 2", p.Meta.Version)
	}

	// Save again
	if err := Save(path, p); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if p.Meta.Version != 3 {
		t.Errorf("After second save: Meta.Version = %d, want 3", p.Meta.Version)
	}

	// Verify on disk
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.Meta.Version != 3 {
		t.Errorf("Loaded version = %d, want 3", loaded.Meta.Version)
	}
}

func TestVersion_InYAMLOutput(t *testing.T) {
	p := NewPlan("plan-ver12345", "Version Test", "desc")
	data, err := yaml.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	if !strings.Contains(string(data), "version: 1") {
		t.Errorf("Expected 'version: 1' in YAML output, got:\n%s", string(data))
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte("{{invalid yaml"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}

func TestEmptySlicesAndQuestions_Marshal(t *testing.T) {
	p := NewPlan("plan-empty1234", "Empty", "desc")

	data, err := yaml.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	content := string(data)
	// Empty slices and questions should marshal as empty arrays
	if !strings.Contains(content, "slices: []") {
		t.Errorf("Expected 'slices: []' in output, got:\n%s", content)
	}
	if !strings.Contains(content, "questions: []") {
		t.Errorf("Expected 'questions: []' in output, got:\n%s", content)
	}
}
