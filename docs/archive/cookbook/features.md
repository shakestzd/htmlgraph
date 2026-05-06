# Feature Management Recipes

## Create Feature

**Problem**: Create a new feature to track work.

**Solution**:

```python
from wipnote import SDK

sdk = SDK(agent="claude")

# Simple feature
feature = sdk.features.create("Add dark mode toggle")

# Feature with details
feature = sdk.features.create("User Authentication") \
    .set_priority("high") \
    .set_description("Implement OAuth 2.0 with JWT tokens") \
    .add_steps([
        "Research OAuth providers",
        "Implement auth routes",
        "Add JWT middleware",
        "Create user profile endpoint",
        "Write integration tests"
    ]) \
    .add_tags(["security", "backend"]) \
    .save()

print(f"Created: {feature.id}")
print(f"Steps: {len(feature.steps)}")
```

**Explanation**:
- Fluent API for building features
- Auto-generates ID with timestamp
- Creates HTML file in `.wipnote/features/`
- Ready to start working immediately

---

## Mark Steps Complete

**Problem**: Track progress as you complete steps.

**Solution**:

```python
from wipnote import SDK

sdk = SDK(agent="claude")

# Mark single step complete (0-indexed)
with sdk.features.edit("feature-123") as f:
    f.steps[0].completed = True

# Mark multiple steps at once
with sdk.features.edit("feature-123") as f:
    f.steps[0].completed = True
    f.steps[1].completed = True
    f.steps[2].completed = True

# Mark step and add notes
with sdk.features.edit("feature-123") as f:
    f.steps[3].completed = True
    f.steps[3].agent = "claude"
    f.steps[3].timestamp = datetime.now()
```

**Explanation**:
- Context manager auto-saves changes
- Steps are 0-indexed (first step = 0)
- Can add agent and timestamp for audit trail
- Changes immediately visible in dashboard

**Best Practice**: Mark steps complete IMMEDIATELY after finishing them, not all at the end.

---

## Add Dependencies

**Problem**: Express that one feature blocks another.

**Solution**:

```python
from wipnote import SDK

sdk = SDK(agent="claude")

# Method 1: Add single dependency
sdk.features.add_dependency(
    blocked_id="feature-auth",      # Feature that's blocked
    blocker_id="feature-database",  # Feature that blocks it
    relationship="blocks"
)

# Method 2: Add multiple dependencies
auth = sdk.features.get("feature-auth")
database = sdk.features.get("feature-database")
sessions = sdk.features.get("feature-sessions")

sdk.features.add_dependency(auth.id, database.id, "blocks")
sdk.features.add_dependency(auth.id, sessions.id, "blocks")

# Method 3: Add when creating feature
feature = sdk.features.create("API Endpoints") \
    .blocked_by(["feature-auth"]) \
    .save()
```

**Explanation**:
- Dependencies create graph edges
- Used for bottleneck analysis
- Determines parallel work capacity
- Visualized in dashboard graph view

---

## Query Features

**Problem**: Find features matching specific criteria.

**Solution**:

```python
from wipnote import SDK

sdk = SDK(agent="claude")

# By status
in_progress = sdk.features.where(status="in-progress")
todo = sdk.features.where(status="todo")
done = sdk.features.where(status="done")

# By priority
high_priority = sdk.features.where(priority="high")

# Multiple criteria
urgent = sdk.features.where(status="todo", priority="high")

# By tag
security_features = sdk.features.where(tags__contains="security")

# Get single feature
feature = sdk.features.get("feature-123")

# Get all features
all_features = sdk.features.all()

# Count features
count = len(sdk.features.where(status="todo"))
print(f"{count} todo features")
```

**Explanation**:
- `where()` returns list of matching features
- `get()` returns single feature by ID
- Can filter by any field (status, priority, tags, etc.)
- Use `tags__contains` for tag filtering

---

## Update Feature Status

**Problem**: Change feature status as work progresses.

**Solution**:

```python
from wipnote import SDK

sdk = SDK(agent="claude")

# Method 1: Direct edit
with sdk.features.edit("feature-123") as f:
    f.status = "in-progress"

# Method 2: Use CLI
# wipnote feature start feature-123
# wipnote feature complete feature-123

# Method 3: Batch update multiple features
sdk.features.batch_update(
    ["feature-001", "feature-002", "feature-003"],
    {"status": "done"}
)

# Update with validation
with sdk.features.edit("feature-123") as f:
    if all(step.completed for step in f.steps):
        f.status = "done"
    else:
        print(f"Warning: {len([s for s in f.steps if not s.completed])} steps incomplete")
```

**Explanation**:
- Status can be: "todo", "in-progress", "done", "blocked"
- Context manager validates and auto-saves
- Batch updates are more efficient for multiple features
- Hooks may track status changes automatically

---

## Add Notes and Activity

**Problem**: Document decisions and progress.

**Solution**:

```python
from wipnote import SDK
from datetime import datetime

sdk = SDK(agent="claude")

# Add activity log entry
with sdk.features.edit("feature-123") as f:
    if not hasattr(f, 'activity_log'):
        f.activity_log = []

    f.activity_log.append({
        "timestamp": datetime.now().isoformat(),
        "agent": "claude",
        "action": "decision",
        "description": "Chose GitHub OAuth over Google due to simpler integration"
    })

# Add notes to description
with sdk.features.edit("feature-123") as f:
    f.content += "\n\n## Decision Log\n"
    f.content += "- 2024-12-24: Selected Supabase for auth provider\n"
    f.content += "- 2024-12-24: Decided to use row-level security\n"
```

**Explanation**:
- Activity log preserves decision context
- Useful for onboarding and retrospectives
- Timestamped entries show progression
- Content field supports markdown

---

## Clone a Feature

**Problem**: Create a similar feature based on an existing one.

**Solution**:

```python
from wipnote import SDK

sdk = SDK(agent="claude")

# Get existing feature
template = sdk.features.get("feature-template")

# Create new feature with same structure
new_feature = sdk.features.create(f"{template.title} (v2)") \
    .set_priority(template.priority) \
    .add_steps([step.description for step in template.steps]) \
    .add_tags(template.tags) \
    .save()

print(f"Cloned to: {new_feature.id}")
```

**Explanation**:
- Useful for repeated workflows
- Preserves structure but creates new ID
- Steps start uncompleted
- Can modify before saving

---

## Archive Completed Features

**Problem**: Clean up completed work while preserving history.

**Solution**:

```python
from wipnote import SDK
import shutil
import os

sdk = SDK(agent="claude")

# Create archive directory
os.makedirs(".wipnote/archive/features", exist_ok=True)

# Get completed features older than 30 days
from datetime import datetime, timedelta
cutoff = datetime.now() - timedelta(days=30)

done_features = sdk.features.where(status="done")
for f in done_features:
    if f.updated < cutoff:
        # Move to archive
        src = f".wipnote/features/{f.id}.html"
        dst = f".wipnote/archive/features/{f.id}.html"
        shutil.move(src, dst)
        print(f"Archived: {f.id}")

# Rebuild index
# wipnote index rebuild
```

**Caution**:
- Archived features won't appear in queries
- Rebuild index after archiving
- Keep archives in git for full history
