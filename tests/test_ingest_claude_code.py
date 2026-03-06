from __future__ import annotations

"""
Tests for Claude Code native session ingester.
"""

import json
from pathlib import Path

from htmlgraph.ingest.claude_code import ClaudeCodeIngester, IngestResult, IngestSummary

# ---------------------------------------------------------------------------
# Sample JSONL fixture data
# ---------------------------------------------------------------------------

SAMPLE_SESSION_ID = "abc12345-1234-5678-abcd-abcdef123456"

SAMPLE_JSONL_LINES: list[dict] = [
    {
        "parentUuid": None,
        "isSidechain": False,
        "cwd": "/home/user/projects/myapp",
        "sessionId": SAMPLE_SESSION_ID,
        "version": "2.1.0",
        "gitBranch": "main",
        "type": "user",
        "message": {"role": "user", "content": "Add a login function"},
        "uuid": "uuid-user-001",
        "timestamp": "2026-01-15T10:00:00.000Z",
    },
    {
        "parentUuid": "uuid-user-001",
        "isSidechain": False,
        "cwd": "/home/user/projects/myapp",
        "sessionId": SAMPLE_SESSION_ID,
        "version": "2.1.0",
        "gitBranch": "main",
        "type": "assistant",
        "message": {
            "role": "assistant",
            "content": [
                {"type": "text", "text": "I'll add a login function."},
                {
                    "type": "tool_use",
                    "id": "toolu_001",
                    "name": "Bash",
                    "input": {"command": "ls src/"},
                },
            ],
        },
        "uuid": "uuid-asst-001",
        "timestamp": "2026-01-15T10:00:05.000Z",
    },
    {
        "parentUuid": "uuid-asst-001",
        "isSidechain": False,
        "cwd": "/home/user/projects/myapp",
        "sessionId": SAMPLE_SESSION_ID,
        "version": "2.1.0",
        "gitBranch": "main",
        "type": "tool_result",
        "message": {
            "role": "user",
            "content": [
                {
                    "type": "tool_result",
                    "tool_use_id": "toolu_001",
                    "content": "auth.py\nmain.py",
                }
            ],
        },
        "uuid": "uuid-res-001",
        "timestamp": "2026-01-15T10:00:10.000Z",
    },
    {
        "parentUuid": "uuid-res-001",
        "isSidechain": False,
        "cwd": "/home/user/projects/myapp",
        "sessionId": SAMPLE_SESSION_ID,
        "version": "2.1.0",
        "gitBranch": "main",
        "type": "assistant",
        "message": {
            "role": "assistant",
            "content": [
                {"type": "text", "text": "Done! I've added the login function."},
            ],
        },
        "uuid": "uuid-asst-002",
        "timestamp": "2026-01-15T10:00:20.000Z",
    },
]


def write_sample_jsonl(directory: Path, session_id: str = SAMPLE_SESSION_ID) -> Path:
    """Write sample JSONL lines to a file in directory."""
    jsonl_path = directory / f"{session_id}.jsonl"
    with jsonl_path.open("w") as f:
        for line in SAMPLE_JSONL_LINES:
            f.write(json.dumps(line) + "\n")
    return jsonl_path


# ---------------------------------------------------------------------------
# Tests for ClaudeCodeIngester.ingest_file
# ---------------------------------------------------------------------------


