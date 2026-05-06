# Analytics API Documentation

**Phase 2: Work Type Analytics** - Analyze work type distribution, spike-to-feature ratios, and maintenance burden across sessions.

## Overview

The Analytics API provides methods to analyze work patterns across sessions and projects. Use it to:

- **Understand work distribution** - What % of time is spent on features vs spikes vs maintenance?
- **Identify research-heavy sessions** - High spike-to-feature ratios indicate exploratory work
- **Track maintenance burden** - Measure technical debt and maintenance workload
- **Filter sessions by work type** - Find all exploratory or implementation sessions

## Quick Start

```python
from wipnote import SDK

sdk = SDK(agent="claude")

# Get work type distribution for a session
dist = sdk.analytics.work_type_distribution(session_id="session-123")
print(dist)
# {"feature-implementation": 45.2, "spike-investigation": 28.3, "maintenance": 18.5, ...}

# Calculate spike-to-feature ratio
ratio = sdk.analytics.spike_to_feature_ratio(session_id="session-123")
print(f"Spike-to-feature ratio: {ratio:.2f}")
# Spike-to-feature ratio: 0.63 (research-heavy session)

# Get maintenance burden
burden = sdk.analytics.maintenance_burden(session_id="session-123")
print(f"Maintenance burden: {burden:.1f}%")
# Maintenance burden: 18.5%

# Find all exploratory sessions
from wipnote import WorkType
spike_sessions = sdk.analytics.get_sessions_by_work_type(WorkType.SPIKE.value)
print(f"Found {len(spike_sessions)} exploratory sessions")
```

## API Reference

### `sdk.analytics`

The Analytics interface is accessible via the `sdk.analytics` property.

---

### `work_type_distribution()`

Calculate work type distribution as percentages.

**Signature:**
```python
def work_type_distribution(
    self,
    session_id: str | None = None,
    start_date: datetime | None = None,
    end_date: datetime | None = None,
) -> dict[str, float]
```

**Parameters:**
- `session_id` (optional): Session ID to analyze. If None, analyzes all sessions.
- `start_date` (optional): Filter sessions after this date.
- `end_date` (optional): Filter sessions before this date.

**Returns:**
Dictionary mapping work type to percentage (0-100).

**Example:**
```python
# Single session
dist = sdk.analytics.work_type_distribution(session_id="session-123")
print(dist)
# {
#     "feature-implementation": 45.2,
#     "spike-investigation": 28.3,
#     "maintenance": 18.5,
#     "documentation": 8.0
# }

# Across date range
from datetime import datetime
dist = sdk.analytics.work_type_distribution(
    start_date=datetime(2024, 12, 1),
    end_date=datetime(2024, 12, 31)
)
```

**Interpretation:**
- High feature %: Implementation-focused work
- High spike %: Research and exploration
- High maintenance %: Refactoring and bug fixes

---

### `spike_to_feature_ratio()`

Calculate ratio of spike events to feature events.

**Signature:**
```python
def spike_to_feature_ratio(
    self,
    session_id: str | None = None,
    start_date: datetime | None = None,
    end_date: datetime | None = None,
) -> float
```

**Parameters:**
- `session_id` (optional): Session ID to analyze.
- `start_date` (optional): Filter sessions after this date.
- `end_date` (optional): Filter sessions before this date.

**Returns:**
Ratio of spike events to feature events (0.0 to infinity). Returns 0.0 if no feature events found.

**Example:**
```python
ratio = sdk.analytics.spike_to_feature_ratio(session_id="session-123")
print(f"Spike-to-feature ratio: {ratio:.2f}")

if ratio > 0.5:
    print("This was a research-heavy session")
elif ratio > 0.2:
    print("This was a balanced session")
else:
    print("This was an implementation-heavy session")
```

**Interpretation:**
- **>0.5**: Research-heavy (more exploration than implementation)
- **0.2-0.5**: Balanced (healthy mix)
- **<0.2**: Implementation-heavy (mostly building features)

---

### `maintenance_burden()`

Calculate percentage of work spent on maintenance vs new features.

**Signature:**
```python
def maintenance_burden(
    self,
    session_id: str | None = None,
    start_date: datetime | None = None,
    end_date: datetime | None = None,
) -> float
```

**Parameters:**
- `session_id` (optional): Session ID to analyze.
- `start_date` (optional): Filter sessions after this date.
- `end_date` (optional): Filter sessions before this date.

