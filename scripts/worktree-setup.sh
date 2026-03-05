#!/usr/bin/env bash
set -euo pipefail

# Usage: ./scripts/worktree-setup.sh [--base-dir DIR] [--branch-prefix PREFIX]
# Creates worktrees for all in-progress or todo features on the active track

BASE_DIR="worktrees"
BRANCH_PREFIX="feature"

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --base-dir)
            BASE_DIR="$2"
            shift 2
            ;;
        --branch-prefix)
            BRANCH_PREFIX="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--base-dir DIR] [--branch-prefix PREFIX]"
            exit 1
            ;;
    esac
done

# Get task list from HtmlGraph
TASKS=$(uv run python -c "
from htmlgraph import SDK

sdk = SDK()

# Get features on the most recent track that are todo/in-progress
features = sdk.features.where(status=['todo', 'in_progress'])

if not features:
    print('')
else:
    for f in features:
        # Output: feature_id|short_name
        short = f.id.replace('feat-', '')
        print(f'{f.id}|{short}')
" 2>/dev/null || echo "")

if [ -z "$TASKS" ]; then
    echo "No pending tasks found. Create a plan first with /htmlgraph:plan"
    exit 1
fi

mkdir -p "$BASE_DIR"

echo "Creating worktrees from HtmlGraph features..."
echo "$TASKS" | while IFS='|' read -r feat_id short_name; do
    branch="${BRANCH_PREFIX}/${short_name}"
    worktree="${BASE_DIR}/${short_name}"

    if [ -d "$worktree" ]; then
        echo "  Worktree exists: $short_name"
    elif git show-ref --verify --quiet "refs/heads/$branch"; then
        git worktree add "$worktree" "$branch" 2>/dev/null
        echo "  Created: $short_name (existing branch)"
    else
        git worktree add "$worktree" -b "$branch" 2>/dev/null
        echo "  Created: $short_name (new branch)"
    fi
done

echo ""
echo "Setup complete! Active worktrees:"
git worktree list | grep "$BASE_DIR" || echo "  No worktrees active"

echo ""
echo "Pre-warming virtual environments (this takes ~60s per worktree)..."
ls -d "$BASE_DIR"/*/ 2>/dev/null | while read -r wt; do
    (
        cd "$wt"
        uv sync --quiet 2>/dev/null && echo "  ✅ venv ready: $(basename "$wt")" || echo "  ⚠️  venv setup failed: $(basename "$wt")"
    ) &
done
wait

echo ""
echo "All worktrees ready for parallel development!"
