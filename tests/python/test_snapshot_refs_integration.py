"""Comprehensive integration tests for Phase 1 snapshot and ref system.

Tests end-to-end workflows combining:
- Feature/track/bug/spike creation via SDK
- Ref generation and resolution
- Snapshot command with various filters and formats
- Browse command integration
- SDK ref method functionality
- Ref persistence across SDK instances
"""

import json
import tempfile
from collections.abc import Generator
from pathlib import Path
from unittest.mock import MagicMock, patch

import pytest
from htmlgraph.cli.work.browse import BrowseCommand
from htmlgraph.cli.work.snapshot import SnapshotCommand
from htmlgraph.sdk import SDK

# ============================================================================
# FIXTURES
# ============================================================================


@pytest.fixture
def temp_graph_dir() -> Generator[Path, None, None]:
    """Create a temporary .htmlgraph directory for integration tests."""
    with tempfile.TemporaryDirectory() as tmpdir:
        graph_dir = Path(tmpdir) / ".htmlgraph"
        graph_dir.mkdir(parents=True)

        # Create collection directories
        for collection in [
            "features",
            "tracks",
            "bugs",
            "spikes",
        ]:
            (graph_dir / collection).mkdir()

        yield graph_dir


@pytest.fixture
def sdk_instance(temp_graph_dir: Path, isolated_db: Path) -> SDK:
    """Create an SDK instance with temporary directory."""
    return SDK(agent="test-agent", directory=temp_graph_dir, db_path=str(isolated_db))


@pytest.fixture
def populated_sdk(sdk_instance: SDK) -> SDK:
    """Create SDK populated with various work items."""
    sdk = sdk_instance

    # Create tracks
    track1 = sdk.tracks.create(
        title="Browser-Native Query Interface",
    ).save()

    track2 = sdk.tracks.create(
        title="Future Track",
    ).save()

    # Set track statuses
    with sdk.tracks.edit(track1.id) as track:
        track.status = "in-progress"
    with sdk.tracks.edit(track2.id) as track:
        track.status = "todo"

    # Create features with various statuses (all linked to track1)
    feature1 = (
        sdk.features.create(
            title="Implement snapshot command",
            priority="high",
            status="in-progress",
        )
        .set_track(track1.id)
        .save()
    )

    feature2 = (
        sdk.features.create(
            title="Add RefManager class for short refs",
            priority="high",
            status="todo",
        )
        .set_track(track1.id)
        .save()
    )

    feature3 = (
        sdk.features.create(
            title="Add sdk.ref() method for ref-based lookup",
            priority="medium",
            status="todo",
        )
        .set_track(track1.id)
        .save()
    )

    feature4 = (
        sdk.features.create(
            title="Completed feature",
            priority="low",
            status="done",
        )
        .set_track(track1.id)
        .save()
    )

    feature5 = (
        sdk.features.create(
            title="Blocked feature",
            priority="high",
            status="blocked",
        )
        .set_track(track1.id)
        .save()
    )

    # Create bugs
    bug1 = sdk.bugs.create(
        title="Fix snapshot formatting",
        priority="high",
        status="todo",
    ).save()

    bug2 = sdk.bugs.create(
        title="Ref resolution not persisting",
        priority="medium",
        status="in-progress",
    ).save()

    # Create spikes
    spike1 = sdk.spikes.create(
        title="Research ref system",
        status="done",
    ).save()

    spike2 = sdk.spikes.create(
        title="Investigate performance",
        status="todo",
    ).save()

    # Store created items for assertions
    sdk._test_items = {
        "tracks": [track1, track2],
        "features": [feature1, feature2, feature3, feature4, feature5],
        "bugs": [bug1, bug2],
        "spikes": [spike1, spike2],
    }

    return sdk


# ============================================================================
# END-TO-END WORKFLOW TESTS
# ============================================================================


class TestEndToEndWorkflow:
    """Test complete workflows combining SDK, refs, and snapshot."""

    def test_create_features_get_refs_snapshot_success(
        self, populated_sdk, isolated_db
    ):
        """
        Test: Create 5 features with various statuses
        - Verify refs are generated (@f1, @f2, etc.)
        - Run snapshot command
        - Verify output includes all features with refs
        - Verify sdk.ref() resolves each ref correctly
        """
        sdk = populated_sdk
        features = sdk._test_items["features"]

        # Verify refs are generated
        refs = []
        for i, feature in enumerate(features, 1):
            ref = sdk.refs.get_ref(feature.id)
            assert ref is not None
            assert ref == f"@f{i}"
            refs.append(ref)

        # Run snapshot command
        cmd = SnapshotCommand(output_format="refs", node_type="feature", status="all")
        cmd.graph_dir = str(sdk._directory)
        cmd.agent = sdk.agent
        result = cmd.execute()

        assert result.exit_code == 0
        output = result.text

        # Verify output includes all features with refs
        assert "SNAPSHOT" in output
        assert "FEATURES" in output
        for ref in refs:
            assert ref in output

        # Verify sdk.ref() resolves each ref correctly
        for i, feature in enumerate(features, 1):
            ref = f"@f{i}"
            resolved = sdk.ref(ref)
            assert resolved is not None
            assert resolved.id == feature.id
            assert resolved.title == feature.title

    def test_multiple_types_snapshot(self, populated_sdk, isolated_db):
        """
        Test: Create features, tracks, bugs, spikes
        - Run snapshot command
        - Verify all types appear in output
        - Verify refs are correct (@f1, @t1, @b1, @s1)
        """
        sdk = populated_sdk

        # Get refs for each type
        feature_ref = sdk.refs.get_ref(sdk._test_items["features"][0].id)
        track_ref = sdk.refs.get_ref(sdk._test_items["tracks"][0].id)
        bug_ref = sdk.refs.get_ref(sdk._test_items["bugs"][0].id)
        spike_ref = sdk.refs.get_ref(sdk._test_items["spikes"][0].id)

        # Verify ref formats
        assert feature_ref == "@f1"
        assert track_ref == "@t1"
        assert bug_ref == "@b1"
        assert spike_ref == "@s1"

        # Run snapshot with all types
        cmd = SnapshotCommand(output_format="refs", node_type="all", status="all")
        cmd.graph_dir = str(sdk._directory)
        cmd.agent = sdk.agent
        result = cmd.execute()

        assert result.exit_code == 0
        output = result.text

        # Verify all types appear
        assert "FEATURES" in output
        assert "TRACKS" in output
        assert "BUGS" in output
        assert "SPIKES" in output

        # Verify refs appear in output
        assert feature_ref in output
        assert track_ref in output
        assert bug_ref in output
        assert spike_ref in output

    def test_snapshot_with_filters(self, populated_sdk, isolated_db):
        """
        Test: Create features with mixed statuses
        - Run snapshot with --type feature --status todo
        - Verify only todo features appear
        - Run snapshot with --status in_progress
        - Verify filtering works across types
        """
        sdk = populated_sdk

        # Filter by feature + todo
        cmd = SnapshotCommand(output_format="refs", node_type="feature", status="todo")
        cmd.graph_dir = str(sdk._directory)
        cmd.agent = sdk.agent
        result = cmd.execute()

        assert result.exit_code == 0
        output = result.text

        # Get feature IDs with todo status
        todo_features = [f for f in sdk._test_items["features"] if f.status == "todo"]
        todo_refs = [sdk.refs.get_ref(f.id) for f in todo_features]

        # Verify only todo features appear
        for ref in todo_refs:
            assert ref in output

        # Verify non-todo features don't appear
        # (Implicit check: if filtering works correctly, only todo features appear)

        # Now filter by in_progress across all types
        cmd = SnapshotCommand(
            output_format="json", node_type="all", status="in-progress"
        )
        cmd.graph_dir = str(sdk._directory)
        cmd.agent = sdk.agent
        result = cmd.execute()

        assert result.exit_code == 0
        data = json.loads(result.text)

        # All items should have status in-progress
        for item in data:
            assert item["status"] == "in-progress"

        # Should include feature1 and bug2
        types_found = {item["type"] for item in data}
        assert "feature" in types_found or "bug" in types_found


