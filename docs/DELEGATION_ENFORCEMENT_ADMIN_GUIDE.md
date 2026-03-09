# Delegation Enforcement Admin Guide

Administrative guide for setting up and managing HtmlGraph delegation enforcement across teams and projects.

## Overview

HtmlGraph's delegation enforcement system ensures cost-optimal AI agent coordination by automatically enforcing delegation patterns, maintaining system prompt persistence across sessions, and monitoring compliance with orchestrator directives.

**Key Benefits:**
- Cost reduction through intelligent model selection (Haiku for simple tasks, Sonnet for complex work)
- Consistent agent behavior across session boundaries
- Automatic compliance monitoring without manual intervention
- Flexible deployment modes (strict enforcement or monitoring only)

## What is Delegation Enforcement?

Delegation enforcement is a system that:

1. **Enforces orchestrator directives** - Ensures AI agents delegate appropriately rather than executing all work directly
2. **Maintains system prompt persistence** - Injects project-specific guidance at session start, surviving compact/resume cycles
3. **Monitors compliance** - Tracks delegation patterns and alerts on violations
4. **Optimizes costs** - Routes work to the most appropriate model (Haiku, Sonnet, Opus)

### Orchestrator Pattern

The orchestrator pattern defines how agents should work:

```
✅ DO: Use Task() for work tracking, delegate complex operations to specialized subagents
✅ DO: Batch parallel tool calls to improve efficiency
✅ DO: Research before implementing (researcher agent)
✅ DO: Use specialized agents (debugger, test-runner, code-quality)

❌ DON'T: Execute all work directly without delegation
❌ DON'T: Skip research and jump to implementation
❌ DON'T: Ignore quality gates and testing
```

## Setup and Configuration

### Step 1: Enable System Prompt Persistence

Create `.claude/system-prompt.md` in your project root with your team's guidance:

```markdown
# Project System Prompt

## Your Team's Directives

[Your delegation rules, quality standards, and orchestrator directives]

## Cost-Optimal Models

- **Haiku** (fast, cheap) - Simple tasks, straightforward fixes
- **Sonnet** (balanced) - Complex features, multi-file refactoring
- **Opus** (powerful) - Strategic planning, architecture decisions

## Required Practices

1. Always read before write
2. Use absolute paths only
3. Run quality gates before committing
4. Delegate complex work to subagents
```

**Verification:**
```bash
# Test that system prompt persists across sessions
uv run pytest tests/hooks/test_system_prompt_persistence.py
```

### Step 2: Configure Delegation Modes

HtmlGraph supports two deployment modes:

#### Strict Mode (Enforcement)
```json
{
  "delegation_enforcement": {
    "mode": "strict",
    "enforcement_level": "high",
    "block_direct_execution": true,
    "model_selection_required": true
  }
}
```

**Behavior:**
- Direct tool execution blocked with detailed reflection
- All work requires task tracking
- Model selection enforced (Haiku/Sonnet/Opus)
- Violations logged and reported

#### Monitoring Mode (Advisory)
```json
{
  "delegation_enforcement": {
    "mode": "monitoring",
    "enforcement_level": "medium",
    "block_direct_execution": false,
    "provide_reflections": true
  }
}
```

**Behavior:**
- Direct execution allowed but flagged
- Helpful reflections guide agents toward better practices
- Compliance metrics tracked
- Non-blocking warnings on violations

### Step 3: Configure System Prompt Injection

System prompts auto-inject at session start via PostSessionStart hooks. Configure which layers to include:

```python
# .claude/config/system-prompt-config.json
{
  "layers": [
    {
      "name": "base_principles",
      "source": "PRINCIPLES.md",
      "priority": 1,
      "required": true
    },
    {
      "name": "project_rules",
      "source": ".claude/project-rules.md",
      "priority": 2,
      "required": true
    },
    {
      "name": "delegation_patterns",
      "source": ".claude/orchestrator-directives.md",
      "priority": 3,
      "required": true
    }
  ]
}
```

## Monitoring Delegation Compliance

### Real-Time Compliance Dashboard

View delegation compliance metrics in HtmlGraph:

```bash
# View overall compliance status
uv run htmlgraph status

# Check delegation patterns by agent
uv run htmlgraph delegation-report

# View cost optimization metrics
uv run htmlgraph costs --breakdown
```

### Compliance Metrics

Track these key metrics:

| Metric | Target | Formula |
|--------|--------|---------|
| **Task Coverage** | >90% | Tasks created / Total work items |
| **Delegation Rate** | >70% | Subagent invocations / Direct executions |
| **Model Efficiency** | >80% | Haiku use for simple tasks / Total simple tasks |
| **Batch Efficiency** | >60% | Parallel tool calls / Sequential tool calls |

### Automated Alerts

Configure alerts for compliance violations:

```python
# .claude/config/alerts.json
{
  "alerts": [
    {
      "type": "low_task_coverage",
      "threshold": 0.8,
      "action": "email_admins"
    },
    {
      "type": "direct_execution_spike",
      "threshold": 10,  // in 1 hour
      "action": "slack_notification"
    },
    {
      "type": "all_opus_usage",
      "threshold": 0.05,  // >5% Opus for non-strategic work
      "action": "warn_agent"
    }
  ]
}
```

## Team Deployment

### Multi-Team Setup

For organizations with multiple teams, configure per-team governance:

