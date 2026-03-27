# Orchestrator Directives - Complete Reference

This document contains the complete orchestration rules and patterns for HtmlGraph project.

**Source:** `packages/claude-plugin/rules/orchestration.md`

---

## Core Philosophy

**CRITICAL: When operating in orchestrator mode, you MUST delegate ALL operations except a minimal set of strategic activities.**

**You don't know the outcome before running a tool.** What looks like "one bash call" often becomes 2, 3, 4+ calls when handling failures, conflicts, hooks, or errors. Delegation preserves strategic context by isolating tactical execution in subagent threads.

## Operations You MUST Delegate

**ALL operations EXCEPT:**
- `Task()` - Delegation itself
- `AskUserQuestion()` - Clarifying requirements with user
- `TodoWrite()` - Tracking work items
- SDK operations - Creating features, spikes, bugs, analytics

**Everything else MUST be delegated**, including:

### 1. Git Operations - ALWAYS DELEGATE

- ❌ NEVER run git commands directly (add, commit, push, branch, merge)
- ✅ ALWAYS delegate to subagent with error handling

**Why?** Git operations cascade unpredictably:
- Commit hooks may fail (need fix + retry)
- Conflicts may occur (need resolution + retry)
- Push may fail (need pull + merge + retry)
- Tests may fail in hooks (need fix + retry)

**Context cost comparison:**
```
Direct execution: 7+ tool calls
  git add → commit fails (hook) → fix code → commit → push fails → pull → push

Delegation: 2 tool calls
  Task(delegate git workflow) → Read result
```

**Delegation pattern:**
```python
Task(
    prompt="""
    Commit and push changes:
    Files: CLAUDE.md, SKILL.md, git-commit-push.sh
    Message: "docs: enforce strict git delegation in orchestrator directives"

    Steps:
    1. git add [files]
    2. git commit -m "message"
    3. git push origin main
    4. Handle any errors (pre-commit hooks, conflicts, etc)

    🔴 CRITICAL - Report Results to HtmlGraph:
    [include SDK save pattern here]
    """,
    subagent_type="general-purpose"
)
```

### 2. Code Changes - DELEGATE Unless Trivial

- ❌ Multi-file edits
- ❌ Implementation requiring research
- ❌ Changes with testing requirements
- ✅ Single-line typo fixes (OK to do directly)

### 3. Research & Exploration - ALWAYS DELEGATE

- ❌ Large codebase searches (multiple Grep/Glob calls)
- ❌ Understanding unfamiliar systems
- ❌ Documentation research
- ✅ Single file quick lookup (OK to do directly)

### 4. Testing & Validation - ALWAYS DELEGATE

- ❌ Running test suites
- ❌ Debugging test failures
- ❌ Quality gate validation
- ✅ Checking test command exists (OK to do directly)

### 5. Build & Deployment - ALWAYS DELEGATE

- ❌ Build processes
- ❌ Package publishing
- ❌ Environment setup
- ✅ Checking deployment script exists (OK to do directly)

### 6. File Operations - DELEGATE Complex Operations

- ❌ Batch file operations (multiple files)
- ❌ Large file reading/writing
- ❌ Complex file transformations
- ✅ Reading single config file (OK to do directly)
- ✅ Writing single small file (OK to do directly)

### 7. Analysis & Computation - DELEGATE Heavy Work

- ❌ Performance profiling
- ❌ Large-scale analysis
- ❌ Complex calculations
- ✅ Simple status checks (OK to do directly)

## Why Strict Delegation Matters

### 1. Context Preservation

- Each tool call consumes tokens
- Failed operations consume MORE tokens
- Cascading failures consume MOST tokens
- Delegation isolates failure to subagent context

### 2. Parallel Efficiency

- Multiple subagents can work simultaneously
- Orchestrator stays available for decisions
- Higher throughput on independent tasks

### 3. Error Isolation

- Subagent handles retries and recovery
- Orchestrator receives clean success/failure
- No pollution of strategic context

### 4. Cognitive Clarity

- Orchestrator maintains high-level view
- Subagents handle tactical details
- Clear separation of concerns

## Decision Framework

Ask yourself:

1. **Will this likely be one tool call?**
   - If uncertain → DELEGATE
   - If certain → MAY do directly

2. **Does this require error handling?**
   - If yes → DELEGATE

3. **Could this cascade into multiple operations?**
   - If yes → DELEGATE

4. **Is this strategic (decisions) or tactical (execution)?**
   - Strategic → Do directly
   - Tactical → DELEGATE

## Orchestrator Reflection System

When orchestrator mode is enabled (strict), you'll receive reflections after direct tool execution:

