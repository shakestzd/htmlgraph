# RoboRev Integration - Quick Reference

## Command Quick Reference

```bash
# Review commands
roborev review <commit>              # Review specific commit
roborev review HEAD                  # Review most recent commit
roborev review-branch                # Review all branch commits
roborev review main..HEAD            # Review commits since main

# Status commands
roborev list-jobs                    # List recent review jobs
roborev show --job <id>              # Show job results
roborev show --job <id> --json       # JSON format (for parsing)
roborev show --job <id> --summary    # Summary view

# Action commands
roborev address <id>                 # Mark job addressed
roborev comment --job <id> "msg"     # Add comment
roborev auth login                   # Authenticate
```

## CLI Integration

### Create Bug from Finding

```bash
htmlgraph bug create "[roborev] <finding title>"
```

### Batch Create from Review

```bash
# Get review results and create bugs for medium+ findings
JOB_ID=$(roborev review HEAD --json | jq -r '.job_id')
roborev show --job "$JOB_ID" --json | jq -r '.findings[] | select(.severity | test("high|critical|medium")) | .title' | \
  while IFS= read -r title; do
    htmlgraph bug create "[roborev] $title"
  done
```

## Severity Mapping

| roborev | HtmlGraph | Action |
|---------|-----------|--------|
| critical | high | Create bug |
| high | high | Create bug |
| medium | medium | Create bug |
| low | - | Skip |
| info | - | Skip |

## Common Workflows

### 1. Simple Review

```bash
roborev review HEAD && roborev list-jobs | head -1
```

### 2. Review & Bug Creation

```bash
# Get job ID
JOB=$(roborev review HEAD --json | jq -r '.job_id')

# Create bugs via CLI
roborev show --job "$JOB" --json | jq -r '.findings[] | select(.severity | test("high|critical|medium")) | .title' | \
  while IFS= read -r title; do
    htmlgraph bug create "[roborev] $title"
  done
```

### 3. Review Multiple Commits

```bash
for commit in $(git log --oneline -5 | awk '{print $1}'); do
    roborev review $commit
done
roborev list-jobs -limit 5
```

### 4. Review Branch Before Merge

```bash
# Review all commits since main
roborev review main..HEAD

# Get summary
JOBS=$(roborev list-jobs --limit 1)
JOB_ID=$(echo "$JOBS" | jq -r '.[0].id')
roborev show --job $JOB_ID --summary
```

## HtmlGraph Query Examples

### Find All RoboRev Bugs

```bash
sqlite3 .htmlgraph/htmlgraph.db "
SELECT id, title, priority, created_at
FROM bugs
WHERE title LIKE '%roborev%'
ORDER BY created_at DESC;
"
```

### Count by Priority

```bash
sqlite3 .htmlgraph/htmlgraph.db "
SELECT priority, COUNT(*) as count
FROM bugs
WHERE title LIKE '%roborev%'
GROUP BY priority;
"
```

### Find High Priority RoboRev Issues

```bash
uv run htmlgraph status --filter "title:roborev priority:high"
```

## Environment Setup

### Install roborev

```bash
pip install roborev
roborev auth login
# Enter API token from https://roborev.io/manage/tokens
```

### Verify Setup

```bash
which roborev                    # Check installation
roborev --version               # Check version
roborev list-jobs --limit 1     # Verify authentication
```

## Configuration

### Commit Message Flags

Skip review for specific commits:

```bash
git commit -m "docs: update README (skip-review)"
git commit -m "chore: bump version (no-review)"
```

Agent checks commit message before running review.

### Auto-Trigger Settings

In `.claude/rules/orchestration.md`:

```yaml
auto_triggers:
  roborev:
    - condition: "feature_completed AND file_count >= 3"
      action: "spawn roborev agent"
    - condition: "pre_commit_hook"
      action: "run roborev review HEAD"
```

## Troubleshooting

### roborev not found

```bash
pip install roborev
which roborev  # Should show path
```

### Authentication failed

```bash
roborev auth login
# If still failing, check token at https://roborev.io/manage/tokens
```

### Job stuck in pending

```bash
# Check status
roborev show --job <id> --verbose

# Retry
roborev review <commit> --retry
```

### HtmlGraph CLI not available

```bash
# Verify CLI is installed
htmlgraph version
```

### No findings returned

```bash
# Check if review completed
roborev show --job <id> --json | jq '.status'

# If completed, check actual findings
roborev show --job <id> --json | jq '.findings | length'
```

## Performance Notes

- Reviews typically complete in 5-30 seconds
- Larger codebases may take 1-2 minutes
- Use `--wait` flag to block until completion
- Or check status with `roborev show --job <id>`

## Integration Examples

### Pre-Commit Hook

```bash
#!/bin/bash
# .git/hooks/pre-commit

roborev review HEAD --fail-on critical
# Exit code > 0 blocks commit if critical findings
```

### Post-Commit Hook

```bash
#!/bin/bash
# .git/hooks/post-commit

JOB=$(roborev review HEAD --json | jq -r '.job_id')
echo "Started roborev review: $JOB"
# Creates HtmlGraph bugs asynchronously
```

### GitHub Actions

```yaml
name: Code Review
on: [push]
jobs:
  review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - run: pip install roborev
      - run: |
          roborev auth login --token ${{ secrets.ROBOREV_TOKEN }}
          roborev review HEAD
```

## Status Codes

```
pending     - Review in progress
completed   - Review finished
failed      - Review error (check --verbose)
timeout     - Took too long (rare)
```

## Billing/Limits

- Free tier: 10 reviews/month
- Pro tier: 1000 reviews/month
- Enterprise: Unlimited

Check usage at https://roborev.io/account/usage
