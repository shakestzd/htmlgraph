package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

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
	return cmd
}

func wiCreateCmd(typeName, dirName string) *cobra.Command {
	var trackID, priority string
	var start bool

	cmd := &cobra.Command{
		Use:   "create <title>",
		Short: "Create a new " + typeName,
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runWiCreate(typeName, args[0], trackID, priority, start)
		},
	}
	cmd.Flags().StringVar(&trackID, "track", "", "track ID to link to")
	cmd.Flags().StringVar(&priority, "priority", "medium", "priority (low|medium|high|critical)")
	cmd.Flags().BoolVar(&start, "start", false, "immediately mark as in-progress")
	return cmd
}

func runWiCreate(typeName, title, trackID, priority string, start bool) error {
	dir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}
	p, err := workitem.Open(dir, "claude-code")
	if err != nil {
		return fmt.Errorf("open project: %w", err)
	}
	defer p.Close()

	var node *models.Node
	switch typeName {
	case "feature":
		opts := []workitem.FeatureOption{workitem.FeatWithPriority(priority)}
		if trackID != "" {
			opts = append(opts, workitem.FeatWithTrack(trackID))
		}
		node, err = p.Features.Create(title, opts...)
	case "bug":
		opts := []workitem.BugOption{workitem.BugWithPriority(priority)}
		node, err = p.Bugs.Create(title, opts...)
	case "spike":
		opts := []workitem.SpikeOption{workitem.SpikeWithPriority(priority)}
		node, err = p.Spikes.Create(title, opts...)
	case "track":
		opts := []workitem.TrackOption{workitem.TrackWithPriority(priority)}
		node, err = p.Tracks.Create(title, opts...)
	case "plan":
		opts := []workitem.PlanOption{workitem.PlanWithPriority(priority)}
		if trackID != "" {
			opts = append(opts, workitem.PlanWithTrack(trackID))
		}
		node, err = p.Plans.Create(title, opts...)
	case "spec":
		opts := []workitem.SpecOption{workitem.SpecWithPriority(priority)}
		if trackID != "" {
			opts = append(opts, workitem.SpecWithTrack(trackID))
		}
		node, err = p.Specs.Create(title, opts...)
	default:
		return fmt.Errorf("unknown type: %s", typeName)
	}
	if err != nil {
		return fmt.Errorf("create %s: %w", typeName, err)
	}
	if start {
		if _, startErr := collectionFor(p, typeName).Start(node.ID); startErr != nil {
			return fmt.Errorf("start %s: %w", typeName, startErr)
		}
		fmt.Printf("Created and started: %s  %s\n", node.ID, node.Title)
	} else {
		fmt.Printf("Created: %s  %s\n", node.ID, node.Title)
	}
	return nil
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
	path := resolveNodePath(dir, id)
	if path == "" {
		return fmt.Errorf("work item %q not found", id)
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
	p, err := workitem.Open(dir, "claude-code")
	if err != nil {
		return fmt.Errorf("open project: %w", err)
	}
	defer p.Close()

	col := collectionFor(p, typeName)
	var node *models.Node
	if status == "in-progress" {
		node, err = col.Start(id)
	} else {
		node, err = col.Complete(id)
	}
	if err != nil {
		return fmt.Errorf("set %s %s: %w", typeName, status, err)
	}

	// When starting a work item, update active_feature_id on the DB session row
	// so the YOLO guard can see the active work item without reading HTML files.
	if status == "in-progress" && p.DB != nil {
		projectDir := strings.TrimSuffix(dir, "/.htmlgraph")
		sessionID := hooks.EnvSessionID("")
		if sessionID != "" {
			_ = hooks.UpdateActiveFeature(p.DB, sessionID, id)
		}
		_ = projectDir // used implicitly via EnvSessionID resolution
	}

	verb := "Started"
	if status == "done" {
		verb = "Completed"
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
	path := resolveNodePath(dir, id)
	if path == "" {
		return fmt.Errorf("work item %q not found", id)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("delete %s: %w", id, err)
	}
	fmt.Printf("Deleted: %s\n", id)
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

// resolveNodePath searches all subdirectories for a file matching id.
func resolveNodePath(htmlgraphDir, id string) string {
	subdirs := []string{"features", "bugs", "spikes", "tracks", "plans", "specs"}
	for _, sub := range subdirs {
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
