# Subagent Event Attribution Bug - Investigation Report

## Executive Summary

**Root Cause**: Subagent events (Read, Grep, Edit, etc.) are being recorded to the **parent orchestrator's session** instead of creating a **separate subagent session**. This is caused by a **session ID resolution conflict** in the event tracking hook.

**Impact Level**: HIGH - All subagent work is incorrectly attributed to parent agent

**Affected Components**:
- Event tracking hook: `track_event.py`
- Session context detection: `context.py`
- Session initialization: `session-start.py`

---

## Problem Statement

### Current Behavior
```
Orchestrator (Sonnet 4.5):
  ✅ Task() call recorded in session-abc123
  ❌ Read, Grep, Edit calls ALSO recorded in session-abc123
  ❌ Model shows "claude-sonnet" for ALL events
  ❌ Cannot distinguish orchestrator work from subagent work

Subagent (Opus):
  ❌ No session created
  ❌ Events attributed to parent session
  ❌ Model field shows parent model (Sonnet) instead of Opus
  ❌ Cannot distinguish subagent from orchestrator in Wipnote
```

### Expected Behavior
```
Orchestrator (Sonnet 4.5):
  ✅ Task() call recorded in session-abc123
  ✅ Model: claude-sonnet

Subagent (Opus):
  ✅ Task() spawns new session: session-xyz789
  ✅ Model: claude-opus
  ✅ Read, Grep, Edit in session-xyz789
  ✅ Parent-child relationship tracked: session-xyz789.parent_session_id = session-abc123
  ✅ Parent event linking: tool_events.parent_event_id = Task_event_id
```

---

## Root Cause Analysis

### 1. Session ID Detection Bug in `track_event.py`

**File**: `src/python/wipnote/hooks/event_tracker.py`
**Lines**: 711-723

```python
# Get active session ID
active_session = manager.get_active_session()
if not active_session:
    # No active Wipnote session yet; start one (stable internal id).
    try:
        active_session = manager.start_session(
            session_id=None,
            agent=detected_agent,
            title=f"Session {datetime.now().strftime('%Y-%m-%d %H:%M')}",
        )
    except Exception:
        return {"continue": True}

active_session_id = active_session.id
```

**Problem**: `manager.get_active_session()` returns the **globally cached active session** stored in `.wipnote/session.json`, which is shared across ALL Claude Code windows and agents.

**Why This Causes Misattribution**:
1. When orchestrator (Sonnet) starts, it creates `session-abc123` and stores it in `.wipnote/session.json`
2. When subagent (Opus) spawns, it calls `track_event()` hook for its first tool call
3. The hook calls `manager.get_active_session()` which returns `session-abc123` (the cached global session)
4. Subagent's Read/Grep/Edit events are recorded to `session-abc123` instead of creating a new subagent session
5. **No subagent session is ever created**

### 2. Environment Variable Not Checked for Subagent Detection

**File**: `src/python/wipnote/hooks/event_tracker.py`
**Lines**: 710-723

The hook **does not check** for environment variables that indicate a subagent context:
- `HTMLGRAPH_PARENT_SESSION` - Set by PreToolUse hook when Task() spawns subagent
- `HTMLGRAPH_SUBAGENT_TYPE` - Indicates this is a spawned subagent
- `HTMLGRAPH_PARENT_EVENT` - Parent Task() event ID for linking

**What Should Happen**:
```python
# BEFORE calling get_active_session(), check if this is a subagent
is_subagent = os.environ.get("HTMLGRAPH_SUBAGENT_TYPE") is not None
parent_session_id = os.environ.get("HTMLGRAPH_PARENT_SESSION")

if is_subagent and parent_session_id:
    # Create NEW subagent session, NOT reuse parent session
    active_session = manager.start_session(
        session_id=None,
        agent=detected_agent,
        is_subagent=True,
        parent_session_id=parent_session_id,
        title=f"Subagent {detected_agent} spawned by Task()"
    )
else:
    # Normal flow: get or create orchestrator session
    active_session = manager.get_active_session()
    if not active_session:
        active_session = manager.start_session(...)
```

### 3. Session Context Not Passed from PreToolUse Hook

**File**: `packages/claude-plugin/.claude-plugin/hooks/scripts/pretooluse-spawner-router.py`
**Lines**: 412-424

