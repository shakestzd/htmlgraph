# WebSocket Spawner Event Broadcasting - Debug Fix Summary

## Problem Statement

Events were being inserted into `live_events` table and marked as broadcast, but clients were not receiving `spawner_event` messages. No `[SpawnerEvent]` logs appeared in browser console despite successful WebSocket connection.

## Root Cause

**Unknown** - Insufficient logging made it impossible to determine where events were getting lost in the pipeline:
1. Were events being found by the polling query?
2. Were they being sent by the server?
3. Were they received by the client?
4. Was the handler function being called?

## Solution: Comprehensive Logging

Added logging at every step of the event pipeline to identify the failure point.

### Server-Side Changes (`src/python/wipnote/api/main.py`)

**Line 2148** - Log when pending events are found:
```python
logger.info(f"[WebSocket] Found {len(live_rows)} pending live_events to broadcast")
```

**Line 2182** - Log before sending each event:
```python
logger.info(f"[WebSocket] Sending spawner_event: id={live_id}, type={event_type}, spawner={spawner_type}")
```

**Line 2192** - Log when marking events as broadcast:
```python
logger.info(f"[WebSocket] Marking {len(broadcast_ids)} events as broadcast: {broadcast_ids}")
```

**Line 2145** - Type fix for mypy:
```python
live_rows = list(await live_cursor.fetchall())  # Convert to list for len()
```

### Client-Side Changes (`src/python/wipnote/api/templates/dashboard.html`)

**Line 164** - Log all incoming WebSocket message types:
```javascript
console.log('[WebSocket] Received message type:', data.type);
```

**Line 193** - Enhanced spawner_event logging:
```javascript
console.log('[WebSocket] spawner_event received:', data.event_type, data.spawner_type, 'handler exists:', typeof window.handleSpawnerEvent === 'function');
```

**Line 198** - Warning when handler missing:
```javascript
console.warn('[WebSocket] handleSpawnerEvent not available, spawner event dropped:', data.event_type, data.spawner_type);
```

## Testing Tools Created

### 1. Test Script (`test_spawner_broadcast.sh`)
Interactive script that:
- Guides user through testing process
- Inserts test event into database
- Shows expected server and client logs
- Executable: `./test_spawner_broadcast.sh`

### 2. Debug Guide (`SPAWNER_BROADCAST_DEBUG.md`)
Comprehensive troubleshooting guide with:
- Step-by-step testing procedure
- Expected log output at each stage
- Troubleshooting decision tree
- Database cleanup commands
- File locations reference

## Testing Procedure

1. **Start server**: `uv run wipnote serve`
2. **Open dashboard**: http://localhost:8888 (open console with F12)
3. **Insert test event**: `./test_spawner_broadcast.sh` or manual SQL insert
4. **Observe logs**: Check both server terminal and browser console

### Expected Log Flow

**Server (within 1-2 seconds of insert):**
```
INFO: [WebSocket] Found 1 pending live_events to broadcast
INFO: [WebSocket] Sending spawner_event: id=1, type=spawner_start, spawner=gemini
INFO: [WebSocket] Marking 1 events as broadcast: [1]
```

**Browser Console (immediately after server send):**
```
[WebSocket] Received message type: spawner_event
[WebSocket] spawner_event received: spawner_start gemini handler exists: true
[SpawnerEvent] spawner_start gemini {...}
```

## Diagnostic Capability

Logs now reveal exactly where events are getting lost:

1. **No "Found" log** → Events not in database or already broadcast
2. **"Found" but no "Sending"** → Loop/parsing error in server
3. **"Sending" but no client "Received"** → WebSocket disconnected
4. **"Received" but no "handler exists"** → activity-feed.html not loaded
5. **"handler exists: true" but no [SpawnerEvent]** → Exception in handler

## Files Modified

1. `/Users/shakes/DevProjects/htmlgraph/src/python/wipnote/api/main.py`
   - Lines 2145, 2148, 2182, 2192

2. `/Users/shakes/DevProjects/htmlgraph/src/python/wipnote/api/templates/dashboard.html`
   - Lines 164, 193, 198

3. `/Users/shakes/DevProjects/htmlgraph/index.html`
   - Synced from dashboard.html

## Files Created

1. `/Users/shakes/DevProjects/htmlgraph/test_spawner_broadcast.sh`
   - Interactive test script

2. `/Users/shakes/DevProjects/htmlgraph/SPAWNER_BROADCAST_DEBUG.md`
   - Comprehensive debugging guide

3. `/Users/shakes/DevProjects/htmlgraph/SPAWNER_BROADCAST_FIX_SUMMARY.md`
   - This file

## Next Steps

1. **Run test script** to identify where events are being lost
2. **Fix identified issue** based on log output
3. **Reduce logging verbosity** once working (keep error/warning logs)
4. **Test with real spawner agents** (gemini-spawner, codex-spawner)
5. **Verify UI updates** in activity feed
6. **Add error boundaries** for edge cases

## Quality Gates

- ✅ Mypy passes (type checking)
- ✅ Ruff passes (linting)
- ✅ Code formatted
- ⏳ Tests running (in background)

## Context for Future Sessions

This fix adds **observability** to the WebSocket streaming pipeline. The actual bug may still exist, but we can now identify it precisely by running the test script and examining the logs.

The problem could be at any of these points:
- Database query not finding events
- Server not sending messages
- WebSocket connection issues
- Client message handler not called
- Handler function missing or broken

With comprehensive logging, we'll know exactly which one it is within seconds of running the test.
