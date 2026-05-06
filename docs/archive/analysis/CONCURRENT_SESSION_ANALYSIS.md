# Cross-Session Awareness Research & Implementation Plan

## Executive Summary

Wipnote has implemented database-only storage for parent-child event linking, making the database the single source of truth. This analysis explores how to leverage this for concurrent session awareness—enabling all parallel orchestrator sessions to know about each other, coordinate work, and avoid duplicates.

**Current State**: Sessions are tracked in the database but **no cross-session awareness exists**. Each session operates independently.

**Opportunity**: The database now provides a unified backend to:
1. Detect concurrent sessions running in parallel
2. Inject awareness of other active sessions into each session's context
3. Enable orchestrator to coordinate work across sessions
4. Prevent duplicate work via shared state

---

## 1. Current Session-Start Hook Architecture

### Location
`packages/claude-plugin/.claude-plugin/hooks/scripts/session-start.py`
- **Type**: Thin shell wrapper (~110 lines)
- **Purpose**: Initializes session tracking when Claude Code session starts
- **Responsibility**: Hook entry point only; delegates all logic to `session_handler.py`

### Flow

```
session-start.py
  ↓ (loads hook input from stdin)
HookContext.from_input()
  ↓ (creates execution context with project/graph dirs)
init_or_get_session()
  ↓ (retrieves or creates session via SessionManager)
handle_session_start()
  ↓ (initializes database, loads features, checks version)
Output: sessionFeatureContext + versionInfo
```

### Key Details

**Session Creation** (`init_or_get_session` in `session_handler.py:43-89`):
```python
def init_or_get_session(context: HookContext) -> Any | None:
    manager = context.session_manager
    agent = context.agent_id

    # Try to get existing session for this agent
    active = manager.get_active_session_for_agent(agent=agent)
    if not active:
        # Create new session with commit info
        active = manager.start_session(
            session_id=None,
            agent=agent,
            start_commit=head_commit,
            title=f"Session {datetime.now().strftime('%Y-%m-%d %H:%M')}"
        )
    return active
```

**Session Start** (`handle_session_start` in `session_handler.py:92-204`):
1. Ensures session exists in database
2. Loads active features (those in "in-progress" status)
3. Checks version status
4. Returns feature context for injection

**Current Output**:
```json
{
  "continue": true,
  "hookSpecificOutput": {
    "sessionFeatureContext": "## Active Features\n- **feat-123**: Implement auth\n...",
    "versionInfo": null
  }
}
```

### Orchestrator Integration

**System Prompt** (`.claude/system-prompt.md`):
- Loaded automatically at session start
- Instructs Haiku on delegation patterns
- **CURRENTLY**: No mention of concurrent sessions or cross-window coordination
- Persists across compact/resume cycles via hook

**No Current Cross-Session Context**:
- Each session starts independently
- No awareness of other windows working on same project
- No coordination mechanism for parallel work

---

## 2. Database Schema for Concurrent Sessions

### Sessions Table

```sql
CREATE TABLE sessions (
    session_id TEXT PRIMARY KEY,
    agent_assigned TEXT NOT NULL,
    parent_session_id TEXT,
    parent_event_id TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME,
    total_events INTEGER DEFAULT 0,
    total_tokens_used INTEGER DEFAULT 0,
    context_drift REAL DEFAULT 0.0,
    status TEXT NOT NULL DEFAULT 'active' CHECK(
        status IN ('active', 'completed', 'paused', 'failed')
    ),
    transcript_id TEXT,
    transcript_path TEXT,
    transcript_synced DATETIME,
    start_commit TEXT,
    end_commit TEXT,
    is_subagent BOOLEAN DEFAULT FALSE,
    features_worked_on JSON,
    metadata JSON
);
```

### Indexes for Concurrent Session Queries

```sql
-- Active sessions (status='active', ordered by creation)
CREATE INDEX idx_sessions_status_created ON sessions(status, created_at DESC);

-- Query for concurrent sessions in time window:
SELECT session_id, agent_assigned, created_at, total_events
FROM sessions
WHERE status = 'active'
  AND created_at > datetime('now', '-15 minutes')
ORDER BY created_at DESC;
```

### Missing: Session Metadata

**Gap**: Sessions lack:
- `last_user_query_at` - When was the last user query in this session?
- `current_task_description` - What is this session currently working on?
- `concurrent_sessions` - JSON array of known peer sessions

### Available Queries

