# Tool Call Tracing Implementation Guide

## Overview

This guide provides step-by-step instructions to implement industry-standard event tracing for tool calls in Wipnote, based on research of Logfire, Langfuse, and OpenTelemetry patterns.

---

## Architecture Overview

### Current State
```
PostToolUse Hook
  ↓
Single event captured
  ↓
No duration calculation
  ↓
No correlation mechanism
```

### Target State
```
PreToolUse Hook          PostToolUse Hook
  ↓                        ↓
Record start           Record end + calculate
  ↓                        ↓
tool_use_id ←—— CORRELATE ——→ tool_use_id
  ↓                        ↓
Duration = end - start
```

---

## Phase 1: Immediate Implementation (Sprint 1)

### Step 1.1: Add PreToolUse Hook Handler

**File**: `/Users/shakes/DevProjects/htmlgraph/src/python/wipnote/event_log.py`

**What to add**:
```python
def log_pre_tool_use(self, event_data):
    """
    Log PreToolUse event.

    Args:
        event_data: Hook payload containing:
            - tool_use_id (str): Unique identifier for this tool execution
            - tool_name (str): Name of the tool being called
            - tool_input (dict): Arguments passed to the tool
            - timestamp (str): ISO8601 timestamp (from hook or generated)
    """
    pre_event = {
        'event_type': 'PreToolUse',
        'tool_use_id': event_data.get('tool_use_id'),
        'tool_name': event_data.get('tool_name'),
        'input': event_data.get('tool_input'),
        'start_time': event_data.get('timestamp') or self._iso_now(),
        'session_id': event_data.get('session_id'),
        'trace_id': event_data.get('trace_id'),
    }

    self._write_event(pre_event)
```

### Step 1.2: Enhance PostToolUse Handler

**What to add**:
```python
def log_post_tool_use(self, event_data):
    """
    Log PostToolUse event with duration calculation.

    Args:
        event_data: Hook payload containing:
            - tool_use_id (str): Correlates to PreToolUse
            - tool_name (str): Name of the tool
            - tool_output (dict): Return value
            - timestamp (str): ISO8601 timestamp
            - status (str): 'Ok' or 'Error'
    """
    end_time = event_data.get('timestamp') or self._iso_now()

    # Look up corresponding PreToolUse event for accurate duration
    start_event = self._find_event_by_tool_use_id(
        event_data.get('tool_use_id'),
        event_type='PreToolUse'
    )

    duration_ms = None
    if start_event:
        duration_ms = self._calculate_duration_ms(
            start_event['start_time'],
            end_time
        )

    post_event = {
        'event_type': 'PostToolUse',
        'tool_use_id': event_data.get('tool_use_id'),
        'tool_name': event_data.get('tool_name'),
        'input': event_data.get('tool_input'),
        'output': event_data.get('tool_output'),
        'end_time': end_time,
        'start_time': start_event['start_time'] if start_event else None,
        'duration_ms': duration_ms,
        'status': event_data.get('status', 'Ok'),
        'error_message': event_data.get('error_message'),
        'session_id': event_data.get('session_id'),
        'trace_id': event_data.get('trace_id'),
    }

    self._write_event(post_event)
```

### Step 1.3: Add Helper Methods

```python
def _find_event_by_tool_use_id(self, tool_use_id, event_type=None):
    """Find event by tool_use_id (for correlation)."""
    # Search JSONL file for matching event
    # Start from most recent entries (likely close in time)
    pass

def _calculate_duration_ms(self, start_time_iso, end_time_iso):
    """Calculate duration in milliseconds from ISO timestamps."""
    from datetime import datetime

    start = datetime.fromisoformat(start_time_iso.replace('Z', '+00:00'))
    end = datetime.fromisoformat(end_time_iso.replace('Z', '+00:00'))

    return (end - start).total_seconds() * 1000

def _iso_now(self):
    """Get current time in ISO8601 format with microseconds."""
    from datetime import datetime, timezone
    return datetime.now(timezone.utc).isoformat().replace('+00:00', 'Z')
```

### Step 1.4: Update Hook Configuration

**File**: `.claude/settings.json`

**Add or update PreToolUse hook**:
```json
{
  "hooks": [
    {
      "event": "PreToolUse",
      "script": "path/to/hook-handler.py",
      "description": "Capture tool execution start"
    },
    {
      "event": "PostToolUse",
      "script": "path/to/hook-handler.py",
      "description": "Capture tool execution completion and calculate duration"
    }
  ]
}
```

### Step 1.5: Create Hook Handler Script

**File**: `.claude/hooks/tool-call-tracer.py`

