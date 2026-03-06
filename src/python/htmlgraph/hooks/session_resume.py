#!/usr/bin/env python3
"""
SessionResume Hook Handler - Record session resume events.

This hook fires when Claude Code resumes an existing conversation session
(SessionStart with "resume" matcher). Recording resume events helps
distinguish fresh sessions from resumed ones and track session continuity.

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
    Handle SessionStart resume events.

    Args:
        hook_input: Hook input containing:
            - session_id: Current session ID being resumed
            - cwd: Current working directory

    Returns:
        Standard hook response: {"continue": True}
    """
    try:
        from htmlgraph.hooks.db_helpers import (
            ensure_session_exists,
            get_db,
            resolve_project_path,
        )
        from htmlgraph.ids import generate_id

        session_id = hook_input.get("session_id", "unknown")
        cwd = hook_input.get("cwd")

        project_path = resolve_project_path(cwd)
        db = get_db(project_path)

        if db and session_id and session_id != "unknown":
            # Ensure session record exists (may have been created in a prior process)
            ensure_session_exists(db, session_id)

            event_id = generate_id("event")
            db.insert_event(
                event_id=event_id,
                agent_id="claude-code",
                event_type="check_point",
                session_id=session_id,
                tool_name="SessionResume",
                input_summary=f"Session resumed: {session_id}",
                output_summary="Resume recorded",
                context={
                    "session_id": session_id,
                    "is_resume": True,
                    "timestamp": datetime.now(timezone.utc).isoformat(),
                },
            )
            logger.debug(f"Recorded SessionResume event: {event_id}")

            # Update session metadata to mark as resumed
            try:
                cursor = db.connection.cursor()
                cursor.execute(
                    """
                    UPDATE sessions
                    SET status = 'active'
                    WHERE session_id = ?
                    """,
                    (session_id,),
                )
                db.connection.commit()
            except Exception as e:
                logger.debug(f"Could not update session status: {e}")

    except Exception as e:
        logger.warning(f"SessionResume hook error (non-blocking): {e}")

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
