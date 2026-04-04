package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/spf13/cobra"
)

// planValidation holds the result of structural validation for a CRISPI plan HTML file.
type planValidation struct {
	PlanID   string   `json:"plan_id"`
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
	Stats    struct {
		Slices     int `json:"slices"`
		Questions  int `json:"questions"`
		GraphNodes int `json:"graph_nodes"`
	} `json:"stats"`
}

// validPlanStatuses is the set of accepted data-status values on the article element.
var validPlanStatuses = map[string]bool{
	"draft":     true,
	"in-review": true,
	"finalized": true,
}

// sectionsJSONRe matches the SECTIONS array between the JS markers.
var sectionsJSONRe = regexp.MustCompile(`/\*PLAN_SECTIONS_JSON\*/(.*?)/\*END_PLAN_SECTIONS_JSON\*/`)

// planValidateCmd returns the cobra command for "plan validate".
func planValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate <plan-id>",
		Short: "Validate structural integrity of a CRISPI plan HTML file",
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

	planPath := fmt.Sprintf("%s/plans/%s.html", htmlgraphDir, planID)
	result, err := validatePlanHTML(planPath)
	if err != nil {
		return fmt.Errorf("validate plan: %w", err)
	}
	result.PlanID = planID

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

// validatePlanHTML performs structural validation on a CRISPI plan HTML file
// at planPath. Returns the validation result and any I/O error.
func validatePlanHTML(planPath string) (planValidation, error) {
	f, err := os.Open(planPath)
	if err != nil {
		return planValidation{}, fmt.Errorf("open plan: %w", err)
	}
	defer f.Close()

	doc, err := goquery.NewDocumentFromReader(f)
	if err != nil {
		return planValidation{}, fmt.Errorf("parse HTML: %w", err)
	}

	// Also read raw text for JS section extraction.
	rawBytes, err := os.ReadFile(planPath)
	if err != nil {
		return planValidation{}, fmt.Errorf("read plan: %w", err)
	}
	rawContent := string(rawBytes)

	var result planValidation
	result.Valid = true

	addError := func(msg string) {
		result.Errors = append(result.Errors, msg)
		result.Valid = false
	}

	// 1. Validate article status.
	article := doc.Find("article").First()
	status, _ := article.Attr("data-status")
	if status == "" {
		addError("article element is missing data-status attribute")
	} else if !validPlanStatuses[status] {
		addError(fmt.Sprintf("invalid plan status %q — must be one of: draft, in-review, finalized", status))
	}

	// 2. Required sections: design, outline, slices phase (implicitly checked via slice cards).
	if doc.Find(`[data-phase="design"]`).Length() == 0 {
		addError("missing required design section ([data-phase=\"design\"])")
	}
	if doc.Find(`[data-phase="outline"]`).Length() == 0 {
		addError("missing required outline section ([data-phase=\"outline\"])")
	}

	// 3. Finalize button must exist.
	if doc.Find("#finalizeBtn").Length() == 0 {
		addError("missing required finalize button (#finalizeBtn)")
	}

	// 4. Count graph nodes.
	graphNodes := doc.Find("#graph-data [data-node]").Length()
	result.Stats.GraphNodes = graphNodes

	// 5. Slice card integrity.
	sliceCount := 0
	doc.Find(".slice-card[data-slice]").Each(func(_ int, sel *goquery.Selection) {
		sliceNum, _ := sel.Attr("data-slice")
		sectionKey := "slice-" + sliceNum
		sliceCount++

		// Each slice card must have an approval checkbox.
		checkbox := sel.Find(fmt.Sprintf(`input[type="checkbox"][data-section="%s"][data-action="approve"]`, sectionKey))
		if checkbox.Length() == 0 {
			addError(fmt.Sprintf("slice card %s is missing approval checkbox (input[data-section=%q][data-action=\"approve\"])", sectionKey, sectionKey))
		}

		// Each slice card must have a comment textarea.
		textarea := sel.Find(fmt.Sprintf(`textarea[data-comment-for="%s"]`, sectionKey))
		if textarea.Length() == 0 {
			addError(fmt.Sprintf("slice card %s is missing comment textarea (textarea[data-comment-for=%q])", sectionKey, sectionKey))
		}

		// data-status attribute must be present on the slice card.
		if _, ok := sel.Attr("data-status"); !ok {
			addError(fmt.Sprintf("slice card %s is missing data-status attribute", sectionKey))
		}
	})
	result.Stats.Slices = sliceCount

	// 6. Graph node count must match slice card count (only when both are non-zero).
	if sliceCount > 0 && graphNodes > 0 && sliceCount != graphNodes {
		addError(fmt.Sprintf("graph node count (%d) does not match slice card count (%d)", graphNodes, sliceCount))
	}

	// 7. Question integrity: each question block must have radio inputs and a recap row.
	questionCount := 0
	doc.Find(".question-block[data-question]").Each(func(_ int, sel *goquery.Selection) {
		questionID, _ := sel.Attr("data-question")
		questionCount++

		// Must have at least one radio input referencing this question.
		radios := sel.Find(fmt.Sprintf(`input[type="radio"][data-question="%s"]`, questionID))
		if radios.Length() == 0 {
			addError(fmt.Sprintf("question %q has no radio inputs (input[type=\"radio\"][data-question=%q])", questionID, questionID))
		}

		// Must have a corresponding recap row in the questions recap table.
		recap := doc.Find(fmt.Sprintf(`[data-recap-for="%s"]`, questionID))
		if recap.Length() == 0 {
			addError(fmt.Sprintf("question %q is missing a recap row ([data-recap-for=%q])", questionID, questionID))
		}
	})
	result.Stats.Questions = questionCount

	// 8. SECTIONS JSON consistency: every entry in the JS SECTIONS array must have a
	//    corresponding approval checkbox in the HTML.
	sections := extractSectionsFromJS(rawContent)
	for _, sec := range sections {
		cb := doc.Find(fmt.Sprintf(`input[data-section="%s"][data-action="approve"]`, sec))
		if cb.Length() == 0 {
			addError(fmt.Sprintf("SECTIONS array references %q but no matching approval checkbox found in HTML", sec))
		}
	}

	return result, nil
}

// extractSectionsFromJS parses the SECTIONS JS array from the plan HTML source.
// Returns nil if the marker is absent (e.g. the template placeholder was never replaced).
func extractSectionsFromJS(rawContent string) []string {
	m := sectionsJSONRe.FindStringSubmatch(rawContent)
	if len(m) < 2 {
		return nil
	}
	arrayLiteral := strings.TrimSpace(m[1])
	// Expect format: ['a','b','c'] or ["a","b","c"]
	arrayLiteral = strings.TrimPrefix(arrayLiteral, "[")
	arrayLiteral = strings.TrimSuffix(arrayLiteral, "]")
	if arrayLiteral == "" {
		return nil
	}

	var sections []string
	for _, part := range strings.Split(arrayLiteral, ",") {
		part = strings.TrimSpace(part)
		part = strings.Trim(part, `'"`)
		if part != "" {
			sections = append(sections, part)
		}
	}
	return sections
}
