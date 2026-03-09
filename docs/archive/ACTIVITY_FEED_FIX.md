# Activity Feed Nesting Fix

## Problem Summary

The HtmlGraph dashboard had a critical bug where real-time events streamed via WebSocket displayed with incorrect nesting. On page reload, the proper hierarchical structure appeared, but during live streaming, events appeared flat.

### Root Cause

**Server-side template** (`src/python/htmlgraph/api/templates/partials/activity-feed.html`):
- Uses nested `.conversation-turn` containers
- Creates `.child-event-row` divs with recursive `.child-children` containers
- Properly indented with tree structure (├─, └─ connectors)

**WebSocket handler** (`src/python/htmlgraph/dashboard.html`):
- Was trying to insert flat `<tr>` table rows into a `<tbody>`
- Completely incompatible DOM structure
- No re-parenting logic for orphaned children
- No support for nested hierarchy

## Solution

Rewrote `insertNewEventIntoActivityFeed()` function in `dashboard.html` to:

1. **Match template DOM structure** - Insert into `.conversation-turn` containers with nested `.child-event-row` divs
2. **Handle UserQuery events** - Create new top-level conversation turns
3. **Handle child events** - Insert into parent's `.child-children` container
4. **Handle nested children** - Support arbitrary depth nesting
5. **Handle orphaned children** - Store children that arrive before parents, adopt when parent arrives
6. **Update stats badges** - Recalculate tool count and duration as children arrive
7. **Maintain timestamp ordering** - Insert newest events first within siblings

## Key Changes

### File: `src/python/htmlgraph/dashboard.html`

**Function: `insertNewEventIntoActivityFeed(eventData)`**
- Lines 233-339: Completely rewritten
- Now creates proper DOM structure matching server template
- Supports nested hierarchy with arbitrary depth
- Handles orphaned children gracefully

**New Functions:**
- `storeOrphanedEvent(eventData)` - Cache orphaned children
- `adoptOrphanedChildren(parentEventId)` - Re-parent when parent arrives
- `findConversationTurnForEvent(eventId)` - Locate turn container for event
- `createConversationTurnHTML(eventData)` - Generate UserQuery turn HTML
- `createChildEventHTML(eventData, depth)` - Generate child event HTML with correct depth
- `updateTurnStats(turnContainer)` - Recalculate turn statistics
- `highlightElement(element)` - Renamed from `highlightRow()` to work with divs

**Removed Functions:**
- `createActivityRowHTML(eventData)` - No longer needed (was generating table rows)

### File: `src/python/htmlgraph/api/main.py`

**New Test Endpoint:**
- `POST /api/test-event` - Inject test events during development
- Inserts events into `agent_events` table
- Events are picked up by WebSocket polling and broadcast to clients

## DOM Structure

### Conversation Turn (UserQuery)
```html
<div class="conversation-turn" data-turn-id="evt-123">
  <div class="userquery-parent" onclick="toggleConversationTurn('evt-123')">
    <span class="expand-toggle-turn">▶</span>
    <div class="prompt-section">
      <span class="prompt-text">User's question here...</span>
    </div>
    <div class="turn-stats">
      <span class="stat-badge tool-count">5</span>
      <span class="stat-badge duration">2.34s</span>
    </div>
  </div>

  <div class="turn-children collapsed" id="children-evt-123">
    <!-- Child events go here -->
  </div>
</div>
```

### Child Event (Top-level)
```html
<div class="child-event-row depth-0" data-event-id="evt-456" data-depth="0">
  <span class="tree-connector">├─</span>
  <span class="child-tool-name">Bash</span>
  <span class="child-summary">ls -la</span>
  <span class="child-agent-badge">claude-code</span>
  <span class="child-duration">0.50s</span>
  <span class="child-timestamp">2026-02-11 10:30:45</span>
</div>
```

### Nested Child Event
```html
<div class="child-event-row depth-0 has-nested-children" data-event-id="evt-456" onclick="toggleChildEvent('evt-456', event)">
  <span class="expand-toggle-child">▶</span>
  <span class="child-tool-name">Task</span>
  <span class="child-summary">Background task</span>
  <span class="nested-count-badge">3</span>
  ...
</div>

<div class="child-children collapsed" id="children-child-evt-456" data-depth="1">
  <!-- Nested children at depth 1 -->
  <div class="child-event-row depth-1" data-event-id="evt-789">
    ...
  </div>
</div>
```

## Testing

### Automated Test Harness

Open `test_activity_feed_fix.html` in a browser and follow these steps:

1. **Start the dashboard server:**
   ```bash
   uv run htmlgraph serve
   ```

2. **Open the dashboard in one browser tab:**
   ```
   http://localhost:8765
   ```

3. **Open test harness in another tab:**
   ```
   file:///path/to/htmlgraph/test_activity_feed_fix.html
   ```

