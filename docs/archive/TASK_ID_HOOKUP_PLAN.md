# Task ID Hookup Plan - Linking Claude Code Tasks to HtmlGraph Events

## Executive Summary

Claude Code's Task() tool creates tasks with internal `task_id` values, but we don't currently capture these IDs. When a Task completes, Claude Code shows "No child events" in its notification because it can't link to our HtmlGraph events.

**Solution:** Capture Claude Code's `task_id` in our hooks and store the mapping to our `event_id` values.

---

## What Data Flows Through Hooks

### Scenario: User calls Task()

```
Claude Code             HtmlGraph Hook
─────────────────────────────────────

User: Task(...)
  ↓
Task() tool executes
  ↓
PreToolUse fires ─────→ pretooluse.py
                       Input:
                       {
                           "name": "Task",
                           "input": { "prompt": "...", ... },
                           "session_id": "sess-abc",
                           "cwd": "/path"
                       }

                       ❌ NO task_id HERE YET

  ↓
Task() completes
  ↓
PostToolUse fires ────→ posttooluse.py
                       Input:
                       {
                           "name": "Task",
                           "tool_response": {
                               "task_id": "task-xyz?",  ← NEED TO VERIFY
                               "status": "started",
                               "result": {...}
                           },
                           "session_id": "sess-abc"
                       }
  ↓
Subagent starts running
  ↓
SubagentStop fires ───→ subagent_stop.py
                       Input:
                       {
                           "session_id": "sess-xyz",
                           "status": "completed"
                       }

                       ❌ NO task_id HERE EITHER
```

---

## Current Hook Implementation

### PreToolUse: Creates task_delegation Event
**File:** `src/python/htmlgraph/hooks/pretooluse.py` (lines 218-313)

```python
def create_task_parent_event(db, tool_input, session_id, start_time):
    """Create parent event for Task() delegations."""
    parent_event_id = f"evt-{str(uuid.uuid4())[:8]}"
    subagent_type = extract_subagent_type(tool_input)

    cursor.execute("""
        INSERT INTO agent_events
        (event_id, agent_id, event_type, timestamp, tool_name,
         input_summary, session_id, status, subagent_type, parent_event_id)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    """, (
        parent_event_id,           # ← Our UUID
        "claude-code",
        "task_delegation",         # ← Event type
        start_time,
        "Task",
        input_summary,
        session_id,
        "started",
        subagent_type,
        user_query_event_id,
    ))

    # Export for subagent context
    os.environ["HTMLGRAPH_PARENT_EVENT"] = parent_event_id

    return parent_event_id
```

**What's captured:**
- ✅ `event_id` = `evt-abc123` (our UUID)
- ✅ `event_type` = `task_delegation`
- ✅ `subagent_type` = from input
- ❌ `claude_task_id` = NOT CAPTURED

**Issue:** We create our own event_id but never capture Claude Code's task_id.

### PostToolUse: Records Event Results
**File:** `src/python/htmlgraph/hooks/posttooluse.py` (lines 39-65)

```python
async def run_event_tracking(hook_type, hook_input):
    """Track tool execution."""
    return await loop.run_in_executor(
        None,
        track_event,
        hook_type,
        hook_input,
    )
```

**What happens:**
1. Calls `track_event()` from `event_tracker.py`
2. That function records tool usage but **doesn't check for task_id**
3. Stores to `agent_events` table but **no task_id mapping**

### SubagentStop: Updates Parent Event
**File:** `src/python/htmlgraph/hooks/subagent_stop.py` (lines 207-285)

```python
def handle_subagent_stop(hook_input):
    """Handle subagent completion."""
    parent_event_id = get_parent_event_id()  # From env var

    # Get parent event from database
    parent_start_time = get_parent_event_start_time(db_path, parent_event_id)

    # Count child spikes
    child_spike_count = count_child_spikes(db_path, parent_event_id, parent_start_time)

    # Update parent event
    update_parent_event(
        db_path,
        parent_event_id,
        child_spike_count,
        completion_time
    )
```

**What's updated:**
- ✅ Parent event status → `completed`
- ✅ Child spike count
- ❌ No task_id linking

---

## Database Schema

### agent_events Table (Current)
**File:** `src/python/htmlgraph/db/schema.py` (lines 205-235)

```sql
CREATE TABLE agent_events (
    event_id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    tool_name TEXT,
    input_summary TEXT,
    output_summary TEXT,
    context JSON,  ← Can store claude_task_id here
    session_id TEXT NOT NULL,
    feature_id TEXT,
    parent_event_id TEXT,
    subagent_type TEXT,
    status TEXT DEFAULT 'recorded',
    model TEXT,
    -- NO column for claude_task_id
);
```

**Options to store task_id:**

**Option 1: Use context JSON (non-breaking)**
```json
{
  "file_paths": [...],
  "tool_input_keys": [...],
  "is_error": false,
  "claude_task_id": "task-xyz123"  ← Add here
}
```

