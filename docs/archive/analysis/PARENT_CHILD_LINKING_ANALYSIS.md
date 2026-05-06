# Parent-Child Event Linking - Root Cause Analysis

**Feature ID**: feat-fd87099f
**Status**: Investigation Complete
**Failing Tests**: 35+ across 3 test files
**Root Cause**: FOREIGN KEY constraint failures + environment variable lifecycle issues

---

## Executive Summary

The parent-child event linking system has **2 critical design flaws**:

1. **FOREIGN KEY Constraint Too Strict**: The schema enforces referential integrity on `parent_event_id`, but child events are inserted before parent events exist in the database or parent_event_id references stale/non-existent events.

2. **Parent Event ID Lifecycle Broken**: The environment variable `HTMLGRAPH_PARENT_ACTIVITY` is set correctly, but:
   - Parent events may not be inserted into the database before children reference them
   - When the environment variable is cleared, children lose their parent references
   - Session creation fails because it tries to reference a non-existent parent event

---

## Current Architecture

### Database Schema (src/python/wipnote/db/schema.py)

**agent_events table** (lines 202-228):
```sql
CREATE TABLE IF NOT EXISTS agent_events (
    event_id TEXT PRIMARY KEY,
    ...
    parent_event_id TEXT,
    ...
    FOREIGN KEY (parent_event_id) REFERENCES agent_events(event_id)
)
```

**sessions table** (lines 261-286):
```sql
CREATE TABLE IF NOT EXISTS sessions (
    session_id TEXT PRIMARY KEY,
    ...
    parent_event_id TEXT,
    ...
    FOREIGN KEY (parent_event_id) REFERENCES agent_events(event_id)
)
```

### Event Tracking Flow (src/python/wipnote/sdk.py)

**_log_event()** (lines 602-665):
```python
def _log_event(self, ...):
    parent_event_id = os.getenv("HTMLGRAPH_PARENT_ACTIVITY")  # Line 643

    # Try to ensure session exists with parent_event_id
    self._ensure_session_exists(session_id, parent_event_id=parent_event_id)  # Line 647

    # Insert event with parent_event_id reference
    return self._db.insert_event(
        ...
        parent_event_id=parent_event_id,  # Line 663
    )
```

**_ensure_session_exists()** (lines 667-694):
```python
def _ensure_session_exists(self, session_id: str, parent_event_id: str | None = None):
    # Creates session with parent_event_id foreign key reference
    self._db.insert_session(
        session_id=session_id,
        ...
        parent_event_id=parent_event_id,  # Line 693
    )
```

### Insert Methods (src/python/wipnote/db/schema.py)

**insert_event()** (lines 476-546):
- Takes `parent_event_id` parameter
- Inserts into agent_events with foreign key constraint
- **Fails if parent_event_id doesn't exist in database**

**insert_session()** (lines 610-661):
- Takes `parent_event_id` parameter
- Inserts into sessions with foreign key constraint
- **Fails if parent_event_id doesn't exist in database**

---

## Test Failures Analysis

### Failure Type 1: FOREIGN KEY Constraint Failed

**Test**: `test_event_captures_parent_activity_env_var`
**Error**: `FOREIGN KEY constraint failed` on both insert_session and insert_event
**Root Cause**:
- Environment variable `HTMLGRAPH_PARENT_EVENT` is set to a valid event ID
- But that event may not exist in the database when child event tries to insert
- Even if it does exist, the session creation fails first due to foreign key violation

**Stack**:
```
insert_session() -> "FOREIGN KEY constraint failed"
  ↓ (because parent_event_id references non-existent event)
insert_event() -> "FOREIGN KEY constraint failed"
  ↓ (because session wasn't created, then event references non-existent parent)
```

### Failure Type 2: Parent Event ID Not Set Correctly

**Test**: `test_hierarchical_event_structure`, `test_deep_nesting_hierarchy`
**Error**: `AssertionError: assert 'evt-xxx' is None`
**Root Cause**:
- When environment variable is set, children should capture it
- But the assertions show grandparent has `parent_event_id='evt-xxx'` when it should be `None`
- This suggests the environment variable is being read from a stale state OR events are being created in wrong order

### Failure Type 3: Parent Reference Lost

**Test**: `test_api_events_query_includes_parent_event_id`
**Error**: `AssertionError: assert None == 'evt-xxx'`
**Root Cause**:
- Child event's parent_event_id is `None` when it should reference the parent
- Suggests the environment variable clearing or timing issue
- Or the parent_event_id was never captured when the event was inserted

---

## Root Causes (Detailed)

