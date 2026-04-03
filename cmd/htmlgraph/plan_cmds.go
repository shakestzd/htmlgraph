package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	dbpkg "github.com/shakestzd/htmlgraph/internal/db"
	"github.com/shakestzd/htmlgraph/internal/htmlparse"
	"github.com/spf13/cobra"
)

//go:embed templates/plan-template.html
var planTemplateFS embed.FS

// planCmdWithExtras builds the standard workitem commands for plans,
// then adds CRISPI-specific subcommands: generate, open, wait, read-feedback.
func planCmdWithExtras() *cobra.Command {
	cmd := workitemCmd("plan", "plans")
	cmd.AddCommand(planGenerateCmd())
	cmd.AddCommand(planOpenCmd())
	cmd.AddCommand(planWaitCmd())
	cmd.AddCommand(planReadFeedbackCmd())
	cmd.AddCommand(planAddQuestionCmd())
	cmd.AddCommand(planSetSectionCmd())
	cmd.AddCommand(planSetSliceCmd())
	return cmd
}

// planGenerateCmd scaffolds a plan HTML file from a feature or track ID.
func planGenerateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "generate <feature-or-track-id>",
		Short: "Scaffold a plan HTML file from a feature or track",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runPlanGenerate(args[0])
		},
	}
}

func runPlanGenerate(sourceID string) error {
	htmlgraphDir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}

	resolved, err := resolveID(htmlgraphDir, sourceID)
	if err != nil {
		return fmt.Errorf("resolve %s: %w", sourceID, err)
	}
	nodePath := resolveNodePath(htmlgraphDir, resolved)
	if nodePath == "" {
		return fmt.Errorf("work item %q not found", resolved)
	}

	info, err := parseNodeForPlan(nodePath)
	if err != nil {
		return fmt.Errorf("parse work item: %w", err)
	}

	planID := derivePlanID(info.title)
	plansDir := filepath.Join(htmlgraphDir, "plans")
	if err := os.MkdirAll(plansDir, 0o755); err != nil {
		return fmt.Errorf("create plans dir: %w", err)
	}
	outPath := filepath.Join(plansDir, planID+".html")

	tmplData, err := planTemplateFS.ReadFile("templates/plan-template.html")
	if err != nil {
		return fmt.Errorf("read plan template: %w", err)
	}

	graphNodes, sliceCards, sectionsJSON, totalSections := buildPlanSections(nodePath, htmlgraphDir)

	content := applyPlanTemplateVars(string(tmplData), planTemplateVars{
		PlanID:        planID,
		FeatureID:     resolved,
		Title:         info.title,
		Description:   info.description,
		Date:          time.Now().UTC().Format("2006-01-02"),
		GraphNodes:    graphNodes,
		SliceCards:    sliceCards,
		SectionsJSON:  sectionsJSON,
		TotalSections: totalSections,
	})

	if err := os.WriteFile(outPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write plan: %w", err)
	}

	fmt.Println(outPath)
	return nil
}

// planOpenCmd opens a plan in the browser.
func planOpenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "open <plan-id>",
		Short: "Open a plan in the browser",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runPlanOpen(args[0])
		},
	}
}

func runPlanOpen(planID string) error {
	htmlgraphDir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}

	planPath := filepath.Join(htmlgraphDir, "plans", planID+".html")
	if _, err := os.Stat(planPath); err != nil {
		return fmt.Errorf("plan %q not found at %s", planID, planPath)
	}

	if !isServerRunning("http://localhost:8080") {
		// Auto-start server so plan feedback API works.
		cmd := exec.Command(os.Args[0], "serve", "-p", "8080")
		cmd.Stdout = nil
		cmd.Stderr = nil
		_ = cmd.Start()
		time.Sleep(500 * time.Millisecond)
	}

	url := "http://localhost:8080/plans/" + planID + ".html"
	return openBrowser(url)
}

// planWaitCmd blocks until a plan is finalized.
func planWaitCmd() *cobra.Command {
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:   "wait <plan-id>",
		Short: "Block until a plan is finalized",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runPlanWait(args[0], timeout)
		},
	}
	cmd.Flags().DurationVar(&timeout, "timeout", time.Hour, "Maximum wait time (e.g. 30m, 1h)")
	return cmd
}