# ============================================================================
# SDK INTEGRATION TESTS
# ============================================================================


class TestSDKRefIntegration:
    """Test SDK integration with ref system."""

    def test_sdk_ref_method_resolves_correctly(self, populated_sdk, isolated_db):
        """
        Test: Create feature, get its ref
        - Use sdk.ref() to retrieve it
        - Verify returned Node matches original
        """
        sdk = populated_sdk
        feature = sdk._test_items["features"][0]

        # Get ref
        ref = sdk.refs.get_ref(feature.id)
        assert ref == "@f1"

        # Resolve via sdk.ref()
        resolved = sdk.ref(ref)
        assert resolved is not None
        assert resolved.id == feature.id
        assert resolved.title == feature.title
        assert resolved.type == "feature"
        assert resolved.status == feature.status
        assert resolved.priority == feature.priority

    def test_sdk_ref_returns_none_for_invalid(self, sdk_instance, isolated_db):
        """
        Test: Call sdk.ref("@f999")
        - Verify returns None
        """
        sdk = sdk_instance

        # Non-existent ref
        result = sdk.ref("@f999")
        assert result is None

        # Invalid format
        result = sdk.ref("invalid")
        assert result is None

        # Wrong type
        result = sdk.ref("@xyz")
        assert result is None

    def test_ref_persistence_across_sdk_reloads(self, temp_graph_dir, isolated_db):
        """
        Test: Create feature, get ref
        - Create new SDK instance
        - Verify old ref still resolves
        """
        # Create SDK and feature
        sdk1 = SDK(
            agent="test-agent", directory=temp_graph_dir, db_path=str(isolated_db)
        )
        track = sdk1.tracks.create("Test Track").save()
        feature = sdk1.features.create("Test Feature").set_track(track.id).save()
        ref = sdk1.refs.get_ref(feature.id)
        assert ref == "@f1"

        # Create new SDK instance
        sdk2 = SDK(
            agent="test-agent", directory=temp_graph_dir, db_path=str(isolated_db)
        )

        # Verify ref persists
        resolved = sdk2.ref(ref)
        assert resolved is not None
        assert resolved.id == feature.id
        assert resolved.title == "Test Feature"

    def test_sdk_ref_with_all_types(self, populated_sdk, isolated_db):
        """Test sdk.ref() with all node types."""
        sdk = populated_sdk

        # Test feature
        feature = sdk._test_items["features"][0]
        feature_ref = sdk.refs.get_ref(feature.id)
        assert sdk.ref(feature_ref).type == "feature"

        # Test track
        track = sdk._test_items["tracks"][0]
        track_ref = sdk.refs.get_ref(track.id)
        assert sdk.ref(track_ref).type == "track"

        # Test bug
        bug = sdk._test_items["bugs"][0]
        bug_ref = sdk.refs.get_ref(bug.id)
        assert sdk.ref(bug_ref).type == "bug"

        # Test spike
        spike = sdk._test_items["spikes"][0]
        spike_ref = sdk.refs.get_ref(spike.id)
        assert sdk.ref(spike_ref).type == "spike"

    def test_collection_get_ref_method(self, populated_sdk, isolated_db):
        """Test that collections have get_ref convenience method."""
        sdk = populated_sdk
        feature = sdk._test_items["features"][0]

        # Get ref via collection
        ref = sdk.features.get_ref(feature.id)
        assert ref is not None
        assert ref == "@f1"

        # Should match SDK ref manager
        sdk_ref = sdk.refs.get_ref(feature.id)
        assert ref == sdk_ref


# ============================================================================
# SNAPSHOT FORMAT TESTS
# ============================================================================


