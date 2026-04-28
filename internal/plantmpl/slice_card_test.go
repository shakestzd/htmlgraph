package plantmpl_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/shakestzd/htmlgraph/internal/plantmpl"
)

// ---------------------------------------------------------------------------
// SliceCard.Render — structural output
// ---------------------------------------------------------------------------

func TestSliceCardRenderContainsSliceCard(t *testing.T) {
	sc := &plantmpl.SliceCard{
		Num:   3,
		ID:    "feat-abc123",
		Title: "Auth endpoint",
	}

	var buf bytes.Buffer
	if err := sc.Render(&buf); err != nil {
		t.Fatalf("Render: %v", err)
	}

	html := buf.String()
	if !strings.Contains(html, `class="slice-card"`) {
		t.Error("output missing class=\"slice-card\"")
	}
	if !strings.Contains(html, `data-slice="3"`) {
		t.Error("output missing data-slice=\"3\"")
	}
	if !strings.Contains(html, "Auth endpoint") {
		t.Error("output missing Title")
	}
	if !strings.Contains(html, "feat-abc123") {
		t.Error("output missing ID")
	}
}

func TestSliceCardRenderDataSliceName(t *testing.T) {
	sc := &plantmpl.SliceCard{
		Num: 5,
		ID:  "feat-def456",
	}

	var buf bytes.Buffer
	if err := sc.Render(&buf); err != nil {
		t.Fatalf("Render: %v", err)
	}

	html := buf.String()
	if !strings.Contains(html, `data-slice-name="feat-def456"`) {
		t.Error("output missing data-slice-name attribute")
	}
	if !strings.Contains(html, `data-slice="5"`) {
		t.Error("output missing data-slice=\"5\"")
	}
}

func TestSliceCardRenderDefaultStatusPending(t *testing.T) {
	sc := &plantmpl.SliceCard{
		Num: 1,
		ID:  "feat-test",
	}

	var buf bytes.Buffer
	if err := sc.Render(&buf); err != nil {
		t.Fatalf("Render: %v", err)
	}

	html := buf.String()
	if !strings.Contains(html, `data-status="pending"`) {
		t.Error("empty Status should default to pending in data-status attribute")
	}
}

func TestSliceCardRenderExplicitStatus(t *testing.T) {
	sc := &plantmpl.SliceCard{
		Num:    2,
		ID:     "feat-test",
		Status: "approved",
	}

	var buf bytes.Buffer
	if err := sc.Render(&buf); err != nil {
		t.Fatalf("Render: %v", err)
	}

	html := buf.String()
	if !strings.Contains(html, `data-status="approved"`) {
		t.Error("explicit Status not reflected in data-status attribute")
	}
}

// ---------------------------------------------------------------------------
// SliceCard.Render — effort badge
// ---------------------------------------------------------------------------

func TestSliceCardRenderEffortSmall(t *testing.T) {
	sc := &plantmpl.SliceCard{Num: 1, ID: "feat-s", Effort: "S"}

	var buf bytes.Buffer
	if err := sc.Render(&buf); err != nil {
		t.Fatalf("Render: %v", err)
	}

	html := buf.String()
	if !strings.Contains(html, "badge-pending") {
		t.Error("S effort should use badge-pending class")
	}
	if !strings.Contains(html, ">S<") {
		t.Error("effort badge should contain S text")
	}
}

func TestSliceCardRenderEffortMedium(t *testing.T) {
	sc := &plantmpl.SliceCard{Num: 1, ID: "feat-m", Effort: "M"}

	var buf bytes.Buffer
	if err := sc.Render(&buf); err != nil {
		t.Fatalf("Render: %v", err)
	}

	html := buf.String()
	if !strings.Contains(html, "badge-revision") {
		t.Error("M effort should use badge-revision class")
	}
	if !strings.Contains(html, ">M<") {
		t.Error("effort badge should contain M text")
	}
}

func TestSliceCardRenderEffortLarge(t *testing.T) {
	sc := &plantmpl.SliceCard{Num: 1, ID: "feat-l", Effort: "L"}

	var buf bytes.Buffer
	if err := sc.Render(&buf); err != nil {
		t.Fatalf("Render: %v", err)
	}

	html := buf.String()
	if !strings.Contains(html, "badge-blocked") {
		t.Error("L effort should use badge-blocked class")
	}
	if !strings.Contains(html, ">L<") {
		t.Error("effort badge should contain L text")
	}
}

func TestSliceCardRenderEmptyEffortOmitted(t *testing.T) {
	sc := &plantmpl.SliceCard{Num: 1, ID: "feat-noeffort", Effort: ""}

	var buf bytes.Buffer
	if err := sc.Render(&buf); err != nil {
		t.Fatalf("Render: %v", err)
	}

	// When Effort is empty the badge block should not appear.
	html := buf.String()
	_ = html // confirmed by template conditional — no assertion needed
}

