# Track Creation

Examples of creating tracks with specs and plans.

## Simple Track

```python
from wipnote import SDK

sdk = SDK(agent="claude")

# Minimal track
track = sdk.tracks.builder() \
    .title("Website Redesign") \
    .description("Redesign company website") \
    .priority("high") \
    .create()

print(f"Created: {track.track_id}")
```

## Track with Specification

```python
# Track with detailed spec
track = sdk.tracks.builder() \
    .title("User Authentication System") \
    .description("Implement secure authentication") \
    .priority("critical") \
    .with_spec(
        overview="Add OAuth 2.0 authentication with Google and GitHub providers",
        context="""
Current State:
- No authentication system
- Users cannot save preferences
- No access control

Goal:
- Secure OAuth 2.0 login
- JWT token management
- User profile management
        """,
        requirements=[
            ("Google OAuth integration", "must-have"),
            ("GitHub OAuth integration", "must-have"),
            ("JWT token management", "must-have"),
            ("Token refresh mechanism", "must-have"),
            ("User profile endpoint", "should-have"),
            ("Remember me functionality", "nice-to-have")
        ],
        acceptance_criteria=[
            ("Users can log in with Google", "OAuth login test passes"),
            ("Users can log in with GitHub", "OAuth login test passes"),
            ("Tokens refresh automatically", "Token refresh test passes"),
            "Session persists across browser restarts",
            "Logout clears all tokens"
        ],
        constraints=[
            "Must comply with GDPR",
            "Must use OAuth 2.0 standard",
            "Maximum 500ms latency for token validation",
            "Support 10,000 concurrent users"
        ]
    ) \
    .create()
```

## Track with Implementation Plan

```python
# Track with phased plan
track = sdk.tracks.builder() \
    .title("Database Migration") \
    .description("Migrate from SQLite to PostgreSQL") \
    .priority("high") \
    .with_plan_phases([
        ("Phase 1: Setup & Planning", [
            "Install PostgreSQL (0.5h)",
            "Design new schema (2h)",
            "Create migration scripts (4h)",
            "Set up staging environment (1h)"
        ]),
        ("Phase 2: Data Migration", [
            "Export data from SQLite (1h)",
            "Transform data for PostgreSQL (3h)",
            "Import to staging database (2h)",
            "Validate data integrity (4h)"
        ]),
        ("Phase 3: Testing", [
            "Update application config (1h)",
            "Run integration tests (3h)",
            "Performance testing (4h)",
            "Load testing (2h)"
        ]),
        ("Phase 4: Production Deployment", [
            "Schedule maintenance window",
            "Backup production database (1h)",
            "Run production migration (3h)",
            "Validate and monitor (2h)",
            "Update documentation (1h)"
        ])
    ]) \
    .create()

print(f"Total estimated time: {track.plan.total_estimate_hours}h")
```

## Complete Track (Spec + Plan)

```python
# Full track with spec and plan
track = sdk.tracks.builder() \
    .title("API Rate Limiting") \
    .description("Implement rate limiting for all API endpoints") \
    .priority("high") \
    .with_spec(
        overview="Protect API from abuse with token bucket rate limiting",
        context="Current API has no rate limits, vulnerable to DoS attacks",
        requirements=[
            ("Token bucket algorithm", "must-have"),
            ("Redis for distributed state", "must-have"),
            ("Rate limit middleware", "must-have"),
            ("Rate limit headers in responses", "should-have"),
            ("Per-endpoint limits", "should-have"),
            ("Admin bypass capability", "nice-to-have")
        ],
        acceptance_criteria=[
            ("100 req/min per API key", "Load test confirms"),
            ("429 status when exceeded", "Integration test passes"),
            "Rate limits reset every 60 seconds",
            "Distributed across multiple servers",
            "Sub-5ms latency overhead"
        ]
    ) \
    .with_plan_phases([
        ("Phase 1: Core Implementation", [
            "Implement token bucket (3h)",
            "Add Redis client (1h)",
            "Create rate limit middleware (2h)",
            "Add configuration system (1h)"
        ]),
        ("Phase 2: Integration", [
            "Add middleware to all routes (2h)",
            "Implement rate limit headers (1h)",
            "Add admin bypass logic (1h)",
            "Error handling and logging (2h)"
        ]),
        ("Phase 3: Testing & Docs", [
            "Unit tests for algorithm (2h)",
            "Integration tests (3h)",
            "Load testing (4h)",
            "API documentation (2h)",
            "Deployment guide (1h)"
        ])
    ]) \
    .create()

# View the created files
print(f"\nCreated track: {track.track_id}")
print(f"  - Spec: .wipnote/tracks/{track.track_id}/spec.html")
print(f"  - Plan: .wipnote/tracks/{track.track_id}/plan.html")
print(f"  - Index: .wipnote/tracks/{track.track_id}/index.html")
```

## Creating Features from Track

```python
# Create a track
track = sdk.tracks.builder() \
    .title("E-commerce Platform") \
    .with_plan_phases([
        ("Phase 1: Product Catalog", [
            "Product listing page",
            "Product detail page",
            "Search functionality"
        ]),
        ("Phase 2: Shopping Cart", [
            "Add to cart",
            "Cart management",
            "Checkout flow"
        ]),
        ("Phase 3: Payment", [
            "Payment gateway integration",
            "Order confirmation",
            "Email notifications"
        ])
    ]) \
    .create()

# Create a feature for each task
all_features = []

for phase in track.plan.phases:
    for task in phase.tasks:
        feature = sdk.features.create(
            title=task.description,
            track_id=track.track_id,
            priority=track.priority,
            properties={
                "phase": phase.name,
                "estimate_hours": task.estimate_hours or 0
            }
        )
        all_features.append(feature)

print(f"Created {len(all_features)} features for track {track.track_id}")

# Open in browser
import webbrowser
webbrowser.open(f".wipnote/tracks/{track.track_id}/index.html")
```

## Track-Driven Development Workflow

```python
from wipnote import SDK

def create_project_track(title, phases):
    """Create a track with features for each phase"""
    sdk = SDK(agent="claude")

    # Create track
    track = sdk.tracks.builder() \
        .title(title) \
        .priority("high") \
        .with_plan_phases(phases) \
        .create()

    # Create features for each task
    features_by_phase = {}

    for phase in track.plan.phases:
        phase_features = []

        for task in phase.tasks:
            feature = sdk.features.create(
                title=task.description,
                track_id=track.track_id,
                priority="high",
                steps=[]  # Will be filled in during implementation
            )
            phase_features.append(feature)

        features_by_phase[phase.name] = phase_features

    return track, features_by_phase

# Usage
track, features = create_project_track(
    title="Mobile App MVP",
    phases=[
        ("Phase 1: Core Features", [
            "User authentication (8h)",
            "Profile management (6h)",
            "Settings screen (4h)"
        ]),
        ("Phase 2: Main Functionality", [
            "Dashboard (10h)",
            "Data sync (8h)",
            "Offline mode (12h)"
        ]),
        ("Phase 3: Polish", [
            "Onboarding flow (6h)",
            "UI/UX refinements (8h)",
            "Performance optimization (6h)"
        ])
    ]
)

# Work through phases
for phase_name, phase_features in features.items():
    print(f"\n{phase_name}:")
    for feature in phase_features:
        print(f"  - {feature.title}")
```

## Next Steps

- [Basic Examples](basic.md) - Simple feature workflows
- [Agent Workflows](agents.md) - Agent integration patterns
- [TrackBuilder Guide](../guide/track-builder.md) - Complete TrackBuilder documentation
