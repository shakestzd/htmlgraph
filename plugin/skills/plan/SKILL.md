---
name: htmlgraph:plan
description: Plan development work with interactive HTML review before any code is written. Generates a human-reviewable plan, waits for structured feedback, then hands off to execute. Use when asked to plan, create a development plan, parallelize work, or build a feature with design clarity first.
---

# HtmlGraph Plan

Use this skill when asked to plan development work, create a parallel execution plan, organize tasks for multi-agent execution, or build a feature with human review before implementation.

**Trigger keywords:** create plan, development plan, parallel plan, plan tasks, parallelize work, organize work, task breakdown, crispi, interactive plan, plan with review, design and build, plan this feature, review before building

---

## What This Skill Does

Generates an interactive HTML plan, opens it for human review, reads structured feedback, then hands off to `/htmlgraph:execute`.

The human sees the plan, approves sections, answers open design questions, and clicks Finalize — before any code is written. Only finalized, approved slices are dispatched.

---

## Step 0: Work Item Attribution (MANDATORY)

Before anything else:

1. Check: `htmlgraph status` — is there an active feature/track for this work?
2. If yes: `htmlgraph feature start <id>`
3. If no: `htmlgraph feature create "<title>"` then `htmlgraph feature start <id>`

Plans without attribution produce untracked work.

---

## Step 1: Research (Parallel Agents)

Spawn research agents in a single message — do not proceed until both complete:

```
Agent(description="Understand current codebase state", subagent_type="htmlgraph:researcher",
      prompt="Investigate [area]. Answer: current state, key files, existing patterns, constraints.")

Agent(description="Check for prior work", subagent_type="Explore",
      prompt="Search .htmlgraph/ for any features or spikes related to [area]. Report feature IDs and status.")
```

Research must answer:
- Current state of the codebase in this area
- Desired end state (from the request)
- Open design questions (choices that affect the architecture)
- Candidate vertical slices (end-to-end, not horizontal layers)
- Real dependencies between slices

---

## Step 2: Generate the Interactive Plan HTML

Using research findings, generate the interactive plan file.

### Slice Design Rules

- Each slice is independently deployable (not "DB layer" or "API layer")
- Each slice is vertical: CLI command + storage + tests — all in one slice
- Each slice has a concrete test strategy (what tests prove it works)
- Slices that depend on another slice say so explicitly
- If a slice agent prompt would exceed 40 instructions, split the slice

### Dependency vs File Conflict

**Dependency** (use `addBlockedBy`): Task B literally cannot be written until Task A exists.
- Task B imports a module Task A creates
- Task B extends an interface Task A defines

**File conflict** (handle at merge time, NOT with `addBlockedBy`): Both tasks edit the same file but don't depend on each other's logic.
- Both tasks add a registration line to `main.go`
- Both tasks add an entry to a config file

### Create Feature IDs for Each Slice

```bash
htmlgraph feature create "<slice-title>" --track <track-id>
# Repeat for each slice. Note the returned feature IDs.
```

### Write the Plan File

Output path: `.htmlgraph/plans/plan-<kebab-title>.html`

Use the template from `internal/templates/plan-template.html`. Every interactive element needs `data-*` attributes so `htmlgraph plan read-feedback` can parse structured output.

Key sections to include:
- **Summary** — one paragraph describing what will be built and why
- **Open Questions** — design decisions with pre-selected sensible defaults (human overrides only where they disagree)
- **Slices** — one card per slice, with: title, feature ID, goal, files, test strategy, dependencies
- **Finalize button** — triggers plan state transition to `finalized`

Pre-selecting defaults is important: the human should only need to act where they have a strong opinion.

---

## Step 3: Open for Human Review (PAUSE HERE)

Start the dashboard if not already running:

```bash
htmlgraph serve &
```

Tell the human:

```
Plan ready for review: http://localhost:8080/plans/plan-<id>.html

Please:
1. Read the Summary — does it describe what you want built?
2. Review the Open Questions — defaults are pre-selected, override where you disagree
3. Check each Slice — approve or flag for revision
4. Add comments on any section
5. Click Finalize when ready

I will wait until you finalize the plan before writing any code.
```

**STOP. Do not proceed until the human finalizes the plan.**

