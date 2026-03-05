#!/usr/bin/env bash
set -euo pipefail

# Usage: ./scripts/worktree-merge.sh <worktree-name> [base-branch]
# Merges a task branch back to base and cleans up

if [ $# -lt 1 ]; then
    echo "Usage: $0 <worktree-name> [base-branch]"
    echo ""
    echo "Examples:"
    echo "  $0 task-name"
    echo "  $0 task-name main"
    exit 1
fi

TASK="$1"
BASE="${2:-main}"
BASE_DIR="worktrees"
BRANCH="${BASE_DIR##*/}/${TASK}"
WORKTREE="${BASE_DIR}/${TASK}"

# Adjust branch name to match feature/ prefix convention
BRANCH="feature/${TASK}"

echo "Merging ${BRANCH} into ${BASE}..."

# Verify worktree exists
if [ ! -d "$WORKTREE" ]; then
    echo "❌ Worktree not found: $WORKTREE"
    exit 1
fi

# Run tests in worktree first
echo "Running tests in worktree..."
cd "$WORKTREE"
if ! uv run pytest; then
    echo "❌ Tests failed! Fix before merging."
    cd - >/dev/null
    exit 1
fi

# Switch to main repo and merge
cd "$(git rev-parse --show-toplevel)"
git checkout "$BASE"
git pull origin "$BASE" 2>/dev/null || true

# Generate commit message with recent commits from the branch
COMMIT_MSG="Merge branch '${BRANCH}' into ${BASE}

Implements: ${TASK}"

if [ -d "$WORKTREE" ]; then
    RECENT_COMMITS=$(cd "$WORKTREE" && git log --oneline "origin/${BASE}..HEAD" 2>/dev/null | head -5 | sed 's/^/  /')
    if [ -n "$RECENT_COMMITS" ]; then
        COMMIT_MSG="${COMMIT_MSG}

Recent commits:
${RECENT_COMMITS}"
    fi
fi

# Merge with no-ff for clear history
git merge --no-ff "$BRANCH" -m "$COMMIT_MSG"

echo "✅ Merged ${BRANCH} into ${BASE}"

# Cleanup
echo "Cleaning up..."
git worktree remove "$WORKTREE" 2>/dev/null || true
git branch -d "$BRANCH" 2>/dev/null || true

echo "✅ Cleanup complete"
