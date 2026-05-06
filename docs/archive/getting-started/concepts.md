# Core Concepts

Wipnote coordinates AI-assisted development through a local-first stack: HTML files store work items, SQLite indexes them for fast queries, JSONL logs track all events, and a Phoenix LiveView dashboard surfaces observability. This guide explains the core concepts and how they work together.

## Architecture Layers

Wipnote stacks multiple representations of the same data for different purposes:

| Layer | Format | Purpose | Examples |
|-------|--------|---------|----------|
| **Artifact** | HTML files | Durable work items, human-readable | `.wipnote/features/feat-123.html` |
| **Query Index** | SQLite database | Fast lookups & analytics | Sessions, activities, relationships |
| **Event Log** | JSONL (append-only) | Immutable history for auditing | `.wipnote/events/session-id.jsonl` |
| **Observability** | Phoenix LiveView | Live dashboard, activity feeds | `http://localhost:8080` |

The artifact (HTML) is the source of truth for durable work items. SQLite and JSONL are derived, indexed representations that enable fast queries and event tracking.

## Key Components

### Features

**Features** are the atomic units of work in Wipnote. Each feature is an HTML file with:

- **Status**: `todo`, `in-progress`, `blocked`, `done`
- **Priority**: `low`, `medium`, `high`, `critical`
- **Steps**: Checklist of implementation tasks
- **Properties**: Custom metadata (`effort`, `completion`, etc.)
- **Edges**: Links to related features (blocks, blocked_by, related)

```python
from wipnote import SDK

sdk = SDK(agent="claude")

feature = sdk.features.create(
    title="User Authentication",
    status="todo",
    priority="high",
    steps=["Create endpoint", "Add middleware", "Write tests"]
)
```

**File location**: `.wipnote/features/feat-{hash8}.html`

### Tracks

**Tracks** are multi-feature projects that bundle related work with specs and plans. Each track is a directory containing:

- **index.html**: Track overview and dashboard
- **spec.html**: Requirements and success criteria
- **plan.html**: Phased implementation plan with time estimates

```python
track = sdk.tracks.builder() \
    .title("OAuth Integration") \
    .with_spec(
        overview="Add OAuth 2.0 support",
        requirements=[("Google OAuth", "must-have")]
    ) \
    .with_plan_phases([
        ("Phase 1", ["Configure OAuth (2h)", "Setup endpoints (1h)"])
    ]) \
    .create()
```

**File location**: `.wipnote/tracks/trk-{hash8}/`

### Sessions

**Sessions** track all activity during an agent's work session. Each session is an HTML file with:

- **Events**: Log of all tool calls and interactions
- **Features worked on**: Which features received attribution
- **Timestamps**: Start and end times
- **Agent**: Which agent did the work

Sessions are automatically created and managed by Wipnote hooks.

**File location**: `.wipnote/sessions/session-{id}.html`

### Events

**Events** are the append-only log of all activity. Each event is a JSON line with:

- **Timestamp**: When the event occurred
- **Event type**: `ToolUse`, `UserPrompt`, `SessionStart`, etc.
- **Session ID**: Which session generated the event
- **Feature ID**: Which feature receives attribution
- **Data**: Event-specific payload

**File location**: `.wipnote/events/{session-id}.jsonl`

## Graph Structure

### Nodes

Every HTML file in Wipnote is a graph node. Nodes have:

- **ID**: Unique, collision-resistant identifier (e.g., `feat-a1b2c3d4`)
- **Type**: `feature`, `track`, `session`, or custom
- **Properties**: Stored in `data-*` attributes
- **Content**: Human-readable description in HTML

#### Hash-Based IDs

Wipnote uses hash-based IDs for multi-agent collaboration:

| Type | Prefix | Example |
|------|--------|---------|
| Feature | `feat-` | `feat-a1b2c3d4` |
| Bug | `bug-` | `bug-12345678` |
| Track | `trk-` | `trk-abcdef12` |
| Session | `sess-` | `sess-7890abcd` |
| Spike | `spk-` | `spk-87654321` |
| Event | `evt-` | `evt-11223344` |

