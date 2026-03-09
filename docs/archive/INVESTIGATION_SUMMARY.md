# Task ID Correlation Investigation - Summary & Findings

## Problem Statement

Claude Code Task notifications show "No child events" even though HtmlGraph has properly logged all the child events with correct parent_event_id linking in the database.

**Root Cause Identified:** Claude Code's task system and HtmlGraph's event system are operating independently:
- Claude Code maintains `task_id` internally when Task() is called
- HtmlGraph generates our own `event_id` (UUID-based) for event tracking
- These two ID systems are never linked, so Claude Code can't map its task notifications to our events

---

## Investigation Findings

### 1. **Hook Data Flow Analysis**

#### PreToolUse Hook
When Task() is called, PreToolUse receives:
```python
{
    "name": "Task",
    "input": {
        "prompt": "...",
        "description": "...",
        "subagent_type": "general-purpose",
        "model": "haiku",
    },
    "session_id": "sess-abc123",
    "cwd": "/path/to/project"
}
```

**Key Finding:** ❌ **No `task_id` in PreToolUse**
- Claude Code hasn't yet generated the task_id
- The task_id is generated internally by Claude Code when Task() executes
- We have no way to capture it at this point

**What we DO capture:**
- ✅ Tool name ("Task")
- ✅ Tool input (prompt, model, subagent_type)
- ✅ Session ID

**What we create:**
- ✅ `parent_event_id` = `evt-abc123` (our UUID)
- ✅ Store in `agent_events` table as `event_type='task_delegation'`

#### PostToolUse Hook
After Task() completes, PostToolUse receives:
```python
{
    "name": "Task",
    "tool_response": {
        # Likely contains task execution result
        # ❓ UNKNOWN: Does it contain "task_id"?
    },
    "session_id": "sess-abc123"
}
```

**Key Finding:** ❓ **CRITICAL: Must verify if PostToolUse receives task_id in tool_response**

This is the most likely place where Claude Code would expose the task_id to us. We need debug logging to confirm.

#### SubagentStop Hook
When the subagent completes:
```python
{
    "session_id": "sess-xyz123",  # Subagent's session
    "parent_session_id": "sess-abc123",  # Parent session
    "status": "completed",
    "cwd": "/path/to/project"
}
```

**Key Finding:** ❌ **No `task_id` in SubagentStop**
- Only receives session IDs and status
- No direct reference to Claude Code's task_id

### 2. **Current HtmlGraph Implementation**

#### What's Working
```
PreToolUse (Task() call)
  ↓
  Creates: agent_events record
  - event_id = evt-abc123
  - event_type = task_delegation
  - subagent_type = (extracted from input)
  - parent_event_id = (UserQuery event)
  ↓
  Exports: HTMLGRAPH_PARENT_EVENT = evt-abc123
  ↓
Subagent executes (all tool calls)
  ↓
  Each tool call parents to evt-abc123
  ↓
SubagentStop (Task completes)
  ↓
  Updates: agent_events
  - status = completed
  - child_spike_count = N
```

**Result:** ✅ HtmlGraph can show complete task hierarchy with proper nesting

#### What's Missing
```
Claude Code Task()
  ↓
  task_id = task-xyz123 (internal to Claude Code)
  ↓
PostToolUse fires
  ↓
  ❌ We don't capture task_id
  ❌ No mapping created: evt-abc123 ↔ task-xyz123
  ↓
Claude Code's Task Notification System
  ↓
  Looks for "child events" associated with task_id
  ↓
  Finds: NOTHING (because we never linked our event_id to its task_id)
  ↓
  Shows: "No child events"
```

### 3. **Database Schema Analysis**

#### agent_events Table
Currently has fields:
- ✅ `event_id` - our UUID
- ✅ `parent_event_id` - for nesting
- ✅ `context` - JSON field for flexible storage
- ❌ No `claude_task_id` field
- ❌ No separate mapping to task_id

**Good News:** The `context` JSON field can store task_id without schema changes:
```json
{
  "file_paths": [...],
  "tool_input_keys": [...],
  "is_error": false,
  "claude_task_id": "task-xyz123"  ← Can add here
}
```

#### Existing Tables
- ✅ `agent_events` - stores all events
- ✅ `sessions` - tracks sessions
- ✅ `features` - work items
- ✅ `agent_collaboration` - delegation tracking
- ❌ No mapping table for task_id correlation

### 4. **Hook Implementation Review**

#### pretooluse.py (Lines 218-313)
**Function:** `create_task_parent_event()`

```python
# What we do:
parent_event_id = f"evt-{str(uuid.uuid4())[:8]}"  # Generate our UUID

# What we export:
os.environ["HTMLGRAPH_PARENT_EVENT"] = parent_event_id
os.environ["HTMLGRAPH_SUBAGENT_TYPE"] = subagent_type

# What we DON'T do:
# - Don't capture task_id (not available yet)
# - Don't export anything about Claude Code's task
# - No way for subagent to know the task_id
```

