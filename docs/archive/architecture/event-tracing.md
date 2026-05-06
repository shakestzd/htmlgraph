# Event Tracing System

Complete guide to Wipnote's event tracing system for tracking tool executions with millisecond precision.

## Overview

Event tracing captures and correlates every tool execution in Claude Code via:
- **PreToolUse Hook** - Records tool START event when tool is invoked
- **PostToolUse Hook** - Records tool END event when tool completes
- **tool_traces Table** - SQLite database storing correlated traces
- **TraceCollection API** - Query interface for analysis and debugging

### Why Track Tool Executions?

Tool tracing enables:
- **Performance Analysis** - Identify slow tools and bottlenecks
- **Error Debugging** - Track which tools failed and why
- **Execution Hierarchies** - Understand nested tool invocations
- **Cost Optimization** - Analyze token usage by tool
- **Reliability Metrics** - Track success rates and error patterns

## Architecture

### System Design

```
Claude Code Hook System
├── PreToolUse Hook (fires before tool execution)
│   ├── Generates tool_use_id (UUID v4)
│   ├── Captures tool_name, tool_input, start_time
│   ├── Stores start event in tool_traces table
│   └── Sets HTMLGRAPH_TOOL_USE_ID environment variable
│
└── PostToolUse Hook (fires after tool execution)
    ├── Reads tool_use_id from environment
    ├── Queries tool_traces for matching pre-event
    ├── Calculates duration (end_time - start_time)
    ├── Stores tool_output, status, error_message
    └── Updates tool_traces with completion data
```

### Correlation Mechanism

Tool execution traces are correlated via **tool_use_id**:

1. **PreToolUse Hook generates UUID v4** → `tool_use_id = "a1b2c3d4-..."`
2. **Stores in environment** → `HTMLGRAPH_TOOL_USE_ID="a1b2c3d4-..."`
3. **PostToolUse Hook reads from environment** → finds matching trace
4. **Updates with completion data** → duration, output, status

Example flow:

```
PreToolUse Hook
├── Generates: tool_use_id = "a1b2c3d4-e5f6-47a8-9b0c-1d2e3f4a5b6c"
├── Inserts into tool_traces:
│   - tool_use_id: "a1b2c3d4-..."
│   - tool_name: "Bash"
│   - tool_input: {"command": "ls -la"}
│   - start_time: "2025-01-07T12:34:56.789000+00:00"
│   - status: "started"
└── Sets: HTMLGRAPH_TOOL_USE_ID="a1b2c3d4-..."

[Tool executes for 1.234 seconds...]

PostToolUse Hook
├── Reads: tool_use_id = "a1b2c3d4-..." from environment
├── Queries: SELECT * FROM tool_traces WHERE tool_use_id = "a1b2c3d4-..."
├── Calculates: duration_ms = 1234
└── Updates tool_traces:
    - end_time: "2025-01-07T12:34:58.023000+00:00"
    - duration_ms: 1234
    - tool_output: {"stdout": "..."}
    - status: "Ok"
    - error_message: null
```

### Duration Calculation

Duration is calculated as:

```python
duration_ms = int((end_time - start_time).total_seconds() * 1000)
```

- Both times in ISO8601 UTC format
- Accuracy within 1 millisecond
- Handles timezone-aware datetimes automatically

Example:

```python
start = "2025-01-07T12:34:56.789000+00:00"
end = "2025-01-07T12:34:58.023000+00:00"
# duration = 1234 milliseconds
```

## PreToolUse Hook

### When It Fires

PreToolUse hook fires **before** Claude Code executes any tool call.

Timing:
- Tool call in progress: User invokes a tool
- Hook system intercepts: Calls registered PreToolUse hooks
- User's tool runs: Hook completes before tool executes

### What It Captures

PreToolUse captures and stores:

| Field | Type | Example | Purpose |
|-------|------|---------|---------|
| `tool_use_id` | UUID v4 | `a1b2c3d4-...` | Unique identifier for this execution (for correlation) |
| `trace_id` | UUID v4 | `trace-123-...` | Parent trace ID (groups related executions) |
| `session_id` | string | `sess-abc123` | Which session this execution belongs to |
| `tool_name` | string | `"Bash"` | Tool being invoked |
| `tool_input` | JSON | `{"command": "ls"}` | Input parameters (sanitized) |
| `start_time` | ISO8601 | `2025-01-07T...` | When execution started (UTC) |
| `status` | string | `"started"` | Current status (always "started" at this point) |

### Data Schema

```sql
CREATE TABLE tool_traces (
    tool_use_id TEXT PRIMARY KEY,
    trace_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    tool_name TEXT NOT NULL,
    tool_input JSON,
    tool_output JSON,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP,
    duration_ms INTEGER,
    status TEXT NOT NULL DEFAULT 'started',
    error_message TEXT,
    parent_tool_use_id TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Input Sanitization

Sensitive data is removed before storage:

```python
Removed fields: password, token, secret, key, auth, api_key
Truncated: Values > 10,000 characters
Example:
  Input:  {"command": "curl -X POST", "api_key": "secret123"}
  Stored: {"command": "curl -X POST", "api_key": "[REDACTED]"}
```

### Error Handling

| Scenario | Behavior |
|----------|----------|
| No session ID | Skip event tracing (return empty response) |
| Database unavailable | Log warning, continue (non-blocking) |
| Invalid input | Sanitize gracefully, store partial data |
| Environment variable not set | Generate new tool_use_id, store in environment |

## PostToolUse Hook

### When It Fires

PostToolUse hook fires **after** tool execution completes (success or failure).

Timing:
- Tool execution completes: stdout/stderr captured
- Hook system intercepts: Calls registered PostToolUse hooks
- User's code resumes: Hook completes before execution returns

### What It Updates

PostToolUse updates the existing trace with:

| Field | Type | Example | Purpose |
|-------|------|---------|---------|
| `end_time` | ISO8601 | `2025-01-07T12:34:58...` | When execution completed (UTC) |
| `duration_ms` | integer | `1234` | Milliseconds elapsed (end - start) |
| `tool_output` | JSON | `{"stdout": "..."}` | Result of tool execution |
| `status` | string | `"Ok"` or `"Error"` | Success/failure indicator |
| `error_message` | string | `"Connection timeout"` | Error details if failed |

### Duration Calculation

Duration is calculated in milliseconds:

```python
def calculate_duration(start_time_iso: str, end_time_iso: str) -> int:
    """Calculate duration in milliseconds between two ISO8601 UTC timestamps."""
    start_dt = datetime.fromisoformat(start_time_iso.replace("Z", "+00:00"))
    end_dt = datetime.fromisoformat(end_time_iso.replace("Z", "+00:00"))
    delta = end_dt - start_dt
    return int(delta.total_seconds() * 1000)
```

Accuracy: Within 1 millisecond

### Status Determination

Status is determined from tool response:

| Indicator | Status | Reason |
|-----------|--------|--------|
| `stderr` non-empty | `Error` | Command output errors |
| `error` field present | `Error` | Explicit error field |
| `success: false` | `Error` | Success flag set to false |
| `status: "error"` | `Error` | Status field indicates error |
| Default | `Ok` | No error indicators found |

### Error Message Storage

Error messages are captured from:
1. **stderr** - Command standard error
2. **error** field - Explicit error from tool
3. **reason** field - Explanation of failure
4. **message** field - Message from error response

Limits: Truncated to 500 characters

### Error Handling

| Scenario | Behavior |
|----------|----------|
| Missing pre-event | Log warning, skip duration update, continue |
| Invalid timestamp | Log warning, set duration_ms to None, continue |
| JSON serialization fails | Log error, store stringified output, continue |
| Database update fails | Log error, return False (non-blocking) |
| Invalid status | Replace with "Ok", log warning, continue |

All errors are **non-blocking** - execution continues regardless.

## Querying Traces

### TraceCollection API

```python
from wipnote.sdk import SDK

