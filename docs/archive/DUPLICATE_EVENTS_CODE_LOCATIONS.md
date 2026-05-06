# Duplicate Events - Code Location Reference

## Hook Configuration Files

### 1. hooks.json - Main Hook Registry

**File**: `/Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/.claude-plugin/hooks/hooks.json`

**UserPromptSubmit Hook Chain (Lines 3-18)**:
```json
"UserPromptSubmit": [
  {
    "matcher": "",
    "hooks": [
      {
        "type": "command",
        "comment": "Record UserQuery event to SQLite",
        "command": "HTMLGRAPH_HOOK_TYPE=UserPromptSubmit uv run \"${CLAUDE_PLUGIN_ROOT}/hooks/scripts/track-event.py\""
      },
      {
        "type": "command",
        "comment": "CIGS analysis and workflow guidance",
        "command": "uv run \"${CLAUDE_PLUGIN_ROOT}/hooks/scripts/user-prompt-submit.py\""
      }
    ]
  }
]
```
**Issue**: Both hooks execute, both create UserQuery events → **DUPLICATES**

**PostToolUse Hook Chain (Lines 70-85)**:
```json
"PostToolUse": [
  {
    "matcher": "",
    "hooks": [
      {
        "type": "command",
        "comment": "Record tool execution to SQLite",
        "command": "HTMLGRAPH_HOOK_TYPE=PostToolUse uv run \"${CLAUDE_PLUGIN_ROOT}/hooks/scripts/track-event.py\""
      },
      {
        "type": "command",
        "command": "uv run \"${CLAUDE_PLUGIN_ROOT}/hooks/scripts/posttooluse-integrator.py\""
      }
    ]
  }
]
```
**Issue**: track-event.py + posttooluse-integrator both call track_event() → **DUPLICATES**

**PreToolUse Hook Chain (Lines 53-68)**:
```json
"PreToolUse": [
  {
    "matcher": "",
    "hooks": [
      {
        "type": "command",
        "comment": "Route Task() calls to spawner agents (gemini, codex, copilot)",
        "command": "uv run \"${CLAUDE_PLUGIN_ROOT}/hooks/scripts/pretooluse-integrator.py\""
      },
      {
        "type": "command",
        "comment": "Route Task() calls to spawner agents (gemini, codex, copilot)",
        "command": "uv run \"${CLAUDE_PLUGIN_ROOT}/hooks/scripts/pretooluse-spawner-router.py\""
      }
    ]
  }
]
```
**Issue**: Both handle Task() routing independently → **DUPLICATE ROUTING**

---

## Hook Script Files

### 2. track-event.py - Event Tracking Wrapper

**File**: `/Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/.claude-plugin/hooks/scripts/track-event.py`

**Key Lines**:
- **Line 10**: Hook definition in hooks.json references this script
- **Line 93**: Gets hook type from environment: `hook_type = os.environ.get("HTMLGRAPH_HOOK_TYPE", "PostToolUse")`
- **Line 103**: Calls event tracker: `response = track_event(hook_type, hook_input)`

**What it does**:
```python
def main() -> None:
    hook_type = os.environ.get("HTMLGRAPH_HOOK_TYPE", "PostToolUse")
    hook_input = json.load(sys.stdin)
    response = track_event(hook_type, hook_input)  # <- Creates event
    print(json.dumps(response))
```

**Problem**: Called twice per UserPromptSubmit event
- Once directly via hooks.json
- Once via user-prompt-submit.py → create_user_query_event()

---

### 3. user-prompt-submit.py - Prompt Analysis & Event Creation

**File**: `/Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/.claude-plugin/hooks/scripts/user-prompt-submit.py`

**Key Lines**:
- **Line 31-40**: Main entry point, loads prompt
- **Line 43**: Creates execution context
- **Line 51**: **CREATES SECOND UserQuery EVENT**: `user_query_event_id = create_user_query_event(context, prompt)`
- **Line 85**: Returns event ID in response

**What it does**:
```python
def main():
    hook_input = json.load(sys.stdin)
    prompt = hook_input.get("prompt", "")
    context = HookContext.from_input(hook_input)

    # ... analysis code ...

    user_query_event_id = create_user_query_event(context, prompt)  # <- Creates event

    result = {
        "user_query_event": {"event_id": user_query_event_id},
        # ... other fields ...
    }
    print(json.dumps(result))
```