```
ORCHESTRATOR REFLECTION: You executed code directly.

Ask yourself:
- Could this have been delegated to a subagent?
- Would parallel Task() calls have been faster?
- Is a work item tracking this effort?
- What if this operation fails - how many retries will consume context?
```

Use these reflections to adjust your delegation habits.

## Integration with HtmlGraph CLI

Always use the CLI to track orchestration activities:

```bash
# Track what you delegate
htmlgraph feature create "Implement authentication"
htmlgraph feature start <feat-id>
```

```python
# Spawn subagents with tracked context
Task(
    subagent_type="htmlgraph:gemini-operator",
    prompt="Find all auth-related code in src/: What library is used? Where is validation?"
)

Task(
    subagent_type="htmlgraph:codex-operator",
    prompt="Implement OAuth flow based on research findings"
)
```

**See:** `packages/go-plugin/skills/orchestrator-directives-skill/SKILL.md` for complete orchestrator patterns

## Parallel Task Coordination

**Problem:** Multiple parallel tasks need independent result tracking.

**Solution:** Dispatch all tasks in a single message — Claude Code runs them in parallel automatically.

```python
# Spawn 3 parallel tasks in a single message
Task(
    subagent_type="htmlgraph:codex-operator",
    description="Implement auth",
    prompt="Add JWT auth to API endpoints..."
)
Task(
    subagent_type="htmlgraph:sonnet-coder",
    description="Write tests",
    prompt="Write unit + integration tests for auth endpoints..."
)
Task(
    subagent_type="htmlgraph:gemini-operator",
    description="Update docs",
    prompt="Update API documentation for auth endpoints..."
)
# All three run in parallel; each reports results independently
```

**Benefits:**
- True parallelism (all dispatched in one message)
- Each task runs in isolation
- Cheaper agents used for each task type

## Git Workflow Patterns

### Orchestrator Pattern (REQUIRED)

When operating as orchestrator, delegate ALL git operations:

```python
# ✅ CORRECT - Delegate git workflow to subagent
Task(
    prompt="""
    Commit and push changes to git:

    Files to commit: [list files or use 'all changes']
    Commit message: "chore: update session tracking"

    Steps:
    1. Run ./scripts/git-commit-push.sh "chore: update session tracking" --no-confirm
    2. If that script doesn't exist, use manual git workflow:
       - git add [files]
       - git commit -m "message"
       - git push origin main
    3. Handle any errors (pre-commit hooks, conflicts, push failures)
    4. Retry with fixes if needed

    Report final status: success or failure with details.

    🔴 CRITICAL - Track in HtmlGraph:
    After successful commit, update the active feature/spike with completion status.
    """,
    subagent_type="general-purpose"
)

# Then read subagent result and continue orchestration
```

**Why delegate?** Git operations cascade unpredictably:
- Pre-commit hooks may fail → need code fix → retry commit
- Push may fail due to conflicts → need pull → merge → retry push
- Tests may fail in hooks → need debugging → fix → retry

**Context cost:**
- Direct execution: 5-10+ tool calls (with failures and retries)
- Delegation: 2 tool calls (Task + result review)

## Detailed Delegation Examples

### Example 1: Feature Implementation Workflow

```bash
# 1. Create feature (orchestrator does this directly)
htmlgraph feature create "Add user authentication"
htmlgraph feature start <feat-id>
```

```python
# 2. Delegate research (sequential: research blocks implementation)
Task(
    subagent_type="htmlgraph:gemini-operator",
    description="Research auth patterns",
    prompt="Research existing auth patterns: What library is used? Where is validation? What OAuth providers are supported? Document findings."
)

# 3. Implement + test in parallel (after research completes)
Task(
    subagent_type="htmlgraph:codex-operator",
    description="Implement OAuth",
    prompt="Implement OAuth flow: Add JWT auth to API endpoints, create middleware for token validation, support Google and GitHub OAuth"
)

# 4. Commit
Task(
    subagent_type="htmlgraph:copilot-operator",
    description="Commit auth feature",
    prompt="Commit and push with message: 'feat: add user authentication with OAuth support'. Handle any errors."
)
```

```bash
# 5. Mark feature complete
htmlgraph feature complete <feat-id>
```

### Example 2: Bug Fix Workflow

```bash
# 1. Create bug
htmlgraph bug create "Session timeout not working"
```