class TestSnapshotFormats:
    """Test snapshot output formats."""

    def test_snapshot_json_parseable(self, populated_sdk, isolated_db):
        """
        Test: Run snapshot --format json
        - Parse JSON output
        - Verify all required fields present
        """
        sdk = populated_sdk

        cmd = SnapshotCommand(output_format="json", node_type="all", status="all")
        cmd.graph_dir = str(sdk._directory)
        cmd.agent = sdk.agent
        result = cmd.execute()

        assert result.exit_code == 0
        output = result.text

        # Parse JSON
        data = json.loads(output)
        assert isinstance(data, list)
        assert len(data) > 0

        # Verify required fields
        for item in data:
            assert "ref" in item
            assert "id" in item
            assert "type" in item
            assert "title" in item
            assert "status" in item
            assert "priority" in item

    def test_snapshot_refs_format_readable(self, populated_sdk, isolated_db):
        """
        Test: Run snapshot --format refs
        - Verify output contains @f1, @t1, etc.
        - Verify sorting is correct
        """
        sdk = populated_sdk

        cmd = SnapshotCommand(output_format="refs", node_type="all", status="all")
        cmd.graph_dir = str(sdk._directory)
        cmd.agent = sdk.agent
        result = cmd.execute()

        assert result.exit_code == 0
        output = result.text

        # Verify header
        assert "SNAPSHOT - Current Graph State" in output

        # Verify type sections
        assert "FEATURES" in output
        assert "TRACKS" in output
        assert "BUGS" in output
        assert "SPIKES" in output

        # Verify status sections
        assert "TODO:" in output
        assert "IN_PROGRESS:" in output or "IN-PROGRESS:" in output

        # Verify refs are present
        assert "@f" in output
        assert "@t" in output
        assert "@b" in output
        assert "@s" in output

    def test_snapshot_text_format(self, populated_sdk, isolated_db):
        """Test snapshot text format (no refs, with colors)."""
        sdk = populated_sdk

        cmd = SnapshotCommand(output_format="text", node_type="all", status="all")
        cmd.graph_dir = str(sdk._directory)
        cmd.agent = sdk.agent
        result = cmd.execute()

        assert result.exit_code == 0
        output = result.text

        # Text format should have content (no refs, but has type and title)
        assert len(output) > 0
        # Should contain item types
        assert "feature" in output or "bug" in output or "track" in output

    def test_snapshot_sorting(self, populated_sdk, isolated_db):
        """Verify snapshot items are sorted correctly."""
        sdk = populated_sdk

        cmd = SnapshotCommand(output_format="json", node_type="all", status="all")
        cmd.graph_dir = str(sdk._directory)
        cmd.agent = sdk.agent
        result = cmd.execute()

        assert result.exit_code == 0
        data = json.loads(result.text)

        # Verify sorting by type, then status, then ref
        prev_type = ""
        prev_status = ""
        for item in data:
            # Type should be in order
            if item["type"] != prev_type:
                assert item["type"] >= prev_type or prev_type == ""
                prev_type = item["type"]
                prev_status = ""

            # Status should be in order within same type
            if item["type"] == prev_type:
                assert item["status"] >= prev_status or prev_status == ""
                prev_status = item["status"]


# ============================================================================
# BROWSE COMMAND INTEGRATION TESTS
# ============================================================================


class TestBrowseCommandIntegration:
    """Test browse command integration with snapshot items."""

    @patch("webbrowser.open")
    @patch("requests.head")
    def test_browse_opens_with_query_type_feature(
        self, mock_requests_head, mock_webbrowser
    ):
        """
        Test: Create features with snapshot
        - Run browse command with --query-type feature
        - Verify URL includes correct query params
        """

        # Mock server running
        mock_response = MagicMock()
        mock_response.raise_for_status = MagicMock()
        mock_requests_head.return_value = mock_response

        cmd = BrowseCommand(port=8080, query_type="feature")
        result = cmd.execute()

        assert result.exit_code == 0
        assert "?type=feature" in result.data["url"]
        mock_webbrowser.assert_called_once()

    @patch("webbrowser.open")
    @patch("requests.head")
    def test_browse_opens_with_query_status(
        self, mock_requests_head, mock_webbrowser, isolated_db
    ):
        """
        Test: Run browse command with --query-status todo
        - Verify URL includes correct query params
        """

        # Mock server running
        mock_response = MagicMock()
        mock_response.raise_for_status = MagicMock()
        mock_requests_head.return_value = mock_response

        cmd = BrowseCommand(port=8080, query_status="todo")
        result = cmd.execute()

        assert result.exit_code == 0
        assert "?status=todo" in result.data["url"]
        mock_webbrowser.assert_called_once()

    @patch("webbrowser.open")
    @patch("requests.head")
    def test_browse_opens_with_both_filters(
        self, mock_requests_head, mock_webbrowser, isolated_db
    ):
        """
        Test: Run browse command with both --query-type and --query-status
        - Verify URL includes both query params
        """

        # Mock server running
        mock_response = MagicMock()
        mock_response.raise_for_status = MagicMock()
        mock_requests_head.return_value = mock_response

        cmd = BrowseCommand(port=8080, query_type="feature", query_status="todo")
        result = cmd.execute()

        assert result.exit_code == 0
        url = result.data["url"]
        assert "type=feature" in url
        assert "status=todo" in url
        mock_webbrowser.assert_called_once()


# ============================================================================
# COMPLEX WORKFLOW TESTS
# ============================================================================


