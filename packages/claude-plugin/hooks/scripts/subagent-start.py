#!/usr/bin/env -S uv run
# /// script
# requires-python = ">=3.10"
# dependencies = [
#   "htmlgraph>=0.34.13",
# ]
# ///
"""
HtmlGraph SubagentStart Hook - Map agent_id to Task Delegations

This hook fires when a subagent spawned via Task() starts.
It maps the unique agent_id to the corresponding task_delegation event,
enabling correct parent attribution for parallel subagents.

Hook input provides:
- agent_id: Unique identifier for this subagent (e.g., "agent-abc123")
- agent_type: Agent type name (e.g., "Explore", "general-purpose")

Thin wrapper around SDK subagent_start module. All business logic lives in:
    htmlgraph.hooks.subagent_start
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
    from htmlgraph.hooks.subagent_start import handle_subagent_start
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
    result = handle_subagent_start(hook_input)

    # Output response
    print(json.dumps(result))


if __name__ == "__main__":
    main()
