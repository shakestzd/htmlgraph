#!/usr/bin/env -S uv run
"""
Pre-commit hook to detect incomplete systematic changes.

Scans staged changes for refactoring patterns (renamed variables, moved imports,
changed function signatures) and warns about potentially incomplete changes by
checking if old patterns still exist elsewhere in the codebase.

Exit code: Always 0 (warnings don't block commits)
"""

import re
import shutil
import subprocess
import sys
from pathlib import Path

# Directories to exclude from checks
EXCLUDED_DIRS = {".htmlgraph", "tests", ".git", "__pycache__", ".venv"}
EXCLUDED_PATTERNS = {".pyc", ".pyo", ".egg-info", ".mypy_cache", ".ruff_cache"}

# Short/common names that are not meaningful refactoring signals
SKIP_NAMES = {
    "run", "get", "set", "do", "on", "id", "to", "is", "ok", "fn", "f", "x",
    "n", "i", "j", "k", "v", "s", "go", "up", "it", "at", "no", "or", "so",
    "key", "val", "arg", "ctx", "err", "msg", "req", "res", "cmd", "log", "db",
    "app", "api", "url", "uri", "obj", "cls", "self", "init", "main", "test",
    "data", "name", "path", "file", "line", "node", "item", "list", "dict",
    "type", "info", "stop", "open", "read", "save", "load", "send", "call",
    "make", "find", "sort", "copy", "move", "show", "hide", "help",
}
MIN_NAME_LEN = 5


def should_skip_symbol(name: str) -> bool:
    """Return True if this name is too short or too generic to be a signal."""
    return len(name) < MIN_NAME_LEN or name.lower() in SKIP_NAMES


def get_staged_diff() -> str:
    """Get the staged changes from git diff."""
    result = subprocess.run(
        ["git", "diff", "--cached"],
        capture_output=True,
        text=True,
        check=False,
        timeout=5,
    )
    return result.stdout


def get_changed_files() -> set[str]:
    """Get list of staged Python files being changed."""
    result = subprocess.run(
        ["git", "diff", "--cached", "--name-only", "--diff-filter=ACM"],
        capture_output=True,
        text=True,
        check=False,
        timeout=5,
    )
    files = {f for f in result.stdout.strip().split("\n") if f.endswith(".py") and f}
    return files


def should_skip_file(filepath: str) -> bool:
    """Check if file should be skipped."""
    path_parts = Path(filepath).parts
    for excluded in EXCLUDED_DIRS:
        if excluded in path_parts:
            return True
    for pattern in EXCLUDED_PATTERNS:
        if pattern in filepath:
            return True
    return False


def get_search_command(pattern: str, search_dirs: list[str]) -> list[str]:
    """Return rg or grep command to find whole-word occurrences of pattern."""
    if shutil.which("rg"):
        return ["rg", "-w", "--no-unicode", "-n", "--no-heading", pattern] + search_dirs
    else:
        return ["grep", "-r", "-n", f"\\b{pattern}\\b"] + search_dirs


