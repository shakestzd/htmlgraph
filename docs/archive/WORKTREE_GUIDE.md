# Parallel Worktree Development Guide

Generalized worktree management scripts for coordinated parallel development using HtmlGraph feature tracking.

## Quick Start

```bash
# 1. Plan your work (creates features in HtmlGraph)
/htmlgraph:plan

# 2. Create worktrees for all planned features
./scripts/worktree-setup.sh

# 3. Develop in parallel (one feature per worktree)
cd worktrees/feature-name-1
# ... make changes, commit ...

cd ../feature-name-2
# ... make changes, commit ...

# 4. Check status anytime
./scripts/worktree-status.sh

# 5. Merge completed features back to main
./scripts/worktree-merge.sh feature-name-1

# 6. Clean up
./scripts/worktree-cleanup.sh
```

## Scripts Overview

### 1. `worktree-setup.sh` - Create Parallel Worktrees

Creates git worktrees for all todo/in-progress features from HtmlGraph.

**Usage:**
```bash
./scripts/worktree-setup.sh [--base-dir DIR] [--branch-prefix PREFIX]
```

**Options:**
- `--base-dir DIR` - Where to create worktrees (default: `worktrees`)
- `--branch-prefix PREFIX` - Branch name prefix (default: `feature`)

**What it does:**
1. Reads HtmlGraph features (todo/in-progress status)
2. Creates git worktree for each feature
3. Creates new branches if they don't exist
4. Pre-warms Python virtual environments in parallel

**Example:**
```bash
./scripts/worktree-setup.sh --base-dir work --branch-prefix task
# Creates: work/feature-id-1, work/feature-id-2, etc.
```

### 2. `worktree-status.sh` - View Worktree Status

Shows all active worktrees with their branches, commit counts, and latest commits.

**Usage:**
```bash
./scripts/worktree-status.sh [--base-dir DIR]
```

**Options:**
- `--base-dir DIR` - Worktrees directory (default: `worktrees`)

**Output:**
```
=== Parallel Worktree Status ===

TASK                      BRANCH                         COMMITS    LAST COMMIT
----                      ------                         -------    -----------
task-1                    feature/task-1                 3          a1b2c3d fix: update handler
task-2                    feature/task-2                 2          e4f5g6h feat: add validation
task-3                    feature/task-3                 1          i7j8k9l refactor: simplify logic

Total worktrees: 3
```

**Example:**
```bash
# Monitor progress during parallel development
watch -n 5 ./scripts/worktree-status.sh
```

### 3. `worktree-merge.sh` - Merge a Feature Branch

Merges a completed feature branch back to main with tests.

**Usage:**
```bash
./scripts/worktree-merge.sh <worktree-name> [base-branch]
```

**Arguments:**
- `<worktree-name>` - Name of worktree to merge (required)
- `[base-branch]` - Target branch (default: `main`)

**What it does:**
1. Runs tests in the worktree
2. Switches to main branch and pulls latest
3. Merges with `--no-ff` for clear history
4. Removes worktree and feature branch
5. Cleans up stale references

**Example:**
```bash
# Merge task-1 to main (after tests pass)
./scripts/worktree-merge.sh task-1

# Merge to specific branch
./scripts/worktree-merge.sh task-1 develop
```

### 4. `worktree-cleanup.sh` - Clean Up Worktrees

Removes merged worktrees and optionally unmerged ones.

**Usage:**
```bash
./scripts/worktree-cleanup.sh [--base-dir DIR] [--force]
```

**Options:**
- `--base-dir DIR` - Worktrees directory (default: `worktrees`)
- `--force` - Remove unmerged worktrees too

**Behavior:**
- By default, only removes **merged** worktrees
- With `--force`, removes **all** worktrees (even unmerged)
- Prunes stale git references
- Removes empty base directory

**Example:**
```bash
# Clean up merged worktrees
./scripts/worktree-cleanup.sh

# Force cleanup all (use with caution)
./scripts/worktree-cleanup.sh --force

# Use custom directory
./scripts/worktree-cleanup.sh --base-dir work
```

## Workflow Examples

### Example 1: Feature Development Wave

```bash
# Plan work using HtmlGraph
/htmlgraph:plan

# Create worktrees from plan
./scripts/worktree-setup.sh

# Work on features in parallel
cd worktrees/auth-service && git checkout -b auth-work && make changes...
cd worktrees/api-server && git checkout -b api-work && make changes...
cd worktrees/database && git checkout -b db-work && make changes...

# Monitor progress
./scripts/worktree-status.sh

# Merge when ready
./scripts/worktree-merge.sh auth-service
./scripts/worktree-merge.sh api-server
./scripts/worktree-merge.sh database

# Clean up
./scripts/worktree-cleanup.sh
```

