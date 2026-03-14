"""
Tests for Wisps — ephemeral cross-agent coordination signals.

Tests cover:
1. test_wisp_signal — create a wisp, verify it exists in DB
2. test_wisp_active_filters_expired — expired wisps not returned
3. test_wisp_lazy_expiry — expired wisps deleted from DB on active() call
4. test_wisp_ttl_default — default TTL is 3600 seconds
5. test_wisp_lifecycle — signal, read active, wait for expiry, verify gone
6. test_wisps_cigs_inclusion — wisps appear in CIGS guidance when active
"""

from __future__ import annotations

import sqlite3
from datetime import datetime, timedelta, timezone
from pathlib import Path

import pytest
from htmlgraph import SDK
from htmlgraph.hooks.context import HookContext
from htmlgraph.hooks.prompt_analyzer import (
    _build_wisps_block,
    generate_guidance,
    get_active_wisps,
)

# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------


@pytest.fixture()
def tmp_htmlgraph(tmp_path: Path) -> Path:
    """Create a minimal .htmlgraph directory with an initialised DB."""
    hg_dir = tmp_path / ".htmlgraph"
    hg_dir.mkdir(parents=True)
    # Subdirs expected by SDK
    for subdir in ("features", "bugs", "spikes", "sessions", "tracks"):
        (hg_dir / subdir).mkdir()
    return hg_dir


@pytest.fixture()
def sdk(tmp_htmlgraph: Path) -> SDK:
    """Return an SDK instance pointing at a temp .htmlgraph directory."""
    return SDK(directory=tmp_htmlgraph, agent="test-agent")


@pytest.fixture()
def hook_context(tmp_htmlgraph: Path) -> HookContext:
    """Return a HookContext pointing at the same temp .htmlgraph directory."""
    return HookContext(
        project_dir=str(tmp_htmlgraph.parent),
        graph_dir=tmp_htmlgraph,
        session_id="sess-test-wisps",
        agent_id="test-agent",
        hook_input={},
    )


# ---------------------------------------------------------------------------
# Helper
# ---------------------------------------------------------------------------


def _count_wisps(db_path: Path) -> int:
    """Count all rows in the wisps table."""
    conn = sqlite3.connect(str(db_path))
    try:
        cursor = conn.execute("SELECT COUNT(*) FROM wisps")
        return cursor.fetchone()[0]
    finally:
        conn.close()


def _insert_expired_wisp(db_path: Path, wisp_id: str = "wisp-expired01") -> None:
    """Insert a wisp that already expired."""
    past = (datetime.now(timezone.utc) - timedelta(hours=2)).isoformat()
    conn = sqlite3.connect(str(db_path))
    try:
        conn.execute(
            "INSERT INTO wisps (id, agent_id, message, category, created_at, expires_at) "
            "VALUES (?, ?, ?, ?, ?, ?)",
            (wisp_id, "old-agent", "old message", "general", past, past),
        )
        conn.commit()
    finally:
        conn.close()


# ---------------------------------------------------------------------------
# Tests
# ---------------------------------------------------------------------------


class TestWispSignal:
    """test_wisp_signal — create a wisp, verify it exists in DB."""

    def test_signal_creates_row(self, sdk: SDK, tmp_htmlgraph: Path) -> None:
        wisp = sdk.wisps.signal("Starting migration", ttl=3600, category="warning")

        assert wisp["id"].startswith("wisp-")
        assert wisp["agent_id"] == "test-agent"
        assert wisp["message"] == "Starting migration"
        assert wisp["category"] == "warning"

        # Verify the row is in the DB
        assert _count_wisps(tmp_htmlgraph / "htmlgraph.db") == 1

    def test_signal_returns_dict_with_all_fields(self, sdk: SDK) -> None:
        wisp = sdk.wisps.signal("Hello", ttl=60)

        required_keys = {
            "id",
            "agent_id",
            "message",
            "category",
            "created_at",
            "expires_at",
        }
        assert required_keys.issubset(wisp.keys())

    def test_signal_default_category(self, sdk: SDK) -> None:
        wisp = sdk.wisps.signal("No category specified")
        assert wisp["category"] == "general"

    def test_signal_multiple_wisps(self, sdk: SDK, tmp_htmlgraph: Path) -> None:
        sdk.wisps.signal("First signal")
        sdk.wisps.signal("Second signal")
        sdk.wisps.signal("Third signal")
        assert _count_wisps(tmp_htmlgraph / "htmlgraph.db") == 3


