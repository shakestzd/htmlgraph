---
name: htmlgraph:plan
description: Plan development work with interactive marimo notebook review before any code is written. Generates a YAML plan, populates slices and questions, runs dual-agent critique, opens marimo for human review (approvals persist to SQLite), then reads finalized decisions to dispatch tasks. Use when asked to plan, create a development plan, or build a feature with design clarity first.
---

# HtmlGraph Plan

Use this skill when asked to plan development work, create a parallel execution plan, organize tasks for multi-agent execution, or build a feature with human review before implementation.

**Trigger keywords:** create plan, development plan, parallel plan, plan tasks, parallelize work, organize work, task breakdown, crispi, interactive plan, plan with review, design and build, plan this feature, review before building, generate plan, scaffold plan

---

## What This Skill Does

Generates a structured YAML plan, populates it with vertical slices, design questions with recommendations, and dual-agent critique. Opens a marimo notebook for interactive human review — approvals persist to SQLite on every click. After finalization, reads approved slices and design decisions to dispatch tasks via `/htmlgraph:execute`.

**Architecture:**
- **YAML** = plan content (agent-written, read by notebook)
- **SQLite `plan_feedback`** = human approvals (persisted on every interaction)
- **Marimo notebook** = interactive review UI (reads both, writes approvals to SQLite)
- **Static HTML** = archival export (on finalize)

---

## Step 0: Work Item Attribution (MANDATORY)

Before anything else:

1. Check: `htmlgraph status` — is there an active feature/track for this work?
2. If yes: `htmlgraph feature start <id>`
3. If no: `htmlgraph feature create "<title>" --track <trk-id>` then `htmlgraph feature start <id>`

Plans without attribution produce untracked work.

---

## Step 1b: Check for Existing Plan from Plan Mode

If the user just exited Claude Code's plan mode, the `ExitPlanMode` hook may have already created a skeleton YAML:

```bash
# Check for recently-created plan YAMLs (within last 5 minutes)
find .htmlgraph/plans/ -name "plan-*.yaml" -mmin -5 2>/dev/null
```

If a recent YAML exists:
1. Read it — it will have slices with `what` populated but `why`, `done_when`, `tests`, `files` empty
2. **Do NOT create a new plan** — use this as the base
3. Skip to Step 2 (research) to gather context, then enrich the existing YAML in Step 2's output
4. Use `htmlgraph plan rewrite-yaml <plan-id>` to update the enriched version

If no recent YAML exists, proceed to Step 2 as normal.

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
- Prior art or existing patterns to leverage

**Skip research only for:** trivial changes (<10 lines), bug fixes with known root cause, documentation-only changes.

---

## Step 2: Generate Plan YAML

Create the plan YAML file. The orchestrator agent constructs the full YAML structure and writes it directly. The file goes in `.htmlgraph/plans/plan-<hex8>.yaml`.

Generate the plan ID:
```bash
htmlgraph plan create "<title>" --description "<description>"
```
Note the returned plan ID. Then create the YAML file at `.htmlgraph/plans/<plan-id>.yaml`.

### YAML Schema

Every plan YAML MUST follow this exact structure:

