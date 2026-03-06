#!/usr/bin/env python3
"""
PermissionRequest Hook Handler - Record permission request events.

This hook fires when Claude Code requests permission to perform an action.
Recording permission requests helps track what actions were authorized or
denied during a session, useful for auditing and understanding tool usage.

CRITICAL REQUIREMENTS:
- MUST exit with code 0 (exit 1 blocks Claude)
- MUST execute quickly (< 1 second)
- MUST handle all exceptions gracefully
"""

import json
import logging
import os
import sys
from datetime import datetime, timezone
from typing import Any

logger = logging.getLogger(__name__)


def run(hook_input: dict[str, Any]) -> dict[str, Any]:
    """
    Handle PermissionRequest events.

    Args:
        hook_input: Hook input containing:
            - session_id: Current session ID
            - tool_name: The tool requesting permission
            - tool_input: The input parameters for the tool
            - action: The action being requested

    Returns:
        Standard hook response: {"continue": True}
    """
    try:
        from htmlgraph.hooks.db_helpers import (
            ensure_session_exists,
            get_current_session_id,
            get_db,
            resolve_project_path,
        )
        from htmlgraph.ids import generate_id

        session_id = hook_input.get("session_id") or get_current_session_id()

        # Extract permission request details
        # Claude Code may use different field names for the tool/action
        tool_name = (
            hook_input.get("tool_name")
            or hook_input.get("tool")
            or hook_input.get("name", "unknown")
        )
        tool_input = hook_input.get("tool_input") or hook_input.get("input", {})
        action = hook_input.get("action", "unknown")

        # Build a summary of what's being requested
        if isinstance(tool_input, dict):
            # Extract key fields for summary (avoid logging sensitive values)
            input_keys = list(tool_input.keys())[:5]
            input_summary = f"tool={tool_name}, fields={input_keys}"
        else:
            input_summary = f"tool={tool_name}, action={action}"

        project_path = resolve_project_path()
        db = get_db(project_path)

        if db and session_id:
            ensure_session_exists(db, session_id)
            event_id = generate_id("event")
            db.insert_event(
                event_id=event_id,
                agent_id="claude-code",
                event_type="check_point",
                session_id=session_id,
                tool_name="PermissionRequest",
                input_summary=input_summary,
                output_summary="Permission request recorded",
                context={
                    "requested_tool": tool_name,
                    "action": action,
                    "timestamp": datetime.now(timezone.utc).isoformat(),
                },
            )
            logger.debug(f"Recorded PermissionRequest event: {event_id}")

    except Exception as e:
        logger.warning(f"PermissionRequest hook error (non-blocking): {e}")

    return {"continue": True}


def main() -> None:
    """Hook entry point for script wrapper."""
    if os.environ.get("HTMLGRAPH_DISABLE_TRACKING") == "1":
        print(json.dumps({"continue": True}))
        sys.exit(0)

    try:
        hook_input = json.load(sys.stdin)
    except json.JSONDecodeError:
        hook_input = {}

    result = run(hook_input)
    print(json.dumps(result))
    sys.exit(0)


if __name__ == "__main__":
    main()
