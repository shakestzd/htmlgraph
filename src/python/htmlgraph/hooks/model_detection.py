"""
Model Detection for HtmlGraph Hooks.

This module provides utilities for detecting the Claude model being used during
hook execution. Since Claude Code doesn't pass model information directly to hooks,
we use multiple detection strategies with fallback chain.

Public Functions:
    get_model_from_status_cache(session_id: str | None = None) -> str | None
        Read current model from SQLite model_cache table

    normalize_model_name(model: str | None) -> str | None
        Convert any model format to consistent display format

    detect_model_from_hook_input(hook_input: dict[str, Any]) -> str | None
        Detect the Claude model from hook input data

    get_model_from_parent_event(db_path: str | None = None) -> str | None
        Look up the model from the parent Task delegation event

    detect_agent_from_environment() -> tuple[str, str | None]
        Detect the agent/model name from environment variables and status cache
"""

import logging
import os
import sqlite3
from pathlib import Path
from typing import Any

logger = logging.getLogger(__name__)


def get_model_from_status_cache(session_id: str | None = None) -> str | None:
    """
    Read current model from SQLite model_cache table.

    The status line script writes model info to the model_cache table.
    This allows hooks to know which Claude model is currently running,
    even though hooks don't receive model info directly from Claude Code.

    Args:
        session_id: Unused, kept for backward compatibility.

    Returns:
        Model display name (e.g., "Opus 4.5", "Sonnet", "Haiku") or None if not found.
    """
    try:
        # Try project database first
        db_path = Path.cwd() / ".htmlgraph" / "htmlgraph.db"
        if not db_path.exists():
            return None

        conn = sqlite3.connect(str(db_path), timeout=1.0)
        cursor = conn.cursor()

        # Check if model_cache table exists and has data
        cursor.execute("SELECT model FROM model_cache WHERE id = 1 LIMIT 1")
        row = cursor.fetchone()
        conn.close()

        if row and row[0] and row[0] != "Claude":
            return str(row[0])
        return str(row[0]) if row else None

    except Exception:
        # Table doesn't exist or read error - silently fail
        pass

    return None


def normalize_model_name(model: str | None) -> str | None:
    """Convert any model format to consistent display format."""
    if not model:
        return None
    model_lower = model.strip().lower()
    mapping = {
        "claude-opus-4-6": "Opus 4.6",
        "claude-opus": "Opus 4.6",
        "opus": "Opus 4.6",
        "claude-sonnet-4-5-20250929": "Sonnet 4.5",
        "claude-sonnet": "Sonnet 4.5",
        "sonnet": "Sonnet 4.5",
        "claude-haiku-4-5-20251001": "Haiku 4.5",
        "claude-haiku": "Haiku 4.5",
        "haiku": "Haiku 4.5",
    }
    # Check exact match first
    if model_lower in mapping:
        return mapping[model_lower]
    # Check partial match (e.g., "claude-opus-4-6-20250101")
    for key, value in mapping.items():
        if key in model_lower:
            return value
    # Already in display format?
    if model.strip() in ("Opus 4.6", "Sonnet 4.5", "Haiku 4.5"):
        return model.strip()
    return model.strip()


def detect_model_from_hook_input(hook_input: dict[str, Any]) -> str | None:
    """
    Detect the Claude model from hook input data.

    Checks in order of priority:
    1. Task() model parameter (if tool_name == 'Task')
    2. HTMLGRAPH_MODEL environment variable (set by hooks)
    3. ANTHROPIC_MODEL or CLAUDE_MODEL environment variables

    Args:
        hook_input: Hook input dict containing tool_name and tool_input

    Returns:
        Model name (e.g., 'claude-opus', 'claude-sonnet', 'claude-haiku') or None
    """
    # Get tool info
    tool_name_value: Any = hook_input.get("tool_name", "") or hook_input.get("name", "")
    tool_name = tool_name_value if isinstance(tool_name_value, str) else ""
    tool_input_value: Any = hook_input.get("tool_input", {}) or hook_input.get(
        "input", {}
    )
    tool_input = tool_input_value if isinstance(tool_input_value, dict) else {}

    # 1. Check for Task() model parameter first
    if tool_name == "Task" and "model" in tool_input:
        model_value: Any = tool_input.get("model")
        if model_value and isinstance(model_value, str):
            model = model_value.strip().lower()
            if model:
                if not model.startswith("claude-"):
                    model = f"claude-{model}"
                return normalize_model_name(model)

    # 2. Check environment variables (set by PreToolUse hook)
    for env_var in ["HTMLGRAPH_MODEL", "ANTHROPIC_MODEL", "CLAUDE_MODEL"]:
        value = os.environ.get(env_var)
        if value and isinstance(value, str):
            model = value.strip()
            if model:
                return normalize_model_name(model)

    return None


