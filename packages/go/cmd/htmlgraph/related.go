package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	dbpkg "github.com/shakestzd/htmlgraph/internal/db"
	"github.com/spf13/cobra"
)

// featureCmdWithExtras builds the standard workitem commands for features,
// then adds the feature-specific "related" subcommand.
func featureCmdWithExtras() *cobra.Command {
	cmd := workitemCmd("feature", "features")
	cmd.AddCommand(relatedCmd())
	return cmd
}

// relatedCmd returns a cobra.Command that lists features sharing files with a
// given feature, ordered by number of shared files descending.
func relatedCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "related <feature-id>",
		Short: "Find features sharing files with a given feature",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runRelated(args[0])
		},
	}
}

func runRelated(featureID string) error {
	htmlgraphDir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}

	dbPath := filepath.Join(htmlgraphDir, "htmlgraph.db")
	database, err := dbpkg.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer database.Close()

	related, err := dbpkg.FindRelatedFeatures(database, featureID)
	if err != nil {
		return fmt.Errorf("find related features: %w", err)
	}

	if len(related) == 0 {
		fmt.Printf("No features share files with %s.\n", featureID)
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "FEATURE ID\tSHARED\tTITLE\tFILES")
	fmt.Fprintln(w, strings.Repeat("-", 80))
	for _, r := range related {
		title := r.Title
		if title == "" {
			title = "(not indexed)"
		}
		files := strings.Join(r.SharedFiles, ", ")
		if len(files) > 60 {
			files = files[:57] + "..."
		}
		fmt.Fprintf(w, "%s\t%d\t%s\t%s\n",
			r.FeatureID, r.SharedCount, truncate(title, 30), files)
	}
	w.Flush()
	fmt.Printf("\n%d related feature(s)\n", len(related))
	return nil
}
