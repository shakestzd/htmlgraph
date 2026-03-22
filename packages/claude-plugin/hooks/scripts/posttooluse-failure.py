#!/usr/bin/env -S uv run
# /// script
# requires-python = ">=3.10"
# dependencies = [
#   "htmlgraph>=0.34.13",
# ]
# ///
"""
PostToolUseFailure Hook - Track Tool Crashes and Exceptions

This hook is triggered when tool executions CRASH or throw exceptions,
as opposed to PostToolUse which handles successful tool executions.

Key differences:
- PostToolUse: Tool executed successfully (may contain error in response)
- PostToolUseFailure: Tool crashed/threw exception (execution failed)

This script is a thin wrapper that delegates to the existing
post_tool_use_failure.run() implementation in the htmlgraph package.

CRITICAL REQUIREMENTS:
- MUST exit with code 0 (never block Claude)
- MUST execute quickly (< 1 second)
- MUST handle all exceptions gracefully
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
    from htmlgraph.hooks.post_tool_use_failure import run as track_failure
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
    Main entry point - delegate to SDK track_failure function.

    The SDK handles:
    - Error logging to .htmlgraph/errors.jsonl
    - Pattern detection for recurring errors (3+ occurrences)
    - Automatic debug spike creation
    - Error context preservation
    """
    # Read hook input from stdin
    try:
        hook_input = json.load(sys.stdin)
    except json.JSONDecodeError:
        hook_input = {}

    # Delegate to SDK track_failure function
    # This function handles ALL error tracking logic
    try:
        response = track_failure(hook_input)
        print(json.dumps(response))
    except Exception as e:
        print(f"Error: track_failure failed: {e}", file=sys.stderr)
        import traceback

        traceback.print_exc(file=sys.stderr)
        # CRITICAL: Always exit 0 to never block Claude
        print(json.dumps({"continue": True}))


if __name__ == "__main__":
    main()
    sys.exit(0)  # Always exit 0
