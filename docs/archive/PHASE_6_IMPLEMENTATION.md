# Phase 6: Dashboard UI Components - Real-Time Agent Coordination

**Status**: In Development
**Track**: `trk-71efce52`
**Features**: 3 widgets for real-time multi-agent visibility

---

## Overview

Phase 6 builds dashboard widgets that display real-time sync state across multiple agents. Using Phase 5's broadcast, reactive, presence, and sync APIs, agents can see each other's work in real-time without polling or manual refresh.

**Key Goals**:
- ✅ Real-time presence visibility: Active agents, current work, metrics (<100ms latency)
- ✅ Live sync status: Git push/pull state across multi-device deployments
- ✅ Auto-updating queries: Feature status changes reflected instantly
- ✅ Cross-session coordination: Multiple Claude Code sessions aware of each other

---

## Three Widget Subtasks

### 1️⃣ Presence Widget - Show Active Agents in Real-Time

**Feature ID**: `feat-aa1f17eb`

**Purpose**: Display all active agents and their current work with real-time updates.

**Display Elements**:
- Agent name and status (active/idle/offline)
- Current feature being worked on
- Last tool executed (Bash, Read, Write, etc.)
- Time since last activity
- Total tools executed in session
- Total cost (tokens spent)

**Real-Time Updates**:
- WebSocket listener: `WS /ws/broadcasts`
- Filters for: `event_type: "presence_update"`
- Latency: <100ms from event to UI update
- Auto-detect offline after 5 min inactivity

**Implementation**:

```javascript
// Connect to broadcast stream
const ws = new WebSocket('ws://localhost:8000/ws/broadcasts');

ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);

  // Listen for presence updates
  if (msg.type === 'presence_update') {
    const { agent_id, presence } = msg;

    // Update card with agent info
    updateAgentCard(agent_id, {
      status: presence.status,
      current_feature: presence.current_feature_id,
      last_tool: presence.last_tool_name,
      last_activity: presence.last_activity,
      tools_count: presence.total_tools_executed,
      cost: presence.total_cost_tokens
    });
  }
};
```

**Key APIs**:
- `PresenceManager.update_presence()` - Called on each event
- `PresenceManager.get_all_presence()` - Get current state
- `WS /ws/broadcasts` - Real-time events
- `GET /api/presence` - Manual poll (optional)

**Demo**: Visit `/views/presence-widget` in dashboard

---

### 2️⃣ Sync Status Widget - Display Git Sync State

**Feature ID**: `feat-9f30da4b`

**Purpose**: Show multi-device git sync status and allow manual push/pull operations.

**Display Elements**:
- Current sync status: `idle | pushing | pulling | error | success`
- Last push timestamp and files changed count
- Last pull timestamp and conflicts (if any)
- Sync configuration: push interval, pull interval, conflict strategy
- Manual push/pull buttons
- Conflict resolution mode display

**Real-Time Updates**:
- Poll `GET /api/sync/status` every 5 seconds
- Listen for `SyncResult` broadcasts on push/pull completion
- Show progress indicator when operations in-flight

**Implementation**:

```python
# Backend: Emit sync status on push/pull completion
from wipnote.api.sync_routes import sync_manager

# After push operation completes
sync_result = await sync_manager.push()
# Broadcast automatically sent to all clients

# Frontend: Listen for sync results
ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);

  if (msg.event_type === 'sync_result') {
    // Update widget with push/pull status
    updateSyncStatus({
      status: msg.payload.status,
      operation: msg.payload.operation,
      files_changed: msg.payload.files_changed,
      conflicts: msg.payload.conflicts,
      timestamp: msg.timestamp
    });
  }
};

// Manual operations
async function pushChanges() {
  const response = await fetch('/api/sync/push', {
    method: 'POST',
    headers: {
      'X-Agent-ID': agentId,
      'X-Session-ID': sessionId
    }
  });
  const result = await response.json();
  // Widget updates via broadcast
}
```

**Key APIs**:
- `GET /api/sync/status` - Current sync state
- `POST /api/sync/push` - Manual push operation
- `POST /api/sync/pull` - Manual pull operation
- `GET /api/sync/history` - Recent operations
- Broadcast events: sync status changes

**Configuration**:
```python
SyncConfig(
  push_interval_seconds=300,     # 5 min
  pull_interval_seconds=60,      # 1 min
  conflict_strategy=SyncStrategy.AUTO_MERGE,
  auto_stash=True
)
```

---

### 3️⃣ Reactive Query Widget - Auto-Refresh Feature Status

**Feature ID**: `feat-bbed2efb`

**Purpose**: Dashboard showing live-updating feature counts by status with Kanban layout.

**Display Elements**:
- Features grouped by status: `todo | in_progress | blocked | done`
- Count per status
- Total feature count
- Auto-updating without manual refresh
- Drag-drop to change status

**Real-Time Updates**:
- WebSocket listener: `WS /ws/query/features_by_status`
- Receives: `{ type: "query_update", rows: [{status, count}, ...] }`
- Latency: <100ms from status change to chart update
- Query dependencies: `*features` (invalidates on any change)

