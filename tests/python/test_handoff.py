"""
Tests for the handoff context system.

Tests the ability to hand off tasks between agents with preserved context,
including metadata serialization and lightweight context generation.
"""

from datetime import datetime
from pathlib import Path
from tempfile import TemporaryDirectory

import pytest
from htmlgraph.converter import html_to_session, session_to_html
from htmlgraph.models import Node, Session
from htmlgraph.sdk import SDK
from htmlgraph.session_manager import SessionManager


class TestHandoffFields:
    """Test handoff fields on Node model."""

    def test_node_has_handoff_fields(self, isolated_db):
        """Verify Node model has all required handoff fields."""
        node = Node(
            id="feature-001",
            title="Test Feature",
        )

        assert hasattr(node, "handoff_required")
        assert hasattr(node, "previous_agent")
        assert hasattr(node, "handoff_reason")
        assert hasattr(node, "handoff_notes")
        assert hasattr(node, "handoff_timestamp")

    def test_handoff_fields_default_to_none(self, isolated_db):
        """Verify handoff fields have sensible defaults."""
        node = Node(
            id="feature-001",
            title="Test Feature",
        )

        assert node.handoff_required is False
        assert node.previous_agent is None
        assert node.handoff_reason is None
        assert node.handoff_notes is None
        assert node.handoff_timestamp is None

    def test_handoff_fields_can_be_set(self, isolated_db):
        """Verify handoff fields can be set on creation."""
        now = datetime.now()
        node = Node(
            id="feature-001",
            title="Test Feature",
            handoff_required=True,
            previous_agent="alice",
            handoff_reason="blocked on dependency",
            handoff_notes="Waiting for database migration",
            handoff_timestamp=now,
        )

        assert node.handoff_required is True
        assert node.previous_agent == "alice"
        assert node.handoff_reason == "blocked on dependency"
        assert node.handoff_notes == "Waiting for database migration"
        assert node.handoff_timestamp == now


class TestHandoffHTML:
    """Test HTML serialization of handoff context."""

    def test_node_without_handoff_has_no_section(self, isolated_db):
        """Verify HTML doesn't include handoff section if not required."""
        node = Node(
            id="feature-001",
            title="Test Feature",
        )
        html = node.to_html()

        assert "data-handoff" not in html
        assert "Handoff Context" not in html

    def test_node_with_handoff_includes_section(self, isolated_db):
        """Verify HTML includes handoff section when required."""
        node = Node(
            id="feature-001",
            title="Test Feature",
            handoff_required=True,
            previous_agent="alice",
            handoff_reason="blocked on dependency",
        )
        html = node.to_html()

        assert "<section data-handoff" in html
        assert "Handoff Context" in html
        assert "alice" in html
        assert "blocked on dependency" in html

    def test_handoff_html_includes_all_fields(self, isolated_db):
        """Verify all handoff fields appear in HTML."""
        now = datetime.now()
        node = Node(
            id="feature-001",
            title="Test Feature",
            handoff_required=True,
            previous_agent="alice",
            handoff_reason="blocked on dependency",
            handoff_notes="Waiting for database migration",
            handoff_timestamp=now,
        )
        html = node.to_html()

        assert "data-previous-agent=" in html
        assert "alice" in html
        assert "data-reason=" in html
        assert "blocked on dependency" in html
        assert "data-timestamp=" in html
        assert "Waiting for database migration" in html
        assert now.isoformat() in html


class TestSessionHandoffHTML:
    """Test Session handoff HTML serialization."""

    def test_session_handoff_roundtrip(self, isolated_db):
        """Verify session handoff fields persist through HTML serialization."""
        with TemporaryDirectory() as tmpdir:
            session = Session(
                id="session-001",
                agent="tester",
                handoff_notes="Wrap up payment refactor",
                recommended_next="Add migration tests",
                blockers=["Waiting on API keys", "Need QA signoff"],
            )
            path = Path(tmpdir) / "session-001.html"
            session_to_html(session, path)

            loaded = html_to_session(path)
            assert loaded.handoff_notes == "Wrap up payment refactor"
            assert loaded.recommended_next == "Add migration tests"
            assert loaded.blockers == ["Waiting on API keys", "Need QA signoff"]


