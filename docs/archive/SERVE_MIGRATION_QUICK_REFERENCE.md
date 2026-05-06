# Wipnote `serve` Migration - Quick Reference

## Current Status at a Glance

```
┌─────────────────────────────────────────────────────────────┐
│  MIGRATION STATUS: 70% Complete                             │
│  Currently Running: FastAPI/uvicorn                          │
│  Status: Functional but missing legacy features             │
└─────────────────────────────────────────────────────────────┘
```

## What's Working NOW ✅

```
wipnote serve --port 8080 --auto-port
    ↓
FastAPI (uvicorn) starts
    ↓
┌──────────────────────────────────────┐
│ ✅ Dashboard UI (HTMX-based)         │
│ ✅ WebSocket /ws/events              │
│ ✅ Real-time activity feed           │
│ ✅ Query caching (30s TTL)           │
│ ✅ Agent workload visualization      │
│ ✅ Orchestration chains              │
│ ✅ Session metrics                   │
│ ✅ Port auto-detection               │
└──────────────────────────────────────┘
```

## What's MISSING ❌

```
┌──────────────────────────────────────┐
│ ❌ REST API Endpoints                │
│    • /api/status                     │
│    • /api/query?selector=...         │
│    • /api/{collection}               │
│    • /api/analytics/*                │
│                                      │
│ ❌ File Watching (--watch)           │
│    • Graph reload on file changes    │
│                                      │
│ ❌ Graph Directory Argument          │
│    • --graph-dir ignored             │
│                                      │
│ ❌ Quiet Mode (--quiet)              │
└──────────────────────────────────────┘
```

## File Inventory

### Active (In Use)
```
src/python/wipnote/
├── api/
│   └── main.py                    (2,300 lines) ← FastAPI app
├── operations/
│   └── fastapi_server.py          (230 lines)  ← Server ops
├── cli.py                         (4,700 lines)← CLI commands
└── file_watcher.py                            ← File watching (ready to use)

pyproject.toml
├── fastapi>=0.104.0   ✅ installed
├── uvicorn>=0.24.0    ✅ installed
└── aiosqlite>=0.19.0  ✅ installed
```

### Inactive (Legacy, Can Remove)
```
src/python/wipnote/
├── server.py                      (1,600 lines) ← LEGACY
└── operations/server.py           (300 lines)   ← LEGACY

tests/
└── operations/test_server.py      (300+ lines)  ← LEGACY TESTS
```

## Quick Comparison Table

| Feature | Legacy | FastAPI | Gap? |
|---------|--------|---------|------|
| Server Type | SimpleHTTPRequestHandler | FastAPI/uvicorn | ✅ Migrated |
| REST API | Comprehensive | None | ❌ MISSING |
| WebSocket | No | Yes | ✅ Improved |
| File Watching | Yes | No | ❌ MISSING |
| Dashboard | Static HTML | Dynamic HTMX | ✅ Improved |
| Query Caching | No | Yes | ✅ New |
| Analytics Index | Build on-demand | Pre-built only | ⚠️ Regression |
| CORS Support | Yes | No | ❌ MISSING |
| CLI Arguments | --graph-dir --watch | --db --reload | ⚠️ Changed |

## Implementation Effort Estimate

### Phase 1: Restore REST API (CRITICAL)
```
Tasks:
  ☐ Add /api/status endpoint
  ☐ Add /api/query endpoint
  ☐ Add /api/{collection} endpoints
  ☐ Add /api/analytics/* endpoints
  ☐ Add CORS support
  ☐ Write comprehensive tests

Files:
  • src/python/wipnote/api/main.py (add ~500 lines)
  • tests/api/test_rest_api.py (new, ~400 lines)

Effort: 2-3 days
Risk: Medium (REST compatibility testing)
```

### Phase 2: Restore File Watching
```
Tasks:
  ☐ Integrate GraphWatcher into FastAPI
  ☐ Add --watch flag to CLI
  ☐ Implement graph reload on file changes

Files:
  • src/python/wipnote/operations/fastapi_server.py (add ~100 lines)
  • src/python/wipnote/cli.py (update arg parser)

Effort: 1-2 days
Risk: Low (GraphWatcher already implemented)
```

### Phase 3: CLI Argument Compatibility
```
Tasks:
  ☐ Make --graph-dir work with FastAPI
  ☐ Add --quiet flag
  ☐ Update argument validation

Files:
  • src/python/wipnote/cli.py (update)
  • src/python/wipnote/operations/fastapi_server.py (update)

Effort: 1 day
Risk: Low (straightforward changes)
```

### Phase 4: Remove Legacy Server (BREAKING)
```
Tasks:
  ☐ Delete src/python/wipnote/server.py (1,600 lines)
  ☐ Delete src/python/wipnote/operations/server.py (300 lines)
  ☐ Delete tests/operations/test_server.py
  ☐ Update any imports/references
  ☐ Update documentation
  ☐ Release as major version bump

Files to DELETE:
  × src/python/wipnote/server.py
  × src/python/wipnote/operations/server.py
  × tests/operations/test_server.py

Effort: 0.5 days (cleanup only)
Risk: Low (only deletions, FastAPI already active)
```

## Timeline

```
Week 1:  Phase 1 (REST API)              [████████░] 80%
Week 2:  Phase 2 (File Watching)         [██████░░░] 60%
         Phase 3 (CLI Arguments)         [███░░░░░░] 30%
Week 3:  Testing & Documentation        [██████░░░] 60%
Week 4:  Phase 4 (Remove Legacy)        [░░░░░░░░░]  0% (next major version)
```

## Breaking Changes to Document

### For Users Upgrading
```
OLD COMMAND:
  $ wipnote serve --port 8080 --graph-dir .wipnote --watch

NEW EQUIVALENT:
  $ wipnote serve --port 8080 --db .wipnote/index.sqlite --watch
```

