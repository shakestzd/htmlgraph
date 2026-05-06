# Bug Report: Task Delegation Event Hierarchy Not Nested Correctly

**Bug ID**: bug-event-hierarchy-201fcc67
**Date**: January 12, 2026
**Status**: IDENTIFIED AND DOCUMENTED
**Severity**: High (affects observability/tracking)
**Component**: Event tracking system, PreToolUse hook

---

## Summary

When using `Task()` to delegate work to Claude subagents (Haiku, Sonnet, Opus), the tool events executed within the subagent are recorded with incorrect parent_event_id values. They appear as **sibling events** of the Task delegation instead of **nested child events**.

---

## Current Behavior (INCORRECT)

```
UserQuery Event (uq-9ba303bf)
├─ Task Delegation (event-690e2e8e) → parent: uq-9ba303bf ✅
├─ Bash Tool (evt-5c2a6774) → parent: uq-9ba303bf ❌
├─ Bash Tool (evt-00df8df6) → parent: uq-9ba303bf ❌
├─ Read Tool (evt-5b5fdce1) → parent: uq-9ba303bf ❌
└─ Read Tool (evt-d62f7b17) → parent: uq-9ba303bf ❌
```

**Problem**: All tool events point to the UserQuery event, creating a flat structure. They should be children of the Task delegation event.

---

## Expected Behavior (CORRECT)

```
UserQuery Event (uq-9ba303bf)
└─ Task Delegation (event-690e2e8e) → parent: uq-9ba303bf ✅
   ├─ Bash Tool (evt-5c2a6774) → parent: event-690e2e8e ✅
   ├─ Bash Tool (evt-00df8df6) → parent: event-690e2e8e ✅
   ├─ Read Tool (evt-5b5fdce1) → parent: event-690e2e8e ✅
   └─ Read Tool (evt-d62f7b17) → parent: event-690e2e8e ✅
```

**Expected**: Tool events should point to their Task delegation parent, creating proper nesting.

---

## Evidence from Database

### Actual Query Results

```sql
SELECT type, event_id, tool_name, parent_event_id, status
FROM agent_events
WHERE created_at > datetime('now', '-2 hours')
ORDER BY created_at DESC LIMIT 20
```

**Results show**:
- ✅ Task events: `parent_event_id = event-query-XXXXX` (correct)
- ❌ Bash/Read tools: `parent_event_id = uq-XXXXX` (incorrect - points to UserQuery, not Task)
- ✅ Spawner subprocess events: `parent_event_id = event-XXXXX` (correct - points to Task)

### Comparison: Spawner Events Work Correctly

```
Spawner subprocess.copilot (event-33ff877a)
└─ parent_event_id: event-690e2e8e (Task) ✅ CORRECT

Spawner subprocess.gemini (event-c42164d6)
└─ parent_event_id: event-1b6dc531 (Task) ✅ CORRECT

Spawner subprocess.codex (event-444e0a25)
└─ parent_event_id: event-dfccf956 (Task) ✅ CORRECT
```

**Key Finding**: Spawner subprocess events ARE being recorded with correct parent_event_id pointing to their Task delegation event. Only regular tool events (Bash, Read, etc.) from within subagents have the bug.

---

## Root Cause Analysis

### Hook Flow

**Current (BUGGY)**:

```
1. User submits prompt
2. UserPromptSubmit hook runs
   → Creates UserQuery event (uq-9ba303bf)
   → Sets HTMLGRAPH_PARENT_EVENT=uq-9ba303bf ❌

3. PreToolUse hook (for Task tool)
   → Creates Task delegation event (event-690e2e8e)
   → parent_event_id = uq-9ba303bf ✅ (correct)
   → Sets HTMLGRAPH_PARENT_EVENT=event-690e2e8e for subagent ✅

4. Task() delegates to subagent
   → Subagent process inherits HTMLGRAPH_PARENT_EVENT=event-690e2e8e ✅

5. Subagent executes tools (Bash, Read, etc.)
   → PreToolUse hook runs in subagent
   → Reads HTMLGRAPH_PARENT_EVENT from environment
   → But sets parent_event_id to original UserQuery (uq-9ba303bf) ❌

6. Tool events recorded with wrong parent
   → parent_event_id = uq-9ba303bf (UserQuery) ❌
   → Should be = event-690e2e8e (Task) ✅
```