class TestHandoffContext:
    """Test lightweight context generation for handoff."""

    def test_context_without_handoff(self, isolated_db):
        """Verify context doesn't mention handoff if not required."""
        node = Node(
            id="feature-001",
            title="Test Feature",
            status="todo",
        )
        context = node.to_context()

        assert "🔄 Handoff" not in context

    def test_context_with_handoff_shows_info(self, isolated_db):
        """Verify context includes handoff information."""
        node = Node(
            id="feature-001",
            title="Test Feature",
            status="todo",
            handoff_required=True,
            previous_agent="alice",
            handoff_reason="blocked on dependency",
        )
        context = node.to_context()

        assert "🔄 Handoff" in context
        assert "alice" in context
        assert "blocked on dependency" in context

    def test_context_includes_handoff_notes(self, isolated_db):
        """Verify context includes handoff notes when present."""
        node = Node(
            id="feature-001",
            title="Test Feature",
            handoff_required=True,
            previous_agent="alice",
            handoff_reason="blocked",
            handoff_notes="Waiting for DB migration",
        )
        context = node.to_context()

        assert "Notes:" in context
        assert "Waiting for DB migration" in context

    def test_context_token_efficiency(self, isolated_db):
        """Verify handoff context is lightweight (<200 tokens)."""
        node = Node(
            id="feature-001",
            title="Test Feature with Very Long Title That Should Still Be Lightweight",
            handoff_required=True,
            previous_agent="alice",
            handoff_reason="blocked on dependency",
            handoff_notes="Detailed notes about what was done and what needs to happen next",
        )
        context = node.to_context()

        # Rough estimate: ~4 chars per token, should be <200 tokens
        estimated_tokens = len(context) / 4
        assert estimated_tokens < 200, f"Context too large: {estimated_tokens} tokens"


class TestSessionManagerHandoff:
    """Test SessionManager.create_handoff() method."""

    def test_create_handoff_sets_metadata(self, isolated_db):
        """Verify create_handoff sets all metadata fields."""
        with TemporaryDirectory() as tmpdir:
            manager = SessionManager(tmpdir)

            # Create a feature first
            node = Node(
                id="feature-001",
                title="Test Feature",
                agent_assigned="alice",
                claimed_at=datetime.now(),
                claimed_by_session="session-001",
            )
            manager.features_graph.add(node)

            # Create handoff
            result = manager.create_handoff(
                feature_id="feature-001",
                reason="blocked on dependency",
                notes="Waiting for database migration",
                agent="alice",
                next_agent="bob",
            )

            assert result is not None
            assert result.handoff_required is True
            assert result.previous_agent == "alice"
            assert result.handoff_reason == "blocked on dependency"
            assert result.handoff_notes == "Waiting for database migration"
            assert result.handoff_timestamp is not None

    def test_create_handoff_releases_feature(self, isolated_db):
        """Verify create_handoff releases feature for next agent."""
        with TemporaryDirectory() as tmpdir:
            manager = SessionManager(tmpdir)

            # Create a claimed feature
            node = Node(
                id="feature-001",
                title="Test Feature",
                agent_assigned="alice",
                claimed_at=datetime.now(),
                claimed_by_session="session-001",
            )
            manager.features_graph.add(node)

            # Create handoff
            result = manager.create_handoff(
                feature_id="feature-001",
                reason="blocked on dependency",
                agent="alice",
            )

            # Feature should be released
            assert result.agent_assigned is None
            assert result.claimed_at is None
            assert result.claimed_by_session is None

    def test_create_handoff_rejects_if_not_owned(self, isolated_db):
        """Verify create_handoff rejects if agent doesn't own feature."""
        with TemporaryDirectory() as tmpdir:
            manager = SessionManager(tmpdir)

            # Create a feature claimed by alice
            node = Node(
                id="feature-001",
                title="Test Feature",
                agent_assigned="alice",
                claimed_at=datetime.now(),
                claimed_by_session="session-001",
            )
            manager.features_graph.add(node)

            # Try to handoff as bob (not owner)
            with pytest.raises(ValueError):
                manager.create_handoff(
                    feature_id="feature-001",
                    reason="blocked",
                    agent="bob",
                )

    def test_create_handoff_returns_none_if_not_found(self, isolated_db):
        """Verify create_handoff returns None if feature not found."""
        with TemporaryDirectory() as tmpdir:
            manager = SessionManager(tmpdir)

            result = manager.create_handoff(
                feature_id="feature-nonexistent",
                reason="blocked",
                agent="alice",
            )

            assert result is None

    def test_create_handoff_works_for_unclaimed_feature(self, isolated_db):
        """Verify create_handoff works on unclaimed features."""
        with TemporaryDirectory() as tmpdir:
            manager = SessionManager(tmpdir)

            # Create an unclaimed feature
            node = Node(
                id="feature-001",
                title="Test Feature",
            )
            manager.features_graph.add(node)

            # Should be able to handoff (mark as needing handoff)
            result = manager.create_handoff(
                feature_id="feature-001",
                reason="needs review",
                agent="alice",
            )

            assert result is not None
            assert result.handoff_required is True
            assert result.previous_agent == "alice"


