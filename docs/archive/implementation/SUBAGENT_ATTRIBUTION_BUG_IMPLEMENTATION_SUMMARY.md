# Subagent Attribution Bug Fix - Implementation Summary

## Quick Status: ✅ COMPLETE & VERIFIED

The subagent event attribution bug has been **fully implemented, deployed, and verified**. Subagent tool calls are now correctly attributed to separate subagent sessions instead of being misattributed to the parent orchestrator's session.

---

## What Was Wrong

When a Task() delegation spawned a subagent (e.g., Gemini spawner), all tool calls (Read, Grep, Edit) were being recorded to the **parent orchestrator's session** instead of creating a **separate subagent session**.

**Example of broken behavior:**
```
Orchestrator Session: session-abc123
├── Task() call
├── Read (from Gemini) ❌ Wrong - attributed to orchestrator
├── Grep (from Gemini) ❌ Wrong - attributed to orchestrator
└── Edit (from Gemini) ❌ Wrong - attributed to orchestrator
```

**Root Cause**: The event tracking hook was calling `manager.get_active_session()`, which reads from a shared global cache (`.wipnote/session.json`) and returns the parent's session ID.

---

## What Was Fixed

Three coordinated fixes enable proper subagent session creation and event attribution:

### Fix #1: PreToolUse Hook Sets Environment Variables
**File**: `packages/claude-plugin/.claude-plugin/hooks/scripts/pretooluse-spawner-router.py`

When spawning a subagent, the hook now passes critical context via environment variables:

```python
env["HTMLGRAPH_SUBAGENT_TYPE"] = spawner_type       # "gemini", "codex", "copilot"
env["HTMLGRAPH_PARENT_SESSION"] = parent_session_id # Parent's session ID
env["HTMLGRAPH_PARENT_AGENT"] = parent_agent        # "orchestrator" or "claude-code"
```

**Result**: Subagent process has all information needed to identify itself as a subagent.

### Fix #2: Track Event Hook Detects Subagent Context
**File**: `src/python/wipnote/hooks/event_tracker.py`

When recording tool events, the hook now:

1. **Checks for subagent environment variables FIRST** (before using global session cache)
2. **Creates a separate subagent session** if environment indicates this is a subagent
3. **Links subagent session to parent** via `parent_session_id` parameter

```python
subagent_type = os.environ.get("HTMLGRAPH_SUBAGENT_TYPE")
parent_session_id = os.environ.get("HTMLGRAPH_PARENT_SESSION")

if subagent_type and parent_session_id:
    # Create new subagent session with parent link
    subagent_session_id = f"{parent_session_id}-{subagent_type}"
    active_session = manager.start_session(
        session_id=subagent_session_id,
        agent=f"{subagent_type}-spawner",
        is_subagent=True,
        parent_session_id=parent_session_id,
    )
```

**Result**: Subagent tool calls are recorded to a separate, properly-linked session.

### Fix #3: Documentation Explains the Design
**File**: `src/python/wipnote/hooks/context.py`

Added documentation explaining why the global session cache is hazardous and how database-based fallback works instead.

---

## After the Fix

**Example of correct behavior:**
```
Orchestrator (Sonnet) Session: session-abc123
├── Task() call → session-abc123

Subagent (Gemini) Session: session-abc123-gemini
├── Parent: session-abc123
├── Read → session-abc123-gemini ✅ Correct
├── Grep → session-abc123-gemini ✅ Correct
└── Edit → session-abc123-gemini ✅ Correct
```

**Database View:**
```sql
-- Sessions table
session-abc123      | orchestrator | is_subagent=0 | parent=NULL
session-abc123-gemini | gemini-spawner | is_subagent=1 | parent=session-abc123

-- Events table
event-task-001 | session-abc123 | Task | gemini-2.0-flash
event-read-001 | session-abc123-gemini | Read | gemini-2.0-flash | parent=event-task-001
event-grep-001 | session-abc123-gemini | Grep | gemini-2.0-flash | parent=event-task-001
```

---

## Key Implementation Details

### Session ID Determinism
Subagent session IDs follow a deterministic pattern:
```
{parent_session_id}-{subagent_type}
Example: session-abc123-gemini
```

**Benefits**:
- Multiple tool calls in same subagent reuse same session
- Human-readable (shows parent and type)
- Collision-proof (unique per parent + type)
- Reproducible across invocations

### Environment Variable Flow
```
Orchestrator has:
  HTMLGRAPH_PARENT_SESSION=session-abc123

PreToolUse hook adds:
  HTMLGRAPH_SUBAGENT_TYPE=gemini
  HTMLGRAPH_PARENT_SESSION=session-abc123
  HTMLGRAPH_PARENT_AGENT=orchestrator

Subagent inherits all three, track_event() detects them
```

### Session Creation is Idempotent
The first tool call in a subagent creates the session:
```python
existing = manager.session_converter.load(subagent_session_id)
if existing:
    active_session = existing  # Reuse if exists
else:
    active_session = manager.start_session(...)  # Create if new
```

Subsequent tool calls reuse the same session (no duplicates).

---

## Verification

### Database Queries to Verify Fix

**Check subagent sessions exist:**
```bash
sqlite3 .wipnote/wipnote.db "
SELECT session_id, agent_assigned, is_subagent, parent_session_id
FROM sessions
WHERE is_subagent = 1
LIMIT 5;
"
```

**Check subagent events are in correct session:**
```bash
sqlite3 .wipnote/wipnote.db "
SELECT session_id, tool_name, model
FROM agent_events
WHERE session_id LIKE '%-gemini' OR session_id LIKE '%-codex'
LIMIT 10;
"
```

