# Migration Guide: SDK 0.9 Module Reorganization

## Overview

Wipnote SDK 0.9 introduces a major refactoring to improve code organization and maintainability. This guide helps contributors adapt to the new module structure.

**TL;DR:** Modules have been reorganized into logical directories (`builders/`, `collections/`, `analytics/`), but **public imports remain unchanged**. Most code will continue to work without modification.

---

## What Changed?

### Module Reorganization

**Before (0.8.x):**
```
wipnote/
├── sdk.py
├── analytics.py               # Analytics class
├── dependency_analytics.py    # DependencyAnalytics class
├── cli_analytics.py
├── track_builder.py           # TrackBuilder + TrackCollection
├── builders/
│   ├── base.py
│   ├── feature.py
│   └── spike.py
└── collections/
    ├── base.py
    ├── feature.py
    └── spike.py
```

**After (0.9.x):**
```
wipnote/
├── sdk.py                     # Main entry point
├── track_builder.py           # TrackCollection only
├── builders/                  # All builders organized here
│   ├── base.py
│   ├── feature.py
│   ├── spike.py
│   └── track.py              # TrackBuilder (moved from track_builder.py)
├── collections/               # All collections
│   ├── base.py               # Added start(), complete() methods
│   ├── feature.py            # Added set_primary() method
│   └── spike.py
└── analytics/                 # All analytics organized here
    ├── __init__.py           # Exports Analytics, DependencyAnalytics
    ├── work_type.py          # Analytics class (from analytics.py)
    ├── dependency.py         # DependencyAnalytics (from dependency_analytics.py)
    └── cli.py                # CLI helpers (from cli_analytics.py)
```

### SDK Integration

**SessionManager Integration:**
- SDK now owns a `SessionManager` instance
- Session operations go through SDK methods
- Collections delegate to SessionManager for tracking

**Collection Enhancements:**
- `BaseCollection` gained `start()`, `complete()` methods
- `FeatureCollection` gained `set_primary()` method
- All methods delegate to SessionManager for smart tracking

---

## Migration Steps

### For Users (Public API)

**✅ No changes required!** Public imports work exactly as before:

```python
# These still work (unchanged)
from wipnote import SDK, Analytics, DependencyAnalytics
from wipnote.builders import FeatureBuilder, SpikeBuilder, TrackBuilder
from wipnote.collections import BaseCollection, FeatureCollection

sdk = SDK(agent="claude")
# All existing code continues to work
```

### For Contributors (Internal Imports)

**If you import from internal modules**, update your imports:

#### Analytics Imports

**Before:**
```python
from wipnote.analytics import Analytics
from wipnote.dependency_analytics import DependencyAnalytics
```

**After:**
```python
from wipnote.analytics import Analytics, DependencyAnalytics
# Or direct module imports:
from wipnote.analytics.work_type import Analytics
from wipnote.analytics.dependency import DependencyAnalytics
```

#### TrackBuilder Imports

**Before:**
```python
from wipnote.track_builder import TrackBuilder, TrackCollection
```

**After:**
```python
from wipnote.builders.track import TrackBuilder  # Moved location
from wipnote.track_builder import TrackCollection  # Unchanged
# Or use the convenience import:
from wipnote.track_builder import TrackBuilder  # Re-exported for compatibility
```

#### SessionManager Access

**Before (direct import):**
```python
from wipnote.session_manager import SessionManager

manager = SessionManager(graph_dir)
manager.start_feature(feature_id="feat-123", agent="claude")
```

**After (via SDK):**
```python
from wipnote import SDK

sdk = SDK(agent="claude", directory=graph_dir)
sdk.features.start("feat-123")  # Uses SessionManager internally
# Or direct access if needed:
sdk.session_manager.start_feature(feature_id="feat-123", collection="features", agent="claude")
```

---

## Breaking Changes

### Internal Imports Only

**✅ Public API unchanged** - Most users won't be affected.

**⚠️ Internal module paths changed:**

1. **`wipnote.analytics` → `wipnote.analytics.work_type`**
   - Public import `from wipnote import Analytics` still works
   - Direct import path changed for internal use

2. **`wipnote.dependency_analytics` → `wipnote.analytics.dependency`**
   - Public import `from wipnote import DependencyAnalytics` still works
   - Direct import path changed for internal use

3. **`wipnote.track_builder.TrackBuilder` → `wipnote.builders.track.TrackBuilder`**
   - Re-exported in `track_builder.py` for backward compatibility
   - Preferred import: `from wipnote.builders import TrackBuilder`

### CLI Changes

**✅ No breaking changes** - CLI commands work exactly as before.

The CLI was refactored to use SDK instead of SessionManager directly, but all command signatures remain unchanged.

---

## New Features

### Collection Methods

**start(), complete(), claim(), release():**

```python
sdk = SDK(agent="claude")

# Start working on any node type
sdk.features.start("feat-123")
sdk.bugs.start("bug-456")
sdk.chores.start("chore-789")

# Complete nodes
sdk.features.complete("feat-123")

# Claim/release nodes
sdk.features.claim("feat-123")
sdk.features.release("feat-123")
```

**set_primary() for features:**

```python
# Set primary feature for attribution
sdk.features.set_primary("feat-123")
```

### SDK Session Methods

```python
sdk = SDK(agent="claude")

# Start/end sessions via SDK
sdk.start_session(title="Feature implementation")
sdk.end_session(session_id="...", handoff_notes="Completed auth module")

# Get project status
status = sdk.get_status()
# Returns: {'wip_count': 2, 'wip_limit': 3, 'active_features': [...], ...}
```

---

## Testing Your Code

**Run the test suite:**

```bash
uv run pytest tests/ -v
```

**Expected results:**
- 411+ tests should pass
- 3 pre-existing failures in `test_agent_routing.py` (unrelated to SDK changes)

**Check your imports:**

```python
# Verify all imports work
from wipnote import SDK, Analytics, DependencyAnalytics
from wipnote.builders import FeatureBuilder, SpikeBuilder, TrackBuilder
from wipnote.collections import BaseCollection, FeatureCollection
from wipnote.models import SpikeType, MaintenanceType, WorkType

print("✓ All imports successful")
```

---

## Benefits of This Refactoring

1. **Better Organization** - Logical grouping of related modules
2. **Separation of Concerns** - Clear boundaries between components
3. **Easier Maintenance** - Modules are easier to find and modify
4. **Consistent API** - All collections have the same interface
5. **Single Entry Point** - SDK coordinates all operations
6. **Improved Testing** - Modules can be tested in isolation

---

## Need Help?

- **Documentation:** See `docs/api/sdk.md` for complete SDK reference
- **Examples:** Check `docs/examples/` for updated code samples
- **Issues:** Report problems at https://github.com/shakestzd/wipnote/issues

---

## Summary

**For most users:** No changes needed - existing code continues to work.

**For contributors:** Update internal imports to use new module paths.

**All changes are backward compatible** via re-exports in main `__init__.py`.
