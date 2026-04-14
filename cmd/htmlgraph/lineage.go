package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/shakestzd/htmlgraph/internal/graph"
	"github.com/shakestzd/htmlgraph/internal/models"
	"github.com/spf13/cobra"
)

// lineageKind classifies the routing target for a `htmlgraph lineage <id>`
// invocation. Routing is purely string-based: prefix → kind.
type lineageKind int

const (
	kindUnknown lineageKind = iota
	kindFeature
	kindBug
	kindSpike
	kindPlan
	kindTrack
	kindSession
	kindCommit
	kindFile
)

// String makes lineageKind printable for test failures.
func (k lineageKind) String() string {
	switch k {
	case kindFeature:
		return "feature"
	case kindBug:
		return "bug"
	case kindSpike:
		return "spike"
	case kindPlan:
		return "plan"
	case kindTrack:
		return "track"
	case kindSession:
		return "session"
	case kindCommit:
		return "commit"
	case kindFile:
		return "file"
	default:
		return "unknown"
	}
}

// lineageHexRe matches commit-shaped hex strings (7-40 chars).
var lineageHexRe = regexp.MustCompile(`^[0-9a-f]{7,40}$`)

// detectLineageKind inspects a CLI argument and returns its routing kind.
// Order matters: ID prefixes win over file path heuristics so an exotic file
// named "feat-x" is still parsed as a work item by intent.
func detectLineageKind(arg string) lineageKind {
	switch {
	case strings.HasPrefix(arg, "feat-"):
		return kindFeature
	case strings.HasPrefix(arg, "bug-"):
		return kindBug
	case strings.HasPrefix(arg, "spk-"):
		return kindSpike
	case strings.HasPrefix(arg, "plan-"):
		return kindPlan
	case strings.HasPrefix(arg, "trk-"):
		return kindTrack
	case strings.HasPrefix(arg, "sess-"):
		return kindSession
	}
	if lineageHexRe.MatchString(arg) {
		return kindCommit
	}
	if strings.ContainsAny(arg, "/.") {
		return kindFile
	}
	return kindUnknown
}

// lineageOpts is the flag bundle for `htmlgraph lineage`.
type lineageOpts struct {
	depth    int
	jsonOut  bool
	timeline bool
}

// lineageNode is one hop in a forward or backward chain. It is the wire format
// for --json output and a convenient internal representation for tree rendering.
type lineageNode struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Title    string `json:"title,omitempty"`
	EdgeType string `json:"edge_type"`
	Depth    int    `json:"depth"`
	// timestamp is populated for --timeline rendering by joining git_commits /
	// agent_events. Empty when no temporal data is available.
	Timestamp string `json:"timestamp,omitempty"`
}

// lineageJSON is the stable schema emitted by `htmlgraph lineage --json`.
//
//	{
//	  "root":     "<id>",
//	  "kind":     "feature|bug|...",
//	  "forward":  [{id,type,title,edge_type,depth,timestamp?}, ...],
//	  "backward": [{id,type,title,edge_type,depth,timestamp?}, ...]
//	}
//
// Forward edges follow `from_node_id = root` outward; backward edges follow
// `to_node_id = root` inward. Each list is depth-ordered (BFS).
type lineageJSON struct {
	Root     string        `json:"root"`
	Kind     string        `json:"kind"`
	Forward  []lineageNode `json:"forward"`
	Backward []lineageNode `json:"backward"`
}

// allLineageRels lists all 10 relationship types we traverse. We do NOT subset:
// any of these can carry causal meaning depending on the slice in question.
var allLineageRels = []string{
	string(models.RelBlocks),
	string(models.RelBlockedBy),
	string(models.RelRelatesTo),
	string(models.RelImplements),
	string(models.RelCausedBy),
	string(models.RelSpawnedFrom),
	string(models.RelImplementedIn),
	string(models.RelPartOf),
	string(models.RelContains),
	string(models.RelPlannedIn),
}

