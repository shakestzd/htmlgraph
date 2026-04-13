package main

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// ---- plan set-section -------------------------------------------------------

func planSetSectionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-section <plan-id> <placeholder> <html-content>",
		Short: "(deprecated) Set content for a plan section placeholder",
		Long: `Inject HTML content into a named placeholder in the plan.

Placeholders: PLAN_DESIGN_CONTENT, PLAN_OUTLINE_CONTENT, PLAN_QUESTIONS,
PLAN_QUESTIONS_RECAP, PLAN_GRAPH_NODES, PLAN_SLICE_CARDS.

The placeholder marker is replaced with the content. If the placeholder
has already been replaced, the command has no effect (won't duplicate).

Example:
  htmlgraph plan set-section plan-my-feature PLAN_OUTLINE_CONTENT '<h4>Helpers</h4><pre><code>func ErrNotFound(kind, id string) error</code></pre>'`,
		Args: cobra.ExactArgs(3),
		RunE: func(_ *cobra.Command, args []string) error {
			fmt.Fprintln(os.Stderr, "⚠ Deprecated: use 'plan set-design-yaml' for YAML plans")
			return runPlanSetSection(args[0], args[1], args[2])
		},
	}
}

func runPlanSetSection(planID, placeholder, content string) error {
	htmlgraphDir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}

	planPath := filepath.Join(htmlgraphDir, "plans", planID+".html")
	data, err := os.ReadFile(planPath)
	if err != nil {
		return fmt.Errorf("plan %q not found: %w", planID, err)
	}

	marker := "<!--" + strings.TrimSpace(placeholder) + "-->"
	fileContent := string(data)
	if !strings.Contains(fileContent, marker) {
		return fmt.Errorf("placeholder %s not found in plan (already replaced or misspelled)", marker)
	}

	fileContent = strings.Replace(fileContent, marker, content+"\n    "+marker, 1)
	if err := os.WriteFile(planPath, []byte(fileContent), 0o644); err != nil {
		return err
	}

	fmt.Printf("Set %s in %s\n", placeholder, planID)
	return nil
}

// ---- plan set-slice ---------------------------------------------------------

func planSetSliceCmd() *cobra.Command {
	var tests, deps, files string
	cmd := &cobra.Command{
		Use:   "set-slice <plan-id> <slice-number>",
		Short: "(deprecated) Update a slice's test strategy, dependencies, and files",
		Long: `Update a vertical slice's metadata in a plan.

Example:
  htmlgraph plan set-slice plan-my-feature 1 \
    --tests "Unit: ErrNotFound returns correct format. Integration: resolveID failure includes hint." \
    --deps "none (foundation slice)" \
    --files "internal/workitem/errors.go, internal/workitem/resolve.go"`,
		Args: cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			return runPlanSetSlice(args[0], args[1], tests, deps, files)
		},
	}
	cmd.Flags().StringVar(&tests, "tests", "", "test strategy (rendered as list items)")
	cmd.Flags().StringVar(&deps, "deps", "", "dependency description")
	cmd.Flags().StringVar(&files, "files", "", "affected files (comma-separated)")
	return cmd
}

