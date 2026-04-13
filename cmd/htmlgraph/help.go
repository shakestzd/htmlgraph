package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// compactCLIRef is the canonical compact CLI reference injected into LLM context.
// Keep under 30 lines. Update when commands change.
const compactCLIRef = `htmlgraph CLI commands:
  feature [create|show|start|complete|list|add-step|delete] — Feature work items
  bug [create|show|start|complete|list|add-step|delete] — Bug tracking
  spike [create|show|start|complete|list|add-step|delete] — Investigation spikes
  track [create|show|start|complete|list|add-step|delete] — Multi-feature tracks
  plan [create|show|start|complete|list] — Planning work items
  find <query> — Search work items by title/id
  wip — Show in-progress work items
  status — Quick project status
  snapshot [--summary] — Full project overview
  statusline — OMP/Starship prompt integration
  link [add|remove|list] — Typed edges between work items
  session [list|show] — Session management
  analytics [summary|velocity] — Work analytics and insights
  check — Automated quality gate checks
  health — Code health metrics (module sizes, function lengths)
  spec [generate|show] <feature-id> — Feature specifications
  tdd <feature-id> — Generate test stubs from spec acceptance criteria
  review — Structured diff summary against base branch
  compliance <feature-id> — Score implementation against spec
  batch [apply|export] — Bulk work item operations (YAML)
  ingest — Ingest Claude Code session transcripts (JSONL)
  backfill [feature-files|tool-calls-feature] — Rebuild derived tables
  reindex — Sync HTML work items to SQLite index
  yolo --feature <id> [--track <id>] — Autonomous dev mode
  upgrade [--check] [--version X.Y.Z] — Self-update CLI from GitHub releases (alias: update)
  build — Build Go binary (dev workflow)
  serve — Start local dashboard server
  agent-init — Output shared agent context (safety, attribution, quality gates)

Required flags: feature/bug create require --track <id> --description "…"`

// helpCmd returns the "htmlgraph help" command with --compact flag support.
func helpCmd() *cobra.Command {
	var compact bool

	cmd := &cobra.Command{
		Use:   "help",
		Short: "Show help or compact CLI reference for LLM context",
		Long:  "Show full help or --compact one-line-per-command reference for injecting into LLM context.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if compact {
				fmt.Println(compactCLIRef)
				return nil
			}
			// Default: show root help via parent
			return cmd.Parent().Help()
		},
	}

	cmd.Flags().BoolVar(&compact, "compact", false, "Output a concise per-command reference for LLM context injection")
	return cmd
}
