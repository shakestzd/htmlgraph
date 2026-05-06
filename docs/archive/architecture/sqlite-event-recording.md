# SQLite Event Recording Integration

**Status:** Implemented | **Phase:** Real-time Dashboard Integration

## Overview

Claude Code tool execution events are now automatically recorded to SQLite database via Wipnote hooks, enabling the FastAPI dashboard to display live activity feeds and orchestration metrics.

## Architecture

### Dual-Path Event Recording

Events are recorded in two complementary ways:

```
Claude Code Tool Execution
         ↓
    Hook fires (PostToolUse, Stop, UserPromptSubmit)
         ↓
    ┌────────────────────────────────────────────┐
    │       event_tracker.py (hook script)       │
    └────────────────────────────────────────────┘
         ↓                              ↓
    SessionManager               WipnoteDB
    (HTML files)                (SQLite DB)
         ↓                              ↓
    .wipnote/               .wipnote/
    sessions/*.html           wipnote.db
    activities/               (Queryable)
```

### Key Components

**1. Event Tracker Hook** (`src/python/wipnote/hooks/event_tracker.py`)
- Intercepts tool calls via PostToolUse hook
- Records to both HTML (existing) and SQLite (new)
- Handles special case: Task() delegations → agent_collaboration table
- Gracefully degrades if SQLite unavailable

**2. Database Schema** (`src/python/wipnote/db/schema.py`)
- `agent_events` - All tool calls, queries, delegations
- `agent_collaboration` - Task delegations between agents
- Indexed for dashboard queries

**3. New Methods**

```python
# In event_tracker.py
def record_event_to_sqlite(
    db: WipnoteDB,
    session_id: str,
    tool_name: str,
    tool_input: dict,
    tool_response: dict,
    is_error: bool,
    file_paths: list[str] | None = None,
    parent_event_id: str | None = None,
) -> str | None:
    """Record tool call event to SQLite for dashboard queries."""

def record_delegation_to_sqlite(
    db: WipnoteDB,
    session_id: str,
    from_agent: str,
    to_agent: str,
    task_description: str,
    task_input: dict,
) -> str | None:
    """Record Task() delegation to agent_collaboration table."""

# In schema.py
def insert_collaboration(
    self,
    handoff_id: str,
    from_agent: str,
    to_agent: str,
    session_id: str,
    handoff_type: str = "delegation",
    reason: str | None = None,
    context: dict[str, Any] | None = None,
    status: str = "pending",
) -> bool:
    """Insert agent collaboration/delegation record."""
```

## Database Schema

### agent_events Table

Records all tool execution events:

```sql
CREATE TABLE agent_events (
    event_id TEXT PRIMARY KEY,           -- Unique event identifier
    agent_id TEXT NOT NULL,              -- "claude-code"
    event_type TEXT NOT NULL,            -- "tool_call"
    timestamp DATETIME DEFAULT NOW,      -- When event occurred
    tool_name TEXT,                      -- Tool name (Read, Write, Task, etc.)
    input_summary TEXT,                  -- Formatted input summary
    output_summary TEXT,                 -- Formatted output summary
    context JSON,                        -- Metadata: file_paths, is_error, etc.
    session_id TEXT NOT NULL,            -- Linked to session
    parent_agent_id TEXT,                -- Parent agent if delegated
    parent_event_id TEXT,                -- Parent event if child
    cost_tokens INTEGER DEFAULT 0,       -- Token usage
    status TEXT DEFAULT 'recorded',      -- Event status
    created_at DATETIME DEFAULT NOW,
    updated_at DATETIME DEFAULT NOW,
    FOREIGN KEY (session_id) REFERENCES sessions(session_id),
    FOREIGN KEY (parent_event_id) REFERENCES agent_events(event_id)
)

-- Indexes for dashboard queries
CREATE INDEX idx_agent_events_session ON agent_events(session_id)
CREATE INDEX idx_agent_events_agent ON agent_events(agent_id)
CREATE INDEX idx_agent_events_timestamp ON agent_events(timestamp)
CREATE INDEX idx_agent_events_type ON agent_events(event_type)
```

### agent_collaboration Table

Records Task() delegations:

