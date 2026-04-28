package plantmpl

import (
	"fmt"
	"html/template"
	"io"
	"strings"

	"github.com/shakestzd/htmlgraph/internal/planyaml"
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

	// V2 lifecycle fields (additive — legacy plans omit these and remain valid).
	ApprovalStatus  string // pending | approved | rejected | changes_requested
	ExecutionStatus string // not_started | promoted | in_progress | done | blocked | superseded

	// V2 slice-local spec fields.
	Questions       []planyaml.SliceQuestion  // slice-local open questions
	CriticRevisions []planyaml.CriticRevision // critic feedback specific to this slice
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

// ApprovalStatusClass returns the CSS badge class for the approval status.
func (sc *SliceCard) ApprovalStatusClass() string {
	switch sc.ApprovalStatus {
	case "approved":
		return "badge-approved"
	case "rejected":
		return "badge-blocked"
	case "changes_requested":
		return "badge-revision"
	default:
		return "badge-pending"
	}
}

// ApprovalStatusLabel returns the display label for the approval status.
func (sc *SliceCard) ApprovalStatusLabel() string {
	switch sc.ApprovalStatus {
	case "approved":
		return "Approved"
	case "rejected":
		return "Rejected"
	case "changes_requested":
		return "Changes Requested"
	default:
		return "Pending"
	}
}

// ExecutionStatusLabel returns a display label for the execution status.
func (sc *SliceCard) ExecutionStatusLabel() string {
	switch sc.ExecutionStatus {
	case "not_started":
		return "Not Started"
	case "promoted":
		return "Promoted"
	case "in_progress":
		return "In Progress"
	case "done":
		return "Done"
	case "blocked":
		return "Blocked"
	case "superseded":
		return "Superseded"
	default:
		return sc.ExecutionStatus
	}
}

// ExecutionStatusClass returns the CSS badge class for the execution status.
func (sc *SliceCard) ExecutionStatusClass() string {
	switch sc.ExecutionStatus {
	case "done":
		return "badge-approved"
	case "in_progress", "promoted":
		return "badge-revision"
	case "blocked", "superseded":
		return "badge-blocked"
	default:
		return "badge-pending"
	}
}

// CriticSeverityClass returns the CSS badge class for a critic revision severity.
func (sc *SliceCard) CriticSeverityClass(severity string) string {
	upper := strings.ToUpper(severity)
	switch {
	case upper == "HIGH" || upper == "DANGER":
		return "badge-blocked"
	case upper == "MED" || upper == "MEDIUM":
		return "badge-revision"
	default:
		return "badge-pending"
	}
}

// SliceQuestionSectionKey returns the plan_feedback section key for a
// slice-local question, following the contract: slice-<num>-question-<id>.
func (sc *SliceCard) SliceQuestionSectionKey(questionID string) string {
	return fmt.Sprintf("slice-%d-question-%s", sc.Num, questionID)
}

// SliceCardFromPlanSlice maps a planyaml.PlanSlice to a SliceCard.
// This is the canonical mapping used by enrichPageFromYAML and tests.
func SliceCardFromPlanSlice(s planyaml.PlanSlice) SliceCard {
	depsStr := ""
	for i, d := range s.Deps {
		if i > 0 {
			depsStr += ","
		}
		depsStr += fmt.Sprintf("%d", d)
	}
	filesStr := strings.Join(s.Files, ", ")

	return SliceCard{
		Num:             s.Num,
		ID:              s.ID,
		FeatureID:       s.FeatureID,
		Title:           s.Title,
		What:            s.What,
		Why:             s.Why,
		DoneWhen:        s.DoneWhen,
		Tests:           s.Tests,
		Effort:          s.Effort,
		Risk:            s.Risk,
		Deps:            depsStr,
		Files:           filesStr,
		Status:          "pending",
		ApprovalStatus:  s.ApprovalStatus,
		ExecutionStatus: s.ExecutionStatus,
		Questions:       s.Questions,
		CriticRevisions: s.CriticRevisions,
	}
}
