# Subagent Event Attribution Bug - Exact Code Locations

## Quick Reference: Where the Bug Is

| Issue | File | Lines | Problem |
|-------|------|-------|---------|
| Missing env vars | `pretooluse-spawner-router.py` | 412-430 | Not setting `HTMLGRAPH_SUBAGENT_TYPE`, `HTMLGRAPH_PARENT_SESSION`, `HTMLGRAPH_PARENT_AGENT` |
| No subagent detection | `event_tracker.py` | 710-723 | Doesn't check for subagent environment before using global cache |
| Global cache bug | `event_tracker.py` | 711 | `manager.get_active_session()` uses shared `.wipnote/session.json` |

---

## Detailed Code Analysis

### Issue #1: PreToolUse Hook Not Setting Environment Variables

**File**: `packages/claude-plugin/.claude-plugin/hooks/scripts/pretooluse-spawner-router.py`

**Current Code (Lines 412-430)**:
```python
412 |        # Build environment with parent context
413 |        env = os.environ.copy()
414 |
415 |        # Set project root for spawner database access
416 |        project_root = os.environ.get("HTMLGRAPH_PROJECT_ROOT", os.getcwd())
417 |        env["HTMLGRAPH_PROJECT_ROOT"] = project_root
418 |
419 |        if parent_query_event_id:
420 |            env["HTMLGRAPH_PARENT_EVENT"] = parent_query_event_id
421 |            logger.info(
422 |                f"Passing parent query event to spawner: {parent_query_event_id}"
423 |            )
424 |
425 |        # Detect and pass model to spawner
426 |        if hook_input:
427 |            detected_model = detect_current_model(hook_input)
428 |            if detected_model:
429 |                env["HTMLGRAPH_MODEL"] = detected_model
430 |                logger.info(f"Passing model to spawner: {detected_model}")
```

**What's Missing** (After line 430):
```python
431 |        # ❌ MISSING CODE: Set subagent context
432 |        # Should add:
433 |        # env["HTMLGRAPH_SUBAGENT_TYPE"] = base_spawner_type
434 |        # env["HTMLGRAPH_PARENT_SESSION"] = current_session_id
435 |        # env["HTMLGRAPH_PARENT_AGENT"] = detected_agent
```

**What Should Be Added**:

Insert after line 430 (before `result = subprocess.run(...)`):

```python
        # Set subagent context for spawned process
        env["HTMLGRAPH_SUBAGENT_TYPE"] = base_spawner_type

        # Get current orchestrator session to link as parent
        try:
            from wipnote.session_manager import SessionManager
            from pathlib import Path

            graph_dir = Path(project_root) / ".wipnote"
            manager = SessionManager(graph_dir)
            current_session = manager.get_active_session()
            if current_session:
                env["HTMLGRAPH_PARENT_SESSION"] = current_session.id
                logger.info(
                    f"Set HTMLGRAPH_PARENT_SESSION={current_session.id} "
                    f"for subagent"
                )
        except Exception as e:
            logger.warning(f"Could not get parent session: {e}")

        # Set parent agent info
        if detected_agent:
            env["HTMLGRAPH_PARENT_AGENT"] = detected_agent
            logger.info(f"Set HTMLGRAPH_PARENT_AGENT={detected_agent}")
```

---

### Issue #2: Track Event Hook Not Detecting Subagent

**File**: `src/python/wipnote/hooks/event_tracker.py`

**Current Code (Lines 702-723)**:
```python
702 |    # Detect agent and model from environment
703 |    detected_agent, detected_model = detect_agent_from_environment()
704 |
705 |    # Also try to detect model from hook input (more specific than environment)
706 |    model_from_input = detect_model_from_hook_input(hook_input)
707 |    if model_from_input:
708 |        detected_model = model_from_input
709 |
710 |    # Get active session ID
711 |    active_session = manager.get_active_session()
712 |    if not active_session:
713 |        # No active Wipnote session yet; start one (stable internal id).
714 |        try:
715 |            active_session = manager.start_session(
716 |                session_id=None,
717 |                agent=detected_agent,
718 |                title=f"Session {datetime.now().strftime('%Y-%m-%d %H:%M')}",
719 |            )
720 |        except Exception:
721 |            return {"continue": True}
722 |
723 |    active_session_id = active_session.id
```