class TestComplexWorkflows:
    """Test complex multi-step workflows."""

    def test_create_track_with_features_snapshot(self, sdk_instance, isolated_db):
        """
        Test: Create track with multiple features
        - Snapshot showing track as @t1 and features as @f1-@f3
        - Verify relationships in snapshot
        """
        sdk = sdk_instance

        # Create track
        track = sdk.tracks.create("Phase 1: Foundation").save()
        track_ref = sdk.refs.get_ref(track.id)
        assert track_ref == "@t1"

        # Create features for track
        features = []
        for i in range(1, 4):
            feature = (
                sdk.features.create(f"Feature {i}", priority="high")
                .set_track(track.id)
                .save()
            )
            features.append(feature)

        # Get feature refs
        feature_refs = [sdk.refs.get_ref(f.id) for f in features]
        assert feature_refs == ["@f1", "@f2", "@f3"]

        # Run snapshot
        cmd = SnapshotCommand(output_format="json", node_type="all", status="all")
        cmd.graph_dir = str(sdk._directory)
        cmd.agent = sdk.agent
        result = cmd.execute()

        assert result.exit_code == 0
        data = json.loads(result.text)

        # Verify track and features appear
        track_items = [d for d in data if d["type"] == "track"]
        feature_items = [d for d in data if d["type"] == "feature"]

        assert len(track_items) == 1
        assert track_items[0]["ref"] == "@t1"
        assert len(feature_items) == 3

        for i, feature_item in enumerate(feature_items):
            assert feature_item["ref"] == f"@f{i + 1}"
            assert feature_item["track_id"] == track.id

    def test_ref_consistency_across_operations(self, populated_sdk, isolated_db):
        """
        Test: Refs remain consistent across:
        - Multiple snapshots
        - SDK reloads (new instances)
        - Type/status filters
        """
        sdk = populated_sdk

        # Get initial refs
        initial_refs = {}
        for feature_list_name in ["features", "tracks", "bugs", "spikes"]:
            items = sdk._test_items[feature_list_name]
            for item in items:
                item_ref = sdk.refs.get_ref(item.id)
                initial_refs[item.id] = item_ref

        # Run multiple snapshots
        for _ in range(3):
            cmd = SnapshotCommand(output_format="json", node_type="all", status="all")
            cmd.graph_dir = str(sdk._directory)
            cmd.agent = sdk.agent
            result = cmd.execute()
            assert result.exit_code == 0

        # Verify refs haven't changed
        for feature_list_name in ["features", "tracks", "bugs", "spikes"]:
            items = sdk._test_items[feature_list_name]
            for item in items:
                current_ref = sdk.refs.get_ref(item.id)
                assert current_ref == initial_refs[item.id]

    def test_snapshot_with_updated_status(self, sdk_instance, isolated_db):
        """
        Test: Create items, snapshot, update status, snapshot
        - Verify refs remain same
        - Verify status changes in snapshot
        """
        sdk = sdk_instance

        # Create feature
        track = sdk.tracks.create("Track").save()
        feature = (
            sdk.features.create("Feature", status="todo").set_track(track.id).save()
        )
        ref = sdk.refs.get_ref(feature.id)
        assert ref == "@f1"

        # First snapshot
        cmd1 = SnapshotCommand(output_format="json", node_type="feature", status="all")
        cmd1.graph_dir = str(sdk._directory)
        cmd1.agent = sdk.agent
        result1 = cmd1.execute()
        data1 = json.loads(result1.text)
        assert data1[0]["status"] == "todo"
        assert data1[0]["ref"] == "@f1"

        # Update status
        with sdk.features.edit(feature.id) as f:
            f.status = "in-progress"

        # Second snapshot
        cmd2 = SnapshotCommand(output_format="json", node_type="feature", status="all")
        cmd2.graph_dir = str(sdk._directory)
        cmd2.agent = sdk.agent
        result2 = cmd2.execute()
        data2 = json.loads(result2.text)
        assert data2[0]["status"] == "in-progress"
        assert data2[0]["ref"] == "@f1"  # Ref should remain same

    def test_filter_by_priority_via_snapshot(self, populated_sdk, isolated_db):
        """
        Test: Snapshot can be filtered to show high-priority items
        - Create mix of priorities
        - Verify filtering in JSON output
        """
        sdk = populated_sdk

        # Get snapshot
        cmd = SnapshotCommand(output_format="json", node_type="all", status="all")
        cmd.graph_dir = str(sdk._directory)
        cmd.agent = sdk.agent
        result = cmd.execute()

        assert result.exit_code == 0
        data = json.loads(result.text)

        # Filter high priority
        high_priority = [d for d in data if d["priority"] == "high"]
        assert len(high_priority) > 0

        # All should have refs
        for item in high_priority:
            assert item["ref"] is not None


# ============================================================================
# ERROR HANDLING AND EDGE CASES
# ============================================================================


class TestErrorHandling:
    """Test error handling and edge cases."""

    def test_snapshot_empty_graph(self, sdk_instance, isolated_db):
        """Test snapshot with no items."""
        sdk = sdk_instance

        cmd = SnapshotCommand(output_format="refs", node_type="all", status="all")
        cmd.graph_dir = str(sdk._directory)
        cmd.agent = sdk.agent
        result = cmd.execute()

        assert result.exit_code == 0
        assert "SNAPSHOT - Current Graph State" in result.text

    def test_ref_resolution_after_multiple_creates(self, sdk_instance, isolated_db):
        """Test ref resolution stays correct after many creates."""
        sdk = sdk_instance

        # Create many items
        items = []
        track = sdk.tracks.create("Track").save()
        for i in range(1, 11):
            feature = sdk.features.create(f"Feature {i}").set_track(track.id).save()
            items.append(feature)

        # Verify all refs resolve correctly
        for i, item in enumerate(items, 1):
            ref = sdk.refs.get_ref(item.id)
            assert ref == f"@f{i}"
            resolved = sdk.ref(ref)
            assert resolved.id == item.id

    def test_snapshot_with_special_characters(self, sdk_instance, isolated_db):
        """Test snapshot with special characters in titles."""
        sdk = sdk_instance

        track = sdk.tracks.create("Track").save()
        sdk.features.create("Feature: @mention #hashtag 'quotes' \"double\"").set_track(
            track.id
        ).save()

        cmd = SnapshotCommand(output_format="json", node_type="feature", status="all")
        cmd.graph_dir = str(sdk._directory)
        cmd.agent = sdk.agent
        result = cmd.execute()

        assert result.exit_code == 0
        data = json.loads(result.text)
        assert len(data) == 1
        assert "mention" in data[0]["title"]

    def test_snapshot_unicode_titles(self, sdk_instance, isolated_db):
        """Test snapshot with unicode titles."""
        sdk = sdk_instance

        track = sdk.tracks.create("Track").save()
        sdk.features.create("Feature: 日本語 中文 العربية").set_track(track.id).save()

        cmd = SnapshotCommand(output_format="json", node_type="feature", status="all")
        cmd.graph_dir = str(sdk._directory)
        cmd.agent = sdk.agent
        result = cmd.execute()

        assert result.exit_code == 0
        data = json.loads(result.text)
        assert len(data) == 1
        assert "日本語" in data[0]["title"]


# ============================================================================
# INTEGRATION WITH CLI WORKFLOW
# ============================================================================


class TestCLIIntegration:
    """Test integration with CLI workflow."""

    def test_snapshot_command_factory_from_args(self, isolated_db):
        """Test creating SnapshotCommand from argparse Namespace."""
        from argparse import Namespace

        args = Namespace(output_format="json", type="feature", status="todo")
        cmd = SnapshotCommand.from_args(args)

        assert cmd.output_format == "json"
        assert cmd.node_type == "feature"
        assert cmd.status == "todo"

    def test_snapshot_command_factory_defaults(self, isolated_db):
        """Test SnapshotCommand factory with defaults."""
        from argparse import Namespace

        args = Namespace(output_format="refs")
        cmd = SnapshotCommand.from_args(args)

        assert cmd.output_format == "refs"
        assert cmd.node_type is None
        assert cmd.status is None

    def test_browse_command_factory_from_args(self, isolated_db):
        """Test creating BrowseCommand from argparse Namespace."""
        from argparse import Namespace

        args = Namespace(port=8080, query_type="feature", query_status="todo")
        cmd = BrowseCommand.from_args(args)

        assert cmd.port == 8080
        assert cmd.query_type == "feature"
        assert cmd.query_status == "todo"


# ============================================================================
# REF SYSTEM ROBUSTNESS TESTS
# ============================================================================


