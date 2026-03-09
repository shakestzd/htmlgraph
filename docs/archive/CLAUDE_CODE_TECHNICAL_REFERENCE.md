# Claude Code Technical Reference for HtmlGraph Developers

**Quick Reference for Hook Development**
**Last Updated:** January 13, 2026

---

## Hook System Overview

### Hook Lifecycle

```
Claude Code Event
    ↓
Hook receives JSON input via stdin
    ↓
Hook executes Python code
    ↓
Hook outputs JSON response via stdout
    ↓
Claude Code processes response
    (inject context, block tool, modify input, etc.)
    ↓
Session continues or stops
```

### Hook Input/Output Contract

Every hook follows the same JSON protocol:

**INPUT** (from Claude Code):
```json
{
  "session_id": "sess-abc123",
  "hook_event_name": "SessionStart|PreToolUse|PostToolUse|...",
  "tool_name": "Read|Edit|Bash|...",  // Only in tool-related hooks
  "tool_input": {...},                 // Only in tool-related hooks
  "tool_response": {...},              // Only in PostToolUse
  "prompt": "user text",               // Only in UserPromptSubmit
  ...
}
```

**OUTPUT** (to Claude Code):
```json
{
  "continue": true,                    // Always required
  "hookSpecificOutput": {
    "hookEventName": "SessionStart",
    "additionalContext": "...",        // Injected to Claude
    "permissionDecision": "allow",     // Only in PreToolUse
    "updatedInput": {...}              // Only in PreToolUse
  },
  "systemMessage": "...",              // User-facing message
  "suppressOutput": false
}
```

---

## Hook Type Reference

### SessionStart Hook

**When:** Session begins (new, resumed, or compacted)
**Input:**
```json
{
  "session_id": "sess-abc123",
  "transcript_path": "/path/to/session.jsonl",
  "cwd": "/project/root",
  "permission_mode": "default",
  "hook_event_name": "SessionStart",
  "source": "startup|resume|clear|compact"
}
```

**Responsibilities:**
- Initialize session tracking
- Inject project context
- Provide guidance/recommendations
- Check for version updates

**Available Operations:**
```python
# Access project context
context = HookContext.from_input(hook_input)
project_dir = context.project_dir              # /project/root
graph_dir = context.graph_dir                  # /project/root/.htmlgraph
session_id = context.session_id                # sess-abc123
agent_id = context.agent_id                    # "claude"
model_name = context.model_name                # "claude-opus"

# Query database
db = context.get_database()
cursor = db.connection.cursor()
cursor.execute("""
    SELECT * FROM sessions WHERE session_id = ?
""", (session_id,))

# Load features
features_dir = graph_dir / "features"
from htmlgraph.graph import HtmlGraph
graph = HtmlGraph(features_dir, auto_load=True)

# Read transcript
with open(hook_input["transcript_path"]) as f:
    events = [json.loads(line) for line in f]
```

**Output:**
```python
{
    "continue": True,
    "hookSpecificOutput": {
        "hookEventName": "SessionStart",
        "additionalContext": "## Project Status\n..."  # Injected!
    },
    "systemMessage": "💡 Note: You have 3 high-priority tasks"
}
```

**Timeout:** 60 seconds

---

### UserPromptSubmit Hook

**When:** User submits a message
**Input:**
```json
{
  "prompt": "What the user asked",
  "session_id": "sess-abc123",
  "transcript_path": "/path/to/session.jsonl"
}
```

**Use Cases:**
- Analyze user intent
- Detect pattern (refactor? test? debug?)
- Provide targeted guidance
- Block problematic requests

**Available Operations:**
```python
# Analyze prompt
prompt = hook_input["prompt"]
intent = detect_intent(prompt)  # "feature|bugfix|refactor|test|debug"

# Get session context
context = HookContext.from_input(hook_input)
db = context.get_database()
session = get_session(db, session_id)

# Provide guidance
if intent == "refactor":
    message = "💡 Remember to write tests before refactoring"
```

