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
- Single Go binary — zero runtime dependencies

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
(cd packages/go && go build ./... && go vet ./... && go test ./...)
# Commit only when ALL pass
```

**For complete workflow:** Use `/htmlgraph:code-quality-skill`

---

## Deployment

```bash
(cd packages/go && go test ./...)                # Run tests
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

The binary on your PATH is set up via a symlink:
```
~/.local/bin/htmlgraph → packages/go-plugin/hooks/bin/htmlgraph
```

`htmlgraph build` outputs to `packages/go-plugin/hooks/bin/htmlgraph` — the symlink target. Running `go build` directly puts the binary in `packages/go/htmlgraph` which is NOT on your PATH. You'll keep running the stale binary.

### How Plugin Users Get the Binary

**Recommended install path (CLI-first):** Install the CLI binary first via Homebrew, shell script, or `go install`, then run `htmlgraph plugin install` to add the Claude Code plugin. The bootstrap script detects a PATH-installed binary and uses it directly, skipping the download.

```bash
# 1. Install CLI
brew install shakestzd/tap/htmlgraph   # or curl install script or go install

# 2. Install plugin (optional — adds Claude Code hooks and agents)
htmlgraph plugin install
```

Plugin-only users (no CLI pre-installed) install via `claude plugin install htmlgraph`. The plugin ships with a **bootstrap script** at `hooks/bin/htmlgraph` that:

1. On first run, checks whether `htmlgraph` is already on PATH — uses it if so
2. Otherwise detects OS/architecture (darwin/linux, amd64/arm64)
3. Downloads the correct pre-built binary from GitHub Releases
4. Caches it at `~/.claude/plugins/data/htmlgraph/htmlgraph-bin`
5. `exec`s into the real binary, passing stdin (CloudEvent JSON) through
6. On subsequent runs, checks cached version against `plugin.json` version — only re-downloads on version mismatch

The bootstrap is a POSIX shell script (~170 lines) that requires only `curl`/`tar`. It never blocks Claude Code — on any error it outputs `{}` and exits 0.

**Binary locations:**
```
CLI install:  /usr/local/bin/htmlgraph (brew) or ~/.local/bin/htmlgraph (go install / shell script)
Developer:    ~/.local/bin/htmlgraph → packages/go-plugin/hooks/bin/htmlgraph (built locally via setup-cli)
Plugin-only:  hooks/bin/htmlgraph (bootstrap script) → ~/.claude/plugins/data/htmlgraph/htmlgraph-bin (downloaded)
```

---

## Development Mode

Dev mode enables local plugin development by loading the plugin directly from the source directory instead of from the Claude Code marketplace.

### Starting Dev Mode

```bash
htmlgraph claude --dev
```

This launches Claude Code with:
- Plugin loaded from local source: `packages/go-plugin/`
- Orchestrator system prompt injected
- Multi-AI delegation rules enabled
- All slash commands available with the plugin namespace prefix

### Plugin Directory Structure

```
packages/go-plugin/                  <- PLUGIN ROOT (passed to --plugin-dir)
├── .claude-plugin/
│   └── plugin.json                  <- Plugin manifest
├── commands/                        <- At plugin root (NOT in .claude-plugin)
├── agents/                          <- At plugin root
├── skills/                          <- At plugin root (SKILL.md files)
├── hooks/
│   ├── hooks.json                   <- Hook event routing
│   └── bin/htmlgraph                <- Go binary hook handler
└── config/
```

**CRITICAL MISTAKE TO AVOID:** Don't put `commands/`, `agents/`, `skills/`, or `hooks/` inside `.claude-plugin/`. Only `plugin.json` belongs in `.claude-plugin/`.

### How Hooks Work

Hooks are handled by the Go binary at `hooks/bin/htmlgraph`. The binary receives CloudEvent JSON on stdin and processes events (session start/end, tool use tracking, attribution checks, etc.).

### Development Workflow

1. **Make changes** to `packages/go/`
2. **Run tests**: `(cd packages/go && go test ./...)`
3. **Build binary**: `htmlgraph build`
4. **Test in dev mode**: `htmlgraph claude --dev`
5. **Deploy**: `./scripts/deploy-all.sh X.Y.Z --no-confirm`

---

## System Prompt Persistence & Delegation Enforcement

**Automatic context injection across session boundaries with cost-optimal delegation.**

Your project's critical guidance (model selection, delegation patterns, quality gates) persists via `.claude/system-prompt.md` and auto-injects at session start, surviving compact/resume cycles.

