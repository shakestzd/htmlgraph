"""
Unit tests for auto-spike creation functionality.

Tests:
1. Session-init spike creation
2. Transition spike creation
3. Auto-spike completion
4. SessionConverter handling of auto-spike fields
"""

from datetime import datetime

import pytest
from htmlgraph.converter import NodeConverter, SessionConverter
from htmlgraph.models import Node
from htmlgraph.session_manager import SessionManager


class TestSessionInitSpike:
    """Test session-init spike auto-creation."""

    def test_session_init_spike_created_on_start(self, tmp_path):
        """Test that session-init spike is auto-created when session starts."""
        graph_dir = tmp_path / ".htmlgraph"
        graph_dir.mkdir()

        manager = SessionManager(graph_dir)
        session = manager.start_session(agent="test-agent", title="Test Session")

        # Check that session-init spike was created
        spike_converter = NodeConverter(graph_dir / "spikes")
        spikes = spike_converter.load_all()

        # Should have one session-init spike
        session_init_spikes = [
            s for s in spikes if s.spike_subtype == "session-init" and s.auto_generated
        ]
        assert len(session_init_spikes) == 1

        spike = session_init_spikes[0]
        assert spike.type == "spike"
        assert spike.spike_subtype == "session-init"
        assert spike.auto_generated is True
        assert spike.status == "in-progress"
        assert spike.session_id == session.id
        assert "Session Init:" in spike.title

    def test_session_init_spike_idempotent(self, tmp_path):
        """Test that restarting session doesn't create duplicate spikes."""
        graph_dir = tmp_path / ".htmlgraph"
        graph_dir.mkdir()

        manager = SessionManager(graph_dir)

        # Start session twice (simulating session restart)
        manager.start_session(
            session_id="test-session", agent="test-agent", title="Test Session"
        )
        manager.start_session(
            session_id="test-session", agent="test-agent", title="Test Session"
        )

        # Should only have one session-init spike
        spike_converter = NodeConverter(graph_dir / "spikes")
        spikes = spike_converter.load_all()
        session_init_spikes = [s for s in spikes if s.spike_subtype == "session-init"]

        assert len(session_init_spikes) == 1

    def test_session_init_spike_linked_to_session(self, tmp_path):
        """Test that session-init spike is linked to session."""
        graph_dir = tmp_path / ".htmlgraph"
        graph_dir.mkdir()

        manager = SessionManager(graph_dir)
        session = manager.start_session(agent="test-agent", title="Test Session")

        # Reload session to check worked_on
        converter = SessionConverter(graph_dir / "sessions")
        reloaded_session = converter.load(session.id)

        # Session should have spike in worked_on
        spike_converter = NodeConverter(graph_dir / "spikes")
        spikes = spike_converter.load_all()
        session_init_spikes = [s for s in spikes if s.spike_subtype == "session-init"]

        assert len(session_init_spikes) == 1
        spike = session_init_spikes[0]
        assert spike.id in reloaded_session.worked_on


