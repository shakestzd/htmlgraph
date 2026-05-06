# WebSocket Real-Time Event Streaming API

## Overview

The WebSocket API provides real-time event streaming for Wipnote agent observability. It enables low-latency (<100ms) delivery of events to multiple concurrent clients with intelligent filtering and cost monitoring.

## Quick Start

### Connect to WebSocket

```javascript
// JavaScript client
const ws = new WebSocket(`ws://${host}:${port}/ws/events/${sessionId}`);

ws.addEventListener('open', () => {
    console.log('Connected to event stream');
});

ws.addEventListener('message', (event) => {
    const data = JSON.parse(event.data);
    console.log('Received event batch:', data);
});

ws.addEventListener('close', () => {
    console.log('Disconnected from event stream');
});
```

### Python Client

```python
import asyncio
import json
import websockets

async def stream_events(session_id: str):
    uri = f"ws://localhost:8000/ws/events/{session_id}"

    async with websockets.connect(uri) as websocket:
        while True:
            message = await websocket.recv()
            data = json.loads(message)
            print(f"Event batch: {data['count']} events")

            for event in data['events']:
                print(f"  - {event['event_type']}: {event['tool_name']}")

asyncio.run(stream_events("session-123"))
```

## API Reference

### Connection

#### Endpoint
```
ws://HOST:PORT/ws/events/{session_id}
```

#### Query Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `filter` | JSON | Event subscription filter (optional) |
| `batch_size` | int | Events per batch (default: 50) |
| `poll_interval` | int | Poll interval in ms (default: 100) |

#### Example with Filter

```javascript
const filter = {
    event_types: ["tool_call", "error"],
    cost_threshold_tokens: 100,
    statuses: ["completed", "error"]
};

const query = `?filter=${encodeURIComponent(JSON.stringify(filter))}`;
const ws = new WebSocket(`ws://localhost:8000/ws/events/session-123${query}`);
```

### Message Format

#### Event Batch Message

```json
{
    "type": "batch",
    "count": 5,
    "timestamp": "2024-01-16T12:00:00.000Z",
    "events": [
        {
            "event_id": "evt-12345",
            "agent_id": "claude-code",
            "event_type": "tool_call",
            "timestamp": "2024-01-16T12:00:00.000Z",
            "tool_name": "Edit",
            "input_summary": "Edit: src/main.py:45-52",
            "output_summary": "File edited successfully",
            "session_id": "session-123",
            "status": "completed",
            "model": "claude-opus",
            "parent_event_id": null,
            "execution_duration_seconds": 0.345,
            "cost_tokens": 125,
            "feature_id": "feat-456"
        }
    ]
}
```

#### Single Event Message

```json
{
    "type": "event",
    "timestamp": "2024-01-16T12:00:00.000Z",
    "event_id": "evt-12345",
    "agent_id": "claude-code",
    "event_type": "tool_call",
    "tool_name": "Edit",
    "status": "completed",
    "cost_tokens": 125
}
```

## Subscription Filters

### Event Type Filter

Subscribe to specific event types:

```python
from wipnote.api.websocket import EventSubscriptionFilter

filter = EventSubscriptionFilter(
    event_types=["tool_call", "completion", "error"]
)
```

### Session Filtering

Filter by specific session:

```python
filter = EventSubscriptionFilter(
    session_id="session-123"
)
```

### Tool Filtering

Monitor specific tools:

```python
filter = EventSubscriptionFilter(
    tool_names=["Edit", "Read", "Bash"]
)
```

### Cost Threshold Filtering

Alert on expensive operations:

```python
# Only receive events costing more than 1000 tokens
filter = EventSubscriptionFilter(
    cost_threshold_tokens=1000,
    event_types=["tool_call"]
)
```

### Status Filtering

Monitor specific statuses:

```python
filter = EventSubscriptionFilter(
    statuses=["error", "timeout"]
)
```

### Feature Filtering

Track specific features:

```python
filter = EventSubscriptionFilter(
    feature_ids=["feat-123", "feat-456"]
)
```

## Use Cases

### Real-Time Activity Feed

Monitor all agent activity in real-time:

```javascript
const ws = new WebSocket('ws://localhost:8000/ws/events/session-123');