From `db/queries.py:476-496`:
```python
@staticmethod
def get_active_sessions() -> tuple[str, tuple]:
    """Get all currently active sessions."""
    sql = """
        SELECT
            session_id,
            agent_assigned,
            created_at,
            total_events,
            total_tokens_used,
            status
        FROM sessions
        WHERE status = 'active'
        ORDER BY created_at DESC
    """
    return sql, ()
```

---

## 3. Session-End Hook & Cleanup

### Location
`packages/claude-plugin/.claude-plugin/hooks/scripts/session-end.py`

### Design Philosophy
- **Lightweight**: Doesn't block on cleanup
- **Deferred Import**: Imports SessionManager only if needed
- **Graceful Degradation**: Continues even if components fail

### Current Operations

```python
def main():
    # 1. Get active session
    manager = SessionManager(graph_dir)
    active = manager.get_active_session()

    # 2. Link transcript (if Claude provides session_id)
    if active and external_session_id:
        reader = TranscriptReader()
        transcript = reader.read_session(external_session_id)
        if transcript:
            manager.link_transcript(...)

    # 3. Capture handoff notes (optional)
    if active and (handoff_notes or recommended_next or blockers):
        manager.set_session_handoff(...)

    # Output empty response (session end doesn't add context)
    print(json.dumps({"continue": True}))
```

### Missing: Session Completion

**Gap**: session-end.py doesn't:
- Mark session as `completed` in database
- Record `completed_at` timestamp
- Log what was accomplished
- Clean up temporary state files properly

---

## 4. Cross-Session Awareness Design

### 4.1 Detect Concurrent Sessions

**Query Pattern**: Find sessions active in last N minutes

```sql
SELECT
    session_id,
    agent_assigned,
    created_at,
    total_events,
    status,
    metadata
FROM sessions
WHERE status = 'active'
  AND created_at > datetime('now', '-{window_minutes} minutes')
ORDER BY created_at DESC;
```

**Implementation Location**: New module `wipnote/hooks/concurrent_sessions.py`

```python
def get_concurrent_sessions(
    db: WipnoteDB,
    current_session_id: str,
    window_minutes: int = 15,
) -> list[dict]:
    """
    Get all active sessions except current one.

    Args:
        db: Database connection
        current_session_id: Current session to exclude
        window_minutes: Time window to consider "concurrent"

    Returns:
        List of concurrent session dicts with metadata
    """
    cursor = db.connection.cursor()
    sql = """
        SELECT
            session_id,
            agent_assigned,
            created_at,
            total_events,
            status
        FROM sessions
        WHERE status = 'active'
          AND session_id != ?
          AND created_at > datetime('now', '-? minutes')
        ORDER BY created_at DESC
    """
    cursor.execute(sql, (current_session_id, window_minutes))

    concurrent = []
    for row in cursor.fetchall():
        session = dict(row)
        # Enrich with last event info
        last_event = _get_last_user_query(db, row['session_id'])
        if last_event:
            session['last_query'] = last_event['input_summary']
            session['last_query_at'] = last_event['timestamp']
        concurrent.append(session)

    return concurrent


def _get_last_user_query(db: WipnoteDB, session_id: str) -> dict | None:
    """Get most recent user query event in a session."""
    cursor = db.connection.cursor()
    cursor.execute("""
        SELECT event_id, input_summary, timestamp
        FROM agent_events
        WHERE session_id = ?
          AND event_type IN ('user_query', 'start')
        ORDER BY timestamp DESC
        LIMIT 1
    """, (session_id,))

    row = cursor.fetchone()
    return dict(row) if row else None
```

### 4.2 Inject Concurrent Context at Session-Start

**Modification**: Update `handle_session_start()` in `session_handler.py`

**Before** (current):
```python
def handle_session_start(context: HookContext, session: Any | None) -> dict:
    output = {
        "hookSpecificOutput": {
            "sessionFeatureContext": "",
            "versionInfo": None,
        }
    }
    # Load active features only
    features = _load_features(context.graph_dir)
    # ...
```

