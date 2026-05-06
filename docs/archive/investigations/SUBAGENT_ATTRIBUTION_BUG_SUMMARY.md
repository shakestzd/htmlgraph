# Subagent Event Attribution Bug - Executive Summary

## The Problem (One Sentence)

**When a subagent (e.g., Opus via Gemini spawner) runs tool calls, they're recorded to the parent orchestrator's session instead of creating a separate subagent session.**

---

## Root Cause (The Critical Bug)

The event tracking hook uses `manager.get_active_session()` which reads the **global session cache** (`.wipnote/session.json`), returning the **parent's session ID** instead of creating a new subagent session.

### Three Missing Pieces:

1. **PreToolUse Hook** doesn't pass subagent context to spawned process
   - Missing env vars: `HTMLGRAPH_SUBAGENT_TYPE`, `HTMLGRAPH_PARENT_SESSION`

2. **Track Event Hook** doesn't check for subagent environment
   - Should detect: `HTMLGRAPH_SUBAGENT_TYPE` and `HTMLGRAPH_PARENT_SESSION`
   - Should create: New session with `is_subagent=True` and `parent_session_id` link

3. **Session Manager** gets bypassed
   - Hook uses global cache instead of checking environment variables
   - No session separation between orchestrator and subagent

---

## Where to Fix

### Fix #1: PreToolUse Hook
**File**: `packages/claude-plugin/.claude-plugin/hooks/scripts/pretooluse-spawner-router.py` (line ~420)

```python
# Add before spawning subagent:
env["HTMLGRAPH_SUBAGENT_TYPE"] = base_spawner_type
env["HTMLGRAPH_PARENT_SESSION"] = current_session_id
env["HTMLGRAPH_PARENT_AGENT"] = detected_agent
```

### Fix #2: Track Event Hook
**File**: `src/python/wipnote/hooks/event_tracker.py` (line ~710)

```python
# Add BEFORE manager.get_active_session():
is_subagent = os.environ.get("HTMLGRAPH_SUBAGENT_TYPE") is not None
parent_session_id = os.environ.get("HTMLGRAPH_PARENT_SESSION")

if is_subagent and parent_session_id:
    # Create NEW subagent session
    active_session = manager.start_session(
        session_id=None,
        agent=detected_agent,
        is_subagent=True,
        parent_session_id=parent_session_id,
        title=f"{detected_agent} (subagent)"
    )
else:
    # Normal flow: get or create orchestrator session
    active_session = manager.get_active_session()
    if not active_session:
        active_session = manager.start_session(...)
```

### Fix #3: Documentation
**File**: `src/python/wipnote/hooks/context.py` (add comment)

Explain why `HTMLGRAPH_SESSION_ID` isn't used in `track_event()` - to avoid cross-contamination between orchestrator and subagent sessions.

---

## Why This Happened

The infrastructure for subagent session tracking exists:

✅ Environment variables defined: `HTMLGRAPH_PARENT_SESSION`, `HTMLGRAPH_PARENT_EVENT`
✅ Database schema has: `is_subagent`, `parent_session_id`, `parent_event_id`
✅ Session API supports: `is_subagent` parameter, `parent_session_id` parameter
✅ SpawnerEventTracker class exists for this purpose

**But** the event tracking hook never **uses** any of it. The hook skips the environment variable checks and goes straight to the global session cache.

---

## What Will Be Fixed

### Before (Current Broken State)
```
Orchestrator (Sonnet 4.5) Session: session-abc123
├── Task() call
│   └── Spawns Gemini subagent
├── Read → session-abc123 ❌ WRONG (should be subagent session)
├── Grep → session-abc123 ❌ WRONG (should be subagent session)
├── Edit → session-abc123 ❌ WRONG (should be subagent session)
```

### After (Fixed)
```
Orchestrator (Sonnet 4.5) Session: session-abc123
├── Task() event → event-task-001
│
Subagent (Gemini) Session: session-xyz789
├── Parent: session-abc123
├── Model: gemini-2.0-flash
├── Read → session-xyz789 ✅ Correct (parent_event_id: event-task-001)
├── Grep → session-xyz789 ✅ Correct (parent_event_id: event-task-001)
├── Edit → session-xyz789 ✅ Correct (parent_event_id: event-task-001)
```

---

## Impact Areas

| Component | Current Behavior | After Fix |
|-----------|------------------|-----------|
| Session separation | None - all in parent session | Separate subagent sessions created |
| Event attribution | Subagent events in parent session | Subagent events in subagent session |
| Model field | Shows parent model (Sonnet) | Shows correct subagent model (Opus/Gemini) |
| Parent-child linking | Task() has no children | Task() → subagent tool calls linked |
| Wipnote dashboard | Confusing mixed events | Clear separation of work |
| Cost analysis | Cannot distinguish | Can measure subagent costs separately |

---

## Testing Required

**Unit Tests**:
- Verify environment variables set by PreToolUse hook
- Verify subagent session created by track_event hook
- Verify parent-child links established

**Integration Tests**:
- End-to-end orchestrator → Gemini spawner → tool calls
- Verify correct session/model/parent-child attributes in database
- Verify Wipnote dashboard shows separation

**Manual Verification**:
```bash
# After running orchestrator → subagent workflow:
sqlite3 .wipnote/index.sqlite

# Should show:
# - Two sessions (parent + subagent)
# - Parent has is_subagent=0
# - Subagent has is_subagent=1 and parent_session_id set
# - Subagent events have parent_event_id set to Task() event

SELECT session_id, agent_assigned, is_subagent FROM sessions;
SELECT session_id, tool_name, model FROM agent_events ORDER BY session_id;
```

---

## Deployment Notes

- **Low risk**: Infrastructure already exists, just needs to be used
- **Backward compatible**: Normal orchestrator flow unchanged
- **Fallback**: If parent_session_id missing, create standalone subagent session
- **Monitoring**: Add logging to track session creation for debugging

---

## Next Steps

1. Read full investigation report: `SUBAGENT_ATTRIBUTION_BUG_INVESTIGATION.md`
2. Implement fixes following the checklist in the report
3. Run tests to verify fixes
4. Deploy to test environment
5. Monitor session creation and event attribution

---

## Questions?

See the full investigation report for:
- Detailed code analysis with line numbers
- Data flow diagrams
- Implementation checklist
- Testing strategy
- Success criteria
- Risk assessment