These IDs are collision-resistant, meaning multiple agents can create nodes simultaneously without conflicts. The 8-character hash is generated from SHA256 of (title + microsecond timestamp + random entropy), providing effectively zero collision probability even with thousands of concurrent agents.

Hierarchical sub-tasks are supported: `feat-a1b2c3d4.1.2`

**Learn more:**
- [ID Generation API](../api/ids.md) - Usage examples and API reference
- [Hash-Based IDs Design](../design/hash-based-ids.md) - Architecture and implementation details

Example node structure:

```html
<article id="feature-001"
         data-type="feature"
         data-status="in-progress"
         data-priority="high">
    <h1>User Authentication</h1>

    <nav data-graph-edges>
        <section data-edge-type="blocks">
            <h3>⚠️ Blocked By:</h3>
            <ul>
                <li><a href="feature-005.html">Database Schema</a></li>
            </ul>
        </section>
    </nav>
</article>
```

### Edges

Edges are created using standard HTML hyperlinks. The relationship type is specified using `data-relationship` attributes:

```html
<a href="feature-005.html"
   data-relationship="blocks">Database Schema</a>
```

Common relationship types:

- `blocks`: This feature blocks another
- `blocked_by`: This feature is blocked by another
- `related`: General relationship
- `implements`: Session implements a feature
- `part_of`: Feature is part of a track

### Queries

Query the graph using CSS selectors:

```python
# All high-priority features
high = sdk.features.query('[data-priority="high"]')

# Blocked features
blocked = sdk.features.query('[data-status="blocked"]')

# Features assigned to claude
claude_features = sdk.features.query('[data-agent-assigned="claude"]')
```

## Data Flow

```
┌─────────────────────────────────────────────────────────┐
│ 1. Agent creates/updates nodes via SDK                  │
└────────────────┬────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────┐
│ 2. Pydantic models validate and convert to HTML        │
└────────────────┬────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────┐
│ 3. HTML files written to .wipnote/ directory         │
└────────────────┬────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────┐
│ 4. Hooks log events to JSONL file                      │
└────────────────┬────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────┐
│ 5. SQLite index updated for fast queries               │
└────────────────┬────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────┐
│ 6. Browser/Dashboard displays graph visually           │
└─────────────────────────────────────────────────────────┘
```

## Why HTML for Work Items?

### Human Readable & Durable

Open any work item in a browser and see it beautifully rendered with CSS styling. HTML is a stable format that survives decades of technology change. No special tools required.

### Git Native

HTML is plain text. Git diffs show exactly what changed. Merge conflicts are readable. Work items live safely in source control.

### Minimal Infrastructure

No Docker, no JVM, no external database servers. Python dependencies include pydantic, justhtml, rich, jinja2, networkx, and others (14 runtime dependencies total). SQLite for indexing is local and append-only JSONL tracks all events.

### Offline First

Everything works offline. No server required for core functionality. Copy the `.wipnote/` directory anywhere and it works immediately.

### Standards-Based Artifact Layer

HTML is W3C standard. Every developer knows it. When work items are just HTML files, no proprietary format blocks future interoperability.

### Presentation Layer Included

Styling, layout, and interactivity are built-in using CSS and JavaScript. No separate UI framework needed for basic viewing. The Phoenix LiveView dashboard layers observability on top.

## SDK vs CLI vs Dashboard

### SDK (Python)

For programmatic access and agent integration:

```python
from wipnote import SDK
sdk = SDK(agent="claude")
feature = sdk.features.create("Task")
```

### CLI (Bash)

For command-line workflows:

```bash
wipnote feature create "Task"
wipnote feature start feature-001
wipnote serve
```

### Dashboard (Browser)

For visual exploration:

- Kanban board view
- Graph visualization
- Timeline view
- Session history

Open `index.html` in any browser or run `wipnote serve`.

## Next Steps

- [Features & Tracks Guide](../guide/features-tracks.md) - Detailed feature and track workflows
- [TrackBuilder Guide](../guide/track-builder.md) - Master the TrackBuilder API
- [Sessions Guide](../guide/sessions.md) - Understanding session tracking
- [API Reference](../api/index.md) - Complete SDK documentation