**Check parent-child event links:**
```bash
sqlite3 .wipnote/wipnote.db "
SELECT e.event_id, e.parent_event_id, e.tool_name, e.session_id
FROM agent_events e
WHERE e.parent_event_id IS NOT NULL
LIMIT 5;
"
```

---

## Files Modified

| File | Change | Lines |
|------|--------|-------|
| `packages/claude-plugin/.claude-plugin/hooks/scripts/pretooluse-spawner-router.py` | Set environment variables | 432-446 |
| `src/python/wipnote/hooks/event_tracker.py` | Detect subagent context and create separate sessions | 714-756 |
| `src/python/wipnote/hooks/context.py` | Document session separation hazards | 107-122 |

---

## Deployment Timeline

| Version | Status | Changes |
|---------|--------|---------|
| v0.26.0-v0.26.3 | Before Fix | Bug present, subagent events misattributed |
| v0.26.4+ | ✅ FIXED | All three fixes implemented and deployed |

**Key Commits**:
- `4ee65ce` - Implement multi-AI orchestration spawner agents (foundation)
- `a0f5b55` - Activity Feed fixes + database-only parent-child linking
- `8aac640` - Implement proper 4-level event hierarchy for spawner agents
- `c142e0e` - Use session_id from hook_input in track_event() (latest)

---

## Impact

### Before Fix
- ❌ Cannot distinguish orchestrator work from subagent work
- ❌ All events show same model (parent's model)
- ❌ Cannot track subagent costs separately
- ❌ Dashboard shows confusing mixed events
- ❌ Hard to debug subagent execution

### After Fix
- ✅ Clear separation of orchestrator and subagent sessions
- ✅ Events show correct model (subagent's model)
- ✅ Can measure subagent costs separately
- ✅ Dashboard shows clear hierarchy
- ✅ Easy to find and debug subagent work

---

## Testing

### Manual Testing Steps

1. **Start orchestrator with Wipnote enabled**
   ```bash
   uv run wipnote claude --dev
   ```

2. **Run a Task() delegation to spawner**
   ```python
   Task(
       subagent_type="gemini",
       prompt="Analyze the codebase and find all API endpoints",
       description="Codebase analysis"
   )
   ```

3. **Check sessions created**
   ```bash
   sqlite3 .wipnote/wipnote.db "
   SELECT session_id, agent_assigned, is_subagent
   FROM sessions
   ORDER BY created_at DESC
   LIMIT 2;
   "
   ```

   **Expected Output:**
   ```
   session-xyz-gemini | gemini-spawner | 1
   session-xyz       | claude-code    | 0
   ```

4. **Check events in correct session**
   ```bash
   sqlite3 .wipnote/wipnote.db "
   SELECT session_id, tool_name, COUNT(*) as count
   FROM agent_events
   GROUP BY session_id, tool_name
   ORDER BY session_id;
   "
   ```

   **Expected Output:**
   ```
   session-xyz       | Task          | 1
   session-xyz-gemini | Read          | N
   session-xyz-gemini | Grep          | M
   session-xyz-gemini | Edit          | K
   ```

### Automated Tests

Tests verify:
- ✅ Subagent session created with correct parent
- ✅ Subagent events linked to subagent session (not parent)
- ✅ Model field shows subagent model
- ✅ Parent-child session linkage established
- ✅ Parent-child event linkage established

---

## Known Limitations & Mitigations

### Session ID Length
Subagent session IDs can be long: `session-abc123-gemini`
- **Mitigation**: Database handles arbitrary length strings

### Multiple Spawner Types
If orchestrator spawns multiple spawner types in parallel:
- **Each creates separate subagent session** (by design)
- **No collision** (session ID includes spawner type)

### Nested Subagents
If a subagent spawns another subagent:
- **Currently not supported** (environment variables would need hierarchical tracking)
- **Fallback**: Subagent would create grandchild session

---

## Success Criteria - All Met ✅

1. **Session Separation**: ✅ Orchestrator and subagent have separate session IDs
2. **Event Attribution**: ✅ Events recorded to correct session
3. **Parent-Child Links**: ✅ Sessions and events properly linked
4. **Model Accuracy**: ✅ Subagent model field shows correct model
5. **Dashboard Clarity**: ✅ Visual separation in UI

---

## Documentation References

For detailed information, see:
- **Root Cause Analysis**: `SUBAGENT_ATTRIBUTION_BUG_INVESTIGATION.md`
- **Executive Summary**: `SUBAGENT_ATTRIBUTION_BUG_SUMMARY.md`
- **Code Locations**: `SUBAGENT_ATTRIBUTION_BUG_CODE_LOCATIONS.md`
- **Flow Diagram**: `SUBAGENT_ATTRIBUTION_BUG_FLOW_DIAGRAM.md`
- **Verification Report**: `SUBAGENT_ATTRIBUTION_BUG_FIX_VERIFICATION.md`

---

## Conclusion

The subagent attribution bug has been successfully fixed through three coordinated changes:

1. **PreToolUse hook** passes subagent context via environment variables
2. **Track event hook** detects subagent context and creates separate sessions
3. **Documentation** explains the design and hazards

The implementation is:
- ✅ **Complete** - All three fixes implemented
- ✅ **Tested** - Automated and manual tests pass
- ✅ **Deployed** - Live in v0.26.4+
- ✅ **Verified** - Database checks confirm correct behavior

**No further action needed.**

---

**Implementation Date**: 2026-01-11
**Latest Update**: Verification Report Created
**Status**: COMPLETE ✅
