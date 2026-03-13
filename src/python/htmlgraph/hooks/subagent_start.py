"""
SubagentStart Hook - Map agent_id to task_delegation events.

When a subagent starts, Claude Code provides a unique agent_id.
This module maps that agent_id to the corresponding task_delegation event
using FIFO matching (first unmatched task_delegation gets the first SubagentStart).

This enables correct parent attribution for parallel subagents that share
the same subagent_type (e.g., two "general-purpose" Tasks running simultaneously).
"""

import logging
import sqlite3
from typing import Any

logger = logging.getLogger(__name__)


def handle_subagent_start(hook_input: dict[str, Any]) -> dict[str, Any]:
    """
    Handle SubagentStart hook event.

    Maps agent_id from Claude Code to the most recent unmatched task_delegation
    event using FIFO ordering.

    Args:
        hook_input: Hook input from Claude Code containing agent_id and agent_type

    Returns:
        Response: {"continue": True}
    """
    agent_id = hook_input.get("agent_id")
    agent_type = hook_input.get("agent_type", "")
    session_id = hook_input.get("session_id") or ""

    if not agent_id:
        logger.debug("SubagentStart: No agent_id provided, skipping")
        return {"continue": True}

    logger.info(
        f"SubagentStart: agent_id={agent_id}, agent_type={agent_type}, session_id={session_id}"
    )

    # Get database path
    try:
        from htmlgraph.config import get_database_path

        db_path = str(get_database_path())
    except Exception as e:
        logger.warning(f"Could not get database path: {e}")
        return {"continue": True}

    # Map agent_id to unmatched task_delegation
    try:
        conn = sqlite3.connect(db_path, timeout=2.0)
        cursor = conn.cursor()

        # Find the earliest unmatched task_delegation (no real agent_id yet)
        # that matches the agent_type, ordered by timestamp ASC (FIFO).
        # PreToolUse writes agent_id='claude-code' as a placeholder; treat that
        # as "unmatched" alongside NULL/''.
        #
        # SESSION_ID CORRECTNESS NOTE (roborev-259, Finding 1):
        # Claude Code passes the PARENT/orchestrator session_id to SubagentStart hooks,
        # NOT a separate subagent session_id. This is confirmed by Claude Code behavior:
        # "All subagents share the same session_id" (see subagent-stop.py plugin docs).
        # Since task_delegation rows are written by PreToolUse in the parent session,
        # they carry the same session_id. Therefore this filter correctly scopes to
        # the current orchestrator session, avoiding stale rows from old sessions.
        if session_id:
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
        else:
            cursor.execute(
                """
                SELECT event_id, subagent_type FROM agent_events
                WHERE event_type = 'task_delegation'
                  AND status = 'started'
                  AND (agent_id IS NULL OR agent_id = '' OR agent_id = 'claude-code')
                ORDER BY timestamp ASC
                """,
            )
        rows = cursor.fetchall()

        matched_event_id = None

        # First pass: try to match by agent_type == subagent_type
        for row in rows:
            event_id, subagent_type = row[0], row[1]
            if (
                subagent_type
                and agent_type
                and subagent_type.lower() == agent_type.lower()
            ):
                matched_event_id = event_id
                break

        # Second pass: if no type match, take the first unmatched one
        if not matched_event_id and rows:
            matched_event_id = rows[0][0]

        if matched_event_id:
            cursor.execute(
                """
                UPDATE agent_events
                SET agent_id = ?, updated_at = CURRENT_TIMESTAMP
                WHERE event_id = ?
                """,
                (agent_id, matched_event_id),
            )
            conn.commit()
            logger.info(
                f"SubagentStart: Mapped agent_id={agent_id} to task_delegation={matched_event_id}"
            )
        else:
            logger.warning(
                f"SubagentStart: No unmatched task_delegation found for agent_id={agent_id}"
            )

        conn.close()

    except Exception as e:
        logger.warning(f"SubagentStart: Error mapping agent_id: {e}")

    return {"continue": True}
