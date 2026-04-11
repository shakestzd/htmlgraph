// parallel agent A was here
// agent_init provides shared context to subagents via CLI pull.
package main

import (
	"fmt"
	"strings"

	"github.com/shakestzd/htmlgraph/internal/paths"
	"github.com/spf13/cobra"
)

func agentInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "agent-init",
		Short: "Output shared agent context (safety rules, attribution, quality gates)",
		Long: `Outputs shared instructions that all HtmlGraph agents must follow.
Agents call this command at startup to load safety rules, work attribution
patterns, and project-appropriate quality gates into their context.

This replaces embedded boilerplate in agent prompts — a single source of
truth maintained in the CLI binary and distributed via the plugin.`,
		RunE: runAgentInit,
	}
}

func runAgentInit(_ *cobra.Command, _ []string) error {
	var sections []string

	sections = append(sections, workAttributionSection())
	sections = append(sections, safetyRulesSection())
	sections = append(sections, qualityGatesSection())

	fmt.Print(strings.Join(sections, "\n"))
	return nil
}

func workAttributionSection() string {
	return `## Work Attribution (MANDATORY)

Before ANY other work, identify and activate the work item for this task:

` + "```bash" + `
# Check what's currently in-progress
htmlgraph find --status in-progress

# Start the relevant work item (check task description for the feature/bug ID)
htmlgraph feature start feat-XXXX  # or: htmlgraph bug start bug-XXXX
` + "```" + `

If no work item exists for this task, create one first:
` + "```bash" + `
htmlgraph feature create "Short description of what you're implementing" --track <trk-id> --description "optional detail"
` + "```" + `
`
}

func safetyRulesSection() string {
	return `## HtmlGraph Safety Rules

### FORBIDDEN: Do NOT touch .htmlgraph/ directory
NEVER:
- Edit files in ` + "`.htmlgraph/`" + ` directory
- Create new files in ` + "`.htmlgraph/`" + `
- Modify ` + "`.htmlgraph/*.html`" + ` files
- Write to ` + "`.htmlgraph/*.db`" + ` or any database files
- Delete or rename ` + "`.htmlgraph/`" + ` files
- Read ` + "`.htmlgraph/`" + ` files directly (` + "`cat`" + `, ` + "`grep`" + `, ` + "`sqlite3`" + `)

The .htmlgraph directory is managed by the CLI and hooks.

### Use CLI instead of direct file operations
` + "```bash" + `
# CORRECT
htmlgraph status              # View work status
htmlgraph snapshot --summary  # View all items
htmlgraph find "<query>"      # Search work items

# INCORRECT — never do this
cat .htmlgraph/features/feat-xxx.html
sqlite3 .htmlgraph/htmlgraph.db "SELECT ..."
grep -r topic .htmlgraph/
` + "```" + `
`
}

func qualityGatesSection() string {
	switch detectProjectType() {
	case paths.ProjectTypeGo:
		return goQualityGates()
	case paths.ProjectTypePython:
		return pythonQualityGates()
	case paths.ProjectTypeNode:
		return nodeQualityGates()
	default:
		return genericQualityGates()
	}
}

// detectProjectType resolves the project root from the --project-dir
// flag (or the standard fallback chain) and delegates to
// paths.DetectProjectType for the actual marker scan. Kept as a thin
// wrapper so the cobra-flag dependency stays in cmd/ and the reusable
// detection lives in internal/paths/ alongside ResolveProjectDir.
func detectProjectType() paths.ProjectType {
	root, err := paths.ResolveProjectDir(paths.ProjectDirOptions{
		ExplicitDir: projectDirFlag,
	})
	if err != nil {
		return paths.ProjectTypeUnknown
	}
	return paths.DetectProjectType(root)
}

func goQualityGates() string {
	return `## Quality Gates (Go project detected)

Before committing ANY changes, ALL checks must pass:
` + "```bash" + `
go build ./... && go vet ./... && go test ./...
` + "```" + `

### Development Principles
- **DRY** — Check ` + "`internal/`" + ` for existing utilities before writing new ones
- **SRP** — Each package has one clear purpose
- **KISS** — Simplest solution that works
- **YAGNI** — Only implement what's needed now
- Functions: <50 lines | Modules: <500 lines
- Check ` + "`go.mod`" + ` for existing dependencies before adding new ones
- Prefer stdlib over external packages
`
}

func pythonQualityGates() string {
	return `## Quality Gates (Python project detected)

Before committing ANY changes, ALL checks must pass:
` + "```bash" + `
uv run ruff check --fix && uv run ruff format
uv run mypy src/
uv run pytest
` + "```" + `

### Development Principles
- **DRY** — Check existing utils before writing new helpers
- **SRP** — Each module has one clear purpose
- **KISS** — Simplest solution that works
- **YAGNI** — Only implement what's needed now
- Functions: <50 lines | Modules: <500 lines
- Check ` + "`pyproject.toml`" + ` for existing dependencies before adding new ones
- Prefer stdlib over external packages
`
}

func nodeQualityGates() string {
	return `## Quality Gates (Node.js project detected)

Before committing ANY changes, run available checks:
` + "```bash" + `
npm test
npm run lint  # if available
npm run build # if available
` + "```" + `

### Development Principles
- **DRY** — Check existing utils before writing new helpers
- **SRP** — Each module has one clear purpose
- **KISS** — Simplest solution that works
- **YAGNI** — Only implement what's needed now
- Check ` + "`package.json`" + ` for existing dependencies before adding new ones
`
}

func genericQualityGates() string {
	return `## Quality Gates

Before committing ANY changes, run the project's test suite and linter.

### Development Principles
- **DRY** — Check for existing utilities before writing new ones
- **SRP** — Each module has one clear purpose
- **KISS** — Simplest solution that works
- **YAGNI** — Only implement what's needed now
`
}