```python
# 2. Investigate + fix (sequential: need root cause before fix)
Task(
    subagent_type="htmlgraph:gemini-operator",
    description="Investigate session timeout",
    prompt="Debug session timeout: expected 30min, observed ~5min. Find config, check middleware, review logs, identify root cause."
)

Task(
    subagent_type="htmlgraph:codex-operator",
    description="Fix session timeout",
    prompt="Fix session timeout to 30 minutes. Add regression test. Verify fix works."
)

# 3. Commit
Task(
    subagent_type="htmlgraph:copilot-operator",
    description="Commit bug fix",
    prompt="Commit with message: 'fix: correct session timeout to 30 minutes'"
)
```

```bash
# 4. Mark bug resolved
htmlgraph bug complete <bug-id>
```

### Example 3: Parallel Task Coordination

```bash
# Create feature
htmlgraph feature create "Refactor API layer"
```

```python
# Dispatch 3 parallel tasks in a single message
Task(subagent_type="htmlgraph:gemini-operator", description="Update API docs", prompt="Update API documentation to reflect new endpoints")
Task(subagent_type="htmlgraph:sonnet-coder", description="Update API tests", prompt="Update test suite for refactored API endpoints")
Task(subagent_type="htmlgraph:gemini-operator", description="Create migration guide", prompt="Create migration guide for API changes")

# After all complete — commit everything
Task(
    subagent_type="htmlgraph:copilot-operator",
    description="Commit API refactor",
    prompt="Commit all API refactoring changes with message: 'refactor: update API layer with improved endpoints'"
)
```

```bash
htmlgraph feature complete <feat-id>
```

## Common Anti-Patterns to Avoid

### Anti-Pattern 1: Direct Git Execution

```python
# ❌ WRONG - Orchestrator executing git directly
Bash(command="git add .")
Bash(command="git commit -m 'feat: new feature'")
Bash(command="git push origin main")

# This will likely fail due to:
# - Pre-commit hooks
# - Merge conflicts
# - Remote changes
# Each failure consumes context and requires recovery
```

```python
# ✅ CORRECT - Delegate to subagent
Task(
    prompt="""
    Commit and push changes:
    Message: "feat: new feature"
    Handle all errors (hooks, conflicts, etc)
    """,
    subagent_type="general-purpose"
)
```

### Anti-Pattern 2: Sequential When Parallel is Possible

```python
# ❌ WRONG - Sequential delegation
Task(prompt="Update docs")
# Wait for result...
Task(prompt="Update tests")
# Wait for result...
Task(prompt="Update migration guide")

# Total time: T1 + T2 + T3
```

```python
# ✅ CORRECT - Parallel delegation
Task(prompt="Update docs")
Task(prompt="Update tests")
Task(prompt="Update migration guide")

# Total time: max(T1, T2, T3)
```

### Anti-Pattern 3: Not Using Task IDs

```python
# ❌ WRONG - No task IDs, can't distinguish results
Task(prompt="Research auth patterns")
Task(prompt="Research caching patterns")
Task(prompt="Research logging patterns")

# Which result is which?
```

```python
# ✅ CORRECT - Use task IDs
auth_id, auth_prompt = delegate_with_id("Research auth", "...", "general-purpose")
cache_id, cache_prompt = delegate_with_id("Research caching", "...", "general-purpose")
log_id, log_prompt = delegate_with_id("Research logging", "...", "general-purpose")

Task(prompt=auth_prompt, description=f"{auth_id}: Research auth")
Task(prompt=cache_prompt, description=f"{cache_id}: Research caching")
Task(prompt=log_prompt, description=f"{log_id}: Research logging")

# Retrieve results independently
auth_results = get_results_by_task_id(sdk, auth_id)
cache_results = get_results_by_task_id(sdk, cache_id)
log_results = get_results_by_task_id(sdk, log_id)
```

### Anti-Pattern 4: Not Tracking Work Items

```python
# ❌ WRONG - No feature/bug tracking
Task(prompt="Implement new feature")
# No record of what was planned or completed
```

```bash
# ✅ CORRECT - Track with HtmlGraph CLI
htmlgraph feature create "Implement new feature"
htmlgraph feature start <feat-id>
```

```python
Task(prompt="Implement new feature")
```

```bash
# Update status after completion
htmlgraph feature complete <feat-id>
```

## Summary

**Key Principles:**

1. **Delegate Everything** - Except Task(), AskUserQuestion(), TodoWrite(), and CLI operations
2. **Parallel Dispatch** - Send all independent Tasks in one message
3. **Track Work** - Use HtmlGraph CLI for all features, bugs, spikes
4. **Parallel > Sequential** - Delegate independently when possible
5. **Git = Always Delegate** - Never run git commands directly

**Benefits:**

- Context preservation (fewer tokens consumed)
- Parallel efficiency (faster completion)
- Error isolation (cleaner orchestration)
- Cognitive clarity (strategic focus)

**When in doubt, DELEGATE.**