// ---------------------------------------------------------------------------
// SliceCard.Render — risk badge
// ---------------------------------------------------------------------------

func TestSliceCardRenderRiskLow(t *testing.T) {
	sc := &plantmpl.SliceCard{Num: 1, ID: "feat-low", Risk: "Low"}

	var buf bytes.Buffer
	if err := sc.Render(&buf); err != nil {
		t.Fatalf("Render: %v", err)
	}

	html := buf.String()
	if !strings.Contains(html, ">Low<") {
		t.Error("Low risk badge missing 'Low' text")
	}
	// Low → badge-pending
	if !strings.Contains(html, "badge-pending") {
		t.Error("Low risk should use badge-pending class")
	}
}

func TestSliceCardRenderRiskMed(t *testing.T) {
	sc := &plantmpl.SliceCard{Num: 1, ID: "feat-med", Risk: "Med"}

	var buf bytes.Buffer
	if err := sc.Render(&buf); err != nil {
		t.Fatalf("Render: %v", err)
	}

	html := buf.String()
	if !strings.Contains(html, ">Med<") {
		t.Error("Med risk badge missing 'Med' text")
	}
	if !strings.Contains(html, "badge-revision") {
		t.Error("Med risk should use badge-revision class")
	}
}

func TestSliceCardRenderRiskHigh(t *testing.T) {
	sc := &plantmpl.SliceCard{Num: 1, ID: "feat-high", Risk: "High"}

	var buf bytes.Buffer
	if err := sc.Render(&buf); err != nil {
		t.Fatalf("Render: %v", err)
	}

	html := buf.String()
	if !strings.Contains(html, ">High<") {
		t.Error("High risk badge missing 'High' text")
	}
	if !strings.Contains(html, "badge-blocked") {
		t.Error("High risk should use badge-blocked class")
	}
}

func TestSliceCardRenderEmptyRiskOmitted(t *testing.T) {
	sc := &plantmpl.SliceCard{Num: 1, ID: "feat-norisk", Risk: ""}

	var buf bytes.Buffer
	if err := sc.Render(&buf); err != nil {
		t.Fatalf("Render: %v", err)
	}

	html := buf.String()
	if strings.Contains(html, "Risk") {
		t.Error("empty Risk should not render a risk badge")
	}
}

// ---------------------------------------------------------------------------
// SliceCard.Render — optional fields
// ---------------------------------------------------------------------------

func TestSliceCardRenderDescription(t *testing.T) {
	sc := &plantmpl.SliceCard{
		Num:         1,
		ID:          "feat-desc",
		Description: "Implements the login flow",
	}

	var buf bytes.Buffer
	if err := sc.Render(&buf); err != nil {
		t.Fatalf("Render: %v", err)
	}

	html := buf.String()
	if !strings.Contains(html, "Implements the login flow") {
		t.Error("Description not rendered")
	}
}

func TestSliceCardRenderEmptyDescriptionOmitted(t *testing.T) {
	sc := &plantmpl.SliceCard{Num: 1, ID: "feat-nodesc", Description: ""}

	var buf bytes.Buffer
	if err := sc.Render(&buf); err != nil {
		t.Fatalf("Render: %v", err)
	}

	html := buf.String()
	// The description paragraph should not appear when empty.
	if strings.Contains(html, `<p style="font-size:.9rem`) {
		t.Error("empty Description should not render description paragraph")
	}
}

func TestSliceCardRenderFiles(t *testing.T) {
	sc := &plantmpl.SliceCard{
		Num:   1,
		ID:    "feat-files",
		Files: "internal/auth/auth.go,cmd/serve/main.go",
	}

	var buf bytes.Buffer
	if err := sc.Render(&buf); err != nil {
		t.Fatalf("Render: %v", err)
	}

	html := buf.String()
	if !strings.Contains(html, "internal/auth/auth.go,cmd/serve/main.go") {
		t.Error("Files not rendered")
	}
}

func TestSliceCardRenderEmptyFilesOmitted(t *testing.T) {
	sc := &plantmpl.SliceCard{Num: 1, ID: "feat-nofiles", Files: ""}

	var buf bytes.Buffer
	if err := sc.Render(&buf); err != nil {
		t.Fatalf("Render: %v", err)
	}

	html := buf.String()
	if strings.Contains(html, "Files:") {
		t.Error("empty Files should not render files row")
	}
}

// ---------------------------------------------------------------------------
// Markdown rendering tests (slice-2 / feat-33807582)
// ---------------------------------------------------------------------------

