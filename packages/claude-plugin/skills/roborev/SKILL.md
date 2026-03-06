# HtmlGraph RoboRev Integration

Automatically run code reviews with roborev after completing significant features and track findings as HtmlGraph bugs.

## What This Skill Does

The RoboRev skill integrates automated code reviews into your HtmlGraph workflow:

1. **Triggers after feature completion** - When a feature with 3+ file changes is completed
2. **Runs roborev review** - Analyzes recent commits for code quality issues
3. **Creates HtmlGraph bugs** - Tracks all medium+ severity findings as actionable bugs
4. **Links to features** - Associates findings with the originating feature
5. **Reports summary** - Provides clear overview of findings by severity

## Quick Start

### Manual Trigger

Run a roborev review on your current branch:

```bash
# Review most recent commit
roborev review HEAD

# Review all branch commits
roborev review-branch

# Review specific commit
roborev review abc1234
```

### Get Results

```bash
# Wait for review to complete
JOB_ID=$(roborev review HEAD --json | jq -r '.job_id')

# Get findings (structured format)
roborev show --job $JOB_ID --json
```

### Create HtmlGraph Bugs

```python
from htmlgraph import SDK
import json, subprocess

sdk = SDK(agent='roborev')

# Get review results
result = subprocess.run(
    ['roborev', 'show', '--job', JOB_ID, '--json'],
    capture_output=True, text=True
)
data = json.loads(result.stdout)

# Create bugs for medium+ findings
for finding in data.get('findings', []):
    if finding['severity'] in ('high', 'critical', 'medium'):
        priority = 'high' if finding['severity'] in ('high', 'critical') else 'medium'
        sdk.bugs.create(f"[roborev] {finding['title']}") \
            .set_priority(priority) \
            .save()
```

## Integration with HtmlGraph Orchestrator

The roborev agent is automatically spawned after feature completion:

```python
# In orchestrator after feature.complete()
Task(
    prompt="Run roborev review on HEAD commit and create HtmlGraph bugs for any medium+ findings.",
    subagent_type="htmlgraph:roborev"
)
```

## Complete Workflow Script

```bash
#!/bin/bash
# Review feature, create bugs, report findings

set -e

FEATURE_ID=${1:-HEAD}
JOB_ID=""

echo "Starting roborev review for $FEATURE_ID..."

# 1. Run review
RESULT=$(roborev review "$FEATURE_ID" --wait --json)
JOB_ID=$(echo "$RESULT" | jq -r '.job_id')
echo "Review job: $JOB_ID"

# 2. Wait for completion
while true; do
    STATUS=$(roborev show --job "$JOB_ID" --json 2>/dev/null | jq -r '.status // "pending"')
    if [[ "$STATUS" == "completed" || "$STATUS" == "failed" ]]; then
        break
    fi
    sleep 2
done

# 3. Parse findings
FINDINGS=$(roborev show --job "$JOB_ID" --json)
TOTAL=$(echo "$FINDINGS" | jq '.findings | length')
echo "Found $TOTAL issues"

# 4. Create HtmlGraph bugs
uv run python3 - <<'PYTHON'
import json, subprocess, sys
from htmlgraph import SDK

findings_json = sys.stdin.read()
data = json.loads(findings_json)
sdk = SDK(agent='roborev')

created = 0
for finding in data.get('findings', []):
    if finding['severity'] in ('high', 'critical', 'medium'):
        priority = 'high' if finding['severity'] in ('high', 'critical') else 'medium'
        sdk.bugs.create(f"[roborev] {finding['title']}") \
            .set_priority(priority) \
            .save()
        created += 1

print(f"✓ Created {created} bugs")
sys.exit(0)
PYTHON <<< "$FINDINGS"

# 5. Summary
echo ""
echo "Review Complete:"
roborev show --job "$JOB_ID" --summary
```

## Severity Levels and Actions