class TestSessionHandoffManager:
    """Test session handoff updates via SessionManager and SDK."""

    def test_session_manager_set_handoff(self, isolated_db):
        """Verify SessionManager.set_session_handoff updates session fields."""
        with TemporaryDirectory() as tmpdir:
            manager = SessionManager(tmpdir)
            session = manager.start_session("session-001", agent="alice")

            updated = manager.set_session_handoff(
                session_id=session.id,
                handoff_notes="Investigate flaky tests",
                recommended_next="Run test suite with verbose logs",
                blockers=["CI is down"],
            )

            assert updated is not None
            assert updated.handoff_notes == "Investigate flaky tests"
            assert updated.recommended_next == "Run test suite with verbose logs"
            assert updated.blockers == ["CI is down"]

    def test_sdk_set_session_handoff(self, isolated_db):
        """Verify SDK.set_session_handoff proxies to SessionManager."""
        with TemporaryDirectory() as tmpdir:
            graph_dir = Path(tmpdir) / ".htmlgraph"
            graph_dir.mkdir(parents=True, exist_ok=True)
            manager = SessionManager(graph_dir)
            session = manager.start_session("session-002", agent="bob")

            sdk = SDK(directory=graph_dir, agent="bob", db_path=str(isolated_db))
            updated = sdk.set_session_handoff(
                session_id=session.id,
                handoff_notes="Refactor parser",
                recommended_next="Add tests for edge cases",
                blockers=["Waiting on spec"],
            )

            assert updated is not None
            assert updated.handoff_notes == "Refactor parser"
            assert updated.recommended_next == "Add tests for edge cases"
            assert updated.blockers == ["Waiting on spec"]


