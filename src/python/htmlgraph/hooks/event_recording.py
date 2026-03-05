"""
Event Recording for HtmlGraph Hooks.

This module provides utilities for recording events to SQLite database,
including tool calls, delegations, and extracting metadata from tool inputs.

Public Functions:
    extract_file_paths(tool_input: dict[str, Any], tool_name: str) -> list[str]
        Extract file paths from tool input based on tool type

    format_tool_summary(tool_name: str, tool_input: dict[str, Any], tool_result: dict | None = None) -> str
        Format a human-readable summary of the tool call

    record_event_to_sqlite(...) -> str | None
        Record a tool call event to SQLite database for dashboard queries

    record_delegation_to_sqlite(...) -> str | None
        Record a Task() delegation to agent_collaboration table
"""

import json
import logging
import os
import re
from datetime import datetime, timedelta, timezone
from typing import Any

from htmlgraph.db.schema import HtmlGraphDB
from htmlgraph.ids import generate_id

logger = logging.getLogger(__name__)


def extract_file_paths(tool_input: dict[str, Any], tool_name: str) -> list[str]:
    """Extract file paths from tool input based on tool type."""
    paths = []

    # Common path fields
    for field in ["file_path", "path", "filepath"]:
        if field in tool_input:
            paths.append(tool_input[field])

    # Glob/Grep patterns
    if "pattern" in tool_input and tool_name in ["Glob", "Grep"]:
        pattern = tool_input.get("pattern", "")
        if "." in pattern:
            paths.append(f"pattern:{pattern}")

    # Bash commands - extract paths heuristically
    if tool_name == "Bash" and "command" in tool_input:
        cmd = tool_input["command"]
        file_matches = re.findall(r"[\w./\-_]+\.[a-zA-Z]{1,5}", cmd)
        paths.extend(file_matches[:3])

    return paths


def format_tool_summary(
    tool_name: str, tool_input: dict[str, Any], tool_result: dict | None = None
) -> str:
    """
    Format a human-readable summary of the tool call.

    Returns only the description part (without tool name prefix) since tool_name
    is stored as a separate field in the database. Frontend can format as needed.
    """
    if tool_name == "Read":
        path = str(tool_input.get("file_path", "unknown"))
        return path

    elif tool_name == "Write":
        path = str(tool_input.get("file_path", "unknown"))
        return path

    elif tool_name == "Edit":
        path = str(tool_input.get("file_path", "unknown"))
        old = str(tool_input.get("old_string", ""))[:30]
        return f"{path} ({old}...)"

    elif tool_name == "Bash":
        cmd = str(tool_input.get("command", ""))[:60]
        desc = str(tool_input.get("description", ""))
        if desc:
            return desc
        return cmd

    elif tool_name == "Glob":
        pattern = str(tool_input.get("pattern", ""))
        return pattern

    elif tool_name == "Grep":
        pattern = str(tool_input.get("pattern", ""))
        return pattern

    elif tool_name == "Task":
        desc = str(tool_input.get("description", ""))[:50]
        agent = str(tool_input.get("subagent_type", ""))
        return f"({agent}): {desc}"

    elif tool_name == "TodoWrite":
        todos = tool_input.get("todos", [])
        return f"{len(todos)} items"

    elif tool_name == "WebSearch":
        query = str(tool_input.get("query", ""))[:40]
        return query

    elif tool_name == "WebFetch":
        url = str(tool_input.get("url", ""))[:40]
        return url

    elif tool_name == "UserQuery":
        # Extract the actual prompt text from the tool_input
        prompt = str(tool_input.get("prompt", ""))
        preview = prompt[:100].replace("\n", " ")
        if len(prompt) > 100:
            preview += "..."
        return preview

    else:
        return str(tool_input)[:50]


