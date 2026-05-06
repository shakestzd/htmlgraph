# Agent Coordination Examples

Real-world patterns for coordinating multiple AI agents using Wipnote.

## Multi-Agent Workflow with Subagents

Wipnote provides `spawn_explorer` and `spawn_coder` for delegating specialized tasks to subagents, preserving main session context for orchestration.

### Two-Phase Pattern: Explorer → Coder

```python
from wipnote import SDK

sdk = SDK(agent="claude")

# Create a feature
feature = sdk.features.create("Add OAuth authentication") \
    .set_priority("high") \
    .save()

# Phase 1: Spawn explorer to discover codebase
explorer_prompt = sdk.spawn_explorer(
    task="Map the authentication system",
    scope="src/auth/",
    patterns=["**/*.py"],
    questions=[
        "What authentication framework is currently used?",
        "Where are auth routes defined?",
        "What patterns should I follow?"
    ]
)

# Execute explorer with Task tool
# Task(prompt=explorer_prompt["prompt"],
#      description=explorer_prompt["description"])

# After explorer completes, you receive results like:
explorer_results = """
SUMMARY: Found FastAPI-based auth system
FILES: src/auth/routes.py, src/auth/middleware.py
PATTERNS: Uses JWT tokens, OAuth handlers in routes.py
"""

# Phase 2: Spawn coder with explorer context
coder_prompt = sdk.spawn_coder(
    feature_id=feature.id,
    context=explorer_results,
    files_to_modify=["src/auth/routes.py", "src/auth/oauth.py"],
    test_command="uv run pytest tests/auth/"
)

# Execute coder with Task tool
# Task(prompt=coder_prompt["prompt"],
#      description=coder_prompt["description"])
```

### Benefits of Subagent Pattern

1. **Context Efficiency**: Main session preserves context for orchestration decisions
2. **Specialized Focus**: Explorer finds patterns, coder implements changes
3. **Stateless Execution**: Each subagent is ephemeral and task-focused
4. **Parallel Potential**: Multiple subagents can work simultaneously

## Parallel Task Execution

Execute multiple features in parallel using orchestration:

```python
from wipnote import SDK

sdk = SDK(agent="claude")

# Create multiple related features
features = [
    sdk.features.create("Add Google OAuth").set_priority("high").save(),
    sdk.features.create("Add GitHub OAuth").set_priority("high").save(),
    sdk.features.create("Implement token refresh").set_priority("medium").save()
]

# Spawn explorers in parallel (each discovers their domain)
explorer_prompts = []
for feat in features:
    prompt = sdk.spawn_explorer(
        task=f"Explore codebase for: {feat.title}",
        scope="src/auth/",
        questions=[
            "What existing OAuth code can be reused?",
            "What files need changes?"
        ]
    )
    explorer_prompts.append((feat.id, prompt))
    # Execute each with Task tool in parallel

# After exploration, spawn coders in parallel
coder_prompts = []
for feat_id, explorer_result in zip([f.id for f in features], explorer_results):
    prompt = sdk.spawn_coder(
        feature_id=feat_id,
        context=explorer_result,
        test_command="uv run pytest tests/auth/"
    )
    coder_prompts.append(prompt)
    # Execute each with Task tool in parallel
```

## Agent Handoff Pattern

Transfer work between different agents with full context preservation:

```python
# Agent 1 (Claude) starts work
claude_sdk = SDK(agent="claude")

feature = claude_sdk.features.create("Implement OAuth flow") \
    .add_steps([
        "Configure OAuth providers",
        "Implement callback handler",
        "Add token storage"
    ]) \
    .save()

# Claude completes initial steps
with claude_sdk.features.edit(feature.id) as f:
    f.status = "in-progress"
    f.assigned_agent = "claude"
    f.steps[0].completed = True

# Claude hands off to Gemini
with claude_sdk.features.edit(feature.id) as f:
    f.handoff_notes = """
    Completed OAuth provider configuration for Google and GitHub.

    Next Steps:
    - Implement callback handler at /auth/callback
    - Files to modify: src/auth/oauth.py, src/routes/auth.py
    - Use existing JWT middleware pattern from src/auth/middleware.py

    Blockers: None
    """
    f.assigned_agent = "gemini"

# Agent 2 (Gemini) picks up work
gemini_sdk = SDK(agent="gemini")

feature = gemini_sdk.features.get(feature.id)
print(f"Handoff notes: {feature.handoff_notes}")

# Gemini continues from step 2
with gemini_sdk.features.edit(feature.id) as f:
    f.steps[1].completed = True
    # Continue work...
```

