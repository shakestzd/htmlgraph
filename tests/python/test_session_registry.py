"""
Unit tests for SessionRegistry - Core file-based session tracking system.

Tests cover:
- Directory creation and initialization
- Instance ID generation (stable, unique)
- Session registration and reading
- Activity updates and heartbeats
- Session archival
- Index file management
- Error handling and edge cases
- Atomic file operations
"""

import json
import os
from datetime import datetime

import pytest
from htmlgraph.session_registry import SessionRegistry


class TestSessionRegistryInitialization:
    """Test registry initialization and directory creation."""

    def test_init_with_default_directory(self, tmp_path, monkeypatch):
        """Test initialization with default directory."""
        monkeypatch.chdir(tmp_path)

        registry = SessionRegistry()

        assert registry.registry_dir == tmp_path / ".htmlgraph/sessions/registry"
        assert registry.active_dir == tmp_path / ".htmlgraph/sessions/registry/active"
        assert registry.archive_dir == tmp_path / ".htmlgraph/sessions/registry/archive"
        assert (
            registry.index_file == tmp_path / ".htmlgraph/sessions/registry/.index.json"
        )

    def test_init_with_custom_directory(self, tmp_path):
        """Test initialization with custom directory."""
        custom_dir = tmp_path / "custom/registry"

        registry = SessionRegistry(registry_dir=custom_dir)

        assert registry.registry_dir == custom_dir
        assert registry.active_dir == custom_dir / "active"

    def test_init_creates_directories(self, tmp_path):
        """Test that initialization creates required directories."""
        registry_dir = tmp_path / "registry"

        registry = SessionRegistry(registry_dir=registry_dir)

        assert registry.registry_dir.exists()
        assert registry.active_dir.exists()
        assert registry.archive_dir.exists()

    def test_init_idempotent_with_existing_directories(self, tmp_path):
        """Test that initialization is idempotent."""
        registry_dir = tmp_path / "registry"
        registry_dir.mkdir(parents=True, exist_ok=True)

        # Should not raise
        registry = SessionRegistry(registry_dir=registry_dir)

        assert registry.registry_dir.exists()


class TestInstanceIdGeneration:
    """Test instance ID generation."""

    def test_get_instance_id_format(self):
        """Test that instance ID has correct format."""
        registry = SessionRegistry()
        instance_id = registry.get_instance_id()

        # Format: inst-{pid}-{hostname}-{timestamp}
        parts = instance_id.split("-")
        assert len(parts) >= 4
        assert parts[0] == "inst"
        assert parts[1].isdigit()  # PID is numeric

    def test_get_instance_id_stable(self):
        """Test that instance ID is stable for same process."""
        registry = SessionRegistry()

        id1 = registry.get_instance_id()
        id2 = registry.get_instance_id()

        assert id1 == id2

    def test_get_instance_id_contains_pid(self):
        """Test that instance ID includes current PID."""
        registry = SessionRegistry()
        instance_id = registry.get_instance_id()

        assert str(os.getpid()) in instance_id

    def test_get_instance_id_contains_hostname(self):
        """Test that instance ID includes hostname."""
        import socket

        registry = SessionRegistry()
        instance_id = registry.get_instance_id()

        assert socket.gethostname() in instance_id


