# Basic Usage

Simple examples to get started with Wipnote.

## Creating Your First Feature

```python
from wipnote import SDK

# Initialize SDK
sdk = SDK(agent="me")

# Create a feature
feature = sdk.features.create(
    title="Add user login",
    priority="high",
    steps=[
        "Create login form",
        "Add authentication logic",
        "Write tests"
    ]
)

print(f"Created: {feature.id}")
# Output: Created: feature-20241216-103045
```

## Updating a Feature

```python
# Get the feature
feature = sdk.features.get("feature-20241216-103045")

# Mark as in progress
feature.status = "in-progress"
feature.save()

# Complete first step
feature.steps[0].completed = True
feature.save()

# Check progress
completed = sum(1 for s in feature.steps if s.completed)
total = len(feature.steps)
print(f"Progress: {completed}/{total} steps complete")
```

## Querying Features

```python
# Get all features
all_features = sdk.features.all()

# Filter by status
todo_features = sdk.features.where(status="todo")
in_progress = sdk.features.where(status="in-progress")

# Filter by priority
high_priority = sdk.features.where(priority="high")

# Multiple filters
urgent_tasks = sdk.features.where(
    status="todo",
    priority="high"
)

# Display results
for feature in urgent_tasks:
    print(f"- {feature.title} ({feature.priority})")
```

## Viewing in Browser

```python
# All features are HTML files
# Open in any browser to view
import webbrowser

feature = sdk.features.get("feature-20241216-103045")
webbrowser.open(f".wipnote/features/{feature.id}.html")
```

## Using the CLI

```bash
# Create a feature
wipnote feature create "Add user login" --priority high

# List all features
wipnote feature list

# Start working on a feature
wipnote feature start feature-20241216-103045

# Mark complete
wipnote feature complete feature-20241216-103045
```

## Launching the Dashboard

```bash
# Start the server
wipnote serve

# Open http://localhost:8080 in your browser
# See Kanban board, graph view, timeline, etc.
```

## Complete Example Script

```python
#!/usr/bin/env python3
from wipnote import SDK

def main():
    # Initialize
    sdk = SDK(agent="example-script")

    # Create features for a small project
    features = []

    features.append(sdk.features.create(
        title="Set up database",
        priority="high",
        steps=[
            "Install PostgreSQL",
            "Create database schema",
            "Add migrations"
        ]
    ))

    features.append(sdk.features.create(
        title="Create API endpoints",
        priority="high",
        steps=[
            "Define routes",
            "Implement handlers",
            "Add validation"
        ]
    ))

    features.append(sdk.features.create(
        title="Write tests",
        priority="medium",
        steps=[
            "Unit tests",
            "Integration tests",
            "E2E tests"
        ]
    ))

    # Add dependency: API depends on database
    sdk.features.add_edge(
        from_id=features[0].id,  # database
        to_id=features[1].id,    # API
        relationship="blocks"
    )

    print(f"Created {len(features)} features")
    print("\nNext steps:")
    print(f"  1. wipnote serve")
    print(f"  2. Open http://localhost:8080")
    print(f"  3. View your features in the dashboard")

if __name__ == "__main__":
    main()
```

Run it:

```bash
python example.py
wipnote serve
```

## Next Steps

- [Agent Workflows](agents.md) - Integrate with AI agents
- [Track Creation](tracks.md) - Create multi-feature tracks
- [User Guide](../guide/index.md) - In-depth documentation
