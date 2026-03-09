"""Pydantic models for SQLite oplog sync endpoints."""

from __future__ import annotations

from datetime import datetime
from typing import Any, Literal

from pydantic import BaseModel, ConfigDict, Field, field_validator, model_validator

AllowedOp = Literal["create", "update", "delete", "upsert", "patch"]


class OplogEntry(BaseModel):
    """Incoming oplog mutation entry."""

    model_config = ConfigDict(extra="forbid")

    entry_id: str = Field(min_length=1)
    entity_type: str = Field(min_length=1)
    entity_id: str = Field(min_length=1)
    op: AllowedOp
    payload: dict[str, Any] = Field(default_factory=dict)
    actor: str = Field(min_length=1)
    ts: datetime
    idempotency_key: str = Field(min_length=1)
    field_mask: list[str] | None = None
    session_id: str | None = None

    @field_validator("field_mask")
    @classmethod
    def _validate_field_mask(cls, value: list[str] | None) -> list[str] | None:
        if value is None:
            return value
        filtered = [f.strip() for f in value if f and f.strip()]
        return filtered or None


class OplogPushRequest(BaseModel):
    """Request body for /api/sync/push."""

    model_config = ConfigDict(extra="forbid")

    consumer_id: str | None = None
    entries: list[OplogEntry] = Field(min_length=1)


class OplogPushResult(BaseModel):
    """Per-entry push result."""

    model_config = ConfigDict(extra="forbid")

    entry_id: str
    seq: int
    deduped: bool = False
    conflict_id: str | None = None
    winner_entry_id: str | None = None


class OplogPushResponse(BaseModel):
    """Response body for /api/sync/push."""

    model_config = ConfigDict(extra="forbid")

    applied_seq: int
    inserted_count: int
    deduped_count: int
    conflict_count: int
    results: list[OplogPushResult]


class CursorState(BaseModel):
    """Per-consumer cursor state."""

    model_config = ConfigDict(extra="forbid")

    consumer_id: str = Field(min_length=1)
    last_seen_seq: int = Field(ge=0)
    last_acked_seq: int = Field(ge=0)
    updated_at: datetime

    @model_validator(mode="after")
    def _validate_cursor(self) -> CursorState:
        if self.last_acked_seq > self.last_seen_seq:
            raise ValueError("last_acked_seq cannot exceed last_seen_seq")
        return self


class OplogPullResponse(BaseModel):
    """Response body for /api/sync/pull."""

    model_config = ConfigDict(extra="forbid")

    since_seq: int = Field(ge=0)
    server_max_seq: int = Field(ge=0)
    entries: list[dict[str, Any]]
    count: int = Field(ge=0)


class OplogAckRequest(BaseModel):
    """Request body for /api/sync/ack."""

    model_config = ConfigDict(extra="forbid")

    consumer_id: str = Field(min_length=1)
    last_seen_seq: int = Field(ge=0)
    last_acked_seq: int = Field(ge=0)

    @model_validator(mode="after")
    def _validate_ack(self) -> OplogAckRequest:
        if self.last_acked_seq > self.last_seen_seq:
            raise ValueError("last_acked_seq cannot exceed last_seen_seq")
        return self


class OplogAckResponse(BaseModel):
    """Response body for /api/sync/ack."""

    model_config = ConfigDict(extra="forbid")

    cursor: CursorState


class ConflictRecord(BaseModel):
    """Conflict record emitted for overlapping concurrent mutations."""

    model_config = ConfigDict(extra="forbid")

    conflict_id: str
    local_entry_id: str
    remote_entry_id: str
    entity_type: str
    entity_id: str
    policy: str
    resolution: str
    status: str
    created_at: datetime


class SyncStatusResponse(BaseModel):
    """Response body for /api/sync/status."""

    model_config = ConfigDict(extra="forbid")

    health: str
    server_max_seq: int = Field(ge=0)
    pending_conflicts: int = Field(ge=0)
    max_consumer_lag: int = Field(ge=0)
    consumers: list[CursorState]
