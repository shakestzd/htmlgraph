# Optimal Parallel Agent Workflow

This document defines the optimal workflow for parallel agent execution in Wipnote, based on:
- Transcript analytics from real parallel sessions
- Industry best practices (Anthropic, Microsoft, Google patterns)
- Anti-pattern detection and health scoring

---

## Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    PARALLEL WORKFLOW PHASES                      │
├─────────────────────────────────────────────────────────────────┤
│  1. PRE-FLIGHT    →  Analyze, partition, assess risk            │
│  2. CONTEXT PREP  →  Cache shared context, generate summaries   │
│  3. DISPATCH      →  Spawn agents with isolated tasks           │
│  4. MONITOR       →  Track health, detect drift, handle errors  │
│  5. AGGREGATE     →  Collect results, reconcile conflicts       │
│  6. VALIDATE      →  Verify outputs, update dependencies        │
└─────────────────────────────────────────────────────────────────┘
```

---

## Phase 1: Pre-Flight Analysis

**Goal:** Determine what can be parallelized and assess if parallelization is worth the cost.

### 1.1 Dependency Analysis

```bash
# Get parallel opportunities
wipnote analytics recommend --agent-count 5

# Check for blockers
wipnote analytics bottlenecks --top 3
# Fix blockers first to unlock more parallel work
```

### 1.2 Risk Assessment

```bash
# Get a project health overview (circular deps, orphaned tasks, etc.)
wipnote snapshot --summary

# Find bottlenecks to identify single points of failure
wipnote analytics bottlenecks
```

### 1.3 Cost-Benefit Decision

```
┌────────────────────────────────────────────────┐
│  PARALLELIZE WHEN:                             │
├────────────────────────────────────────────────┤
│  ✅ Tasks are truly independent (no shared deps)│
│  ✅ Each task takes >2 minutes                  │
│  ✅ No overlapping file edits                   │
│  ✅ Value justifies ~15x token cost            │
├────────────────────────────────────────────────┤
│  DON'T PARALLELIZE WHEN:                       │
├────────────────────────────────────────────────┤
│  ❌ Tasks share dependencies                    │
│  ❌ Quick tasks (<1 minute each)               │
│  ❌ Same files will be edited                   │
│  ❌ Sequential handoff needed                   │
└────────────────────────────────────────────────┘
```

---

## Phase 2: Context Preparation

**Goal:** Reduce redundant file reads by preparing shared context upfront.

### 2.1 Identify Shared Files

```python
# Before spawning agents, identify commonly needed files
shared_files = [
    "src/models.py",      # Data models
    "src/config.py",      # Configuration
    "tests/conftest.py",  # Test fixtures
]

# Read once, summarize for agents
context_cache = {}
for file in shared_files:
    content = Read(file)
    context_cache[file] = {
        "summary": summarize(content),  # 50-100 tokens
        "key_classes": extract_classes(content),
        "key_functions": extract_functions(content),
    }
```

### 2.2 Generate Task Context

Each agent receives:
```markdown
## Task: {feature_id}
Title: {title}
Priority: {priority}

## Your Assignment
{specific_instructions}

## Pre-Cached Context (DO NOT re-read these files)
- models.py: Contains User, Session, Feature classes
- config.py: DATABASE_URL, API_KEY settings
- conftest.py: pytest fixtures for db, client

## Files You Should Read
- {specific_files_for_this_task}

## Constraints
- DO NOT edit: {files_other_agents_are_editing}
- Focus ONLY on: {your_specific_scope}
```

### 2.3 Anti-Pattern Prevention

Based on transcript analytics, include these reminders:
```markdown
## Efficiency Guidelines
- Use Grep before Read (search → read, not read everything)
- Batch Edit operations (multiple changes in one edit)
- Use Glob to find files (not repeated Read attempts)
- Check cached context before reading shared files
```

---

## Phase 3: Dispatch

**Goal:** Spawn agents with isolated, well-defined tasks.

### 3.1 Optimal Dispatch Pattern

```python
# GOOD: Spawn all independent agents in single message
# This enables TRUE parallelism

