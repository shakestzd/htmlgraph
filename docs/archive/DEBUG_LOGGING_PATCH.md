# Debug Logging Patch - Verify task_id Availability

## Purpose

Determine if Claude Code's PostToolUse hook provides `task_id` in the `tool_response` field when Task() completes.

This is the CRITICAL first step that will determine if the entire solution is feasible.

---

## What to Add

### File: `src/python/htmlgraph/hooks/posttooluse.py`

**Location:** In the `run_event_tracking()` function (after line 40)

**Add this debug code:**

```python
async def run_event_tracking(
    hook_type: str, hook_input: dict[str, Any]
) -> dict[str, Any]:
    """
    Run event tracking (async wrapper).

    Args:
        hook_type: "PostToolUse" or "Stop"
        hook_input: Hook input with tool execution details

    Returns:
        Event tracking response: {"continue": True, "hookSpecificOutput": {...}}
    """

    # ============ DEBUG: Log Task() responses ============
    import sys
    import json

    tool_name = hook_input.get("name", "") or hook_input.get("tool_name", "")
    if tool_name == "Task":
        tool_response = hook_input.get("tool_response", {})

        print("\n" + "="*60, file=sys.stderr)
        print("DEBUG: Task() PostToolUse Hook Input", file=sys.stderr)
        print("="*60, file=sys.stderr)

        print(f"\nTool Name: {tool_name}", file=sys.stderr)
        print(f"Hook Type: {hook_type}", file=sys.stderr)

        if isinstance(tool_response, dict):
            print(f"\nTool Response Type: dict", file=sys.stderr)
            print(f"Tool Response Keys: {list(tool_response.keys())}", file=sys.stderr)

            # Check for task_id field
            if "task_id" in tool_response:
                task_id = tool_response.get("task_id")
                print(f"\n✅ FOUND task_id: {task_id}", file=sys.stderr)
            else:
                print(f"\n❌ task_id NOT FOUND in response", file=sys.stderr)

            # Print full response for inspection
            print(f"\nFull Tool Response:", file=sys.stderr)
            try:
                response_str = json.dumps(tool_response, indent=2, default=str)
                print(response_str, file=sys.stderr)
            except Exception as e:
                print(f"Could not serialize response: {e}", file=sys.stderr)
                print(f"Raw response: {tool_response}", file=sys.stderr)
        else:
            print(f"\nTool Response Type: {type(tool_response).__name__}", file=sys.stderr)
            print(f"Tool Response Value: {tool_response}", file=sys.stderr)

        print("\n" + "="*60 + "\n", file=sys.stderr)
    # ============ END DEBUG ============

    # Original code continues here
    try:
        loop = asyncio.get_event_loop()

        # Run in thread pool since it involves I/O
        return await loop.run_in_executor(
            None,
            track_event,
            hook_type,
            hook_input,
        )
    except Exception:
        # Graceful degradation - allow on error
        return {"continue": True}
```

---

## How to Test

### Step 1: Apply the Debug Patch
Copy the debug code above into `src/python/htmlgraph/hooks/posttooluse.py`

### Step 2: Deploy to PyPI (or use dev mode)
```bash
# Option A: Deploy to PyPI
./scripts/deploy-all.sh 0.26.12 --no-confirm

# Option B: Dev mode (hooks run from plugin source)
uv run htmlgraph claude --dev
```

### Step 3: Run a Task() in Claude Code
```
User: Create a spike documenting your findings

Claude: I'll help! Let me create a spike...
  → Uses Task() tool to delegate
  → PostToolUse hook fires
  → DEBUG output appears in stderr
```

### Step 4: Check Claude Code Logs
```bash
# Check where Claude Code logs appear
# Usually in:
# - Claude Code's stderr output
# - ~/.cache/claude-code/logs/
# - Terminal where Claude Code was launched

# Look for output like:
# ============================================================
# DEBUG: Task() PostToolUse Hook Input
# ============================================================
#
# Tool Name: Task
# Hook Type: PostToolUse
#
# Tool Response Type: dict
# Tool Response Keys: ['task_id', 'status', 'result', ...]
#
# ✅ FOUND task_id: task-abc123
# ...
```

---

## Expected Outputs

