# SDK Operations Layer Implementation

**Feature:** `feat-0a49152e`
**Spike:** `spk-562b0417`
**Date:** 2025-01-02

## Overview

Added SDK wrapper methods for the operations layer (Wave 3, Task 1), exposing server, hooks, events, and analytics operations via the SDK for programmatic access.

## Implementation

### 1. Server Operations

```python
# Start server
result = sdk.start_server(port=8080, watch=True, auto_port=False)
print(f"Server at {result.handle.url}")

# Stop server
sdk.stop_server(result.handle)

# Check status
status = sdk.get_server_status(result.handle)
```

**Methods added:**
- `start_server()` - Start Wipnote web server
- `stop_server()` - Stop running server
- `get_server_status()` - Check server status

### 2. Hook Operations

```python
# Install Git hooks
result = sdk.install_hooks(use_copy=False)
print(f"Installed: {result.installed}")

# List hooks
result = sdk.list_hooks()
print(f"Enabled: {result.enabled}")

# Validate configuration
result = sdk.validate_hook_config()
```

**Methods added:**
- `install_hooks()` - Install Git hooks for tracking
- `list_hooks()` - List hook status
- `validate_hook_config()` - Validate hook configuration

### 3. Event Operations

```python
# Export sessions to JSONL
result = sdk.export_sessions(overwrite=False)
print(f"Exported {result.written} sessions")

# Rebuild SQLite index
result = sdk.rebuild_event_index()
print(f"Inserted {result.inserted} events")

# Query events
result = sdk.query_events(
    session_id="sess-123",
    tool="Bash",
    limit=10
)

# Get statistics
stats = sdk.get_event_stats()
```

**Methods added:**
- `export_sessions()` - Export HTML sessions to JSONL
- `rebuild_event_index()` - Rebuild SQLite index
- `query_events()` - Query JSONL event logs
- `get_event_stats()` - Get event statistics

### 4. Analytics Operations

```python
# Analyze session
result = sdk.analyze_session("sess-123")
print(f"Primary work: {result.metrics['primary_work_type']}")

# Analyze project
result = sdk.analyze_project()
print(f"Total sessions: {result.metrics['total_sessions']}")

# Get recommendations
result = sdk.get_work_recommendations()
for rec in result.recommendations:
    print(f"{rec['title']} (score: {rec['score']})")
```

**Methods added:**
- `analyze_session()` - Analyze single session metrics
- `analyze_project()` - Analyze project-wide metrics
- `get_work_recommendations()` - Get work recommendations

## Design Decisions

### 1. Thin Wrapper Pattern

SDK methods are thin wrappers that delegate directly to the operations layer:

```python
def start_server(self, port: int = 8080, ...) -> Any:
    from wipnote.operations import server

    return server.start_server(
        port=port,
        graph_dir=self._directory,
        static_dir=self._directory.parent,
        ...
    )
```

**Benefits:**
- Single source of truth (operations layer)
- No logic duplication
- Easy to maintain
- Type hints preserved via operations layer

### 2. Return Type Annotations

Used `Any` return types for operations methods to avoid circular imports and keep SDK simple:

```python
def analyze_session(self, session_id: str) -> Any:
    """Returns AnalyticsSessionResult..."""
```

**Rationale:**
- Operations layer defines concrete types
- SDK doesn't need to import all operation types
- Docstrings document expected return types
- Users can import types from `wipnote.operations` if needed

### 3. Parameter Mapping

SDK methods match operations layer parameters but add sensible defaults:

```python
# Operations layer
def start_server(*, port: int, graph_dir: Path, ...)

# SDK wrapper
def start_server(self, port: int = 8080, ...)
    # Maps SDK state to operations parameters
    return server.start_server(
        port=port,
        graph_dir=self._directory,  # From SDK state
        ...
    )
```

### 4. Path Handling

SDK automatically provides `graph_dir` and `project_dir` from internal state:

```python
# User doesn't need to know about paths
sdk.install_hooks()

# SDK maps to operations layer
hooks.install_hooks(project_dir=self._directory.parent)
```

## Documentation

### 1. Help System Integration

Added operations to main help text:

```python
sdk.help()  # Lists operations methods
sdk.help('operations')  # Detailed operations help
```

### 2. Docstring Examples

All methods have comprehensive docstrings with:
- Parameter descriptions
- Return type documentation
- Usage examples
- Related methods ("See also" sections)

### 3. Discovery via `__dir__`

