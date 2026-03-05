---
name: htmlgraph:cleanup
description: Clean up git worktrees and branches after parallel execution completes. Activate when asked to clean up worktrees, remove branches, or finalize parallel work.
---

# HtmlGraph Cleanup

Use this skill after parallel execution completes to clean up worktrees and branches.

**Trigger keywords:** cleanup, clean up worktrees, remove branches, finalize, finish parallel

---

## Quick Cleanup

```bash
# Remove all merged worktrees
uv run htmlgraph worktree cleanup

# Force remove all worktrees (including unmerged)
uv run htmlgraph worktree cleanup --force

# Remove specific task worktree
uv run htmlgraph worktree cleanup --task task-001
```

---

## Full Cleanup Workflow

### Step 1: Check Status First

```bash
# See what's still active
uv run htmlgraph worktree status
```

### Step 2: Merge Any Remaining Work

```bash
# Merge completed tasks first
uv run htmlgraph worktree merge <task-id>
```

### Step 3: Run Cleanup

```bash
# Safe cleanup - only removes merged branches
uv run htmlgraph worktree cleanup

# Or use the shell script directly
./scripts/worktree-cleanup.sh
```

### Step 4: Verify

```bash
# Confirm no stale worktrees remain
git worktree list

# Run full quality gates
uv run ruff check --fix && uv run ruff format && uv run mypy src/ && uv run pytest
```

---

## Related Skills

- **[/htmlgraph:plan](/htmlgraph:plan)** - Create execution plans
- **[/htmlgraph:execute](/htmlgraph:execute)** - Execute plans
- **[/htmlgraph:parallel-status](/htmlgraph:parallel-status)** - Monitor progress
