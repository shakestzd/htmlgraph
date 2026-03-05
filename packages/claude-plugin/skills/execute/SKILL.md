---
name: htmlgraph:execute
description: Execute a parallelized plan using git worktrees and multiple Claude agents. Activate when asked to run a plan, execute tasks in parallel, or start parallel development.
---

# HtmlGraph Parallel Execute

Use this skill when asked to execute a plan, run tasks in parallel, or start multi-agent development work.

**Trigger keywords:** execute plan, run plan, run tasks, parallelize work, work in parallel, start execution, dispatch agents

---

## Core Principle: Wave-Based Execution

Tasks in the same wave run concurrently in isolated worktrees. Each wave must complete (with quality gates) before the next wave starts.

```
Wave 1: [task-A] [task-B] [task-C]  ← all in parallel
           ↓ quality gates ↓
Wave 2: [task-D] [task-E]           ← depends on Wave 1
           ↓ quality gates ↓
Wave 3: [task-F]                    ← depends on Wave 2
```

---

## Step 1: Load the Plan

```python
from htmlgraph import SDK
sdk = SDK(agent="claude-code")

# Load the plan created by /htmlgraph:plan
track = sdk.tracks.get_latest()
plan = track.get_plan()

print(plan.summary())
print(f"Waves: {len(plan.waves)}")
print(f"Tasks: {plan.task_count}")
```

---

## Step 2: Set Up Worktrees (if not done)

```bash
# Set up a worktree for each task
uv run htmlgraph worktree setup

# Pre-warm each worktree (faster agent startup)
for worktree in $(git worktree list --porcelain | grep "worktree" | awk '{print $2}'); do
    (cd "$worktree" && uv sync) &
done
wait

# Verify
git worktree list
```

---

## Step 3: Execute Each Wave

For each wave, spawn all tasks simultaneously in a single message:

```python
# Wave N execution pattern
wave = plan.waves[N]
print(f"Executing Wave {N}: {len(wave.tasks)} tasks in parallel")

for task in wave.tasks:
    Task(
        description=task.title,
        prompt=generate_task_prompt(task, plan),
        subagent_type=f"htmlgraph:{task.agent_type}-coder",
        isolation="worktree"  # Each agent works in isolated branch
    )

# All tasks in this wave run in parallel - wait for all to complete
```

### Task Prompt Template

Each agent receives a fully specified prompt:

```
## Task: {task.id}
**Title:** {task.title}
**Priority:** {task.priority}
**Agent type:** {task.agent_type}

## Specification
{task.description}

## Files to Create/Edit
{task.files}

## Do NOT Touch (other agents own these)
{files_owned_by_other_wave_tasks}

## Acceptance Criteria
{task.acceptance_criteria}

## Quality Requirements
Before marking complete, ALL must pass:
- uv run ruff check --fix && uv run ruff format
- uv run mypy src/
- uv run pytest (relevant tests only: pytest {task.test_paths})

## HtmlGraph Tracking
```python
from htmlgraph import SDK
sdk = SDK(agent='htmlgraph:{task.agent_type}-coder')
feature = sdk.features.get('{task.id}')
feature.set_status('in-progress').save()
# ... do work ...
feature.set_status('done').save()
```

## Commit Your Work
After all checks pass:
git add {task.files}
git commit -m "feat({task.id}): {task.title}"

Report back: summary of changes made, files modified, tests added.
```

---

## Step 4: Quality Gates After Each Wave

After all tasks in a wave complete, run gates on main before merging:

```bash
# Merge completed tasks to main
for task in wave.completed_tasks:
    git checkout main
    git merge --no-ff feature/{task.id} -m "Merge: {task.title}"

# Run quality gates on merged result
uv run ruff check --fix
uv run ruff format
uv run mypy src/
uv run pytest

# Only proceed to next wave if all gates pass
```

If quality gates fail:
1. Identify which merged task introduced the failure
2. Fix in a follow-up commit on main (or revert and fix in the task's branch)
3. Re-run gates before proceeding

---

## Step 5: Merge Strategy

```bash
# Merge each completed task branch to main
git checkout main

# Use --no-ff to preserve branch history
git merge --no-ff feature/<task-id> -m "feat: merge <task-title>"

# Push after each wave completes
git push origin main
```

### Conflict Resolution

If merge produces a conflict:
1. Identify which tasks touched the same file (this was a missed conflict)
2. For the conflicting file, manually resolve or assign one task as the canonical version
3. Record in plan as a lesson for future conflict detection

---

## Step 6: Proceed to Next Wave

Only start the next wave after:
- All current wave tasks report complete
- All branches merged to main
- Quality gates pass on main

```python
# Update plan progress
for task in wave.tasks:
    if task.status == "complete":
        feature = sdk.features.get(task.id)
        feature.set_status("done").save()

# Check if next wave is unblocked
next_wave = plan.get_next_wave()
if next_wave and next_wave.is_ready():
    print(f"Wave {next_wave.number} is ready - proceeding...")
    # Execute next wave (repeat Step 3)
```

---

## Error Handling

### Blocked Task (dependency not met)
```
If task reports: "Cannot start - {dependency} not complete"
→ Check if dependency task actually finished
→ If dependency failed, fix it first
→ If dependency succeeded but status not updated, update manually
```

### Test Failures
```
If task reports: "Tests failing: {test_names}"
→ Do NOT merge to main
→ Assign as follow-up fix task
→ Mark original task as "needs-fix"
→ Continue other tasks in wave that don't depend on failing task
```

### Merge Conflicts
```
If merge fails with conflicts:
→ Resolve conflicts on a resolution branch
→ Run quality gates on resolution branch
→ Merge resolution branch to main
→ Update conflict detection rules for future plans
```

---

## Monitoring During Execution

While agents are running, you can check status:

```bash
# See active worktrees
git worktree list

# Check which branches have recent commits
git for-each-ref --sort=-committerdate refs/heads/ --format='%(refname:short) %(committerdate:relative)'
```

Or use `/htmlgraph:parallel-status` for a complete view.

---

## Integration with HtmlGraph

```python
from htmlgraph import SDK
sdk = SDK(agent="claude-code")

# Track execution
spike = sdk.spikes.create("Execution: Wave 1")
spike.set_findings(f"""
Wave 1 status:
- task-001: DONE (3 files, 2 tests)
- task-002: DONE (1 file, 5 tests)
- task-003: IN PROGRESS

Quality gates: PASSING
Proceeding to Wave 2.
""").save()
```

---

## Related Skills

- **[/htmlgraph:plan](/htmlgraph:plan)** - Create the plan before executing
- **[/htmlgraph:parallel-status](/htmlgraph:parallel-status)** - Monitor progress
- **[/htmlgraph:cleanup](/htmlgraph:cleanup)** - Clean up after completion
- **[/code-quality](/code-quality)** - Quality gate details