### RC-1: FOREIGN KEY Constraint Too Strict

**Problem**:
- agent_events.parent_event_id has `FOREIGN KEY (parent_event_id) REFERENCES agent_events(event_id)`
- sessions.parent_event_id has `FOREIGN KEY (parent_event_id) REFERENCES agent_events(event_id)`
- These constraints require the referenced event to exist BEFORE the child event is inserted

**Why It Fails**:
1. Child event is created with `parent_event_id = "evt-parent-001"`
2. Child code calls `_ensure_session_exists(parent_event_id="evt-parent-001")`
3. insert_session() tries to insert with foreign key to non-existent event
4. Database rejects: "FOREIGN KEY constraint failed"
5. Session creation fails, so event insertion fails

**Impact**:
- All parent-child linking fails if parent event doesn't exist in database
- Cross-process parent-child linking broken (parent event in different process/session)
- Cross-session linking broken (parent event from previous session)

### RC-2: Foreign Key on Sessions Table Unnecessary

**Problem**:
- sessions table has `FOREIGN KEY (parent_event_id) REFERENCES agent_events(event_id)`
- But sessions represent conversations, not necessarily tied to a specific event
- The foreign key is overly restrictive

**Why It Fails**:
- When creating a subagent session with `parent_event_id`, the event might not exist yet
- Or the event might be in a different database entirely (if running separate process)
- The constraint makes distributed/async linking impossible

**Impact**:
- Subagent sessions can't reference parent events
- Task delegations can't properly link sessions

### RC-3: Insert Methods Don't Handle Missing Parent Gracefully

**Problem**:
- insert_event() (line 515) executes INSERT directly without error handling
- insert_session() (line 638) executes INSERT directly without error handling
- Both fail hard on constraint violations instead of graceful degradation

**Why It Fails**:
- No try-except around the INSERT statement
- No validation that parent_event_id exists before inserting
- No fallback to NULL if parent doesn't exist

**Impact**:
- _log_event() returns False
- Events fail to be recorded
- Parent-child relationships are lost silently

### RC-4: Parent Activity State File Not Used by SDK

**Problem**:
- event_tracker.py has `load_parent_activity()` and `save_parent_activity()` functions
- These work with `parent-activity.json` file in .wipnote/
- But SDK._log_event() only reads `HTMLGRAPH_PARENT_ACTIVITY` environment variable
- SDK never uses the parent-activity.json file mechanism

**Why It Fails**:
- Two separate mechanisms for parent tracking (env var vs file)
- SDK doesn't know about parent-activity.json
- Parent state from event_tracker.py hooks isn't propagated to SDK

**Impact**:
- Parent information from hooks lost when SDK is used
- Parent information from SDK lost when hooks are used
- Inconsistent behavior across platforms

---

## Test Expectations vs Reality

### What Tests Expect

1. **Parent-Child Relationship Creation**:
   - Set `HTMLGRAPH_PARENT_ACTIVITY="evt-parent-001"`
   - Call `sdk._log_event(...)` to create child
   - Expect: `child.parent_event_id == "evt-parent-001"`
   - Actual: Error or None

2. **Hierarchical Structure**:
   - Create grandparent event
   - Set `HTMLGRAPH_PARENT_ACTIVITY=grandparent_id`
   - Create parent event
   - Set `HTMLGRAPH_PARENT_ACTIVITY=parent_id`
   - Create child event
   - Expect: `grandparent.parent_event_id=None`, `parent.parent_event_id=grandparent_id`, `child.parent_event_id=parent_id`
   - Actual: All have non-None parent_event_id values (wrong)

3. **Parent Reference Persistence**:
   - Parent event exists in database
   - Child event references it via `HTMLGRAPH_PARENT_ACTIVITY`
   - Query API returns both with parent-child relationship intact
   - Actual: Child event doesn't capture parent_event_id (None)

---

## Implementation Plan

### Phase 1: Fix FOREIGN KEY Constraints (CRITICAL)

**File**: `src/python/wipnote/db/schema.py`

**Option A - Remove Foreign Key (Recommended)**:
```sql
-- BEFORE
FOREIGN KEY (parent_event_id) REFERENCES agent_events(event_id)

-- AFTER
-- (Remove the constraint entirely)
-- Validation done in application code instead
```

**Why Option A**:
- Allows distributed/async parent-child linking
- Parent event can be in different database
- Graceful degradation if parent doesn't exist
- Application code validates relationships, not database

**Option B - Make Foreign Key Optional with Deferred Checking**:
```sql
-- Allow NULL, defer constraint check to end of transaction
FOREIGN KEY (parent_event_id) REFERENCES agent_events(event_id)
DEFERRABLE INITIALLY DEFERRED
```