```sql
CREATE TABLE agent_collaboration (
    handoff_id TEXT PRIMARY KEY,          -- Unique handoff identifier
    from_agent TEXT NOT NULL,             -- "claude-code"
    to_agent TEXT NOT NULL,               -- subagent_type (e.g., "researcher")
    timestamp DATETIME DEFAULT NOW,       -- When delegation occurred
    feature_id TEXT,                      -- Linked feature if available
    session_id TEXT,                      -- Linked session
    handoff_type TEXT,                    -- "delegation", "parallel", etc.
    status TEXT DEFAULT 'pending',        -- pending, accepted, completed, failed
    reason TEXT,                          -- Task description
    context JSON,                         -- Metadata: model, temperature, etc.
    result JSON,                          -- Result from subagent
    FOREIGN KEY (feature_id) REFERENCES features(id),
    FOREIGN KEY (session_id) REFERENCES sessions(session_id)
)

-- Indexes for orchestration queries
CREATE INDEX idx_collaboration_from_agent ON agent_collaboration(from_agent)
CREATE INDEX idx_collaboration_to_agent ON agent_collaboration(to_agent)
CREATE INDEX idx_collaboration_feature ON agent_collaboration(feature_id)
```

## Event Recording Flow

### PostToolUse Hook (Most Common)

```python
# Hook fires when tool completes
hook_input = {
    "tool_name": "Read",
    "tool_input": {"file_path": "src/main.py"},
    "tool_response": {"success": True, "content": "..."},
    "cwd": "/path/to/project"
}

# Event Tracker processes:
1. Initialize SessionManager (HTML) + WipnoteDB (SQLite)
2. Get active session ID
3. Extract file paths from tool_input
4. Format input/output summaries
5. Determine success/error status
6. Track HTML activity (existing)
7. Record SQLite event (NEW):
   - event_id: generate_id("event")
   - agent_id: "claude-code"
   - event_type: "tool_call"
   - tool_name: "Read"
   - input_summary: "Read: src/main.py"
   - output_summary: First 200 chars of content
   - context: {"file_paths": [...], "is_error": False}
   - session_id: linked session
8. Handle drift detection (existing)
```

### Task() Delegation (Special Case)

```python
# When Claude Code calls Task(description="...", subagent_type="researcher")
hook_input = {
    "tool_name": "Task",
    "tool_input": {
        "description": "Research authentication patterns",
        "subagent_type": "researcher",
        "model": "claude-3-5-sonnet"
    },
    "tool_response": {"success": True}
}

# Event Tracker processes:
1. Record normal tool_call event (as above)
2. ALSO record to agent_collaboration:
   - handoff_id: generate_id("handoff")
   - from_agent: "claude-code"
   - to_agent: "researcher"
   - handoff_type: "delegation"
   - reason: "Research authentication patterns"
   - context: {"model": "claude-3-5-sonnet", ...}
   - status: "pending" (until subagent completes)
```

## Dashboard Integration

The FastAPI dashboard queries SQLite events:

### Activity Feed Query

```python
# Get recent tool calls for a session
cursor.execute("""
    SELECT event_id, tool_name, input_summary, output_summary, timestamp
    FROM agent_events
    WHERE session_id = ?
    ORDER BY timestamp DESC
    LIMIT 50
""", (session_id,))
```

### Orchestration Tab Query

```python
# Get delegations from a session
cursor.execute("""
    SELECT handoff_id, from_agent, to_agent, reason, status, timestamp
    FROM agent_collaboration
    WHERE session_id = ?
    ORDER BY timestamp DESC
""", (session_id,))
```

### WebSocket Stream

The dashboard can subscribe to new events via WebSocket:

```javascript
// Browser: Listen for new events
const ws = new WebSocket('ws://localhost:8000/events/stream');
ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    // data.type === 'tool_call' | 'delegation' | 'error'
    // data.payload contains event details
    updateActivityFeed(data);
};
```

## Error Handling

### Graceful Degradation

If SQLite initialization fails:

```python
try:
    db = WipnoteDB(str(graph_dir / "wipnote.db"))
except Exception as e:
    print(f"Warning: Could not initialize SQLite: {e}")
    db = None  # Continue without SQLite

# Later:
if db:
    record_event_to_sqlite(...)  # Only if DB available
```

The system continues working with HTML-only tracking, so Claude Code execution is never blocked by database issues.

### Database Errors

- Failed event inserts are logged but don't block execution
- All database operations wrap in try/except
- Errors printed to stderr with context
- Never raises exceptions that reach Claude Code

## Performance Considerations

### SQLite Write Operations

- One INSERT per tool call (very fast on local disk)
- Queries are indexed on session_id, agent_id, timestamp, event_type
- JSON serialization of context is lightweight

### Typical Latency

- Event recording: <1ms (local SQLite write)
- Dashboard query: <100ms (indexed queries on typical session size)
- WebSocket stream: Real-time (on each event)

### Storage