class TestWispActiveFiltersExpired:
    """test_wisp_active_filters_expired — expired wisps not returned by active()."""

    def test_expired_not_returned(self, sdk: SDK, tmp_htmlgraph: Path) -> None:
        # Insert an expired wisp directly into DB
        _insert_expired_wisp(tmp_htmlgraph / "htmlgraph.db")

        # active() should not return it
        active = sdk.wisps.active()
        assert len(active) == 0

    def test_active_returns_non_expired(self, sdk: SDK) -> None:
        sdk.wisps.signal("Still valid", ttl=3600)
        active = sdk.wisps.active()
        assert len(active) == 1
        assert active[0]["message"] == "Still valid"

    def test_mixed_expired_and_active(self, sdk: SDK, tmp_htmlgraph: Path) -> None:
        _insert_expired_wisp(tmp_htmlgraph / "htmlgraph.db", wisp_id="wisp-old00001")
        sdk.wisps.signal("Still valid", ttl=3600)

        active = sdk.wisps.active()
        assert len(active) == 1
        assert active[0]["message"] == "Still valid"

    def test_active_filters_by_category(self, sdk: SDK) -> None:
        sdk.wisps.signal("Warning one", ttl=3600, category="warning")
        sdk.wisps.signal("Info one", ttl=3600, category="info")
        sdk.wisps.signal("Warning two", ttl=3600, category="warning")

        warnings = sdk.wisps.active(category="warning")
        assert len(warnings) == 2
        assert all(w["category"] == "warning" for w in warnings)

        infos = sdk.wisps.active(category="info")
        assert len(infos) == 1


class TestWispLazyExpiry:
    """test_wisp_lazy_expiry — expired wisps deleted from DB on active() call."""

    def test_active_deletes_expired_rows(self, sdk: SDK, tmp_htmlgraph: Path) -> None:
        db_path = tmp_htmlgraph / "htmlgraph.db"

        # Insert expired wisp directly
        _insert_expired_wisp(db_path)
        assert _count_wisps(db_path) == 1

        # active() triggers lazy expiry
        sdk.wisps.active()

        # Expired row should now be gone
        assert _count_wisps(db_path) == 0

    def test_active_only_deletes_expired(self, sdk: SDK, tmp_htmlgraph: Path) -> None:
        db_path = tmp_htmlgraph / "htmlgraph.db"
        _insert_expired_wisp(db_path)
        sdk.wisps.signal("Still valid", ttl=3600)

        assert _count_wisps(db_path) == 2

        sdk.wisps.active()

        # Only the expired one should be gone
        assert _count_wisps(db_path) == 1


class TestWispTtlDefault:
    """test_wisp_ttl_default — default TTL is 3600 seconds."""

    def test_default_ttl_approx_one_hour(self, sdk: SDK) -> None:
        wisp = sdk.wisps.signal("Default TTL test")

        expires_dt = datetime.fromisoformat(wisp["expires_at"])
        if expires_dt.tzinfo is None:
            expires_dt = expires_dt.replace(tzinfo=timezone.utc)

        created_dt = datetime.fromisoformat(wisp["created_at"])
        if created_dt.tzinfo is None:
            created_dt = created_dt.replace(tzinfo=timezone.utc)

        delta = expires_dt - created_dt
        # Should be approximately 3600 seconds (allow 2s tolerance for test timing)
        assert abs(delta.total_seconds() - 3600) < 2

    def test_custom_ttl(self, sdk: SDK) -> None:
        wisp = sdk.wisps.signal("Short-lived", ttl=60)

        expires_dt = datetime.fromisoformat(wisp["expires_at"])
        if expires_dt.tzinfo is None:
            expires_dt = expires_dt.replace(tzinfo=timezone.utc)
        created_dt = datetime.fromisoformat(wisp["created_at"])
        if created_dt.tzinfo is None:
            created_dt = created_dt.replace(tzinfo=timezone.utc)

        delta = expires_dt - created_dt
        assert abs(delta.total_seconds() - 60) < 2