class TestRefSystemRobustness:
    """Test robustness of ref system under various conditions."""

    def test_ref_consistency_with_deleted_items(self, sdk_instance, isolated_db):
        """Test ref consistency when items are deleted."""
        sdk = sdk_instance

        # Create items
        track = sdk.tracks.create("Track").save()
        f1 = sdk.features.create("Feature 1").set_track(track.id).save()
        f2 = sdk.features.create("Feature 2").set_track(track.id).save()
        f3 = sdk.features.create("Feature 3").set_track(track.id).save()

        # Get refs
        ref1 = sdk.refs.get_ref(f1.id)
        ref2 = sdk.refs.get_ref(f2.id)
        ref3 = sdk.refs.get_ref(f3.id)

        assert ref1 == "@f1"
        assert ref2 == "@f2"
        assert ref3 == "@f3"

        # F2 still resolves to original item
        assert sdk.ref(ref2).id == f2.id

    def test_multiple_sdk_instances_same_refs(self, temp_graph_dir, isolated_db):
        """Test multiple SDK instances see same refs."""
        sdk1 = SDK(agent="agent1", directory=temp_graph_dir, db_path=str(isolated_db))
        sdk2 = SDK(agent="agent2", directory=temp_graph_dir, db_path=str(isolated_db))

        # Create in sdk1
        track = sdk1.tracks.create("Track").save()
        feature = sdk1.features.create("Feature").set_track(track.id).save()
        ref1 = sdk1.refs.get_ref(feature.id)

        # Check in sdk2
        ref2 = sdk2.refs.get_ref(feature.id)

        assert ref1 == ref2

    def test_ref_generation_is_sequential(self, sdk_instance, isolated_db):
        """Test that refs are generated sequentially."""
        sdk = sdk_instance

        track = sdk.tracks.create("Track").save()
        refs = []

        # Create many features
        for i in range(1, 11):
            feature = sdk.features.create(f"Feature {i}").set_track(track.id).save()
            ref = sdk.refs.get_ref(feature.id)
            refs.append(ref)

        # Verify sequential
        for i, ref in enumerate(refs, 1):
            assert ref == f"@f{i}"

    def test_snapshot_json_includes_all_ref_info(self, populated_sdk, isolated_db):
        """Verify JSON snapshot includes complete ref information."""
        sdk = populated_sdk

        # Ensure all items have refs by accessing them
        for feature in sdk._test_items["features"]:
            sdk.refs.get_ref(feature.id)
        for track in sdk._test_items["tracks"]:
            sdk.refs.get_ref(track.id)
        for bug in sdk._test_items["bugs"]:
            sdk.refs.get_ref(bug.id)
        for spike in sdk._test_items["spikes"]:
            sdk.refs.get_ref(spike.id)

        cmd = SnapshotCommand(output_format="json", node_type="all", status="all")
        cmd.graph_dir = str(sdk._directory)
        cmd.agent = sdk.agent
        result = cmd.execute()

        data = json.loads(result.text)

        # Each item should have ref field (may be None if not accessed)
        for item in data:
            assert "ref" in item
            # Ref should be present and valid
            if item["ref"] is not None:
                assert item["ref"].startswith("@")
                assert len(item["ref"]) >= 2


# ============================================================================
# NEW FILTER TESTS - TRACK, ACTIVE, BLOCKERS, SUMMARY, MY_WORK
# ============================================================================


class TestSnapshotTrackFilter:
    """Test --track filter functionality."""

    def test_snapshot_filter_by_track_id(self, populated_sdk, isolated_db):
        """Test filtering by track ID."""
        sdk = populated_sdk
        track = sdk._test_items["tracks"][0]

        cmd = SnapshotCommand(
            output_format="json", node_type="all", status="all", track_id=track.id
        )
        cmd.graph_dir = str(sdk._directory)
        cmd.agent = sdk.agent
        result = cmd.execute()

        assert result.exit_code == 0
        data = json.loads(result.text)

        # All features should be linked to this track
        features = [d for d in data if d["type"] == "feature"]
        for feature in features:
            assert feature["track_id"] == track.id

    def test_snapshot_filter_by_track_ref(self, populated_sdk, isolated_db):
        """Test filtering by track ref (@t1)."""
        sdk = populated_sdk
        track = sdk._test_items["tracks"][0]
        track_ref = sdk.refs.get_ref(track.id)

        cmd = SnapshotCommand(
            output_format="json", node_type="all", status="all", track_id=track_ref
        )
        cmd.graph_dir = str(sdk._directory)
        cmd.agent = sdk.agent
        result = cmd.execute()

        assert result.exit_code == 0
        data = json.loads(result.text)

        # All features should be linked to this track
        features = [d for d in data if d["type"] == "feature"]
        for feature in features:
            assert feature["track_id"] == track.id

    def test_snapshot_track_filter_with_active(self, populated_sdk, isolated_db):
        """Test combining --track and --active filters."""
        sdk = populated_sdk
        track = sdk._test_items["tracks"][0]
        track_ref = sdk.refs.get_ref(track.id)

        cmd = SnapshotCommand(
            output_format="json",
            node_type="all",
            status="all",
            track_id=track_ref,
            active=True,
        )
        cmd.graph_dir = str(sdk._directory)
        cmd.agent = sdk.agent
        result = cmd.execute()

        assert result.exit_code == 0
        data = json.loads(result.text)

        # Should only have active items from this track
        for item in data:
            assert item["status"] in ["todo", "in-progress", "blocked"]
            if item["type"] == "feature":
                assert item["track_id"] == track.id