**Likely Cause**: PreToolUse hook may not be correctly reading the updated HTMLGRAPH_PARENT_EVENT from the subagent environment, or there's a default fallback to the original UserQuery event.

### Suspect Code Location

**File**: `packages/claude-plugin/.claude-plugin/hooks/scripts/pretooluse-integrator.py`

**Suspected Issue**:
```python
# Current (buggy) behavior might be:
parent_event_id = os.getenv("HTMLGRAPH_PARENT_EVENT", default_to_userquery)
# Instead of:
parent_event_id = os.getenv("HTMLGRAPH_PARENT_EVENT")
if not parent_event_id:
    # Query database for most recent Task delegation
    parent_event_id = query_most_recent_task_event()
```

The hook may have a hardcoded fallback to the original UserQuery instead of respecting the subagent's updated parent context.

---

## Impact Assessment

### Visual Impact (Dashboard)
- Events appear flat instead of hierarchical
- Task delegation breadth appears inflated (all tools shown at same level)
- Event depth/nesting not properly represented
- Makes it hard to understand subagent execution structure

### Observability Impact
- Can't tell which tools were executed by which Task() delegation
- Multiple Task() delegations create confusing event soup
- Parent-child relationships are lost
- Makes debugging and analysis harder

### Database Impact
- Event hierarchy is broken for subagent executions
- Can't reconstruct execution flow from parent_event_id
- Queries for "all events under this Task" won't work correctly

---

## Reproduction Steps

1. Execute `Task()` delegation with multiple tool uses:
```python
Task(
    subagent_type="general-purpose",
    prompt="Do this work"
)
# Subagent executes: Bash, Read, Bash, Edit, Bash
```

2. Query database:
```sql
SELECT event_id, tool_name, parent_event_id
FROM agent_events
WHERE created_at > datetime('now', '-10 minutes')
ORDER BY created_at
```

3. Observe:
- All tool events have `parent_event_id = uq-XXXXX` (UserQuery)
- Should have `parent_event_id = event-XXXXX` (Task)

---

## Workaround

**No workaround** - This is a system-level bug that requires hook changes.

Users can:
- Use spawners directly (they work correctly)
- Await fixes in next version
- Understand that event hierarchy may not be perfect in dashboard

---

## Fix Strategy

### Option 1: Fix PreToolUse Hook (Recommended)

Update `pretooluse-integrator.py`:

```python
# BEFORE (buggy)
parent_event_id = os.getenv("HTMLGRAPH_PARENT_EVENT")
if not parent_event_id:
    parent_event_id = query_most_recent_user_query()

# AFTER (correct)
parent_event_id = os.getenv("HTMLGRAPH_PARENT_EVENT")
if not parent_event_id:
    # Only query UserQuery if no parent in environment
    # This preserves subagent Task delegation context
    parent_event_id = query_most_recent_user_query()
```

The fix ensures:
1. Respect HTMLGRAPH_PARENT_EVENT set by PreToolUse hook
2. Only fallback to UserQuery if no parent context available
3. Subagent tools inherit Task delegation as parent

### Option 2: Fix PreToolUse Hook for Task Tool

Update PreToolUse hook when handling Task tool:

```python
# When Task tool is called:
1. Create Task delegation event
2. Set parent_event_id = current_parent_event_id
3. Export to subagent:
   os.environ["HTMLGRAPH_PARENT_EVENT"] = task_event_id  # NEW PARENT
4. Subagent PreToolUse reads this and uses it as parent
```

### Implementation Requirements

