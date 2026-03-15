"""Test that test fixtures provide proper isolation."""

from pathlib import Path

from htmlgraph import SDK


def test_isolated_db_provides_unique_path(isolated_db: Path) -> None:
    """Verify isolated_db provides unique database path."""
    assert isolated_db.suffix == ".db"
    # Database is in a temporary directory (pytest tmp_path)
    assert "pytest" in str(isolated_db) or "tmp" in str(isolated_db)
    assert not isolated_db.exists()  # Not created until SDK uses it


def test_isolated_graph_dir_creates_directory(isolated_graph_dir: Path) -> None:
    """Verify isolated_graph_dir creates .htmlgraph directory."""
    assert isolated_graph_dir.exists()
    assert isolated_graph_dir.is_dir()
    assert isolated_graph_dir.name == ".htmlgraph"


def test_isolated_graph_dir_full_creates_all_subdirs(
    isolated_graph_dir_full: Path,
) -> None:
    """Verify isolated_graph_dir_full creates all subdirectories."""
    expected_dirs = [
        # Work item types (DEFAULT_COLLECTIONS)
        "features",
        "bugs",
        "chores",
        "spikes",
        "epics",
        "tracks",
        "sessions",
        "insights",
        "metrics",
        "cigs",
        "patterns",
        "todos",
        "task-delegations",
        # Additional directories (ADDITIONAL_DIRECTORIES)
        "events",
        "logs",
        "archive-index",
        "archives",
    ]

    for subdir in expected_dirs:
        subdir_path = isolated_graph_dir_full / subdir
        assert subdir_path.exists(), f"Missing subdirectory: {subdir}"
        assert subdir_path.is_dir(), f"Not a directory: {subdir}"


def test_isolated_sdk_provides_working_sdk(isolated_sdk: SDK) -> None:
    """Verify isolated_sdk provides functional SDK instance."""
    assert isinstance(isolated_sdk, SDK)
    assert isolated_sdk._agent_id == "test-agent"

    # Verify database is isolated
    db_path = Path(isolated_sdk._db.db_path)
    # Database is in a temporary directory (pytest tmp_path)
    assert "pytest" in str(db_path) or "tmp" in str(db_path)
    assert db_path != Path.home() / ".htmlgraph" / "htmlgraph.db"


def test_isolated_sdk_minimal_provides_minimal_structure(
    isolated_sdk_minimal: SDK,
) -> None:
    """Verify isolated_sdk_minimal uses minimal structure."""
    assert isinstance(isolated_sdk_minimal, SDK)

    # Directory exists but subdirs created on demand
    graph_dir = isolated_sdk_minimal._directory
    assert graph_dir.exists()


def test_multiple_isolated_sdks_are_independent(
    tmp_path: Path,
) -> None:
    """Verify multiple SDK instances can coexist with separate databases."""
    # Create separate directories AND databases for each SDK
    graph_dir1 = tmp_path / "graph1" / ".htmlgraph"
    graph_dir1.mkdir(parents=True)
    db1 = tmp_path / "graph1" / "test.db"

    graph_dir2 = tmp_path / "graph2" / ".htmlgraph"
    graph_dir2.mkdir(parents=True)
    db2 = tmp_path / "graph2" / "test.db"

    sdk1 = SDK(directory=graph_dir1, agent="agent1", db_path=str(db1))
    sdk2 = SDK(directory=graph_dir2, agent="agent2", db_path=str(db2))

    # Create tracks in each (required for features)
    track1 = sdk1.tracks.create("Track 1").save()
    track2 = sdk2.tracks.create("Track 2").save()

    # Create features in each
    sdk1.features.create("Feature 1").set_track(track1.id).save()
    sdk2.features.create("Feature 2").set_track(track2.id).save()

    # Verify isolation - each SDK only sees its own features
    assert len(sdk1.features.all()) == 1
    assert len(sdk2.features.all()) == 1

    sdk1._db.disconnect()
    sdk2._db.disconnect()


def test_isolated_sdk_can_create_and_retrieve_features(isolated_sdk: SDK) -> None:
    """Verify isolated_sdk can perform basic SDK operations."""
    # Create track first (required for features)
    track = isolated_sdk.tracks.create("Test Track").save()

    # Create feature
    feature = isolated_sdk.features.create("Test Feature").set_track(track.id).save()

    assert feature.id
    assert feature.title == "Test Feature"

    # Retrieve feature
    retrieved = isolated_sdk.features.get(feature.id)
    assert retrieved.id == feature.id
    assert retrieved.title == "Test Feature"


def test_isolated_sdk_can_work_with_tracks(isolated_sdk: SDK) -> None:
    """Verify isolated_sdk can work with different collections."""
    # Create track
    track = isolated_sdk.tracks.create("Test Track").save()

    assert track.id
    # Track title includes "Track:" prefix
    assert "Test Track" in track.title

    # Retrieve track
    retrieved = isolated_sdk.tracks.get(track.id)
    assert retrieved.id == track.id
    assert "Test Track" in retrieved.title

    # Verify features and tracks are separate
    assert len(isolated_sdk.features.all()) == 0
    assert len(isolated_sdk.tracks.all()) == 1


def test_isolated_sdk_persists_data_across_operations(isolated_sdk: SDK) -> None:
    """Verify data persists in isolated database across operations."""
    # Create track first (required for features)
    track = isolated_sdk.tracks.create("Test Track").save()

    # Create multiple features
    f1 = isolated_sdk.features.create("Feature 1").set_track(track.id).save()
    f2 = isolated_sdk.features.create("Feature 2").set_track(track.id).save()
    f3 = isolated_sdk.features.create("Feature 3").set_track(track.id).save()

    # Verify all are retrieved
    all_features = isolated_sdk.features.all()
    assert len(all_features) == 3
    assert {f.id for f in all_features} == {f1.id, f2.id, f3.id}
