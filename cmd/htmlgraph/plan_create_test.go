package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestRunPlanCreateFromTopic(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"plans"} {
		os.MkdirAll(filepath.Join(dir, sub), 0o755)
	}

	planID, err := createPlanFromTopic(dir, "Auth Middleware Rewrite", "Rewrite auth for compliance")
	if err != nil {
		t.Fatalf("createPlanFromTopic: %v", err)
	}

	// Verify hex8 format.
	matched, _ := regexp.MatchString(`^plan-[0-9a-f]{8}$`, planID)
	if !matched {
		t.Errorf("plan ID %q does not match hex8 format", planID)
	}

	// Verify file exists.
	planPath := filepath.Join(dir, "plans", planID+".html")
	data, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatalf("plan file not found: %v", err)
	}
	html := string(data)

	// Verify title is present.
	if !strings.Contains(html, "Auth Middleware Rewrite") {
		t.Error("plan HTML missing title")
	}

	// Verify description is present.
	if !strings.Contains(html, "Rewrite auth for compliance") {
		t.Error("plan HTML missing description")
	}

	// Verify it uses the standard node template (links to styles.css).
	if !strings.Contains(html, `href="../styles.css"`) {
		t.Error("plan HTML should use styles.css (standard node template)")
	}

	// Verify it does NOT use the old CRISPI template.
	if strings.Contains(html, "btn-finalize") {
		t.Error("plan HTML should NOT contain CRISPI btn-finalize")
	}
	if strings.Contains(html, "plan-sidebar") {
		t.Error("plan HTML should NOT contain CRISPI sidebar")
	}
}

func TestRunPlanAddSlice(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "plans"), 0o755)

	planID, err := createPlanFromTopic(dir, "Test Plan", "A test plan")
	if err != nil {
		t.Fatalf("createPlanFromTopic: %v", err)
	}

	// Add a slice.
	if err := addSliceToPlan(dir, planID, "Implement error handling"); err != nil {
		t.Fatalf("addSliceToPlan: %v", err)
	}

	// Verify slice exists as a step in the plan HTML.
	planPath := filepath.Join(dir, "plans", planID+".html")
	data, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatal(err)
	}
	html := string(data)

	if !strings.Contains(html, "Implement error handling") {
		t.Error("plan HTML missing slice title as step")
	}

	// Add a second slice.
	if err := addSliceToPlan(dir, planID, "Add tests"); err != nil {
		t.Fatalf("addSliceToPlan second: %v", err)
	}

	data, _ = os.ReadFile(planPath)
	html = string(data)
	if !strings.Contains(html, "Add tests") {
		t.Error("plan HTML missing second slice")
	}
}
