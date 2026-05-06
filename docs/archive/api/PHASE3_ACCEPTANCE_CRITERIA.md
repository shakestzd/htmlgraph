# Phase 3: Acceptance Criteria & Verification Checklist

## Overview

This document specifies measurable acceptance criteria for Phase 3 (Testing & Validation). Each criterion can be verified through automated tests, manual checks, or performance measurements.

---

## Acceptance Criterion #1: Code Organization & Architecture

### Description
The API refactor should significantly reduce the size and complexity of `main.py` by moving database queries to repositories, business logic to services, and models to dedicated schema modules.

### Measurable Targets

| Metric | Before | Target | Verification |
|--------|--------|--------|--------------|
| `main.py` line count | 2761 | <500 | `wc -l src/python/wipnote/api/main.py` |
| Database queries in `main.py` | 50+ | 0 | Grep for `SELECT`, `INSERT`, etc. |
| Business logic functions in `main.py` | 30+ | <5 | Grep for complex functions |
| Model classes in `main.py` | 3+ | 0 | Grep for `class.*Model` |
| Repository modules | 0 | 4+ | Check `repositories/` directory |
| Service modules | 0 | 4+ | Check `services/` directory |

### Verification Steps

```bash
# 1. Check main.py size
wc -l src/python/wipnote/api/main.py
# Expected: <500 lines

# 2. Verify no queries in main.py
grep -E "SELECT|INSERT|UPDATE|DELETE|execute\(" \
  src/python/wipnote/api/main.py | grep -v "app\|get_app\|# " | wc -l
# Expected: 0 occurrences

# 3. Check repositories exist and are used
ls -1 src/python/wipnote/api/repositories/*.py
# Expected: base_repository.py, events_repository.py, etc.

# 4. Check services exist and are used
ls -1 src/python/wipnote/api/services/*.py
# Expected: base_service.py, activity_service.py, etc.

# 5. Verify schemas module
test -f src/python/wipnote/api/schemas.py && echo "OK" || echo "FAIL"
# Expected: OK
```

### Test Coverage

**Unit Tests**: `tests/unit/api/test_architecture.py`
```python
def test_main_py_is_thin():
    """Verify main.py is <500 lines and contains only routing."""

def test_no_queries_in_main():
    """Verify all SQL queries moved to repositories."""

def test_repositories_encapsulate_queries():
    """Verify repositories contain all database queries."""

def test_services_encapsulate_logic():
    """Verify services contain all business logic."""
```

---

## Acceptance Criterion #2: API Backward Compatibility

### Description
All API endpoints must return identical response shapes, status codes, and behaviors as before the refactor. No breaking changes.

### Measurable Targets

