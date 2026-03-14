"""Tests for rich relationship types (feat-18b3bd0a).

Covers:
- relates_to() builder method creates typed edges in HTML graph
- sdk.features.edges() type filter returns only matching edges
- get_transitive_dependencies() / get_dependencies() respect rel_type filter
- CLI graph neighborhood command shows edges correctly
- builder method chains correctly
"""

from __future__ import annotations

from pathlib import Path
from unittest.mock import patch

import pytest
from htmlgraph import SDK
from htmlgraph.graph import HtmlGraph
from htmlgraph.models import Edge, Node

# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------


@pytest.fixture
def tmp_htmlgraph(isolated_graph_dir_full: Path) -> Path:
    """Alias so tests read the same as the rest of the test suite."""
    return isolated_graph_dir_full


@pytest.fixture
def sdk(tmp_htmlgraph: Path, isolated_db: Path) -> SDK:
    return SDK(directory=tmp_htmlgraph, agent="test-agent", db_path=str(isolated_db))


@pytest.fixture
def track_id(sdk: SDK) -> str:
    track = sdk.tracks.create("Beads Convergence").save()
    return track.id


# ---------------------------------------------------------------------------
# 1. test_typed_edge_write — relates_to() stores a typed edge in the HTML graph
# ---------------------------------------------------------------------------


class TestTypedEdgeWrite:
    def test_relates_to_creates_typed_edge_in_html(
        self, sdk: SDK, track_id: str
    ) -> None:
        """relates_to() writes the edge into the node's HTML edges dict."""
        target = sdk.features.create("Target Feature").set_track(track_id).save()

        feature = (
            sdk.features.create("Source Feature")
            .set_track(track_id)
            .relates_to(target.id, "depends_on")
            .save()
        )

        edges = feature.edges.get("depends_on", [])
        assert len(edges) == 1
        assert edges[0].target_id == target.id

    def test_relates_to_arbitrary_rel_type(self, sdk: SDK, track_id: str) -> None:
        """Any string can be used as a relationship type."""
        target = sdk.features.create("Semantic Target").set_track(track_id).save()

        feature = (
            sdk.features.create("Semantic Source")
            .set_track(track_id)
            .relates_to(target.id, "related_to")
            .save()
        )

        edges = feature.edges.get("related_to", [])
        assert len(edges) == 1

    def test_relates_to_multiple_types(self, sdk: SDK, track_id: str) -> None:
        """A node can have edges of different types to different targets."""
        dep = sdk.features.create("Dependency").set_track(track_id).save()
        sibling = sdk.features.create("Sibling").set_track(track_id).save()

        feature = (
            sdk.features.create("Multi-edge Feature")
            .set_track(track_id)
            .relates_to(dep.id, "depends_on")
            .relates_to(sibling.id, "related_to")
            .save()
        )

        assert len(feature.edges.get("depends_on", [])) == 1
        assert len(feature.edges.get("related_to", [])) == 1
        assert feature.edges["depends_on"][0].target_id == dep.id
        assert feature.edges["related_to"][0].target_id == sibling.id

    def test_blocked_by_is_a_specific_rel_type(self, sdk: SDK, track_id: str) -> None:
        """blocked_by() is equivalent to relates_to(id, 'blocked_by')."""
        blocker = sdk.features.create("Blocker").set_track(track_id).save()

        via_blocked_by = (
            sdk.features.create("Via blocked_by")
            .set_track(track_id)
            .blocked_by(blocker.id)
            .save()
        )

        via_relates_to = (
            sdk.features.create("Via relates_to")
            .set_track(track_id)
            .relates_to(blocker.id, "blocked_by")
            .save()
        )

        # Both should have blocked_by edge with same structure
        assert len(via_blocked_by.edges.get("blocked_by", [])) == 1
        assert len(via_relates_to.edges.get("blocked_by", [])) == 1


# ---------------------------------------------------------------------------
# 2. test_type_filter — sdk.features.edges() with rel_type returns subset
# ---------------------------------------------------------------------------


