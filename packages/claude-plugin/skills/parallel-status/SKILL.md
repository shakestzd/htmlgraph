---
name: htmlgraph:parallel-status
description: Monitor the status of parallel execution across git worktrees. Activate when asked for parallel status, worktree progress, or execution monitoring.
---

# HtmlGraph Parallel Status

Use this skill to monitor parallel execution progress across worktrees.

**Trigger keywords:** parallel status, worktree status, execution progress, monitor tasks, check progress

---

## Quick Status

```bash
# Show all worktree status
uv run htmlgraph worktree status

# Or use the shell script directly
./scripts/worktree-status.sh
```

---

## Detailed Monitoring

### Worktree Status

```bash
# Table view of all worktrees
uv run htmlgraph worktree status

# Git-level view
git worktree list

# Recent commits across all branches
git for-each-ref --sort=-committerdate refs/heads/ --format='%(refname:short) %(committerdate:relative) %(subject)'
```

### HtmlGraph Feature Status

```bash
# Show all features and their status
uv run htmlgraph feature list

# Show specific feature
uv run htmlgraph feature show <id>
```

### Wave Progress

```python
from htmlgraph import SDK
sdk = SDK()

# Check feature statuses
features = sdk.features.all()
done = [f for f in features if getattr(f, 'status', '') == 'done']
in_progress = [f for f in features if getattr(f, 'status', '') == 'in_progress']
todo = [f for f in features if getattr(f, 'status', '') == 'todo']

print(f"Done: {len(done)} | In Progress: {len(in_progress)} | Todo: {len(todo)}")
```

---

## Related Skills

- **[/htmlgraph:plan](/htmlgraph:plan)** - Create execution plans
- **[/htmlgraph:execute](/htmlgraph:execute)** - Execute plans
- **[/htmlgraph:cleanup](/htmlgraph:cleanup)** - Clean up after completion
