# Dashboard Activity Feed Issues - Root Cause Analysis

## Issue 1: Stop Events Appearing Under Wrong User Queries

### Observed Behavior
Stop events with old timestamps (e.g., 01:40:40, 01:30:19, 01:14:52) appear nested under recent user queries like "yes deploy it" (03:24:15) and "yes, fix it" (03:31:11).

### Root Cause: Cross-Session Event Grouping Logic

**Location:** `src/python/htmlgraph/api/services.py` lines 108-134

The activity feed uses a cross-session lookup pattern to find orphaned events:

```python
first_level_children_sql = """
    SELECT ... FROM agent_events
    WHERE (
        parent_event_id = ?
        OR (
            parent_event_id IS NULL
            AND session_id LIKE ? || '%'      # ← PROBLEM HERE
            AND session_id != ?
            AND tool_name != 'UserQuery'
        )
    )
    ORDER BY timestamp DESC
"""
```

**The Bug:**
- When querying events for UserQuery in session `b92f0ebe-81f7-4e50-956f-fa8d0d775df9`
- The `LIKE 'b92f0ebe-81f7-4e50-956f-fa8d0d775df9%'` pattern matches:
  - ✅ Sub-sessions: `b92f0ebe-81f7-4e50-956f-fa8d0d775df9-htmlgraph:haiku-coder`
  - ❌ **Main session itself**: `b92f0ebe-81f7-4e50-956f-fa8d0d775df9`
  
**Why Stop Events Get Misplaced:**
1. Stop events are recorded with `parent_event_id = NULL` (they're session-level events)
2. Old Stop events from the same session (timestamps 01:32:08, 01:40:40, etc.) match the `LIKE` pattern
3. These orphaned Stop events get grouped under the **LATEST** UserQuery in that session
4. Result: Old Stop events from 01:40 appear under new UserQuery from 03:31

**Evidence from Database:**
```sql
-- Session b92f0ebe has 3 UserQuery events:
evt-836876ee  2026-02-13 01:35:41
evt-bcfa6063  2026-02-13 03:24:01
evt-d4cb3e88  2026-02-13 03:31:11  ← Latest query

-- Orphaned Stop events with no parent_event_id:
evt-e2f678ff  Stop  01:32:08  parent_event_id=NULL
evt-753299a9  Stop  01:32:27  parent_event_id=NULL
evt-327c9558  Stop  01:32:44  parent_event_id=NULL

-- These ALL get grouped under evt-d4cb3e88 (latest UserQuery)
```

### Fix: Add Temporal Proximity Filter

**Minimal Fix (lines 108-134 in services.py):**

```python
first_level_children_sql = """
    SELECT ... FROM agent_events
    WHERE (
        parent_event_id = ?
        OR (
            parent_event_id IS NULL
            AND session_id LIKE ? || '%'
            AND session_id != ?
            AND tool_name != 'UserQuery'
            AND timestamp >= ?  # ← ADD: Only events after UserQuery
        )
    )
    ORDER BY timestamp DESC
"""

# Update the query execution (line 162):
async with self.db.execute(
    first_level_children_sql,
    [uq_event_id, uq_session_id, uq_session_id, uq_timestamp],  # ← Add uq_timestamp
) as cur:
```

**Why This Works:**
- Only orphaned events **after** the UserQuery timestamp get grouped under it
- Stop events from 01:40 won't match UserQuery from 03:31 (01:40 < 03:31)
- Preserves sub-session matching for legitimate Task → subagent flows

---

## Issue 2: Duplicate Task Tool Calls

### Observed Behavior
Two identical "Task (htmlgraph:haiku-coder): Fix truthiness bug in schema.py" entries appear:
- One at 03:33:12 (status: recorded)
- One at 03:31:28 (status: completed)

### Root Cause: PreToolUse + PostToolUse Double Recording

**Evidence from Database:**
```sql
evt-262d1f2d  Task  03:33:12  status=recorded   parent=evt-d4cb3e88
evt-ce20f07b  Task  03:31:28  status=completed  parent=evt-d4cb3e88
```

**The Issue:**
Task tool calls are being recorded **twice**:
1. **PreToolUse hook** creates event with `status=recorded` (when task is initiated)
2. **PostToolUse hook** creates event with `status=completed` (when task finishes)

**Why This Happens:**
- PreToolUse hook (via `track_event`) records the Task call immediately
- PostToolUse hook (via `run_event_tracking` → `track_event`) records it again
- Both have the same `parent_event_id` (the UserQuery)
- Activity feed shows both as separate entries

**Expected Behavior:**
- PreToolUse should record with `status=pending` or `status=initiated`
- PostToolUse should **update** the existing event to `status=completed`
- OR: Only record in PostToolUse (skip PreToolUse for Task events)

### Fix: Update Instead of Insert in PostToolUse

**Location:** `src/python/htmlgraph/hooks/event_tracker.py`

**Option 1: Update Existing Event (Recommended)**
```python
def track_event(hook_type: str, hook_input: dict) -> dict:
    # ... existing code ...
    
    if hook_type == "PostToolUse":
        # Check if event already exists (from PreToolUse)
        cursor.execute("""
            SELECT event_id FROM agent_events 
            WHERE tool_name = ? 
            AND session_id = ? 
            AND input_summary = ?
            AND timestamp > datetime('now', '-5 minutes')
            ORDER BY timestamp DESC LIMIT 1
        """, [tool_name, session_id, input_summary])
        
        existing = cursor.fetchone()
        if existing:
            # Update existing event
            cursor.execute("""
                UPDATE agent_events 
                SET status = ?, 
                    execution_duration_seconds = ?,
                    output_summary = ?
                WHERE event_id = ?
            """, [status, duration, output_summary, existing[0]])
        else:
            # Insert new event (fallback)
            cursor.execute("INSERT INTO agent_events ...")
```

**Option 2: Skip PreToolUse for Task Events**
```python
def track_event(hook_type: str, hook_input: dict) -> dict:
    tool_name = hook_input.get("name", "") or hook_input.get("tool_name", "")
    
    # Skip PreToolUse for Task events (wait for PostToolUse)
    if hook_type == "PreToolUse" and tool_name == "Task":
        return {"continue": True}
    
    # ... rest of tracking logic ...
```

**Why Option 1 is Better:**
- Preserves both initiation and completion timestamps
- Allows tracking task duration accurately
- No data loss if PostToolUse fails

---

## Summary

### Issue 1: Wrong Stop Event Grouping
- **Root Cause:** Cross-session LIKE pattern matches main session's orphaned events
- **Impact:** Old events appear under new UserQuery prompts
- **Fix:** Add temporal filter (`timestamp >= ?`) to only group events after UserQuery

### Issue 2: Duplicate Task Events  
- **Root Cause:** PreToolUse and PostToolUse both INSERT instead of INSERT + UPDATE
- **Impact:** Same task appears twice with different statuses
- **Fix:** PostToolUse should UPDATE existing event instead of creating new one

### Testing Plan
1. **Issue 1:** Query events for session with multiple UserQuery events, verify temporal grouping
2. **Issue 2:** Create Task delegation, verify single event with updated status

### Files to Modify
1. `src/python/htmlgraph/api/services.py` - Add temporal filter (lines 108-165)
2. `src/python/htmlgraph/hooks/event_tracker.py` - Add update logic for PostToolUse
