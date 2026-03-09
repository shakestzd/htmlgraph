"""API tests for /api/sync oplog endpoints."""

from __future__ import annotations

from datetime import datetime, timedelta, timezone

import pytest
from fastapi.testclient import TestClient
from htmlgraph.api.main import get_app
from htmlgraph.db.schema import HtmlGraphDB


@pytest.fixture
def sync_api_client(tmp_path) -> TestClient:
    db_path = tmp_path / "sync-api.db"
    db = HtmlGraphDB(str(db_path))
    db.connect()
    db.create_tables()
    db.disconnect()
    app = get_app(str(db_path))
    return TestClient(app)


def test_sync_push_pull_ack_status(sync_api_client: TestClient) -> None:
    now = datetime.now(timezone.utc)
    push_resp = sync_api_client.post(
        "/api/sync/push",
        json={
            "consumer_id": "consumer-a",
            "entries": [
                {
                    "entry_id": "entry-1",
                    "entity_type": "feature",
                    "entity_id": "feat-1",
                    "op": "update",
                    "payload": {"status": "in-progress"},
                    "actor": "agent-a",
                    "ts": now.isoformat(),
                    "idempotency_key": "idemp-1",
                }
            ],
        },
    )
    assert push_resp.status_code == 200, push_resp.text
    push_body = push_resp.json()
    assert push_body["inserted_count"] == 1
    assert push_body["conflict_count"] == 0
    assert push_body["applied_seq"] >= 1

    pull_resp = sync_api_client.get(
        "/api/sync/pull",
        params={"since_seq": 0, "limit": 50, "consumer_id": "consumer-a"},
    )
    assert pull_resp.status_code == 200, pull_resp.text
    pull_body = pull_resp.json()
    assert pull_body["count"] == 1
    assert pull_body["entries"][0]["entry_id"] == "entry-1"

    ack_resp = sync_api_client.post(
        "/api/sync/ack",
        json={
            "consumer_id": "consumer-a",
            "last_seen_seq": push_body["applied_seq"],
            "last_acked_seq": push_body["applied_seq"],
        },
    )
    assert ack_resp.status_code == 200, ack_resp.text
    assert ack_resp.json()["cursor"]["last_acked_seq"] == push_body["applied_seq"]

    status_resp = sync_api_client.get(
        "/api/sync/status", params={"consumer_id": "consumer-a"}
    )
    assert status_resp.status_code == 200, status_resp.text
    status_body = status_resp.json()
    assert status_body["server_max_seq"] == push_body["applied_seq"]
    assert status_body["max_consumer_lag"] == 0


def test_sync_conflict_is_recorded(sync_api_client: TestClient) -> None:
    now = datetime.now(timezone.utc)
    first = sync_api_client.post(
        "/api/sync/push",
        json={
            "entries": [
                {
                    "entry_id": "entry-a",
                    "entity_type": "feature",
                    "entity_id": "feat-1",
                    "op": "update",
                    "payload": {"status": "todo", "priority": "high"},
                    "field_mask": ["status", "priority"],
                    "actor": "agent-a",
                    "ts": now.isoformat(),
                    "idempotency_key": "idemp-a",
                }
            ]
        },
    )
    assert first.status_code == 200, first.text

    second = sync_api_client.post(
        "/api/sync/push",
        json={
            "entries": [
                {
                    "entry_id": "entry-b",
                    "entity_type": "feature",
                    "entity_id": "feat-1",
                    "op": "update",
                    "payload": {"status": "done"},
                    "field_mask": ["status"],
                    "actor": "agent-b",
                    "ts": (now + timedelta(seconds=1)).isoformat(),
                    "idempotency_key": "idemp-b",
                }
            ]
        },
    )
    assert second.status_code == 200, second.text
    body = second.json()
    assert body["conflict_count"] == 1
    assert body["results"][0]["conflict_id"] is not None
