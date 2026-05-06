# SDK

The SDK is the main interface for interacting with Wipnote.

## Overview

The SDK provides a high-level API for:

- Creating and managing features
- Creating and managing tracks
- Querying the graph
- Session management
- Activity tracking

## Initialization

```python
from wipnote import SDK

# Basic initialization
sdk = SDK(agent="claude")

# Custom graph directory
sdk = SDK(agent="claude", graph_dir="/path/to/.wipnote")

# Disable auto-session management
sdk = SDK(agent="claude", auto_session=False)
```

## Features API

### Creating Features

```python
# Basic feature
feature = sdk.features.create(
    title="Add login page",
    priority="high"
)

# Feature with steps
feature = sdk.features.create(
    title="Add login page",
    priority="high",
    steps=["Create component", "Add routing", "Write tests"]
)

# Feature with custom properties
feature = sdk.features.create(
    title="Add login page",
    priority="high",
    properties={"effort": 4, "assignee": "claude"}
)
```

### Querying Features

```python
# Get all features
all_features = sdk.features.all()

# Filter by status
in_progress = sdk.features.where(status="in-progress")

# Filter by priority
high_priority = sdk.features.where(priority="high")

# Multiple filters
blocked_high = sdk.features.where(status="blocked", priority="high")

# Get specific feature
feature = sdk.features.get("feature-20241216-103045")
```

### Updating Features

```python
# Get feature
feature = sdk.features.get("feature-20241216-103045")

# Update and save
feature.status = "in-progress"
feature.steps[0].completed = True
feature.save()
```

### Deleting Features

```python
# Delete a single feature
deleted = sdk.features.delete("feature-20241216-103045")
# Returns: True if deleted, False if not found

# Delete multiple features (batch operation)
count = sdk.features.batch_delete([
    "feature-001",
    "feature-002",
    "feature-003"
])
# Returns: Number of features successfully deleted

# Delete works across all collections
sdk.bugs.delete("bug-001")
sdk.chores.delete("chore-001")
sdk.spikes.delete("spike-001")

# Batch delete with edge cleanup
# Edges involving deleted nodes are automatically cleaned up
sdk.features.batch_delete(["duplicate-1", "duplicate-2"])
```

**Note:** Delete operations automatically:
- Remove the HTML file from disk
- Clean up all edges involving the node (incoming and outgoing)
- Update the in-memory edge index

## Tracks API

### Creating Tracks

```python
# Simple track
track = sdk.tracks.create(title="Project", priority="high")

# Track with TrackBuilder
track = sdk.tracks.builder() \
    .title("OAuth Integration") \
    .priority("high") \
    .with_spec(
        overview="Add OAuth 2.0 support",
        requirements=[("Google OAuth", "must-have")]
    ) \
    .with_plan_phases([
        ("Phase 1", ["Setup OAuth (2h)", "Configure (1h)"])
    ]) \
    .create()
```

### Querying Tracks

```python
# Get all tracks
all_tracks = sdk.tracks.all()

# Get specific track
track = sdk.tracks.get("track-20241216-120000")

# Get features for a track
features = sdk.features.where(track_id=track.track_id)
```

## Sessions API

### Session Management

```python
# Get all sessions
sessions = sdk.sessions.all()

# Get current session
current = sdk.sessions.current()

# Get specific session
session = sdk.sessions.get("session-abc-123")
```

### Activity Tracking

```python
# Track custom activity
sdk.track_activity(
    feature_id="feature-001",
    activity="Chose PostgreSQL over MongoDB for better transactions"
)
```

## Status API

```python
# Get current status
status = sdk.status()

print(f"Current session: {status.current_session}")
print(f"Active features: {status.active_features}")
print(f"Total features: {status.total_features}")
print(f"Progress: {status.progress}%")
```

## Graph Operations

### Relationships

```python
# Add edge between features
sdk.features.add_edge(
    from_id="feature-001",
    to_id="feature-002",
    relationship="blocks"
)

# Get dependencies
deps = sdk.features.get_dependencies("feature-001")

# Get blocking features
blocking = sdk.features.get_blocking("feature-001")
```

### Graph Queries

