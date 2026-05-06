# Agent Workflows

Examples of integrating Wipnote with AI agents.

## Claude Code Integration

```python
from wipnote import SDK

# SDK automatically detects agent from environment
sdk = SDK(agent="claude")

# Create feature with TrackBuilder decision framework
user_request = "Add user authentication"

# Decision: Complex feature, create a track
track = sdk.tracks.builder() \
    .title("User Authentication System") \
    .priority("high") \
    .with_spec(
        overview="Implement secure authentication",
        requirements=[
            ("OAuth 2.0 support", "must-have"),
            ("JWT token management", "must-have")
        ]
    ) \
    .with_plan_phases([
        ("Phase 1: Setup", ["Configure OAuth (2h)", "Setup DB (1h)"]),
        ("Phase 2: Implementation", ["Implement login (4h)", "Add middleware (3h)"])
    ]) \
    .create()

# Create features for each phase
for phase in track.plan.phases:
    sdk.features.create(
        title=phase.name,
        track_id=track.track_id,
        steps=[task.description for task in phase.tasks]
    )
```

## Multi-Agent Collaboration

```python
# Agent 1: Claude creates and starts work
claude_sdk = SDK(agent="claude")

feature = claude_sdk.features.create(
    title="Implement OAuth flow",
    priority="high",
    steps=[
        "Configure OAuth providers",
        "Implement callback handler",
        "Add token storage"
    ]
)

# Claude completes first step
feature.steps[0].completed = True
feature.assigned_agent = "claude"
feature.save()

# Claude hands off to Gemini
feature.handoff_notes = """
Completed OAuth provider configuration for Google and GitHub.

Next: Implement callback handler at /auth/callback
Files: src/auth/oauth.py, src/routes/auth.py
"""
feature.assigned_agent = "gemini"
feature.save()

# Agent 2: Gemini picks up
gemini_sdk = SDK(agent="gemini")

feature = gemini_sdk.features.get(feature.id)
print(feature.handoff_notes)

# Gemini continues work
feature.steps[1].completed = True
feature.save()
```

## Custom Agent Integration

```python
from wipnote import SDK

class MyCustomAgent:
    def __init__(self, name):
        self.sdk = SDK(agent=name)
        self.name = name

    def process_task(self, task_description):
        """Process a task with automatic tracking"""
        # Create feature
        feature = self.sdk.features.create(
            title=task_description,
            priority=self.assess_priority(task_description)
        )

        # Start working
        self.sdk.features.start(feature.id)

        try:
            # Execute task
            result = self.execute_task(feature)

            # Mark complete
            feature.status = "done"
            feature.save()

            return result

        except Exception as e:
            # Handle failure
            feature.status = "blocked"
            feature.handoff_notes = f"Error: {str(e)}"
            feature.save()
            raise

    def assess_priority(self, description):
        # Your priority logic
        keywords = {"urgent", "critical", "important"}
        if any(kw in description.lower() for kw in keywords):
            return "high"
        return "medium"

    def execute_task(self, feature):
        # Your execution logic
        for i, step in enumerate(feature.steps):
            self.execute_step(step.description)
            feature.steps[i].completed = True
            feature.save()

    def execute_step(self, step_description):
        # Your step execution logic
        pass

# Usage
agent = MyCustomAgent("my-agent")
agent.process_task("Add user authentication")
```

## Session Management

```python
from wipnote import SDK

sdk = SDK(agent="claude")

# Sessions start automatically
status = sdk.status()
print(f"Current session: {status.current_session}")

# All activity is logged automatically via hooks
feature = sdk.features.create("Implement API")

# Document decisions
sdk.track_activity(
    feature_id=feature.id,
    activity="Chose FastAPI over Flask (async support)"
)

# Session ends automatically when work completes
feature.status = "done"
feature.save()
```

## Next Steps

- [Track Creation Examples](tracks.md) - Complex track workflows
- [Agents Guide](../guide/agents.md) - Full agent integration guide
- [Sessions Guide](../guide/sessions.md) - Session tracking details
