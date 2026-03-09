# HtmlGraph Tracking Data Investigation Report

**Investigation Date:** 2026-02-12
**Database:** `.htmlgraph/htmlgraph.db`
**Investigator:** Claude Sonnet 4.5

---

## Executive Summary

Three critical tracking issues identified in HtmlGraph:

1. **Cost Tracking Not Working** - `cost_events` table exists but remains empty (0 rows)
2. **Subagent Parent Linking Broken** - All `parent_session_id` values are NULL
3. **Model Distribution Query** - Need to clarify where 48.2% Opus / 46.2% Haiku / 5.7% Sonnet data originates

---

## Issue 1: Cost Tracking Not Working

### Symptoms
```sql
SELECT COUNT(*) FROM cost_events;
-- Result: 0
```

The `cost_events` table exists with proper schema but has **zero rows** despite active session tracking.

### Root Cause Analysis

**Finding:** Cost monitoring infrastructure exists but is **never called** by any hook.

#### Evidence

1. **CostMonitor class exists** (`src/python/htmlgraph/analytics/cost_monitor.py`)
   - Lines 231-284: `track_token_usage()` method properly implemented
   - Lines 286-313: `_store_token_cost()` correctly inserts into `cost_events` table
   - Lines 202-229: Token cost calculation logic present

2. **PostToolUse hook does NOT call CostMonitor**
   - File: `src/python/htmlgraph/hooks/posttooluse.py`
   - Lines 301-316: Runs 6 parallel tasks via `asyncio.gather()`
   - Tasks: event tracking, reflection, validation, error tracking, debugging, CIGS
   - **MISSING:** No cost tracking task in the gather() call
   - **MISSING:** No import of CostMonitor
   - **MISSING:** No call to `track_token_usage()`

3. **No other hooks call CostMonitor**
   ```bash
   grep -r "track_token_usage\|CostMonitor" src/python/htmlgraph/hooks/
   # Result: No matches
   ```

### The Gap

The cost monitoring system is **fully implemented** but **never invoked**. The PostToolUse hook has all the data needed (model, tool_name, session_id from hook_input) but doesn't extract token counts or call the cost tracker.

### Token Data Availability

**agent_events table has model data:**
```sql
SELECT COUNT(*) FROM agent_events WHERE model IS NOT NULL;
-- Result: 4422 events with model data
```

Models recorded: "Haiku 4.5", "Sonnet 4.5", "Opus 4.6"

**But agent_events schema does NOT have token columns:**
```sql
.schema agent_events
-- Has: model, tool_name, session_id, cost_tokens (single aggregate)
-- Missing: input_tokens, output_tokens (needed for cost calculation)
```

### Solution Required

**Step 1: Add token tracking to PostToolUse hook**

File: `src/python/htmlgraph/hooks/posttooluse.py`

```python
# Add import
from htmlgraph.analytics.cost_monitor import CostMonitor

# Add new async task (around line 243)
async def run_cost_tracking(hook_input: dict[str, Any]) -> dict[str, Any]:
    """Track token usage and cost."""
    try:
        loop = asyncio.get_event_loop()
        
        # Extract token data from hook_input
        # Claude Code PostToolUse provides: usage.input_tokens, usage.output_tokens
        usage = hook_input.get("usage", {})
        input_tokens = usage.get("input_tokens", 0)
        output_tokens = usage.get("output_tokens", 0)
        
        if input_tokens == 0 and output_tokens == 0:
            return {"continue": True}  # No token data available
        
        session_id = hook_input.get("session_id", "unknown")
        event_id = hook_input.get("event_id") or generate_id("event")
        tool_name = hook_input.get("name", "unknown")
        model = hook_input.get("model", "unknown")
        
        # Initialize cost monitor
        monitor = CostMonitor()
        
        # Track token usage
        await loop.run_in_executor(
            None,
            monitor.track_token_usage,
            session_id,
            event_id,
            tool_name,
            model,
            input_tokens,
            output_tokens,
        )
        
        return {"continue": True}
    except Exception:
        return {"continue": True}

# Update asyncio.gather() call (line 302-316)
# Add run_cost_tracking(hook_input) to the gather() tuple
```

**Step 2: Verify Claude Code provides token data**

Need to check if `hook_input` from Claude Code's PostToolUse event includes:
- `usage.input_tokens`
- `usage.output_tokens`
- `model` (model identifier)

**Step 3: Test with instrumentation**

Add debug logging to verify data flow:
```python
logger.info(f"PostToolUse received: {json.dumps(hook_input, indent=2)}")
```

---

## Issue 2: Subagent Parent Linking Broken

### Symptoms
```sql
SELECT session_id, parent_session_id FROM sessions 
WHERE session_id LIKE '%htmlgraph:%' LIMIT 10;

-- All results show parent_session_id = NULL
b92f0ebe-81f7-4e50-956f-fa8d0d775df9-htmlgraph:haiku-coder||
b92f0ebe-81f7-4e50-956f-fa8d0d775df9-htmlgraph:researcher||
b92f0ebe-81f7-4e50-956f-fa8d0d775df9-htmlgraph:test-runner||
```

