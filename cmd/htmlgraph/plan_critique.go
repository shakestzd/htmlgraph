package main

import (
	"encoding/json"
	"fmt"
	"os"

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

	out := &critiqueOutput{
		PlanID:      planID,
		Title:       node.Title,
		Description: node.Content,
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
