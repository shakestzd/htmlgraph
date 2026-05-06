# Multi-Agent Workflow: Real-World Example

Implementing user authentication with Wipnote orchestration: parallel delegation of backend, frontend, and testing work with automatic session tracking.

## Scenario: Feature Implementation with Parallel Tasks

You need to implement user authentication in a web application. The work naturally divides into:
- **Backend**: OAuth integration with FastAPI
- **Frontend**: Login/signup UI components
- **Testing**: Unit tests, integration tests, E2E tests

Without delegation, this sequential work fills your context with intermediate results and takes 3x longer. With orchestration, three subagents work in parallel while you coordinate at a high level.

## Architecture: Parent Orchestrator + Three Subagents

```
┌─────────────────────────────────────────────────────────────┐
│ ORCHESTRATOR SESSION (main agent)                           │
│ - Creates feature and track                                 │
│ - Spawns 3 subagents (Task tool)                            │
│ - Monitors progress                                         │
│ - Consolidates results                                      │
│ - Wipnote automatically captures all session relationships │
└─────────────────────────────────────────────────────────────┘
         │                    │                    │
         ▼                    ▼                    ▼
    ┌────────────┐      ┌────────────┐      ┌────────────┐
    │ SUBAGENT 1 │      │ SUBAGENT 2 │      │ SUBAGENT 3 │
    │ Backend    │      │ Frontend   │      │ Testing    │
    │ Dev        │      │ Dev        │      │ Dev        │
    └────────────┘      └────────────┘      └────────────┘
         │                    │                    │
         ▼                    ▼                    ▼
    - Implement OAuth  - Build login UI    - Write unit tests
    - Setup tokens     - Signup form       - Integration tests
    - API routes       - Profile page      - E2E tests
```

## Orchestrator Implementation

```python
from wipnote import SDK

# Initialize orchestrator
sdk = SDK(agent="claude-orchestrator")

# Create feature tracking the entire initiative
auth_feature = sdk.features.create(
    title="Implement User Authentication",
    description="OAuth integration with UI components and comprehensive test coverage",
    priority="high"
)

# Create track to organize parallel work
auth_track = sdk.tracks.builder() \
    .title("Authentication Initiative") \
    .with_plan_phases([
        ("Phase 1: Exploration", [
            "Map existing auth architecture (1h)",
            "Research OAuth libraries (1h)",
            "Design component structure (1h)"
        ]),
        ("Phase 2: Parallel Implementation", [
            "Backend OAuth (4h)",
            "Frontend UI (4h)",
            "Testing suite (3h)"
        ]),
        ("Phase 3: Integration", [
            "Connect frontend to backend (2h)",
            "E2E testing (1h)",
            "Deployment (1h)"
        ])
    ]) \
    .create()

# Log orchestration start
sdk.sessions.log_event("orchestration_start", {
    "feature_id": auth_feature.id,
    "track_id": auth_track.id,
    "subagent_count": 3,
    "execution_mode": "parallel"
})

print(f"✅ Orchestration started")
print(f"   Feature: {auth_feature.id}")
print(f"   Track: {auth_track.id}")
```

## Subagent 1: Backend Implementation

```python
# SUBAGENT 1 - Backend Developer
# Spawned with Task() by orchestrator

from wipnote import SDK

sdk = SDK(agent="claude-subagent-backend")

# Get feature context (passed by orchestrator)
feature_id = "feat-abc123"  # Passed in Task prompt
feature = sdk.features.get(feature_id)

print(f"🔧 Backend Dev: Starting OAuth implementation")
print(f"   Feature: {feature.title}")

# Phase 1: Implement OAuth provider configuration
print("Step 1: Setting up OAuth provider...")
# - Add google_oauth and github_oauth packages
# - Configure environment variables
# - Create OAuth client instances

# Phase 2: Implement API routes
print("Step 2: Implementing authentication routes...")
# - POST /auth/google - Google callback handler
# - POST /auth/github - GitHub callback handler
# - POST /auth/logout - Logout handler
# - GET /auth/user - Get current user
# - POST /auth/refresh - Refresh tokens

# Phase 3: Implement token management
print("Step 3: Implementing token management...")
# - Create JWT token schema
# - Implement token generation
# - Implement token validation middleware
# - Add token refresh logic

# Log completion
sdk.features.log_completion_step(
    feature_id=feature_id,
    step_name="Backend OAuth Implementation",
    details={
        "files_created": 3,
        "files_modified": 5,
        "tests_passing": 12,
        "implementation_time": "4h"
    }
)

print("✅ Backend OAuth implementation complete")
```

