# Spawner Event Broadcasting Debug Guide

## Changes Made

### Server-Side Logging (main.py)
Added three logging points in the WebSocket streaming handler:

1. **Line 2148**: When live_events are found
   ```python
   logger.info(f"[WebSocket] Found {len(live_rows)} pending live_events to broadcast")
   ```

2. **Line 2182**: Before sending each spawner_event
   ```python
   logger.info(f"[WebSocket] Sending spawner_event: id={live_id}, type={event_type}, spawner={spawner_type}")
   ```

3. **Line 2192**: When marking events as broadcast
   ```python
   logger.info(f"[WebSocket] Marking {len(broadcast_ids)} events as broadcast: {broadcast_ids}")
   ```

### Client-Side Logging (dashboard.html)

1. **Line 164**: Log all incoming WebSocket message types
   ```javascript
   console.log('[WebSocket] Received message type:', data.type);
   ```

2. **Line 193**: Enhanced spawner_event logging
   ```javascript
   console.log('[WebSocket] spawner_event received:', data.event_type, data.spawner_type, 'handler exists:', typeof window.handleSpawnerEvent === 'function');
   ```

3. **Line 198**: Warning if handler not available
   ```javascript
   console.warn('[WebSocket] handleSpawnerEvent not available, spawner event dropped:', data.event_type, data.spawner_type);
   ```

## Testing Procedure

### 1. Start Dashboard Server
```bash
# Terminal 1: Start server with logging
uv run wipnote serve
```

Watch for startup logs and WebSocket connection messages.

### 2. Open Dashboard in Browser
```bash
# Open in browser
open http://localhost:8888
```

Open browser console (F12) and watch for WebSocket connection:
```
WebSocket connected for real-time events
```

### 3. Insert Test Event

**Option A: Using test script**
```bash
# Terminal 2: Run automated test
./test_spawner_broadcast.sh
```

**Option B: Manual insert**
```bash
sqlite3 .wipnote/index.sqlite << EOF
INSERT INTO live_events (event_type, event_data, spawner_type, parent_event_id, session_id)
VALUES (
    'spawner_start',
    '{"spawner_type": "gemini", "prompt_preview": "Manual test", "status": "started"}',
    'gemini',
    'test-parent-123',
    'test-session-456'
);
EOF
```

### 4. Expected Logs

**Server Terminal (within 1-2 seconds):**
```
INFO:     [WebSocket] Found 1 pending live_events to broadcast
INFO:     [WebSocket] Sending spawner_event: id=1, type=spawner_start, spawner=gemini
INFO:     [WebSocket] Marking 1 events as broadcast: [1]
```

**Browser Console (immediately after server sends):**
```
[WebSocket] Received message type: spawner_event
[WebSocket] spawner_event received: spawner_start gemini handler exists: true
[SpawnerEvent] spawner_start gemini {spawner_type: "gemini", prompt_preview: "Manual test", status: "started"}
```

### 5. Verify Event Was Marked Broadcast
```bash
sqlite3 .wipnote/index.sqlite "SELECT id, event_type, broadcast_at FROM live_events ORDER BY id DESC LIMIT 5;"
```

Expected: `broadcast_at` should have a timestamp (not NULL).

## Troubleshooting

### Issue: No server logs appear

**Symptom**: No `[WebSocket] Found X pending live_events` in server terminal

**Possible causes**:
1. WebSocket not polling (client disconnected)
2. Events already marked as broadcast
3. Database query failing silently

**Debug**:
```bash
# Check if events exist and are pending
sqlite3 .wipnote/index.sqlite "SELECT * FROM live_events WHERE broadcast_at IS NULL;"

# Check WebSocket connections
# Should see connection in server logs when browser opens dashboard
```

### Issue: Server logs show "Found" but no "Sending"

**Symptom**: See `Found X pending live_events` but no `Sending spawner_event`

**Possible causes**:
1. Loop not iterating (should not happen with list())
2. Exception in JSON parsing
3. Logic error in event processing

**Debug**: Check server for exceptions or traceback.

### Issue: Server sends but client doesn't receive

**Symptom**: Server shows `Sending spawner_event` but browser console has no logs

**Possible causes**:
1. WebSocket disconnected (check browser console for disconnect message)
2. Message type mismatch (typo in type check)
3. JSON parsing error on client

**Debug**:
```javascript
// In browser console, check WebSocket state
// Should be open and connected
```

### Issue: Client receives but handler not called

**Symptom**: Browser shows `Received message type: spawner_event` but no `[SpawnerEvent]`

**Possible causes**:
1. `window.handleSpawnerEvent` not defined (activity-feed.html not loaded)
2. Handler function has a different name
3. Script loading order issue

**Debug**:
```javascript
// In browser console
typeof window.handleSpawnerEvent
// Should return: "function"

// If undefined, check if activity-feed.html is included
```

### Issue: Handler exists but no [SpawnerEvent] log

**Symptom**: Browser shows `handler exists: true` but no `[SpawnerEvent]` from handler

**Possible causes**:
1. Exception in `handleSpawnerEvent()` function
2. Console.log statement not executing
3. Event type not matching switch cases

**Debug**:
```javascript
// Check browser console for errors
// Try calling handler manually:
window.handleSpawnerEvent({
    event_type: 'spawner_start',
    spawner_type: 'gemini',
    data: {test: true}
});
```

## Database Cleanup

To reset for fresh testing:
```bash
# Clear all live_events
sqlite3 .wipnote/index.sqlite "DELETE FROM live_events;"

# Or just reset broadcast status
sqlite3 .wipnote/index.sqlite "UPDATE live_events SET broadcast_at = NULL;"
```

## Next Steps After Debugging

Once logs show the full pipeline working:
1. Remove or reduce verbose logging (keep errors/warnings)
2. Test with real spawner agents (gemini-spawner, codex-spawner)
3. Verify UI updates correctly in activity feed
4. Add error boundaries for edge cases
5. Performance test with multiple rapid events

## File Locations

- Server code: `/Users/shakes/DevProjects/htmlgraph/src/python/wipnote/api/main.py` (lines 2134-2200)
- Client WebSocket handler: `/Users/shakes/DevProjects/htmlgraph/src/python/wipnote/api/templates/dashboard.html` (lines 161-204)
- Spawner event handler: `/Users/shakes/DevProjects/htmlgraph/src/python/wipnote/api/templates/partials/activity-feed.html` (lines 818-842)
- Test script: `/Users/shakes/DevProjects/htmlgraph/test_spawner_broadcast.sh`
