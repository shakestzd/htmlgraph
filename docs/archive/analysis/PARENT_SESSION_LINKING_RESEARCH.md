# Wipnote Parent Session Linking Research

## Executive Summary

Wipnote has a **complete, working parent-child session linking system** using environment variables to connect Task() delegations across session boundaries. The system is implemented but **not fully integrated into the dashboard visualization**, which is why subagent events appear disconnected on the UI.

## What Works: The Environment Variable System

### 1. Environment Variables Used

Wipnote uses three environment variables to link parent and child sessions:

| Variable | Purpose | Set By | Read By |
|----------|---------|--------|---------|
| `HTMLGRAPH_PARENT_SESSION` | Parent session ID | session-start.py hook | SDK, task_enforcer, spawners |
| `HTMLGRAPH_PARENT_ACTIVITY` | Parent activity ID | task_enforcer | SDK.track_activity() |
| `HTMLGRAPH_NESTING_DEPTH` | Nesting level (0=root) | session-start, task_enforcer | task_enforcer |

### 2. Workflow: How Parent Sessions Are Set Up

**Step 1: Session Start Hook (session-start.py)**
```python
# Line 1250-1259 in packages/claude-plugin/hooks/scripts/session-start.py
if active and active.id:
    os.environ["HTMLGRAPH_PARENT_SESSION"] = active.id
    os.environ["HTMLGRAPH_PARENT_AGENT"] = "claude-code"
    os.environ["HTMLGRAPH_NESTING_DEPTH"] = "0"  # Root level

    # Export to shell environment
    print(f"export HTMLGRAPH_PARENT_SESSION={active.id}", file=sys.stderr)
```

**Result**: When Claude Code starts, HTMLGRAPH_PARENT_SESSION is set to the current session ID so all activities are tracked to that session.

**Step 2: Task Tool Call (task_enforcer.py)**
```python
# Line 130-170 in src/python/wipnote/hooks/task_enforcer.py
parent_session = os.environ.get("HTMLGRAPH_PARENT_SESSION")
if parent_session:
    # Create activity entry in parent session to track Task invocation
    sdk = SDK(agent=parent_agent, parent_session=parent_session)
    entry = sdk.track_activity(
        tool="Task",
        summary=f"Task invoked: {tool_params.get('description')}",
        payload={...}
    )
    if entry:
        task_activity_id = entry.id

    # Pass to child process
    os.environ["HTMLGRAPH_PARENT_ACTIVITY"] = task_activity_id
    os.environ["HTMLGRAPH_NESTING_DEPTH"] = str(nesting_depth + 1)
```

**Result**: When Task() is invoked, it:
1. Records the Task invocation as an activity in the PARENT session
2. Captures the activity ID (task_activity_id)
3. Passes it to child via HTMLGRAPH_PARENT_ACTIVITY environment variable
4. Increments nesting depth for recursion detection

**Step 3: Child Session Receives Parent Context**

```python
# Line 265 in src/python/wipnote/sdk.py
self._parent_session = parent_session or os.getenv("HTMLGRAPH_PARENT_SESSION")

# Line 901 in src/python/wipnote/sdk.py
if not parent_activity_id:
    parent_activity_id = os.getenv("HTMLGRAPH_PARENT_ACTIVITY")
```

**Result**: The child SDK automatically:
1. Reads HTMLGRAPH_PARENT_SESSION from environment
2. Sets `_parent_session` attribute
3. Routes all `track_activity()` calls to parent session instead of its own

**Step 4: Activity Routing (track_activity)**

```python
# Line 885-897 in src/python/wipnote/sdk.py
def track_activity(...):
    # Determine target session: explicit > parent > active
    if not session_id:
        # Use parent session if available (for nested contexts)
        if self._parent_session:
            session_id = self._parent_session
        else:
            # Fall back to active session
            active = self.session_manager.get_active_session(agent=self._agent_id)
            session_id = active.id
```