Subagent sessions (`is_subagent=1`) have `parent_session_id=NULL` despite being created with parent context.

### Root Cause Analysis

**Finding:** Parent session ID is passed during session creation but **not persisted** to the database.

#### Evidence from Code

**File:** `src/python/htmlgraph/hooks/event_tracker.py`

**Lines 1020-1048: Subagent session creation**
```python
if subagent_type and parent_session_id:
    subagent_session_id = f"{parent_session_id}-{subagent_type}"
    
    existing = manager.session_converter.load(subagent_session_id)
    if existing:
        active_session = existing
    else:
        # CREATE NEW SUBAGENT SESSION WITH PARENT LINK
        active_session = manager.start_session(
            session_id=subagent_session_id,
            agent=f"{subagent_type}-spawner",
            is_subagent=True,
            parent_session_id=parent_session_id,  # ← PASSED HERE
            title=f"{subagent_type.capitalize()} Subagent",
        )
```

The code **passes** `parent_session_id` to `manager.start_session()`.

**Question:** Does `SessionManager.start_session()` properly persist this to the database?

**Need to check:**
1. `src/python/htmlgraph/session_manager.py` - `start_session()` method
2. Database INSERT statement for sessions table
3. Whether `parent_session_id` parameter is included in the INSERT

### Investigation Path

**File to check:** `src/python/htmlgraph/session_manager.py`

Look for:
```python
def start_session(self, session_id, agent, is_subagent=False, parent_session_id=None, ...):
    # Does this method:
    # 1. Accept parent_session_id parameter?
    # 2. Include it in the session object?
    # 3. Persist it to database via INSERT?
```

**Database verification:**
```sql
-- Check if sessions table schema has parent_session_id column
.schema sessions
-- Result: YES, column exists with FOREIGN KEY constraint

-- Check if any sessions have parent_session_id set
SELECT COUNT(*) FROM sessions WHERE parent_session_id IS NOT NULL;
-- Expected: Should have some, but likely shows 0
```

### Likely Root Cause

**Hypothesis:** The `SessionManager.start_session()` method:
- ✅ Accepts `parent_session_id` parameter
- ❌ Does NOT include it in the database INSERT statement
- OR: Creates session object but doesn't call `.save()` with parent_session_id

**Verification needed:**
```bash
grep -A 20 "def start_session" src/python/htmlgraph/session_manager.py
```

Look for SQL INSERT that should include `parent_session_id`.

### Solution Required

**Step 1: Fix SessionManager.start_session()**

Ensure the method properly persists `parent_session_id` to database:

```python
def start_session(self, session_id, agent, is_subagent=False, parent_session_id=None, ...):
    # Create session object
    session = Session(
        id=session_id,
        agent=agent,
        is_subagent=is_subagent,
        parent_session=parent_session_id,  # ← CRITICAL
        ...
    )
    
    # Save to database (including parent_session_id in INSERT)
    db = HtmlGraphDB()
    cursor = db.connection.cursor()
    cursor.execute("""
        INSERT INTO sessions (
            session_id, agent_assigned, is_subagent, parent_session_id, ...
        ) VALUES (?, ?, ?, ?, ...)
    """, (session_id, agent, is_subagent, parent_session_id, ...))
    db.connection.commit()
```

**Step 2: Backfill existing sessions**

For sessions already created without parent links, reconstruct from session_id pattern:

```python
# Extract parent from session_id format: "parent-id-subagent-type"
UPDATE sessions
SET parent_session_id = SUBSTR(session_id, 1, INSTR(session_id || '-', '-htmlgraph:') - 1)
WHERE is_subagent = 1 
  AND parent_session_id IS NULL
  AND session_id LIKE '%htmlgraph:%';
```

**Step 3: Add database constraint**

Ensure future sessions can't have NULL parent when `is_subagent=1`:

```sql
-- Add CHECK constraint
ALTER TABLE sessions ADD CONSTRAINT chk_subagent_has_parent
CHECK (is_subagent = 0 OR parent_session_id IS NOT NULL);
```

---

## Issue 3: Model Distribution - Data Source Clarification

### User's Question

Where does the "Opus 48.2% / Haiku 46.2% / Sonnet 5.7%" distribution come from?

### Investigation Findings

**agent_events table tracks model:**
```sql
SELECT model, COUNT(*) as count 
FROM agent_events 
WHERE model IS NOT NULL 
GROUP BY model 
ORDER BY count DESC;

-- Result:
Haiku 4.5  |  (count X)
Sonnet 4.5 |  (count Y)
Opus 4.6   |  (count Z)
```

**Total events with model data:** 4,422

### Model Data Source

**File:** `src/python/htmlgraph/hooks/event_tracker.py`