```python
# Query with CSS selectors
blocked = sdk.features.query('[data-status="blocked"]')
high_priority = sdk.features.query('[data-priority="high"]')

# Graph traversal
path = sdk.graph.shortest_path("feature-001", "feature-045")
transitive_deps = sdk.graph.transitive_deps("feature-001")
bottlenecks = sdk.graph.find_bottlenecks()
```

## SDK Architecture & Module Organization

### Module Structure

The SDK is organized into modular components for better maintainability and separation of concerns:

```
wipnote/
├── builders/          # Fluent builders for node creation
│   ├── base.py       # BaseBuilder with common methods
│   ├── feature.py    # FeatureBuilder
│   ├── spike.py      # SpikeBuilder
│   └── track.py      # TrackBuilder
├── collections/       # Collection interfaces for querying/updating
│   ├── base.py       # BaseCollection with CRUD operations
│   ├── feature.py    # FeatureCollection with builder support
│   └── spike.py      # SpikeCollection with builder support
├── analytics/         # Analytics and strategic planning
│   ├── work_type.py  # Work type analytics (Analytics class)
│   ├── dependency.py # Dependency analytics (DependencyAnalytics class)
│   └── cli.py        # CLI analytics helpers
├── sdk.py            # Main SDK class (single entry point)
└── session_manager.py # Session and activity tracking
```

### Import Paths

**Recommended imports:**

```python
from wipnote import SDK, Analytics, DependencyAnalytics
from wipnote.builders import FeatureBuilder, SpikeBuilder, TrackBuilder
from wipnote.collections import BaseCollection, FeatureCollection
from wipnote.models import SpikeType, MaintenanceType, WorkType

# Direct module imports (advanced usage)
from wipnote.analytics import Analytics, DependencyAnalytics
from wipnote.analytics.work_type import Analytics
from wipnote.analytics.dependency import DependencyAnalytics
```

### SDK Components

**1. Collections** - Query and manage nodes

```python
sdk.features   # FeatureCollection - features with builder support
sdk.bugs       # BaseCollection - bug reports
sdk.chores     # BaseCollection - maintenance tasks
sdk.spikes     # SpikeCollection - investigation spikes
sdk.epics      # BaseCollection - large bodies of work
sdk.tracks     # TrackCollection - multi-feature initiatives
```

**2. Builders** - Fluent interfaces for node creation

```python
# Features
feature = sdk.features.create("Add auth") \
    .set_priority("high") \
    .add_steps(["Step 1", "Step 2"]) \
    .save()

# Spikes
spike = sdk.spikes.create("Research OAuth") \
    .set_spike_type(SpikeType.TECHNICAL) \
    .set_timebox_hours(4) \
    .save()

# Tracks
track = sdk.tracks.builder() \
    .title("User Management") \
    .with_spec(overview="...") \
    .with_plan_phases([...]) \
    .create()
```

**3. Analytics** - Strategic planning and insights

```python
# Work type analytics
distribution = sdk.analytics.work_type_distribution(session_id="...")
ratio = sdk.analytics.spike_to_feature_ratio()

# Dependency analytics
bottlenecks = sdk.dep_analytics.find_bottlenecks(top_n=5)
recommendations = sdk.dep_analytics.recommend_next_work(agent_count=3)
parallel_work = sdk.dep_analytics.get_parallel_work(max_agents=5)
```

**4. Session Management** - Activity tracking

```python
# Sessions are managed automatically via SDK
# Access SessionManager through SDK if needed
session = sdk.session_manager.get_active_session(agent="claude")
sdk.start_session(title="Feature implementation")
sdk.end_session(session_id="...", handoff_notes="...")
```

### Design Philosophy

**Separation of Concerns:**
- **Builders** - Node creation (immutable, fluent API)
- **Collections** - Node querying and updates (CRUD operations)
- **Analytics** - Strategic insights (read-only, computational)
- **SessionManager** - Activity tracking (event logging)
- **SDK** - Single entry point (coordinates all components)

**Benefits:**
- Clear responsibilities for each module
- Easy to test and maintain
- Consistent API across all node types
- Extensible for new node types

## Complete API Reference

For detailed API documentation with type signatures and docstrings, see the Python source code in `src/python/wipnote/sdk.py`.