class TestWispLifecycle:
    """test_wisp_lifecycle — signal, read active, expire, verify gone."""

    def test_lifecycle_with_manual_expire(self, sdk: SDK, tmp_htmlgraph: Path) -> None:
        db_path = tmp_htmlgraph / "htmlgraph.db"

        # 1. Signal a wisp
        wisp = sdk.wisps.signal("Lifecycle test", ttl=3600)
        assert _count_wisps(db_path) == 1

        # 2. It appears in active()
        active = sdk.wisps.active()
        assert len(active) == 1
        assert active[0]["id"] == wisp["id"]

        # 3. Manually expire it by updating expires_at in the DB
        conn = sqlite3.connect(str(db_path))
        past = (datetime.now(timezone.utc) - timedelta(seconds=1)).isoformat()
        conn.execute("UPDATE wisps SET expires_at = ? WHERE id = ?", (past, wisp["id"]))
        conn.commit()
        conn.close()

        # 4. Now active() should not return it and should delete it
        active_after = sdk.wisps.active()
        assert len(active_after) == 0

        # 5. Row should be deleted from DB
        assert _count_wisps(db_path) == 0

    def test_explicit_expire_method(self, sdk: SDK, tmp_htmlgraph: Path) -> None:
        db_path = tmp_htmlgraph / "htmlgraph.db"

        # Insert expired wisp directly
        _insert_expired_wisp(db_path)

        # expire() should delete it and return count
        count = sdk.wisps.expire()
        assert count == 1
        assert _count_wisps(db_path) == 0

    def test_expire_returns_zero_when_nothing_expired(self, sdk: SDK) -> None:
        sdk.wisps.signal("Still valid", ttl=3600)
        count = sdk.wisps.expire()
        assert count == 0


class TestWispsCigsInclusion:
    """test_wisps_cigs_inclusion — wisps appear in CIGS guidance when active."""

    def test_wisps_appear_in_get_active_wisps(
        self, sdk: SDK, hook_context: HookContext
    ) -> None:
        # Publish a wisp
        sdk.wisps.signal("DB migration in progress", ttl=3600, category="block")

        # Fetch active wisps via prompt_analyzer function
        active = get_active_wisps(hook_context)
        assert len(active) == 1
        assert active[0]["message"] == "DB migration in progress"

    def test_build_wisps_block_format(self) -> None:
        wisps = [
            {
                "id": "wisp-abc12345",
                "agent_id": "haiku-coder",
                "message": "Writing to src/python/",
                "category": "warning",
                "created_at": datetime.now(timezone.utc).isoformat(),
                "expires_at": (
                    datetime.now(timezone.utc) + timedelta(hours=1)
                ).isoformat(),
            }
        ]
        block = _build_wisps_block(wisps)
        assert block is not None
        assert "haiku-coder" in block
        assert "warning" in block
        assert "Writing to src/python/" in block
        assert "expires in" in block

    def test_build_wisps_block_empty_returns_none(self) -> None:
        assert _build_wisps_block([]) is None

    def test_generate_guidance_includes_wisps(
        self, sdk: SDK, hook_context: HookContext
    ) -> None:
        # Publish a wisp
        sdk.wisps.signal("Migrating DB schema now", ttl=3600, category="block")

        active_wisps = get_active_wisps(hook_context)
        assert len(active_wisps) == 1

        # generate_guidance should include the wisps block
        from htmlgraph.hooks.prompt_analyzer import classify_prompt

        classification = classify_prompt("hello world")
        guidance = generate_guidance(
            classification,
            active_work=None,
            prompt="hello world",
            open_work_items=None,
            active_wisps=active_wisps,
        )
        assert guidance is not None
        assert "Migrating DB schema now" in guidance
        assert "block" in guidance

    def test_get_active_wisps_empty_when_no_wisps(
        self, hook_context: HookContext
    ) -> None:
        wisps = get_active_wisps(hook_context)
        assert wisps == []

    def test_get_active_wisps_no_db(self, tmp_path: Path) -> None:
        """Returns empty list gracefully when no DB exists."""
        context = HookContext(
            project_dir=str(tmp_path),
            graph_dir=tmp_path / "nonexistent",
            session_id="sess-test",
            agent_id="test",
            hook_input={},
        )
        result = get_active_wisps(context)
        assert result == []