tasks = [
    Task(
        subagent_type="general-purpose",
        prompt=f"""
Work on feature {task1_id}: {task1_title}

{task1_context}  # From Phase 2

Complete these steps:
1. {step1}
2. {step2}
3. Update feature file when done

Return: Summary of changes made, files modified, any blockers found.
""",
        description="Task 1: {short_desc}"
    ),
    Task(
        subagent_type="general-purpose",
        prompt=f"""...""",
        description="Task 2: {short_desc}"
    ),
    Task(
        subagent_type="general-purpose",
        prompt=f"""...""",
        description="Task 3: {short_desc}"
    ),
]

# All three run in parallel!
```

### 3.2 Task Isolation Rules

```
┌─────────────────────────────────────────────────────┐
│  ISOLATION REQUIREMENTS                              │
├─────────────────────────────────────────────────────┤
│  1. Separate feature IDs (one per agent)            │
│  2. Non-overlapping file edits                      │
│  3. Independent test files                          │
│  4. Clear scope boundaries                          │
├─────────────────────────────────────────────────────┤
│  IF OVERLAP UNAVOIDABLE:                            │
├─────────────────────────────────────────────────────┤
│  → Use sequential handoff instead                    │
│  → Or: Agent A writes, Agent B reads (not both edit)│
└─────────────────────────────────────────────────────┘
```

### 3.3 Agent Assignment (Capability Routing)

Use analytics to route tasks to appropriate agents:

```bash
# Get recommended agent assignments
wipnote analytics recommend --agent-count 5
# Returns tasks grouped by recommended agent count and priority
```

---

## Phase 4: Monitor

**Goal:** Track progress, detect issues early.

### 4.1 Health Metrics to Watch

```bash
# After agents complete, check session health
wipnote session list --recent 10

# Check specific session details
wipnote session show <session-id>
```

### 4.2 Anti-Pattern Detection

```python
# Real-time anti-pattern checks
ANTI_PATTERNS = {
    ("Read", "Read", "Read"): "Cache file content instead",
    ("Edit", "Edit", "Edit"): "Batch into single edit",
    ("Bash", "Bash", "Bash", "Bash"): "Check for errors",
    ("Grep", "Grep", "Grep"): "Read results before searching more",
}
```

### 4.3 Drift Detection

```python
# Monitor if agent is working on assigned feature
drift = manager.detect_drift(session_id, feature_id)

# Indicators of drift:
# - Time stalled >15 min
# - Repetitive tool patterns (5+ same tool)
# - High avg drift scores (>0.6)
# - Failed tool calls (3+ failures)
```

---

## Phase 5: Aggregate

**Goal:** Collect and reconcile results from parallel agents.

### 5.1 Result Collection

```python
# After all agents complete
results = {
    "agent_1": {
        "status": "success",
        "files_modified": ["auth.py", "test_auth.py"],
        "feature_completed": "feat-001",
        "duration_seconds": 235,
        "health_score": 0.75,
    },
    "agent_2": {
        "status": "success",
        "files_modified": ["api.py", "test_api.py"],
        "feature_completed": "feat-002",
        "duration_seconds": 198,
        "health_score": 0.84,
    },
    # ...
}
```

### 5.2 Conflict Detection

```python
# Check for file conflicts (shouldn't happen if Phase 2 done right)
all_modified = []
for result in results.values():
    for file in result["files_modified"]:
        if file in all_modified:
            raise ConflictError(f"Multiple agents modified {file}")
        all_modified.append(file)