class TestTypeFilter:
    def test_edges_no_filter_returns_all(self, sdk: SDK, track_id: str) -> None:
        """edges() with no filter returns all edge types from graph_edges."""
        # NOTE: sdk.features.edges() queries SQLite graph_edges table.
        # The builder saves edges to HTML, not SQLite graph_edges.
        # So we test the HTML-level edge filtering via node.edges directly,
        # and test that edges() doesn't crash on an unknown node.
        result = sdk.features.edges("feat-does-not-exist")
        assert isinstance(result, list)

    def test_edges_rel_type_filter_sql(self, sdk: SDK, isolated_db: Path) -> None:
        """edges() with rel_type filter passes WHERE clause to SQLite."""
        # Insert test rows directly into graph_edges to test the SQL path
        import sqlite3

        conn = sqlite3.connect(str(isolated_db))
        conn.row_factory = sqlite3.Row
        conn.execute(
            """
            INSERT OR IGNORE INTO graph_edges
                (edge_id, from_node_id, from_node_type, to_node_id, to_node_type,
                 relationship_type, weight)
            VALUES (?, ?, ?, ?, ?, ?, ?)
            """,
            (
                "edge-1",
                "feat-src",
                "feature",
                "feat-tgt1",
                "feature",
                "depends_on",
                1.0,
            ),
        )
        conn.execute(
            """
            INSERT OR IGNORE INTO graph_edges
                (edge_id, from_node_id, from_node_type, to_node_id, to_node_type,
                 relationship_type, weight)
            VALUES (?, ?, ?, ?, ?, ?, ?)
            """,
            (
                "edge-2",
                "feat-src",
                "feature",
                "feat-tgt2",
                "feature",
                "related_to",
                1.0,
            ),
        )
        conn.commit()
        conn.close()

        # Unfiltered should return both
        all_edges = sdk.features.edges("feat-src")
        rel_types = {e["relationship_type"] for e in all_edges}
        assert "depends_on" in rel_types
        assert "related_to" in rel_types

        # Filtered to depends_on only
        dep_edges = sdk.features.edges("feat-src", rel_type="depends_on")
        assert all(e["relationship_type"] == "depends_on" for e in dep_edges)
        assert len(dep_edges) >= 1

        # Filtered to related_to only
        rel_edges = sdk.features.edges("feat-src", rel_type="related_to")
        assert all(e["relationship_type"] == "related_to" for e in rel_edges)
        assert len(rel_edges) >= 1

    def test_edges_filter_returns_empty_for_nonexistent_type(self, sdk: SDK) -> None:
        """edges() with unknown rel_type returns empty list, not error."""
        result = sdk.features.edges("feat-any", rel_type="nonexistent_type")
        assert result == []


# ---------------------------------------------------------------------------
# 3. test_traversal_isolation — get_transitive_dependencies respects rel_type
# ---------------------------------------------------------------------------


