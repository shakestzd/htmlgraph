"""
Integration tests for post-compact delegation verification.

Verifies that delegation enforcement persists across the complete session lifecycle:
startup → interact → compact → resume.

Test Scenarios:
1. Environment variables persist across compact
2. SessionState auto-detects post-compact
3. Agent attribution survives post-compact
4. Full delegation workflow post-compact
5. Subagent session linking
6. SDK mandatory agent parameter
7. All builders enforce agent parameter
8. Post-compact skill activation

These tests ensure that delegation constraints and session tracking remain
consistent across session boundaries.

NOTE: Skipped - requires Claude Code session infrastructure and environment variables.
"""

import json
import os
from uuid import uuid4

import pytest

# Skip entire module - tests require Claude Code session infrastructure
pytestmark = pytest.mark.skip(
    reason="Post-compact delegation tests require Claude Code session infrastructure"
)
from htmlgraph.sdk import SDK
from htmlgraph.session_state import SessionStateManager


@pytest.fixture
def temp_htmlgraph_dir(isolated_graph_dir_full):
    """Create a temporary .htmlgraph directory structure."""
    return isolated_graph_dir_full


@pytest.fixture
def session_manager(temp_htmlgraph_dir):
    """Create a SessionStateManager for testing."""
    return SessionStateManager(temp_htmlgraph_dir)


@pytest.fixture
def monkeypatch_cwd(monkeypatch, tmp_path):
    """Set current working directory to temp path."""
    monkeypatch.chdir(tmp_path)
    return monkeypatch


class TestEnvironmentVariablesPeristPostCompact:
    """Test that environment variables persist across compact cycles."""

    def test_env_vars_persist_across_compact(
        self, temp_htmlgraph_dir, session_manager, monkeypatch
    ):
        """
        Verify environment variables are set and persist across compact.

        Test flow:
        1. Initialize session with orchestrator
        2. Verify environment variables are correctly set
        3. Simulate compact (change session ID)
        4. Verify post-compact environment variables persist
        """
        # First session setup
        session_id_1 = f"sess-{uuid4().hex[:8]}"
        monkeypatch.setenv("CLAUDE_SESSION_ID", session_id_1)

        # Get first session state
        state1 = session_manager.get_current_state()
        env_vars1 = session_manager.setup_environment_variables(state1)

        # Verify initial environment variables are set
        assert os.environ.get("CLAUDE_SESSION_ID") == session_id_1
        assert os.environ.get("CLAUDE_SESSION_COMPACTED") == "false"
        assert state1["delegation_enabled"] is not None

        # Record first session
        session_manager.record_state(
            session_id=session_id_1,
            source="startup",
            is_post_compact=False,
            delegation_enabled=True,
            environment_vars=env_vars1,
        )

        # Mark first session as ended
        state_file = (
            temp_htmlgraph_dir / "sessions" / session_manager.SESSION_STATE_FILE
        )
        state_data = json.loads(state_file.read_text())
        state_data["is_ended"] = True
        state_file.write_text(json.dumps(state_data))

        # Simulate compact - new session ID
        session_id_2 = f"sess-{uuid4().hex[:8]}"
        monkeypatch.setenv("CLAUDE_SESSION_ID", session_id_2)

        # Get post-compact session state
        state2 = session_manager.get_current_state()
        env_vars2 = session_manager.setup_environment_variables(state2)

        # Verify post-compact environment variables
        assert state2["is_post_compact"] is True
        assert state2["previous_session_id"] == session_id_1
        assert os.environ.get("CLAUDE_SESSION_ID") == session_id_2
        assert os.environ.get("CLAUDE_SESSION_COMPACTED") == "true"
        assert "CLAUDE_PREVIOUS_SESSION_ID" in env_vars2 or state2["delegation_enabled"]