**Output:**
```python
{
    "continue": True,
    "hookSpecificOutput": {
        "hookEventName": "UserPromptSubmit",
        "additionalContext": "# Guidance\nConsider writing tests first"
    }
}
```

**Timeout:** 30 seconds

---

### PreToolUse Hook

**When:** Before tool execution (can block or modify)
**Input:**
```json
{
  "session_id": "sess-abc123",
  "tool_name": "Read",
  "tool_input": {"path": "/file/path"},
  "tool_use_id": "tool-xyz789"
}
```

**Use Cases:**
- Validate tool input
- Detect anti-patterns
- Check for conflicts
- Provide suggestions
- Block if necessary

**Available Operations:**
```python
# Detect anti-patterns
recent_tools = get_recent_tools(context.session_id, limit=5)
if len([t for t in recent_tools if t == tool_name]) > 3:
    # Multiple consecutive same-tool calls
    guidance = "Multiple " + tool_name + " calls. Consider batching?"
    return {
        "continue": True,
        "systemMessage": guidance
    }

# Check for conflicts
if tool_name == "Edit":
    file_path = tool_input.get("path")
    if check_concurrent_edits(file_path, context.session_id):
        return {
            "permissionDecision": "block",
            "systemMessage": "Another agent editing this file"
        }

# Modify input if needed
updated_input = tool_input.copy()
updated_input["some_field"] = "modified_value"
return {
    "continue": True,
    "hookSpecificOutput": {
        "hookEventName": "PreToolUse",
        "updatedInput": updated_input
    }
}
```

**Output Options:**
```python
# Option 1: Allow with suggestion
{"continue": True, "systemMessage": "💡 Consider..."}

# Option 2: Block with reason
{"permissionDecision": "block", "systemMessage": "Not allowed because..."}

# Option 3: Modify input
{"hookSpecificOutput": {"updatedInput": {...}}}

# Option 4: Allow silently
{"continue": True}
```

**Timeout:** 5 seconds (fast!)

---

### PostToolUse Hook

**When:** After tool execution (success or failure)
**Input:**
```json
{
  "session_id": "sess-abc123",
  "tool_name": "Bash",
  "tool_input": {"command": "..."},
  "tool_response": {"stdout": "...", "stderr": "...", "returncode": 0},
  "tool_use_id": "tool-xyz789"
}
```

**Use Cases:**
- Record tool execution
- Detect errors and suggest recovery
- Provide feedback
- Track costs
- Update state

**Available Operations:**
```python
# Record event
db = context.get_database()
db.log_event(
    session_id=context.session_id,
    agent_id=context.agent_id,
    event_type="tool_result",
    tool_name=tool_name,
    tool_input=tool_input,
    tool_response=tool_response,
    status="success" if tool_response["returncode"] == 0 else "error"
)

# Analyze errors
if tool_response.get("returncode") != 0:
    error = tool_response.get("stderr", "")
    error_type = categorize_error(error)
    if error_type == "test_failure":
        suggestion = "Run single failing test to isolate issue"
        return {
            "continue": True,
            "systemMessage": f"💡 {suggestion}"
        }

# Track cost
token_estimate = estimate_tokens(tool_input, tool_response)
db.update_session_cost(session_id, token_estimate)
```

**Output:**
```python
{
    "continue": True,
    "hookSpecificOutput": {
        "hookEventName": "PostToolUse",
        "additionalContext": "💡 Test failed. Try running single test first."
    }
}
```

**Timeout:** 10 seconds

---

### SubagentStop Hook

**When:** Subagent (Task delegation) completes
**Input:**
```json
{
  "session_id": "parent-sess-id",
  "subagent_session_id": "child-sess-id",
  "subagent_type": "general-purpose",
  "subagent_status": "completed|failed|timeout",
  "subagent_output": "Work completed. Results..."
}
```

**Use Cases:**
- Link parent-child sessions
- Track delegation outcome
- Update parent state
- Detect failures