### Example 2: Fix Multiple Bugs in Parallel

```bash
# Create features for each bug
/htmlgraph:plan "bug-fixes"

# Setup worktrees
./scripts/worktree-setup.sh

# Fix in parallel
for bug in worktrees/*/; do
  cd "$bug"
  make && npm test
  cd -
done

# Check which are ready
./scripts/worktree-status.sh

# Merge all fixed bugs
for wt in worktrees/*/; do
  name=$(basename "$wt")
  ./scripts/worktree-merge.sh "$name" || echo "⚠️ Merge failed: $name"
done

# Cleanup
./scripts/worktree-cleanup.sh
```

### Example 3: Continuous Development

```bash
# Terminal 1: Setup and monitor
./scripts/worktree-setup.sh
watch -n 5 ./scripts/worktree-status.sh

# Terminal 2-4: Work on different features
cd worktrees/feature-1 && vim code...
cd worktrees/feature-2 && vim code...
cd worktrees/feature-3 && vim code...

# Terminal 1: When feature done, merge
./scripts/worktree-merge.sh feature-1
./scripts/worktree-merge.sh feature-2
./scripts/worktree-merge.sh feature-3

# Cleanup
./scripts/worktree-cleanup.sh
```

## Integration with HtmlGraph

These scripts automatically:
- Read feature status from HtmlGraph SDK
- Create worktrees only for todo/in-progress features
- Use HtmlGraph feature IDs for branch naming
- Track development progress via feature status

**Check HtmlGraph status:**
```bash
uv run htmlgraph status
uv run htmlgraph snapshot --summary
```

## Technical Details

### Branch Naming Convention

Features are mapped to branches using this pattern:
```
HtmlGraph Feature ID: feat-a1b2c3d4
Short Name: a1b2c3d
Branch: feature/a1b2c3d
Worktree: worktrees/a1b2c3d
```

### Worktree Isolation

Each worktree is a **complete git checkout** with:
- Isolated working directory
- Independent staging area
- Separate HEAD position
- Own virtual environment (when pre-warmed)

Changes in one worktree **do not affect others**.

### Virtual Environment Warm-up

Setup script pre-warms venvs in parallel for speed:
```bash
# Takes ~60 seconds per worktree
# Run in parallel with xargs -P 5
```

### Merge Behavior

The merge script:
1. **Tests first** - Runs pytest before merge
2. **No-ff merge** - Creates merge commit for clear history
3. **Atomic cleanup** - Removes worktree and branch together
4. **Safe deletion** - Won't delete if tests fail

## Troubleshooting

### Issue: "No pending tasks found"

**Cause:** No features in HtmlGraph with todo/in-progress status

**Fix:**
```bash
# Create a plan first
/htmlgraph:plan

# Or manually create features
uv run python -c "
from htmlgraph import SDK
sdk = SDK()
sdk.features.create('Feature 1')
sdk.features.create('Feature 2')
"
```

### Issue: Worktree already exists

**Cause:** Script tried to create duplicate worktree

**Fix:**
```bash
# Check existing worktrees
git worktree list

# Remove stale ones
git worktree remove worktrees/old-task
git worktree prune
```

### Issue: Merge fails with "Tests failed"

**Cause:** Tests don't pass in the worktree

**Fix:**
```bash
# Debug in the worktree
cd worktrees/feature-name
uv run pytest -v

# Fix issues
# Re-run tests
uv run pytest

# Try merge again
cd ../..
./scripts/worktree-merge.sh feature-name
```

### Issue: Out of disk space with many worktrees

**Cause:** Each worktree has full copy of repo + node_modules

**Solution:**
```bash
# Clean node_modules in all worktrees
for wt in worktrees/*/; do
  cd "$wt"
  rm -rf node_modules
  cd -
done

# Or use cleanup
./scripts/worktree-cleanup.sh --force
```

## Performance Tips

### 1. Parallel Setup

Setup creates venvs in parallel by default. Speed is limited by:
- Network (downloading packages)
- Disk I/O (extracting files)
- CPU (compilation of C extensions)

### 2. Parallel Development

Best when tasks are **truly independent**:
- ✅ Different files being modified
- ✅ Different languages/frameworks
- ✅ Different dependencies
- ❌ Same file (merge conflicts)
- ❌ Shared state (database migrations)

### 3. Batch Merging

Merge in dependency order:
```bash
# Merge dependencies first
./scripts/worktree-merge.sh database-schema
./scripts/worktree-merge.sh api-server
./scripts/worktree-merge.sh web-client
```

## See Also

- [HtmlGraph Feature Tracking](./AGENTS.md#feature-tracking)
- [Deployment Automation](./scripts/README.md)
- [Development Workflow](./CLAUDE.md)
