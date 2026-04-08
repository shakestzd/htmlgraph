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

	// Verify it uses the CRISPI interactive template.
	if !strings.Contains(html, "btn-finalize") {
		t.Error("plan HTML should contain CRISPI btn-finalize")
	}
	if !strings.Contains(html, "dep-graph-svg") {
		t.Error("plan HTML should contain dep-graph-svg")
	}
	if !strings.Contains(html, "PLAN_SECTIONS_JSON") {
		t.Error("plan HTML should contain PLAN_SECTIONS_JSON")
	}

	// Verify SECTIONS_JSON starts with just design (outline hidden when empty).
	if !strings.Contains(html, `["design"]`) {
		t.Error("CRISPI plan should start with [design] section")
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
	if err := addSliceToPlan(dir, planID, "Implement error handling", sliceFlags{}); err != nil {
		t.Fatalf("addSliceToPlan: %v", err)
	}

	// Verify slice exists in the CRISPI HTML.
	planPath := filepath.Join(dir, "plans", planID+".html")
	data, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatal(err)
	}
	html := string(data)

	if !strings.Contains(html, "Implement error handling") {
		t.Error("plan HTML missing slice title")
	}

	// Verify CRISPI-specific elements were injected.
	if !strings.Contains(html, `data-node="1"`) {
		t.Error("plan HTML missing graph node data-node=1")
	}
	if !strings.Contains(html, `data-slice="1"`) {
		t.Error("plan HTML missing slice card data-slice=1")
	}
	if !strings.Contains(html, `"slice-1"`) {
		t.Error("plan HTML SECTIONS_JSON missing slice-1")
	}

	// Add a second slice.
	if err := addSliceToPlan(dir, planID, "Add tests", sliceFlags{}); err != nil {
		t.Fatalf("addSliceToPlan second: %v", err)
	}

	data, _ = os.ReadFile(planPath)
	html = string(data)
	if !strings.Contains(html, "Add tests") {
		t.Error("plan HTML missing second slice")
	}
	if !strings.Contains(html, `data-node="2"`) {
		t.Error("plan HTML missing graph node data-node=2")
	}
	if !strings.Contains(html, `data-slice="2"`) {
		t.Error("plan HTML missing slice card data-slice=2")
	}
	if !strings.Contains(html, `"slice-2"`) {
		t.Error("plan HTML SECTIONS_JSON missing slice-2")
	}
}