class TestIngestFile:
    def test_ingest_creates_session_html(self, tmp_path: Path) -> None:
        """Ingesting a JSONL file should create a session HTML file."""
        source_dir = tmp_path / "claude_sessions"
        source_dir.mkdir()
        graph_dir = tmp_path / ".htmlgraph"
        graph_dir.mkdir()

        jsonl = write_sample_jsonl(source_dir)
        ingester = ClaudeCodeIngester(graph_dir=graph_dir)
        result = ingester.ingest_file(jsonl)

        assert result.success
        assert result.session_id == SAMPLE_SESSION_ID
        assert result.output_path is not None
        assert result.output_path.exists(), f"Expected HTML at {result.output_path}"
        assert result.imported > 0

    def test_ingest_links_transcript_id(self, tmp_path: Path) -> None:
        """The created session should have transcript_id set to the source session ID."""
        source_dir = tmp_path / "claude_sessions"
        source_dir.mkdir()
        graph_dir = tmp_path / ".htmlgraph"
        graph_dir.mkdir()

        jsonl = write_sample_jsonl(source_dir)
        ingester = ClaudeCodeIngester(graph_dir=graph_dir)
        result = ingester.ingest_file(jsonl)

        # Load the session and verify
        from htmlgraph.converter import html_to_session

        session = html_to_session(result.output_path)  # type: ignore[arg-type]
        assert session.transcript_id == SAMPLE_SESSION_ID
        assert session.transcript_git_branch == "main"
        assert session.status == "ended"

    def test_ingest_counts_user_and_tool_entries(self, tmp_path: Path) -> None:
        """Only user messages and tool_use entries should be imported as events."""
        source_dir = tmp_path / "claude_sessions"
        source_dir.mkdir()
        graph_dir = tmp_path / ".htmlgraph"
        graph_dir.mkdir()

        jsonl = write_sample_jsonl(source_dir)
        ingester = ClaudeCodeIngester(graph_dir=graph_dir)
        result = ingester.ingest_file(jsonl)

        # The sample has: 1 user message + 1 tool_use embedded in assistant = 2 imported
        # tool_result and pure assistant text entries are skipped
        assert result.imported >= 1
        assert result.total_entries == len(SAMPLE_JSONL_LINES)

    def test_ingest_skip_existing_without_overwrite(self, tmp_path: Path) -> None:
        """Re-ingesting without overwrite=True should skip existing sessions."""
        source_dir = tmp_path / "claude_sessions"
        source_dir.mkdir()
        graph_dir = tmp_path / ".htmlgraph"
        graph_dir.mkdir()

        jsonl = write_sample_jsonl(source_dir)
        ingester = ClaudeCodeIngester(graph_dir=graph_dir, overwrite=False)

        result1 = ingester.ingest_file(jsonl)
        assert result1.was_existing is False

        result2 = ingester.ingest_file(jsonl)
        assert result2.was_existing is True
        assert result2.session_id == SAMPLE_SESSION_ID
        # Was skipped (no re-import)
        assert result2.imported == 0

    def test_ingest_overwrite_reimports(self, tmp_path: Path) -> None:
        """With overwrite=True, re-ingesting should re-import events."""
        source_dir = tmp_path / "claude_sessions"
        source_dir.mkdir()
        graph_dir = tmp_path / ".htmlgraph"
        graph_dir.mkdir()

        jsonl = write_sample_jsonl(source_dir)
        ingester_no_overwrite = ClaudeCodeIngester(graph_dir=graph_dir, overwrite=False)
        result1 = ingester_no_overwrite.ingest_file(jsonl)
        assert result1.was_existing is False

        ingester_overwrite = ClaudeCodeIngester(graph_dir=graph_dir, overwrite=True)
        result2 = ingester_overwrite.ingest_file(jsonl)
        assert result2.was_existing is True
        assert result2.imported > 0

    def test_ingest_sets_agent_name(self, tmp_path: Path) -> None:
        """The agent name should be applied to the created session."""
        source_dir = tmp_path / "claude_sessions"
        source_dir.mkdir()
        graph_dir = tmp_path / ".htmlgraph"
        graph_dir.mkdir()

        jsonl = write_sample_jsonl(source_dir)
        ingester = ClaudeCodeIngester(graph_dir=graph_dir, agent="my-custom-agent")
        result = ingester.ingest_file(jsonl)

        from htmlgraph.converter import html_to_session

        session = html_to_session(result.output_path)  # type: ignore[arg-type]
        assert session.agent == "my-custom-agent"

    def test_ingest_activity_log_populated(self, tmp_path: Path) -> None:
        """The created session should have an activity_log with imported events."""
        source_dir = tmp_path / "claude_sessions"
        source_dir.mkdir()
        graph_dir = tmp_path / ".htmlgraph"
        graph_dir.mkdir()

        jsonl = write_sample_jsonl(source_dir)
        ingester = ClaudeCodeIngester(graph_dir=graph_dir)
        result = ingester.ingest_file(jsonl)

        from htmlgraph.converter import html_to_session

        session = html_to_session(result.output_path)  # type: ignore[arg-type]
        assert len(session.activity_log) == result.imported
        assert session.event_count == result.imported


# ---------------------------------------------------------------------------
# Tests for ClaudeCodeIngester.ingest_from_path
# ---------------------------------------------------------------------------


