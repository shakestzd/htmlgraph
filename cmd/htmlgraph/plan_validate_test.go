package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidatePlan_ValidPlan(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "plans"), 0o755)

	planID, err := createPlanFromTopic(dir, "Valid Plan", "A valid plan")
	if err != nil {
		t.Fatal(err)
	}
	if err := addSliceToPlan(dir, planID, "Slice One"); err != nil {
		t.Fatal(err)
	}

	result, err := validatePlan(dir, planID)
	if err != nil {
		t.Fatalf("validatePlan: %v", err)
	}
	if !result.Valid {
		t.Errorf("plan should be valid, got errors: %v", result.Errors)
	}
	if result.Stats.Slices != 1 {
		t.Errorf("slices = %d, want 1", result.Stats.Slices)
	}
}

func TestValidatePlan_EmptyPlanWarns(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "plans"), 0o755)

	planID, err := createPlanFromTopic(dir, "Empty Plan", "")
	if err != nil {
		t.Fatal(err)
	}

	result, err := validatePlan(dir, planID)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Valid {
		t.Errorf("empty plan should be valid (warnings only), got: %v", result.Errors)
	}
	if len(result.Warnings) == 0 {
		t.Error("empty plan should have warnings about missing slices/description")
	}
}

func TestValidatePlan_NotFound(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "plans"), 0o755)

	_, err := validatePlan(dir, "plan-nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent plan")
	}
}
