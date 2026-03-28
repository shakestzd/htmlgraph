# /htmlgraph:recommend

Get smart recommendations on what to work on next

## Usage

```
/htmlgraph:recommend [--count N] [--check-bottlenecks]
```

## Parameters

- `--count` (optional) (default: 3): Number of recommendations to show
- `--check-bottlenecks` (optional) (default: True): Also show bottlenecks


## Examples

```bash
/htmlgraph:recommend
```
Get top 3 recommendations with bottleneck check

```bash
/htmlgraph:recommend --count 5
```
Get top 5 recommendations

```bash
/htmlgraph:recommend --no-check-bottlenecks
```
Recommendations only, skip bottleneck analysis



## Instructions for Claude

This command uses the Go binary to retrieve work recommendations and bottleneck analysis.

### Implementation:

1. Run `packages/go-plugin/hooks/bin/htmlgraph analytics summary` to get recommended work items
2. If `--check-bottlenecks` is true (default), also run `packages/go-plugin/hooks/bin/htmlgraph analytics summary`
3. Parse the CLI output (table format) and present it nicely in markdown
4. Include the command output as-is, formatting it for readability

### Sample CLI Commands:

```bash
# Get recommendations (shows top N items with scores)
packages/go-plugin/hooks/bin/htmlgraph analytics summary

# Get bottlenecks (shows blocking items with impact scores)
packages/go-plugin/hooks/bin/htmlgraph analytics summary

# Optional: Get work summary
packages/go-plugin/hooks/bin/htmlgraph analytics summary
```

### Output Format:

Present the CLI output in a user-friendly markdown format:

```markdown
## Work Recommendations

{CLI output from analytics recommend command}

{if check-bottlenecks}
### ⚠️ Bottlenecks Detected

{CLI output from analytics bottlenecks command}
{end if}

---
💡 Use `/htmlgraph:plan [id]` to start planning any of these items.
```

**PARALLEL ANALYSIS (MANDATORY when 2+ recommendations):**
Before presenting results, analyze whether recommendations can execute in parallel:
1. Check dependency graph between recommended items
2. Check file/module overlap (would they touch the same files?)
3. If independent → present a parallel execution plan as the DEFAULT action
4. Format: table showing Feature | Agent | Scope | Parallelizable?

**DELEGATION** (per-task model selection):
- Simple fixes (1-2 files) → `Agent(subagent_type="htmlgraph:haiku-coder", isolation="worktree")`
- Features (3-8 files) → `Agent(subagent_type="htmlgraph:sonnet-coder", isolation="worktree")`
- Architecture (10+ files) → `Agent(subagent_type="htmlgraph:opus-coder", isolation="worktree")`

**When parallelizable**, propose: "These N items have no dependencies or file overlap. Launch in parallel?"
