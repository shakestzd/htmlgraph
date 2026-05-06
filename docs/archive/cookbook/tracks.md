# Track & Planning Recipes

## Create Track with Spec and Plan

**Problem**: Plan a multi-feature initiative with specification and implementation plan.

**Solution**:

```python
from wipnote import SDK

sdk = SDK(agent="claude")

# Create track with TrackBuilder (recommended)
track = sdk.tracks.builder() \
    .title("User Authentication System") \
    .description("Implement OAuth 2.0 authentication with JWT") \
    .priority("high") \
    .with_spec(
        overview="Add secure authentication with OAuth 2.0 support",
        context="Current system has no authentication. Users need secure login.",
        requirements=[
            ("Implement OAuth 2.0 flow", "must-have"),
            ("Add JWT token management", "must-have"),
            ("Create user profile endpoint", "should-have"),
            "Add password reset functionality"  # Defaults to "must-have"
        ],
        acceptance_criteria=[
            ("Users can log in with Google/GitHub", "OAuth integration test passes"),
            "JWT tokens expire after 1 hour",
            "Password reset emails sent within 5 minutes"
        ]
    ) \
    .with_plan_phases([
        ("Phase 1: OAuth Setup", [
            "Configure OAuth providers (1h)",
            "Implement OAuth callback (2h)",
            "Add state verification (1h)"
        ]),
        ("Phase 2: JWT Integration", [
            "Create JWT signing logic (2h)",
            "Add token refresh endpoint (1.5h)",
            "Implement token validation middleware (2h)"
        ]),
        ("Phase 3: User Management", [
            "Create user profile endpoint (3h)",
            "Add password reset flow (4h)",
            "Write integration tests (3h)"
        ])
    ]) \
    .create()

print(f"Created track: {track.id}")
print(f"  Spec: {len(track.spec.requirements)} requirements")
print(f"  Plan: {len(track.plan.phases)} phases, {sum(len(p.tasks) for p in track.plan.phases)} tasks")
```

**What Gets Created**:
```
.wipnote/tracks/track-20241224-120000/
├── index.html   # Track metadata
├── spec.html    # Specification with requirements
└── plan.html    # Implementation plan with phases
```

**Explanation**:
- TrackBuilder provides fluent API for creation
- Auto-generates track ID with timestamp
- Creates index.html, spec.html, plan.html automatically
- Parses time estimates from task descriptions (e.g., "(2h)")
- Validates requirements and acceptance criteria via Pydantic

---

## Link Features to Track

**Problem**: Associate features with their parent track.

**Solution**:

```python
from wipnote import SDK

sdk = SDK(agent="claude")

track_id = "track-20241224-120000"

# Method 1: Link when creating feature
oauth_feature = sdk.features.create("OAuth Integration") \
    .set_track(track_id) \
    .set_priority("high") \
    .add_steps([
        "Configure OAuth providers",
        "Implement OAuth callback",
        "Add state verification"
    ]) \
    .save()

# Method 2: Link existing feature
feature = sdk.features.get("feature-123")
with sdk.features.edit(feature.id) as f:
    f.track = track_id

# Method 3: Bulk link multiple features
feature_ids = ["feat-001", "feat-002", "feat-003"]
sdk.features.batch_update(
    feature_ids,
    {"track": track_id}
)

print(f"Linked {len(feature_ids)} features to track {track_id}")
```

**Explanation**:
- track_id field links features to parent track
- Enables track-level progress tracking
- Used for querying related features
- Automatically indexed for fast lookups

---

## Track Progress Across Features

**Problem**: Monitor overall progress for a multi-feature track.

**Solution**:

```python
from wipnote import SDK

sdk = SDK(agent="claude")

track_id = "track-20241224-120000"

# Get all features in track
track_features = sdk.features.where(track=track_id)

# Calculate progress
total = len(track_features)
done = len([f for f in track_features if f.status == "done"])
in_progress = len([f for f in track_features if f.status == "in-progress"])
todo = len([f for f in track_features if f.status == "todo"])

progress_pct = (done / total * 100) if total > 0 else 0

print(f"Track Progress: {track_id}")
print(f"  Total features: {total}")
print(f"  Done: {done} ({done/total*100:.1f}%)")
print(f"  In Progress: {in_progress}")
print(f"  Todo: {todo}")
print(f"  Overall: {progress_pct:.1f}% complete")

# Get step-level progress
total_steps = sum(len(f.steps) for f in track_features)
completed_steps = sum(len([s for s in f.steps if s.completed]) for f in track_features)
step_progress_pct = (completed_steps / total_steps * 100) if total_steps > 0 else 0

print(f"\nStep Progress:")
print(f"  Completed: {completed_steps}/{total_steps} ({step_progress_pct:.1f}%)")
```

