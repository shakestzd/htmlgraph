#!/usr/bin/env python3
"""
InstructionsLoaded Hook Handler - Record instruction file loading events.

This hook fires when Claude Code loads instruction files (CLAUDE.md, etc.).
Recording which files were loaded helps track project configuration and
understand which instructions are active in a given session.

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
    Handle InstructionsLoaded events.

    Args:
        hook_input: Hook input containing:
            - session_id: Current session ID
            - files: List of instruction file paths loaded (if available)
            - type: Type of instruction loading event

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

        # Extract file paths - Claude Code may provide these in various fields
        files: list[str] = []
        raw_files = hook_input.get("files") or hook_input.get("instruction_files", [])
        if isinstance(raw_files, list):
            files = [str(f) for f in raw_files]
        elif isinstance(raw_files, str) and raw_files:
            files = [raw_files]

        instruction_type = hook_input.get("type", "unknown")
        file_count = len(files)
        file_summary = ", ".join(files[:3]) if files else "none"
        if file_count > 3:
            file_summary += f" (+{file_count - 3} more)"

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
                tool_name="InstructionsLoaded",
                input_summary=f"Loaded {file_count} instruction file(s): {file_summary}",
                output_summary="Instructions recorded",
                context={
                    "files": files,
                    "file_count": file_count,
                    "instruction_type": instruction_type,
                    "timestamp": datetime.now(timezone.utc).isoformat(),
                },
            )
            logger.debug(f"Recorded InstructionsLoaded event: {event_id}")

    except Exception as e:
        logger.warning(f"InstructionsLoaded hook error (non-blocking): {e}")

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
