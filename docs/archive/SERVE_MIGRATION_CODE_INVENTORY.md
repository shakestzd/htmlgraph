# Wipnote `serve` Migration - Code Inventory

## File-by-File Breakdown

### 1. Active FastAPI Implementation

#### `/src/python/wipnote/api/main.py` (2,300+ lines)
**Status:** ✅ Active, core server

**Structure:**
```python
Lines 1-35:       Imports & configuration
Lines 37-77:      QueryCache class (30-second TTL)
Lines 79-120:     Data models (EventModel, FeatureModel, etc.)
Lines 160-212:    App initialization & template setup
Lines 214-223:    Database connection helpers
Lines 225-236:    GET / (dashboard root)
Lines 238-330:    GET /views/agents (agent stats, HTMX)
Lines 332-430:    GET /views/activity-feed (activity, HTMX)
Lines ...         GET /views/features (feature tracker)
Lines ...         GET /views/orchestration (delegation chains)
Lines ...         GET /views/sessions (session metrics)
Lines 2076-2246:  WebSocket /ws/events (real-time streaming)
Lines 2253-2300:  create_app() factory function
```

**Key Functions:**
- `create_app(db_path)` - Factory creates FastAPI app
- `websocket_events()` - Real-time event streaming
- `agents_view()` - Agent workload visualization
- `activity_feed()` - Activity feed with grouping
- Query cache for performance

**Dependencies:**
- fastapi
- aiosqlite (async SQLite)
- jinja2 (templates)
- pydantic (models)

**WebSocket Features:**
- Subscribes to live_events table
- Broadcasts spawner agent events
- Groups events by parent_event_id
- Marks events as broadcast to prevent duplication

---

#### `/src/python/wipnote/operations/fastapi_server.py` (230 lines)
**Status:** ✅ Active, server operations

**Functions:**
```python
Lines 46-143:     start_fastapi_server()
  • Creates uvicorn Config
  • Handles port binding
  • Returns FastAPIServerStartResult

Lines 146-162:    run_fastapi_server() (async)
  • Runs the server with await

Lines 165-181:    stop_fastapi_server()
  • Sets server.should_exit = True

Lines 184-202:    _check_port_in_use()
  • Helper: socket-based port check

Lines 205-230:    _find_available_port()
  • Helper: finds available port in range
```

**Returns:**
```python
FastAPIServerStartResult(
    handle=FastAPIServerHandle(
        url="http://localhost:8000",
        port=8000,
        host="127.0.0.1",
        server=<uvicorn.Server>
    ),
    warnings=["Port 8080 is in use, using 8081 instead"],
    config_used={
        "port": 8000,
        "original_port": 8080,
        "host": "127.0.0.1",
        "db_path": "/Users/shakes/.wipnote/index.sqlite",
        "auto_port": True,
        "reload": False,
    }
)
```

---

### 2. Legacy Server Implementation (INACTIVE)

#### `/src/python/wipnote/server.py` (1,600+ lines)
**Status:** ❌ Inactive, can be deleted

**Structure:**
```python
Lines 1-31:       Module docstring, imports
Lines 33-163:     WipnoteAPIHandler class
  • Lines 33-58:    Class variables & initialization
  • Lines 60-162:   _get_graph() - lazy graph loading

Lines 164-225:    HTTP response helpers
  • _send_json()
  • _send_error_json()
  • _read_body()
  • _parse_path()
  • _serve_packaged_dashboard()

Lines 226-235:    do_OPTIONS() - CORS
Lines 236-300:    do_GET() - GET request routing
Lines 302-342:    do_POST() - POST request routing
Lines 344-362:    do_PUT() - PUT request routing
Lines 364-382:    do_PATCH() - PATCH request routing
Lines 384-396:    do_DELETE() - DELETE request routing

Lines 398-1268:   API Handlers (~870 lines)
  • _handle_status() - Graph status overview
  • _handle_list() - List nodes with filtering/sorting
  • _handle_get() - Get single node
  • _handle_create() - Create new node
  • _handle_update() - Update node (PUT/PATCH)
  • _handle_delete() - Delete node
  • _handle_query() - CSS selector queries
  • _handle_analytics() - Analytics endpoints
  • _handle_orchestration_view() - Delegation chains
  • _handle_task_delegations_stats() - Delegation stats
  • _handle_track_features() - Features for track
  • _handle_feature_context() - Feature context
  • _handle_session_transcript() - Session transcripts
  • _handle_generate_features() - Generate from plan
  • _handle_sync_track() - Sync track completion

Lines 1270-1271:  log_message() - Custom logging

Lines 1274-1317:  Port management utilities
  • find_available_port()
  • check_port_in_use()

Lines 1319-1362:  Dashboard sync functions
  • sync_dashboard_files()

Lines 1364-1604:  serve() function
  • Main entry point for legacy server
  • Handles startup, file watching, shutdown
```

