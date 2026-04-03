package workowners

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParse_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "WORKOWNERS")
	os.WriteFile(path, []byte("cmd/**  trk-abc\ninternal/*.go  feat-xyz\n"), 0o644)

	wf, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if wf == nil {
		t.Fatal("expected non-nil file")
	}
	if len(wf.Rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(wf.Rules))
	}
	if wf.Rules[0].Pattern != "cmd/**" || wf.Rules[0].OwnerID != "trk-abc" {
		t.Errorf("rule 0: %+v", wf.Rules[0])
	}
}

func TestParse_MissingFile(t *testing.T) {
	wf, err := Parse("/nonexistent/WORKOWNERS")
	if err != nil {
		t.Fatalf("missing file should not error: %v", err)
	}
	if wf != nil {
		t.Error("expected nil for missing file")
	}
}

func TestParse_CommentsAndBlanks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "WORKOWNERS")
	os.WriteFile(path, []byte("# comment\n\ncmd/**  trk-abc\n# another comment\n"), 0o644)

	wf, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(wf.Rules) != 1 {
		t.Fatalf("expected 1 rule (skipping comments/blanks), got %d", len(wf.Rules))
	}
}

func TestResolve_DoubleStarPrefix(t *testing.T) {
	wf := &File{Rules: []Rule{
		{Pattern: "cmd/htmlgraph/**", OwnerID: "trk-cli"},
	}}
	if got := wf.Resolve("cmd/htmlgraph/main.go"); got != "trk-cli" {
		t.Errorf("expected trk-cli, got %q", got)
	}
	if got := wf.Resolve("cmd/htmlgraph/sub/file.go"); got != "trk-cli" {
		t.Errorf("expected trk-cli for nested path, got %q", got)
	}
	if got := wf.Resolve("internal/db/schema.go"); got != "" {
		t.Errorf("expected empty for non-matching path, got %q", got)
	}
}

func TestResolve_GlobPattern(t *testing.T) {
	wf := &File{Rules: []Rule{
		{Pattern: "*.md", OwnerID: "trk-docs"},
	}}
	if got := wf.Resolve("README.md"); got != "trk-docs" {
		t.Errorf("expected trk-docs, got %q", got)
	}
	if got := wf.Resolve("docs/guide.md"); got != "trk-docs" {
		t.Errorf("expected trk-docs for nested .md, got %q", got)
	}
}

func TestResolve_LastMatchWins(t *testing.T) {
	wf := &File{Rules: []Rule{
		{Pattern: "cmd/**", OwnerID: "trk-general"},
		{Pattern: "cmd/htmlgraph/**", OwnerID: "trk-specific"},
	}}
	if got := wf.Resolve("cmd/htmlgraph/main.go"); got != "trk-specific" {
		t.Errorf("expected trk-specific (last match), got %q", got)
	}
	if got := wf.Resolve("cmd/other/main.go"); got != "trk-general" {
		t.Errorf("expected trk-general for cmd/other, got %q", got)
	}
}

func TestResolve_NilFile(t *testing.T) {
	var wf *File
	if got := wf.Resolve("anything.go"); got != "" {
		t.Errorf("expected empty for nil file, got %q", got)
	}
}

func TestMatchPattern_ExactPath(t *testing.T) {
	if !matchPattern("cmd/main.go", "cmd/main.go") {
		t.Error("exact match should succeed")
	}
	if matchPattern("cmd/main.go", "cmd/other.go") {
		t.Error("different file should not match")
	}
}
