package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestExecutePreview_BuildsEnvelope is a unit test for buildExecutePreview that
// sets up a minimal .htmlgraph/ tree and verifies the JSON envelope contains the
// track, linked bugs, and git state fields.
func TestExecutePreview_BuildsEnvelope(t *testing.T) {
	dir := t.TempDir()
	hgDir := filepath.Join(dir, ".htmlgraph")

	// Create directory skeleton.
	for _, sub := range []string{"tracks", "features", "bugs", "plans", "spikes"} {
		if err := os.MkdirAll(filepath.Join(hgDir, sub), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", sub, err)
		}
	}

	// Track with two edges — one to a bug, one to a feature.
	trackHTML := `<!DOCTYPE html><html><body>
<article id="trk-test001" data-type="track" data-status="in-progress" data-priority="medium">
<header><h1>Sample Track</h1></header>
<nav data-graph-edges>
  <section data-edge-type="contains">
    <ul>
      <li><a href="bug-test001.html" data-relationship="contains">Sample Bug</a></li>
      <li><a href="feat-test001.html" data-relationship="contains">Sample Feature</a></li>
    </ul>
  </section>
</nav>
</article>
</body></html>`
	writeFile(t, filepath.Join(hgDir, "tracks", "trk-test001.html"), trackHTML)

	bugHTML := `<!DOCTYPE html><html><body>
<article id="bug-test001" data-type="bug" data-status="todo" data-priority="medium">
<header><h1>Sample Bug</h1></header>
</article>
</body></html>`
	writeFile(t, filepath.Join(hgDir, "bugs", "bug-test001.html"), bugHTML)

	featHTML := `<!DOCTYPE html><html><body>
<article id="feat-test001" data-type="feature" data-status="done" data-priority="medium">
<header><h1>Sample Feature</h1></header>
</article>
</body></html>`
	writeFile(t, filepath.Join(hgDir, "features", "feat-test001.html"), featHTML)

	preview, err := buildExecutePreview(hgDir, "trk-test001")
	if err != nil {
		t.Fatalf("buildExecutePreview: %v", err)
	}

	if preview.Track == nil {
		t.Fatal("Track is nil")
	}
	if preview.Track.ID != "trk-test001" {
		t.Errorf("Track.ID = %q, want trk-test001", preview.Track.ID)
	}
	if got := len(preview.Bugs); got != 1 {
		t.Errorf("len(Bugs) = %d, want 1", got)
	}
	if got := len(preview.Features); got != 1 {
		t.Errorf("len(Features) = %d, want 1", got)
	}

	// Marshal to JSON to prove the envelope serializes cleanly — mirrors what
	// the --format json path does.
	b, err := json.Marshal(preview)
	if err != nil {
		t.Fatalf("marshal preview: %v", err)
	}
	s := string(b)
	for _, key := range []string{`"track"`, `"bugs"`, `"features"`, `"git"`} {
		if !strings.Contains(s, key) {
			t.Errorf("json envelope missing %s", key)
		}
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile %s: %v", path, err)
	}
}
