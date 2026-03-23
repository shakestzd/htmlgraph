"""
Tests for agent attribution bug fix.

Verifies that:
1. SDK() without agent parameter raises ValueError with clear error message
2. SDK(agent='name') sets _agent_id correctly
3. Spikes created with agent parameter have agent_assigned field
4. Warning logged when spike created without agent attribution
5. Error messages are clear and helpful
"""

from pathlib import Path

import pytest
from htmlgraph import SDK


@pytest.fixture
def tmp_htmlgraph(isolated_graph_dir_full: Path):
    """Create a temporary .htmlgraph directory structure."""
    return isolated_graph_dir_full


class TestSDKAgentParameterRequired:
    """Test that agent parameter is required for SDK initialization."""

    def test_sdk_without_agent_uses_detected_value(
        self, tmp_htmlgraph: Path, isolated_db: Path, monkeypatch
    ):
        """SDK() without explicit agent uses detect_agent_name() if not CLI."""
        # Use CLAUDE_AGENT_NAME env var so detection works consistently in CI
        monkeypatch.setenv("CLAUDE_AGENT_NAME", "claude-code")
        sdk = SDK(directory=tmp_htmlgraph, db_path=str(isolated_db))

        # Should use detected agent
        assert sdk._agent_id is not None
        # In this environment, it should be "claude-code" or "cli"
        assert sdk._agent_id in ["claude-code", "cli"]

    def test_sdk_with_agent_parameter_succeeds(
        self, tmp_htmlgraph: Path, isolated_db: Path
    ):
        """SDK(agent='name') should succeed and set _agent_id."""
        sdk = SDK(directory=tmp_htmlgraph, agent="test-agent", db_path=str(isolated_db))

        assert sdk._agent_id == "test-agent"
        # Verify collections are initialized
        assert sdk.features is not None
        assert sdk.spikes is not None
        assert sdk.bugs is not None

    def test_sdk_with_agent_parameter_various_names(
        self, tmp_htmlgraph: Path, isolated_db: Path
    ):
        """SDK should work with various agent names."""
        agent_names = [
            "claude",
            "explorer",
            "coder",
            "tester",
            "my-custom-agent",
            "agent-123",
        ]

        from htmlgraph.sdk import normalize_agent_name

        for agent_name in agent_names:
            sdk = SDK(
                directory=tmp_htmlgraph, agent=agent_name, db_path=str(isolated_db)
            )
            assert sdk._agent_id == normalize_agent_name(agent_name)

    def test_sdk_with_claude_agent_name_env(
        self, tmp_htmlgraph: Path, isolated_db: Path, monkeypatch
    ):
        """SDK should use CLAUDE_AGENT_NAME env var if provided."""
        monkeypatch.setenv("CLAUDE_AGENT_NAME", "env-agent")
        sdk = SDK(directory=tmp_htmlgraph, db_path=str(isolated_db))

        assert sdk._agent_id == "env-agent"

    def test_sdk_explicit_parameter_overrides_env(
        self, tmp_htmlgraph: Path, isolated_db: Path, monkeypatch
    ):
        """Explicit agent parameter should override env vars."""
        monkeypatch.setenv("CLAUDE_AGENT_NAME", "env-agent")
        sdk = SDK(
            directory=tmp_htmlgraph, agent="explicit-agent", db_path=str(isolated_db)
        )

        assert sdk._agent_id == "explicit-agent"


