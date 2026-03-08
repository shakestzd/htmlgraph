"""SQLite oplog-based sync service for local-first coordination."""

from __future__ import annotations

import json
import logging
import uuid
from collections.abc import Sequence
from datetime import datetime, timezone
from typing import Any

import aiosqlite

from htmlgraph.api.sync_models import (
    ConflictRecord,
    CursorState,
    OplogAckRequest,
    OplogAckResponse,
    OplogEntry,
    OplogPullResponse,
    OplogPushResponse,
    OplogPushResult,
    SyncStatusResponse,
)
from htmlgraph.db.pragmas import apply_async_pragmas

logger = logging.getLogger(__name__)


class OplogSyncService:
    """Service implementing push/pull/ack/status over SQLite oplog."""

    def __init__(self, db_path: str):
        self.db_path = db_path

    async def push_entries(
        self, entries: Sequence[OplogEntry], consumer_id: str | None = None
    ) -> OplogPushResponse:
        results: list[OplogPushResult] = []
        inserted_count = 0
        deduped_count = 0
        conflict_count = 0
        applied_seq = 0

        async with aiosqlite.connect(self.db_path) as db:
            db.row_factory = aiosqlite.Row
            await apply_async_pragmas(db)
            await db.execute("BEGIN IMMEDIATE")

            try:
                for entry in entries:
                    existing = await self._find_existing_entry(db, entry)
                    if existing is not None:
                        deduped_count += 1
                        seq = int(existing["seq"])
                        applied_seq = max(applied_seq, seq)
                        results.append(
                            OplogPushResult(
                                entry_id=str(existing["entry_id"]),
                                seq=seq,
                                deduped=True,
                            )
                        )
                        continue

                    previous = await self._find_previous_entry(db, entry)

                    cursor = await db.execute(
                        """
                        INSERT INTO oplog
                        (entry_id, idempotency_key, entity_type, entity_id, op,
                         payload, actor, ts, field_mask, session_id)
                        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                        """,
                        [
                            entry.entry_id,
                            entry.idempotency_key,
                            entry.entity_type,
                            entry.entity_id,
                            entry.op,
                            json.dumps(entry.payload),
                            entry.actor,
                            entry.ts.isoformat(),
                            json.dumps(entry.field_mask or list(entry.payload.keys())),
                            entry.session_id,
                        ],
                    )
                    seq = cursor.lastrowid if cursor.lastrowid is not None else 0
                    applied_seq = max(applied_seq, seq)
                    inserted_count += 1

                    conflict_id: str | None = None
                    winner_entry_id: str | None = None
                    if previous and self._is_conflict(previous, entry):
                        conflict = await self._record_conflict(db, previous, entry)
                        conflict_id = conflict.conflict_id
                        winner_entry_id = conflict.resolution
                        conflict_count += 1
                        await self._emit_live_event(
                            db,
                            event_type="sync_conflict",
                            payload={
                                "conflict_id": conflict.conflict_id,
                                "entity_type": conflict.entity_type,
                                "entity_id": conflict.entity_id,
                                "winner_entry_id": conflict.resolution,
                                "policy": conflict.policy,
                            },
                            session_id=entry.session_id,
                        )

                    await self._emit_live_event(
                        db,
                        event_type="sync_applied",
                        payload={
                            "entry_id": entry.entry_id,
                            "seq": seq,
                            "entity_type": entry.entity_type,
                            "entity_id": entry.entity_id,
                            "actor": entry.actor,
                        },
                        session_id=entry.session_id,
                    )

                    results.append(
                        OplogPushResult(
                            entry_id=entry.entry_id,
                            seq=seq,
                            deduped=False,
                            conflict_id=conflict_id,
                            winner_entry_id=winner_entry_id,
                        )
                    )

                if consumer_id:
                    await self._upsert_cursor(
                        db=db,
                        consumer_id=consumer_id,
                        last_seen_seq=applied_seq,
                        last_acked_seq=0,
                    )

                await db.commit()
            except Exception:
                await db.rollback()
                raise

        return OplogPushResponse(
            applied_seq=applied_seq,
            inserted_count=inserted_count,
            deduped_count=deduped_count,
            conflict_count=conflict_count,
            results=results,
        )

    async def pull_entries(
        self,
        *,
        since_seq: int = 0,
        limit: int = 200,
        consumer_id: str | None = None,
    ) -> OplogPullResponse:
        safe_limit = min(max(limit, 1), 1000)
        entries: list[dict[str, Any]] = []
        server_max_seq = 0

        async with aiosqlite.connect(self.db_path) as db:
            db.row_factory = aiosqlite.Row
            await apply_async_pragmas(db)

            max_cursor = await db.execute(
                "SELECT COALESCE(MAX(seq), 0) as max_seq FROM oplog"
            )
            max_row = await max_cursor.fetchone()
            server_max_seq = int(max_row["max_seq"]) if max_row else 0

            cursor = await db.execute(
                """
                SELECT seq, entry_id, idempotency_key, entity_type, entity_id, op,
                       payload, actor, ts, field_mask, session_id, created_at
                FROM oplog
                WHERE seq > ?
                ORDER BY seq ASC
                LIMIT ?
                """,
                [since_seq, safe_limit],
            )
            rows = await cursor.fetchall()

            for row in rows:
                payload = json.loads(row["payload"]) if row["payload"] else {}
                field_mask = (
                    json.loads(row["field_mask"])
                    if row["field_mask"]
                    else list(payload.keys())
                )
                entries.append(
                    {
                        "seq": int(row["seq"]),
                        "entry_id": str(row["entry_id"]),
                        "idempotency_key": str(row["idempotency_key"]),
                        "entity_type": str(row["entity_type"]),
                        "entity_id": str(row["entity_id"]),
                        "op": str(row["op"]),
                        "payload": payload,
                        "actor": str(row["actor"]),
                        "ts": str(row["ts"]),
                        "field_mask": field_mask,
                        "session_id": row["session_id"],
                        "created_at": row["created_at"],
                    }
                )

            if consumer_id:
                last_seen = entries[-1]["seq"] if entries else since_seq
                await self._upsert_cursor(
                    db=db,
                    consumer_id=consumer_id,
                    last_seen_seq=int(last_seen),
                    last_acked_seq=0,
                )
                await db.commit()

        return OplogPullResponse(
            since_seq=since_seq,
            server_max_seq=server_max_seq,
            entries=entries,
            count=len(entries),
        )

    async def ack_cursor(self, request: OplogAckRequest) -> OplogAckResponse:
        async with aiosqlite.connect(self.db_path) as db:
            db.row_factory = aiosqlite.Row
            await apply_async_pragmas(db)

            cursor_state = await self._upsert_cursor(
                db=db,
                consumer_id=request.consumer_id,
                last_seen_seq=request.last_seen_seq,
                last_acked_seq=request.last_acked_seq,
            )
            await db.commit()

        return OplogAckResponse(cursor=cursor_state)

    async def get_status(self, consumer_id: str | None = None) -> SyncStatusResponse:
        async with aiosqlite.connect(self.db_path) as db:
            db.row_factory = aiosqlite.Row
            await apply_async_pragmas(db)

            max_cursor = await db.execute(
                "SELECT COALESCE(MAX(seq), 0) as max_seq FROM oplog"
            )
            max_row = await max_cursor.fetchone()
            server_max_seq = int(max_row["max_seq"]) if max_row else 0

            conflict_cursor = await db.execute(
                "SELECT COUNT(*) as cnt FROM sync_conflicts WHERE status != 'resolved'"
            )
            conflict_row = await conflict_cursor.fetchone()
            pending_conflicts = int(conflict_row["cnt"]) if conflict_row else 0

            if consumer_id:
                consumer_cursor = await db.execute(
                    """
                    SELECT consumer_id, last_seen_seq, last_acked_seq, updated_at
                    FROM sync_cursors
                    WHERE consumer_id = ?
                    """,
                    [consumer_id],
                )
            else:
                consumer_cursor = await db.execute(
                    """
                    SELECT consumer_id, last_seen_seq, last_acked_seq, updated_at
                    FROM sync_cursors
                    ORDER BY updated_at DESC
                    """
                )
            consumer_rows = await consumer_cursor.fetchall()

        consumers = [
            CursorState(
                consumer_id=str(row["consumer_id"]),
                last_seen_seq=int(row["last_seen_seq"]),
                last_acked_seq=int(row["last_acked_seq"]),
                updated_at=datetime.fromisoformat(
                    str(row["updated_at"]).replace("Z", "+00:00")
                )
                if "T" in str(row["updated_at"])
                else datetime.fromisoformat(str(row["updated_at"]).replace(" ", "T")),
            )
            for row in consumer_rows
        ]
        max_consumer_lag = (
            max(server_max_seq - c.last_acked_seq for c in consumers)
            if consumers
            else 0
        )

        return SyncStatusResponse(
            health="degraded" if pending_conflicts > 0 else "ok",
            server_max_seq=server_max_seq,
            pending_conflicts=pending_conflicts,
            max_consumer_lag=max_consumer_lag,
            consumers=consumers,
        )

    async def _find_existing_entry(
        self, db: aiosqlite.Connection, entry: OplogEntry
    ) -> aiosqlite.Row | None:
        cursor = await db.execute(
            """
            SELECT seq, entry_id
            FROM oplog
            WHERE entry_id = ? OR idempotency_key = ?
            ORDER BY seq DESC
            LIMIT 1
            """,
            [entry.entry_id, entry.idempotency_key],
        )
        return await cursor.fetchone()

    async def _find_previous_entry(
        self, db: aiosqlite.Connection, entry: OplogEntry
    ) -> aiosqlite.Row | None:
        cursor = await db.execute(
            """
            SELECT seq, entry_id, entity_type, entity_id, actor, ts, payload, field_mask
            FROM oplog
            WHERE entity_type = ? AND entity_id = ?
            ORDER BY seq DESC
            LIMIT 1
            """,
            [entry.entity_type, entry.entity_id],
        )
        return await cursor.fetchone()

    def _is_conflict(self, previous: aiosqlite.Row, incoming: OplogEntry) -> bool:
        prev_actor = str(previous["actor"] or "")
        if prev_actor == incoming.actor:
            return False

        prev_payload = json.loads(previous["payload"]) if previous["payload"] else {}
        prev_fields = (
            set(json.loads(previous["field_mask"]))
            if previous["field_mask"]
            else set(prev_payload.keys())
        )
        incoming_fields = set(incoming.field_mask or list(incoming.payload.keys()))
        if not prev_fields or not incoming_fields:
            return False
        return bool(prev_fields.intersection(incoming_fields))

    async def _record_conflict(
        self, db: aiosqlite.Connection, previous: aiosqlite.Row, incoming: OplogEntry
    ) -> ConflictRecord:
        prev_ts = datetime.fromisoformat(str(previous["ts"]).replace("Z", "+00:00"))
        incoming_ts = incoming.ts
        if incoming_ts.tzinfo is None:
            incoming_ts = incoming_ts.replace(tzinfo=timezone.utc)
        if prev_ts.tzinfo is None:
            prev_ts = prev_ts.replace(tzinfo=timezone.utc)

        previous_entry_id = str(previous["entry_id"])
        if incoming_ts > prev_ts:
            winner_entry_id = incoming.entry_id
        elif incoming_ts < prev_ts:
            winner_entry_id = previous_entry_id
        else:
            winner_entry_id = max(incoming.entry_id, previous_entry_id)

        prev_payload = json.loads(previous["payload"]) if previous["payload"] else {}
        prev_fields = (
            set(json.loads(previous["field_mask"]))
            if previous["field_mask"]
            else set(prev_payload.keys())
        )
        incoming_fields = set(incoming.field_mask or list(incoming.payload.keys()))
        field_set = sorted(prev_fields.intersection(incoming_fields))

        conflict_id = f"sync-conf-{uuid.uuid4().hex[:12]}"
        now_iso = datetime.now(timezone.utc).isoformat()
        await db.execute(
            """
            INSERT INTO sync_conflicts
            (conflict_id, local_entry_id, remote_entry_id, entity_type, entity_id,
             field_set, policy, resolution, status, created_at)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
            """,
            [
                conflict_id,
                incoming.entry_id,
                previous_entry_id,
                incoming.entity_type,
                incoming.entity_id,
                json.dumps(field_set),
                "last_write_wins",
                winner_entry_id,
                "resolved",
                now_iso,
            ],
        )

        return ConflictRecord(
            conflict_id=conflict_id,
            local_entry_id=incoming.entry_id,
            remote_entry_id=previous_entry_id,
            entity_type=incoming.entity_type,
            entity_id=incoming.entity_id,
            policy="last_write_wins",
            resolution=winner_entry_id,
            status="resolved",
            created_at=datetime.fromisoformat(now_iso),
        )

    async def _upsert_cursor(
        self,
        *,
        db: aiosqlite.Connection,
        consumer_id: str,
        last_seen_seq: int,
        last_acked_seq: int,
    ) -> CursorState:
        cursor = await db.execute(
            """
            SELECT consumer_id, last_seen_seq, last_acked_seq
            FROM sync_cursors
            WHERE consumer_id = ?
            """,
            [consumer_id],
        )
        existing = await cursor.fetchone()

        if existing:
            seen = max(int(existing["last_seen_seq"]), int(last_seen_seq))
            acked = max(int(existing["last_acked_seq"]), int(last_acked_seq))
            acked = min(acked, seen)
        else:
            seen = int(last_seen_seq)
            acked = min(int(last_acked_seq), seen)

        now_iso = datetime.now(timezone.utc).isoformat()
        await db.execute(
            """
            INSERT INTO sync_cursors(consumer_id, last_seen_seq, last_acked_seq, updated_at)
            VALUES(?, ?, ?, ?)
            ON CONFLICT(consumer_id) DO UPDATE SET
                last_seen_seq=excluded.last_seen_seq,
                last_acked_seq=excluded.last_acked_seq,
                updated_at=excluded.updated_at
            """,
            [consumer_id, seen, acked, now_iso],
        )

        lag = max(seen - acked, 0)
        await self._emit_live_event(
            db,
            event_type="sync_lag",
            payload={
                "consumer_id": consumer_id,
                "last_seen_seq": seen,
                "last_acked_seq": acked,
                "lag": lag,
            },
            session_id=None,
        )

        return CursorState(
            consumer_id=consumer_id,
            last_seen_seq=seen,
            last_acked_seq=acked,
            updated_at=datetime.fromisoformat(now_iso),
        )

    async def _emit_live_event(
        self,
        db: aiosqlite.Connection,
        *,
        event_type: str,
        payload: dict[str, Any],
        session_id: str | None,
    ) -> None:
        await db.execute(
            """
            INSERT INTO live_events
            (event_type, event_data, parent_event_id, session_id, spawner_type)
            VALUES (?, ?, NULL, ?, 'sync')
            """,
            [event_type, json.dumps(payload), session_id],
        )
