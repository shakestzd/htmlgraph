# Local-First Sync Spike Report

## Scope
- Runtime: FastAPI + HTMX + SQLite (no Phoenix migration)
- Topology: single repo, multi-agent on one host
- Transport truth: append-only SQLite `oplog`

## Implemented
- Added canonical sync transport tables:
  - `oplog`
  - `sync_cursors`
  - `sync_conflicts`
- Added `/api/sync/*` oplog endpoints:
  - `POST /api/sync/push`
  - `GET /api/sync/pull`
  - `POST /api/sync/ack`
  - `GET /api/sync/status`
- Kept legacy git sync endpoints under `/api/sync/git/*`.
- Added deterministic conflict recording for overlapping updates (LWW policy).
- Added idempotent replay safety using unique `idempotency_key`.
- Added sync status SSE events (`sync-status`) and dashboard badges for:
  - sync lag
  - conflict count

## Reliability Checks
- Unit coverage:
  - Sync model validation (`tests/python/test_sync_models.py`)
  - Oplog service behavior (`tests/python/test_oplog_sync_service.py`)
- API coverage:
  - End-to-end push/pull/ack/status and conflict flow (`tests/python/test_sync_routes_api.py`)

## Go/No-Go Decision Inputs
- **Go** if:
  - No duplicated entries under retry with same `idempotency_key`
  - Pull order remains monotonic by `seq`
  - Cursor ack progression never regresses
  - Conflict records are consistently generated for overlapping writes
- **No-Go** if:
  - `oplog` ordering gaps or duplicate application are observed
  - Cursor lag or conflict counts become inconsistent under concurrent writes
  - Live dashboard sync indicators stop updating during normal load

## Next Recommendation
- Continue SQLite-first hardening (cursor semantics, retry strategy, observability)
- Defer CRDT adapter evaluation until production traces reveal concrete merge pain
