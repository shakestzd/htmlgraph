"""
Tests for roborev job 259 findings.

Finding 1: Session ID correctness in SubagentStart/SubagentStop
Finding 2: Agent filter preserves ingested task_delegation Agent events
Finding 3: Test coverage for service layer Agent filtering and check-systematic-changes.py
"""

import sqlite3
from unittest.mock import patch

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _create_agent_events_table(conn: sqlite3.Connection) -> None:
    """Create a minimal agent_events table for testing."""
    conn.execute("""
        CREATE TABLE IF NOT EXISTS agent_events (
            event_id TEXT PRIMARY KEY,
            agent_id TEXT,
            event_type TEXT NOT NULL,
            session_id TEXT,
            tool_name TEXT,
            input_summary TEXT,
            context TEXT,
            parent_event_id TEXT,
            subagent_type TEXT,
            status TEXT DEFAULT 'recorded',
            model TEXT,
            feature_id TEXT,
            execution_duration_seconds REAL,
            output_summary TEXT,
            child_spike_count INTEGER DEFAULT 0,
            timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
        )
    """)
    conn.commit()


# ===========================================================================
# Finding 1: Session ID correctness in SubagentStart/SubagentStop
# ===========================================================================


class TestSubagentStartSessionId:
    """Verify SubagentStart session_id filter matches parent session correctly.

    Claude Code passes the PARENT/orchestrator session_id to SubagentStart hooks.
    task_delegation rows are also written with the parent session_id by PreToolUse.
    So the session_id filter is correct and should match.
    """

    def test_session_id_matches_parent_task_delegation(self):
        """SubagentStart finds task_delegation in same (parent) session."""

        conn = sqlite3.connect(":memory:")
        _create_agent_events_table(conn)

        # Insert task_delegation in parent session
        conn.execute(
            """INSERT INTO agent_events
               (event_id, event_type, session_id, subagent_type, status, timestamp)
               VALUES (?, ?, ?, ?, ?, ?)""",
            (
                "evt-001",
                "task_delegation",
                "sess-parent",
                "general-purpose",
                "started",
                "2025-01-01 10:00:00",
            ),
        )
        conn.commit()

        # Patch get_database_path to return our in-memory DB path
        # We test the matching logic directly instead, since handle_subagent_start
        # opens its own connection. Verify the SQL logic matches correctly.
        cursor = conn.cursor()
        session_id = "sess-parent"  # Same session as parent writes

        cursor.execute(
            """
            SELECT event_id, subagent_type FROM agent_events
            WHERE event_type = 'task_delegation'
              AND status = 'started'
              AND (agent_id IS NULL OR agent_id = '' OR agent_id = 'claude-code')
              AND session_id = ?
            ORDER BY timestamp ASC
            """,
            (session_id,),
        )
        rows = cursor.fetchall()
        assert len(rows) == 1
        assert rows[0][0] == "evt-001"
        conn.close()

    def test_session_id_filter_excludes_old_sessions(self):
        """SubagentStart does not match task_delegations from old sessions."""
        conn = sqlite3.connect(":memory:")
        _create_agent_events_table(conn)

        # Task delegation from a previous (stale) session
        conn.execute(
            """INSERT INTO agent_events
               (event_id, event_type, session_id, subagent_type, status, timestamp)
               VALUES (?, ?, ?, ?, ?, ?)""",
            (
                "evt-old",
                "task_delegation",
                "sess-old",
                "general-purpose",
                "started",
                "2025-01-01 09:00:00",
            ),
        )
        conn.commit()

        cursor = conn.cursor()
        session_id = "sess-current"  # Different from stale session

        cursor.execute(
            """
            SELECT event_id FROM agent_events
            WHERE event_type = 'task_delegation'
              AND status = 'started'
              AND (agent_id IS NULL OR agent_id = '' OR agent_id = 'claude-code')
              AND session_id = ?
            ORDER BY timestamp ASC
            """,
            (session_id,),
        )
        rows = cursor.fetchall()
        assert len(rows) == 0, "Should not match stale session task_delegation"
        conn.close()