The spawner router **does set** `HTMLGRAPH_PARENT_EVENT`:
```python
if parent_query_event_id:
    env["HTMLGRAPH_PARENT_EVENT"] = parent_query_event_id
    logger.info(f"Passing parent query event to spawner: {parent_query_event_id}")
```

**But it DOES NOT set**:
- `HTMLGRAPH_SUBAGENT_TYPE` - Needed to signal "create new session"
- `HTMLGRAPH_PARENT_SESSION` - Needed to link parent-child sessions
- `HTMLGRAPH_SESSION_ID` - Needed to override global session cache

**Current Environment Setup** (Line 413-430):
```python
env = os.environ.copy()
env["HTMLGRAPH_PROJECT_ROOT"] = project_root

if parent_query_event_id:
    env["HTMLGRAPH_PARENT_EVENT"] = parent_query_event_id

if hook_input:
    detected_model = detect_current_model(hook_input)
    if detected_model:
        env["HTMLGRAPH_MODEL"] = detected_model
```

**Missing Context Variables**:
```python
# These should be added:
env["HTMLGRAPH_SUBAGENT_TYPE"] = base_spawner_type
env["HTMLGRAPH_PARENT_SESSION"] = current_session_id
env["HTMLGRAPH_PARENT_AGENT"] = detected_agent  # Parent orchestrator agent
```

### 4. Context Manager Not Using Environment Variables

**File**: `src/python/wipnote/hooks/context.py`
**Lines**: 111-126

The `HookContext.from_input()` method checks for `HTMLGRAPH_SESSION_ID` but **the event tracker doesn't use it**:

```python
session_id = (
    hook_input.get("session_id")
    or hook_input.get("sessionId")
    or os.environ.get("HTMLGRAPH_SESSION_ID")
    or os.environ.get("CLAUDE_SESSION_ID")
)
```

But in `track_event()`, the hook **ignores this** and uses the global `manager.get_active_session()` instead.

---

## Data Flow Analysis

### Current (Broken) Flow

```
┌─────────────────────────────────────────────────────┐
│ Orchestrator (Sonnet) - Window A                    │
│                                                     │
│ 1. Session Start Hook                              │
│    → Creates session-abc123                         │
│    → Stores in .wipnote/session.json              │
│                                                     │
│ 2. User Query                                       │
│    → UserQuery event in session-abc123              │
│                                                     │
│ 3. Task() call (spawns Opus)                       │
│    → Task event in session-abc123                  │
│    → PreToolUse hook spawns Opus subagent          │
│    → Sets HTMLGRAPH_PARENT_EVENT=event-123         │
│    ✗ Does NOT set HTMLGRAPH_PARENT_SESSION         │
│                                                     │
└─────────────────────────────────────────────────────┘
         │
         └──────────────────────────────┐
                                        ▼
┌─────────────────────────────────────────────────────┐
│ Subagent (Opus) - Spawned Process                   │
│                                                     │
│ 1. PostToolUse Hook (Read)                         │
│    → detect_agent_from_environment()                │
│    → Returns detected_agent="claude-opus"           │
│    ✓ Detects correct model!                         │
│                                                     │
│    BUT:                                             │
│    → manager.get_active_session()                  │
│    ✗ Reads .wipnote/session.json (global cache)  │
│    ✗ Returns session-abc123 (parent's session!)    │
│    ✗ No HTMLGRAPH_PARENT_SESSION env var set       │
│    ✗ No subagent session created                   │
│                                                     │
│ 2. Read event recorded to session-abc123            │
│    → Model: claude-opus (correct)                  │
│    → Session: session-abc123 (WRONG - should be    │
│              new subagent session)                 │
│    → parent_event_id: event-123 (correct)          │
│                                                     │
│ 3. Grep, Edit events same pattern                  │
│    All recorded to session-abc123                  │
│                                                     │
└─────────────────────────────────────────────────────┘
```

**Result**: All subagent events show in Wipnote under parent session, making them **indistinguishable from orchestrator events**.

---

## Solution Architecture

### Fix 1: PreToolUse Hook - Set Environment Variables

**File**: `packages/claude-plugin/.claude-plugin/hooks/scripts/pretooluse-spawner-router.py`

**Location**: `route_to_spawner()` function around line 412-430

