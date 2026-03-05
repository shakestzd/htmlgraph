#!/usr/bin/env bash
set -euo pipefail

# Usage: ./scripts/worktree-cleanup.sh [--base-dir DIR] [--force]
# Removes all worktrees and optionally deletes branches

BASE_DIR="worktrees"
FORCE=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --base-dir)
            BASE_DIR="$2"
            shift 2
            ;;
        --force)
            FORCE=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--base-dir DIR] [--force]"
            exit 1
            ;;
    esac
done

echo "Cleaning up worktrees in ${BASE_DIR}/..."

if [ ! -d "$BASE_DIR" ]; then
    echo "No worktrees directory found."
    exit 0
fi

# Track if we removed anything
REMOVED=0

for wt in "$BASE_DIR"/*/; do
    [ -d "$wt" ] || continue
    name=$(basename "$wt")
    branch="feature/${name}"

    # Check if branch is merged
    if git branch --merged main 2>/dev/null | grep -q "$branch"; then
        echo "  Removing (merged): $name"
        git worktree remove "$wt" 2>/dev/null || true
        git branch -d "$branch" 2>/dev/null || true
        REMOVED=$((REMOVED + 1))
    elif [ "$FORCE" = true ]; then
        echo "  Force removing (unmerged): $name"
        git worktree remove --force "$wt" 2>/dev/null || true
        git branch -D "$branch" 2>/dev/null || true
        REMOVED=$((REMOVED + 1))
    else
        echo "  Skipping (unmerged): $name — use --force to remove"
    fi
done

# Prune stale worktree references
git worktree prune

echo ""
if [ "$REMOVED" -gt 0 ]; then
    echo "Remaining worktrees:"
    if git worktree list | grep -q "$BASE_DIR"; then
        git worktree list | grep "$BASE_DIR" || true
    else
        echo "  None"
    fi
else
    echo "No worktrees were removed."
fi

# Remove empty base dir
if [ -d "$BASE_DIR" ] && [ -z "$(ls -A "$BASE_DIR" 2>/dev/null)" ]; then
    rmdir "$BASE_DIR" 2>/dev/null && echo "✅ Removed empty ${BASE_DIR}/" || true
fi