**After** (with concurrent awareness):
```python
def handle_session_start(context: HookContext, session: Any | None) -> dict:
    output = {
        "hookSpecificOutput": {
            "sessionFeatureContext": "",
            "concurrentSessions": "",  # NEW
            "versionInfo": None,
        }
    }

    # Load active features
    features = _load_features(context.graph_dir)
    if features:
        # ... existing code ...

    # NEW: Load concurrent sessions
    try:
        from wipnote.hooks.concurrent_sessions import get_concurrent_sessions

        concurrent = get_concurrent_sessions(
            db=context.database,
            current_session_id=context.session_id,
            window_minutes=15
        )

        if concurrent:
            session_list = "\n".join([
                f"- **{s['session_id'][:12]}** ({s['agent_assigned']}): "
                f"'{s.get('last_query', 'No query recorded')}' ({s['total_events']} events)"
                for s in concurrent
            ])
            concurrent_context = f"""## Concurrent Sessions (Active Now)

{session_list}

These sessions are running in parallel. Consider:
- Avoid duplicating work already started in other windows
- Check if another session already handles your planned work
- Use Task() to delegate if a more specialized session exists"""

            output["hookSpecificOutput"]["concurrentSessions"] = concurrent_context
            context.log("info", f"Found {len(concurrent)} concurrent sessions")
    except Exception as e:
        context.log("warning", f"Could not load concurrent sessions: {e}")

    return output
```

### 4.3 Update Database When Session Ends

**Modification**: Update `session-end.py` hook

**Current** (incomplete):
```python
def main():
    try:
        manager = SessionManager(graph_dir)
        active = manager.get_active_session()

        # Link transcript, capture notes
        # ...

    # Output empty response (does NOT mark session as completed!)
    print(json.dumps({"continue": True}))
```

**After** (properly mark completion):
```python
def main():
    try:
        manager = SessionManager(graph_dir)
        db = WipnoteDB(str(Path(graph_dir) / "wipnote.db"))
        active = manager.get_active_session()

        # Existing: Link transcript, capture notes
        if active and external_session_id:
            reader = TranscriptReader()
            transcript = reader.read_session(external_session_id)
            if transcript:
                manager.link_transcript(...)

        if active and (handoff_notes or recommended_next or blockers):
            manager.set_session_handoff(...)

        # NEW: Mark session as completed
        if active:
            try:
                db.update_session_status(
                    session_id=active.id,
                    status='completed',
                    completed_at=datetime.now(timezone.utc).isoformat()
                )
                context.log("info", f"Session marked complete: {active.id}")
            except Exception as e:
                context.log("warning", f"Could not mark session complete: {e}")

        # Cleanup temp files
        _cleanup_temp_files(graph_dir)

    except Exception as e:
        print(f"Warning: Could not end session: {e}", file=sys.stderr)

    print(json.dumps({"continue": True}))
```

### 4.4 Orchestrator Coordination

**Modification**: Update orchestrator directives in system prompt

**Current** (`.claude/system-prompt.md`):
```markdown
## Orchestration Pattern
- Use `Task()` tool for multi-session work, deep research, or complex reasoning
- Execute directly only for straightforward file operations or quick implementations
- Haiku: Default orchestrator—excellent at following delegation instructions
```

**After** (with concurrent awareness):
```markdown
## Orchestration Pattern & Cross-Window Coordination

**Primary Directives:**
- Use `Task()` tool for multi-session work, deep research, or complex reasoning
- Execute directly only for straightforward file operations or quick implementations
- When concurrent sessions detected: Avoid duplicate work, coordinate with peers

**Concurrent Session Awareness:**
If this session finds other active sessions:
1. **Check their work** - What are they focused on? (shown in SessionStart context)
2. **Avoid duplicates** - Don't start work already in progress in another session
3. **Coordinate via Task()** - If you need results from another session, use Task()
4. **Share findings** - Record discoveries in .wipnote/concurrent-findings.md

**Cross-Window Scenarios:**
- Window A: Implementing feature X; Window B: Researching feature Y
  → Windows should coordinate via database (write findings, share links)
- Window A: Testing; Window B: Implementation
  → Coordinate via Task() for test execution in Window A

**Decision Framework:**
- Same project, different features → Parallel work (good!)
- Same project, same feature → Coordinate (bad to duplicate)
- Different projects → Fully independent
```

---

## 5. Database Schema Extensions

### Add Concurrent Session Tracking Columns

