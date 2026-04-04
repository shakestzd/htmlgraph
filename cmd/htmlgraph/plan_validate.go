package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/shakestzd/htmlgraph/internal/workitem"
	"github.com/spf13/cobra"
)

// planValidation holds the result of structural validation for a plan.
type planValidation struct {
	PlanID   string   `json:"plan_id"`
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
	Stats    struct {
		Slices    int `json:"slices"`
		Questions int `json:"questions"`
		GraphNodes int `json:"graph_nodes"`
	} `json:"stats"`
}

// planValidateCmd returns the cobra command for "plan validate".
func planValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate <plan-id>",
		Short: "Validate a plan's structure and content",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runPlanValidate(args[0])
		},
	}
}

func runPlanValidate(planID string) error {
	htmlgraphDir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}

	result, err := validatePlan(htmlgraphDir, planID)
	if err != nil {
		return fmt.Errorf("validate plan: %w", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

// validatePlan performs structural validation on a plan node.
func validatePlan(htmlgraphDir, planID string) (planValidation, error) {
	p, err := workitem.Open(htmlgraphDir, agentForClaim())
	if err != nil {
		return planValidation{}, fmt.Errorf("open project: %w", err)
	}
	defer p.Close()

	node, err := p.Plans.Get(planID)
	if err != nil {
		return planValidation{}, fmt.Errorf("plan %q not found: %w", planID, err)
	}

	var result planValidation
	result.PlanID = planID
	result.Valid = true

	addError := func(msg string) {
		result.Errors = append(result.Errors, msg)
		result.Valid = false
	}

	addWarning := func(msg string) {
		result.Warnings = append(result.Warnings, msg)
	}

	// Validate status.
	validStatuses := map[string]bool{
		"todo": true, "in-progress": true, "done": true, "finalized": true,
	}
	if !validStatuses[string(node.Status)] {
		addError(fmt.Sprintf("invalid plan status %q", node.Status))
	}

	// Validate title.
	if node.Title == "" {
		addError("plan is missing a title")
	}

	// Count slices (steps).
	result.Stats.Slices = len(node.Steps)
	result.Stats.GraphNodes = len(node.Steps) // nodes = steps in node template

	// Warn if no description.
	if node.Content == "" {
		addWarning("plan has no description")
	}

	// Warn if no slices.
	if len(node.Steps) == 0 {
		addWarning("plan has no slices (steps)")
	}

	// Verify plan HTML file exists on disk.
	planPath := findPlanFile(htmlgraphDir, planID)
	if planPath == "" {
		addError("plan HTML file not found on disk")
	}

	return result, nil
}
