package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	dbpkg "github.com/shakestzd/htmlgraph/internal/db"
	"github.com/shakestzd/htmlgraph/internal/graph"
	"github.com/shakestzd/htmlgraph/internal/hooks"
	"github.com/shakestzd/htmlgraph/internal/htmlparse"
	"github.com/shakestzd/htmlgraph/internal/models"
	"github.com/shakestzd/htmlgraph/internal/workitem"
	"github.com/spf13/cobra"
)

// workitemCmd builds a standard CRUD command group for any work item type.
// Usage: workitemCmd("feature", "features"), workitemCmd("bug", "bugs"), etc.
func workitemCmd(typeName, dirName string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   typeName,
		Short: "Manage " + dirName,
	}
	cmd.AddCommand(wiCreateCmd(typeName, dirName))
	cmd.AddCommand(wiListCmd(typeName, dirName))
	cmd.AddCommand(wiShowCmd(typeName))
	cmd.AddCommand(wiStartCmd(typeName))
	cmd.AddCommand(wiCompleteCmd(typeName))
	cmd.AddCommand(wiDeleteCmd(typeName))
	cmd.AddCommand(wiAddStepCmd(typeName))
	cmd.AddCommand(wiUpdateCmd(typeName))
	if typeName != "track" {
		cmd.AddCommand(wiMoveCmd(typeName))
	}
	return cmd
}

func wiListCmd(typeName, dirName string) *cobra.Command {
	var statusFilter string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List " + dirName,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runWiList(dirName, statusFilter)
		},
	}
	cmd.Flags().StringVarP(&statusFilter, "status", "s", "",
		"Filter by status (todo, in-progress, blocked, done)")
	return cmd
}

func runWiList(dirName, statusFilter string) error {
	dir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}
	nodes, err := graph.LoadDir(filepath.Join(dir, dirName))
	if err != nil {
		return fmt.Errorf("load %s: %w", dirName, err)
	}

	var filtered []*models.Node
	for _, n := range nodes {
		if statusFilter != "" && string(n.Status) != statusFilter {
			continue
		}
		filtered = append(filtered, n)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].ID < filtered[j].ID
	})

	if len(filtered) == 0 {
		fmt.Printf("No %s found.\n", dirName)
		return nil
	}

	fmt.Printf("%-22s  %-11s  %-8s  %s\n", "ID", "STATUS", "PRIORITY", "TITLE")
	fmt.Println(strings.Repeat("-", 80))
	for _, n := range filtered {
		marker := "  "
		if n.Status == models.StatusInProgress {
			marker = "* "
		}
		fmt.Printf("%s%-20s  %-11s  %-8s  %s\n",
			marker, n.ID, n.Status, n.Priority, truncate(n.Title, 44))
	}
	fmt.Printf("\n%d %s\n", len(filtered), dirName)
	return nil
}

func wiShowCmd(typeName string) *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show " + typeName + " details",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runWiShow(args[0])
		},
	}
}

func runWiShow(id string) error {
	dir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}
	resolved, err := resolveID(dir, id)
	if err != nil {
		return err
	}
	path := resolveNodePath(dir, resolved)
	if path == "" {
		kind := kindFromPrefix(resolved)
		return workitem.ErrNotFoundOnDisk(kind, resolved)
	}
	node, err := htmlparse.ParseFile(path)
	if err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	printNodeDetail(node)
	return nil
}

func wiStartCmd(typeName string) *cobra.Command {
	return &cobra.Command{
		Use:   "start <id>",
		Short: "Mark a " + typeName + " as in-progress",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runWiSetStatus(typeName, args[0], "in-progress")
		},
	}
}

func wiCompleteCmd(typeName string) *cobra.Command {
	return &cobra.Command{
		Use:   "complete <id>",
		Short: "Mark a " + typeName + " as done",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runWiSetStatus(typeName, args[0], "done")
		},
	}
}

func runWiSetStatus(typeName, id, status string) error {
	dir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}
	id, err = resolveID(dir, id)
	if err != nil {
		return err
	}
	p, err := workitem.Open(dir, "claude-code")
	if err != nil {
		return fmt.Errorf("open project: %w", err)
	}
	defer p.Close()

	col := collectionFor(p, typeName)
	var node *models.Node
	switch status {
	case "in-progress":
		node, err = col.Start(id)
	case "blocked":
		err = col.Edit(id).SetStatus("blocked").Save()
		if err == nil {
			node, err = col.Get(id)
		}
	default:
		node, err = col.Complete(id)
	}
	if err != nil {
		return fmt.Errorf("set %s %s: %w", typeName, status, err)
	}

	// When starting a work item, update active_feature_id, create a claim
	// with per-agent attribution, and create an implemented_in edge.
	if status == "in-progress" {
		sessionID := hooks.EnvSessionID("")
		agentID := os.Getenv("HTMLGRAPH_AGENT_ID")
		if sessionID != "" {
			if p.DB != nil {
				_ = hooks.UpdateActiveFeature(p.DB, sessionID, id)
				claim := &models.Claim{
					ClaimID:          "clm-" + uuid.NewString()[:8],
					WorkItemID:       id,
					OwnerSessionID:   sessionID,
					OwnerAgent:       agentForClaim(),
					ClaimedByAgentID: agentID,
					Status:           models.ClaimInProgress,
				}
				_ = dbpkg.ClaimItem(p.DB, claim, 30*time.Minute)
			}
			autoImplementedInEdge(col, id, sessionID, p.DB)
		}
	}

	// Update status line cache for subagent visibility.
	if status == "in-progress" {
		WriteStatuslineCache(dir, id)
	} else {
		WriteStatuslineCache(dir, "")
	}

	verb := "Started"
	switch status {
	case "done":
		verb = "Completed"
	case "blocked":
		verb = "Blocked"
	}
	fmt.Printf("%s: %s  %s\n", verb, node.ID, node.Title)
	return nil
}

