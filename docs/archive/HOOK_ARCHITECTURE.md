# Hook Architecture

Complete reference for Claude Code hook system in Wipnote.

## Overview

Wipnote hooks integrate with Claude Code's hook system to provide:
- **Event tracking** - Record all agent activities and tool invocations
- **Validation** - Check work items before execution
- **Orchestration** - Enforce delegation patterns
- **Cost accounting** - Track token usage via CIGS
- **Tracing** - Correlate tool executions with millisecond precision

## Hook Types

### PreToolUse Hook

Fires **before** any tool is executed by Claude Code.

**Purpose:**
- Generate and store tool execution start events
- Capture tool name and input parameters
- Initiate event tracing via tool_use_id correlation
- Validate that tool execution is allowed

**Location:** `src/python/wipnote/hooks/pretooluse.py`

**Flow:**

```
Tool Invocation
    ↓
PreToolUse Hook
├── Generate tool_use_id (UUID v4)
├── Capture tool_name, tool_input, start_time
├── Sanitize sensitive data
├── Insert start event into tool_traces table
├── Store tool_use_id in environment (HTMLGRAPH_TOOL_USE_ID)
└── Return continue=True to allow execution
    ↓
Tool Executes
```

**Implementation Details:**

```python
def create_start_event(
    tool_name: str,
    tool_input: dict[str, Any],
    session_id: str
) -> str | None:
    """
    Capture and store tool execution start event.

    Args:
        tool_name: Tool being executed (e.g., "Bash", "Read")
        tool_input: Input parameters (will be sanitized)
        session_id: Current session ID

    Returns:
        tool_use_id for correlation, or None on error
    """
    # 1. Generate tool_use_id
    tool_use_id = str(uuid.uuid4())

    # 2. Get start_time in UTC
    start_time = datetime.now(timezone.utc).isoformat()

    # 3. Sanitize input (remove passwords, tokens, etc.)
    sanitized_input = sanitize_tool_input(tool_input)

    # 4. Insert into tool_traces
    db = WipnoteDB()
    cursor = db.connection.cursor()
    cursor.execute("""
        INSERT INTO tool_traces
        (tool_use_id, trace_id, session_id, tool_name,
         tool_input, start_time, status)
        VALUES (?, ?, ?, ?, ?, ?, ?)
    """, (tool_use_id, tool_use_id, session_id, tool_name,
          json.dumps(sanitized_input), start_time, "started"))
    db.connection.commit()

    # 5. Store in environment for PostToolUse
    os.environ["HTMLGRAPH_TOOL_USE_ID"] = tool_use_id

    return tool_use_id
```

**Runs In Parallel With:**
- Orchestrator enforcement
- Task validation
- Event tracing (this function)

**Error Handling:**
- Non-blocking - errors don't prevent tool execution
- Missing session - skip tracing
- Database unavailable - log warning, continue
- Invalid input - sanitize gracefully

