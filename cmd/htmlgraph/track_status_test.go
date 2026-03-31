package main

import (
	"testing"
)

func TestGroupByPrefix_ParsesFeatureIDs(t *testing.T) {
	lines := []string{
		"abc1234 feat(yolo): add track worktree (feat-78081efd)",
		"def5678 feat(cli): add set-description command (feat-eace1f9d)",
		"ghi9012 fix: random bugfix",
	}
	groups := groupByPrefix(lines)

	// Should have 3 groups: feat-78081efd, feat-eace1f9d, unattributed
	featCount := 0
	unattributed := 0
	for _, g := range groups {
		if g.FeatureID == "" {
			unattributed++
		} else {
			featCount++
		}
	}
	if featCount != 2 {
		t.Errorf("expected 2 feature groups, got %d", featCount)
	}
	if unattributed != 1 {
		t.Errorf("expected 1 unattributed group, got %d", unattributed)
	}
}

func TestGroupByPrefix_HandlesNoPrefix(t *testing.T) {
	lines := []string{
		"abc1234 just a regular commit",
		"def5678 another regular commit",
	}
	groups := groupByPrefix(lines)
	if len(groups) != 1 {
		t.Errorf("expected 1 group (unattributed), got %d", len(groups))
	}
	if groups[0].FeatureID != "" {
		t.Errorf("expected empty feature ID for unattributed")
	}
	if len(groups[0].Commits) != 2 {
		t.Errorf("expected 2 commits, got %d", len(groups[0].Commits))
	}
}

func TestGroupByPrefix_HandlesColonPrefix(t *testing.T) {
	lines := []string{
		"abc1234 feat-12345678: some change",
	}
	groups := groupByPrefix(lines)
	if len(groups) != 1 || groups[0].FeatureID != "feat-12345678" {
		t.Errorf("expected feat-12345678, got groups: %+v", groups)
	}
}

func TestGroupByPrefix_EmptyInput(t *testing.T) {
	groups := groupByPrefix(nil)
	if len(groups) != 0 {
		t.Errorf("expected 0 groups for nil input, got %d", len(groups))
	}
}

func TestGroupByPrefix_SortsAlphabetically(t *testing.T) {
	lines := []string{
		"z1 feat-ffffffff: commit",
		"a1 feat-11111111: commit",
		"m1 feat-88888888: commit",
		"u1 unattributed",
	}
	groups := groupByPrefix(lines)

	// Should be: feat-11111111, feat-88888888, feat-ffffffff, unattributed
	expectedOrder := []string{"feat-11111111", "feat-88888888", "feat-ffffffff", ""}
	if len(groups) != 4 {
		t.Errorf("expected 4 groups, got %d", len(groups))
		return
	}

	for i, expected := range expectedOrder {
		if groups[i].FeatureID != expected {
			t.Errorf("expected group %d to be %q, got %q", i, expected, groups[i].FeatureID)
		}
	}
}

func TestGroupByPrefix_ParsesCommitHash(t *testing.T) {
	lines := []string{
		"abc1234def5678 feat-xxx: some change",
	}
	groups := groupByPrefix(lines)

	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}

	if len(groups[0].Commits) != 1 {
		t.Fatalf("expected 1 commit, got %d", len(groups[0].Commits))
	}

	commit := groups[0].Commits[0]
	if commit.Hash != "abc1234def5678" {
		t.Errorf("expected hash abc1234def5678, got %s", commit.Hash)
	}
	if commit.Subject != "feat-xxx: some change" {
		t.Errorf("expected subject 'feat-xxx: some change', got %s", commit.Subject)
	}
}

func TestGroupByPrefix_HandlesParenthesesPrefix(t *testing.T) {
	lines := []string{
		"abc1234 (feat-12345abc) some commit message",
	}
	groups := groupByPrefix(lines)

	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}

	if groups[0].FeatureID != "feat-12345abc" {
		t.Errorf("expected feat-12345abc, got %q", groups[0].FeatureID)
	}
}
