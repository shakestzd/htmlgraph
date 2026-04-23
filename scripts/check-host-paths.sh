#!/usr/bin/env bash
# check-host-paths.sh — scan committed artifacts for host-local absolute paths.
#
# Scans files under .htmlgraph/ and .claude/ for patterns that indicate
# host-specific absolute paths that should never be committed.
#
# PATTERNS DETECTED:
#   /Users/<anything>/          — macOS home directories
#   /home/<user>/               — Linux home directories (except /home/runner/ for CI)
#   /workspaces/<username>/     — GitHub Codespaces workspace paths
#   /private/var/folders/       — macOS temp directories
#
# ALLOWLIST:
#   Files listed in scripts/host-paths-allowlist.txt (relative to repo root)
#   are skipped entirely. The allowlist covers files that legitimately
#   document or describe host-local paths (e.g. bug reports about this issue).
#
# EXIT CODES:
#   0  — no violations found (prints "OK — N files scanned")
#   1  — one or more violations found (prints "file:line: <matched-path>")
#
# USAGE:
#   scripts/check-host-paths.sh                    # scan entire scope
#   scripts/check-host-paths.sh --staged           # scan only git-staged files
#   scripts/check-host-paths.sh path/to/file       # scan specific file(s)
#
set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
ALLOWLIST_FILE="$REPO_ROOT/scripts/host-paths-allowlist.txt"
STAGED_ONLY=0
EXPLICIT_FILES=()

# Parse arguments
for arg in "$@"; do
    if [[ "$arg" == "--staged" ]]; then
        STAGED_ONLY=1
    else
        EXPLICIT_FILES+=("$arg")
    fi
done

# Build allowlist set (relative paths from repo root)
declare -A ALLOWLIST
if [[ -f "$ALLOWLIST_FILE" ]]; then
    while IFS= read -r line; do
        # Skip comments and blank lines
        [[ "$line" =~ ^[[:space:]]*# ]] && continue
        [[ -z "${line// }" ]] && continue
        ALLOWLIST["$line"]=1
    done < "$ALLOWLIST_FILE"
fi

# Host-local path patterns (grep extended regex)
# Note: /home/runner/ is excluded (GitHub Actions CI user)
PATTERN='/Users/[^/[:space:]]+/|/home/(?!runner/)[^/[:space:]]+/|/workspaces/[^/[:space:]]+/|/private/var/folders/'

# Collect files to scan
declare -a FILES_TO_SCAN

if [[ ${#EXPLICIT_FILES[@]} -gt 0 ]]; then
    FILES_TO_SCAN=("${EXPLICIT_FILES[@]}")
elif [[ "$STAGED_ONLY" -eq 1 ]]; then
    while IFS= read -r f; do
        # Only include files matching our scope
        if [[ "$f" == .htmlgraph/* || "$f" == .claude/* ]]; then
            # Skip files that are intentionally machine-specific (matches full-scan exclusions)
            [[ "$(basename "$f")" == "htmlgraph.db" ]] && continue
            [[ "$(basename "$f")" == "settings.local.json" ]] && continue
            FILES_TO_SCAN+=("$REPO_ROOT/$f")
        fi
    done < <(git -C "$REPO_ROOT" diff --cached --name-only --diff-filter=ACMR 2>/dev/null || true)
else
    # Full scan: .htmlgraph/** and .claude/**
    while IFS= read -r f; do
        FILES_TO_SCAN+=("$f")
    done < <(find "$REPO_ROOT/.htmlgraph" "$REPO_ROOT/.claude" \
        -type f \
        ! -name "htmlgraph.db" \
        ! -name "settings.local.json" \
        2>/dev/null || true)
fi

# Filter out allowlisted files and scan
VIOLATIONS=0
SCANNED=0

for abs_file in "${FILES_TO_SCAN[@]}"; do
    # Skip if file doesn't exist (e.g. deleted staged file)
    [[ -f "$abs_file" ]] || continue

    # Compute relative path for allowlist lookup
    rel_file="${abs_file#"$REPO_ROOT"/}"

    # Skip allowlisted files
    if [[ -n "${ALLOWLIST[$rel_file]+x}" ]]; then
        continue
    fi

    SCANNED=$((SCANNED + 1))

    # Scan for host-local patterns using perl (supports lookahead for /home/runner/ exclusion)
    while IFS= read -r hit; do
        echo "$hit"
        VIOLATIONS=$((VIOLATIONS + 1))
    done < <(perl -ne '
        while (m{(/Users/[^/\s]+/|/home/(?!runner/)[^/\s]+/|/workspaces/[^/\s]+/|/private/var/folders/)}g) {
            print ARGV . ":" . $. . ": " . $1 . "\n";
        }
    ' "$abs_file" 2>/dev/null || true)
done

if [[ "$VIOLATIONS" -gt 0 ]]; then
    echo ""
    echo "FAIL: $VIOLATIONS host-local path violation(s) found in $SCANNED file(s) scanned."
    echo "      These paths must not be committed — they are machine-specific."
    echo "      To allowlist a file, add its repo-relative path to scripts/host-paths-allowlist.txt"
    exit 1
fi

echo "OK — $SCANNED file(s) scanned, no host-local path violations found."