```

### 5.3 Aggregate Metrics

```python
# Session-level summary
aggregate = {
    "total_agents": 3,
    "successful": 3,
    "failed": 0,
    "total_duration": sum(r["duration_seconds"] for r in results.values()),
    "parallel_speedup": max_duration / sequential_estimate,
    "avg_health": mean(r["health_score"] for r in results.values()),
    "anti_patterns_total": sum_anti_patterns(results),
}
```

---

## Phase 6: Validate

**Goal:** Verify outputs and update system state.

### 6.1 Validation Checklist

```python
# Post-parallel validation
validation = {
    "tests_pass": run_tests(),
    "no_conflicts": check_git_conflicts(),
    "features_updated": verify_feature_files_updated(),
    "dependencies_resolved": check_dependency_graph(),
}

if not all(validation.values()):
    trigger_remediation(validation)
```

### 6.2 Update Dependencies

```bash
# Mark completed features
wipnote feature complete feat-001
wipnote feature complete feat-002
wipnote feature complete feat-003

# Unlock Level 1 tasks (depended on Level 0)
wipnote analytics recommend --agent-count 5
# Now Level 1 tasks are in the ready list
```

### 6.3 Commit Aggregate Changes

```python
# Single commit for all parallel work
git add -A
git commit -m """feat: complete parallel work batch

Features completed:
- feat-001: {title1}
- feat-002: {title2}
- feat-003: {title3}

Parallel execution metrics:
- Agents: 3
- Duration: 235s (vs ~600s sequential)
- Health: 80% avg
"""
```

---

## Optimal Patterns Summary

### DO:

| Pattern | Description | Benefit |
|---------|-------------|---------|
| **Grep → Read** | Search before reading | Reduces context rebuilds |
| **Read → Edit → Bash** | Read, modify, test | Complete workflow |
| **Glob → Read** | Find files first | Avoids failed reads |
| **Single dispatch** | All Task calls in one message | True parallelism |
| **Pre-cached context** | Share summaries upfront | Reduces redundant reads |

### DON'T:

| Anti-Pattern | Problem | Solution |
|--------------|---------|----------|
| **Read → Read → Read** | Context rebuilds | Cache content |
| **Edit → Edit → Edit** | Unbatched edits | Combine edits |
| **Bash → Bash → Bash** | Command loops | Check errors |
| **Overlapping files** | Merge conflicts | Isolate scope |
| **Sequential Task calls** | Lost parallelism | Single message |

---

## Quick Reference

### Parallel Workflow Command

```bash
# Analyze what can be parallelized
wipnote analytics recommend --agent-count 5

# Check bottlenecks first
wipnote analytics bottlenecks --top 5

# Get project health overview
wipnote snapshot --summary
```

### Health Check After Parallel Work

```bash
# View recent session activity
wipnote session list --recent 10

# Check specific session
wipnote session show <session-id>
```

---

## Decision Tree

```
START: Need to complete multiple tasks
│
├─ Are tasks independent (no shared deps)?
│   ├─ YES → Check file overlap
│   │   ├─ No overlap → PARALLELIZE (this doc)
│   │   └─ Overlap → Sequential or partition files
│   └─ NO → Use dependency levels
│       └─ Complete Level 0 first, then Level 1, etc.
│
├─ Is each task >2 minutes?
│   ├─ YES → Worth parallelizing
│   └─ NO → Sequential may be simpler
│
└─ Is 15x token cost justified?
    ├─ YES → PARALLELIZE
    └─ NO → Sequential with handoffs
```

---

## References

- [Wipnote Transcript Analytics](./src/python/wipnote/transcript_analytics.py)
- [Dependency Analysis](./src/python/wipnote/analytics/dependency.py)
- [Multi-Agent Coordination Tests](./tests/integration/test_multi_agent_coordination.py)
- [Anthropic Multi-Agent Research System](https://www.anthropic.com/engineering/multi-agent-research-system)
- [Microsoft Agent Orchestration Patterns](https://learn.microsoft.com/en-us/azure/architecture/ai-ml/guide/ai-agent-design-patterns)