class TestSessionRegistration:
    """Test session registration."""

    def test_register_session_creates_file(self, tmp_path):
        """Test that register_session creates registration file."""
        registry = SessionRegistry(registry_dir=tmp_path / "registry")

        repo_info = {
            "path": "/path/to/repo",
            "remote": "https://github.com/user/repo.git",
            "branch": "main",
            "commit": "abc123",
        }
        instance_info = {
            "pid": 12345,
            "hostname": "testhost",
            "start_time": "2026-01-08T12:34:56Z",
        }

        path = registry.register_session("sess-test123", repo_info, instance_info)

        assert path.exists()
        assert path.parent == registry.active_dir

    def test_register_session_writes_correct_content(self, tmp_path):
        """Test that register_session writes correct JSON content."""
        registry = SessionRegistry(registry_dir=tmp_path / "registry")

        repo_info = {
            "path": "/path/to/repo",
            "remote": "https://github.com/user/repo.git",
            "branch": "main",
            "commit": "abc123",
        }
        instance_info = {
            "pid": os.getpid(),
            "hostname": "testhost",
            "start_time": "2026-01-08T12:34:56Z",
        }

        path = registry.register_session("sess-test123", repo_info, instance_info)

        with open(path) as f:
            data = json.load(f)

        assert data["session_id"] == "sess-test123"
        assert data["status"] == "active"
        assert data["repo"] == repo_info
        assert data["instance"]["pid"] == instance_info["pid"]
        assert "created" in data
        assert "last_activity" in data

    def test_register_session_invalid_session_id(self, tmp_path):
        """Test that register_session rejects invalid session IDs."""
        registry = SessionRegistry(registry_dir=tmp_path / "registry")

        repo_info = {"path": "/path"}
        instance_info = {
            "pid": 12345,
            "hostname": "host",
            "start_time": "2026-01-08T12:34:56Z",
        }

        with pytest.raises(ValueError):
            registry.register_session("", repo_info, instance_info)

        with pytest.raises(ValueError):
            registry.register_session(None, repo_info, instance_info)

    def test_register_session_updates_index(self, tmp_path):
        """Test that register_session updates index file."""
        registry = SessionRegistry(registry_dir=tmp_path / "registry")

        repo_info = {"path": "/path/to/repo"}
        instance_info = {
            "pid": 12345,
            "hostname": "host",
            "start_time": "2026-01-08T12:34:56Z",
        }

        registry.register_session("sess-test123", repo_info, instance_info)

        assert registry.index_file.exists()

        with open(registry.index_file) as f:
            index = json.load(f)

        assert "sess-test123" in index["active_sessions"]
        assert index["active_sessions"]["sess-test123"]["instance_id"].startswith(
            "inst-"
        )


class TestSessionRead:
    """Test reading session registrations."""

    def test_read_session_existing(self, tmp_path):
        """Test reading existing session."""
        registry = SessionRegistry(registry_dir=tmp_path / "registry")

        repo_info = {"path": "/path/to/repo"}
        instance_info = {
            "pid": 12345,
            "hostname": "host",
            "start_time": "2026-01-08T12:34:56Z",
        }

        registry.register_session("sess-test123", repo_info, instance_info)
        instance_id = registry.get_instance_id()

        session = registry.read_session(instance_id)

        assert session is not None
        assert session["session_id"] == "sess-test123"
        assert session["status"] == "active"

    def test_read_session_not_found(self, tmp_path):
        """Test reading non-existent session."""
        registry = SessionRegistry(registry_dir=tmp_path / "registry")

        session = registry.read_session("inst-nonexistent-12345-67890")

        assert session is None

    def test_read_session_corrupt_file(self, tmp_path):
        """Test handling of corrupt JSON file."""
        registry = SessionRegistry(registry_dir=tmp_path / "registry")

        instance_id = "inst-corrupt-12345-67890"
        reg_file = registry.active_dir / f"{instance_id}.json"

        # Write corrupt JSON
        with open(reg_file, "w") as f:
            f.write("{ invalid json")

        session = registry.read_session(instance_id)

        assert session is None