### Best Case: task_id IS Available ✅
```
============================================================
DEBUG: Task() PostToolUse Hook Input
============================================================

Tool Name: Task
Hook Type: PostToolUse

Tool Response Type: dict
Tool Response Keys: ['task_id', 'status', 'result', 'output']

✅ FOUND task_id: task-f0a1e8c9-8d2e-4a1f-8b3c-1e5f8a9c2d3e

Full Tool Response:
{
  "task_id": "task-f0a1e8c9-8d2e-4a1f-8b3c-1e5f8a9c2d3e",
  "status": "started",
  "result": "Task created and delegated",
  "output": "Subagent is now executing",
  "session_id": "sess-xyz123"
}

============================================================
```

**Interpretation:** ✅ Solution is feasible - task_id IS available
**Next Step:** Implement full capture logic

### Worst Case: task_id NOT Available ❌
```
============================================================
DEBUG: Task() PostToolUse Hook Input
============================================================

Tool Name: Task
Hook Type: PostToolUse

Tool Response Type: dict
Tool Response Keys: ['status', 'result', 'output', 'session_id']

❌ task_id NOT FOUND in response

Full Tool Response:
{
  "status": "started",
  "result": "Task created and delegated",
  "output": "Subagent is now executing",
  "session_id": "sess-xyz123"
}

============================================================
```

**Interpretation:** ❌ task_id not exposed by Claude Code
**Next Step:** Contact Anthropic, explore workarounds

### Alternative: Response is Not a Dict ⚠️
```
============================================================
DEBUG: Task() PostToolUse Hook Input
============================================================

Tool Name: Task
Hook Type: PostToolUse

Tool Response Type: string
Tool Response Value: "Task executed successfully"

============================================================
```

**Interpretation:** ⚠️ Response format unexpected
**Next Step:** Investigate what format Claude Code uses

---

## How to Interpret Results

| Scenario | Output | Action |
|----------|--------|--------|
| task_id found | ✅ FOUND task_id: ... | Proceed with implementation |
| task_id missing | ❌ task_id NOT FOUND | Request feature from Anthropic |
| Empty response | Tool Response Type: NoneType | Check Claude Code version |
| String response | Tool Response Type: string | Investigate response format |
| Error in logging | Exception while logging | Check debug code syntax |

---

## What Each Field Tells Us

### tool_response Keys
```python
{
    "task_id": "...",           # ← CRITICAL: What we're looking for
    "status": "started",        # State of the task
    "result": "...",            # Output message
    "output": "...",            # Additional output
    "session_id": "...",        # Session context
    "model": "...",             # Model used
    "temperature": "...",       # Temperature setting
    # ... potentially other fields
}
```

### What We Care About
- **task_id** - The key to linking Claude Code tasks to our events
- **status** - Can help us track task lifecycle
- **session_id** - Should match what we have in database

### What We Can Ignore
- **result/output** - Already have this in tool_response
- **model/temperature** - Already captured in tool_input

---

## Removing Debug Code

Once verification is complete, remove the debug section:

```python
# DELETE THIS ENTIRE SECTION:
# ============ DEBUG: Log Task() responses ============
import sys
import json

tool_name = hook_input.get("name", "") or hook_input.get("tool_name", "")
if tool_name == "Task":
    # ... all the debug code ...
# ============ END DEBUG ============
```

The rest of the function remains unchanged.

---

## Alternative: More Comprehensive Debug

If you want to log ALL PostToolUse calls (not just Task), use:

```python
async def run_event_tracking(
    hook_type: str, hook_input: dict[str, Any]
) -> dict[str, Any]:
    """Run event tracking (async wrapper)."""

    # ============ DEBUG: Log all PostToolUse calls ============
    import sys
    import json

    if hook_type == "PostToolUse":
        tool_name = hook_input.get("name", "") or hook_input.get("tool_name", "")
        print(f"\nDEBUG PostToolUse: {tool_name}", file=sys.stderr)

        tool_response = hook_input.get("tool_response", {})
        if isinstance(tool_response, dict) and tool_name == "Task":
            print(f"  Response keys: {list(tool_response.keys())}", file=sys.stderr)
            if "task_id" in tool_response:
                print(f"  ✅ task_id found: {tool_response['task_id']}", file=sys.stderr)
    # ============ END DEBUG ============

    # Original code
    try:
        loop = asyncio.get_event_loop()
        return await loop.run_in_executor(
            None,
            track_event,
            hook_type,
            hook_input,
        )
    except Exception:
        return {"continue": True}
```

