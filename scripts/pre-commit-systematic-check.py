#!/usr/bin/env -S uv run
"""
Pre-commit hook: Detect incomplete systematic changes.

Detects incomplete systematic changes using two complementary strategies:

1. **Diff-based detection** (primary): Reads ``git diff --cached`` and uses
   difflib.SequenceMatcher to find word-level renames between removed and added
   lines.  This catches systematic renames even when the commit message is
   generic (e.g. "refactor").

2. **Commit-message detection** (secondary): Parses the pending commit message
   for explicit rename patterns such as "rename foo to bar", "s/old/new/g", or
   "foo -> bar".

Both strategies search the un-staged source tree for remaining occurrences of
the old identifier and emit a warning if any are found.  The hook always exits
0 — it warns but never blocks commits.

Usage:
    # Run as pre-commit hook (reads COMMIT_EDITMSG automatically):
    uv run scripts/pre-commit-systematic-check.py

    # Install as .git/hooks/pre-commit:
    uv run scripts/pre-commit-systematic-check.py --install

    # Test against a specific commit message:
    uv run scripts/pre-commit-systematic-check.py --message "rename foo to bar"

Exit code: Always 0 (warns but does not block commits).
"""

import argparse
import difflib
import re
import shutil
import subprocess
import sys
from pathlib import Path

# ---------------------------------------------------------------------------
# Keyword detection
# ---------------------------------------------------------------------------

# Commit message keywords that suggest a systematic change was intended.
SYSTEMATIC_KEYWORDS = [
    "replace",
    "rename",
    "migrate",
    "refactor",
    "update all",
    "change all",
    "move all",
    "rewrite",
    "s/",  # sed-style pattern
]

# Regex for sed-style substitution patterns: s/old/new/ or s/old/new/g
_SED_PATTERN = re.compile(r"\bs/([^/]+)/([^/]+)/[gi]*\b")

# Regex for "rename X to Y" / "replace X with Y" / "X -> Y" / "X => Y"
_RENAME_PATTERN = re.compile(
    r"""
    (?:
        (?:rename|replace|move)\s+                  # action verb
        [`'"]?(\w[\w.]+)`?'?"?                       # old name (group 1)
        \s+(?:to|with|->|=>)\s+                     # separator
        [`'"]?(\w[\w.]+)[`'"]?                       # new name (group 2)
    )
    |
    (?:
        [`'"]?(\w[\w.]+)[`'"]?                       # old name (group 3)
        \s*(?:->|=>)\s*                              # arrow
        [`'"]?(\w[\w.]+)[`'"]?                       # new name (group 4)
    )
    """,
    re.VERBOSE | re.IGNORECASE,
)

# Minimum name length to avoid flagging noise like "a" → "b"
MIN_NAME_LEN = 4


def detect_systematic_keywords(message: str) -> bool:
    """Return True if the commit message contains systematic change keywords."""
    lower = message.lower()
    return any(kw in lower for kw in SYSTEMATIC_KEYWORDS)


def extract_rename_pairs_from_message(message: str) -> list[tuple[str, str]]:
    """
    Extract (old, new) rename pairs from a commit message.

    Handles patterns like:
      - "rename foo_bar to baz_qux"
      - "replace OldClass with NewClass"
      - "s/old_name/new_name/"
      - "foo_bar -> baz_qux"
    """
    pairs: list[tuple[str, str]] = []

    # sed-style: s/old/new/
    for m in _SED_PATTERN.finditer(message):
        old, new = m.group(1).strip(), m.group(2).strip()
        if len(old) >= MIN_NAME_LEN and len(new) >= MIN_NAME_LEN and old != new:
            pairs.append((old, new))

    # Natural language / arrow patterns
    for m in _RENAME_PATTERN.finditer(message):
        if m.group(1) and m.group(2):
            old, new = m.group(1).strip(), m.group(2).strip()
        elif m.group(3) and m.group(4):
            old, new = m.group(3).strip(), m.group(4).strip()
        else:
            continue
        if len(old) >= MIN_NAME_LEN and len(new) >= MIN_NAME_LEN and old != new:
            pairs.append((old, new))

    # Deduplicate while preserving order
    seen: set[tuple[str, str]] = set()
    result = []
    for pair in pairs:
        if pair not in seen:
            seen.add(pair)
            result.append(pair)
    return result


# ---------------------------------------------------------------------------
# Diff-based detection
# ---------------------------------------------------------------------------

# Minimum identifier length for diff-based detection
_MIN_IDENT_LEN = 4

# Regex matching Python/identifier-like tokens worth tracking
_IDENT_RE = re.compile(r"[A-Za-z_][A-Za-z0-9_]{3,}")