**Changes**:
```python
# Get current session ID from environment or active session
current_session_id = os.environ.get("HTMLGRAPH_SESSION_ID", "")
if not current_session_id:
    try:
        from wipnote.session_manager import SessionManager
        manager = SessionManager(Path.cwd() / ".wipnote")
        active = manager.get_active_session()
        if active:
            current_session_id = active.id
    except Exception:
        pass

# Enhance environment with subagent context
env["HTMLGRAPH_SUBAGENT_TYPE"] = base_spawner_type
if current_session_id:
    env["HTMLGRAPH_PARENT_SESSION"] = current_session_id
env["HTMLGRAPH_PARENT_AGENT"] = detected_agent or "unknown"
```

### Fix 2: Track Event Hook - Check Subagent Environment

**File**: `src/python/wipnote/hooks/event_tracker.py`

**Location**: `track_event()` function around line 710-723

**Changes**:
```python
# Check if this is a subagent spawned via Task()
is_subagent = os.environ.get("HTMLGRAPH_SUBAGENT_TYPE") is not None
parent_session_id = os.environ.get("HTMLGRAPH_PARENT_SESSION")

if is_subagent and parent_session_id:
    # Subagent context: create new session linked to parent
    try:
        active_session = manager.start_session(
            session_id=None,
            agent=detected_agent,
            is_subagent=True,
            parent_session_id=parent_session_id,
            title=f"{detected_agent} (subagent)",
        )
        logger.info(
            f"Created subagent session {active_session.id} "
            f"for parent {parent_session_id}"
        )
    except Exception as e:
        logger.warning(f"Failed to create subagent session: {e}")
        # Fallback: create standalone session (not ideal, but continues execution)
        active_session = manager.start_session(
            session_id=None,
            agent=detected_agent,
            is_subagent=True,
            title=f"{detected_agent} (subagent - fallback)",
        )
else:
    # Normal orchestrator flow: get or create normal session
    active_session = manager.get_active_session()
    if not active_session:
        try:
            active_session = manager.start_session(
                session_id=None,
                agent=detected_agent,
                title=f"Session {datetime.now().strftime('%Y-%m-%d %H:%M')}",
            )
        except Exception:
            return {"continue": True}
```

### Fix 3: Document Session Separation

Update `context.py` to add comment explaining the hazard:

```python
# NOTE: This is intentionally NOT used in track_event.py because
# we must distinguish between:
# 1. Orchestrator sessions (use global cache via get_active_session)
# 2. Subagent sessions (create new session linked to parent)
#
# The hazard: If we used HTMLGRAPH_SESSION_ID in get_active_session(),
# subagents would incorrectly reuse parent session, causing event
# misattribution.
#
# Solution: track_event() checks HTMLGRAPH_SUBAGENT_TYPE environment
# variable BEFORE calling get_active_session(). If subagent, creates
# new session with parent_session_id link.
```

---

## Implementation Checklist

### Phase 1: Environment Variable Setup (PreToolUse Hook)

**File**: `packages/claude-plugin/.claude-plugin/hooks/scripts/pretooluse-spawner-router.py`

- [ ] Import SessionManager to read current session
- [ ] Extract current_session_id before spawning
- [ ] Add `HTMLGRAPH_SUBAGENT_TYPE` to env (value: spawner type)
- [ ] Add `HTMLGRAPH_PARENT_SESSION` to env (value: current session ID)
- [ ] Add `HTMLGRAPH_PARENT_AGENT` to env (value: parent agent)
- [ ] Add logging for environment setup
- [ ] Test: Verify env vars passed to spawner process

### Phase 2: Subagent Session Creation (Track Event Hook)

**File**: `src/python/wipnote/hooks/event_tracker.py`

- [ ] Add subagent detection logic (check HTMLGRAPH_SUBAGENT_TYPE)
- [ ] Add parent session ID retrieval (HTMLGRAPH_PARENT_SESSION)
- [ ] Create conditional: if subagent → create new session, else → normal flow
- [ ] Pass is_subagent=True and parent_session_id to start_session()
- [ ] Add logging for session creation (distinguish subagent vs normal)
- [ ] Add fallback for missing parent_session_id (create standalone subagent session)
- [ ] Test: Verify new subagent sessions created correctly

### Phase 3: Database Schema Verification

**File**: `src/python/wipnote/db/schema.py`

- [ ] Verify sessions table has parent_session_id column
- [ ] Verify sessions table has is_subagent column
- [ ] Verify agent_events table has parent_event_id column
- [ ] Verify foreign key constraints between sessions

### Phase 4: Testing

**Files**: `tests/integration/test_orchestrator_spawner_delegation.py`