**Problem**: Creates event independently from track-event.py

---

### 4. posttooluse-integrator.py - Unified PostToolUse Handler

**File**: `/Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/.claude-plugin/hooks/scripts/posttooluse-integrator.py`

**Key Lines**:
- **Line 12**: Imports handler: `from wipnote.hooks.posttooluse import main`
- **Line 14-15**: Entry point calls main: `if __name__ == "__main__": main()`

**What it does**:
```python
from wipnote.hooks.posttooluse import main

if __name__ == "__main__":
    main()
```

**Then delegates to** (see below):

---

### 5. pretooluse-integrator.py - Unified PreToolUse Handler

**File**: `/Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/.claude-plugin/hooks/scripts/pretooluse-integrator.py`

**Key Lines**:
- **Line 18**: Imports handler: `from wipnote.hooks.pretooluse import main`
- **Line 20-21**: Entry point: `if __name__ == "__main__": main()`

**What it does**:
```python
from wipnote.hooks.pretooluse import main

if __name__ == "__main__":
    main()
```

---

### 6. pretooluse-spawner-router.py - Task Spawner Routing

**File**: `/Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/.claude-plugin/hooks/scripts/pretooluse-spawner-router.py`

**Key Lines**:
- **Line 562-626**: `main()` function handles Task() routing
- **Line 576**: Checks if tool is Task: `if tool_name != "Task"`
- **Line 617**: Routes to spawner: `response = route_to_spawner(...)`

**What it does**:
```python
def main() -> None:
    hook_input = json.load(sys.stdin)
    tool_name = hook_input.get("name", "") or hook_input.get("tool_name", "")

    if tool_name == "Task":
        # Extract and route to spawner agent
        response = route_to_spawner(base_spawner_type, prompt, manifest, ...)
    else:
        # Pass through
        print(json.dumps({"continue": True}))
```

**Problem**: Duplicates spawner routing logic with pretooluse-integrator.py

---

## Core Event Tracking Implementation

### 7. event_tracker.py - Event Recording Logic

**File**: `/Users/shakes/DevProjects/htmlgraph/src/python/wipnote/hooks/event_tracker.py`

**Key Functions**:

**track_event()** (Line 669-1088):
- Main entry point for all event tracking
- Handles: PostToolUse, Stop, UserPromptSubmit
- Returns response dict with continue flag

**record_event_to_sqlite()** (Line 534-614):
```python
def record_event_to_sqlite(
    db, session_id, tool_name, tool_input, tool_response, is_error, ...
) -> str | None:
    event_id = generate_id("event")  # <- Random ID each time
    db.insert_event(
        event_id=event_id,
        tool_name=tool_name,
        input_summary=input_summary,
        ...
    )
```

**UserPromptSubmit Handling** (Lines 841-870):
```python
elif hook_type == "UserPromptSubmit":
    prompt = hook_input.get("prompt", "")
    preview = prompt[:100].replace("\n", " ")

    if db:
        record_event_to_sqlite(
            db=db,
            session_id=active_session_id,
            tool_name="UserQuery",  # <- Creates UserQuery event
            tool_input={"prompt": prompt},
            ...
        )
```

**PostToolUse Handling** (Lines 872-980):
```python
elif hook_type == "PostToolUse":
    tool_name = hook_input.get("tool_name", "unknown")
    ...
    if db:
        record_event_to_sqlite(
            db=db,
            session_id=active_session_id,
            tool_name=tool_name,  # <- Creates event for each tool
            ...
        )
```

---

### 8. posttooluse.py - Unified PostToolUse Hook

**File**: `/Users/shakes/DevProjects/htmlgraph/src/python/wipnote/hooks/posttooluse.py`

**Key Functions**:

**run_event_tracking()** (Line 39-64):
```python
async def run_event_tracking(
    hook_type: str, hook_input: dict[str, Any]
) -> dict[str, Any]:
    loop = asyncio.get_event_loop()
    return await loop.run_in_executor(
        None,
        track_event,  # <- CALLS track_event() HERE
        hook_type,
        hook_input,
    )
```