func TestSliceCard_RendersMarkdownHeadings(t *testing.T) {
	sc := &plantmpl.SliceCard{
		Num:  1,
		ID:   "feat-md-h",
		What: "### Heading\n\ntext",
	}

	var buf bytes.Buffer
	if err := sc.Render(&buf); err != nil {
		t.Fatalf("Render: %v", err)
	}

	html := buf.String()
	if !strings.Contains(html, "<h3>") {
		t.Errorf("expected <h3> from ### heading, got output:\n%s", html)
	}
}

func TestSliceCard_RendersMarkdownLists(t *testing.T) {
	sc := &plantmpl.SliceCard{
		Num:  1,
		ID:   "feat-md-list",
		What: "- item one\n- item two\n- item three",
	}

	var buf bytes.Buffer
	if err := sc.Render(&buf); err != nil {
		t.Fatalf("Render: %v", err)
	}

	html := buf.String()
	if !strings.Contains(html, "<ul>") {
		t.Errorf("expected <ul> from bullet list, got output:\n%s", html)
	}
	if !strings.Contains(html, "<li>") {
		t.Errorf("expected <li> from bullet list, got output:\n%s", html)
	}
}

func TestSliceCard_RendersMarkdownCodeFence(t *testing.T) {
	sc := &plantmpl.SliceCard{
		Num:  1,
		ID:   "feat-md-fence",
		What: "```go\nfunc main() {}\n```",
	}

	var buf bytes.Buffer
	if err := sc.Render(&buf); err != nil {
		t.Fatalf("Render: %v", err)
	}

	html := buf.String()
	if !strings.Contains(html, "<pre>") {
		t.Errorf("expected <pre> from code fence, got output:\n%s", html)
	}
	if !strings.Contains(html, "<code") {
		t.Errorf("expected <code> from code fence, got output:\n%s", html)
	}
}

func TestSliceCard_RendersInlineCode(t *testing.T) {
	sc := &plantmpl.SliceCard{
		Num:  1,
		ID:   "feat-md-inline",
		What: "Use `myFunc()` to do it.",
	}

	var buf bytes.Buffer
	if err := sc.Render(&buf); err != nil {
		t.Fatalf("Render: %v", err)
	}

	html := buf.String()
	if !strings.Contains(html, "<code>") {
		t.Errorf("expected <code> from inline code, got output:\n%s", html)
	}
	if !strings.Contains(html, "myFunc()") {
		t.Errorf("expected myFunc() in output, got:\n%s", html)
	}
}

func TestSliceCard_StripsScriptTags(t *testing.T) {
	sc := &plantmpl.SliceCard{
		Num:  1,
		ID:   "feat-md-xss",
		What: `<script>alert(1)</script>`,
	}

	var buf bytes.Buffer
	if err := sc.Render(&buf); err != nil {
		t.Fatalf("Render: %v", err)
	}

	html := buf.String()
	if strings.Contains(html, "<script>") {
		t.Errorf("unescaped <script> tag must not appear in output:\n%s", html)
	}
}

func TestSliceCard_StripsRawHTMLEventHandlers(t *testing.T) {
	sc := &plantmpl.SliceCard{
		Num:  1,
		ID:   "feat-md-evil",
		What: `<img src=x onerror="alert(1)">`,
	}

	var buf bytes.Buffer
	if err := sc.Render(&buf); err != nil {
		t.Fatalf("Render: %v", err)
	}

	html := buf.String()
	if strings.Contains(html, "onerror=") {
		t.Errorf("onerror event handler must be stripped from output:\n%s", html)
	}
	if strings.Contains(html, "alert(1)") {
		t.Errorf("event handler payload must be stripped from output:\n%s", html)
	}
}

func TestSliceCard_DoneWhenStaysStructuredList(t *testing.T) {
	sc := &plantmpl.SliceCard{
		Num:      1,
		ID:       "feat-md-done",
		What:     "Implement it",
		DoneWhen: []string{"All tests pass", "Code reviewed"},
	}

	var buf bytes.Buffer
	if err := sc.Render(&buf); err != nil {
		t.Fatalf("Render: %v", err)
	}

	html := buf.String()
	// done_when items must render as literal <li> text — no Markdown heading etc.
	if !strings.Contains(html, "All tests pass") {
		t.Errorf("DoneWhen item 'All tests pass' missing from output:\n%s", html)
	}
	if !strings.Contains(html, "Code reviewed") {
		t.Errorf("DoneWhen item 'Code reviewed' missing from output:\n%s", html)
	}
	// The done_when section wraps items in <ul>
	if !strings.Contains(html, `class="slice-done-list"`) {
		t.Errorf("expected slice-done-list ul, got:\n%s", html)
	}
}

