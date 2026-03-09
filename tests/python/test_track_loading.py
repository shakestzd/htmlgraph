"""
Test track loading including directory-based tracks.

Tests for bug-20251221-042515: Directory-based tracks not loading in graph loader
"""

import shutil
import tempfile
from pathlib import Path

import pytest
from htmlgraph.sdk import SDK


@pytest.fixture
def temp_graph_dir():
    """Create a temporary .htmlgraph directory."""
    tmpdir = tempfile.mkdtemp()
    graph_dir = Path(tmpdir) / ".htmlgraph"
    graph_dir.mkdir()

    # Create tracks directory
    tracks_dir = graph_dir / "tracks"
    tracks_dir.mkdir()

    yield graph_dir

    # Cleanup
    shutil.rmtree(tmpdir)


def test_load_directory_based_track(temp_graph_dir, isolated_db):
    """Test that directory-based tracks (track-xxx/index.html) are loaded correctly."""
    # Create a directory-based track
    track_dir = temp_graph_dir / "tracks" / "track-test-001"
    track_dir.mkdir()

    track_html = """<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Track: Test Track</title>
    <link rel="stylesheet" href="../../styles.css">
</head>
<body>
    <article id="track-test-001" data-type="track" data-status="active" data-priority="high">
        <header>
            <h1>Track: Test Track</h1>
            <div class="metadata">
                <span class="badge status-active">Active</span>
                <span class="badge priority-high">High Priority</span>
            </div>
        </header>

        <section data-description>
            <p>Test track description</p>
        </section>
    </article>
</body>
</html>"""

    (track_dir / "index.html").write_text(track_html)

    # Initialize SDK
    sdk = SDK(directory=temp_graph_dir, db_path=str(isolated_db), agent="test-agent")

    # Test that the track is loaded
    all_tracks = sdk.tracks.all()
    assert len(all_tracks) == 1, f"Expected 1 track, got {len(all_tracks)}"

    track = all_tracks[0]
    assert track.id == "track-test-001"
    assert track.title == "Track: Test Track"
    assert track.status == "active"
    assert track.priority == "high"


def test_load_file_based_track(temp_graph_dir, isolated_db):
    """Test that single-file tracks (track-xxx.html) still work."""
    # Create a file-based track (using valid Node status values)
    track_html = """<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Track: File Track</title>
    <link rel="stylesheet" href="../styles.css">
</head>
<body>
    <article id="track-file-001" data-type="track" data-status="todo" data-priority="medium">
        <header>
            <h1>Track: File Track</h1>
            <div class="metadata">
                <span class="badge status-todo">Todo</span>
                <span class="badge priority-medium">Medium Priority</span>
            </div>
        </header>

        <section data-description>
            <p>File-based track description</p>
        </section>
    </article>
</body>
</html>"""

    (temp_graph_dir / "tracks" / "track-file-001.html").write_text(track_html)

    # Initialize SDK
    sdk = SDK(directory=temp_graph_dir, db_path=str(isolated_db), agent="test-agent")

    # Test that the track is loaded
    all_tracks = sdk.tracks.all()
    assert len(all_tracks) == 1, f"Expected 1 track, got {len(all_tracks)}"

    track = all_tracks[0]
    assert track.id == "track-file-001"
    assert track.title == "Track: File Track"
    assert track.status == "todo"
    assert track.priority == "medium"


def test_load_mixed_tracks(temp_graph_dir, isolated_db):
    """Test that both file-based and directory-based tracks can coexist."""
    # Create directory-based track
    track_dir = temp_graph_dir / "tracks" / "track-dir-001"
    track_dir.mkdir()

    dir_track_html = """<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Track: Directory Track</title>
    <link rel="stylesheet" href="../../styles.css">
</head>
<body>
    <article id="track-dir-001" data-type="track" data-status="active" data-priority="high">
        <header>
            <h1>Track: Directory Track</h1>
        </header>
        <section data-description>
            <p>Directory track</p>
        </section>
    </article>
</body>
</html>"""

    (track_dir / "index.html").write_text(dir_track_html)

    # Create file-based track (using valid Node status values)
    file_track_html = """<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Track: File Track</title>
    <link rel="stylesheet" href="../styles.css">
</head>
<body>
    <article id="track-file-002" data-type="track" data-status="todo" data-priority="medium">
        <header>
            <h1>Track: File Track</h1>
        </header>
        <section data-description>
            <p>File track</p>
        </section>
    </article>
</body>
</html>"""

    (temp_graph_dir / "tracks" / "track-file-002.html").write_text(file_track_html)

    # Initialize SDK
    sdk = SDK(directory=temp_graph_dir, db_path=str(isolated_db), agent="test-agent")

    # Test that both tracks are loaded
    all_tracks = sdk.tracks.all()
    assert len(all_tracks) == 2, f"Expected 2 tracks, got {len(all_tracks)}"

    track_ids = {track.id for track in all_tracks}
    assert "track-dir-001" in track_ids
    assert "track-file-002" in track_ids


