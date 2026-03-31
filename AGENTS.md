# HtmlGraph for AI Agents

**CRITICAL: AI agents must NEVER edit `.htmlgraph/` HTML files directly.**

Use the CLI or REST API instead. This ensures all HTML is validated through the Go binary.

---

## NOTE: Dogfooding in Action

**IF YOU'RE WORKING ON THE HTMLGRAPH PROJECT ITSELF:**

This project uses HtmlGraph to track its own development. The `.htmlgraph/` directory in this repo is:
- ✅ **Real usage** - Not a demo, actual development tracking
- ✅ **Live examples** - Learn from these patterns for YOUR projects
- ✅ **Our roadmap** - Features we're building for HtmlGraph

**See [CLAUDE.md#dogfooding-context](./CLAUDE.md#dogfooding-context) for full details** on:
- What's general-purpose vs project-specific
- Workflows we should package for all users
- How to distinguish HtmlGraph development from HtmlGraph usage

**IF YOU'RE USING HTMLGRAPH IN YOUR OWN PROJECT:**

Ignore the HtmlGraph-specific features in `.htmlgraph/`. Focus on:
- ✅ CLI patterns shown below
- ✅ Workflow examples (they work for ANY project)
- ✅ Best practices (universal)

---

## Quick Start (CLI)

```bash
# Initialize project
htmlgraph init --install-hooks

# Get project status
htmlgraph snapshot --summary

# Create a feature
htmlgraph feature create "User Authentication"
# Returns: feat-abc12345

# Start working on it
htmlgraph feature start feat-abc12345

# List high-priority todo features
htmlgraph find features --status todo --priority high
```

### Delegating Complex Work with Task()

For complex tasks that require multiple operations, delegate to subagents to preserve your context. This is especially useful when you need to:
- Run multiple test suites in parallel
- Explore a large codebase (many Grep/Glob calls)
- Make coordinated changes across many files

**Example: Delegating test execution**

```python
# Delegate test runs to parallel subagents
Task(subagent_type="general-purpose",
     prompt="Run unit tests in tests/unit/ and report failures")

Task(subagent_type="general-purpose",
     prompt="Run integration tests in tests/integration/ and report failures")

# Orchestrator preserves context while subagents work in parallel
```

**Parent-Child Session Tracking**

HtmlGraph automatically links parent and child sessions. View session history:
```bash
# List sessions
htmlgraph session list

# Find sessions linked to a feature
htmlgraph session list  # filter by feature context
```

→ [Complete delegation guide](docs/guide/delegation.md) - Best practices, patterns, cost optimization

**New to HtmlGraph?** See [Architecture Guide](./docs/ARCHITECTURE.md) for design philosophy and common patterns.

---

## Core Principle: NEVER Edit HTML Directly

❌ **FORBIDDEN:**
```python
# NEVER DO THIS
with open(".htmlgraph/features/feature-123.html", "w") as f:
    f.write("<html>...</html>")

# NEVER DO THIS
Edit("/path/to/.htmlgraph/features/feature-123.html", ...)
```

✅ **REQUIRED - Use CLI:**
```bash
# Complete a feature
htmlgraph feature complete feature-123

# Start a feature
htmlgraph feature start feature-123
```

**Why this matters:**
- Direct edits bypass HTML validation
- Break SQLite index sync
- Can corrupt graph structure
- Skip event logging

---

## CLI Reference

### Installation

```bash
pip install htmlgraph
# or
uv pip install htmlgraph
```

### Get Oriented

```bash
# Project snapshot
htmlgraph snapshot --summary

# My in-progress work
htmlgraph find features --status in-progress
```

### Feature Commands

```bash
# Create
htmlgraph feature create "Title"         # Returns feat-<id>

# Read
htmlgraph feature show feat-abc12345     # Show feature details
htmlgraph feature list                   # List all features

# Update state
htmlgraph feature start feat-abc12345    # Mark in-progress
htmlgraph feature complete feat-abc12345 # Mark done

# Query
htmlgraph find features --status todo
htmlgraph find features --status in-progress
```

### Bug Commands

```bash
htmlgraph bug create "Bug title"
htmlgraph bug list
```

### Spike Commands

```bash
htmlgraph spike create "Investigation title"
htmlgraph spike list
```

### Track Commands

```bash
htmlgraph track new "Track title"
htmlgraph track list
```

### Analytics

```bash
htmlgraph analytics recommend      # Recommended next work
htmlgraph analytics bottlenecks    # Find blockers
```

### Version

```bash
htmlgraph version
```

---

## REST API (Alternative)

### Start Server

```bash
uvx htmlgraph serve
# Open http://localhost:8080
```

### Endpoints

#### Get All Features
```bash
curl http://localhost:8080/api/query?type=feature
```

#### Get Feature by ID
```bash
curl http://localhost:8080/api/features/feature-001
```

#### Create Feature
```bash
curl -X POST http://localhost:8080/api/features \
  -H "Content-Type: application/json" \
  -d '{
    "title": "User Authentication",
    "priority": "high",
    "status": "todo",
    "steps": [
      {"description": "Create login endpoint"},
      {"description": "Add JWT middleware"}
    ]
  }'
```

#### Update Feature
```bash
curl -X PATCH http://localhost:8080/api/features/feature-001 \
  -H "Content-Type: application/json" \
  -d '{"status": "in-progress"}'
```

#### Complete Step
```bash
curl -X PATCH http://localhost:8080/api/features/feature-001 \
  -H "Content-Type: application/json" \
  -d '{"complete_step": 0}'
```

**Step numbering is 0-based** (first step = 0, second step = 1, etc.)

---

## CLI (Alternative)

**IMPORTANT:** Use `uvx htmlgraph` to always get the latest installed version.
If you are working inside the `htmlgraph` project itself (developing HtmlGraph), use `uv run htmlgraph` instead.

### Check Status
```bash
uvx htmlgraph status
uvx htmlgraph feature list
```

### Start Feature
```bash
uvx htmlgraph feature start <feature-id>
```

### Set Primary Feature
```bash
# When multiple features are active
uvx htmlgraph feature primary <feature-id>
```

### Complete Feature
```bash
uvx htmlgraph feature complete <feature-id>
```

### Server
```bash
uvx htmlgraph serve
```

---

## Decision Matrix: CLI vs API

| Use Case | Recommended Interface |
|----------|----------------------|
| AI agent work tracking | **CLI** (stateless, fast) |
| Scripting/automation | CLI |
| Manual testing | CLI or Dashboard |
| External integration | REST API |
| Debugging | CLI + Dashboard |

---

## Best Practices for AI Agents

### 1. Always Use CLI for Work Tracking

```bash
# ✅ GOOD - Use CLI commands
htmlgraph feature create "Title"
# Returns feat-abc12345
htmlgraph feature start feat-abc12345

# ❌ BAD - Don't edit .htmlgraph/ files directly
# Edit("/path/to/.htmlgraph/features/feat-abc.html", ...)
```

---

## Debugging & Quality

**See [.claude/rules/debugging.md](./.claude/rules/debugging.md) for the complete debugging guide**

HtmlGraph provides specialized debugging agents for systematic problem-solving:

### Debugging Agents

- **Researcher Agent** (`packages/claude-plugin/agents/researcher.md`)
  - Research documentation BEFORE implementing solutions
  - Use for: Unfamiliar errors, Claude Code hooks/plugins, multiple failed attempts

- **Debugger Agent** (`packages/claude-plugin/agents/debugger.md`)
  - Systematically analyze and resolve errors
  - Use for: Known errors, test failures, reproduction needed

- **Test Runner Agent** (`packages/claude-plugin/agents/test-runner.md`)
  - Validate all changes, enforce quality gates
  - Use for: Pre-commit validation, deployment, regression prevention

### Tool Selection Matrix

| Scenario | Use This Agent | Why |
|----------|----------------|-----|
| Unfamiliar error | Researcher | Research docs first |
| Claude Code hooks issue | Researcher | Official guidance needed |
| Error with known cause | Debugger | Systematic root cause analysis |
| Before committing | Test Runner | Validate quality gates |
| Multiple failed attempts | Researcher | Stop guessing, start researching |

### Quick Reference

```bash
# Research first
packages/claude-plugin/agents/researcher.md

# Debug systematically
packages/claude-plugin/agents/debugger.md

# Validate changes
packages/claude-plugin/agents/test-runner.md
```

---

### 2. Check Status Before Working

```bash
# Get orientation
htmlgraph snapshot --summary

# Check in-progress work
htmlgraph find features --status in-progress
```

### 3. Use CLI for Queries

```bash
# ✅ GOOD
htmlgraph find features --status todo
htmlgraph find features --status in-progress
```

### 4. Complete Features Properly

```bash
# ✅ GOOD - Use CLI to close out work
htmlgraph feature complete feat-001
htmlgraph feature complete feat-002
htmlgraph feature complete feat-003
```

---

## Complete Workflow Example

```bash
# 1. Get oriented
htmlgraph snapshot --summary

# 2. Check in-progress work
htmlgraph find features --status in-progress

# 3. Get recommended next work
htmlgraph analytics recommend

# 4. Start working on a feature
htmlgraph feature start feat-abc12345

# 5. (Do the actual implementation work...)

# 6. Complete the feature
htmlgraph feature complete feat-abc12345
```

---

## Orchestrator Mode

### What is Orchestrator Mode?

Orchestrator Mode is an **enforcement system** that guides AI agents to delegate low-cognitive, context-filling work to specialized subagents using the Task tool. When enabled, certain operations are blocked or warned against to encourage efficient workflow patterns.

**Key Principles:**
- **Context preservation** - Keep orchestrator context for high-level decisions
- **Parallel execution** - Delegate to subagents for concurrent work
- **Pattern enforcement** - Block operations that fill context unnecessarily
- **Progressive guidance** - Start with warnings, escalate to blocks

### Quick Start

```bash
# Enable orchestrator mode (strict enforcement)
uvx htmlgraph orchestrator enable

# Enable with guidance only (warnings, no blocks)
uvx htmlgraph orchestrator enable --mode guidance

# Check current status
uvx htmlgraph orchestrator status

# Disable orchestrator mode
uvx htmlgraph orchestrator disable
```

### How It Works

Orchestrator Mode uses HtmlGraph's **PreToolUse hook** to intercept tool calls before execution:

1. **Tool call initiated** - Agent attempts to use a tool (e.g., Bash, Edit, Grep)
2. **Hook intercepts** - PreToolUse hook examines the tool and context
3. **Classification** - Determines if operation should be allowed, warned, or blocked
4. **Guidance** - Provides feedback and suggests delegation
5. **Execution** - Either allows the operation or blocks it (depending on mode)

**Enforcement Modes:**

- **Strict** (default) - Blocks disallowed operations, agent must delegate
- **Guidance** - Shows warnings but allows all operations (learning mode)

### Operation Classification

#### ✅ Always Allowed (No restrictions)

- **CLI Operations** - `htmlgraph feature start`, `htmlgraph spike create`, etc.
- **Task Tool** - Delegation to subagents
- **TodoWrite** - Task list management
- **Read** - Reading files (≤5 per session)
- **Strategic Analysis** - `htmlgraph analytics recommend`, `htmlgraph analytics bottlenecks`

#### ⚠️ Warned (Allowed with guidance)

- **Bash** - First 3 calls allowed, then warned
- **Edit** - First 5 calls allowed, then warned
- **Grep** - First 5 calls allowed, then warned
- **Glob** - First 5 calls allowed, then warned

#### 🚫 Blocked in Strict Mode

- **Excessive Read** - More than 5 file reads
- **Excessive Bash** - More than 3 bash calls
- **Excessive Edit** - More than 5 file edits
- **Excessive Grep** - More than 5 searches
- **Excessive Glob** - More than 5 pattern matches

### Examples

#### ❌ Direct Execution (Fills Context)

```python
# Orchestrator runs tests directly - sequential, fills context
result1 = bash("uv run pytest tests/unit/")
result2 = bash("uv run pytest tests/integration/")
result3 = bash("uv run pytest tests/e2e/")
# Result: 3 sequential calls, full output in orchestrator context
# Orchestrator mode: BLOCKED after 3rd call
```

#### ✅ Delegated Execution (Preserves Context)

```python
# Orchestrator spawns parallel subagents
Task(
    subagent_type="general-purpose",
    prompt="Run unit tests and report only failures"
)
Task(
    subagent_type="general-purpose",
    prompt="Run integration tests and report only failures"
)
Task(
    subagent_type="general-purpose",
    prompt="Run e2e tests and report only failures"
)
# Result: 3 parallel agents, orchestrator gets summaries only
# Orchestrator mode: ALLOWED
```

#### ❌ Multiple File Edits (Fills Context)

```python
# Orchestrator edits 10 files
for file in files:
    Edit(file, ...)  # Each edit adds to context
# Orchestrator mode: BLOCKED after 5 edits
```

#### ✅ Delegated File Edits

```python
# Orchestrator delegates to subagent
Task(
    subagent_type="general-purpose",
    prompt=f"Update all files in {files} to use new API. Report summary of changes."
)
# Orchestrator mode: ALLOWED
```

### Configuration

Orchestrator mode is configured via `.htmlgraph/orchestrator.json`:

```json
{
  "enabled": true,
  "mode": "strict",
  "thresholds": {
    "max_bash_calls": 3,
    "max_file_reads": 5,
    "max_file_edits": 5,
    "max_grep_calls": 5,
    "max_glob_calls": 5
  },
  "allowed_tools": [
    "CLI",
    "Task",
    "TodoWrite"
  ]
}
```

**Customization:**

```bash
# Edit thresholds directly
vim .htmlgraph/orchestrator.json

# Or use CLI (future)
uvx htmlgraph orchestrator set-threshold max_bash_calls 5
```

### When to Use Orchestrator Mode

**Use Orchestrator Mode When:**
- ✅ Managing complex multi-step workflows
- ✅ Coordinating multiple features or phases
- ✅ Running comprehensive test suites
- ✅ Large-scale refactoring across many files
- ✅ Exploratory analysis of large codebases

**Skip Orchestrator Mode When:**
- ❌ Working on a single, focused task
- ❌ Quick bug fixes (1-2 files)
- ❌ Prototyping or experimentation
- ❌ Writing documentation

### Troubleshooting

**Problem: Operation blocked but I need to do it**

Solution: Use `--mode guidance` for warnings only:
```bash
uvx htmlgraph orchestrator enable --mode guidance
```

**Problem: Too many operations blocked**

Solution: Increase thresholds or disable temporarily:
```bash
# Increase thresholds
vim .htmlgraph/orchestrator.json  # Edit max_* values

# Or disable temporarily
uvx htmlgraph orchestrator disable
```

**Problem: Don't understand why operation was blocked**

Solution: Check the guidance message - it explains why and suggests delegation:
```
⚠️ ORCHESTRATOR MODE: Exceeded threshold for Bash calls (3/3)
Suggestion: Delegate to subagent using Task tool
Example: Task(subagent_type="general-purpose", prompt="Run pytest and report failures")
```

### Best Practices

1. **Start with Guidance Mode** - Learn the patterns before enforcing
   ```bash
   uvx htmlgraph orchestrator enable --mode guidance
   ```

2. **Delegate Early** - Don't wait until you hit thresholds
   ```python
   # As soon as you see multiple similar operations
   Task(prompt="Handle all test files in tests/ directory")
   ```

3. **Use Task Tool Liberally** - It's designed for this
   ```python
   # Good delegation patterns
   Task(prompt="Explore codebase and find all API endpoints")
   Task(prompt="Run full test suite and report failures")
   Task(prompt="Update all imports to use new module structure")
   ```

4. **Monitor Context Usage** - Check your context regularly
   ```python
   # If you're filling context, delegate
   if len(messages) > 50:
       Task(prompt="Complete this implementation")
   ```

5. **Review Guidance Messages** - Learn from warnings
   ```
   # Each warning teaches a pattern
   ⚠️ Orchestrator mode suggests delegation
   # → Adjust your workflow
   ```

### FAQ

**Q: Will this slow me down?**
A: No - delegation is faster (parallel) and preserves context for high-level decisions.

**Q: Can I bypass orchestrator mode?**
A: Yes - use `--mode guidance` or disable it. But you'll lose the benefits.

**Q: What if I disagree with a block?**
A: Open an issue - we want to improve the classification logic.

**Q: Does this work with all AI agents?**
A: Yes - any agent using HtmlGraph will respect orchestrator mode.

**Q: How do I know it's working?**
A: Check status: `uvx htmlgraph orchestrator status`

---

## Orchestrator Success Patterns

### Pattern 1: Parallel Test Execution
**❌ Direct (Sequential)**:
```python
# Orchestrator runs tests directly - fills context
uv run pytest tests/unit/
uv run pytest tests/integration/
uv run pytest tests/e2e/
# Result: 3 sequential calls, full output in orchestrator context
```

**✅ Delegated (Parallel)**:
```python
# Orchestrator spawns parallel subagents
Task(subagent_type="general-purpose", prompt="Run unit tests and report failures")
Task(subagent_type="general-purpose", prompt="Run integration tests and report failures")
Task(subagent_type="general-purpose", prompt="Run e2e tests and report failures")
# Result: 3 parallel agents, orchestrator gets summaries only
```

### Pattern 2: Multi-File Implementation
**❌ Direct**: Orchestrator edits 5 files, context fills with diffs
**✅ Delegated**: Subagent handles all edits, returns summary

### Pattern 3: Codebase Exploration
**❌ Direct**: 10 Grep/Glob calls pollute orchestrator context
**✅ Delegated**: `Task(subagent_type="Explore")` returns structured findings

### Why Delegation Wins
| Metric | Direct | Delegated |
|--------|--------|-----------|
| Context used | HIGH | LOW |
| Parallelization | None | Full |
| Work tracking | Manual | Automatic |
| Learning/Patterns | Lost | Captured |

---

## Architecture: Operations Layer

HtmlGraph uses a **unified operations layer** that both CLI and SDK call. This eliminates code duplication and ensures consistent behavior.

```
CLI ────┐
        ├──→ Operations Layer (shared backend)
SDK ────┘
```

**Benefits:**
- ✅ No code duplication between CLI and SDK
- ✅ Consistent results regardless of interface
- ✅ Single source of truth for business logic
- ✅ Easier testing and maintenance

**Operations modules:**
- `operations/server.py` - Server lifecycle (start, stop, status)
- `operations/hooks.py` - Git hooks management
- `operations/events.py` - Event log indexing
- `operations/analytics.py` - Analytics operations

**See [docs/OPERATIONS_LAYER.md](./docs/OPERATIONS_LAYER.md) for complete documentation.**

**Example - SDK uses operations:**
```python
# In SDK
def start_server(self, port: int = 8080) -> ServerHandle:
    from htmlgraph.operations import server
    result = server.start_server(port=port, ...)
    return result.handle
```

**Example - CLI uses operations:**
```python
# In CLI
def cmd_serve(args):
    from htmlgraph.operations import server
    result = server.start_server(port=args.port, ...)
    print(f"Server started at {result.handle.url}")
```

---

## CLI Reference

### Feature Commands

```bash
htmlgraph feature create "Title"          # Create feature, prints ID
htmlgraph feature show <id>               # Show feature details
htmlgraph feature list                    # List all features
htmlgraph feature start <id>              # Mark in-progress
htmlgraph feature complete <id>           # Mark done
htmlgraph find features --status <status> # Query by status
htmlgraph find features --status <status> --priority <priority>
```

### Bug Commands

```bash
htmlgraph bug create "Title"   # Create bug, prints ID
htmlgraph bug list             # List all bugs
htmlgraph bug show <id>        # Show bug details
```

### Spike Commands

```bash
htmlgraph spike create "Title"  # Create spike, prints ID
htmlgraph spike list            # List all spikes
```

### Track Commands

```bash
htmlgraph track new "Title"     # Create track
htmlgraph track list            # List all tracks
```

### Session Commands

```bash
htmlgraph session list          # List sessions
htmlgraph session start         # Start a session
```

### Analytics Commands

```bash
htmlgraph analytics recommend    # Recommended next work
htmlgraph analytics bottlenecks  # Find blockers
```

### Snapshot Commands

```bash
htmlgraph snapshot               # Full snapshot
htmlgraph snapshot --summary     # Summary view
```

---

## Examples

See `examples/` directory for complete demonstrations. Run the CLI to explore:

```bash
htmlgraph --help
htmlgraph feature --help
```

---

## Agent Handoff Context

Handoff enables smooth context transfer between agents when a task requires different expertise.

### Handoff Best Practices

1. **Provide context**: Create a spike documenting what's done and what's next
2. **Mark progress**: Use `htmlgraph feature start` / `htmlgraph feature complete` to keep status current
3. **Document blockers**: `htmlgraph spike create "Handoff notes: <reason>"` to capture context
4. **Leave breadcrumbs**: Record your approach so the next agent can continue

```bash
# Before handing off
htmlgraph spike create "Handoff: feature-001 blocked on testing — implementation complete, needs test coverage"
htmlgraph feature show feature-001  # Verify status is accurate
```

---

## Agent Routing & Capabilities

Capability-based routing automatically assigns tasks to agents with matching skills.

Use the analytics CLI to find recommended next work and understand workload:

```bash
# Find recommended next work
htmlgraph analytics recommend

# Find bottlenecks and blockers
htmlgraph analytics bottlenecks

# Check in-progress work (workload view)
htmlgraph find features --status in-progress
```

---

## Claude Code Transcript Integration

HtmlGraph integrates with Claude Code transcripts to capture development context and enable analytics.

### What Are Transcripts?

Claude Code stores conversation transcripts as JSONL files in:
```
~/.claude/projects/[encoded-path]/[session-uuid].jsonl
```

These contain:
- User messages and assistant responses
- Tool calls (Read, Write, Edit, Bash, etc.)
- Thinking traces (optional)
- Timestamps and session metadata
- Git branch context

### Why Transcripts Matter

Transcripts capture the **reasoning** behind code changes:
- **What was asked for** - Original user prompts
- **What Claude suggested** - AI recommendations and alternatives
- **Decisions made** - Why certain approaches were chosen
- **Implementation context** - Claude's reasoning during development

### CLI Commands

```bash
# List available transcripts
uvx htmlgraph transcript list [--limit N]

# Import a transcript session
uvx htmlgraph transcript import SESSION_ID [--link-feature FEAT_ID]

# Auto-link transcripts by git branch
uvx htmlgraph transcript auto-link [--branch BRANCH]

# Export transcript to HTML
uvx htmlgraph transcript export SESSION_ID -o output.html

# Get session health metrics
uvx htmlgraph transcript health SESSION_ID

# Detect workflow patterns
uvx htmlgraph transcript patterns [--transcript-id ID]

# Show tool transition matrix
uvx htmlgraph transcript transitions

# Get improvement recommendations
uvx htmlgraph transcript recommendations

# Comprehensive analytics
uvx htmlgraph transcript insights

# Track-level aggregation
uvx htmlgraph transcript track-stats TRACK_ID
```

### Analytics Features

**Session Health Scoring:**
- Efficiency score (tool calls per user message)
- Retry rate (consecutive same-tool usage)
- Context rebuilds (repeated file reads)
- Tool diversity (variety of tools used)

**Pattern Detection:**
- Anti-patterns: 4x Bash, 3x Edit, 3x Grep, 4x Read (repeated)
- Optimal patterns: Grep→Read, Read→Edit, Edit→Bash

**Track-Level Aggregation:**
- Aggregate stats across all sessions in a track
- Health trends (improving/stable/declining)
- Combined tool frequency and transitions

### PreToolUse Hook Integration

HtmlGraph's PreToolUse hook provides real-time guidance based on transcript patterns:

```python
# Active learning from tool history
ANTI_PATTERNS = {
    ("Bash", "Bash", "Bash", "Bash"): "4 consecutive Bash commands. Check for errors.",
    ("Edit", "Edit", "Edit"): "3 consecutive Edits. Consider batching.",
}

OPTIMAL_PATTERNS = {
    ("Grep", "Read"): "Good: Search then read - efficient exploration.",
    ("Read", "Edit"): "Good: Read then edit - informed changes.",
}
```

The hook tracks tool usage and provides guidance (never blocks) to improve workflows.

### HTML Export

Export transcripts to browser-viewable HTML:

```bash
uvx htmlgraph transcript export SESSION_ID -o transcript.html --include-thinking
```

Compatible with [claude-code-transcripts](https://github.com/simonw/claude-code-transcripts) format.

---

## Troubleshooting

### CLI not finding .htmlgraph directory

Run from the project root directory, or initialize first:

```bash
htmlgraph init --install-hooks
```

### Feature not found

List all features to verify the ID:

```bash
htmlgraph feature list
htmlgraph feature show <id>
```

---

## Documentation

- **CLI Reference**: Run `htmlgraph --help` or `htmlgraph <command> --help` for all options
- **Quickstart**: `docs/quickstart.md`
- **Dashboard**: Run `uvx htmlgraph serve` and open http://localhost:8080

---

## Deployment & Release

### Using the Deployment Script (FLEXIBLE OPTIONS)

HtmlGraph includes `scripts/deploy-all.sh` with multiple modes for different scenarios:

**Quick Usage:**
```bash
# Documentation changes only (commit + push)
./scripts/deploy-all.sh --docs-only

# Full release
./scripts/deploy-all.sh 0.7.1

# Preview what would happen
./scripts/deploy-all.sh --dry-run

# Show all options
./scripts/deploy-all.sh --help
```

**Available Flags:**
- `--docs-only` - Only commit and push to git (skip build/publish)
- `--build-only` - Only build package (skip git/publish/install)
- `--skip-pypi` - Skip PyPI publishing step
- `--skip-plugins` - Skip plugin update steps
- `--dry-run` - Show what would happen without executing

**What full deployment does (7 steps):**
1. **Git Push** - Pushes commits and tags to origin/main
2. **Build Package** - Creates wheel and source distributions with `uv build`
3. **Publish to PyPI** - Uploads to PyPI using token from .env
4. **Local Install** - Installs latest version locally with pip
5. **Update Claude Plugin** - Runs `claude plugin update htmlgraph`
6. **Update Gemini Extension** - Updates version in gemini-extension.json
7. **Update Codex Skill** - Checks for Codex and updates if present

**Prerequisites:**

Set your PyPI token in `.env` file:
```bash
PyPI_API_TOKEN=pypi-YOUR_TOKEN_HERE
```

**Complete Release Workflow:**

```bash
# 1. Update version numbers
# Edit: pyproject.toml, __init__.py, plugin.json, gemini-extension.json

# 2. Commit version bump
git add pyproject.toml src/python/htmlgraph/__init__.py \
  packages/claude-plugin/.claude-plugin/plugin.json \
  packages/gemini-extension/gemini-extension.json
git commit -m "chore: bump version to 0.7.1"

# 3. Create git tag
git tag v0.7.1
git push origin main --tags

# 4. Run deployment script
./scripts/deploy-all.sh 0.7.1
```

**Manual Steps (if script fails):**

```bash
# Build
uv build

# Publish to PyPI
source .env
uv publish dist/htmlgraph-0.7.1* --token "$PyPI_API_TOKEN"

# Install locally
pip install --upgrade htmlgraph==0.7.1

# Update plugins manually
claude plugin update htmlgraph
```

**Verify Deployment:**

```bash
# Check PyPI
open https://pypi.org/project/htmlgraph/

# Verify local install
htmlgraph version

# Test Claude plugin
claude plugin list | grep htmlgraph
```

---

### Generalized Deployment System (NEW!)

**For YOUR Projects** - HtmlGraph now includes a flexible deployment system that any project can use!

#### Quick Start

```bash
# 1. Initialize deployment configuration
htmlgraph deploy init

# 2. Edit htmlgraph-deploy.toml to customize
# 3. Run deployment
htmlgraph deploy run

# Or with flags
htmlgraph deploy run --dry-run        # Preview
htmlgraph deploy run --build-only     # Just build
htmlgraph deploy run --docs-only      # Just git push
```

#### Configuration

The `htmlgraph deploy init` command creates a template configuration file:

```toml
[project]
name = "my-project"
pypi_package = "my-package"

[deployment]
# Customize which steps to run and in what order
steps = [
    "git-push",
    "build",
    "pypi-publish",
    "local-install",
    "update-plugins"
]

[deployment.git]
branch = "main"
remote = "origin"
push_tags = true

[deployment.build]
command = "uv build"  # Or "python -m build", "poetry build", etc.
clean_dist = true

[deployment.pypi]
token_env_var = "PyPI_API_TOKEN"
wait_after_publish = 10

[deployment.plugins]
# Update platform-specific plugins
claude = "claude plugin update {package}"
gemini = "gemini extensions update {package}"

[deployment.hooks]
# Custom commands to run at various stages
pre_build = ["python scripts/update_version.py {version}"]
post_build = []
pre_publish = []
post_publish = ["python scripts/notify_release.py {version}"]
```

#### Available Steps

1. **git-push** - Push commits and tags to remote
2. **build** - Build package distributions
3. **pypi-publish** - Upload to PyPI
4. **local-install** - Install package locally
5. **update-plugins** - Update platform-specific plugins

#### Custom Hooks

Add custom commands at key points in the deployment process:

- **pre_build** - Before building (e.g., update version files)
- **post_build** - After building (e.g., validate artifacts)
- **pre_publish** - Before PyPI publish (e.g., run tests)
- **post_publish** - After publishing (e.g., notify Slack, create GitHub release)

Hooks support placeholders:
- `{version}` - Current package version
- `{package}` - Package name

#### Deployment Modes

```bash
# Full deployment (all steps)
htmlgraph deploy run

# Documentation only (git push)
htmlgraph deploy run --docs-only

# Build only (no git, no publish)
htmlgraph deploy run --build-only

# Skip specific steps
htmlgraph deploy run --skip-pypi
htmlgraph deploy run --skip-plugins

# Preview mode (no changes)
htmlgraph deploy run --dry-run
```

#### Example: Flask Project Deployment

```toml
[project]
name = "my-flask-app"
pypi_package = "my-flask-app"

[deployment]
steps = [
    "git-push",
    "build",
    "pypi-publish",
    "local-install"
]

[deployment.build]
command = "python -m build"
clean_dist = true

[deployment.hooks]
pre_build = [
    "python -m pytest",  # Run tests first
    "python scripts/bump_version.py {version}"
]
post_publish = [
    "python scripts/deploy_docs.py",
    "curl -X POST https://hooks.slack.com/... -d 'Released {version}'"
]
```

#### Example: Multi-Platform Plugin

```toml
[deployment.plugins]
# Update multiple platforms
claude = "claude plugin update {package}"
gemini = "gemini extensions update {package}"
codex = "codex skills update {package}"
vscode = "vsce publish"
```

#### Benefits Over Shell Scripts

- ✅ **Portable** - Works across platforms (Windows, Mac, Linux)
- ✅ **Configurable** - TOML config instead of editing bash
- ✅ **Extensible** - Custom hooks for any workflow
- ✅ **Safe** - Dry-run mode and step-by-step execution
- ✅ **Integrated** - Works with htmlgraph tracking
- ✅ **Reusable** - Share config across projects

---

## Documentation Synchronization

### Memory File Sync Tool

HtmlGraph includes `scripts/sync_memory_files.py` to maintain consistency across AI agent documentation files:

**Usage:**
```bash
# Check if files are synchronized
python scripts/sync_memory_files.py --check

# Generate platform-specific file
python scripts/sync_memory_files.py --generate gemini
python scripts/sync_memory_files.py --generate claude
python scripts/sync_memory_files.py --generate codex

# Overwrite existing file
python scripts/sync_memory_files.py --generate gemini --force
```

**What it checks:**
- ✅ AGENTS.md exists (required central documentation)
- ✅ Platform files reference AGENTS.md properly
- ✅ Consistency across Claude, Gemini, Codex docs

**File structure:**
```
project/
├── AGENTS.md                    # Central documentation (SDK, deployment, workflows)
├── CLAUDE.md                    # Project vision + references AGENTS.md
├── GEMINI.md                    # Gemini-specific + references AGENTS.md
└── packages/
    ├── claude-plugin/skills/htmlgraph-tracker/SKILL.md
    ├── gemini-extension/GEMINI.md
    └── codex-skill/SKILL.md
```

**Why this matters:**
- Single source of truth (AGENTS.md)
- Platform files add platform-specific notes
- Easy maintenance (update once, not 3+ times)
- Automated validation

---

## Git-Based Continuity Spine

### Overview

HtmlGraph uses Git as a universal continuity spine that enables agent-agnostic session tracking. This means HtmlGraph works with ANY coding agent (Claude, Codex, Cursor, vim), not just those with native integrations.

**Core Principle**: Git commits are universal continuity points that work regardless of which agent wrote the code.

### Quick Start

**Install Git hooks**:
```bash
htmlgraph install-hooks
```

**What this does**:
- Installs hooks in `.git/hooks/` (symlinked to `.htmlgraph/hooks/`)
- Tracks commits, checkouts, merges, pushes automatically
- Links sessions across agents via commit graph
- Works offline (Git is local)

### How It Works

**Git hooks log events** to `.htmlgraph/events/`:

```
Session S1 (Claude)          Session S2 (Codex)         Session S3 (Claude)
─────────────────────       ─────────────────────      ─────────────────────
start_commit: abc1          start_commit: abc3         start_commit: abc5
continued_from: None        continued_from: S1         continued_from: S2

Events:                     Events:                    Events:
  - Edit file               - Edit file                - Edit file
  - GitCommit abc1          - GitCommit abc3           - GitCommit abc5
  - GitCommit abc2          - GitCommit abc4           - GitCommit abc6

Git Commit Graph:
abc1 → abc2 → abc3 → abc4 → abc5 → abc6
 │             │             │
S1            S2            S3
```

**Session continuity survives crashes** - Git history is durable.

### Commit Message Convention

Include feature references for better attribution:

```bash
# Good - explicit feature reference
git commit -m "feat: add login endpoint (feature-auth-001)"

# Better - structured format
git commit -m "feat: add login endpoint

Implements: feature-auth-001
Related: feature-session-002
"
```

### Cross-Agent Collaboration

**Example: Work starts in Claude, continues in Codex**:

```bash
# Day 1 (Claude)
htmlgraph feature start feature-auth-001
# ... work ...
git commit -m "feat: start auth (feature-auth-001)"  # → abc123

# Day 2 (Codex - different agent!)
# ... continue work ...
git commit -m "feat: continue auth (feature-auth-001)"  # → def456

# Query for full session history
htmlgraph session list
```

### Event Types

**GitCommit** - Primary continuity anchor:
```json
{
  "type": "GitCommit",
  "commit_hash": "abc123",
  "branch": "main",
  "author": "alice@example.com",
  "message": "feat: add user authentication",
  "files_changed": ["src/auth/login.py"],
  "insertions": 145,
  "deletions": 23,
  "features": ["feature-auth-001"]
}
```

**GitCheckout** - Branch continuity:
```json
{
  "type": "GitCheckout",
  "from_branch": "main",
  "to_branch": "feature/auth"
}
```

**GitMerge** - Integration events:
```json
{
  "type": "GitMerge",
  "orig_head": "abc123",
  "new_head": "def456"
}
```

**GitPush** - Team boundaries:
```json
{
  "type": "GitPush",
  "remote_name": "origin",
  "updates": [...]
}
```

### Agent Compatibility

| Agent | Git Hooks | Session Tracking | Notes |
|-------|-----------|------------------|-------|
| Claude Code | ✅ | ✅ | Full integration via plugin |
| GitHub Codex | ✅ | ✅ | Git hooks + SDK |
| Google Gemini | ✅ | ✅ | Git hooks + SDK |
| Cursor | ✅ | ✅ | Git hooks + SDK |
| vim/emacs | ✅ | ⚠️ | Manual session start |
| Any CLI tool | ✅ | ❌ | Commits tracked only |

### Benefits

- ✅ **Agent agnostic** - Works with ANY agent
- ✅ **Survives crashes** - Git history is durable
- ✅ **Team collaboration** - Multi-agent tracking
- ✅ **Offline-first** - Git is local
- ✅ **Simple** - Just Git hooks, no complex setup

### Advanced: Session Reconstruction

HtmlGraph can reconstruct session continuity using multiple signals:

**1. Session listing**:
```bash
htmlgraph session list
```

**2. Feature-based linking**:
```bash
# View sessions and filter by feature context
htmlgraph session list
```

### Documentation

For complete details, see:
- [Git Continuity Architecture](./docs/GIT_CONTINUITY_ARCHITECTURE.md) - Technical deep-dive
- [Git Hooks Guide](./docs/GIT_HOOKS.md) - Hook installation and config

---

## Related Files

- `cmd/` - Go CLI binary source
- `internal/` - Go business logic packages
- `examples/` - Complete examples
- `scripts/deploy-all.sh` - Deployment automation script