func collectionFor(p *workitem.Project, typeName string) *workitem.Collection {
	switch typeName {
	case "bug":
		return p.Bugs.Collection
	case "spike":
		return p.Spikes.Collection
	case "track":
		return p.Tracks.Collection
	case "plan":
		return p.Plans.Collection
	case "spec":
		return p.Specs.Collection
	default:
		return p.Features.Collection
	}
}

func wiDeleteCmd(typeName string) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a " + typeName,
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runWiDelete(args[0])
		},
	}
}

func runWiDelete(id string) error {
	dir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}
	resolved, err := resolveID(dir, id)
	if err != nil {
		return err
	}
	path := resolveNodePath(dir, resolved)
	if path == "" {
		kind := kindFromPrefix(resolved)
		return workitem.ErrNotFoundOnDisk(kind, resolved)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("delete %s: %w", resolved, err)
	}
	fmt.Printf("Deleted: %s\n", resolved)
	return nil
}

func wiAddStepCmd(typeName string) *cobra.Command {
	return &cobra.Command{
		Use:   "add-step <id> <description>",
		Short: "Add an implementation step to a " + typeName,
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			return runWiAddStep(typeName, args[0], args[1])
		},
	}
}

func runWiAddStep(typeName, id, description string) error {
	dir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}
	id, err = resolveID(dir, id)
	if err != nil {
		return err
	}
	p, err := workitem.Open(dir, "claude-code")
	if err != nil {
		return fmt.Errorf("open project: %w", err)
	}
	defer p.Close()

	col := collectionFor(p, typeName)
	if err := col.Edit(id).AddStep(description).Save(); err != nil {
		return fmt.Errorf("add step: %w", err)
	}
	fmt.Printf("Added step to %s: %s\n", id, description)
	return nil
}

// splitSteps splits a comma-separated steps string into trimmed non-empty parts.
func splitSteps(s string) []string {
	var steps []string
	for _, part := range strings.Split(s, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			steps = append(steps, trimmed)
		}
	}
	return steps
}

// agentForClaim returns the agent string for claim ownership.
func agentForClaim() string {
	if v := os.Getenv("HTMLGRAPH_AGENT_TYPE"); v != "" {
		return v
	}
	return "claude-code"
}

// resolveID resolves a partial or full work item ID to its canonical form.
func resolveID(htmlgraphDir, id string) (string, error) {
	return workitem.ResolvePartialID(htmlgraphDir, id)
}

// resolveNodePath searches all subdirectories for a file matching id.
func resolveNodePath(htmlgraphDir, id string) string {
	dirs := []string{"features", "bugs", "spikes", "tracks", "plans", "specs"}
	for _, sub := range dirs {
		p := filepath.Join(htmlgraphDir, sub, id+".html")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func printNodeDetail(n *models.Node) {
	sep := strings.Repeat("─", 60)
	fmt.Println(sep)
	fmt.Printf("  %s\n", n.Title)
	fmt.Println(sep)
	fmt.Printf("  ID        %s\n", n.ID)
	fmt.Printf("  Type      %s\n", n.Type)
	fmt.Printf("  Status    %s\n", n.Status)
	fmt.Printf("  Priority  %s\n", n.Priority)
	if n.TrackID != "" {
		fmt.Printf("  Track     %s\n", n.TrackID)
	}
	if !n.CreatedAt.IsZero() {
		fmt.Printf("  Created   %s\n", n.CreatedAt.Format("2006-01-02"))
	}

	if len(n.Steps) > 0 {
		done := 0
		for _, s := range n.Steps {
			if s.Completed {
				done++
			}
		}
		fmt.Printf("\nSteps: %d/%d complete\n", done, len(n.Steps))
		for _, s := range n.Steps {
			tick := "[ ]"
			if s.Completed {
				tick = "[x]"
			}
			fmt.Printf("  %s  %s\n", tick, s.Description)
		}
	}

	if len(n.Edges) > 0 {
		fmt.Println("\nEdges:")
		for rel, edges := range n.Edges {
			for _, e := range edges {
				fmt.Printf("  %-15s → %s\n", rel, e.TargetID)
			}
		}
	}

	if n.Content != "" {
		fmt.Println("\nContent:")
		for _, line := range strings.Split(n.Content, "\n") {
			fmt.Printf("  %s\n", line)
		}
	}
}

// kindFromPrefix determines the work item kind from an ID prefix.
func kindFromPrefix(id string) string {
	if strings.HasPrefix(id, "feat-") {
		return "feature"
	}
	if strings.HasPrefix(id, "bug-") {
		return "bug"
	}
	if strings.HasPrefix(id, "spk-") {
		return "spike"
	}
	if strings.HasPrefix(id, "trk-") {
		return "track"
	}
	if strings.HasPrefix(id, "pln-") {
		return "plan"
	}
	if strings.HasPrefix(id, "spc-") {
		return "spec"
	}
	return "work item"
}
