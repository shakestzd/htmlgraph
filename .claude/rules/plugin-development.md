---
paths:
  - "plugin/**"
  - "cmd/**"
  - "internal/**"
---

# Plugin Development

**Source of truth:** `plugin/` — never edit `.claude/` directly (auto-synced from plugin).

## Directory Structure

- `plugin/.claude-plugin/plugin.json` — manifest
- `plugin/hooks/hooks.json` + `bin/htmlgraph` — Go binary hook handler
- `plugin/agents/` — markdown agent definitions
- `plugin/skills/` — skill directories with SKILL.md
- `plugin/commands/` — slash commands
- `plugin/config/` — classification, drift, validation

**CRITICAL:** Don't put `commands/`, `agents/`, `skills/`, or `hooks/` inside `.claude-plugin/`. Only `plugin.json` belongs there.

## Workflow

1. Edit files in `plugin/`, `cmd/`, or `internal/`
2. Run: `go build ./... && go vet ./... && go test ./...`
3. Build: `htmlgraph build`
4. Test: `htmlgraph claude --dev`
5. Deploy: `./scripts/deploy-all.sh X.Y.Z --no-confirm`

## Rules

- Edit `plugin/hooks/hooks.json`, never `.claude/hooks/hooks.json`
- Edit Go source in `cmd/` or `internal/` for hook/CLI logic
- Add agents to `plugin/agents/`, skills to `plugin/skills/`
- Hooks receive CloudEvent JSON on stdin — process via Go binary
- No stderr from hooks (causes "hook error" in Claude Code UI)
- Return `{}` to allow, `{"decision":"block","reason":"..."}` to block