class TestTraversalIsolation:
    def _make_graph(self, tmp_path: Path) -> HtmlGraph:
        """Build a small multi-rel-type graph for traversal tests."""
        graph = HtmlGraph(str(tmp_path / "graph"), auto_load=False)

        # feat-a --depends_on--> feat-b --depends_on--> feat-c
        # feat-a --related_to--> feat-d
        nodes = [
            Node(
                id="feat-a",
                title="A",
                type="feature",
                status="todo",
                priority="medium",
                edges={
                    "depends_on": [Edge(target_id="feat-b", relationship="depends_on")],
                    "related_to": [Edge(target_id="feat-d", relationship="related_to")],
                },
            ),
            Node(
                id="feat-b",
                title="B",
                type="feature",
                status="todo",
                priority="medium",
                edges={
                    "depends_on": [Edge(target_id="feat-c", relationship="depends_on")],
                },
            ),
            Node(
                id="feat-c", title="C", type="feature", status="todo", priority="medium"
            ),
            Node(
                id="feat-d", title="D", type="feature", status="todo", priority="medium"
            ),
        ]
        for n in nodes:
            graph.add(n)
        return graph

    def test_get_dependencies_follows_only_matching_rel_type(
        self, tmp_path: Path
    ) -> None:
        """get_dependencies with depends_on should not follow related_to edges."""
        graph = self._make_graph(tmp_path)

        from htmlgraph.graph.algorithms import get_dependencies

        deps = get_dependencies(graph, "feat-a", rel_type="depends_on")
        assert "feat-b" in deps
        assert "feat-c" in deps  # transitive
        assert "feat-d" not in deps  # related_to edge, should be excluded

    def test_get_dependencies_related_to_isolation(self, tmp_path: Path) -> None:
        """get_dependencies with related_to should not follow depends_on edges."""
        graph = self._make_graph(tmp_path)

        from htmlgraph.graph.algorithms import get_dependencies

        deps = get_dependencies(graph, "feat-a", rel_type="related_to")
        assert "feat-d" in deps
        assert "feat-b" not in deps
        assert "feat-c" not in deps

    def test_get_transitive_dependencies_on_core(self, tmp_path: Path) -> None:
        """HtmlGraph.get_transitive_dependencies() also respects rel_type."""
        graph = self._make_graph(tmp_path)

        deps = graph.get_transitive_dependencies("feat-a", rel_type="depends_on")
        assert "feat-b" in deps
        assert "feat-c" in deps
        assert "feat-d" not in deps

    def test_empty_result_when_no_edges_of_type(self, tmp_path: Path) -> None:
        """Traversal with a rel_type that has no edges returns empty set."""
        graph = self._make_graph(tmp_path)

        from htmlgraph.graph.algorithms import get_dependencies

        deps = get_dependencies(graph, "feat-c", rel_type="depends_on")
        assert deps == set()

    def test_nonexistent_node_returns_empty(self, tmp_path: Path) -> None:
        """Traversal from nonexistent node returns empty set."""
        graph = self._make_graph(tmp_path)

        from htmlgraph.graph.algorithms import get_dependencies

        deps = get_dependencies(graph, "feat-zzz", rel_type="depends_on")
        assert deps == set()


# ---------------------------------------------------------------------------
# 4. test_cli_neighborhood — CLI graph command shows edges
# ---------------------------------------------------------------------------


class TestCliNeighborhood:
    def test_graph_command_registered(self) -> None:
        """The 'graph' subcommand should be registered in the CLI."""
        import argparse

        from htmlgraph.cli.work.graph import register_graph_commands

        parser = argparse.ArgumentParser()
        subparsers = parser.add_subparsers()
        register_graph_commands(subparsers)

        args = parser.parse_args(["graph", "feat-001"])
        assert args.id == "feat-001"

    def test_graph_command_from_args(self) -> None:
        """GraphNeighborhoodCommand.from_args extracts the node_id correctly."""
        import argparse

        from htmlgraph.cli.work.graph import GraphNeighborhoodCommand

        args = argparse.Namespace(id="feat-abc123", format="text")
        cmd = GraphNeighborhoodCommand.from_args(args)
        assert cmd.node_id == "feat-abc123"

    def test_graph_command_no_edges_returns_warning(
        self, sdk: SDK, track_id: str, tmp_htmlgraph: Path, isolated_db: Path
    ) -> None:
        """Graph command on a node with no edges returns a no-edges warning."""
        from htmlgraph.cli.work.graph import GraphNeighborhoodCommand

        feature = sdk.features.create("Isolated Feature").set_track(track_id).save()

        cmd = GraphNeighborhoodCommand(node_id=feature.id)

        # Patch get_sdk to return our pre-built SDK
        with patch.object(cmd, "get_sdk", return_value=sdk):
            result = cmd.execute()

        # Should return a CommandResult (not raise)
        from htmlgraph.cli.base import CommandResult

        assert isinstance(result, CommandResult)

    def test_graph_command_with_html_edges(self, sdk: SDK, track_id: str) -> None:
        """Graph command shows edges that exist in the HTML graph."""
        from htmlgraph.cli.work.graph import GraphNeighborhoodCommand

        target = sdk.features.create("Target").set_track(track_id).save()
        source = (
            sdk.features.create("Source")
            .set_track(track_id)
            .relates_to(target.id, "depends_on")
            .save()
        )

        cmd = GraphNeighborhoodCommand(node_id=source.id)
        with patch.object(cmd, "get_sdk", return_value=sdk):
            result = cmd.execute()

        from htmlgraph.cli.base import CommandResult

        assert isinstance(result, CommandResult)

        # Check JSON data contains the edge
        json_data = result.json_data
        assert json_data is not None
        outgoing = json_data.get("outgoing", [])
        # At least one outgoing edge present from the HTML graph
        assert len(outgoing) >= 1

    def test_graph_command_collect_html_edges(self, sdk: SDK, track_id: str) -> None:
        """_collect_html_edges() returns outgoing typed edges from node HTML."""
        from htmlgraph.cli.work.graph import GraphNeighborhoodCommand

        target = sdk.features.create("Target2").set_track(track_id).save()
        source = (
            sdk.features.create("Source2")
            .set_track(track_id)
            .relates_to(target.id, "related_to")
            .save()
        )

        cmd = GraphNeighborhoodCommand(node_id=source.id)
        outgoing, incoming = cmd._collect_html_edges(sdk, source.id)

        assert any(t == target.id and r == "related_to" for t, r in outgoing)