class TestSessionStateAutoDetection:
    """Test SessionState auto-detection of post-compact."""

    def test_session_state_auto_detects_compact(
        self, temp_htmlgraph_dir, session_manager, monkeypatch
    ):
        """
        Verify SessionState automatically detects post-compact.

        Test flow:
        1. Initialize first session
        2. Get initial state (is_post_compact=false)
        3. Simulate compact cycle
        4. Initialize second session
        5. Verify post-compact detection with correct metadata
        """
        # First session
        session_id_1 = f"sess-{uuid4().hex[:8]}"
        monkeypatch.setenv("CLAUDE_SESSION_ID", session_id_1)

        state1 = session_manager.get_current_state()
        assert state1["session_source"] == "startup"
        assert state1["is_post_compact"] is False
        assert state1["previous_session_id"] is None

        # Record and end first session
        session_manager.record_state(
            session_id=session_id_1,
            source="startup",
            is_post_compact=False,
            delegation_enabled=False,
        )

        state_file = (
            temp_htmlgraph_dir / "sessions" / session_manager.SESSION_STATE_FILE
        )
        state_data = json.loads(state_file.read_text())
        state_data["is_ended"] = True
        state_file.write_text(json.dumps(state_data))

        # Second session (post-compact)
        session_id_2 = f"sess-{uuid4().hex[:8]}"
        monkeypatch.setenv("CLAUDE_SESSION_ID", session_id_2)

        state2 = session_manager.get_current_state()

        # Verify post-compact auto-detection
        assert state2["session_source"] == "compact"
        assert state2["is_post_compact"] is True
        assert state2["previous_session_id"] == session_id_1
        assert state2["delegation_enabled"] is True
        assert state2["session_id"] == session_id_2


class TestAgentAttributionPostCompact:
    """Test agent attribution persistence across compact."""

    def test_agent_attribution_required_post_compact(
        self, temp_htmlgraph_dir, isolated_db, monkeypatch_cwd, monkeypatch
    ):
        """
        Verify agent parameter is required both pre and post-compact.

        Test flow:
        1. First session: Create spike with agent='explorer'
        2. Verify spike has agent_assigned
        3. Simulate compact (new session ID)
        4. Second session: Create spike with agent='coder'
        5. Verify both spikes have agent_assigned with their respective agents
        """
        # First session - create SDK with agent
        session_id_1 = f"sess-explorer-{uuid4().hex[:8]}"
        monkeypatch.setenv("CLAUDE_SESSION_ID", session_id_1)

        sdk1 = SDK(
            directory=temp_htmlgraph_dir, agent="explorer", db_path=str(isolated_db)
        )
        spike1 = sdk1.spikes.create("Research API endpoints").save()
        assert spike1 is not None
        assert spike1.agent_assigned == "explorer"

        # Second session - new SDK with different agent (simulating compact)
        session_id_2 = f"sess-coder-{uuid4().hex[:8]}"
        monkeypatch.setenv("CLAUDE_SESSION_ID", session_id_2)

        sdk2 = SDK(
            directory=temp_htmlgraph_dir, agent="coder", db_path=str(isolated_db)
        )
        spike2 = sdk2.spikes.create("Implement API endpoints").save()
        assert spike2 is not None
        assert spike2.agent_assigned == "coder"

        # Verify both spikes have correct agent attribution
        spike1_retrieved = sdk1.spikes.get(spike1.id)
        assert spike1_retrieved is not None
        assert spike1_retrieved.agent_assigned == "explorer"


class TestFullDelegationWorkflowPostCompact:
    """Test complete delegation workflow across compact."""

    def test_delegation_workflow_post_compact_enforced(
        self, temp_htmlgraph_dir, isolated_db, session_manager, monkeypatch
    ):
        """
        Verify delegation workflow remains enforced post-compact.

        Test flow:
        1. First session: SDK with agent='orchestrator'
        2. Create work item
        3. Simulate compact
        4. Second session: Verify delegation still enforced
        5. Create work item with agent parameter
        """
        # First session
        session_id_1 = f"sess-{uuid4().hex[:8]}"
        monkeypatch.setenv("CLAUDE_SESSION_ID", session_id_1)
        monkeypatch.setenv("CLAUDE_ORCHESTRATOR_ACTIVE", "true")

        # Initialize SDK - should require agent
        sdk1 = SDK(
            directory=temp_htmlgraph_dir, agent="orchestrator", db_path=str(isolated_db)
        )
        assert sdk1.agent == "orchestrator"

        # Create track first
        track1 = sdk1.tracks.create("Test Track").save()

        # Create work item
        feature1 = sdk1.features.create("Feature 1").set_track(track1.id).save()
        assert feature1.agent_assigned == "orchestrator"

        # Record session and end it
        session_manager.record_state(
            session_id=session_id_1,
            source="startup",
            is_post_compact=False,
            delegation_enabled=True,
        )

        state_file = (
            temp_htmlgraph_dir / "sessions" / session_manager.SESSION_STATE_FILE
        )
        state_data = json.loads(state_file.read_text())
        state_data["is_ended"] = True
        state_file.write_text(json.dumps(state_data))

        # Simulate compact
        session_id_2 = f"sess-{uuid4().hex[:8]}"
        monkeypatch.setenv("CLAUDE_SESSION_ID", session_id_2)

        # Second session - SDK should still require agent
        sdk2 = SDK(
            directory=temp_htmlgraph_dir, agent="orchestrator", db_path=str(isolated_db)
        )

        # Verify delegation is still enforced
        state2 = session_manager.get_current_state()
        assert state2["is_post_compact"] is True
        assert state2["delegation_enabled"] is True

        # Create track first
        track2 = sdk2.tracks.create("Test Track 2").save()

        # Create another work item
        feature2 = sdk2.features.create("Feature 2").set_track(track2.id).save()
        assert feature2.agent_assigned == "orchestrator"