---

## Documentation After Testing

### If task_id IS Available
**Update:** `INVESTIGATION_SUMMARY.md`
- Change "CRITICAL" question answer to: ✅ YES, task_id is available
- Mark Phase 2-5 as ready to implement
- Update effort estimate to 2-4 hours

### If task_id is NOT Available
**Update:** `INVESTIGATION_SUMMARY.md`
- Change "CRITICAL" question answer to: ❌ NO, task_id not exposed
- Mark investigation as "blocked on Anthropic feature"
- Document workarounds and their limitations
- Note in CLAUDE.md as known limitation

---

## Quick Reference: Where to Look for Logs

### Claude Code Logs
```bash
# Terminal where Claude Code is running
# Appears in stderr immediately after Task() PostToolUse fires

# Or check log files
~/.cache/claude-code/logs/
# Look for files with recent timestamps
```

### HtmlGraph Database
```bash
# Verify the event was created
sqlite3 .htmlgraph/htmlgraph.db << EOF
SELECT event_id, tool_name, event_type, context
FROM agent_events
WHERE tool_name = 'Task'
ORDER BY timestamp DESC
LIMIT 1;
EOF
```

### Environment Variables
```bash
# Check if task_id exported to environment
printenv | grep -i task
printenv | grep -i htmlgraph
```

---

## Expected Timeline

- **Add debug code:** 5 minutes
- **Deploy/restart Claude:** 2 minutes
- **Run test Task() call:** 2 minutes
- **Check logs:** 5 minutes
- **Interpret results:** 5 minutes
- **Update documentation:** 10 minutes

**Total:** ~30 minutes for complete verification

---

## Success Criteria for Debug Phase

✅ Debug code compiles without syntax errors
✅ Debug output appears in logs when Task() is called
✅ Output clearly shows presence or absence of task_id
✅ Can see full response structure
✅ No performance impact from debug logging
✅ Documentation updated with findings

---

## Commands Summary

```bash
# Apply patch
# Edit: src/python/htmlgraph/hooks/posttooluse.py
# Add: Debug code from above

# Deploy (choose one)
./scripts/deploy-all.sh 0.26.12 --no-confirm
# OR
uv run htmlgraph claude --dev

# Test in Claude Code
# → Create a spike or similar work that uses Task()
# → Watch logs for DEBUG output

# Check database for created event
sqlite3 .htmlgraph/htmlgraph.db "SELECT event_id, tool_name, context FROM agent_events WHERE tool_name='Task' LIMIT 1;"

# Remove debug code
# Edit: src/python/htmlgraph/hooks/posttooluse.py
# Delete: Debug section

# Commit findings
git add INVESTIGATION_SUMMARY.md
git commit -m "doc: verify task_id availability in PostToolUse hook"
```

---

## Notes

- Debug code is **non-breaking** - it only logs, doesn't change behavior
- Safe to deploy to production - output goes to stderr, not user-visible
- Can be left in place temporarily for monitoring
- Easy to remove once verification complete
- Recommended to keep in codebase as optional debug mode via env var

---

## Future: Conditional Debug (Recommended)

For production use, make debug optional:

```python
async def run_event_tracking(hook_type, hook_input):
    """Run event tracking (async wrapper)."""

    import os
    import sys
    import json

    # Only log if debug enabled
    if os.environ.get("HTMLGRAPH_DEBUG_TASK_ID") == "1":
        tool_name = hook_input.get("name", "") or hook_input.get("tool_name", "")
        if tool_name == "Task":
            tool_response = hook_input.get("tool_response", {})
            print(f"DEBUG Task Response: {json.dumps(tool_response, indent=2, default=str)}",
                  file=sys.stderr)

    # Original code...
```

**Usage:**
```bash
# Enable debug
export HTMLGRAPH_DEBUG_TASK_ID=1
uv run htmlgraph claude --dev

# Disable debug (default)
export HTMLGRAPH_DEBUG_TASK_ID=0
```