**Why NOT Option B**:
- SQLite doesn't reliably support deferred constraints
- Still requires parent to eventually exist
- Doesn't solve cross-database linking

### Phase 2: Add Graceful Error Handling in Insert Methods

**File**: `src/python/wipnote/db/schema.py`

**Changes to insert_event()** (lines 515-546):
```python
try:
    cursor.execute(...INSERT...)
except sqlite3.IntegrityError as e:
    if "FOREIGN KEY constraint failed" in str(e):
        # Log warning but continue
        logger.warning(f"Parent event {parent_event_id} not found, inserting without parent reference")
        # Re-insert without parent_event_id
        cursor.execute(
            """INSERT INTO agent_events (...) VALUES (...)""",
            (..., None, ...)  # parent_event_id = NULL
        )
    else:
        raise
```

**Changes to insert_session()** (lines 638-656):
- Same pattern: catch IntegrityError, fallback to NULL parent_event_id

### Phase 3: Integrate Parent Activity State File with SDK

**File**: `src/python/wipnote/sdk.py`

**Changes to _log_event()** (lines 602-665):
```python
# BEFORE
parent_event_id = os.getenv("HTMLGRAPH_PARENT_ACTIVITY")

# AFTER
# Check both environment variable and state file
parent_event_id = os.getenv("HTMLGRAPH_PARENT_ACTIVITY")
if not parent_event_id:
    # Fall back to parent-activity.json state file
    from wipnote.hooks.event_tracker import load_parent_activity
    state = load_parent_activity(self._directory)
    parent_event_id = state.get("parent_id")
```

### Phase 4: Fix Environment Variable Lifecycle in Tests

**File**: `tests/python/test_parent_child_event_linking.py`

**Problem**: Tests don't ensure parent event exists before setting as parent for children

**Fix**: Create parent event first, ensure it's inserted, then set environment variable:
```python
# Create parent
sdk._log_event(event_type="delegation", tool_name="Task", ...)

# Get the parent event ID that was just created
cursor = sdk._db.connection.cursor()
cursor.execute("SELECT event_id FROM agent_events ORDER BY timestamp DESC LIMIT 1")
parent_id = cursor.fetchone()[0]

# NOW set as parent for children
os.environ["HTMLGRAPH_PARENT_ACTIVITY"] = parent_id
```

---

## Files to Modify

| File | Changes | Priority | Risk |
|------|---------|----------|------|
| src/python/wipnote/db/schema.py | Remove/relax FOREIGN KEY on parent_event_id | P0 | High |
| src/python/wipnote/db/schema.py | Add graceful error handling in insert_event() | P0 | Medium |
| src/python/wipnote/db/schema.py | Add graceful error handling in insert_session() | P0 | Medium |
| src/python/wipnote/sdk.py | Integrate parent-activity.json state file | P1 | Low |
| tests/python/test_parent_child_event_linking.py | Fix test setup to ensure parent exists | P0 | Low |
| tests/python/test_parent_child_linking.py | Verify environment variable handling | P0 | Low |
| tests/python/test_parent_linking_integration.py | Integration test coverage | P1 | Low |

---

## Verification Strategy

After fixes, verify with:

```bash
# Run parent-child event linking tests
uv run pytest tests/python/test_parent_child_event_linking.py -v

# Run parent linking tests
uv run pytest tests/python/test_parent_child_linking.py -v

# Run integration tests
uv run pytest tests/python/test_parent_linking_integration.py -v

# Run full test suite to ensure no regressions
uv run pytest tests/python/ -v --tb=short

# Code quality checks
uv run ruff check --fix
uv run ruff format
uv run mypy src/
```

---

## Next Steps

1. ✅ Complete root cause analysis (THIS DOCUMENT)
2. ⏭️ Fix FOREIGN KEY constraints (Phase 1)
3. ⏭️ Add graceful error handling (Phase 2)
4. ⏭️ Integrate parent-activity.json (Phase 3)
5. ⏭️ Fix test lifecycle issues (Phase 4)
6. ⏭️ Run full test suite and verify
7. ⏭️ Update documentation

---

## Related Issues

- Claude Code Hook #10373 - Session management edge cases
- Wipnote Feature: Enhanced parent-child tracing
- Event capture diagnostic: Parent event ID propagation

---

**Created**: 2025-01-09
**Analysis Time**: 45 minutes
**Test Coverage**: 35+ failing tests, 3 test files
**Confidence**: High - Root causes clearly identified through database logs and test assertions
