"""
Tests for RefManager and SDK ref integration.

Tests short ref generation, resolution, persistence, and SDK integration.
"""

import json
import tempfile
from pathlib import Path

import pytest
from htmlgraph.refs import RefManager
from htmlgraph.sdk import SDK


@pytest.fixture
def temp_graph_dir():
    """Create a temporary .htmlgraph directory for testing."""
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
def ref_manager(temp_graph_dir):
    """Create a RefManager instance for testing."""
    return RefManager(temp_graph_dir)


class TestRefManager:
    """Tests for RefManager class."""

    def test_ref_generation(self, ref_manager, isolated_db):
        """Test generating refs for different node types."""
        # Generate refs for different types
        feature_ref = ref_manager.generate_ref("feat-abc123")
        track_ref = ref_manager.generate_ref("trk-def456")
        bug_ref = ref_manager.generate_ref("bug-ghi789")
        spike_ref = ref_manager.generate_ref("spk-jkl012")
        chore_ref = ref_manager.generate_ref("chr-mno345")
        epic_ref = ref_manager.generate_ref("epc-pqr678")
        todo_ref = ref_manager.generate_ref("todo-stu901")
        phase_ref = ref_manager.generate_ref("phs-vwx234")

        # Verify format
        assert feature_ref == "@f1"
        assert track_ref == "@t1"
        assert bug_ref == "@b1"
        assert spike_ref == "@s1"
        assert chore_ref == "@c1"
        assert epic_ref == "@e1"
        assert todo_ref == "@d1"
        assert phase_ref == "@p1"

        # Generate more features - should increment
        feature2_ref = ref_manager.generate_ref("feat-xyz789")
        assert feature2_ref == "@f2"

    def test_ref_resolution(self, ref_manager, isolated_db):
        """Test resolving short refs back to full IDs."""
        # Generate refs
        feature_id = "feat-abc123"
        feature_ref = ref_manager.generate_ref(feature_id)

        # Resolve back
        resolved_id = ref_manager.resolve_ref(feature_ref)
        assert resolved_id == feature_id

        # Non-existent ref
        assert ref_manager.resolve_ref("@f999") is None

    def test_ref_persistence(self, ref_manager, temp_graph_dir, isolated_db):
        """Test that refs persist across RefManager reloads."""
        # Generate refs
        feature_id = "feat-abc123"
        feature_ref = ref_manager.generate_ref(feature_id)
        track_id = "trk-def456"
        track_ref = ref_manager.generate_ref(track_id)

        # Verify refs.json was created
        refs_file = temp_graph_dir / "refs.json"
        assert refs_file.exists()

        # Load refs file and verify content
        with open(refs_file, encoding="utf-8") as f:
            data = json.load(f)
            assert "@f1" in data["refs"]
            assert data["refs"]["@f1"] == feature_id
            assert "@t1" in data["refs"]
            assert data["refs"]["@t1"] == track_id

        # Create new RefManager instance (reload)
        new_ref_manager = RefManager(temp_graph_dir)

        # Verify refs still resolve
        assert new_ref_manager.resolve_ref(feature_ref) == feature_id
        assert new_ref_manager.resolve_ref(track_ref) == track_id

    def test_ref_idempotency(self, ref_manager, isolated_db):
        """Test that getting same ref twice returns same value."""
        feature_id = "feat-abc123"

        # Generate ref twice
        ref1 = ref_manager.generate_ref(feature_id)
        ref2 = ref_manager.generate_ref(feature_id)

        assert ref1 == ref2
        assert ref1 == "@f1"

        # get_ref should also return same ref
        ref3 = ref_manager.get_ref(feature_id)
        assert ref3 == ref1

    def test_get_refs_by_type(self, ref_manager, isolated_db):
        """Test filtering refs by node type."""
        # Generate multiple features
        ref_manager.generate_ref("feat-abc123")
        ref_manager.generate_ref("feat-def456")
        ref_manager.generate_ref("feat-ghi789")

        # Generate other types
        ref_manager.generate_ref("bug-xyz123")
        ref_manager.generate_ref("trk-uvw456")

        # Get feature refs
        feature_refs = ref_manager.get_refs_by_type("feature")
        assert len(feature_refs) == 3
        assert feature_refs[0] == ("@f1", "feat-abc123")
        assert feature_refs[1] == ("@f2", "feat-def456")
        assert feature_refs[2] == ("@f3", "feat-ghi789")

        # Get bug refs
        bug_refs = ref_manager.get_refs_by_type("bug")
        assert len(bug_refs) == 1
        assert bug_refs[0] == ("@b1", "bug-xyz123")

        # Get non-existent type
        unknown_refs = ref_manager.get_refs_by_type("unknown")
        assert len(unknown_refs) == 0

    def test_rebuild_refs(self, ref_manager, temp_graph_dir, isolated_db):
        """Test rebuilding refs from filesystem."""
        # Create some HTML files in collections
        features_dir = temp_graph_dir / "features"
        (features_dir / "feat-abc123.html").write_text("<html></html>")
        (features_dir / "feat-def456.html").write_text("<html></html>")

        bugs_dir = temp_graph_dir / "bugs"
        (bugs_dir / "bug-xyz789.html").write_text("<html></html>")

        # Rebuild refs
        ref_manager.rebuild_refs()

        # Verify refs were generated
        refs = ref_manager.get_all_refs()
        assert "@f1" in refs
        assert "@f2" in refs
        assert "@b1" in refs

        # Verify they resolve correctly
        assert ref_manager.resolve_ref("@f1") == "feat-abc123"
        assert ref_manager.resolve_ref("@f2") == "feat-def456"
        assert ref_manager.resolve_ref("@b1") == "bug-xyz789"

    def test_rebuild_refs_preserves_existing(
        self, ref_manager, temp_graph_dir, isolated_db
    ):
        """Test that rebuild preserves existing refs when possible."""
        # Generate initial refs
        feature_id = "feat-abc123"
        original_ref = ref_manager.generate_ref(feature_id)
        assert original_ref == "@f1"

        # Create HTML file for the existing ref
        features_dir = temp_graph_dir / "features"
        (features_dir / f"{feature_id}.html").write_text("<html></html>")

        # Rebuild refs (should find the one HTML file)
        ref_manager.rebuild_refs()

        # Original ref should be preserved
        assert ref_manager.resolve_ref("@f1") == feature_id

        # Verify it still works
        assert ref_manager.get_ref(feature_id) == "@f1"

    def test_invalid_node_id(self, ref_manager, isolated_db):
        """Test handling of invalid node IDs."""
        # No hyphen
        with pytest.raises(ValueError):
            ref_manager.generate_ref("invalid")

        # Unknown prefix
        with pytest.raises(ValueError):
            ref_manager.generate_ref("xxx-abc123")

        # get_ref returns None for invalid
        assert ref_manager.get_ref("invalid") is None
        assert ref_manager.get_ref("xxx-abc123") is None

    def test_corrupted_refs_file(self, temp_graph_dir, isolated_db):
        """Test handling of corrupted refs.json file."""
        # Write corrupted JSON
        refs_file = temp_graph_dir / "refs.json"
        refs_file.write_text("{invalid json")

        # Should start fresh without crashing
        ref_manager = RefManager(temp_graph_dir)
        assert len(ref_manager.get_all_refs()) == 0

        # Should be able to generate new refs
        ref = ref_manager.generate_ref("feat-abc123")
        assert ref == "@f1"


