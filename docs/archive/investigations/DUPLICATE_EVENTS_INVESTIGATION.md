# Duplicate Events Investigation Report

## Problem Statement

The Wipnote dashboard displays duplicate events with identical content but different event IDs and timestamps (1 second apart):

- Event "this is not how you invoke..." appears twice:
  - `uq-a43a48e4` at 2026-01-11 08:03:15
  - `evt-0b51fc47` at 2026-01-11 08:03:14

- Event "use playwrite to test this in the ui" appears twice:
  - `uq-0e296212` at 2026-01-11 07:58:26
  - `evt-a6da1954` at 2026-01-11 07:58:25

## Root Cause Analysis

**Two hooks are both inserting UserQuery events for the same user query:**

### 1. **Hook Chain Issue in hooks.json**

In `packages/claude-plugin/.claude-plugin/hooks/hooks.json`, the **UserPromptSubmit** event has TWO hooks registered:

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

### 2. **Both Hooks Create Events**

Both hooks are creating UserQuery events in the database:

- **track-event.py** (Line 10): Calls `track_event("UserPromptSubmit", hook_input)`
  - Calls `record_event_to_sqlite()` (event_tracker.py:857-866)
  - Creates event with `tool_name="UserQuery"`

- **user-prompt-submit.py** (Line 51): Calls `create_user_query_event(context, prompt)`
  - Creates a SECOND UserQuery event in the database
  - Confirmed in user-prompt-submit.py line 51

### 3. **Event ID Generation Creates Different IDs**

Each time `track_event()` runs, it generates a new random event ID using `generate_id("event")`:
- First call: `evt-0b51fc47` at 08:03:14
- Second call: `uq-a43a48e4` at 08:03:15

Since the IDs are different (random generation), duplicate detection doesn't work.

## Evidence from Database

```sql
SELECT input_summary, COUNT(*) as count, GROUP_CONCAT(event_id) as ids
FROM agent_events
WHERE tool_name = 'UserQuery' AND input_summary LIKE '%not how you invoke%'
GROUP BY input_summary;

-- Result:
-- "this is not how you invoke..." | 2 | evt-0b51fc47, uq-a43a48e4
```

Same identical text content, different event IDs, 1 second apart.

## Hook Chain Duplication Analysis

### PostToolUse Chain (Also Duplicated)

The PostToolUse hook has similar duplication:

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

**However**: `posttooluse-integrator.py` delegates to `wipnote.hooks.posttooluse`, which calls `track_event()` from within `run_event_tracking()`. This is ALSO creating duplicate events!

### PreToolUse Chain (Also Duplicated)

```json
"PreToolUse": [
  {
    "type": "command",
    "comment": "Route Task() calls to spawner agents",
    "command": "uv run \"${CLAUDE_PLUGIN_ROOT}/hooks/scripts/pretooluse-integrator.py\""
  },
  {
    "type": "command",
    "comment": "Route Task() calls to spawner agents (DUPLICATE!)",
    "command": "uv run \"${CLAUDE_PLUGIN_ROOT}/hooks/scripts/pretooluse-spawner-router.py\""
  }
]
```

Two separate executables handling Task() routing. This is redundant.

## Files Involved

### Hook Registration (Root Cause)
- `/Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/.claude-plugin/hooks/hooks.json`
  - Lines 3-18: UserPromptSubmit with 2 hooks
  - Lines 70-85: PostToolUse with 2 hooks
  - Lines 53-68: PreToolUse with 2 hooks

### Event Tracking Scripts
- `/Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/.claude-plugin/hooks/scripts/track-event.py`
  - Wrapper that calls `event_tracker.track_event()`

- `/Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/.claude-plugin/hooks/scripts/user-prompt-submit.py`
  - Unknown - likely also calls event tracking

- `/Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/.claude-plugin/hooks/scripts/posttooluse-integrator.py`
  - Thin wrapper calling `wipnote.hooks.posttooluse.main()`

- `/Users/shakes/DevProjects/htmlgraph/src/python/wipnote/hooks/posttooluse.py`
  - Line 306: Calls `run_event_tracking(hook_type, hook_input)`
  - Which calls `track_event(hook_type, hook_input)` (Line 58)

### Core Event Tracking
- `/Users/shakes/DevProjects/htmlgraph/src/python/wipnote/hooks/event_tracker.py`
  - Lines 857-866: Creates UserQuery events in PostToolUse handling
  - No duplicate detection based on content hash

## Recommended Solution

### Option 1: Remove Duplicate Hook (RECOMMENDED)

**In hooks.json**, keep only ONE hook per event type that does all tracking:

**For UserPromptSubmit:**
- Keep `track-event.py` (comprehensive event tracking)
- Remove `user-prompt-submit.py` (if it only duplicates tracking)

**For PostToolUse:**
- Keep `posttooluse-integrator.py` (unified parallel execution)
- Remove separate `track-event.py` call (it's redundant since posttooluse-integrator calls track_event)

**For PreToolUse:**
- Keep `pretooluse-integrator.py` (unified orchestrator + validation)
- Remove `pretooluse-spawner-router.py` (spawner routing should be in pretooluse-integrator)

### Option 2: Add Duplicate Detection

Add idempotency check in `record_event_to_sqlite()`:

```python
# Check if similar event already exists (within 2 seconds, same session/tool/summary)
cursor.execute("""
    SELECT event_id FROM agent_events
    WHERE session_id = ?
      AND tool_name = ?
      AND input_summary = ?
      AND datetime(timestamp) > datetime('now', '-2 seconds')
    LIMIT 1
""", (session_id, tool_name, input_summary))

if cursor.fetchone():
    print(f"Duplicate event detected: {tool_name} '{input_summary[:50]}'", file=sys.stderr)
    return None  # Skip duplicate
```

### Option 3: Deduplicate on Dashboard

Add query-time deduplication when displaying events, but this masks the underlying issue.

## Recommendation

**Implement Option 1 (Remove Duplicates from hooks.json):**

1. **UserPromptSubmit**: Remove `user-prompt-submit.py` hook
   - Keep `track-event.py` for comprehensive tracking

2. **PostToolUse**: Remove separate `track-event.py` hook
   - `posttooluse-integrator.py` already calls `track_event()` via posttooluse.main()
   - Remove the redundant hook chain

3. **PreToolUse**: Keep only `pretooluse-integrator.py`
   - Verify spawner routing is integrated
   - Remove `pretooluse-spawner-router.py`

This eliminates the root cause: duplicate hook execution creating duplicate events with the same content but different IDs.

## Testing After Fix

1. Submit a user query
2. Check database:
   ```sql
   SELECT COUNT(*) FROM agent_events
   WHERE tool_name='UserQuery'
   AND input_summary='test query'
   ```
   Should return **1**, not 2+

3. Verify dashboard shows single events, no duplicates

4. Run test suite to ensure hooks still function:
   ```bash
   uv run pytest tests/hooks/ -v
   ```