**Implementation**:

```javascript
// Subscribe to reactive query
const ws = new WebSocket('ws://localhost:8000/ws/query/features_by_status');

ws.onmessage = (event) => {
  const result = JSON.parse(event.data);

  if (result.type === 'query_update') {
    // Update Kanban columns with new counts
    result.rows.forEach(row => {
      updateKanbanColumn(row.status, row.count);
    });
  }
};

// When user drags feature to new status
async function updateFeatureStatus(featureId, newStatus) {
  const response = await fetch(
    `/api/broadcast/features/${featureId}/status`,
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Agent-ID': agentId,
        'X-Session-ID': sessionId
      },
      body: JSON.stringify({ new_status: newStatus })
    }
  );

  const result = await response.json();
  console.log(`Updated feature, notified ${result.clients_notified} clients`);
  // All dashboards auto-update via broadcast + reactive query
}
```

**Key APIs**:
- `WS /ws/query/{query_id}` - Subscribe to reactive updates
- `GET /api/query/{query_id}` - Get current results
- `POST /api/broadcast/features/{id}/status` - Update status (triggers query invalidation)
- `GET /api/queries` - List available queries

**Pre-Registered Queries**:
- `features_by_status` - Count by status (used in widget)
- `agent_workload` - Tools executed per agent
- `recent_activity` - Last 20 events
- `blocked_features` - Features with status='blocked'
- `cost_trends` - Hourly cost aggregation

---

## Architecture: How Phase 5 APIs Enable Phase 6

### Broadcast API → Presence Widget

```
Agent 1 (Claude Code Session)
  ↓ Executes tool
  ↓ PresenceManager.update_presence() called
  ↓ broadcast_event({ type: 'presence_update', agent_id: 'claude-1', ... })
  ↓ WebSocketManager broadcasts to all clients

All Connected Dashboards
  ↓ Receive presence_update event via WS /ws/broadcasts
  ↓ Render agent card with updated status
  ↓ Show real-time activity
```

**Latency**: <100ms (Tool completes → Event created → Broadcast → UI updates)

---

### Reactive API → Feature Status Widget

```
Agent 2 (Different Claude Session) Updates Feature Status
  ↓ POST /api/broadcast/features/{id}/status
  ↓ Creates broadcast_event
  ↓ ReactiveQueryManager detects dependency match (*features)
  ↓ Marks query "features_by_status" as invalidated

ReactiveQueryManager
  ↓ Re-executes cached query
  ↓ Detects results changed
  ↓ Broadcasts query_update to all subscribed clients

All Subscribed Dashboards
  ↓ Receive query_update via WS /ws/query/features_by_status
  ↓ Update Kanban columns with new counts
  ↓ Zero manual refresh needed
```

**Latency**: <100ms (Status change → Query invalidation → Re-execution → Broadcast)

---

### Sync API → Sync Status Widget

```
GitSyncManager (Background Task)
  ↓ Runs every 5 minutes: git add .wipnote → commit → push
  ↓ On push completion: SyncResult created
  ↓ Broadcasts SyncResult event to all clients

All Connected Dashboards
  ↓ Receive sync_result event
  ↓ Update widget: status → success, files_changed → 5, timestamp → now

Multi-Device Scenario
  ↓ Device 1 pushes after 5 min: auto-commit + push
  ↓ Device 2 pulls after 1 min: detects new changes, merges
  ↓ Device 3 pulls after 1 min: gets all work from devices 1+2
  ↓ All three dashboards show sync status in real-time
```

**Latency**: 60-300s (configurable sync intervals)

---

## Phase 5 Integration Details

### WebSocket Event Model

All widgets subscribe to events via WebSocket:

```javascript
// Base WebSocket connection
const ws = new WebSocket('ws://localhost:8000/ws/broadcasts');

// Message format (all broadcast events)
{
  "type": "broadcast_event",
  "event_type": "presence_update|feature_updated|sync_result|...",
  "resource_id": "feat-123|trk-456|spk-789",
  "resource_type": "feature|track|spike",
  "agent_id": "claude-1",
  "session_id": "sess-abc123",
  "payload": { /* event-specific data */ },
  "timestamp": "2025-01-14T14:50:30Z"
}
```

### Broadcasting Feature Updates

```python
# Backend: When feature status changes
from wipnote.api.broadcast import CrossSessionBroadcaster

await broadcaster.broadcast_event(
  event_type="feature_updated",
  resource_id="feat-123",
  resource_type="feature",
  agent_id="claude-1",
  payload={
    "title": "Add Presence Widget",
    "status": "in_progress",
    "description": "...",
    "tags": ["phase-6", "ui"],
    "priority": "high"
  }
)

# All connected WebSocket clients receive immediately
# Latency: <100ms from POST /api/broadcast/features/feat-123 to client
```

### Query Invalidation Pattern