```sql
-- Add to sessions table
ALTER TABLE sessions ADD COLUMN last_user_query_at DATETIME;
ALTER TABLE sessions ADD COLUMN last_user_query TEXT;
ALTER TABLE sessions ADD COLUMN concurrent_sessions TEXT;  -- JSON array
ALTER TABLE sessions ADD COLUMN coordination_notes TEXT;   -- Hand-off notes

-- Create concurrent_events table (optional, for activity feed)
CREATE TABLE IF NOT EXISTS concurrent_events (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    event_type TEXT NOT NULL,  -- 'session_active', 'session_complete', 'coordinate'
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    from_session_id TEXT,
    to_session_id TEXT,
    message TEXT,
    FOREIGN KEY (session_id) REFERENCES sessions(session_id)
);

-- Index for concurrent event queries
CREATE INDEX idx_concurrent_events_session ON concurrent_events(session_id, timestamp DESC);
CREATE INDEX idx_concurrent_events_type ON concurrent_events(event_type, timestamp DESC);
```

### Update Session Insert to Record Last Query

```python
def update_session_last_query(
    db: WipnoteDB,
    session_id: str,
    query_text: str,
    event_id: str | None = None
) -> bool:
    """Record the last user query in a session."""
    try:
        cursor = db.connection.cursor()
        cursor.execute("""
            UPDATE sessions
            SET last_user_query_at = CURRENT_TIMESTAMP,
                last_user_query = ?
            WHERE session_id = ?
        """, (query_text[:500], session_id))  # Truncate to 500 chars

        db.connection.commit()
        return True
    except Exception as e:
        logger.error(f"Error updating last query: {e}")
        return False
```

---

## 6. Session Lifecycle States

### Current State Machine (INCOMPLETE)

```
session created (created_at set)
  ↓
session active (user works)
  ↓
session ends (session-end hook fires)
  ✗ BUG: completed_at never set!
  ✗ BUG: status never marked 'completed'
```

### Proposed State Machine (FIXED)

```
+─────────────────────────────────────────────┐
│ SESSION LIFECYCLE                           │
├─────────────────────────────────────────────┤
│                                             │
│  CREATE (session-start.py)                  │
│  ├─ Insert: session_id, agent, created_at  │
│  ├─ Set: status='active'                    │
│  └─ Log: SessionStart event                 │
│                                             │
│  ACTIVE (Claude Code window is open)        │
│  ├─ Record: user_query events               │
│  ├─ Update: last_user_query_at              │
│  ├─ Query: Other active sessions (detect)   │
│  └─ Detect: Concurrent sessions on refresh  │
│                                             │
│  END (session-end.py)                       │
│  ├─ Update: status='completed'              │
│  ├─ Set: completed_at = now()               │
│  ├─ Record: SessionEnd event                │
│  ├─ Link: Transcript (if available)         │
│  ├─ Save: Handoff notes                     │
│  └─ Cleanup: Temp files                     │
│                                             │
│  COMPLETED (archive window)                 │
│  ├─ Visible: In session history             │
│  ├─ Query: For post-analysis                │
│  └─ Linked: To features worked on           │
│                                             │
└─────────────────────────────────────────────┘
```

### State Transitions

```python
class SessionStatus(Enum):
    """Session lifecycle states."""
    ACTIVE = "active"       # Session is running
    PAUSED = "paused"       # Temporarily paused (for future use)
    COMPLETED = "completed" # Session ended normally
    FAILED = "failed"       # Session ended with error
    TIMEOUT = "timeout"     # Session timed out (no activity for hours)

class SessionMetadata(BaseModel):
    """Track important session metadata."""
    created_at: datetime           # When session started
    completed_at: datetime | None  # When session ended
    last_user_query_at: datetime   # Last user input time
    last_user_query: str           # Last user query text
    concurrent_sessions: list[str] # Peer sessions detected at start
    features_worked_on: list[str]  # Feature IDs linked to this session
    total_events: int              # Event count for this session
    completion_reason: str         # Why did it end? ("window_closed", "user_timeout", etc.)
    coordination_notes: str        # Notes about cross-window work
```

---

## 7. Hook Modifications Summary

### session-start.py Changes

**What to Add**:
1. Call `get_concurrent_sessions()` after session created
2. Inject concurrent sessions into `sessionFeatureContext`
3. Log concurrent sessions for debugging

**Changes**:
```python
# In handle_session_start()
concurrent = get_concurrent_sessions(context.database, context.session_id)
if concurrent:
    output["hookSpecificOutput"]["concurrentSessions"] = format_concurrent_context(concurrent)
```

### session-end.py Changes

**What to Add**:
1. Mark session as `completed` in database
2. Set `completed_at` timestamp
3. Record completion reason
4. Properly cleanup temp files

