"""Tests for atomic claim primitive (SQL compare-and-swap)."""

from __future__ import annotations

import threading
from pathlib import Path

import pytest
from htmlgraph import SDK


@pytest.fixture
def sdk(tmp_path: Path) -> SDK:
    """Isolated SDK instance backed by a fresh tmp directory."""
    graph_dir = tmp_path / ".htmlgraph"
    graph_dir.mkdir()
    for subdir in ("features", "bugs", "spikes"):
        (graph_dir / subdir).mkdir()
    db_path = str(tmp_path / "test.db")
    return SDK(directory=str(graph_dir), agent="test-agent", db_path=db_path)


def _make_feature(sdk: SDK, title: str = "Test Feature") -> str:
    """Create a feature and insert it into SQLite; return its ID."""
    feature = sdk.features.create(title).set_track(None).save()  # type: ignore[attr-defined]
    return feature.id


def _make_feature_simple(sdk: SDK, title: str = "Test Feature") -> str:
    """Create a feature without requiring a track (uses builder if available, else fallback)."""
    from htmlgraph.ids import generate_id
    from htmlgraph.models import Node

    node_id = generate_id(node_type="feature", title=title)
    node = Node(
        id=node_id, title=title, type="feature", priority="medium", status="todo"
    )
    sdk._graph.add(node)

    # Also insert into SQLite so the SQL CAS has a row to update
    sdk._db.insert_feature(
        feature_id=node_id,
        feature_type="feature",
        title=title,
        status="todo",
    )
    return node_id


class TestAtomicClaim:
    def test_claim_unclaimed_succeeds(self, sdk: SDK) -> None:
        """Claiming an unclaimed item should succeed."""
        node_id = _make_feature_simple(sdk)

        result = sdk.features.atomic_claim(node_id, agent="agent-a")

        assert result is True

        # Verify DB row updated
        row = sdk._db.connection.execute(
            "SELECT assignee FROM features WHERE id = ?", (node_id,)
        ).fetchone()
        assert row is not None
        assert row[0] == "agent-a"

    def test_claim_already_claimed_by_other_fails(self, sdk: SDK) -> None:
        """Claiming an item already claimed by another agent should fail."""
        node_id = _make_feature_simple(sdk)

        # agent-a claims first
        assert sdk.features.atomic_claim(node_id, agent="agent-a") is True

        # agent-b tries to claim the same item — must lose
        result = sdk.features.atomic_claim(node_id, agent="agent-b")

        assert result is False

        # Assignee remains agent-a
        row = sdk._db.connection.execute(
            "SELECT assignee FROM features WHERE id = ?", (node_id,)
        ).fetchone()
        assert row[0] == "agent-a"

    def test_claim_own_item_idempotent(self, sdk: SDK) -> None:
        """Re-claiming your own item should succeed (idempotent)."""
        node_id = _make_feature_simple(sdk)

        assert sdk.features.atomic_claim(node_id, agent="agent-a") is True
        assert sdk.features.atomic_claim(node_id, agent="agent-a") is True

        row = sdk._db.connection.execute(
            "SELECT assignee FROM features WHERE id = ?", (node_id,)
        ).fetchone()
        assert row[0] == "agent-a"

    def test_unclaim_releases(self, sdk: SDK) -> None:
        """Unclaiming should allow another agent to claim."""
        node_id = _make_feature_simple(sdk)

        # agent-a claims
        assert sdk.features.atomic_claim(node_id, agent="agent-a") is True

        # agent-a unclaims
        sdk.features.atomic_unclaim(node_id)

        row = sdk._db.connection.execute(
            "SELECT assignee FROM features WHERE id = ?", (node_id,)
        ).fetchone()
        assert row[0] is None

        # Now agent-b can claim
        assert sdk.features.atomic_claim(node_id, agent="agent-b") is True

    def test_concurrent_claims_only_one_wins(self, tmp_path: Path) -> None:
        """Only one of two concurrent claims should succeed."""
        graph_dir = tmp_path / ".htmlgraph"
        graph_dir.mkdir()
        for subdir in ("features", "bugs", "spikes"):
            (graph_dir / subdir).mkdir()
        db_path = str(tmp_path / "test.db")

        sdk1 = SDK(directory=str(graph_dir), agent="agent-1", db_path=db_path)
        sdk2 = SDK(directory=str(graph_dir), agent="agent-2", db_path=db_path)

        # Create the feature via sdk1; sdk2 shares the same DB file
        node_id = _make_feature_simple(sdk1)

        results: list[bool] = []
        barrier = threading.Barrier(2)

        def do_claim(sdk: SDK, agent: str) -> None:
            barrier.wait()  # both threads start at the same moment
            results.append(sdk.features.atomic_claim(node_id, agent=agent))

        t1 = threading.Thread(target=do_claim, args=(sdk1, "agent-1"))
        t2 = threading.Thread(target=do_claim, args=(sdk2, "agent-2"))
        t1.start()
        t2.start()
        t1.join()
        t2.join()

        # Exactly one claim must have won
        assert results.count(True) == 1
        assert results.count(False) == 1

    def test_html_assignee_updated_on_claim(self, sdk: SDK, tmp_path: Path) -> None:
        """HTML file should gain data-assignee attribute after atomic_claim."""
        node_id = _make_feature_simple(sdk)

        # Create a minimal HTML file in the features dir so _update_html_assignee can find it
        html_path = Path(sdk._directory) / "features" / f"{node_id}.html"
        html_path.write_text(
            f'<html><body data-id="{node_id}" data-status="todo"></body></html>'
        )

        sdk.features.atomic_claim(node_id, agent="agent-a")

        content = html_path.read_text()
        assert 'data-assignee="agent-a"' in content

    def test_html_assignee_cleared_on_unclaim(self, sdk: SDK) -> None:
        """HTML file should lose data-assignee attribute after atomic_unclaim."""
        node_id = _make_feature_simple(sdk)

        html_path = Path(sdk._directory) / "features" / f"{node_id}.html"
        html_path.write_text(
            f'<html><body data-id="{node_id}" data-assignee="agent-a" data-status="todo"></body></html>'
        )

        sdk.features.atomic_unclaim(node_id)

        content = html_path.read_text()
        assert "data-assignee" not in content
