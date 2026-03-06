"""
HtmlGraph FTS5 Full-Text Search

Provides full-text search across all sessions and events stored in the
SQLite database using SQLite's built-in FTS5 extension.

Tables:
- sessions_fts: Virtual FTS5 table over sessions (session_id, agent, query)
- events_fts:   Virtual FTS5 table over agent_events (session_id, tool, summaries)

Public API:
- build_fts_index(db_path): Rebuild FTS index from existing data
- search_sessions(query, db_path, limit): Full-text search across sessions
"""

from __future__ import annotations

import logging
import sqlite3
from pathlib import Path
from typing import Any

logger = logging.getLogger(__name__)

_DEFAULT_DB = str(Path.home() / ".htmlgraph" / "htmlgraph.db")


# ---------------------------------------------------------------------------
# DDL helpers (called from schema.py create_tables flow)
# ---------------------------------------------------------------------------


def create_fts_tables(cursor: sqlite3.Cursor) -> None:
    """Create FTS5 virtual tables if they don't already exist.

    Args:
        cursor: Active SQLite cursor
    """
    # Sessions FTS: index session metadata and last user query
    cursor.execute("""
        CREATE VIRTUAL TABLE IF NOT EXISTS sessions_fts USING fts5(
            session_id UNINDEXED,
            agent_assigned,
            last_user_query,
            content='sessions',
            content_rowid='rowid',
            tokenize='porter ascii'
        )
    """)

    # Events FTS: index tool names and summaries for deep search
    cursor.execute("""
        CREATE VIRTUAL TABLE IF NOT EXISTS events_fts USING fts5(
            event_id UNINDEXED,
            session_id UNINDEXED,
            tool_name,
            input_summary,
            output_summary,
            content='agent_events',
            content_rowid='rowid',
            tokenize='porter ascii'
        )
    """)

    # Triggers to keep sessions_fts in sync with sessions table
    cursor.execute("""
        CREATE TRIGGER IF NOT EXISTS sessions_fts_insert
        AFTER INSERT ON sessions BEGIN
            INSERT INTO sessions_fts(rowid, session_id, agent_assigned, last_user_query)
            VALUES (new.rowid, new.session_id, new.agent_assigned, new.last_user_query);
        END
    """)
    cursor.execute("""
        CREATE TRIGGER IF NOT EXISTS sessions_fts_delete
        AFTER DELETE ON sessions BEGIN
            INSERT INTO sessions_fts(sessions_fts, rowid, session_id, agent_assigned, last_user_query)
            VALUES ('delete', old.rowid, old.session_id, old.agent_assigned, old.last_user_query);
        END
    """)
    cursor.execute("""
        CREATE TRIGGER IF NOT EXISTS sessions_fts_update
        AFTER UPDATE ON sessions BEGIN
            INSERT INTO sessions_fts(sessions_fts, rowid, session_id, agent_assigned, last_user_query)
            VALUES ('delete', old.rowid, old.session_id, old.agent_assigned, old.last_user_query);
            INSERT INTO sessions_fts(rowid, session_id, agent_assigned, last_user_query)
            VALUES (new.rowid, new.session_id, new.agent_assigned, new.last_user_query);
        END
    """)

    # Triggers to keep events_fts in sync with agent_events table
    cursor.execute("""
        CREATE TRIGGER IF NOT EXISTS events_fts_insert
        AFTER INSERT ON agent_events BEGIN
            INSERT INTO events_fts(rowid, event_id, session_id, tool_name, input_summary, output_summary)
            VALUES (new.rowid, new.event_id, new.session_id, new.tool_name, new.input_summary, new.output_summary);
        END
    """)
    cursor.execute("""
        CREATE TRIGGER IF NOT EXISTS events_fts_delete
        AFTER DELETE ON agent_events BEGIN
            INSERT INTO events_fts(events_fts, rowid, event_id, session_id, tool_name, input_summary, output_summary)
            VALUES ('delete', old.rowid, old.event_id, old.session_id, old.tool_name, old.input_summary, old.output_summary);
        END
    """)
    cursor.execute("""
        CREATE TRIGGER IF NOT EXISTS events_fts_update
        AFTER UPDATE ON agent_events BEGIN
            INSERT INTO events_fts(events_fts, rowid, event_id, session_id, tool_name, input_summary, output_summary)
            VALUES ('delete', old.rowid, old.event_id, old.session_id, old.tool_name, old.input_summary, old.output_summary);
            INSERT INTO events_fts(rowid, event_id, session_id, tool_name, input_summary, output_summary)
            VALUES (new.rowid, new.event_id, new.session_id, new.tool_name, new.input_summary, new.output_summary);
        END
    """)


# ---------------------------------------------------------------------------
# Index management
# ---------------------------------------------------------------------------


