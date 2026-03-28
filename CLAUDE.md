# HtmlGraph

Local-first observability and coordination platform for AI-assisted development.

## For AI Agents

**Documentation:** [AGENTS.md](./AGENTS.md) | **Gemini:** [GEMINI.md](./GEMINI.md)

---

## Project Vision

**"Local-first observability and coordination platform for AI-assisted development"**

- **Purpose-built for AI-assisted development** — Natively understands Claude Code sessions, hooks, and agent attribution. Built from the ground up for how developers actually work with AI coding agents, not a generic observability tool adapted for AI.
- HTML files = canonical work item store (features, bugs, spikes, tracks)
- SQLite = operational read index for queries, dashboard, analytics
- Phoenix LiveView = live observability dashboard with real-time event feed (optional, requires Elixir/Erlang runtime)
- No external infrastructure required (no Postgres, no Redis, no cloud)
- 10 Python runtime dependencies (justhtml, pydantic, rich, jinja2, networkx, etc.)

> **Historical note:** The original tagline "HTML is All You Need" reflects the project's design influence -- HTML as a human-readable, git-diffable, browser-viewable storage format. It is not a literal architecture claim.

---

## Orchestrator Mode

**Delegate ALL operations except:** `Task()`, `AskUserQuestion()`, `TodoWrite()`, SDK operations.

**For complete patterns:** Use `/htmlgraph:orchestrator-directives-skill`

### Development Principles

All delegated work must follow: DRY, SRP, KISS, YAGNI. Research existing libraries before implementing. Module size limits enforced (functions <50 lines, modules <500 lines). See agent system prompts for full details.

---

## Code Quality

```bash
uv run ruff check --fix && uv run ruff format && uv run mypy src/ && uv run pytest
# Commit only when ALL pass
```

**For complete workflow:** Use `/htmlgraph:code-quality-skill`

---

## Deployment

```bash
uv run pytest                                   # Run tests
./scripts/deploy-all.sh X.Y.Z --no-confirm      # Deploy
```

See `.claude/rules/deployment.md` for full deployment workflow and options.

---

## Quick Commands

| Task | Command |
|------|---------|
| View work | `htmlgraph snapshot --summary` |
| Run tests | `(cd packages/go && go test ./...)` |
| **Build binary** | **`htmlgraph build`** |
| Deploy | `./scripts/deploy-all.sh VERSION --no-confirm` |
| Serve dashboard | `htmlgraph serve` |
| Status | `htmlgraph status` |
| YOLO session | `htmlgraph yolo --feature <id>` |

---

## Building the Go Binary

**CRITICAL: Always use `htmlgraph build`, never `go build` directly.**

```bash
# Correct — builds to the PATH-linked location
htmlgraph build

# Also correct — calls the same build script
packages/go-plugin/build.sh
```

**NEVER do this:**
```bash
# WRONG — builds to packages/go/htmlgraph, NOT on your PATH
(cd packages/go && go build -o htmlgraph ./cmd/htmlgraph/)
```

### Why This Matters

The binary on your PATH is a symlink chain:
```
.venv/bin/htmlgraph → packages/go-plugin/hooks/bin/htmlgraph
```

`htmlgraph build` outputs to `packages/go-plugin/hooks/bin/htmlgraph` — the symlink target. Running `go build` directly puts the binary in `packages/go/htmlgraph` which is NOT on your PATH. You'll keep running the stale binary.

### How Plugin Users Get the Binary

Plugin users install via `claude plugin install htmlgraph`. The plugin ships with a **bootstrap script** at `hooks/bin/htmlgraph` that:

1. On first run, detects OS/architecture (darwin/linux, amd64/arm64)
2. Downloads the correct pre-built binary from GitHub Releases
3. Caches it at `~/.claude/plugins/data/htmlgraph/htmlgraph-bin`
4. `exec`s into the real binary, passing stdin (CloudEvent JSON) through
5. On subsequent runs, checks cached version against `plugin.json` version — only re-downloads on version mismatch

The bootstrap is a POSIX shell script (~170 lines) that requires only `curl`/`tar`. It never blocks Claude Code — on any error it outputs `{}` and exits 0.

**Binary locations:**
```
Developer:  .venv/bin/htmlgraph → packages/go-plugin/hooks/bin/htmlgraph (built locally)
Plugin user: hooks/bin/htmlgraph (bootstrap script) → ~/.claude/plugins/data/htmlgraph/htmlgraph-bin (downloaded)
```