class TestSubagentSessionLinking:
    """Test parent-child session linking."""

    def test_subagent_sessions_linked_to_parent(
        self, temp_htmlgraph_dir, isolated_db, monkeypatch
    ):
        """
        Verify subagent sessions are properly linked to parent.

        Test flow:
        1. Parent session creates work item
        2. Subagent receives parent session ID in environment
        3. Subagent creates spike linked to parent
        4. Verify parent_session_id is recorded
        """
        # Parent session
        parent_session_id = f"sess-parent-{uuid4().hex[:8]}"
        monkeypatch.setenv("CLAUDE_SESSION_ID", parent_session_id)

        sdk_parent = SDK(
            directory=temp_htmlgraph_dir, agent="orchestrator", db_path=str(isolated_db)
        )
        track = sdk_parent.tracks.create("Parent Track").save()
        feature = (
            sdk_parent.features.create("Parent Feature").set_track(track.id).save()
        )
        assert feature is not None

        # Simulate subagent environment setup
        monkeypatch.setenv("CLAUDE_PARENT_SESSION_ID", parent_session_id)

        subagent_session_id = f"sess-sub-{uuid4().hex[:8]}"
        monkeypatch.setenv("CLAUDE_SESSION_ID", subagent_session_id)

        # Subagent SDK with parent session context
        sdk_subagent = SDK(
            directory=temp_htmlgraph_dir,
            agent="coder",
            parent_session=parent_session_id,
            db_path=str(isolated_db),
        )

        # Create spike in subagent
        spike = sdk_subagent.spikes.create("Subagent Investigation").save()
        assert spike is not None
        assert spike.agent_assigned == "coder"

        # Verify parent session ID is accessible
        assert os.environ.get("CLAUDE_PARENT_SESSION_ID") == parent_session_id


class TestSDKMandatoryAgentParameter:
    """Test SDK requires agent parameter."""

    def test_sdk_requires_agent_parameter_post_compact(
        self, temp_htmlgraph_dir, isolated_db, session_manager, monkeypatch
    ):
        """
        Verify SDK explicit agent parameter is preferred.

        Test flow:
        1. First session: SDK(agent='explicit-agent-1') → Works
        2. Verify agent is the explicit parameter
        3. Simulate compact
        4. Second session: SDK(agent='explicit-agent-2') → Works
        5. Verify agent is the explicit parameter
        """
        session_id_1 = f"sess-{uuid4().hex[:8]}"
        monkeypatch.setenv("CLAUDE_SESSION_ID", session_id_1)

        # First session - with explicit agent parameter
        sdk1 = SDK(
            directory=temp_htmlgraph_dir,
            agent="explicit-agent-1",
            db_path=str(isolated_db),
        )
        assert sdk1.agent == "explicit-agent-1"

        # Create track first
        track1 = sdk1.tracks.create("Test Track").save()

        # Create work item to verify agent attribution
        feature1 = sdk1.features.create("Feature 1").set_track(track1.id).save()
        assert feature1.agent_assigned == "explicit-agent-1"

        # Record and end first session
        session_manager.record_state(
            session_id=session_id_1,
            source="startup",
            is_post_compact=False,
            delegation_enabled=True,
        )

        state_file = (
            temp_htmlgraph_dir / "sessions" / session_manager.SESSION_STATE_FILE
        )
        state_data = json.loads(state_file.read_text())
        state_data["is_ended"] = True
        state_file.write_text(json.dumps(state_data))

        # Simulate compact
        session_id_2 = f"sess-{uuid4().hex[:8]}"
        monkeypatch.setenv("CLAUDE_SESSION_ID", session_id_2)

        # Second session - with different explicit agent
        sdk2 = SDK(
            directory=temp_htmlgraph_dir,
            agent="explicit-agent-2",
            db_path=str(isolated_db),
        )
        assert sdk2.agent == "explicit-agent-2"

        # Create track first
        track2 = sdk2.tracks.create("Test Track 2").save()

        # Create work item to verify agent attribution in post-compact session
        feature2 = sdk2.features.create("Feature 2").set_track(track2.id).save()
        assert feature2.agent_assigned == "explicit-agent-2"