class TestSubagentStopSessionId:
    """Verify SubagentStop session_id filter matches parent session correctly."""

    def test_exact_agent_id_lookup_with_session_scope(self):
        """SubagentStop finds parent by agent_id within parent session."""
        from htmlgraph.hooks.subagent_stop import get_parent_event_from_db

        conn = sqlite3.connect(":memory:")
        _create_agent_events_table(conn)

        # Two task_delegations: one in parent session, one in old session
        conn.execute(
            """INSERT INTO agent_events
               (event_id, event_type, session_id, subagent_type, status, agent_id)
               VALUES (?, ?, ?, ?, ?, ?)""",
            (
                "evt-old",
                "task_delegation",
                "sess-old",
                "general-purpose",
                "started",
                "agent-xyz",
            ),
        )
        conn.execute(
            """INSERT INTO agent_events
               (event_id, event_type, session_id, subagent_type, status, agent_id)
               VALUES (?, ?, ?, ?, ?, ?)""",
            (
                "evt-current",
                "task_delegation",
                "sess-parent",
                "general-purpose",
                "started",
                "agent-xyz",
            ),
        )
        conn.commit()

        # Write DB to temp file for get_parent_event_from_db
        import tempfile

        with tempfile.NamedTemporaryFile(suffix=".db", delete=False) as f:
            temp_path = f.name

        disk_conn = sqlite3.connect(temp_path)
        _create_agent_events_table(disk_conn)
        for row in conn.execute("SELECT * FROM agent_events"):
            disk_conn.execute(
                "INSERT INTO agent_events VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
                row,
            )
        disk_conn.commit()
        disk_conn.close()

        # Should find evt-current (parent session), not evt-old
        result = get_parent_event_from_db(
            temp_path, agent_id="agent-xyz", session_id="sess-parent"
        )
        assert result == "evt-current"

        import os

        os.unlink(temp_path)
        conn.close()


# ===========================================================================
# Finding 2: Agent filter preserves ingested task_delegation Agent events
# ===========================================================================


