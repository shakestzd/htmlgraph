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
htmlgraph claude --dev   # loads plugin from source via --plugin-dir
```

Dev mode uninstalls the marketplace plugin, clears cache, and launches with `claude --plugin-dir plugin/`. This ensures agents, skills, tools, and hooks all load from your local source — not stale marketplace copies. The marketplace plugin is reinstalled on exit.

**Why full removal is required:** Disabling a marketplace plugin only affects hooks. Agent definitions and skill content continue loading from `~/.claude/plugins/marketplaces/`, silently shadowing dev source changes.

## Resuming a Specific Session

`htmlgraph claude`, `htmlgraph yolo`, and `htmlgraph dev` all accept `--resume <session-id>` to resume a specific prior Claude Code session. On exit, Claude Code prints a line like `claude --resume d846b50d-…`; pass that ID through the htmlgraph launcher to get the HtmlGraph plugin, system prompt, and (in `--dev`) local `--plugin-dir` applied to the resumed session:

```bash
htmlgraph claude --resume d846b50d-9ce4-45c1-8ad2-0f84da537efd
htmlgraph claude --dev --resume <session-id>
htmlgraph yolo --dev --resume <session-id>
htmlgraph dev --resume <session-id>
```

`--resume <id>` differs from `--continue` (which resumes the most recent session). If both are passed, `--resume <id>` wins.

## Dev Mode in Codespaces

Codespaces clients disconnect on idle, browser refresh, or network blips — killing long dev sessions. Wrap dev in tmux:

    htmlgraph claude --dev --tmux

First run creates a tmux session named `htmlgraph-dev`. On disconnect, detach instead of dying. Re-run the same command to reattach to the surviving session. Manually detach with `Ctrl-b d`; kill the session with `tmux kill-session -t htmlgraph-dev`.

## Yolo Mode in Codespaces

Codespaces clients disconnect on idle, browser refresh, or network blips — killing long yolo runs. Wrap yolo in tmux:

    htmlgraph yolo --dev --tmux

First run creates a tmux session named `htmlgraph-yolo`. On disconnect, detach instead of dying. Re-run the same command to reattach to the surviving session. Manually detach with `Ctrl-b d`; kill the session with `tmux kill-session -t htmlgraph-yolo`.

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
| Self-update | `htmlgraph upgrade` |

---

## Monitoring Claude Code Upstream

Claude Code is our only integration surface. Plugins, hooks, skills, slash commands, and observability (logging, events, sessions) are how HtmlGraph influences behavior — if upstream changes those contracts, our plugin either breaks silently or misses new capabilities.

**Periodically review the Claude Code docs for changes to:**

- **Plugin system** — manifest format, `plugin.json` schema, marketplace structure
- **Hooks** — event names, payload shapes, exit-code semantics, new hook types
- **Skills** — frontmatter fields, activation triggers, invocation patterns
- **Agent teams** — still experimental as of last review; watch for graduation out of `CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1` (v2.1.32+), API stability, nested-team support
- **Observability** — session metadata, telemetry, cost/token reporting, transcript format

**Upstream sources to monitor:**

- https://code.claude.com/docs/en/plugins
- https://code.claude.com/docs/en/hooks
- https://code.claude.com/docs/en/skills
- https://code.claude.com/docs/en/agent-teams
- Claude Code release notes / changelog

When upstream contracts change, the fix lands in `plugin/hooks/hooks.json`, `internal/hooks/`, `plugin/skills/`, or `cmd/htmlgraph/prompts/system-prompt.md` — not in AGENTS.md or CLAUDE.md (which are user-facing project docs, not plugin surfaces).

---

## Dogfooding

This project uses HtmlGraph to develop itself. `.htmlgraph/` contains real work items — not demos.
