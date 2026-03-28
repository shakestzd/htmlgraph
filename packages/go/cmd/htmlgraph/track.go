package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/shakestzd/htmlgraph/internal/graph"
	"github.com/shakestzd/htmlgraph/internal/htmlparse"
	"github.com/shakestzd/htmlgraph/internal/models"
	"github.com/shakestzd/htmlgraph/internal/workitem"
	"github.com/spf13/cobra"
)

func trackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "track",
		Short: "Manage tracks (multi-feature initiatives)",
	}
	cmd.AddCommand(trackNewCmd())
	cmd.AddCommand(trackListCmd())
	cmd.AddCommand(trackShowCmd())
	cmd.AddCommand(trackDeleteCmd())
	return cmd
}

// trackNewCmd creates a new track.
func trackNewCmd() *cobra.Command {
	var priority, description string

	cmd := &cobra.Command{
		Use:   "new <title>",
		Short: "Create a new track",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runTrackNew(args[0], priority, description)
		},
	}
	cmd.Flags().StringVarP(&priority, "priority", "p", "medium",
		"Priority (critical/high/medium/low)")
	cmd.Flags().StringVarP(&description, "description", "d", "",
		"Track description")
	return cmd
}

func runTrackNew(title, priority, description string) error {
	dir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}

	p, err := workitem.Open(dir, "cli")
	if err != nil {
		return fmt.Errorf("open project: %w", err)
	}
	defer p.Close()

	opts := []workitem.TrackOption{
		workitem.TrackWithPriority(priority),
	}
	if description != "" {
		opts = append(opts, workitem.TrackWithContent(description))
	}

	node, err := p.Tracks.Create(title, opts...)
	if err != nil {
		return fmt.Errorf("create track: %w", err)
	}

	fmt.Printf("Created track %s\n  Title:    %s\n  Priority: %s\n",
		node.ID, node.Title, node.Priority)
	return nil
}

// trackListCmd lists all tracks.
func trackListCmd() *cobra.Command {
	var statusFilter string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tracks",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runTrackList(statusFilter)
		},
	}
	cmd.Flags().StringVarP(&statusFilter, "status", "s", "",
		"Filter by status (todo, in-progress, blocked, done)")
	return cmd
}

func runTrackList(statusFilter string) error {
	dir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}

	tracks, err := graph.LoadDir(filepath.Join(dir, "tracks"))
	if err != nil {
		return fmt.Errorf("load tracks: %w", err)
	}

	var filtered []*models.Node
	for _, n := range tracks {
		if statusFilter != "" && string(n.Status) != statusFilter {
			continue
		}
		filtered = append(filtered, n)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].ID < filtered[j].ID
	})

	if len(filtered) == 0 {
		fmt.Println("No tracks found.")
		return nil
	}

	featCounts := loadFeatureCounts(dir)

	fmt.Printf("%-22s  %-11s  %-8s  %5s  %s\n",
		"ID", "STATUS", "PRIORITY", "FEATS", "TITLE")
	fmt.Println(strings.Repeat("-", 80))
	for _, n := range filtered {
		marker := "  "
		if n.Status == models.StatusInProgress {
			marker = "* "
		}
		fmt.Printf("%s%-20s  %-11s  %-8s  %5d  %s\n",
			marker, n.ID, n.Status, n.Priority,
			featCounts[n.ID], truncate(n.Title, 38))
	}
	fmt.Printf("\n%d track(s)\n", len(filtered))
	return nil
}

// loadFeatureCounts returns a map of track ID → feature count.
func loadFeatureCounts(htmlgraphDir string) map[string]int {
	counts := make(map[string]int)
	nodes, err := graph.LoadDir(filepath.Join(htmlgraphDir, "features"))
	if err != nil {
		return counts
	}
	for _, n := range nodes {
		if n.TrackID != "" {
			counts[n.TrackID]++
		}
	}
	return counts
}

// trackShowCmd shows a single track by ID.
func trackShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show track details",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runTrackShow(args[0])
		},
	}
}

func runTrackShow(id string) error {
	dir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}

	path := filepath.Join(dir, "tracks", id+".html")
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("track %q not found", id)
	}

	node, err := htmlparse.ParseFile(path)
	if err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}

	printTrackDetail(node, dir)
	return nil
}

func printTrackDetail(n *models.Node, htmlgraphDir string) {
	sep := strings.Repeat("─", 60)
	fmt.Println(sep)
	fmt.Printf("  %s\n", n.Title)
	fmt.Println(sep)
	fmt.Printf("  ID        %s\n", n.ID)
	fmt.Printf("  Type      %s\n", n.Type)
	fmt.Printf("  Status    %s\n", n.Status)
	fmt.Printf("  Priority  %s\n", n.Priority)
	if !n.CreatedAt.IsZero() {
		fmt.Printf("  Created   %s\n", n.CreatedAt.Format("2006-01-02"))
	}

	features := loadLinkedFeatures(htmlgraphDir, n.ID)
	if len(features) > 0 {
		fmt.Printf("\nLinked features (%d):\n", len(features))
		for _, f := range features {
			marker := "  "
			if f.Status == models.StatusInProgress {
				marker = "* "
			}
			fmt.Printf("  %s%-20s  %-11s  %s\n",
				marker, f.ID, f.Status, truncate(f.Title, 38))
		}
	}

	if n.Content != "" {
		fmt.Println("\nDescription:")
		for _, line := range strings.Split(n.Content, "\n") {
			fmt.Printf("  %s\n", line)
		}
	}

	if len(n.Steps) > 0 {
		done := 0
		for _, s := range n.Steps {
			if s.Completed {
				done++
			}
		}
		fmt.Printf("\nRequirements: %d/%d complete\n", done, len(n.Steps))
		for _, s := range n.Steps {
			tick := "[ ]"
			if s.Completed {
				tick = "[x]"
			}
			fmt.Printf("  %s  %s\n", tick, s.Description)
		}
	}
}

// loadLinkedFeatures returns features whose TrackID matches trackID.
func loadLinkedFeatures(htmlgraphDir, trackID string) []*models.Node {
	nodes, err := graph.LoadDir(filepath.Join(htmlgraphDir, "features"))
	if err != nil {
		return nil
	}
	var linked []*models.Node
	for _, n := range nodes {
		if n.TrackID == trackID {
			linked = append(linked, n)
		}
	}
	sort.Slice(linked, func(i, j int) bool {
		return linked[i].ID < linked[j].ID
	})
	return linked
}

// trackDeleteCmd deletes a track by ID.
func trackDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a track",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runTrackDelete(args[0], force)
		},
	}
	cmd.Flags().BoolVarP(&force, "force", "f", false,
		"Skip confirmation prompt")
	return cmd
}

func runTrackDelete(id string, force bool) error {
	dir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}

	path := filepath.Join(dir, "tracks", id+".html")
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("track %q not found", id)
	}

	if !force {
		fmt.Printf("Delete track %s? [y/N] ", id)
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if answer != "y" && answer != "yes" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("delete track %s: %w", id, err)
	}

	fmt.Printf("Deleted track %s\n", id)
	return nil
}
