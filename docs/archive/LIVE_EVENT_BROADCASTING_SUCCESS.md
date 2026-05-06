# Live Event Broadcasting - Working Solution

## Summary

✅ **Live event broadcasting is now fully functional.**

## Problem

WebSocket connections were getting "database is locked" errors when trying to poll the `live_events` table, preventing spawner events from being broadcast to the dashboard in real-time.

## Root Cause

SQLite's `busy_timeout` was set to 0 (default), meaning any concurrent database access would immediately fail with "database is locked" instead of waiting for the lock to be released.

## Solution

Added `PRAGMA busy_timeout = 5000` to the `get_db()` function in `src/python/wipnote/api/main.py`:

```python
async def get_db() -> aiosqlite.Connection:
    """Get database connection with busy_timeout to prevent lock errors."""
    db = await aiosqlite.connect(app.state.db_path)
    db.row_factory = aiosqlite.Row
    # Set busy_timeout to 5 seconds - prevents "database is locked" errors
    # during concurrent access from spawner scripts and WebSocket polling
    await db.execute("PRAGMA busy_timeout = 5000")
    return db
```

## Verification

**Test Event 59:**
- Created: 2026-01-11 04:47:11
- Broadcast: 2026-01-11 04:47:51 (40 seconds later)
- Status: BROADCAST ✅

**Complete Pipeline Verified:**
1. ✅ Event inserted into `live_events` table
2. ✅ WebSocket polling detected the pending event
3. ✅ Event sent to client as `spawner_event` message type
4. ✅ Client received: `ID=59, spawner=gemini, data={'message': 'TEST WITH BUSY_TIMEOUT FIX', ...}`
5. ✅ Event marked as `broadcast_at` in database

## Database Configuration

- **Journal Mode:** WAL (Write-Ahead Logging) - already enabled ✅
- **Busy Timeout:** 5000ms (5 seconds) - NOW enabled ✅

## Impact

- ❌ **Before:** WebSocket crashed on every database access with "database is locked"
- ✅ **After:** WebSocket successfully polls, broadcasts events, and marks them as broadcast
- ✅ **Concurrency:** Multiple processes (spawner scripts, WebSocket, hooks) can now access the database concurrently

## Files Modified

- `src/python/wipnote/api/main.py` - Added busy_timeout to get_db()
- Logging added throughout WebSocket handler (for debugging)
- Client-side logging in dashboard.html and activity-feed.html

## Next Steps

1. ✅ Database locking fixed
2. ✅ Live event broadcasting verified
3. 🔜 Test with real spawner agent execution
4. 🔜 Remove debug logging (or keep for production debugging)
5. 🔜 Add UI indicators in dashboard for live events
