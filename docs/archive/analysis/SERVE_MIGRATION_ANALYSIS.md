# Wipnote `serve` Command Migration Analysis

## Executive Summary

The `wipnote serve` command has **already been migrated** to use FastAPI/uvicorn as of the current codebase. However, there are architectural inconsistencies and a legacy SimpleHTTPRequestHandler-based server still exists alongside the new FastAPI implementation.

**Current Status:**
- ✅ FastAPI app with WebSocket support exists (`src/python/wipnote/api/main.py`)
- ✅ CLI command uses FastAPI/uvicorn (`cmd_serve` in `cli.py`)
- ⚠️ Legacy SimpleHTTPRequestHandler server still exists (`src/python/wipnote/server.py`)
- ⚠️ Two separate server implementations create confusion and duplication

**Recommendation:** Complete the migration by consolidating both server implementations and removing the legacy SimpleHTTPRequestHandler entirely.

---

## Current Architecture

### 1. Legacy Server Implementation (SimpleHTTPRequestHandler)

**Location:** `/Users/shakes/DevProjects/htmlgraph/src/python/wipnote/server.py`

**Key Characteristics:**
- Uses Python standard library `http.server.HTTPServer` and `SimpleHTTPRequestHandler`
- No WebSocket support
- REST API implemented via HTTP verb handlers (`do_GET`, `do_POST`, `do_PUT`, `do_PATCH`, `do_DELETE`)
- Implements custom graph operations and analytics queries
- File watching via `GraphWatcher` class
- Dashboard file auto-sync functionality

**REST Endpoints:**
- `/api/status` - Overall graph status
- `/api/collections` - List available collections
- `/api/query?selector=...` - CSS selector queries
- `/api/{collection}` - CRUD operations (list, create)
- `/api/{collection}/{id}` - CRUD operations (get, update, delete)
- `/api/analytics/{endpoint}` - Analytics endpoints (overview, features, continuity, etc.)
- `/api/orchestration` - Delegation chains
- `/api/task-delegations/stats` - Delegation statistics
- `/api/tracks/{id}/features` - Feature tracking
- `/api/sessions/{id}?transcript=true` - Session transcripts

**Features:**
- 1,600+ lines of implementation
- Comprehensive REST API
- File watching for auto-reload
- Analytics index building on-demand
- Dashboard syncing
- Static file serving

### 2. FastAPI Implementation

**Location:** `/Users/shakes/DevProjects/htmlgraph/src/python/wipnote/api/main.py`

**Key Characteristics:**
- Modern async-first architecture using FastAPI
- **WebSocket support** for real-time event streaming (`/ws/events`)
- Jinja2 template rendering for HTMX-based UI
- Query caching with TTL (30-second default)
- SQLite-backed database (aiosqlite for async access)
- Real-time activity feed
- Agent workload visualization
- Orchestration chains visualization

**Routes:**
- `GET /` - Main dashboard (Jinja2 template)
- `GET /views/agents` - Agent workload stats (HTMX partial)
- `GET /views/activity-feed` - Activity feed (HTMX partial)
- `GET /views/features` - Feature tracker (Kanban view)
- `GET /views/orchestration` - Delegation chains
- `GET /views/sessions` - Session metrics
- `WebSocket /ws/events` - Real-time event streaming

**Features:**
- ~2,300+ lines of implementation
- Real-time event broadcasting via WebSocket
- HTMX-powered interactive UI
- Query performance metrics and caching
- Busy-timeout handling for concurrent access
- Template-based rendering

### 3. Server Operations Module

**Location:** `/Users/shakes/DevProjects/htmlgraph/src/python/wipnote/operations/server.py`

**Purpose:** High-level server start/stop operations

**Functions:**
- `start_server()` - Start HTTPServer (legacy)
- `stop_server()` - Stop HTTPServer
- `get_server_status()` - Check server status
- Helper functions: `_check_port_in_use()`, `_find_available_port()`

### 4. FastAPI Server Operations Module

**Location:** `/Users/shakes/DevProjects/htmlgraph/src/python/wipnote/operations/fastapi_server.py`

**Purpose:** FastAPI-specific server operations

**Functions:**
- `start_fastapi_server()` - Start FastAPI with uvicorn
- `run_fastapi_server()` - Run FastAPI server (async)
- `stop_fastapi_server()` - Stop FastAPI server
- Helper functions: `_check_port_in_use()`, `_find_available_port()`

---

## CLI Command Implementation

