# Quick Start

Get up and running with Wipnote in 5 minutes.

## Installation

```bash
pip install wipnote
```

## Initialize Your Project

```bash
# Create project directory
mkdir my-project && cd my-project

# Initialize Wipnote
wipnote init --install-hooks

# Start the dashboard
wipnote serve
```

This creates a `.wipnote/` directory with features, sessions, tracks, and events.

## Basic Usage

### Create a Feature

```python
from wipnote import SDK

# Initialize SDK (auto-discovers .wipnote directory)
sdk = SDK(agent="claude")

# Create a feature with fluent API
feature = sdk.features.create("User Authentication") \
    .set_priority("high") \
    .add_steps([
        "Create login endpoint",
        "Add JWT middleware",
        "Write tests"
    ]) \
    .save()

print(f"Created: {feature.id}")
# Output: Created: feat-a1b2c3d4
```

### Query Features

```python
# Get high-priority todos
high_priority = sdk.features.where(status="todo", priority="high")
for feat in high_priority:
    print(f"- {feat.title}")

# Get specific feature
feature = sdk.features.get("feat-a1b2c3d4")
```

### Update Features

```python
# Start working on a feature
with sdk.features.edit(feature.id) as f:
    f.status = "in-progress"
    f.steps[0].completed = True

# Mark complete
with sdk.features.edit(feature.id) as f:
    f.status = "done"
```

## Create a Track

For multi-feature projects, use TrackBuilder:

```python
track = sdk.tracks.builder() \
    .title("OAuth Integration") \
    .priority("high") \
    .with_spec(
        overview="Add OAuth 2.0 support",
        requirements=[
            ("Google OAuth", "must-have"),
            ("GitHub OAuth", "must-have")
        ]
    ) \
    .with_plan_phases([
        ("Phase 1: Setup", [
            "Configure OAuth providers (2h)",
            "Set up endpoints (1h)"
        ]),
        ("Phase 2: Implementation", [
            "Implement Google OAuth (4h)",
            "Implement GitHub OAuth (4h)"
        ])
    ]) \
    .create()

print(f"Created track: {track.track_id}")
# View at: .wipnote/tracks/{track.track_id}/index.html
```

## CLI Usage

```bash
# List features
wipnote feature list

# Create feature
wipnote feature create "Add OAuth support" --priority high

# Start feature
wipnote feature start feat-a1b2c3d4

# Mark complete
wipnote feature complete feat-a1b2c3d4

# View status
wipnote status
```

## View Your Graph

All graph nodes are standard HTML files:

```bash
# Open feature in browser
open .wipnote/features/feat-a1b2c3d4.html

# Open track
open .wipnote/tracks/trk-a1b2c3d4/index.html

# Dashboard
open index.html
```

## Next Steps

- **[API Reference](api/)** - Complete SDK documentation
- **[Examples](examples/)** - Real-world use cases including agent coordination
- **[Getting Started Guide](getting-started/)** - In-depth concepts and workflows

## Key Concepts

- **Features**: Individual tasks or work items
- **Tracks**: Multi-feature initiatives with specs and plans
- **Sessions**: Automatic tracking of development activity
- **SDK**: Fluent Python API for managing the graph
- **CLI**: Command-line interface for quick operations

## Important Notes

**Never edit HTML files directly!** Always use the SDK, API, or CLI:

```python
# ✅ CORRECT
with sdk.features.edit("feature-123") as f:
    f.status = "done"

# ❌ WRONG - bypasses validation
with open(".wipnote/features/feature-123.html", "w") as f:
    f.write("<html>...</html>")
```

Direct edits bypass Pydantic validation and break the SQLite index.