func runPlanWait(planID string, timeout time.Duration) error {
	htmlgraphDir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	fmt.Printf("Waiting for plan %s to be finalized", planID)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println()
			return fmt.Errorf("timeout: plan %s was not finalized within %s", planID, timeout)
		case <-ticker.C:
			finalized, err := checkPlanFinalized(htmlgraphDir, planID)
			if err != nil {
				fmt.Print(".")
				continue
			}
			if finalized {
				fmt.Println("\nPlan finalized.")
				return nil
			}
			fmt.Print(".")
		}
	}
}

// checkPlanFinalized returns true when the plan's status is "finalized".
// Prefers the live API; falls back to reading the HTML file directly.
func checkPlanFinalized(htmlgraphDir, planID string) (bool, error) {
	if isServerRunning("http://localhost:8080") {
		status, err := fetchPlanStatusFromAPI(planID)
		if err == nil {
			return status == "finalized", nil
		}
	}

	planPath := filepath.Join(htmlgraphDir, "plans", planID+".html")
	status, err := parsePlanHTMLStatus(planPath)
	if err != nil {
		return false, err
	}
	return status == "finalized", nil
}

// fetchPlanStatusFromAPI calls GET /api/plans/{id}/status and returns the status.
func fetchPlanStatusFromAPI(planID string) (string, error) {
	url := "http://localhost:8080/api/plans/" + planID + "/status"
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url) //nolint:gosec,noctx
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned %d", resp.StatusCode)
	}
	var result struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.Status, nil
}

// ---- plan template helpers --------------------------------------------------

type planNodeInfo struct {
	title       string
	description string
}

// parseNodeForPlan reads a work item HTML file and returns its title and description.
func parseNodeForPlan(nodePath string) (planNodeInfo, error) {
	data, err := os.ReadFile(nodePath)
	if err != nil {
		return planNodeInfo{}, err
	}
	return extractPlanNodeInfo(string(data)), nil
}

// extractPlanNodeInfo extracts title and description from raw HTML using
// simple string scanning — keeps this file free of goquery import.
func extractPlanNodeInfo(html string) planNodeInfo {
	info := planNodeInfo{}

	if start := strings.Index(html, "<h1>"); start >= 0 {
		rest := html[start+4:]
		if end := strings.Index(rest, "</h1>"); end >= 0 {
			info.title = strings.TrimSpace(rest[:end])
		}
	}

	// Try data-section="description" first, fall back to first <p> after </header>.
	if s := strings.Index(html, `data-section="description"`); s >= 0 {
		rest := html[s:]
		if p := strings.Index(rest, "<p>"); p >= 0 {
			rest2 := rest[p+3:]
			if e := strings.Index(rest2, "</p>"); e >= 0 {
				info.description = strings.TrimSpace(rest2[:e])
			}
		}
	} else if headerEnd := strings.Index(html, "</header>"); headerEnd >= 0 {
		rest := html[headerEnd:]
		pIdx := strings.Index(rest, "<p>")
		navIdx := strings.Index(rest, "<nav")
		if pIdx >= 0 && (navIdx < 0 || pIdx < navIdx) {
			rest2 := rest[pIdx+3:]
			if e := strings.Index(rest2, "</p>"); e >= 0 {
				info.description = strings.TrimSpace(rest2[:e])
			}
		}
	}

	return info
}

type planTemplateVars struct {
	PlanID        string
	FeatureID     string
	Title         string
	Description   string
	Date          string
	GraphNodes    string // HTML for <!--PLAN_GRAPH_NODES-->
	SliceCards    string // HTML for <!--PLAN_SLICE_CARDS-->
	SectionsJSON  string // JS array literal for /*PLAN_SECTIONS_JSON*/
	TotalSections string // integer string for <!--PLAN_TOTAL_SECTIONS-->
}