**Location:** `src/python/wipnote/cli.py` (lines 140-194)

**Current Implementation:**

```python
def cmd_serve(args: argparse.Namespace) -> None:
    """Start the Wipnote server (FastAPI-based)."""
    import asyncio

    from wipnote.operations.fastapi_server import (
        run_fastapi_server,
        start_fastapi_server,
    )

    try:
        # Default to database in graph dir if not specified
        db_path = getattr(args, "db", None)
        if not db_path:
            db_path = str(Path(args.graph_dir) / "index.sqlite")

        result = start_fastapi_server(
            port=args.port,
            host=args.host,
            db_path=db_path,
            auto_port=args.auto_port,
            reload=getattr(args, "reload", False),
        )

        # Print server info...
        asyncio.run(run_fastapi_server(result.handle))
```

**CLI Arguments:**
- `--port` - Port to listen on (default: 8080)
- `--host` - Host to bind to (default: localhost)
- `--graph-dir` - Graph data directory (default: .wipnote)
- `--auto-port` - Automatically find available port if in use
- `--reload` - Enable auto-reload on file changes
- `--db` - Database path (default: {graph_dir}/index.sqlite)
- `--verbose` - Verbose error output

---

## Dependencies

**Current Dependencies in pyproject.toml:**

```
fastapi>=0.104.0
uvicorn>=0.24.0
aiosqlite>=0.19.0
jinja2>=3.1.0
pydantic>=2.0.0
watchdog>=3.0.0
rich>=13.0.0
```

All FastAPI dependencies are already included in the base package.

---

## Architectural Issues & Inconsistencies

### Issue 1: Dual Server Implementations
- **Problem:** Two separate server implementations co-exist with different architectures
- **Impact:** Code duplication, maintenance burden, confusion about which to use
- **Current:** CLI uses FastAPI; legacy server still exists but unused
- **Files Involved:**
  - `/src/python/wipnote/server.py` (1,600+ lines, unused)
  - `/src/python/wipnote/operations/server.py` (300 lines, unused)
  - `/src/python/wipnote/api/main.py` (2,300+ lines, active)
  - `/src/python/wipnote/operations/fastapi_server.py` (230 lines, active)

### Issue 2: Missing REST API Endpoints in FastAPI
The legacy REST API endpoints are **not** exposed in the FastAPI implementation:
- `/api/status` - ❌ Missing
- `/api/query?selector=...` - ❌ Missing
- `/api/analytics/{endpoint}` - ❌ Missing
- `/api/{collection}` - ❌ Missing (only partial dashboard views exist)

**Impact:** Users running `wipnote serve` cannot use the REST API that was previously available.

### Issue 3: Database Dependency
- Legacy server: Works directly with HTML files (graph database)
- FastAPI server: Requires SQLite database (index.sqlite)
- **Gap:** FastAPI doesn't rebuild analytics index on-the-fly like legacy server

### Issue 4: File Watching/Auto-Reload
- Legacy server: Includes file watching via `GraphWatcher`
- FastAPI server: No file watching implementation
- `--reload` flag exists but only controls uvicorn reload, not graph reloading

### Issue 5: WebSocket Broadcasting
- FastAPI has WebSocket implementation (`/ws/events`)
- Broadcasts live events from `live_events` table
- But REST API clients cannot subscribe to real-time updates

---

## Implementation Status of Key Features

| Feature | Legacy Server | FastAPI | Status |
|---------|---|---|---|
| REST API | ✅ Full | ❌ Partial | Incomplete migration |
| WebSocket | ❌ No | ✅ Yes | Only in FastAPI |
| Dashboard | ⚠️ Static HTML | ✅ Dynamic HTMX | Improved in FastAPI |
| File Watching | ✅ Yes | ❌ No | Lost in migration |
| Analytics Index | ✅ Build on-demand | ❌ Requires pre-built | Regression |
| Query Caching | ❌ No | ✅ Yes | New in FastAPI |
| Static File Serving | ✅ Yes | ⚠️ Limited | Partial |
| CORS Support | ✅ Yes | ❌ No | Regression |

---

## User Impact: Current vs Expected Behavior

