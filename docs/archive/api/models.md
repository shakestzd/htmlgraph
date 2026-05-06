# Models

Pydantic data models for Wipnote entities.

## Overview

Wipnote uses Pydantic models for:

- Type safety and validation
- Serialization to/from HTML
- JSON export/import
- Schema documentation

All models are immutable by default and validate on construction.

## Feature

Represents a unit of work.

```python
from wipnote.models import Feature, FeatureStatus, Priority

feature = Feature(
    id="feature-20241216-103045",
    title="Add user authentication",
    status=FeatureStatus.IN_PROGRESS,
    priority=Priority.HIGH,
    steps=[
        Step(description="Create endpoint", completed=True),
        Step(description="Add middleware", completed=False)
    ]
)
```

### Fields

- `id: str` - Unique identifier
- `title: str` - Feature title
- `status: FeatureStatus` - Current status
- `priority: Priority` - Priority level
- `created: datetime` - Creation timestamp
- `updated: datetime` - Last update timestamp
- `steps: list[Step]` - Implementation steps
- `properties: dict` - Custom properties
- `track_id: Optional[str]` - Parent track ID

### Methods

- `to_html() -> str` - Convert to HTML file content
- `save()` - Save to disk
- `to_dict() -> dict` - Convert to dictionary
- `to_json() -> str` - Convert to JSON

## Track

Represents a multi-feature project.

```python
from wipnote.models import Track

track = Track(
    track_id="track-20241216-120000",
    title="User Authentication System",
    priority=Priority.HIGH,
    has_spec=True,
    has_plan=True
)
```

### Fields

- `track_id: str` - Unique identifier
- `title: str` - Track title
- `description: str` - Track description
- `priority: Priority` - Priority level
- `status: str` - Current status
- `created: datetime` - Creation timestamp
- `updated: datetime` - Last update timestamp
- `has_spec: bool` - Whether track has a spec
- `has_plan: bool` - Whether track has a plan

## Session

Represents an agent work session.

```python
from wipnote.models import Session

session = Session(
    id="session-abc-123",
    agent="claude",
    start_time=datetime.now(),
    features_worked_on=["feature-001", "feature-002"]
)
```

### Fields

- `id: str` - Unique identifier
- `agent: str` - Agent name
- `start_time: datetime` - Session start
- `end_time: Optional[datetime]` - Session end
- `features_worked_on: list[str]` - Feature IDs
- `event_count: int` - Number of events logged

## Step

Represents a single implementation step.

```python
from wipnote.models import Step

step = Step(
    description="Create login endpoint",
    completed=False,
    agent="claude",
    timestamp=datetime.now()
)
```

### Fields

- `description: str` - Step description
- `completed: bool` - Completion status
- `agent: Optional[str]` - Agent who completed it
- `timestamp: Optional[datetime]` - Completion time

## Enums

### FeatureStatus

```python
from wipnote.models import FeatureStatus

FeatureStatus.TODO         # Not started
FeatureStatus.IN_PROGRESS  # Currently working
FeatureStatus.BLOCKED      # Waiting on dependencies
FeatureStatus.DONE         # Completed
FeatureStatus.CANCELLED    # Work abandoned
```

### Priority

```python
from wipnote.models import Priority

Priority.LOW       # Low priority
Priority.MEDIUM    # Medium priority (default)
Priority.HIGH      # High priority
Priority.CRITICAL  # Critical priority
```

## Complete API Reference

For detailed API documentation with type signatures and complete field definitions, see the Python source code in `src/python/wipnote/models.py`.