class TestAllBuilderEnforceAgent:
    """Test that all builders enforce agent parameter."""

    def test_all_builders_enforce_agent_parameter(
        self, temp_htmlgraph_dir, isolated_db, monkeypatch
    ):
        """
        Verify ALL builders assign agent from SDK.

        Test flow:
        1. SDK with agent works
        2. All collections accessible with agent
        3. All work items have agent_assigned from SDK
        """
        session_id = f"sess-{uuid4().hex[:8]}"
        monkeypatch.setenv("CLAUDE_SESSION_ID", session_id)

        # SDK with agent should work
        sdk = SDK(
            directory=temp_htmlgraph_dir, agent="test-agent", db_path=str(isolated_db)
        )

        # Create track first (required for features, bugs, chores, epics, phases)
        track = sdk.tracks.create("Test Track").save()

        # Verify core collections are accessible
        assert sdk.features is not None
        assert sdk.bugs is not None
        assert sdk.spikes is not None

        # Verify core collections have builder support and assign agent
        feature = sdk.features.create("Test Feature").set_track(track.id).save()
        assert feature.agent_assigned == "test-agent"

        bug = sdk.bugs.create("Test Bug").set_track(track.id).save()
        assert bug.agent_assigned == "test-agent"

        spike = sdk.spikes.create("Test Spike").save()
        assert spike.agent_assigned == "test-agent"


class TestPostCompactSkillActivation:
    """Test Orchestrator Skill activation post-compact."""

    def test_orchestrator_skill_activates_post_compact(
        self, temp_htmlgraph_dir, isolated_db, session_manager, monkeypatch
    ):
        """
        Verify post-compact session detection works across compact.

        Test flow:
        1. Initialize first session
        2. Verify delegation state
        3. Simulate compact
        4. Verify post-compact is detected
        5. Verify delegation still available
        """
        session_id_1 = f"sess-{uuid4().hex[:8]}"
        monkeypatch.setenv("CLAUDE_SESSION_ID", session_id_1)

        # Get first session state
        state1 = session_manager.get_current_state()
        env_vars1 = session_manager.setup_environment_variables(state1)

        # Create SDK with agent
        sdk1 = SDK(
            directory=temp_htmlgraph_dir, agent="orchestrator", db_path=str(isolated_db)
        )
        assert sdk1.agent == "orchestrator"

        # Record and end first session
        session_manager.record_state(
            session_id=session_id_1,
            source="startup",
            is_post_compact=False,
            delegation_enabled=True,
            environment_vars=env_vars1,
        )

        state_file = (
            temp_htmlgraph_dir / "sessions" / session_manager.SESSION_STATE_FILE
        )
        state_data = json.loads(state_file.read_text())
        state_data["is_ended"] = True
        state_file.write_text(json.dumps(state_data))

        # Simulate compact
        session_id_2 = f"sess-{uuid4().hex[:8]}"
        monkeypatch.setenv("CLAUDE_SESSION_ID", session_id_2)

        # Get post-compact session state
        state2 = session_manager.get_current_state()
        session_manager.setup_environment_variables(state2)

        # Verify post-compact detection
        assert state2["is_post_compact"] is True
        assert state2["delegation_enabled"] is True

        # Create SDK post-compact
        sdk2 = SDK(
            directory=temp_htmlgraph_dir, agent="orchestrator", db_path=str(isolated_db)
        )
        assert sdk2.agent == "orchestrator"