sdk = SDK(agent="claude")
traces = sdk.traces  # TraceCollection instance
```

### Query Methods

#### Get Single Trace

```python
trace = sdk.traces.get_trace(tool_use_id="a1b2c3d4-...")
if trace:
    print(f"{trace.tool_name}: {trace.duration_ms}ms")
    print(f"Status: {trace.status}")
    if trace.error_message:
        print(f"Error: {trace.error_message}")
```

Returns: `Optional[TraceRecord]`

#### Get Traces for Session

```python
traces = sdk.traces.get_traces(
    session_id="sess-abc123",
    limit=100,  # Maximum traces to return (default 100)
    start_time=datetime.now(timezone.utc) - timedelta(hours=1)  # Optional filter
)

for trace in traces:
    print(f"{trace.tool_name}: {trace.duration_ms}ms")
```

Returns: `list[TraceRecord]` ordered by start_time DESC (newest first)

#### Get Traces by Tool Name

```python
bash_traces = sdk.traces.get_traces_by_tool(
    tool_name="Bash",
    limit=50
)

for trace in bash_traces:
    print(f"Command: {trace.tool_input}")
    print(f"Duration: {trace.duration_ms}ms")
```

Returns: `list[TraceRecord]` ordered by start_time DESC (newest first)

#### Get Hierarchical Trace Tree

```python
tree = sdk.traces.get_trace_tree(trace_id="trace-xyz")
if tree:
    print(f"Root tool: {tree.root.tool_name}")
    print(f"Children: {len(tree.children)}")
    for child_tree in tree.children:
        print(f"  - {child_tree.root.tool_name}")
```

Returns: `Optional[TraceTree]` with parent-child relationships

#### Find Slow Tool Calls

```python
slow = sdk.traces.get_slow_traces(
    threshold_ms=1000,  # Tools taking > 1 second
    limit=20
)

for trace in slow:
    print(f"{trace.tool_name}: {trace.duration_ms}ms")
```

Returns: `list[TraceRecord]` ordered by duration_ms DESC (slowest first)

#### Get Error Traces

```python
errors = sdk.traces.get_error_traces(
    session_id="sess-abc123",
    limit=50
)

for trace in errors:
    print(f"{trace.tool_name} failed: {trace.error_message}")
    print(f"Status: {trace.status}")
```

Returns: `list[TraceRecord]` ordered by start_time DESC (newest first)

### TraceRecord Fields

```python
@dataclass
class TraceRecord:
    tool_use_id: str              # Unique execution ID
    trace_id: str                 # Parent trace ID
    session_id: str               # Session this belongs to
    tool_name: str                # Tool name (e.g., "Bash")
    tool_input: Optional[dict]    # Input parameters
    tool_output: Optional[dict]   # Result/output
    start_time: datetime          # When it started (UTC)
    end_time: Optional[datetime]  # When it ended (UTC)
    duration_ms: Optional[int]    # Milliseconds elapsed
    status: Optional[str]         # "started", "completed", "failed", etc.
    error_message: Optional[str]  # Error details if failed
    parent_tool_use_id: Optional[str]  # Parent tool if nested
```

## Common Patterns

### Find Performance Bottlenecks

```python
# Get slowest tools in last hour
from datetime import datetime, timedelta, timezone

one_hour_ago = datetime.now(timezone.utc) - timedelta(hours=1)
slow_tools = sdk.traces.get_traces(
    session_id="sess-123",
    start_time=one_hour_ago,
    limit=100
)

# Group by tool name
from collections import Counter
tool_times = Counter()
for trace in slow_tools:
    tool_times[trace.tool_name] += trace.duration_ms or 0

# Show slowest tools
for tool, total_ms in tool_times.most_common(5):
    print(f"{tool}: {total_ms}ms total")
```

### Debug Tool Failures

```python
# Get all errors in session
errors = sdk.traces.get_error_traces(session_id="sess-123")

# Analyze failure patterns
failure_by_tool = {}
for trace in errors:
    if trace.tool_name not in failure_by_tool:
        failure_by_tool[trace.tool_name] = []
    failure_by_tool[trace.tool_name].append({
        "error": trace.error_message,
        "status": trace.status,
        "time": trace.start_time,
    })

