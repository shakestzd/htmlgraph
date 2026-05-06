# Spawner Internal Activity Tracking Implementation

**Date:** 2025-01-10
**Status:** IMPLEMENTED
**Scope:** Gemini, Codex, and Copilot spawner agents

## Overview

This implementation adds internal activity tracking to spawner agents, making work performed in isolated spawned sessions visible in Wipnote with proper parent-child event linking.

## Problem Statement

Previously, spawner agents (Gemini, Codex, Copilot) only recorded delegation start/end events. Internal phases like initialization, execution, and completion were invisible to the dashboard activity feed.

## Solution Architecture

### 1. SpawnerEventTracker Helper Module

**File:** `packages/claude-plugin/.claude-plugin/agents/spawner_event_tracker.py`

A utility class that:
- Records phase transitions during spawner execution
- Links child events to parent delegation events
- Tracks timing and execution metrics
- Integrates with WipnoteDB for persistence

**Key Features:**
- Parent-child event linking via `parent_event_id`
- Phase duration tracking
- Graceful degradation (tracking is optional)
- Environment variable integration for parent context

**Usage:**
```python
tracker = SpawnerEventTracker(
    delegation_event_id="event-abc123",
    parent_agent="orchestrator",
    spawner_type="gemini"
)

# Record phases
init_event = tracker.record_phase("Initializing Spawner", ...)
exec_event = tracker.record_phase("Executing Gemini", ...)

# Complete phases with results
tracker.complete_phase(exec_event["event_id"], output_summary="...", status="completed")
```

### 2. Updated Spawner Agents

#### Gemini Spawner
**File:** `packages/claude-plugin/.claude-plugin/agents/gemini-spawner.py`

Tracks:
- Initializing Spawner → Executing Gemini → Completion

#### Codex Spawner
**File:** `packages/claude-plugin/.claude-plugin/agents/codex-spawner.py`

Tracks:
- Initializing Codex Spawner → Setting Up Sandbox → Executing Codex → Completion

#### Copilot Spawner
**File:** `packages/claude-plugin/.claude-plugin/agents/copilot-spawner.py`

Tracks:
- Initializing Copilot Spawner → Authenticating GitHub → Executing Copilot → Completion

### 3. Event Hierarchy Structure

```
Delegation Event (event-abc123)
├─ Child Event: Initialize Spawner
│  │  Status: completed
│  │  Duration: 0.5s
│  └─ Tool: HeadlessSpawner.initialize
│
├─ Child Event: Setup Sandbox (Codex only)
│  │  Status: completed
│  │  Duration: 1.2s
│  └─ Tool: HeadlessSpawner.setup_sandbox
│
├─ Child Event: Execute [Gemini/Codex/Copilot]
│  │  Status: completed
│  │  Duration: 45.3s
│  └─ Tool: [gemini|codex|copilot]-cli
│
└─ Child Event: Process Result
   Status: completed
   Duration: 0.2s
   Tool: HeadlessSpawner.complete
```

## Implementation Details

### Database Schema

Child events use these fields to link to parents:

```sql
-- In agent_events table
parent_event_id TEXT          -- Links to parent delegation event
event_type TEXT               -- 'tool_use' for child events
subagent_type TEXT            -- 'gemini', 'codex', 'copilot'
status TEXT                   -- 'running', 'completed', 'failed'
execution_duration_seconds    -- Time for this phase
created_at DATETIME           -- Timestamp when created
```

### Phase Recording Flow

For each spawner, the flow is:

1. **Create delegation event** (parent)
   - Records Task() delegation
   - Gets `delegation_event_id`

2. **Initialize tracker**
   - Links to parent via `HTMLGRAPH_PARENT_EVENT`
   - Prepares for child event creation

3. **Record init phase**
   - Creates child event: "Initializing Spawner"
   - `parent_event_id = delegation_event_id`
   - Status: running

4. **Record execution phases**
   - "Setting Up Sandbox" (Codex only)
   - "Executing [Gemini/Codex/Copilot]"
   - Status: running

5. **Execute spawner**
   - Actual work happens

6. **Complete execution phase**
   - Updates child event with results
   - Status: completed/failed

7. **Complete other phases**
   - Updates remaining child events
   - Finalizes timing

8. **Update delegation event**
   - Final status and metrics

### Parent Context via Environment Variables

Spawner agents read parent context from environment:

