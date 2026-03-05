"""
Shared Database Helpers for HtmlGraph Hooks.

This module provides common database operations used across multiple hook files,
eliminating duplicated boilerplate code for:
- Project path resolution
- Database connection initialization
- Session ID lookup with fallback chain

All helpers are designed for graceful degradation - they return None on errors
rather than raising exceptions, allowing hooks to continue execution.
"""

import json
import logging
import os
import re
import subprocess
from pathlib import Path
from typing import Any

logger = logging.getLogger(__name__)


def resolve_project_path(cwd: str | None = None) -> str:
    """
    Resolve project path (git root or cwd).

    Attempts to find the git repository root directory. If not in a git repo,
    falls back to the current working directory.

    Args:
        cwd: Starting directory for git root search. Defaults to os.getcwd()

    Returns:
        Absolute path to project root (git root or cwd)

    Example:
        >>> project_path = resolve_project_path()
        >>> db_path = Path(project_path) / ".htmlgraph" / "htmlgraph.db"
    """
    start_dir = cwd or os.getcwd()
    try:
        result = subprocess.run(
            ["git", "rev-parse", "--show-toplevel"],
            capture_output=True,
            text=True,
            cwd=start_dir,
            timeout=5,
        )
        if result.returncode == 0:
            return result.stdout.strip()
    except Exception:
        pass
    return start_dir


def get_db(project_path: str | None = None) -> Any:
    """
    Get database connection for the current project.

    Initializes HtmlGraphDB with the project's database file.
    Uses resolve_project_path() if project_path not provided.

    Args:
        project_path: Optional project root path. If None, uses resolve_project_path()

    Returns:
        HtmlGraphDB instance or None if initialization fails

    Example:
        >>> db = get_db()
        >>> if db:
        ...     cursor = db.connection.cursor()
        ...     cursor.execute("SELECT * FROM sessions LIMIT 1")

    Note:
        Returns None on import errors or connection failures (graceful degradation)
    """
    try:
        from htmlgraph.db.schema import HtmlGraphDB

        if project_path is None:
            project_path = resolve_project_path()

        db_path = Path(project_path) / ".htmlgraph" / "htmlgraph.db"
        return HtmlGraphDB(str(db_path))
    except ImportError as e:
        logger.warning(f"Could not import HtmlGraphDB: {e}")
        return None
    except Exception as e:
        logger.warning(f"Could not initialize database: {e}")
        return None


def get_db_from_config() -> Any:
    """
    Get database connection using get_database_path() from config.

    This is the preferred method when working within the HtmlGraph package,
    as it respects configuration settings and environment variables.

    Returns:
        HtmlGraphDB instance or None if initialization fails

    Example:
        >>> db = get_db_from_config()
        >>> if db:
        ...     events = db.get_recent_events(limit=10)

    Note:
        Falls back to get_db() if config method fails
    """
    try:
        from htmlgraph.config import get_database_path
        from htmlgraph.db.schema import HtmlGraphDB

        db_path = str(get_database_path())
        return HtmlGraphDB(db_path)
    except ImportError:
        logger.debug("Config module not available, using direct path resolution")
        return get_db()
    except Exception as e:
        logger.warning(f"Could not get database from config: {e}")
        return get_db()