def get_staged_diff() -> str:
    """Return the full unified diff of staged changes (git diff --cached)."""
    try:
        result = subprocess.run(
            ["git", "diff", "--cached", "-U0"],
            capture_output=True,
            text=True,
            check=False,
            timeout=10,
        )
        return result.stdout
    except (subprocess.TimeoutExpired, FileNotFoundError):
        return ""


def _tokenize(line: str) -> list[str]:
    """Split a source line into identifier tokens."""
    return _IDENT_RE.findall(line)


def extract_replacements(diff_text: str) -> list[tuple[str, str]]:
    """
    Find old→new identifier pairs from removed/added line pairs in a diff.

    Strategy:
    - Collect removed lines (``-``) and added lines (``+``) within each hunk.
    - For each (removed, added) pair compare token sequences with
      difflib.SequenceMatcher to find tokens that were replaced.
    - Return unique (old_token, new_token) pairs where both tokens look like
      meaningful identifiers (length >= _MIN_IDENT_LEN, not equal).
    """
    removed: list[str] = []
    added: list[str] = []
    pairs: list[tuple[str, str]] = []

    def _flush_hunk() -> None:
        """Compare accumulated removed/added lines and extract rename pairs."""
        # Pair them up positionally; any surplus lines are compared against
        # the last counterpart to catch block-level renames.
        limit = max(len(removed), len(added))
        for i in range(limit):
            r_line = removed[min(i, len(removed) - 1)] if removed else ""
            a_line = added[min(i, len(added) - 1)] if added else ""
            if r_line == a_line:
                continue
            r_tokens = _tokenize(r_line)
            a_tokens = _tokenize(a_line)
            if not r_tokens or not a_tokens:
                continue
            # Use SequenceMatcher at the token level
            sm = difflib.SequenceMatcher(None, r_tokens, a_tokens, autojunk=False)
            for tag, i1, i2, j1, j2 in sm.get_opcodes():
                if tag == "replace":
                    # Map each replaced token to its counterpart
                    r_chunk = r_tokens[i1:i2]
                    a_chunk = a_tokens[j1:j2]
                    for old, new in zip(r_chunk, a_chunk):
                        if (
                            old != new
                            and len(old) >= _MIN_IDENT_LEN
                            and len(new) >= _MIN_IDENT_LEN
                        ):
                            pairs.append((old, new))

    for raw_line in diff_text.split("\n"):
        if raw_line.startswith("---") or raw_line.startswith("+++"):
            continue
        if raw_line.startswith("@@"):
            # Hunk boundary — flush previous hunk buffers
            _flush_hunk()
            removed.clear()
            added.clear()
            continue
        if raw_line.startswith("-"):
            removed.append(raw_line[1:])
        elif raw_line.startswith("+"):
            added.append(raw_line[1:])

    # Flush the final hunk
    _flush_hunk()

    # Deduplicate while preserving order
    seen: set[tuple[str, str]] = set()
    result: list[tuple[str, str]] = []
    for pair in pairs:
        if pair not in seen:
            seen.add(pair)
            result.append(pair)
    return result


# ---------------------------------------------------------------------------
# Filesystem search
# ---------------------------------------------------------------------------

# Directories to skip when searching for remaining occurrences
_SKIP_DIRS = {
    ".git",
    "__pycache__",
    ".venv",
    ".mypy_cache",
    ".ruff_cache",
    ".htmlgraph",
}
_SEARCH_ROOTS = ["src/", "packages/", "scripts/", "tests/"]


def _get_search_cmd(pattern: str, roots: list[str]) -> list[str]:
    """Build rg or grep command for whole-word search."""
    existing = [r for r in roots if Path(r).exists()]
    if not existing:
        existing = ["."]
    if shutil.which("rg"):
        return (
            ["rg", "--word-regexp", "-n", "--no-heading", "--no-filename-separator"]
            + [f"--glob=!{d}" for d in _SKIP_DIRS]
            + [pattern]
            + existing
        )
    else:
        return ["grep", "-rn", f"\\b{pattern}\\b"] + existing


def find_remaining_occurrences(
    old_name: str,
    exclude_files: set[str],
) -> list[tuple[str, int, str]]:
    """
    Search source tree for remaining uses of `old_name`.

    Returns list of (filepath, line_number, line_content) tuples (max 10).
    """
    cmd = _get_search_cmd(old_name, _SEARCH_ROOTS)
    try:
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            check=False,
            timeout=8,
        )
    except (subprocess.TimeoutExpired, FileNotFoundError):
        return []

    hits: list[tuple[str, int, str]] = []
    for line in result.stdout.splitlines():
        if not line:
            continue
        # rg format: path:lineno:content  OR  path:lineno-content (--no-filename-separator)
        # grep format: path:lineno:content
        parts = line.split(":", 2)
        if len(parts) < 3:
            continue
        filepath, lineno_str, content = parts[0], parts[1], parts[2]
        # Skip excluded files
        if filepath in exclude_files:
            continue
        # Skip files in excluded dirs
        path_obj = Path(filepath)
        if any(d in path_obj.parts for d in _SKIP_DIRS):
            continue
        try:
            lineno = int(lineno_str)
        except ValueError:
            continue
        hits.append((filepath, lineno, content.strip()))
        if len(hits) >= 10:
            break

    return hits


