#!/usr/bin/env -S uv run
# /// script
# requires-python = ">=3.10"
# dependencies = [
#   "htmlgraph>=0.34.13",
# ]
# ///
"""
HtmlGraph Event Tracker - Unified Hook Script

Thin wrapper that delegates all event tracking to SDK track_event function.

All business logic lives in:
    htmlgraph.hooks.event_tracker.track_event()

This script simply reads hook input and delegates to the SDK.
"""

import json
import os
import sys

# Bootstrap Python path and setup
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
from bootstrap import bootstrap_pythonpath, is_tracking_disabled, resolve_project_dir

if is_tracking_disabled():
    print(json.dumps({"continue": True}))
    sys.exit(0)

project_dir_for_import = resolve_project_dir()
bootstrap_pythonpath(project_dir_for_import)

try:
    from htmlgraph.hooks.event_tracker import track_event
    from htmlgraph.hooks.version_check import check_hook_version

    check_hook_version("0.34.14")
except Exception as e:
    # Do not break Claude execution if the dependency isn't installed.
    print(
        f"Warning: HtmlGraph not available ({e}). Install with: pip install htmlgraph",
        file=sys.stderr,
    )
    print(json.dumps({"continue": True}))
    sys.exit(0)


def main() -> None:
    """
    Main entry point - delegate to SDK track_event function.

    The SDK handles:
    - Session management
    - SQLite database recording
    - Model detection
    - Parent-child event linking
    - Drift detection and classification
    - Live event WebSocket updates
    """
    # Get hook type from environment (set by hooks.json)
    hook_type = os.environ.get("HTMLGRAPH_HOOK_TYPE", "PostToolUse")

    # Read hook input from stdin
    try:
        hook_input = json.load(sys.stdin)
    except json.JSONDecodeError:
        hook_input = {}

    # Delegate to SDK track_event function
    # This function handles ALL event tracking logic
    try:
        response = track_event(hook_type=hook_type, hook_input=hook_input)
        print(json.dumps(response))
    except Exception as e:
        print(f"Error: track_event failed: {e}", file=sys.stderr)
        import traceback

        traceback.print_exc(file=sys.stderr)
        print(json.dumps({"continue": True}))


if __name__ == "__main__":
    main()