- [ ] Add test: Verify subagent session created with correct parent
- [ ] Add test: Verify subagent events linked to subagent session, not parent
- [ ] Add test: Verify model field shows subagent model (not parent)
- [ ] Add test: Verify parent-child session link in database
- [ ] Add test: Verify parent-child event link (Task → subagent Read/Grep/Edit)
- [ ] Add test: Verify multi-level spawning (orchestrator → spawner → subagent)

---

## Testing Strategy

### Unit Tests

```python
# Test 1: PreToolUse hook sets environment variables
def test_pretooluse_hook_sets_subagent_env_vars():
    """Verify env vars set when spawning subagent."""
    env = {...}  # Initial environment

    # Simulate Task() call
    result = route_to_spawner("gemini", "prompt", manifest, ...)

    # Verify environment had subagent context set
    assert "HTMLGRAPH_SUBAGENT_TYPE" in env
    assert "HTMLGRAPH_PARENT_SESSION" in env
    assert "HTMLGRAPH_PARENT_AGENT" in env

# Test 2: Track event hook creates subagent session
def test_track_event_creates_subagent_session():
    """Verify new session created for subagent."""
    os.environ["HTMLGRAPH_SUBAGENT_TYPE"] = "gemini"
    os.environ["HTMLGRAPH_PARENT_SESSION"] = "session-abc123"

    hook_input = {"tool_name": "Read", "tool_input": {...}}
    response = track_event("PostToolUse", hook_input)

    # Verify subagent session created
    manager = SessionManager(".wipnote")
    subagent_sessions = [s for s in manager._list_active_sessions()
                         if s.is_subagent]
    assert len(subagent_sessions) > 0

# Test 3: Subagent events linked to correct session
def test_subagent_events_in_subagent_session():
    """Verify subagent events recorded to subagent session."""
    # Setup: Create parent session, spawn subagent
    parent_session = manager.start_session(agent="orchestrator")

    os.environ["HTMLGRAPH_SUBAGENT_TYPE"] = "gemini"
    os.environ["HTMLGRAPH_PARENT_SESSION"] = parent_session.id

    # Record subagent event
    hook_input = {"tool_name": "Read", "tool_input": {"file_path": "file.py"}}
    track_event("PostToolUse", hook_input)

    # Verify event in subagent session, NOT parent session
    db = WipnoteDB(".wipnote/index.sqlite")
    events = db.query_events(session_id=parent_session.id)
    read_events = [e for e in events if e["tool_name"] == "Read"]
    assert len(read_events) == 0  # Should NOT be in parent session

    # Verify event in NEW subagent session
    subagent_sessions = [s for s in manager._list_active_sessions()
                         if s.is_subagent and s.parent_session == parent_session.id]
    assert len(subagent_sessions) == 1
    subagent_session = subagent_sessions[0]
    events = db.query_events(session_id=subagent_session.id)
    read_events = [e for e in events if e["tool_name"] == "Read"]
    assert len(read_events) == 1  # Should be in subagent session
```

### Integration Tests

```python
# Test 4: Full orchestrator → subagent flow
def test_orchestrator_spawner_attribution_flow():
    """End-to-end test of event attribution through spawner delegation."""
    # Orchestrator starts
    orch_session = manager.start_session(agent="orchestrator")

    # Orchestrator issues Task()
    task_event = manager.track_activity(
        session_id=orch_session.id,
        tool="Task",
        summary="Task: analyze codebase"
    )

    # PreToolUse hook captures parent context
    parent_env = {
        "HTMLGRAPH_PARENT_SESSION": orch_session.id,
        "HTMLGRAPH_PARENT_EVENT": task_event.id,
        "HTMLGRAPH_PARENT_AGENT": "orchestrator"
    }

    # Subagent spawns (simulate subprocess with env)
    with patch.dict(os.environ, parent_env):
        with patch.dict(os.environ, {"HTMLGRAPH_SUBAGENT_TYPE": "gemini"}):
            # First hook: PostToolUse for Grep
            hook_input = {"tool_name": "Grep", "tool_input": {...}}
            track_event("PostToolUse", hook_input)

            # Verify subagent session created
            subagent_sessions = [s for s in manager._list_active_sessions()
                                if s.is_subagent]
            assert len(subagent_sessions) == 1
            subagent_session = subagent_sessions[0]

            # Verify parent linkage
            assert subagent_session.parent_session == orch_session.id

            # Verify Grep event in subagent session
            db = WipnoteDB(".wipnote/index.sqlite")
            events = db.query_events(session_id=subagent_session.id)
            grep_events = [e for e in events if e["tool_name"] == "Grep"]
            assert len(grep_events) == 1

            # Verify parent-child event link
            assert grep_events[0]["parent_event_id"] == task_event.id
```

