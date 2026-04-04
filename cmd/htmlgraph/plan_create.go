package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shakestzd/htmlgraph/internal/models"
	"github.com/shakestzd/htmlgraph/internal/workitem"
	"github.com/spf13/cobra"
)

// planCreateFromTopicCmd creates a plan node directly from a topic,
// without requiring a pre-existing track or feature.
func planCreateFromTopicCmd() *cobra.Command {
	var description string
	cmd := &cobra.Command{
		Use:   "create <title>",
		Short: "Create a plan from a topic",
		Long: `Create a plan node from a title and optional description.

Unlike 'plan generate' (which scaffolds from an existing work item), this
creates a standalone plan for design-first workflows. Add slices with
'plan add-slice', questions with 'plan add-question', then review and finalize.

Example:
  htmlgraph plan create "Auth Middleware Rewrite" --description "Rewrite for compliance"`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			htmlgraphDir, err := findHtmlgraphDir()
			if err != nil {
				return err
			}
			planID, err := createPlanFromTopic(htmlgraphDir, args[0], description)
			if err != nil {
				return err
			}
			planPath := filepath.Join(htmlgraphDir, "plans", planID+".html")
			fmt.Println(planPath)
			return nil
		},
	}
	cmd.Flags().StringVar(&description, "description", "", "plan description")
	return cmd
}

// createPlanFromTopic creates a plan node using the standard workitem template.
// Returns the plan ID (e.g. plan-a1b2c3d4).
func createPlanFromTopic(htmlgraphDir, title, description string) (string, error) {
	p, err := workitem.Open(htmlgraphDir, agentForClaim())
	if err != nil {
		return "", fmt.Errorf("open project: %w", err)
	}
	defer p.Close()

	opts := []workitem.PlanOption{
		workitem.PlanWithPriority("medium"),
	}
	if description != "" {
		opts = append(opts, workitem.PlanWithContent(description))
	}

	node, err := p.Plans.Create(title, opts...)
	if err != nil {
		return "", fmt.Errorf("create plan: %w", err)
	}

	return node.ID, nil
}

// ---- plan add-slice ---------------------------------------------------------

// planAddSliceCmd adds a new vertical slice (as a step) to an existing plan.
func planAddSliceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add-slice <plan-id> <title>",
		Short: "Add a vertical slice to a plan",
		Long: `Add a new slice as a step to an existing plan.

Example:
  htmlgraph plan add-slice plan-a1b2c3d4 "Implement error handling"`,
		Args: cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			htmlgraphDir, err := findHtmlgraphDir()
			if err != nil {
				return err
			}
			return addSliceToPlan(htmlgraphDir, args[0], args[1])
		},
	}
}

// addSliceToPlan adds a slice as a step to the plan node.
func addSliceToPlan(htmlgraphDir, planID, sliceTitle string) error {
	p, err := workitem.Open(htmlgraphDir, agentForClaim())
	if err != nil {
		return fmt.Errorf("open project: %w", err)
	}
	defer p.Close()

	edit := p.Plans.Edit(planID)
	edit = edit.AddStep(sliceTitle)
	if err := edit.Save(); err != nil {
		return fmt.Errorf("add slice to plan %s: %w", planID, err)
	}

	// Count steps to report the slice number.
	node, err := p.Plans.Get(planID)
	if err != nil {
		fmt.Printf("Added slice: %s\n", sliceTitle)
		return nil
	}
	fmt.Printf("Added slice #%d: %s\n", len(node.Steps), sliceTitle)
	return nil
}

// ---- helpers for finalize (slice parsing from node steps) --------------------

// parsePlanStepsAsSlices converts plan node steps into planSlice structs
// for the finalize workflow.
func parsePlanStepsAsSlices(node *models.Node) []planSlice {
	var slices []planSlice
	for i, step := range node.Steps {
		slices = append(slices, planSlice{
			num:   i + 1,
			name:  step.StepID,
			title: step.Description,
		})
	}
	return slices
}

// isPlanApproved checks if a plan has been marked as approved/finalized
// by looking at its status.
func isPlanApproved(node *models.Node) bool {
	return node.Status == "finalized" || node.Status == "done"
}

// findPlanFile returns the path to a plan's HTML file, or empty string.
func findPlanFile(htmlgraphDir, planID string) string {
	p := filepath.Join(htmlgraphDir, "plans", planID+".html")
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return ""
}

// updatePlanStatus updates the data-status attribute in a plan's HTML file.
func updatePlanStatus(htmlgraphDir, planID, newStatus string) error {
	planPath := findPlanFile(htmlgraphDir, planID)
	if planPath == "" {
		return fmt.Errorf("plan file not found: %s", planID)
	}
	data, err := os.ReadFile(planPath)
	if err != nil {
		return err
	}
	content := string(data)

	// Replace the status in data-status="..."
	for _, old := range []string{"todo", "draft", "in-progress", "done", "finalized"} {
		old := fmt.Sprintf(`data-status="%s"`, old)
		new := fmt.Sprintf(`data-status="%s"`, newStatus)
		if strings.Contains(content, old) {
			content = strings.Replace(content, old, new, 1)
			break
		}
	}

	return os.WriteFile(planPath, []byte(content), 0o644)
}