# Print failure analysis
for tool, failures in failure_by_tool.items():
    print(f"\n{tool}: {len(failures)} failures")
    for failure in failures[:3]:  # Show first 3
        print(f"  - {failure['error']}")
```

### Trace Nested Executions

```python
# Find parent-child relationships
tree = sdk.traces.get_trace_tree(trace_id="trace-123")
if tree:
    def print_tree(node, indent=0):
        prefix = "  " * indent
        print(f"{prefix}{node.root.tool_name}: {node.root.duration_ms}ms")
        for child in node.children:
            print_tree(child, indent + 1)

    print_tree(tree)
```

Example output:
```
Bash: 2500ms
  Read: 150ms
  Read: 200ms
  Bash: 800ms
    Write: 100ms
```

### Track Tool Usage Statistics

```python
# Get all traces for session
all_traces = sdk.traces.get_traces(session_id="sess-123", limit=1000)

# Calculate statistics
stats = {}
for trace in all_traces:
    if trace.tool_name not in stats:
        stats[trace.tool_name] = {
            "count": 0,
            "total_ms": 0,
            "errors": 0,
            "min_ms": float('inf'),
            "max_ms": 0,
        }

    s = stats[trace.tool_name]
    s["count"] += 1
    s["total_ms"] += trace.duration_ms or 0
    if trace.status == "failed":
        s["errors"] += 1
    if trace.duration_ms:
        s["min_ms"] = min(s["min_ms"], trace.duration_ms)
        s["max_ms"] = max(s["max_ms"], trace.duration_ms)

# Print statistics
for tool, s in sorted(stats.items()):
    avg = s["total_ms"] // s["count"] if s["count"] > 0 else 0
    print(f"{tool}:")
    print(f"  Count: {s['count']}, Errors: {s['errors']}")
    print(f"  Total: {s['total_ms']}ms, Avg: {avg}ms")
    print(f"  Range: {s['min_ms']}-{s['max_ms']}ms")
```

## Examples

### Example 1: Analyze Recent Session

```python
from datetime import datetime, timedelta, timezone

# Get last 50 traces from current session
recent_traces = sdk.traces.get_traces(
    session_id="sess-current",
    limit=50
)

print(f"Found {len(recent_traces)} traces")
for trace in recent_traces:
    status = "✓" if trace.status == "completed" else "✗"
    print(f"{status} {trace.tool_name:15} {trace.duration_ms:6}ms")
```

### Example 2: Find Repeated Failures

```python
# Get all errors
errors = sdk.traces.get_error_traces("sess-123", limit=100)

# Group by error message
error_patterns = {}
for trace in errors:
    error_key = (trace.tool_name, trace.error_message[:50])
    if error_key not in error_patterns:
        error_patterns[error_key] = 0
    error_patterns[error_key] += 1

# Show repeated failures
for (tool, error), count in sorted(
    error_patterns.items(),
    key=lambda x: x[1],
    reverse=True
):
    if count > 1:
        print(f"{count}x {tool}: {error}")
```

### Example 3: Monitor Tool Performance

```python
# Track performance over time
import time

while True:
    # Get last 10 traces
    recent = sdk.traces.get_traces("sess-123", limit=10)

    if recent:
        avg_time = sum(t.duration_ms or 0 for t in recent) / len(recent)
        error_count = sum(1 for t in recent if t.status == "failed")
        print(f"Avg: {avg_time:.0f}ms, Errors: {error_count}/10")

    time.sleep(5)
```

## Troubleshooting

### Missing Traces

**Problem:** Expected traces not appearing in database

**Causes & Solutions:**
1. **Session ID not set** - `HTMLGRAPH_SESSION_ID` environment variable not set
   - Solution: Ensure SessionStart hook is running

2. **PreToolUse hook not running** - Hook not registered or disabled
   - Solution: Check `.claude/hooks/hooks.json` for PreToolUse hook

3. **Tool execution too fast** - Tool completed before PreToolUse saved
   - Solution: Rare, but check logs for timing issues

**Debug Steps:**
```python
import os

# Check environment
print(f"Session ID: {os.environ.get('HTMLGRAPH_SESSION_ID')}")
print(f"Tool Use ID: {os.environ.get('HTMLGRAPH_TOOL_USE_ID')}")