class TestCompactCycleIntegration:
    """Integration tests for complete compact cycles."""

    def test_complete_compact_cycle_with_workflow(
        self, temp_htmlgraph_dir, isolated_db, session_manager, monkeypatch
    ):
        """
        Test complete compact cycle with realistic workflow.

        Test flow:
        1. Session 1: Create features with agent='explorer'
        2. Compact occurs
        3. Session 2: Create more features with agent='coder'
        4. Verify both sessions' work items have correct agents
        5. Verify session metadata is properly recorded
        """
        # Session 1: Explorer creates features
        session_id_1 = f"sess-explorer-{uuid4().hex[:8]}"
        monkeypatch.setenv("CLAUDE_SESSION_ID", session_id_1)
        monkeypatch.setenv("CLAUDE_ORCHESTRATOR_ACTIVE", "true")

        sdk1 = SDK(
            directory=temp_htmlgraph_dir, agent="explorer", db_path=str(isolated_db)
        )
        track1 = sdk1.tracks.create("Explorer Track").save()
        features_session1 = [
            sdk1.features.create(f"Research Task {i}").set_track(track1.id).save()
            for i in range(3)
        ]

        assert all(f.agent_assigned == "explorer" for f in features_session1)

        # Record session 1
        session_manager.record_state(
            session_id=session_id_1,
            source="startup",
            is_post_compact=False,
            delegation_enabled=True,
        )

        # End session 1
        state_file = (
            temp_htmlgraph_dir / "sessions" / session_manager.SESSION_STATE_FILE
        )
        state_data = json.loads(state_file.read_text())
        state_data["is_ended"] = True
        state_file.write_text(json.dumps(state_data))

        # Session 2: Coder continues work
        session_id_2 = f"sess-coder-{uuid4().hex[:8]}"
        monkeypatch.setenv("CLAUDE_SESSION_ID", session_id_2)

        sdk2 = SDK(
            directory=temp_htmlgraph_dir, agent="coder", db_path=str(isolated_db)
        )

        # Get post-compact state
        state2 = session_manager.get_current_state()
        assert state2["is_post_compact"] is True
        assert state2["previous_session_id"] == session_id_1

        # Create track in session 2
        track2 = sdk2.tracks.create("Coder Track").save()

        # Create features in session 2
        features_session2 = [
            sdk2.features.create(f"Implementation Task {i}").set_track(track2.id).save()
            for i in range(3)
        ]

        assert all(f.agent_assigned == "coder" for f in features_session2)

        # Record session 2
        session_manager.record_state(
            session_id=session_id_2,
            source="compact",
            is_post_compact=True,
            delegation_enabled=True,
        )

        # Verify we can retrieve all features
        explorer_features = sdk1.features.where(agent_assigned="explorer")
        coder_features = sdk2.features.where(agent_assigned="coder")

        assert len(explorer_features) == 3
        assert len(coder_features) == 3

    def test_multiple_compact_cycles(
        self, temp_htmlgraph_dir, isolated_db, session_manager, monkeypatch
    ):
        """
        Test multiple compact cycles with different agents.

        Test flow:
        1. Session 1 (explorer): Create 2 items
        2. Session 2 (coder): Create 2 items
        3. Session 3 (tester): Create 2 items
        4. Verify all items properly attributed
        5. Verify session chain is recorded
        """
        agents = ["explorer", "coder", "tester"]
        items_per_agent = []

        for i, agent in enumerate(agents):
            session_id = f"sess-{agent}-{uuid4().hex[:8]}"
            monkeypatch.setenv("CLAUDE_SESSION_ID", session_id)

            sdk = SDK(
                directory=temp_htmlgraph_dir, agent=agent, db_path=str(isolated_db)
            )

            # Create track first
            track = sdk.tracks.create(f"{agent.title()} Track").save()

            # Create features
            created = [
                sdk.features.create(f"{agent.title()} Feature {j}")
                .set_track(track.id)
                .save()
                for j in range(2)
            ]
            items_per_agent.append((agent, created))

            # Record session
            session_manager.record_state(
                session_id=session_id,
                source="compact" if i > 0 else "startup",
                is_post_compact=i > 0,
                delegation_enabled=True,
            )

            # End session (except last one)
            if i < len(agents) - 1:
                state_file = (
                    temp_htmlgraph_dir / "sessions" / session_manager.SESSION_STATE_FILE
                )
                state_data = json.loads(state_file.read_text())
                state_data["is_ended"] = True
                state_file.write_text(json.dumps(state_data))

        # Final verification with last SDK instance
        sdk_final = SDK(
            directory=temp_htmlgraph_dir, agent="tester", db_path=str(isolated_db)
        )

        for agent, items in items_per_agent:
            agent_items = sdk_final.features.where(agent_assigned=agent)
            assert len(agent_items) == 2
            for item in agent_items:
                assert item.agent_assigned == agent
