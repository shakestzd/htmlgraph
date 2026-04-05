package main

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shakestzd/htmlgraph/internal/models"
	"github.com/shakestzd/htmlgraph/internal/plantmpl"
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

// createPlanFromTopic creates a plan node and scaffolds the CRISPI interactive
// template. Returns the plan ID (e.g. plan-a1b2c3d4).
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

	// Overwrite the generic node HTML with the CRISPI interactive template.
	if err := scaffoldCRISPIPlan(htmlgraphDir, node.ID, title, description); err != nil {
		return "", fmt.Errorf("scaffold CRISPI: %w", err)
	}

	return node.ID, nil
}

// scaffoldCRISPIPlan renders the CRISPI interactive plan HTML using the
// typed plantmpl package. This produces the full interactive HTML with
// dagre graphs, approval checkboxes, progress bars, and the finalize button.
func scaffoldCRISPIPlan(htmlgraphDir, planID, title, description string) error {
	plansDir := filepath.Join(htmlgraphDir, "plans")
	if err := os.MkdirAll(plansDir, 0o755); err != nil {
		return fmt.Errorf("create plans dir: %w", err)
	}

	page := plantmpl.BuildFromTopic(planID, title, description, time.Now().UTC().Format("2006-01-02"))

	outPath := filepath.Join(plansDir, planID+".html")
	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create plan file: %w", err)
	}
	defer f.Close()

	return page.Render(f)
}

// scaffoldCRISPIPlanFromNode regenerates the CRISPI template from a full node,
// including any existing slices (steps). Used by the re-scaffold path.
func scaffoldCRISPIPlanFromNode(htmlgraphDir string, node *models.Node) error {
	plansDir := filepath.Join(htmlgraphDir, "plans")
	if err := os.MkdirAll(plansDir, 0o755); err != nil {
		return fmt.Errorf("create plans dir: %w", err)
	}

	page := plantmpl.BuildFromWorkItem(
		node.ID, node.TrackID, node.Title, node.Content,
		time.Now().UTC().Format("2006-01-02"),
	)

	// Convert node steps into typed SliceCards and GraphNodes.
	for i, step := range node.Steps {
		num := i + 1
		page.Slices = append(page.Slices, plantmpl.SliceCard{
			Num:    num,
			Title:  step.Description,
			Status: "pending",
		})
		page.Graph = ensureGraph(page.Graph)
		page.Graph.Nodes = append(page.Graph.Nodes, plantmpl.GraphNode{
			Num:    num,
			Name:   step.Description,
			Status: "pending",
		})
	}

	outPath := filepath.Join(plansDir, node.ID+".html")
	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create plan file: %w", err)
	}
	defer f.Close()

	return page.Render(f)
}

// ensureGraph lazily initializes the DependencyGraph if nil.
func ensureGraph(g *plantmpl.DependencyGraph) *plantmpl.DependencyGraph {
	if g == nil {
		return &plantmpl.DependencyGraph{}
	}
	return g
}

// ---- plan add-slice ---------------------------------------------------------

// sliceFlags holds optional metadata for a new slice card.
type sliceFlags struct {
	description string
	effort      string
	risk        string
	deps        string
	files       string
}

// planAddSliceCmd adds a new vertical slice (as a step) to an existing plan.
func planAddSliceCmd() *cobra.Command {
	var sf sliceFlags
	cmd := &cobra.Command{
		Use:   "add-slice <plan-id> <title>",
		Short: "Add a vertical slice to a plan",
		Long: `Add a new slice as a step to an existing plan.

Example:
  htmlgraph plan add-slice plan-a1b2c3d4 "Implement error handling" \
    --description "Add structured error types with context hints" \
    --effort S --risk Low --deps 1,2 --files "internal/errors.go"`,
		Args: cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			htmlgraphDir, err := findHtmlgraphDir()
			if err != nil {
				return err
			}
			return addSliceToPlan(htmlgraphDir, args[0], args[1], sf)
		},
	}
	cmd.Flags().StringVar(&sf.description, "description", "", "what this slice does and why")
	cmd.Flags().StringVar(&sf.effort, "effort", "", "effort estimate: S, M, or L")
	cmd.Flags().StringVar(&sf.risk, "risk", "Low", "risk level: Low, Med, or High")
	cmd.Flags().StringVar(&sf.deps, "deps", "", "dependency slice numbers (e.g. 1,2)")
	cmd.Flags().StringVar(&sf.files, "files", "", "affected files (comma-separated)")
	return cmd
}

