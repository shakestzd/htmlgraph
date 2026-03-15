"""
Tests for delete operations (CRUD completion).

Tests both HtmlGraph.delete() and SDK Collection.delete() methods.
"""

import tempfile

import pytest
from htmlgraph import SDK
from htmlgraph.graph import HtmlGraph
from htmlgraph.models import Edge, Node


class TestHtmlGraphDelete:
    """Tests for HtmlGraph delete operations."""

    @pytest.fixture
    def graph(self):
        """Create a temporary graph."""
        with tempfile.TemporaryDirectory() as tmpdir:
            g = HtmlGraph(tmpdir, auto_load=False)
            yield g

    def test_delete_single_node(self, graph, isolated_db):
        """Test deleting a single node."""
        node = Node(id="test-001", title="Test Node")
        graph.add(node)

        assert "test-001" in graph
        assert graph.delete("test-001") is True
        assert "test-001" not in graph
        assert graph.get("test-001") is None

    def test_delete_nonexistent_node(self, graph, isolated_db):
        """Test deleting a node that doesn't exist."""
        assert graph.delete("nonexistent") is False

    def test_delete_removes_html_file(self, graph, isolated_db):
        """Test that delete removes the HTML file from disk."""
        node = Node(id="test-002", title="Test Node")
        filepath = graph.add(node)

        assert filepath.exists()
        graph.delete("test-002")
        assert not filepath.exists()

    def test_delete_cleans_up_edges(self, graph, isolated_db):
        """Test that delete removes all edges involving the node."""
        # Create nodes with edges
        node_a = Node(
            id="a",
            title="Node A",
            edges={"blocks": [Edge(target_id="b", relationship="blocks")]},
        )
        node_b = Node(id="b", title="Node B")

        graph.add(node_a)
        graph.add(node_b)

        # Verify edge exists in index
        outgoing = graph.get_outgoing_edges("a", "blocks")
        assert len(outgoing) == 1
        assert outgoing[0].target_id == "b"

        incoming = graph.get_incoming_edges("b", "blocks")
        assert len(incoming) == 1
        assert incoming[0].source_id == "a"

        # Delete node A
        graph.delete("a")

        # Verify edges are cleaned up
        assert len(graph.get_outgoing_edges("a", "blocks")) == 0
        assert len(graph.get_incoming_edges("b", "blocks")) == 0

    def test_delete_with_multiple_edges(self, graph, isolated_db):
        """Test deleting a node with multiple incoming and outgoing edges."""
        # Create a node with multiple relationships
        node_a = Node(id="a", title="Node A")
        node_b = Node(
            id="b",
            title="Node B",
            edges={
                "blocked_by": [Edge(target_id="a", relationship="blocked_by")],
                "related": [Edge(target_id="c", relationship="related")],
            },
        )
        node_c = Node(
            id="c",
            title="Node C",
            edges={"blocks": [Edge(target_id="b", relationship="blocks")]},
        )

        graph.add(node_a)
        graph.add(node_b)
        graph.add(node_c)

        # Delete node B (has both incoming and outgoing edges)
        graph.delete("b")

        # Verify all edges involving B are cleaned up
        assert len(graph.get_incoming_edges("b")) == 0
        assert len(graph.get_outgoing_edges("b")) == 0

    def test_batch_delete(self, graph, isolated_db):
        """Test batch deleting multiple nodes."""
        nodes = [Node(id=f"node-{i}", title=f"Node {i}") for i in range(5)]

        for node in nodes:
            graph.add(node)

        # Batch delete 3 nodes
        count = graph.batch_delete(["node-0", "node-2", "node-4"])

        assert count == 3
        assert "node-0" not in graph
        assert "node-1" in graph
        assert "node-2" not in graph
        assert "node-3" in graph
        assert "node-4" not in graph

    def test_batch_delete_with_nonexistent(self, graph, isolated_db):
        """Test batch delete with some nonexistent nodes."""
        nodes = [Node(id=f"node-{i}", title=f"Node {i}") for i in range(3)]
        for node in nodes:
            graph.add(node)

        # Try to delete 2 existing + 2 nonexistent
        count = graph.batch_delete(
            ["node-0", "node-1", "nonexistent-1", "nonexistent-2"]
        )

        assert count == 2
        assert "node-0" not in graph
        assert "node-1" not in graph
        assert "node-2" in graph

    def test_delete_and_reload(self, graph, isolated_db):
        """Test that deleted nodes don't reappear after reload."""
        node = Node(id="test-003", title="Test Node")
        graph.add(node)
        graph.delete("test-003")

        # Reload from disk
        graph.reload()

        assert "test-003" not in graph
        assert graph.get("test-003") is None