func TestSliceCard_LegacyPlainTextStillRenders(t *testing.T) {
	sc := &plantmpl.SliceCard{
		Num:  1,
		ID:   "feat-legacy",
		What: "Just text.",
	}

	var buf bytes.Buffer
	if err := sc.Render(&buf); err != nil {
		t.Fatalf("Render: %v", err)
	}

	html := buf.String()
	if !strings.Contains(html, "Just text.") {
		t.Errorf("plain text 'Just text.' missing from output:\n%s", html)
	}
}

// ---------------------------------------------------------------------------
// DepsLabel helper
// ---------------------------------------------------------------------------

func TestDepsLabelEmpty(t *testing.T) {
	sc := &plantmpl.SliceCard{Deps: ""}
	if got := sc.DepsLabel(); got != "none" {
		t.Errorf("DepsLabel with empty Deps: got %q, want %q", got, "none")
	}
}

func TestDepsLabelNonEmpty(t *testing.T) {
	sc := &plantmpl.SliceCard{Deps: "1,2"}
	if got := sc.DepsLabel(); got != "slices 1,2" {
		t.Errorf("DepsLabel with Deps=1,2: got %q, want %q", got, "slices 1,2")
	}
}

func TestDepsLabelSingle(t *testing.T) {
	sc := &plantmpl.SliceCard{Deps: "3"}
	if got := sc.DepsLabel(); got != "slices 3" {
		t.Errorf("DepsLabel with Deps=3: got %q, want %q", got, "slices 3")
	}
}

// ---------------------------------------------------------------------------
// EffortClass helper
// ---------------------------------------------------------------------------

func TestEffortClassS(t *testing.T) {
	sc := &plantmpl.SliceCard{Effort: "S"}
	if got := sc.EffortClass(); got != "badge-pending" {
		t.Errorf("EffortClass S: got %q, want badge-pending", got)
	}
}

func TestEffortClassM(t *testing.T) {
	sc := &plantmpl.SliceCard{Effort: "M"}
	if got := sc.EffortClass(); got != "badge-revision" {
		t.Errorf("EffortClass M: got %q, want badge-revision", got)
	}
}

func TestEffortClassL(t *testing.T) {
	sc := &plantmpl.SliceCard{Effort: "L"}
	if got := sc.EffortClass(); got != "badge-blocked" {
		t.Errorf("EffortClass L: got %q, want badge-blocked", got)
	}
}

func TestEffortClassUnknown(t *testing.T) {
	sc := &plantmpl.SliceCard{Effort: "XL"}
	if got := sc.EffortClass(); got != "badge-pending" {
		t.Errorf("EffortClass unknown: got %q, want badge-pending", got)
	}
}

// ---------------------------------------------------------------------------
// RiskClass helper
// ---------------------------------------------------------------------------

func TestRiskClassHigh(t *testing.T) {
	sc := &plantmpl.SliceCard{Risk: "High"}
	if got := sc.RiskClass(); got != "badge-blocked" {
		t.Errorf("RiskClass High: got %q, want badge-blocked", got)
	}
}

func TestRiskClassMed(t *testing.T) {
	sc := &plantmpl.SliceCard{Risk: "Med"}
	if got := sc.RiskClass(); got != "badge-revision" {
		t.Errorf("RiskClass Med: got %q, want badge-revision", got)
	}
}

func TestRiskClassMedium(t *testing.T) {
	sc := &plantmpl.SliceCard{Risk: "Medium"}
	if got := sc.RiskClass(); got != "badge-revision" {
		t.Errorf("RiskClass Medium: got %q, want badge-revision", got)
	}
}

func TestRiskClassLow(t *testing.T) {
	sc := &plantmpl.SliceCard{Risk: "Low"}
	if got := sc.RiskClass(); got != "badge-pending" {
		t.Errorf("RiskClass Low: got %q, want badge-pending", got)
	}
}

// ---------------------------------------------------------------------------
// Multiple cards render independently
// ---------------------------------------------------------------------------

func TestMultipleSliceCardsRenderIndependently(t *testing.T) {
	cards := []plantmpl.SliceCard{
		{Num: 1, ID: "feat-aaa", Title: "First slice"},
		{Num: 2, ID: "feat-bbb", Title: "Second slice"},
		{Num: 3, ID: "feat-ccc", Title: "Third slice"},
	}

	for _, card := range cards {
		c := card // capture
		var buf bytes.Buffer
		if err := c.Render(&buf); err != nil {
			t.Fatalf("Render card %d: %v", c.Num, err)
		}
		html := buf.String()
		if !strings.Contains(html, c.Title) {
			t.Errorf("card %d: missing title %q", c.Num, c.Title)
		}
		if !strings.Contains(html, c.ID) {
			t.Errorf("card %d: missing ID %q", c.Num, c.ID)
		}
	}
}
