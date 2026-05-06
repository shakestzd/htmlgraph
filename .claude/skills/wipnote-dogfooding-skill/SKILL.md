# Wipnote Dogfooding Skill

Use this skill to understand how Wipnote dogfoods itself and learn from real examples.

**Trigger keywords:** dogfooding, examples, learn, patterns, real usage, wipnote development, self-tracking

---

## Overview

Wipnote uses itself to track its own development. This creates a unique learning opportunity: every workflow you see is both a real-world example AND actual development tracking.

**Key Insight:** The `.wipnote/` directory contains real features, sessions, and tracks we're using to build Wipnote. Learn from them!

---

## Dual Purpose - Examples ARE Real Usage

When you see workflows in this project:
- ✅ They're **real examples** of Wipnote usage
- ✅ They're **actual tracking** of Wipnote development
- ✅ Learn from them for YOUR projects

```python
# This IS real - we use this to track Wipnote development
sdk = SDK(agent="claude")
feature = sdk.features.create("Add deployment automation")  # Real feature!
```

**What this means:**
- Browse `.wipnote/features/` to see real features in development
- Browse `.wipnote/sessions/` to see how we track work
- Browse `.wipnote/tracks/` to see multi-feature planning
- Copy these patterns for your own projects

---

## General vs Project-Specific

Understanding what's reusable vs specific to Wipnote development:

### GENERAL WORKFLOWS (package for all users)

These patterns work for ANY project using Wipnote:

- ✅ **Feature creation and tracking** → SDK already provides this
- ✅ **Track planning with TrackBuilder** → SDK provides this
- ✅ **Strategic analytics** (recommend_next_work, find_bottlenecks) → SDK provides this
- ✅ **Session management** → Hooks provide this
- ⚠️ **Deployment automation** → Should package `deploy-all.sh` pattern
- ⚠️ **Memory file sync** → Should package `sync_memory_files.py` pattern

### PROJECT-SPECIFIC (only for Wipnote itself)

These are specific to building Wipnote:

- ❌ Publishing to PyPI (specific to Wipnote package)
- ❌ The specific features in `.wipnote/features/` (our roadmap)
- ❌ Phase 1-6 implementation plan (our project structure)

---

## How to Read This Codebase

### The `.wipnote/` Directory is a Live Example

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

### What to Learn From

**1. Feature Tracking Patterns**
- Open any `.wipnote/features/feat-*.html` file
- See how we structure features, steps, and status
- Copy the pattern for your own features

**2. Session Tracking**
- View `.wipnote/sessions/sess-*.html` files
- See how work sessions are automatically tracked
- Understand the parent-child relationship (sessions → features)

**3. Track Planning**
- View `.wipnote/tracks/` directory
- See how we group related features into tracks
- Learn multi-feature planning strategies

**4. Analytics Workflows**
- See how we use `recommend_next_work()` to prioritize
- See how we use `find_bottlenecks()` to identify issues
- Apply these same analytics to your projects

---

## Workflows to Package for Users

These are patterns we need to extract and generalize:

### TODO - Extract into Package

1. **Deployment Script Pattern** - Generalize `deploy-all.sh` for any Python package
2. **Memory File Sync** - Include `sync_memory_files.py` in the package
3. **Project Initialization** - `wipnote init` should set up `.wipnote/`
4. **Pre-commit Hooks** - Package the git hooks for automatic tracking

### Current Status

- ✅ SDK provides feature/track/analytics workflows
- ⚠️ Deployment scripts are project-specific (need to generalize)
- ⚠️ Memory sync is project-specific (need to package)

---

## Quick Reference

### Learning Commands

```bash
# View all features we're tracking
ls .wipnote/features/

# View recent sessions
ls -lt .wipnote/sessions/ | head

# View track planning
ls .wipnote/tracks/

# Open the dashboard to see everything
open index.html
```

### Example Workflows to Study

```python
# 1. Feature creation (see any feat-*.html)
sdk = SDK(agent="claude")
feature = sdk.features.create("Your feature name") \
    .set_priority("high") \
    .add_steps(["Step 1", "Step 2"]) \
    .save()

# 2. Track planning (see track-*.html)
track = sdk.tracks.create("Your track name") \
    .add_features(["feat-abc123", "feat-xyz789"]) \
    .save()

# 3. Strategic analytics
next_work = sdk.analytics.recommend_next_work()
bottlenecks = sdk.analytics.find_bottlenecks()
```

---

## Common Questions

**Q: Are the features in `.wipnote/features/` just examples?**
A: No! They're the actual features we're building for Wipnote. They're real, not demos.

**Q: Can I copy these patterns for my project?**
A: Yes! That's the whole point. The patterns are general-purpose.

**Q: What's the difference between a feature and a session?**
A: Features are what you're building. Sessions are when you work on them. Sessions link to features.

**Q: How do I know what's reusable vs Wipnote-specific?**
A: Check the "General vs Project-Specific" section above. SDK workflows are general, PyPI publishing is specific.

---

## See Also

- **reference.md** - Full dogfooding context and details
- **CLAUDE.md** - Project overview with dogfooding section
- **AGENTS.md** - SDK documentation with workflow examples
- `.wipnote/` - Live examples of real usage

---

## When to Use This Skill

Activate this skill when:
- Learning how to use Wipnote effectively
- Looking for real-world examples
- Understanding the difference between demos and real usage
- Planning your own Wipnote workflows
- Extracting patterns for your projects
- Contributing to Wipnote development

**Remember:** Every example you see is real. Learn from our actual development process!