Added operations methods to priority list for auto-complete:

```python
priority = [
    # ...existing methods...
    # Operations
    "start_server",
    "install_hooks",
    "export_sessions",
    "analyze_project",
]
```

## Testing

### Integration Tests

Created `tests/test_sdk_operations.py` with 17 tests covering:

1. **Server Operations** (3 tests)
   - `test_start_server` - Verifies wrapper calls operations layer correctly
   - `test_stop_server` - Verifies stop functionality
   - `test_get_server_status` - Verifies status checking

2. **Hook Operations** (3 tests)
   - `test_install_hooks` - Verifies installation
   - `test_list_hooks` - Verifies listing
   - `test_validate_hook_config` - Verifies validation

3. **Event Operations** (4 tests)
   - `test_export_sessions` - Verifies export
   - `test_rebuild_event_index` - Verifies index rebuild
   - `test_query_events` - Verifies querying
   - `test_get_event_stats` - Verifies statistics

4. **Analytics Operations** (3 tests)
   - `test_analyze_session` - Verifies session analysis
   - `test_analyze_project` - Verifies project analysis
   - `test_get_work_recommendations` - Verifies recommendations

5. **SDK Integration** (4 tests)
   - `test_sdk_has_all_operation_methods` - Verifies all methods exist
   - `test_operations_in_help` - Verifies help text
   - `test_operations_help_topic` - Verifies help topic
   - `test_operations_in_dir` - Verifies __dir__ output

**All tests pass:** ✅ 17/17

### Type Checking

Verified with mypy:
```bash
uv run mypy src/python/wipnote/sdk.py
# Success: no issues found
```

## Quality Gates Met

- [x] All operations exposed via SDK
- [x] Docstrings with examples for each method
- [x] Integration tests pass (17/17)
- [x] Type hints correct (mypy passes)
- [x] Imports from `wipnote.operations` only
- [x] Help system updated
- [x] __dir__ method updated

## Usage Examples

### Starting a Server

```python
from wipnote import SDK

sdk = SDK(agent="claude")

# Start server with auto-port selection
result = sdk.start_server(port=8080, auto_port=True, watch=True)
print(f"Server running at {result.handle.url}")

# Open browser to view graph
import webbrowser
webbrowser.open(result.handle.url)

# Stop when done
sdk.stop_server(result.handle)
```

### Setting Up Hooks

```python
from wipnote import SDK

sdk = SDK(agent="claude")

# Install hooks
result = sdk.install_hooks()
print(f"Installed: {', '.join(result.installed)}")

if result.warnings:
    print(f"Warnings: {result.warnings}")

# Verify installation
hooks = sdk.list_hooks()
print(f"Enabled hooks: {hooks.enabled}")
```

### Analyzing Project

```python
from wipnote import SDK

sdk = SDK(agent="claude")

# Get project analytics
result = sdk.analyze_project()

print(f"Total sessions: {result.metrics['total_sessions']}")
print(f"Work distribution: {result.metrics['work_distribution']}")
print(f"Spike-to-feature ratio: {result.metrics['spike_to_feature_ratio']:.2f}")

# Get recommendations
recs = sdk.get_work_recommendations()
print("\nRecommended work:")
for rec in recs.recommendations[:5]:
    print(f"  {rec['title']} (score: {rec['score']:.2f})")
    print(f"    Reasons: {', '.join(rec['reasons'])}")
```

## Related Files

**Modified:**
- `src/python/wipnote/sdk.py` - Added 13 new methods, updated help system

**Created:**
- `tests/test_sdk_operations.py` - Integration tests (17 tests)
- `docs/sdk-operations-implementation.md` - This document

**Dependencies:**
- `src/python/wipnote/operations/__init__.py` - Operations layer exports
- `src/python/wipnote/operations/server.py` - Server operations
- `src/python/wipnote/operations/hooks.py` - Hook operations
- `src/python/wipnote/operations/events.py` - Event operations
- `src/python/wipnote/operations/analytics.py` - Analytics operations

## Next Steps

With SDK wrapper methods complete, users can now:

1. **Start servers programmatically** instead of CLI only
2. **Install hooks via SDK** for automated setup
3. **Export and query events** for custom analytics
4. **Analyze sessions and projects** for insights

**Future enhancements:**
- Add async variants for long-running operations
- Add progress callbacks for exports/rebuilds
- Add batch operations (e.g., analyze multiple sessions)
- Add caching for frequently accessed analytics