**Changes**:
```python
# In main()
if active:
    db.update_session_status(
        session_id=active.id,
        status='completed',
        completed_at=datetime.now(timezone.utc).isoformat()
    )
```

### user-prompt-submit.py Changes

**What to Add**:
1. Update `last_user_query_at` when user submits query
2. Update `last_user_query` with truncated prompt text
3. Check for concurrent sessions and log coordination info

**Pattern** (if this hook exists):
```python
# When recording UserQuery event
db.update_session_last_query(
    session_id=context.session_id,
    query_text=prompt[:500]
)
```

---

## 8. Missing Pieces & Recommendations

### Critical Gaps

1. **Session Completion Not Recorded**
   - `session-end.py` doesn't mark session as `completed`
   - `completed_at` is never set
   - Sessions appear "active" forever
   - **Fix**: Update session-end.py to call `db.update_session_status()`

2. **No Last Query Tracking**
   - Concurrent sessions don't know what peers are working on
   - Can't display meaningful "other windows" info
   - **Fix**: Add `last_user_query_at` and `last_user_query` columns

3. **No Concurrent Session Awareness Module**
   - Query exists in `queries.py` but no utility module wraps it
   - session-start.py doesn't use it
   - **Fix**: Create `wipnote/hooks/concurrent_sessions.py`

4. **System Prompt Doesn't Mention Coordination**
   - Orchestrator has no guidance on cross-window work
   - No patterns for detecting duplicate work
   - **Fix**: Update `.claude/system-prompt.md` with coordination patterns

5. **No Session-End Cleanup**
   - Temp files (parent-activity.json, etc.) not properly cleaned
   - Orphaned state files accumulate
   - **Fix**: Implement proper cleanup in session-end.py

### Optional Enhancements

1. **Concurrent Activity Feed**
   - New table: `concurrent_events`
   - Track: session created, completed, major milestones
   - Display: "other windows" activity in real-time

2. **Cross-Window Coordination Table**
   - Record: when windows coordinate work
   - Track: which sessions collaborated on features
   - Query: "which sessions worked together?"

3. **Session Timeout Detection**
   - Mark sessions as `timeout` if no activity for 1+ hour
   - Assume window was closed without proper session-end hook
   - **Current risk**: Sessions marked `active` indefinitely

4. **Concurrent Session Metrics**
   - Count: How many parallel sessions typically run?
   - Duration: How long are concurrent sessions active?
   - Overlap: When do sessions work together vs independently?

---

## 9. Implementation Roadmap

### Phase 1: Database Foundations (2-3 days)

**Objective**: Make database properly track session completion

**Tasks**:
1. Add missing schema columns
   - `last_user_query_at`
   - `last_user_query`
   - `completed_at` (fix existing)
   - `completion_reason`

2. Add database methods
   - `update_session_status()`
   - `update_session_last_query()`
   - `mark_session_complete()`

3. Update session-end.py
   - Mark session as `completed`
   - Set `completed_at`
   - Test: Verify sessions are no longer "active" after end hook

### Phase 2: Concurrent Detection (3-4 days)

**Objective**: Detect and surface concurrent sessions

**Tasks**:
1. Create `wipnote/hooks/concurrent_sessions.py`
   - Implement `get_concurrent_sessions()`
   - Implement `_get_last_user_query()`
   - Add helper for formatting context

2. Update session-start.py
   - Call `get_concurrent_sessions()` after session init
   - Inject into `sessionFeatureContext`
   - Test: Open 2 windows, verify session awareness

3. Add logging/debugging
   - Log concurrent sessions found
   - Track how many sessions run concurrently
   - Measure performance impact

### Phase 3: Orchestrator Integration (4-5 days)

**Objective**: Enable cross-window coordination

**Tasks**:
1. Update system prompt
   - Add coordination directives
   - Include concurrent session scenario patterns
   - Explain cross-window decision framework

2. Update user-prompt-submit hook (if exists)
   - Record `last_user_query_at` on each prompt
   - Update `last_user_query` text

3. Create coordination patterns
   - Document how sessions should work together
   - Create examples of parallel work
   - Test: Verify orchestrator respects patterns

4. Add optional: coordination table
   - Track which sessions collaborated
   - Record coordination events
   - Enable post-analysis of parallel work

### Phase 4: Polish & Optimization (2-3 days)

**Objective**: Harden for production

