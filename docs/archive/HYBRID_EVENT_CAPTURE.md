# Hybrid Event Capture System - Parent-Child Event Nesting

## Overview

The Hybrid Event Capture System provides full traceability of delegated work by creating proper parent-child event relationships in Wipnote. When you delegate work to a subagent using `Task()`, the system creates a complete trace showing:

1. **Parent Event**: The Task() delegation itself (created by PreToolUse hook)
2. **Child Events**: Work performed by the subagent (captured by SubagentStop hook)
3. **Knowledge Created**: Spikes and findings generated during task execution

This enables complete visibility into delegated work that would otherwise be invisible due to Claude Code's hook isolation design.

## Architecture

### Event Flow Example

```
evt-123 Task(gemini-spawner) STARTED [16:40:54]
│   Input Summary:
│   - subagent_type: gemini-spawner
│   - prompt: "Analyze architecture and provide findings"
│
├─ spk-456 Architecture Analysis [created 16:41:02 by gemini agent]
│   findings: "System uses event-driven architecture..."
│
├─ spk-789 Research Spike [created 16:43:15 by gemini agent]
│   findings: "Identified optimization opportunities..."
│
└─ evt-123 Task(gemini-spawner) COMPLETED [16:45:22]
    Output Summary:
    - status: completed
    - child_spike_count: 2
    - completion_time: 2025-01-08T16:45:22Z
    - duration: 287 seconds
```

## Components

### 1. PreToolUse Hook (Create Parent Event)

**File**: `src/python/wipnote/hooks/pretooluse.py`

When a Task() call is detected, creates parent event:
- Extracts subagent type from Task() parameters
- Creates entry in agent_events table with type='task_delegation'
- Exports HTMLGRAPH_PARENT_EVENT to environment for subagent reference
- Non-blocking - gracefully degrades if database unavailable

### 2. SubagentStop Hook (Close Trace)

**File**: `src/python/wipnote/hooks/subagent_stop.py`

When a subagent completes:
- Reads HTMLGRAPH_PARENT_EVENT from environment
- Counts spikes created within 5-minute window
- Updates parent event with completion status and child spike count
- Handles missing parent events gracefully

### 3. Database Schema

Key tables and columns:

```sql
agent_events table:
- event_id: Unique event identifier (evt-XXXXX)
- event_type: 'task_delegation' for Task() calls
- parent_event_id: Links to parent event if nested
- subagent_type: Type of subagent (gemini-spawner, researcher, etc.)
- child_spike_count: Number of spikes created during task execution
- status: 'started' or 'completed'
- timestamp: ISO8601 UTC timestamp
```

### 4. API Endpoint: /api/event-traces

**Location**: `src/python/wipnote/api/main.py` (lines 573-754)

Retrieves parent-child event traces with full hierarchy and metadata.

## Usage Pattern

### Basic Task Delegation

```python
from wipnote import Task

Task(
    prompt="""
    Analyze the codebase architecture.
    Create a spike with your findings.
    """,
    subagent_type="gemini-spawner"
)
```

**Result**:
1. PreToolUse creates evt-123 (Task delegation started)
2. Subagent runs, creates spk-456 with findings
3. SubagentStop updates evt-123 (completed, 1 child spike)
4. Dashboard shows hierarchical trace: evt-123 → spk-456

## Dashboard Visualization

New "Event Traces" view shows:

```
[TASK] evt-abc123 gemini-spawner [COMPLETED] 2025-01-08T16:40:54
├─ Duration: 287.4s
└─ Child Events: 2
  ├─ Subagent Activity (2 events)
  │  ├─ ▪ delegation evt-xyz789 2025-01-08T16:41:02 [COMPLETED]
  │  ├─ ✦ spike spk-456 [Knowledge Created]
  │  ├─ ✦ spike spk-789 [Knowledge Created]
  │  └─
```

Features:
- Color-coded status badges
- Hierarchical display
- Duration calculations
- Child event/spike counts

## Verification Checklist

- ✅ Database schema has parent_event_id, subagent_type, child_spike_count
- ✅ PreToolUse hook detects Task() calls and creates parent events
- ✅ HTMLGRAPH_PARENT_EVENT exported to subagent environment
- ✅ SubagentStop hook updates parent event with completion info
- ✅ /api/event-traces endpoint returns parent-child structure
- ✅ Dashboard shows event nesting with visual indicators
- ✅ Test delegation: Task START → Subagent work → Task COMPLETE

## Performance

### Creation (PreToolUse)
- Time: <10ms (database insert)
- Non-blocking, graceful degradation on error

### Completion (SubagentStop)
- Time: 10-50ms (2 queries + 1 update)
- Spike counting uses 5-minute window

### Query (Dashboard)
- Time: 50-200ms for 50 traces
- Cached with 30-second TTL

## Test Scenario

1. Orchestrator calls Task() with prompt and subagent_type
2. PreToolUse hook creates evt-123, exports HTMLGRAPH_PARENT_EVENT
3. Subagent executes, creates spk-456 with findings
4. SubagentStop hook counts spikes, updates evt-123
5. Dashboard shows complete trace in Event Traces view

## References

- Schema: `src/python/wipnote/db/schema.py`
- PreToolUse Hook: `src/python/wipnote/hooks/pretooluse.py`
- SubagentStop Hook: `src/python/wipnote/hooks/subagent_stop.py`
- API Endpoint: `src/python/wipnote/api/main.py`
- Dashboard Template: `src/python/wipnote/api/templates/partials/event-traces.html`
