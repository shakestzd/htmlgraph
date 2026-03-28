---
name: htmlgraph-tracker
description: ARCHIVED — Use htmlgraph skill instead. HtmlGraph workflow combining session tracking, orchestration, and parallel coordination.
---

<!-- ARCHIVED: This skill has been superseded by the htmlgraph skill -->
<!-- Python SDK references removed — use Go CLI commands instead -->

# HtmlGraph Tracker Skill (ARCHIVED)

> **This skill is archived.** Use `/htmlgraph:htmlgraph` for current workflow patterns.

---

## Core Workflow

```bash
# Session start
htmlgraph status
htmlgraph analytics summary

# Create and track work
htmlgraph feature create "Title"
htmlgraph feature start <feat-id>

# Mark complete
htmlgraph feature complete <feat-id>
```

---

## Work Item Commands

```bash
# Features
htmlgraph feature create "Title"
htmlgraph feature start <feat-id>
htmlgraph feature complete <feat-id>
htmlgraph find features --status todo
htmlgraph find features --status in-progress

# Bugs
htmlgraph bug create "Title"
htmlgraph bug start <bug-id>
htmlgraph bug complete <bug-id>

# Spikes (investigation)
htmlgraph spike create "Title"
htmlgraph spike start <spike-id>
htmlgraph spike complete <spike-id>

# Tracks (multi-feature initiatives)
htmlgraph track new "Title"
```

---

## Analytics

```bash
htmlgraph analytics summary
htmlgraph analytics summary
htmlgraph snapshot --summary
htmlgraph find features --status todo
```

---

## Parallel Orchestration

Dispatch independent tasks in a single message:

```python
# All in one message = parallel execution
Task(subagent_type="htmlgraph:gemini-operator", prompt="Research...")
Task(subagent_type="htmlgraph:sonnet-coder", prompt="Implement feat-123...")
Task(subagent_type="htmlgraph:sonnet-coder", prompt="Implement feat-456...")
```

See `/htmlgraph:orchestrator-directives-skill` for complete patterns.

---

## Work Type Classification

Work type is inferred from work item ID prefix:
- `feat-*` → feature-implementation
- `spike-*` → spike-investigation
- `bug-*` → bug-fix
- `chore-*` → maintenance
