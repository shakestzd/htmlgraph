# Wipnote 0.7.0 Release Notes

## 🚀 Major Release: Planning Workflow & Strategic Analytics

Released: December 22, 2025

### 📦 Installation

```bash
# Upgrade Python package
uv pip install --upgrade wipnote

# Or with pip
pip install --upgrade wipnote

# Verify version
python -c "import wipnote; print(wipnote.__version__)"  # Should show 0.7.0
```

### 🆕 New Features

#### 1. Dependency-Based Strategic Analytics

AI agents can now make data-driven decisions about what to work on:

```python
from wipnote import SDK

sdk = SDK(agent="claude")

# Get smart recommendations
recs = sdk.recommend_next_work(agent_count=3)
# Returns: Top tasks with scores and reasoning

# Find bottlenecks
bottlenecks = sdk.find_bottlenecks(top_n=5)
# Returns: Tasks blocking the most downstream work

# Get parallel work opportunities
parallel = sdk.get_parallel_work(max_agents=5)
# Returns: Tasks that can be done simultaneously

# Assess project risks
risks = sdk.assess_risks()
# Returns: SPOFs, circular dependencies, orphaned nodes

# Analyze impact
impact = sdk.analyze_impact("feature-001")
# Returns: What completing this task will unlock
```

**Benefits:**
- ✅ Data-driven work prioritization
- ✅ Identify critical path automatically
- ✅ Optimize team coordination
- ✅ Prevent wasted effort on blocked work

#### 2. Integrated Planning Workflow

Complete workflow from strategic analysis to implementation:

```python
# Step 1: Get recommendations
recs = sdk.recommend_next_work(agent_count=1)
top = recs[0]

# Step 2: Start planning with context
plan = sdk.smart_plan(
    top['title'],
    create_spike=True,
    timebox_hours=4.0
)
# Analyzes project, creates planning spike

# Step 3: Complete spike research
# (Do the research, complete steps)

# Step 4: Create track from findings
track_info = sdk.create_track_from_plan(
    title="User Authentication",
    description="OAuth 2.0 + JWT",
    spike_id=plan['spike_id'],
    requirements=[
        ("OAuth integration", "must-have"),
        ("JWT tokens", "must-have")
    ],
    phases=[
        ("Phase 1", ["Task 1 (2h)", "Task 2 (3h)"]),
        ("Phase 2", ["Task 3 (4h)"])
    ]
)

# Step 5: Create features and implement
feature = sdk.features.create("OAuth") \
    .set_track(track_info['track_id']) \
    .add_steps(["Step 1", "Step 2"]) \
    .save()
```

**New SDK Methods:**
- `smart_plan()` - Context-aware planning entry point
- `start_planning_spike()` - Create research spikes
- `create_track_from_plan()` - Convert findings to tracks

#### 3. DRY Command System

**Single source of truth** for all slash commands:

```
packages/common/
├── command_definitions/          # 14 YAML files
│   ├── plan.yaml
│   ├── spike.yaml
│   ├── recommend.yaml
│   └── ... (11 more)
└── generators/
    └── generate_commands.py      # Generates platform files
```

**Benefits:**
- ✅ Update once, regenerate for all platforms
- ✅ Perfect consistency across Claude Code, Codex, Gemini
- ✅ ~2000+ lines of duplication eliminated
- ✅ Imperative language tells agents what to DO

**Regenerate Commands:**
```bash
uv run python packages/common/generators/generate_commands.py
```

#### 4. New Slash Commands

Available on **all platforms** (Claude Code, Codex, Gemini):

**`/wipnote:plan`** - Smart planning workflow
```bash
/wipnote:plan "User authentication system"
/wipnote:plan "Real-time notifications" --timebox 3
/wipnote:plan "Simple fix" --no-spike
```

**`/wipnote:spike`** - Create research spike
```bash
/wipnote:spike "Research OAuth providers"
/wipnote:spike "Investigate caching" --timebox 2
```

**`/wipnote:recommend`** - Get recommendations
```bash
/wipnote:recommend
/wipnote:recommend --count 5
```