**Output**:
```
Track Progress: track-20241224-120000
  Total features: 3
  Done: 1 (33.3%)
  In Progress: 1
  Todo: 1
  Overall: 33.3% complete

Step Progress:
  Completed: 7/15 (46.7%)
```

---

## Create Features from Plan Phases

**Problem**: Convert plan phases into concrete features.

**Solution**:

```python
from wipnote import SDK

sdk = SDK(agent="claude")

track_id = "track-20241224-120000"
track = sdk.tracks.get(track_id)

# Create one feature per phase
for phase_idx, phase in enumerate(track.plan.phases):
    feature = sdk.features.create(phase.name) \
        .set_track(track_id) \
        .set_priority(track.priority) \
        .add_steps([task.description for task in phase.tasks]) \
        .save()

    print(f"Created feature {feature.id} for phase {phase_idx + 1}: {phase.name}")

print(f"\nCreated {len(track.plan.phases)} features from plan phases")
```

**Explanation**:
- Each phase becomes a feature
- Tasks within phase become feature steps
- Inherits track priority
- Automatically linked to parent track

---

## Query Tracks by Status

**Problem**: Find tracks in specific states.

**Solution**:

```python
from wipnote import SDK

sdk = SDK(agent="claude")

# Get all tracks
all_tracks = sdk.tracks.all()

# Filter by status (computed from features)
for track in all_tracks:
    features = sdk.features.where(track=track.id)

    if not features:
        status = "not-started"
    elif all(f.status == "done" for f in features):
        status = "complete"
    elif any(f.status == "in-progress" for f in features):
        status = "in-progress"
    else:
        status = "todo"

    print(f"{track.id}: {track.title} ({status})")
```

**Explanation**:
- Track status is derived from feature statuses
- Not-started = no features yet
- Complete = all features done
- In-progress = at least one feature in-progress
- Todo = has features, none in-progress

---

## Update Specification

**Problem**: Refine requirements as you learn more.

**Solution**:

```python
from wipnote import SDK

sdk = SDK(agent="claude")

track = sdk.tracks.get("track-20241224-120000")

# Add new requirement
with sdk.tracks.edit(track.id) as t:
    t.spec.requirements.append({
        "description": "Add 2FA support",
        "priority": "should-have"
    })

# Update acceptance criteria
with sdk.tracks.edit(track.id) as t:
    t.spec.acceptance_criteria.append({
        "description": "2FA enrollment rate > 50%",
        "test_case": "Analytics query shows >50% enrollment"
    })

# Add context note
with sdk.tracks.edit(track.id) as t:
    t.spec.context += "\n\nUpdate (2024-12-24): Added 2FA requirement based on security audit."
```

**Explanation**:
- Specs are living documents
- Update as requirements change
- Preserves history via git
- Automatically updates spec.html file

---

## Estimate Total Effort

**Problem**: Calculate estimated hours for a track.

**Solution**:

```python
from wipnote import SDK
import re

sdk = SDK(agent="claude")

track = sdk.tracks.get("track-20241224-120000")

# Parse time estimates from task descriptions
total_hours = 0
for phase in track.plan.phases:
    for task in phase.tasks:
        # Look for patterns like (2h), (1.5h), etc.
        match = re.search(r'\((\d+(?:\.\d+)?)\s*h\)', task.description)
        if match:
            hours = float(match.group(1))
            total_hours += hours

print(f"Track: {track.title}")
print(f"Estimated effort: {total_hours} hours")
print(f"                 {total_hours / 8:.1f} days")
print(f"                 {total_hours / 40:.1f} weeks")
```

**Output**:
```
Track: User Authentication System
Estimated effort: 15.5 hours
                 1.9 days
                 0.4 weeks
```

**Best Practice**: Include time estimates in task descriptions: "Implement OAuth (3h)"