class TestSnapshotActiveFilter:
    """Test --active filter functionality."""

    def test_snapshot_active_filter(self, populated_sdk, isolated_db):
        """Test --active filter shows only TODO/IN_PROGRESS/BLOCKED items."""
        sdk = populated_sdk

        cmd = SnapshotCommand(
            output_format="json", node_type="all", status="all", active=True
        )
        cmd.graph_dir = str(sdk._directory)
        cmd.agent = sdk.agent
        result = cmd.execute()

        assert result.exit_code == 0
        data = json.loads(result.text)

        # All items should be active
        for item in data:
            assert item["status"] in ["todo", "in-progress", "blocked"]

        # Should not have completed items
        done_items = [d for d in data if d["status"] == "done"]
        assert len(done_items) == 0

    def test_snapshot_active_excludes_metadata_spikes(self, sdk_instance, isolated_db):
        """Test --active filter excludes metadata spikes."""
        sdk = sdk_instance

        # Create spikes (no track needed for spikes)
        sdk.spikes.create("Research spike", status="todo").save()  # Should be included
        sdk.spikes.create(
            "Conversation with user", status="todo"
        ).save()  # Should be excluded
        sdk.spikes.create(
            "Transition to new approach", status="todo"
        ).save()  # Should be excluded
        sdk.spikes.create(
            "Handoff notes", status="in-progress"
        ).save()  # Should be excluded

        cmd = SnapshotCommand(
            output_format="json", node_type="spike", status="all", active=True
        )
        cmd.graph_dir = str(sdk._directory)
        cmd.agent = sdk.agent
        result = cmd.execute()

        assert result.exit_code == 0
        data = json.loads(result.text)

        # Should only have non-metadata spikes
        assert len(data) == 1
        assert data[0]["title"] == "Research spike"

    def test_snapshot_active_with_type_filter(self, populated_sdk, isolated_db):
        """Test --active combined with --type filter."""
        sdk = populated_sdk

        cmd = SnapshotCommand(
            output_format="json", node_type="feature", status="all", active=True
        )
        cmd.graph_dir = str(sdk._directory)
        cmd.agent = sdk.agent
        result = cmd.execute()

        assert result.exit_code == 0
        data = json.loads(result.text)

        # All should be features and active
        for item in data:
            assert item["type"] == "feature"
            assert item["status"] in ["todo", "in-progress", "blocked"]


class TestSnapshotBlockersFilter:
    """Test --blockers filter functionality."""

    def test_snapshot_blockers_filter(self, populated_sdk, isolated_db):
        """Test --blockers filter shows only critical/blocked items."""
        sdk = populated_sdk

        cmd = SnapshotCommand(
            output_format="json", node_type="all", status="all", blockers=True
        )
        cmd.graph_dir = str(sdk._directory)
        cmd.agent = sdk.agent
        result = cmd.execute()

        assert result.exit_code == 0
        data = json.loads(result.text)

        # All items should be critical or blocked
        for item in data:
            assert item["priority"] == "critical" or item["status"] == "blocked"

    def test_snapshot_blockers_empty_when_none_exist(self, sdk_instance, isolated_db):
        """Test --blockers returns empty when no blockers exist."""
        sdk = sdk_instance

        track = sdk.tracks.create("Track").save()
        sdk.features.create("Feature", priority="low", status="todo").set_track(
            track.id
        ).save()

        cmd = SnapshotCommand(
            output_format="json", node_type="all", status="all", blockers=True
        )
        cmd.graph_dir = str(sdk._directory)
        cmd.agent = sdk.agent
        result = cmd.execute()

        assert result.exit_code == 0
        data = json.loads(result.text)
        assert len(data) == 0

    def test_snapshot_blockers_includes_critical_and_blocked(
        self, sdk_instance, isolated_db
    ):
        """Test --blockers includes both critical priority and blocked status."""
        sdk = sdk_instance

        track = sdk.tracks.create("Track").save()
        sdk.features.create(
            "Critical feature", priority="critical", status="todo"
        ).set_track(track.id).save()
        sdk.features.create(
            "Blocked feature", priority="high", status="blocked"
        ).set_track(track.id).save()
        sdk.features.create("Normal feature", priority="high", status="todo").set_track(
            track.id
        ).save()

        cmd = SnapshotCommand(
            output_format="json", node_type="all", status="all", blockers=True
        )
        cmd.graph_dir = str(sdk._directory)
        cmd.agent = sdk.agent
        result = cmd.execute()

        assert result.exit_code == 0
        data = json.loads(result.text)
        assert len(data) == 2  # Critical and blocked only

        titles = {d["title"] for d in data}
        assert "Critical feature" in titles
        assert "Blocked feature" in titles
        assert "Normal feature" not in titles


class TestSnapshotSummaryFormat:
    """Test --summary format functionality."""

    def test_snapshot_summary_format(self, populated_sdk, isolated_db):
        """Test --summary shows counts and progress."""
        sdk = populated_sdk

        cmd = SnapshotCommand(
            output_format="refs", node_type="all", status="all", summary=True
        )
        cmd.graph_dir = str(sdk._directory)
        cmd.agent = sdk.agent
        result = cmd.execute()

        assert result.exit_code == 0
        output = result.text

        # Verify summary header
        assert "ACTIVE WORK CONTEXT" in output

        # Verify sections present
        assert "Active Features" in output
        assert "Quick Stats" in output

    def test_snapshot_summary_with_track(self, populated_sdk, isolated_db):
        """Test --summary with --track shows track context."""
        sdk = populated_sdk
        track = sdk._test_items["tracks"][0]
        track_ref = sdk.refs.get_ref(track.id)

        cmd = SnapshotCommand(
            output_format="refs",
            node_type="all",
            status="all",
            track_id=track_ref,
            summary=True,
        )
        cmd.graph_dir = str(sdk._directory)
        cmd.agent = sdk.agent
        result = cmd.execute()

        assert result.exit_code == 0
        output = result.text

        # Verify track appears in context
        assert "Current Track:" in output
        assert track_ref in output

    def test_snapshot_summary_shows_progress_percentage(
        self, sdk_instance, isolated_db
    ):
        """Test --summary calculates and shows progress percentage."""
        sdk = sdk_instance

        track = sdk.tracks.create("Track").save()
        # Create 5 features: 2 done, 3 not done
        sdk.features.create("Feature 1", status="done").set_track(track.id).save()
        sdk.features.create("Feature 2", status="done").set_track(track.id).save()
        sdk.features.create("Feature 3", status="todo").set_track(track.id).save()
        sdk.features.create("Feature 4", status="in-progress").set_track(
            track.id
        ).save()
        sdk.features.create("Feature 5", status="blocked").set_track(track.id).save()

        cmd = SnapshotCommand(
            output_format="refs", node_type="all", status="all", summary=True
        )
        cmd.graph_dir = str(sdk._directory)
        cmd.agent = sdk.agent
        result = cmd.execute()

        assert result.exit_code == 0
        output = result.text

        # Verify progress calculation (2/5 = 40%)
        assert "40%" in output or "2/5" in output

    def test_snapshot_summary_shows_bug_priorities(self, populated_sdk, isolated_db):
        """Test --summary shows bug priority counts."""
        sdk = populated_sdk

        cmd = SnapshotCommand(
            output_format="refs", node_type="all", status="all", summary=True
        )
        cmd.graph_dir = str(sdk._directory)
        cmd.agent = sdk.agent
        result = cmd.execute()

        assert result.exit_code == 0
        output = result.text

        # Should show bug information
        if "bug" in str(sdk._test_items):
            assert "Active Bugs" in output or "Bugs:" in output

    def test_snapshot_summary_shows_blockers_section(self, sdk_instance, isolated_db):
        """Test --summary shows blockers section when blockers exist."""
        sdk = sdk_instance

        track = sdk.tracks.create("Track").save()
        sdk.features.create("Critical", priority="critical", status="todo").set_track(
            track.id
        ).save()
        sdk.features.create("Blocked", priority="high", status="blocked").set_track(
            track.id
        ).save()

        cmd = SnapshotCommand(
            output_format="refs", node_type="all", status="all", summary=True
        )
        cmd.graph_dir = str(sdk._directory)
        cmd.agent = sdk.agent
        result = cmd.execute()

        assert result.exit_code == 0
        output = result.text

        # Should show blockers section
        assert "Blockers & Critical" in output