**Tasks**:
1. Error handling
   - Graceful degradation if DB unavailable
   - Timeout protection for queries
   - Proper logging

2. Performance
   - Measure index effectiveness
   - Profile concurrent session queries
   - Optimize for 10+ concurrent sessions

3. Testing
   - Unit tests for concurrent detection
   - Integration tests with multiple sessions
   - E2E tests simulating user workflow

4. Documentation
   - User guide: "Working across multiple windows"
   - Architecture docs: "Concurrent session awareness"
   - Troubleshooting: "Why aren't concurrent sessions showing?"

---

## 10. Code Examples

### Example 1: Get Concurrent Sessions

```python
from wipnote.db.schema import WipnoteDB
from wipnote.hooks.concurrent_sessions import get_concurrent_sessions

db = WipnoteDB()
current_session = "sess-abc123"

concurrent = get_concurrent_sessions(
    db=db,
    current_session_id=current_session,
    window_minutes=15
)

# Output:
# [
#   {
#     'session_id': 'sess-def456',
#     'agent_assigned': 'claude',
#     'created_at': '2025-01-10 12:30:45',
#     'total_events': 42,
#     'last_query': 'Implement user authentication',
#     'last_query_at': '2025-01-10 12:35:10'
#   }
# ]
```

### Example 2: Inject Concurrent Context

```python
def handle_session_start(context: HookContext, session: Any | None) -> dict:
    output = {
        "hookSpecificOutput": {
            "sessionFeatureContext": "",
            "concurrentSessions": ""
        }
    }

    # Existing: load features
    # ... (feature loading code)

    # NEW: load concurrent sessions
    concurrent = get_concurrent_sessions(
        db=context.database,
        current_session_id=context.session_id
    )

    if concurrent:
        # Format for display
        lines = ["## Concurrent Sessions (Active Now)", ""]
        for s in concurrent:
            lines.append(
                f"- **{s['session_id'][:12]}** ({s['agent_assigned']}): "
                f"'{s.get('last_query', 'working')}'"
            )

        output["hookSpecificOutput"]["concurrentSessions"] = "\n".join(lines)

    return output
```

### Example 3: Update Session Status

```python
from datetime import datetime, timezone
from wipnote.db.schema import WipnoteDB

db = WipnoteDB()

# Mark session as completed
db.update_session_status(
    session_id="sess-abc123",
    status="completed",
    completed_at=datetime.now(timezone.utc).isoformat()
)

# Update last query
db.update_session_last_query(
    session_id="sess-abc123",
    query_text="Implement password reset feature"
)
```

---

## 11. Testing Strategy

### Unit Tests

**Test: get_concurrent_sessions()**
```python
def test_get_concurrent_sessions():
    db = WipnoteDB(":memory:")  # In-memory test DB

    # Create 3 sessions: 2 active, 1 completed
    db.insert_session("sess-1", "claude")
    db.insert_session("sess-2", "claude")
    db.insert_session("sess-3", "claude")

    # Mark sess-3 as completed
    db.update_session_status("sess-3", "completed")

    # Query from sess-1's perspective
    concurrent = get_concurrent_sessions(
        db=db,
        current_session_id="sess-1",
        window_minutes=60
    )

    # Should return only sess-2 (sess-3 is completed, sess-1 is excluded)
    assert len(concurrent) == 1
    assert concurrent[0]['session_id'] == 'sess-2'
```

### Integration Tests

**Test: Full session lifecycle with concurrent detection**
```python
def test_session_lifecycle_with_concurrent():
    # Setup
    project_dir = "/tmp/test-project"
    Path(project_dir).mkdir(exist_ok=True)

    # Window 1: Start session
    context1 = HookContext(
        project_dir=project_dir,
        session_id="sess-window1",
        agent_id="claude"
    )

    # Window 2: Start session (concurrent)
    context2 = HookContext(
        project_dir=project_dir,
        session_id="sess-window2",
        agent_id="claude"
    )

    # Session 1 should see Session 2
    concurrent1 = get_concurrent_sessions(context1.database, context1.session_id)
    assert len(concurrent1) == 1
    assert concurrent1[0]['session_id'] == 'sess-window2'

    # Session 2 should see Session 1
    concurrent2 = get_concurrent_sessions(context2.database, context2.session_id)
    assert len(concurrent2) == 1
    assert concurrent2[0]['session_id'] == 'sess-window1'

    # Window 1 ends
    mark_session_complete("sess-window1")

    # Window 2 should no longer see Window 1
    concurrent2_after = get_concurrent_sessions(context2.database, context2.session_id)
    assert len(concurrent2_after) == 0
```

