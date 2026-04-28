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
	ID          string // YAML slice id like "auth-init" (displayed in slice card)
	FeatureID   string // generated feature ID like "feat-abc123" (for Related Features lookup)
	Title       string
	Description string   // Legacy: flat description text (used when What is empty)
	What        string   // Structured: what to implement (Markdown source)
	Why         string   // Structured: rationale / motivation (Markdown source)
	DoneWhen    []string // Structured: acceptance criteria bullets (literal, no Markdown)
	Tests       string   // Test strategy text (Markdown source)
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

// WhatHTML returns the What field rendered as sanitized HTML.
func (sc *SliceCard) WhatHTML() template.HTML { return RenderMd(sc.What) }

// WhyHTML returns the Why field rendered as sanitized HTML.
func (sc *SliceCard) WhyHTML() template.HTML { return RenderMd(sc.Why) }

// DescriptionHTML returns the Description field rendered as sanitized HTML.
func (sc *SliceCard) DescriptionHTML() template.HTML { return RenderMd(sc.Description) }

// TestsHTML returns the Tests field rendered as sanitized HTML.
func (sc *SliceCard) TestsHTML() template.HTML { return RenderMd(sc.Tests) }

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
