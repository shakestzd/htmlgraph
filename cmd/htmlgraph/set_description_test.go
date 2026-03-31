package main

import (
	"strings"
	"testing"
)

func TestBuildDescription_PlainText(t *testing.T) {
	result := buildDescription("hello world", "", "", "")
	if result != "hello world" {
		t.Errorf("expected plain text, got: %s", result)
	}
}

func TestBuildDescription_WithAcceptance(t *testing.T) {
	result := buildDescription("main text", "must pass tests", "", "")
	if !strings.Contains(result, "Acceptance Criteria") {
		t.Error("expected Acceptance Criteria header")
	}
	if !strings.Contains(result, "must pass tests") {
		t.Error("expected acceptance content")
	}
}

func TestBuildDescription_WithTestStrategy(t *testing.T) {
	result := buildDescription("main text", "", "unit and integration tests", "")
	if !strings.Contains(result, "Test Strategy") {
		t.Error("expected Test Strategy header")
	}
	if !strings.Contains(result, "unit and integration tests") {
		t.Error("expected test strategy content")
	}
}

func TestBuildDescription_WithExpectedBehavior(t *testing.T) {
	result := buildDescription("main text", "", "", "should return true when valid")
	if !strings.Contains(result, "Expected Behavior") {
		t.Error("expected Expected Behavior header")
	}
	if !strings.Contains(result, "should return true when valid") {
		t.Error("expected behavior content")
	}
}

func TestBuildDescription_AllSections(t *testing.T) {
	result := buildDescription("desc", "accept", "strategy", "behavior")
	for _, section := range []string{"Acceptance Criteria", "Test Strategy", "Expected Behavior"} {
		if !strings.Contains(result, section) {
			t.Errorf("expected section: %s", section)
		}
	}
	if !strings.Contains(result, "desc") {
		t.Error("expected main description content")
	}
	if !strings.Contains(result, "accept") {
		t.Error("expected acceptance content")
	}
	if !strings.Contains(result, "strategy") {
		t.Error("expected strategy content")
	}
	if !strings.Contains(result, "behavior") {
		t.Error("expected behavior content")
	}
}

func TestBuildDescription_EmptyStrings(t *testing.T) {
	result := buildDescription("", "", "", "")
	if result != "" {
		t.Errorf("expected empty string, got: %s", result)
	}
}

func TestBuildDescription_OnlyAcceptance(t *testing.T) {
	result := buildDescription("", "accept only", "", "")
	if !strings.Contains(result, "Acceptance Criteria") {
		t.Error("expected Acceptance Criteria header")
	}
	if !strings.Contains(result, "accept only") {
		t.Error("expected acceptance content")
	}
}
