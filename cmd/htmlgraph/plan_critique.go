package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shakestzd/htmlgraph/internal/workitem"
	"github.com/spf13/cobra"
)

// critiqueOutput is the structured JSON output from plan critique.
type critiqueOutput struct {
	PlanID            string          `json:"plan_id"`
	Title             string          `json:"title"`
	Description       string          `json:"description,omitempty"`
	Status            string          `json:"status"`
	Complexity        string          `json:"complexity"`
	SliceCount        int             `json:"slice_count"`
	CritiqueWarranted bool            `json:"critique_warranted"`
	Slices            []critiqueSlice `json:"slices,omitempty"`
}

type critiqueSlice struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
}

// planCritiqueCmd extracts plan content for AI critique.
func planCritiqueCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "critique <plan-id>",
		Short: "Extract plan content for AI review",
		Long: `Read a plan and output structured JSON for AI critique.

Complexity-gated: plans with fewer than 3 slices output
critique_warranted=false.

Example:
  htmlgraph plan critique plan-a1b2c3d4`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			htmlgraphDir, err := findHtmlgraphDir()
			if err != nil {
				return err
			}
			return runPlanCritique(htmlgraphDir, args[0])
		},
	}
}

func runPlanCritique(htmlgraphDir, planID string) error {
	out, err := extractCritiqueData(htmlgraphDir, planID)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// extractCritiqueData reads a plan node and extracts structured data for critique.
func extractCritiqueData(htmlgraphDir, planID string) (*critiqueOutput, error) {
	p, err := workitem.Open(htmlgraphDir, agentForClaim())
	if err != nil {
		return nil, fmt.Errorf("open project: %w", err)
	}
	defer p.Close()

	node, err := p.Plans.Get(planID)
	if err != nil {
		return nil, fmt.Errorf("plan %q not found: %w", planID, err)
	}

	// Extract description from plan HTML, not node.Content.
	// For CRISPI plans, the description is in the <p> tag after the <h1>.
	description := extractPlanDescription(htmlgraphDir, planID)

	out := &critiqueOutput{
		PlanID:      planID,
		Title:       strings.TrimPrefix(node.Title, "Plan: "),
		Description: description,
		Status:      string(node.Status),
	}

	// Extract slices from steps.
	for i, step := range node.Steps {
		out.Slices = append(out.Slices, critiqueSlice{
			Number: i + 1,
			Title:  step.Description,
		})
	}

	// Complexity gate.
	out.SliceCount = len(out.Slices)
	out.Complexity, out.CritiqueWarranted = classifyComplexity(out.SliceCount)

	return out, nil
}

// extractPlanDescription reads the plan HTML file and extracts the description
// from the <p> tag immediately after the <h1> in the header.
func extractPlanDescription(htmlgraphDir, planID string) string {
	planPath := filepath.Join(htmlgraphDir, "plans", planID+".html")
	data, err := os.ReadFile(planPath)
	if err != nil {
		return ""
	}

	htmlContent := string(data)

	// Find the <h1> tag.
	h1Start := strings.Index(htmlContent, "<h1>")
	if h1Start < 0 {
		return ""
	}

	// Find the end of <h1> tag.
	h1End := strings.Index(htmlContent[h1Start:], "</h1>")
	if h1End < 0 {
		return ""
	}

	// Search for <p (allowing for attributes like <p style="...">).
	searchStart := h1Start + h1End + 5 // 5 = len("</h1>")
	pStart := strings.Index(htmlContent[searchStart:], "<p")
	if pStart < 0 {
		return ""
	}

	// Extract text between the closing > and </p>.
	pStart += searchStart
	closeTagIdx := strings.Index(htmlContent[pStart:], ">")
	if closeTagIdx < 0 {
		return ""
	}
	rest := htmlContent[pStart+closeTagIdx+1:]
	pEnd := strings.Index(rest, "</p>")
	if pEnd < 0 {
		return ""
	}

	desc := strings.TrimSpace(rest[:pEnd])

	// Strip HTML tags if present (e.g., <strong>, <em>).
	desc = strings.ReplaceAll(desc, "<strong>", "")
	desc = strings.ReplaceAll(desc, "</strong>", "")
	desc = strings.ReplaceAll(desc, "<em>", "")
	desc = strings.ReplaceAll(desc, "</em>", "")
	desc = strings.ReplaceAll(desc, "<i>", "")
	desc = strings.ReplaceAll(desc, "</i>", "")
	desc = strings.ReplaceAll(desc, "<b>", "")
	desc = strings.ReplaceAll(desc, "</b>", "")

	return desc
}

// classifyComplexity determines plan complexity and whether critique is warranted.
func classifyComplexity(sliceCount int) (complexity string, warranted bool) {
	switch {
	case sliceCount < 3:
		return "low", false
	case sliceCount < 6:
		return "medium", true
	default:
		return "high", true
	}
}