// applyPlanTemplateVars replaces placeholder values in the template HTML
// with real values from the source work item.
func applyPlanTemplateVars(tmpl string, v planTemplateVars) string {
	tmpl = strings.ReplaceAll(tmpl, "plan-webhook-support", v.PlanID)
	tmpl = strings.ReplaceAll(tmpl, "feat-xxx", v.FeatureID)
	// Ensure article uses id= (htmlparse expects it), fixing any stale data-plan-id= from cache.
	tmpl = strings.Replace(tmpl, `data-plan-id="`+v.PlanID+`"`, `id="`+v.PlanID+`"`, 1)
	tmpl = strings.ReplaceAll(tmpl, "Plan: Webhook Support", "Plan: "+v.Title)
	tmpl = strings.ReplaceAll(tmpl, "Webhook Support", v.Title)

	const sampleDesc = "HTTP POST notifications for HtmlGraph events with retry and config management."
	if v.Description != "" {
		tmpl = strings.ReplaceAll(tmpl, sampleDesc, v.Description)
	} else {
		tmpl = strings.ReplaceAll(tmpl, sampleDesc, "")
	}

	tmpl = strings.ReplaceAll(tmpl, "2026-04-01", v.Date)

	// Always populate PLAN_META regardless of whether slices exist.
	sliceCount := strings.Count(v.SectionsJSON, `"slice-`)
	meta := fmt.Sprintf("%d slices &middot; Created %s", sliceCount, v.Date)
	tmpl = strings.ReplaceAll(tmpl, "<!--PLAN_META-->", meta)

	if v.GraphNodes != "" {
		tmpl = strings.ReplaceAll(tmpl, "<!--PLAN_GRAPH_NODES-->", v.GraphNodes)
	}
	if v.SliceCards != "" {
		tmpl = strings.ReplaceAll(tmpl, "<!--PLAN_SLICE_CARDS-->", v.SliceCards)
	}
	if v.TotalSections != "" {
		tmpl = strings.ReplaceAll(tmpl, "<!--PLAN_TOTAL_SECTIONS-->", v.TotalSections)
	}
	if v.SectionsJSON != "" {
		// Replace the default array between the JS markers.
		const start = "/*PLAN_SECTIONS_JSON*/"
		const end = "/*END_PLAN_SECTIONS_JSON*/"
		if si := strings.Index(tmpl, start); si >= 0 {
			if ei := strings.Index(tmpl[si:], end); ei >= 0 {
				tmpl = tmpl[:si+len(start)] + v.SectionsJSON + tmpl[si+ei:]
			}
		}
	}

	return tmpl
}

// planFeature holds the data needed to render one slice card and graph node.
type planFeature struct {
	num   int
	id    string
	title string
}

