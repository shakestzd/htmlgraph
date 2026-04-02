// Register in main.go: rootCmd.AddCommand(specCmd())
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shakestzd/htmlgraph/internal/htmlparse"
	"github.com/shakestzd/htmlgraph/internal/workitem"
	"github.com/spf13/cobra"
)

// specTemplate holds the structured fields of a feature spec.
type specTemplate struct {
	FeatureID          string   `json:"feature_id"`
	Title              string   `json:"title"`
	Problem            string   `json:"problem"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
	Files              []string `json:"files"`
	APISurface         []string `json:"api_surface"`
	Notes              []string `json:"notes"`
}

func specCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spec",
		Short: "Manage specs and feature specifications",
		Long:  "Create spec work items and generate structured spec templates for features.",
	}
	// Spec work item CRUD
	cmd.AddCommand(wiCreateCmd("spec", "specs"))
	cmd.AddCommand(wiListCmd("spec", "specs"))
	cmd.AddCommand(wiStartCmd("spec"))
	cmd.AddCommand(wiCompleteCmd("spec"))
	cmd.AddCommand(wiDeleteCmd("spec"))
	// Spec template generation (feature-specific)
	cmd.AddCommand(specGenerateCmd())
	cmd.AddCommand(specShowCmd())
	return cmd
}

func specGenerateCmd() *cobra.Command {
	var format, output string

	cmd := &cobra.Command{
		Use:   "generate <feature-id>",
		Short: "Generate a spec template for a feature",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runSpecGenerate(args[0], format, output)
		},
	}
	cmd.Flags().StringVar(&format, "format", "markdown", "Output format: markdown or json")
	cmd.Flags().StringVar(&output, "output", "", "Write output to file instead of stdout")
	return cmd
}

func specShowCmd() *cobra.Command {
	var format, output string

	cmd := &cobra.Command{
		Use:   "show <feature-id>",
		Short: "Show the spec for a feature if one exists",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runSpecShow(args[0], format, output)
		},
	}
	cmd.Flags().StringVar(&format, "format", "markdown", "Output format: markdown or json")
	cmd.Flags().StringVar(&output, "output", "", "Write output to file instead of stdout")
	return cmd
}

// runSpecGenerate reads the feature title and outputs a blank spec template.
func runSpecGenerate(featureID, format, output string) error {
	title, err := resolveFeatureTitle(featureID)
	if err != nil {
		return err
	}

	spec := buildBlankSpec(featureID, title)
	return writeSpec(spec, format, output, featureID)
}

// runSpecShow looks for an existing spec section in the feature HTML.
func runSpecShow(featureID, format, output string) error {
	dir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}

	path := filepath.Join(dir, "features", featureID+".html")
	if _, err := os.Stat(path); err != nil {
		return workitem.ErrNotFound("feature", featureID)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read feature file: %w", err)
	}

	specContent := extractSpecSection(string(content))
	if specContent == "" {
		fmt.Printf("No spec found. Run: htmlgraph spec generate %s\n", featureID)
		return nil
	}

	if format == "json" {
		return writeOutput(buildSpecJSON(featureID, specContent), output)
	}
	return writeOutput(specContent, output)
}

// resolveFeatureTitle looks up a feature by ID and returns its title.
func resolveFeatureTitle(featureID string) (string, error) {
	dir, err := findHtmlgraphDir()
	if err != nil {
		return "", err
	}

	path := filepath.Join(dir, "features", featureID+".html")
	if _, err := os.Stat(path); err != nil {
		return "", workitem.ErrNotFound("feature", featureID)
	}

	node, err := htmlparse.ParseFile(path)
	if err != nil {
		return "", fmt.Errorf("parse feature %s: %w", featureID, err)
	}
	return node.Title, nil
}

// buildBlankSpec constructs an empty spec template for the given feature.
func buildBlankSpec(featureID, title string) *specTemplate {
	return &specTemplate{
		FeatureID:          featureID,
		Title:              title,
		Problem:            "What problem does this solve? Why is it needed?",
		AcceptanceCriteria: []string{"Criterion 1", "Criterion 2", "Criterion 3"},
		Files:              []string{"NEW: path/to/new/file.go", "EDIT: path/to/existing/file.go"},
		APISurface:         []string{"Function/command signatures", "Input/output formats"},
		Notes:              []string{"Dependencies, risks, edge cases"},
	}
}

// writeSpec serialises and outputs the spec in the requested format.
func writeSpec(spec *specTemplate, format, output, _ string) error {
	var content string
	switch format {
	case "json":
		data, err := json.MarshalIndent(spec, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal spec: %w", err)
		}
		content = string(data)
	default:
		content = renderSpecMarkdown(spec)
	}
	return writeOutput(content, output)
}

// renderSpecMarkdown formats the spec as a Markdown document.
func renderSpecMarkdown(s *specTemplate) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Spec: %s\n\n", s.Title))

	sb.WriteString("## Problem\n")
	sb.WriteString(s.Problem + "\n\n")

	sb.WriteString("## Acceptance Criteria\n")
	for i, c := range s.AcceptanceCriteria {
		sb.WriteString(fmt.Sprintf("%d. [ ] %s\n", i+1, c))
	}
	sb.WriteString("\n")

	sb.WriteString("## Files\n")
	for _, f := range s.Files {
		sb.WriteString(fmt.Sprintf("- %s\n", f))
	}
	sb.WriteString("\n")

	sb.WriteString("## API Surface\n")
	for _, a := range s.APISurface {
		sb.WriteString(fmt.Sprintf("- %s\n", a))
	}
	sb.WriteString("\n")

	sb.WriteString("## Notes\n")
	for _, n := range s.Notes {
		sb.WriteString(fmt.Sprintf("- %s\n", n))
	}

	return sb.String()
}

// extractSpecSection looks for <section class="spec"> inside feature HTML.
func extractSpecSection(html string) string {
	const open = `<section class="spec">`
	const close = `</section>`

	start := strings.Index(html, open)
	if start == -1 {
		return ""
	}
	end := strings.Index(html[start:], close)
	if end == -1 {
		return ""
	}
	inner := html[start+len(open) : start+end]
	return strings.TrimSpace(inner)
}

// buildSpecJSON wraps raw spec content in a minimal JSON envelope.
func buildSpecJSON(featureID, content string) string {
	m := map[string]string{
		"feature_id": featureID,
		"spec":       content,
	}
	data, _ := json.MarshalIndent(m, "", "  ")
	return string(data)
}

// writeOutput writes content to a file or stdout.
func writeOutput(content, path string) error {
	if path == "" {
		fmt.Println(content)
		return nil
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	fmt.Printf("Spec written to %s\n", path)
	return nil
}
