# Quick Start

Get up and running with Wipnote in 5 minutes.

## Initialize a Project

```bash
# Create a new directory for your graph
mkdir my-project
cd my-project

# Initialize Wipnote
wipnote init
```

This creates a `.wipnote/` directory with the following structure:

```
.wipnote/
├── features/       # Feature nodes
├── sessions/       # Session activity logs
├── tracks/         # Multi-feature tracks
├── events/         # Event log (JSONL)
└── index.db        # SQLite index (auto-generated)
```

## Using the SDK

### Basic Feature Creation

```python
from wipnote import SDK

# Initialize SDK (auto-discovers .wipnote directory)
sdk = SDK(agent="claude")

# Create a feature
feature = sdk.features.create(
    title="User Authentication",
    priority="high",
    steps=[
        "Create login endpoint",
        "Add JWT middleware",
        "Write tests"
    ]
)

print(f"Created feature: {feature.id}")
# Output: Created feature: feat-a1b2c3d4
```

### Query Features

```python
# Get all high-priority features
high_priority = sdk.features.where(priority="high")

# Get features by status
in_progress = sdk.features.where(status="in-progress")

# Get a specific feature
feature = sdk.features.get("feat-a1b2c3d4")
```

### Update Feature Status

```python
# Start working on a feature
feature.status = "in-progress"
feature.save()

# Complete a step
feature.steps[0].completed = True
feature.save()

# Mark feature as complete
feature.status = "done"
feature.save()
```

## Using the CLI

### Feature Management

```bash
# List all features
wipnote feature list

# Create a new feature
wipnote feature create "Add OAuth support" --priority high

# Start working on a feature
wipnote feature start feat-a1b2c3d4

# Mark a feature as complete
wipnote feature complete feat-a1b2c3d4
```

### Session Management

```bash
# View session status
wipnote status

# List all sessions
wipnote session list

# View session details
wipnote session show session-abc-123
```

### Dashboard

Launch the interactive dashboard:

```bash
wipnote serve
```

Then open [http://localhost:8080](http://localhost:8080) in your browser.

## Creating a Track

For multi-feature projects, create a track with spec and plan:

```python
from wipnote import SDK

sdk = SDK(agent="claude")

# Use TrackBuilder for complex tracks
track = sdk.tracks.builder() \
    .title("OAuth Integration") \
    .priority("high") \
    .with_spec(
        overview="Add OAuth 2.0 support for Google and GitHub",
        requirements=[
            ("Support Google OAuth", "must-have"),
            ("Support GitHub OAuth", "must-have"),
            ("JWT token management", "must-have")
        ],
        success_criteria=[
            "Users can sign in with Google",
            "Users can sign in with GitHub",
            "Tokens refresh automatically"
        ]
    ) \
    .with_plan_phases([
        ("Phase 1: OAuth Setup", [
            "Configure OAuth providers (2h)",
            "Set up redirect endpoints (1h)"
        ]),
        ("Phase 2: Implementation", [
            "Implement Google OAuth (4h)",
            "Implement GitHub OAuth (4h)",
            "Add JWT middleware (3h)"
        ]),
        ("Phase 3: Testing", [
            "Write integration tests (3h)",
            "Test with real providers (2h)"
        ])
    ]) \
    .create()

print(f"Created track: {track.track_id}")
# View at: .wipnote/tracks/{track.track_id}/index.html
```

## View Your Graph

All graph nodes are HTML files that you can open in any browser:

```bash
# Open a feature in your browser
open .wipnote/features/feat-a1b2c3d4.html

# Open a track
open .wipnote/tracks/trk-a1b2c3d4/index.html

# Open the dashboard
open index.html
```

## Next Steps

- [Core Concepts](concepts.md) - Understand features, tracks, and sessions
- [User Guide](../guide/index.md) - In-depth guides for all features
- [API Reference](../api/index.md) - Complete SDK documentation
- [Examples](../examples/index.md) - Real-world use cases