def record_event_to_sqlite(
    db: HtmlGraphDB,
    session_id: str,
    tool_name: str,
    tool_input: dict[str, Any],
    tool_response: dict[str, Any],
    is_error: bool,
    file_paths: list[str] | None = None,
    parent_event_id: str | None = None,
    agent_id: str | None = None,
    subagent_type: str | None = None,
    model: str | None = None,
    feature_id: str | None = None,
    claude_task_id: str | None = None,
) -> str | None:
    """
    Record a tool call event to SQLite database for dashboard queries.

    Args:
        db: HtmlGraphDB instance
        session_id: Session ID from HtmlGraph
        tool_name: Name of the tool called
        tool_input: Tool input parameters
        tool_response: Tool response/result
        is_error: Whether the tool call resulted in an error
        file_paths: File paths affected by the tool
        parent_event_id: Parent event ID if this is a child event
        agent_id: Agent identifier (optional)
        subagent_type: Subagent type for Task delegations (optional)
        model: Claude model name (e.g., claude-haiku, claude-opus) (optional)
        feature_id: Feature ID for attribution (optional)
        claude_task_id: Claude Code's internal task ID for tool attribution (optional)

    Returns:
        event_id if successful, None otherwise
    """
    try:
        input_summary = format_tool_summary(tool_name, tool_input, tool_response)

        # Build output summary from tool response
        output_summary = ""
        if isinstance(tool_response, dict):  # type: ignore[arg-type]
            if is_error:
                output_summary = tool_response.get("error", "error")[:200]
            else:
                # Extract summary from response
                content = tool_response.get("content", tool_response.get("output", ""))
                if isinstance(content, str):
                    output_summary = content[:200]
                elif isinstance(content, list):
                    output_summary = f"{len(content)} items"
                else:
                    output_summary = "success"

        # If we have a parent event, inherit its model (child events inherit from parent Task)
        if parent_event_id and db and db.connection:
            try:
                cursor = db.connection.cursor()
                cursor.execute(
                    "SELECT model FROM agent_events WHERE event_id = ? LIMIT 1",
                    (parent_event_id,),
                )
                row = cursor.fetchone()
                if row and row[0]:
                    model = row[0]  # Inherit parent's model
            except Exception:
                pass

        # Build context metadata
        context = {
            "file_paths": file_paths or [],
            "tool_input_keys": list(tool_input.keys()),
            "is_error": is_error,
        }

        # Extract task_id from Tool response if not provided
        if (
            not claude_task_id
            and tool_name == "Task"
            and isinstance(tool_response, dict)
        ):
            claude_task_id = tool_response.get("task_id")

        # CRITICAL FIX: Check if a matching PreToolUse event exists and UPDATE it
        # instead of creating a duplicate entry
        existing_event_id = None

        # Priority 1: Check environment variable (set by PreToolUse)
        existing_event_id = os.environ.get("HTMLGRAPH_PRETOOL_EVENT_ID")

        # Priority 2: Query database for matching event
        if not existing_event_id and db.connection:
            cursor = db.connection.cursor()

            # Build matching criteria (priority order):
            # 1. Same session_id + tool_name + claude_task_id (if available)
            # 2. Same session_id + tool_name + input_summary (fallback)
            # 3. Within last 5 minutes
            # 4. Status is NOT 'completed' (hasn't been updated yet)

            five_minutes_ago = (
                datetime.now(timezone.utc) - timedelta(minutes=5)
            ).strftime("%Y-%m-%d %H:%M:%S")

            if claude_task_id:
                # Match by claude_task_id (most reliable)
                cursor.execute(
                    """
                    SELECT event_id FROM agent_events
                    WHERE session_id = ?
                      AND tool_name = ?
                      AND claude_task_id = ?
                      AND status != 'completed'
                      AND datetime(REPLACE(SUBSTR(timestamp, 1, 19), 'T', ' ')) >= datetime(?)
                    ORDER BY datetime(REPLACE(SUBSTR(timestamp, 1, 19), 'T', ' ')) DESC
                    LIMIT 1
                    """,
                    (session_id, tool_name, claude_task_id, five_minutes_ago),
                )
                row = cursor.fetchone()
                if row:
                    existing_event_id = row[0]

            if not existing_event_id:
                # Fallback: Match by input_summary (less reliable but works for most cases)
                cursor.execute(
                    """
                    SELECT event_id FROM agent_events
                    WHERE session_id = ?
                      AND tool_name = ?
                      AND input_summary = ?
                      AND status != 'completed'
                      AND datetime(REPLACE(SUBSTR(timestamp, 1, 19), 'T', ' ')) >= datetime(?)
                    ORDER BY datetime(REPLACE(SUBSTR(timestamp, 1, 19), 'T', ' ')) DESC
                    LIMIT 1
                    """,
                    (session_id, tool_name, input_summary, five_minutes_ago),
                )
                row = cursor.fetchone()
                if row:
                    existing_event_id = row[0]

        if existing_event_id:
            # UPDATE existing PreToolUse event with PostToolUse data
            if db.connection:
                cursor = db.connection.cursor()
                cursor.execute(
                    """
                    UPDATE agent_events
                    SET output_summary = ?,
                        status = 'completed',
                        updated_at = CURRENT_TIMESTAMP,
                        context = json(?)
                    WHERE event_id = ?
                    """,
                    (output_summary, json.dumps(context), existing_event_id),
                )
                db.connection.commit()
                logger.debug(
                    f"Updated existing event {existing_event_id} with PostToolUse data "
                    f"(tool={tool_name}, session={session_id})"
                )
                event_id = existing_event_id
                success = True

                # Clean up environment variable after use
                if "HTMLGRAPH_PRETOOL_EVENT_ID" in os.environ:
                    del os.environ["HTMLGRAPH_PRETOOL_EVENT_ID"]
        else:
            # No matching PreToolUse event found - INSERT new event (fallback)
            event_id = generate_id("event")
            success = db.insert_event(
                event_id=event_id,
                agent_id=agent_id or "claude-code",
                event_type="tool_call",
                session_id=session_id,
                tool_name=tool_name,
                input_summary=input_summary,
                tool_input=tool_input,  # CRITICAL: Pass tool_input for dashboard display
                output_summary=output_summary,
                context=context,
                parent_event_id=parent_event_id,
                cost_tokens=0,
                subagent_type=subagent_type,
                model=model,
                feature_id=feature_id,
                claude_task_id=claude_task_id,
            )
            logger.debug(
                f"Created new event {event_id} (no matching PreToolUse found) "
                f"(tool={tool_name}, session={session_id})"
            )

        if success:
            # Also insert into live_events for real-time WebSocket dashboard
            try:
                event_data = {
                    "tool": tool_name,
                    "summary": input_summary,
                    "success": not is_error,
                    "feature_id": feature_id,
                    "file_paths": file_paths,
                    "timestamp": datetime.now(timezone.utc).isoformat(),
                }

                db.insert_live_event(
                    event_type="tool_call",
                    event_data=event_data,
                    parent_event_id=parent_event_id,
                    session_id=session_id,
                    spawner_type=None,
                )
            except Exception as e:
                # Don't fail the hook if live event insertion fails
                logger.debug(f"Could not insert live event: {e}")

            return event_id
        return None

    except Exception as e:
        logger.warning(f"Warning: Could not record event to SQLite: {e}")
        return None


