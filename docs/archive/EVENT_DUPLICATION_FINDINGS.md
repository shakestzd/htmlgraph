# Event Duplication Root Cause & Fix Guide

## Quick Summary

**Problem**: Events appear twice in the dashboard with identical content but different event IDs.

**Root Cause**: Multiple hooks are registered for the same event types, each creating duplicate database records.

**Solution**: Remove redundant hooks from `hooks.json` to ensure single execution per event.

---

## Evidence

### Database Query Results

```sql
SELECT input_summary, COUNT(*) as count, GROUP_CONCAT(event_id) as ids
FROM agent_events
WHERE tool_name = 'UserQuery'
GROUP BY input_summary
HAVING COUNT(*) > 1;
```

**Results**:
- "this is not how you invoke..." → 2 events (evt-0b51fc47, uq-a43a48e4)
- "use playwrite to test this in the ui" → 2 events (evt-a6da1954, uq-0e296212)

### Why Different Event IDs?

Each hook execution calls `generate_id("event")` which creates a random unique ID:
```python
event_id = generate_id("event")  # Random → evt-xxxx or uq-xxxx
```

Even though the content is identical, the IDs differ, making them appear as separate events.

---

## Hook Duplication Locations

### 1. UserPromptSubmit Event (HIGHEST PRIORITY)

**File**: `packages/claude-plugin/.claude-plugin/hooks/hooks.json` (Lines 3-18)

**Current State**:
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

**Execution Chain 1** (track-event.py):
```
track-event.py
  → event_tracker.track_event("UserPromptSubmit", hook_input)
    → record_event_to_sqlite()
      → INSERT INTO agent_events (tool_name='UserQuery', ...)
         [CREATES EVENT #1: evt-0b51fc47]
```

**Execution Chain 2** (user-prompt-submit.py):
```
user-prompt-submit.py
  → create_user_query_event(context, prompt)
    → INSERT INTO agent_events (tool_name='UserQuery', ...)
       [CREATES EVENT #2: uq-a43a48e4]
```

**Both hooks run**, both create the same UserQuery event → **DUPLICATE**

**Code References**:
- `track-event.py`: Line 103 calls `track_event(hook_type, hook_input)`
- `event_tracker.py`: Lines 857-866 create UserQuery events
- `user-prompt-submit.py`: Line 51 calls `create_user_query_event(context, prompt)`

---

### 2. PostToolUse Event (HIGH PRIORITY)

**File**: `packages/claude-plugin/.claude-plugin/hooks/hooks.json` (Lines 70-85)

**Current State**:
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

**Execution Chain 1** (track-event.py):
```
track-event.py
  → event_tracker.track_event("PostToolUse", hook_input)
    → record_event_to_sqlite()
       [CREATES EVENT #1]
```

**Execution Chain 2** (posttooluse-integrator.py):
```
posttooluse-integrator.py
  → wipnote.hooks.posttooluse.main()
    → posttooluse_hook("PostToolUse", hook_input)
      → run_event_tracking("PostToolUse", hook_input)
        → event_tracker.track_event("PostToolUse", hook_input)
          → record_event_to_sqlite()
             [CREATES EVENT #2 - DUPLICATE!]
```

**Code References**:
- `posttooluse.py`: Line 306 calls `run_event_tracking(hook_type, hook_input)`
- `posttooluse.py`: Line 58 calls `track_event(hook_type, hook_input)`
- `event_tracker.py`: Lines 872-980 handle PostToolUse events

---

### 3. PreToolUse Event (MEDIUM PRIORITY)

**File**: `packages/claude-plugin/.claude-plugin/hooks/hooks.json` (Lines 53-68)

**Current State**:
```json
"PreToolUse": [
  {
    "matcher": "",
    "hooks": [
      {
        "type": "command",
        "comment": "Route Task() calls to spawner agents",
        "command": "uv run \"${CLAUDE_PLUGIN_ROOT}/hooks/scripts/pretooluse-integrator.py\""
      },
      {
        "type": "command",
        "comment": "Route Task() calls to spawner agents",
        "command": "uv run \"${CLAUDE_PLUGIN_ROOT}/hooks/scripts/pretooluse-spawner-router.py\""
      }
    ]
  }
]
```

**Issue**: Both hooks handle Task() routing independently
- `pretooluse-integrator.py`: Line 18 calls `wipnote.hooks.pretooluse.main()`
- `pretooluse-spawner-router.py`: Line 562 implements separate spawner routing logic

**Result**: Task() calls may be routed twice or handled by both hooks

---

## Implementation Fix

### Step 1: Update hooks.json

Remove the duplicate hooks, keeping only the comprehensive integrator:

**Before (UserPromptSubmit)**:
```json
"UserPromptSubmit": [
  {
    "matcher": "",
    "hooks": [
      { "type": "command", "command": "... track-event.py ..." },
      { "type": "command", "command": "... user-prompt-submit.py ..." }
    ]
  }
]
```

**After (UserPromptSubmit)**:
```json
"UserPromptSubmit": [
  {
    "matcher": "",
    "hooks": [
      { "type": "command", "command": "uv run \"${CLAUDE_PLUGIN_ROOT}/hooks/scripts/user-prompt-submit.py\"" }
    ]
  }
]
```

**Before (PostToolUse)**:
```json
"PostToolUse": [
  {
    "matcher": "",
    "hooks": [
      { "type": "command", "command": "... track-event.py ..." },
      { "type": "command", "command": "... posttooluse-integrator.py ..." }
    ]
  }
]
```

**After (PostToolUse)**:
```json
"PostToolUse": [
  {
    "matcher": "",
    "hooks": [
      { "type": "command", "command": "uv run \"${CLAUDE_PLUGIN_ROOT}/hooks/scripts/posttooluse-integrator.py\"" }
    ]
  }
]
```

**Before (PreToolUse)**:
```json
"PreToolUse": [
  {
    "matcher": "",
    "hooks": [
      { "type": "command", "command": "... pretooluse-integrator.py ..." },
      { "type": "command", "command": "... pretooluse-spawner-router.py ..." }
    ]
  }
]
```

**After (PreToolUse)**:
```json
"PreToolUse": [
  {
    "matcher": "",
    "hooks": [
      { "type": "command", "command": "uv run \"${CLAUDE_PLUGIN_ROOT}/hooks/scripts/pretooluse-integrator.py\"" }
    ]
  }
]
```

### Step 2: Verify Integrator Scripts Handle All Functionality

Ensure that `user-prompt-submit.py` calls `create_user_query_event()`:
- ✅ Confirmed at Line 51 in user-prompt-submit.py

Ensure that `posttooluse-integrator.py` handles all tracking:
- ✅ Calls `posttooluse_hook()` which runs `run_event_tracking()`

Ensure that `pretooluse-integrator.py` handles spawner routing:
- Need to verify if spawner routing is included in pretooluse.main()

### Step 3: Run Tests

```bash
# Unit tests for hooks
uv run pytest tests/hooks/ -v

# Integration tests
uv run pytest tests/integration/ -v

# Check for duplicate events
sqlite3 .wipnote/wipnote.db << 'EOF'
SELECT input_summary, COUNT(*) as count
FROM agent_events
GROUP BY input_summary
HAVING COUNT(*) > 1;
EOF
```

Expected output: Empty (no duplicates)

### Step 4: Manual Testing

1. Submit a user query
2. Check database:
   ```bash
   sqlite3 .wipnote/wipnote.db \
     "SELECT COUNT(*) FROM agent_events WHERE tool_name='UserQuery'"
   ```
   Expected: 1 event (not 2)

3. Verify dashboard displays single events
4. Check git status for any remaining issues

---

## Files to Modify

```
packages/claude-plugin/.claude-plugin/hooks/hooks.json
  - Remove track-event.py from UserPromptSubmit
  - Remove track-event.py from PostToolUse
  - Remove pretooluse-spawner-router.py from PreToolUse
```

## Files to Review (But Not Modify)

```
src/python/wipnote/hooks/event_tracker.py
  - Verify record_event_to_sqlite() only called once per event
  - OK - just a utility function

src/python/wipnote/hooks/posttooluse.py
  - Verify run_event_tracking() is the single source for PostToolUse events
  - OK - unified implementation

packages/claude-plugin/.claude-plugin/hooks/scripts/pretooluse-integrator.py
  - Verify spawner routing is integrated
  - May need to merge spawner-router.py logic if missing
```

---

## Summary Table

| Event Type | Current Hooks | Issue | Solution |
|---|---|---|---|
| UserPromptSubmit | 2 hooks | Both create UserQuery events | Keep user-prompt-submit.py, remove track-event.py |
| PostToolUse | 2 hooks | track-event.py + posttooluse-integrator both call track_event() | Keep posttooluse-integrator.py, remove track-event.py |
| PreToolUse | 2 hooks | Duplicate spawner routing logic | Keep pretooluse-integrator.py, verify spawner routing is included |

---

## Expected Outcome

After applying these fixes:
- ✅ Each user query creates 1 event (not 2)
- ✅ Each tool use creates 1 event (not 2)
- ✅ Dashboard shows no duplicate events
- ✅ Database contains single records per activity
- ✅ All tests pass
- ✅ Hooks remain fully functional
