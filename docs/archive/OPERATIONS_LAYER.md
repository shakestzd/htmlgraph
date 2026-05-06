# Operations Layer Architecture

## Overview

Wipnote uses a shared operations layer that both CLI and SDK call. This eliminates code duplication and ensures consistent behavior across all interfaces.

```
CLI ────┐
        ├──→ Operations Layer (shared backend)
SDK ────┘
```

## Benefits

- **No duplication** - CLI and SDK share same code
- **Consistent behavior** - Same results regardless of interface
- **Single source of truth** - Operations layer owns business logic
- **Easier testing** - Test operations once, both CLI and SDK benefit
- **Better maintainability** - Fix bugs in one place

## Architecture Layers

```
┌─────────────────────────────────────────────┐
│           User Interfaces                   │
│  CLI (argparse) │ SDK (Python API)          │
└────────┬────────────────────┬────────────────┘
         │                    │
         └────────┬───────────┘
                  ▼
┌─────────────────────────────────────────────┐
│         Operations Layer (Shared)           │
│  • server.py   - Server lifecycle           │
│  • hooks.py    - Git hooks management       │
│  • events.py   - Event log indexing         │
│  • analytics.py - Analytics operations      │
└────────┬────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────┐
│         Core Wipnote Library              │
│  • graph.py    - Graph operations           │
│  • models.py   - Data models                │
│  • session_manager.py - Session tracking    │
└─────────────────────────────────────────────┘
```

## Operations Modules

### operations/server.py

Server lifecycle operations.

**Functions:**
- `start_server()` - Start Wipnote server with configuration
- `stop_server()` - Stop running server gracefully
- `get_server_status()` - Check server status

**Return Types:**
- `ServerStartResult` - Contains handle, warnings, config used
- `ServerStatus` - Current server state
- `ServerHandle` - Reference to running server

**Example:**
```python
from wipnote.operations import server

result = server.start_server(
    port=8080,
    graph_dir=Path(".wipnote"),
    static_dir=Path("."),
    watch=True,
    auto_port=False
)

print(f"Server started at {result.handle.url}")
if result.warnings:
    print(f"Warnings: {result.warnings}")
```

### operations/hooks.py

Git hooks management operations.

**Functions:**
- `install_hooks()` - Install Wipnote git hooks
- `list_hooks()` - List enabled/disabled/missing hooks
- `validate_hook_config()` - Validate hook configuration

**Return Types:**
- `HookInstallResult` - Installation details (installed, skipped, warnings)
- `HookListResult` - Hook status lists
- `HookValidationResult` - Validation errors and warnings

**Example:**
```python
from wipnote.operations import hooks
from pathlib import Path

result = hooks.install_hooks(
    project_dir=Path("."),
    use_copy=False  # Use symlinks
)

print(f"Installed: {result.installed}")
print(f"Skipped: {result.skipped}")
```

### operations/events.py

Event log indexing and querying operations.

**Functions:**
- `export_sessions()` - Export session events from JSONL logs
- `query_events()` - Query events with filters
- `rebuild_index()` - Rebuild SQLite index from event logs
- `get_event_stats()` - Get event log statistics

**Return Types:**
- `EventExportResult` - Export statistics
- `EventQueryResult` - Query results
- `EventRebuildResult` - Rebuild statistics
- `EventStats` - Event log statistics

**Example:**
```python
from wipnote.operations import events
from pathlib import Path

stats = events.get_event_stats(graph_dir=Path(".wipnote"))
print(f"Total events: {stats.total_events}")
print(f"Total sessions: {stats.total_sessions}")
```

### operations/analytics.py

Analytics operations for sessions and projects.

**Functions:**
- `analyze_session()` - Analyze single session
- `analyze_project()` - Analyze entire project

**Return Types:**
- `AnalyticsSessionResult` - Session analysis
- `AnalyticsProjectResult` - Project-wide analysis

**Example:**
```python
from wipnote.operations import analytics
from pathlib import Path

result = analytics.analyze_project(
    graph_dir=Path(".wipnote")
)

print(f"Total features: {result.total_features}")
print(f"Recommendations: {len(result.recommendations)}")
```

## Using Operations from CLI

The CLI uses operations internally:

```python
# In wipnote/cli.py

def cmd_serve(args):
    from wipnote.operations import server

    result = server.start_server(
        port=args.port,
        graph_dir=Path(args.dir),
        static_dir=Path("."),
        watch=not args.no_watch,
        auto_port=args.auto_port
    )

    # CLI-specific output formatting
    print(f"Server started at {result.handle.url}")
    # ...
```

## Using Operations from SDK

The SDK wraps operations with a fluent API:

```python
# In wipnote/sdk.py

class SDK:
    def start_server(self, port: int = 8080, **kwargs) -> ServerHandle:
        """Start Wipnote server."""
        from wipnote.operations import server

        result = server.start_server(
            port=port,
            graph_dir=self.graph_dir,
            static_dir=Path("."),
            **kwargs
        )

        # SDK returns handle directly
        return result.handle
```

## Error Handling

Operations raise specific exceptions:

```python
from wipnote.operations.server import PortInUseError, ServerStartError
from wipnote.operations.hooks import HookInstallError, HookConfigError

try:
    result = server.start_server(port=8080, ...)
except PortInUseError:
    print("Port 8080 is already in use")
except ServerStartError as e:
    print(f"Server failed to start: {e}")
```

## Migration Guide

If you have custom scripts using the old API:

### Before (direct imports)
```python
from wipnote.server import serve

serve(port=8080)
```

### After (use SDK - recommended)
```python
from wipnote import SDK

sdk = SDK()
handle = sdk.start_server(port=8080)
```

### After (use operations - advanced)
```python
from wipnote.operations import server
from pathlib import Path

result = server.start_server(
    port=8080,
    graph_dir=Path(".wipnote"),
    static_dir=Path(".")
)

# Access server handle
handle = result.handle
```

## Testing

The operations layer makes testing easier:

```python
import pytest
from wipnote.operations import server
from pathlib import Path

def test_server_start(tmp_path):
    """Test server starts successfully."""
    result = server.start_server(
        port=8081,  # Use different port for tests
        graph_dir=tmp_path / ".wipnote",
        static_dir=tmp_path,
        watch=False  # Disable watcher in tests
    )

    assert result.handle.port == 8081
    assert result.handle.url == "http://localhost:8081"

    # Cleanup
    server.stop_server(result.handle)
```

## Design Principles

1. **Pure functions** - Operations are stateless, accept parameters
2. **Dataclass results** - Return structured results, not primitives
3. **Explicit errors** - Raise specific exception types
4. **No side effects** - Operations don't modify global state
5. **Testable** - All operations can be tested in isolation

## Adding New Operations

When adding new functionality:

1. Create operation in `operations/` module
2. Define result dataclass
3. Add to `operations/__init__.py`
4. Wrap in SDK method (optional)
5. Add CLI command (optional)
6. Write tests

Example:

```python
# operations/backup.py

from dataclasses import dataclass
from pathlib import Path

@dataclass(frozen=True)
class BackupResult:
    backup_path: Path
    files_backed_up: int
    warnings: list[str]

def create_backup(
    *,
    graph_dir: Path,
    backup_dir: Path
) -> BackupResult:
    """Create backup of Wipnote data."""
    # Implementation
    return BackupResult(...)
```

## See Also

- [AGENTS.md](../AGENTS.md) - Complete SDK documentation
- [CLI Reference](./cli-reference.md) - CLI command reference
- [API Reference](./api-reference.md) - Complete API documentation