class TestIngestFromPath:
    def test_ingest_directory_multiple_files(self, tmp_path: Path) -> None:
        """Ingesting a directory should process all JSONL files."""
        source_dir = tmp_path / "sessions"
        source_dir.mkdir()
        graph_dir = tmp_path / ".htmlgraph"
        graph_dir.mkdir()

        # Create two sessions
        sid1 = "aaaaaaaa-0000-0000-0000-000000000001"
        sid2 = "bbbbbbbb-0000-0000-0000-000000000002"

        for sid in [sid1, sid2]:
            lines = [
                {
                    "parentUuid": None,
                    "isSidechain": False,
                    "cwd": "/home/user/proj",
                    "sessionId": sid,
                    "version": "2.0.0",
                    "gitBranch": "feat",
                    "type": "user",
                    "message": {"role": "user", "content": "hello"},
                    "uuid": f"uuid-{sid[:4]}",
                    "timestamp": "2026-01-15T09:00:00.000Z",
                }
            ]
            jsonl = source_dir / f"{sid}.jsonl"
            with jsonl.open("w") as f:
                for line in lines:
                    f.write(json.dumps(line) + "\n")

        ingester = ClaudeCodeIngester(graph_dir=graph_dir)
        summary = ingester.ingest_from_path(source_dir)

        assert summary.sessions_processed == 2
        assert summary.sessions_created == 2
        assert summary.sessions_skipped == 0

    def test_ingest_single_file_via_from_path(self, tmp_path: Path) -> None:
        """ingest_from_path with a single file should work."""
        source_dir = tmp_path / "sessions"
        source_dir.mkdir()
        graph_dir = tmp_path / ".htmlgraph"
        graph_dir.mkdir()

        jsonl = write_sample_jsonl(source_dir)
        ingester = ClaudeCodeIngester(graph_dir=graph_dir)
        summary = ingester.ingest_from_path(jsonl)

        assert summary.sessions_processed == 1
        assert summary.sessions_created == 1

    def test_ingest_respects_limit(self, tmp_path: Path) -> None:
        """The limit parameter should cap the number of sessions processed."""
        source_dir = tmp_path / "sessions"
        source_dir.mkdir()
        graph_dir = tmp_path / ".htmlgraph"
        graph_dir.mkdir()

        # Create 3 JSONL files
        for i in range(3):
            sid = f"cccccccc-0000-0000-0000-{i:012d}"
            lines = [
                {
                    "parentUuid": None,
                    "isSidechain": False,
                    "cwd": "/home/user/proj",
                    "sessionId": sid,
                    "version": "2.0.0",
                    "gitBranch": "main",
                    "type": "user",
                    "message": {"role": "user", "content": f"task {i}"},
                    "uuid": f"uuid-{i}",
                    "timestamp": f"2026-01-1{i + 1}T09:00:00.000Z",
                }
            ]
            jsonl = source_dir / f"{sid}.jsonl"
            with jsonl.open("w") as f:
                for line in lines:
                    f.write(json.dumps(line) + "\n")

        ingester = ClaudeCodeIngester(graph_dir=graph_dir)
        summary = ingester.ingest_from_path(source_dir, limit=2)

        assert summary.sessions_processed == 2

    def test_ingest_empty_directory(self, tmp_path: Path) -> None:
        """An empty directory should return an empty summary."""
        source_dir = tmp_path / "sessions"
        source_dir.mkdir()
        graph_dir = tmp_path / ".htmlgraph"
        graph_dir.mkdir()

        ingester = ClaudeCodeIngester(graph_dir=graph_dir)
        summary = ingester.ingest_from_path(source_dir)

        assert summary.sessions_processed == 0
        assert summary.sessions_created == 0

    def test_ingest_nonexistent_path_returns_error(self, tmp_path: Path) -> None:
        """A non-existent path should return a summary with an error."""
        graph_dir = tmp_path / ".htmlgraph"
        graph_dir.mkdir()

        ingester = ClaudeCodeIngester(graph_dir=graph_dir)
        summary = ingester.ingest_from_path(tmp_path / "does-not-exist")

        assert len(summary.errors) > 0
        assert summary.sessions_processed == 0


# ---------------------------------------------------------------------------
# Tests for IngestResult dataclass
# ---------------------------------------------------------------------------


class TestIngestResult:
    def test_success_property(self) -> None:
        result = IngestResult(
            session_id="abc",
            htmlgraph_session_id="sess-abc",
            imported=5,
        )
        assert result.success is True

    def test_str_representation(self) -> None:
        result = IngestResult(
            session_id="abc12345-0000-0000-0000-000000000000",
            htmlgraph_session_id="sess-abc12345",
            imported=10,
            skipped=2,
            was_existing=False,
        )
        s = str(result)
        assert "sess-abc12345" in s
        assert "created" in s
        assert "10 events" in s

    def test_str_shows_updated_for_existing(self) -> None:
        result = IngestResult(
            session_id="abc12345-0000-0000-0000-000000000000",
            htmlgraph_session_id="sess-abc12345",
            imported=3,
            was_existing=True,
        )
        assert "updated" in str(result)


# ---------------------------------------------------------------------------
# Tests for IngestSummary dataclass
# ---------------------------------------------------------------------------


class TestIngestSummary:
    def test_str_representation(self) -> None:
        summary = IngestSummary(
            sessions_processed=3,
            sessions_created=2,
            sessions_updated=1,
            sessions_skipped=0,
            total_events_imported=42,
        )
        s = str(summary)
        assert "3" in s
        assert "Created: 2" in s
        assert "Updated: 1" in s
        assert "42" in s