def test_track_where_query(temp_graph_dir, isolated_db):
    """Test that where() query works with directory-based tracks."""
    # Create multiple tracks
    track_dir1 = temp_graph_dir / "tracks" / "track-001"
    track_dir1.mkdir()

    track1_html = """<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Track 1</title>
    <link rel="stylesheet" href="../../styles.css">
</head>
<body>
    <article id="track-001" data-type="track" data-status="active" data-priority="high">
        <header><h1>Track 1</h1></header>
        <section data-description><p>High priority active track</p></section>
    </article>
</body>
</html>"""

    (track_dir1 / "index.html").write_text(track1_html)

    track_dir2 = temp_graph_dir / "tracks" / "track-002"
    track_dir2.mkdir()

    track2_html = """<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Track 2</title>
    <link rel="stylesheet" href="../../styles.css">
</head>
<body>
    <article id="track-002" data-type="track" data-status="todo" data-priority="low">
        <header><h1>Track 2</h1></header>
        <section data-description><p>Low priority todo track</p></section>
    </article>
</body>
</html>"""

    (track_dir2 / "index.html").write_text(track2_html)

    # Initialize SDK
    sdk = SDK(directory=temp_graph_dir, db_path=str(isolated_db), agent="test-agent")

    # Test where query
    active_tracks = sdk.tracks.where(status="active")
    assert len(active_tracks) == 1
    assert active_tracks[0].id == "track-001"

    high_priority = sdk.tracks.where(priority="high")
    assert len(high_priority) == 1
    assert high_priority[0].id == "track-001"

    todo_tracks = sdk.tracks.where(status="todo")
    assert len(todo_tracks) == 1
    assert todo_tracks[0].id == "track-002"


def test_track_get_by_id(temp_graph_dir, isolated_db):
    """Test that get() works with directory-based tracks."""
    # Create directory-based track
    track_dir = temp_graph_dir / "tracks" / "track-get-test"
    track_dir.mkdir()

    track_html = """<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Get Test Track</title>
    <link rel="stylesheet" href="../../styles.css">
</head>
<body>
    <article id="track-get-test" data-type="track" data-status="active" data-priority="medium">
        <header><h1>Get Test Track</h1></header>
        <section data-description><p>Test get method</p></section>
    </article>
</body>
</html>"""

    (track_dir / "index.html").write_text(track_html)

    # Initialize SDK
    sdk = SDK(directory=temp_graph_dir, db_path=str(isolated_db), agent="test-agent")

    # Test get
    track = sdk.tracks.get("track-get-test")
    assert track is not None
    assert track.id == "track-get-test"
    assert track.title == "Get Test Track"
    assert track.status == "active"


def test_track_edit_context_manager(temp_graph_dir, isolated_db):
    """Test that edit() context manager works for tracks."""
    # Create directory-based track
    track_dir = temp_graph_dir / "tracks" / "track-edit-test"
    track_dir.mkdir()

    track_html = """<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Edit Test Track</title>
    <link rel="stylesheet" href="../../styles.css">
</head>
<body>
    <article id="track-edit-test" data-type="track" data-status="active" data-priority="medium">
        <header><h1>Edit Test Track</h1></header>
        <section data-description><p>Test edit method</p></section>
    </article>
</body>
</html>"""

    (track_dir / "index.html").write_text(track_html)

    # Initialize SDK
    sdk = SDK(directory=temp_graph_dir, db_path=str(isolated_db), agent="test-agent")

    # Test edit context manager
    with sdk.tracks.edit("track-edit-test") as track:
        assert track.id == "track-edit-test"
        assert track.status == "active"
        assert track.priority == "medium"

        # Modify the track
        track.status = "completed"
        track.priority = "high"
        track.title = "Updated Track Title"

    # Verify changes were saved
    track = sdk.tracks.get("track-edit-test")
    assert track.status == "completed"
    assert track.priority == "high"
    assert track.title == "Updated Track Title"


def test_track_edit_nonexistent_track(temp_graph_dir, isolated_db):
    """Test that edit() raises NodeNotFoundError for nonexistent tracks."""
    from htmlgraph.exceptions import NodeNotFoundError

    # Initialize SDK
    sdk = SDK(directory=temp_graph_dir, db_path=str(isolated_db), agent="test-agent")

    # Try to edit a nonexistent track
    with pytest.raises(NodeNotFoundError) as exc_info:
        with sdk.tracks.edit("track-nonexistent"):
            pass

    # Verify exception details
    assert "track" in str(exc_info.value).lower()
    assert "track-nonexistent" in str(exc_info.value)


def test_track_edit_file_based(temp_graph_dir, isolated_db):
    """Test that edit() works with file-based tracks."""
    # Create file-based track
    track_html = """<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>File Track</title>
    <link rel="stylesheet" href="../styles.css">
</head>
<body>
    <article id="track-file-edit" data-type="track" data-status="todo" data-priority="low">
        <header><h1>File Track</h1></header>
        <section data-description><p>File-based track</p></section>
    </article>
</body>
</html>"""

    (temp_graph_dir / "tracks" / "track-file-edit.html").write_text(track_html)

    # Initialize SDK
    sdk = SDK(directory=temp_graph_dir, db_path=str(isolated_db), agent="test-agent")

    # Test edit on file-based track
    with sdk.tracks.edit("track-file-edit") as track:
        track.status = "in-progress"
        track.priority = "high"

    # Verify changes were saved
    track = sdk.tracks.get("track-file-edit")
    assert track.status == "in-progress"
    assert track.priority == "high"


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