class TestActivityUpdate:
    """Test activity/heartbeat updates."""

    @pytest.mark.skipif(
        os.environ.get("CI") == "true" or os.environ.get("GITHUB_ACTIONS") == "true",
        reason="Session registry instance ID timing flaky in CI",
    )
    def test_update_activity_success(self, tmp_path):
        """Test successful activity update."""
        registry = SessionRegistry(registry_dir=tmp_path / "registry")

        repo_info = {"path": "/path/to/repo"}
        instance_info = {
            "pid": 12345,
            "hostname": "host",
            "start_time": "2026-01-08T12:34:56Z",
        }

        registry.register_session("sess-test123", repo_info, instance_info)
        instance_id = registry.get_instance_id()

        session_before = registry.read_session(instance_id)
        original_activity = session_before["last_activity"]

        # Wait slightly to ensure timestamp changes (microseconds now matter)
        import time

        time.sleep(0.001)

        success = registry.update_activity(instance_id)

        assert success
        session_after = registry.read_session(instance_id)
        assert session_after["last_activity"] > original_activity

    def test_update_activity_not_found(self, tmp_path):
        """Test update_activity with non-existent session."""
        registry = SessionRegistry(registry_dir=tmp_path / "registry")

        success = registry.update_activity("inst-nonexistent-12345-67890")

        assert not success

    def test_update_activity_updates_index(self, tmp_path):
        """Test that update_activity updates index file."""
        registry = SessionRegistry(registry_dir=tmp_path / "registry")

        repo_info = {"path": "/path/to/repo"}
        instance_info = {
            "pid": 12345,
            "hostname": "host",
            "start_time": "2026-01-08T12:34:56Z",
        }

        registry.register_session("sess-test123", repo_info, instance_info)
        instance_id = registry.get_instance_id()

        import time

        time.sleep(0.01)
        registry.update_activity(instance_id)

        with open(registry.index_file) as f:
            index = json.load(f)

        index_activity = index["active_sessions"]["sess-test123"]["last_activity"]
        session_activity = registry.read_session(instance_id)["last_activity"]

        assert index_activity == session_activity


class TestSessionArchival:
    """Test session archival."""

    def test_archive_session_moves_file(self, tmp_path):
        """Test that archive_session moves file to archive."""
        registry = SessionRegistry(registry_dir=tmp_path / "registry")

        repo_info = {"path": "/path/to/repo"}
        instance_info = {
            "pid": 12345,
            "hostname": "host",
            "start_time": "2026-01-08T12:34:56Z",
        }

        registry.register_session("sess-test123", repo_info, instance_info)
        instance_id = registry.get_instance_id()

        active_file = registry.get_session_file_path(instance_id)
        assert active_file.exists()

        success = registry.archive_session(instance_id)

        assert success
        assert not active_file.exists()
        assert (registry.archive_dir / f"{instance_id}.json").exists()

    def test_archive_session_removes_from_index(self, tmp_path):
        """Test that archive_session removes session from index."""
        registry = SessionRegistry(registry_dir=tmp_path / "registry")

        repo_info = {"path": "/path/to/repo"}
        instance_info = {
            "pid": 12345,
            "hostname": "host",
            "start_time": "2026-01-08T12:34:56Z",
        }

        registry.register_session("sess-test123", repo_info, instance_info)
        instance_id = registry.get_instance_id()

        registry.archive_session(instance_id)

        with open(registry.index_file) as f:
            index = json.load(f)

        assert "sess-test123" not in index.get("active_sessions", {})

    def test_archive_session_not_found(self, tmp_path):
        """Test archive_session with non-existent session."""
        registry = SessionRegistry(registry_dir=tmp_path / "registry")

        success = registry.archive_session("inst-nonexistent-12345-67890")

        assert not success