**Result**: All child activities go to PARENT session, creating the hierarchical linkage.

### 3. HTML Data Structure: Parent-Child Attributes

Sessions and activities store parent information in HTML data attributes:

```html
<!-- Session with parent metadata -->
<article id="sess-child"
         data-type="session"
         data-parent-session="sess-parent"
         data-nesting-depth="1">
    ...
</article>

<!-- Activity with parent link -->
<li data-ts="2026-01-06T08:48:04.493681"
    data-tool="Grep"
    data-event-id="evt-f94b7d95"
    data-feature="spk-05ddee15"
    data-parent="evt-4cf9978e">  <!-- Links to parent activity! -->
    Grep: HTMLGRAPH_PARENT_SESSION.*=|parent_session.*=
</li>
```

**Current session HTML shows this working**:
- `data-parent="evt-4cf9978e"` - Links activities to their parent activity ID
- Found in `/Users/shakes/DevProjects/htmlgraph/.wipnote/sessions/sess-529faa2c.html`
- Activities are properly linked to their parent activities

### 4. Test Coverage

Comprehensive tests exist to verify the system:

| Test File | Coverage |
|-----------|----------|
| `tests/python/test_sdk_parent_session.py` | SDK parent session handling (11 tests) |
| `tests/python/test_headless_spawner_parent_session.py` | Spawner parent context (10+ tests) |
| `tests/integration/test_post_compact_delegation.py` | Cross-session delegation |

**Test coverage includes**:
- Environment variable reading
- Parent session explicit parameter
- Activity tracking to parent session
- Nesting depth tracking
- Backward compatibility (no parent)
- Priority chain: explicit > parent > active session

## What's Missing: Dashboard Integration

### Current State: Events ARE Linked in HTML

Looking at `/Users/shakes/DevProjects/htmlgraph/.wipnote/sessions/sess-529faa2c.html`:

```html
<li data-parent="evt-4cf9978e">Grep: ...</li>
<li data-parent="evt-4cf9978e">Read: ...</li>
<li data-parent="evt-4cf9978e">Read: ...</li>
```

**The data is there!** Events have `data-parent` attributes linking them to parent activities.

### Problem: Dashboard Doesn't Visualize Parent-Child Relationships

The dashboard HTML/JavaScript:
1. **Loads session files** correctly
2. **Parses data attributes** from events
3. **Shows activities in activity log** in reverse chronological order
4. **BUT**: Does not visualize the parent-child hierarchy

**Missing features**:
1. **Indentation**: Child events should be indented under parent activity
2. **Grouping**: Group events by parent Task invocation
3. **Nesting visualization**: Show call stack with depth indicators
4. **Parent activity highlighting**: Show which Task() spawned this event
5. **Subagent badges**: Mark which events came from Task() delegations

### Example of What's Missing

Currently shows (flat list):
```
Task: Invoke subagent task
Grep: Search for pattern
Read: Read file
Edit: Update code
Task: Invoke subagent task
Bash: Run command
```

Should show (hierarchical):
```
Task: Invoke subagent task
  ├─ Grep: Search for pattern
  ├─ Read: Read file
  └─ Edit: Update code
Task: Invoke subagent task
  └─ Bash: Run command
```

## Implementation Status

### Fully Implemented (Phase 1)
- ✅ Environment variable setup (HTMLGRAPH_PARENT_SESSION, HTMLGRAPH_PARENT_ACTIVITY)
- ✅ Task enforcer hook tracks Task invocations
- ✅ SDK reads parent_session from environment
- ✅ Activity routing: child activities go to parent session
- ✅ HTML data attributes store parent-child relationships
- ✅ Nesting depth tracking for recursion detection
- ✅ Session model includes parent_session field
- ✅ Test coverage for all linking mechanisms