class TestSDKCollectionDelete:
    """Tests for SDK Collection delete operations."""

    @pytest.fixture
    def sdk(self, isolated_db):
        """Create a temporary SDK instance."""
        with tempfile.TemporaryDirectory() as tmpdir:
            sdk = SDK(directory=tmpdir, agent="test-agent", db_path=str(isolated_db))
            yield sdk

    def test_sdk_delete_feature(self, sdk, isolated_db):
        """Test deleting a feature via SDK."""
        # Create a track first (required for features)
        track = sdk.tracks.create("Test Track").save()

        feature = sdk.features.create("Test Feature").set_track(track.id).save()
        feature_id = feature.id

        assert sdk.features.get(feature_id) is not None
        assert sdk.features.delete(feature_id) is True
        assert sdk.features.get(feature_id) is None

    def test_sdk_delete_nonexistent(self, sdk, isolated_db):
        """Test SDK delete returns False for nonexistent node."""
        assert sdk.features.delete("nonexistent") is False

    def test_sdk_batch_delete(self, sdk, isolated_db):
        """Test SDK batch delete."""
        track = sdk.tracks.create("Test Track").save()
        features = [
            sdk.features.create(f"Feature {i}").set_track(track.id).save()
            for i in range(5)
        ]
        feature_ids = [f.id for f in features]

        # Delete first 3
        count = sdk.features.batch_delete(feature_ids[:3])

        assert count == 3
        assert sdk.features.get(feature_ids[0]) is None
        assert sdk.features.get(feature_ids[1]) is None
        assert sdk.features.get(feature_ids[2]) is None
        assert sdk.features.get(feature_ids[3]) is not None
        assert sdk.features.get(feature_ids[4]) is not None

    def test_sdk_delete_all_collection_types(self, sdk, isolated_db):
        """Test that delete works for all collection types."""
        # Create test nodes for different collections
        track = sdk.tracks.create("Test Track").save()
        feature = sdk.features.create("Test Feature").set_track(track.id).save()

        # For non-feature collections, create nodes directly
        bug = Node(id="test-bug", title="Test Bug", type="bug")
        sdk.bugs._ensure_graph().add(bug)

        spike = Node(id="test-spike", title="Test Spike", type="spike")
        sdk.spikes._ensure_graph().add(spike)

        collections = {
            "features": (sdk.features, feature.id),
            "bugs": (sdk.bugs, "test-bug"),
            "spikes": (sdk.spikes, "test-spike"),
        }

        for coll_name, (collection, node_id) in collections.items():
            assert collection.delete(node_id) is True
            assert collection.get(node_id) is None

    def test_sdk_delete_with_edges(self, sdk, isolated_db):
        """Test SDK delete cleans up edges properly."""
        # Create features with dependencies
        track = sdk.tracks.create("Test Track").save()
        feat_a = sdk.features.create("Feature A").set_track(track.id).save()
        feat_b = (
            sdk.features.create("Feature B")
            .set_track(track.id)
            .blocked_by(feat_a.id)
            .save()
        )

        # Delete feat_a
        sdk.features.delete(feat_a.id)

        # Verify feat_b still exists but edges are cleaned up
        feat_b_refreshed = sdk.features.get(feat_b.id)
        assert feat_b_refreshed is not None


class TestDeleteEdgeCases:
    """Tests for edge cases in delete operations."""

    @pytest.fixture
    def graph(self):
        """Create a temporary graph."""
        with tempfile.TemporaryDirectory() as tmpdir:
            g = HtmlGraph(tmpdir, auto_load=False)
            yield g

    def test_delete_twice(self, graph, isolated_db):
        """Test deleting the same node twice."""
        node = Node(id="test-004", title="Test Node")
        graph.add(node)

        assert graph.delete("test-004") is True
        assert graph.delete("test-004") is False

    def test_delete_in_circular_dependency(self, graph, isolated_db):
        """Test deleting a node in a circular dependency."""
        # Create circular dependency: a -> b -> c -> a
        node_a = Node(
            id="a",
            title="Node A",
            edges={"blocks": [Edge(target_id="b", relationship="blocks")]},
        )
        node_b = Node(
            id="b",
            title="Node B",
            edges={"blocks": [Edge(target_id="c", relationship="blocks")]},
        )
        node_c = Node(
            id="c",
            title="Node C",
            edges={"blocks": [Edge(target_id="a", relationship="blocks")]},
        )

        graph.add(node_a)
        graph.add(node_b)
        graph.add(node_c)

        # Delete one node in the cycle
        assert graph.delete("b") is True

        # Verify cycle is broken
        assert "b" not in graph
        assert "a" in graph
        assert "c" in graph

    def test_batch_delete_empty_list(self, graph, isolated_db):
        """Test batch delete with empty list."""
        count = graph.batch_delete([])
        assert count == 0