func runPlanSetSlice(planID, sliceNum, tests, deps, files string) error {
	htmlgraphDir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}

	planPath := filepath.Join(htmlgraphDir, "plans", planID+".html")
	data, err := os.ReadFile(planPath)
	if err != nil {
		return fmt.Errorf("plan %q not found: %w", planID, err)
	}
	content := string(data)

	// Find the slice card by data-slice="N".
	sliceMarker := fmt.Sprintf(`data-slice="%s"`, sliceNum)
	if !strings.Contains(content, sliceMarker) {
		return fmt.Errorf("slice %s not found in plan", sliceNum)
	}

	if tests != "" {
		// Find the <ul> after "Test Strategy" within this slice and replace.
		sliceIdx := strings.Index(content, sliceMarker)
		afterSlice := content[sliceIdx:]
		testH4 := strings.Index(afterSlice, "<h4>Test Strategy</h4>")
		if testH4 >= 0 {
			afterH4 := afterSlice[testH4:]
			ulStart := strings.Index(afterH4, "<ul>")
			ulEnd := strings.Index(afterH4, "</ul>")
			if ulStart >= 0 && ulEnd >= 0 {
				var listItems strings.Builder
				for _, t := range strings.Split(tests, ".") {
					t = strings.TrimSpace(t)
					if t != "" {
						listItems.WriteString("<li>" + html.EscapeString(t) + "</li>")
					}
				}
				absStart := sliceIdx + testH4 + ulStart
				absEnd := sliceIdx + testH4 + ulEnd + len("</ul>")
				content = content[:absStart] + "<ul>" + listItems.String() + "</ul>" + content[absEnd:]
			}
		}
	}

	if deps != "" {
		// Replace "Dependencies: none" with actual text.
		sliceIdx := strings.Index(content, sliceMarker)
		afterSlice := content[sliceIdx:]
		depIdx := strings.Index(afterSlice, "Dependencies: ")
		if depIdx >= 0 {
			pEnd := strings.Index(afterSlice[depIdx:], "</p>")
			if pEnd >= 0 {
				absStart := sliceIdx + depIdx + len("Dependencies: ")
				absEnd := sliceIdx + depIdx + pEnd
				content = content[:absStart] + html.EscapeString(deps) + content[absEnd:]
			}
		}
	}

	if files != "" {
		// Replace the slice-meta span content.
		sliceIdx := strings.Index(content, sliceMarker)
		afterSlice := content[sliceIdx:]
		metaIdx := strings.Index(afterSlice, `class="slice-meta"`)
		if metaIdx >= 0 {
			spanStart := strings.Index(afterSlice[metaIdx:], "<span>")
			spanEnd := strings.Index(afterSlice[metaIdx:], "</span>")
			if spanStart >= 0 && spanEnd >= 0 {
				absStart := sliceIdx + metaIdx + spanStart + len("<span>")
				absEnd := sliceIdx + metaIdx + spanEnd
				var fileSB strings.Builder
				fileSB.WriteString("Files: ")
				for i, f := range strings.Split(files, ",") {
					f = strings.TrimSpace(f)
					if i > 0 {
						fileSB.WriteString(", ")
					}
					fileSB.WriteString("<code>" + html.EscapeString(f) + "</code>")
				}
				fileHTML := fileSB.String()
				content = content[:absStart] + fileHTML + content[absEnd:]
			}
		}
	}

	if err := os.WriteFile(planPath, []byte(content), 0o644); err != nil {
		return err
	}

	fmt.Printf("Updated slice %s in %s\n", sliceNum, planID)
	return nil
}

// ---- plan add-question ------------------------------------------------------

func planAddQuestionCmd() *cobra.Command {
	var desc, options string
	cmd := &cobra.Command{
		Use:   `add-question <plan-id> <question> --options "opt1:explanation1,opt2:explanation2"`,
		Short: "(deprecated) Add a design question to a plan",
		Long: `Add an interactive question to a plan's design section.

Each option has a short label and an explanation separated by ":".
The first option is selected by default.

Example:
  htmlgraph plan add-question plan-my-feature "Error message length?" \
    --options "one-line:Keep hints to a single sentence after the error,two-line:Allow a second line with more context and examples" \
    --description "Longer messages give agents more guidance but consume more context tokens."`,
		Args: cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			return runPlanAddQuestion(args[0], args[1], desc, options)
		},
	}
	cmd.Flags().StringVar(&desc, "description", "", "explanation of why this question matters")
	cmd.Flags().StringVar(&options, "options", "", `comma-separated "value:explanation" pairs`)
	_ = cmd.MarkFlagRequired("options")
	return cmd
}

// planQuestionOption is one radio choice with a label and explanation.
type planQuestionOption struct {
	Value       string
	Explanation string
}

func parsePlanOptions(raw string) []planQuestionOption {
	var opts []planQuestionOption
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		val, expl, _ := strings.Cut(part, ":")
		opts = append(opts, planQuestionOption{Value: strings.TrimSpace(val), Explanation: strings.TrimSpace(expl)})
	}
	return opts
}