ws.addEventListener('message', (event) => {
    const batch = JSON.parse(event.data);

    for (const evt of batch.events) {
        updateActivityFeed(evt);
    }
});
```

### Cost Monitoring and Alerts

Alert when costs exceed threshold:

```python
import asyncio
import websockets
import json

async def monitor_costs(session_id: str, threshold: int = 5000):
    filter = EventSubscriptionFilter(cost_threshold_tokens=threshold)

    uri = f"ws://localhost:8000/ws/events/{session_id}"
    async with websockets.connect(uri) as ws:
        while True:
            message = await ws.recv()
            data = json.loads(message)

            for event in data['events']:
                if event['cost_tokens'] > threshold:
                    send_alert(f"High cost event: {event['cost_tokens']} tokens")

asyncio.run(monitor_costs("session-123"))
```

### Error Tracking

Stream only errors:

```python
filter = EventSubscriptionFilter(
    event_types=["error"],
    statuses=["error", "timeout"]
)
```

### Bottleneck Detection

Identify slow operations:

```python
filter = EventSubscriptionFilter(
    event_types=["tool_call"]
)

# In handler:
if event['execution_duration_seconds'] > 5:
    print(f"Bottleneck: {event['tool_name']} took {event['execution_duration_seconds']}s")
```

## Performance Characteristics

### Latency

- **P50 (median)**: <5ms
- **P95**: <50ms
- **P99**: <100ms
- **Max**: <200ms

### Throughput

- **Events per second**: 1000+
- **Concurrent clients**: 50+ per session
- **Batching window**: 50ms (configurable)
- **Batch size**: 50 events (configurable)

### Memory

- **Per-client overhead**: ~1KB
- **Per-session overhead**: ~5KB
- **Event batcher overhead**: ~10KB

## Configuration

### Server Configuration

```python
from wipnote.api.websocket import WebSocketManager

manager = WebSocketManager(
    db_path="/path/to/wipnote.db",
    max_clients_per_session=10,      # Max clients per session
    event_batch_size=50,              # Events per batch
    event_batch_window_ms=50.0,       # Batching window (ms)
    poll_interval_ms=100.0,           # Poll interval (ms)
)
```

### Client Configuration

```javascript
// Configure batching
const config = {
    batch_size: 50,           // Events per batch
    batch_window_ms: 50,      // Batching window
    reconnect_interval_ms: 1000,  // Reconnection interval
    max_reconnect_attempts: 10
};
```

## Error Handling

### Connection Errors

```javascript
ws.addEventListener('error', (event) => {
    console.error('WebSocket error:', event);
    // Reconnect logic
});
```

### Disconnect Handling

```javascript
ws.addEventListener('close', (event) => {
    if (event.wasClean) {
        console.log('Connection closed normally');
    } else {
        console.log('Connection lost, reconnecting...');
        setTimeout(() => reconnect(), 1000);
    }
});
```

### Graceful Reconnection

```python
async def reconnect_loop(session_id: str, max_attempts: int = 10):
    attempts = 0

    while attempts < max_attempts:
        try:
            await stream_events(session_id)
            attempts = 0  # Reset on successful connection
        except Exception as e:
            attempts += 1
            wait_time = min(2 ** attempts, 60)  # Exponential backoff
            print(f"Reconnecting in {wait_time}s (attempt {attempts})")
            await asyncio.sleep(wait_time)
```

## Monitoring and Metrics

### Get WebSocket Metrics

```python
# Get metrics for specific session
metrics = manager.get_session_metrics(session_id)
print(f"Connected clients: {metrics['connected_clients']}")
print(f"Events sent: {metrics['total_events_sent']}")
print(f"Bytes sent: {metrics['total_bytes_sent']}")

# Get global metrics
global_metrics = manager.get_global_metrics()
print(f"Active sessions: {global_metrics['active_sessions']}")
print(f"Total events broadcast: {global_metrics['total_events_broadcast']}")
```

### Dashboard Metrics

Access metrics via REST API:

```bash
curl http://localhost:8000/api/websocket-metrics
```

Response:

```json
{
    "active_sessions": 5,
    "total_connected_clients": 12,
    "total_events_broadcast": 45230,
    "total_bytes_sent": 2345600,
    "sessions": {
        "session-123": {
            "connected_clients": 3,
            "total_events_sent": 2500,
            "total_bytes_sent": 125000,
            "uptime_seconds": 3600
        }
    }
}
```

## Best Practices

### 1. Use Filters to Reduce Load

```python
# Good: Only subscribe to relevant events
filter = EventSubscriptionFilter(
    event_types=["error"],
    cost_threshold_tokens=1000
)