**Quick Setup**: Create `.claude/system-prompt.md` with project guidance

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
htmlgraph sync-docs           # Sync all files
htmlgraph sync-docs --check   # Check sync status
```

---

## Dogfooding

This project uses HtmlGraph to develop HtmlGraph. The `.htmlgraph/` directory contains real usage examples.

---

## Hook & Plugin Development

**CRITICAL: ALL Claude Code integrations (hooks, agents, skills) must be built in the PLUGIN SOURCE.**

**Plugin Source:** `packages/go-plugin/`
**Do NOT edit:** `.claude/` directory (auto-synced from plugin)

### Plugin Components - What Belongs in the Plugin

Everything that extends Claude Code functionality should be in `packages/go-plugin/`:

#### 1. **Hooks** (All CloudEvent handlers)
   - **Location:** `packages/go-plugin/hooks/`
   - **What:** Go binary that processes CloudEvent JSON on stdin
   - **Events handled:** session start/resume/end, tool use tracking, attribution checks, subagent tracking, compaction, permission requests
   - **Why plugin:** Hooks are Claude Code infrastructure -- must be packaged for distribution

#### 2. **Agents** (Specialized AI agents)
   - **Location:** `packages/go-plugin/agents/`
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
   - **Location:** `packages/go-plugin/skills/`
   - **What:** Markdown skill definitions for orchestration
   - **15 skills** including: orchestrator-directives-skill, code-quality-skill, strategic-planning, plan, execute, parallel-status, cleanup, multi-ai-orchestration-skill, gemini, codex, copilot, htmlgraph, htmlgraph-coder, htmlgraph-explorer, roborev
   - **Why plugin:** Skills are Claude Code UI components -- must be packaged for distribution

#### 4. **Plugin Configuration**
   - **Location:** `packages/go-plugin/.claude-plugin/plugin.json`
   - **What:** Plugin metadata (name, version, description)
   - **Why plugin:** Defines how Claude Code loads and runs the plugin

#### 5. **Configuration & Prompts**
   - **Location:** `packages/go-plugin/config/`
   - **What:** System prompts, classification rules, drift thresholds
   - **Files:**
     - `classification-prompt.md` - Prompt for work type classification
     - `drift-config.json` - Context drift detection settings
     - `validation-config.json` - Validation configuration
   - **Why plugin:** Shared across all users; updates distributed via plugin

### Directory Structure

```
packages/go-plugin/                      <-- PLUGIN SOURCE (make changes here)
├── .claude-plugin/
│   └── plugin.json                      <- Plugin manifest
├── hooks/
│   ├── hooks.json                       <- Hook event routing
│   └── bin/htmlgraph                    <- Go binary hook handler
├── agents/                              <- Markdown agent definitions
├── skills/                              <- Skill directories with SKILL.md
├── commands/                            <- Slash commands
├── config/                              <- Classification, drift, validation
└── README.md

packages/go/                             <-- GO SOURCE (core logic)
├── cmd/htmlgraph/                       <- CLI entry point
└── internal/                            <- Business logic packages

.claude/  <-- AUTO-SYNCED (do not edit)
├── hooks/ (synced from plugin)
├── skills/ (synced from plugin)
└── config/ (synced from plugin)
```

### Critical Rule: Single Source of Truth

**NEVER edit `.claude/` expecting changes to persist.**

- Do NOT edit `.claude/hooks/hooks.json` -- changes lost on plugin update
- Do NOT edit `.claude/agents/` -- changes lost on plugin update
- Do NOT add hooks to `.claude/` -- not published, not shareable

**ALWAYS edit in plugin source:**

- Edit `packages/go-plugin/hooks/hooks.json`
- Edit Go source in `packages/go/` for hook logic
- Add agents to `packages/go-plugin/agents/`
- Add skills to `packages/go-plugin/skills/`

### Workflow: Making Changes to Plugin

1. **Make changes in plugin source:**
   ```bash
   # Edit files in packages/go-plugin/ or packages/go/
   vim packages/go-plugin/.claude-plugin/plugin.json
   vim packages/go/cmd/htmlgraph/reindex.go
   ```

2. **Run quality checks:**
   ```bash
   (cd packages/go && go build ./... && go vet ./... && go test ./...)
   ```

3. **Verify plugin is synced (in dev mode, hooks run from plugin source):**
   ```bash
   # In dev mode, Claude Code runs hooks from plugin source directly
   # No need to manually sync during development
   ```

4. **Commit changes:**
   ```bash
   git add packages/go-plugin/
   git commit -m "fix: update hook X with Y changes"
   ```

5. **Deploy (publishes plugin update):**
   ```bash
   ./scripts/deploy-all.sh X.Y.Z --no-confirm
   # This updates version in plugin.json and publishes to distribution
   ```

### Never Do This

- Edit `.claude/hooks/hooks.json` directly
- Edit `.claude/agents/` directly
- Add new hooks to `.claude/` expecting them to run
- Make changes to `.claude/` expecting them to persist

### Always Do This

- Edit `packages/go-plugin/hooks/hooks.json`
- Edit Go source in `packages/go/` for hook/CLI logic
- Add agents to `packages/go-plugin/agents/`
- Add skills to `packages/go-plugin/skills/`
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
