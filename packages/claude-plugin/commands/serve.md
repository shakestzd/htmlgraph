<!-- Efficiency: SDK calls: 1, Bash calls: 0, Context: ~3% -->

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

1. **Parse port from arguments** (default: 8080)

2. **Start server:**
   ```bash
   htmlgraph serve --port <port>
   ```
   If port is busy, try the next available port.

3. **Present dashboard information** using the output template below

4. **Explain dashboard features:**
   - Real-time feature progress tracking
   - Kanban board for task organization
   - Session activity logs
   - Dependency graph visualization

5. **Provide stop instructions:**
   - Press Ctrl+C to stop the server

### Output Format:

## Dashboard Running

**URL:** {url}{port_note}

The dashboard shows:
- Feature progress and kanban board
- Session history with activity logs
- Graph visualization of dependencies

To stop: press Ctrl+C.
