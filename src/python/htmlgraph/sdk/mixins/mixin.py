"""
Core mixin for SDK - database, refs, export, and utility methods.

Provides essential SDK functionality that doesn't belong in specialized mixins.
"""

from __future__ import annotations

import logging
import os
from typing import TYPE_CHECKING, Any, cast

if TYPE_CHECKING:
    from pathlib import Path

    from htmlgraph.agents import AgentInterface
    from htmlgraph.db.schema import HtmlGraphDB
    from htmlgraph.models import Node
    from htmlgraph.refs import RefManager
    from htmlgraph.session_manager import SessionManager

logger = logging.getLogger(__name__)


class CoreMixin:
    """
    Mixin providing core SDK functionality.

    Includes:
    - Database access (db, query, execute_query_builder)
    - Ref resolution (ref)
    - Export functionality (export_to_html)
    - Event logging (_log_event)
    - Utility methods (reload, summary, my_work, next_task, get_status, dedupe_sessions)
    """

    _directory: Path
    _db: HtmlGraphDB
    _agent_id: str | None
    _parent_session: str | None
    _agent_interface: AgentInterface
    _graph: Any
    session_manager: SessionManager
    refs: RefManager

    # Collection references (for ref resolution)
    features: Any
    tracks: Any
    bugs: Any
    spikes: Any
    chores: Any
    epics: Any
    todos: Any
    phases: Any

    def ref(self, short_ref: str) -> Node | None:
        """
        Resolve a short ref to a Node object.

        Short refs are stable identifiers like @f1, @t2, @b5 that map to
        full node IDs. This method resolves the short ref and fetches the
        corresponding node from the appropriate collection.

        Args:
            short_ref: Short ref like "@f1", "@t2", "@b5", etc.

        Returns:
            Node object or None if not found

        Example:
            >>> sdk = SDK(agent="claude")
            >>> feature = sdk.ref("@f1")
            >>> if feature:
            ...     logger.info("%s", feature.title)
            ...     feature.status = "done"
            ...     sdk.features.update(feature)
        """
        # Resolve short ref to full ID
        full_id = self.refs.resolve_ref(short_ref)
        if not full_id:
            return None

        # Determine type from ref prefix and fetch from appropriate collection
        if len(short_ref) < 2:
            return None

        prefix = short_ref[1]  # Get letter after @

        # Map prefix to collection
        collection_map = {
            "f": self.features,
            "t": self.tracks,
            "b": self.bugs,
            "s": self.spikes,
        }

        collection = collection_map.get(prefix)
        if not collection:
            return None

        # Get node from collection
        if hasattr(collection, "get"):
            return cast("Node | None", collection.get(full_id))

        return None

    # =========================================================================
    # SQLite Database Integration
    # =========================================================================

    def db(self) -> HtmlGraphDB:
        """
        Get the SQLite database instance.

        Returns:
            HtmlGraphDB instance for executing queries

        Example:
            >>> sdk = SDK(agent="claude")
            >>> db = sdk.db()
            >>> events = db.get_session_events("sess-123")
            >>> features = db.get_features_by_status("todo")
        """
        return self._db

    def query(self, sql: str, params: tuple[Any, ...] = ()) -> list[dict[str, Any]]:
        """
        Execute a raw SQL query on the SQLite database.

        Args:
            sql: SQL query string
            params: Query parameters (for safe parameterized queries)

        Returns:
            List of result dictionaries

        Example:
            >>> sdk = SDK(agent="claude")
            >>> results = sdk.query(
            ...     "SELECT * FROM features WHERE status = ? AND priority = ?",
            ...     ("todo", "high")
            ... )
            >>> for row in results:
            ...     print(row["title"])
        """
        if not self._db.connection:
            self._db.connect()

        cursor = self._db.connection.cursor()  # type: ignore[union-attr]
        cursor.execute(sql, params)
        rows = cursor.fetchall()
        return [dict(row) for row in rows]

    def execute_query_builder(
        self, sql: str, params: tuple[Any, ...] = ()
    ) -> list[dict[str, Any]]:
        """
        Execute a query using the Queries builder.

        Args:
            sql: SQL query from Queries builder
            params: Parameters from Queries builder

        Returns:
            List of result dictionaries

        Example:
            >>> sdk = SDK(agent="claude")
            >>> sql, params = Queries.get_features_by_status("todo", limit=5)
            >>> results = sdk.execute_query_builder(sql, params)
        """
        return self.query(sql, params)

    def export_to_html(
        self,
        output_dir: str | None = None,
        include_features: bool = True,
        include_sessions: bool = True,
        include_events: bool = False,
    ) -> dict[str, int]:
        """
        Export SQLite data to HTML files for backward compatibility.

        Args:
            output_dir: Directory to export to (defaults to .htmlgraph)
            include_features: Export features
            include_sessions: Export sessions
            include_events: Export events (detailed, use with care)

        Returns:
            Dict with export counts: {"features": int, "sessions": int, "events": int}

        Example:
            >>> sdk = SDK(agent="claude")
            >>> result = sdk.export_to_html()
            >>> logger.info(f"Exported {result['features']} features")
        """
        from pathlib import Path

        if output_dir is None:
            output_dir = str(self._directory)

        output_path = Path(output_dir)
        counts: dict[str, int] = {"features": 0, "sessions": 0, "events": 0}

        if include_features:
            # Export all features from SQLite to HTML
            features_dir = output_path / "features"
            features_dir.mkdir(parents=True, exist_ok=True)

            try:
                cursor = self._db.connection.cursor()  # type: ignore[union-attr]
                cursor.execute("SELECT * FROM features")
                rows = cursor.fetchall()

                for row in rows:
                    feature_dict = dict(row)
                    feature_id = feature_dict["id"]
                    # Write HTML file (simplified export)
                    html_file = features_dir / f"{feature_id}.html"
                    html_file.write_text(
                        f"<h1>{feature_dict['title']}</h1>"
                        f"<p>Status: {feature_dict['status']}</p>"
                        f"<p>Type: {feature_dict['type']}</p>"
                    )
                    counts["features"] += 1
            except Exception as e:
                logger.error(f"Error exporting features: {e}")

        if include_sessions:
            # Export all sessions from SQLite to HTML
            sessions_dir = output_path / "sessions"
            sessions_dir.mkdir(parents=True, exist_ok=True)

            try:
                cursor = self._db.connection.cursor()  # type: ignore[union-attr]
                cursor.execute("SELECT * FROM sessions")
                rows = cursor.fetchall()

                for row in rows:
                    session_dict = dict(row)
                    session_id = session_dict["session_id"]
                    # Write HTML file (simplified export)
                    html_file = sessions_dir / f"{session_id}.html"
                    html_file.write_text(
                        f"<h1>Session {session_id}</h1>"
                        f"<p>Agent: {session_dict['agent_assigned']}</p>"
                        f"<p>Status: {session_dict['status']}</p>"
                    )
                    counts["sessions"] += 1
            except Exception as e:
                logger.error(f"Error exporting sessions: {e}")

        return counts

    def _log_event(
        self,
        event_type: str,
        tool_name: str | None = None,
        input_summary: str | None = None,
        output_summary: str | None = None,
        context: dict[str, Any] | None = None,
        cost_tokens: int = 0,
    ) -> bool:
        """
        Log an event to the SQLite database with parent-child linking.

        Internal method used by collections to track operations.
        Automatically creates a session if one doesn't exist.
        Reads parent event ID from HTMLGRAPH_PARENT_ACTIVITY env var for hierarchical tracking.

        Args:
            event_type: Type of event (tool_call, completion, error, etc.)
            tool_name: Tool that was called
            input_summary: Summary of input
            output_summary: Summary of output
            context: Additional context metadata
            cost_tokens: Token cost estimate

        Returns:
            True if logged successfully, False otherwise

        Example (internal use):
            >>> sdk._log_event(
            ...     event_type="tool_call",
            ...     tool_name="Edit",
            ...     input_summary="Edit file.py",
            ...     cost_tokens=100
            ... )
        """
        from uuid import uuid4

        event_id = f"evt-{uuid4().hex[:12]}"
        session_id = self._parent_session or "cli-session"

        # Read parent event ID from environment variable for hierarchical linking
        parent_event_id = os.getenv("HTMLGRAPH_PARENT_ACTIVITY")

        # Ensure session exists before logging event
        try:
            self._ensure_session_exists(session_id, parent_event_id=parent_event_id)  # type: ignore[attr-defined]
        except Exception as e:
            logger.debug(f"Failed to ensure session exists: {e}")
            # Continue anyway - session creation failure shouldn't block event logging

        # Ensure agent_id is set for event logging
        agent_id = self._agent_id or "unknown"

        return self._db.insert_event(
            event_id=event_id,
            agent_id=agent_id,
            event_type=event_type,
            session_id=session_id,
            tool_name=tool_name,
            input_summary=input_summary,
            output_summary=output_summary,
            context=context,
            parent_event_id=parent_event_id,
            cost_tokens=cost_tokens,
        )

    def reload(self) -> None:
        """Reload all data from disk."""
        self._graph.reload()
        self._agent_interface.reload()
        # SessionManager reloads implicitly on access via its converters/graphs

    def summary(self, max_items: int = 10) -> str:
        """
        Get project summary.

        Returns:
            Compact overview for AI agent orientation
        """
        return self._agent_interface.get_summary(max_items)

    def my_work(self) -> dict[str, Any]:
        """
        Get current agent's workload.

        Returns:
            Dict with in_progress, completed counts
        """
        if not self._agent_id:
            raise ValueError("No agent ID set")
        return self._agent_interface.get_workload(self._agent_id)

    def next_task(
        self, priority: str | None = None, auto_claim: bool = True
    ) -> Node | None:
        """
        Get next available task for this agent.

        Args:
            priority: Optional priority filter
            auto_claim: Automatically claim the task

        Returns:
            Next available Node or None
        """
        return self._agent_interface.get_next_task(
            agent_id=self._agent_id,
            priority=priority,
            node_type="feature",
            auto_claim=auto_claim,
        )

    def get_status(self) -> dict[str, Any]:
        """
        Get project status.

        Returns:
            Dict with status metrics (WIP, counts, etc.)
        """
        return self.session_manager.get_status()

    def dedupe_sessions(
        self,
        max_events: int = 1,
        move_dir_name: str = "_orphans",
        dry_run: bool = False,
        stale_extra_active: bool = True,
    ) -> dict[str, int]:
        """
        Move low-signal sessions (e.g. SessionStart-only) out of the main sessions dir.

        Args:
            max_events: Maximum events threshold (sessions with <= this many events are moved)
            move_dir_name: Directory name to move orphaned sessions to
            dry_run: If True, only report what would be done without actually moving files
            stale_extra_active: If True, also mark extra active sessions as stale

        Returns:
            Dict with counts: {"scanned": int, "moved": int, "missing": int, "staled_active": int, "kept_active": int}

        Example:
            >>> sdk = SDK(agent="claude")
            >>> result = sdk.dedupe_sessions(max_events=1, dry_run=False)
            >>> logger.info(f"Scanned: {result['scanned']}, Moved: {result['moved']}")
        """
        return self.session_manager.dedupe_orphan_sessions(
            max_events=max_events,
            move_dir_name=move_dir_name,
            dry_run=dry_run,
            stale_extra_active=stale_extra_active,
        )