func runPlanAddQuestion(planID, question, description, optionsRaw string) error {
	htmlgraphDir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}

	planPath := filepath.Join(htmlgraphDir, "plans", planID+".html")
	if _, err := os.Stat(planPath); err != nil {
		return fmt.Errorf("plan %q not found at %s", planID, planPath)
	}

	opts := parsePlanOptions(optionsRaw)
	if len(opts) == 0 {
		return fmt.Errorf("at least one option required (format: value:explanation)")
	}

	// Build a kebab-case question ID from the question text.
	qID := slugify(question)
	radioName := "q-" + qID

	// Build question block HTML.
	var qHTML strings.Builder
	fmt.Fprintf(&qHTML, `      <div class="question-block" data-question="%s" data-status="pending">`+"\n", html.EscapeString(qID))
	fmt.Fprintf(&qHTML, "        <p><strong>%s</strong></p>\n", html.EscapeString(question))
	if description != "" {
		fmt.Fprintf(&qHTML, `        <p style="font-size:.85rem;color:var(--text-dim);margin-bottom:8px">%s</p>`+"\n", html.EscapeString(description))
	}
	for i, opt := range opts {
		checked := ""
		if i == 0 {
			checked = " checked"
		}
		label := html.EscapeString(opt.Value)
		if opt.Explanation != "" {
			label += fmt.Sprintf(` <span style="color:var(--text-muted);font-size:.85rem">&mdash; %s</span>`, html.EscapeString(opt.Explanation))
		}
		fmt.Fprintf(&qHTML, `        <label><input type="radio" name="%s" value="%s"%s data-question="%s"> %s</label>`+"\n",
			html.EscapeString(radioName), html.EscapeString(opt.Value), checked, html.EscapeString(qID), label)
	}
	qHTML.WriteString("      </div>\n")

	// Build recap row HTML.
	recapHTML := fmt.Sprintf(
		`        <tr data-recap-for="%s"><td>%s</td><td class="recap-answer">%s</td><td><span class="badge badge-pending">Pending</span></td></tr>`,
		html.EscapeString(qID), html.EscapeString(question), html.EscapeString(opts[0].Value))

	// Read the plan file and inject.
	data, err := os.ReadFile(planPath)
	if err != nil {
		return err
	}
	content := string(data)

	// Insert question block — append before the PLAN_QUESTIONS marker or at end of questions section.
	if strings.Contains(content, "<!--PLAN_QUESTIONS-->") {
		content = strings.Replace(content, "<!--PLAN_QUESTIONS-->", qHTML.String()+"      <!--PLAN_QUESTIONS-->", 1)
	} else {
		// Fallback: insert before the design approval row.
		content = strings.Replace(content, `<div class="approval-row">`, qHTML.String()+`      <div class="approval-row">`, 1)
	}

	// Insert recap row (idempotent — skip if already exists for this question).
	recapAttr := fmt.Sprintf(`data-recap-for="%s"`, html.EscapeString(qID))
	if !strings.Contains(content, recapAttr) {
		content = strings.Replace(content, "<!--PLAN_QUESTIONS_RECAP-->", recapHTML+"\n        <!--PLAN_QUESTIONS_RECAP-->", 1)
	}

	if err := os.WriteFile(planPath, []byte(content), 0o644); err != nil {
		return err
	}

	fmt.Printf("Added question: %s (%d options)\n", question, len(opts))
	return nil
}

// slugify converts text to a kebab-case string suitable for HTML IDs.
// Used for question IDs within plans where human-readability matters.
func slugify(text string) string {
	if text == "" {
		return "untitled"
	}
	slug := strings.ToLower(text)
	var b strings.Builder
	prevDash := false
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevDash = false
		} else if !prevDash {
			b.WriteRune('-')
			prevDash = true
		}
	}
	result := strings.Trim(b.String(), "-")
	if len(result) > 40 {
		truncated := result[:40]
		if lastHyphen := strings.LastIndex(truncated, "-"); lastHyphen > 0 {
			truncated = truncated[:lastHyphen]
		}
		result = truncated
	}
	return result
}