def record_delegation_to_sqlite(
    db: HtmlGraphDB,
    session_id: str,
    from_agent: str,
    to_agent: str,
    task_description: str,
    task_input: dict[str, Any],
) -> str | None:
    """
    Record a Task() delegation to agent_collaboration table.

    Args:
        db: HtmlGraphDB instance
        session_id: Session ID from HtmlGraph
        from_agent: Agent delegating the task (usually 'orchestrator' or 'claude-code')
        to_agent: Target subagent type (e.g., 'general-purpose', 'researcher')
        task_description: Task description/prompt
        task_input: Full task input parameters

    Returns:
        handoff_id if successful, None otherwise
    """
    try:
        handoff_id = generate_id("handoff")

        # Build context with task input
        context = {
            "task_input_keys": list(task_input.keys()),
            "model": task_input.get("model"),
            "temperature": task_input.get("temperature"),
        }

        # Insert delegation record
        success = db.insert_collaboration(
            handoff_id=handoff_id,
            from_agent=from_agent,
            to_agent=to_agent,
            session_id=session_id,
            handoff_type="delegation",
            reason=task_description[:200],
            context=context,
        )

        if success:
            return handoff_id
        return None

    except Exception as e:
        logger.warning(f"Warning: Could not record delegation to SQLite: {e}")
        return None


__all__ = [
    "extract_file_paths",
    "format_tool_summary",
    "record_event_to_sqlite",
    "record_delegation_to_sqlite",
]