```
org/
├── .claude/
│   └── system-prompt.md              ← Organization defaults
├── team-a/
│   └── .claude/
│       └── system-prompt.md          ← Team A overrides
└── team-b/
    └── .claude/
        └── system-prompt.md          ← Team B overrides
```

**Hierarchy:** Team-specific > Organization > Project defaults

### Cost Allocation

Track costs per team/project:

```bash
# View costs by team
uv run htmlgraph costs --by-team

# View costs by project
uv run htmlgraph costs --by-project

# Export for billing
uv run htmlgraph costs --export csv --period monthly
```

## Troubleshooting

### Issue: System Prompt Not Persisting

**Symptom:** System prompt guidance disappears after session compact/resume

**Solution:**
```bash
# 1. Verify hook is executing
uv run htmlgraph hooks --type PostSessionStart

# 2. Check system prompt file exists
ls -la .claude/system-prompt.md

# 3. Test persistence directly
uv run pytest tests/hooks/test_system_prompt_persistence.py -v

# 4. Restart Claude Code to reload hooks
claude --restart
```

### Issue: Delegation Not Being Enforced

**Symptom:** Agents execute work directly without task creation or delegation

**Solution:**
```bash
# 1. Verify enforcement mode is enabled
grep "enforcement_level" .claude/config/*.json

# 2. Check hook is loaded
/hooks PreToolUse

# 3. Look for bypass flags in session
sqlite3 .htmlgraph/htmlgraph.db \
  "SELECT * FROM sessions WHERE status='active' LIMIT 1;"

# 4. Re-enable enforcement
uv run htmlgraph config set delegation_enforcement.mode strict
```

### Issue: Cost Overruns Due to Model Selection

**Symptom:** Haiku tasks running on Sonnet/Opus unexpectedly

**Solution:**
```bash
# 1. Review recent model assignments
uv run htmlgraph costs --breakdown --period 1d

# 2. Check model selection rules
cat .claude/config/model-selection.json

# 3. Analyze pattern (manual override?)
sqlite3 .htmlgraph/htmlgraph.db \
  "SELECT model, COUNT(*) FROM agent_events \
   WHERE timestamp > datetime('now', '-1 day') \
   GROUP BY model;"

# 4. Adjust rules or retrain agents on guidelines
```

### Issue: False Positive Alerts

**Symptom:** Alerts triggered but behavior is actually compliant

**Solution:**
```bash
# 1. Review alert thresholds
cat .claude/config/alerts.json

# 2. Analyze false positive context
uv run htmlgraph delegation-report --detailed

# 3. Adjust thresholds for your team
uv run htmlgraph config set alerts.direct_execution_spike.threshold 15

# 4. Add exclusions for legitimate direct execution
uv run htmlgraph config add alerts.exclusions.task_types emergency_fixes
```

## Best Practices

### 1. Incremental Rollout

Don't enable strict enforcement immediately. Phased approach:

```
Week 1: Monitoring mode (advisory only)
  → Establish baseline metrics
  → Train team on patterns

Week 2-3: Monitoring with alerts
  → Alert on violations
  → Coach agents on improvements

Week 4+: Strict mode with exceptions
  → Enforce directives
  → Allow emergency overrides
```

### 2. Clear Documentation

Ensure your system prompt is clear and actionable:

```markdown
# BAD - Vague
Delegate work appropriately.

# GOOD - Specific
For tasks >30 minutes estimated time:
1. Create Task() with clear acceptance criteria
2. Delegate to specialized subagent if available
3. Provide context in task description
```

### 3. Regular Review

Review delegation metrics weekly:

```bash
# Weekly review
cron: "0 9 * * 1"  # Monday 9am
task: "uv run htmlgraph delegation-report --export json > report-$(date +%Y%m%d).json"
```

### 4. Exception Handling

Define when exceptions are allowed:

```json
{
  "exceptions": {
    "emergency_fixes": {
      "allowed": true,
      "requires_annotation": true,
      "requires_followup_task": true
    },
    "spike_investigations": {
      "allowed": true,
      "time_limit_minutes": 30,
      "requires_report": true
    }
  }
}
```

## API Reference

### Configuration

```bash
# View current config
uv run htmlgraph config show

# Update setting
uv run htmlgraph config set delegation_enforcement.mode strict

# Reset to defaults
uv run htmlgraph config reset --section delegation_enforcement
```

### Reporting

```bash
# Delegation compliance report
uv run htmlgraph delegation-report [--format json|csv|html]

# Cost breakdown
uv run htmlgraph costs --breakdown --period [1d|1w|1m|all]

# Model usage metrics
uv run htmlgraph metrics --type models --period 1w

# Task coverage analysis
uv run htmlgraph metrics --type tasks --period 1w
```

### Hooks

Active hooks for delegation enforcement:

| Hook | Event | Action |
|------|-------|--------|
| `PostSessionStart` | Session begins | Inject system prompt |
| `PreToolUse` | Tool called | Check enforcement rules |
| `UserPromptSubmit` | User submits prompt | Create Task, track model |
| `PostToolUse` | Tool completes | Record metrics |

## Support and Feedback

For issues or feature requests:

1. **Check logs:** `.htmlgraph/errors.jsonl`
2. **Review metrics:** `uv run htmlgraph status`
3. **Test in isolation:** Run specific hooks with `--debug`
4. **Report issue:** Include metrics snapshot and recent logs