#### 5. Enhanced Start Command

`/wipnote:start` now includes strategic analytics:

**What it shows:**
- ✅ Basic status (completion %, active features)
- ✅ Previous session summary
- ✅ Current feature progress
- **🆕 Bottlenecks** (count, impact scores)
- **🆕 Smart recommendations** (with scores and reasons)
- **🆕 Parallel work capacity**

**Smart suggestions:**
```
Based on strategic analysis, I recommend:
1. **User Authentication** (score: 10.0)
   - Why: High priority, Directly unblocks 2 features
2. Continue current feature
3. Create new work (`/wipnote:plan`)
```

### 📊 Statistics

| Metric | Count |
|--------|-------|
| New SDK Methods | 8 |
| New Slash Commands | 3 |
| Total Commands (All Platforms) | 14 |
| YAML Definitions | 14 |
| Generated Command Files | 42 |
| Lines of Code Changed | 6,868 |
| Files Created | 53 |
| Platforms Supported | 3 |

### 📚 New Documentation

| Document | Purpose |
|----------|---------|
| `docs/PLANNING_WORKFLOW.md` | Complete planning workflow guide |
| `docs/AGENT_STRATEGIC_PLANNING.md` | Strategic analytics API reference |
| `packages/common/README.md` | DRY command system documentation |
| `packages/common/IMPLEMENTATION_SUMMARY.md` | Technical implementation details |

### 🔧 Breaking Changes

**None!** This release is fully backward compatible.

All existing SDK methods and commands continue to work exactly as before.

### 🐛 Bug Fixes

- Fixed Pylance type warnings in dependency analytics
- Fixed spike creation with SpikeType import
- Updated TrackBuilder to support planning spike references

### ⚡ Performance

- Analytics queries are O(N) or O(N log N)
- No impact on existing operations
- Agent-friendly dict format minimizes token usage

### 🎯 Migration Guide

**For Python SDK Users:**

```python
# Before (still works)
sdk = SDK(agent="claude")
feature = sdk.features.create("Title").save()

# After (new capabilities)
sdk = SDK(agent="claude")

# Get recommendations first
recs = sdk.recommend_next_work(agent_count=1)

# Plan with context
plan = sdk.smart_plan(recs[0]['title'])

# Create track from plan
track = sdk.create_track_from_plan(...)
```

**For Claude Code Users:**

```bash
# New commands available immediately
/wipnote:recommend
/wipnote:plan "New feature"
/wipnote:spike "Research topic"

# Enhanced start command
/wipnote:start  # Now shows analytics!
```

**For Plugin Developers:**

Update command definitions:
```bash
# Edit YAML
vim packages/common/command_definitions/my_command.yaml

# Regenerate
uv run python packages/common/generators/generate_commands.py

# Test on all platforms
```

### 🔮 What's Next (0.8.0)

- CLI commands for dependency analytics
- Auto-integration of command sections into skill docs
- YAML schema validation
- Test generation from command definitions
- CI/CD sync checks

### 📦 Upgrade Instructions

**Python Package:**
```bash
uv pip install --upgrade wipnote
```

**Claude Code Plugin:**
Plugin will auto-update on next Claude Code restart.
Or manually: Files already updated in local marketplace.

**Codex Skill:**
Files updated in `packages/codex-skill/`.
Re-import skill or restart to pick up changes.

**Gemini Extension:**
Files updated in `packages/gemini-extension/`.
Reload extension to pick up changes.

### 🙏 Acknowledgments

Built with Claude Sonnet 4.5 using Claude Code.

This release demonstrates the power of AI-assisted development:
- 6,868 lines of code written
- 53 files created
- 3 platforms supported
- Complete planning workflow implemented
- All in a single development session

### 📞 Support

- GitHub Issues: https://github.com/shakestzd/wipnote/issues
- Documentation: `docs/` directory
- Examples: `demo_agent_planning.py`, `demo_real_project_analytics.py`

---

**Full Changelog:** https://github.com/shakestzd/wipnote/compare/v0.6.1...v0.7.0
