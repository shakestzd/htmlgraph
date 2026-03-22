#!/usr/bin/env -S uv run
# /// script
# requires-python = ">=3.10"
# dependencies = [
#   "htmlgraph>=0.34.13",
# ]
# ///
"""
PostToolUse Hook - Thin wrapper around package logic.

This script is a minimal entry point that delegates all logic to the
htmlgraph.hooks.posttooluse package module, which runs event tracking
and orchestrator reflection in parallel.

Performance: ~40-50% faster than previous subprocess-based approach.
"""

import os
import sys

# Bootstrap Python path and setup
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
from bootstrap import bootstrap_pythonpath, resolve_project_dir

project_dir_for_import = resolve_project_dir()
bootstrap_pythonpath(project_dir_for_import)

try:
    from htmlgraph.hooks.version_check import check_hook_version

    check_hook_version("0.34.14")
except Exception:
    pass

from htmlgraph.hooks.posttooluse import main

if __name__ == "__main__":
    main()