| Endpoint | Shape Match | Status Codes | Response Time Δ |
|----------|-------------|--------------|-----------------|
| GET / | 100% | 200 | ±10% |
| GET /views/agents | 100% | 200 | ±10% |
| GET /api/events | 100% | 200 | ±10% |
| GET /api/sessions | 100% | 200 | ±10% |
| GET /api/features | 100% | 200 | ±10% |
| GET /api/orchestration | 100% | 200 | ±10% |
| POST /api/* (if any) | 100% | 201/200 | ±10% |
| WebSocket /ws/broadcast | 100% | 101 | ±10% |

### Verification Steps

```bash
# 1. Record baseline response shapes
python tests/api/record_baseline.py --save baseline.json

# 2. Compare current responses
python tests/api/compare_responses.py --baseline baseline.json

# 3. Check status codes match
curl -I http://localhost:8000/api/events
# Expected: 200 OK (or whatever was original)

# 4. Validate JSON schema
python tests/api/validate_schemas.py --endpoints all
```

### Test Coverage

**Integration Tests**: `tests/integration/api/test_backward_compatibility.py`
```python
async def test_activity_feed_response_shape():
    """Verify activity feed response unchanged."""

async def test_agent_stats_response_shape():
    """Verify agent stats response unchanged."""

async def test_all_endpoints_return_200():
    """Verify HTTP status codes unchanged."""

async def test_response_times_within_baseline():
    """Verify response times within ±10% of baseline."""
```

---

## Acceptance Criterion #3: Database Architecture

### Description
The database layer should use FastSQLA exclusively with proper connection pooling, timeout configuration, and transaction handling. No raw aiosqlite usage outside repositories.

### Measurable Targets

| Component | Target | Verification |
|-----------|--------|--------------|
| FastSQLA used for sessions | 100% | Grep for FastSQLA imports |
| aiosqlite only in repositories | 100% | No aiosqlite outside `repositories/` |
| Connection pool size | Configured | Check FastSQLA config |
| Busy timeout value | 5000ms | Check `PRAGMA busy_timeout` |
| Transaction isolation | SERIALIZABLE | Check isolation level |
| Connection leak rate | 0/24hrs | Monitor connection count |

### Verification Steps

```bash
# 1. Find all aiosqlite imports
grep -r "import aiosqlite" src/python/wipnote/api/
# Expected: Only in repositories/base_repository.py

# 2. Verify FastSQLA usage in main.py
grep -c "FastSQLA\|get_session" src/python/wipnote/api/main.py
# Expected: >0 (using FastSQLA sessions)

# 3. Check connection pool configuration
grep -A 10 "FastSQLA\|pool_size" src/python/wipnote/api/db.py
# Expected: pool_size configured

# 4. Verify busy timeout setting
grep "busy_timeout\|PRAGMA" src/python/wipnote/api/repositories/base_repository.py
# Expected: 5000ms timeout configured
```

### Test Coverage

**Unit Tests**: `tests/unit/api/test_database_architecture.py`
```python
async def test_fastsqla_used_for_sessions():
    """Verify FastSQLA is used for database sessions."""

async def test_no_raw_aiosqlite_in_main():
    """Verify aiosqlite not used directly in main.py."""

async def test_connection_pool_configured():
    """Verify connection pool properly configured."""

async def test_busy_timeout_set():
    """Verify PRAGMA busy_timeout = 5000."""

async def test_no_connection_leaks():
    """Verify connection pool doesn't leak connections."""
```

---

## Acceptance Criterion #4: Caching Strategy

### Description
HTTP caching should use fastapi-cache2 exclusively. Cache TTL should be configurable, with proper invalidation on data changes.

### Measurable Targets

| Component | Target | Verification |
|-----------|--------|--------------|
| fastapi-cache2 usage | 100% of cached endpoints | Grep for @cache decorator |
| QueryCache removal | Complete | No QueryCache() in main routes |
| Cache TTL configurable | Yes | Check CACHE_TTL constant |
| Cache hit rate | >80% | Monitor cache metrics |
| Invalidation latency | <10ms | Performance test |
| Memory overhead | <50MB | Monitor cache size |

### Verification Steps

```bash
# 1. Count @cache decorators on routes
grep -c "@cache" src/python/wipnote/api/main.py
# Expected: >10 (for cached endpoints)

# 2. Verify QueryCache removed from routes
grep "QueryCache(" src/python/wipnote/api/main.py | grep -v "# " | wc -l
# Expected: 0

# 3. Check CACHE_TTL configurable
grep "CACHE_TTL\|cache_ttl" src/python/wipnote/api/cache.py
# Expected: CACHE_TTL = ... (configurable constant)

# 4. Monitor cache hit rate
curl http://localhost:8000/api/events -H "X-Cache: true"
# Expected: X-Cache-Hit: true (on second request)
```

### Test Coverage

**Integration Tests**: `tests/integration/api/test_caching.py`
```python
async def test_cache_decorator_applied():
    """Verify @cache decorator on endpoints."""

async def test_cache_hit_rate_high():
    """Verify cache hit rate >80%."""

async def test_cache_invalidation():
    """Verify cache invalidates on data changes."""

async def test_cache_ttl_configurable():
    """Verify TTL can be configured."""

async def test_cache_memory_bounded():
    """Verify cache memory <50MB."""
```

---

## Acceptance Criterion #5: Query Performance & N+1 Prevention

### Description
All endpoints should use bulk queries to prevent N+1 query patterns. Performance should improve by at least 25% on complex queries like activity feed.

### Measurable Targets

| Query | N+1 Check | Latency Before | Latency After | Improvement |
|-------|-----------|-----------------|-----------------|-------------|
| Activity feed (100 events) | 1 query | 200ms | <150ms | >25% |
| Agent stats | 1 query | 150ms | <120ms | >20% |
| Orchestration chain | 1-2 queries | 250ms | <180ms | >28% |
| Analytics aggregation | 1 query | 180ms | <140ms | >22% |
| WebSocket cycle | 1 query | 50ms | <30ms | >40% |

### Verification Steps

```bash
# 1. Enable query logging
SQLALCHEMY_ECHO=true python -m wipnote serve

# 2. Count queries per endpoint
curl http://localhost:8000/api/events?limit=100
# Expected: 1 SELECT query (not N queries)

# 3. Measure latency
time curl http://localhost:8000/api/events?limit=100
# Expected: <150ms for 100 events

# 4. Check WebSocket query pattern
# Expected: 1 query per cycle, not 1 per event

# 5. Profile with py-spy
py-spy record -o profile.svg -- python -m wipnote serve
# Expected: Bulk query operations visible in flame graph
```

### Test Coverage

**Performance Tests**: `tests/benchmarks/test_query_performance.py`
```python
async def test_activity_feed_uses_bulk_query():
    """Verify activity feed doesn't use N+1 pattern."""

async def test_orchestration_bulk_loads_dependencies():
    """Verify orchestration bulk-loads all dependencies."""

async def test_analytics_bulk_aggregates():
    """Verify analytics uses single bulk aggregation."""

@pytest.mark.benchmark
async def test_activity_feed_latency():
    """Verify activity feed latency <150ms for 100 events."""

@pytest.mark.benchmark
async def test_agent_stats_latency():
    """Verify agent stats latency <120ms."""
```

---

## Acceptance Criterion #6: WebSocket Performance & Connection Management

### Description
WebSocket handlers should reuse database connections efficiently using connection pooling. No new connections per cycle. Memory usage should remain stable.

### Measurable Targets

| Metric | Target | Verification |
|--------|--------|--------------|
| Connections per cycle | 0 (reuse) | Monitor connection pool |
| Connection pool utilization | 60-80% | Check pool metrics |
| WebSocket latency | <30ms per message | Benchmark |
| Memory growth over 1hr | <10MB | Monitor RSS |
| Graceful reconnection | 100% success | Simulate network failure |
| Error recovery time | <5 seconds | Measure reconnection time |

### Verification Steps

```bash
# 1. Monitor connection pool
sqlite3 .wipnote/wipnote.db "PRAGMA database_list;"

# 2. Connect WebSocket and monitor
python tests/websockets/monitor_connections.py

# 3. Measure WebSocket latency
python tests/websockets/measure_latency.py
# Expected: <30ms per message

# 4. Monitor memory usage
python tests/websockets/monitor_memory.py --duration 3600
# Expected: <10MB growth over 1 hour

# 5. Test reconnection
python tests/websockets/test_reconnection.py
# Expected: 100% successful reconnections
```

### Test Coverage

**Integration Tests**: `tests/integration/api/test_websocket_performance.py`
```python
async def test_websocket_reuses_connections():
    """Verify WebSocket reuses DB connections."""

async def test_websocket_latency_under_30ms():
    """Verify WebSocket message latency <30ms."""

async def test_websocket_memory_stable():
    """Verify memory usage stable over 1 hour."""

async def test_websocket_graceful_reconnection():
    """Verify WebSocket gracefully reconnects."""

async def test_websocket_error_recovery():
    """Verify error recovery in <5 seconds."""
```

---

## Summary Verification Checklist

### Automated Verification (CI/CD)
- [ ] All unit tests passing (target: 2665+ tests)
- [ ] All integration tests passing
- [ ] Code coverage >85%
- [ ] Type checking 100% pass (`mypy src/`)
- [ ] Linting 100% pass (`ruff check`)
- [ ] Performance benchmarks met
- [ ] No memory leaks detected

### Manual Verification (QA)
- [ ] Dashboard loads correctly
- [ ] Activity feed displays all events
- [ ] Pagination works (next/prev)
- [ ] Search/filter functional
- [ ] WebSocket updates appear in real-time
- [ ] No console errors in browser
- [ ] Database queries optimized (no N+1)
- [ ] Cache hits visible in logs
- [ ] Response times match baseline ±10%
- [ ] Memory usage stable over 24 hours

### Performance Verification
- [ ] Activity feed: <150ms (was 200ms)
- [ ] Agent stats: <120ms (was 150ms)
- [ ] Orchestration: <180ms (was 250ms)
- [ ] WebSocket: <30ms (was 50ms)
- [ ] Cache hit rate: >80%

### Code Quality Verification
- [ ] main.py: <500 lines (was 2761)
- [ ] No SQL in main.py
- [ ] Repositories encapsulate all queries
- [ ] Services encapsulate all logic
- [ ] Models in dedicated schemas.py
- [ ] FastSQLA used throughout
- [ ] No raw aiosqlite outside repositories

---

## Sign-Off

**Phase 3 Complete When**:
1. ✅ All acceptance criteria verified
2. ✅ All automated tests passing
3. ✅ All manual tests completed
4. ✅ Performance improvements documented
5. ✅ Code quality gates passed
6. ✅ Ready for v0.29.0 deployment

**Sign-Off Authority**: Project Lead / QA Team

**Expected Completion**: 2026-02-10

---

**Last Updated**: 2026-02-03
**Document Version**: 1.0
**Owner**: Claude (Refactoring Team)
