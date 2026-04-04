---
name: htmlgraph:plan
description: Plan development work with interactive HTML review before any code is written. Creates a standalone CRISPI plan from a topic, populates slices and questions, runs critique and validation, opens for human review, then finalizes to generate a track and features. Use when asked to plan, create a development plan, or build a feature with design clarity first.
---

# HtmlGraph Plan

Use this skill when asked to plan development work, create a parallel execution plan, organize tasks for multi-agent execution, or build a feature with human review before implementation.

**Trigger keywords:** create plan, development plan, parallel plan, plan tasks, parallelize work, organize work, task breakdown, crispi, interactive plan, plan with review, design and build, plan this feature, review before building, generate plan html, plan html file, generate the plan, scaffold plan

---

## What This Skill Does

Creates an interactive HTML plan from a topic, populates it with vertical slices and design questions, validates and critiques the structure, opens it for human review, then finalizes to generate a track and features with edges. Plans are the design space — tracks are derived on finalize.

---

## Step 0: Work Item Attribution (MANDATORY)

Before anything else:

1. Check: `htmlgraph status` — is there an active feature/track for this work?
2. If yes: `htmlgraph feature start <id>`
3. If no: `htmlgraph feature create "<title>" --track <trk-id>` then `htmlgraph feature start <id>`

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

## Step 2: Create Plan from Topic

**Plan-first flow** — create a standalone plan directly from the topic. No track needed yet.

```bash
htmlgraph plan create "<topic title>" --description "<what this plan designs>"
```

This creates `.htmlgraph/plans/plan-<hex8>.html` with the interactive CRISPI template: empty design sections, no slices, ready for population.

Note the returned plan ID (e.g. `plan-a1b2c3d4`).

---

## Step 3: Populate the Plan

### 3a. Add Vertical Slices

Design vertical slices (not horizontal layers). Each slice should be independently deployable with its own tests.

```bash
htmlgraph plan add-slice <plan-id> "<slice title>"
# Repeat for each slice
```

### 3b. Add Design Questions

Pre-select sensible defaults — the human overrides only where they disagree:

```bash
htmlgraph plan add-question <plan-id> "Error message length?" \
  --options "one-line:Keep hints to a single sentence,two-line:Allow a second line with more context" \
  --description "Longer messages give agents more guidance but consume more context tokens."
```

### 3c. Set Section Content

Fill in the design discussion and outline:

```bash
htmlgraph plan set-section <plan-id> PLAN_DESIGN_CONTENT '<p>Summary of what will be built and why.</p>'
htmlgraph plan set-section <plan-id> PLAN_OUTLINE_CONTENT '<h4>Key Changes</h4><pre><code>func NewHelper() error</code></pre>'
```

### 3d. Set Slice Details

Add test strategy, dependencies, and affected files for each slice:

```bash
htmlgraph plan set-slice <plan-id> 1 \
  --tests "Unit: ErrNotFound returns correct format. Integration: resolveID failure includes hint." \
  --deps "none (foundation slice)" \
  --files "internal/workitem/errors.go, internal/workitem/resolve.go"
```

Repeat `set-slice` for each slice number.

---

## Step 4: Validate Plan Structure

```bash
htmlgraph plan validate <plan-id>
```

Checks: required sections exist, slice/graph node counts match, sections JSON is consistent, status is valid. Fix any errors before proceeding.

---

## Step 5: Critique (if >= 3 slices)

```bash
htmlgraph plan critique <plan-id>
```

Outputs structured JSON with slices, questions, dependencies, and a complexity assessment. Plans with fewer than 3 slices skip critique (`critique_warranted: false`).

For plans that warrant critique, pipe the output to AI reviewers for design feedback. Address any issues before opening for human review.

---

## Step 6: Open for Human Review (PAUSE HERE)

```bash
htmlgraph plan open <plan-id>
htmlgraph plan wait <plan-id> --timeout 1h
```

Tell the human:

```
Plan ready for review: http://localhost:8080/plans/<plan-id>.html

Please:
1. Read the Design Discussion — does it describe what you want built?
2. Review the Open Questions — defaults are pre-selected, override where you disagree
3. Check each Slice — approve or flag for revision
4. Add comments on any section
5. Click Finalize when ready

I will wait until you finalize the plan before writing any code.
```

**STOP. Do not proceed until the human finalizes the plan.**

---

## Step 7: Read Structured Feedback

After the human finalizes:

```bash
htmlgraph plan read-feedback <plan-id>
```

Parse the JSON output:
- If any slice has `false`: summarize what needs revision. Ask the human — revise now, or proceed without it?
- If revising: update the plan, loop to Step 6.
- If proceeding: note which slices are excluded from execution.

---

## Step 8: Finalize — Generate Work Items

```bash
htmlgraph plan finalize <plan-id>
```

This reads the plan's approved slices and generates:
- A **track** (trk-{hex8}) for the project
- **Features** (feat-{hex8}) for each approved slice
- **Edges**: part_of, contains, blocked_by based on slice dependencies
- An **implemented_in** link from the plan to the track

Finalize is **idempotent** — safe to re-run. Already-finalized plans return the existing track and features.

---

## Step 9: Hand Off to Execute

Announce what was decided, then dispatch:

```
Plan finalized. Track: <track-id>

Approved slices (N of M):
  feat-XXXX  Slice 1 title       -> implement
  feat-XXXX  Slice 2 title       -> implement

Design decisions:
  Q1: answer (overridden from default)
  Q2: answer (default accepted)
```

Embed each question answer into affected slice descriptions as "Accepted Design Decisions". Then hand off to `/htmlgraph:execute <track-id>`.

---

## Alternative: Retroactive (Track-First)

For existing tracks that already have features:

```bash
htmlgraph plan generate <track-id>
```

Scaffolds a CRISPI plan from the track's existing features, including dependency graph and descriptions. Useful for design review of work already in progress.

---

## Alternative: Dual-Mode Generate

`plan generate` auto-detects its argument:

| Input | Mode |
|-------|------|
| `trk-*`, `feat-*` | Retroactive: scaffold from work item |
| `plan-*` | Re-scaffold existing plan |
| Free text | Plan-first: create from topic (same as `plan create`) |

---

## Key Rules

- **Plans are the design space** — tracks and features are derived on finalize
- **All slices must have test strategies** before critique
- **Critique is complexity-gated** — skip for plans with <3 slices
- **Only approved slices** generate features on finalize
- **Finalize is idempotent** — safe to re-run without creating duplicates
- **TDD is mandatory** — every dispatched task includes tests before implementation
