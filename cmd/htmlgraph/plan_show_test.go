package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const driftTestYAML = `meta:
    id: plan-drifttest
    title: Original Title
    description: test
    created_at: "2026-04-15"
    status: finalized
    version: 1
design:
    problem: p
    goals: []
    constraints: []
    approved: false
    comment: ""
slices:
    - id: slice-1
      num: 1
      title: s1
      what: w
      why: y
      files: []
      deps: []
      done_when: []
      effort: S
      risk: Low
      tests: ""
      approved: false
      comment: ""
    - id: slice-2
      num: 2
      title: s2
      what: w
      why: y
      files: []
      deps: []
      done_when: []
      effort: S
      risk: Low
      tests: ""
      approved: false
      comment: ""
questions: []
`

func writeDriftTestPair(t *testing.T, yamlBody, htmlBody string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "plan-drifttest.yaml")
	htmlPath := filepath.Join(dir, "plan-drifttest.html")
	if err := os.WriteFile(yamlPath, []byte(yamlBody), 0o644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}
	if err := os.WriteFile(htmlPath, []byte(htmlBody), 0o644); err != nil {
		t.Fatalf("write html: %v", err)
	}
	return yamlPath, htmlPath
}

func TestCheckPlanDrift_StatusMismatch(t *testing.T) {
	html := `<html><head><title>Plan: Original Title</title></head><body>
<article id="plan-drifttest" data-plan-id="plan-drifttest" data-status="draft">
<div data-slice="1"></div><div data-slice="2"></div>
</article></body></html>`
	yamlPath, htmlPath := writeDriftTestPair(t, driftTestYAML, html)

	var buf bytes.Buffer
	checkPlanDrift(yamlPath, htmlPath, &buf)

	out := buf.String()
	if !strings.Contains(out, "status: yaml=\"finalized\" html=\"draft\"") {
		t.Errorf("expected status drift warning, got:\n%s", out)
	}
	if strings.Contains(out, "title:") || strings.Contains(out, "slice count:") {
		t.Errorf("unexpected extra warnings:\n%s", out)
	}
}

func TestCheckPlanDrift_TitleMismatch(t *testing.T) {
	html := `<html><head><title>Plan: Stale Title</title></head><body>
<article id="plan-drifttest" data-status="finalized">
<div data-slice="1"></div><div data-slice="2"></div>
</article></body></html>`
	yamlPath, htmlPath := writeDriftTestPair(t, driftTestYAML, html)

	var buf bytes.Buffer
	checkPlanDrift(yamlPath, htmlPath, &buf)

	if !strings.Contains(buf.String(), `title: yaml="Original Title" html="Stale Title"`) {
		t.Errorf("expected title drift warning, got:\n%s", buf.String())
	}
}

func TestCheckPlanDrift_SliceCountMismatch(t *testing.T) {
	html := `<html><head><title>Plan: Original Title</title></head><body>
<article id="plan-drifttest" data-status="finalized">
<div data-slice="1"></div>
</article></body></html>`
	yamlPath, htmlPath := writeDriftTestPair(t, driftTestYAML, html)

	var buf bytes.Buffer
	checkPlanDrift(yamlPath, htmlPath, &buf)

	if !strings.Contains(buf.String(), "slice count: yaml=2 html=1") {
		t.Errorf("expected slice count drift warning, got:\n%s", buf.String())
	}
}

func TestCheckPlanDrift_NoDrift(t *testing.T) {
	html := `<html><head><title>Plan: Original Title</title></head><body>
<article id="plan-drifttest" data-status="finalized">
<div data-slice="1"></div><div data-slice="2"></div>
</article></body></html>`
	yamlPath, htmlPath := writeDriftTestPair(t, driftTestYAML, html)

	var buf bytes.Buffer
	checkPlanDrift(yamlPath, htmlPath, &buf)

	if buf.Len() != 0 {
		t.Errorf("expected no warnings, got:\n%s", buf.String())
	}
}

func TestCheckPlanDrift_MissingFilesSilent(t *testing.T) {
	var buf bytes.Buffer
	checkPlanDrift("/nonexistent/foo.yaml", "/nonexistent/foo.html", &buf)
	if buf.Len() != 0 {
		t.Errorf("expected silent on missing files, got:\n%s", buf.String())
	}
}
