# Wipnote for OpenCode

**MANDATORY instructions for OpenCode AI agents working with Wipnote projects.**

---

## 📚 REQUIRED READING - DO THIS FIRST

**→ READ [AGENTS.md](./AGENTS.md) BEFORE USING HTMLGRAPH**

The AGENTS.md file contains ALL core documentation:
- ✅ **Python SDK Quick Start** - REQUIRED installation and usage
- ✅ **Deployment Instructions** - How to use `deploy-all.sh`
- ✅ **API & CLI Alternatives** - When SDK isn't available
- ✅ **Best Practices** - MUST-FOLLOW patterns for AI agents
- ✅ **Complete Workflow Examples** - Copy these patterns
- ✅ **API Reference** - Full method documentation

**DO NOT proceed without reading AGENTS.md first.**

---

## OpenCode-Specific REQUIREMENTS

### ABSOLUTE RULE: Use SDK, Never Direct File Edits

**CRITICAL: NEVER use file operations on `.wipnote/` HTML files.**

❌ **FORBIDDEN:**
```python
# NEVER DO THIS
Write('/path/to/.wipnote/features/feature-123.html', ...)
Edit('/path/to/.wipnote/sessions/session-456.html', ...)
```

✅ **REQUIRED - Use SDK:**
```python
from wipnote import SDK

# ALWAYS initialize with agent="opencode"
sdk = SDK(agent="opencode")

# Get project summary (DO THIS at session start)
print(sdk.summary(max_items=10))

# Create features (USE builder pattern)
feature = sdk.features.create("Implement Search") \
    .set_priority("high") \
    .add_steps([
        "Design search UI",
        "Implement backend API",
        "Add search indexing",
        "Write tests"
    ]) \
    .save()

print(f"Created: {feature.id}")
```

---

## Quick Start (Python SDK)

### Installation & Setup

```bash
# Install wipnote in your project
pip install wipnote
# or
uv pip install wipnote

# Initialize Wipnote in your project
wipnote init
```

### Basic Workflow

```python
from wipnote import SDK

# Initialize (auto-discovers .wipnote directory)
sdk = SDK(agent="opencode")

# Get project status and recommendations
summary = sdk.summary(max_items=10)
print(summary)

# Check your current workload
workload = sdk.my_work()
print(f"In progress: {workload['in_progress']}")
print(f"Completed: {workload['completed']}")
```

### Feature Management

```python
# Create a new feature
feature = sdk.features.create("User Authentication") \
    .set_priority("high") \
    .set_description("Implement OAuth 2.0 login") \
    .add_steps([
        "Create login endpoint",
        "Add JWT middleware", 
        "Write integration tests"
    ]) \
    .save()

print(f"Created: {feature.id}")

# Work on it (use context manager for auto-save)
with sdk.features.edit(feature.id) as f:
    f.status = "in-progress"
    f.agent_assigned = "opencode"
    f.steps[0].completed = True
    f.steps[0].agent = "opencode"

# Query features
high_priority_todos = sdk.features.where(status="todo", priority="high")
for feat in high_priority_todos:
    print(f"- {feat.id}: {feat.title}")
```

### Get Next Task

```python
# Automatically find and claim next task
task = sdk.next_task(priority="high", auto_claim=True)

if task:
    print(f"Working on: {task.id} - {task.title}")
    
    # Work on it
    with sdk.features.edit(task.id) as f:
        for i, step in enumerate(f.steps):
            if not step.completed:
                # Do the work...
                step.completed = True
                step.agent = "opencode"
                print(f"✓ Completed: {step.description}")
                break
else:
    print("No high-priority tasks available")
```

---

## CLI Commands (When SDK isn't available)

**IMPORTANT:** Always use `uv run` when running wipnote commands.

```bash
# Check project status
uv run wipnote status
uv run wipnote feature list
uv run wipnote session list

# Work with features
uv run wipnote feature start <feature-id>
uv run wipnote feature complete <feature-id>
uv run wipnote feature create "Title"

# Dashboard
uv run wipnote serve
# Open http://localhost:8080
```

---

## Hook Integration

OpenCode automatically runs hooks at key events:

### SessionStart Hook
- Automatically initializes Wipnote session tracking
- Provides project status and feature context
- Shows handoff information from previous agents

### SessionEnd Hook  
- Finalizes session tracking
- Captures handoff context for next agent
- Ensures all work is properly attributed

### Post-Tool Hook
- Tracks tool usage for activity attribution
- Links all work to current session and active features

**Hook Scripts Location:** `packages/opencode-extension/hooks/scripts/`
- All hooks use Python scripts with `uv run` for dependency management
- Core functionality defined in wipnote Python package
- Hooks are just execution layer calling into the package

---

## OpenCode-Specific Patterns

### Agent Detection

Wipnote automatically detects OpenCode environment:

```python
# Agent detection happens automatically
sdk = SDK()  # Will detect agent="opencode"

# Or explicit (recommended for clarity)
sdk = SDK(agent="opencode")
```

Detection markers:
- `OPENCODE_VERSION` environment variable
- `OPENCODE_API_KEY` environment variable  
- `OPENCODE_SESSION_ID` environment variable
- `opencode` in command line arguments
- `.opencode` configuration files

