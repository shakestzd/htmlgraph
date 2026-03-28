<!-- Efficiency: SDK calls: 2, Bash calls: 0, Context: ~5% -->

# /htmlgraph:feature-complete

Mark a feature as complete

## Usage

```
/htmlgraph:feature-complete [feature-id]
```

## Parameters

- `feature-id` (optional): The feature ID to complete. If not provided, completes the current active feature.


## Examples

```bash
/htmlgraph:feature-complete feature-001
```
Complete a specific feature

```bash
/htmlgraph:feature-complete
```
Complete the current active feature



## Instructions for Claude

This command uses the CLI's `feature complete` command.

### Implementation:

**DO THIS:**

1. **Get current feature if not specified:**
   ```bash
   htmlgraph find features --status in-progress
   ```
   If no feature-id given, use the first in-progress feature.

2. **Complete the feature:**
   ```bash
   htmlgraph feature complete <feature-id>
   ```

3. **Get project status:**
   ```bash
   htmlgraph status
   htmlgraph find features --status todo
   ```

4. **Present summary** using the output template below

5. **Recommend next steps:**
   - If pending features exist → Suggest starting the next feature
   - If all features done → Congratulate on completion
   - Offer to run `/htmlgraph:plan` for new work

### Output Format:

## Feature Completed

**ID:** {feature_id}
**Title:** {title}
**Status:** done

### Progress Update
**Completed:** {status['done_count']}/{status['total_nodes']} ({percentage}%)
**Active:** {status['in_progress_count']} features

### What's Next?

Get recommendations for next work:
```bash
htmlgraph analytics summary
```

If no pending work, plan new features with `/htmlgraph:plan`.

**DELEGATION**: Delegate implementation based on complexity:
- Simple fixes (1-2 files) → `Task(subagent_type="htmlgraph:haiku-coder")`
- Features (3-8 files) → `Task(subagent_type="htmlgraph:sonnet-coder")`
- Architecture (10+ files) → `Task(subagent_type="htmlgraph:opus-coder")`