class TestSDKIntegration:
    """Tests for SDK integration with RefManager."""

    def test_sdk_initializes_ref_manager(self, temp_graph_dir, isolated_db):
        """Test that SDK initializes RefManager."""
        sdk = SDK(
            agent="test-agent", directory=temp_graph_dir, db_path=str(isolated_db)
        )
        assert hasattr(sdk, "refs")
        assert isinstance(sdk.refs, RefManager)

    def test_sdk_ref_method(self, temp_graph_dir, isolated_db):
        """Test sdk.ref() method resolves refs to nodes."""
        sdk = SDK(
            agent="test-agent", directory=temp_graph_dir, db_path=str(isolated_db)
        )

        # Create a track first
        track = sdk.tracks.create("Test Track").save()

        # Create a feature
        feature = sdk.features.create("Test Feature").set_track(track.id).save()

        # Get ref for the feature
        ref = sdk.refs.get_ref(feature.id)
        assert ref is not None
        assert ref.startswith("@f")

        # Resolve ref back to node
        resolved_node = sdk.ref(ref)
        assert resolved_node is not None
        assert resolved_node.id == feature.id
        assert resolved_node.title == "Test Feature"

    def test_sdk_ref_method_multiple_types(self, temp_graph_dir, isolated_db):
        """Test sdk.ref() with multiple node types."""
        sdk = SDK(
            agent="test-agent", directory=temp_graph_dir, db_path=str(isolated_db)
        )

        # Create a track first
        track = sdk.tracks.create("Test Track").save()

        # Create different types
        feature = sdk.features.create("Test Feature").set_track(track.id).save()
        bug = sdk.bugs.create("Test Bug").set_track(track.id).save()
        spike = sdk.spikes.create("Test Spike").set_track(track.id).save()

        # Get refs
        feature_ref = sdk.refs.get_ref(feature.id)
        bug_ref = sdk.refs.get_ref(bug.id)
        spike_ref = sdk.refs.get_ref(spike.id)

        # Resolve all refs
        assert sdk.ref(feature_ref).id == feature.id
        assert sdk.ref(bug_ref).id == bug.id
        assert sdk.ref(spike_ref).id == spike.id

    def test_sdk_ref_nonexistent(self, temp_graph_dir, isolated_db):
        """Test sdk.ref() with non-existent ref."""
        sdk = SDK(
            agent="test-agent", directory=temp_graph_dir, db_path=str(isolated_db)
        )

        # Non-existent ref
        result = sdk.ref("@f999")
        assert result is None

        # Invalid format
        result = sdk.ref("invalid")
        assert result is None

    def test_collection_get_ref(self, temp_graph_dir, isolated_db):
        """Test collection.get_ref() convenience method."""
        sdk = SDK(
            agent="test-agent", directory=temp_graph_dir, db_path=str(isolated_db)
        )

        # Create a track first
        track = sdk.tracks.create("Test Track").save()

        # Create a feature
        feature = sdk.features.create("Test Feature").set_track(track.id).save()

        # Get ref via collection
        ref = sdk.features.get_ref(feature.id)
        assert ref is not None
        assert ref.startswith("@f")

        # Verify it matches SDK ref manager
        sdk_ref = sdk.refs.get_ref(feature.id)
        assert ref == sdk_ref

    def test_ref_persistence_across_sdk_instances(self, temp_graph_dir, isolated_db):
        """Test refs persist across SDK reloads."""
        # Create SDK and generate refs
        sdk1 = SDK(
            agent="test-agent", directory=temp_graph_dir, db_path=str(isolated_db)
        )
        track = sdk1.tracks.create("Test Track").save()
        feature = sdk1.features.create("Test Feature").set_track(track.id).save()
        ref = sdk1.refs.get_ref(feature.id)

        # Create new SDK instance
        sdk2 = SDK(
            agent="test-agent", directory=temp_graph_dir, db_path=str(isolated_db)
        )

        # Verify ref still resolves
        resolved = sdk2.ref(ref)
        assert resolved is not None
        assert resolved.id == feature.id

    def test_ref_manager_set_on_all_collections(self, temp_graph_dir, isolated_db):
        """Test that all collections have ref_manager set."""
        sdk = SDK(
            agent="test-agent", directory=temp_graph_dir, db_path=str(isolated_db)
        )

        collections = [
            sdk.features,
            sdk.bugs,
            sdk.spikes,
            sdk.tracks,
        ]

        for collection in collections:
            assert collection._ref_manager is not None
            assert collection._ref_manager is sdk.refs

    def test_ref_generation_on_create(self, temp_graph_dir, isolated_db):
        """Test that refs are auto-generated when accessing nodes."""
        sdk = SDK(
            agent="test-agent", directory=temp_graph_dir, db_path=str(isolated_db)
        )

        # Create a track first
        track = sdk.tracks.create("Test Track").save()

        # Create features
        f1 = sdk.features.create("Feature 1").set_track(track.id).save()
        f2 = sdk.features.create("Feature 2").set_track(track.id).save()
        f3 = sdk.features.create("Feature 3").set_track(track.id).save()

        # Get refs - should auto-generate
        ref1 = sdk.refs.get_ref(f1.id)
        ref2 = sdk.refs.get_ref(f2.id)
        ref3 = sdk.refs.get_ref(f3.id)

        # Verify sequential numbering
        assert ref1 == "@f1"
        assert ref2 == "@f2"
        assert ref3 == "@f3"