```python
parent_event_id = os.getenv("HTMLGRAPH_PARENT_EVENT")      # Parent delegation
parent_session = os.getenv("HTMLGRAPH_PARENT_SESSION")    # Parent session
parent_agent = os.getenv("HTMLGRAPH_PARENT_AGENT")        # Agent that delegated
```

These are set in the spawner wrapper before executing the spawned agent.

## Testing & Verification

### Verification Script

**File:** `verify_spawner_tracking.py`

Run with:
```bash
python verify_spawner_tracking.py
```

Checks:
1. Delegation events exist
2. Child events are properly linked
3. Event hierarchy is correct
4. Timing metrics are captured
5. Spawner breakdown statistics

### Manual Testing

```bash
# Test Gemini spawner
python -m wipnote.spawner gemini \
  -p "Write hello world" \
  -m gemini-2.0-flash

# Test Codex spawner
python -m wipnote.spawner codex \
  -p "Implement feature" \
  --sandbox read-only

# Test Copilot spawner
python -m wipnote.spawner copilot \
  -p "Debug this" \
  --allow-all-tools
```

### Database Verification

```bash
# Query parent-child relationships
uv run python3 << 'EOF'
from wipnote.db.schema import WipnoteDB
db = WipnoteDB()
cursor = db.connection.cursor()

# Get delegations with child count
cursor.execute("""
    SELECT e.event_id, e.agent_id, COUNT(c.event_id) as child_count
    FROM agent_events e
    LEFT JOIN agent_events c ON c.parent_event_id = e.event_id
    WHERE e.event_type = 'delegation'
    GROUP BY e.event_id
    ORDER BY child_count DESC
""")

for row in cursor.fetchall():
    print(f"{row[0]}: {row[1]} children")
EOF
```

## Benefits

1. **Complete Observability**
   - See internal phases in activity feed
   - Track time spent in each phase
   - Identify performance bottlenecks

2. **Better Debugging**
   - Identify which phase failed
   - See timing for each operation
   - Understand spawner behavior

3. **Analytics**
   - Average init time per spawner
   - Total execution time breakdown
   - Error tracking by phase

4. **Dashboard Integration**
   - Events visible in Wipnote
   - Hierarchical display
   - Filtering by spawner type

## Backward Compatibility

- Tracking is optional (graceful degradation)
- Existing spawner APIs unchanged
- Works with or without WipnoteDB
- No performance impact if tracking disabled

## Future Enhancements

1. **Dashboard Filtering**
   - Filter by spawner type
   - View time breakdown per phase
   - Compare across spawners

2. **Performance Optimization**
   - Identify slowest phases
   - Optimize initialization overhead
   - Cache frequent operations

3. **Cost Analysis**
   - Track tokens per phase
   - Estimate costs by spawner
   - Optimize expensive phases

4. **Distributed Tracking**
   - Support cross-process event linking
   - Track async operations
   - Handle timeouts gracefully

## Files Modified

1. **New:**
   - `packages/claude-plugin/.claude-plugin/agents/spawner_event_tracker.py`
   - `verify_spawner_tracking.py`
   - `SPAWNER_TRACKING_IMPLEMENTATION.md`

2. **Updated:**
   - `packages/claude-plugin/.claude-plugin/agents/gemini-spawner.py`
   - `packages/claude-plugin/.claude-plugin/agents/codex-spawner.py`
   - `packages/claude-plugin/.claude-plugin/agents/copilot-spawner.py`

## Testing Checklist

- [x] SpawnerEventTracker created and functional
- [x] Gemini spawner updated with phase tracking
- [x] Codex spawner updated with phase tracking
- [x] Copilot spawner updated with phase tracking
- [x] Verification script created
- [ ] Manual testing of each spawner
- [ ] Database verification of parent-child links
- [ ] Dashboard visibility testing

## Example Event Hierarchy

From recent test (event-9910cd38):

```
Delegation Event (event-9910cd38)
  Agent: claude-code
  Type: delegation
  Tool: Task
  Input: Get git status...
  Status: completed
  Duration: 100.51s
  Tokens: 0

  Child Events: (to be populated on next spawner run)
    1. Initializing Spawner
    2. Executing Gemini
    (or similar for Codex/Copilot with more phases)
```

## References

- WipnoteDB schema: `src/python/wipnote/db/schema.py`
- Agent events table: `agent_events` with `parent_event_id` foreign key
- Environment variables: Set by spawner wrapper, read by tracker