**Available Operations:**
```python
# Record delegation outcome
db = context.get_database()
parent_event = db.log_event(
    session_id=hook_input["session_id"],
    event_type="task_completion",
    subagent_type=hook_input["subagent_type"],
    status=hook_input["subagent_status"]
)

# Link child session to parent
db.link_sessions(
    parent_session_id=hook_input["session_id"],
    child_session_id=hook_input["subagent_session_id"],
    parent_event_id=parent_event["event_id"]
)

# Analyze delegation success
if hook_input["subagent_status"] == "failed":
    output = hook_input.get("subagent_output", "")
    context.logger.error(f"Subagent failed: {output}")
```

**Output:**
```python
{
    "continue": True,
    "hookSpecificOutput": {
        "hookEventName": "SubagentStop"
    }
}
```

**Timeout:** 30 seconds

---

### SessionEnd Hook

**When:** Session ends (user stops or completes)
**Input:**
```json
{
  "session_id": "sess-abc123"
}
```

**Use Cases:**
- Archive session
- Export transcript analytics
- Save handoff notes
- Cleanup

**Available Operations:**
```python
# Save handoff context
context = HookContext.from_input(hook_input)
summary = generate_session_summary(context.session_id)
db = context.get_database()
db.update_session(
    session_id=context.session_id,
    status="completed",
    summary=summary
)

# Export transcript analytics
transcript_path = "..."  # Must be passed via environment
metrics = extract_transcript_metrics(transcript_path)
db.store_session_analytics(context.session_id, metrics)

# Cleanup
# Close database connections, clear caches, etc.
```

**Output:**
```python
{
    "continue": True,
    "hookSpecificOutput": {
        "hookEventName": "SessionEnd"
    }
}
```

**Timeout:** 60 seconds

---

## HookContext API Reference

### Initialization

```python
from htmlgraph.hooks.context import HookContext

# From hook input
context = HookContext.from_input(hook_input)

# Or manual
context = HookContext(
    project_dir="/project/root",
    graph_dir=Path("/project/root/.htmlgraph"),
    session_id="sess-abc123",
    agent_id="claude",
    hook_input=hook_input,
    model_name="claude-opus"
)
```

### Properties & Methods

```python
# Read-only properties
context.project_dir: str                    # Project root
context.graph_dir: Path                     # .htmlgraph directory
context.session_id: str                     # Current session
context.agent_id: str                       # Agent name
context.hook_input: dict                    # Raw hook input
context.model_name: str | None              # Model being used

# Lazy-loaded resources
db = context.get_database()                 # HtmlGraphDB instance
session = context.get_session()             # Current session
logger = context.logger                     # Logging (to stderr)
```

---

## Database Query Patterns

### Query Recent Tool Calls

```python
db = context.get_database()
cursor = db.connection.cursor()

# Last N tools in session
cursor.execute("""
    SELECT tool_name, timestamp, status
    FROM agent_events
    WHERE session_id = ?
    ORDER BY timestamp DESC
    LIMIT 5
""", (context.session_id,))

tools = [(row[0], row[1], row[2]) for row in cursor.fetchall()]
```

### Query Error History

```python
# Find similar errors in past sessions
cursor.execute("""
    SELECT DISTINCT event_id, tool_response, session_id
    FROM agent_events
    WHERE event_type = 'error'
      AND tool_response LIKE ?
    ORDER BY timestamp DESC
    LIMIT 10
""", ('%' + error_keyword + '%',))

errors = cursor.fetchall()
```

### Query Active Sessions

```python
# Find other active sessions
cursor.execute("""
    SELECT session_id, agent_id, created_at
    FROM sessions
    WHERE status = 'active'
      AND session_id != ?
    ORDER BY created_at DESC
""", (context.session_id,))

other_sessions = cursor.fetchall()
```

### Query Session Metrics