### E2E Tests

**Test: Orchestrator respects concurrent sessions**
```python
def test_orchestrator_avoids_duplicate_work():
    # Setup 2 concurrent windows
    window1 = start_claude_code_session()
    window2 = start_claude_code_session()

    # Window 1: User asks to implement feature X
    window1.input("Implement user authentication")

    # Wait for orchestrator to detect work starting
    time.sleep(2)

    # Window 2: User also asks to implement feature X
    window2.input("Implement user authentication")

    # Check: Window 2 should be aware of Window 1's work
    output = window2.get_session_context()
    assert "sess-window1" in output["concurrentSessions"]
    assert "user authentication" in output["concurrentSessions"]

    # Check: Window 2 should delegate to Window 1 (avoid duplicate)
    assert window2.last_task_type == "Task"
    assert "coordinate" in window2.last_task_description.lower()
```

---

## 12. Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| **Sessions marked active forever** | Wrong concurrent detection | Fix session-end.py to mark `completed` |
| **Query performance on 100+ sessions** | Slow context injection | Add indexes, pagination, TTL for old sessions |
| **Race condition: 2 windows start simultaneously** | Duplicate session IDs | Use UUID, test with concurrent.futures |
| **Orphaned sessions (window crash)** | False "concurrent" detection | Add timeout: sessions inactive 1h → timeout |
| **Circular delegation (A delegates to B, B to A)** | Infinite loops | Add delegation depth limit in orchestrator |
| **User not aware of concurrent work** | Duplicated effort | Surface in system prompt and context |

---

## 13. Success Metrics

### Phase 1: Database Foundation
- [ ] Sessions properly marked `completed` at end
- [ ] `completed_at` timestamp recorded
- [ ] Session status query returns correct results

### Phase 2: Concurrent Detection
- [ ] `get_concurrent_sessions()` returns correct peer sessions
- [ ] `last_user_query` updated on each user prompt
- [ ] Performance: Query completes in <50ms for 100 sessions

### Phase 3: Orchestrator Integration
- [ ] System prompt mentions concurrent sessions
- [ ] Orchestrator acknowledges "other windows" in initial response
- [ ] Users report aware of parallel work

### Phase 4: Polish & Testing
- [ ] 90%+ test coverage for new modules
- [ ] E2E tests pass with 5+ concurrent sessions
- [ ] Zero false positives in concurrent detection

---

## Appendix: Database Migration

### Safe Migration Path

```sql
-- 1. Add new columns (backwards compatible)
ALTER TABLE sessions ADD COLUMN last_user_query_at DATETIME;
ALTER TABLE sessions ADD COLUMN last_user_query TEXT;
ALTER TABLE sessions ADD COLUMN completion_reason TEXT DEFAULT 'unknown';

-- 2. Update existing sessions (nullable, safe)
UPDATE sessions
SET last_user_query_at = created_at
WHERE last_user_query_at IS NULL;

-- 3. Create new optional table (doesn't affect existing data)
CREATE TABLE IF NOT EXISTS concurrent_events (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    message TEXT,
    FOREIGN KEY (session_id) REFERENCES sessions(session_id)
);

-- 4. Create index for performance
CREATE INDEX IF NOT EXISTS idx_sessions_status_created
ON sessions(status, created_at DESC);
```

### Version Check
```python
def check_schema_version():
    """Verify schema has concurrent session columns."""
    cursor = db.connection.cursor()
    cursor.execute("PRAGMA table_info(sessions)")
    columns = {row[1] for row in cursor.fetchall()}

    required = {'last_user_query_at', 'last_user_query'}
    missing = required - columns

    if missing:
        raise SchemaError(
            f"Schema missing columns: {missing}. "
            "Run migrations first: wipnote migrate"
        )
```

---

## Conclusion

The database-first architecture provides a solid foundation for cross-session awareness. Key work:

1. **Fix session lifecycle** - Mark completion in session-end.py
2. **Track last query** - Inject metadata on user-prompt-submit
3. **Create concurrent module** - Query and format peer sessions
4. **Update orchestrator** - Add coordination directives
5. **Test thoroughly** - E2E tests with concurrent windows

This enables Haiku to make intelligent cross-window decisions, coordinate parallel work, and avoid duplicating effort across projects.

