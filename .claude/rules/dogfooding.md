# Dogfooding Context - Using HtmlGraph to Build HtmlGraph

**THIS PROJECT USES HTMLGRAPH TO DEVELOP HTMLGRAPH.**

We are dogfooding our own tool. The `.htmlgraph/` directory in this repo tracks:
- ✅ **Features** - New capabilities we're building (e.g., strategic analytics, track planning)
- ✅ **Sessions** - Our development work (tracked automatically via hooks)
- ✅ **Tracks** - Multi-feature initiatives (e.g., "Planning Workflow")
- ✅ **Development progress** - What's done, in-progress, and planned

## What This Means for AI Agents

### 1. Dual Purpose - Examples ARE Real Usage

When you see workflows in this project:
- ✅ They're **real examples** of HtmlGraph usage
- ✅ They're **actual tracking** of HtmlGraph development
- ✅ Learn from them for YOUR projects

```bash
# This IS real - we use this to track HtmlGraph development
htmlgraph feature create "Add deployment automation"  # Real feature!
```

### 2. General vs Project-Specific

**GENERAL WORKFLOWS** (package these for all users):
- Feature creation and tracking → CLI provides this
- Track planning → CLI provides this
- Strategic analytics (recommend, bottlenecks) → CLI provides this
- Session management → Hooks provide this

**PROJECT-SPECIFIC** (only for HtmlGraph development):
- The specific features in `.htmlgraph/features/` (our roadmap)
- Phase 1-6 implementation plan (our project structure)

### 4. How to Read This Codebase

When you see `.htmlgraph/` in this repo:
- **It's a live example** - This is real usage, not a demo
- **It's our roadmap** - Features here are what we're building
- **Learn from it** - Use these patterns in your projects

**Example:**
```bash
# In THIS repo
ls .htmlgraph/features/
# → feature-20251221-211348.html  # Real feature we're tracking
# → feat-5f0fca41.html            # Another real feature

# In YOUR project (after using HtmlGraph)
ls .htmlgraph/features/
# → Your features will look the same!
```