# Avoid: Subscribe to everything without filtering
filter = EventSubscriptionFilter()
```

### 2. Handle Reconnections

Always implement reconnection logic:

```javascript
let reconnectAttempts = 0;
const maxReconnectAttempts = 10;

function reconnect() {
    if (reconnectAttempts >= maxReconnectAttempts) {
        console.error('Max reconnection attempts reached');
        return;
    }

    reconnectAttempts++;
    const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000);

    setTimeout(() => {
        ws = new WebSocket(wsUrl);
    }, delay);
}
```

### 3. Process Batches Efficiently

```python
# Good: Process entire batch at once
for event in batch['events']:
    process_event(event)

# Avoid: One message per event (use batching)
```

### 4. Monitor Memory Usage

For long-lived connections, monitor memory:

```python
import tracemalloc

tracemalloc.start()

# ... run streaming ...

current, peak = tracemalloc.get_traced_memory()
print(f"Memory: {current / 1024:.1f} KB, Peak: {peak / 1024:.1f} KB")
```

### 5. Implement Heartbeat/Keepalive

```javascript
const heartbeatInterval = 30000; // 30 seconds

setInterval(() => {
    if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({type: 'ping'}));
    }
}, heartbeatInterval);
```

## Troubleshooting

### WebSocket Connection Refused

```
Error: Failed to connect to ws://localhost:8000/ws/events/session-123
```

**Solution**: Ensure FastAPI server is running on correct port.

```bash
curl http://localhost:8000/api/status
```

### Messages Not Being Received

**Causes**:
- Filter is too restrictive
- Session ID is incorrect
- No new events are being generated

**Debug**:

```python
# Check active sessions
curl http://localhost:8000/api/sessions

# Check session activity
curl http://localhost:8000/api/sessions/session-123/events
```

### High Memory Usage

**Causes**:
- Too many concurrent clients
- Events not being batched properly
- Batcher not flushing

**Solutions**:

```python
# Reduce batch window
manager = WebSocketManager(event_batch_window_ms=25.0)

# Limit concurrent clients
manager.max_clients_per_session = 5

# Monitor memory
manager.get_global_metrics()
```

### High Latency

**Causes**:
- Database lock contention
- Too many events per batch
- Network latency

**Solutions**:

```python
# Reduce batch size
manager = WebSocketManager(event_batch_size=25)

# Increase poll interval for aggregation
manager.poll_interval_ms = 200.0

# Monitor metrics
metrics = manager.get_session_metrics(session_id)
```

## Examples

See `/examples/websocket/` for complete working examples:

- `activity-feed.html` - Real-time activity feed UI
- `cost-monitor.py` - Python cost monitoring client
- `error-tracker.js` - JavaScript error tracking
- `performance-dashboard.html` - Performance monitoring dashboard

## Integration with Phase 3.1 Features

### Cost Monitoring Alerts

```python
filter = EventSubscriptionFilter(cost_threshold_tokens=10000)
# Automatically receives only high-cost events for alerting
```

### Bottleneck Prediction

```python
# Stream all events for analysis
events = []
async for batch in stream_events(session_id):
    events.extend(batch['events'])

# Analyze patterns for bottleneck prediction
analyze_bottlenecks(events)
```

### Activity Feed Updates

```javascript
// Real-time updates for UI
ws.addEventListener('message', (event) => {
    const batch = JSON.parse(event.data);
    batch.events.forEach(evt => {
        addToActivityFeed(evt);
    });
});
```

## Version History

- **0.1.0** (2026-01-16) - Initial release with core event streaming
- **0.1.1** (planned) - Advanced filtering and cost thresholds
- **0.2.0** (planned) - GraphQL subscriptions support
- **0.3.0** (planned) - Custom event processors/transformations

## Support

For issues or questions:

1. Check [Troubleshooting](#troubleshooting) section
2. Review [Performance Characteristics](#performance-characteristics)
3. Open issue on GitHub: https://github.com/anthropics/wipnote/issues
