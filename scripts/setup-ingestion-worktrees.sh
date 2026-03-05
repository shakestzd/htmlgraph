#!/usr/bin/env bash
set -euo pipefail

echo "Setting up worktrees for Session Ingestion parallel development..."

MAIN_BRANCH=$(git branch --show-current)
TASKS=(
    "ingester-base-claude"
    "hook-hierarchy"
    "ingester-gemini"
    "ingester-copilot-codex"
    "mcp-server"
    "missing-hooks"
    "ingester-opencode-cursor"
    "fts5-search"
    "http-hooks-otel"
)

mkdir -p worktrees

for task in "${TASKS[@]}"; do
    branch="feature/$task"
    worktree="worktrees/$task"

    if [ -d "$worktree" ]; then
        echo "  Worktree exists: $task"
    elif git show-ref --verify --quiet "refs/heads/$branch"; then
        git worktree add "$worktree" "$branch" 2>/dev/null
        echo "  Created: $task (existing branch)"
    else
        git worktree add "$worktree" -b "$branch" 2>/dev/null
        echo "  Created: $task (new branch from $MAIN_BRANCH)"
    fi
done

echo ""
echo "Setup complete! Active worktrees:"
git worktree list | grep "worktrees/"
echo ""
echo "Wave 0 (start now):  worktrees/ingester-base-claude, worktrees/hook-hierarchy"
echo "Wave 1 (after Wave 0): worktrees/ingester-gemini, worktrees/ingester-copilot-codex, worktrees/mcp-server, worktrees/missing-hooks, worktrees/ingester-opencode-cursor"
echo "Wave 2 (after Wave 1): worktrees/fts5-search, worktrees/http-hooks-otel"