class TestTransitionSpike:
    """Test transition spike auto-creation (disabled — see bug-63423134)."""

    def test_transition_spike_not_created_on_feature_complete(self, tmp_path):
        """Test that transition spike is NOT created when feature completes.

        Transition spike auto-creation was disabled in bug-63423134 because
        it polluted the dashboard with meaningless work items. CIGS guidance
        handles prompting the agent to pick the next work item instead.
        """
        graph_dir = tmp_path / ".htmlgraph"
        graph_dir.mkdir()

        manager = SessionManager(graph_dir)

        # Create and start a session
        manager.start_session(agent="test-agent", title="Test Session")

        # Create and complete a feature
        feature = manager.create_feature("Test Feature", agent="test-agent")
        manager.start_feature(feature.id, agent="test-agent")
        manager.complete_feature(feature.id, agent="test-agent")

        # Check that NO transition spike was created
        spike_converter = NodeConverter(graph_dir / "spikes")
        spikes = spike_converter.load_all()

        transition_spikes = [
            s for s in spikes if s.spike_subtype == "transition" and s.auto_generated
        ]
        assert len(transition_spikes) == 0

    def test_no_transition_spike_linked_to_session(self, tmp_path):
        """Test that completing a feature does not add a transition spike to the session."""
        graph_dir = tmp_path / ".htmlgraph"
        graph_dir.mkdir()

        manager = SessionManager(graph_dir)

        session = manager.start_session(agent="test-agent", title="Test Session")
        feature = manager.create_feature("Test Feature", agent="test-agent")
        manager.start_feature(feature.id, agent="test-agent")
        manager.complete_feature(feature.id, agent="test-agent")

        # Reload session
        converter = SessionConverter(graph_dir / "sessions")
        reloaded_session = converter.load(session.id)

        # No transition spike should exist
        spike_converter = NodeConverter(graph_dir / "spikes")
        spikes = spike_converter.load_all()
        transition_spikes = [s for s in spikes if s.spike_subtype == "transition"]

        assert len(transition_spikes) == 0
        # session.worked_on should only contain session-init spike (if any)
        for item_id in reloaded_session.worked_on:
            assert "transition" not in item_id


class TestAutoSpikeCompletion:
    """Test auto-spike completion when features start."""

    def test_session_init_spike_completes_on_feature_start(self, tmp_path):
        """Test that session-init spike completes when first feature starts."""
        graph_dir = tmp_path / ".htmlgraph"
        graph_dir.mkdir()

        manager = SessionManager(graph_dir)

        # Start session (creates session-init spike)
        manager.start_session(agent="test-agent", title="Test Session")

        # Verify session-init spike is in-progress
        spike_converter = NodeConverter(graph_dir / "spikes")
        spikes = spike_converter.load_all()
        init_spike = [s for s in spikes if s.spike_subtype == "session-init"][0]
        assert init_spike.status == "in-progress"

        # Start a feature (should complete session-init spike)
        feature = manager.create_feature("Test Feature", agent="test-agent")
        manager.start_feature(feature.id, agent="test-agent")

        # Reload and verify spike is completed
        spikes = spike_converter.load_all()
        init_spike = [s for s in spikes if s.spike_subtype == "session-init"][0]
        assert init_spike.status == "done"
        assert init_spike.to_feature_id == feature.id

    def test_no_transition_spike_between_features(self, tmp_path):
        """Test that no transition spike is created between features (disabled, bug-63423134)."""
        graph_dir = tmp_path / ".htmlgraph"
        graph_dir.mkdir()

        manager = SessionManager(graph_dir)

        manager.start_session(agent="test-agent", title="Test Session")

        # Complete first feature — should NOT create transition spike
        feature1 = manager.create_feature("Feature 1", agent="test-agent")
        manager.start_feature(feature1.id, agent="test-agent")
        manager.complete_feature(feature1.id, agent="test-agent")

        # Verify NO transition spike was created
        spike_converter = NodeConverter(graph_dir / "spikes")
        spikes = spike_converter.load_all()
        transition_spikes = [s for s in spikes if s.spike_subtype == "transition"]
        assert len(transition_spikes) == 0

        # Start second feature — still no transition spike
        feature2 = manager.create_feature("Feature 2", agent="test-agent")
        manager.start_feature(feature2.id, agent="test-agent")

        spikes = spike_converter.load_all()
        transition_spikes = [s for s in spikes if s.spike_subtype == "transition"]
        assert len(transition_spikes) == 0

    def test_session_init_spike_completes_without_transition_spike(self, tmp_path):
        """Test that session-init completes when feature starts, with no transition spike after."""
        graph_dir = tmp_path / ".htmlgraph"
        graph_dir.mkdir()

        manager = SessionManager(graph_dir)

        # Start session (creates session-init spike)
        manager.start_session(agent="test-agent", title="Test Session")

        # Verify session-init spike exists
        spike_converter = NodeConverter(graph_dir / "spikes")
        spikes = spike_converter.load_all()
        assert len([s for s in spikes if s.spike_subtype == "session-init"]) == 1

        # Start first feature (should complete session-init)
        feature1 = manager.create_feature("Feature 1", agent="test-agent")
        manager.start_feature(feature1.id, agent="test-agent")

        # Complete first feature — no transition spike created
        manager.complete_feature(feature1.id, agent="test-agent")

        # Verify we have session-init (done) and NO transition spike
        spikes = spike_converter.load_all()
        init_spikes = [s for s in spikes if s.spike_subtype == "session-init"]
        transition_spikes = [s for s in spikes if s.spike_subtype == "transition"]
        assert len(init_spikes) == 1
        assert init_spikes[0].status == "done"
        assert len(transition_spikes) == 0


