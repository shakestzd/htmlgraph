# Quick Start: Debug Spawner Event Broadcasting

## TL;DR

Run this to test if spawner events are being broadcast:

```bash
# Terminal 1: Start server
uv run wipnote serve

# Terminal 2: Run test (while dashboard is open in browser with console visible)
./test_spawner_broadcast.sh
```

Watch for logs in **both** server terminal and browser console.

## What Was Fixed

Added comprehensive logging to trace spawner events through the entire pipeline:
- ✅ Server logs when events are found
- ✅ Server logs when sending to client
- ✅ Server logs when marking as broadcast
- ✅ Client logs when receiving message
- ✅ Client logs when calling handler
- ✅ Handler logs when processing event

## Expected Output

### Server Terminal
```
INFO: [WebSocket] Found 1 pending live_events to broadcast
INFO: [WebSocket] Sending spawner_event: id=1, type=spawner_start, spawner=gemini
INFO: [WebSocket] Marking 1 events as broadcast: [1]
```

### Browser Console (F12)
```
[WebSocket] Received message type: spawner_event
[WebSocket] spawner_event received: spawner_start gemini handler exists: true
[SpawnerEvent] spawner_start gemini {spawner_type: "gemini", ...}
```

## If Logs Don't Appear

See `SPAWNER_BROADCAST_DEBUG.md` for detailed troubleshooting.

Quick checks:
1. **No server logs** → Check WebSocket is connected (browser should show "WebSocket connected")
2. **Server logs but no client logs** → Check browser console for errors
3. **Client logs but no [SpawnerEvent]** → Check if handler exists: `typeof window.handleSpawnerEvent`

## Files Changed

- `src/python/wipnote/api/main.py` - Server-side logging
- `src/python/wipnote/api/templates/dashboard.html` - Client-side logging
- `index.html` - Synced from dashboard.html

## Test Status

✅ 2875 tests passed
✅ Mypy type checking passed
✅ Ruff linting passed

## Next Steps

1. Run test script to identify where events are lost
2. Fix the identified issue
3. Reduce logging verbosity (keep error/warning logs)
4. Test with real spawner agents
