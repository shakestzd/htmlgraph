package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/shakestzd/htmlgraph/internal/htmlparse"
	"github.com/shakestzd/htmlgraph/internal/models"
	"github.com/shakestzd/htmlgraph/internal/workitem"
	"github.com/spf13/cobra"
)

// executePreview is the JSON envelope returned by `htmlgraph execute-preview`.
// It aggregates everything an orchestrator needs to start dispatching work on a
// track: track metadata, linked work items grouped by kind, and current git state.
type executePreview struct {
	Track    *models.Node    `json:"track"`
	Features []*models.Node  `json:"features,omitempty"`
	Bugs     []*models.Node  `json:"bugs,omitempty"`
	Plans    []*models.Node  `json:"plans,omitempty"`
	Spikes   []*models.Node  `json:"spikes,omitempty"`
	Git      executeGitState `json:"git"`
}

type executeGitState struct {
	Branch            string `json:"branch"`
	HeadSHA           string `json:"head_sha"`
	CommitsAheadMain  int    `json:"commits_ahead_main"`
	CommitsBehindMain int    `json:"commits_behind_main"`
	WorktreePath      string `json:"worktree_path"`
}

func executePreviewCmd() *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "execute-preview <trk-id>",
		Short: "Return everything /htmlgraph:execute needs to start dispatching — one call",
		Long: "Aggregates track metadata, linked features/bugs/plans, and current git state " +
			"into a single structured payload. Collapses the ~10-call discovery sequence " +
			"that orchestrators previously needed before first dispatch.",
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runExecutePreview(args[0], format)
		},
	}
	cmd.Flags().StringVar(&format, "format", "text", "Output format: json or text")
	return cmd
}

func runExecutePreview(id, format string) error {
	dir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}
	resolved, err := resolveID(dir, id)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(resolved, "trk-") {
		return fmt.Errorf("execute-preview: expected a track id, got %q", resolved)
	}

	preview, err := buildExecutePreview(dir, resolved)
	if err != nil {
		return err
	}

	switch format {
	case "json":
		data, err := json.MarshalIndent(preview, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal json: %w", err)
		}
		fmt.Println(string(data))
	default:
		printExecutePreviewText(preview)
	}
	return nil
}

func buildExecutePreview(dir, trackID string) (*executePreview, error) {
	trackPath := resolveNodePath(dir, trackID)
	if trackPath == "" {
		return nil, workitem.ErrNotFoundOnDisk("track", trackID)
	}
	trackNode, err := htmlparse.ParseFile(trackPath)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", trackPath, err)
	}

	preview := &executePreview{Track: trackNode}

	// Walk every edge from the track and categorize linked nodes by id prefix.
	seen := make(map[string]bool)
	for _, edges := range trackNode.Edges {
		for _, edge := range edges {
			if seen[edge.TargetID] {
				continue
			}
			seen[edge.TargetID] = true
			path := resolveNodePath(dir, edge.TargetID)
			if path == "" {
				continue
			}
			node, err := htmlparse.ParseFile(path)
			if err != nil {
				continue
			}
			switch {
			case strings.HasPrefix(edge.TargetID, "feat-"):
				preview.Features = append(preview.Features, node)
			case strings.HasPrefix(edge.TargetID, "bug-"):
				preview.Bugs = append(preview.Bugs, node)
			case strings.HasPrefix(edge.TargetID, "plan-"):
				preview.Plans = append(preview.Plans, node)
			case strings.HasPrefix(edge.TargetID, "spike-"):
				preview.Spikes = append(preview.Spikes, node)
			}
		}
	}

	sort.Slice(preview.Features, func(i, j int) bool { return preview.Features[i].ID < preview.Features[j].ID })
	sort.Slice(preview.Bugs, func(i, j int) bool { return preview.Bugs[i].ID < preview.Bugs[j].ID })
	sort.Slice(preview.Plans, func(i, j int) bool { return preview.Plans[i].ID < preview.Plans[j].ID })
	sort.Slice(preview.Spikes, func(i, j int) bool { return preview.Spikes[i].ID < preview.Spikes[j].ID })

	preview.Git = currentGitState(dir)
	return preview, nil
}

// currentGitState resolves git state for the caller's current working directory.
// The orchestrator calling execute-preview wants to know "where am I in git?",
// so probes run in the caller's cwd (typically a worktree) rather than in the
// shared htmlgraph project root. On any failure returns a zero-valued struct.
func currentGitState(htmlgraphDir string) executeGitState {
	state := executeGitState{}
	cwd, err := os.Getwd()
	if err != nil {
		// Fall back to the htmlgraph project root when cwd is unavailable.
		cwd = filepath.Dir(htmlgraphDir)
	}
	state.WorktreePath = cwd

	if branch, err := gitOutputIn(cwd, "rev-parse", "--abbrev-ref", "HEAD"); err == nil {
		state.Branch = branch
	}
	if sha, err := gitOutputIn(cwd, "rev-parse", "HEAD"); err == nil {
		state.HeadSHA = sha
	}
	// --left-right main...HEAD with --count emits "<behind>\t<ahead>".
	if counts, err := gitOutputIn(cwd, "rev-list", "--left-right", "--count", "main...HEAD"); err == nil {
		parts := strings.Fields(counts)
		if len(parts) == 2 {
			if behind, err := strconv.Atoi(parts[0]); err == nil {
				state.CommitsBehindMain = behind
			}
			if ahead, err := strconv.Atoi(parts[1]); err == nil {
				state.CommitsAheadMain = ahead
			}
		}
	}
	return state
}

// gitOutputIn runs a git sub-command with cwd set to repoRoot, returning
// trimmed stdout. Distinct from gitOutput in review.go (which runs in the
// current working directory) because execute-preview needs to resolve paths
// relative to the discovered htmlgraph project root, not the caller's cwd.
func gitOutputIn(repoRoot string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	cmd.Stderr = nil
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func printExecutePreviewText(p *executePreview) {
	fmt.Printf("Track: %s  %s  [%s]\n", p.Track.ID, p.Track.Title, p.Track.Status)
	fmt.Printf("Git:   branch=%s  ahead=%d  behind=%d  head=%s\n",
		p.Git.Branch, p.Git.CommitsAheadMain, p.Git.CommitsBehindMain, firstN(p.Git.HeadSHA, 8))
	printNodeGroup("Features", p.Features)
	printNodeGroup("Bugs", p.Bugs)
	printNodeGroup("Plans", p.Plans)
	printNodeGroup("Spikes", p.Spikes)
}

func printNodeGroup(label string, nodes []*models.Node) {
	if len(nodes) == 0 {
		return
	}
	fmt.Printf("\n%s (%d):\n", label, len(nodes))
	for _, n := range nodes {
		fmt.Printf("  %-20s  %-12s  %s\n", n.ID, n.Status, n.Title)
	}
}

func firstN(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

