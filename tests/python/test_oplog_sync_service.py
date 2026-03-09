"""Tests for SQLite oplog sync service behavior."""

from __future__ import annotations

from datetime import datetime, timedelta, timezone

import aiosqlite
import pytest
from htmlgraph.api.oplog_sync import OplogSyncService
from htmlgraph.api.sync_models import OplogAckRequest, OplogEntry
from htmlgraph.db.schema import HtmlGraphDB


@pytest.fixture
def sync_db_path(tmp_path):
    db_path = tmp_path / "sync.db"
    db = HtmlGraphDB(str(db_path))
    db.connect()
    db.create_tables()
    db.disconnect()
    return str(db_path)


@pytest.mark.asyncio
async def test_push_pull_ack_status_roundtrip(sync_db_path: str) -> None:
    service = OplogSyncService(sync_db_path)
    now = datetime.now(timezone.utc)

    push = await service.push_entries(
        [
            OplogEntry(
                entry_id="entry-1",
                entity_type="feature",
                entity_id="feat-1",
                op="update",
                payload={"status": "in-progress"},
                actor="agent-a",
                ts=now,
                idempotency_key="idemp-1",
                field_mask=["status"],
            ),
            OplogEntry(
                entry_id="entry-2",
                entity_type="feature",
                entity_id="feat-2",
                op="create",
                payload={"title": "Feature 2"},
                actor="agent-a",
                ts=now + timedelta(seconds=1),
                idempotency_key="idemp-2",
            ),
        ],
        consumer_id="consumer-a",
    )
    assert push.inserted_count == 2
    assert push.deduped_count == 0
    assert push.conflict_count == 0
    assert push.applied_seq >= 2

    pulled = await service.pull_entries(
        since_seq=0, limit=100, consumer_id="consumer-a"
    )
    assert pulled.count == 2
    assert pulled.entries[0]["seq"] < pulled.entries[1]["seq"]
    assert pulled.server_max_seq == push.applied_seq

    acked = await service.ack_cursor(
        OplogAckRequest(
            consumer_id="consumer-a",
            last_seen_seq=push.applied_seq,
            last_acked_seq=push.applied_seq - 1,
        )
    )
    assert acked.cursor.last_seen_seq == push.applied_seq
    assert acked.cursor.last_acked_seq == push.applied_seq - 1

    status = await service.get_status(consumer_id="consumer-a")
    assert status.server_max_seq == push.applied_seq
    assert status.max_consumer_lag == 1
    assert status.pending_conflicts == 0
    assert len(status.consumers) == 1


@pytest.mark.asyncio
async def test_push_is_idempotent_by_idempotency_key(sync_db_path: str) -> None:
    service = OplogSyncService(sync_db_path)
    ts = datetime.now(timezone.utc)

    first = await service.push_entries(
        [
            OplogEntry(
                entry_id="entry-1",
                entity_type="feature",
                entity_id="feat-1",
                op="update",
                payload={"status": "todo"},
                actor="agent-a",
                ts=ts,
                idempotency_key="idemp-1",
            )
        ]
    )
    assert first.inserted_count == 1

    second = await service.push_entries(
        [
            OplogEntry(
                entry_id="entry-1-retry",
                entity_type="feature",
                entity_id="feat-1",
                op="update",
                payload={"status": "todo"},
                actor="agent-a",
                ts=ts,
                idempotency_key="idemp-1",
            )
        ]
    )
    assert second.inserted_count == 0
    assert second.deduped_count == 1
    assert second.results[0].deduped is True
    assert second.results[0].seq == first.results[0].seq


@pytest.mark.asyncio
async def test_conflict_recorded_for_overlapping_fields(sync_db_path: str) -> None:
    service = OplogSyncService(sync_db_path)
    ts = datetime.now(timezone.utc)

    await service.push_entries(
        [
            OplogEntry(
                entry_id="entry-a",
                entity_type="feature",
                entity_id="feat-1",
                op="update",
                payload={"status": "todo", "priority": "high"},
                actor="agent-a",
                ts=ts,
                idempotency_key="idemp-a",
                field_mask=["status", "priority"],
            )
        ]
    )
    second = await service.push_entries(
        [
            OplogEntry(
                entry_id="entry-b",
                entity_type="feature",
                entity_id="feat-1",
                op="update",
                payload={"status": "done"},
                actor="agent-b",
                ts=ts + timedelta(seconds=1),
                idempotency_key="idemp-b",
                field_mask=["status"],
            )
        ]
    )
    assert second.conflict_count == 1
    assert second.results[0].conflict_id is not None
    assert second.results[0].winner_entry_id == "entry-b"

    async with aiosqlite.connect(sync_db_path) as db:
        cursor = await db.execute(
            "SELECT COUNT(*) FROM sync_conflicts WHERE local_entry_id = ?",
            ["entry-b"],
        )
        row = await cursor.fetchone()
    assert row is not None
    assert int(row[0]) == 1
