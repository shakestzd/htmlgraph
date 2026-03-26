package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/shakestzd/htmlgraph/internal/graph"
	"github.com/shakestzd/htmlgraph/internal/models"
	"github.com/shakestzd/htmlgraph/pkg/sdk"
	"github.com/spf13/cobra"
)

func bugCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bug",
		Short: "Manage bugs",
	}
	cmd.AddCommand(bugListCmd())
	cmd.AddCommand(bugCreateCmd())
	return cmd
}

func bugListCmd() *cobra.Command {
	var statusFilter string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List bugs",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runBugList(statusFilter)
		},
	}
	cmd.Flags().StringVarP(&statusFilter, "status", "s", "",
		"Filter by status (todo, in-progress, blocked, done)")
	return cmd
}

func runBugList(statusFilter string) error {
	dir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}

	nodes, err := graph.LoadDir(filepath.Join(dir, "bugs"))
	if err != nil {
		return fmt.Errorf("load bugs: %w", err)
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
		fmt.Println("No bugs found.")
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
	fmt.Printf("\n%d bug(s)\n", len(filtered))
	return nil
}

func bugCreateCmd() *cobra.Command {
	var trackID, priority string

	cmd := &cobra.Command{
		Use:   "create <title>",
		Short: "Create a new bug",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runBugCreate(args[0], trackID, priority)
		},
	}
	cmd.Flags().StringVar(&trackID, "track", "", "track ID to link to")
	cmd.Flags().StringVar(&priority, "priority", "medium", "priority (low|medium|high|critical)")
	return cmd
}

func runBugCreate(title, trackID, priority string) error {
	dir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}
	s, err := sdk.New(dir, "claude-code")
	if err != nil {
		return fmt.Errorf("open SDK: %w", err)
	}
	defer s.Close()

	opts := []sdk.BugOption{sdk.BugWithPriority(priority)}
	if trackID != "" {
		opts = append(opts, sdk.BugWithTrack(trackID))
	}

	node, err := s.Bugs.Create(title, opts...)
	if err != nil {
		return fmt.Errorf("create bug: %w", err)
	}
	fmt.Printf("Created: %s  %s\n", node.ID, node.Title)
	return nil
}