## Subagent 2: Frontend Implementation

```python
# SUBAGENT 2 - Frontend Developer
# Spawned with Task() by orchestrator

from wipnote import SDK

sdk = SDK(agent="claude-subagent-frontend")

feature_id = "feat-abc123"  # Same feature as backend
feature = sdk.features.get(feature_id)

print(f"🎨 Frontend Dev: Starting UI implementation")
print(f"   Feature: {feature.title}")

# Phase 1: Build login page component
print("Step 1: Building login page...")
# - Create LoginPage component
# - Add Google OAuth button
# - Add GitHub OAuth button
# - Add "Sign up" link
# - Styling and responsiveness

# Phase 2: Build signup flow
print("Step 2: Building signup flow...")
# - Create SignupPage component
# - Email verification step
# - Profile completion form
# - Success notification

# Phase 3: Build user profile page
print("Step 3: Building user profile...")
# - Create ProfilePage component
# - Display user information
# - Show connected OAuth providers
# - Allow account linking/unlinking

# Phase 4: Integrate with backend
print("Step 4: API integration...")
# - Create API client
# - Implement login function
# - Implement signup function
# - Add authentication state management
# - Add error handling

# Log completion
sdk.features.log_completion_step(
    feature_id=feature_id,
    step_name="Frontend UI Implementation",
    details={
        "components_created": 5,
        "files_modified": 3,
        "visual_tests_passing": 8,
        "implementation_time": "4h"
    }
)

print("✅ Frontend UI implementation complete")
```

## Subagent 3: Testing Implementation

```python
# SUBAGENT 3 - QA/Testing Developer
# Spawned with Task() by orchestrator

from wipnote import SDK

sdk = SDK(agent="claude-subagent-testing")

feature_id = "feat-abc123"  # Same feature as backend/frontend
feature = sdk.features.get(feature_id)

print(f"🧪 Test Dev: Starting test implementation")
print(f"   Feature: {feature.title}")

# Phase 1: Unit tests
print("Step 1: Writing unit tests...")
# - Test JWT token generation
# - Test token validation
# - Test OAuth provider mocking
# - Test user model serialization
# Result: 12 unit tests passing

# Phase 2: Integration tests
print("Step 2: Writing integration tests...")
# - Test Google OAuth flow
# - Test GitHub OAuth flow
# - Test token refresh flow
# - Test logout flow
# Result: 8 integration tests passing

# Phase 3: E2E tests
print("Step 3: Writing E2E tests...")
# - Test complete login workflow
# - Test complete signup workflow
# - Test account linking
# - Test error scenarios
# Result: 5 E2E tests passing

# Phase 4: Test coverage analysis
print("Step 4: Coverage analysis...")
# - Generate coverage report
# - Identify gaps
# - Add missing tests
# - Target: 90% coverage

# Log completion
sdk.features.log_completion_step(
    feature_id=feature_id,
    step_name="Testing Suite Implementation",
    details={
        "unit_tests": 12,
        "integration_tests": 8,
        "e2e_tests": 5,
        "coverage": "92%",
        "implementation_time": "3h"
    }
)

print("✅ Testing suite complete")
```

## Orchestrator: Consolidation Phase

```python
# ORCHESTRATOR - Back to consolidation
# Called after all three subagents complete

from wipnote import SDK

sdk = SDK(agent="claude-orchestrator")

# Collect results from all subagents
backend_results = """
OAuth setup complete. Routes functional.
Tests: 12/12 passing.
Time: 4 hours.
"""

frontend_results = """
UI fully implemented. 5 components created.
Tests: 8/8 passing.
Time: 4 hours.
"""

testing_results = """
Test suite comprehensive (25 tests total).
Coverage: 92%.
Time: 3 hours.
"""

# Log consolidation
sdk.sessions.log_event("orchestration_consolidation", {
    "backend_status": "complete",
    "frontend_status": "complete",
    "testing_status": "complete",
    "total_time": "4 hours (parallel vs 11 hours sequential)",
    "time_saved": "7 hours (64% reduction)"
})

# Update feature with final summary
feature = sdk.features.get("feat-abc123")
feature.summary = """
AUTHENTICATION IMPLEMENTATION - COMPLETE

Backend (Subagent 1):
✅ OAuth provider setup (Google, GitHub)
✅ 3 API routes implemented
✅ Token management middleware
✅ 12/12 unit tests passing

Frontend (Subagent 2):
✅ Login page with OAuth buttons
✅ Signup flow with verification
✅ User profile management
✅ 8/8 UI tests passing

Testing (Subagent 3):
✅ 25 total tests (12 unit, 8 integration, 5 E2E)
✅ 92% code coverage
✅ All critical paths tested

Total Implementation Time: 4 hours (parallel execution)
vs 11 hours (sequential execution) = 64% faster
"""
feature.save()

print("✅ Feature complete!")
print("📊 Total time: 4 hours (parallel)")
print("   vs 11 hours (sequential)")
print("   Savings: 7 hours (64% faster)")
```

