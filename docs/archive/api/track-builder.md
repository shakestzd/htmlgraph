# TrackBuilder

Fluent API for creating tracks with specs and plans.

## Overview

The TrackBuilder provides a chainable API for creating complex tracks in a single expression.

## Basic Usage

```python
from wipnote import SDK

sdk = SDK(agent="claude")

track = sdk.tracks.builder() \
    .title("User Authentication") \
    .description("Implement OAuth 2.0") \
    .priority("high") \
    .create()
```

## With Spec

```python
track = sdk.tracks.builder() \
    .title("User Authentication") \
    .priority("high") \
    .with_spec(
        overview="Add OAuth 2.0 support",
        context="Current system has no auth",
        requirements=[
            ("OAuth 2.0 flow", "must-have"),
            ("JWT tokens", "must-have")
        ],
        acceptance_criteria=[
            ("Users can log in", "Login test passes")
        ]
    ) \
    .create()
```

## With Plan

```python
track = sdk.tracks.builder() \
    .title("User Authentication") \
    .priority("high") \
    .with_plan_phases([
        ("Phase 1: Setup", [
            "Configure OAuth (2h)",
            "Setup database (1h)"
        ]),
        ("Phase 2: Implementation", [
            "Implement login (4h)",
            "Add middleware (3h)"
        ])
    ]) \
    .create()
```

## Complete Track

```python
track = sdk.tracks.builder() \
    .title("User Authentication") \
    .description("Implement OAuth 2.0 authentication") \
    .priority("high") \
    .with_spec(
        overview="Add OAuth 2.0 support",
        context="No auth currently",
        requirements=[
            ("OAuth 2.0", "must-have"),
            ("JWT tokens", "must-have")
        ],
        acceptance_criteria=[
            ("Users can log in", "Test passes")
        ]
    ) \
    .with_plan_phases([
        ("Phase 1", ["Setup (2h)", "Config (1h)"]),
        ("Phase 2", ["Implement (4h)", "Test (3h)"])
    ]) \
    .create()
```

## Methods

### `.title(title: str) -> TrackBuilder`

Set the track title (required).

### `.description(desc: str) -> TrackBuilder`

Set the track description (optional).

### `.priority(priority: str) -> TrackBuilder`

Set priority: "low", "medium", "high", "critical" (default: "medium").

### `.with_spec(...) -> TrackBuilder`

Add specification with requirements and criteria.

**Parameters:**
- `overview: str` - High-level summary
- `context: str` - Background and constraints
- `requirements: list` - Requirements as (description, priority) tuples or strings
- `acceptance_criteria: list` - Success criteria as (description, test_case) tuples or strings
- `constraints: Optional[list[str]]` - Constraints

### `.with_plan_phases(phases: list[tuple[str, list[str]]]) -> TrackBuilder`

Add implementation plan with phases.

**Format:** `[(phase_name, [task_descriptions]), ...]`

Time estimates: Include `(Xh)` in task description for time estimates.

### `.create() -> Track`

Execute the build and create all files. Returns Track object.

## Complete API Reference

For detailed API documentation with method signatures and builder pattern implementation, see the Python source code in `src/python/wipnote/track_builder.py`.