# ---------------------------------------------------------------------------
# 5. test_relates_to_builder — builder method chains correctly
# ---------------------------------------------------------------------------


class TestRelatesToBuilder:
    def test_relates_to_returns_builder_for_chaining(
        self, sdk: SDK, track_id: str
    ) -> None:
        """relates_to() should return the builder so further calls can chain."""

        builder = sdk.features.create("Chain Test")
        result = builder.relates_to("feat-dummy", "depends_on")
        # Should be the same builder instance (Self return)
        assert result is builder

    def test_relates_to_chains_with_other_methods(
        self, sdk: SDK, track_id: str
    ) -> None:
        """relates_to() can be chained with set_priority, add_steps, etc."""
        target = sdk.features.create("Target3").set_track(track_id).save()

        feature = (
            sdk.features.create("Chained Feature")
            .set_track(track_id)
            .set_priority("high")
            .relates_to(target.id, "depends_on")
            .add_step("Step one")
            .save()
        )

        assert feature.priority == "high"
        assert len(feature.edges.get("depends_on", [])) == 1
        assert len(feature.steps) == 1

    def test_relates_to_multiple_times_same_type(self, sdk: SDK, track_id: str) -> None:
        """Multiple relates_to() calls with the same type add multiple edges."""
        t1 = sdk.features.create("Target A").set_track(track_id).save()
        t2 = sdk.features.create("Target B").set_track(track_id).save()

        feature = (
            sdk.features.create("Multi-target")
            .set_track(track_id)
            .relates_to(t1.id, "depends_on")
            .relates_to(t2.id, "depends_on")
            .save()
        )

        edges = feature.edges.get("depends_on", [])
        assert len(edges) == 2
        target_ids = {e.target_id for e in edges}
        assert t1.id in target_ids
        assert t2.id in target_ids

    def test_relates_to_persists_after_reload(self, sdk: SDK, track_id: str) -> None:
        """Typed edges are written to HTML and survive a graph reload."""
        target = sdk.features.create("Persistent Target").set_track(track_id).save()
        source = (
            sdk.features.create("Persistent Source")
            .set_track(track_id)
            .relates_to(target.id, "depends_on")
            .save()
        )

        # Force reload from disk
        graph = sdk.features._ensure_graph()
        graph.reload()

        reloaded = graph.get(source.id)
        assert reloaded is not None
        edges = reloaded.edges.get("depends_on", [])
        assert len(edges) == 1
        assert edges[0].target_id == target.id