**Key Features:**
- 9 collection types (features, bugs, spikes, chores, epics, sessions, agents, tracks, task-delegations)
- Analytics index building on-demand
- File watching with GraphWatcher
- Dashboard file syncing
- Track/feature integration
- Session activity tracking
- Rich output (progress bars, formatted output)

**REST Endpoints Provided:**
```
GET    /api/status                              - Graph status
GET    /api/collections                         - List collections
GET    /api/query?selector=...                  - CSS selector query
GET    /api/{collection}                        - List nodes
POST   /api/{collection}                        - Create node
GET    /api/{collection}/{id}                   - Get node
PUT    /api/{collection}/{id}                   - Replace node
PATCH  /api/{collection}/{id}                   - Update node
DELETE /api/{collection}/{id}                   - Delete node
GET    /api/analytics/overview                  - Analytics overview
GET    /api/analytics/features?limit=50         - Top features
GET    /api/analytics/session?id=...            - Session events
GET    /api/analytics/continuity?feature_id=... - Feature continuity
GET    /api/analytics/transitions               - Tool transitions
GET    /api/analytics/commits?feature_id=...    - Feature commits
GET    /api/analytics/commit-graph?feature_id=...- Commit graph
GET    /api/orchestration                       - Delegation chains
GET    /api/task-delegations?stats=true         - Delegation stats
GET    /api/tracks/{id}?features=true           - Track features
GET    /api/features/{id}?context=true          - Feature context
GET    /api/sessions/{id}?transcript=true       - Session transcript
POST   /api/tracks/{id}?generate-features=true  - Generate features
POST   /api/tracks/{id}?sync=true               - Sync track
```

---

#### `/src/python/wipnote/operations/server.py` (300 lines)
**Status:** ❌ Inactive, can be deleted

**Functions:**
```python
Lines 41-187:     start_server()
  • Validates configuration
  • Creates directories
  • Builds analytics index if needed
  • Starts HTTPServer & GraphWatcher
  • Returns ServerStartResult

Lines 190-223:    stop_server()
  • Stops HTTPServer & GraphWatcher

Lines 226-250:    get_server_status()
  • Returns ServerStatus indicating if running

Lines 256-274:    _check_port_in_use()
  • Helper: socket-based port check

Lines 277-302:    _find_available_port()
  • Helper: finds available port in range
```

---

### 3. CLI Integration

#### `/src/python/wipnote/cli.py` (4,700+ lines)
**Status:** ✅ Active, but needs updates

**Current serve command:**
```python
Lines 140-194:    cmd_serve(args)
  • Entry point when user runs: wipnote serve
  • Calls: start_fastapi_server()
  • Calls: run_fastapi_server() (async)
  • Prints server info with Rich console
  • Handles KeyboardInterrupt

Lines 197-239:    cmd_serve_api(args)
  • Alternate name for same command

Lines 5120-5135:  Argument parser setup
  • Defines: --port, --host, --graph-dir, --auto-port, --db, --reload
```

**Current Argument Definitions:**
```python
parser_serve = subparsers.add_parser('serve', ...)
parser_serve.add_argument('--port', type=int, default=8080)
parser_serve.add_argument('--host', default='localhost')
parser_serve.add_argument('--graph-dir', default='.wipnote')
parser_serve.add_argument('--auto-port', action='store_true')
parser_serve.add_argument('--db', default=None)
parser_serve.add_argument('--reload', action='store_true')
parser_serve.add_argument('--verbose', action='store_true')
```

**Missing Arguments:**
- `--watch` - File watching (legacy: enabled by default, ignored in FastAPI)
- `--quiet` - Quiet mode (legacy: suppresses progress output)

---

### 4. Supporting Components

#### `/src/python/wipnote/file_watcher.py` (active)
**Status:** ✅ Implemented, ready to use

**GraphWatcher class:**
- Monitors graph directories for changes
- Calls reload callbacks on file changes
- Background thread-based
- Methods: `start()`, `stop()`

**Can be integrated into FastAPI server ops**

---

#### `/src/python/wipnote/analytics_index.py` (active)
**Status:** ✅ Implemented, used by both

**AnalyticsIndex class:**
- SQLite-based analytics database
- Methods: `rebuild_from_events()`, `overview()`, `top_features()`, etc.
- Pre-built database OR built on-demand