```python
# When feature status changes, invalidate dependent queries
from wipnote.api.reactive import ReactiveQueryManager

query_mgr = ReactiveQueryManager()

# Register feature status change as dependency
query_mgr.record_dependency(
  resource_type="feature",
  pattern="*features"  # Wildcard: affects all feature queries
)

# Any update to features triggers:
# 1. "features_by_status" query re-execution
# 2. "agent_workload" query re-execution
# 3. Results pushed via WS /ws/query/{query_id} to subscribers
```

---

## Running the Demo

### Start the Wipnote Server

```bash
cd /Users/shakes/DevProjects/htmlgraph
uv run wipnote serve
```

The server starts on `http://localhost:8000`

### View Presence Widget Demo

```bash
# Option 1: Direct URL
open http://localhost:8000/views/presence-widget

# Option 2: Via dashboard navigation
# Open http://localhost:8000 → Look for "Phase 6" section
```

### Generate Test Presence Events

```bash
python3 << 'EOF'
from wipnote import SDK
from wipnote.api.presence import PresenceManager
from datetime import datetime

# Initialize
sdk = SDK('demo-agent')
pm = PresenceManager()

# Simulate agent activity
pm.update_presence(
  agent_id='claude-1',
  event='tool_execute',
  websocket_manager=None  # In production, pass WebSocketManager for broadcast
)

print("✅ Presence updated - check dashboard for real-time update!")
EOF
```

### Trigger Broadcast Events

```bash
python3 << 'EOF'
import asyncio
import httpx
from datetime import datetime

async def trigger_events():
  async with httpx.AsyncClient() as client:
    # Update feature status (triggers broadcast)
    response = await client.post(
      'http://localhost:8000/api/broadcast/features/feat-aa1f17eb/status',
      headers={
        'X-Agent-ID': 'claude-1',
        'X-Session-ID': 'sess-demo'
      },
      json={'new_status': 'in_progress'}
    )
    print(f"✅ Broadcast sent: {response.json()}")

asyncio.run(trigger_events())
EOF
```

---

## Testing Cross-Session Coordination

### Multi-Tab Test

1. Open dashboard in Tab A: `http://localhost:8000`
2. Open dashboard in Tab B: Same URL
3. In a third terminal, trigger presence event:
   ```bash
   python3 -c "from wipnote import SDK; SDK('test').features.create('Test Feature').save()"
   ```
4. **Observe**: Both Tab A and Tab B update simultaneously (<100ms)

### Multi-Device Test

1. Dashboard on Laptop: `http://laptop.local:8000`
2. Dashboard on Desktop: `http://desktop.local:8000`
3. Agent work on Laptop triggers presence update
4. **Observe**: Both dashboards show agent status in real-time

---

## Performance Characteristics

| Metric | Target | Implementation |
|--------|--------|-----------------|
| **Presence latency** | <100ms | WebSocket broadcast |
| **Query update latency** | <100ms | Reactive query + invalidation |
| **Sync update latency** | 60-300s | Background push/pull |
| **Max clients/session** | 10 | WebSocketManager.connect() limit |
| **Throughput** | 1000+ events/sec | Event batching (50/50ms) |
| **Memory/connection** | ~100KB | Minimal WebSocket overhead |

---

## Integration Checklist

- [x] Phase 5 Broadcast API available
- [x] Phase 5 Reactive Query API available
- [x] Phase 5 Presence Manager available
- [x] Phase 5 Sync Manager available
- [x] Presence Widget demo created
- [ ] Sync Status Widget implementation
- [ ] Reactive Query Widget implementation
- [ ] Documentation complete
- [ ] Cross-session testing complete
- [ ] Performance testing complete

---

## Next Steps

1. **Implement Sync Status Widget** (`feat-9f30da4b`)
   - Use `GET /api/sync/status` for polling
   - Listen to sync broadcasts for real-time updates
   - Add manual push/pull buttons

2. **Implement Reactive Query Widget** (`feat-bbed2efb`)
   - Subscribe to `WS /ws/query/features_by_status`
   - Build Kanban UI (todo/in_progress/blocked/done)
   - Implement drag-drop to update status

3. **Advanced Features** (Post-Phase 6)
   - Workload balancing: Use "agent_workload" query to auto-assign features
   - Cost prediction: Trending chart from "cost_trends" query
   - Conflict resolution UI: Visual merge tool for sync conflicts
   - Performance profiling: Dashboard for bottleneck detection

---

## References

- [Phase 5 Broadcast API](./BROADCAST_SYNC.md) - Cross-session feature updates
- [Phase 5 Reactive API](./PHASE_4_OFFLINE_MERGE.md) - Auto-updating queries
- [Phase 5 Presence API](../src/python/wipnote/api/presence.py) - Agent status tracking
- [Phase 5 Sync API](../src/python/wipnote/api/sync_routes.py) - Multi-device git sync
- [Dashboard Demo](../src/python/wipnote/api/static/presence-widget-demo.html) - Working example

---

## Questions?

This is Phase 6 in action:
- **Multiple Claude Code sessions** working in parallel
- **Real-time visibility** across sessions (no polling)
- **Auto-updating UI** (no manual refresh)
- **Coordinated workflows** (agents aware of each other)

All enabled by Phase 5's broadcast, reactive, presence, and sync infrastructure.