def build_fts_index(db_path: str | None = None) -> int:
    """Rebuild the FTS5 index from existing database content.

    Safe to run multiple times (clears and repopulates).  Useful after
    importing historical data or when the index drifts out of sync.

    Args:
        db_path: Path to the SQLite database. Defaults to ~/.htmlgraph/htmlgraph.db

    Returns:
        Total number of rows indexed (sessions + events)
    """
    path = db_path or _DEFAULT_DB
    conn = sqlite3.connect(path)
    conn.row_factory = sqlite3.Row
    try:
        cursor = conn.cursor()

        # Ensure tables exist
        create_fts_tables(cursor)
        conn.commit()

        # Repopulate sessions_fts
        cursor.execute("DELETE FROM sessions_fts")
        cursor.execute("""
            INSERT INTO sessions_fts(rowid, session_id, agent_assigned, last_user_query)
            SELECT rowid, session_id, agent_assigned, last_user_query
            FROM sessions
        """)
        session_count = cursor.rowcount

        # Repopulate events_fts
        cursor.execute("DELETE FROM events_fts")
        cursor.execute("""
            INSERT INTO events_fts(rowid, event_id, session_id, tool_name, input_summary, output_summary)
            SELECT rowid, event_id, session_id, tool_name, input_summary, output_summary
            FROM agent_events
        """)
        event_count = cursor.rowcount

        conn.commit()
        total = session_count + event_count
        logger.info(
            "FTS index rebuilt: %d sessions, %d events", session_count, event_count
        )
        return total
    finally:
        conn.close()


# ---------------------------------------------------------------------------
# Search API
# ---------------------------------------------------------------------------


def search_sessions(
    query: str,
    *,
    db_path: str | None = None,
    limit: int = 20,
) -> list[dict[str, Any]]:
    """Full-text search across all sessions.

    Searches the sessions_fts index for sessions whose agent name or
    last user query matches *query*.  Results are ranked by BM25 relevance.

    Args:
        query: FTS5 search query string (supports phrase search, column
               filters, and boolean operators as per FTS5 syntax).
        db_path: Path to the SQLite database. Defaults to ~/.htmlgraph/htmlgraph.db
        limit: Maximum number of results to return (default: 20)

    Returns:
        List of dicts, each containing:
        - session_id: str
        - agent_assigned: str
        - created_at: str
        - status: str
        - snippet: str  (highlighted excerpt from matched text)
        - rank: float   (BM25 score; more negative = higher relevance)

    Example:
        results = search_sessions("JWT authentication")
        for r in results:
            print(r["session_id"], r["snippet"])
    """
    path = db_path or _DEFAULT_DB
    conn = sqlite3.connect(path)
    conn.row_factory = sqlite3.Row
    try:
        cursor = conn.cursor()

        # Ensure FTS tables exist before querying
        _ensure_fts_tables(cursor)

        cursor.execute(
            """
            SELECT
                s.session_id,
                s.agent_assigned,
                s.created_at,
                s.status,
                s.last_user_query,
                snippet(sessions_fts, 2, '[', ']', '...', 10) AS snippet,
                sessions_fts.rank AS rank
            FROM sessions_fts
            JOIN sessions s ON s.rowid = sessions_fts.rowid
            WHERE sessions_fts MATCH ?
            ORDER BY rank
            LIMIT ?
            """,
            (query, limit),
        )
        rows = cursor.fetchall()
        return [dict(row) for row in rows]
    except sqlite3.OperationalError as exc:
        # FTS table might not exist on very old DBs - return empty rather than crash
        logger.warning("FTS search failed (index may need rebuild): %s", exc)
        return []
    finally:
        conn.close()


def search_events(
    query: str,
    *,
    db_path: str | None = None,
    limit: int = 20,
) -> list[dict[str, Any]]:
    """Full-text search across all agent events (tool calls).

    Searches tool names, input summaries, and output summaries.

    Args:
        query: FTS5 search query string
        db_path: Path to the SQLite database. Defaults to ~/.htmlgraph/htmlgraph.db
        limit: Maximum number of results to return (default: 20)

    Returns:
        List of dicts, each containing:
        - event_id: str
        - session_id: str
        - tool_name: str
        - snippet: str  (highlighted excerpt)
        - rank: float   (BM25 score)
    """
    path = db_path or _DEFAULT_DB
    conn = sqlite3.connect(path)
    conn.row_factory = sqlite3.Row
    try:
        cursor = conn.cursor()
        _ensure_fts_tables(cursor)

        cursor.execute(
            """
            SELECT
                ae.event_id,
                ae.session_id,
                ae.tool_name,
                ae.event_type,
                ae.timestamp,
                snippet(events_fts, 3, '[', ']', '...', 10) AS snippet,
                events_fts.rank AS rank
            FROM events_fts
            JOIN agent_events ae ON ae.rowid = events_fts.rowid
            WHERE events_fts MATCH ?
            ORDER BY rank
            LIMIT ?
            """,
            (query, limit),
        )
        rows = cursor.fetchall()
        return [dict(row) for row in rows]
    except sqlite3.OperationalError as exc:
        logger.warning("FTS event search failed (index may need rebuild): %s", exc)
        return []
    finally:
        conn.close()


# ---------------------------------------------------------------------------
# Internal helpers
# ---------------------------------------------------------------------------


def _ensure_fts_tables(cursor: sqlite3.Cursor) -> None:
    """Create FTS tables if missing (idempotent)."""
    cursor.execute(
        "SELECT name FROM sqlite_master WHERE type='table' AND name='sessions_fts'"
    )
    if not cursor.fetchone():
        create_fts_tables(cursor)
        # Populate from existing data
        cursor.execute("""
            INSERT INTO sessions_fts(rowid, session_id, agent_assigned, last_user_query)
            SELECT rowid, session_id, agent_assigned, last_user_query
            FROM sessions
        """)
        cursor.execute("""
            INSERT INTO events_fts(rowid, event_id, session_id, tool_name, input_summary, output_summary)
            SELECT rowid, event_id, session_id, tool_name, input_summary, output_summary
            FROM agent_events
        """)
        if cursor.connection:
            cursor.connection.commit()