def get_current_session_id() -> str | None:
    """
    Query current session_id from environment or session files.

    Implements a fallback chain to find the active session:
    1. Environment variable HTMLGRAPH_SESSION_ID (set by SessionStart hook)
    2. Latest session HTML file in .htmlgraph/sessions/
    3. Session registry file in .htmlgraph/sessions/registry/active/
    4. Most recent UserQuery event from database (last resort)

    Returns:
        Session ID string or None if not found

    Example:
        >>> session_id = get_current_session_id()
        >>> if session_id:
        ...     db = get_db()
        ...     events = db.get_session_events(session_id)

    Note:
        This function is primarily used by PreToolUse hooks that don't receive
        session_id in hook_input. PostToolUse hooks should use the session_id
        from hook_input when available.
    """
    # Priority 1: Environment variable
    session_id = os.environ.get("HTMLGRAPH_SESSION_ID")
    if session_id:
        logger.debug(f"Session ID from environment: {session_id}")
        return session_id

    # Priority 2: Read from latest session HTML file
    try:
        graph_dir = Path.cwd() / ".htmlgraph"
        sessions_dir = graph_dir / "sessions"

        logger.debug(f"Looking for session files in: {sessions_dir}")

        if sessions_dir.exists():
            # Get the most recent session HTML file
            session_files = sorted(
                sessions_dir.glob("sess-*.html"),
                key=lambda p: p.stat().st_mtime,
                reverse=True,
            )
            logger.debug(f"Found {len(session_files)} session files")

            for session_file in session_files:
                try:
                    # Extract session_id from filename (sess-XXXXX.html)
                    match = re.search(r"sess-([a-f0-9]+)", session_file.name)
                    if match:
                        session_id = f"sess-{match.group(1)}"
                        logger.debug(f"Found session ID from file: {session_id}")
                        return session_id
                except Exception as e:
                    logger.debug(f"Error reading session file {session_file}: {e}")
                    continue
            logger.debug("No valid session files found")
        else:
            logger.debug(f"Sessions directory not found: {sessions_dir}")
    except Exception as e:
        logger.debug(f"Could not read from session files: {e}")

    # Priority 3: Read from session registry
    try:
        graph_dir = Path.cwd() / ".htmlgraph"
        registry_dir = graph_dir / "sessions" / "registry" / "active"

        if registry_dir.exists():
            # Get the most recent session file
            session_files = sorted(
                registry_dir.glob("*.json"),
                key=lambda p: p.stat().st_mtime,
                reverse=True,
            )

            for session_file in session_files:
                try:
                    with open(session_file) as f:
                        data = json.load(f)
                        if data.get("status") == "active":
                            session_id = data.get("session_id")
                            if isinstance(session_id, str):
                                logger.debug(
                                    f"Found session ID from registry: {session_id}"
                                )
                                return session_id
                except Exception:
                    continue
    except Exception as e:
        logger.debug(f"Could not read from session registry: {e}")

    # Priority 4: Query database for most recent UserQuery event
    try:
        db = get_db()
        if db and db.connection:
            cursor = db.connection.cursor()
            cursor.execute("""
                SELECT session_id FROM agent_events
                WHERE tool_name = 'UserQuery'
                ORDER BY datetime(REPLACE(SUBSTR(timestamp, 1, 19), 'T', ' ')) DESC
                LIMIT 1
            """)
            row = cursor.fetchone()
            if row and row[0]:
                session_id = str(row[0])
                logger.debug(f"Found session ID from database: {session_id}")
                return session_id
    except Exception as e:
        logger.debug(f"Could not query database for session: {e}")

    logger.debug("Could not resolve session_id from any source")
    return None


def ensure_session_exists(db: Any, session_id: str, agent_id: str = "system") -> bool:
    """
    Ensure session exists in database (create placeholder if needed).

    Creates a minimal session entry if one doesn't exist. This is useful for
    hooks that need to insert events but can't guarantee the session was
    created by SessionStart hook.

    Args:
        db: HtmlGraphDB instance
        session_id: Session ID to check/create
        agent_id: Agent identifier for the session (default: "system")

    Returns:
        True if session exists or was created, False on error

    Example:
        >>> db = get_db()
        >>> if db:
        ...     ensure_session_exists(db, "sess-abc123", "claude-code")
        ...     db.insert_event(...)  # Now safe to insert events

    Note:
        Uses INSERT OR IGNORE to handle race conditions gracefully
    """
    try:
        if not db or not db.connection:
            return False

        cursor = db.connection.cursor()

        # Check if session exists
        cursor.execute(
            "SELECT COUNT(*) FROM sessions WHERE session_id = ?",
            (session_id,),
        )
        exists = cursor.fetchone()[0] > 0

        if not exists:
            # Create placeholder session
            from datetime import datetime, timezone

            cursor.execute(
                """
                INSERT OR IGNORE INTO sessions
                (session_id, agent_assigned, created_at, status)
                VALUES (?, ?, ?, 'active')
                """,
                (
                    session_id,
                    agent_id,
                    datetime.now(timezone.utc).isoformat(),
                ),
            )
            db.connection.commit()
            logger.debug(f"Created placeholder session: {session_id}")

        return True
    except Exception as e:
        logger.warning(f"Could not ensure session exists: {e}")
        return False


def get_parent_user_query(db: Any, session_id: str) -> str | None:
    """
    Get the most recent UserQuery event_id for this session from database.

    This is the primary method for parent-child event linking.
    Database is the single source of truth - no file-based state.

    Args:
        db: HtmlGraphDB instance
        session_id: Session ID to query

    Returns:
        event_id of the most recent UserQuery event, or None if not found

    Example:
        >>> db = get_db()
        >>> session_id = get_current_session_id()
        >>> if db and session_id:
        ...     parent_id = get_parent_user_query(db, session_id)
        ...     # Use parent_id for event linking

    Note:
        Returns None if database unavailable or no UserQuery found
    """
    try:
        if not db or db.connection is None:
            return None
        cursor = db.connection.cursor()
        cursor.execute(
            """
            SELECT event_id FROM agent_events
            WHERE session_id = ? AND tool_name = 'UserQuery'
            ORDER BY datetime(REPLACE(SUBSTR(timestamp, 1, 19), 'T', ' ')) DESC
            LIMIT 1
            """,
            (session_id,),
        )
        row = cursor.fetchone()
        if row:
            return str(row[0])
        return None
    except Exception as e:
        logger.warning(f"Database query for UserQuery failed: {e}")
        return None


__all__ = [
    "resolve_project_path",
    "get_db",
    "get_db_from_config",
    "get_current_session_id",
    "ensure_session_exists",
    "get_parent_user_query",
]
