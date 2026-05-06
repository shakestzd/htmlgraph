# Parent-Child Event Linking Fix

**Status:** ✅ Complete and Tested
**Date:** 2026-01-08
**Spike:** spk-bbddf263

## Problem

Subagent events didn't have `parent_event_id` set, preventing nested event tracing UI from working.

**Evidence:**
```sql
SELECT COUNT(*) as total,
       SUM(CASE WHEN parent_event_id IS NOT NULL THEN 1 ELSE 0 END) as with_parents
FROM agent_events
-- Before: 9 total events, 0 had parent_event_id
```

## Root Cause

Line 741 in `src/python/wipnote/hooks/event_tracker.py` was hardcoded to:
```python
parent_event_id=None,  # Parent linking handled after result
```

But the `parent_activity_id` variable was already being calculated correctly from:
1. `parent-activity.json` state file (same-process tracking)
2. `HTMLGRAPH_PARENT_EVENT` environment variable (cross-process tracking)

## Solution

### Changes Made

**File:** `src/python/wipnote/hooks/event_tracker.py`

**Change 1 (Lines 708-713):** Added environment variable fallback
```python
# Also check environment variable for cross-process parent linking
# This is set by PreToolUse hook when Task() spawns a subagent
if not parent_activity_id:
    env_parent = os.environ.get("HTMLGRAPH_PARENT_EVENT")
    if env_parent:
        parent_activity_id = env_parent
```

**Change 2 (Line 748):** Use actual parent_activity_id
```python
# BEFORE:
parent_event_id=None,  # Parent linking handled after result

# AFTER:
parent_event_id=parent_activity_id,  # Link to parent event
```

## Testing

### Integration Tests
```bash
pytest tests/python/test_parent_linking_integration.py -v
```
**Result:** 4/4 tests passed ✅

- ✅ Basic parent-child linking
- ✅ Multiple children per parent
- ✅ Three-level nesting (Task → Skill → Read)
- ✅ Recursive query support

### Verification Results

**Scenario 1: Task with multiple children**
```
Parent: evt-task-001 (Task)
├── evt-read-001 (Read)
├── evt-read-002 (Read)
├── evt-edit-001 (Edit)
└── evt-bash-001 (Bash)

Result: ✅ 4 children correctly linked
```

**Scenario 2: Nested delegation (3 levels)**
```
evt-task-002 (Task)
  └── evt-skill-001 (Skill)
        └── evt-read-nested (Read)

Result: ✅ 3-level hierarchy verified
```

**Scenario 3: Parallel work**
```
Parent: evt-task-003 (Task)
├── evt-edit-readme (Edit)
├── evt-edit-api (Edit)
└── evt-edit-guide (Edit)

Result: ✅ 3 parallel children linked
```

**Database Statistics:**
- Total events: 12
- Events with parent_event_id: 9 (75%)
- Root events: 3 (Task delegations)

## How It Works

### Flow: Task Delegation → Child Events

1. **Orchestrator calls Task()**
   - PreToolUse hook creates parent event in database
   - Sets `HTMLGRAPH_PARENT_EVENT` environment variable
   - Saves to `parent-activity.json` state file

2. **Subagent executes tools (Read, Edit, etc.)**
   - PostToolUse hook captures each tool call
   - Reads parent ID from environment or state file
   - Records event with `parent_event_id` set

3. **Database stores hierarchy**
   - Child events link to parent via `parent_event_id` foreign key
   - Dashboard can query and display nested structure

### Architecture

```
┌─────────────────────────────────────────────┐
│ PreToolUse Hook (for Task/Skill)           │
│  - Creates parent event in DB               │
│  - Sets HTMLGRAPH_PARENT_EVENT env var      │
│  - Saves to parent-activity.json            │
└─────────────────────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────────┐
│ Subagent Process                            │
│  - Inherits HTMLGRAPH_PARENT_EVENT          │
│  - Executes child tools (Read, Edit, etc.)  │
└─────────────────────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────────┐
│ PostToolUse Hook (for child events)         │
│  - Reads parent ID from env or state file   │
│  - Calls record_event_to_sqlite()           │
│  - Sets parent_event_id parameter ✅        │
└─────────────────────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────────┐
│ SQLite Database                             │
│  - Stores event with parent_event_id        │
│  - Foreign key to parent event              │
│  - Enables recursive queries                │
└─────────────────────────────────────────────┘
```

## Dashboard Impact

The fix enables nested event tracing in the dashboard:

**Before:**
```
Activity Feed (flat list):
- Task: Fix bug
- Read: src/auth.py
- Edit: src/auth.py
- Bash: pytest
```

**After:**
```
Activity Feed (nested tree):
▶ Task: Fix bug
  ├── Read: src/auth.py
  ├── Edit: src/auth.py
  └── Bash: pytest
```

### Supported Queries

```sql
-- Get all children of a Task
SELECT * FROM agent_events WHERE parent_event_id = 'evt-task-001';

-- Build event tree recursively
WITH RECURSIVE event_tree AS (
    SELECT event_id, tool_name, parent_event_id, 0 as depth
    FROM agent_events WHERE event_id = ?
    UNION ALL
    SELECT e.event_id, e.tool_name, e.parent_event_id, t.depth + 1
    FROM agent_events e
    JOIN event_tree t ON e.parent_event_id = t.event_id
)
SELECT * FROM event_tree ORDER BY depth;

-- Count children per parent
SELECT parent_event_id, COUNT(*) as child_count
FROM agent_events
WHERE parent_event_id IS NOT NULL
GROUP BY parent_event_id;
```

## Backward Compatibility

✅ **No breaking changes**
- Existing events with `parent_event_id=NULL` remain unchanged
- New events automatically get parent links
- Dashboard handles both NULL and non-NULL gracefully

## Performance

**Minimal impact:**
- One additional `os.environ.get()` call per event (O(1))
- No additional database queries
- No additional file I/O

## Files Modified

1. **src/python/wipnote/hooks/event_tracker.py** (2 changes)
   - Lines 708-713: Added environment variable fallback
   - Line 748: Fixed parent_event_id parameter

## Files Added

1. **tests/python/test_parent_linking_integration.py** (comprehensive integration tests)
2. **PARENT_CHILD_LINKING_FIX.md** (this document)

## Verification Commands

```bash
# Run integration tests
uv run pytest tests/python/test_parent_linking_integration.py -v

# Check database statistics
sqlite3 .wipnote/wipnote.db "
SELECT
    COUNT(*) as total,
    SUM(CASE WHEN parent_event_id IS NOT NULL THEN 1 ELSE 0 END) as with_parents
FROM agent_events"

# Show parent-child relationships
sqlite3 .wipnote/wipnote.db "
SELECT
    p.event_id as parent,
    p.tool_name,
    COUNT(c.event_id) as children
FROM agent_events p
LEFT JOIN agent_events c ON c.parent_event_id = p.event_id
WHERE p.parent_event_id IS NULL
GROUP BY p.event_id"
```

## Next Steps

1. ✅ Fix implemented and tested
2. ✅ Integration tests passing (4/4)
3. ✅ Verification complete
4. 🚀 Ready for production deployment
5. 📊 Dashboard can now display nested event structure

## References

- **Spike:** `.wipnote/spikes/spk-bbddf263.html`
- **Tests:** `tests/python/test_parent_linking_integration.py`
- **PreToolUse Hook:** `src/python/wipnote/hooks/pretooluse.py` (line 287)
- **SubagentStop Hook:** `src/python/wipnote/hooks/subagent_stop.py` (line 41)
- **Database Schema:** `src/python/wipnote/db/schema.py` (line 115, 124)