**Option 2: Add explicit column (breaking change)**
```sql
ALTER TABLE agent_events
ADD COLUMN claude_task_id TEXT;
```

**Option 3: Create mapping table (best for queries)**
```sql
CREATE TABLE claude_task_mappings (
    event_id TEXT PRIMARY KEY,
    claude_task_id TEXT UNIQUE NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (event_id) REFERENCES agent_events(event_id) ON DELETE CASCADE
);
```

---

## Implementation Plan

### Phase 1: Verification (CRITICAL FIRST STEP)

**Add debug logging to PostToolUse hook:**

```python
# In src/python/htmlgraph/hooks/posttooluse.py
# Add to run_event_tracking() function

async def run_event_tracking(hook_type, hook_input):
    """Track tool execution."""
    import sys

    # DEBUG: Log what we receive for Task() tool
    tool_name = hook_input.get("name", "") or hook_input.get("tool_name", "")
    if tool_name == "Task":
        tool_response = hook_input.get("tool_response", {})
        print(f"\n=== TASK RESPONSE DEBUG ===", file=sys.stderr)
        print(f"tool_name: {tool_name}", file=sys.stderr)
        print(f"tool_response type: {type(tool_response)}", file=sys.stderr)
        print(f"tool_response keys: {list(tool_response.keys()) if isinstance(tool_response, dict) else 'N/A'}", file=sys.stderr)
        if isinstance(tool_response, dict):
            print(f"Full response: {json.dumps(tool_response, indent=2, default=str)}", file=sys.stderr)
            task_id = tool_response.get("task_id")
            print(f"task_id field: {task_id}", file=sys.stderr)
        print(f"=== END DEBUG ===\n", file=sys.stderr)

    return await loop.run_in_executor(None, track_event, hook_type, hook_input)
```

**Run this and check Claude Code logs to see if task_id is present.**

### Phase 2: Capture task_id (If Available)

**If Phase 1 shows task_id IS present:**

**Step 2a: Modify event_tracker.py to capture task_id**

```python
# In record_event_to_sqlite()
def record_event_to_sqlite(
    db: HtmlGraphDB,
    session_id: str,
    tool_name: str,
    tool_input: dict[str, Any],
    tool_response: dict[str, Any],
    is_error: bool,
    ...,
    claude_task_id: str | None = None,  # NEW
    ...
):
    """Record a tool call event to SQLite."""
    try:
        event_id = generate_id("event")

        # Build context with task_id
        context = {
            "file_paths": file_paths or [],
            "tool_input_keys": list(tool_input.keys()),
            "is_error": is_error,
        }

        # Add Claude task_id if present
        if claude_task_id:
            context["claude_task_id"] = claude_task_id

        # Insert event
        success = db.insert_event(
            event_id=event_id,
            ...,
            context=context,  # context is JSON, stores task_id
            ...
        )
```

**Step 2b: Extract task_id in posttooluse.py**

```python
# In track_event() function in event_tracker.py
# When hook_type == "PostToolUse"

elif hook_type == "PostToolUse":
    tool_name = hook_input.get("tool_name", "unknown")
    tool_response = hook_input.get("tool_response", {}) or {}

    # Extract claude_task_id if this is a Task() call
    claude_task_id = None
    if tool_name == "Task" and isinstance(tool_response, dict):
        claude_task_id = tool_response.get("task_id")

    # ... existing code ...

    # When recording to SQLite, pass claude_task_id
    record_event_to_sqlite(
        db=db,
        session_id=active_session_id,
        tool_name=tool_name,
        tool_input=tool_input_data,
        tool_response=tool_response,
        is_error=is_error,
        ...,
        claude_task_id=claude_task_id,  # NEW
        ...
    )
```

### Phase 3: Create Lookup Table (Optional but Recommended)

**Add migration to schema.py:**

```python
def create_tables(self):
    """Create all required tables."""
    if not self.connection:
        self.connect()

    cursor = self.connection.cursor()

    # ... existing tables ...

    # NEW: Claude task ID mappings
    cursor.execute("""
        CREATE TABLE IF NOT EXISTS claude_task_mappings (
            event_id TEXT PRIMARY KEY,
            claude_task_id TEXT UNIQUE NOT NULL,
            tool_name TEXT NOT NULL,
            session_id TEXT NOT NULL,
            created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY (event_id) REFERENCES agent_events(event_id) ON DELETE CASCADE
        )
    """)
```

### Phase 4: Update PreToolUse to Link to Task

**When Task() is detected, store the mapping immediately:**

```python
# In pretooluse.py - after creating parent_event

def create_task_parent_event(db, tool_input, session_id, start_time):
    """Create parent event for Task() delegations."""
    parent_event_id = f"evt-{str(uuid.uuid4())[:8]}"

    # ... existing code to create agent_events record ...

    # NEW: Store mapping placeholder
    # We'll update with real task_id when PostToolUse fires
    cursor.execute("""
        INSERT OR IGNORE INTO claude_task_mappings
        (event_id, claude_task_id, tool_name, session_id, created_at)
        VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
    """, (
        parent_event_id,
        "pending",  # Will be updated when we get task_id
        "Task",
        session_id,
    ))

    return parent_event_id
```