**Returns:**
Percentage of work spent on maintenance (0-100). Maintenance includes bug fixes and chores.

**Example:**
```python
burden = sdk.analytics.maintenance_burden(session_id="session-123")
print(f"Maintenance burden: {burden:.1f}%")

if burden > 40:
    print("⚠️  High maintenance burden - consider addressing technical debt")
elif burden > 20:
    print("Moderate maintenance burden - healthy balance")
else:
    print("Low maintenance burden - mostly new development")
```

**Interpretation:**
- **<20%**: Healthy (mostly new development)
- **20-40%**: Moderate (balanced maintenance)
- **>40%**: High burden (technical debt accumulation)

---

### `get_sessions_by_work_type()`

Get list of session IDs where the primary work type matches.

**Signature:**
```python
def get_sessions_by_work_type(
    self,
    primary_work_type: str,
    start_date: datetime | None = None,
    end_date: datetime | None = None,
) -> list[str]
```

**Parameters:**
- `primary_work_type`: Work type to filter by (e.g., "spike-investigation", "feature-implementation").
- `start_date` (optional): Filter sessions after this date.
- `end_date` (optional): Filter sessions before this date.

**Returns:**
List of session IDs matching the criteria.

**Example:**
```python
from wipnote import WorkType

# Find all exploratory sessions
spike_sessions = sdk.analytics.get_sessions_by_work_type(WorkType.SPIKE.value)
print(f"Found {len(spike_sessions)} exploratory sessions")

# Find all implementation sessions
feature_sessions = sdk.analytics.get_sessions_by_work_type(WorkType.FEATURE.value)
print(f"Found {len(feature_sessions)} implementation sessions")

# Filter by date range
from datetime import datetime
recent_spikes = sdk.analytics.get_sessions_by_work_type(
    WorkType.SPIKE.value,
    start_date=datetime(2024, 12, 1)
)
```

---

### `calculate_session_work_breakdown()`

Calculate work type breakdown (event counts) for a session.

**Signature:**
```python
def calculate_session_work_breakdown(self, session_id: str) -> dict[str, int]
```

**Parameters:**
- `session_id`: Session ID to analyze.

**Returns:**
Dictionary mapping work type to event count.

**Example:**
```python
breakdown = sdk.analytics.calculate_session_work_breakdown("session-123")
print(breakdown)
# {
#     "feature-implementation": 45,
#     "spike-investigation": 28,
#     "maintenance": 15
# }

# Calculate percentages
total = sum(breakdown.values())
for work_type, count in breakdown.items():
    pct = (count / total) * 100
    print(f"{work_type}: {pct:.1f}% ({count} events)")
```

---

### `calculate_session_primary_work_type()`

Calculate the primary work type for a session.

**Signature:**
```python
def calculate_session_primary_work_type(self, session_id: str) -> str | None
```

**Parameters:**
- `session_id`: Session ID to analyze.

**Returns:**
Primary work type (most common), or None if no work type data.

**Example:**
```python
primary = sdk.analytics.calculate_session_primary_work_type("session-123")
print(f"Primary work type: {primary}")
# Primary work type: spike-investigation

if primary == WorkType.SPIKE.value:
    print("This session was primarily exploratory")
elif primary == WorkType.FEATURE.value:
    print("This session was primarily implementation")
```

---

## Use Cases

### 1. Project Health Dashboard

```python
from wipnote import SDK
from datetime import datetime, timedelta

sdk = SDK(agent="claude")

# Get last 30 days
end_date = datetime.now()
start_date = end_date - timedelta(days=30)

# Calculate metrics
dist = sdk.analytics.work_type_distribution(
    start_date=start_date,
    end_date=end_date
)
ratio = sdk.analytics.spike_to_feature_ratio(
    start_date=start_date,
    end_date=end_date
)
burden = sdk.analytics.maintenance_burden(
    start_date=start_date,
    end_date=end_date
)

print("=== Project Health (Last 30 Days) ===")
print(f"Spike-to-Feature Ratio: {ratio:.2f}")
print(f"Maintenance Burden: {burden:.1f}%")
print(f"\nWork Distribution:")
for work_type, pct in sorted(dist.items(), key=lambda x: x[1], reverse=True):
    print(f"  {work_type}: {pct:.1f}%")
```

### 2. Session Analysis