### Current Behavior (FastAPI)
```bash
$ wipnote serve --port 8080
```
- Starts FastAPI server on port 8080
- Serves dashboard UI with real-time activity feed
- WebSocket support available at `/ws/events`
- **REST API NOT available** (users can't programmatically query graph)
- **File watching NOT available** (changes require manual refresh)

### Expected Behavior (Post-Migration)
```bash
$ wipnote serve --port 8080
```
- Starts FastAPI server with uvicorn
- Serves dashboard UI with real-time activity feed
- WebSocket support for live events
- REST API available for programmatic access
- File watching enabled for auto-reload
- Analytics index built on-demand if needed

---

## Backward Compatibility Concerns

### CLI Arguments
| Argument | Legacy | FastAPI | Status |
|----------|--------|---------|--------|
| `--port` | ✅ | ✅ | Compatible |
| `--host` | ✅ | ✅ | Compatible |
| `--graph-dir` | ✅ | ❌ `--graph-dir` not used | **Breaking** |
| `--auto-port` | ✅ | ✅ | Compatible |
| `--db` | ❌ | ✅ | New (needed) |
| `--watch` | ✅ | ❌ | **Missing** |
| `--quiet` | ✅ | ❌ | **Missing** |

### Breaking Changes
1. `--graph-dir` is ignored in FastAPI (uses environment detection)
2. `--watch` flag is not available
3. `--quiet` flag is not available
4. REST API endpoints removed/unavailable

---

## Recommended Migration Path

### Phase 1: Complete REST API Parity (Current Gap)
1. Expose REST API endpoints in FastAPI app
2. Maintain backward compatibility with URL structure
3. Add CORS support for browser clients
4. Document REST API for FastAPI

**Files to modify:**
- `/src/python/wipnote/api/main.py` - Add REST endpoints
- `/src/python/wipnote/operations/fastapi_server.py` - Update if needed

### Phase 2: Restore File Watching
1. Add file watching integration to FastAPI server
2. Implement graph reload callback
3. Restore `--watch` flag functionality

**Files to modify:**
- `/src/python/wipnote/operations/fastapi_server.py`
- `/src/python/wipnote/api/main.py`

### Phase 3: Restore CLI Argument Compatibility
1. Add `--watch` and `--quiet` flags
2. Make `--graph-dir` work with FastAPI
3. Update argument parser

**Files to modify:**
- `/src/python/wipnote/cli.py`

### Phase 4: Remove Legacy Server (Breaking Change)
1. Delete `/src/python/wipnote/server.py` (1,600+ lines)
2. Delete `/src/python/wipnote/operations/server.py` (300 lines)
3. Update tests to use FastAPI server only
4. Document breaking change in release notes

**Files to delete:**
- `/src/python/wipnote/server.py`
- `/src/python/wipnote/operations/server.py`
- `tests/operations/test_server.py` (use FastAPI tests instead)

---

## Implementation Details

### Preserving Legacy REST API in FastAPI

The legacy REST endpoints can be preserved by creating a compatibility layer:

```python
# In src/python/wipnote/api/main.py

@app.get("/api/status")
async def api_status(db: aiosqlite.Connection = Depends(get_db)):
    """Legacy REST endpoint: Overall status."""
    # Query stats from agent_events table
    pass

@app.get("/api/query")
async def api_query(selector: str = "", db: aiosqlite.Connection = Depends(get_db)):
    """Legacy REST endpoint: CSS selector query."""
    # Query from graph files or database
    pass

@app.get("/api/{collection}")
async def api_list(collection: str, db: aiosqlite.Connection = Depends(get_db)):
    """Legacy REST endpoint: List nodes in collection."""
    pass

@app.get("/api/{collection}/{node_id}")
async def api_get(collection: str, node_id: str, db: aiosqlite.Connection = Depends(get_db)):
    """Legacy REST endpoint: Get single node."""
    pass
```

### Adding File Watching to FastAPI

```python
# In src/python/wipnote/operations/fastapi_server.py

async def run_fastapi_with_watcher(
    handle: FastAPIServerHandle,
    graph_dir: Path,
    enable_watch: bool = True
) -> None:
    """Run FastAPI server with optional file watching."""
    import asyncio

    watcher = None
    if enable_watch:
        # Start GraphWatcher in background task
        watcher = GraphWatcher(graph_dir)
        watcher.start()

    try:
        await handle.server.serve()
    finally:
        if watcher:
            watcher.stop()
```

---

## Testing Strategy

### Current Tests
- `tests/operations/test_server.py` - Tests for legacy HTTPServer (1,200+ lines)
- No tests for FastAPI server operations

### Required Tests
1. **FastAPI server startup** - Port binding, configuration
2. **REST API endpoints** - All legacy endpoints
3. **WebSocket connectivity** - Real-time event streaming
4. **File watching** - Graph reload on file changes
5. **CLI argument handling** - All flags work correctly
6. **Backward compatibility** - Existing scripts still work

---

## Files Summary

### Active (Currently Used)
- ✅ `/src/python/wipnote/api/main.py` (2,300 lines) - FastAPI app
- ✅ `/src/python/wipnote/operations/fastapi_server.py` (230 lines) - FastAPI ops
- ✅ `/src/python/wipnote/cli.py` (4,700+ lines) - CLI commands

### Inactive (Legacy, Can Be Removed)
- ❌ `/src/python/wipnote/server.py` (1,600 lines) - Legacy HTTPServer
- ❌ `/src/python/wipnote/operations/server.py` (300 lines) - Legacy ops
- ❌ `tests/operations/test_server.py` (300+ lines) - Legacy tests

### Related
- `pyproject.toml` - Dependencies (already has FastAPI/uvicorn)
- `src/python/wipnote/file_watcher.py` - GraphWatcher (still used)
- `src/python/wipnote/analytics_index.py` - Analytics (used by both)

---

## Decision Framework

### Option A: Complete FastAPI Migration (Recommended)
**Approach:** Restore all legacy features in FastAPI, then remove legacy server

**Pros:**
- ✅ Single server implementation (simpler maintenance)
- ✅ Modern async architecture
- ✅ WebSocket support for real-time updates
- ✅ Better performance with query caching
- ✅ Template-based UI improvements

**Cons:**
- ❌ Requires implementing REST API compatibility layer
- ❌ Requires restoring file watching
- ❌ Requires updating tests
- ❌ Breaking change for Phase 4 (removing legacy server)

**Effort:** ~3-4 days
**Risk:** Medium (REST API layer needs comprehensive testing)

### Option B: Keep Both Servers (Status Quo)
**Approach:** Maintain both implementations, document which to use

**Pros:**
- ✅ No breaking changes
- ✅ Can migrate users gradually
- ✅ Fallback if FastAPI has issues

**Cons:**
- ❌ Code duplication (1,600+ lines)
- ❌ Maintenance burden (bug fixes in 2 places)
- ❌ Confusion about which server to use
- ❌ Inconsistent features between servers
- ❌ Adds technical debt

**Effort:** ~0 (no work)
**Risk:** High (ongoing maintenance burden)

### Option C: Hybrid Approach
**Approach:** FastAPI with optional fallback to legacy server

**Pros:**
- ✅ Can deprecate legacy server gradually
- ✅ Safety fallback if FastAPI fails
- ✅ Time to migrate REST API clients

**Cons:**
- ❌ Still code duplication during transition
- ❌ More complex deployment logic
- ❌ Confusing for users

**Effort:** ~2-3 days
**Risk:** Medium (complexity)

---

## Recommended Action Items

### Immediate (Week 1)
1. ✅ Complete REST API in FastAPI (`/api/status`, `/api/query`, `/api/{collection}`, etc.)
2. ✅ Add CORS support for browser clients
3. ✅ Document REST API changes in migration guide
4. ✅ Add comprehensive REST API tests

### Short-term (Week 2)
5. ✅ Restore file watching functionality
6. ✅ Add `--watch` flag back to CLI
7. ✅ Update CLI argument parser for `--graph-dir` compatibility
8. ✅ Update tests to cover all scenarios

### Medium-term (Week 3)
9. ✅ Create FastAPI server tests (replacing legacy tests)
10. ✅ Deprecate legacy server in documentation
11. ✅ Add migration guide for users

### Long-term (Version X+1)
12. ✅ Remove legacy server files (breaking change)
13. ✅ Remove legacy server tests
14. ✅ Update documentation
15. ✅ Release as major version bump

---

## Conclusion

The migration from SimpleHTTPRequestHandler to FastAPI is **already in progress** but **incomplete**. The FastAPI implementation is active and functional, but it's missing:

1. REST API endpoints (previously available in legacy server)
2. File watching capability (previously available in legacy server)
3. CLI argument compatibility (`--watch`, `--quiet`, `--graph-dir`)

**Recommendation:** Complete the migration by restoring all legacy features in FastAPI, then remove the legacy server in a future major version.

**Current State:** 70% migrated (core server works, but features missing)
**Effort to Complete:** ~3-4 days of development + 2 days testing
**Risk Level:** Medium (requires careful REST API testing)
