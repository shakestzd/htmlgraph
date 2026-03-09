# Claude Code Task ID Correlation Investigation

**Issue:** Task notifications in Claude Code show "No child events" even though HtmlGraph database has the children properly linked via `parent_event_id`.

**Root Cause:** We're not linking Claude Code's `task_id` to our HtmlGraph `event_id` values. Claude Code's task system has its own task_id that's separate from our event tracking system.

---

## What Data is Available in Hooks

### PreToolUse Hook Input
The PreToolUse hook receives tool execution data but **does NOT include task_id**. Current available fields:

```python
{
    "name": "Task",              # Tool name
    "input": {...},              # Tool parameters
    "session_id": "sess-abc",    # Claude Code session (may not be present)
    "cwd": "/path/to/project"    # Current working directory
}
```

**Key Finding:** PreToolUse hooks do NOT receive `task_id` from Claude Code, even though the Task() tool creates a task_id internally.

### PostToolUse Hook Input
The PostToolUse hook receives tool execution results:

```python
{
    "name": "Task",                    # Tool name
    "tool_response": {                 # Result from Task()
        "task_id": "task-xyz123"?,    # UNKNOWN if present
        "status": "completed",
        "result": "...",
        ...
    },
    "session_id": "sess-abc",
    "cwd": "/path/to/project"
}
```

**Key Question:** Does `tool_response` contain task_id when Task() completes?

### SubagentStop Hook Input
This hook fires when a subagent (spawned via Task()) completes:

```python
{
    "session_id": "sess-xyz",     # Subagent session
    "parent_session_id": "sess-abc",  # Parent session
    "cwd": "/path/to/project",
    "status": "completed",        # Or "failed"
}
```

**Note:** SubagentStop likely doesn't have direct task_id either, but it knows the parent context.

---

## Current Implementation Analysis

### 1. **PreToolUse Hook** (`pretooluse.py`)
**Lines 218-313:** Creates task_delegation parent event

```python
def create_task_parent_event(db, tool_input, session_id, start_time):
    parent_event_id = f"evt-{str(uuid.uuid4())[:8]}"
    # ... creates agent_events record with event_type='task_delegation'
    return parent_event_id
```

**What's captured:**
- ✅ `parent_event_id` (our UUID-based ID)
- ✅ `subagent_type` (from tool_input)
- ✅ `tool_name = "Task"`
- ✅ `event_type = "task_delegation"`
- ❌ `claude_task_id` (NOT captured)

**Environment export (line 297):**
```python
os.environ["HTMLGRAPH_PARENT_EVENT"] = parent_event_id
```

This passes our event_id to subagent context, but NOT Claude Code's task_id.

### 2. **PostToolUse Hook** (`posttooluse.py`)
**Lines 39-65:** Event tracking with no task_id handling

```python
async def run_event_tracking(hook_type, hook_input):
    return await loop.run_in_executor(
        None,
        track_event,
        hook_type,
        hook_input,
    )
```

**No special handling for Task() results with task_id.**

### 3. **SubagentStop Hook** (`subagent_stop.py`)
**Lines 207-285:** Updates parent event on completion

```python
def handle_subagent_stop(hook_input):
    parent_event_id = get_parent_event_id()
    child_spike_count = count_child_spikes(db_path, parent_event_id, parent_start_time)
    update_parent_event(db_path, parent_event_id, child_spike_count)
```

**What's captured:**
- ✅ Parent event completion
- ✅ Child spike count
- ❌ Link to Claude Code's task_id

### 4. **Database Schema** (`schema.py`)
**Lines 205-235:** agent_events table definition

```sql
CREATE TABLE agent_events (
    event_id TEXT PRIMARY KEY,
    ...
    parent_event_id TEXT,
    status TEXT DEFAULT 'recorded',
    ...
    -- NO claude_task_id column exists
)
```

**No field for storing Claude Code's task_id.**

---

## Investigation Plan

### Step 1: Verify task_id is Available in PostToolUse
**Action:** Modify PostToolUse hook to log all received data

```python
# In posttooluse.py - run_event_tracking()
print(f"DEBUG: Full hook_input = {json.dumps(hook_input, indent=2)}", file=sys.stderr)

# Check if task_id exists
tool_response = hook_input.get("tool_response", {})
if isinstance(tool_response, dict):
    task_id = tool_response.get("task_id")
    print(f"DEBUG: task_id in tool_response = {task_id}", file=sys.stderr)
```

**Expected Output:**
- If task_id is present, we'll see it in stderr logs
- If NOT present, we know Claude Code doesn't expose it

### Step 2: Check Claude Code Hook Documentation
**Action:** Research Claude Code hooks docs to understand:
- What fields are available in PreToolUse, PostToolUse, SubagentStop
- Whether task_id is ever exposed
- How Claude Code tracks task parent-child relationships internally

**Resources to check:**
- https://code.claude.com/docs/en/hooks.md
- Claude Code GitHub issues/discussions
- Hook test files and examples

### Step 3: Design Storage Location
If task_id IS available, decide where to store it:

**Option A: New column in agent_events**
```sql
ALTER TABLE agent_events
ADD COLUMN claude_task_id TEXT;
```

**Option B: JSON context field** (non-breaking)
```python
context = {
    "claude_task_id": "task-xyz123",
    "file_paths": [...],
    ...
}
INSERT INTO agent_events (..., context JSON)
```

**Option C: New mapping table** (most flexible)
```sql
CREATE TABLE task_id_mappings (
    event_id TEXT PRIMARY KEY,
    claude_task_id TEXT,
    created_at DATETIME
);
```

---

## What We Know Works

### Current parent_event_id Linking (Working)
```
PreToolUse (Task() call)
  ↓
  Creates: agent_events record with event_type='task_delegation'
  ↓
  Exports: HTMLGRAPH_PARENT_EVENT env var
  ↓
SubagentStop (Task completes)
  ↓
  Updates: parent event's status='completed', child_spike_count=N
```

**Result:** HtmlGraph can show nested events and counts correctly.

**Problem:** Claude Code doesn't know about our event_id, so its task notification system can't link back to our events.

---

## What Needs to Change

### If task_id IS Available in PostToolUse:

**1. Capture task_id** (in PostToolUse hook)
```python
# In event_tracker.py - record_event_to_sqlite()
tool_response = hook_input.get("tool_response", {})
claude_task_id = tool_response.get("task_id") if isinstance(tool_response, dict) else None

# Store it
db.insert_event(
    ...,
    context={
        "claude_task_id": claude_task_id,
        ...
    }
)
```

**2. Create mapping table** (optional but recommended)
```sql
CREATE TABLE claude_task_mappings (
    event_id TEXT PRIMARY KEY,
    claude_task_id TEXT UNIQUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (event_id) REFERENCES agent_events(event_id)
);
```

**3. Update Task delegation parent** (in PreToolUse)
- After creating parent_event_id
- Store mapping so we can find event_id given task_id (for reverse lookups)

**4. Expose via API** (future)
- Create endpoint: `GET /api/tasks/{task_id}` → returns our event_id
- Or: `GET /api/events/{event_id}` → returns claude_task_id

### If task_id is NOT Available:

**Problem:** Claude Code's task system doesn't expose task_id to hooks.

**Workarounds:**
1. **Request feature** from Anthropic to expose task_id in PostToolUse/SubagentStop hooks
2. **Use approximate matching** based on timing (not reliable)
3. **Use session correlation** (Task creates new session, we track that)

---

## Effort Estimation

### If task_id IS Available:
- Add claude_task_id to context JSON: **< 1 hour**
- Create task_id_mappings table (optional): **< 1 hour**
- Update PreToolUse to capture when Task detected: **< 1 hour**
- Update PostToolUse to store mapping: **< 1 hour**
- Add API endpoints for task lookup: **2-3 hours**
- **Total:** ~2-4 hours for complete solution

### If task_id is NOT Available:
- Investigate Claude Code source/docs: **2-3 hours**
- Design workaround (timing-based or session-based): **2-3 hours**
- Implement and test: **2-3 hours**
- **Total:** ~6-9 hours, but may not fully solve the problem

---

## Next Steps

1. **Verify task_id availability** - Add debug logging to see what's in PostToolUse hook_input
2. **Check Claude Code documentation** - Understand how task_id is exposed (if at all)
3. **Design storage schema** - Decide on column vs JSON vs mapping table
4. **Implement capture** - Add task_id extraction in PreToolUse and PostToolUse hooks
5. **Create mapping table** - Build queryable index of event_id → task_id
6. **Test end-to-end** - Verify Claude Code can resolve task → events
7. **Document** - Update hook architecture docs with task_id correlation strategy

---

## Files to Modify

| File | Change | Priority |
|------|--------|----------|
| `src/python/htmlgraph/hooks/pretooluse.py` | Capture claude_task_id in Task() detection | High |
| `src/python/htmlgraph/hooks/posttooluse.py` | Extract and store task_id from response | High |
| `src/python/htmlgraph/db/schema.py` | Add task_id_mappings table (optional) | Medium |
| `src/python/htmlgraph/hooks/event_tracker.py` | Add task_id to context JSON | Medium |
| `tests/hooks/test_task_id_correlation.py` | New test file for task_id linking | High |
| `HOOK_TASK_ID_CORRELATION.md` | Architecture doc for solution | Medium |

---

## Questions for Investigation

1. **Does PostToolUse receive task_id in tool_response?**
   - Need to log actual hook_input to verify

2. **Does SubagentStop provide task_id context?**
   - Check Claude Code hook documentation

3. **Can Claude Code's task notification system query our API/database?**
   - Might need reverse lookup: task_id → event_id

4. **Do we need bidirectional mapping?**
   - event_id → task_id (we know our event, lookup Claude's task)
   - task_id → event_id (Claude's task, lookup our event)

---

## Success Criteria

When complete, users should see:

```
Claude Code Task Notification:
- Task ID: task-xyz123
- Status: Completed
- Child Events: 5 events
- View Details: [Link to our dashboard showing events]
```

Instead of current "No child events" message.