```python
# Analyze a specific session
session_id = "session-abc-123"

dist = sdk.analytics.work_type_distribution(session_id=session_id)
primary = sdk.analytics.calculate_session_primary_work_type(session_id)

print(f"Session {session_id}")
print(f"Primary work type: {primary}")
print(f"\nBreakdown:")
for work_type, pct in dist.items():
    print(f"  {work_type}: {pct:.1f}%")

# Interpretation
ratio = sdk.analytics.spike_to_feature_ratio(session_id=session_id)
if ratio > 0.5:
    print("\n💡 This was a research-heavy session")
else:
    print("\n🔨 This was an implementation-heavy session")
```

### 3. Find Similar Sessions

```python
from wipnote import WorkType

# Find all exploratory sessions
spike_sessions = sdk.analytics.get_sessions_by_work_type(WorkType.SPIKE.value)

print(f"Found {len(spike_sessions)} exploratory sessions:")
for session_id in spike_sessions[:5]:  # Show first 5
    breakdown = sdk.analytics.calculate_session_work_breakdown(session_id)
    spike_count = breakdown.get(WorkType.SPIKE.value, 0)
    print(f"  {session_id}: {spike_count} spike events")
```

### 4. Maintenance Burden Tracking

```python
from datetime import datetime, timedelta

# Track maintenance burden over time
intervals = []
for i in range(4):  # Last 4 weeks
    end = datetime.now() - timedelta(weeks=i)
    start = end - timedelta(weeks=1)

    burden = sdk.analytics.maintenance_burden(
        start_date=start,
        end_date=end
    )

    intervals.append((start.strftime("%Y-%m-%d"), burden))

print("Maintenance Burden Trend:")
for date, burden in reversed(intervals):
    print(f"  Week of {date}: {burden:.1f}%")
```

---

## Work Type Reference

Work types are defined in `wipnote.models.WorkType`:

- `FEATURE` = "feature-implementation" - Building new functionality
- `SPIKE` = "spike-investigation" - Research and exploration
- `BUG_FIX` = "bug-fix" - Correcting defects
- `MAINTENANCE` = "maintenance" - Refactoring and tech debt
- `DOCUMENTATION` = "documentation" - Writing docs
- `PLANNING` = "planning" - Design and architecture
- `REVIEW` = "review" - Code review
- `ADMIN` = "admin" - Administrative tasks

---

## Best Practices

### 1. Regular Health Checks

Run analytics weekly to track project health:

```python
# Weekly health check
burden = sdk.analytics.maintenance_burden()
if burden > 40:
    print("⚠️  High maintenance burden - schedule refactoring sprint")
```

### 2. Session Retrospectives

Analyze sessions after major work:

```python
# After completing a feature
session_id = "session-latest"
primary = sdk.analytics.calculate_session_primary_work_type(session_id)
ratio = sdk.analytics.spike_to_feature_ratio(session_id)

print(f"Session completed: {primary}")
print(f"Exploration vs Implementation: {ratio:.2f}")
```

### 3. Sprint Planning

Use analytics to inform planning:

```python
# Check last sprint's work distribution
recent_dist = sdk.analytics.work_type_distribution()

if recent_dist.get("spike-investigation", 0) > 40:
    print("Last sprint was research-heavy - focus on implementation this sprint")
```

---

## Troubleshooting

### No work type data

If analytics return empty results, check:

1. **Events have work_type field**: Work types are auto-inferred from feature IDs
   ```python
   # Verify events have work_type
   session = sdk.sessions.get("session-123")
   events = session.get_events(limit=5)
   for evt in events:
       print(evt.get("work_type"))  # Should show work type
   ```

2. **Sessions exist**: Check session count
   ```python
   sessions = sdk.sessions.all()
   print(f"Found {len(sessions)} sessions")
   ```

### Unexpected ratios

If ratios seem off:

1. **Check event counts**:
   ```python
   breakdown = sdk.analytics.calculate_session_work_breakdown("session-123")
   print(f"Total events: {sum(breakdown.values())}")
   ```

2. **Verify work type inference**:
   ```python
   from wipnote import infer_work_type_from_id

   work_type = infer_work_type_from_id("feat-123")
   print(f"Feature ID 'feat-123' → {work_type}")
   ```

---

## See Also

- [Work Type Classification (Phase 1)](./WORK_TYPE_CATEGORIZATION.md)
- [SDK Documentation](./SDK_FOR_AI_AGENTS.md)
- [Session Management](./SESSIONS.md)
