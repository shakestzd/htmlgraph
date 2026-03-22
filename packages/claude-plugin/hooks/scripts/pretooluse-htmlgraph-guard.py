#!/usr/bin/env -S uv run --with htmlgraph
"""
PreToolUse guard: block direct .htmlgraph/ file modifications.

All .htmlgraph/ writes must go through the HtmlGraph SDK to ensure
consistent HTML formatting, validation, and database sync.

Exit code 2 = block the tool call (stderr becomes the error message Claude sees).
Exit code 0 = allow the tool call.
"""

import json
import re
import sys

try:
    from htmlgraph.hooks.version_check import check_hook_version

    check_hook_version("0.34.14")
except Exception:
    pass

# --- Constants ---

BLOCK_MESSAGE = """BLOCKED: Direct .htmlgraph/ file modification detected. Use the HtmlGraph SDK instead.

Examples:
  sdk.spikes.delete('spike-id')           # Instead of: rm .htmlgraph/spikes/spike-xxx.html
  sdk.features.create('title').save()     # Instead of: echo > .htmlgraph/features/feat-xxx.html
  sdk.bugs.edit('bug-id', status='done')  # Instead of editing HTML directly

The SDK ensures consistent formatting, validation, and database sync."""

# Bash commands that mutate the filesystem
MUTATING_BASH_COMMANDS = re.compile(
    r"\b(rm|mv|cp|touch|mkdir|rmdir|chmod|chown|truncate|tee)\b"
    r"|sed\s+(--in-place|-i)"
    r"|\btruncate\b"
)

# Shell redirections that write to a file path within .htmlgraph/
# Matches: > .htmlgraph/..., >> .htmlgraph/..., >| .htmlgraph/...
REDIRECT_TO_HTMLGRAPH = re.compile(r">{1,2}\|?\s*['\"]?[^;|&\s]*\.htmlgraph/")

# SQLite mutating statements
SQLITE_MUTATING = re.compile(
    r"\b(INSERT|UPDATE|DELETE|DROP|CREATE|ALTER)\b", re.IGNORECASE
)

# Signals that the command is an SDK call or safe Python usage, not raw file ops
SDK_USAGE_PATTERNS = re.compile(
    r"from\s+htmlgraph\s+import"
    r"|import\s+htmlgraph"
    r"|htmlgraph\s+\w"  # CLI: uv run htmlgraph <subcommand>
    r"|\bpytest\b"  # pytest may reference .htmlgraph/ in fixtures — always allow
    r"|\bgit\b"  # git operations (add, commit, diff, log, status) — allow
)


def contains_htmlgraph_path(text: str) -> bool:
    """Return True if the text references a .htmlgraph/ path."""
    return ".htmlgraph/" in text


def is_sdk_or_safe_command(command: str) -> bool:
    """Return True if the command uses the SDK, CLI, pytest, or git — allow these."""
    return bool(SDK_USAGE_PATTERNS.search(command))


def bash_is_mutating(command: str) -> bool:
    """Return True if the Bash command would mutate the .htmlgraph/ directory."""
    if not contains_htmlgraph_path(command):
        return False

    # Allow SDK calls, CLI invocations, pytest, and git operations
    if is_sdk_or_safe_command(command):
        return False

    # Check for explicit mutating shell commands
    if MUTATING_BASH_COMMANDS.search(command):
        return True

    # Check for shell redirections targeting .htmlgraph/
    if REDIRECT_TO_HTMLGRAPH.search(command):
        return True

    # Check for sqlite3 with mutating SQL targeting .htmlgraph/
    if "sqlite3" in command and SQLITE_MUTATING.search(command):
        return True

    return False


def check(hook_input: dict) -> bool:
    """
    Return True if the tool call should be BLOCKED, False to allow.

    Checks:
      - Write / Edit / MultiEdit: block if file_path is inside .htmlgraph/
      - Bash: block if command mutates .htmlgraph/
    """
    tool_name = hook_input.get("tool_name", "")
    tool_input = hook_input.get("tool_input", {})

    if tool_name in ("Write", "Edit", "MultiEdit"):
        file_path = tool_input.get("file_path", "")
        return contains_htmlgraph_path(file_path)

    if tool_name == "Bash":
        command = tool_input.get("command", "")
        return bash_is_mutating(command)

    # All other tools — allow
    return False


def main() -> None:
    raw = sys.stdin.read()
    try:
        hook_input = json.loads(raw)
    except json.JSONDecodeError:
        # Malformed input — allow (fail open; don't break normal operation)
        sys.exit(0)

    if check(hook_input):
        print(BLOCK_MESSAGE, file=sys.stderr)
        sys.exit(2)

    sys.exit(0)


if __name__ == "__main__":
    main()