### Partially Implemented (Phase 2)
- ✅ Database schema includes parent_session_id column
- ⚠️ Activity model has parent_activity_id field
- ⚠️ Event log stores parent_activity_id
- ❌ Dashboard visualization of parent-child hierarchy (NOT IMPLEMENTED)

### Not Yet Started (Phase 3)
- ❌ FastAPI endpoint for querying parent-child relationships
- ❌ GraphQL schema for activity hierarchy
- ❌ Real-time parent-child visualization
- ❌ Subagent session grouping and filtering

## Code References

### Key Files Implementing Parent Session Linking

**SDK Initialization** (reads parent from environment):
- `/Users/shakes/DevProjects/htmlgraph/src/python/wipnote/sdk.py` - Line 265

**Activity Tracking** (routes to parent session):
- `/Users/shakes/DevProjects/htmlgraph/src/python/wipnote/sdk.py` - Lines 885-912

**Task Enforcer Hook** (sets up parent context):
- `/Users/shakes/DevProjects/htmlgraph/src/python/wipnote/hooks/task_enforcer.py` - Lines 101-217

**Session Start Hook** (initializes parent for root session):
- `/Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/hooks/scripts/session-start.py` - Lines 1248-1260

**Session Model** (stores parent metadata):
- `/Users/shakes/DevProjects/htmlgraph/src/python/wipnote/models.py` - Lines 974, 1443-1449

**Tests** (verify linking works):
- `/Users/shakes/DevProjects/htmlgraph/tests/python/test_sdk_parent_session.py` - 11 test cases
- `/Users/shakes/DevProjects/htmlgraph/tests/python/test_headless_spawner_parent_session.py` - 10+ test cases

## Why Subagent Events Aren't Visible on Dashboard

The parent-child linking works **at the data layer** but the **dashboard UI doesn't render the hierarchy**:

1. **Events ARE recorded in parent session** ✅
2. **Events HAVE data-parent attributes** ✅
3. **Activity IDs ARE linked** ✅
4. **Dashboard loads the data** ✅
5. **Dashboard IGNORES parent attributes** ❌

The dashboard currently:
- Displays activity log as flat list
- Doesn't filter by parent activity
- Doesn't show nesting depth visually
- Doesn't group Task() invocations with their child activities
- Doesn't distinguish subagent sessions from parent sessions

## What Would Need to Change

To make parent-child relationships visible on the dashboard:

**Phase 1 (Current)**: Store parent context ✅ DONE
**Phase 2 (Needed)**: Visualize parent context
- Modify dashboard.html to parse `data-parent` attributes
- Add CSS for indentation/grouping based on nesting depth
- Add JavaScript to render activity hierarchy
- Add filter toggles for "show subagent events", "group by Task"

**Phase 3 (Advanced)**: Interactive navigation
- Click parent Task to expand/collapse child activities
- View activity stack trace (parent → grandparent → root)
- Timeline view showing concurrent subagent tasks
- Cost attribution per Task delegation

## Summary Table

| Component | Status | Notes |
|-----------|--------|-------|
| Parent session env vars | ✅ Working | HTMLGRAPH_PARENT_SESSION, etc. |
| Task enforcer tracking | ✅ Working | Records Task invocations |
| SDK parent routing | ✅ Working | Activities go to parent |
| HTML data attributes | ✅ Present | data-parent, data-nesting-depth |
| Test coverage | ✅ Complete | 11+ integration tests |
| Dashboard visualization | ❌ Missing | Not implemented |
| Database queries | ✅ Partial | Schema ready, queries need UI |
| API endpoints | ⚠️ Partial | FastAPI phase not started |

## Conclusion

The parent session linking system is **architecturally complete and functionally working**. The reason subagent events don't appear linked on the dashboard is not because the system is broken—it's because the **dashboard UI layer doesn't visualize the hierarchy that already exists in the data**.

The data is there, stored correctly, linked via IDs and attributes. The next step is dashboard enhancement to make these parent-child relationships visible to users.
