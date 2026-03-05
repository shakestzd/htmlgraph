"""
HtmlGraph Database Extensions - Auxiliary Operations

Mixin class providing less-frequently-used auxiliary operations:
- Tool trace CRUD (insert_tool_trace, update_tool_trace, get_tool_trace, etc.)
- Live event streaming (insert_live_event, get_pending_live_events, etc.)
- Collaboration/delegation tracking (record_collaboration, record_delegation_event, etc.)
- Sync operation tracking (insert_sync_operation, get_sync_operations)

Inherited by HtmlGraphDB in schema.py via:
    class HtmlGraphDB(ExtensionOps): ...
"""

from __future__ import annotations

import json
import logging
import sqlite3
from datetime import datetime, timedelta, timezone
from typing import Any

logger = logging.getLogger(__name__)


class ExtensionOps:
    """Mixin providing auxiliary database operations for HtmlGraphDB."""

    connection: sqlite3.Connection | None
    connect: Any  # provided by HtmlGraphDB
    _ensure_session_exists: Any  # provided by HtmlGraphDB core CRUD

    # ------------------------------------------------------------------
    # Tool Traces
    # ------------------------------------------------------------------

    def insert_tool_trace(
        self,
        tool_use_id: str,
        trace_id: str,
        session_id: str,
        tool_name: str,
        tool_input: dict[str, Any] | None = None,
        start_time: str | None = None,
        parent_tool_use_id: str | None = None,
    ) -> bool:
        """
        Insert a tool trace start event.

        Args:
            tool_use_id: Unique tool use identifier (UUID)
            trace_id: Parent trace ID for correlation
            session_id: Session this tool use belongs to
            tool_name: Name of the tool being executed
            tool_input: Tool input parameters as dict (optional)
            start_time: Start time ISO8601 UTC (optional, defaults to now)
            parent_tool_use_id: Parent tool use ID if nested (optional)

        Returns:
            True if insert successful, False otherwise
        """
        if not self.connection:
            self.connect()

        try:
            cursor = self.connection.cursor()  # type: ignore[union-attr]

            if start_time is None:
                start_time = datetime.now(timezone.utc).isoformat()

            cursor.execute(
                """
                INSERT INTO tool_traces
                (tool_use_id, trace_id, session_id, tool_name, tool_input,
                 start_time, status, parent_tool_use_id)
                VALUES (?, ?, ?, ?, ?, ?, ?, ?)
            """,
                (
                    tool_use_id,
                    trace_id,
                    session_id,
                    tool_name,
                    json.dumps(tool_input) if tool_input else None,
                    start_time,
                    "started",
                    parent_tool_use_id,
                ),
            )
            self.connection.commit()  # type: ignore[union-attr]
            return True
        except sqlite3.Error as e:
            logger.error(f"Error inserting tool trace: {e}")
            return False

    def update_tool_trace(
        self,
        tool_use_id: str,
        tool_output: dict[str, Any] | None = None,
        end_time: str | None = None,
        duration_ms: int | None = None,
        status: str = "completed",
        error_message: str | None = None,
    ) -> bool:
        """
        Update tool trace with completion data.

        Args:
            tool_use_id: Tool use ID to update
            tool_output: Tool output result (optional)
            end_time: End time ISO8601 UTC (optional, defaults to now)
            duration_ms: Execution duration in milliseconds (optional)
            status: Final status (completed, failed, timeout, cancelled)
            error_message: Error message if failed (optional)

        Returns:
            True if update successful, False otherwise
        """
        if not self.connection:
            self.connect()

        try:
            cursor = self.connection.cursor()  # type: ignore[union-attr]

            if end_time is None:
                end_time = datetime.now(timezone.utc).isoformat()

            cursor.execute(
                """
                UPDATE tool_traces
                SET tool_output = ?, end_time = ?, duration_ms = ?,
                    status = ?, error_message = ?
                WHERE tool_use_id = ?
            """,
                (
                    json.dumps(tool_output) if tool_output else None,
                    end_time,
                    duration_ms,
                    status,
                    error_message,
                    tool_use_id,
                ),
            )
            self.connection.commit()  # type: ignore[union-attr]
            return True
        except sqlite3.Error as e:
            logger.error(f"Error updating tool trace: {e}")
            return False

    def get_tool_trace(self, tool_use_id: str) -> dict[str, Any] | None:
        """
        Get a tool trace by tool_use_id.

        Args:
            tool_use_id: Tool use ID to retrieve

        Returns:
            Tool trace dictionary or None if not found
        """
        if not self.connection:
            self.connect()

        try:
            cursor = self.connection.cursor()  # type: ignore[union-attr]
            cursor.execute(
                """
                SELECT * FROM tool_traces
                WHERE tool_use_id = ?
            """,
                (tool_use_id,),
            )

            row = cursor.fetchone()
            return dict(row) if row else None
        except sqlite3.Error as e:
            logger.error(f"Error fetching tool trace: {e}")
            return None

    def get_session_tool_traces(
        self, session_id: str, limit: int = 1000
    ) -> list[dict[str, Any]]:
        """
        Get all tool traces for a session ordered by start time DESC.

        Args:
            session_id: Session to query
            limit: Maximum number of results

        Returns:
            List of tool trace dictionaries
        """
        if not self.connection:
            self.connect()

        try:
            cursor = self.connection.cursor()  # type: ignore[union-attr]
            cursor.execute(
                """
                SELECT * FROM tool_traces
                WHERE session_id = ?
                ORDER BY start_time DESC
                LIMIT ?
            """,
                (session_id, limit),
            )

            rows = cursor.fetchall()
            return [dict(row) for row in rows]
        except sqlite3.Error as e:
            logger.error(f"Error querying tool traces: {e}")
            return []

    # ------------------------------------------------------------------
    # Live Events
    # ------------------------------------------------------------------

    def insert_live_event(
        self,
        event_type: str,
        event_data: dict[str, Any],
        parent_event_id: str | None = None,
        session_id: str | None = None,
        spawner_type: str | None = None,
    ) -> int | None:
        """
        Insert a live event for real-time WebSocket streaming.

        These events are temporary and should be cleaned up after broadcast.

        Args:
            event_type: Type of live event (spawner_start, spawner_phase, etc.)
            event_data: Event payload as dictionary (will be JSON serialized)
            parent_event_id: Parent event ID for hierarchical linking (optional)
            session_id: Session this event belongs to (optional)
            spawner_type: Spawner type (gemini, codex, copilot) if applicable (optional)

        Returns:
            Live event ID if successful, None otherwise
        """
        if not self.connection:
            self.connect()

        try:
            cursor = self.connection.cursor()  # type: ignore[union-attr]
            cursor.execute(
                """
                INSERT INTO live_events
                (event_type, event_data, parent_event_id, session_id, spawner_type)
                VALUES (?, ?, ?, ?, ?)
            """,
                (
                    event_type,
                    json.dumps(event_data),
                    parent_event_id,
                    session_id,
                    spawner_type,
                ),
            )
            self.connection.commit()  # type: ignore[union-attr]
            return cursor.lastrowid
        except sqlite3.Error as e:
            logger.error(f"Error inserting live event: {e}")
            return None

    def get_pending_live_events(self, limit: int = 100) -> list[dict[str, Any]]:
        """
        Get live events that haven't been broadcast yet.

        Args:
            limit: Maximum number of events to return

        Returns:
            List of pending live event dictionaries
        """
        if not self.connection:
            self.connect()

        try:
            cursor = self.connection.cursor()  # type: ignore[union-attr]
            cursor.execute(
                """
                SELECT id, event_type, event_data, parent_event_id, session_id,
                       spawner_type, created_at
                FROM live_events
                WHERE broadcast_at IS NULL
                ORDER BY created_at ASC
                LIMIT ?
            """,
                (limit,),
            )

            rows = cursor.fetchall()
            events = []
            for row in rows:
                event = dict(row)
                # Parse JSON event_data
                if event.get("event_data"):
                    try:
                        event["event_data"] = json.loads(event["event_data"])
                    except json.JSONDecodeError:
                        pass
                events.append(event)
            return events
        except sqlite3.Error as e:
            logger.error(f"Error fetching pending live events: {e}")
            return []

    def mark_live_events_broadcast(self, event_ids: list[int]) -> bool:
        """
        Mark live events as broadcast (sets broadcast_at timestamp).

        Args:
            event_ids: List of live event IDs to mark as broadcast

        Returns:
            True if successful, False otherwise
        """
        if not self.connection or not event_ids:
            return False

        try:
            cursor = self.connection.cursor()  # type: ignore[union-attr]
            placeholders = ",".join("?" for _ in event_ids)
            cursor.execute(
                f"""
                UPDATE live_events
                SET broadcast_at = CURRENT_TIMESTAMP
                WHERE id IN ({placeholders})
            """,
                event_ids,
            )
            self.connection.commit()  # type: ignore[union-attr]
            return True
        except sqlite3.Error as e:
            logger.error(f"Error marking live events as broadcast: {e}")
            return False

    def cleanup_old_live_events(self, max_age_minutes: int = 5) -> int:
        """
        Delete live events that have been broadcast and are older than max_age_minutes.

        Args:
            max_age_minutes: Maximum age in minutes for broadcast events

        Returns:
            Number of deleted events
        """
        if not self.connection:
            self.connect()

        try:
            cursor = self.connection.cursor()  # type: ignore[union-attr]
            cutoff = (
                datetime.now(timezone.utc) - timedelta(minutes=max_age_minutes)
            ).isoformat()
            cursor.execute(
                """
                DELETE FROM live_events
                WHERE broadcast_at IS NOT NULL
                  AND created_at < ?
            """,
                (cutoff,),
            )
            deleted_count = cursor.rowcount
            self.connection.commit()  # type: ignore[union-attr]
            return deleted_count
        except sqlite3.Error as e:
            logger.error(f"Error cleaning up old live events: {e}")
            return 0

    # ------------------------------------------------------------------
    # Collaboration & Delegation
    # ------------------------------------------------------------------

    def record_collaboration(
        self,
        handoff_id: str,
        from_agent: str,
        to_agent: str,
        session_id: str,
        feature_id: str | None = None,
        handoff_type: str = "delegation",
        reason: str | None = None,
        context: dict[str, Any] | None = None,
    ) -> bool:
        """
        Record an agent handoff or collaboration event.

        Args:
            handoff_id: Unique handoff identifier
            from_agent: Agent handing off work
            to_agent: Agent receiving work
            session_id: Session this handoff occurs in
            feature_id: Feature being handed off (optional)
            handoff_type: Type of handoff (delegation, parallel, sequential, fallback)
            reason: Reason for handoff (optional)
            context: Additional context (optional)

        Returns:
            True if record successful, False otherwise
        """
        if not self.connection:
            self.connect()

        try:
            cursor = self.connection.cursor()  # type: ignore[union-attr]
            cursor.execute(
                """
                INSERT INTO agent_collaboration
                (handoff_id, from_agent, to_agent, session_id, feature_id,
                 handoff_type, reason, context)
                VALUES (?, ?, ?, ?, ?, ?, ?, ?)
            """,
                (
                    handoff_id,
                    from_agent,
                    to_agent,
                    session_id,
                    feature_id,
                    handoff_type,
                    reason,
                    json.dumps(context) if context else None,
                ),
            )
            self.connection.commit()  # type: ignore[union-attr]
            return True
        except sqlite3.Error as e:
            logger.error(f"Error recording collaboration: {e}")
            return False

    def record_delegation_event(
        self,
        from_agent: str,
        to_agent: str,
        task_description: str,
        session_id: str | None = None,
        feature_id: str | None = None,
        context: dict[str, Any] | None = None,
    ) -> str | None:
        """
        Record a delegation event from one agent to another.

        Convenience method wrapping record_collaboration with sensible defaults
        for Task() delegation tracking. Handles foreign key constraints by
        creating a placeholder session if it doesn't exist.

        Args:
            from_agent: Agent delegating work
            to_agent: Agent receiving work
            task_description: Description of the delegated task
            session_id: Session this delegation occurs in (optional, auto-creates)
            feature_id: Feature being delegated (optional)
            context: Additional metadata (optional)

        Returns:
            Handoff ID if successful, None otherwise
        """
        import uuid

        if not self.connection:
            self.connect()

        if not session_id:
            session_id = f"session-{uuid.uuid4().hex[:8]}"

        self._ensure_session_exists(session_id, from_agent)

        handoff_id = f"hand-{uuid.uuid4().hex[:8]}"

        delegation_context = context or {}
        delegation_context["task_description"] = task_description

        success = self.record_collaboration(
            handoff_id=handoff_id,
            from_agent=from_agent,
            to_agent=to_agent,
            session_id=session_id,
            feature_id=feature_id,
            handoff_type="delegation",
            reason=task_description,
            context=delegation_context,
        )

        return handoff_id if success else None

    def get_delegations(
        self,
        session_id: str | None = None,
        from_agent: str | None = None,
        to_agent: str | None = None,
        limit: int = 100,
    ) -> list[dict[str, Any]]:
        """
        Query delegation events from agent_collaboration table.

        Args:
            session_id: Filter by session (optional)
            from_agent: Filter by source agent (optional)
            to_agent: Filter by target agent (optional)
            limit: Maximum number of results

        Returns:
            List of delegation events as dictionaries
        """
        if not self.connection:
            self.connect()

        try:
            cursor = self.connection.cursor()  # type: ignore[union-attr]

            where_clauses = ["handoff_type = 'delegation'"]
            params: list[str | int] = []

            if session_id:
                where_clauses.append("session_id = ?")
                params.append(session_id)
            if from_agent:
                where_clauses.append("from_agent = ?")
                params.append(from_agent)
            if to_agent:
                where_clauses.append("to_agent = ?")
                params.append(to_agent)

            where_sql = " AND ".join(where_clauses)

            cursor.execute(
                f"""
                SELECT
                    handoff_id,
                    from_agent,
                    to_agent,
                    session_id,
                    feature_id,
                    handoff_type,
                    reason,
                    context,
                    timestamp
                FROM agent_collaboration
                WHERE {where_sql}
                ORDER BY datetime(REPLACE(SUBSTR(timestamp, 1, 19), 'T', ' ')) DESC
                LIMIT ?
            """,
                params + [limit],
            )

            rows = cursor.fetchall()

            delegations = []
            for row in rows:
                row_dict = dict(row)
                delegations.append(row_dict)

            return delegations
        except sqlite3.Error as e:
            logger.error(f"Error querying delegations: {e}")
            return []

    def insert_collaboration(
        self,
        handoff_id: str,
        from_agent: str,
        to_agent: str,
        session_id: str,
        handoff_type: str = "delegation",
        reason: str | None = None,
        context: dict[str, Any] | None = None,
        status: str = "pending",
    ) -> bool:
        """
        Record an agent collaboration/delegation event.

        Args:
            handoff_id: Unique handoff identifier
            from_agent: Agent initiating the handoff
            to_agent: Target agent receiving the task
            session_id: Session this handoff belongs to
            handoff_type: Type of handoff (delegation, parallel, sequential, fallback)
            reason: Reason for the handoff (optional)
            context: Additional metadata as JSON (optional)
            status: Status of the handoff (pending, accepted, rejected, completed, failed)

        Returns:
            True if insert successful, False otherwise
        """
        if not self.connection:
            self.connect()

        try:
            cursor = self.connection.cursor()  # type: ignore[union-attr]
            cursor.execute(
                """
                INSERT INTO agent_collaboration
                (handoff_id, from_agent, to_agent, session_id, handoff_type,
                 reason, context, status)
                VALUES (?, ?, ?, ?, ?, ?, ?, ?)
            """,
                (
                    handoff_id,
                    from_agent,
                    to_agent,
                    session_id,
                    handoff_type,
                    reason,
                    json.dumps(context) if context else None,
                    status,
                ),
            )
            self.connection.commit()  # type: ignore[union-attr]
            return True
        except sqlite3.Error as e:
            logger.error(f"Error inserting collaboration record: {e}")
            return False

    # ------------------------------------------------------------------
    # Sync Operations
    # ------------------------------------------------------------------

    def insert_sync_operation(
        self,
        sync_id: str,
        operation: str,
        status: str,
        timestamp: str,
        files_changed: int = 0,
        conflicts: list[str] | None = None,
        message: str | None = None,
        hostname: str | None = None,
    ) -> bool:
        """
        Record a sync operation in the database.

        Args:
            sync_id: Unique sync operation ID
            operation: Operation type (push, pull)
            status: Sync status (idle, pushing, pulling, success, error, conflict)
            timestamp: Operation timestamp
            files_changed: Number of files changed
            conflicts: List of conflicted files (optional)
            message: Status message (optional)
            hostname: Hostname that performed the sync (optional)

        Returns:
            True if insert successful, False otherwise
        """
        if not self.connection:
            self.connect()

        try:
            cursor = self.connection.cursor()  # type: ignore[union-attr]
            cursor.execute(
                """
                INSERT INTO sync_operations
                (sync_id, operation, status, timestamp, files_changed, conflicts,
                 message, hostname)
                VALUES (?, ?, ?, ?, ?, ?, ?, ?)
            """,
                (
                    sync_id,
                    operation,
                    status,
                    timestamp,
                    files_changed,
                    json.dumps(conflicts) if conflicts else None,
                    message,
                    hostname,
                ),
            )
            self.connection.commit()  # type: ignore[union-attr]
            return True
        except sqlite3.Error as e:
            logger.error(f"Error inserting sync operation: {e}")
            return False

    def get_sync_operations(
        self, limit: int = 100, operation: str | None = None
    ) -> list[dict[str, Any]]:
        """
        Get recent sync operations.

        Args:
            limit: Maximum number of results
            operation: Filter by operation type (optional)

        Returns:
            List of sync operation dictionaries
        """
        if not self.connection:
            self.connect()

        try:
            cursor = self.connection.cursor()  # type: ignore[union-attr]

            if operation:
                cursor.execute(
                    """
                    SELECT * FROM sync_operations
                    WHERE operation = ?
                    ORDER BY datetime(REPLACE(SUBSTR(timestamp, 1, 19), 'T', ' ')) DESC
                    LIMIT ?
                """,
                    (operation, limit),
                )
            else:
                cursor.execute(
                    """
                    SELECT * FROM sync_operations
                    ORDER BY datetime(REPLACE(SUBSTR(timestamp, 1, 19), 'T', ' ')) DESC
                    LIMIT ?
                """,
                    (limit,),
                )

            rows = cursor.fetchall()
            results = []
            for row in rows:
                row_dict = dict(row)
                if row_dict.get("conflicts"):
                    try:
                        row_dict["conflicts"] = json.loads(row_dict["conflicts"])
                    except json.JSONDecodeError:
                        pass
                results.append(row_dict)
            return results
        except sqlite3.Error as e:
            logger.error(f"Error querying sync operations: {e}")
            return []
