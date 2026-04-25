---
name: External CLI Integration (Codex, Gemini, Copilot)
description: Capabilities, invocation flags, and current roles for the three external AI CLIs used in the htmlgraph workflow
type: reference
---

# External CLI Integration

All three CLIs are installed at `$HOME/.nvm/versions/node/v22.20.0/bin/`.

## Codex CLI

**Role in htmlgraph:** Code implementation, testing, refactoring — "70% cheaper than Claude"

**Non-interactive invocation:**
```bash
codex exec "<prompt>" --full-auto --json -m gpt-4.1-mini -C . 2>&1
```

**Key flags:**
- `exec` subcommand — runs non-interactively
- `--full-auto` — alias for sandboxed auto-approval (low friction)
- `--json` — outputs NDJSON stream (type/item events)
- `-m <model>` — model selection (e.g. gpt-4.1-mini, o3)
- `-C <dir>` — working directory
- `--dangerously-bypass-approvals-and-sandbox` — skip all sandboxing (use in already-sandboxed envs)

**JSON output format:** NDJSON stream with `type` field:
- `thread.started`, `turn.started`, `item.completed` (agent_message or command_execution), `turn.completed`
- `turn.completed` includes token usage stats

**Performance tuning (researched 2026-04-03):**

- `--json` is a **real-time JSONL stream** (not buffered) — events arrive as they happen; pipe to `jq` or process line-by-line for partial results before Codex finishes
- `--sandbox read-only` does **not** reduce setup overhead — sandbox level affects write permissions only, not startup cost; both modes initialize equally fast
- `--ephemeral` saves ~4% tokens (~341 tokens) by skipping session context — negligible speed gain, mainly useful to avoid disk I/O accumulation over thousands of runs
- `model_reasoning_effort = "low"` (or `minimal`) is the single biggest speed lever for o4-mini/o3 class models — set per-run with `-c model_reasoning_effort=low`
- System prompt is ~99.4% cached after first run — repeated invocations are fast; **switching models invalidates 59% of cache**, so stick to one model per workflow
- `--disable shell_tool` saves ~440 tokens from system prompt — use for pure code-review prompts that don't need shell commands
- `service_tier = "fast"` in config.toml (vs `flex`) may reduce queue latency on paid tiers
- `web_search = "cached"` in config.toml uses index instead of live fetch — faster if web access isn't needed
- `features.shell_snapshot = true` (default) snapshots shell env to speed up repeated commands

**Fastest flags for read-only code review:**
```bash
codex exec "<prompt>" \
  --sandbox read-only \
  --ephemeral \
  --json \
  -m gpt-4.1-mini \
  -c model_reasoning_effort=low \
  --disable shell_tool \
  -C .
```

**JSONL event sequence:** `thread.started` → `turn.started` → `item.completed` (agent_message or reasoning) → `turn.completed` (includes token usage). Pipe with `| grep '"type":"item.completed"' | jq '.item.content'` to extract partial output.

**Current plan/skill usage:**
- `plugin/skills/plan/SKILL.md` line 278: feasibility check — scaffolds stubs, runs `go build ./...`
- `plugin/skills/orchestrator-directives-skill/SKILL.md`: implementation tasks (priority 2 after Gemini)
- `orchestrator-directives-skill/reference.md`: multiple examples with `--full-auto --json -m gpt-4.1-mini`

**Known issue:** Stale symlink error at start (`failed to stat skills entry $HOME/.codex/skills/htmlgraph-tracker`) — non-fatal, doesn't affect execution.

---

## Gemini CLI

**Role in htmlgraph:** Exploration, research, file analysis, documentation — "FREE, 2M tokens/min"

**Non-interactive invocation:**
```bash
gemini -p "<prompt>" --yolo --output-format json 2>&1
# or
gemini -p "<prompt>" --yolo --output-format text --include-directories . 2>&1
```

**Key flags:**
- `-p` / `--prompt` — non-interactive (headless) mode
- `-y` / `--yolo` — auto-approve all tool calls
- `--approval-mode` — choices: default, auto_edit, yolo, plan (read-only)
- `--output-format` — text, json, stream-json
- `--include-directories` — add workspace directories
- `-m` / `--model` — model selection
- `--policy` — additional policy files

**Current plan/skill usage:**
- `plugin/skills/plan/SKILL.md` line 272-274: design critique — scope, file coverage, dependencies, test strategy, risks per slice
- `orchestrator-directives-skill/SKILL.md`: research/exploration tasks (priority 1 — FREE)
- `reference.md`: multiple patterns for research, debugging, documentation

**Known noise:** Emits extension/MCP error lines to stderr on startup (htmlgraph hooks config issue, clasp import). These are non-fatal. Filter with `grep -v "^\[ERROR\]"` or use `2>/dev/null` for clean output.

---

## Copilot CLI

**Role in htmlgraph:** Git operations, PRs, GitHub integration — "60% cheaper, GitHub-native"

**Non-interactive invocation:**
```bash
copilot -p "<prompt>" --allow-all-tools --no-color --add-dir . 2>&1
```

**Key flags:**
- `-p` — prompt (non-interactive mode, required with `--allow-all-tools` for headless)
- `--allow-all-tools` — auto-approve all tools (required for non-interactive)
- `--no-color` — clean output for parsing
- `--add-dir <dir>` — grant file access to directory
- `--allow-all-paths` — disable path verification
- `--deny-tool` / `--allow-tool` — granular tool control
- `--continue` — resume most recent session
- `--model` — model selection (default: claude-sonnet-4.5 as observed)

**Underlying model:** Uses Claude Sonnet 4.5 (observed from usage stats). Not actually cheaper in API cost terms — "cheaper" likely refers to Claude Code quota usage since it runs in its own process.

**Current plan/skill usage:**
- `orchestrator-directives-skill/SKILL.md` line 55: git/GitHub ops (priority 3)
- `reference.md`: commit, PR creation, branch management examples
- The "copilot skill" is the primary git delegation mechanism

**Output:** Includes usage stats footer (model, token counts, duration) — parse or strip as needed.