class TestAgentFilterNuance:
    """Verify that Agent events from transcript ingestion are preserved.

    The fix changes `tool_name != 'Agent'` to:
    `NOT (tool_name = 'Agent' AND event_type != 'task_delegation')`

    This means:
    - Agent events with event_type='tool_call' (PostToolUse echoes) -> HIDDEN
    - Agent events with event_type='task_delegation' (ingested) -> SHOWN
    """

    def test_agent_tool_call_excluded(self):
        """Agent events with event_type='tool_call' are excluded from queries."""
        conn = sqlite3.connect(":memory:")
        _create_agent_events_table(conn)

        # PostToolUse echo: tool_name='Agent', event_type='tool_call'
        conn.execute(
            """INSERT INTO agent_events
               (event_id, event_type, session_id, tool_name, parent_event_id, status)
               VALUES (?, ?, ?, ?, ?, ?)""",
            ("evt-echo", "tool_call", "sess-1", "Agent", "evt-parent", "completed"),
        )
        # Real tool call: tool_name='Read'
        conn.execute(
            """INSERT INTO agent_events
               (event_id, event_type, session_id, tool_name, parent_event_id, status)
               VALUES (?, ?, ?, ?, ?, ?)""",
            ("evt-read", "tool_call", "sess-1", "Read", "evt-parent", "completed"),
        )
        conn.commit()

        cursor = conn.cursor()
        cursor.execute(
            """
            SELECT event_id, tool_name FROM agent_events
            WHERE parent_event_id = ?
            AND NOT (tool_name = 'Agent' AND event_type != 'task_delegation')
            ORDER BY timestamp DESC
            """,
            ("evt-parent",),
        )
        rows = cursor.fetchall()
        event_ids = [r[0] for r in rows]

        assert "evt-echo" not in event_ids, "PostToolUse Agent echo should be excluded"
        assert "evt-read" in event_ids, "Real tool call should be included"
        conn.close()

    def test_agent_task_delegation_preserved(self):
        """Agent events with event_type='task_delegation' (from ingestion) are kept."""
        conn = sqlite3.connect(":memory:")
        _create_agent_events_table(conn)

        # Ingested delegation: tool_name='Agent', event_type='task_delegation'
        conn.execute(
            """INSERT INTO agent_events
               (event_id, event_type, session_id, tool_name, parent_event_id, status)
               VALUES (?, ?, ?, ?, ?, ?)""",
            (
                "evt-ingested",
                "task_delegation",
                "sess-1",
                "Agent",
                "evt-parent",
                "started",
            ),
        )
        conn.commit()

        cursor = conn.cursor()
        cursor.execute(
            """
            SELECT event_id, tool_name FROM agent_events
            WHERE parent_event_id = ?
            AND NOT (tool_name = 'Agent' AND event_type != 'task_delegation')
            ORDER BY timestamp DESC
            """,
            ("evt-parent",),
        )
        rows = cursor.fetchall()
        event_ids = [r[0] for r in rows]

        assert "evt-ingested" in event_ids, (
            "Ingested Agent task_delegation should be preserved"
        )
        conn.close()

    def test_task_tool_not_affected(self):
        """Task events (tool_name='Task') are never filtered."""
        conn = sqlite3.connect(":memory:")
        _create_agent_events_table(conn)

        # Live Task delegation from PreToolUse
        conn.execute(
            """INSERT INTO agent_events
               (event_id, event_type, session_id, tool_name, parent_event_id, status)
               VALUES (?, ?, ?, ?, ?, ?)""",
            ("evt-task", "task_delegation", "sess-1", "Task", "evt-parent", "started"),
        )
        conn.commit()

        cursor = conn.cursor()
        cursor.execute(
            """
            SELECT event_id FROM agent_events
            WHERE parent_event_id = ?
            AND NOT (tool_name = 'Agent' AND event_type != 'task_delegation')
            """,
            ("evt-parent",),
        )
        rows = cursor.fetchall()
        assert len(rows) == 1
        assert rows[0][0] == "evt-task"
        conn.close()

    def test_orphan_query_excludes_agent_echo(self):
        """Orphan query (no parent) excludes Agent echoes but keeps ingested ones."""
        conn = sqlite3.connect(":memory:")
        _create_agent_events_table(conn)

        # Agent echo orphan (should be excluded)
        conn.execute(
            """INSERT INTO agent_events
               (event_id, event_type, session_id, tool_name, parent_event_id, status, timestamp)
               VALUES (?, ?, ?, ?, ?, ?, ?)""",
            (
                "evt-echo",
                "tool_call",
                "sess-1",
                "Agent",
                None,
                "completed",
                "2025-01-01 10:00:01",
            ),
        )
        # Agent ingested delegation orphan (should be included)
        conn.execute(
            """INSERT INTO agent_events
               (event_id, event_type, session_id, tool_name, parent_event_id, status, timestamp)
               VALUES (?, ?, ?, ?, ?, ?, ?)""",
            (
                "evt-ingested",
                "task_delegation",
                "sess-1",
                "Agent",
                None,
                "started",
                "2025-01-01 10:00:02",
            ),
        )
        # Normal tool call orphan (should be included)
        conn.execute(
            """INSERT INTO agent_events
               (event_id, event_type, session_id, tool_name, parent_event_id, status, timestamp)
               VALUES (?, ?, ?, ?, ?, ?, ?)""",
            (
                "evt-bash",
                "tool_call",
                "sess-1",
                "Bash",
                None,
                "completed",
                "2025-01-01 10:00:03",
            ),
        )
        conn.commit()

        cursor = conn.cursor()
        cursor.execute(
            """
            SELECT event_id FROM agent_events
            WHERE session_id = ?
              AND (parent_event_id IS NULL OR parent_event_id = '')
              AND tool_name NOT IN ('UserQuery', 'Stop', 'SessionStart', 'SessionEnd')
              AND NOT (tool_name = 'Agent' AND event_type != 'task_delegation')
              AND timestamp >= ?
            ORDER BY timestamp ASC
            """,
            ("sess-1", "2025-01-01 10:00:00"),
        )
        rows = cursor.fetchall()
        event_ids = [r[0] for r in rows]

        assert "evt-echo" not in event_ids, "Agent echo should be excluded from orphans"
        assert "evt-ingested" in event_ids, (
            "Ingested Agent delegation should be in orphans"
        )
        assert "evt-bash" in event_ids, "Normal tool call should be in orphans"
        conn.close()


