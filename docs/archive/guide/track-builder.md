# TrackBuilder

The TrackBuilder provides a fluent API for creating tracks with specs and plans in a single command. No manual file creation, ID generation, or path management needed.

## Overview

TrackBuilder is the recommended way to create tracks in Wipnote. It:

- Auto-generates track IDs and timestamps
- Creates HTML files in the correct directory structure
- Validates all input with Pydantic schemas
- Parses time estimates from task descriptions
- Provides a clean, readable API

## Basic Usage

### Minimal Track

```bash
# Create a simple track via CLI
wipnote track new "Simple Feature" --priority medium
# Creates: .wipnote/tracks/trk-xxxxxxxx/index.html
```

### Track with Specification

The TrackBuilder fluent API is available via the Python library for programmatic track creation with spec and plan. For most workflows, use the CLI or dashboard:

```bash
wipnote track new "User Authentication" --priority high
# Then open .wipnote/tracks/trk-xxxxxxxx/ to add spec and plan HTML files
```

### Track with Implementation Plan

```bash
wipnote track new "Database Migration" --priority critical
# Add phases by editing plan.html in the track directory
```

### Complete Track (Spec + Plan)

```bash
wipnote track new "API Rate Limiting" --priority high
# ✓ Created track: trk-a1b2c3d4
# Open .wipnote/tracks/trk-a1b2c3d4/ to add spec and plan
```

## API Reference

### CLI Options

```bash
wipnote track new "Title" --priority <low|medium|high|critical>
wipnote track list
wipnote track show trk-<id>
```

### TrackBuilder (Python Library)

The TrackBuilder fluent API is available in the Python library for programmatic track creation. It accepts:

- `.title(str)` — Track title (required)
- `.description(str)` — Track description (optional)
- `.priority(str)` — Priority: `"low"`, `"medium"`, `"high"`, `"critical"`
- `.with_spec(overview, context, requirements, acceptance_criteria)` — Add spec content
- `.with_plan_phases([(phase_name, [task_descriptions]), ...])` — Add phased plan
- `.create()` — Build and write all HTML files

**Time Estimates in plan tasks:** Include `(Xh)` in task descriptions:
- `"Implement auth (3h)"` → 3 hours
- `"Write tests (1.5h)"` → 1.5 hours
- `"Deploy"` → No estimate

## File Structure

TrackBuilder creates a directory with up to three HTML files:

```
.wipnote/tracks/trk-xxxxxxxx/
├── index.html    # Track metadata with links to spec/plan
├── spec.html     # Specification (if with_spec() used)
└── plan.html     # Implementation plan (if with_plan_phases() used)
```

All files are fully styled and can be opened in any browser.

## When to Use TrackBuilder

**Create a track when:**

- Work involves **3+ features**
- **Multi-phase** implementation needed
- Need **coordination** across sessions
- Requires **detailed planning** upfront

**Implement directly when:**

- Single feature, straightforward work
- No need for planning
- Quick fix or enhancement

## Workflow Example

### 1. Create the Track

```bash
wipnote track new "Multi-Agent Collaboration" --priority high
# Note the track ID: trk-a1b2c3d4
```

### 2. Create Features from Phases

```bash
# Create a feature for each phase, linked to the track
wipnote feature create "Phase 1: Agent Claiming" --priority high --track trk-a1b2c3d4
wipnote feature create "Phase 2: Agent Handoffs" --priority high --track trk-a1b2c3d4
wipnote feature create "Phase 3: Testing" --priority medium --track trk-a1b2c3d4
```

### 3. Work on Features

```bash
# Start working on Phase 1
wipnote feature start {phase1.id}

# Complete steps as you go
# Features are automatically attributed to the track
```

## Error Handling

**Missing Title:** Always provide a title — it is required.

**Invalid Priority:** Use valid requirement priority values: `"must-have"`, `"should-have"`, `"nice-to-have"`.

## Tips

1. **Auto-generated IDs** - Never manually create track IDs
2. **Timestamps** - Created/updated timestamps are automatic
3. **File paths** - Builder handles all path generation
4. **HTML generation** - Spec and Plan models convert to HTML automatically
5. **Estimates** - Include `(Xh)` in task descriptions
6. **Validation** - Pydantic validates all fields before HTML generation

## Next Steps

- [Features & Tracks Guide](features-tracks.md) - Linking features to tracks
- [Sessions Guide](sessions.md) - Session tracking and attribution
- [API Reference](../api/track-builder.md) - Complete API documentation
