package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/shakestzd/htmlgraph/internal/graph"
	"github.com/shakestzd/htmlgraph/internal/htmlparse"
	"github.com/shakestzd/htmlgraph/internal/models"
	"github.com/shakestzd/htmlgraph/pkg/sdk"
	"github.com/spf13/cobra"
)

func featureCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "feature",
		Short: "Manage and inspect features",
	}
	cmd.AddCommand(featureListCmd())
	cmd.AddCommand(featureShowCmd())
	cmd.AddCommand(featureStartCmd())
	cmd.AddCommand(featureCompleteCmd())
	cmd.AddCommand(featureCreateCmd())
	return cmd
}

// featureListCmd lists features, optionally filtered by status.
func featureListCmd() *cobra.Command {
	var statusFilter string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List features",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runFeatureList(statusFilter)
		},
	}
	cmd.Flags().StringVarP(&statusFilter, "status", "s", "",
		"Filter by status (todo, in-progress, blocked, done)")
	return cmd
}

func runFeatureList(statusFilter string) error {
	dir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}

	nodes, err := graph.LoadDir(filepath.Join(dir, "features"))
	if err != nil {
		return fmt.Errorf("load features: %w", err)
	}

	// Filter.
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
		fmt.Println("No features found.")
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
	fmt.Printf("\n%d feature(s)\n", len(filtered))
	return nil
}

// featureShowCmd shows a single feature by ID.
func featureShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show feature details",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runFeatureShow(args[0])
		},
	}
}

func runFeatureShow(id string) error {
	dir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}

	path := resolveNodePath(dir, id)
	if path == "" {
		return fmt.Errorf("work item %q not found in %s", id, dir)
	}

	node, err := htmlparse.ParseFile(path)
	if err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}

	printNodeDetail(node)
	return nil
}

// resolveNodePath searches all subdirectories for a file matching id.
func resolveNodePath(htmlgraphDir, id string) string {
	subdirs := []string{"features", "bugs", "spikes", "tracks"}
	for _, sub := range subdirs {
		p := filepath.Join(htmlgraphDir, sub, id+".html")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// featureStartCmd marks a feature as in-progress.
func featureStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start <id>",
		Short: "Mark a feature as in-progress",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runFeatureStart(args[0])
		},
	}
}

func runFeatureStart(id string) error {
	dir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}
	s, err := sdk.New(dir, "claude-code")
	if err != nil {
		return fmt.Errorf("open SDK: %w", err)
	}
	defer s.Close()

	node, err := s.Features.Start(id)
	if err != nil {
		return fmt.Errorf("start feature: %w", err)
	}
	fmt.Printf("Started: %s  %s\n", node.ID, node.Title)
	return nil
}

// featureCompleteCmd marks a feature as done.
func featureCompleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "complete <id>",
		Short: "Mark a feature as done",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runFeatureComplete(args[0])
		},
	}
}

func runFeatureComplete(id string) error {
	dir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}
	s, err := sdk.New(dir, "claude-code")
	if err != nil {
		return fmt.Errorf("open SDK: %w", err)
	}
	defer s.Close()

	node, err := s.Features.Complete(id)
	if err != nil {
		return fmt.Errorf("complete feature: %w", err)
	}
	fmt.Printf("Completed: %s  %s\n", node.ID, node.Title)
	return nil
}

// featureCreateCmd creates a new feature.
func featureCreateCmd() *cobra.Command {
	var trackID, priority string

	cmd := &cobra.Command{
		Use:   "create <title>",
		Short: "Create a new feature",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runFeatureCreate(args[0], trackID, priority)
		},
	}
	cmd.Flags().StringVar(&trackID, "track", "", "track ID to link to")
	cmd.Flags().StringVar(&priority, "priority", "medium", "priority (low|medium|high|critical)")
	return cmd
}

func runFeatureCreate(title, trackID, priority string) error {
	dir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}
	s, err := sdk.New(dir, "claude-code")
	if err != nil {
		return fmt.Errorf("open SDK: %w", err)
	}
	defer s.Close()

	opts := []sdk.FeatureOption{sdk.FeatWithPriority(priority)}
	if trackID != "" {
		opts = append(opts, sdk.FeatWithTrack(trackID))
	}

	node, err := s.Features.Create(title, opts...)
	if err != nil {
		return fmt.Errorf("create feature: %w", err)
	}
	fmt.Printf("Created: %s  %s\n", node.ID, node.Title)
	return nil
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