### Phase 5: Update PostToolUse to Complete the Mapping

```python
# In track_event() - PostToolUse handler

elif hook_type == "PostToolUse":
    tool_name = hook_input.get("tool_name", "unknown")
    tool_response = hook_input.get("tool_response", {}) or {}

    # Extract task_id for Task() calls
    claude_task_id = None
    if tool_name == "Task" and isinstance(tool_response, dict):
        claude_task_id = tool_response.get("task_id")

    # ... record event ...

    # Update mapping with real task_id
    if tool_name == "Task" and claude_task_id and db:
        try:
            cursor = db.connection.cursor()

            # Find the parent_event we created in PreToolUse
            # It should be the most recent task_delegation for this session
            cursor.execute("""
                SELECT event_id FROM agent_events
                WHERE session_id = ?
                AND event_type = 'task_delegation'
                AND status = 'started'
                ORDER BY timestamp DESC
                LIMIT 1
            """, (active_session_id,))

            row = cursor.fetchone()
            if row:
                parent_event_id = row[0]

                # Update mapping with real task_id
                cursor.execute("""
                    UPDATE claude_task_mappings
                    SET claude_task_id = ?
                    WHERE event_id = ?
                """, (claude_task_id, parent_event_id))

                db.connection.commit()
                print(f"Linked Task: event_id={parent_event_id}, task_id={claude_task_id}",
                      file=sys.stderr)
        except Exception as e:
            print(f"Warning: Could not update task mapping: {e}", file=sys.stderr)
```

---

## Implementation Checklist

- [ ] **Phase 1: Verification**
  - [ ] Add debug logging to posttooluse.py for Task() responses
  - [ ] Run Claude Code with Task() and check logs for task_id presence
  - [ ] Document findings in spike

- [ ] **Phase 2: Capture task_id**
  - [ ] Modify `record_event_to_sqlite()` to accept `claude_task_id` parameter
  - [ ] Extract task_id from tool_response in `track_event()`
  - [ ] Pass task_id to context JSON

- [ ] **Phase 3: Create Lookup Table**
  - [ ] Add `claude_task_mappings` table to schema
  - [ ] Create migration logic for existing databases

- [ ] **Phase 4: Update PreToolUse**
  - [ ] Store placeholder mapping when Task() detected
  - [ ] Export task_id to environment for subagent use

- [ ] **Phase 5: Update PostToolUse**
  - [ ] Extract real task_id from tool_response
  - [ ] Update mapping with actual task_id value

- [ ] **Phase 6: Testing**
  - [ ] Write unit tests for task_id capture
  - [ ] Integration test: Task() → PostToolUse → verify mapping
  - [ ] Integration test: SubagentStop → verify parent linkage

- [ ] **Phase 7: Documentation**
  - [ ] Update hook architecture docs
  - [ ] Document task_id correlation strategy
  - [ ] Add API docs for task lookup endpoint (if needed)

---

## Files to Modify

| File | Lines | Change |
|------|-------|--------|
| `src/python/htmlgraph/hooks/posttooluse.py` | 39-65 | Add debug logging for Task() |
| `src/python/htmlgraph/hooks/event_tracker.py` | 534-618 | Add claude_task_id parameter |
| `src/python/htmlgraph/hooks/event_tracker.py` | 990-1103 | Extract and pass task_id |
| `src/python/htmlgraph/hooks/pretooluse.py` | 218-313 | Store mapping placeholder |
| `src/python/htmlgraph/db/schema.py` | 182-450 | Add claude_task_mappings table |
| `tests/hooks/test_task_id_correlation.py` | NEW | New test file |

---

## Success Criteria

When complete:

1. **task_id is captured** - Every Task() call stores mapping to claude_task_id
2. **Mapping is queryable** - Can lookup event_id given task_id
3. **Tests pass** - Unit and integration tests verify correlation
4. **No errors** - Graceful degradation if task_id not available
5. **Performance** - Minimal overhead (JSON context or single lookup table)

Claude Code should then be able to:
- Call our API: `GET /api/tasks/{task_id}` → get event_id
- Display in notification: "5 events logged for this task"
- Link to dashboard for full task details

---

## Risk Assessment

**Low Risk Changes:**
- Adding to context JSON (backward compatible)
- Debug logging
- New lookup table (separate from existing data)

**Medium Risk:**
- Modifying event_tracker.py signature (need to handle all call sites)
- Adding columns to existing tables (requires migration)

**High Risk:**
- None identified

**Mitigation:**
- Start with context JSON (safest)
- Add comprehensive logging
- Test with real Claude Code Task() calls
- Rollback plan: Remove debug code, context is optional
