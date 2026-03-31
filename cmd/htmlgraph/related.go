package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	dbpkg "github.com/shakestzd/htmlgraph/internal/db"
	"github.com/shakestzd/htmlgraph/internal/workitem"
	"github.com/spf13/cobra"
)

// featureCmdWithExtras builds the standard workitem commands for features,
// then adds the feature-specific "related" and "set-description" subcommands.
func featureCmdWithExtras() *cobra.Command {
	cmd := workitemCmd("feature", "features")
	cmd.AddCommand(relatedCmd())
	cmd.AddCommand(setDescriptionCmd())
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

// setDescriptionCmd returns a cobra.Command that sets a feature's description with optional structured sections.
func setDescriptionCmd() *cobra.Command {
	var acceptance, testStrategy, expectedBehavior string
	cmd := &cobra.Command{
		Use:   "set-description <id> <text>",
		Short: "Set or update a feature's description with optional structured sections",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			return runSetDescription(args[0], args[1], acceptance, testStrategy, expectedBehavior)
		},
	}
	cmd.Flags().StringVar(&acceptance, "acceptance", "", "Acceptance criteria")
	cmd.Flags().StringVar(&testStrategy, "test-strategy", "", "Test strategy")
	cmd.Flags().StringVar(&expectedBehavior, "expected-behavior", "", "Expected behavior")
	return cmd
}

func runSetDescription(id, text, acceptance, testStrategy, expectedBehavior string) error {
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

	content := buildDescription(text, acceptance, testStrategy, expectedBehavior)
	if err := p.Features.Edit(id).SetDescription(content).Save(); err != nil {
		return fmt.Errorf("set description: %w", err)
	}
	fmt.Printf("Updated description for %s\n", id)
	return nil
}

// buildDescription formats the description text with optional structured sections.
// If no sections are provided, returns plain text. Otherwise, returns formatted HTML sections.
func buildDescription(text, acceptance, testStrategy, expectedBehavior string) string {
	if acceptance == "" && testStrategy == "" && expectedBehavior == "" {
		return text
	}
	var sb strings.Builder
	if text != "" {
		sb.WriteString("<p>" + text + "</p>")
	}
	if acceptance != "" {
		sb.WriteString("\n<h2>Acceptance Criteria</h2>\n<p>" + acceptance + "</p>")
	}
	if testStrategy != "" {
		sb.WriteString("\n<h2>Test Strategy</h2>\n<p>" + testStrategy + "</p>")
	}
	if expectedBehavior != "" {
		sb.WriteString("\n<h2>Expected Behavior</h2>\n<p>" + expectedBehavior + "</p>")
	}
	return sb.String()
}