// buildPlanSections parses the source node for "contains" edges and generates
// graph node HTML, slice card HTML, sections JSON, and total section count.
// Falls back to empty strings (leaving template placeholders intact) on any error.
func buildPlanSections(nodePath, htmlgraphDir string) (graphNodes, sliceCards, sectionsJSON, totalSections string) {
	node, err := htmlparse.ParseFile(nodePath)
	if err != nil {
		return
	}

	containsEdges := node.Edges["contains"]
	if len(containsEdges) == 0 {
		return
	}

	// Build feature list and an ID→node-number index for dependency resolution.
	idToNum := make(map[string]int, len(containsEdges))
	features := make([]planFeature, 0, len(containsEdges))
	for i, edge := range containsEdges {
		title := strings.TrimSpace(edge.Title)
		if title == "" {
			title = edge.TargetID
		}
		if len(title) > 60 {
			title = title[:57] + "..."
		}
		num := i + 1
		idToNum[edge.TargetID] = num
		features = append(features, planFeature{num: num, id: edge.TargetID, title: title})
	}

	// Open SQLite for file count queries (best-effort).
	database, dbErr := dbpkg.Open(filepath.Join(htmlgraphDir, "htmlgraph.db"))
	if dbErr == nil {
		defer database.Close()
	}

	// Read each child feature's blocked_by edges, description, and file count.
	featureDeps := make(map[int]string, len(features))
	featureDescs := make(map[int]string, len(features))
	featureFiles := make(map[int]int, len(features))
	for _, f := range features {
		childPath := resolveNodePath(htmlgraphDir, f.id)
		if childPath == "" {
			continue
		}
		childNode, err := htmlparse.ParseFile(childPath)
		if err != nil {
			continue
		}
		if childNode.Content != "" {
			desc := childNode.Content
			// Strip HTML tags for plain text display.
			desc = strings.ReplaceAll(desc, "<p>", "")
			desc = strings.ReplaceAll(desc, "</p>", "")
			desc = strings.TrimSpace(desc)
			if len(desc) > 200 {
				desc = desc[:197] + "..."
			}
			featureDescs[f.num] = desc
		}
		// Query actual file count from SQLite feature_files table.
		if database != nil {
			if count, err := dbpkg.CountFilesByFeature(database, f.id); err == nil {
				featureFiles[f.num] = count
			}
		}
		var depNums []string
		for _, blockedEdge := range childNode.Edges["blocked_by"] {
			if num, ok := idToNum[blockedEdge.TargetID]; ok {
				depNums = append(depNums, fmt.Sprintf("%d", num))
			}
		}
		if len(depNums) > 0 {
			featureDeps[f.num] = strings.Join(depNums, ",")
		}
	}

	// Build graph nodes HTML.
	var gnBuf strings.Builder
	for _, f := range features {
		gnBuf.WriteString(fmt.Sprintf(
			`    <div data-node="%d" data-name="%s" data-status="pending" data-deps="%s" data-files="%d"></div>`+"\n",
			f.num, html.EscapeString(f.title), featureDeps[f.num], featureFiles[f.num],
		))
	}
	graphNodes = gnBuf.String()

	// Build slice cards HTML.
	var scBuf strings.Builder
	for _, f := range features {
		n := f.num
		files := featureFiles[n]
		scBuf.WriteString(fmt.Sprintf(
			`    <div class="slice-card" data-slice="%d" data-slice-name="%s" data-status="pending" data-files="%d">`+"\n",
			n, html.EscapeString(f.id), files,
		))
		scBuf.WriteString(fmt.Sprintf(
			`      <div class="slice-header"><span class="slice-num">#%d</span><span class="slice-name">%s</span><span class="badge badge-pending" data-badge-for="slice-%d">Pending</span></div>`+"\n",
			n, html.EscapeString(f.title), n,
		))
		scBuf.WriteString(fmt.Sprintf(
			`      <div class="slice-meta"><span>%s</span></div>`+"\n",
			html.EscapeString(f.id),
		))
		// Show description if available.
		if desc := featureDescs[n]; desc != "" {
			scBuf.WriteString(fmt.Sprintf(
				`      <p style="font-size:.9rem;margin:8px 0">%s</p>`+"\n",
				html.EscapeString(desc),
			))
		}
		scBuf.WriteString("      <h4>Test Strategy</h4>\n")
		scBuf.WriteString("      <ul><li>Add test strategy here</li></ul>\n")
		// Show real dependency labels.
		depStr := "none"
		if d := featureDeps[n]; d != "" {
			depStr = "slices " + d
		}
		scBuf.WriteString(fmt.Sprintf(
			`      <p style="font-size:.8rem;color:var(--text-muted);margin-top:6px">Dependencies: %s</p>`+"\n",
			depStr,
		))
		scBuf.WriteString(fmt.Sprintf(
			`      <div class="approval-row"><label><input type="checkbox" data-section="slice-%d" data-action="approve"> Approve slice</label><textarea data-section="slice-%d" data-comment-for="slice-%d" placeholder="Comments on slice %d..."></textarea></div>`+"\n",
			n, n, n, n,
		))
		scBuf.WriteString("    </div>\n")
	}
	sliceCards = scBuf.String()

	// Build sections JSON array: ["design","outline","slice-1","slice-2",...]
	sections := []string{"design", "outline"}
	for _, f := range features {
		sections = append(sections, fmt.Sprintf("slice-%d", f.num))
	}
	sectionStrs := make([]string, len(sections))
	for i, s := range sections {
		sectionStrs[i] = `"` + s + `"`
	}
	sectionsJSON = "[" + strings.Join(sectionStrs, ",") + "]"
	totalSections = fmt.Sprintf("%d", len(sections))
	return
}

// derivePlanID builds a kebab-case plan file ID from the work item title.
func derivePlanID(title string) string {
	if title == "" {
		return "plan-untitled"
	}
	slug := strings.ToLower(title)
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
	return "plan-" + result
}

// ---- plan set-section -------------------------------------------------------

func planSetSectionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-section <plan-id> <placeholder> <html-content>",
		Short: "Set content for a plan section placeholder",
		Long: `Inject HTML content into a named placeholder in the plan.

Placeholders: PLAN_DESIGN_CONTENT, PLAN_OUTLINE_CONTENT, PLAN_QUESTIONS,
PLAN_QUESTIONS_RECAP, PLAN_GRAPH_NODES, PLAN_SLICE_CARDS.

The placeholder marker is replaced with the content. If the placeholder
has already been replaced, the command has no effect (won't duplicate).

Example:
  htmlgraph plan set-section plan-my-feature PLAN_OUTLINE_CONTENT '<h4>Helpers</h4><pre><code>func ErrNotFound(kind, id string) error</code></pre>'`,
		Args: cobra.ExactArgs(3),
		RunE: func(_ *cobra.Command, args []string) error {
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
		Short: "Update a slice's test strategy, dependencies, and files",
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
		// Replace the placeholder test strategy list.
		oldTests := fmt.Sprintf(`data-slice="%s"`, sliceNum)
		// Find the <ul> after "Test Strategy" within this slice.
		// Strategy: find the slice marker, then find the next <ul>...</ul> and replace.
		sliceIdx := strings.Index(content, sliceMarker)
		afterSlice := content[sliceIdx:]
		testH4 := strings.Index(afterSlice, "<h4>Test Strategy</h4>")
		if testH4 >= 0 {
			afterH4 := afterSlice[testH4:]
			ulStart := strings.Index(afterH4, "<ul>")
			ulEnd := strings.Index(afterH4, "</ul>")
			if ulStart >= 0 && ulEnd >= 0 {
				// Build new test list.
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
				_ = oldTests // suppress unused
			}
		}
	}

	if deps != "" {
		// Replace "Dependencies: none" with actual text.
		sliceIdx := strings.Index(content, sliceMarker)
		afterSlice := content[sliceIdx:]
		depIdx := strings.Index(afterSlice, "Dependencies: ")
		if depIdx >= 0 {
			// Find the end of the <p> tag.
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
				fileHTML := "Files: "
				for i, f := range strings.Split(files, ",") {
					f = strings.TrimSpace(f)
					if i > 0 {
						fileHTML += ", "
					}
					fileHTML += "<code>" + html.EscapeString(f) + "</code>"
				}
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
		Short: "Add a design question to a plan",
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
	qID := derivePlanID(question)
	qID = strings.TrimPrefix(qID, "plan-")
	radioName := "q-" + qID

	// Build question block HTML.
	var qHTML strings.Builder
	qHTML.WriteString(fmt.Sprintf(`      <div class="question-block" data-question="%s" data-status="pending">`+"\n", html.EscapeString(qID)))
	qHTML.WriteString(fmt.Sprintf("        <p><strong>%s</strong></p>\n", html.EscapeString(question)))
	if description != "" {
		qHTML.WriteString(fmt.Sprintf(`        <p style="font-size:.85rem;color:var(--text-dim);margin-bottom:8px">%s</p>`+"\n", html.EscapeString(description)))
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
		qHTML.WriteString(fmt.Sprintf(`        <label><input type="radio" name="%s" value="%s"%s data-question="%s"> %s</label>`+"\n",
			html.EscapeString(radioName), html.EscapeString(opt.Value), checked, html.EscapeString(qID), label))
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
		if strings.Contains(content, "<!--PLAN_QUESTIONS_RECAP-->") {
			content = strings.Replace(content, "<!--PLAN_QUESTIONS_RECAP-->", recapHTML+"\n        <!--PLAN_QUESTIONS_RECAP-->", 1)
		}
	}

	if err := os.WriteFile(planPath, []byte(content), 0o644); err != nil {
		return err
	}

	fmt.Printf("Added question: %s (%d options)\n", question, len(opts))
	return nil
}

// ---- browser / server helpers -----------------------------------------------

// isServerRunning returns true when a GET to baseURL succeeds within 500ms.
func isServerRunning(baseURL string) bool {
	client := &http.Client{Timeout: 500 * time.Millisecond}
	resp, err := client.Get(baseURL) //nolint:gosec,noctx
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode < 500
}

// openBrowser opens the given URL or file path in the default OS browser.
func openBrowser(target string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", target)
	case "linux":
		cmd = exec.Command("xdg-open", target)
	default:
		fmt.Println(target)
		return nil
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open browser: %w", err)
	}
	return nil
}