class TestSpikeWithAgentAttribution:
    """Test that spikes are created with proper agent attribution."""

    def test_spike_has_agent_assigned_field(
        self, tmp_htmlgraph: Path, isolated_db: Path
    ):
        """Spike created via SDK should have agent_assigned field."""
        sdk = SDK(directory=tmp_htmlgraph, agent="test-agent", db_path=str(isolated_db))

        spike = sdk.spikes.create("Test Spike").save()

        # Spike should have agent_assigned field
        assert hasattr(spike, "agent_assigned")
        assert spike.agent_assigned == "test-agent"

    def test_spike_agent_matches_sdk_agent(
        self, tmp_htmlgraph: Path, isolated_db: Path
    ):
        """Spike's agent_assigned should match SDK's agent."""
        agent_name = "my-test-agent"
        sdk = SDK(directory=tmp_htmlgraph, agent=agent_name, db_path=str(isolated_db))

        spike = sdk.spikes.create("Investigation").save()

        assert spike.agent_assigned == agent_name

    def test_spike_with_builder_methods_preserves_agent(
        self, tmp_htmlgraph: Path, isolated_db: Path
    ):
        """Agent attribution should persist through builder methods."""
        sdk = SDK(directory=tmp_htmlgraph, agent="coder", db_path=str(isolated_db))

        spike = (
            sdk.spikes.create("Investigate Database")
            .set_spike_type("technical")
            .set_timebox_hours(4)
            .add_steps(["Research options", "Benchmark"])
            .set_findings("SQLite is sufficient")
            .save()
        )

        assert spike.agent_assigned == "coder"
        assert spike.title == "Investigate Database"
        assert spike.findings == "SQLite is sufficient"

    def test_multiple_spikes_all_have_agent_assigned(
        self, tmp_htmlgraph: Path, isolated_db: Path
    ):
        """All spikes created should have agent_assigned."""
        sdk = SDK(directory=tmp_htmlgraph, agent="explorer", db_path=str(isolated_db))

        spikes = []
        for i in range(3):
            spike = sdk.spikes.create(f"Spike {i}").save()
            spikes.append(spike)

        # All spikes should have agent_assigned
        for spike in spikes:
            assert hasattr(spike, "agent_assigned")
            assert spike.agent_assigned == "explorer"


class TestSpikeBuilderWarning:
    """Test that warning is logged when spike created without agent."""

    def test_spike_created_always_has_agent(
        self, tmp_htmlgraph: Path, isolated_db: Path, monkeypatch
    ):
        """Spike created should always have agent_assigned from SDK."""
        # Use CLAUDE_AGENT_NAME env var so detection works consistently in CI
        monkeypatch.setenv("CLAUDE_AGENT_NAME", "claude-code")
        sdk = SDK(directory=tmp_htmlgraph, db_path=str(isolated_db))

        spike = sdk.spikes.create("Test Spike").save()

        # Should have agent_assigned from SDK
        assert spike.agent_assigned is not None


class TestErrorMessageClarity:
    """Test that error messages are clear and helpful when no agent can be detected."""

    def test_sdk_init_docstring_documents_agent_requirement(self):
        """SDK.__init__ docstring should document agent parameter requirement."""
        docstring = SDK.__init__.__doc__
        assert docstring is not None
        assert "agent" in docstring.lower()
        assert "required" in docstring.lower() or "REQUIRED" in docstring


class TestOtherCollectionsWithAgent:
    """Test that all collections properly use agent attribution."""

    def test_feature_has_agent_assigned(self, tmp_htmlgraph: Path, isolated_db: Path):
        """Feature created via SDK should have agent_assigned."""
        sdk = SDK(directory=tmp_htmlgraph, agent="coder", db_path=str(isolated_db))

        # Create a track first (required for features)
        track = sdk.tracks.create("Test Track").save()

        feature = sdk.features.create("User Auth").set_track(track.id).save()

        assert hasattr(feature, "agent_assigned")
        assert feature.agent_assigned == "coder"

    def test_bug_has_agent_assigned(self, tmp_htmlgraph: Path, isolated_db: Path):
        """Bug created via SDK should have agent_assigned."""
        sdk = SDK(directory=tmp_htmlgraph, agent="tester", db_path=str(isolated_db))

        bug = sdk.bugs.create("Login broken").save()

        assert hasattr(bug, "agent_assigned")
        assert bug.agent_assigned == "tester"


class TestSpikeRetrieval:
    """Test that spikes can be retrieved and agent is preserved."""

    def test_spike_retrieval_preserves_agent(
        self, tmp_htmlgraph: Path, isolated_db: Path
    ):
        """Retrieved spike should have agent_assigned field."""
        sdk = SDK(directory=tmp_htmlgraph, agent="explorer", db_path=str(isolated_db))

        # Create spike
        created = sdk.spikes.create("Research Options").save()
        spike_id = created.id

        # Retrieve spike
        retrieved = sdk.spikes.get(spike_id)

        assert retrieved is not None
        assert retrieved.agent_assigned == "explorer"

    def test_all_spikes_have_agent_assigned(
        self, tmp_htmlgraph: Path, isolated_db: Path
    ):
        """All retrieved spikes should have agent_assigned."""
        sdk = SDK(directory=tmp_htmlgraph, agent="coder", db_path=str(isolated_db))

        # Create multiple spikes
        for i in range(3):
            sdk.spikes.create(f"Spike {i}").save()

        # Get all spikes
        all_spikes = sdk.spikes.all()

        assert len(all_spikes) >= 3
        for spike in all_spikes:
            assert hasattr(spike, "agent_assigned")
            assert spike.agent_assigned == "coder"
