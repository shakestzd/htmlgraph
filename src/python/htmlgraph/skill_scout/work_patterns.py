"""Work Patterns: mine event history from SQLite to find tool usage and agent patterns."""

from __future__ import annotations

import sqlite3
from pathlib import Path

from pydantic import BaseModel

_TOP_TOOLS_LIMIT = 10


class WorkPatterns(BaseModel):
    tool_frequency: dict[str, int]
    top_tools: list[str]
    agent_types: dict[str, int]
    active_hours: dict[int, int]
    failed_tools: dict[str, int]


def analyze_work_patterns(db_path: Path) -> WorkPatterns:
    """Query SQLite event history to build a picture of how the project is used."""
    if not db_path.exists():
        return _empty_patterns()

    try:
        conn = sqlite3.connect(str(db_path))
        conn.row_factory = sqlite3.Row
        try:
            return WorkPatterns(
                tool_frequency=_tool_frequency(conn),
                top_tools=_top_tools(conn),
                agent_types=_agent_types(conn),
                active_hours=_active_hours(conn),
                failed_tools=_failed_tools(conn),
            )
        finally:
            conn.close()
    except sqlite3.Error:
        return _empty_patterns()


def _tool_frequency(conn: sqlite3.Connection) -> dict[str, int]:
    rows = conn.execute(
        "SELECT tool_name, COUNT(*) AS cnt FROM agent_events "
        "WHERE tool_name IS NOT NULL GROUP BY tool_name ORDER BY cnt DESC"
    ).fetchall()
    return {r["tool_name"]: r["cnt"] for r in rows}


def _top_tools(conn: sqlite3.Connection) -> list[str]:
    rows = conn.execute(
        "SELECT tool_name FROM agent_events WHERE tool_name IS NOT NULL "
        "GROUP BY tool_name ORDER BY COUNT(*) DESC LIMIT ?",
        (_TOP_TOOLS_LIMIT,),
    ).fetchall()
    return [r["tool_name"] for r in rows]


def _agent_types(conn: sqlite3.Connection) -> dict[str, int]:
    rows = conn.execute(
        "SELECT subagent_type, COUNT(*) AS cnt FROM agent_events "
        "WHERE subagent_type IS NOT NULL AND event_type = 'task_delegation' "
        "GROUP BY subagent_type ORDER BY cnt DESC"
    ).fetchall()
    return {r["subagent_type"]: r["cnt"] for r in rows}


def _active_hours(conn: sqlite3.Connection) -> dict[int, int]:
    rows = conn.execute(
        "SELECT CAST(strftime('%H', created_at) AS INTEGER) AS hour, "
        "COUNT(*) AS cnt FROM agent_events GROUP BY hour ORDER BY hour"
    ).fetchall()
    return {r["hour"]: r["cnt"] for r in rows}


def _failed_tools(conn: sqlite3.Connection) -> dict[str, int]:
    rows = conn.execute(
        "SELECT tool_name, COUNT(*) AS cnt FROM agent_events "
        "WHERE status = 'failed' AND tool_name IS NOT NULL "
        "GROUP BY tool_name ORDER BY cnt DESC"
    ).fetchall()
    return {r["tool_name"]: r["cnt"] for r in rows}


def _empty_patterns() -> WorkPatterns:
    return WorkPatterns(
        tool_frequency={},
        top_tools=[],
        agent_types={},
        active_hours={},
        failed_tools={},
    )
