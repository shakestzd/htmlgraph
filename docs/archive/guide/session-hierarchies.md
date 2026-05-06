# Session Hierarchies Guide

## What Are Session Hierarchies?

Session hierarchies capture the **parent-child relationships** created when you delegate work to subagents. Every time an orchestrator spawns a Task(), Wipnote automatically creates a session hierarchy that shows:

- **Who did the work** - Orchestrator agent, subagent types
- **When work happened** - Timeline of session creation and completion
- **What was delegated** - Prompts, task scope, constraints
- **How results link back** - Parent-child session relationships

This creates a complete **development trace** that survives agent crashes, context switches, and multi-day workflows.

---

## Why Session Hierarchies Matter

### Problem: Context Loss in Complex Workflows

Without session tracking:
```
Day 1 (Claude):
- Explore codebase
- Find 20 issues
- Document findings
- (Context lost when session ends)

Day 2 (Different agent):
- "What did Claude find yesterday?"
- (No record - have to re-explore)
```

### Solution: Session Hierarchies

With Wipnote tracking:
```
Orchestrator Session (Claude)
├── Child Session 1 (Subagent-Gemini): Explore codebase
│   └── Found 20 issues, documented in session summary
├── Child Session 2 (Subagent-Claude): Analyze issues
│   └── Prioritized issues by severity
└── Child Session 3 (Subagent-Test): Validate fixes
    └── All tests passed

Day 2: Pick up where you left off
├── Query: sdk.get_feature_sessions("feature-001")
├── View: All 4 sessions with full context
└── Continue: From the last session's findings
```

### Benefits

- ✅ **Complete lineage** - Full development history
- ✅ **Context recovery** - No information loss
- ✅ **Debugging aid** - Trace exactly what happened
- ✅ **Team visibility** - See other agents' work
- ✅ **Cost tracking** - Understand work distribution
- ✅ **Analytics** - Measure productivity per agent/session

---

## How Parent-Child Sessions Work

### Session Creation Flow

```
1. Orchestrator starts session
   └── session_id = "sess-abc123"
       agent = "orchestrator"
       status = "in-progress"

2. Orchestrator calls Task()
   └── Wipnote creates child session
       parent_session_id = "sess-abc123"
       subagent_type = "general-purpose"
       delegated_prompt = "..."
       status = "pending" → "in-progress"

3. Subagent executes work
   └── Child session captures all tool calls
       tools_used = ["Bash", "Read", "Edit", "Grep"]
       events = [...]  # All activities logged
       status = "in-progress"

4. Subagent completes
   └── Child session summary created
       result = "..."
       status = "completed"
       total_time = "2m 34s"

5. Results available to orchestrator
   └── Parent can query child:
       wipnote session show sess-child-xyz
```

### Event Capture

Wipnote captures everything that happens in a session:

```json
{
  "session_id": "sess-child-xyz",
  "parent_session_id": "sess-abc123",
  "agent": "subagent-gemini",
  "status": "completed",
  "events": [
    {
      "type": "ToolUse",
      "tool": "Bash",
      "command": "uv run pytest tests/unit/",
      "result": "45 passed, 2 failed"
    },
    {
      "type": "ToolUse",
      "tool": "Grep",
      "pattern": "TODO:",
      "result": "Found 12 TODOs"
    }
  ],
  "summary": "Ran tests and found issues",
  "total_time_seconds": 154
}
```

---

## Viewing Session Hierarchies

### Using the CLI

```bash
# List all sessions
wipnote session list

# Get session details (includes child sessions)
wipnote session show sess-abc123

# View session hierarchy as a tree
wipnote session tree sess-abc123

# Find sessions for a feature
wipnote session find-feature feature-001
```

### Using the Dashboard

The Wipnote dashboard has an **Orchestration** tab that visualizes hierarchies:

```
┌─────────────────────────────────────────┐
│ Orchestrator Session (Claude)           │
│ Duration: 5m 23s | Status: completed   │
├─────────────────────────────────────────┤
│                                         │
│  ├─ Child: Test Runner (Haiku)         │
│  │  ├─ Tests: 145 passed, 0 failed     │
│  │  └─ Time: 1m 45s                    │
│  │                                     │
│  ├─ Child: Code Analyzer (Gemini)      │
│  │  ├─ Issues found: 3                 │
│  │  └─ Time: 2m 10s                    │
│  │                                     │
│  └─ Child: Documenter (Haiku)          │
│     ├─ Files updated: 5                │
│     └─ Time: 1m 28s                    │
│                                         │
└─────────────────────────────────────────┘
```

Click on any session to see:
- Full event log
- Tool calls executed
- Results and output
- Timing breakdown
- Agent details

### Using the CLI (continued)

```bash
# All the commands above work with uv run wipnote as well
uv run wipnote session list
uv run wipnote session show sess-abc123
uv run wipnote session tree sess-abc123
uv run wipnote session find-feature feature-001
```

---

## Understanding Session Events

### Event Types

Each session captures different event types:

#### ToolUse Event
```python
{
    "type": "ToolUse",
    "timestamp": "2025-01-10T15:30:45Z",
    "tool": "Bash",
    "input": "uv run pytest tests/",
    "result": "45 passed, 2 failed",
    "duration_ms": 3245
}
```

#### FileEdit Event
```python
{
    "type": "FileEdit",
    "timestamp": "2025-01-10T15:31:20Z",
    "file": "src/auth/login.py",
    "old_content": "...",
    "new_content": "...",
    "reason": "Fix token validation"
}
```

#### SessionStart Event
```python
{
    "type": "SessionStart",
    "timestamp": "2025-01-10T15:30:00Z",
    "agent": "subagent-gemini",
    "parent_session_id": "sess-abc123",
    "delegated_prompt": "Run tests and report failures"
}
```