**The Bug**: Line 711 uses `manager.get_active_session()` which reads the global session cache without checking if this is a subagent.

**What Should Happen**:

Replace lines 710-723 with:

```python
710 |    # Check if this is a subagent spawned via Task()
711 |    is_subagent = os.environ.get("HTMLGRAPH_SUBAGENT_TYPE") is not None
712 |    parent_session_id = os.environ.get("HTMLGRAPH_PARENT_SESSION")
713 |
714 |    # Get or create session based on context
715 |    if is_subagent and parent_session_id:
716 |        # Subagent context: create new session linked to parent
716 |        try:
717 |            active_session = manager.start_session(
718 |                session_id=None,
719 |                agent=detected_agent,
720 |                is_subagent=True,
721 |                parent_session_id=parent_session_id,
722 |                title=f"{detected_agent} (subagent)",
723 |            )
724 |            print(
725 |                f"Debug: Created subagent session {active_session.id} "
726 |                f"for parent {parent_session_id}",
726 |                file=sys.stderr
727 |            )
728 |        except Exception as e:
729 |            print(
730 |                f"Debug: Failed to create subagent session: {e}",
731 |                file=sys.stderr
732 |            )
733 |            # Fallback: create standalone subagent session
734 |            try:
735 |                active_session = manager.start_session(
736 |                    session_id=None,
737 |                    agent=detected_agent,
738 |                    is_subagent=True,
739 |                    title=f"{detected_agent} (subagent - fallback)",
740 |                )
741 |            except Exception:
742 |                return {"continue": True}
743 |    else:
744 |        # Normal orchestrator flow: get or create normal session
744 |        active_session = manager.get_active_session()
745 |        if not active_session:
746 |            # No active Wipnote session yet; start one
747 |            try:
748 |                active_session = manager.start_session(
749 |                    session_id=None,
750 |                    agent=detected_agent,
751 |                    title=f"Session {datetime.now().strftime('%Y-%m-%d %H:%M')}",
752 |                )
753 |            except Exception:
754 |                return {"continue": True}
755 |
756 |    active_session_id = active_session.id
```

---

### Issue #3: Need to Add os import for environment variables

**File**: `src/python/wipnote/hooks/event_tracker.py`

**Current Imports (Top of file)**:
```python
20 | import json
21 | import os            # ← Already imported ✓
22 | import re
23 | import subprocess
```

**Status**: ✓ Already imported, no changes needed

---

## Supporting Changes

### Addition to context.py - Documentation Comment

**File**: `src/python/wipnote/hooks/context.py`

**Location**: Around line 98-108 where `session_id` is extracted

**Add Comment**:
```python
98  |        # Extract session ID with multiple fallbacks
99  |        # Priority order:
100 |        # 1. hook_input["session_id"] (if Claude Code passes it)
101 |        # 2. hook_input["sessionId"] (camelCase variant)
102 |        # 3. HTMLGRAPH_SESSION_ID environment variable
103 |        # 4. CLAUDE_SESSION_ID environment variable
104 |        # 5. "unknown" as last resort
105 |        #
106 |        # IMPORTANT: This is extracted but NOT used in track_event.py
107 |        # for session ID selection. Here's why:
108 |        #
109 |        # The global .wipnote/session.json is shared across ALL Claude
110 |        # Code windows (for multi-window support). If we used it to look
111 |        # up sessions in track_event(), subagents would incorrectly reuse
112 |        # the parent's session instead of creating a new one.
113 |        #
114 |        # Solution: track_event() checks HTMLGRAPH_SUBAGENT_TYPE env
115 |        # variable. If set, it creates a NEW session with parent_session_id
116 |        # link. Otherwise, it uses the normal flow.
117 |        #
118 |        # This prevents cross-window event contamination while allowing
119 |        # parent-child session hierarchies for subagent spawning.
```

