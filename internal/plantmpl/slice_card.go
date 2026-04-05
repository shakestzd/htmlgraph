package plantmpl

import (
	"html/template"
	"io"
)

var sliceCardTmpl = template.Must(
	template.ParseFS(templateFS, "templates/slice_card.gohtml"),
)

// SliceCard renders a single implementation slice with its metadata,
// dependencies, and approval status.
type SliceCard struct {
	Num         int
	ID          string // feature ID like "feat-abc123"
	Title       string
	Description string   // Legacy: flat description text (used when What is empty)
	What        string   // Structured: what to implement
	Why         string   // Structured: rationale / motivation
	DoneWhen    []string // Structured: acceptance criteria bullets
	Tests       string   // Test strategy text
	Effort      string   // "S", "M", "L"
	Risk        string   // "Low", "Med", "High"
	Deps        string   // comma-separated slice numbers
	Files       string   // comma-separated file paths
	Status      string
}

// HasStructuredContent returns true when the slice has What/Why fields
// (benchmark format) rather than just a flat description.
func (sc *SliceCard) HasStructuredContent() bool {
	return sc.What != "" || sc.Why != ""
}

// Render writes the slice card HTML.
func (sc *SliceCard) Render(w io.Writer) error {
	return sliceCardTmpl.Execute(w, sc)
}

// EffortClass returns the CSS class for the effort badge.
func (sc *SliceCard) EffortClass() string {
	switch sc.Effort {
	case "S":
		return "badge-pending"
	case "M":
		return "badge-revision"
	case "L":
		return "badge-blocked"
	default:
		return "badge-pending"
	}
}

// RiskClass returns the CSS class for the risk badge.
func (sc *SliceCard) RiskClass() string {
	switch sc.Risk {
	case "High":
		return "badge-blocked"
	case "Med", "Medium":
		return "badge-revision"
	default:
		return "badge-pending"
	}
}

// DepsLabel returns a human-readable dependency string.
func (sc *SliceCard) DepsLabel() string {
	if sc.Deps == "" {
		return "none"
	}
	return "slices " + sc.Deps
}