---

## Development Mode

**CRITICAL: Hooks load htmlgraph from PyPI, not local source, even in dev mode.**

### What Dev Mode Does

Dev mode enables local plugin development by loading the plugin directly from the source directory instead of from the Claude Code marketplace. This allows you to:
- Test changes to commands, agents, skills, and hooks immediately
- Work with the latest code without deploying to PyPI
- Debug plugin functionality in a live Claude Code session

### Starting Dev Mode

```bash
uv run htmlgraph claude --dev
```

This launches Claude Code with:
- Plugin loaded from local source: `packages/claude-plugin/`
- Orchestrator system prompt injected
- Multi-AI delegation rules enabled
- All slash commands available with the plugin namespace prefix

### Plugin Directory Structure

When dev mode runs, it needs to find all plugin components. The structure must be:

```
packages/claude-plugin/              <- PLUGIN ROOT (passed to --plugin-dir)
├── .claude-plugin/
│   └── plugin.json                  <- Plugin manifest
├── commands/                        <- At plugin root (NOT in .claude-plugin)
│   ├── deploy.md
│   ├── init.md
│   ├── plan.md
│   └── ...
├── agents/                          <- At plugin root
│   ├── researcher.md
│   ├── sonnet-coder.md
│   ├── haiku-coder.md
│   └── ...
├── skills/                          <- At plugin root
│   ├── gemini/
│   │   └── SKILL.md                 <- Must be uppercase SKILL.md
│   ├── codex/
│   │   └── SKILL.md
│   └── copilot/
│       └── SKILL.md
├── hooks/                           <- At plugin root
│   ├── hooks.json
│   └── scripts/
│       ├── session-start.py
│       └── ...
└── config/
```

**CRITICAL MISTAKE TO AVOID:** Don't put `commands/`, `agents/`, `skills/`, or `hooks/` inside `.claude-plugin/`. According to Claude Code documentation, only `plugin.json` belongs in `.claude-plugin/`. All other directories must be at the plugin root level.

### How Dev Mode Plugin Loading Works

1. **`get_plugin_dir()` returns the plugin root:** `packages/claude-plugin/`
2. **This directory is passed to Claude Code:** `claude --plugin-dir ./packages/claude-plugin`
3. **Claude Code scans the root directory for:**
   - `.claude-plugin/plugin.json` - Plugin metadata
   - `commands/` - Slash commands (discovered automatically)
   - `agents/` - Agent definitions (discovered automatically)
   - `skills/` - Agent skills with `SKILL.md` files (discovered automatically)
   - `hooks/` - Hook definitions in `hooks.json` (loaded automatically)
4. **Commands appear namespaced:** `/htmlgraph:deploy`, `/htmlgraph:init`, etc.

### Verifying Dev Mode Components

After running `uv run htmlgraph claude --dev`, you should see:

- **Slash commands** visible in `/help`:
  `/htmlgraph:deploy`, `/htmlgraph:init`, `/htmlgraph:plan`, `/htmlgraph:research`, `/htmlgraph:status`, etc.

- **Agent skills** available to Claude when working on relevant tasks (automatic based on context)

- **Hooks** executing based on Claude Code events (PreToolUse, PostToolUse, etc.)

If commands don't appear, verify:
1. `get_plugin_dir()` returns the correct path (root, not `.claude-plugin`)
2. Command files exist in `packages/claude-plugin/commands/`
3. Skill files are named `SKILL.md` (uppercase), not `skill.md`
4. No files are in `.claude-plugin/` except `plugin.json`

### How Hooks Load HtmlGraph

**Hook scripts use PEP 723 inline metadata:**
```python
#!/usr/bin/env -S uv run
# /// script
# requires-python = ">=3.10"
# dependencies = [
#   "htmlgraph>=0.34.15",
# ]
# ///
```

**Key behavior:**
- `uv run` reads the inline `dependencies` block and installs from PyPI
- Even when running from project root, hooks use PyPI package
- Hooks pin a minimum version (e.g., `>=0.34.15`)
- Version pins are updated during deployment via `deploy-all.sh`

### Why PyPI in Dev Mode?

**Testing in production-like environment:**
- Ensures changes work the same way for users
- Catches integration issues before distribution
- No surprises when hooks run in production
- Single source of truth (PyPI package)

