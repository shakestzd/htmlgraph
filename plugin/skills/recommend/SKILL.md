# /htmlgraph:recommend

Get smart recommendations on what to work on next, including project health, bottleneck analysis, and parallel opportunities.

## Usage

```
/htmlgraph:recommend [--top N]
```

## Parameters

- `--top` (optional, default: 5): Number of recommendations to show

## Examples

```bash
/htmlgraph:recommend
```
Get top 5 recommendations with full analysis

```bash
/htmlgraph:recommend --top 10
```
Get top 10 recommendations

## What This Skill Does

1. **Runs one command** — `htmlgraph recommend [--top N]`
2. **Presents the output** — Project health, WIP status, bottlenecks, recommendations, parallel opportunities
3. **Analyzes parallelization** — Checks if recommended items are independent
4. **Proposes next action** — Either parallel execution plan or individual task delegation

## Instructions for Claude

### Step 1: Run the recommendation command

```bash
htmlgraph recommend --top N
```

Where N is:
- User's `--top N` value if provided
- Default: 5 if not specified

### Step 2: Present the output

Display the CLI output with light markdown formatting for readability:

```markdown
## Project Health & Recommendations

{full CLI output from htmlgraph recommend}
```

### Step 3: Parallel Analysis (when 2+ recommendations present)

Before proposing next steps, analyze whether the **top recommendations** can execute in parallel:

1. **Check dependencies** — Do any of the recommended items block each other?
2. **Check file overlap** — Do they modify the same files or modules?
3. **Decision** — If independent → propose parallel execution; if dependent → sequential order

**Present as:** Simple summary
```
Parallelizable: feat-a506fe1b ✓, feat-e734b5e6 ✓
Dependent: feat-c08cdb8e (blocks others)
```

### Step 4: Propose Next Action

**If parallel opportunities exist:**
```
These top 3 items have no dependencies or file overlap.
Launch in parallel using /htmlgraph:execute?
```

**If sequential required:**
```
feat-c08cdb8e should complete first (blocks others).
Start with: /htmlgraph:plan feat-c08cdb8e
```

## Per-Recommendation Delegation

When proposing individual task execution:

- **Simple fixes** (1-2 files) → `/htmlgraph:htmlgraph-coder feat-id`
- **Features** (3-8 files) → `/htmlgraph:htmlgraph-coder feat-id` (uses sonnet internally)
- **Architecture** (10+ files) → `/htmlgraph:htmlgraph-coder feat-id` (uses opus internally)

Or use `/htmlgraph:plan [id]` to create a full execution plan with dependencies.
