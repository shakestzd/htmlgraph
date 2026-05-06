# Gemini CLI Investigation Report

**Date**: 2026-01-12
**Status**: ROOT CAUSE IDENTIFIED
**Severity**: Configuration Issue (Easy Fix)

## Executive Summary

The GeminiSpawner subprocess tracking is working correctly, but CLI execution fails due to a **model compatibility issue**. The Gemini CLI v0.22.5 has "thinking" mode enabled by default, which is incompatible with older models like `gemini-2.0-flash` and `gemini-2.0-flash-exp`.

## Root Cause

**Error Message**:
```
Unable to submit request because thinking is not supported by this model.
```

**Technical Details**:
- Gemini CLI v0.22.5 defaults to models that support "thinking" (reasoning) mode
- When `-m gemini-2.0-flash` is specified, the CLI still tries to use thinking mode
- The 2.0 series models don't support thinking, causing API error 400
- Default models (`gemini-2.5-flash-lite`, `gemini-3-flash-preview`) work correctly

## Evidence

### Working Commands
```bash
# No model specified - uses defaults with thinking support
gemini "What is 2+2?" --output-format json
# Result: Success, uses gemini-2.5-flash-lite and gemini-3-flash-preview

gemini -p "What is 2+2?" --output-format json --yolo
# Result: Success, response: "4"

gemini -p "What is 2+2?" --output-format json -m gemini-2.5-flash-lite
# Result: Success
```

### Failing Commands
```bash
gemini -p "What is 2+2?" --output-format json -m gemini-2.0-flash
# Result: Error 400 - thinking not supported

gemini -p "What is 2+2?" --output-format json -m gemini-2.0-flash-exp
# Result: Error 400 - thinking not supported

gemini -p "What is 2+2?" --output-format json -m gemini-1.5-flash
# Result: Error - thinking not supported
```

### Database Evidence
All failed Gemini events in `agent_events` table used explicit model specification:
```sql
SELECT DISTINCT json_extract(context, '$.spawned_agent') as model
FROM agent_events
WHERE tool_name LIKE '%gemini%' AND status = 'failed';
-- Result: gemini-2.0-flash
```

### Error Report (from /var/folders/.../gemini-client-error-*.json)
```json
{
  "error": {
    "code": 400,
    "message": "Unable to submit request because thinking is not supported by this model.",
    "status": "INVALID_ARGUMENT"
  }
}
```

## GeminiSpawner Code Analysis

**File**: `src/python/wipnote/orchestration/spawners/gemini.py`

The spawner correctly builds commands but hardcodes `gemini-2.0-flash` in tracking:
```python
# Line 141 - Command building
cmd = ["gemini", "-p", prompt, "--output-format", output_format]

# Line 144-145 - Model option (when specified by caller)
if model:
    cmd.extend(["-m", model])

# Line 188 - Hardcoded model in tracking (cosmetic issue)
spawned_agent="gemini-2.0-flash",
```

**Key Finding**: The spawner passes whatever model is requested. The issue is that callers were requesting `gemini-2.0-flash-exp` which doesn't support thinking.

## Solutions

### Option A: Remove Model Specification (Recommended - Immediate Fix)

**Change**: Don't pass `-m` flag, let Gemini CLI use its smart defaults.

```python
# In gemini.py spawn() method
# Remove or comment out:
# if model:
#     cmd.extend(["-m", model])
```

**Pros**:
- Immediate fix
- Always uses latest compatible models
- Benefits from CLI's automatic model selection

**Cons**:
- Less control over model selection
- May use more expensive models

### Option B: Update to Thinking-Compatible Models

**Change**: Update default model references from `gemini-2.0-flash` to `gemini-2.5-flash-lite`.

```python
# Update default model
DEFAULT_MODEL = "gemini-2.5-flash-lite"  # Was: gemini-2.0-flash
```

**Pros**:
- Explicit model control
- Predictable behavior

**Cons**:
- Requires code change
- May need periodic updates as models evolve

### Option C: Add Thinking Mode Toggle (Future Enhancement)

**Change**: Add CLI flag to disable thinking mode if/when Gemini CLI supports it.

```python
# Hypothetical future flag
cmd.append("--no-thinking")
```

**Status**: Not currently supported by Gemini CLI v0.22.5

### Option D: Document as User Responsibility

**Change**: Update documentation to warn users about model compatibility.

**Pros**:
- No code changes needed

**Cons**:
- Poor user experience
- Errors not obvious

## Recommended Fix

**Implement Option A + B**:

1. **Immediate**: Remove explicit model specification in GeminiSpawner (let CLI choose)
2. **Update tracking**: Change hardcoded `spawned_agent="gemini-2.0-flash"` to dynamic value
3. **Document**: Add note about thinking mode compatibility

### Code Changes Required

**File**: `src/python/wipnote/orchestration/spawners/gemini.py`

```python
# Change 1: Don't force model selection (around line 144-145)
# Remove or make optional:
# if model:
#     cmd.extend(["-m", model])

# Change 2: Update tracking to reflect actual model (line 188)
spawned_agent=model or "gemini-default",  # Was: "gemini-2.0-flash"
```

## Installation Status

| Component | Status | Version |
|-----------|--------|---------|
| Gemini CLI | Installed | 0.22.5 |
| Location | `/Users/shakes/.nvm/versions/node/v22.20.0/bin/gemini` | - |
| OAuth Credentials | Valid | `~/.gemini/oauth_creds.json` |
| API Access | Working | Default models only |
| MCP Extensions | 3 loaded | wipnote, chrome-devtools, clasp |

## Extension Warning (Non-Critical)

```
[ERROR] [ImportProcessor] Failed to import google/clasp: ENOENT
```

This warning appears but doesn't affect Gemini CLI functionality. The clasp extension is configured but its files are missing.

## Verification Steps

After implementing fix:

```bash
# 1. Test spawner directly
cd /Users/shakes/DevProjects/htmlgraph
python -c "
from wipnote.orchestration.spawners import GeminiSpawner
spawner = GeminiSpawner()
result = spawner.spawn('What is 2+2?')
print(f'Success: {result.success}')
print(f'Response: {result.response}')
"

# 2. Verify database shows successful events
sqlite3 .wipnote/wipnote.db "
SELECT tool_name, status, created_at
FROM agent_events
WHERE tool_name LIKE '%gemini%'
ORDER BY created_at DESC LIMIT 5;
"
```

## Conclusion

The GeminiSpawner tracking system works correctly. The failures are due to requesting models (`gemini-2.0-flash`) that don't support the "thinking" feature that Gemini CLI v0.22.5 enables by default.

**Fix Priority**: Medium (functional workaround exists - don't specify model)
**Estimated Effort**: 15 minutes code change + testing
**Risk**: Low (removing model flag is backward compatible)
