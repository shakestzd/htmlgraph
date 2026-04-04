# HtmlGraph

**Local-first observability and coordination platform for AI-assisted development.**

Work items, session tracking, custom agents, hooks, slash commands, quality gates, and a real-time dashboard — managed by a single Go binary, stored as HTML files in your repo. No external infrastructure required.

## Architecture

| Layer | Role |
|-------|------|
| `.htmlgraph/*.html` | Canonical store — single source of truth |
| SQLite (`.htmlgraph/htmlgraph.db`) | Read index for queries and dashboard |
| Go binary (`htmlgraph`) | CLI + hook handler |

## Install

```bash
# Claude Code plugin (recommended)
claude plugin install htmlgraph

# Or build from source
git clone https://github.com/shakestzd/htmlgraph.git
cd htmlgraph && go build -o htmlgraph ./cmd/htmlgraph/
```

## Quick Start

```bash
htmlgraph init                          # creates .htmlgraph/ in your repo
htmlgraph track create "Auth Overhaul"
htmlgraph feature create "Add OAuth" --track <trk-id> --description "Implement OAuth2 flow"
htmlgraph feature start <feat-id>
# ... do work ...
htmlgraph feature complete <feat-id>
htmlgraph serve                         # dashboard at localhost:4000
```

## What It Does

**Work item tracking** — Features, bugs, spikes, and tracks as HTML files in `.htmlgraph/`. Every change is a git diff. Every item has a lifecycle: create, start, complete.

**Session observability** — Hooks capture every tool call, every prompt, and attribute them to the active work item. See exactly what happened in any session via the dashboard.

**Custom agents** — Define specialized agents with specific models, tools, and system prompts. A researcher agent for investigation, a coder for implementation, a test runner for quality — each scoped to its job.

**Hooks & automation** — Event-driven hooks on SessionStart, PreToolUse, PostToolUse, and Stop. Enforce safety rules, capture telemetry, block dangerous operations, or trigger custom workflows automatically.

**Skills & slash commands** — Reusable workflows as slash commands: `/deploy`, `/diagnose`, `/plan`, `/code-quality`. Package complex multi-step procedures into single invocations.

**Quality gates** — Enforce software engineering discipline: build, lint, and test before every commit. Spec compliance scoring, code health metrics, and structured diff reviews.

**Real-time dashboard** — Activity feed, kanban board, session viewer, and work item detail — served locally by `htmlgraph serve`.

**Multi-agent coordination** — Claude Code, Gemini CLI, Codex, and GitHub Copilot all read from and write to the same work items. Orchestration patterns control which agent handles which task.

**Plans & specifications** — CRISPI plans break initiatives into trackable steps. Feature specs define acceptance criteria. Agents execute against the plan and report progress.

## Work Item Types

| Type | Prefix | Purpose |
|------|--------|---------|
| Feature | `feat-` | Units of deliverable work |
| Bug | `bug-` | Defects to fix |
| Spike | `spk-` | Time-boxed investigations |
| Track | `trk-` | Initiatives grouping related work |
| Plan | `plan-` | CRISPI implementation plans |

## CLI Reference

```
htmlgraph help --compact
```

See full CLI documentation at [shakestzd.github.io/htmlgraph](https://shakestzd.github.io/htmlgraph/reference/cli/).

## Contributing

HtmlGraph is developed using HtmlGraph itself (dogfooding). `.htmlgraph/` contains real work items — not demos.

```bash
git clone https://github.com/shakestzd/htmlgraph
cd htmlgraph
go build -o htmlgraph ./cmd/htmlgraph/
./htmlgraph init
```

Quality gates: `go build ./... && go vet ./... && go test ./...`

## License

MIT

## Links

- [Documentation](https://shakestzd.github.io/htmlgraph/)
- [GitHub](https://github.com/shakestzd/htmlgraph)