def extract_patterns_from_diff(
    diff: str,
) -> tuple[set[tuple[str, str]], set[str]]:
    """
    Extract refactoring patterns from diff.

    Returns:
        - set of (old_pattern, new_pattern) rename tuples
        - set of deleted symbol names with no corresponding add
    Detects:
    - Function renames: def old_func( → def new_func(
    - Variable renames: old_var = → new_var =
    - Class renames: class OldClass( → class NewClass(
    - Import changes: from foo import old_name → from foo import new_name
    - Pure deletions: symbols removed with no replacement
    """
    patterns: set[tuple[str, str]] = set()
    deleted_only: set[str] = set()

    # Split diff into individual file changes
    file_diffs = re.split(r"^diff --git", diff, flags=re.MULTILINE)

    for file_diff in file_diffs[1:]:  # Skip first empty split
        lines = file_diff.split("\n")

        # Track removed and added lines separately
        removed_lines: list[str] = []
        added_lines: list[str] = []

        for line in lines:
            if line.startswith("-") and not line.startswith("---"):
                removed_lines.append(line[1:])
            elif line.startswith("+") and not line.startswith("+++"):
                added_lines.append(line[1:])

        # Collect all added function/class names for deletion detection
        added_func_names: set[str] = set()
        added_class_names: set[str] = set()
        for added in added_lines:
            m = re.search(r"def\s+(\w+)\s*\(", added)
            if m:
                added_func_names.add(m.group(1))
            m = re.search(r"class\s+(\w+)\s*[\(:]", added)
            if m:
                added_class_names.add(m.group(1))

        # Find patterns in removed lines
        for removed in removed_lines:
            # Look for function definitions
            func_match = re.search(r"def\s+(\w+)\s*\(", removed)
            if func_match:
                old_name = func_match.group(1)
                if not should_skip_symbol(old_name):
                    matched = False
                    for added in added_lines:
                        new_func = re.search(r"def\s+(\w+)\s*\(", added)
                        if new_func:
                            new_name = new_func.group(1)
                            if old_name != new_name:
                                patterns.add((old_name, new_name))
                                matched = True
                    # Pure deletion: removed but no new function added at all
                    if not matched and old_name not in added_func_names:
                        deleted_only.add(old_name)

            # Look for class definitions
            class_match = re.search(r"class\s+(\w+)\s*[\(:]", removed)
            if class_match:
                old_name = class_match.group(1)
                if not should_skip_symbol(old_name):
                    matched = False
                    for added in added_lines:
                        new_class = re.search(r"class\s+(\w+)\s*[\(:]", added)
                        if new_class:
                            new_name = new_class.group(1)
                            if old_name != new_name:
                                patterns.add((old_name, new_name))
                                matched = True
                    if not matched and old_name not in added_class_names:
                        deleted_only.add(old_name)

            # Look for variable assignments (simple heuristic)
            var_match = re.search(r"^(\w+)\s*=", removed)
            if var_match:
                old_var = var_match.group(1)
                # Only consider if it looks like a constant or module-level var
                if old_var.isupper() or old_var.startswith("_"):
                    if not should_skip_symbol(old_var):
                        for added in added_lines:
                            new_var = re.search(r"^(\w+)\s*=", added)
                            if new_var:
                                new_name = new_var.group(1)
                                if old_var != new_name:
                                    patterns.add((old_var, new_name))

    return patterns, deleted_only


def find_remaining_uses(
    old_pattern: str, exclude_files: set[str], use_rg: bool
) -> list[tuple[str, int]]:
    """
    Search codebase for remaining uses of old_pattern.

    Returns list of (filepath, line_number) tuples.
    """
    results: list[tuple[str, int]] = []
    search_dirs = ["src/", "packages/"]
    cmd = get_search_command(old_pattern, search_dirs)

    try:
        search_result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            check=False,
            timeout=5,
        )

        if search_result.stdout:
            for line in search_result.stdout.strip().split("\n"):
                if not line:
                    continue

                # Parse output: filepath:line_number:content
                parts = line.split(":", 2)
                if len(parts) >= 2:
                    filepath = parts[0]

                    # When using grep (not rg), apply manual exclusion
                    if not use_rg and should_skip_file(filepath):
                        continue

                    if filepath in exclude_files:
                        continue

                    try:
                        line_num = int(parts[1])
                        results.append((filepath, line_num))
                    except ValueError:
                        pass

    except subprocess.TimeoutExpired:
        print(f"  ⏱️  Timed out checking for '{old_pattern}', skipping")
    except FileNotFoundError:
        pass

    return results


def main() -> int:
    """Main hook logic."""
    diff = get_staged_diff()
    if not diff:
        return 0

    changed_files = get_changed_files()
    use_rg = shutil.which("rg") is not None
    patterns, deleted_only = extract_patterns_from_diff(diff)

    if not patterns and not deleted_only:
        return 0

    # Check each rename pattern
    warnings: list[str] = []

    for old_pattern, new_pattern in patterns:
        remaining = find_remaining_uses(old_pattern, changed_files, use_rg)

        if remaining:
            warning = (
                f"\n⚠️  Potential incomplete systematic change detected:\n"
                f"  You renamed '{old_pattern}' → '{new_pattern}'\n"
                f"  But '{old_pattern}' still appears in:\n"
            )
            for filepath, line_num in remaining[:5]:
                warning += f"    - {filepath}:{line_num}\n"

            if len(remaining) > 5:
                warning += f"    ... and {len(remaining) - 5} more occurrences\n"

            warning += "  Consider updating these references."
            warnings.append(warning)

    # Check pure deletions
    for deleted_name in deleted_only:
        remaining = find_remaining_uses(deleted_name, changed_files, use_rg)

        if remaining:
            warning = (
                f"\n⚠️  Deleted '{deleted_name}' but references still found in:\n"
            )
            for filepath, line_num in remaining[:5]:
                warning += f"    - {filepath}:{line_num}\n"

            if len(remaining) > 5:
                warning += f"    ... and {len(remaining) - 5} more occurrences\n"

            warning += "  Consider removing or updating these references."
            warnings.append(warning)

    if warnings:
        print("\n" + "".join(warnings))
        return 0  # Don't block commit, just warn

    return 0


if __name__ == "__main__":
    sys.exit(main())