**Legacy server:** Builds on-demand if missing
**FastAPI:** Requires pre-built (gap to fix)

---

### 5. Tests

#### `tests/operations/test_server.py` (300+ lines)
**Status:** ❌ Tests legacy HTTPServer

**Test Classes:**
```python
Lines 23-50:      TestHelperFunctions
  • test_check_port_in_use_available
  • test_check_port_in_use_occupied
  • test_find_available_port_success
  • test_find_available_port_no_ports_available

Lines 53-198:     TestStartServer
  • test_start_server_basic
  • test_start_server_auto_port
  • test_start_server_port_in_use_no_auto
  • test_start_server_dashboard_sync_warning
  • test_start_server_with_watcher

Lines 201-261:    TestStopServer
  • test_stop_server_with_dict_server
  • test_stop_server_with_direct_server
  • test_stop_server_none_server
  • test_stop_server_shutdown_failure

Lines 264-305:    TestGetServerStatus
  • test_get_server_status_no_handle
  • test_get_server_status_running
  • test_get_server_status_not_running
```

**Can be replaced with FastAPI-specific tests**

---

## Data Flow Comparison

### Legacy Server Flow
```
User runs: wipnote serve --port 8080
       ↓
cli.py: cmd_serve()
       ↓
operations/server.py: start_server()
       ├─ sync_dashboard_files()
       ├─ create graph directories
       ├─ build analytics index if needed
       ├─ HTTPServer((host, port), WipnoteAPIHandler)
       └─ GraphWatcher.start()
       ↓
server.py: serve_forever() in HTTPServer
       ├─ do_GET() → _handle_* methods
       ├─ do_POST() → _handle_create()
       ├─ do_PUT/PATCH() → _handle_update()
       └─ do_DELETE() → _handle_delete()
       ↓
HTTP requests from clients
```

### FastAPI Server Flow
```
User runs: wipnote serve --port 8080
       ↓
cli.py: cmd_serve()
       ↓
operations/fastapi_server.py: start_fastapi_server()
       ├─ resolve database path
       ├─ uvicorn.Config(app, host, port, ...)
       └─ uvicorn.Server(config)
       ↓
cli.py: asyncio.run(run_fastapi_server(handle))
       ↓
handle.server.serve() (async)
       ├─ FastAPI app startup
       ├─ Routes:
       │  ├─ GET / → dashboard template
       │  ├─ GET /views/* → HTMX partials
       │  └─ WebSocket /ws/events → real-time streaming
       └─ uvicorn event loop
       ↓
HTTP/WebSocket requests from clients
```

---

## Dependency Graph

### Current (FastAPI)
```
cli.py
  └─ operations/fastapi_server.py
      └─ api/main.py
          ├─ aiosqlite
          ├─ fastapi
          ├─ jinja2
          └─ pydantic

config/
  └─ pyproject.toml
      ├─ fastapi>=0.104.0 ✅
      ├─ uvicorn>=0.24.0 ✅
      └─ aiosqlite>=0.19.0 ✅
```

### Legacy (SimpleHTTPRequestHandler)
```
cli.py
  └─ operations/server.py
      ├─ server.py
      │  ├─ Wipnote
      │  ├─ GraphWatcher
      │  ├─ AnalyticsIndex
      │  └─ http.server (stdlib)
      └─ file_watcher.py
```

### Hybrid (What We Need)
```
cli.py
  └─ operations/fastapi_server.py
      ├─ api/main.py (FastAPI)
      │  ├─ REST endpoints (add)
      │  ├─ WebSocket /ws/events ✅
      │  ├─ aiosqlite
      │  └─ jinja2
      ├─ file_watcher.py (add)
      │  └─ GraphWatcher integration
      └─ analytics_index.py (add)
          └─ Build on-demand if missing
```

---

## Code Statistics

### Active Code
```
src/python/wipnote/api/main.py              2,300 lines
src/python/wipnote/operations/fastapi_server.py  230 lines
src/python/wipnote/cli.py (serve section)   ~50 lines
                                              ─────────
Total Active Code:                           ~2,580 lines
```

### Inactive Code (Can Delete)
```
src/python/wipnote/server.py                1,600 lines
src/python/wipnote/operations/server.py      300 lines
tests/operations/test_server.py                300 lines
                                              ─────────
Total Legacy Code:                           ~2,200 lines
```

### Code to Add (Restoration)
```
REST API endpoints in api/main.py             ~500 lines
File watching integration                     ~100 lines
REST API tests                                ~400 lines
                                              ─────────
Total New Code:                              ~1,000 lines
```