### Feature Creation Decision Framework

**Use this framework for EVERY user request:**

Create a **FEATURE** if ANY apply:
- >30 minutes work
- 3+ files
- New tests needed
- Multi-component impact
- Hard to revert
- Needs docs

Implement **DIRECTLY** if ALL apply:
- Single file
- <30 minutes
- Trivial change
- Easy to revert
- No tests needed

**When in doubt, CREATE A FEATURE.** Over-tracking is better than losing attribution.

---

## Complete Workflow Example

```python
from wipnote import SDK

def opencode_workflow():
    """Complete OpenCode agent workflow."""

    # 1. Initialize (auto-detects opencode)
    sdk = SDK(agent="opencode")

    # 2. Get orientation
    print("=== Project Summary ===")
    print(sdk.summary(max_items=10))

    # 3. Check workload
    workload = sdk.my_work()
    print(f"\nMy Workload:")
    print(f"  In progress: {workload['in_progress']}")
    print(f"  Completed: {workload['completed']}")

    if workload['in_progress'] > 5:
        print("\n⚠️  Already at capacity!")
        return

    # 4. Get next task
    task = sdk.next_task(priority="high", auto_claim=True)

    if not task:
        print("\n✅ No high-priority tasks available")
        return

    print(f"\n=== Working on: {task.title} ===")

    # 5. Work on task
    with sdk.features.edit(task.id) as feature:
        print(f"\nSteps:")
        for i, step in enumerate(feature.steps):
            if step.completed:
                print(f"  ✅ {step.description}")
            else:
                print(f"  ⏳ {step.description}")

                # Do the work here...
                # (implementation details)

                # Mark step complete
                step.completed = True
                step.agent = "opencode"
                print(f"  ✓ Completed: {step.description}")
                break

        # Check if all done
        all_done = all(s.completed for s in feature.steps)
        if all_done:
            feature.status = "done"
            print(f"\n✅ Feature complete: {feature.id}")

if __name__ == "__main__":
    opencode_workflow()
```

---

## Debugging & Quality

### Common Issues

**SDK not finding .wipnote directory:**
```python
# Specify path explicitly
sdk = SDK(directory="/path/to/project/.wipnote", agent="opencode")
```

**Changes not persisting:**
```python
# Use context manager (auto-saves on exit)
with sdk.features.edit("feature-001") as f:
    f.status = "done"

# Or manually save
feature = sdk.features.get("feature-001")
feature.status = "done"
sdk._graph.update(feature)  # Manual save
```

### Quality Gates

Before completing any work:
1. ✅ All features have proper status
2. ✅ All completed steps are marked
3. ✅ Agent attribution is correct
4. ✅ Handoff context provided if needed

---

## Advanced Features

### Handoff Context

```python
# Complete work and hand off with context
with sdk.features.edit("feature-001") as feature:
    feature.steps[0].completed = True

# Create handoff for next agent
if sdk._session_manager:
    sdk._session_manager.create_handoff(
        feature_id="feature-001",
        reason="blocked_on_testing",
        notes="Implementation complete. Needs comprehensive test coverage.",
        agent="opencode"
    )
```

### Parallel Work Coordination

```python
# Create multiple related features
auth_feature = sdk.features.create("User Authentication").save()
session_feature = sdk.features.create("Session Management") \
    .blocked_by(auth_feature.id) \
    .save()

# Batch operations
sdk.features.mark_done([auth_feature.id, session_feature.id])
```

---

## Deployment Integration

When you need to deploy Wipnote changes:

```bash
# Full deployment (includes OpenCode extension)
./scripts/deploy-all.sh 0.7.1

# Documentation only
./scripts/deploy-all.sh --docs-only

# Preview changes
./scripts/deploy-all.sh --dry-run
```

The deployment script automatically updates all integrations including OpenCode.

---

## Plugin Configuration

OpenCode extension configuration in `packages/opencode-extension/opencode-extension.json`:

```json
{
  "name": "wipnote",
  "version": "0.22.0",
  "description": "Wipnote session tracking for OpenCode",
  "contextFileName": "OPENCODE.md",
  "agent": "opencode"
}
```

Hooks configuration in `hooks/hooks.json` defines the three main hooks:
- SessionStart: Initializes tracking and provides context
- SessionEnd: Finalizes session with handoff support
- Post-Tool: Tracks activity attribution

---

## Testing Your Integration

```python
# Test OpenCode detection
from wipnote.agent_detection import detect_agent_name
print(f"Detected agent: {detect_agent_name()}")  # Should be "opencode"

# Test SDK initialization
from wipnote import SDK
sdk = SDK(agent="opencode")
print(f"Agent: {sdk.agent}")
print(f"Summary: {sdk.summary()}")
```

---

## Getting Help

- **Documentation:** [AGENTS.md](./AGENTS.md) - Complete reference
- **API Reference:** `docs/api-reference.md`
- **Quickstart:** `docs/quickstart.md`
- **Dashboard:** Run `uv run wipnote serve` and open http://localhost:8080

**Remember:** Wipnote is tracking this session. All your work will be automatically attributed to the appropriate features and sessions.