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
   The command will report whether `.htmlgraph/` was created or already exists.

2. **Present next steps** using the output template below.

3. **Guide the user:**
   - How to plan work: `/htmlgraph:plan "title"`
   - How to start session: `/htmlgraph:start`
   - How to view dashboard: `/htmlgraph:serve`

4. **Highlight key points:**
   - All subsequent work will be tracked automatically
   - Use CLI/slash commands for all operations
   - Access dashboard to view progress visually

### Output Format:

## HtmlGraph Initialized

Created `.htmlgraph/` directory with:
- `features/` - Feature work items
- `sessions/` - Session activity logs
- `tracks/` - Multi-feature tracks
- `spikes/` - Research and investigation
- `bugs/` - Bug tracking
- `htmlgraph.db` - SQLite read index for queries and dashboard
- `refs.json` - Project metadata references
- `styles.css` - Default stylesheet for HtmlGraph HTML nodes

Note:
- Additional paths such as plans, events, and launch/session markers may appear later as other HtmlGraph commands and hooks run.
- Current `htmlgraph init` does not create legacy analytics directories like `insights/`, `metrics/`, or `cigs/`.

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