# ---------------------------------------------------------------------------
# Git helpers
# ---------------------------------------------------------------------------


def get_commit_message() -> str:
    """Read the pending commit message from COMMIT_EDITMSG."""
    msg_file = Path(".git/COMMIT_EDITMSG")
    if msg_file.exists():
        try:
            return msg_file.read_text(encoding="utf-8", errors="replace")
        except OSError:
            pass
    return ""


def get_staged_files() -> set[str]:
    """Return set of staged Python file paths."""
    result = subprocess.run(
        ["git", "diff", "--cached", "--name-only", "--diff-filter=ACM"],
        capture_output=True,
        text=True,
        check=False,
        timeout=5,
    )
    return {f for f in result.stdout.splitlines() if f.endswith(".py") and f}


# ---------------------------------------------------------------------------
# Install support
# ---------------------------------------------------------------------------


def install_hook() -> int:
    """
    Install this script as .git/hooks/pre-commit.

    Wraps the existing scripts/hooks/pre-commit bash script which already
    calls check-systematic-changes.py.  If that wrapper exists, we prefer
    it; otherwise we install this script directly.
    """
    git_hooks_dir = Path(".git/hooks")
    if not git_hooks_dir.exists():
        print(
            "Error: Not in a git repository (no .git/hooks directory).", file=sys.stderr
        )
        return 1

    this_file = Path(__file__).resolve()
    project_root = this_file.parent.parent  # scripts/ -> project root
    bash_wrapper = project_root / "scripts" / "hooks" / "pre-commit"
    target = git_hooks_dir / "pre-commit"

    source = bash_wrapper if bash_wrapper.exists() else this_file

    try:
        import shutil as _shutil

        _shutil.copy2(source, target)
        target.chmod(0o755)
        print(f"Installed pre-commit hook from {source.relative_to(project_root)}")
        return 0
    except OSError as exc:
        print(f"Error installing hook: {exc}", file=sys.stderr)
        return 1


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------


def main(argv: list[str] | None = None) -> int:
    """Entry point."""
    parser = argparse.ArgumentParser(
        description="Detect incomplete systematic changes before committing."
    )
    parser.add_argument(
        "--install",
        action="store_true",
        help="Install as .git/hooks/pre-commit and exit.",
    )
    parser.add_argument(
        "--message",
        metavar="MSG",
        help="Commit message to analyse (default: read from .git/COMMIT_EDITMSG).",
    )
    args = parser.parse_args(argv)

    if args.install:
        return install_hook()

    staged_files = get_staged_files()
    all_pairs: list[tuple[str, str]] = []

    # --- Strategy 1: diff-based detection (always runs) ---
    diff = get_staged_diff()
    if diff:
        diff_pairs = extract_replacements(diff)
        all_pairs.extend(diff_pairs)

    # --- Strategy 2: commit-message detection ---
    message = args.message if args.message else get_commit_message()
    if message and detect_systematic_keywords(message):
        msg_pairs = extract_rename_pairs_from_message(message)
        # Merge without duplicating pairs already found via diff
        existing = set(all_pairs)
        for pair in msg_pairs:
            if pair not in existing:
                all_pairs.append(pair)
                existing.add(pair)

    if not all_pairs:
        return 0

    warnings: list[str] = []

    for old_name, new_name in all_pairs:
        hits = find_remaining_occurrences(old_name, staged_files)
        if hits:
            lines = [
                f"\n  Systematic change detected: '{old_name}' -> '{new_name}'",
                f"  Found {len(hits)} remaining occurrence(s) in unstaged files:",
            ]
            for filepath, lineno, content in hits:
                lines.append(f"    {filepath}:{lineno}: {content}")
            if len(hits) == 10:
                lines.append("    ... (showing first 10 matches)")
            lines.append("  Consider updating these references before committing.")
            warnings.append("\n".join(lines))

    if warnings:
        print("\n" + "\n".join(f"Warning: {w}" for w in warnings) + "\n")

    # Always exit 0 — this hook warns but never blocks
    return 0


if __name__ == "__main__":
    sys.exit(main())