| Severity | Description | Action | HtmlGraph Priority |
|----------|-------------|--------|-------------------|
| critical | Breaking issue, must fix | Create bug + escalate | high |
| high | Significant issue | Create bug | high |
| medium | Moderate issue | Create bug | medium |
| low | Minor issue | Log only | - |
| info | Informational | Skip | - |

## HtmlGraph Bug Fields

When creating bugs from roborev findings:

```python
sdk.bugs.create(
    title=f"[roborev] {finding['title']}",  # Include [roborev] prefix for filtering
).set_priority(
    'high' if finding['severity'] in ('high', 'critical') else 'medium'
).save()
```

## Filtering and Querying

Find all roborev bugs:

```bash
# Via HtmlGraph CLI
uv run htmlgraph status --filter "title:roborev"

# Via SQL
sqlite3 .htmlgraph/htmlgraph.db "
SELECT id, title, priority, created_at
FROM bugs
WHERE title LIKE '%roborev%'
ORDER BY created_at DESC;
"
```

## Auto-Trigger Configuration

To automatically run roborev after feature completion, add to your PostToolUse hook:

```python
# In .claude/hooks/scripts/posttooluse-integrator.py
if event.get('feature_completed'):
    # Trigger roborev agent
    Task(
        prompt="Run roborev on recent commits",
        subagent_type="htmlgraph:roborev"
    )
```

## Common Patterns

### Review After Merge

```bash
# Review commits since main
roborev review main..HEAD
```

### Review Specific File

```bash
# Review changes in specific file
roborev review HEAD -- src/module.py
```

### Address Review Items

Once you've fixed issues:

```bash
# Mark job as addressed
roborev address $JOB_ID

# Add comment explaining fix
roborev comment --job $JOB_ID "Fixed in commit abc123"
```

### Batch Reviews

```bash
# Review last 5 commits
for commit in $(git log --oneline -5 | awk '{print $1}'); do
    echo "Reviewing $commit..."
    roborev review $commit
done
```

## Troubleshooting

**roborev command not found:**
```bash
# Install and authenticate
pip install roborev
roborev auth login  # Enter API key from https://roborev.io
```

**Review job failed:**
```bash
# Check job logs
roborev show --job $JOB_ID --verbose

# List recent jobs
roborev list-jobs --limit 10
```

**HtmlGraph SDK errors:**
```bash
# Verify SDK installation
uv run python -c "from htmlgraph import SDK; print(SDK.__version__)"

# Check database
sqlite3 .htmlgraph/htmlgraph.db ".tables"
```

## Best Practices

1. **Review before commit** - Integrate into pre-commit hook
2. **Address findings promptly** - Don't let bugs accumulate
3. **Link to features** - Associate with originating feature
4. **Periodic reviews** - Run weekly on main branch
5. **Document exceptions** - If skipping findings, document why

## Advanced: Custom Finding Categories

```python
# Group findings by type
findings_by_type = {}
for finding in data.get('findings', []):
    ftype = finding.get('type', 'other')
    if ftype not in findings_by_type:
        findings_by_type[ftype] = []
    findings_by_type[ftype].append(finding)

# Create bugs organized by type
for ftype, findings in findings_by_type.items():
    for finding in findings:
        if finding['severity'] in ('high', 'critical', 'medium'):
            sdk.bugs.create(
                f"[roborev:{ftype}] {finding['title']}"
            ).set_priority(
                'high' if finding['severity'] in ('high', 'critical') else 'medium'
            ).save()
```

## Integration Points

- **Feature Completion Hook** - Automatically trigger after `sdk.features.complete()`
- **Pre-Commit Hook** - Integrate with git pre-commit
- **Deployment Gate** - Block deployment if critical findings exist
- **Metrics Dashboard** - Track finding trends over time

## For More Information

- **roborev docs**: https://roborev.io/docs
- **HtmlGraph SDK**: See `AGENTS.md` for feature/bug API
- **Agent code**: `packages/claude-plugin/agents/roborev.md`
