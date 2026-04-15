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
- Prefer file/branch state over session state for hook gates (see "Hook State" below)

## Hook State: Prefer File/Branch State Over Session State

**Rule:** Hooks should answer questions from durable state (files, branches, staged diff)
rather than session state (session_id, tool-name history, task IDs). Session state is
brittle — tool names change, sessions rotate, and substring matches fail silently.

**Why it matters.** Two bugs shared the same root cause shape:

- GH#35 asked "is this session the claimant?" — the right question was "does this branch
  correspond to a valid claim?" Checking session_id against a claims table caused false
  blocks when sessions rotated (e.g. after reconnect).
- GH#36 (bug-a10ae96a, fixed in commit 88d2f51b4) asked "did someone call a tool matching
  `%screenshot%`?" — the right question was "does the staged diff contain UI files?" The
  LIKE match never fired because the Chrome MCP tool name didn't contain "screenshot".
  Moving the gate to `git diff --cached --name-only` fixed it cleanly.

Both bugs produced unsatisfiable checks or false positives on unrelated commands. The cost
is invisible failures: the guard either never fires (permissive) or fires on the wrong
trigger (noisy).

**How to apply:**

1. Before adding a hook, write down the question it answers. If the answer lives in
   `git diff --cached`, the branch name, the working tree, or config files — use that.
   Don't approximate it from session history.
2. Session state is fine for *attribution* and *telemetry*, not for *gates*. Gates must
   be reproducible across session restarts; telemetry just needs to be eventually consistent.
3. When blocking a commit, first check the staged diff. Short-circuit to allow when no
   relevant files are staged — see `hasStagedUIFiles()` in `internal/hooks/yolo_guard.go`
   and commit `88d2f51b4` as the canonical pattern.
4. Anchor Bash command matching to the specific command shape, not substrings.
   Use `^\s*git\s+commit(\s|$|--|-[^a-z])`, not `LIKE '%commit%'`.
