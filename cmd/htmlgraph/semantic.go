package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	dbpkg "github.com/shakestzd/htmlgraph/internal/db"
	"github.com/spf13/cobra"
)

func semanticCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "semantic",
		Short: "Semantic search and feature relationship discovery",
		Long: `Search across all work items using full-text semantic matching.
Uses BM25 ranking with porter stemming so "cache" matches "caching", "cached", etc.

Examples:
  htmlgraph semantic search "authentication flow"
  htmlgraph semantic related feat-abc12345
  htmlgraph semantic rebuild`,
	}

	cmd.AddCommand(semanticSearchCmd())
	cmd.AddCommand(semanticRelatedCmd())
	cmd.AddCommand(semanticRebuildCmd())

	return cmd
}

func semanticSearchCmd() *cobra.Command {
	var (
		limit    int
		jsonFlag bool
	)

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search features by semantic content",
		Long: `Performs a BM25-ranked full-text search across all indexed features.
Searches titles, descriptions, content, tags, track names, and related feature context.

Porter stemming enables fuzzy matching: "implement" finds "implementing", "implementation".`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			query := strings.Join(args, " ")
			return runSemanticSearch(query, limit, jsonFlag)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "n", 20, "Maximum results")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")

	return cmd
}

func semanticRelatedCmd() *cobra.Command {
	var (
		limit    int
		jsonFlag bool
	)

	cmd := &cobra.Command{
		Use:   "related <feature-id>",
		Short: "Find features semantically related to a given feature",
		Long: `Discovers features with similar content, tags, or context to the specified feature.
Uses the feature's title and tags as a similarity query against all other features.`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runSemanticRelated(args[0], limit, jsonFlag)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "n", 10, "Maximum results")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")

	return cmd
}

func semanticRebuildCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rebuild",
		Short: "Rebuild the semantic index from scratch",
		Long:  `Drops and recreates the FTS5 semantic index, repopulating from all features with graph context.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runSemanticRebuild()
		},
	}
}

func runSemanticSearch(query string, limit int, jsonOut bool) error {
	dir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}

	database, err := dbpkg.Open(filepath.Join(dir, "htmlgraph.db"))
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer database.Close()

	results, err := dbpkg.SemanticSearch(database, query, limit)
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}

	if jsonOut {
		if results == nil {
			results = []dbpkg.SemanticResult{}
		}
		return json.NewEncoder(os.Stdout).Encode(results)
	}

	if len(results) == 0 {
		fmt.Println("No matching items found.")
		fmt.Println("Tip: run 'htmlgraph semantic rebuild' to populate the index.")
		return nil
	}

	printSemanticResults(results)
	return nil
}

func runSemanticRelated(featureID string, limit int, jsonOut bool) error {
	dir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}

	database, err := dbpkg.Open(filepath.Join(dir, "htmlgraph.db"))
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer database.Close()

	results, err := dbpkg.SemanticRelated(database, featureID, limit)
	if err != nil {
		return fmt.Errorf("related: %w", err)
	}

	if jsonOut {
		if results == nil {
			results = []dbpkg.SemanticResult{}
		}
		return json.NewEncoder(os.Stdout).Encode(results)
	}

	if len(results) == 0 {
		fmt.Printf("No related items found for %s.\n", featureID)
		return nil
	}

	fmt.Printf("Features related to %s:\n\n", featureID)
	printSemanticResults(results)
	return nil
}

func runSemanticRebuild() error {
	dir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}

	database, err := dbpkg.Open(filepath.Join(dir, "htmlgraph.db"))
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer database.Close()

	count, err := dbpkg.RebuildSemanticIndex(database)
	if err != nil {
		return fmt.Errorf("rebuild: %w", err)
	}

	fmt.Printf("Semantic index rebuilt: %d features indexed.\n", count)
	return nil
}

func printSemanticResults(results []dbpkg.SemanticResult) {
	fmt.Printf("%-22s  %-8s  %-11s  %-8s  %s\n",
		"ID", "TYPE", "STATUS", "PRIORITY", "TITLE")
	fmt.Println(strings.Repeat("-", 80))

	for _, r := range results {
		fmt.Printf("%-22s  %-8s  %-11s  %-8s  %s\n",
			r.FeatureID, r.Type, r.Status, r.Priority,
			truncate(r.Title, 36))
	}
	fmt.Printf("\n%d result(s)\n", len(results))
}