### Development Workflow

1. **Make changes** to `src/python/htmlgraph/`
2. **Run tests** locally: `uv run pytest`
3. **Deploy to PyPI**: `./scripts/deploy-all.sh X.Y.Z --no-confirm`
4. **Restart Claude**: Hooks automatically load new version from PyPI
5. **Verify**: Check that changes work correctly

### Troubleshooting Dev Mode

**Hooks not executing?**
- Check PyPI package is latest: `pip show htmlgraph`
- Verify hooks are executable: `ls -la packages/claude-plugin/hooks/scripts/`
- Check hook shebangs: `head -5 packages/claude-plugin/hooks/scripts/*.py`

**Stale hook cache?**
- Clear uv cache: `uv cache clean htmlgraph`
- Hooks may be using a cached older version of the package

**Local changes not reflected?**
- Hooks load from PyPI, not local source
- Must deploy to PyPI for hooks to see changes
- Use incremental versions when deploying

---

## System Prompt Persistence & Delegation Enforcement

**Automatic context injection across session boundaries with cost-optimal delegation.**

Your project's critical guidance (model selection, delegation patterns, quality gates) persists via `.claude/system-prompt.md` and auto-injects at session start, surviving compact/resume cycles.

**Quick Setup**: Create `.claude/system-prompt.md` with project guidance
**Verification**: Run `uv run pytest tests/hooks/test_system_prompt_persistence.py`

### Documentation Guides

| Guide | Audience | Purpose |
|-------|----------|---------|
| [System Prompt Quick Start](./docs/archive/system-prompts/SYSTEM_PROMPT_QUICK_START.md) | Users | Create and customize your system prompt (5-min setup) |
| [System Prompt Architecture](./docs/architecture/system-prompt-architecture.md) | Developers | Deep technical dive + troubleshooting |
| [Delegation Enforcement Admin Guide](./docs/contributing/DELEGATION_ENFORCEMENT_ADMIN_GUIDE.md) | Admins/Teams | Setup and monitor delegation enforcement across your team |
| [System Prompt Developer Guide](./docs/archive/system-prompts/SYSTEM_PROMPT_DEVELOPER_GUIDE.md) | Developers | Extend system with custom layers, hooks, and skills |

**Start here**: [System Prompt Quick Start](./docs/archive/system-prompts/SYSTEM_PROMPT_QUICK_START.md)

---

## Debugging Workflow

**CRITICAL: Research first, implement second.**

```bash
# Built-in debug tools
claude --debug <command>    # Verbose output
/hooks                      # List active hooks
/doctor                     # System diagnostics
```

See `.claude/rules/debugging.md` for the full research-first debugging methodology.

---

## Memory Sync

**Keep documentation synchronized across platforms.**

```bash
uv run htmlgraph sync-docs           # Sync all files
uv run htmlgraph sync-docs --check   # Check sync status
```

---

## Dogfooding

This project uses HtmlGraph to develop HtmlGraph. The `.htmlgraph/` directory contains real usage examples.

---

## Hook & Plugin Development

**CRITICAL: ALL Claude Code integrations (hooks, agents, skills) must be built in the PLUGIN SOURCE.**

**Plugin Source:** `packages/claude-plugin/`
**Do NOT edit:** `.claude/` directory (auto-synced from plugin)

### Plugin Components - What Belongs in the Plugin

Everything that extends Claude Code functionality should be in `packages/claude-plugin/`:

#### 1. **Hooks** (All CloudEvent handlers)
   - **Location:** `packages/claude-plugin/hooks/`
   - **What:** Python scripts that respond to Claude Code events
   - **Scripts:**
     - `session-start.py` - Database session creation
     - `session-resume.py` - Session resumption handling
     - `session-end.py` - Session cleanup
     - `user-prompt-submit.py` - UserQuery event creation
     - `pretooluse-integrator.py` - Track tool use and link to parent activities
     - `posttooluse-integrator.py` - Activity linking
     - `pretooluse-attribution-check.py` - Verify work item attribution
     - `pretooluse-htmlgraph-guard.py` - Guard against .htmlgraph/ edits
     - `posttooluse-failure.py` - Handle tool failures
     - `subagent-start.py` - Subagent launch tracking
     - `subagent-stop.py` - Subagent completion handling
     - `track-event.py` - All event tracking
     - `pre-compact.py` - Pre-compaction handling
     - `instructions-loaded.py` - Instructions load event
     - `permission-request.py` - Permission request handling
   - **Why plugin:** Hooks are Claude Code infrastructure -- must be packaged for distribution