---

## Import Analysis

### Current Imports in cli.py (serve command)
```python
from wipnote.operations.fastapi_server import (
    run_fastapi_server,
    start_fastapi_server,
)
```

### Current Imports in operations/fastapi_server.py
```python
import uvicorn
from wipnote.api.main import create_app
```

### Current Imports in api/main.py
```python
import aiosqlite
from fastapi import FastAPI, Request, WebSocket, WebSocketDisconnect
from fastapi.responses import HTMLResponse
from fastapi.staticfiles import StaticFiles
from fastapi.templating import Jinja2Templates
from pydantic import BaseModel
```

### UNUSED Imports (from legacy server)
```python
# These are never imported when running FastAPI
from wipnote.server import WipnoteAPIHandler, sync_dashboard_files
from wipnote.operations.server import start_server, stop_server, get_server_status
```

---

## Database Schema (FastAPI Only)

The FastAPI server reads from SQLite at `.wipnote/index.sqlite`:

### Tables Used:
- `agent_events` - Stores all agent activity events
  - Columns: agent_id, event_count, session_id, timestamp, cost_tokens

- `live_events` - Pending events to broadcast
  - Columns: event_id, event_type, timestamp, broadcast_at

- Various analytics tables (read-only for dashboard)

### Tables NOT Used in FastAPI:
- HTML files in `.wipnote/{collection}/` directories
- Legacy: Read directly from HTML graph files
- New: Only reads SQLite database

**Gap:** FastAPI doesn't read HTML files, only SQLite database
- This is why REST API endpoints don't exist (no HTML graph access)

---

## Migration Checklist Template

### Phase 1: REST API Restoration
```
☐ Copy logic from server.py handlers
☐ Implement /api/status endpoint
☐ Implement /api/query endpoint
☐ Implement /api/{collection} endpoints (GET, POST)
☐ Implement /api/{collection}/{id} endpoints (GET, PUT, PATCH, DELETE)
☐ Implement /api/analytics/* endpoints
☐ Add CORS support
☐ Write REST endpoint tests
☐ Test with legacy API client scripts
☐ Verify backward compatibility
```

### Phase 2: File Watching
```
☐ Import GraphWatcher in fastapi_server.py
☐ Create background task for file watching
☐ Implement reload callback
☐ Add --watch flag to cli.py argument parser
☐ Update start_fastapi_server() to accept watch parameter
☐ Integrate GraphWatcher into server startup
☐ Write file watching tests
☐ Test graph reload on file change
☐ Verify no race conditions
```

### Phase 3: CLI Arguments
```
☐ Make --graph-dir work (resolve database from it)
☐ Add --watch flag
☐ Add --quiet flag
☐ Update argument validation
☐ Test all argument combinations
☐ Update help text
☐ Document migration for users
```

### Phase 4: Legacy Server Removal
```
☐ Verify all REST endpoints working in FastAPI
☐ Run full test suite with FastAPI only
☐ Delete src/python/wipnote/server.py
☐ Delete src/python/wipnote/operations/server.py
☐ Delete tests/operations/test_server.py
☐ Search for remaining imports of deleted modules
☐ Update import documentation
☐ Create migration guide for users
☐ Release as major version bump
☐ Update CHANGELOG
```

---

## Quick Navigation Guide

To understand the current state:
1. Start: `src/python/wipnote/cli.py` lines 140-194 (entry point)
2. Next: `src/python/wipnote/operations/fastapi_server.py` (server ops)
3. Then: `src/python/wipnote/api/main.py` (FastAPI app)
4. Reference: `src/python/wipnote/server.py` (legacy, for feature comparison)

To implement Phase 1 (REST API):
1. Review: `src/python/wipnote/server.py` lines 398-1268 (handlers)
2. Port to: `src/python/wipnote/api/main.py` (add routes)
3. Test: `tests/api/test_rest_api.py` (new file)

To implement Phase 2 (File Watching):
1. Review: `src/python/wipnote/file_watcher.py` (GraphWatcher)
2. Update: `src/python/wipnote/operations/fastapi_server.py`
3. Update: `src/python/wipnote/cli.py` argument parser

To implement Phase 3 (CLI Arguments):
1. Update: `src/python/wipnote/cli.py` argument definitions
2. Update: `src/python/wipnote/operations/fastapi_server.py` functions
3. Test: All command variations

To implement Phase 4 (Cleanup):
1. Delete: `src/python/wipnote/server.py`
2. Delete: `src/python/wipnote/operations/server.py`
3. Delete: `tests/operations/test_server.py`
4. Verify: `uv run pytest` passes
