"""API routes for local-first oplog sync and legacy git sync."""

from __future__ import annotations

import logging
from pathlib import Path
from typing import Any

from fastapi import APIRouter, HTTPException, Query

from htmlgraph.api.oplog_sync import OplogSyncService
from htmlgraph.api.sync_models import (
    OplogAckRequest,
    OplogAckResponse,
    OplogPullResponse,
    OplogPushRequest,
    OplogPushResponse,
    SyncStatusResponse,
)
from htmlgraph.sync import GitSyncManager, SyncStrategy

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/api/sync", tags=["sync"])

oplog_service: OplogSyncService | None = None
git_sync_manager: GitSyncManager | None = None


def init_sync_routes(
    *,
    db_path: str,
    manager: GitSyncManager | None = None,
) -> None:
    """Initialize sync route globals."""
    global oplog_service, git_sync_manager
    oplog_service = OplogSyncService(db_path)
    git_sync_manager = manager


def init_sync_manager(manager: GitSyncManager) -> None:
    """Backward-compatible initializer for legacy git sync manager."""
    global git_sync_manager
    git_sync_manager = manager


def _get_oplog_service() -> OplogSyncService:
    if oplog_service is None:
        raise HTTPException(
            status_code=503,
            detail="Oplog sync service not initialized",
        )
    return oplog_service


def _get_git_manager() -> GitSyncManager:
    if git_sync_manager is None:
        raise HTTPException(
            status_code=503,
            detail="Git sync manager not initialized",
        )
    return git_sync_manager


@router.post("/push", response_model=OplogPushResponse)
async def push_entries(request: OplogPushRequest) -> OplogPushResponse:
    """Push oplog entries with idempotent dedupe and conflict recording."""
    service = _get_oplog_service()
    return await service.push_entries(request.entries, consumer_id=request.consumer_id)


@router.get("/pull", response_model=OplogPullResponse)
async def pull_entries(
    since_seq: int = Query(default=0, ge=0),
    limit: int = Query(default=200, ge=1, le=1000),
    consumer_id: str | None = Query(default=None),
) -> OplogPullResponse:
    """Pull oplog entries ordered by monotonic sequence."""
    service = _get_oplog_service()
    return await service.pull_entries(
        since_seq=since_seq,
        limit=limit,
        consumer_id=consumer_id,
    )


@router.post("/ack", response_model=OplogAckResponse)
async def ack_cursor(request: OplogAckRequest) -> OplogAckResponse:
    """Advance consumer cursor state."""
    service = _get_oplog_service()
    return await service.ack_cursor(request)


@router.get("/status", response_model=SyncStatusResponse)
async def get_sync_status(consumer_id: str | None = None) -> SyncStatusResponse:
    """Get sync health, lag, conflict count, and cursor state."""
    service = _get_oplog_service()
    return await service.get_status(consumer_id=consumer_id)


# ---------------------------------------------------------------------------
# Legacy Git Sync Endpoints (kept for compatibility under /api/sync/git/*)
# ---------------------------------------------------------------------------


@router.post("/git/push")
async def trigger_git_push(force: bool = False) -> dict[str, Any]:
    manager = _get_git_manager()
    result = await manager.push(force=force)
    return result.to_dict()


@router.post("/git/pull")
async def trigger_git_pull(force: bool = False) -> dict[str, Any]:
    manager = _get_git_manager()
    result = await manager.pull(force=force)
    return result.to_dict()


@router.get("/git/status")
async def get_git_sync_status() -> dict[str, Any]:
    manager = _get_git_manager()
    return manager.get_status()


@router.get("/git/history")
async def get_git_sync_history(limit: int = 50) -> dict[str, Any]:
    manager = _get_git_manager()
    return {"history": manager.get_sync_history(limit)}


@router.post("/git/config")
async def update_git_sync_config(
    push_interval: int | None = None,
    pull_interval: int | None = None,
    conflict_strategy: str | None = None,
) -> dict[str, Any]:
    manager = _get_git_manager()

    try:
        if push_interval is not None:
            if push_interval < 10:
                raise ValueError("Push interval must be >= 10 seconds")
            manager.config.push_interval_seconds = push_interval

        if pull_interval is not None:
            if pull_interval < 10:
                raise ValueError("Pull interval must be >= 10 seconds")
            manager.config.pull_interval_seconds = pull_interval

        if conflict_strategy is not None:
            manager.config.conflict_strategy = SyncStrategy(conflict_strategy)

        return {"success": True, "config": manager.get_status()["config"]}
    except ValueError as e:
        raise HTTPException(status_code=400, detail=str(e))


@router.post("/git/start")
async def start_git_background_sync() -> dict[str, Any]:
    manager = _get_git_manager()
    import asyncio

    asyncio.create_task(manager.start_background_sync())
    return {"success": True, "message": "Background git sync started"}


@router.post("/git/stop")
async def stop_git_background_sync() -> dict[str, Any]:
    manager = _get_git_manager()
    await manager.stop_background_sync()
    return {"success": True, "message": "Background git sync stopped"}


@router.post("/git/init")
async def init_git_sync_manager(
    repo_root: str,
    remote_name: str = "origin",
    branch_name: str = "main",
) -> dict[str, Any]:
    """Optional runtime init endpoint for legacy git sync manager."""
    global git_sync_manager
    repo_path = Path(repo_root)
    if not repo_path.exists():
        raise HTTPException(status_code=404, detail="Repository path not found")
    git_sync_manager = GitSyncManager(
        str(repo_path),
        config=git_sync_manager.config if git_sync_manager else None,  # type: ignore[union-attr]
    )
    git_sync_manager.config.remote_name = remote_name
    git_sync_manager.config.branch_name = branch_name
    logger.info("Initialized git sync manager for %s", repo_root)
    return {"success": True}