#### 2. **Agents** (Specialized AI agents)
   - **Location:** `packages/claude-plugin/agents/`
   - **What:** Markdown agent definitions with system prompts
   - **Current agents:**
     - `researcher.md` - Research-first documentation investigation
     - `debugger.md` - Systematic error analysis
     - `haiku-coder.md` - Fast, low-cost coding tasks
     - `sonnet-coder.md` - Moderate complexity coding
     - `opus-coder.md` - Deep reasoning for architecture
     - `test-runner.md` - Quality gates and testing
     - `task-executor.md` - General task execution
     - `roborev.md` - Automated code review
   - **Why plugin:** Agents are Claude Code infrastructure -- must be packaged for distribution

#### 3. **Skills** (User-invocable commands)
   - **Location:** `packages/claude-plugin/skills/`
   - **What:** Markdown skill definitions + embedded Python for orchestration
   - **15 skills** including: orchestrator-directives-skill, code-quality-skill, strategic-planning, plan, execute, parallel-status, cleanup, multi-ai-orchestration-skill, gemini, codex, copilot, htmlgraph, htmlgraph-coder, htmlgraph-explorer, roborev
   - **Why plugin:** Skills are Claude Code UI components -- must be packaged for distribution

#### 4. **Plugin Configuration**
   - **Location:** `packages/claude-plugin/.claude-plugin/plugin.json`
   - **What:** Plugin metadata (name, version, description)
   - **Why plugin:** Defines how Claude Code loads and runs the plugin

#### 5. **Configuration & Prompts**
   - **Location:** `packages/claude-plugin/config/`
   - **What:** System prompts, classification rules, drift thresholds
   - **Files:**
     - `classification-prompt.md` - Prompt for work type classification
     - `drift-config.json` - Context drift detection settings
     - `validation-config.json` - Validation configuration
   - **Why plugin:** Shared across all users; updates distributed via plugin

### Directory Structure

```
packages/claude-plugin/                  <-- SOURCE (make changes here)
├── .claude-plugin/
│   └── plugin.json                      <- Plugin manifest
├── hooks/
│   ├── hooks.json                       <- Hook event routing
│   └── scripts/
│       ├── session-start.py             <- Database session creation
│       ├── session-resume.py            <- Session resumption
│       ├── session-end.py               <- Session cleanup
│       ├── user-prompt-submit.py        <- UserQuery event creation
│       ├── pretooluse-integrator.py     <- Tool use tracking
│       ├── posttooluse-integrator.py    <- Activity linking
│       ├── pretooluse-attribution-check.py <- Attribution verification
│       ├── pretooluse-htmlgraph-guard.py   <- .htmlgraph/ edit guard
│       ├── posttooluse-failure.py       <- Tool failure handling
│       ├── subagent-start.py            <- Subagent launch tracking
│       ├── subagent-stop.py             <- Subagent completion
│       ├── track-event.py               <- All event tracking
│       ├── pre-compact.py               <- Pre-compaction handling
│       ├── instructions-loaded.py       <- Instructions load event
│       └── permission-request.py        <- Permission request handling
├── agents/
│   ├── researcher.md
│   ├── debugger.md
│   ├── haiku-coder.md
│   ├── sonnet-coder.md
│   ├── opus-coder.md
│   ├── test-runner.md
│   ├── task-executor.md
│   └── roborev.md
├── skills/                              <- 15 skill directories
│   ├── orchestrator-directives-skill/
│   ├── code-quality-skill/
│   ├── strategic-planning/
│   ├── plan/
│   ├── execute/
│   ├── parallel-status/
│   ├── cleanup/
│   ├── multi-ai-orchestration-skill/
│   ├── gemini/
│   ├── codex/
│   ├── copilot/
│   ├── htmlgraph/
│   ├── htmlgraph-coder/
│   ├── htmlgraph-explorer/
│   └── roborev/
├── commands/                            <- 19 slash commands
├── config/
│   ├── classification-prompt.md
│   ├── drift-config.json
│   └── validation-config.json
└── README.md

.claude/  <-- AUTO-SYNCED (do not edit)
├── hooks/ (synced from plugin)
├── skills/ (synced from plugin)
└── config/ (synced from plugin)
```