## Wipnote Automatic Tracking

Wipnote's hook system automatically captures all of this:

```bash
# View orchestration structure
ls .wipnote/sessions/
# Shows:
# - sess-orchestrator-123.html (parent session)
# - sess-subagent-backend-456.html (child 1)
# - sess-subagent-frontend-789.html (child 2)
# - sess-subagent-testing-012.html (child 3)

# View feature tracking
ls .wipnote/features/
# Shows:
# - feat-abc123.html (auth feature, with child events)

# View in dashboard
uv run wipnote serve
# Navigate to:
# - Sessions tab → View parent-child hierarchy
# - Features tab → See auth feature with all work items
# - Orchestration tab → Timeline showing parallel execution
```

## Dashboard Visualization

The **Orchestration** tab in the dashboard shows:

```
Timeline:
├─ 00:00 Orchestrator starts
│  └─ Creates feature feat-abc123
│
├─ 00:05 Spawn subagents
│  ├─ Backend (Subagent 1)
│  ├─ Frontend (Subagent 2)
│  └─ Testing (Subagent 3)
│
├─ 04:00 All subagents complete
│  ├─ Backend: 12 tests passing
│  ├─ Frontend: 8 tests passing
│  └─ Testing: 25 total tests, 92% coverage
│
└─ 04:05 Orchestrator consolidates results
   └─ Feature marked complete

Parallel Execution View:
Backend   ████████████████████ 4h
Frontend  ████████████████████ 4h
Testing   ████████████████     3h
─────────────────────────────────
Max Time: 4h (vs 11h sequential = 64% faster)
```

## Key Metrics & Cost Analysis

### Execution Time
- **Parallel (with delegation)**: 4 hours
- **Sequential (no delegation)**: 11 hours
- **Time saved**: 7 hours (64% reduction)

### Computational Cost
```
Parallel Execution (Recommended):
├─ Orchestrator (Opus): 0.5h @ 30 tokens/min = 900 tokens
├─ Backend Subagent (Haiku): 4h @ 10 tokens/min = 2,400 tokens
├─ Frontend Subagent (Haiku): 4h @ 10 tokens/min = 2,400 tokens
└─ Testing Subagent (Haiku): 3h @ 10 tokens/min = 1,800 tokens
   Total: 7,500 tokens (faster, cheaper)

Sequential Execution (Without Delegation):
└─ Single Agent (Opus): 11h @ 30 tokens/min = 19,800 tokens
   Total: 19,800 tokens (slower, more expensive)

Savings: 12,300 tokens + 7 hours of wall-clock time
```

## When This Pattern Works Best

✅ **Use parallel delegation when:**
- Work divides into independent subtasks
- Parallel execution saves significant time (2+ hours)
- Subagents have clear focus areas
- Results can be easily consolidated
- Cost-per-token for Haiku << Opus

❌ **Avoid when:**
- Tasks are sequential (step N+1 depends on N)
- Heavy context sharing needed (defeats purpose)
- Subagent results hard to consolidate
- Task too small to justify overhead

## Common Mistakes to Avoid

1. **Not isolating subagent work** - Each subagent should have a clear, independent focus
2. **Poor prompt specification** - Vague prompts lead to failed delegations
3. **Expecting immediate context** - Subagents need all context in their prompt
4. **Over-delegation** - Don't delegate trivial tasks (overhead cost > benefit)
5. **Lost tracking** - Always log events so Wipnote captures relationships

## Next Steps

- See [Delegation Guide](../guide/delegation.md) for prompt writing best practices
- See [Orchestration Guide](../guide/orchestration.md) for advanced patterns
- See [Session Hierarchies](../guide/session-hierarchies.md) for capturing complex workflows