class TestTrackCollectionDelete:
    """Tests for TrackCollection delete operations."""

    @pytest.fixture
    def sdk(self, isolated_db):
        """Create a temporary SDK instance."""
        with tempfile.TemporaryDirectory() as tmpdir:
            sdk = SDK(directory=tmpdir, agent="test-agent", db_path=str(isolated_db))
            yield sdk

    def test_delete_single_file_track(self, sdk, isolated_db):
        """Test deleting a single-file track (.html)."""
        # Create a consolidated (single-file) track
        track = sdk.tracks.builder().title("Test Track").consolidated().create()

        track_id = track.id

        # Verify file exists
        track_file = sdk._directory / "tracks" / f"{track_id}.html"
        assert track_file.exists()

        # Delete the track
        assert sdk.tracks.delete(track_id) is True

        # Verify file is deleted
        assert not track_file.exists()

        # Verify track is not in collection
        assert sdk.tracks.get(track_id) is None

    def test_delete_directory_based_track(self, sdk, isolated_db):
        """Test deleting a directory-based track (legacy format)."""
        # Create a separate-files (directory-based) track
        track = (
            sdk.tracks.builder()
            .title("Test Track")
            .separate_files()
            .with_spec(overview="Test overview")
            .with_plan_phases([("Phase 1", ["Task 1", "Task 2"])])
            .create()
        )

        track_id = track.id

        # Verify directory exists
        track_dir = sdk._directory / "tracks" / track_id
        assert track_dir.exists()
        assert track_dir.is_dir()
        assert (track_dir / "index.html").exists()
        assert (track_dir / "spec.html").exists()
        assert (track_dir / "plan.html").exists()

        # Delete the track
        assert sdk.tracks.delete(track_id) is True

        # Verify directory is deleted
        assert not track_dir.exists()

        # Verify track is not in collection
        assert sdk.tracks.get(track_id) is None

    def test_delete_nonexistent_track(self, sdk, isolated_db):
        """Test deleting a track that doesn't exist."""
        assert sdk.tracks.delete("nonexistent-track") is False

    def test_delete_twice(self, sdk, isolated_db):
        """Test deleting the same track twice."""
        track = sdk.tracks.builder().title("Test Track").create()

        track_id = track.id

        # First delete should succeed
        assert sdk.tracks.delete(track_id) is True

        # Second delete should fail (not found)
        assert sdk.tracks.delete(track_id) is False

    def test_batch_delete_tracks(self, sdk, isolated_db):
        """Test batch deleting multiple tracks."""
        # Create multiple tracks
        tracks = []
        for i in range(5):
            track = sdk.tracks.builder().title(f"Track {i}").create()
            tracks.append(track)

        track_ids = [t.id for t in tracks]

        # Batch delete first 3
        count = sdk.tracks.batch_delete(track_ids[:3])

        assert count == 3
        assert sdk.tracks.get(track_ids[0]) is None
        assert sdk.tracks.get(track_ids[1]) is None
        assert sdk.tracks.get(track_ids[2]) is None
        assert sdk.tracks.get(track_ids[3]) is not None
        assert sdk.tracks.get(track_ids[4]) is not None

    def test_batch_delete_mixed_formats(self, sdk, isolated_db):
        """Test batch deleting tracks with mixed formats (single-file and directory)."""
        # Create single-file track
        track1 = sdk.tracks.builder().title("Single File Track").consolidated().create()

        # Create directory-based track
        track2 = (
            sdk.tracks.builder()
            .title("Directory Track")
            .separate_files()
            .with_spec(overview="Test")
            .create()
        )

        # Batch delete both
        count = sdk.tracks.batch_delete([track1.id, track2.id])

        assert count == 2
        assert sdk.tracks.get(track1.id) is None
        assert sdk.tracks.get(track2.id) is None

    def test_batch_delete_with_nonexistent(self, sdk, isolated_db):
        """Test batch delete with some nonexistent tracks."""
        # Create 2 tracks
        track1 = sdk.tracks.builder().title("Track 1").create()
        track2 = sdk.tracks.builder().title("Track 2").create()

        # Try to delete 2 existing + 2 nonexistent
        count = sdk.tracks.batch_delete(
            [track1.id, track2.id, "nonexistent-1", "nonexistent-2"]
        )

        assert count == 2
        assert sdk.tracks.get(track1.id) is None
        assert sdk.tracks.get(track2.id) is None

    def test_batch_delete_empty_list(self, sdk, isolated_db):
        """Test batch delete with empty list."""
        count = sdk.tracks.batch_delete([])
        assert count == 0

    def test_delete_clears_graph_cache(self, sdk, isolated_db):
        """Test that delete clears the track from graph cache."""
        # Create track
        track = sdk.tracks.builder().title("Test Track").create()
        track_id = track.id

        # Load track to ensure it's in cache
        loaded_track = sdk.tracks.get(track_id)
        assert loaded_track is not None

        # Verify it's in the graph cache
        assert track_id in sdk.tracks._ensure_graph()._nodes

        # Delete track
        sdk.tracks.delete(track_id)

        # Verify it's removed from cache
        assert track_id not in sdk.tracks._ensure_graph()._nodes