**Fix Required:** Once we capture task_id in PostToolUse, export it:
```python
os.environ["HTMLGRAPH_CLAUDE_TASK_ID"] = claude_task_id
```

#### posttooluse.py (Lines 39-65)
**Function:** `run_event_tracking()`

```python
# What we do:
await loop.run_in_executor(None, track_event, hook_type, hook_input)

# What we DON'T do:
# - Don't inspect tool_response for task_id
# - Don't extract task_id from response
# - No special handling for Task() results
```

**Fix Required:** Extract task_id before calling track_event:
```python
tool_response = hook_input.get("tool_response", {})
if isinstance(tool_response, dict):
    claude_task_id = tool_response.get("task_id")
```

#### subagent_stop.py (Lines 207-285)
**Function:** `handle_subagent_stop()`

```python
# What we do:
parent_event_id = get_parent_event_id()  # From environment
update_parent_event(db_path, parent_event_id, child_spike_count)

# What we could add:
# - If environment has HTMLGRAPH_CLAUDE_TASK_ID, update mapping table
```

**Fix Required:** Complete the task_id mapping on stop:
```python
claude_task_id = os.environ.get("HTMLGRAPH_CLAUDE_TASK_ID")
if claude_task_id:
    update_task_id_mapping(db_path, parent_event_id, claude_task_id)
```

---

## Data Capture Points

### Capture Opportunity #1: PreToolUse (Limited)
**When:** Task() call is detected
**Available:** Tool name, input parameters, session ID
**Not Available:** Claude Code's task_id (not generated yet)
**Action:** Create placeholder task_delegation event
**Priority:** DONE (already implemented)

### Capture Opportunity #2: PostToolUse (PRIMARY) ⭐
**When:** Task() completes
**Available:** Tool response (likely includes task_id)
**Not Available:** (TBD - need to verify)
**Action:** Extract task_id, update parent event mapping
**Priority:** CRITICAL - THIS IS THE KEY

### Capture Opportunity #3: SubagentStop (Supporting)
**When:** Subagent completes
**Available:** Session IDs, status
**Not Available:** task_id (not directly available)
**Action:** Complete mapping using parent context
**Priority:** HIGH - needed for cleanup

---

## Critical Question

**Q: Does PostToolUse hook receive `task_id` in `tool_response`?**

This is the deciding factor for the entire solution.

**If YES (task_id IS available):**
- ✅ Solution is straightforward (2-4 hours)
- ✅ Capture in PostToolUse, store mapping
- ✅ Update parent event with task_id
- ✅ Claude Code can then query our events by task_id

**If NO (task_id NOT available):**
- ❌ Need alternative approach
- ❌ Possible workarounds: session-based correlation, timing-based matching (unreliable)
- ❌ May need to request feature from Anthropic
- ❌ Solution effort: 6-9 hours + uncertainty

---

## Solution Architecture (Assuming task_id IS Available)

### Storage Strategy: Use Context JSON (Non-Breaking)
```python
# In agent_events.context (JSON field)
{
  "file_paths": [...],
  "tool_input_keys": [...],
  "claude_task_id": "task-xyz123",  ← Add here
  "claude_task_status": "completed"  ← Optional
}
```

**Why this approach:**
- ✅ Non-breaking change (context is flexible JSON)
- ✅ No schema migration needed
- ✅ Backward compatible
- ✅ Can query with JSON functions if needed

### Optional: Create Lookup Table (For Performance)
```sql
CREATE TABLE claude_task_mappings (
  event_id TEXT PRIMARY KEY,
  claude_task_id TEXT UNIQUE NOT NULL,
  tool_name TEXT,
  session_id TEXT,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (event_id) REFERENCES agent_events(event_id)
);
```

**Why optional:**
- Can query agent_events with `context ->> 'claude_task_id'`
- Table adds redundancy but improves query performance
- Recommended for future API endpoints

---

## Implementation Roadmap

### Phase 1: Verification (Critical First Step)
1. Add debug logging to PostToolUse hook
2. Run Claude Code with Task() call
3. Check stderr output for tool_response content
4. **Document findings in spike**

**Estimated Time:** 30 minutes
**Blocking:** Everything else

### Phase 2: Implement task_id Capture
1. Modify event_tracker.py to extract task_id
2. Add task_id to context JSON in agent_events
3. Export task_id to environment for subagent use
4. Update PreToolUse/PostToolUse/SubagentStop to handle task_id

**Estimated Time:** 2-3 hours
**Dependencies:** Phase 1 complete

### Phase 3: Create Mapping Table (Optional)
1. Add schema migration
2. Populate on Task() completion
3. Create query helper functions
4. Add indexes for performance

**Estimated Time:** 1 hour
**Dependencies:** Phase 2 complete

### Phase 4: Add Tests
1. Unit tests for task_id extraction
2. Integration tests for full flow
3. Test error cases (task_id missing, etc.)

**Estimated Time:** 1-2 hours
**Dependencies:** Phase 2 complete

### Phase 5: Documentation
1. Update hook architecture docs
2. Add task_id correlation to AGENTS.md
3. Document API endpoint (if created)

