#!/usr/bin/env bash
set -euo pipefail

# Find plan.yaml
PLAN_FILE=".parallel/plans/active/plan.yaml"
if [ ! -f "$PLAN_FILE" ]; then
  echo "Error: plan.yaml not found at $PLAN_FILE"
  exit 1
fi

# Extract task IDs
if command -v yq &> /dev/null; then
  TASK_IDS=$(yq '.tasks[].id' "$PLAN_FILE")
else
  TASK_IDS=$(grep -A 100 "^tasks:" "$PLAN_FILE" | grep "  - id:" | sed 's/.*id: *"\([^"]*\)".*/\1/')
fi

echo "Creating worktrees for $(echo "$TASK_IDS" | wc -l | tr -d ' ') tasks..."

# Create worktrees in parallel
echo "$TASK_IDS" | while read task_id; do
  branch="feature/$task_id"
  worktree="worktrees/$task_id"

  if [ -d "$worktree" ]; then
    echo "⚠️  Worktree exists: $task_id"
  elif git show-ref --verify --quiet refs/heads/$branch; then
    git worktree add "$worktree" "$branch" 2>/dev/null && echo "✅ Created: $task_id (existing branch)"
  else
    git worktree add "$worktree" -b "$branch" 2>&1 | grep -v "Preparing" && echo "✅ Created: $task_id"
  fi
done

echo ""
echo "✅ Setup complete! Active worktrees:"
git worktree list | grep "worktrees/"