def get_model_from_parent_event(db_path: str | None = None) -> str | None:
    """
    Look up the model from the parent Task delegation event in the database.

    This is used when a child event (Read, Bash, Grep, etc.) is running in a subagent
    and needs to inherit the model from the parent Task that delegated to it.

    Args:
        db_path: Optional database path. If not provided, uses default path.

    Returns:
        Model name from parent event if found, None otherwise.
    """
    parent_event_id = os.environ.get("HTMLGRAPH_PARENT_EVENT")
    if not parent_event_id:
        return None

    try:
        from htmlgraph.config import get_database_path
        from htmlgraph.db.schema import HtmlGraphDB

        path = db_path or str(get_database_path())
        db = HtmlGraphDB(path)
        if db.connection is None:
            return None
        cursor = db.connection.cursor()
        cursor.execute(
            "SELECT model FROM agent_events WHERE event_id = ? LIMIT 1",
            (parent_event_id,),
        )
        row = cursor.fetchone()
        if row and row[0]:
            return str(row[0])
    except Exception:
        pass
    return None


def detect_agent_from_environment() -> tuple[str, str | None]:
    """
    Detect the agent/model name from environment variables and status cache.

    Checks multiple sources in order of priority:
    1. HTMLGRAPH_AGENT - Explicit agent name set by user
    2. HTMLGRAPH_SUBAGENT_TYPE - For subagent sessions
    3. HTMLGRAPH_PARENT_AGENT - Parent agent context
    4. HTMLGRAPH_MODEL - Model name (e.g., claude-haiku, claude-opus)
    5. CLAUDE_MODEL - Model name if exposed by Claude Code
    6. ANTHROPIC_MODEL - Alternative model env var
    7. Parent event model (from database) - If HTMLGRAPH_PARENT_EVENT is set
    8. Status line cache (model only) - ~/.cache/claude-code/status-{session_id}.json

    Falls back to 'claude-code' if no environment variable is set.

    Returns:
        Tuple of (agent_id, model_name). Model name may be None if not detected.
    """
    # Check for explicit agent name first
    agent_id = None
    env_vars_agent = [
        "HTMLGRAPH_AGENT",
        "HTMLGRAPH_SUBAGENT_TYPE",
        "HTMLGRAPH_PARENT_AGENT",
    ]

    for var in env_vars_agent:
        value = os.environ.get(var)
        if value and value.strip():
            agent_id = value.strip()
            break

    # Check for model name separately
    model_name = None
    env_vars_model = [
        "HTMLGRAPH_MODEL",
        "CLAUDE_MODEL",
        "ANTHROPIC_MODEL",
    ]

    for var in env_vars_model:
        value = os.environ.get(var)
        if value and value.strip():
            model_name = value.strip()
            break

    # NEW: Check parent event model from database (before status cache fallback)
    if not model_name:
        model_name = get_model_from_parent_event()

    # Fallback: Try to read model from status line cache
    if not model_name:
        model_name = get_model_from_status_cache()

    # Default fallback for agent_id
    if not agent_id:
        agent_id = "claude-code"

    # Normalize agent_id to lowercase with hyphens
    agent_id = agent_id.lower().replace(" ", "-")

    # Normalize model_name to display format
    model_name = normalize_model_name(model_name)

    return agent_id, model_name


__all__ = [
    "get_model_from_status_cache",
    "normalize_model_name",
    "detect_model_from_hook_input",
    "get_model_from_parent_event",
    "detect_agent_from_environment",
]