**Related:** [EVENT_TRACING.md - PreToolUse Hook](./EVENT_TRACING.md#pretooluse-hook)

### PostToolUse Hook

Fires **after** any tool completes execution (success or failure).

**Purpose:**
- Record tool execution completion
- Calculate execution duration
- Store tool output and error status
- Enable performance analysis and debugging

**Location:** `src/python/wipnote/hooks/posttooluse.py` and `src/python/wipnote/hooks/post_tool_use_handler.py`

**Flow:**

```
Tool Execution Completes
    ↓
PostToolUse Hook
├── Read tool_use_id from environment
├── Query tool_traces for matching pre-event
├── Calculate duration (end_time - start_time)
├── Determine status (Ok/Error) from response
├── Update tool_traces with: end_time, duration_ms, output, status
└── Return continue=True to allow execution to continue
    ↓
Execution Resumes
```

**Implementation Details:**

```python
def update_tool_trace(
    tool_use_id: str,
    tool_output: dict[str, Any],
    status: str,
    error_message: str | None = None
) -> bool:
    """
    Update tool_traces table with execution end event.

    Args:
        tool_use_id: Correlation ID from PreToolUse
        tool_output: Tool execution result
        status: 'Ok' or 'Error'
        error_message: Error details if status='Error'

    Returns:
        True if update successful, False otherwise
    """
    # 1. Query for matching pre-event
    db = WipnoteDB()
    cursor = db.connection.cursor()
    cursor.execute("""
        SELECT start_time FROM tool_traces
        WHERE tool_use_id = ?
    """, (tool_use_id,))

    row = cursor.fetchone()
    if not row:
        # Pre-event not found - log warning, gracefully degrade
        logger.warning(f"PreToolUse event not found for {tool_use_id}")
        return False

    start_time_iso = row[0]

    # 2. Calculate duration
    end_time_iso = datetime.now(timezone.utc).isoformat()
    duration_ms = calculate_duration(start_time_iso, end_time_iso)

    # 3. Update tool_traces
    cursor.execute("""
        UPDATE tool_traces
        SET end_time = ?, duration_ms = ?, tool_output = ?,
            status = ?, error_message = ?
        WHERE tool_use_id = ?
    """, (end_time_iso, duration_ms,
          json.dumps(tool_output), status, error_message, tool_use_id))

    db.connection.commit()
    return True
```

**Runs In Parallel With:**
- Event tracking
- Orchestrator reflection
- Task validation
- Error tracking
- Debugging suggestions
- CIGS analysis

**Error Handling:**
- Non-blocking - errors don't prevent execution continuation
- Missing pre-event - log warning, skip update
- Invalid timestamp - log warning, set duration_ms to None
- JSON serialization fails - store stringified output
- Database unavailable - log error, return False

**Related:** [EVENT_TRACING.md - PostToolUse Hook](./EVENT_TRACING.md#posttooluse-hook)

### PreToolUse + PostToolUse Pattern

The combination of PreToolUse and PostToolUse provides complete tool execution tracing:

```
┌─────────────────────────────────────────────────────────────────┐
│ PreToolUse Hook                                                 │
├─────────────────────────────────────────────────────────────────┤
│ Generates tool_use_id = "a1b2c3d4-..."                          │
│ Inserts into tool_traces:                                       │
│   - tool_use_id: "a1b2c3d4-..."                                 │
│   - tool_name: "Bash"                                           │
│   - tool_input: {"command": "ls"}                               │
│   - start_time: "2025-01-07T12:34:56.789000+00:00"             │
│   - status: "started"                                           │
│ Sets: HTMLGRAPH_TOOL_USE_ID="a1b2c3d4-..."                      │
└─────────────────────────────────────────────────────────────────┘
                              ↓
                    [Tool Executes]
                    [1.234 seconds]
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ PostToolUse Hook                                                │
├─────────────────────────────────────────────────────────────────┤
│ Reads: HTMLGRAPH_TOOL_USE_ID="a1b2c3d4-..."                    │
│ Queries: SELECT * FROM tool_traces                              │
│          WHERE tool_use_id = "a1b2c3d4-..."                    │
│ Calculates: duration_ms = 1234                                  │
│ Updates tool_traces:                                            │
│   - end_time: "2025-01-07T12:34:58.023000+00:00"               │
│   - duration_ms: 1234                                           │
│   - tool_output: {"stdout": "..."}                             │
│   - status: "Ok"                                                │
│   - error_message: null                                         │
└─────────────────────────────────────────────────────────────────┘
```

**Tool Trace Result:**

```python
trace = TraceRecord(
    tool_use_id="a1b2c3d4-...",
    trace_id="a1b2c3d4-...",
    session_id="sess-123",
    tool_name="Bash",
    tool_input={"command": "ls"},
    tool_output={"stdout": "file1\nfile2"},
    start_time=datetime(...),  # 2025-01-07T12:34:56.789000+00:00
    end_time=datetime(...),    # 2025-01-07T12:34:58.023000+00:00
    duration_ms=1234,
    status="Ok",
    error_message=None,
    parent_tool_use_id=None,
)
```

### Other Hooks

For complete reference on other hooks (SessionStart, Stop, UserPromptSubmit, etc.), see:
- Claude Code official documentation: https://code.claude.com/docs/en/hooks.md
- [SYSTEM_PROMPT_ARCHITECTURE.md](./SYSTEM_PROMPT_ARCHITECTURE.md)
- Hook implementations in `src/python/wipnote/hooks/`

## Querying Traces

Once events are recorded via PreToolUse and PostToolUse hooks, query using TraceCollection:

```python
from wipnote.sdk import SDK

sdk = SDK(agent="claude")

# Get single trace
trace = sdk.traces.get_trace("a1b2c3d4-...")

# Get all traces for session
all_traces = sdk.traces.get_traces("sess-123")

# Find slow tools
slow = sdk.traces.get_slow_traces(threshold_ms=1000)

# Get errors
errors = sdk.traces.get_error_traces("sess-123")

# Hierarchical view
tree = sdk.traces.get_trace_tree("trace-id")
```

See [EVENT_TRACING.md - Querying Traces](./EVENT_TRACING.md#querying-traces) for complete API reference.

## Hook Installation

Hooks are installed via `.claude/settings.json` or `plugin.json`.

Example hook configuration:

```json
{
  "hooks": [
    {
      "name": "pretooluse-event-tracing",
      "type": "PreToolUse",
      "description": "Event tracing - generates tool_use_id and starts trace",
      "script": "path/to/pretooluse-integrator.py"
    },
    {
      "name": "posttooluse-unified",
      "type": "PostToolUse",
      "description": "Unified PostToolUse - tracking, reflection, validation, errors, CIGS",
      "script": "path/to/posttooluse-integrator.py"
    }
  ]
}
```

## Hook Environment

Hooks have access to environment variables set by Claude Code and Wipnote:

| Variable | Set By | Used By | Value |
|----------|--------|---------|-------|
| `HTMLGRAPH_SESSION_ID` | SessionStart | PreToolUse, queries | Current session ID |
| `HTMLGRAPH_TOOL_USE_ID` | PreToolUse | PostToolUse | Current tool execution ID |
| `HTMLGRAPH_TRACE_ID` | PreToolUse | Tracing | Root trace ID for grouping |
| `HTMLGRAPH_DISABLE_TRACKING` | User (optional) | All hooks | If "1", skip tracking |

## Performance Considerations

### PreToolUse Performance

PreToolUse runs synchronously before tool execution:
- Target: < 50ms per execution
- Database insert: ~10ms
- UUID generation: < 1ms
- Input sanitization: ~5-10ms

Non-blocking if database is slow - logs warning and continues.

### PostToolUse Performance

PostToolUse runs asynchronously after tool completes:
- Target: < 50ms per execution
- Database query: ~5-10ms
- Duration calculation: < 1ms
- Database update: ~10-20ms
- All other tasks (reflection, validation, CIGS): ~20-30ms (parallel)

Non-blocking - multiple tasks run in parallel via asyncio.gather().

### Database Indexes

Tool traces table has 5 performance indexes:

```sql
CREATE INDEX idx_tool_traces_trace_id
  ON tool_traces(trace_id, start_time DESC);
CREATE INDEX idx_tool_traces_session
  ON tool_traces(session_id);
CREATE INDEX idx_tool_traces_tool_name
  ON tool_traces(tool_name);
CREATE INDEX idx_tool_traces_status
  ON tool_traces(status);
CREATE INDEX idx_tool_traces_start_time
  ON tool_traces(start_time DESC);
```

All queries use these indexes for sub-5ms performance.

## Error Handling Strategy

Hooks follow consistent error handling:

1. **Try-Catch-Log** - All operations wrapped in try-catch
2. **Graceful Degradation** - Errors logged but don't block execution
3. **Non-Blocking** - Hook never returns `continue=False` on internal errors
4. **User-Facing Errors** - Only return blocking response if orchestrator/validator fails

Example:

```python
try:
    # Create start event
    tool_use_id = create_start_event(tool_name, tool_input, session_id)
    if tool_use_id:
        os.environ["HTMLGRAPH_TOOL_USE_ID"] = tool_use_id
except Exception as e:
    # Log but don't block
    logger.warning(f"Error creating start event: {e}")
    # Continue anyway - return empty response
```

## Parallel Hook Execution

PostToolUse hook runs 6 tasks in parallel using asyncio:

```python
(
    event_response,
    reflection_response,
    validation_response,
    error_tracking_response,
    debug_suggestions,
    cigs_response,
) = await asyncio.gather(
    run_event_tracking(hook_type, hook_input),      # Event storage
    run_orchestrator_reflection(hook_input),        # Delegation guidance
    run_task_validation(hook_input),                # Work validation
    run_error_tracking(hook_input),                 # Error logging
    suggest_debugging_resources(hook_input),        # Debug tips
    run_cigs_analysis(hook_input),                  # Cost accounting
)
```

Benefits:
- **40-50% faster** - Parallel execution vs sequential
- **No blocking** - All tasks must return continue=True
- **Combined response** - Guidance from all tasks merged

## Related Documentation

- [EVENT_TRACING.md](./EVENT_TRACING.md) - Complete event tracing guide
- [SYSTEM_PROMPT_ARCHITECTURE.md](./SYSTEM_PROMPT_ARCHITECTURE.md) - Hook persistence and loading
- [Claude Code Hooks Documentation](https://code.claude.com/docs/en/hooks.md) - Official reference
- Hook implementations in `src/python/wipnote/hooks/`