### REST API Changes
| Endpoint | Status | Note |
|----------|--------|------|
| `/api/status` | REMOVED | ❌ Use /views/agents instead |
| `/api/{collection}` | REMOVED | ❌ Query features via /views/features |
| `/api/query?selector=...` | REMOVED | ❌ Use /views/* endpoints |
| `/api/analytics/*` | REMOVED | ⚠️ May restore in Phase 1 |
| `/ws/events` | ADDED | ✅ New real-time WebSocket |

### CLI Arguments
| Argument | Status | Note |
|----------|--------|------|
| `--port` | UNCHANGED | ✅ Works same |
| `--host` | UNCHANGED | ✅ Works same |
| `--graph-dir` | FIXED in Phase 3 | ⚠️ Currently ignored |
| `--auto-port` | UNCHANGED | ✅ Works same |
| `--watch` | RESTORING | ⚠️ Missing now, restored in Phase 2 |
| `--quiet` | RESTORING | ⚠️ Missing now, restored in Phase 3 |
| `--db` | NEW | ✅ New (required for FastAPI) |
| `--reload` | CHANGED | ⚠️ Now for uvicorn reload only |

## Decision Matrix

### Should We Complete the Migration?

| Criterion | Yes | No |
|-----------|-----|-----|
| **Risk** | Medium | High (defer = debt) |
| **Benefit** | High (single codebase) | Low (status quo) |
| **Effort** | 4 days | 0 days |
| **User Impact** | Improved (real-time) | Negative (missing features) |
| **Maintenance** | Reduced | Increased (duplication) |
| **Code Quality** | Better | Worse (debt) |

**Recommendation:** ✅ **Complete the migration**

---

## Key Decision Points

### 1. REST API Priority
Should we expose REST API endpoints in FastAPI?
- **If YES**: Existing programmatic clients keep working
- **If NO**: Breaking change, users must migrate to new endpoints

**Decision:** **YES** - Maintain backward compatibility

### 2. Legacy Server Removal Timing
When should we delete the legacy server code?
- **Option A**: After Phase 1 (REST API) - fast cleanup
- **Option B**: After Phase 4 - major version bump (safer)

**Decision:** **Option B** - Safer for users, gives deprecation notice

### 3. File Watching Scope
Should file watching be the default or opt-in?
- **If Default**: `--no-watch` to disable (better UX)
- **If Opt-in**: `--watch` to enable (safer, less resource usage)

**Decision:** **Opt-in** (`--watch` flag) - safer for production

### 4. Analytics Index Handling
Should we rebuild analytics index on-demand like legacy server?
- **If YES**: Requires rebuilding from events every time
- **If NO**: Users must pre-build index

**Decision:** **YES** - Better UX, match legacy behavior

---

## Success Criteria

Migration is complete when:

- [ ] All legacy REST endpoints restored in FastAPI
- [ ] File watching works with `--watch` flag
- [ ] CLI arguments backward compatible
- [ ] REST API tests pass (90%+ coverage)
- [ ] FastAPI server tests comprehensive
- [ ] Documentation updated
- [ ] No regressions in existing features
- [ ] WebSocket real-time events working
- [ ] Query caching improving performance
- [ ] Users can run without database pre-built

---

## Files to Review

1. **READ FIRST:** `/Users/shakes/DevProjects/htmlgraph/SERVE_MIGRATION_ANALYSIS.md`
   - Full technical analysis
   - Architectural details
   - Implementation guide

2. **THEN:** These key files
   - `src/python/wipnote/api/main.py` - FastAPI app (2,300 lines)
   - `src/python/wipnote/server.py` - Legacy server (1,600 lines, reference)
   - `src/python/wipnote/cli.py` - CLI commands (around line 140)
   - `src/python/wipnote/operations/fastapi_server.py` - Server ops (230 lines)

---

## Quick Start for Implementation

### If Starting Phase 1 (REST API):
1. Read `SERVE_MIGRATION_ANALYSIS.md` section "Preserving Legacy REST API in FastAPI"
2. Review `src/python/wipnote/server.py` lines 236-896 (API handlers)
3. Port handler logic to FastAPI routes in `src/python/wipnote/api/main.py`
4. Add comprehensive tests in `tests/api/test_rest_api.py`
5. Verify with: `uv run pytest tests/api/test_rest_api.py -v`

### If Starting Phase 2 (File Watching):
1. Review `src/python/wipnote/file_watcher.py` (already implemented)
2. Update `src/python/wipnote/operations/fastapi_server.py`
3. Add `--watch` argument to `src/python/wipnote/cli.py`
4. Test with: `wipnote serve --watch --port 8080`

### If Starting Phase 3 (CLI Arguments):
1. Update argument parser in `src/python/wipnote/cli.py`
2. Make `--graph-dir` work with FastAPI database detection
3. Add `--quiet` flag support
4. Test all combinations: `wipnote serve --help`

### If Starting Phase 4 (Cleanup):
1. Delete: `src/python/wipnote/server.py`
2. Delete: `src/python/wipnote/operations/server.py`
3. Delete: `tests/operations/test_server.py`
4. Search for imports: `grep -r "from wipnote.server import\|from wipnote.operations.server import"`
5. Update any remaining references
6. Run full test suite: `uv run pytest`

---

## Resources

- **FastAPI Docs:** https://fastapi.tiangolo.com/
- **Uvicorn Docs:** https://www.uvicorn.org/
- **WebSocket Guide:** https://fastapi.tiangolo.com/advanced/websockets/
- **GraphWatcher Implementation:** `src/python/wipnote/file_watcher.py`
- **Analytics Index:** `src/python/wipnote/analytics_index.py`