---

## Testing Locations

### Unit Test Template

**File to Create**: `tests/python/test_subagent_session_creation.py`

```python
import os
import pytest
from pathlib import Path
from unittest.mock import patch, MagicMock

def test_track_event_detects_subagent_from_environment():
    """Test that track_event detects subagent context from environment."""
    # Setup
    os.environ["HTMLGRAPH_SUBAGENT_TYPE"] = "gemini"
    os.environ["HTMLGRAPH_PARENT_SESSION"] = "session-parent-123"

    # Call track_event
    hook_input = {
        "tool_name": "Read",
        "tool_input": {"file_path": "test.py"},
        "tool_response": {"content": "file contents"}
    }
    response = track_event("PostToolUse", hook_input)

    # Verify
    assert response["continue"] is True

    # Check database for subagent session
    manager = SessionManager(".wipnote")
    subagent_sessions = [s for s in manager._list_active_sessions()
                        if s.is_subagent]
    assert len(subagent_sessions) > 0

    # Verify parent linkage
    assert subagent_sessions[0].parent_session == "session-parent-123"

    # Cleanup
    del os.environ["HTMLGRAPH_SUBAGENT_TYPE"]
    del os.environ["HTMLGRAPH_PARENT_SESSION"]


def test_pretooluse_sets_subagent_environment():
    """Test PreToolUse hook sets subagent environment variables."""
    # This would require mocking the spawner execution
    # Or creating an integration test that runs the actual spawner
    pass
```

### Integration Test Location

**File**: `tests/integration/test_orchestrator_spawner_delegation.py`

**Add Test Class**:
```python
class TestSubagentSessionAttribution:
    """Test that subagent events are attributed to subagent session."""

    def test_orchestrator_to_gemini_session_attribution(self, temp_wipnote_dir):
        """End-to-end: Orchestrator → Gemini → session attribution."""
        # Test implementation here
        pass
```

---

## Database Schema Verification

**File**: `src/python/wipnote/db/schema.py`

**Verify These Columns Exist**:

```python
# Sessions table should have:
session_id: TEXT PRIMARY KEY
agent_assigned: TEXT
is_subagent: INTEGER (0 or 1)
parent_session_id: TEXT (FK to sessions.session_id)
created_at: TIMESTAMP
status: TEXT (active/closed/stale)

# Agent events table should have:
event_id: TEXT PRIMARY KEY
session_id: TEXT (FK to sessions.session_id)
agent_id: TEXT
tool_name: TEXT
model: TEXT
parent_event_id: TEXT (FK to agent_events.event_id)
created_at: TIMESTAMP
```

**Verify Foreign Keys**:
```python
# In sessions table:
FOREIGN KEY (parent_session_id) REFERENCES sessions(session_id)

# In agent_events table:
FOREIGN KEY (session_id) REFERENCES sessions(session_id)
FOREIGN KEY (parent_event_id) REFERENCES agent_events(event_id)
```

---

## Migration Path (If Schema Updates Needed)

If columns are missing, add migration:

**File to Create**: `src/python/wipnote/db/migrations/add_subagent_columns.py`

```python
def migrate_up(db_connection):
    """Add subagent support columns if they don't exist."""
    cursor = db_connection.cursor()

    # Check if is_subagent column exists
    cursor.execute(
        "PRAGMA table_info(sessions)"
    )
    columns = [row[1] for row in cursor.fetchall()]

    if "is_subagent" not in columns:
        cursor.execute(
            "ALTER TABLE sessions ADD COLUMN is_subagent INTEGER DEFAULT 0"
        )

    if "parent_session_id" not in columns:
        cursor.execute(
            "ALTER TABLE sessions ADD COLUMN parent_session_id TEXT"
        )
        cursor.execute(
            "ALTER TABLE sessions ADD FOREIGN KEY (parent_session_id) "
            "REFERENCES sessions(session_id)"
        )

    db_connection.commit()
```