```yaml
meta:
  id: plan-<hex8>          # from plan create
  track_id: <trk-id>       # if retroactive, empty if plan-first
  title: "<plan title>"
  description: >
    One paragraph describing what this plan designs and why.
  created_at: "YYYY-MM-DD"
  status: draft             # draft | review | finalized
  created_by: claude-opus   # agent that generated it

design:
  problem: >
    What is the current state? What's wrong? Why does this matter?
    Include measurements, evidence, or user impact.
  goals:
    - "**Goal 1** — measurable outcome"
    - "**Goal 2** — measurable outcome"
  constraints:
    - "Constraint 1 — why it exists"
    - "Constraint 2 — why it exists"
  approved: false
  comment: ""

slices:
  - id: feat-<hex8>         # existing feature ID or generated
    num: 1
    title: "<slice title>"
    what: >
      What exactly will be implemented. Be specific — name functions,
      files, APIs. An agent reading this should know what to build.
    why: >
      Why this slice exists. What problem does it solve? What breaks
      without it? Link to the design goals.
    files:
      - path/to/file1.go
      - path/to/file2.go
    deps: []                 # slice numbers this depends on, e.g. [1, 3]
    done_when:
      - "Acceptance criterion 1 — testable, concrete"
      - "Acceptance criterion 2 — testable, concrete"
    effort: S                # S | M | L
    risk: Low                # Low | Med | High
    tests: |
      Unit: specific test description with expected input/output
      Integration: specific integration test
      Regression: what existing tests must still pass
    approved: false
    comment: ""

questions:
  - id: q-<kebab-name>
    text: "The design question in plain English?"
    description: >
      Context paragraph explaining WHY this question matters, what
      tradeoffs are involved, and what evidence exists for each option.
      The reviewer needs this to make an informed decision.
    recommended: <option-key>  # agent's best judgment
    options:
      - key: option-a
        label: "A: Short name — Full description of what this option means and its implications"
      - key: option-b
        label: "B: Short name — Full description of what this option means and its implications"
    answer: null               # null = unanswered, set by human

critique: null                 # null = not yet run, populated by Step 4
```

### Quality Requirements for Each Section

**Slices — ALL fields are MANDATORY:**
- `what`: Specific enough that an agent can implement it without asking questions
- `why`: Links to a design goal or explains the business need
- `files`: At least one file path (check codebase if unsure)
- `done_when`: At least two concrete, testable acceptance criteria
- `tests`: At least one unit test and one integration/regression test
- `effort`: S (<50 lines), M (50-200 lines), L (>200 lines)
- `risk`: Low (pure addition), Med (modifies existing), High (changes hot path or shared interfaces)

**Questions — ALL fields are MANDATORY:**
- `description`: At least 2 sentences explaining context and tradeoffs
- `recommended`: Agent MUST pick a default — the human overrides only where they disagree
- `options`: At least 2 options, each with a descriptive label (not just a key)

**Design — ALL subsections are MANDATORY:**
- `problem`: Evidence-based description of current state
- `goals`: Measurable outcomes (not vague aspirations)
- `constraints`: Real limitations that affect design choices

---

## Step 3: Validate Plan Structure

Before proceeding, validate the YAML:

1. All required fields present on every slice
2. All questions have description + recommended
3. Design has problem + goals + constraints
4. Slice deps reference valid slice numbers
5. Effort and risk values are valid (S/M/L, Low/Med/High)
6. No duplicate slice numbers or question IDs

```bash
htmlgraph plan validate-yaml <plan-id>

```

---

## Step 4: Critique (Parallel Agents)

**Trigger:** Always run critique for plans with >= 3 slices. Skip for trivial plans.

Dispatch two critique agents in parallel. Each reads the plan YAML and produces structured findings.

```
Agent(description="Design critique", subagent_type="htmlgraph:haiku-coder",
      prompt="Read [plan.yaml]. Produce structured critique. [See prompt template below]")

Agent(description="Feasibility critique", subagent_type="htmlgraph:sonnet-coder",
      prompt="Read [plan.yaml]. Verify feasibility. [See prompt template below]")
```

### What Critique Agents Must Do

**For ALL plan types:**
- **Identify gaps** — what does the plan not address? What's missing?
- **Challenge assumptions** — what might be wrong? What is the plan taking for granted?
- **Suggest prior art** — what existing solutions, patterns, libraries, or examples are relevant?
- **Assess risks** — what could go wrong? What's the worst case? What's the blast radius?
- **Propose alternatives** — is there a simpler or different approach the plan missed?

**For plans involving existing code, also:**
- Verify claims against actual code (cite file:line numbers)
- Check if proposed changes conflict with existing patterns
- Identify functions/utilities that already exist and can be leveraged

**For greenfield or new ideas, also:**
- Question user/market assumptions — is there evidence of demand?
- Identify MVP scope vs full scope — what can be cut?
- Find real-world examples of similar systems — what can we learn from them?
- Flag "unknown unknowns" — what will we only discover by building?