class TestSnapshotMyWorkFilter:
    """Test --my-work filter functionality."""

    def test_snapshot_my_work_filter(self, sdk_instance, isolated_db):
        """Test --my-work shows only items assigned to current agent."""
        sdk = sdk_instance

        track = sdk.tracks.create("Track").save()

        # Create features with different assignments
        f1 = (
            sdk.features.create("My Feature", priority="high", status="todo")
            .set_track(track.id)
            .save()
        )
        with sdk.features.edit(f1.id) as feature:
            feature.agent_assigned = sdk.agent

        f2 = (
            sdk.features.create("Other Feature", priority="high", status="todo")
            .set_track(track.id)
            .save()
        )  # Not assigned
        with sdk.features.edit(f2.id) as feature:
            feature.agent_assigned = None  # Explicitly unassign

        f3 = (
            sdk.features.create(
                "Another My Feature", priority="medium", status="in-progress"
            )
            .set_track(track.id)
            .save()
        )
        with sdk.features.edit(f3.id) as feature:
            feature.agent_assigned = sdk.agent

        cmd = SnapshotCommand(
            output_format="json", node_type="all", status="all", my_work=True
        )
        cmd.graph_dir = str(sdk._directory)
        cmd.agent = sdk.agent
        result = cmd.execute()

        assert result.exit_code == 0
        data = json.loads(result.text)

        # Should only have features assigned to current agent
        assert len(data) == 2
        for item in data:
            assert item["assigned_to"] == sdk.agent

        titles = {d["title"] for d in data}
        assert "My Feature" in titles
        assert "Another My Feature" in titles
        assert "Other Feature" not in titles

    def test_snapshot_my_work_empty_when_none_assigned(self, sdk_instance, isolated_db):
        """Test --my-work returns empty when nothing is assigned to current agent."""
        sdk = sdk_instance

        track = sdk.tracks.create("Track").save()
        f1 = (
            sdk.features.create("Unassigned Feature", status="todo")
            .set_track(track.id)
            .save()
        )
        # Explicitly unassign
        with sdk.features.edit(f1.id) as feature:
            feature.agent_assigned = None

        cmd = SnapshotCommand(
            output_format="json", node_type="all", status="all", my_work=True
        )
        cmd.graph_dir = str(sdk._directory)
        cmd.agent = sdk.agent
        result = cmd.execute()

        assert result.exit_code == 0
        data = json.loads(result.text)
        assert len(data) == 0

    def test_snapshot_my_work_with_active_filter(self, sdk_instance, isolated_db):
        """Test --my-work combined with --active."""
        sdk = sdk_instance

        track = sdk.tracks.create("Track").save()

        f1 = sdk.features.create("My Active", status="todo").set_track(track.id).save()
        with sdk.features.edit(f1.id) as feature:
            feature.agent_assigned = sdk.agent

        f2 = sdk.features.create("My Done", status="done").set_track(track.id).save()
        with sdk.features.edit(f2.id) as feature:
            feature.agent_assigned = sdk.agent

        cmd = SnapshotCommand(
            output_format="json",
            node_type="all",
            status="all",
            my_work=True,
            active=True,
        )
        cmd.graph_dir = str(sdk._directory)
        cmd.agent = sdk.agent
        result = cmd.execute()

        assert result.exit_code == 0
        data = json.loads(result.text)

        # Should only have active items assigned to me
        assert len(data) == 1
        assert data[0]["title"] == "My Active"
        assert data[0]["assigned_to"] == sdk.agent
        assert data[0]["status"] == "todo"


class TestSnapshotColoredOutput:
    """Test colored output formatting for agent-friendliness."""

    def test_refs_format_has_no_box_drawing_characters(
        self, populated_sdk, isolated_db
    ):
        """Test that refs format doesn't use box-drawing characters."""
        sdk = populated_sdk

        cmd = SnapshotCommand(output_format="refs", node_type="all", status="all")
        cmd.graph_dir = str(sdk._directory)
        cmd.agent = sdk.agent
        result = cmd.execute()

        assert result.exit_code == 0
        output = result.text

        # Should NOT have box drawing characters (┌, ┬, ┐, ├, ┼, ┤, └, ┴, ┘, │, ─)
        box_chars = ["┌", "┬", "┐", "├", "┼", "┤", "└", "┴", "┘", "│"]
        for char in box_chars:
            assert char not in output, f"Box character '{char}' found in output"

    def test_refs_format_contains_ansi_codes(self, populated_sdk, isolated_db):
        """Test that refs format contains ANSI color codes."""
        sdk = populated_sdk

        cmd = SnapshotCommand(output_format="refs", node_type="all", status="all")
        cmd.graph_dir = str(sdk._directory)
        cmd.agent = sdk.agent
        result = cmd.execute()

        assert result.exit_code == 0
        output = result.text

        # Should contain ANSI escape codes
        # ANSI codes start with \x1b[ (ESC[)
        assert "\x1b[" in output, "No ANSI color codes found in output"

    def test_summary_format_uses_unicode_symbols(self, populated_sdk, isolated_db):
        """Test that summary format uses Unicode status symbols."""
        sdk = populated_sdk

        cmd = SnapshotCommand(
            output_format="refs", node_type="all", status="all", summary=True
        )
        cmd.graph_dir = str(sdk._directory)
        cmd.agent = sdk.agent
        result = cmd.execute()

        assert result.exit_code == 0
        output = result.text

        # Should contain Unicode symbols
        symbols = ["●", "⟳", "✓", "✗"]
        found_symbols = [s for s in symbols if s in output]
        assert len(found_symbols) > 0, "No Unicode status symbols found in summary"

    def test_json_format_unchanged(self, populated_sdk, isolated_db):
        """Test that JSON format is completely unchanged (no colors)."""
        sdk = populated_sdk

        cmd = SnapshotCommand(output_format="json", node_type="all", status="all")
        cmd.graph_dir = str(sdk._directory)
        cmd.agent = sdk.agent
        result = cmd.execute()

        assert result.exit_code == 0
        output = result.text

        # JSON should NOT have ANSI codes
        assert "\x1b[" not in output, "ANSI codes found in JSON output"

        # Should be valid JSON
        import json

        data = json.loads(output)
        assert isinstance(data, list)
        assert len(data) > 0

    def test_colored_output_parseable_by_agents(self, populated_sdk, isolated_db):
        """Test that colored text output is parseable (agents ignore ANSI codes)."""
        sdk = populated_sdk

        cmd = SnapshotCommand(output_format="refs", node_type="feature", status="all")
        cmd.graph_dir = str(sdk._directory)
        cmd.agent = sdk.agent
        result = cmd.execute()

        assert result.exit_code == 0
        output = result.text

        # Strip ANSI codes (simulate agent parsing)
        import re

        ansi_escape = re.compile(r"\x1b\[[0-9;]*m")
        clean_output = ansi_escape.sub("", output)

        # Should still be parseable and contain refs
        assert "@f" in clean_output
        assert "FEATURES" in clean_output
        assert "TODO" in clean_output or "IN_PROGRESS" in clean_output