---

## Environment Variable Reference

### Set by PreToolUse Hook (pretooluse-spawner-router.py)

| Variable | Set At Line | Value | Purpose |
|----------|-------------|-------|---------|
| `HTMLGRAPH_SUBAGENT_TYPE` | ~433 | `"gemini"`, `"codex"`, `"copilot"` | Signals to track_event() that this is a subagent |
| `HTMLGRAPH_PARENT_SESSION` | ~434 | Parent session ID from `manager.get_active_session().id` | Links subagent session to parent |
| `HTMLGRAPH_PARENT_AGENT` | ~436 | Parent agent name (e.g., `"orchestrator"`) | For audit trail |
| `HTMLGRAPH_PARENT_EVENT` | ~420 | Event ID from database query | Links subagent events to Task() event |
| `HTMLGRAPH_PROJECT_ROOT` | ~417 | Project root directory | For database path resolution |
| `HTMLGRAPH_MODEL` | ~429 | Parent model (e.g., `"claude-sonnet"`) | For status line cache |

### Read by Track Event Hook (event_tracker.py)

| Variable | Read At Line | Used For |
|----------|------------|----------|
| `HTMLGRAPH_SUBAGENT_TYPE` | ~711 | Detect if this is a subagent |
| `HTMLGRAPH_PARENT_SESSION` | ~712 | Get parent session ID |
| `HTMLGRAPH_PARENT_EVENT` | ~875 | Link subagent events to Task() |
| `HTMLGRAPH_MODEL` | ~382 | Detect model for event recording |
| `HTMLGRAPH_DISABLE_TRACKING` | ~24 | Skip tracking entirely |

---

## Deployment Checklist

### Code Changes
- [ ] Update `pretooluse-spawner-router.py` (Add env vars)
- [ ] Update `event_tracker.py` (Detect subagent)
- [ ] Update `context.py` (Add documentation comment)

### Testing
- [ ] Write unit tests for environment variable detection
- [ ] Write integration tests for orchestrator → subagent flow
- [ ] Run existing tests to verify no regressions
- [ ] Manual test with actual spawner

### Database
- [ ] Verify schema has all required columns
- [ ] Run migrations if needed
- [ ] Test foreign key constraints

### Documentation
- [ ] Update developer guide
- [ ] Add troubleshooting guide
- [ ] Document environment variables

---

## Quick Debug Commands

**Verify Subagent Session Created**:
```bash
sqlite3 .wipnote/index.sqlite \
  "SELECT session_id, agent_assigned, is_subagent, parent_session_id FROM sessions;"
```

**Check Event Attribution**:
```bash
sqlite3 .wipnote/index.sqlite \
  "SELECT session_id, tool_name, model, parent_event_id FROM agent_events LIMIT 10;"
```

**Verify Parent-Child Links**:
```bash
sqlite3 .wipnote/index.sqlite \
  "SELECT s.session_id, s.parent_session_id, COUNT(ae.event_id) as event_count
   FROM sessions s
   LEFT JOIN agent_events ae ON s.session_id = ae.session_id
   WHERE s.is_subagent = 1
   GROUP BY s.session_id;"
```

**Check Environment Variables (in subagent process)**:
```bash
# Add logging to verify env vars are passed
python3 -c "import os; print('HTMLGRAPH_SUBAGENT_TYPE:', os.environ.get('HTMLGRAPH_SUBAGENT_TYPE'))"
python3 -c "import os; print('HTMLGRAPH_PARENT_SESSION:', os.environ.get('HTMLGRAPH_PARENT_SESSION'))"
```