4. **Run tests in sequence:**
   - Click "1. Send UserQuery Event" - Should create new conversation turn at top
   - Click "2. Send Child Event (Bash)" - Should nest under UserQuery
   - Click "3. Send Nested Child Event" - Should nest under Bash (depth 1)
   - Click "4. Send Orphaned Child" - Creates child before parent exists
   - Click "5. Send Parent" - Parent arrives, orphan should be adopted

### Manual Testing

1. **Start dashboard:**
   ```bash
   uv run htmlgraph serve
   ```

2. **In another terminal, trigger real events:**
   ```bash
   # This will create events tracked by hooks
   ls -la
   uv run python -c "print('test')"
   ```

3. **Watch the dashboard:**
   - New conversation turn appears at top (your prompt)
   - Child events (Bash, Read, etc.) appear nested underneath
   - Expand/collapse arrows work correctly
   - Stats badges update in real-time
   - Nested children (if any) show correct indentation

### Visual Checks

✅ **Correct behavior:**
- UserQuery creates new conversation turn at top
- Child events nest under parent with tree connector (├─)
- Last child shows (└─) connector
- Events with nested children show (▶) toggle arrow
- Nested count badge shows number of children
- Stats badges (tool count, duration) update as events arrive
- Timestamps convert to local timezone
- Collapsible sections expand/collapse smoothly

❌ **Incorrect behavior (before fix):**
- All events appear flat (no nesting)
- No hierarchy visible during streaming
- Stats don't update in real-time
- Orphaned children never get adopted

## Architecture Notes

### Why Two Different Structures?

The template uses a nested div structure because:
- HTML/CSS best practice for hierarchical data
- Easier to style and animate
- Supports arbitrary nesting depth
- Semantic markup (divs with meaningful classes)

Table rows (`<tr>`) were incompatible because:
- Tables don't support nested structures well
- Can't have divs inside tbody directly
- Hard to style collapsible nested sections
- Limited depth support

### Event Flow

1. **Agent performs action** → Hook records to database
2. **WebSocket polls database** → Finds new events (once per second)
3. **WebSocket broadcasts** → Sends JSON to all connected clients
4. **Client receives event** → `insertNewEventIntoActivityFeed()` called
5. **DOM insertion** → Event inserted at correct position in hierarchy
6. **Highlight animation** → Brief flash to show new event
7. **Timestamp conversion** → UTC → Local timezone

### Orphaned Children Handling

**Problem:** WebSocket events arrive in any order. Child event might arrive before parent.

**Solution:** Orphan cache
```javascript
const orphanedEvents = new Map(); // parentId → [child1, child2, ...]

// When child arrives before parent:
storeOrphanedEvent(eventData);

// When parent finally arrives:
adoptOrphanedChildren(parentEventId);
// → Re-inserts all cached children under correct parent
```

## Performance Considerations

- **Memory management:** Keep only last 50 conversation turns (removes oldest)
- **Timestamp conversion:** Only converts new elements (not entire page)
- **Highlight animation:** 2-second CSS animation, then class removed
- **Orphan cleanup:** Orphans adopted immediately when parent arrives
- **Stats calculation:** Only recalculates affected turn (not all turns)

## Future Enhancements

1. **Incremental stats updates** - Instead of recalculating from DOM, track in memory
2. **Virtual scrolling** - Only render visible conversation turns
3. **Search/filter** - Filter events by tool, agent, or time range
4. **Export** - Export conversation turn as JSON/HTML
5. **Replay** - Replay events in slow motion for debugging

## Files Modified

1. `src/python/htmlgraph/dashboard.html` - WebSocket handler rewrite
2. `src/python/htmlgraph/api/main.py` - Added test endpoint
3. `index.html` - Synced from dashboard.html
4. `test_activity_feed_fix.html` - New test harness (not part of package)

## Commit Message

```
fix: rewrite WebSocket activity feed insertion to match template DOM structure

**Problem:**
- Real-time events showed flat (no nesting) during streaming
- On page reload, proper hierarchy appeared
- WebSocket handler used incompatible table rows (<tr>) instead of divs

**Solution:**
- Rewrote insertNewEventIntoActivityFeed() to generate nested div structure
- Matches server template (conversation-turn → child-event-row → child-children)
- Added orphaned children handling (cache + adopt when parent arrives)
- Added stats badge updates (tool count, duration)
- Added test endpoint POST /api/test-event for development testing

**Changes:**
- dashboard.html: Complete rewrite of WebSocket insertion logic
- api/main.py: New test endpoint for event injection
- New functions: createConversationTurnHTML, createChildEventHTML, updateTurnStats
- Removed obsolete: createActivityRowHTML (table row generation)

**Testing:**
- Manual: Start dashboard, trigger real events, verify nesting
- Automated: Use test_activity_feed_fix.html test harness
- Verify: Orphan adoption, stats updates, expand/collapse, timestamps

Closes #[issue-number]
```
