<!-- Efficiency: SDK calls: 1, Bash calls: 0, Context: ~3% -->

# /htmlgraph:init

Initialize HtmlGraph in a project

## Usage

```
/htmlgraph:init
```

## Parameters



## Examples

```bash
/htmlgraph:init
```
Set up HtmlGraph directory structure in project



## Instructions for Claude

### Implementation:

**DO THIS:**

1. **Initialize project:**
   ```bash
   htmlgraph init
   ```

2. **Present next steps** using the output template below

3. **Guide the user:**
   - How to plan work: `/htmlgraph:plan "title"`
   - How to start session: `/htmlgraph:start`
   - How to view dashboard: `/htmlgraph:serve`

4. **Highlight key points:**
   - All subsequent work will be tracked automatically
   - Use slash commands and CLI for all operations
   - Access dashboard to view progress visually

### Output Format:

## HtmlGraph Initialized

Created `.htmlgraph/` directory with:
- `features/` - Feature work items
- `sessions/` - Session activity logs
- `tracks/` - Multi-feature tracks
- `spikes/` - Research and investigation
- `bugs/` - Bug tracking
- `patterns/` - Workflow patterns
- `insights/` - Session insights
- `metrics/` - Aggregated metrics
- `todos/` - Persistent tasks
- `task-delegations/` - Subagent work tracking

### Next Steps
1. Plan new work: `/htmlgraph:plan "Feature title"`
2. Start session: `/htmlgraph:start`
3. View dashboard: `/htmlgraph:serve`

### Quick Start
```bash
# Start planning
/htmlgraph:plan "Add user authentication"

# Begin work
/htmlgraph:start

# View progress
/htmlgraph:serve
# Open http://localhost:8080
```