---

## Step 4: Read Structured Feedback

After the human finalizes:

```bash
htmlgraph plan read-feedback <plan-id>
```

This outputs JSON:
```json
{
  "status": "finalized",
  "section_approvals": { "summary": true, "questions": true },
  "question_answers": { "q-delivery-model": "async", "q-storage-format": "sqlite" },
  "slice_approvals": { "slice-1": true, "slice-2": false },
  "comments": { "slice-2": "Needs rate limiting" }
}
```

Parse it:
- If any slice has `false`: summarize what needs revision. Ask the human — revise now, or proceed without it?
- If revising: update the plan HTML, set status back to `draft`, loop to Step 3.
- If proceeding: note which slices are excluded from execution.

---

## Step 5: Announce Finalized Plan

Announce what was decided:

```
Plan finalized. Here's what was approved:

Slices (3 of 3 approved):
  slice-1  feat-XXXX  CLI command skeleton       -> implement
  slice-2  feat-XXXX  Storage layer              -> implement
  slice-3  feat-XXXX  Integration tests          -> implement

Design decisions made:
  Delivery model: async (overridden from default "sync")
  Storage format: sqlite (default accepted)

These answers are wired into each slice's task description.
```

Reconcile question answers into slice specs. If the human chose "async delivery", the dispatch slice description must say "implement async delivery, not sync."

---

## Step 6: Create Tasks and Hand Off to Execute

Create a `TaskCreate` for each approved slice. Descriptions must be self-contained — the executing agent has no other context. **TDD is mandatory** — every task includes test specifications written before implementation.

```
TaskCreate(
  subject="{feature_id}: {slice_name}",
  description="""
## Goal
[One sentence from the plan, incorporating question answers]

## Files to Create/Edit
- NEW: path/to/new_file.go
- EDIT: path/to/existing_file.go (add registration line)

## Shared Files (expect merge conflict)
- main.go: Add registration line — orchestrator resolves conflicts

## Accepted Design Decisions
[List any question answers that affect this slice]

## Test Plan (TDD — write tests FIRST)
[Concrete input/output examples; tests must compile and FAIL before implementation]

## Quality Gate
go build ./... && go vet ./... && go test ./...

## Attribution
htmlgraph feature start {feature_id}
htmlgraph feature complete {feature_id}
""",
  metadata={"feature_id": "feat-XXXX", "slice_num": 1, "plan_id": "<id>",
            "agent_tier": "sonnet"}
)
```

Wire real dependencies only:

```
TaskUpdate(taskId="2", addBlockedBy=["1"])  # only if slice-2 imports slice-1's output
```

After all tasks are created, present the dispatch summary:

```
Plan: [Name]
Total slices: N | Independent: X | Blocked: Y

DISPATCH IMMEDIATELY (X slices, all parallel):
  #1  feat-001 [sonnet] CLI command skeleton     files: cmd.go, main.go
  #2  feat-002 [sonnet] Storage layer            files: store.go

BLOCKED (dispatch after dependencies complete):
  #3  feat-003 [sonnet] Integration tests        blocked-by: #1, #2

To execute: /htmlgraph:execute
```

### Human Review Between Slices

After each slice completes, before dispatching the next:
1. Show a diff summary: files changed, lines added, tests added
2. Ask: "Slice N complete. Approve and continue, or review code first?"
3. Wait for approval. Do not auto-dispatch the next slice.

---

## Agent Type Selection

| Agent | When to Use |
|-------|------------|
| `htmlgraph:haiku-coder` | Single-file, clear spec, <50 lines, no design decisions |
| `htmlgraph:sonnet-coder` | Multi-file, needs codebase context, moderate complexity |
| `htmlgraph:opus-coder` | Architecture decisions, complex algorithms, novel design |

Default to sonnet unless the task is trivially simple (haiku) or requires deep reasoning (opus).

---

## Related Skills

- **[/htmlgraph:execute](/htmlgraph:execute)** — Execute the finalized task list with dependency-driven dispatch
- **[/htmlgraph:parallel-status](/htmlgraph:parallel-status)** — Monitor slice execution progress
- **[/htmlgraph:cleanup](/htmlgraph:cleanup)** — Clean up worktrees after completion
