# FastAPI API Refactor Plan (FastSQLA + Caching)

Date: 2026-02-03
Owner: Codex (with user approval)
Scope: `src/python/wipnote/api/main.py` and related API modules

**Goals**
1. Split `src/python/wipnote/api/main.py` into focused modules with clear boundaries.
2. Adopt FastSQLA to standardize async DB sessions and reduce boilerplate.
3. Replace the homegrown cache with a maintained caching dependency.
4. Improve behavior and performance in high-traffic endpoints and WebSocket loops.
5. Keep public API behavior stable unless explicitly called out.

**Non-Goals**
1. Changing HTML templates or UI behavior (unless required by refactor).
2. Altering DB schema.
3. Migrating to a different framework (staying on FastAPI).

**Key Decisions**
1. DB layer: FastSQLA + SQLAlchemy 2.0 async sessions.
2. Caching: `fastapi-cache2` as the primary response-level cache with in-memory backend by default, and a Redis backend option.
3. JSON speed: evaluate `orjson` for response serialization where appropriate.

**Proposed Module Layout**
1. `src/python/wipnote/api/app.py` for app factory, lifespan events, router registration, and dependency wiring.
2. `src/python/wipnote/api/db.py` for DB initialization, session factory, and `get_db()` dependency.
3. `src/python/wipnote/api/cache.py` for cache initialization and shared key building.
4. `src/python/wipnote/api/templates.py` for Jinja environment and filters.
5. `src/python/wipnote/api/schemas.py` for Pydantic response models.
6. `src/python/wipnote/api/repositories/` for SQL queries and DB access helpers.
7. `src/python/wipnote/api/services/` for aggregation logic and grouping rules.
8. `src/python/wipnote/api/routers/` for per-area endpoints.

**Behavior Improvements (Targeted)**
1. Replace N+1 child event queries with a bulk prefetch and in-memory grouping for the activity feed.
2. Make WebSocket loops use pooled DB sessions instead of opening and closing a connection every cycle.
3. Standardize cache TTLs per endpoint and add a manual invalidation hook for live updates when needed.
4. Consolidate repeated JSON parsing and fallback logic into utilities.
5. Add consistent pagination patterns for list endpoints.

**Milestones**
1. Design and skeleton: create module layout, move existing code with no behavior changes.
2. FastSQLA integration: DB session dependency, repositories, and route updates.
3. Caching integration: replace QueryCache with `fastapi-cache2` and retain metrics with a small wrapper.
4. Behavior improvements: bulk fetch, WebSocket session reuse, pagination, JSON handling.
5. Tests and validation: add repo and service tests and run existing API checks.

**Step-by-Step Plan**
1. Add dependencies and configure them in `pyproject.toml` and `uv.lock`.
2. Introduce `app.py`, `db.py`, `cache.py`, and `templates.py` with minimal logic moved over from `main.py`.
3. Move Pydantic models into `schemas.py` and update imports.
4. Create routers for each functional area and register them in the app factory.
5. Extract SQL into repositories and update routes to call repositories.
6. Create service helpers for complex aggregations like grouped events and orchestration summaries.
7. Integrate FastSQLA session dependency and replace `aiosqlite` calls with async SQLAlchemy Core usage.
8. Replace the custom cache with `fastapi-cache2` and keep a lightweight metrics wrapper for visibility.
9. Implement the behavior improvements and verify output parity for existing endpoints.
10. Add tests for repository functions and service logic, and run the test suite.

**Risk Mitigation**
1. Keep routing paths and response shapes stable unless explicitly approved.
2. Use golden test fixtures for critical JSON outputs like activity feed and event traces.
3. Add feature flags for cache backends so local dev uses in-memory by default.

**Acceptance Criteria**
1. `src/python/wipnote/api/main.py` is reduced to app creation and exports only.
2. All endpoints respond with the same shapes as before (verified via tests or fixtures).
3. FastSQLA is the only DB session lifecycle used in API routes.
4. Cached endpoints use `fastapi-cache2` and no longer use `QueryCache`.
5. WebSocket handlers do not open a new DB connection on every poll cycle.
6. Activity feed grouping uses a single bulk query instead of per-parent queries.

**Open Questions**
1. Should cache metrics be stored in-memory only, or persisted to the DB for analysis?
2. Do we want Redis as a default cache backend in production?
3. Should we formalize pagination response schemas for all list endpoints?