// newLineageCmd registers `htmlgraph lineage <id>` — the headline unified
// causal chain command. It auto-detects the input type, walks graph_edges in
// both directions across all 10 relationship types, and renders a tree.
func newLineageCmd() *cobra.Command {
	opts := lineageOpts{depth: 5}
	cmd := &cobra.Command{
		Use:   "lineage <id>",
		Short: "Walk the causal chain for any work item, session, commit, or file",
		Long: `Auto-detects the ID type and renders the bidirectional causal chain.

Supported inputs:
  feat-/bug-/spk-/plan-/trk- ID  — graph walk across all 10 edge types
  sess-<id>                      — graph walk plus agent spawn tree
  <commit-sha>                   — git commit attribution
  <file/path.go>                 — file-to-feature attribution

Examples:
  htmlgraph lineage feat-48b3783c
  htmlgraph lineage plan-3b0d5133 --depth 8
  htmlgraph lineage sess-abc123 --json
  htmlgraph lineage feat-48b3783c --timeline`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			dir, err := findHtmlgraphDir()
			if err != nil {
				return err
			}
			db, err := openDB(dir)
			if err != nil {
				return err
			}
			defer db.Close()
			return runLineage(os.Stdout, db, args[0], opts)
		},
	}
	cmd.Flags().IntVar(&opts.depth, "depth", 5, "maximum hop count for graph walk")
	cmd.Flags().BoolVar(&opts.jsonOut, "json", false, "emit structured JSON output")
	cmd.Flags().BoolVar(&opts.timeline, "timeline", false, "sort results chronologically instead of as a tree")
	return cmd
}

// runLineage is the testable entry point. It dispatches based on
// detectLineageKind, walks the graph in both directions, and renders.
func runLineage(w io.Writer, db *sql.DB, arg string, opts lineageOpts) error {
	if opts.depth <= 0 {
		opts.depth = 5
	}
	kind := detectLineageKind(arg)

	forward, err := forwardWalk(db, arg, allLineageRels, opts.depth)
	if err != nil {
		return fmt.Errorf("forward walk: %w", err)
	}
	backward, err := backwardWalk(db, arg, allLineageRels, opts.depth)
	if err != nil {
		return fmt.Errorf("backward walk: %w", err)
	}

	if opts.timeline {
		annotateTimestamps(db, forward)
		annotateTimestamps(db, backward)
	}

	if opts.jsonOut {
		return renderLineageJSON(w, arg, kind, forward, backward)
	}

	if err := renderLineageTree(w, db, arg, kind, forward, backward, opts.timeline); err != nil {
		return err
	}

	// For session inputs, additionally render the agent spawn tree.
	if kind == kindSession {
		tree, treeErr := RenderAgentTree(db, arg)
		if treeErr == nil && strings.TrimSpace(tree) != "" {
			fmt.Fprintln(w)
			fmt.Fprintln(w, "Agent spawn chain:")
			fmt.Fprint(w, tree)
		}
	}
	return nil
}

// forwardWalk performs a BFS following from_node_id = current outward.
// Returns nodes in BFS order, each annotated with the edge type that reached
// it and the hop depth (1-indexed).
func forwardWalk(db *sql.DB, root string, rels []string, maxDepth int) ([]lineageNode, error) {
	return bfsWalk(db, root, rels, maxDepth, true)
}

// backwardWalk performs a BFS following to_node_id = current inward — i.e.
// "who points at me?". This is the inline reverse query the plan calls for.
func backwardWalk(db *sql.DB, root string, rels []string, maxDepth int) ([]lineageNode, error) {
	return bfsWalk(db, root, rels, maxDepth, false)
}

// bfsWalk is the shared BFS engine for both directions. When forward=true it
// follows from->to edges; when false it follows to->from edges.
func bfsWalk(db *sql.DB, root string, rels []string, maxDepth int, forward bool) ([]lineageNode, error) {
	if maxDepth <= 0 || len(rels) == 0 {
		return nil, nil
	}

	placeholders := strings.Repeat("?,", len(rels))
	placeholders = placeholders[:len(placeholders)-1]
	var query string
	if forward {
		query = fmt.Sprintf(
			`SELECT to_node_id, to_node_type, relationship_type
			 FROM graph_edges
			 WHERE from_node_id = ? AND relationship_type IN (%s)`,
			placeholders,
		)
	} else {
		query = fmt.Sprintf(
			`SELECT from_node_id, from_node_type, relationship_type
			 FROM graph_edges
			 WHERE to_node_id = ? AND relationship_type IN (%s)`,
			placeholders,
		)
	}

	type queueEntry struct {
		id    string
		depth int
	}
	visited := map[string]bool{root: true}
	queue := []queueEntry{{id: root, depth: 0}}
	var result []lineageNode

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur.depth >= maxDepth {
			continue
		}
		args := make([]any, 0, 1+len(rels))
		args = append(args, cur.id)
		for _, r := range rels {
			args = append(args, r)
		}
		rows, err := db.Query(query, args...)
		if err != nil {
			return nil, fmt.Errorf("query neighbors of %s: %w", cur.id, err)
		}
		for rows.Next() {
			var nid, ntype, rel string
			if err := rows.Scan(&nid, &ntype, &rel); err != nil {
				rows.Close()
				return nil, fmt.Errorf("scan neighbor: %w", err)
			}
			if visited[nid] {
				continue
			}
			visited[nid] = true
			node := lineageNode{
				ID:       nid,
				Type:     ntype,
				EdgeType: rel,
				Depth:    cur.depth + 1,
			}
			result = append(result, node)
			queue = append(queue, queueEntry{id: nid, depth: cur.depth + 1})
		}
		rows.Close()
	}

	// Resolve titles in one shot for display.
	if len(result) > 0 {
		ids := make([]string, len(result))
		for i, n := range result {
			ids[i] = n.ID
		}
		labels := graph.ResolveToMap(db, ids)
		for i := range result {
			if r, ok := labels[result[i].ID]; ok {
				result[i].Title = r.Title
			}
		}
	}

	return result, nil
}