1. **Identify** where PreToolUse hook sets parent context for subagents
2. **Verify** that HTMLGRAPH_PARENT_EVENT is being updated correctly
3. **Test** that subagent tools get correct parent_event_id
4. **Validate** hierarchy in dashboard

---

## Testing Plan

### Unit Test
```python
def test_subagent_tool_parent_event_id():
    # Create UserQuery event
    user_query_id = create_user_query_event()

    # Create Task delegation event
    task_id = create_task_delegation_event(parent=user_query_id)

    # Set environment for subagent
    os.environ["HTMLGRAPH_PARENT_EVENT"] = task_id

    # Execute tool (simulating subagent)
    tool_event_id = execute_tool_with_tracking("bash", "echo test")

    # Verify: Tool event should have Task as parent
    assert get_parent_event_id(tool_event_id) == task_id
    assert get_parent_event_id(tool_event_id) != user_query_id
```

### Integration Test
```python
def test_task_delegation_hierarchy():
    # Delegate work to subagent
    sdk = SDK()

    # Record events before/after
    before_count = count_events()

    Task(
        subagent_type="general-purpose",
        prompt="Run 5 bash commands"
    )

    after_count = count_events()

    # Verify hierarchy
    task_event = get_most_recent_task_event()
    child_tools = get_children(task_event.id)

    assert len(child_tools) >= 5  # At least 5 bash commands
    for tool in child_tools:
        assert tool.parent_event_id == task_event.id  # All children
```

### Dashboard Visual Test
```python
def test_dashboard_event_hierarchy_visualization():
    # Execute Task with multiple tools
    # Open dashboard
    # Verify visual nesting:
    #   UserQuery
    #   └─ Task
    #      ├─ Tool 1
    #      ├─ Tool 2
    #      └─ Tool 3
    # Not:
    #   UserQuery
    #   ├─ Task
    #   ├─ Tool 1 (WRONG - at same level)
    #   ├─ Tool 2
    #   └─ Tool 3
```

---

## Related Issues

- **Spawner Events**: Already working correctly ✅
  - Subprocess events have correct parent_event_id
  - Shows that event tracking system CAN work correctly

- **Dashboard Rendering**: Shows events as provided
  - Not a dashboard bug, a data bug
  - Fix database event recording, not dashboard rendering

---

## Timeline

| Phase | Action | Status |
|-------|--------|--------|
| Identify | Root cause analysis | ✅ DONE |
| Document | Bug report creation | ✅ DONE |
| Investigate | Review PreToolUse hook | ⏳ TODO |
| Fix | Implement hook changes | ⏳ TODO |
| Test | Unit + integration tests | ⏳ TODO |
| Deploy | Release in v0.9.5 | ⏳ TODO |

---

## Key Insights

1. **Spawner tracking works correctly** - Shows event system can handle hierarchy
2. **Issue is environment variable handling** - HTMLGRAPH_PARENT_EVENT not being respected in subagents
3. **Affects all Task() delegations** - Not just specific command types
4. **Doesn't affect functionality** - Tracking happens, just with wrong hierarchy
5. **Dashboard shows what's in database** - Need to fix data, not visualization

---

## Files to Review

- `packages/claude-plugin/.claude-plugin/hooks/scripts/pretooluse-integrator.py` - Set parent context
- `packages/claude-plugin/.claude-plugin/hooks/scripts/user-prompt-submit.py` - Initial context setup
- `src/python/wipnote/hooks/context.py` - Environment variable handling
- `src/python/wipnote/db/schema.py` - Event recording logic

---

## Questions for Investigation

1. Does PreToolUse hook read HTMLGRAPH_PARENT_EVENT correctly in subagent?
2. Is the environment variable being passed correctly to subagent process?
3. Does the hook have a hardcoded fallback to UserQuery?
4. Should subagent sessions have their own session_id or inherit parent's?
5. How does event hierarchy work in Claude Code's own tracking system?

---

**Status**: READY FOR FIX
**Priority**: High (affects observability)
**Effort**: Medium (1-2 files to modify, comprehensive testing needed)

