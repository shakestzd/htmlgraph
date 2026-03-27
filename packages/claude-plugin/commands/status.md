<!-- Efficiency: SDK calls: 1, Bash calls: 0, Context: ~5% -->

# /htmlgraph:status

Check project status and active features

## Usage

```
/htmlgraph:status
```

## Parameters



## Examples

```bash
/htmlgraph:status
```
Show project progress and current feature



## Instructions for Claude

This command uses the CLI's `status` and `find` commands.

### Implementation:

**DO THIS:**

1. **Get comprehensive status:**
   ```bash
   htmlgraph status
   htmlgraph find features --status in-progress
   ```

2. **Extract key metrics** from CLI output:
   - Total features, completed count, in-progress count
   - Active features with titles and progress

3. **Present a summary** using the output template below

4. **Recommend next steps** based on status:
   - If no active features → Suggest `/htmlgraph:recommend`
   - If active features exist → Show their progress
   - If features done → Acknowledge progress
   - Suggest `/htmlgraph:plan` for new work

### Output Format:

## Project Status

**Progress:** {status['done_count']}/{status['total_nodes']} ({percentage}%)
**Active:** {status['in_progress_count']} features in progress

### Current Feature(s)
{active_features with titles and step progress}

### Quick Actions
- Use `/htmlgraph:plan` to start planning new work
- Use `/htmlgraph:recommend` to get recommendations
- Run `htmlgraph serve` to open dashboard