```python
#!/usr/bin/env python3
"""
Hook handler for tool call tracing.
Logs PreToolUse and PostToolUse events to Wipnote event log.
"""

import json
import sys
from wipnote.event_log import EventLog

def main():
    # Read hook input from stdin
    hook_payload = json.load(sys.stdin)

    event_type = hook_payload.get('hook_event_name')

    # Initialize event logger
    event_log = EventLog()

    try:
        if event_type == 'PreToolUse':
            event_log.log_pre_tool_use(hook_payload)
        elif event_type == 'PostToolUse':
            event_log.log_post_tool_use(hook_payload)
    except Exception as e:
        # Log errors but don't block hook execution
        print(f"Error in tool call tracer: {e}", file=sys.stderr)
        sys.exit(0)  # Success exit to not block

if __name__ == '__main__':
    main()
```

### Step 1.6: Testing

**Create test file**: `tests/test_tool_call_tracing.py`

```python
import pytest
from wipnote.event_log import EventLog

def test_pre_tool_use_logging():
    """Test PreToolUse event capture."""
    event_log = EventLog()

    event_log.log_pre_tool_use({
        'tool_use_id': 'test-123',
        'tool_name': 'search',
        'tool_input': {'query': 'test'},
        'session_id': 'sess-abc',
    })

    # Verify event was written
    # (implementation depends on storage backend)

def test_post_tool_use_with_duration():
    """Test PostToolUse duration calculation."""
    event_log = EventLog()

    # First log pre-event
    event_log.log_pre_tool_use({
        'tool_use_id': 'test-456',
        'tool_name': 'search',
        'tool_input': {'query': 'test'},
    })

    # Then log post-event
    event_log.log_post_tool_use({
        'tool_use_id': 'test-456',
        'tool_name': 'search',
        'tool_output': {'results': []},
        'status': 'Ok',
    })

    # Verify duration was calculated
    # (should be small, but > 0)

def test_tool_use_id_correlation():
    """Test that tool_use_id correlates pre and post events."""
    # Pre-event stored
    # Post-event retrieved it via tool_use_id
    # Duration calculated from timestamps
    pass
```

### Step 1.7: Validation Checklist

- [ ] PreToolUse hook fires before tool execution
- [ ] PostToolUse hook fires after tool execution
- [ ] tool_use_id matches between pre and post events
- [ ] Duration calculated as end_time - start_time
- [ ] Timestamps in ISO8601 format with microseconds
- [ ] Events written to event log (JSONL)
- [ ] No blocking of actual tool execution
- [ ] Error handling doesn't crash hooks

---

## Phase 2: Short-Term Enhancements (Sprint 2-3)

### Step 2.1: Parent-Child Relationships

Add support for nested tool calls:

```python
def log_pre_tool_use(self, event_data):
    """..."""
    pre_event = {
        # ... existing fields ...
        'parent_tool_use_id': event_data.get('parent_tool_use_id'),  # NEW
    }
```

### Step 2.2: Session-Level Grouping

Group all tool calls in a session:

```python
# Query all tools in a session
def get_session_tool_calls(self, session_id):
    """Return all tool calls in a session with their durations."""
    pass
```

### Step 2.3: Error Tracking

Enhance error information:

```python
'status': event_data.get('status', 'Ok'),
'error_message': event_data.get('error_message'),
'error_type': event_data.get('error_type'),
```

### Step 2.4: Indexing Strategy

Create indexes for efficient querying:

```sql
CREATE INDEX idx_tool_use_id ON tool_calls(tool_use_id) UNIQUE;
CREATE INDEX idx_trace_id_time ON tool_calls(trace_id, start_time DESC);
CREATE INDEX idx_session_id_time ON tool_calls(session_id, start_time DESC);
CREATE INDEX idx_tool_name ON tool_calls(tool_name);
```

---

## Phase 3: Analysis & Visualization (Sprint 4-5)

### Step 3.1: Query Patterns

Implement common queries:

```python
class ToolCallAnalytics:
    """Query and analyze tool call patterns."""

    def slowest_tools(self, limit=10):
        """Find slowest tools by average duration."""
        pass

    def error_rate_by_tool(self):
        """Calculate error rate for each tool."""
        pass

    def concurrent_tools(self, trace_id):
        """Find tools that executed in parallel."""
        pass

    def nested_calls(self, parent_tool_use_id):
        """Find child tool calls for a parent."""
        pass
```

### Step 3.2: Dashboard Widgets

Create visualization components:

```python
# Dashboard widget: Tool execution timeline
def render_tool_timeline(trace_id):
    """
    Render timeline of tool calls for a trace.
    Shows:
    - Tool name
    - Start/end times
    - Duration
    - Overlapping executions
    - Errors
    """
    pass

# Dashboard widget: Performance histogram
def render_duration_histogram(tool_name):
    """Show distribution of execution durations."""
    pass
```

---

## Phase 4: Advanced Features (Future)

### Step 4.1: Cost Tracking

Add optional cost calculation:

```python
'tokens_input': event_data.get('tokens_input'),
'tokens_output': event_data.get('tokens_output'),
'cost_usd': self._calculate_cost(
    tokens_input=...,
    tokens_output=...,
    model=...
),
```

