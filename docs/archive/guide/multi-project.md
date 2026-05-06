# Multi-Project Dashboard

Run a single Phoenix LiveView dashboard serving multiple Wipnote projects simultaneously.

## Quick Start

```bash
# Set workspace root (directory containing your projects)
export HTMLGRAPH_WORKSPACE=/path/to/workspace

# Start with Docker Compose
docker compose up -d

# Open http://localhost:4000/projects
```

## How It Works

The dashboard scans your workspace root for directories containing `.wipnote/wipnote.db`. Each discovered project appears in the project selector dropdown.

```
/workspace/
├── project-a/
│   └── .wipnote/
│       └── wipnote.db    ← discovered
├── project-b/
│   └── .wipnote/
│       └── wipnote.db    ← discovered
└── project-c/                     ← no .wipnote/, ignored
```

## Configuration

### Environment Variables

| Env Var | Default | Description |
|---------|---------|-------------|
| `HTMLGRAPH_WORKSPACE` | Parent directory of current project | Root directory to scan for `.wipnote/` |
| `HTMLGRAPH_DB_PATH` | `{workspace}/.wipnote/wipnote.db` | Default DB path when no project selected |
| `PHX_HOST` | `localhost` | Phoenix host binding |
| `PHX_PORT` | `4000` | Phoenix port |

### Docker Compose Setup

Create `docker-compose.yml` in your workspace root:

```yaml
version: '3.8'

services:
  dashboard:
    image: ghcr.io/shakestzd/wipnote-dashboard:latest
    ports:
      - "4000:4000"
    environment:
      HTMLGRAPH_WORKSPACE: /workspace
      PHX_HOST: 0.0.0.0
      PHX_PORT: 4000
    volumes:
      - ./:/workspace:ro
    restart: unless-stopped
```

Then run:

```bash
docker compose up -d
```

View logs:

```bash
docker compose logs -f dashboard
```

## Without Docker

Run the dashboard from any Wipnote project with workspace support:

```bash
# From any project directory
cd /path/to/workspace/my-project

# Serve with workspace scanning
wipnote serve --workspace /path/to/workspace
```

The dashboard will:
1. Scan `/path/to/workspace` for all `.wipnote/wipnote.db` files
2. Build a project list
3. Serve on http://localhost:4000

## Project Switching

### Dropdown Selector

Use the project selector dropdown in the navigation bar to switch between projects:

1. Click the project dropdown (top-left of nav)
2. Select a project from the list
3. All views update to show data for the selected project

### Keyboard Shortcuts

- **Cmd/Ctrl + P** - Project selector (when available)
- **Cmd/Ctrl + K** - Command palette

### Deep Linking

Use project selector in URL:

```
http://localhost:4000/?project=project-a
http://localhost:4000/kanban?project=project-b
http://localhost:4000/graph?project=project-c
```

## Views Scoped to Project

All dashboard views are automatically scoped to the selected project:

- **Activity Feed** - Shows events only for this project's sessions
- **Kanban Board** - Displays work items (features, bugs, tasks)
- **Dependency Graph** - Shows work item relationships
- **Costs** - Analytics for this project's activity
- **Analytics** - Metrics and performance data

## Adding New Projects

1. Initialize Wipnote in your project:
   ```bash
   cd /path/to/new-project
   wipnote init
   ```

2. The dashboard automatically discovers it on next scan

3. No restart needed - just refresh the dashboard

## Database Location Discovery

The dashboard uses this search order:

1. **Explicit path**: `HTMLGRAPH_DB_PATH` environment variable
2. **Project selection**: Selected project's `.wipnote/wipnote.db`
3. **Workspace scan**: Find all `.wipnote/wipnote.db` in `HTMLGRAPH_WORKSPACE`
4. **Default**: `~/.wipnote/wipnote.db` (single-project fallback)

## Troubleshooting

### Projects Not Appearing

Check that each project has been initialized:

```bash
ls -la /workspace/project-*/. wipnote/wipnote.db

# Or search for all databases
find /workspace -name "wipnote.db" -type f
```

Initialize missing projects:

```bash
cd /workspace/project-name
wipnote init
```

### Dashboard Not Connecting

Verify workspace path is correct:

```bash
docker compose logs dashboard
# Look for: "Scanning workspace: /path/to/workspace"
```

Check mounted volumes:

```bash
docker exec dashboard ls -la /workspace/
```

### Slow Performance with Many Projects

The dashboard scans all projects on startup. For workspaces with 50+ projects:

1. Use explicit `HTMLGRAPH_DB_PATH` to select a single project
2. Or specify a subset in `HTMLGRAPH_WORKSPACE`

## Performance Tips

- **Use SSD storage** for `.wipnote/` databases
- **Limit projects** to 20-30 per dashboard (use separate dashboards for larger teams)
- **Keep databases small** - export old sessions periodically
- **Mount workspace as read-only** - reduces I/O overhead

## Multi-Machine Setup

For distributed team access:

```bash
# Server machine
export HTMLGRAPH_WORKSPACE=/mnt/shared/projects
docker compose up -d

# Access from any client
# http://server-ip:4000/projects
```

## Exporting Data

Export all project data:

```bash
for project in /workspace/*/; do
  if [ -f "$project/.wipnote/wipnote.db" ]; then
    name=$(basename "$project")
    wipnote export -o "exports/$name.jsonl" --db "$project/.wipnote/wipnote.db"
  fi
done
```

## Next Steps

- [Dashboard User Guide](./dashboard.md) - Full dashboard features
- [Project Workspace Guide](./workspace.md) - Managing workspaces
- [Analytics Guide](./analytics.md) - Multi-project metrics
