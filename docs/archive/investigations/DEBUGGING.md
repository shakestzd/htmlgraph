# Wipnote Debugging Guide

**Philosophy: Research First, Then Debug**

This guide provides practical debugging workflows for Wipnote users. It enforces a research-first approach: always investigate documentation before implementing solutions.

## Table of Contents

1. [Quick Start](#quick-start)
2. [Tool Selection Matrix](#tool-selection-matrix)
3. [Common Debugging Scenarios](#common-debugging-scenarios)
4. [Debug Command Reference](#debug-command-reference)
5. [Research Resources](#research-resources)
6. [Debugging Agents](#debugging-agents)

---

## Quick Start

**When encountering an issue, follow this sequence:**

```bash
# 1. RESEARCH FIRST (most important step)
#    Use claude-code-guide agent for Claude-specific questions
#    Read official documentation
#    Search GitHub issues

# 2. Gather Evidence
claude --debug          # Enable debug mode
/hooks                  # Inspect hooks
/doctor                 # System diagnostics

# 3. Run Diagnostics
uv run wipnote status                     # Check Wipnote state
uv run wipnote orchestrator status        # Check orchestrator
uv run ruff check && uv run mypy src/       # Check code quality

# 4. Test Your Fix
uv run pytest                               # Run tests
```

**CRITICAL: Never implement solutions based on assumptions. Always research documentation first.**

---

## Tool Selection Matrix

Choose the right tool for your debugging task:

| Scenario | Use This Tool | Why |
|----------|---------------|-----|
| Unfamiliar error | **Researcher Agent** | Research docs before trying fixes |
| Claude Code hooks issue | **claude-code-guide** + Researcher | Official guidance for hooks/plugins |
| Error with known cause | **Debugger Agent** | Systematic root cause analysis |
| Before committing code | **Test Runner Agent** | Validate quality gates |
| Multiple failed attempts | **Researcher Agent** | Stop guessing, start researching |
| Hook execution issues | `/hooks` command | Inspect active hooks |
| System health check | `/doctor` command | Overall diagnostics |
| Feature tracking issues | `wipnote status` | Check Wipnote state |
| Type/lint errors | `mypy` + `ruff` | Static analysis |

---

## Common Debugging Scenarios

### Scenario 1: Duplicate Hook Execution

**Symptoms**: Hook runs multiple times, messages show "(1/2 done)" or "(2/2 done)"

**Debugging Steps**:

```bash
# 1. RESEARCH: Understanding hook merging behavior
# Use claude-code-guide to research hook loading
# Read: https://code.claude.com/docs/en/hooks.md

# 2. Inspect active hooks
/hooks

# 3. Check all hook sources
cat .claude/settings.json           # Global hooks
cat .claude/hooks/hooks.json        # Local hooks
# Check plugin hooks in installed plugins

# 4. Identify the issue
# Claude Code MERGES hooks from multiple sources
# Duplicates occur when same hook is in multiple places

# 5. Fix: Remove duplicates
# Keep hooks in ONE location (prefer plugin hooks)
# Remove from other locations
```

**Root Cause**: Claude Code merges hooks from all sources (settings.json, hooks.json, plugin hooks). Same hook in multiple places = duplicate execution.

**Solution**: Define each hook in only ONE location.

**See**: `packages/claude-plugin/agents/researcher.md` for research methodology

---

### Scenario 2: Hook Not Executing

**Symptoms**: Expected hook behavior doesn't happen

**Debugging Steps**:

```bash
# 1. RESEARCH: Hook configuration schema
# Read official hook documentation first

# 2. Verify hook is registered
/hooks
# Should show your hook in the list

# 3. Check hook syntax
cat .claude/settings.json
# Validate JSON structure
# Verify hook type (PreToolUse, PostToolUse, etc.)

# 4. Test hook command manually
uv run wipnote session start coder test-hook
# Or whatever command your hook runs

# 5. Check logs for errors
tail -f ~/.claude/logs/claude-code.log

# 6. Verify environment variables
echo ${CLAUDE_PLUGIN_ROOT}
# Ensure paths resolve correctly
```

**Common Issues**:
- Invalid JSON syntax
- Wrong hook type (PreToolUse vs PostToolUse)
- Command path not found
- Environment variables not set

**See**: `packages/claude-plugin/agents/debugger.md` for systematic debugging

---

### Scenario 3: Feature/Session Tracking Not Working

**Symptoms**: Sessions not created, features not tracked, no HTML files generated

**Debugging Steps**:

```bash
# 1. Check Wipnote initialization
ls .wipnote/
# Should see: features/ sessions/ spikes/ tracks/

# 2. Verify orchestrator status
uv run wipnote orchestrator status
# Should show enabled state

# 3. Check session state
uv run wipnote session list --active
# Shows currently active session

# 4. Inspect feature files
ls .wipnote/features/
# Should see HTML files

# 5. Verify hooks are running
/hooks
# Look for PreToolUse and PostToolUse hooks

# 6. Check hook execution logs
cat .wipnote/sessions/*.jsonl
# See session event logs
```

**Common Issues**:
- `.wipnote/` not initialized
- Hooks not registered
- Orchestrator disabled
- SDK import errors

**Solution**: Run `uv run wipnote init` to reinitialize

---

### Scenario 4: Orchestrator Not Enforcing Delegation

**Symptoms**: No delegation warnings, direct tool execution allowed in strict mode

**Debugging Steps**:

```bash
# 1. Check orchestrator status
uv run wipnote orchestrator status
# Expected: "enabled (strict enforcement)"

# 2. Verify configuration file
cat .wipnote/orchestrator-mode.json
# Should show: {"enabled": true, "mode": "strict"}

# 3. Check hook registration
/hooks
# PostToolUse hook should be registered

# 4. Test with direct tool execution
# Try running a tool directly
# Should receive reflection warning

# 5. Check for conflicting hooks
/hooks PostToolUse
# Should show only one PostToolUse hook

# 6. Restart Claude Code if needed
# Exit and restart to reload configuration
```

**See**: CLAUDE.md section on "Orchestrator Mode" for details

---

### Scenario 5: Plugin Installation Issues

**Symptoms**: Plugin not loading, commands not available

**Debugging Steps**:

```bash
# 1. RESEARCH: Plugin installation docs
# https://code.claude.com/docs/en/plugins.md

# 2. List installed plugins
claude plugin list

# 3. Check plugin status
claude plugin show wipnote

# 4. Verify plugin structure
ls ~/.claude/plugins/wipnote/
# Should see .claude-plugin/ directory

# 5. Check plugin.json validity
cat ~/.claude/plugins/wipnote/.claude-plugin/plugin.json
# Validate JSON structure

# 6. Reinstall if needed
claude plugin uninstall wipnote
claude plugin install wipnote
```

---

### Scenario 6: Type Errors or Lint Warnings

**Symptoms**: mypy or ruff reports errors

**Debugging Steps**:

```bash
# 1. Run type checking
uv run mypy src/
# Shows all type errors with line numbers

# 2. Run linting
uv run ruff check
# Shows all lint warnings

# 3. Auto-fix what's possible
uv run ruff check --fix
uv run ruff format

# 4. Fix remaining errors manually
# Address each error shown in output

# 5. Verify all errors resolved
uv run mypy src/
uv run ruff check
# Both should pass with no errors

# 6. Run tests
uv run pytest
# Ensure no regressions
```

**CRITICAL**: Fix ALL errors before committing, even pre-existing ones (see CLAUDE.md "Code Hygiene")

---

### Scenario 7: Test Failures

**Symptoms**: pytest reports failing tests

**Debugging Steps**:

```bash
# 1. Run tests with verbose output
uv run pytest -v

# 2. Run only failing test
uv run pytest tests/test_specific.py::test_name -v

# 3. Add debug output to test
# Use print() or logging in test
# Re-run to see output

# 4. Run test with debugger
uv run pytest --pdb tests/test_specific.py::test_name
# Drops into pdb on failure

# 5. Check test assumptions
# Verify test setup is correct
# Check mocks and fixtures

# 6. Fix and verify
# Make fix, re-run all tests
uv run pytest
```

**See**: `packages/claude-plugin/agents/test-runner.md` for testing strategy

---

## Debug Command Reference

### Claude Code Commands

```bash
# Debug Mode
claude --debug <command>              # Verbose output
claude --verbose                       # More detailed logging

# Hook Inspection
/hooks                                 # List all active hooks
/hooks PreToolUse                      # Show specific hook type
/hooks PostToolUse                     # Show PostToolUse hooks

# System Diagnostics
/doctor                                # Check system health
claude --version                       # Show Claude Code version

# LSP Debugging
claude --enable-lsp-logging           # Enable LSP logs

# Plugin Management
claude plugin list                     # List installed plugins
claude plugin show <name>             # Show plugin details
claude plugin update <name>           # Update plugin
```

### Wipnote Commands

```bash
# Status & Inspection
uv run wipnote status                # Show all features
uv run wipnote feature show <id>    # Show specific feature
uv run wipnote session list         # List sessions
uv run wipnote session list --active # Active sessions only

# Orchestrator
uv run wipnote orchestrator status   # Check orchestrator state
uv run wipnote orchestrator enable   # Enable orchestrator
uv run wipnote orchestrator disable  # Disable orchestrator

# Analytics
uv run wipnote recommend             # Get work recommendations
uv run wipnote bottlenecks           # Find bottlenecks

# Maintenance
uv run wipnote init                  # Initialize .wipnote/
uv run wipnote sync-docs             # Sync documentation
```

### Testing & Quality

```bash
# Run Tests
uv run pytest                          # All tests
uv run pytest -v                       # Verbose output
uv run pytest -x                       # Stop on first failure
uv run pytest --lf                     # Run last failed
uv run pytest tests/test_file.py       # Specific file

# Type Checking
uv run mypy src/                       # Check all types
uv run mypy --strict src/              # Strict mode
uv run mypy src/wipnote/hooks.py    # Specific file

# Linting
uv run ruff check                      # Check all files
uv run ruff check --fix                # Auto-fix issues
uv run ruff format                     # Format code

# Full Quality Gate (pre-commit)
uv run ruff check --fix && \
uv run ruff format && \
uv run mypy src/ && \
uv run pytest
```

### System Investigation

```bash
# File Inspection
ls -lt .claude/                        # Check Claude config
ls -lt .wipnote/                     # Check Wipnote state
cat .claude/settings.json              # View settings
cat .wipnote/orchestrator-mode.json  # Orchestrator config

# Search & Grep
grep -r "pattern" .claude/             # Search Claude configs
grep -r "pattern" .wipnote/          # Search Wipnote files

# Git State
git status                             # Check working tree
git diff                               # View changes
git log --oneline -10                  # Recent commits

# Environment
which python                           # Python location
which uv                               # UV location
uv --version                           # UV version
echo $CLAUDE_PLUGIN_ROOT               # Plugin root path
```

---

## Research Resources

### Official Documentation

**Claude Code**:
- Main docs: https://code.claude.com/docs
- Hooks guide: https://code.claude.com/docs/en/hooks.md
- Plugin development: https://code.claude.com/docs/en/plugins.md
- GitHub: https://github.com/anthropics/claude-code

**Wipnote**:
- AGENTS.md - SDK, API, CLI reference
- CLAUDE.md - Project overview, workflows
- README.md - Quick start guide
- Agent docs: `packages/claude-plugin/agents/`

### Issue Tracking

**Search before asking**:
1. Claude Code GitHub issues: https://github.com/anthropics/claude-code/issues
2. Check closed issues for solutions
3. Look for related discussions
4. Search Wipnote spikes: `.wipnote/spikes/`

### Research Checklist

Before implementing ANY fix:
- [ ] Has this error been encountered before? (Search GitHub issues)
- [ ] What does the official documentation say?
- [ ] Are there example implementations to reference?
- [ ] What debug tools can provide more information?
- [ ] Have I used the claude-code-guide agent?

---

## Debugging Agents

Wipnote plugin provides three specialized debugging agents. Use them systematically:

### 1. Researcher Agent

**Purpose**: Research documentation BEFORE implementing solutions

**Location**: `packages/claude-plugin/agents/researcher.md`

**Use when**:
- Encountering unfamiliar errors
- Working with Claude Code hooks/plugins
- Multiple attempted fixes have failed
- Before implementing assumptions

**Workflow**:
1. Search official documentation
2. Check GitHub issues
3. Use claude-code-guide agent
4. Document findings in Wipnote spike
5. Implement solution based on research

**Example**:
```python
from wipnote import SDK
sdk = SDK(agent="researcher")

spike = sdk.spikes.create(
    title="Research: Hook Duplication Issue",
    findings="""
    ## Problem
    Hooks executing multiple times

    ## Research Sources
    - Claude Code docs: Hooks merge from multiple sources
    - GitHub issue #123: Same behavior reported

    ## Root Cause
    Hooks defined in both settings.json and plugin

    ## Solution
    Remove from settings.json, keep in plugin only
    """
).save()
```

---

### 2. Debugger Agent

**Purpose**: Systematically analyze and resolve errors

**Location**: `packages/claude-plugin/agents/debugger.md`

**Use when**:
- Error messages appear but cause is unclear
- Behavior doesn't match expectations
- Tests are failing
- Need to trace execution flow

**Methodology**:
1. Gather evidence (logs, stack traces)
2. Reproduce consistently
3. Isolate variables (one change at a time)
4. Analyze error context
5. Form hypothesis
6. Test hypothesis
7. Implement minimal fix

**Example**:
```python
from wipnote import SDK
sdk = SDK(agent="debugger")

bug = sdk.bugs.create(
    title="Hook Not Executing",
    description="""
    ## Symptoms
    PreToolUse hook not running

    ## Reproduction
    1. Register hook in settings.json
    2. Run claude command
    3. Hook doesn't execute

    ## Debug Investigation
    - /hooks shows hook registered
    - Syntax valid (checked with jq)
    - Command path resolves correctly
    - Hypothesis: Environment variable issue

    ## Root Cause
    ${CLAUDE_PLUGIN_ROOT} not expanded in hook command

    ## Fix
    Use absolute path instead of env variable
    """
).save()
```

---

### 3. Test Runner Agent

**Purpose**: Validate all changes, enforce quality gates

**Location**: `packages/claude-plugin/agents/test-runner.md`

**Use when**:
- After implementing code changes
- Before marking features complete
- After fixing bugs
- Before committing code

**Workflow**:
1. Write tests first (TDD)
2. Run tests frequently
3. Pre-commit quality gates
4. Document test results

**Example**:
```bash
# Before marking feature complete
uv run ruff check --fix
uv run ruff format
uv run mypy src/
uv run pytest

# Only mark complete if all pass
```

---

## Debugging Workflow Pattern

**The Research-First Pattern** (from CLAUDE.md):

### Bad: Trial and Error (Don't Do This)
```
1. Try removing file → Still broken
2. Try clearing cache → Still broken
3. Try removing plugins → Still broken
4. Try removing symlinks → Still broken
5. Finally research documentation
6. Find root cause
```

### Good: Research First (Do This)
```
1. Research Claude Code hook loading behavior
2. Use claude-code-guide agent
3. Find: Hooks from multiple sources MERGE
4. Check all hook sources
5. Identify duplicates
6. Remove based on understanding
7. Verify fix works
8. Document in spike
```

**Key Principle**: Evidence > Assumptions | Research > Trial-and-Error

---

## Integration with Orchestrator Mode

When orchestrator mode is enabled (strict), you'll receive reflections after direct tool execution:

```
ORCHESTRATOR REFLECTION: You executed code directly.

Ask yourself:
- Could this have been delegated to a subagent?
- Would parallel Task() calls have been faster?
- Is a work item tracking this effort?
```

This encourages using specialized agents (researcher, debugger, test-runner) for systematic problem-solving.

**See**: CLAUDE.md section "Orchestrator Mode" for details

---

## Code Hygiene (MANDATORY)

From CLAUDE.md - **Always fix ALL errors before committing:**

### Pre-Commit Checklist
- [ ] All tests pass: `uv run pytest`
- [ ] No type errors: `uv run mypy src/`
- [ ] No lint warnings: `uv run ruff check`
- [ ] Code formatted: `uv run ruff format`
- [ ] Fix ALL errors, even pre-existing ones

### Philosophy
"Clean as you go - leave code better than you found it"

### Deployment Blockers
The `deploy-all.sh` script blocks on:
- Mypy type errors
- Ruff lint errors
- Test failures

**This is intentional** - maintain quality gates.

**See**: CLAUDE.md section "Code Hygiene" for full details

---

## Quick Decision Tree

```
Encountered an issue?
│
├─ Is it unfamiliar/unclear?
│  └─ Use Researcher Agent → Research docs → Document findings
│
├─ Is the cause known but complex?
│  └─ Use Debugger Agent → Systematic analysis → Document process
│
├─ Is it a test/quality issue?
│  └─ Use Test Runner Agent → Fix and validate → Document results
│
├─ Multiple fixes tried without research?
│  └─ STOP → Use Researcher Agent → Start over with research
│
└─ Just need quick diagnostics?
   └─ Use debug commands → /hooks, /doctor, wipnote status
```

---

## Summary

**Golden Rules**:
1. **Research FIRST** - Documentation over trial-and-error
2. **Use Agents** - Researcher, Debugger, Test-runner for systematic work
3. **Document Everything** - Capture findings in Wipnote spikes
4. **Fix ALL Errors** - Code hygiene is non-negotiable
5. **Test Before Commit** - Quality gates protect production

**Remember**: The fastest way to fix a bug is to understand it first. Research saves time.

---

## Related Documentation

- **CLAUDE.md** - Project overview, debugging workflow section
- **AGENTS.md** - SDK usage, API reference
- **packages/claude-plugin/agents/researcher.md** - Research methodology
- **packages/claude-plugin/agents/debugger.md** - Debugging methodology
- **packages/claude-plugin/agents/test-runner.md** - Testing strategy
- **.wipnote/spikes/** - Past debugging sessions and research

---

*"Evidence > Assumptions | Research > Trial-and-Error"* - Wipnote Debugging Philosophy