# Check database directly
from wipnote.db.schema import WipnoteDB
db = WipnoteDB()
cursor = db.connection.cursor()
cursor.execute("SELECT COUNT(*) FROM tool_traces")
count = cursor.fetchone()[0]
print(f"Total traces in DB: {count}")
```

### Incorrect Duration

**Problem:** Duration seems wrong or missing

**Causes & Solutions:**
1. **Tool still running** - `end_time` and `duration_ms` are null while executing
   - Solution: Normal - wait for tool to complete

2. **Clock skew** - Start and end times from different systems
   - Solution: Ensure all systems have NTP sync

3. **Invalid timestamp format** - Timestamp parsing failed
   - Solution: Check logs for "Error calculating duration"

**Debug Steps:**
```python
trace = sdk.traces.get_trace("tool-use-id")
if trace:
    print(f"Start: {trace.start_time}")
    print(f"End: {trace.end_time}")
    print(f"Duration: {trace.duration_ms}ms")

    if trace.end_time and not trace.duration_ms:
        # Duration calculation failed
        print("Duration calculation error - check logs")
```

### PreToolUse/PostToolUse Mismatch

**Problem:** Start event exists but no end event

**Causes & Solutions:**
1. **Tool still executing** - Normal if tool is slow
   - Solution: Wait longer, check with get_trace()

2. **PostToolUse hook failed** - Hook error, not blocking
   - Solution: Check logs for errors

3. **Database connection lost** - PostToolUse couldn't update
   - Solution: Check database availability, retry

**Debug Steps:**
```python
# Find traces without end_time
from wipnote.db.schema import WipnoteDB
db = WipnoteDB()
cursor = db.connection.cursor()
cursor.execute("""
    SELECT tool_use_id, tool_name, status, start_time
    FROM tool_traces
    WHERE end_time IS NULL
    ORDER BY start_time DESC
    LIMIT 10
""")
for row in cursor.fetchall():
    print(f"{row[0]}: {row[1]} ({row[2]}) at {row[3]}")
```

### Database Issues

**Problem:** Database queries timing out or failing

**Causes & Solutions:**
1. **Database locked** - Another process holding write lock
   - Solution: Check for hung processes, restart if needed

2. **Disk full** - SQLite can't write
   - Solution: Check disk space, clean up old data

3. **Corrupted database** - Database file corrupted
   - Solution: Close all connections, run PRAGMA integrity_check

**Debug Steps:**
```bash
# Check database integrity
sqlite3 ~/.wipnote/wipnote.db "PRAGMA integrity_check"

# Check disk space
df -h ~/.wipnote/

# Check database size
du -sh ~/.wipnote/wipnote.db
```

## Performance Tips

### Query Performance

- **Use session_id filter** - Most selective, use always
- **Use tool_name filter** - Good for tool-specific analysis
- **Limit result size** - Default 100, increase carefully
- **Start with recent data** - Use start_time parameter

### Large Datasets

For sessions with 1000+ traces:

```python
# Instead of fetching all at once
slow = sdk.traces.get_slow_traces(threshold_ms=1000, limit=100)

# Or paginate manually
for i in range(0, 1000, 100):
    batch = sdk.traces.get_traces("sess-123", limit=100)
    # Process batch
```

### Database Maintenance

Periodically archive old traces:

```python
# Archive traces older than 30 days
import sqlite3
from datetime import datetime, timedelta

db = WipnoteDB()
thirty_days_ago = (datetime.now() - timedelta(days=30)).isoformat()

cursor = db.connection.cursor()
cursor.execute("""
    DELETE FROM tool_traces
    WHERE start_time < ?
""", (thirty_days_ago,))
db.connection.commit()
```

## Related Documentation

- [HOOK_ARCHITECTURE.md](./HOOK_ARCHITECTURE.md) - Hook system overview
- [System Prompt Architecture](./SYSTEM_PROMPT_ARCHITECTURE.md) - Hook persistence
- [Claude Code Documentation](https://code.claude.com/docs) - Official Claude Code docs