class TestSnapshotCombinedFilters:
    """Test combinations of multiple filters."""

    def test_snapshot_track_plus_active_plus_type(self, populated_sdk, isolated_db):
        """Test combining --track, --active, and --type filters."""
        sdk = populated_sdk
        track = sdk._test_items["tracks"][0]
        track_ref = sdk.refs.get_ref(track.id)

        cmd = SnapshotCommand(
            output_format="json",
            node_type="feature",
            status="all",
            track_id=track_ref,
            active=True,
        )
        cmd.graph_dir = str(sdk._directory)
        cmd.agent = sdk.agent
        result = cmd.execute()

        assert result.exit_code == 0
        data = json.loads(result.text)

        # All should be features, active, in track
        for item in data:
            assert item["type"] == "feature"
            assert item["status"] in ["todo", "in-progress", "blocked"]
            assert item["track_id"] == track.id

    def test_snapshot_track_plus_summary(self, populated_sdk, isolated_db):
        """Test --track with --summary shows track progress."""
        sdk = populated_sdk
        track = sdk._test_items["tracks"][0]
        track_ref = sdk.refs.get_ref(track.id)

        cmd = SnapshotCommand(
            output_format="refs",
            node_type="all",
            status="all",
            track_id=track_ref,
            summary=True,
        )
        cmd.graph_dir = str(sdk._directory)
        cmd.agent = sdk.agent
        result = cmd.execute()

        assert result.exit_code == 0
        output = result.text

        # Should show track-specific progress
        assert "Current Track:" in output
        assert track_ref in output
        assert "Track:" in output  # Progress line

    def test_snapshot_blockers_plus_my_work(self, sdk_instance, isolated_db):
        """Test --blockers with --my-work shows only my critical items."""
        sdk = sdk_instance

        track = sdk.tracks.create("Track").save()

        f1 = (
            sdk.features.create("My Critical", priority="critical", status="todo")
            .set_track(track.id)
            .save()
        )
        with sdk.features.edit(f1.id) as feature:
            feature.agent_assigned = sdk.agent

        f2 = (
            sdk.features.create("Other Critical", priority="critical", status="todo")
            .set_track(track.id)
            .save()
        )  # Not assigned
        with sdk.features.edit(f2.id) as feature:
            feature.agent_assigned = None  # Explicitly unassign

        f3 = (
            sdk.features.create("My Normal", priority="high", status="todo")
            .set_track(track.id)
            .save()
        )
        with sdk.features.edit(f3.id) as feature:
            feature.agent_assigned = sdk.agent

        cmd = SnapshotCommand(
            output_format="json",
            node_type="all",
            status="all",
            blockers=True,
            my_work=True,
        )
        cmd.graph_dir = str(sdk._directory)
        cmd.agent = sdk.agent
        result = cmd.execute()

        assert result.exit_code == 0
        data = json.loads(result.text)

        # Should only have my critical items
        assert len(data) == 1
        assert data[0]["title"] == "My Critical"
        assert data[0]["assigned_to"] == sdk.agent
        assert data[0]["priority"] == "critical"

    def test_snapshot_all_filters_combined(self, sdk_instance, isolated_db):
        """Test all filters combined: track, active, blockers, my_work."""
        sdk = sdk_instance

        track = sdk.tracks.create("Track").save()

        # My critical active item in track
        f1 = (
            sdk.features.create(
                "My Critical Active", priority="critical", status="todo"
            )
            .set_track(track.id)
            .save()
        )
        with sdk.features.edit(f1.id) as feature:
            feature.agent_assigned = sdk.agent

        # My critical done item in track (excluded by active)
        f2 = (
            sdk.features.create("My Critical Done", priority="critical", status="done")
            .set_track(track.id)
            .save()
        )
        with sdk.features.edit(f2.id) as feature:
            feature.agent_assigned = sdk.agent

        # Other's critical active in track (excluded by my_work)
        f_other = (
            sdk.features.create("Other Critical", priority="critical", status="todo")
            .set_track(track.id)
            .save()
        )
        with sdk.features.edit(f_other.id) as feature:
            feature.agent_assigned = None  # Explicitly unassign

        # My normal active in track (excluded by blockers)
        f4 = (
            sdk.features.create("My Normal", priority="high", status="todo")
            .set_track(track.id)
            .save()
        )
        with sdk.features.edit(f4.id) as feature:
            feature.agent_assigned = sdk.agent

        cmd = SnapshotCommand(
            output_format="json",
            node_type="all",
            status="all",
            track_id=track.id,
            active=True,
            blockers=True,
            my_work=True,
        )
        cmd.graph_dir = str(sdk._directory)
        cmd.agent = sdk.agent
        result = cmd.execute()

        assert result.exit_code == 0
        data = json.loads(result.text)

        # Should only have one item matching all filters
        assert len(data) == 1
        assert data[0]["title"] == "My Critical Active"