```python
# Get session statistics
cursor.execute("""
    SELECT
        COUNT(*) as tool_calls,
        COUNT(CASE WHEN event_type = 'error' THEN 1 END) as errors,
        SUM(cost_tokens) as total_tokens,
        AVG(execution_duration_seconds) as avg_duration
    FROM agent_events
    WHERE session_id = ?
""", (context.session_id,))

metrics = cursor.fetchone()
```

---

## Hook Development Patterns

### Pattern 1: Simple Suggestion Hook

```python
def main():
    hook_input = json.load(sys.stdin)

    try:
        context = HookContext.from_input(hook_input)

        # Detect condition
        if should_provide_suggestion(context):
            suggestion = generate_suggestion(context)
            return {
                "continue": True,
                "systemMessage": suggestion
            }
        else:
            return {"continue": True}
    except Exception as e:
        logger.error(f"Hook failed: {e}")
        return {"continue": True}  # Never block on error
```

### Pattern 2: Error Handling Hook

```python
def categorize_error(error_message: str) -> str:
    """Categorize error type."""
    if "test failed" in error_message.lower():
        return "test_failure"
    elif "SyntaxError" in error_message:
        return "syntax_error"
    elif "No such file" in error_message:
        return "file_not_found"
    else:
        return "unknown"

def suggest_recovery(error_type: str) -> str:
    """Suggest recovery approach."""
    suggestions = {
        "test_failure": "Run single failing test: pytest -k test_name",
        "syntax_error": "Check indentation and import statements",
        "file_not_found": "Verify file path and check .gitignore"
    }
    return suggestions.get(error_type, "Review error message carefully")
```

### Pattern 3: Database Logging Hook

```python
def log_tool_execution(context: HookContext, event: dict):
    """Log tool execution to database."""
    db = context.get_database()

    event_id = generate_id()
    cursor = db.connection.cursor()

    cursor.execute("""
        INSERT INTO agent_events (
            event_id, session_id, agent_id, event_type,
            tool_name, tool_input, status, created_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
    """, (
        event_id,
        context.session_id,
        context.agent_id,
        event.get("event_type"),
        event.get("tool_name"),
        json.dumps(event.get("tool_input")),
        event.get("status"),
        datetime.now().isoformat()
    ))

    db.connection.commit()
```

### Pattern 4: Feature Analysis Hook

```python
def get_current_feature(context: HookContext) -> dict | None:
    """Load current feature being worked on."""
    features_dir = context.graph_dir / "features"
    if not features_dir.exists():
        return None

    from htmlgraph.graph import HtmlGraph
    graph = HtmlGraph(features_dir, auto_load=True)

    # Get most recent feature (by creation time)
    features = graph.nodes.values()
    return max(features, key=lambda f: f.get("created_at", ""))
```

---

## Debugging Hook Execution

### Enable Detailed Logging

```python
import logging
import sys

# Configure logging to stderr
logging.basicConfig(
    level=logging.DEBUG,
    format='%(name)s - %(levelname)s - %(message)s',
    stream=sys.stderr
)

logger = logging.getLogger(__name__)
logger.debug("Hook starting...")
```

### Log Hook Input/Output

```python
import json
import sys

hook_input = json.load(sys.stdin)
print(f"DEBUG: Hook input = {json.dumps(hook_input, indent=2)}", file=sys.stderr)

output = generate_response(hook_input)
print(f"DEBUG: Hook output = {json.dumps(output, indent=2)}", file=sys.stderr)

print(json.dumps(output))
```

### Test Hook Locally

```bash
# Create test input
cat > test_hook_input.json << 'EOF'
{
  "session_id": "test-sess-123",
  "hook_event_name": "PreToolUse",
  "tool_name": "Read",
  "tool_input": {"path": "/test.txt"}
}
EOF

# Run hook with test input
cat test_hook_input.json | python hook-script.py
```

---

## Performance Tips

### 1. Minimize Database Queries

