#!/usr/bin/env python3
"""
PreCompact Hook Handler - Record conversation compaction events.

This hook fires before Claude Code compacts the conversation context.
Recording these events helps track when context windows are compressed,
which can affect conversation continuity.

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
    Handle PreCompact events.

    Args:
        hook_input: Hook input containing:
            - session_id: Current session ID
            - trigger: What triggered the compaction (e.g., "manual", "auto")
            - custom_instructions: Any custom instructions provided

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
        trigger = hook_input.get("trigger", "unknown")
        custom_instructions = hook_input.get("custom_instructions", "")

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
                tool_name="PreCompact",
                input_summary=f"Compact triggered: {trigger}",
                output_summary="Compaction recorded",
                context={
                    "trigger": trigger,
                    "has_custom_instructions": bool(custom_instructions),
                    "timestamp": datetime.now(timezone.utc).isoformat(),
                },
            )
            logger.debug(f"Recorded PreCompact event: {event_id}")

    except Exception as e:
        logger.warning(f"PreCompact hook error (non-blocking): {e}")

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