class TestSessionConverterAutoSpikeFields:
    """Test that SessionConverter handles auto-spike fields correctly."""

    def test_converter_parses_transcript_fields(self, tmp_path):
        """Test that SessionConverter parses transcript fields from HTML."""
        graph_dir = tmp_path / ".htmlgraph"
        sessions_dir = graph_dir / "sessions"
        sessions_dir.mkdir(parents=True)

        manager = SessionManager(graph_dir)
        session = manager.start_session(agent="test-agent", title="Test Session")

        # Add transcript metadata
        session.transcript_id = "test-transcript-uuid-1234"
        session.transcript_path = "/path/to/transcript.jsonl"
        session.transcript_synced_at = datetime.now()
        session.transcript_git_branch = "main"

        # Save to HTML
        converter = SessionConverter(sessions_dir)
        converter.save(session)

        # Reload from HTML
        reloaded = converter.load(session.id)

        # Verify transcript fields are preserved
        assert reloaded.transcript_id == "test-transcript-uuid-1234"
        assert reloaded.transcript_path == "/path/to/transcript.jsonl"
        assert reloaded.transcript_synced_at is not None
        assert reloaded.transcript_git_branch == "main"

    def test_node_converter_parses_auto_spike_fields(self, tmp_path):
        """Test that NodeConverter parses auto_generated and spike_subtype."""
        graph_dir = tmp_path / ".htmlgraph"
        spikes_dir = graph_dir / "spikes"
        spikes_dir.mkdir(parents=True)

        # Create auto-generated spike manually
        spike = Node(
            id="spike-test-123",
            title="Test Auto Spike",
            type="spike",
            spike_subtype="session-init",
            auto_generated=True,
            status="in-progress",
            session_id="sess-123",
        )

        # Save to HTML
        converter = NodeConverter(spikes_dir)
        converter.save(spike)

        # Reload from HTML
        reloaded = converter.load(spike.id)

        # Verify auto-spike fields are preserved
        assert reloaded.type == "spike"
        assert reloaded.spike_subtype == "session-init"
        assert reloaded.auto_generated is True
        assert reloaded.session_id == "sess-123"

    def test_html_roundtrip_preserves_all_fields(self, tmp_path):
        """Test that HTML roundtrip preserves all auto-spike fields."""
        graph_dir = tmp_path / ".htmlgraph"
        spikes_dir = graph_dir / "spikes"
        spikes_dir.mkdir(parents=True)

        # Create spike with all fields
        original = Node(
            id="spike-roundtrip-test",
            title="Roundtrip Test Spike",
            type="spike",
            spike_subtype="transition",
            auto_generated=True,
            status="done",
            session_id="sess-456",
            from_feature_id="feat-abc",
            to_feature_id="feat-def",
            model_name="test-agent",
        )

        # Save and reload
        converter = NodeConverter(spikes_dir)
        converter.save(original)
        reloaded = converter.load(original.id)

        # Verify all fields preserved
        assert reloaded.id == original.id
        assert reloaded.title == original.title
        assert reloaded.type == original.type
        assert reloaded.spike_subtype == original.spike_subtype
        assert reloaded.auto_generated == original.auto_generated
        assert reloaded.status == original.status
        assert reloaded.session_id == original.session_id
        assert reloaded.from_feature_id == original.from_feature_id
        assert reloaded.to_feature_id == original.to_feature_id
        assert reloaded.model_name == original.model_name


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