## Orchestrated Feature Implementation

Use the high-level `orchestrate()` method for complete automation:

```python
from wipnote import SDK

sdk = SDK(agent="claude")

# Create feature
feature = sdk.features.create("Add user profile endpoint") \
    .add_steps([
        "Create route handler",
        "Add database queries",
        "Write tests"
    ]) \
    .save()

# Orchestrate full implementation (spawns explorer + coder automatically)
result = sdk.orchestrate(
    feature_id=feature.id,
    exploration_scope="src/",
    test_command="uv run pytest tests/api/"
)

# Returns prompts for both subagents
explorer_prompt = result["explorer"]
coder_prompt = result["coder"]

# Execute in sequence:
# 1. Task(prompt=explorer_prompt["prompt"], ...)
# 2. Task(prompt=coder_prompt["prompt"], ...)
```

## Custom Agent Integration

Build your own agent wrapper with automatic tracking:

```python
from wipnote import SDK

class CustomAgent:
    def __init__(self, name: str):
        self.sdk = SDK(agent=name)
        self.name = name

    def execute_task(self, task_description: str):
        """Execute task with automatic feature tracking"""
        # Create feature
        feature = self.sdk.features.create(task_description).save()

        # Start work
        with self.sdk.features.edit(feature.id) as f:
            f.status = "in-progress"
            f.assigned_agent = self.name

        try:
            # Your execution logic here
            result = self._do_work(feature)

            # Mark complete
            with self.sdk.features.edit(feature.id) as f:
                f.status = "done"

            return result

        except Exception as e:
            # Handle failure
            with self.sdk.features.edit(feature.id) as f:
                f.status = "blocked"
                f.handoff_notes = f"Error: {str(e)}\nNeeds investigation"
            raise

    def _do_work(self, feature):
        # Your implementation
        for i, step in enumerate(feature.steps):
            self._execute_step(step.description)
            with self.sdk.features.edit(feature.id) as f:
                f.steps[i].completed = True

    def _execute_step(self, step_description: str):
        # Your step execution logic
        pass

# Usage
agent = CustomAgent("my-agent")
agent.execute_task("Implement payment processing")
```

## Session Tracking

All agent activity is tracked automatically:

```python
from wipnote import SDK

sdk = SDK(agent="claude")

# Sessions start automatically
status = sdk.status()
print(f"Current session: {status.current_session}")
print(f"Active features: {status.active_features}")

# All SDK operations are logged to .wipnote/events/*.jsonl
feature = sdk.features.create("New feature")

# View session activity
# open .wipnote/sessions/{session_id}/index.html
```

## Best Practices

### 1. Use Explorers for Discovery

```python
# Good: Use explorer to understand codebase first
explorer = sdk.spawn_explorer(
    task="Find authentication patterns",
    scope="src/",
    patterns=["**/*.py"]
)
# Then use results to inform implementation
```

### 2. Preserve Context with Handoffs

```python
# Good: Detailed handoff notes
with sdk.features.edit(feature.id) as f:
    f.handoff_notes = """
    What I did: Configured OAuth providers
    What's next: Implement callback at /auth/callback
    Files: src/auth/oauth.py (see TODO comments)
    """
    f.assigned_agent = "next-agent"
```

### 3. Parallel When Possible

```python
# Good: Independent features can run in parallel
features = [create_google_oauth(), create_github_oauth()]
prompts = [sdk.spawn_coder(f.id) for f in features]
# Execute all prompts with Task tool simultaneously
```

### 4. Always Specify Test Commands

```python
# Good: Coders run tests automatically
coder = sdk.spawn_coder(
    feature_id=feature.id,
    test_command="uv run pytest tests/auth/ -v"
)
```

## Next Steps

- **[Agents Guide](../guide/agents.md)** - Full agent integration guide
- **[Track Examples](tracks.md)** - Complex multi-feature workflows
- **[API Reference](../api/sdk.md)** - Complete SDK documentation
