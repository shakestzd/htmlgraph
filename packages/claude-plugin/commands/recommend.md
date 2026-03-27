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

This command uses the CLI's analytics commands.

### Implementation:

1. **Get recommendations:**
   ```bash
   htmlgraph analytics recommend
   ```

2. **Optionally check bottlenecks:**
   ```bash
   htmlgraph analytics bottlenecks
   ```

3. **Present results** using the output template below.

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
