"""Tests for oplog sync request/response models."""

from __future__ import annotations

from datetime import datetime, timezone

import pytest
from htmlgraph.api.sync_models import OplogAckRequest, OplogEntry
from pydantic import ValidationError


def test_oplog_entry_validates_required_fields() -> None:
    entry = OplogEntry(
        entry_id="entry-1",
        entity_type="feature",
        entity_id="feat-1",
        op="update",
        payload={"status": "done"},
        actor="agent-a",
        ts=datetime.now(timezone.utc),
        idempotency_key="idemp-1",
        field_mask=["status", " ", ""],
    )
    assert entry.field_mask == ["status"]


def test_oplog_entry_rejects_extra_fields() -> None:
    with pytest.raises(ValidationError):
        OplogEntry(
            entry_id="entry-1",
            entity_type="feature",
            entity_id="feat-1",
            op="update",
            payload={"status": "done"},
            actor="agent-a",
            ts=datetime.now(timezone.utc),
            idempotency_key="idemp-1",
            extra_field="nope",
        )


def test_ack_rejects_acked_beyond_seen() -> None:
    with pytest.raises(ValidationError):
        OplogAckRequest(
            consumer_id="consumer-a",
            last_seen_seq=10,
            last_acked_seq=11,
        )