### Critique Output Format

Each critique agent produces text in this format:

```
ASSUMPTIONS:
A1: [VERIFIED|PLAUSIBLE|UNVERIFIED|QUESTIONABLE|FALSIFIED] text | evidence

SLICE_ASSESSMENTS:
S1: [assessment with specific concerns]

RISKS:
R1: risk description | severity: Low/Medium/High | mitigation

GAPS:
- What the plan doesn't address

ALTERNATIVES:
- Different approaches worth considering

PRIOR_ART:
- Relevant existing solutions or patterns
```

### Writing Critique to YAML

The orchestrator parses both agents' output and writes the `critique` section:

```yaml
critique:
  reviewed_at: "YYYY-MM-DD"
  reviewers:
    - "Haiku (design review)"
    - "Sonnet (feasibility)"
  assumptions:
    - id: A1
      status: verified       # verified|plausible|unverified|questionable|falsified
      text: "The assumption being tested"
      evidence: "file:line or reasoning"
  critics:
    - title: "DESIGN CRITIC"
      sections:
        - heading: "Slice Assessment"
          items:
            - badge: "S1"
              kind: success   # success|warn|danger|info
              text: "Assessment text"
        - heading: "Gaps & Missing Considerations"
          items:
            - badge: "Gap"
              kind: warn
              text: "What the plan doesn't address"
        - heading: "Alternative Approaches"
          items:
            - badge: "Alt"
              kind: info
              text: "A different approach worth considering"
    - title: "FEASIBILITY CRITIC"
      sections:
        - heading: "What Exists to Leverage"
          items: [...]
        - heading: "What the Plan Gets Wrong"
          items: [...]
        - heading: "Prior Art & Examples"
          items: [...]
  risks:
    - risk: "Risk description"
      severity: High          # High|Medium|Low
      mitigation: "How to mitigate"
  synthesis: >
    Summary of key findings. What must change before the plan is approved?
    Numbered action items.
```

---

## Step 4b: Address Critique Findings (MANDATORY before review)

After critique agents return, the orchestrator MUST update the plan to address findings before launching review. The human should review the critique-informed version, not the stale original.

**For each FALSIFIED assumption:**
- Update the affected slice's `what` and `done_when` to address the falsification
- Add the corrected understanding to `design.constraints` if it affects the whole plan

**For each HIGH severity risk:**
- Add the mitigation as a `done_when` criterion on the affected slice
- If no slice owns the risk, add it to design constraints

**For each missing consideration identified by critics:**
- Add as a new question (if it's a design choice) or a new constraint (if it's a fact)

**Process:**
1. Parse both critique outputs for FALSIFIED/HIGH items
2. Read the current plan YAML
3. Modify the affected sections in memory
4. Write to temp file
5. Run: `htmlgraph plan rewrite-yaml <plan-id> --file /tmp/revised.yaml`
6. Git auto-commits the revision with message: `plan(<id>): address critique findings`
7. Proceed to Step 5 (review) with the updated plan

**Skip Step 4b only if:** all assumptions are verified/plausible AND no HIGH severity risks exist.

---

## Step 5: Open for Human Review (PAUSE HERE)

Launch the plan review notebook:

```bash
htmlgraph plan review <plan-id> --port 3001
```

The `htmlgraph plan review` command handles temp dir extraction, environment variables, and sandbox mode — do not launch marimo directly.

Tell the human:

```
Plan ready for review: http://localhost:3001

The notebook loads your plan from: .htmlgraph/plans/<plan-id>.yaml

Please:
1. Read the Design Discussion — Problem, Goals, Constraints
2. Review each Slice — approve or leave unchecked
3. Answer the Open Questions — defaults are recommended, override where you disagree
4. Read the AI Critique — note any risks or gaps
5. Check the Feedback Summary — progress bar shows completion
6. Click Finalize when all sections are approved

Your approvals persist automatically — you can close and reopen without losing progress.

I will wait until you finalize the plan before writing any code.
```