#### SessionEnd Event
```python
{
    "type": "SessionEnd",
    "timestamp": "2025-01-10T15:34:30Z",
    "status": "completed",
    "result": "All tests passed",
    "total_events": 47,
    "tools_used": ["Bash", "Read", "Grep"]
}
```

### Analyzing Events

```bash
# View session details including event counts
wipnote session show sess-child-xyz

# View full event log in browser
open .wipnote/sessions/sess-child-xyz.html
```

---

## Debugging with Session Traces

### Scenario 1: "What did the subagent do?"

```bash
# View session hierarchy starting from parent
wipnote session tree sess-abc123

# Inspect individual child session details
wipnote session show sess-child-xyz
```

### Scenario 2: "Did the orchestrator make the right decision?"

```bash
# View orchestrator session and all delegated children
wipnote session tree sess-orchestrator-id

# Check overall session status
wipnote session show sess-orchestrator-id
```

### Scenario 3: "Why did feature X take so long?"

```bash
# View all sessions linked to a feature
wipnote session find-feature feature-001

# View session timeline in dashboard
uv run wipnote serve
# Navigate to Sessions tab, filter by feature
```

### Scenario 4: "Trace a specific issue across sessions"

```bash
# Find sessions for the feature then inspect each
wipnote session find-feature feature-001
wipnote session show sess-abc123

# Open session files in browser for full event logs
open .wipnote/sessions/sess-abc123.html
```

---

## Session Hierarchy Patterns

### Pattern 1: Orchestrator + Parallel Subagents

```
Orchestrator (Claude)
├── Subagent 1 (Gemini): Explore codebase
├── Subagent 2 (Copilot): Check GitHub issues
└── Subagent 3 (Claude): Analyze security

All run in parallel
Results: Available immediately after all complete
```

**Query pattern:**
```bash
# View orchestrator session and all parallel children
wipnote session tree sess-orchestrator

# All children are visible in the hierarchy
wipnote session show sess-orchestrator
```

### Pattern 2: Sequential Handoff

```
Task 1 (Exploration)
└── Finds issues

Task 2 (Analysis - uses Task 1 results)
└── Prioritizes issues

Task 3 (Implementation - uses Task 2 results)
└── Fixes issues
```

**Query pattern:**
```bash
# Each Task() call in sequence — results from task1 are passed in task2's prompt
# View the chain via session hierarchy
wipnote session tree sess-orchestrator
```

### Pattern 3: Hierarchical Delegation (Nested)

```
Orchestrator (Level 0)
└── Subagent 1 (Level 1): Complex task
    ├── Grandchild 1 (Level 2): Sub-task A
    ├── Grandchild 2 (Level 2): Sub-task B
    └── Grandchild 3 (Level 2): Sub-task C
```

**Query pattern:**
```bash
# View full multi-level hierarchy
wipnote session tree sess-level0

# Inspect any level by session ID
wipnote session show sess-level1-child
```

---

## Best Practices

### 1. Use Meaningful Feature Linking

Link sessions to features so you can trace all work:

```python
# Good: Feature context included in delegation prompt
Task(
    subagent_type="general-purpose",
    prompt=f"""
    Feature: feature-001 - User Authentication

    Task: Implement login endpoint
    ...
    """
)
```

```bash
# Later: Query all work on feature
wipnote session find-feature feature-001
```

### 2. Document Key Decisions

When delegating critical work, document the decision:

```bash
wipnote spike create "Delegated auth refactoring to parallel subagents"
# Then open the spike HTML and add findings
```

### 3. Monitor Session Health

Periodically review session patterns:

```bash
# View all sessions for a feature
wipnote session find-feature feature-001

# Check for slow or failed sessions in dashboard
uv run wipnote serve
```

### 4. Use Hierarchy for Cost Attribution

```bash
# View session hierarchy to understand work distribution
wipnote session tree sess-orchestrator

# Dashboard shows agent attribution across sessions
uv run wipnote serve
```

### 5. Preserve Context in Handoffs

When handing off between agents, document findings in a spike:

```bash
wipnote spike create "Handoff context from <agent>"
# Add findings to the spike file, reference session and feature IDs
```

---

## FAQ

**Q: Are parent-child sessions created automatically?**

A: Yes! Wipnote automatically creates child sessions when you call Task(). You don't need to manually create them.

**Q: Can I query sessions from completed features?**

A: Yes. Sessions are stored permanently in `.wipnote/sessions/`. You can query them anytime, even if the feature is complete.

**Q: What's the deepest hierarchy I should create?**

A: Usually 2-3 levels (orchestrator → subagent → grandchild). Beyond that, complexity outweighs benefits.

**Q: Do I need to know session IDs?**

A: No. Use `wipnote session find-feature <feature_id>` to get all sessions for a feature without knowing IDs.

**Q: Can different agents see the same session hierarchy?**

A: Yes. Sessions are shared in `.wipnote/sessions/`. Any agent with access to the directory can see all sessions.

**Q: How long are session records kept?**

A: Indefinitely. Sessions are stored in the `.wipnote/` directory which should be committed to git. They're part of your project history.

**Q: Can I manually create a session hierarchy?**

A: Usually not needed. Use Task() for automatic creation. Session relationships are tracked automatically via hooks.

---

## Related Reading

- [Delegation Guide](delegation.md) - How to write effective delegation prompts
- [AGENTS.md - Parent-Child Session Tracking](../AGENTS.md#parent-child-session-tracking) - Quick overview
- [AGENTS.md - Agent Handoff Context](../AGENTS.md#agent-handoff-context) - Handoff patterns
- [AGENTS.md - Orchestrator Mode](../AGENTS.md#orchestrator-mode) - Session tracking benefits
- `examples/session_analysis.py` - Complete session analysis examples