### Step 4.2: Trace Visualization

Create interactive trace view:

```python
def render_trace_waterfall(trace_id):
    """
    Render Gantt-style waterfall of all spans in a trace.
    Shows parent-child relationships and parallel execution.
    """
    pass
```

### Step 4.3: Anomaly Detection

Identify slow operations:

```python
def detect_slow_tools(self, percentile=95):
    """
    Find tools running slower than the given percentile.
    Useful for performance debugging.
    """
    pass
```

---

## Data Schema Reference

### PreToolUse Event
```json
{
  "event_type": "PreToolUse",
  "tool_use_id": "abc-123",
  "tool_name": "search",
  "input": {
    "query": "example",
    "limit": 10
  },
  "start_time": "2025-01-07T10:30:00.123456Z",
  "session_id": "sess-abc",
  "trace_id": "trace-xyz",
  "parent_tool_use_id": null,
  "timestamp": "2025-01-07T10:30:00.123456Z"
}
```

### PostToolUse Event
```json
{
  "event_type": "PostToolUse",
  "tool_use_id": "abc-123",
  "tool_name": "search",
  "input": {
    "query": "example",
    "limit": 10
  },
  "output": {
    "results": [...]
  },
  "start_time": "2025-01-07T10:30:00.123456Z",
  "end_time": "2025-01-07T10:30:00.275000Z",
  "duration_ms": 151.544,
  "status": "Ok",
  "error_message": null,
  "session_id": "sess-abc",
  "trace_id": "trace-xyz",
  "parent_tool_use_id": null,
  "timestamp": "2025-01-07T10:30:00.275000Z"
}
```

---

## Integration Points

### 1. Claude Code Hooks
- PreToolUse hook defined in `.claude/settings.json`
- PostToolUse hook defined in `.claude/settings.json`
- Handler script at `.claude/hooks/tool-call-tracer.py`

### 2. Event Log Storage
- Events appended to `.wipnote/events/tool_calls.jsonl`
- One event per line
- ISO8601 timestamps
- Deterministic tool_use_id for correlation

### 3. SDK Integration
```python
from wipnote import SDK

sdk = SDK()

# Query tool calls
calls = sdk.events.filter(
    event_type='PostToolUse',
    trace_id='trace-xyz'
)

# Analyze
for call in calls:
    print(f"{call['tool_name']}: {call['duration_ms']}ms")
```

### 4. Dashboard Integration
- New "Tool Calls" section in activity feed
- Performance metrics widget
- Timeline visualization

---

## Migration Strategy

### For Existing Installations

1. **Non-Breaking**: New events stored separately
2. **Backward Compatible**: Old PostToolUse events still work
3. **Gradual Rollout**: Can enable PreToolUse capturing incrementally
4. **No Data Loss**: Existing event logs unchanged

### For New Installations

1. **Default**: PreToolUse + PostToolUse enabled by default
2. **Full Features**: Duration, correlation, hierarchy all available
3. **Performance**: Optimized indexes pre-created

---

## Testing Strategy

### Unit Tests
```bash
uv run pytest tests/test_tool_call_tracing.py -v
```

### Integration Tests
```bash
# Simulate tool calls and verify events logged
# Test correlation (pre → post matching)
# Test duration calculation
```

### E2E Tests
```bash
# Run actual Claude Code session
# Verify tool calls captured
# Check dashboard displays correctly
```

### Performance Tests
```bash
# Test with many concurrent tool calls
# Verify no performance regression
# Check memory usage
```

---

## Success Criteria

Phase 1 completion when:
- ✅ PreToolUse events captured
- ✅ PostToolUse events enhanced with duration
- ✅ tool_use_id correlation working
- ✅ Tests passing
- ✅ No performance regression
- ✅ Documentation complete

---

## References

- Full research: `/Users/shakes/DevProjects/htmlgraph/TRACING_RESEARCH.md`
- OpenTelemetry: https://opentelemetry.io/docs/concepts/signals/traces/
- Langfuse: https://langfuse.com/docs/observability/data-model
- Claude Code Hooks: https://code.claude.com/docs/en/hooks-guide

---

## Questions & Decisions

### Q: Should we store pre and post as separate events or merged?
**A**: Separate events (industry standard). Enables independent processing of each lifecycle stage.

### Q: How long to keep old PostToolUse format?
**A**: Deprecate after 2 releases, keep for backward compatibility.

### Q: Should duration calculation be optional?
**A**: No, always calculate when possible. Low cost, high value.

### Q: How to handle tools that don't return in PostToolUse?
**A**: Use timeout value for duration. Flag in status field.

---

## Next Steps

1. ✅ Research complete (see TRACING_RESEARCH.md)
2. ⬜ Implement Phase 1 (3-5 days)
3. ⬜ Add tests and validation (2-3 days)
4. ⬜ Deploy and monitor (1 day)
5. ⬜ Implement Phase 2+ (as prioritized)
