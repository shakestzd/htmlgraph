#!/usr/bin/env bash
set -euo pipefail

BASE_BRANCH="${1:-dev}"
TASKS_DIR=".parallel/plans/active/tasks"

echo "Creating PRs for completed tasks..."

# Find completed tasks
for task_file in "$TASKS_DIR"/task-*.md; do
  [ -f "$task_file" ] || continue

  status=$(grep "^status:" "$task_file" | head -1 | awk '{print $2}')
  [ "$status" = "completed" ] || continue

  task_id=$(basename "$task_file" .md)
  branch="feature/$task_id"
  title=$(grep "^# " "$task_file" | head -1 | sed 's/^# //')
  labels=$(awk '/^labels:/,/^[a-z]/ {if ($0 ~ /^\s*-/) print $2}' "$task_file" | tr '\n' ',' | sed 's/,$//')

  # Check if PR exists
  if gh pr list --head "$branch" --json number -q '.[0].number' &>/dev/null; then
    echo "⚠️  PR exists for $task_id"
    continue
  fi

  # Create PR
  if [ -n "$labels" ]; then
    gh pr create --base "$BASE_BRANCH" --head "$branch" --title "$title" --body-file "$task_file" --label "$labels"
  else
    gh pr create --base "$BASE_BRANCH" --head "$branch" --title "$title" --body-file "$task_file"
  fi

  echo "✅ Created PR for $task_id: $title"
done

echo ""
echo "✅ PR creation complete!"