class TestGetCurrentSessions:
    """Test getting all current sessions."""

    def test_get_current_sessions_empty(self, tmp_path):
        """Test get_current_sessions with no sessions."""
        registry = SessionRegistry(registry_dir=tmp_path / "registry")

        sessions = registry.get_current_sessions()

        assert sessions == []

    def test_get_current_sessions_multiple(self, tmp_path):
        """Test get_current_sessions with multiple sessions."""
        registry = SessionRegistry(registry_dir=tmp_path / "registry")

        repo_info = {"path": "/path/to/repo"}
        instance_info = {
            "pid": 12345,
            "hostname": "host",
            "start_time": "2026-01-08T12:34:56Z",
        }

        # Register two sessions
        registry.register_session("sess-test1", repo_info, instance_info)

        # Simulate second instance by writing another file directly
        instance_id_2 = "inst-54321-host-99999"
        session_data = {
            "instance_id": instance_id_2,
            "session_id": "sess-test2",
            "created": "2026-01-08T12:00:00Z",
            "repo": repo_info,
            "instance": instance_info,
            "status": "active",
            "last_activity": "2026-01-08T12:00:00Z",
        }
        reg_file = registry.active_dir / f"{instance_id_2}.json"
        with open(reg_file, "w") as f:
            json.dump(session_data, f)

        sessions = registry.get_current_sessions()

        assert len(sessions) == 2
        session_ids = {s["session_id"] for s in sessions}
        assert "sess-test1" in session_ids
        assert "sess-test2" in session_ids

    def test_get_current_sessions_skips_corrupt(self, tmp_path):
        """Test that get_current_sessions skips corrupt files."""
        registry = SessionRegistry(registry_dir=tmp_path / "registry")

        repo_info = {"path": "/path/to/repo"}
        instance_info = {
            "pid": 12345,
            "hostname": "host",
            "start_time": "2026-01-08T12:34:56Z",
        }

        registry.register_session("sess-valid", repo_info, instance_info)

        # Write corrupt file
        corrupt_file = registry.active_dir / "inst-corrupt-12345-67890.json"
        with open(corrupt_file, "w") as f:
            f.write("{ invalid")

        sessions = registry.get_current_sessions()

        # Should only get the valid session
        assert len(sessions) == 1
        assert sessions[0]["session_id"] == "sess-valid"


class TestGetSessionFilePath:
    """Test getting session file path."""

    def test_get_session_file_path_format(self, tmp_path):
        """Test get_session_file_path returns correct format."""
        registry = SessionRegistry(registry_dir=tmp_path / "registry")

        path = registry.get_session_file_path("inst-12345-host-99999")

        assert path == tmp_path / "registry/active/inst-12345-host-99999.json"

    def test_get_session_file_path_does_not_verify_existence(self, tmp_path):
        """Test that get_session_file_path doesn't verify file existence."""
        registry = SessionRegistry(registry_dir=tmp_path / "registry")

        path = registry.get_session_file_path("inst-nonexistent-12345-67890")

        # Should return path even if file doesn't exist
        assert path.name == "inst-nonexistent-12345-67890.json"
        assert not path.exists()


class TestAtomicWriteOperations:
    """Test atomic file write operations."""

    def test_write_atomic_creates_file(self, tmp_path):
        """Test that atomic write creates file."""
        test_file = tmp_path / "test.json"
        data = {"key": "value", "number": 42}

        SessionRegistry._write_atomic(test_file, data)

        assert test_file.exists()

        with open(test_file) as f:
            written = json.load(f)

        assert written == data

    def test_write_atomic_overwrites_existing(self, tmp_path):
        """Test that atomic write overwrites existing file."""
        test_file = tmp_path / "test.json"

        # Write initial content
        initial_data = {"old": "data"}
        SessionRegistry._write_atomic(test_file, initial_data)

        # Overwrite with new content
        new_data = {"new": "data"}
        SessionRegistry._write_atomic(test_file, new_data)

        with open(test_file) as f:
            written = json.load(f)

        assert written == new_data
        assert "old" not in written

    def test_write_atomic_no_temp_files_leaked(self, tmp_path):
        """Test that atomic write doesn't leave temp files on success."""
        test_file = tmp_path / "test.json"
        data = {"key": "value"}

        SessionRegistry._write_atomic(test_file, data)

        # Count files in directory
        files = list(tmp_path.glob("*.json"))
        assert len(files) == 1
        assert files[0].name == "test.json"

    def test_write_atomic_formats_json(self, tmp_path):
        """Test that atomic write formats JSON with indentation."""
        test_file = tmp_path / "test.json"
        data = {"key": "value", "nested": {"inner": "data"}}

        SessionRegistry._write_atomic(test_file, data)

        with open(test_file) as f:
            content = f.read()

        # Should be formatted (contain newlines/indentation)
        assert "\n" in content
        assert "  " in content  # Check for indentation