// addSliceToPlan injects a new slice directly into the existing CRISPI HTML.
// It counts existing data-slice elements to determine the next slice number,
// then calls injectSliceIntoCRISPI for the actual HTML mutation.
func addSliceToPlan(htmlgraphDir, planID, sliceTitle string, sf sliceFlags) error {
	planPath := filepath.Join(htmlgraphDir, "plans", planID+".html")
	data, err := os.ReadFile(planPath)
	if err != nil {
		return fmt.Errorf("plan %s not found: %w", planID, err)
	}

	// Count existing slices to determine the next slice number.
	sliceNum := countOccurrences(string(data), `data-slice="`) + 1

	if err := injectSliceIntoCRISPI(htmlgraphDir, planID, sliceNum, sliceTitle, sf); err != nil {
		return fmt.Errorf("inject slice into CRISPI %s: %w", planID, err)
	}

	fmt.Printf("Added slice #%d: %s\n", sliceNum, sliceTitle)
	return nil
}

// injectSliceIntoCRISPI adds a graph node, slice card, and updates the
// SECTIONS_JSON and PLAN_TOTAL_SECTIONS in a CRISPI plan HTML file.
func injectSliceIntoCRISPI(htmlgraphDir, planID string, sliceNum int, sliceTitle string, sf sliceFlags) error {
	planPath := filepath.Join(htmlgraphDir, "plans", planID+".html")
	data, err := os.ReadFile(planPath)
	if err != nil {
		return err
	}
	content := string(data)

	// Only operate on CRISPI files (have btn-finalize or data-zone).
	if !strings.Contains(content, "btn-finalize") && !strings.Contains(content, `data-zone=`) {
		return nil
	}

	// 1. Add graph node inside #graph-data with deps.
	graphNode := fmt.Sprintf(
		`    <div data-node="%d" data-name="%s" data-status="pending" data-deps="%s"></div>`+"\n",
		sliceNum, html.EscapeString(sliceTitle), html.EscapeString(sf.deps),
	)
	content = injectBeforeMarker(content, "<!--PLAN_GRAPH_NODES-->", graphNode)

	// 2. Add slice card inside slices section.
	sliceCard := buildSliceCardHTML(sliceNum, sliceTitle, sf)
	content = injectBeforeMarker(content, "<!--PLAN_SLICE_CARDS-->", sliceCard)

	// 3. Update PLAN_SECTIONS_JSON to include the new slice.
	content = addSliceToSectionsJSON(content, sliceNum)

	// 4. Update PLAN_TOTAL_SECTIONS count.
	content = updateTotalSections(content)

	return os.WriteFile(planPath, []byte(content), 0o644)
}

// buildSliceCardHTML produces a rich slice card matching the benchmark format:
// header with number/title/badges, meta with files, description, test strategy
// placeholder, dependency text, and approval row.
func buildSliceCardHTML(num int, title string, sf sliceFlags) string {
	var b strings.Builder

	// Opening div with data attributes.
	b.WriteString(fmt.Sprintf(
		`    <div class="slice-card" data-slice="%d" data-status="pending">`+"\n", num))

	// Header: #N, title, effort badge, risk badge, pending badge.
	b.WriteString(`      <div class="slice-header">`)
	b.WriteString(fmt.Sprintf(`<span class="slice-num">#%d</span>`, num))
	b.WriteString(fmt.Sprintf(`<span class="slice-name">%s</span>`, html.EscapeString(title)))
	if sf.effort != "" {
		b.WriteString(fmt.Sprintf(` <span class="badge badge-pending">%s</span>`, html.EscapeString(strings.ToUpper(sf.effort))))
	}
	if sf.risk != "" {
		riskClass := "pending"
		switch strings.ToLower(sf.risk) {
		case "high":
			riskClass = "blocked"
		case "med", "medium":
			riskClass = "revision"
		}
		b.WriteString(fmt.Sprintf(` <span class="badge badge-%s">%s Risk</span>`, riskClass, html.EscapeString(sf.risk)))
	}
	b.WriteString(fmt.Sprintf(`<span class="badge badge-pending" data-badge-for="slice-%d">Pending</span>`, num))
	b.WriteString("</div>\n")

	// Meta: files list.
	b.WriteString(`      <div class="slice-meta"><span>`)
	if sf.files != "" {
		b.WriteString("Files: ")
		for i, f := range strings.Split(sf.files, ",") {
			f = strings.TrimSpace(f)
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString("<code>" + html.EscapeString(f) + "</code>")
		}
	}
	b.WriteString("</span></div>\n")

	// Description.
	if sf.description != "" {
		b.WriteString(fmt.Sprintf(
			`      <p style="font-size:.9rem;margin:8px 0">%s</p>`+"\n",
			html.EscapeString(sf.description)))
	}

	// Test strategy placeholder.
	b.WriteString("      <h4>Test Strategy</h4>\n")
	b.WriteString("      <ul><li>Add test strategy here</li></ul>\n")

	// Dependencies.
	depStr := "none"
	if sf.deps != "" {
		depStr = "slices " + html.EscapeString(sf.deps)
	}
	b.WriteString(fmt.Sprintf(
		`      <p style="font-size:.8rem;color:var(--text-muted);margin-top:6px">Dependencies: %s</p>`+"\n",
		depStr))

	// Approval row.
	b.WriteString(fmt.Sprintf(
		`      <div class="approval-row"><label><input type="checkbox" data-section="slice-%d" data-action="approve"> Approve slice</label><textarea data-section="slice-%d" data-comment-for="slice-%d" placeholder="Comments on slice %d..."></textarea></div>`+"\n",
		num, num, num, num))

	b.WriteString("    </div>\n")
	return b.String()
}

