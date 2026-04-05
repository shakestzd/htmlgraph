// Package plantmpl provides typed, component-based plan HTML generation.
//
// Each plan zone (dependency graph, design, outline, slices, questions,
// critique, finalize preview, progress bar) is a separate struct with a
// Render method. PlanPage assembles all zones into a complete HTML5 document.
//
// This replaces the monolithic plan-template.html with composable,
// testable components.
package plantmpl

import (
	"bytes"
	"embed"
	"html/template"
	"io"
	texttemplate "text/template"
)

//go:embed templates/*
var templateFS embed.FS

// renderZone calls Render on a Component and returns the result as
// template.HTML so it can be embedded directly in the page template.
func renderZone(c Component) template.HTML {
	if c == nil {
		return ""
	}
	var buf bytes.Buffer
	if err := c.Render(&buf); err != nil {
		return template.HTML("<!-- render error: " + err.Error() + " -->")
	}
	return template.HTML(buf.String())
}

// renderSlices renders all SliceCards and returns the concatenated HTML.
func renderSlices(cards []SliceCard) template.HTML {
	var buf bytes.Buffer
	for i := range cards {
		if err := cards[i].Render(&buf); err != nil {
			buf.WriteString("<!-- slice render error: " + err.Error() + " -->")
		}
	}
	return template.HTML(buf.String())
}

// planPageTmpl uses text/template (not html/template) because:
//   - Zone components handle their own HTML escaping via html/template
//   - The page shell contains static JS that must survive intact
//     (including JS comment markers used by runtime HTML patching)
//   - All dynamic values inserted at the page level are either
//     pre-rendered template.HTML or known-safe format (SectionsJSON)
var planPageTmpl = texttemplate.Must(
	texttemplate.New("plan_page.gohtml").Funcs(texttemplate.FuncMap{
		"renderZone":   renderZone,
		"renderSlices": renderSlices,
	}).ParseFS(templateFS, "templates/plan_page.gohtml"),
)

// Component is anything that can render itself into a plan zone.
type Component interface {
	Render(w io.Writer) error
}

// AssetRegistry collects CSS/JS blocks from zones for deduplication.
type AssetRegistry struct {
	css []string
	js  []string
}

// AddCSS appends a CSS block to the registry.
func (a *AssetRegistry) AddCSS(block string) { a.css = append(a.css, block) }

// AddJS appends a JS block to the registry.
func (a *AssetRegistry) AddJS(block string) { a.js = append(a.js, block) }

// CSS returns all collected CSS blocks.
func (a *AssetRegistry) CSS() []string { return a.css }

// JS returns all collected JS blocks.
func (a *AssetRegistry) JS() []string { return a.js }

// PlanPage is the top-level struct that assembles all zones into a
// complete plan HTML document.
type PlanPage struct {
	PlanID      string
	FeatureID   string
	Title       string
	Description string
	Date        string
	Status      string // "draft", "in-progress", "finalized", etc.

	// Zone components
	Graph     *DependencyGraph
	Design    *DesignSection
	Outline   *OutlineSection
	Slices    []SliceCard
	Questions *QuestionsSection
	Critique  *CritiqueZone
	Preview   *FinalizePreview
	Progress  *ProgressBar

	// Consolidated assets
	Assets *AssetRegistry
}

// Render writes the complete plan HTML to w.
func (p *PlanPage) Render(w io.Writer) error {
	if p.Assets == nil {
		p.Assets = &AssetRegistry{}
	}
	if p.Status == "" {
		p.Status = "draft"
	}
	return planPageTmpl.Execute(w, p)
}