// annotateTimestamps fills in lineageNode.Timestamp by joining git_commits
// (commit_hash) and agent_events (session_id). Best-effort: missing rows
// silently leave Timestamp empty so timeline rendering still includes them.
func annotateTimestamps(db *sql.DB, nodes []lineageNode) {
	for i := range nodes {
		var ts sql.NullString
		// Try git_commits first (commit-shaped IDs).
		_ = db.QueryRow(
			`SELECT timestamp FROM git_commits WHERE commit_hash = ? LIMIT 1`,
			nodes[i].ID,
		).Scan(&ts)
		if !ts.Valid || ts.String == "" {
			// Fall back to agent_events.timestamp via session_id.
			_ = db.QueryRow(
				`SELECT MIN(timestamp) FROM agent_events WHERE session_id = ?`,
				nodes[i].ID,
			).Scan(&ts)
		}
		if ts.Valid {
			nodes[i].Timestamp = ts.String
		}
	}
}

// renderLineageJSON emits the stable {root, kind, forward, backward} schema.
func renderLineageJSON(w io.Writer, root string, kind lineageKind, forward, backward []lineageNode) error {
	out := lineageJSON{
		Root:     root,
		Kind:     kind.String(),
		Forward:  forward,
		Backward: backward,
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// renderLineageTree prints a human-readable indented tree with the query node
// as the pivot. Backward chains print above the pivot, forward chains below.
// When timeline=true, the same nodes render as a chronological list.
func renderLineageTree(
	w io.Writer,
	db *sql.DB,
	root string,
	kind lineageKind,
	forward, backward []lineageNode,
	timeline bool,
) error {
	rootLabel := graph.FormatNodeLabel(root, graph.ResolveToMap(db, []string{root}))

	sep := strings.Repeat("─", 60)
	fmt.Fprintln(w, sep)
	fmt.Fprintf(w, "  Lineage: %s  [%s]\n", rootLabel, kind)
	fmt.Fprintln(w, sep)

	if timeline {
		all := make([]lineageNode, 0, len(forward)+len(backward))
		all = append(all, backward...)
		all = append(all, forward...)
		sort.SliceStable(all, func(i, j int) bool {
			return all[i].Timestamp < all[j].Timestamp
		})
		fmt.Fprintln(w, "\n  Timeline (oldest first):")
		if len(all) == 0 {
			fmt.Fprintln(w, "    (no related nodes)")
			return nil
		}
		for _, n := range all {
			ts := n.Timestamp
			if ts == "" {
				ts = "—"
			}
			fmt.Fprintf(w, "    %s  %s  (%s, d%d)\n", ts, n.ID, n.EdgeType, n.Depth)
		}
		return nil
	}

	if len(backward) > 0 {
		fmt.Fprintf(w, "\n  Ancestors (%d):\n", len(backward))
		printLineageBranches(w, backward)
	}
	fmt.Fprintf(w, "\n  Pivot: %s\n", rootLabel)
	if len(forward) > 0 {
		fmt.Fprintf(w, "\n  Descendants (%d):\n", len(forward))
		printLineageBranches(w, forward)
	}
	if len(forward) == 0 && len(backward) == 0 {
		fmt.Fprintln(w, "\n  (no related nodes — try `htmlgraph trace` for file/commit attribution)")
	}
	return nil
}

// printLineageBranches indents nodes by depth so branching chains are visually
// distinct. Each line: "<indent>[<edge_type>] <id> (<title>)".
func printLineageBranches(w io.Writer, nodes []lineageNode) {
	for _, n := range nodes {
		indent := strings.Repeat("  ", n.Depth)
		label := n.ID
		if n.Title != "" {
			label = fmt.Sprintf("%s (%s)", n.ID, truncate(n.Title, 40))
		}
		fmt.Fprintf(w, "  %s[%s] %s\n", indent, n.EdgeType, label)
	}
}

