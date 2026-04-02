// Register in main.go: rootCmd.AddCommand(complianceCmd())
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/shakestzd/htmlgraph/internal/workitem"
	"github.com/spf13/cobra"
)

// criterionStatus represents the state of a single acceptance criterion.
type criterionStatus int

const (
	criterionUnchecked criterionStatus = iota
	criterionPassed
	criterionFailed
)

// criterion holds a parsed acceptance criterion and its status.
type criterion struct {
	Index  int             `json:"index"`
	Text   string          `json:"text"`
	Status criterionStatus `json:"status"`
}

// complianceResult holds the full compliance report for a feature.
type complianceResult struct {
	FeatureID  string      `json:"feature_id"`
	Criteria   []criterion `json:"criteria"`
	Total      int         `json:"total"`
	Passed     int         `json:"passed"`
	Failed     int         `json:"failed"`
	Unchecked  int         `json:"unchecked"`
	HasSpec    bool        `json:"has_spec"`
	HasFailure bool        `json:"has_failure"`
}

// criterionPattern matches lines like: "1. [ ] text", "2. [x] text", "- [ ] text", "- [x] text"
var criterionPattern = regexp.MustCompile(`(?i)^[\s\-\d\.]*\[([x\s])\]\s+(.+)$`)

func complianceCmd() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "compliance <feature-id>",
		Short: "Score how well the implementation matches the spec's acceptance criteria",
		Long: `Read the spec section from a feature HTML file and report on each acceptance criterion.

Criteria marked with [ ] are UNCHECKED, [x] are PASSED.
Exit 0 if no failures found; exit 1 if any criteria are explicitly marked as failed.`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runCompliance(args[0], jsonOut)
		},
	}

	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}

func runCompliance(featureID string, jsonOut bool) error {
	result, err := computeCompliance(featureID)
	if err != nil {
		return err
	}

	if jsonOut {
		return printComplianceJSON(result)
	}
	return printComplianceText(result)
}

// computeCompliance reads the feature HTML and scores the spec criteria.
func computeCompliance(featureID string) (*complianceResult, error) {
	dir, err := findHtmlgraphDir()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(dir, "features", featureID+".html")
	if _, err := os.Stat(path); err != nil {
		return nil, workitem.ErrNotFound("feature", featureID)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read feature file: %w", err)
	}

	result := &complianceResult{FeatureID: featureID}

	specContent := extractSpecSection(string(content))
	if specContent == "" {
		result.HasSpec = false
		return result, nil
	}

	result.HasSpec = true
	result.Criteria = parseCriteria(specContent)
	result.Total = len(result.Criteria)

	for _, c := range result.Criteria {
		switch c.Status {
		case criterionPassed:
			result.Passed++
		case criterionFailed:
			result.Failed++
			result.HasFailure = true
		default:
			result.Unchecked++
		}
	}

	return result, nil
}

// parseCriteria extracts acceptance criteria lines from spec content.
func parseCriteria(content string) []criterion {
	var criteria []criterion
	inSection := false
	idx := 1

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)

		// Track when we enter/exit the Acceptance Criteria section.
		if strings.HasPrefix(trimmed, "## Acceptance Criteria") {
			inSection = true
			continue
		}
		if inSection && strings.HasPrefix(trimmed, "## ") {
			inSection = false
			continue
		}

		if !inSection {
			continue
		}

		m := criterionPattern.FindStringSubmatch(trimmed)
		if m == nil {
			continue
		}

		status := criterionUnchecked
		if strings.ToLower(m[1]) == "x" {
			status = criterionPassed
		}

		criteria = append(criteria, criterion{
			Index:  idx,
			Text:   strings.TrimSpace(m[2]),
			Status: status,
		})
		idx++
	}

	return criteria
}

// printComplianceText renders a human-readable compliance report.
func printComplianceText(r *complianceResult) error {
	if !r.HasSpec {
		fmt.Printf("No spec found for %s. Run: htmlgraph spec generate %s\n", r.FeatureID, r.FeatureID)
		return nil
	}

	if r.Total == 0 {
		fmt.Printf("Spec found for %s but contains no acceptance criteria.\n", r.FeatureID)
		return nil
	}

	fmt.Printf("Compliance: %s\n\n", r.FeatureID)

	for _, c := range r.Criteria {
		label, marker := criterionLabel(c.Status)
		fmt.Printf("  %s %d. %s — %s\n", marker, c.Index, c.Text, label)
	}

	fmt.Printf("\nScore: %d/%d criteria checked", r.Passed, r.Total)
	if r.Unchecked > 0 {
		fmt.Printf(" (%d unchecked)", r.Unchecked)
	}
	if r.Failed > 0 {
		fmt.Printf(" (%d failed)", r.Failed)
	}
	fmt.Println()

	if r.HasFailure {
		return fmt.Errorf("compliance check failed: %d criteria marked as failed", r.Failed)
	}
	return nil
}

// criterionLabel returns display label and marker for a criterion status.
func criterionLabel(s criterionStatus) (string, string) {
	switch s {
	case criterionPassed:
		return "PASS", "✓"
	case criterionFailed:
		return "FAIL", "✗"
	default:
		return "UNCHECKED", "·"
	}
}

// printComplianceJSON writes the result as JSON.
func printComplianceJSON(r *complianceResult) error {
	// Reformat criteria for JSON with string status.
	type jsonCriterion struct {
		Index  int    `json:"index"`
		Text   string `json:"text"`
		Status string `json:"status"`
	}
	type jsonResult struct {
		FeatureID  string          `json:"feature_id"`
		Criteria   []jsonCriterion `json:"criteria"`
		Total      int             `json:"total"`
		Passed     int             `json:"passed"`
		Failed     int             `json:"failed"`
		Unchecked  int             `json:"unchecked"`
		HasSpec    bool            `json:"has_spec"`
		HasFailure bool            `json:"has_failure"`
	}

	out := jsonResult{
		FeatureID:  r.FeatureID,
		Total:      r.Total,
		Passed:     r.Passed,
		Failed:     r.Failed,
		Unchecked:  r.Unchecked,
		HasSpec:    r.HasSpec,
		HasFailure: r.HasFailure,
	}
	for _, c := range r.Criteria {
		var statusStr string
		switch c.Status {
		case criterionPassed:
			statusStr = "pass"
		case criterionFailed:
			statusStr = "fail"
		default:
			statusStr = "unchecked"
		}
		out.Criteria = append(out.Criteria, jsonCriterion{
			Index:  c.Index,
			Text:   c.Text,
			Status: statusStr,
		})
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