```python
# ❌ BAD: Query database in loop
for tool in tools:
    cursor.execute("SELECT * FROM agent_events WHERE tool_name = ?", (tool,))

# ✅ GOOD: Single query
cursor.execute("SELECT tool_name FROM agent_events WHERE tool_name IN (?, ?, ?)", tools)
```

### 2. Cache Expensive Computations

```python
# ❌ BAD: Recompute every time
def detect_pattern(tools):
    # Complex analysis...
    return pattern

# ✅ GOOD: Cache in memory
_pattern_cache = {}
def detect_pattern(tools):
    key = tuple(tools)
    if key in _pattern_cache:
        return _pattern_cache[key]
    pattern = analyze(tools)
    _pattern_cache[key] = pattern
    return pattern
```

### 3. Exit Early

```python
# ❌ BAD: Process everything
def hook():
    context = HookContext.from_input(hook_input)
    db = context.get_database()
    # ... lots of operations ...

# ✅ GOOD: Exit if not needed
def hook():
    if not should_process(hook_input):
        return {"continue": True}  # Fast exit

    context = HookContext.from_input(hook_input)
    # ... only process if needed ...
```

### 4. Use Indexes for Queries

```sql
-- Database queries use these indexes:
-- - idx_agent_events_session (session_id)
-- - idx_agent_events_tool (tool_name)
-- - idx_agent_events_type (event_type)

-- Query with indexed columns is fast:
SELECT * FROM agent_events
WHERE session_id = ? AND tool_name = ?  -- Fast!

-- Query without indexed columns is slow:
SELECT * FROM agent_events
WHERE error_message LIKE '%something%'  -- Slow!
```

---

## Troubleshooting

### Hook Not Executing

**Symptom:** Hook script runs but doesn't seem to execute
**Solution:** Check hooks.json is syntactically valid
```bash
python -m json.tool .claude-plugin/hooks/hooks.json
```

### Hook Timeout

**Symptom:** "Hook timed out after 60 seconds"
**Solution:** Optimize database queries or reduce scope
```python
# Add LIMIT to prevent huge result sets
cursor.execute("SELECT * FROM agent_events LIMIT 100")

# Use fast queries only
# Avoid SELECT * (use specific columns)
# Use WHERE with indexed columns
```

### Hook Output Not Appearing

**Symptom:** systemMessage not shown to user
**Solution:** Check JSON format is valid
```python
import json
output = {"continue": True, "systemMessage": "test"}
print(json.dumps(output))  # Must be valid JSON!
```

### Database Lock

**Symptom:** "database is locked" error
**Solution:** Use timeout and close connections
```python
db = sqlite3.connect(str(db_path), timeout=5.0)  # Add timeout
# ...use database...
db.close()  # Always close!
```

---

## Reference: Hook Configuration (hooks.json)

```json
{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "uv run --with htmlgraph>=0.26.5 \"${CLAUDE_PLUGIN_ROOT}/hooks/scripts/session-start.py\"",
            "timeout": 60
          }
        ]
      }
    ],
    "PreToolUse": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "uv run --with htmlgraph>=0.26.5 \"${CLAUDE_PLUGIN_ROOT}/hooks/scripts/pretooluse-integrator.py\"",
            "timeout": 5
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "uv run --with htmlgraph>=0.26.5 \"${CLAUDE_PLUGIN_ROOT}/hooks/scripts/posttooluse-integrator.py\"",
            "timeout": 10
          }
        ]
      }
    ]
  }
}
```

---

## Next Steps

1. **Review existing hooks** in `/packages/claude-plugin/.claude-plugin/hooks/scripts/`
2. **Start with simple hook** (pattern recognition, error recovery)
3. **Test locally** with test input
4. **Deploy and monitor** in real sessions
5. **Iterate based on feedback**

For detailed implementation, see:
- `CLAUDE_CODE_INTEGRATION_ANALYSIS.md` - Full analysis
- `HTMLGRAPH_CLAUDE_CODE_OPPORTUNITIES.md` - Prioritized opportunities
- Existing hook implementations for patterns