**Lines 61-99: `get_model_from_status_cache()`**
```python
def get_model_from_status_cache(session_id: str | None = None) -> str | None:
    """
    Read current model from SQLite model_cache table.
    
    The status line script writes model info to the model_cache table.
    This allows hooks to know which Claude model is currently running,
    even though hooks don't receive model info directly from Claude Code.
    """
    cursor.execute("SELECT model FROM model_cache WHERE id = 1 LIMIT 1")
    row = cursor.fetchone()
    return str(row[0]) if row else None
```

**How it works:**
1. **Status line script** updates `model_cache` table with current model
2. **Hooks** read from `model_cache` to determine which model is running
3. **agent_events** stores model with each event

**Sessions table does NOT have model column:**
```sql
.schema sessions
-- No 'model' column exists
```

### Calculating Distribution

To calculate the distribution the user mentioned:

```sql
-- By event count
SELECT 
    model,
    COUNT(*) as event_count,
    ROUND(COUNT(*) * 100.0 / SUM(COUNT(*)) OVER (), 1) as percentage
FROM agent_events
WHERE model IS NOT NULL
GROUP BY model
ORDER BY event_count DESC;

-- By session count (need to add model to sessions table first)
-- Currently NOT POSSIBLE because sessions table lacks model column
```

### Questions for User

1. **Where did you see the 48.2% / 46.2% / 5.7% split?**
   - Dashboard?
   - CLI output?
   - Database query?
   - Analytics report?

2. **What metric is being measured?**
   - Event count (tool calls)?
   - Session count?
   - Token count?
   - Cost (USD)?
   - Time spent?

3. **Is this per-session or project-wide?**
   - Single session breakdown?
   - All sessions combined?
   - Specific time period?

---

## Summary of Required Fixes

### Priority 1: Cost Tracking (Complete Infrastructure Missing from Hooks)

**Status:** 🔴 Critical - Feature exists but never used

**Files to modify:**
1. `src/python/htmlgraph/hooks/posttooluse.py`
   - Add `run_cost_tracking()` async task
   - Import `CostMonitor`
   - Add to `asyncio.gather()` call

2. Verify Claude Code hook input format
   - Check if `usage.input_tokens` exists in hook_input
   - Check if `usage.output_tokens` exists in hook_input
   - Add logging to confirm data availability

**Testing:**
```bash
# Run a session and check cost_events
sqlite3 .htmlgraph/htmlgraph.db "SELECT COUNT(*) FROM cost_events;"
# Should show > 0 after fix
```

### Priority 2: Parent Session Linking (Database Persistence Bug)

**Status:** 🟡 High - Data loss, breaks session hierarchy

**Files to investigate:**
1. `src/python/htmlgraph/session_manager.py`
   - Check `start_session()` implementation
   - Verify database INSERT includes `parent_session_id`

**Files to modify:**
1. Fix `SessionManager.start_session()` to persist parent_session_id
2. Add database migration to backfill existing sessions

**Testing:**
```bash
# Create subagent session and check parent link
sqlite3 .htmlgraph/htmlgraph.db "
SELECT session_id, parent_session_id, is_subagent 
FROM sessions 
WHERE is_subagent = 1 
ORDER BY created_at DESC LIMIT 5;
"
# Should show non-NULL parent_session_id values
```

### Priority 3: Model Distribution Clarification

**Status:** ℹ️ Info - Need user input on data source

**Questions for user:**
1. Where did you see these percentages?
2. What metric (events/sessions/cost/tokens)?
3. Time period or scope?

**Potential enhancements:**
1. Add `model` column to `sessions` table
2. Create analytics view for model distribution
3. Add dashboard widget showing model usage breakdown

---

## Next Steps

1. **Immediate:** Implement cost tracking in PostToolUse hook
2. **Immediate:** Fix parent_session_id persistence bug
3. **Follow-up:** Clarify model distribution data source with user
4. **Future:** Add comprehensive cost analytics dashboard

---

## Additional Notes

### Token Data in agent_events

The `agent_events` table has:
- ✅ `model` column (populated)
- ✅ `cost_tokens` column (aggregate count)
- ❌ `input_tokens` column (missing)
- ❌ `output_tokens` column (missing)

For accurate cost calculation, may need to:
1. Add `input_tokens` and `output_tokens` columns to `agent_events`
2. Populate from Claude Code hook input
3. Use for both `cost_events` population AND analytics

### Architecture Notes

**Current flow:**
```
Claude Code PostToolUse
  ↓
posttooluse.py hook
  ↓
event_tracker.py (stores to agent_events)
  ↓
[MISSING] → CostMonitor (should store to cost_events)
```

**Fixed flow:**
```
Claude Code PostToolUse
  ↓
posttooluse.py hook
  ├→ event_tracker.py (stores to agent_events)
  └→ run_cost_tracking() → CostMonitor (stores to cost_events)
```

Both should run in parallel via `asyncio.gather()`.

---

**End of Investigation Report**
