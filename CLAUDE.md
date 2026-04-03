@AGENTS.md

# HtmlGraph — Claude Code

Local-first observability and coordination platform for AI-assisted development.

---

## Build

**Always use `htmlgraph build`, never `go build` directly.**

```bash
htmlgraph build      # outputs to plugin/hooks/bin/htmlgraph (on your PATH)
plugin/build.sh      # equivalent
```

Running `go build -o htmlgraph ./cmd/htmlgraph/` puts the binary in CWD, not on your PATH.

---

## Quality Gates

```bash
go build ./... && go vet ./... && go test ./...
# Commit only when ALL pass
```

Use `/htmlgraph:code-quality-skill` for the complete pre-commit workflow.

---

## Deploy

```bash
./scripts/deploy-all.sh X.Y.Z --no-confirm   # full pipeline
```

Or `/htmlgraph:deploy X.Y.Z`. CLI binary and plugin are independent installs — the deploy script updates both. Never update one without the other.

---

## Dev Mode

```bash
htmlgraph claude --dev   # loads plugin from source, injects orchestrator prompt
```

---

## Plugin Source — Single Source of Truth

**Edit `plugin/`, never `.claude/` (auto-synced, changes are lost).**

| Edit here | Never here |
|-----------|-----------|
| `plugin/hooks/hooks.json` | `.claude/hooks/hooks.json` |
| `plugin/agents/` | `.claude/agents/` |
| `plugin/skills/` | `.claude/skills/` |
| `cmd/` / `internal/` for Go logic | `.claude/` anything |

See `.claude/rules/plugin-development.md` for full plugin structure reference.

---

## Orchestration

Delegate ALL operations except `Task()`, `AskUserQuestion()`, `TodoWrite()`, SDK operations.

Use `/htmlgraph:orchestrator-directives-skill` for delegation patterns and model selection.

---

## Quick Commands

| Task | Command |
|------|---------|
| View work | `htmlgraph snapshot --summary` |
| Run tests | `go test ./...` |
| Build binary | `htmlgraph build` |
| Deploy | `./scripts/deploy-all.sh VERSION --no-confirm` |
| Dashboard | `htmlgraph serve` |
| Status | `htmlgraph status` |

---

## Dogfooding

This project uses HtmlGraph to develop itself. `.htmlgraph/` contains real work items — not demos.
