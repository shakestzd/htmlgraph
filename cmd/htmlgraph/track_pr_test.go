package main

import (
	"strings"
	"testing"
)

func TestBuildPRBody_IncludesAllFeatures(t *testing.T) {
	groups := []featureGroup{
		{FeatureID: "feat-abc123", Commits: []commitInfo{
			{Hash: "aaa", Subject: "feat: something (feat-abc123)"},
			{Hash: "bbb", Subject: "fix: related (feat-abc123)"},
		}},
		{FeatureID: "feat-def456", Commits: []commitInfo{
			{Hash: "ccc", Subject: "feat: other (feat-def456)"},
		}},
	}
	body := buildPRBody("trk-test123", groups, "")

	if !strings.Contains(body, "feat-abc123") {
		t.Error("expected feat-abc123 in body")
	}
	if !strings.Contains(body, "feat-def456") {
		t.Error("expected feat-def456 in body")
	}
	if !strings.Contains(body, "2 commits") {
		t.Error("expected '2 commits' in body")
	}
	if !strings.Contains(body, "1 commits") {
		t.Error("expected '1 commits' in body")
	}
	if !strings.Contains(body, "trk-test123") {
		t.Error("expected track ID in body")
	}
}

func TestBuildPRBody_IncludesDiffStat(t *testing.T) {
	groups := []featureGroup{
		{FeatureID: "feat-abc", Commits: []commitInfo{{Hash: "aaa", Subject: "test"}}},
	}
	diffStat := " 3 files changed, 100 insertions(+), 20 deletions(-)"
	body := buildPRBody("trk-test", groups, diffStat)

	if !strings.Contains(body, "100 insertions") {
		t.Error("expected diff stat in body")
	}
}

func TestBuildPRBody_HandlesUnattributed(t *testing.T) {
	groups := []featureGroup{
		{FeatureID: "", Commits: []commitInfo{{Hash: "aaa", Subject: "misc fix"}}},
	}
	body := buildPRBody("trk-test", groups, "")

	if !strings.Contains(body, "unattributed") {
		t.Error("expected unattributed label")
	}
}

func TestBuildPRBody_EmptyDiffStat(t *testing.T) {
	groups := []featureGroup{
		{FeatureID: "feat-abc", Commits: []commitInfo{{Hash: "aaa", Subject: "test"}}},
	}
	body := buildPRBody("trk-test", groups, "")

	// Should not have code block markers for empty diff stat
	if strings.Count(body, "```") > 0 {
		t.Error("expected no code blocks for empty diff stat")
	}
}