// injectBeforeMarker inserts content immediately before a <!--MARKER--> comment.
// If the marker is not found, the content is not inserted.
func injectBeforeMarker(fileContent, marker, injection string) string {
	if !strings.Contains(fileContent, marker) {
		return fileContent
	}
	return strings.Replace(fileContent, marker, injection+"    "+marker, 1)
}

// addSliceToSectionsJSON appends "slice-N" to the PLAN_SECTIONS_JSON array.
func addSliceToSectionsJSON(content string, sliceNum int) string {
	const start = "/*PLAN_SECTIONS_JSON*/"
	const end = "/*END_PLAN_SECTIONS_JSON*/"

	si := strings.Index(content, start)
	if si < 0 {
		return content
	}
	rest := content[si+len(start):]
	ei := strings.Index(rest, end)
	if ei < 0 {
		return content
	}

	currentJSON := strings.TrimSpace(rest[:ei])
	newEntry := fmt.Sprintf(`"slice-%d"`, sliceNum)

	// Insert before the closing bracket.
	if idx := strings.LastIndex(currentJSON, "]"); idx >= 0 {
		currentJSON = currentJSON[:idx] + "," + newEntry + "]"
	}

	return content[:si+len(start)] + currentJSON + content[si+len(start)+ei:]
}

// updateTotalSections recalculates and updates the PLAN_TOTAL_SECTIONS count
// by counting entries in the PLAN_SECTIONS_JSON array.
func updateTotalSections(content string) string {
	const start = "/*PLAN_SECTIONS_JSON*/"
	const end = "/*END_PLAN_SECTIONS_JSON*/"

	si := strings.Index(content, start)
	if si < 0 {
		return content
	}
	rest := content[si+len(start):]
	ei := strings.Index(rest, end)
	if ei < 0 {
		return content
	}

	jsonStr := strings.TrimSpace(rest[:ei])
	// Count entries by counting quoted strings.
	count := strings.Count(jsonStr, `"`) / 2

	// Update the totalSections strong element.
	const tsMarker = `id="totalSections">`
	tsi := strings.Index(content, tsMarker)
	if tsi < 0 {
		return content
	}
	afterMarker := content[tsi+len(tsMarker):]
	closeTag := strings.Index(afterMarker, "<")
	if closeTag < 0 {
		return content
	}

	absStart := tsi + len(tsMarker)
	absEnd := absStart + closeTag
	content = content[:absStart] + fmt.Sprintf("%d", count) + content[absEnd:]

	// Also update the pendingCount strong element (same value initially).
	const pcMarker = `id="pendingCount">`
	pci := strings.Index(content, pcMarker)
	if pci >= 0 {
		afterPC := content[pci+len(pcMarker):]
		closePC := strings.Index(afterPC, "<")
		if closePC >= 0 {
			pcStart := pci + len(pcMarker)
			pcEnd := pcStart + closePC
			content = content[:pcStart] + fmt.Sprintf("%d", count) + content[pcEnd:]
		}
	}

	return content
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