class TestTimestampGeneration:
    """Test UTC timestamp generation."""

    def test_get_utc_timestamp_format(self):
        """Test that UTC timestamp has correct format."""
        timestamp = SessionRegistry._get_utc_timestamp()

        # Format: YYYY-MM-DDTHH:MM:SS.ffffffZ
        assert isinstance(timestamp, str)
        assert "T" in timestamp
        assert timestamp.endswith("Z")
        assert "." in timestamp  # Has microseconds
        assert len(timestamp) == 27  # ISO 8601 format with microseconds

    def test_get_utc_timestamp_parses(self):
        """Test that timestamp can be parsed."""
        timestamp = SessionRegistry._get_utc_timestamp()

        # Should be parseable as datetime
        parsed = datetime.fromisoformat(timestamp.replace("Z", "+00:00"))
        assert isinstance(parsed, datetime)

    def test_get_utc_timestamp_increases(self):
        """Test that subsequent timestamps increase."""
        import time

        ts1 = SessionRegistry._get_utc_timestamp()
        time.sleep(0.001)
        ts2 = SessionRegistry._get_utc_timestamp()

        assert ts2 > ts1


class TestErrorHandling:
    """Test error handling and edge cases."""

    def test_permission_error_on_directory_creation(self, tmp_path):
        """Test handling of permission errors during directory creation."""
        read_only_dir = tmp_path / "readonly"
        read_only_dir.mkdir()

        # Make directory read-only
        os.chmod(read_only_dir, 0o444)

        try:
            with pytest.raises(OSError):
                SessionRegistry(registry_dir=read_only_dir / "registry")
        finally:
            # Restore permissions for cleanup
            os.chmod(read_only_dir, 0o755)

    def test_missing_active_directory_on_read(self, tmp_path):
        """Test reading from missing active directory."""
        registry = SessionRegistry(registry_dir=tmp_path / "registry")

        # Manually remove active directory
        registry.active_dir.rmdir()

        sessions = registry.get_current_sessions()

        assert sessions == []


class TestIntegration:
    """Integration tests for complete workflows."""

    def test_full_lifecycle_register_update_archive(self, tmp_path):
        """Test complete session lifecycle."""
        registry = SessionRegistry(registry_dir=tmp_path / "registry")

        # Register
        repo_info = {"path": "/path/to/repo", "branch": "main"}
        instance_info = {
            "pid": 12345,
            "hostname": "host",
            "start_time": "2026-01-08T12:34:56Z",
        }

        registry.register_session("sess-test", repo_info, instance_info)
        instance_id = registry.get_instance_id()

        # Verify registration
        sessions = registry.get_current_sessions()
        assert len(sessions) == 1

        # Update activity
        import time

        time.sleep(0.01)
        assert registry.update_activity(instance_id)

        # Archive
        assert registry.archive_session(instance_id)

        # Verify removed from active
        sessions = registry.get_current_sessions()
        assert len(sessions) == 0

        # Verify in archive
        archive_file = registry.archive_dir / f"{instance_id}.json"
        assert archive_file.exists()

    def test_concurrent_registration_scenario(self, tmp_path):
        """Test scenario with multiple concurrent registrations."""
        registry = SessionRegistry(registry_dir=tmp_path / "registry")

        repo_info = {"path": "/path/to/repo"}
        instance_info = {
            "pid": 12345,
            "hostname": "host",
            "start_time": "2026-01-08T12:34:56Z",
        }

        # Simulate multiple instances registering different sessions
        for i in range(5):
            # Write directly to simulate different instance IDs
            instance_id = f"inst-{1000 + i}-host-99999"
            session_data = {
                "instance_id": instance_id,
                "session_id": f"sess-test{i}",
                "created": "2026-01-08T12:00:00Z",
                "repo": repo_info,
                "instance": instance_info,
                "status": "active",
                "last_activity": "2026-01-08T12:00:00Z",
            }
            reg_file = registry.active_dir / f"{instance_id}.json"
            with open(reg_file, "w") as f:
                json.dump(session_data, f)

        sessions = registry.get_current_sessions()

        assert len(sessions) == 5
        session_ids = {s["session_id"] for s in sessions}
        assert len(session_ids) == 5
