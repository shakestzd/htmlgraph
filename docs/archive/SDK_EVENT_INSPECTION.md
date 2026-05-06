# SDK Event Inspection Guide

**IMPORTANT: Always use the SDK to inspect Wipnote data. Never read `.wipnote/` files directly.**

## Why Use the SDK?

❌ **WRONG** - Reading files directly:
```python
# DON'T DO THIS
with open('.wipnote/events/session-123.jsonl') as f:
    for line in f:
        evt = json.loads(line)
        print(evt)
```

✅ **RIGHT** - Using SDK:
```python
from wipnote import SDK

sdk = SDK(agent='claude-code')
session = sdk.sessions.get('session-123')
events = session.get_events(limit=10)
```

## Quick Start

```python
from wipnote import SDK
from wipnote.session_manager import SessionManager

sdk = SDK(agent='claude-code')
sm = SessionManager()

# Get current active session
session = sm.get_active_session(agent='claude-code')
print(f"Session: {session.id}")
print(f"Event Count: {session.event_count}")
```

## Event Inspection Methods

### 1. Get Recent Events

```python
# Get last 10 events
recent = session.get_events(limit=10, offset=session.event_count - 10)

for evt in recent:
    print(f"{evt['event_id']}: {evt['tool']} - {evt['summary']}")
```

### 2. Query Events by Tool

```python
# Get all Bash events (newest first)
bash_events = session.query_events(tool='Bash', limit=20)

# Get all Edit events
edit_events = session.query_events(tool='Edit')
```

### 3. Query Events by Feature

```python
# Get all events attributed to a specific feature
feature_events = session.query_events(feature_id='feat-123', limit=50)

for evt in feature_events:
    drift = evt.get('drift_score', 'N/A')
    print(f"{evt['tool']}: {evt['summary'][:50]} (drift={drift})")
```

### 4. Query Events by Time

```python
from datetime import datetime, timedelta

# Get events from last hour
one_hour_ago = datetime.now() - timedelta(hours=1)
recent_events = session.query_events(since=one_hour_ago)

# Get events since specific timestamp
since_timestamp = "2025-12-22T06:00:00"
events_since = session.query_events(since=since_timestamp, limit=100)
```

### 5. Get Event Statistics

```python
# Get comprehensive statistics
stats = session.event_stats()

print(f"Total Events: {stats['total_events']}")
print(f"Tools Used: {stats['tools_used']}")
print(f"Features Worked: {stats['features_worked']}")

# Top tools
print("\nTop Tools:")
for tool, count in sorted(stats['by_tool'].items(), key=lambda x: x[1], reverse=True)[:5]:
    print(f"  {tool}: {count} events")

# Features worked on
print("\nFeatures Worked:")
for feature, count in stats['by_feature'].items():
    print(f"  {feature}: {count} events")
```

## Complete Example: Session Verification

Here's how to properly verify session management and event tracking:

```python
from wipnote import SDK
from wipnote.session_manager import SessionManager

sdk = SDK(agent='claude-code')
sm = SessionManager()

# Get current session
session = sm.get_active_session(agent='claude-code')

print(f"📊 Session Analysis: {session.id}")
print("=" * 70)

# 1. Recent events (last 10)
print("\n🕐 Recent Events:")
recent = session.get_events(limit=10, offset=session.event_count - 10)
for evt in recent:
    feature = evt.get('feature_id') or 'None'
    print(f"  {evt['event_id'][:20]:20} | {evt['tool']:12} | {feature:20}")

# 2. Check if using hash-based IDs
print("\n🔑 Event ID Format Check:")
if recent:
    latest_id = recent[-1]['event_id']
    if latest_id.startswith('evt-'):
        print(f"  ✅ Using hash-based IDs: {latest_id}")
    else:
        print(f"  ⚠️  Using old format: {latest_id}")

# 3. Feature attribution check
print("\n🎯 Feature Attribution:")
feature_events = [e for e in recent if e.get('feature_id')]
if feature_events:
    print(f"  ✅ {len(feature_events)}/{len(recent)} events have feature attribution")
else:
    print(f"  ℹ️  No features active during recent events")

# 4. Event statistics
print("\n📈 Session Statistics:")
stats = session.event_stats()
print(f"  Total Events: {stats['total_events']}")
print(f"  Tools Used: {stats['tools_used']}")
print(f"  Features Worked: {stats['features_worked']}")

print("\n  Top 5 Tools:")
for tool, count in sorted(stats['by_tool'].items(), key=lambda x: x[1], reverse=True)[:5]:
    print(f"    {tool:20}: {count:4} events")
```

## Best Practices

1. **Always use SDK** - Never read `.wipnote/` files directly
2. **Use pagination** - For large sessions, use `limit` and `offset`
3. **Filter early** - Use `query_events()` filters to reduce data
4. **Check attribution** - Use `feature_id` filter to verify attribution
5. **Cache stats** - Event statistics are expensive, cache them

## Common Use Cases

### Verify Hash-Based IDs

```python
session = sdk.sessions.get('session-123')
recent = session.get_events(limit=1, offset=session.event_count - 1)
latest_id = recent[0]['event_id']
is_hash_based = latest_id.startswith('evt-')
print(f"Hash-based IDs: {is_hash_based}")
```

### Check Feature Attribution Rate

```python
session = sdk.sessions.get('session-123')
recent = session.get_events(limit=100)
attributed = [e for e in recent if e.get('feature_id')]
rate = len(attributed) / len(recent) * 100
print(f"Attribution Rate: {rate:.1f}%")
```

### Find Unattributed Work

```python
session = sdk.sessions.get('session-123')
unattributed = session.query_events(feature_id=None, limit=50)
print(f"Found {len(unattributed)} unattributed events")
```

### Analyze Tool Usage Patterns

```python
session = sdk.sessions.get('session-123')
stats = session.event_stats()

# Tools with most events
top_tools = sorted(stats['by_tool'].items(), key=lambda x: x[1], reverse=True)[:10]
for tool, count in top_tools:
    percentage = count / stats['total_events'] * 100
    print(f"{tool:20}: {count:4} ({percentage:5.1f}%)")
```

## Migration Guide

If you have code that reads files directly, here's how to migrate:

### Before (❌ Wrong):
```python
import json

# Reading JSONL directly
with open('.wipnote/events/session-123.jsonl') as f:
    events = [json.loads(line) for line in f if line.strip()]

# Getting last 10
recent = events[-10:]
```

### After (✅ Right):
```python
from wipnote import SDK

sdk = SDK()
session = sdk.sessions.get('session-123')
recent = session.get_events(limit=10, offset=session.event_count - 10)
```

## See Also

- [SDK Guide](./SDK_FOR_AI_AGENTS.md) - Complete SDK documentation
- [Session Management](./guide/sessions.md) - Session workflow guide
- [Event Log](./api/event-log.md) - Event log API reference
