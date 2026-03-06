#!/usr/bin/env -S uv run
# /// script
# requires-python = ">=3.10"
# dependencies = [
#   "htmlgraph",
# ]
# ///
"""
HtmlGraph PreCompact Hook (Thin Wrapper)

Fires before Claude Code compacts the conversation context.
All business logic lives in htmlgraph.hooks.pre_compact.

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
    from htmlgraph.hooks.pre_compact import run as handle_pre_compact
except Exception as e:
    print(
        f"Warning: HtmlGraph not available ({e}). Install with: pip install htmlgraph",
        file=sys.stderr,
    )
    print(json.dumps({"continue": True}))
    sys.exit(0)


def main() -> None:
    """Main entry point - delegate to SDK handler."""
    try:
        hook_input = json.load(sys.stdin)
    except json.JSONDecodeError:
        hook_input = {}

    try:
        response = handle_pre_compact(hook_input)
        print(json.dumps(response))
    except Exception as e:
        print(f"Error: pre_compact handler failed: {e}", file=sys.stderr)
        import traceback

        traceback.print_exc(file=sys.stderr)
        print(json.dumps({"continue": True}))


if __name__ == "__main__":
    main()
    sys.exit(0)