class TestSDKHandoff:
    """Test SDK FeatureBuilder handoff methods."""

    def test_feature_builder_has_complete_and_handoff(self, isolated_db):
        """Verify FeatureBuilder has complete_and_handoff method."""
        with TemporaryDirectory() as tmpdir:
            sdk = SDK(directory=tmpdir, agent="test-agent", db_path=str(isolated_db))

            builder = sdk.features.create("Test Feature")
            assert hasattr(builder, "complete_and_handoff")

    def test_complete_and_handoff_sets_fields(self, isolated_db):
        """Verify complete_and_handoff sets handoff fields."""
        with TemporaryDirectory() as tmpdir:
            sdk = SDK(directory=tmpdir, agent="test-agent", db_path=str(isolated_db))

            track = sdk.tracks.create("Test Track").save()
            feature = (
                sdk.features.create("Test Feature")
                .set_track(track.id)
                .complete_and_handoff(
                    reason="awaiting review",
                    notes="All tests passing",
                )
                .save()
            )

            assert feature.handoff_required is True
            assert feature.handoff_reason == "awaiting review"
            assert feature.handoff_notes == "All tests passing"
            assert feature.handoff_timestamp is not None

    def test_complete_and_handoff_with_priority(self, isolated_db):
        """Verify complete_and_handoff chains with other methods."""
        with TemporaryDirectory() as tmpdir:
            sdk = SDK(directory=tmpdir, agent="test-agent", db_path=str(isolated_db))

            track = sdk.tracks.create("Test Track").save()
            feature = (
                sdk.features.create("Test Feature")
                .set_track(track.id)
                .set_priority("high")
                .add_step("Step 1")
                .complete_and_handoff(
                    reason="ready for deployment",
                    notes="All tests passing, docs updated",
                )
                .save()
            )

            assert feature.priority == "high"
            assert len(feature.steps) == 1
            assert feature.handoff_required is True
            assert feature.handoff_reason == "ready for deployment"


class TestHandoffWorkflow:
    """Test end-to-end handoff workflow."""

    def test_agent_a_to_agent_b_handoff(self, isolated_db):
        """Test complete workflow: Agent A claims, hands off to Agent B."""
        with TemporaryDirectory() as tmpdir:
            manager = SessionManager(tmpdir)

            # Agent A creates and claims feature
            feature = Node(
                id="feature-001",
                title="Implement Authentication",
                agent_assigned="alice",
                claimed_at=datetime.now(),
                claimed_by_session="session-alice-001",
            )
            manager.features_graph.add(feature)

            # Agent A hands off to Agent B
            handed_off = manager.create_handoff(
                feature_id="feature-001",
                reason="requires cryptography expertise",
                notes="JWT setup done, need to implement refresh token rotation",
                agent="alice",
                next_agent="bob",
            )

            # Verify handoff state
            assert handed_off.handoff_required is True
            assert handed_off.previous_agent == "alice"
            assert handed_off.agent_assigned is None  # Released

            # Agent B can now claim it
            bob_claims = manager.claim_feature(
                feature_id="feature-001",
                agent="bob",
            )

            assert bob_claims.agent_assigned == "bob"
            assert bob_claims.handoff_required is True  # Still has handoff context
            assert bob_claims.previous_agent == "alice"  # Audit trail intact

    def test_handoff_preserves_history(self, isolated_db):
        """Test that handoff context is preserved in HTML."""
        feature = Node(
            id="feature-001",
            title="Test Feature",
            handoff_required=True,
            previous_agent="alice",
            handoff_reason="expertise needed",
            handoff_notes="Context preserved",
            handoff_timestamp=datetime.now(),
        )

        html = feature.to_html()

        # All handoff context should be in the HTML
        assert "alice" in html
        assert "expertise needed" in html
        assert "Context preserved" in html
        assert "data-handoff" in html

    def test_handoff_efficiency_metrics(self, isolated_db):
        """Test that handoff is efficient (timing and size)."""
        import time

        feature = Node(
            id="feature-001",
            title="Test Feature",
            handoff_required=True,
            previous_agent="alice",
            handoff_reason="blocked on dependency",
            handoff_notes="Detailed notes about what was done",
        )

        # Measure HTML generation time
        start = time.time()
        for _ in range(100):
            html = feature.to_html()
        elapsed = time.time() - start

        # Should generate HTML in <50ms per feature
        assert elapsed < 5.0, (
            f"HTML generation too slow: {elapsed / 100 * 1000:.2f}ms per feature"
        )

        # Verify HTML size is reasonable
        assert len(html) < 10_000, f"HTML too large: {len(html)} bytes"

        # Verify context is lightweight
        context = feature.to_context()
        estimated_tokens = len(context) / 4
        assert estimated_tokens < 200, f"Context too large: {estimated_tokens} tokens"
