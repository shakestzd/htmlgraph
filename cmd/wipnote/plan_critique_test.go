package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCritique_ComplexityGateLow(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "plans"), 0o755)

	planID, err := createPlanFromTopic(dir, "Small Plan", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := runPlanAddSliceYAML(dir, planID, "Single slice",
		"Do one thing", "", "", "", "", "S", "Low", ""); err != nil {
		t.Fatal(err)
	}

	out, err := extractCritiqueData(dir, planID)
	if err != nil {
		t.Fatalf("extractCritiqueData: %v", err)
	}
	if out.CritiqueWarranted {
		t.Error("expected critique_warranted=false for 1 slice")
	}
	if out.Complexity != "low" {
		t.Errorf("complexity = %q, want low", out.Complexity)
	}
}

func TestCritique_ComplexityGateMedium(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "plans"), 0o755)

	planID, err := createPlanFromTopic(dir, "Medium Plan", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"S1", "S2", "S3", "S4"} {
		if err := runPlanAddSliceYAML(dir, planID, s,
			"Do "+s, "", "", "", "", "S", "Low", ""); err != nil {
			t.Fatal(err)
		}
	}

	out, err := extractCritiqueData(dir, planID)
	if err != nil {
		t.Fatal(err)
	}
	if !out.CritiqueWarranted {
		t.Error("expected critique_warranted=true for 4 slices")
	}
	if out.Complexity != "medium" {
		t.Errorf("complexity = %q, want medium", out.Complexity)
	}
}

func TestCritique_TitleExtraction(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "plans"), 0o755)

	planID, err := createPlanFromTopic(dir, "Auth Rewrite", "compliance driven")
	if err != nil {
		t.Fatal(err)
	}

	out, err := extractCritiqueData(dir, planID)
	if err != nil {
		t.Fatal(err)
	}
	if out.Title != "Auth Rewrite" {
		t.Errorf("title = %q, want Auth Rewrite", out.Title)
	}
	if out.Description != "compliance driven" {
		t.Errorf("description = %q, want 'compliance driven'", out.Description)
	}
}

func TestClassifyComplexity(t *testing.T) {
	tests := []struct {
		count      int
		complexity string
		warranted  bool
	}{
		{0, "low", false},
		{1, "low", false},
		{2, "low", false},
		{3, "medium", true},
		{5, "medium", true},
		{6, "high", true},
		{10, "high", true},
	}

	for _, tc := range tests {
		c, w := classifyComplexity(tc.count)
		if c != tc.complexity || w != tc.warranted {
			t.Errorf("classifyComplexity(%d) = (%q, %v), want (%q, %v)",
				tc.count, c, w, tc.complexity, tc.warranted)
		}
	}
}