**STOP. Do not proceed until the human finalizes the plan.**

### Monitoring Review Progress

Poll SQLite to check if the human has finalized:

```bash
# Check approval progress
sqlite3 .htmlgraph/htmlgraph.db \
  "SELECT section, value FROM plan_feedback WHERE plan_id='<plan-id>' AND action='approve'"

# Check if all slices approved
sqlite3 .htmlgraph/htmlgraph.db \
  "SELECT COUNT(*) as approved FROM plan_feedback WHERE plan_id='<plan-id>' AND action='approve' AND value='True'"

# Check question answers
sqlite3 .htmlgraph/htmlgraph.db \
  "SELECT question_id, value FROM plan_feedback WHERE plan_id='<plan-id>' AND action='answer'"
```

---

## Step 6: Read Finalized Decisions

After the human clicks Finalize in marimo, the YAML status updates to `finalized`. Read the results:

```python
import yaml, sqlite3

# Read plan content
plan = yaml.safe_load(open(f".htmlgraph/plans/{plan_id}.yaml").read())
assert plan["meta"]["status"] == "finalized"

# Read approvals from SQLite
conn = sqlite3.connect(".htmlgraph/htmlgraph.db")
approvals = {row[0]: row[1] for row in conn.execute(
    "SELECT section, value FROM plan_feedback WHERE plan_id=? AND action='approve'",
    (plan_id,)
).fetchall()}

# Read question answers from SQLite
answers = {row[0]: row[1] for row in conn.execute(
    "SELECT question_id, value FROM plan_feedback WHERE plan_id=? AND action='answer'",
    (plan_id,)
).fetchall()}
```

Parse the results:
- If any slice has `value='False'`: summarize what was rejected. Ask the human — revise or proceed without?
- If revising: update the YAML, loop to Step 5.
- If proceeding: note excluded slices.

---

## Step 7: Create Work Items and Dispatch

For each approved slice, create a feature and wire dependencies:

```bash
# Create features for approved slices
htmlgraph feature create "<slice title>" --track <track-id> --description "<what + why>"

# Wire dependencies
htmlgraph link add <feat-blocked> <feat-blocker> --rel blocked_by
```

### Announce Finalized Plan

```
Plan finalized. Track: <track-id>

Approved slices (N of M):
  feat-XXXX  Slice 1 title       -> implement
  feat-XXXX  Slice 2 title       -> implement

Rejected slices:
  Slice 3 title                  -> excluded (not approved)

Design decisions:
  Q1: Migration caching: schema-version (recommended, accepted)
  Q2: Error handling: metric-counter (overrode recommendation: structured-log)
  Q3: SessionStart scope: git-only (recommended, accepted)
```

**Embed each question answer into every affected slice's task description** under "Accepted Design Decisions". If the human chose "metric-counter" for error handling, the dispatch description must explicitly say that.

Then hand off to `/htmlgraph:execute`.

---

## Agent Type Selection

| Agent | When to Use |
|-------|------------|
| `htmlgraph:haiku-coder` | Critique (design review), single-file tasks, <50 lines |
| `htmlgraph:sonnet-coder` | Critique (feasibility), multi-file tasks, moderate complexity |
| `htmlgraph:opus-coder` | Architecture decisions, complex algorithms, novel design |

Default to sonnet unless the task is trivially simple (haiku) or requires deep reasoning (opus).

---

## Key Rules

- **Plans are YAML files** — agents write structured data, not HTML strings
- **All slice fields are mandatory** — What, Why, Files, DoneWhen, Tests, Effort, Risk
- **All questions need context** — description + recommended option
- **Design has three subsections** — Problem, Goals, Constraints (not a single blob)
- **Critique challenges assumptions** — not just verifies code
- **Approvals persist to SQLite** — human can close and reopen without losing state
- **Finalize is explicit** — human clicks the button, not the agent
- **TDD is mandatory** — every dispatched task includes tests before implementation
- **Only approved slices** become features on dispatch
