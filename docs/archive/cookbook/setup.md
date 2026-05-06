# Project Setup Recipes

## Initialize Project

**Problem**: Start a new Wipnote project from scratch.

**Solution**:

```bash
# Create project directory
mkdir my-project
cd my-project

# Initialize Wipnote with git hooks
wipnote init --install-hooks

# Verify setup
wipnote status
```

**Explanation**:
- `wipnote init` creates the `.wipnote/` directory structure
- `--install-hooks` installs git hooks for automatic activity tracking
- Creates `index.html` dashboard at project root
- Sets up JSONL event stream and SQLite index

**What Gets Created**:
```
.wipnote/
├── features/       # Feature HTML files
├── sessions/       # Session HTML files
├── tracks/         # Track directories
├── events/         # JSONL event logs
├── index.sqlite    # Analytics index
└── hooks/          # Git hook scripts
```

---

## Install Git Hooks

**Problem**: Enable automatic tracking of git activity.

**Solution**:

```bash
# Option 1: During init
wipnote init --install-hooks

# Option 2: After init
wipnote setup install-hooks

# Verify hooks are installed
ls -la .git/hooks/
```

**Explanation**:
- Hooks track commits, checkouts, and merges
- Links git activity to active features
- Preserves context across sessions
- Safe: hooks are versioned with your code

**Hook Events Tracked**:
- `post-commit`: Links commits to features
- `post-checkout`: Detects branch switches
- `post-merge`: Tracks merge activity

---

## Configure Agent

**Problem**: Set your agent identity for attribution.

**Solution**:

```python
from wipnote import SDK

# Option 1: Set agent in SDK initialization
sdk = SDK(agent="claude-code")

# Option 2: Use environment variable
import os
os.environ['HTMLGRAPH_AGENT'] = 'my-agent-name'
sdk = SDK()  # Reads from environment

# Option 3: Set in shell
export HTMLGRAPH_AGENT=my-agent-name
wipnote status  # Uses env variable
```

**Explanation**:
- Agent identity is used for attribution in sessions and features
- Appears in activity logs and step completion tracking
- Enables multi-agent coordination
- Use consistent names across sessions

**Best Practices**:
- Use descriptive names: "claude-code", "cursor", "human-dev"
- Set HTMLGRAPH_AGENT in your shell profile for consistency
- Include agent name in git config for fuller attribution

---

## Start Development Server

**Problem**: View the dashboard while developing.

**Solution**:

```bash
# Start server (default port 8080)
wipnote serve

# Custom port
wipnote serve --port 3000

# Auto-reload on file changes
wipnote serve --watch
```

**Explanation**:
- Serves `index.html` dashboard with live data
- Visualizes features, sessions, and dependencies
- Updates automatically as you work
- No build step required

**Access**:
- Dashboard: `http://localhost:4000` (Phoenix LiveView)
- API server: `http://localhost:8080` (FastAPI/WebSocket backend)

---

## Bootstrap Example Data

**Problem**: Start with sample data to understand Wipnote.

**Solution**:

```python
from wipnote import SDK

sdk = SDK(agent="learning")

# Create example features
examples = [
    {
        "title": "User Authentication",
        "priority": "high",
        "steps": ["Create login", "Add JWT", "Write tests"]
    },
    {
        "title": "Database Schema",
        "priority": "high",
        "steps": ["Design tables", "Create migrations", "Seed data"]
    },
    {
        "title": "API Endpoints",
        "priority": "medium",
        "steps": ["Define routes", "Add validation", "Document API"]
    }
]

for ex in examples:
    feature = sdk.features.create(ex["title"]) \
        .set_priority(ex["priority"]) \
        .add_steps(ex["steps"]) \
        .save()
    print(f"Created: {feature.id}")

# Add dependencies
auth = sdk.features.where(title="User Authentication")[0]
db = sdk.features.where(title="Database Schema")[0]
api = sdk.features.where(title="API Endpoints")[0]

sdk.features.add_dependency(auth.id, db.id, "blocks")
sdk.features.add_dependency(api.id, auth.id, "blocks")

print("\nBootstrap complete! Run 'wipnote serve' to see the dashboard.")
```

**What You Get**:
- 3 sample features with steps
- Dependency relationships
- Ready to explore queries and workflows

---

## Cleanup and Reset

**Problem**: Start fresh or remove Wipnote from a project.

**Solution**:

```bash
# Backup first!
cp -r .wipnote .wipnote.backup

# Remove all Wipnote data
rm -rf .wipnote

# Uninstall git hooks
rm .git/hooks/post-commit
rm .git/hooks/post-checkout
rm .git/hooks/post-merge

# Reinitialize if needed
wipnote init --install-hooks
```

**Caution**:
- This deletes ALL tracking data
- Features, sessions, and events are permanently lost
- Only do this if you're sure
- Consider archiving instead of deleting
