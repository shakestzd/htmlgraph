"""Hook version validation — warns when uv cache is stale."""

from __future__ import annotations

import sys


def check_hook_version(required: str) -> None:
    """Check that installed htmlgraph meets the minimum version requirement.

    Prints a warning to stderr if the installed version is older than
    ``required``. Uses simple tuple comparison to avoid needing the
    ``packaging`` library.

    Args:
        required: Minimum version string, e.g. "0.34.14"
    """
    try:
        from importlib.metadata import version as get_version

        installed = get_version("htmlgraph")
        # Simple tuple comparison works for our MAJOR.MINOR.PATCH scheme
        inst_parts = tuple(int(x) for x in installed.split(".")[:3])
        req_parts = tuple(int(x) for x in required.split(".")[:3])
        if inst_parts < req_parts:
            print(
                f"WARNING: Stale hook cache: htmlgraph {installed} < {required}. "
                f"Run: uv cache clean htmlgraph && restart Claude Code",
                file=sys.stderr,
            )
    except Exception:
        pass  # Never break hook execution over version checking
