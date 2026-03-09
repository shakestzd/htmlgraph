from __future__ import annotations

"""
Discovery Logic for HtmlGraph SDK

Auto-discovers project root and .htmlgraph directory.
"""


import os
from pathlib import Path

from htmlgraph.agent_detection import detect_agent_name

# Default directory name for HtmlGraph data
HTMLGRAPH_DIR = ".htmlgraph"


def find_project_root(start_path: Path | None = None) -> Path:
    """
    Find project root by searching for .htmlgraph directory.

    Searches environment variables first, then current directory and all
    parent directories.

    Args:
        start_path: Starting path for search (defaults to cwd)

    Returns:
        Path to directory containing .htmlgraph

    Raises:
        FileNotFoundError: If no .htmlgraph directory found
    """
    # Check env vars first — ensures subagents inherit correct project root
    env_dir = os.environ.get("CLAUDE_PROJECT_DIR") or os.environ.get(
        "HTMLGRAPH_PROJECT_DIR"
    )
    if env_dir:
        candidate = Path(env_dir)
        if (candidate / HTMLGRAPH_DIR).exists():
            return candidate

    current = start_path or Path.cwd()

    # Check current directory
    if (current / HTMLGRAPH_DIR).exists():
        return current

    # Check parent directories
    for parent in current.parents:
        if (parent / HTMLGRAPH_DIR).exists():
            return parent

    # Not found
    raise FileNotFoundError(
        f"Could not find {HTMLGRAPH_DIR} directory in {current} or any parent directory"
    )


def discover_htmlgraph_dir(start_path: Path | None = None) -> Path:
    """
    Auto-discover .htmlgraph directory.

    Checks environment variables first (CLAUDE_PROJECT_DIR or
    HTMLGRAPH_PROJECT_DIR), then searches current directory and parents.
    If not found anywhere, returns path to .htmlgraph in current directory
    (may not exist yet).

    Args:
        start_path: Starting path for search (defaults to cwd)

    Returns:
        Path to .htmlgraph directory
    """
    # Check env vars first — ensures subagents spawned via Task() use the
    # parent project's .htmlgraph rather than defaulting to cwd or home.
    env_dir = os.environ.get("CLAUDE_PROJECT_DIR") or os.environ.get(
        "HTMLGRAPH_PROJECT_DIR"
    )
    if env_dir:
        candidate = Path(env_dir) / HTMLGRAPH_DIR
        if candidate.exists():
            return candidate

    current = start_path or Path.cwd()

    # Check current directory
    if (current / HTMLGRAPH_DIR).exists():
        return current / HTMLGRAPH_DIR

    # Check parent directories
    for parent in current.parents:
        if (parent / HTMLGRAPH_DIR).exists():
            return parent / HTMLGRAPH_DIR

    # Default to current directory (may not exist yet)
    return current / HTMLGRAPH_DIR


def auto_discover_agent() -> str:
    """
    Auto-discover agent identifier from environment.

    Detection order:
        1. CLAUDE_AGENT_NAME environment variable
        2. detect_agent_name() from agent_detection module
        3. Raises ValueError if no valid agent found

    Returns:
        Agent identifier string

    Raises:
        ValueError: If agent cannot be detected
    """
    # Try environment variable first
    agent = os.getenv("CLAUDE_AGENT_NAME")
    if agent:
        return agent

    # Try automatic detection
    detected = detect_agent_name()
    if detected and detected != "cli":
        # Only accept detected if it's not the default fallback
        return detected

    # No valid agent found - fail fast with helpful error message
    raise ValueError(
        "Agent identifier is required for work attribution. "
        "Pass agent='name' to SDK() initialization. "
        "Examples: SDK(agent='explorer'), SDK(agent='coder'), SDK(agent='tester')\n"
        "Alternatively, set CLAUDE_AGENT_NAME environment variable.\n"
        "Critical for: Work attribution, result retrieval, orchestrator tracking"
    )


__all__ = [
    "HTMLGRAPH_DIR",
    "find_project_root",
    "discover_htmlgraph_dir",
    "auto_discover_agent",
]
