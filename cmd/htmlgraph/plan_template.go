package main

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"

	dbpkg "github.com/shakestzd/htmlgraph/internal/db"
	"github.com/shakestzd/htmlgraph/internal/htmlparse"
)

// ---- plan template types & helpers -----------------------------------------

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
func extractPlanNodeInfo(rawHTML string) planNodeInfo {
	info := planNodeInfo{}

	if start := strings.Index(rawHTML, "<h1>"); start >= 0 {
		rest := rawHTML[start+4:]
		if end := strings.Index(rest, "</h1>"); end >= 0 {
			info.title = strings.TrimSpace(rest[:end])
		}
	}

	// Try data-section="description" first, fall back to first <p> after </header>.
	if s := strings.Index(rawHTML, `data-section="description"`); s >= 0 {
		rest := rawHTML[s:]
		if p := strings.Index(rest, "<p>"); p >= 0 {
			rest2 := rest[p+3:]
			if e := strings.Index(rest2, "</p>"); e >= 0 {
				info.description = strings.TrimSpace(rest2[:e])
			}
		}
	} else if headerEnd := strings.Index(rawHTML, "</header>"); headerEnd >= 0 {
		rest := rawHTML[headerEnd:]
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
	PlanID         string
	FeatureID      string
	Title          string
	Description    string
	Date           string
	GraphNodes     string // HTML for <!--PLAN_GRAPH_NODES-->
	SliceCards     string // HTML for <!--PLAN_SLICE_CARDS-->
	SectionsJSON   string // JS array literal for /*PLAN_SECTIONS_JSON*/
	TotalSections  string // integer string for <!--PLAN_TOTAL_SECTIONS-->
	DesignContent  string // HTML for <!--PLAN_DESIGN_CONTENT-->
	OutlineContent string // HTML for <!--PLAN_OUTLINE_CONTENT-->
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

	if v.DesignContent != "" {
		tmpl = strings.ReplaceAll(tmpl, "<!--PLAN_DESIGN_CONTENT-->", v.DesignContent+"\n    <!--PLAN_DESIGN_CONTENT-->")
	}
	if v.OutlineContent != "" {
		tmpl = strings.ReplaceAll(tmpl, "<!--PLAN_OUTLINE_CONTENT-->", v.OutlineContent+"\n    <!--PLAN_OUTLINE_CONTENT-->")
	}

	// Always populate PLAN_META regardless of whether slices exist.
	sliceCount := strings.Count(v.SectionsJSON, `"slice-`)
	meta := fmt.Sprintf("%d slices &middot; Created %s", sliceCount, v.Date)
	tmpl = strings.ReplaceAll(tmpl, "<!--PLAN_META-->", meta)

	if v.GraphNodes != "" {
		tmpl = strings.ReplaceAll(tmpl, "<!--PLAN_GRAPH_NODES-->", v.GraphNodes+"    <!--PLAN_GRAPH_NODES-->")
	}
	if v.SliceCards != "" {
		tmpl = strings.ReplaceAll(tmpl, "<!--PLAN_SLICE_CARDS-->", v.SliceCards+"    <!--PLAN_SLICE_CARDS-->")
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
		filesAttr := ""
		if fc := featureFiles[f.num]; fc > 0 {
			filesAttr = fmt.Sprintf(` data-files="%d"`, fc)
		}
		gnBuf.WriteString(fmt.Sprintf(
			`    <div data-node="%d" data-name="%s" data-status="pending" data-deps="%s"%s></div>`+"\n",
			f.num, html.EscapeString(f.title), featureDeps[f.num], filesAttr,
		))
	}
	graphNodes = gnBuf.String()

	// Build slice cards HTML.
	var scBuf strings.Builder
	for _, f := range features {
		n := f.num
		files := featureFiles[n]
		filesAttr := ""
		if files > 0 {
			filesAttr = fmt.Sprintf(` data-files="%d"`, files)
		}
		scBuf.WriteString(fmt.Sprintf(
			`    <div class="slice-card" data-slice="%d" data-slice-name="%s" data-status="pending"%s>`+"\n",
			n, html.EscapeString(f.id), filesAttr,
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

// buildDesignContent generates the Design Discussion section from the source
// work item's description and a summary of contained features.
// The <!--PLAN_DESIGN_CONTENT--> marker appears first so that manually-set
// content (via plan set-section) renders above the auto-generated scope.
// The feature list is wrapped in a collapsible <details> grouped by status.
func buildDesignContent(info planNodeInfo, nodePath, htmlgraphDir string) string {
	var b strings.Builder
	if info.description != "" {
		b.WriteString(fmt.Sprintf("    <p>%s</p>\n", html.EscapeString(info.description)))
	}

	// Marker goes first — manual content injected here appears above the scope.
	b.WriteString("    <!--PLAN_DESIGN_CONTENT-->\n")

	node, err := htmlparse.ParseFile(nodePath)
	if err != nil || len(node.Edges["contains"]) == 0 {
		return b.String()
	}

	// Classify features by status.
	type scopeItem struct {
		title, desc, status string
	}
	var done, todo []scopeItem
	for _, edge := range node.Edges["contains"] {
		title := strings.TrimSpace(edge.Title)
		if title == "" {
			title = edge.TargetID
		}
		// Skip plan features (meta-noise).
		if strings.HasPrefix(title, "Plan:") || strings.HasPrefix(edge.TargetID, "plan-") {
			continue
		}

		item := scopeItem{title: title, status: "todo"}
		if childPath := resolveNodePath(htmlgraphDir, edge.TargetID); childPath != "" {
			if child, err := htmlparse.ParseFile(childPath); err == nil {
				item.status = string(child.Status)
				if child.Content != "" {
					desc := strings.ReplaceAll(child.Content, "<p>", "")
					desc = strings.ReplaceAll(desc, "</p>", "")
					desc = strings.TrimSpace(desc)
					if len(desc) > 120 {
						desc = desc[:117] + "..."
					}
					item.desc = desc
				}
			}
		}
		if item.status == "done" {
			done = append(done, item)
		} else {
			todo = append(todo, item)
		}
	}

	total := len(done) + len(todo)
	b.WriteString(fmt.Sprintf(
		"    <details style=\"margin-top:12px\"><summary style=\"cursor:pointer;font-size:.85rem;color:var(--text-dim)\">"+
			"Track Features (%d total, %d done, %d remaining)</summary>\n",
		total, len(done), len(todo)))

	if len(todo) > 0 {
		b.WriteString("    <h4 style=\"margin-top:8px\">Remaining</h4>\n    <ul>\n")
		for _, it := range todo {
			if it.desc != "" {
				b.WriteString(fmt.Sprintf("      <li><strong>%s</strong> &mdash; %s</li>\n",
					html.EscapeString(it.title), html.EscapeString(it.desc)))
			} else {
				b.WriteString(fmt.Sprintf("      <li>%s</li>\n", html.EscapeString(it.title)))
			}
		}
		b.WriteString("    </ul>\n")
	}
	if len(done) > 0 {
		b.WriteString("    <h4 style=\"margin-top:8px\">Completed</h4>\n    <ul style=\"color:var(--text-muted)\">\n")
		for _, it := range done {
			b.WriteString(fmt.Sprintf("      <li>&#10003; %s</li>\n", html.EscapeString(it.title)))
		}
		b.WriteString("    </ul>\n")
	}

	b.WriteString("    </details>\n")
	return b.String()
}

// buildOutlineContent generates the Structure Outline section showing
// the dependency chain and execution order.
func buildOutlineContent(nodePath, htmlgraphDir string) string {
	node, err := htmlparse.ParseFile(nodePath)
	if err != nil || len(node.Edges["contains"]) == 0 {
		return ""
	}

	// Build ID → title map and find dependencies.
	type item struct {
		id, title string
		deps      []string
	}
	var items []item
	for _, edge := range node.Edges["contains"] {
		title := edge.Title
		if title == "" {
			title = edge.TargetID
		}
		it := item{id: edge.TargetID, title: title}
		if childPath := resolveNodePath(htmlgraphDir, edge.TargetID); childPath != "" {
			if child, err := htmlparse.ParseFile(childPath); err == nil {
				for _, dep := range child.Edges["blocked_by"] {
					it.deps = append(it.deps, dep.TargetID)
				}
			}
		}
		items = append(items, it)
	}

	// Separate into independent (no deps) and dependent.
	var independent, dependent []item
	for _, it := range items {
		if len(it.deps) == 0 {
			independent = append(independent, it)
		} else {
			dependent = append(dependent, it)
		}
	}

	var b strings.Builder
	if len(independent) > 0 {
		b.WriteString("    <h4>Independent (can run in parallel)</h4>\n    <ul>\n")
		for _, it := range independent {
			b.WriteString(fmt.Sprintf("      <li>%s</li>\n", html.EscapeString(it.title)))
		}
		b.WriteString("    </ul>\n")
	}
	if len(dependent) > 0 {
		idToTitle := make(map[string]string, len(items))
		for _, it := range items {
			idToTitle[it.id] = it.title
		}
		b.WriteString("    <h4>Sequential (has dependencies)</h4>\n    <ul>\n")
		for _, it := range dependent {
			var depNames []string
			for _, d := range it.deps {
				if name, ok := idToTitle[d]; ok {
					depNames = append(depNames, name)
				}
			}
			b.WriteString(fmt.Sprintf("      <li>%s &larr; depends on: %s</li>\n",
				html.EscapeString(it.title), html.EscapeString(strings.Join(depNames, ", "))))
		}
		b.WriteString("    </ul>\n")
	}
	return b.String()
}