**Estimated Time:** 1 hour
**Dependencies:** Phase 4 complete

---

## Files Requiring Changes

| Component | File | Lines | Change | Priority |
|-----------|------|-------|--------|----------|
| **Debug** | `posttooluse.py` | 39-65 | Add task_id logging | CRITICAL |
| **Capture** | `event_tracker.py` | 534-618 | Accept claude_task_id param | HIGH |
| **Capture** | `event_tracker.py` | 990-1103 | Extract task_id from response | HIGH |
| **Capture** | `pretooluse.py` | 218-313 | Store mapping placeholder | HIGH |
| **Schema** | `schema.py` | 182+ | Add claude_task_mappings table | MEDIUM |
| **Tests** | `test_task_id_correlation.py` | NEW | New test file | HIGH |
| **Docs** | `TASK_ID_HOOKUP_PLAN.md` | - | Implementation guide | MEDIUM |

---

## Success Metrics

When implementation is complete:

1. **task_id is Captured** ✅
   - Every Task() creates entry in claude_task_mappings
   - Both event_id and task_id are stored

2. **Mapping is Accessible** ✅
   - Can query: `SELECT * FROM claude_task_mappings WHERE claude_task_id = ?`
   - Can reverse query: `SELECT * FROM agent_events WHERE context ->> 'claude_task_id' = ?`

3. **Tests Pass** ✅
   - Unit tests verify task_id extraction
   - Integration tests verify full flow
   - Error handling tested

4. **No Performance Regression** ✅
   - PostToolUse hook execution time unchanged
   - Query performance acceptable (< 10ms)

5. **Claude Code Integration Ready** ✅
   - API endpoint ready to serve task_id ↔ event_id lookups
   - Task notifications can reference our events
   - Dashboard shows task details correctly

---

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|-----------|
| task_id not in PostToolUse response | MEDIUM | HIGH | Phase 1 verification catches this early |
| Context JSON field too large | LOW | LOW | Monitor with metrics, add archiving if needed |
| Breaking existing queries | LOW | MEDIUM | Use JSON context instead of new column |
| Performance degradation | LOW | MEDIUM | Add indexes, use lookup table |
| Data consistency issues | LOW | MEDIUM | Add uniqueness constraint on task_id |

---

## Next Action Items

**IMMEDIATE (Do First):**
1. ✅ Read hook code (DONE)
2. ✅ Analyze data flow (DONE)
3. ✅ Review schema (DONE)
4. **→ Add debug logging to PostToolUse**
5. **→ Run Claude Code Task() and verify task_id presence**

**AFTER VERIFICATION:**
6. Design storage approach (context JSON vs table)
7. Implement task_id capture in all hooks
8. Create lookup table if needed
9. Write and run tests
10. Update documentation

---

## Appendix: Code Locations Reference

**PreToolUse Hook Logic:**
- File: `src/python/htmlgraph/hooks/pretooluse.py`
- Function: `create_task_parent_event()` (lines 218-313)
- Key export: `HTMLGRAPH_PARENT_EVENT` (line 297)

**PostToolUse Hook Logic:**
- File: `src/python/htmlgraph/hooks/posttooluse.py`
- Function: `run_event_tracking()` (lines 39-65)
- Delegates to: `track_event()` in `event_tracker.py`

**Event Tracking Logic:**
- File: `src/python/htmlgraph/hooks/event_tracker.py`
- Function: `track_event()` (lines 672-1211)
- Function: `record_event_to_sqlite()` (lines 534-618)
- Function: `get_parent_user_query()` (lines 120-156)

**SubagentStop Hook Logic:**
- File: `src/python/htmlgraph/hooks/subagent_stop.py`
- Function: `handle_subagent_stop()` (lines 207-285)
- Function: `update_parent_event()` (lines 108-176)

**Database Schema:**
- File: `src/python/htmlgraph/db/schema.py`
- Table: `agent_events` (lines 205-235)
- Table: `sessions` (lines 268-295)
- Method: `create_tables()` (lines 182-450)

**Hook Context:**
- File: `src/python/htmlgraph/hooks/context.py`
- Class: `HookContext` (lines 29-200)
- Method: `from_input()` (lines 54-200)

---

## Questions for Clarification

If moving forward, clarify:

1. **Should we expose task_id via API?**
   - Endpoint: `GET /api/tasks/{task_id}` → event_id
   - Or: `GET /api/events/{event_id}` → task_id

2. **Should we create a mapping table or use context JSON only?**
   - Table: Better for querying, adds overhead
   - JSON: Simpler, context is already stored

3. **How should we handle task_id not being available?**
   - Gracefully degrade: context JSON stores null/missing
   - Or: Return error if task_id required

4. **Do we need reverse lookup (task_id → event_id)?**
   - For Claude Code to find our events: YES
   - For dashboard: Can use event_id directly

5. **Should SubagentStop update task_id mapping?**
   - If task_id available in SubagentStop: YES
   - Otherwise: PostToolUse is sufficient
