# /htmlgraph:serve

Start the dashboard server

## Usage

```
/htmlgraph:serve [port]
```

## Parameters

- `port` (optional) (default: 8080): Port number for the dashboard server


## Examples

```bash
/htmlgraph:serve
```
Start dashboard on default port 8080

```bash
/htmlgraph:serve 3000
```
Start dashboard on port 3000



## Instructions for Claude

### Implementation:

**DO THIS:**

1. **Start the dashboard server:**
   ```bash
   htmlgraph serve --port {port}
   ```
   Default port is 8080 if not specified.

2. **Present dashboard information** using the output template below.

3. **Explain dashboard features:**
   - Real-time feature progress tracking
   - Kanban board for task organization
   - Session activity logs
   - Dependency graph visualization

4. **Provide stop instructions:**
   - Server runs in background until stopped
   - Press Ctrl+C to stop if running in foreground

### Output Format:

## Dashboard Running

**URL:** http://localhost:{port}

The dashboard shows:
- Feature progress and kanban board
- Session history with activity logs
- Graph visualization of dependencies

To stop: press Ctrl+C or kill the background process.
