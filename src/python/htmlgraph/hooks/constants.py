"""
Shared Constants for HtmlGraph Hooks

This module centralizes constants used across multiple hook files to eliminate
duplication and ensure consistency.

Constants:
    SUBAGENT_SUFFIXES: List of session ID suffixes for subagent types
    SUBAGENT_TYPES: Set of known subagent type names
    NEVER_BLOCK_TOOLS: Tools that should never be blocked by enforcement
    DRIFT_QUEUE_FILE: Filename for drift detection queue
"""

from __future__ import annotations

# Subagent session ID suffixes (used for stripping parent session ID)
# Example: "abc123-general-purpose" -> "abc123"
SUBAGENT_SUFFIXES = [
    "-general-purpose",
    "-Explore",
    "-Bash",
    "-Plan",
    "-htmlgraph:researcher",
    "-htmlgraph:haiku-coder",
    "-htmlgraph:sonnet-coder",
    "-htmlgraph:opus-coder",
    "-htmlgraph:test-runner",
    "-htmlgraph:debugger",
    "-researcher",
    "-debugger",
    "-test-runner",
]

# Known subagent type names (without hyphens)
SUBAGENT_TYPES = {
    "general-purpose",
    "Explore",
    "Bash",
    "Plan",
    "htmlgraph:researcher",
    "htmlgraph:haiku-coder",
    "htmlgraph:sonnet-coder",
    "htmlgraph:opus-coder",
    "htmlgraph:test-runner",
    "htmlgraph:debugger",
    "researcher",
    "debugger",
    "test-runner",
}

# Tools that should NEVER be blocked by enforcement
# These are essential for coordination, orchestration, and exploration
NEVER_BLOCK_TOOLS = {
    "Task",
    "TaskCreate",
    "TaskUpdate",
    "TaskList",
    "TaskGet",
    "AskUserQuestion",
    "TodoWrite",
    "TodoRead",
    "Skill",
    "Read",
    "Grep",
    "Glob",
    "WebSearch",
    "WebFetch",
}

# Drift classification queue filename (stored in session directory)
DRIFT_QUEUE_FILE = "drift-queue.json"


__all__ = [
    "SUBAGENT_SUFFIXES",
    "SUBAGENT_TYPES",
    "NEVER_BLOCK_TOOLS",
    "DRIFT_QUEUE_FILE",
]
