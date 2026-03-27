---
name: diagnose
description: Diagnose orchestrator delegation enforcement gaps in the current session. Use when asked "why didn't you delegate?" or to audit delegation compliance.
user_invocable: true
---

# Diagnose Skill

## When to Activate

Trigger keywords:
- "why didn't you delegate"
- "delegation audit", "diagnose"
- "delegation score", "enforcement gaps"
- "did you delegate", "should have delegated"

## Instructions for Claude

Run the following analysis and present the results as a delegation diagnostic report.

### Step 1: Collect Data

```bash
# Get current session events via CLI
htmlgraph status
htmlgraph session list

# For detailed event analysis, query SQLite directly
sqlite3 .htmlgraph/htmlgraph.db "
SELECT tool_name, COUNT(*) as count
FROM agent_events
WHERE session_id = (SELECT session_id FROM agent_events ORDER BY timestamp DESC LIMIT 1)
GROUP BY tool_name ORDER BY count DESC;
"
```

For a programmatic approach (when reading from DB directly):

```python
import sqlite3

db_path = ".htmlgraph/htmlgraph.db"
conn = sqlite3.connect(db_path)

# Get current session ID (most recent session)
row = conn.execute(
    "SELECT session_id FROM agent_events ORDER BY timestamp DESC LIMIT 1"
).fetchone()
session_id = row[0] if row else None

direct_ops = []
git_writes = []
delegations = []
direct_impl = []

if session_id:
    direct_ops = conn.execute("""
        SELECT event_id, tool_name, input_summary, timestamp
        FROM agent_events
        WHERE session_id = ? AND tool_name = 'Bash'
          AND input_summary NOT LIKE '%ruff%'
          AND input_summary NOT LIKE '%pytest%'
          AND input_summary NOT LIKE '%mypy%'
          AND input_summary NOT LIKE '%git status%'
          AND input_summary NOT LIKE '%git log%'
          AND input_summary NOT LIKE '%git diff%'
          AND input_summary NOT LIKE '%git show%'
          AND input_summary NOT LIKE '%ls %'
        ORDER BY timestamp
    """, (session_id,)).fetchall()

    git_writes = [
        op for op in direct_ops
        if any(kw in (op[2] or '') for kw in [
            'git commit', 'git push', 'git tag', 'git merge',
            'git rebase', 'git reset', 'git branch -d'
        ])
    ]

    delegations = conn.execute("""
        SELECT event_id, tool_name, input_summary, timestamp
        FROM agent_events
        WHERE session_id = ? AND tool_name IN ('Task', 'Agent')
        ORDER BY timestamp
    """, (session_id,)).fetchall()

    direct_impl = conn.execute("""
        SELECT event_id, tool_name, input_summary, timestamp
        FROM agent_events
        WHERE session_id = ? AND tool_name IN ('Edit', 'Write')
        ORDER BY timestamp
    """, (session_id,)).fetchall()

conn.close()
```

### Step 2: Compute Score

```python
# Delegation score: ratio of Task/Agent calls to (Task/Agent + Edit/Write + git writes)
implementation_actions = len(delegations) + len(direct_impl) + len(git_writes)
delegation_score = (
    int(len(delegations) / implementation_actions * 100)
    if implementation_actions > 0 else 100
)
```

### Step 3: Format Report

Present the report in this format:

```
## Delegation Diagnostic Report

### Orchestrator State
- Mode: enabled/disabled
- Enforcement: strict/guidance
- Violations: N/3
- Circuit breaker: triggered/normal

### Delegation Score: X% (N/M actions delegated)

### Gaps Found

#### Git Write Operations (should delegate to copilot skill)
| Time | Command | Should Use |
|------|---------|------------|
| 12:34 | git commit -m "..." | /htmlgraph:copilot |
| 12:35 | git push | /htmlgraph:copilot |

#### Direct Implementation (should delegate to coder agent)
| Time | Tool | File | Should Use |
|------|------|------|------------|
| 12:30 | Edit | src/foo.py | Agent("htmlgraph:sonnet-coder") |

### Recommendations
1. [Based on gaps found, give specific actionable recommendations]
2. Enable strict mode: `uv run htmlgraph orchestrator enable --level strict`
3. Use /htmlgraph:copilot skill for git ops
4. Delegate Edit/Write to coder agents via Task()
```

If no session data is found, report: "No events found in current session. Ensure hooks are running."

If delegation score >= 80%, report success with no gaps to fix.