---

## Deployment Plan

### Step 1: Code Changes
1. Update `pretooluse-spawner-router.py` to set environment variables
2. Update `event_tracker.py` to detect and handle subagent context
3. Add documentation comments in `context.py`

### Step 2: Testing
1. Run new unit and integration tests
2. Verify existing tests still pass
3. Manual testing with actual Gemini/Codex spawning

### Step 3: Verification
1. Deploy to test environment
2. Run orchestrator → Gemini spawner workflow
3. Inspect `.wipnote/` to verify:
   - Separate subagent session created
   - Events recorded to subagent session
   - Parent-child links established
4. Run dashboard queries to verify attribution

### Step 4: Monitoring
1. Add logging to track session creation
2. Monitor for any missed subagent contexts
3. Check database for orphaned sessions

---

## Success Criteria

After fix implementation, these conditions should all be true:

1. **Session Separation**: Orchestrator and subagent have separate session IDs
   ```sql
   SELECT session_id, agent_assigned, is_subagent FROM sessions;
   -- session-abc123 | orchestrator | 0
   -- session-xyz789 | gemini-2.0-flash | 1
   ```

2. **Event Attribution**: Events recorded to correct session
   ```sql
   SELECT session_id, tool_name, model FROM agent_events;
   -- session-abc123 | Task      | claude-sonnet
   -- session-xyz789 | Read      | gemini-2.0-flash
   -- session-xyz789 | Grep      | gemini-2.0-flash
   -- session-xyz789 | Edit      | gemini-2.0-flash
   ```

3. **Parent-Child Links**: Sessions and events properly linked
   ```sql
   SELECT id, parent_session FROM sessions WHERE is_subagent=1;
   -- session-xyz789 | session-abc123

   SELECT event_id, parent_event_id FROM agent_events
   WHERE session_id='session-xyz789' LIMIT 1;
   -- event-read-001 | event-task-001
   ```

4. **Model Accuracy**: Subagent model field shows correct model
   ```sql
   SELECT DISTINCT model FROM agent_events WHERE session_id='session-xyz789';
   -- gemini-2.0-flash (not claude-sonnet)
   ```

5. **Wipnote Dashboard**: Visual separation in UI
   - Orchestrator session shows only Task() calls
   - Subagent session shows Read, Grep, Edit, etc.
   - Parent-child relationship visible in session hierarchy

---

## Risk Assessment

### Low Risk
- Environment variable passing is already used for other context
- Session creation already supports `parent_session_id` parameter
- Database schema already has `is_subagent` and `parent_event_id` columns

### Medium Risk
- May break existing orchestrator workflows if not tested thoroughly
- Fallback logic needed for missing parent_session_id

### Mitigation
- Comprehensive unit + integration tests
- Graceful fallback (create standalone subagent session if parent missing)
- Feature flagging if needed (HTMLGRAPH_SUBAGENT_SESSIONS env var)
- Gradual rollout to test environment first

---

## Files to Modify

1. **`packages/claude-plugin/.claude-plugin/hooks/scripts/pretooluse-spawner-router.py`**
   - Add environment variable setup in `route_to_spawner()`
   - Extract current session ID before spawning

2. **`src/python/wipnote/hooks/event_tracker.py`**
   - Add subagent detection logic at start of `track_event()`
   - Conditional session creation based on subagent context
   - Add logging for debugging

3. **`src/python/wipnote/hooks/context.py`**
   - Add documentation comment about session separation hazard

4. **`tests/integration/test_orchestrator_spawner_delegation.py`**
   - Add subagent session creation tests
   - Add event attribution tests
   - Add parent-child link tests

---

## Related Issues

This bug explains why the project has:
- `HTMLGRAPH_PARENT_EVENT` env var (set but not fully utilized)
- `HTMLGRAPH_PARENT_SESSION` documentation (never actually used)
- `is_subagent` field in database (never set to True by hooks)
- `SpawnerEventTracker` class (intended for subagent tracking but hooks bypass it)

The fix will enable these existing systems to work as designed.