### Critical Rule: Single Source of Truth

**NEVER edit `.claude/` expecting changes to persist.**

- Do NOT edit `.claude/hooks/hooks.json` -- changes lost on plugin update
- Do NOT edit `.claude/hooks/scripts/*.py` -- changes lost on plugin update
- Do NOT edit `.claude/agents/` -- changes lost on plugin update
- Do NOT add hooks to `.claude/` -- not published, not shareable

**ALWAYS edit in plugin source:**

- Edit `packages/claude-plugin/hooks/hooks.json`
- Edit `packages/claude-plugin/hooks/scripts/*.py`
- Add agents to `packages/claude-plugin/agents/`
- Add skills to `packages/claude-plugin/skills/`

### Workflow: Making Changes to Plugin

1. **Make changes in plugin source:**
   ```bash
   # Edit files in packages/claude-plugin/
   vim packages/claude-plugin/hooks/scripts/user-prompt-submit.py
   vim packages/claude-plugin/.claude-plugin/plugin.json
   ```

2. **Run quality checks:**
   ```bash
   uv run ruff check --fix && uv run ruff format && uv run mypy src/ && uv run pytest
   ```

3. **Verify plugin is synced (in dev mode, hooks run from plugin source):**
   ```bash
   # In dev mode, Claude Code runs hooks from plugin source directly
   # No need to manually sync during development
   ```

4. **Commit changes:**
   ```bash
   git add packages/claude-plugin/
   git commit -m "fix: update hook X with Y changes"
   ```

5. **Deploy (publishes plugin update):**
   ```bash
   ./scripts/deploy-all.sh X.Y.Z --no-confirm
   # This updates version in plugin.json and publishes to distribution
   ```

### Never Do This

- Edit `.claude/hooks/hooks.json` directly
- Edit `.claude/hooks/scripts/*.py` directly
- Edit `.claude/agents/` directly
- Add new hooks to `.claude/` expecting them to run
- Make changes to `.claude/` expecting them to persist

### Always Do This

- Edit `packages/claude-plugin/hooks/hooks.json`
- Edit `packages/claude-plugin/hooks/scripts/*.py`
- Add agents to `packages/claude-plugin/agents/`
- Add skills to `packages/claude-plugin/skills/`
- Commit plugin source files
- Test in dev mode (hooks run from plugin automatically)

---

## Project vs General Tooling

**This project is both:**
1. **HtmlGraph Package Development** - Building the tool itself
2. **HtmlGraph Dogfooding** - Using the tool to build itself

**CLAUDE.md contains:**
- Project-specific: Deployment, testing, debugging HtmlGraph package
- Quick reference: Links to skills for general patterns

**Plugin/Skills contain:**
- General patterns: Orchestration, coordination (for all users)
- Progressive disclosure: Load details on-demand

---

## Skills Reference

| Skill | Use For |
|-------|---------|
| `/htmlgraph:orchestrator-directives-skill` | Delegation patterns, decision framework |
| `/htmlgraph:code-quality-skill` | Lint, type check, testing workflow |
| `/htmlgraph:strategic-planning` | Work prioritization, bottleneck analysis |
| `/htmlgraph:plan` | Parallel development planning |
| `/htmlgraph:execute` | Parallel execution with worktrees |
| `/htmlgraph:parallel-status` | Monitor parallel execution progress |
| `/htmlgraph:cleanup` | Clean up worktrees and branches |
| `/htmlgraph:multi-ai-orchestration-skill` | Multi-AI spawner coordination |
| `/htmlgraph:gemini` | GeminiSpawner with tracking |
| `/htmlgraph:codex` | CodexSpawner with tracking |
| `/htmlgraph:copilot` | GitHub CLI + CopilotSpawner |
| `/htmlgraph:htmlgraph` | Core HtmlGraph workflow |
| `/htmlgraph:roborev` | Automated code review |
| `/htmlgraph:error-analysis` | Error investigation workflow |

---

## Rules Reference

Detailed rules in `.claude/rules/`:
- `code-hygiene.md` - Quality standards and module size limits
- `deployment.md` - Release workflow and publishing
- `debugging.md` - Research-first debug methodology
- `dogfooding.md` - Self-hosting context
- `version-sync.md` - Version synchronization rules
