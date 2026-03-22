#!/usr/bin/env -S uv run
# /// script
# requires-python = ">=3.10"
# dependencies = [
#   "htmlgraph>=0.34.13",
# ]
# ///
"""
HtmlGraph SubagentStop Hook - Capture Delegated Task Completions

This hook fires when a subagent spawned via Task() completes.
It updates parent events with completion status and child spike counts.

Thin wrapper around SDK subagent_stop module. All business logic lives in:
    htmlgraph.hooks.subagent_stop

KNOWN LIMITATIONS (GitHub issue #7881, #14859):
- SubagentStop cannot identify which specific subagent finished
- All subagents share the same session_id
- No agent_id, parent_id, or subagent_type fields available

WORKAROUND:
- SDK queries database for most recent task_delegation with status='started'
- Updates parent event: status="completed", child_spike_count=N
- Handles graceful degradation if parent event not found
"""

import json
import os
import sys

# Bootstrap Python path and setup
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
from bootstrap import bootstrap_pythonpath, is_tracking_disabled, resolve_project_dir

# Skip tracking if disabled
if is_tracking_disabled():
    print(json.dumps({"continue": True}))
    sys.exit(0)

project_dir_for_import = resolve_project_dir()
bootstrap_pythonpath(project_dir_for_import)

try:
    from htmlgraph.hooks.subagent_stop import handle_subagent_stop
    from htmlgraph.hooks.version_check import check_hook_version

    check_hook_version("0.34.14")
except Exception as e:
    print(
        f"Warning: HtmlGraph not available ({e}). Install with: pip install htmlgraph",
        file=sys.stderr,
    )
    print(json.dumps({"continue": True}))
    sys.exit(0)


def main() -> None:
    """Main hook entry point - delegates to SDK handler."""
    try:
        hook_input = json.load(sys.stdin)
    except json.JSONDecodeError:
        hook_input = {}

    # Delegate entirely to SDK handler
    result = handle_subagent_stop(hook_input)

    # Output response
    print(json.dumps(result))


if __name__ == "__main__":
    main()