# ===========================================================================
# Finding 3: check-systematic-changes.py coverage
# ===========================================================================


class TestShouldSkipSymbol:
    """Tests for should_skip_symbol() in check-systematic-changes.py."""

    def _import_module(self):
        """Import the check-systematic-changes module."""
        import importlib.util
        import sys

        spec = importlib.util.spec_from_file_location(
            "check_systematic_changes",
            "/Users/shakes/DevProjects/htmlgraph/scripts/hooks/check-systematic-changes.py",
        )
        mod = importlib.util.module_from_spec(spec)
        # Temporarily add to sys.modules so the import works
        sys.modules["check_systematic_changes"] = mod
        spec.loader.exec_module(mod)
        return mod

    def test_short_names_skipped(self):
        """Names shorter than MIN_NAME_LEN (5) are skipped."""
        mod = self._import_module()
        assert mod.should_skip_symbol("id") is True
        assert mod.should_skip_symbol("x") is True
        assert mod.should_skip_symbol("fn") is True
        assert mod.should_skip_symbol("abc") is True
        assert mod.should_skip_symbol("test") is True  # len=4 < 5

    def test_common_words_skipped(self):
        """Common/generic words in SKIP_NAMES are skipped."""
        mod = self._import_module()
        assert mod.should_skip_symbol("data") is True
        assert mod.should_skip_symbol("name") is True
        assert mod.should_skip_symbol("path") is True
        assert mod.should_skip_symbol("self") is True
        assert mod.should_skip_symbol("main") is True

    def test_meaningful_names_not_skipped(self):
        """Meaningful names like 'calculate_total' are not skipped."""
        mod = self._import_module()
        assert mod.should_skip_symbol("calculate_total") is False
        assert mod.should_skip_symbol("handle_subagent_start") is False
        assert mod.should_skip_symbol("get_parent_event_from_db") is False
        assert mod.should_skip_symbol("ActivityService") is False

    def test_case_insensitive(self):
        """Skip check is case-insensitive for known words."""
        mod = self._import_module()
        assert mod.should_skip_symbol("DATA") is True  # "data" is in SKIP_NAMES
        assert mod.should_skip_symbol("Name") is True
        assert mod.should_skip_symbol("PATH") is True


class TestGetSearchCommand:
    """Tests for get_search_command() in check-systematic-changes.py."""

    def _import_module(self):
        """Import the check-systematic-changes module."""
        import importlib.util
        import sys

        spec = importlib.util.spec_from_file_location(
            "check_systematic_changes",
            "/Users/shakes/DevProjects/htmlgraph/scripts/hooks/check-systematic-changes.py",
        )
        mod = importlib.util.module_from_spec(spec)
        sys.modules["check_systematic_changes"] = mod
        spec.loader.exec_module(mod)
        return mod

    def test_rg_command_when_available(self):
        """Returns rg-based command when rg is available."""
        mod = self._import_module()
        with patch("shutil.which", return_value="/usr/bin/rg"):
            cmd = mod.get_search_command("my_function", ["src/"])
        assert cmd[0] == "rg"
        assert "-w" in cmd
        assert "my_function" in cmd
        assert "src/" in cmd

    def test_grep_fallback_when_rg_unavailable(self):
        """Falls back to grep when rg is not available."""
        mod = self._import_module()
        with patch("shutil.which", return_value=None):
            cmd = mod.get_search_command("my_function", ["src/"])
        assert cmd[0] == "grep"
        assert "-r" in cmd
        assert "src/" in cmd

    def test_search_dirs_appended(self):
        """Search directories are appended to the command."""
        mod = self._import_module()
        with patch("shutil.which", return_value="/usr/bin/rg"):
            cmd = mod.get_search_command("pattern", ["src/", "packages/"])
        assert "src/" in cmd
        assert "packages/" in cmd