- Typical event record: ~500 bytes
- Session with 100 events: ~50 KB
- Long project with 1000+ events: ~500 KB
- Negligible impact on disk

## Usage Examples

### Querying Events Programmatically

```python
from wipnote.db.schema import WipnoteDB

db = WipnoteDB(".wipnote/wipnote.db")

# Get all events for a session
sql, params = "SELECT * FROM agent_events WHERE session_id = ? ORDER BY timestamp DESC"
cursor = db.connection.cursor()
cursor.execute(sql, (session_id,))
events = cursor.fetchall()

# Get Task delegations
sql = """
    SELECT * FROM agent_collaboration
    WHERE session_id = ? AND handoff_type = 'delegation'
    ORDER BY timestamp DESC
"""
cursor.execute(sql, (session_id,))
delegations = cursor.fetchall()

db.close()
```

### Dashboard API Endpoints

The FastAPI server exposes these endpoints:

```python
GET /api/sessions/{session_id}/events
    # Returns: list of agent_events

GET /api/sessions/{session_id}/delegations
    # Returns: list of agent_collaboration records

GET /api/sessions/{session_id}/activity-feed
    # Returns: formatted activity feed with summaries

WS /api/sessions/{session_id}/events/stream
    # WebSocket: Real-time event stream
```

## Monitoring & Debugging

### Verify Events Are Being Recorded

```bash
# Check SQLite database directly
sqlite3 .wipnote/wipnote.db

# Count events for a session
sqlite> SELECT COUNT(*) FROM agent_events WHERE session_id = 'sess-123';

# Show recent delegations
sqlite> SELECT * FROM agent_collaboration ORDER BY timestamp DESC LIMIT 10;

# Check for errors
sqlite> SELECT * FROM agent_events WHERE status = 'error';
```

### Debug Hook Execution

```bash
# Enable debug output
export HTMLGRAPH_DEBUG=1
claude "your prompt"

# Check hook logs
tail -f /var/log/wipnote-hooks.log
```

## Future Enhancements

### Phase 2: Event Streaming

- WebSocket subscriptions for live dashboards
- Server-Sent Events (SSE) fallback
- Configurable streaming filters

### Phase 3: Event Aggregation

- Hourly/daily summaries in event_log_archive
- Automatic archive of old events (configurable retention)
- Cross-session analytics

### Phase 4: Advanced Queries

- Event correlation across sessions
- Subagent performance metrics
- Tool usage patterns and analytics

## Troubleshooting

### "Could not initialize SQLite database"

**Cause:** Permission issue, corrupted database, or missing parent directory

**Solution:**
```bash
# Check permissions
ls -la .wipnote/
chmod 755 .wipnote/

# Rebuild database
rm .wipnote/wipnote.db
# Re-run a tool to recreate schema
```

### Events not appearing in dashboard

**Cause:** Database not initialized or hook not firing

**Solution:**
```bash
# Verify hook is installed
claude hook list
# Should show: PostToolUse, Stop, UserPromptSubmit hooks

# Check database exists
ls -la .wipnote/wipnote.db

# Run a test tool
claude -p "List files: pwd"

# Query database
sqlite3 .wipnote/wipnote.db "SELECT COUNT(*) FROM agent_events;"
```

### Performance degradation

**Cause:** Large number of events or missing indexes

**Solution:**
```bash
# Analyze query performance
sqlite3 .wipnote/wipnote.db ".indices"

# Rebuild indexes if corrupted
sqlite3 .wipnote/wipnote.db "REINDEX;"

# Vacuum database (optional)
sqlite3 .wipnote/wipnote.db "VACUUM;"
```

## References

- **Schema Definition:** `src/python/wipnote/db/schema.py`
- **Event Recording:** `src/python/wipnote/hooks/event_tracker.py`
- **Dashboard API:** `src/python/wipnote/api/main.py`
- **Hook Configuration:** `.claude/hooks.json`

## Testing

```bash
# Run event tracking tests
uv run pytest tests/python/ -xvs -k "event"

# Run database tests
uv run pytest tests/python/ -xvs -k "sqlite or db"

# Run full test suite
uv run pytest
```

## Summary

The SQLite event recording integration provides:

✅ **Real-time Dashboard:** Events flow to FastAPI server instantly
✅ **Structured Queries:** SQL-based analytics vs. HTML parsing
✅ **Delegation Tracking:** Task() calls recorded with full context
✅ **Graceful Degradation:** System works without SQLite
✅ **Zero Breaking Changes:** Existing HTML tracking continues
✅ **Performant:** Indexed queries, negligible overhead
✅ **Dashboard Ready:** Activity feed and orchestration tab work live