**posttooluse_hook()** (Line 276-376):
```python
async def posttooluse_hook(
    hook_type: str, hook_input: dict[str, Any]
) -> dict[str, Any]:
    (
        event_response,
        reflection_response,
        validation_response,
        error_tracking_response,
        debug_suggestions,
        cigs_response,
    ) = await asyncio.gather(
        run_event_tracking(hook_type, hook_input),  # <- Line 306
        run_orchestrator_reflection(hook_input),
        ...
    )
```

**Problem**: When posttooluse-integrator.py is called AND track-event.py is called:
1. track-event.py calls track_event() → creates event #1
2. posttooluse-integrator.py calls posttooluse.main() → calls track_event() → creates event #2

---

### 9. prompt_analyzer.py - UserQuery Creation

**File**: `/Users/shakes/DevProjects/htmlgraph/src/python/wipnote/hooks/prompt_analyzer.py`

**Function: create_user_query_event()** (not shown but called by user-prompt-submit.py:51):
- Creates UserQuery event in database
- Called independently from track-event.py

**Problem**: Both track-event.py AND user-prompt-submit.py call event creation functions

---

## SQLite Database

### 10. schema.py - Database Schema

**File**: `/Users/shakes/DevProjects/htmlgraph/src/python/wipnote/db/schema.py`

**insert_event()** (Line 500-588):
```python
def insert_event(
    self,
    event_id: str,
    agent_id: str,
    event_type: str,
    session_id: str,
    tool_name: str | None = None,
    ...
) -> bool:
    cursor.execute("""
        INSERT INTO agent_events
        (event_id, agent_id, event_type, session_id, tool_name, ...)
        VALUES (?, ?, ?, ?, ?, ...)
    """, (...))
    self.connection.commit()
    return True
```

**Key Issue**: No duplicate detection based on content hash
- Each event gets random event_id
- No check for existing event with same tool_name + input_summary + timestamp

**Database Table: agent_events**:
```sql
CREATE TABLE agent_events (
    event_id TEXT PRIMARY KEY,
    agent_id TEXT,
    tool_name TEXT,
    input_summary TEXT,
    output_summary TEXT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    ...
)
```

**Problem**: event_id is PRIMARY KEY, so each random ID is valid
- No constraint prevents duplicate input_summary in same session/timestamp

---

## Summary of Call Chains

### UserPromptSubmit Duplication Chain

```
UserPromptSubmit event fired
├─ Hook #1: track-event.py
│  └─ track_event("UserPromptSubmit", hook_input)
│     └─ record_event_to_sqlite(..., tool_name="UserQuery")
│        └─ INSERT event_id=evt-0b51fc47, input_summary="this is not how..."
│           [EVENT #1 CREATED at 08:03:14]
│
└─ Hook #2: user-prompt-submit.py
   └─ create_user_query_event(context, prompt)
      └─ INSERT event_id=uq-a43a48e4, input_summary="this is not how..."
         [EVENT #2 CREATED at 08:03:15 - DUPLICATE!]
```

### PostToolUse Duplication Chain

```
PostToolUse event fired
├─ Hook #1: track-event.py
│  └─ track_event("PostToolUse", hook_input)
│     └─ record_event_to_sqlite(..., tool_name=tool_name)
│        └─ INSERT event
│           [EVENT #1 CREATED]
│
└─ Hook #2: posttooluse-integrator.py
   └─ posttooluse.main()
      └─ posttooluse_hook("PostToolUse", hook_input)
         └─ run_event_tracking("PostToolUse", hook_input)
            └─ track_event("PostToolUse", hook_input)
               └─ record_event_to_sqlite(...)
                  └─ INSERT event
                     [EVENT #2 CREATED - DUPLICATE!]
```

---

## Files to Modify

**Single file change required**:
1. `/Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/.claude-plugin/hooks/hooks.json`
   - Remove track-event.py from UserPromptSubmit
   - Remove track-event.py from PostToolUse
   - Remove pretooluse-spawner-router.py from PreToolUse

**No code changes needed** - just hook registration changes.

---

## Verification Query

```bash
# Check for duplicates before fix
sqlite3 .wipnote/wipnote.db << 'EOF'
SELECT input_summary, COUNT(*) as count, GROUP_CONCAT(event_id) as ids
FROM agent_events
WHERE tool_name = 'UserQuery'
GROUP BY input_summary
HAVING COUNT(*) > 1;
EOF

# Expected before fix: Multiple rows with count=2
# Expected after fix: No rows (empty result)
```
