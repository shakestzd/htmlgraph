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
	"github.com/spf13/cobra"
)

// trackCmdWithExtras builds the standard workitem commands for tracks,
// then replaces the generic show with a track-specific one that lists
// all linked children (features, bugs, and spikes).
func trackCmdWithExtras() *cobra.Command {
	cmd := workitemCmd("track", "tracks")
	// Replace generic show with track-specific show (shows linked features)
	for i, sub := range cmd.Commands() {
		if sub.Use == "show <id>" {
			cmd.RemoveCommand(sub)
			newCmds := append(cmd.Commands()[:i], cmd.Commands()[i:]...)
			_ = newCmds // removal already happened
			break
		}
	}
	cmd.AddCommand(trackShowCmd())
	return cmd
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
	var deep bool
	cmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Show track details",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runTrackShow(args[0], deep)
		},
	}
	cmd.Flags().BoolVar(&deep, "deep", false, "Show all linked items with steps and edges")
	return cmd
}

func runTrackShow(id string, deep bool) error {
	dir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}

	// Try flat format first: tracks/id.html
	path := filepath.Join(dir, "tracks", id+".html")
	if _, err := os.Stat(path); err != nil {
		// Try subdirectory format: tracks/id/index.html
		path = filepath.Join(dir, "tracks", id, "index.html")
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("track %q not found", id)
		}
	}

	node, err := htmlparse.ParseFile(path)
	if err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}

	if deep {
		printTrackDeep(node, dir)
	} else {
		printTrackDetail(node, dir)
	}
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

	printLinkedSection(htmlgraphDir, "features", "Linked features", n.ID)
	printLinkedSection(htmlgraphDir, "bugs", "Linked bugs", n.ID)
	printLinkedSection(htmlgraphDir, "spikes", "Linked spikes", n.ID)

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

// printLinkedSection prints a labelled section of items linked to a track,
// covering a single work item subdir (features, bugs, or spikes).
func printLinkedSection(htmlgraphDir, subdir, label, trackID string) {
	items := loadLinkedByType(htmlgraphDir, subdir, trackID)
	if len(items) == 0 {
		return
	}
	fmt.Printf("\n%s (%d):\n", label, len(items))
	for _, item := range items {
		marker := "  "
		if item.Status == models.StatusInProgress {
			marker = "* "
		}
		fmt.Printf("  %s%-20s  %-11s  %s\n",
			marker, item.ID, item.Status, truncate(item.Title, 38))
	}
}

// containsEdgeIDs returns the set of target IDs referenced by a track's
// "contains" edges, so loadLinkedByType can include edge-linked children that
// do not carry the data-track-id attribute.
func containsEdgeIDs(htmlgraphDir, trackID string) map[string]bool {
	path := filepath.Join(htmlgraphDir, "tracks", trackID+".html")
	node, err := htmlparse.ParseFile(path)
	if err != nil {
		return nil
	}
	ids := make(map[string]bool)
	for _, e := range node.Edges[string(models.RelContains)] {
		ids[e.TargetID] = true
	}
	return ids
}

// loadLinkedByType returns nodes of a given subdir linked to trackID either
// via the TrackID metadata field or via a "contains" edge on the track.
func loadLinkedByType(htmlgraphDir, subdir, trackID string) []*models.Node {
	nodes, err := graph.LoadDir(filepath.Join(htmlgraphDir, subdir))
	if err != nil {
		return nil
	}
	edgeIDs := containsEdgeIDs(htmlgraphDir, trackID)
	seen := make(map[string]bool)
	var linked []*models.Node
	for _, n := range nodes {
		if seen[n.ID] {
			continue
		}
		if n.TrackID == trackID || edgeIDs[n.ID] {
			linked = append(linked, n)
			seen[n.ID] = true
		}
	}
	sort.Slice(linked, func(i, j int) bool {
		return linked[i].ID < linked[j].ID
	})
	return linked
}

// printItemSteps prints indented step checklist for an item.
func printItemSteps(n *models.Node) {
	done := 0
	for _, s := range n.Steps {
		if s.Completed {
			done++
		}
	}
	fmt.Printf("    Steps: %d/%d complete\n", done, len(n.Steps))
	for _, s := range n.Steps {
		tick := "[ ]"
		if s.Completed {
			tick = "[x]"
		}
		fmt.Printf("      %s %s\n", tick, truncate(s.Description, 60))
	}
}

// printItemEdges prints indented edges for an item, skipping part_of.
func printItemEdges(n *models.Node) {
	if len(n.Edges) == 0 {
		return
	}
	fmt.Println("    Edges:")
	for rel, edges := range n.Edges {
		if rel == "part_of" {
			continue
		}
		for _, e := range edges {
			fmt.Printf("      %s -> %s\n", rel, e.TargetID)
		}
	}
}

// printDeepItem prints a single linked item with steps and edges.
func printDeepItem(n *models.Node) {
	marker := "  "
	if n.Status == models.StatusInProgress {
		marker = "* "
	} else if n.Status == models.StatusDone {
		marker = "✓ "
	}
	fmt.Printf("  %s%-20s  %-11s  %s\n", marker, n.ID, n.Status, truncate(n.Title, 38))
	if len(n.Steps) > 0 {
		printItemSteps(n)
	}
	printItemEdges(n)
}

// printDeepGroup prints a group of linked items by type label.
func printDeepGroup(label string, items []*models.Node) {
	fmt.Printf("\n%s (%d):\n", label, len(items))
	if len(items) == 0 {
		fmt.Println("  (none)")
		return
	}
	for _, n := range items {
		printDeepItem(n)
	}
}

// printTrackDeep prints a track with all linked items (features, bugs, spikes).
func printTrackDeep(n *models.Node, htmlgraphDir string) {
	sep := strings.Repeat("─", 60)
	fmt.Println(sep)
	fmt.Printf("  %s\n", n.Title)
	fmt.Println(sep)
	fmt.Printf("  ID        %s\n", n.ID)
	fmt.Printf("  Status    %s\n", n.Status)
	fmt.Printf("  Priority  %s\n", n.Priority)
	if !n.CreatedAt.IsZero() {
		fmt.Printf("  Created   %s\n", n.CreatedAt.Format("2006-01-02"))
	}
	features := loadLinkedByType(htmlgraphDir, "features", n.ID)
	bugs := loadLinkedByType(htmlgraphDir, "bugs", n.ID)
	spikes := loadLinkedByType(htmlgraphDir, "spikes", n.ID)
	printDeepGroup("Features", features)
	printDeepGroup("Bugs", bugs)
	printDeepGroup("Spikes", spikes)
}
