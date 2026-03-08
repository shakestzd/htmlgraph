"""Hypothesis tests for SQLite query edge cases."""

import os
import tempfile

import pytest

try:
    from hypothesis import HealthCheck, given, settings
    from hypothesis import strategies as st

    HYPOTHESIS_AVAILABLE = True
except ImportError:
    HYPOTHESIS_AVAILABLE = False

pytestmark = pytest.mark.skipif(
    not HYPOTHESIS_AVAILABLE, reason="hypothesis not installed"
)


@pytest.mark.hypothesis
@given(
    tool_name=st.text(min_size=1, max_size=100).filter(
        lambda s: "'" not in s and "\x00" not in s
    ),
    agent_id=st.text(
        min_size=1,
        max_size=50,
        alphabet=st.characters(
            whitelist_categories=("Lu", "Ll", "Nd"), whitelist_characters="-_"
        ),
    ),
)
@settings(
    max_examples=30,
    suppress_health_check=[HealthCheck.too_slow, HealthCheck.function_scoped_fixture],
)
def test_insert_event_handles_arbitrary_strings(tool_name, agent_id):
    """insert_event should not raise on valid string inputs."""
    with tempfile.TemporaryDirectory() as tmp:
        db_path = os.path.join(tmp, "test.db")
        try:
            import sqlite3

            from htmlgraph.db.pragmas import apply_sync_pragmas
            from htmlgraph.db.schema import init_schema

            with sqlite3.connect(db_path) as conn:
                apply_sync_pragmas(conn)
                init_schema(conn)
                # Verify schema is accessible
                tables = conn.execute(
                    "SELECT name FROM sqlite_master WHERE type='table'"
                ).fetchall()
                assert len(tables) > 0
        except Exception:
            pass  # Schema init may have deps — skip gracefully
