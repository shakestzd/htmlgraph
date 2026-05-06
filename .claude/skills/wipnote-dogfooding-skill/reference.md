# Dogfooding Context - Using Wipnote to Build Wipnote

**THIS PROJECT USES WIPNOTE TO DEVELOP WIPNOTE.**

We are dogfooding our own tool. The `.wipnote/` directory in this repo tracks:
- ✅ **Features** - New capabilities we're building (e.g., strategic analytics, track planning)
- ✅ **Sessions** - Our development work (tracked automatically via hooks)
- ✅ **Tracks** - Multi-feature initiatives (e.g., "Planning Workflow")
- ✅ **Development progress** - What's done, in-progress, and planned

## What This Means for AI Agents

### 1. Dual Purpose - Examples ARE Real Usage

When you see workflows in this project:
- ✅ They're **real examples** of Wipnote usage
- ✅ They're **actual tracking** of Wipnote development
- ✅ Learn from them for YOUR projects

```python
# This IS real - we use this to track Wipnote development
sdk = SDK(agent="claude")
feature = sdk.features.create("Add deployment automation")  # Real feature!
```

### 2. General vs Project-Specific

**GENERAL WORKFLOWS** (package these for all users):
- ✅ Feature creation and tracking → SDK already provides this
- ✅ Track planning with TrackBuilder → SDK provides this
- ✅ Strategic analytics (recommend_next_work, find_bottlenecks) → SDK provides this
- ✅ Session management → Hooks provide this
- ⚠️ **Deployment automation** → Should package `deploy-all.sh` pattern
- ⚠️ **Memory file sync** → Should package `sync_memory_files.py` pattern

**PROJECT-SPECIFIC** (only for Wipnote development):
- ❌ Publishing to PyPI (specific to Wipnote package)
- ❌ The specific features in `.wipnote/features/` (our roadmap)
- ❌ Phase 1-6 implementation plan (our project structure)

### 3. Workflows to Package for Users

**TODO - Extract these into the package:**
1. **Deployment Script Pattern** - Generalize `deploy-all.sh` for any Python package
2. **Memory File Sync** - Include `sync_memory_files.py` in the package
3. **Project Initialization** - `wipnote init` should set up `.wipnote/`
4. **Pre-commit Hooks** - Package the git hooks for automatic tracking

**Current Status:**
- ✅ SDK provides feature/track/analytics workflows
- ⚠️ Deployment scripts are project-specific (need to generalize)
- ⚠️ Memory sync is project-specific (need to package)

### 4. How to Read This Codebase

When you see `.wipnote/` in this repo:
- **It's a live example** - This is real usage, not a demo
- **It's our roadmap** - Features here are what we're building
- **Learn from it** - Use these patterns in your projects

**Example:**
```bash
# In THIS repo
ls .wipnote/features/
# → feature-20251221-211348.html  # Real feature we're tracking
# → feat-5f0fca41.html            # Another real feature

# In YOUR project (after using Wipnote)
ls .wipnote/features/
# → Your features will look the same!
```

## Detailed Workflow Analysis

### Feature Tracking in Practice

Wipnote features are tracked using the SDK:

```python
from wipnote import SDK

sdk = SDK(agent="claude")

# Create a feature
feature = sdk.features.create("Add strategic analytics") \
    .set_priority("high") \
    .add_steps([
        "Design analytics API",
        "Implement recommend_next_work()",
        "Implement find_bottlenecks()",
        "Add tests",
        "Document workflows"
    ]) \
    .save()

# Update progress
feature.complete_step(0)
feature.complete_step(1)
feature.save()

# Mark complete
feature.set_status("done").save()
```

**Real Examples:**
- Browse `.wipnote/features/feat-*.html` to see actual features
- Each file shows current status, steps completed, and relationships
- Learn from the structure and adapt for your projects

### Session Tracking in Practice

Sessions are automatically tracked via git hooks:

```python
# Hooks automatically create sessions when you:
# 1. Start work (first git commit)
# 2. Continue work (subsequent commits)
# 3. Complete work (mark feature done)

# Sessions link to parent features
# View: .wipnote/sessions/sess-*.html
```

**Real Examples:**
- Browse `.wipnote/sessions/sess-*.html` for real work sessions
- See how commits are tracked
- Understand parent-child relationships (session → feature)

### Track Planning in Practice

Tracks group related features:

```python
from wipnote import SDK

sdk = SDK(agent="claude")

# Create a track
track = sdk.tracks.create("Planning Workflow") \
    .add_features([
        "feat-abc123",  # TrackBuilder API
        "feat-xyz789",  # Strategic analytics
        "feat-qrs456"   # Dashboard integration
    ]) \
    .save()

# View progress
track.get_progress()  # 60% complete (2/3 features done)
```

**Real Examples:**
- Browse `.wipnote/tracks/` to see multi-feature initiatives
- Study how we group related work
- Apply same planning strategies to your projects

### Analytics Workflows in Practice

Strategic analytics help prioritize work:

```python
from wipnote import SDK

sdk = SDK(agent="claude")

# Get recommendations
recommendations = sdk.analytics.recommend_next_work()
# Returns: Features sorted by priority, considering blockers

# Find bottlenecks
bottlenecks = sdk.analytics.find_bottlenecks()
# Returns: Features blocking the most other work

# Use recommendations to decide what to work on next
for feature in recommendations[:3]:  # Top 3
    print(f"Work on: {feature['title']} (Priority: {feature['priority']})")
```

**Real Examples:**
- We use these analytics to decide what to build next
- Check `.wipnote/spikes/` for research findings
- See how recommendations change as work progresses

## Patterns to Extract and Package

### 1. Deployment Script Pattern

**Current:** `scripts/deploy-all.sh` (Wipnote-specific)

**TODO:** Generalize for any Python package:
```bash
# Generic deployment template
#!/bin/bash
PROJECT_NAME="${1:-myproject}"
VERSION="${2}"

# Quality gates
uv run ruff check --fix
uv run mypy src/
uv run pytest

# Build and publish
uv build
uv publish dist/*

# Tag and push
git tag "v$VERSION"
git push origin main --tags
```

**Package as:** `wipnote deploy` command or template script

### 2. Memory File Sync Pattern

**Current:** `scripts/sync_memory_files.py` (Wipnote-specific)

**TODO:** Generalize for any project with multiple memory files:
```python
# Generic sync pattern
import shutil
from pathlib import Path

def sync_memory_files(source: Path, targets: list[Path]):
    """Sync source-of-truth to multiple target files."""
    for target in targets:
        shutil.copy2(source, target)
        print(f"Synced {source} → {target}")
```

**Package as:** `# sync-docs not yet in Go CLI` generalized to any file patterns

### 3. Project Initialization Pattern

**Current:** Manual setup of `.wipnote/` directory

**TODO:** Automated initialization:
```bash
# Desired workflow
wipnote init

# Creates:
# .wipnote/
#   features/
#   sessions/
#   tracks/
#   spikes/
# .wipnote.json (config)
# index.html (dashboard)
```

**Package as:** `wipnote init` CLI command

### 4. Pre-commit Hooks Pattern

**Current:** Git hooks in `.git/hooks/`

**TODO:** Packaged hooks with installation:
```bash
# Install hooks
wipnote install-hooks

# Hooks automatically:
# 1. Track sessions on commit
# 2. Update feature status
# 3. Link commits to features
# 4. Generate activity logs
```

**Package as:** `wipnote install-hooks` CLI command

## Learning Checklist

Use this checklist to fully understand Wipnote's dogfooding:

### Exploration

- [ ] Browse `.wipnote/features/` directory
- [ ] Open a feature HTML file in browser
- [ ] View `index.html` dashboard
- [ ] Check `.wipnote/sessions/` for work sessions
- [ ] Explore `.wipnote/tracks/` for multi-feature planning
- [ ] Review `.wipnote/spikes/` for research findings

### Understanding

- [ ] Understand feature → session parent-child relationship
- [ ] Understand track → features grouping
- [ ] Understand strategic analytics (recommendations, bottlenecks)
- [ ] Understand the difference between general and project-specific workflows

### Application

- [ ] Create your own feature using SDK
- [ ] Plan a track with multiple features
- [ ] Use `recommend_next_work()` to prioritize
- [ ] Use `find_bottlenecks()` to identify blockers
- [ ] View your work in the dashboard

### Contribution

- [ ] Identify workflows that should be packaged
- [ ] Propose generalizations for project-specific patterns
- [ ] Document lessons learned in spikes
- [ ] Share feedback on dogfooding experience

## Common Dogfooding Questions

**Q: Why dogfood Wipnote?**
A: To validate the tool works in real development, to provide authentic examples, and to surface issues early.

**Q: What makes dogfooding effective?**
A: Using the tool for its intended purpose (project tracking) on a real project (Wipnote development).

**Q: How do I know if something is a demo or real?**
A: If it's in `.wipnote/`, it's real. We don't maintain fake examples.

**Q: Can I use Wipnote workflows without dogfooding?**
A: Yes! Dogfooding is our process. You can use Wipnote without eating your own dog food.

**Q: What if I find issues while dogfooding?**
A: Create a spike to document the issue, then a feature to fix it. That's the process!

**Q: Should I dogfood my own tools?**
A: If feasible, yes. It validates your tool and provides authentic examples.

## Resources

### Documentation

- **CLAUDE.md** - Project overview with dogfooding context
- **AGENTS.md** - SDK usage and workflows
- **GEMINI.md** - Gemini-specific integration
- **scripts/README.md** - Deployment and sync scripts

### Live Examples

- `.wipnote/features/` - Real features in development
- `.wipnote/sessions/` - Real work sessions
- `.wipnote/tracks/` - Real multi-feature planning
- `.wipnote/spikes/` - Real research and findings
- `index.html` - Live dashboard showing everything

### Code

- `src/python/wipnote/` - SDK implementation
- `scripts/deploy-all.sh` - Deployment automation
- `scripts/sync_memory_files.py` - Memory file sync
- `.git/hooks/` - Git hooks for automatic tracking

---

**Remember:** Every example in this project is real. Learn from our actual development process and apply these patterns to your own projects!
