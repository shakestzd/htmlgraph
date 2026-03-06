"""Tests for the Gemini CLI session ingester."""
from __future__ import annotations

import json
import tempfile
from datetime import datetime, timezone
from pathlib import Path

import pytest

from htmlgraph.ingest.gemini import (
    GeminiMessage,
    GeminiSession,
    IngestResult,
    find_gemini_sessions,
    parse_gemini_session,
)


# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------


def _make_session_json(
    session_id: str = "test-session-uuid",
    project_hash: str = "aabbccdd",
    start_time: str = "2025-01-01T10:00:00.000Z",
    last_updated: str = "2025-01-01T10:30:00.000Z",
    messages: list | None = None,
) -> dict:
    if messages is None:
        messages = [
            {
                "id": "msg-1",
                "timestamp": "2025-01-01T10:00:00.000Z",
                "type": "user",
                "content": "Hello Gemini, help me write tests",
            },
            {
                "id": "msg-2",
                "timestamp": "2025-01-01T10:00:05.000Z",
                "type": "gemini",
                "content": "Sure! Here is how to write tests...",
                "toolCalls": [
                    {
                        "id": "read_file-123",
                        "name": "read_file",
                        "args": {"file_path": "test.py"},
                        "result": [],
                    }
                ],
                "thoughts": "The user wants test examples",
                "model": "gemini-2.5-pro",
                "tokens": {"input": 100, "output": 200},
            },
            {
                "id": "msg-3",
                "timestamp": "2025-01-01T10:05:00.000Z",
                "type": "info",
                "content": "Update successful!",
            },
        ]
    return {
        "sessionId": session_id,
        "projectHash": project_hash,
        "startTime": start_time,
        "lastUpdated": last_updated,
        "messages": messages,
    }


@pytest.fixture
def tmp_gemini_dir(tmp_path: Path) -> Path:
    """Create a temporary Gemini session storage structure."""
    project_hash = "aabbccddeeff001122334455"
    chats_dir = tmp_path / project_hash / "chats"
    chats_dir.mkdir(parents=True)

    # Write one session file
    session_data = _make_session_json()
    session_file = chats_dir / "session-2025-01-01T10-00-00test-session.json"
    session_file.write_text(json.dumps(session_data), encoding="utf-8")

    return tmp_path


@pytest.fixture
def tmp_gemini_dir_multi(tmp_path: Path) -> Path:
    """Create a temporary Gemini storage with multiple projects and sessions."""
    for i in range(2):
        project_hash = f"projecthash{i:020d}"
        chats_dir = tmp_path / project_hash / "chats"
        chats_dir.mkdir(parents=True)
        for j in range(3):
            session_data = _make_session_json(
                session_id=f"session-{i}-{j}",
                project_hash=project_hash,
            )
            sf = chats_dir / f"session-2025-0{i+1}-0{j+1}T10-00-00.json"
            sf.write_text(json.dumps(session_data), encoding="utf-8")
    return tmp_path


# ---------------------------------------------------------------------------
# GeminiMessage tests
# ---------------------------------------------------------------------------


class TestGeminiMessage:
    def test_parse_user_message(self) -> None:
        data = {
            "id": "abc",
            "timestamp": "2025-01-01T12:00:00.000Z",
            "type": "user",
            "content": "Hello",
        }
        msg = GeminiMessage.from_dict(data)
        assert msg.id == "abc"
        assert msg.msg_type == "user"
        assert msg.content == "Hello"
        assert msg.tool_calls == []
        assert msg.thoughts is None
        assert msg.model is None

    def test_parse_gemini_message_with_tool_calls(self) -> None:
        data = {
            "id": "def",
            "timestamp": "2025-01-01T12:00:05.000Z",
            "type": "gemini",
            "content": "Sure!",
            "toolCalls": [{"id": "tc-1", "name": "read_file", "args": {}, "result": []}],
            "thoughts": "thinking...",
            "model": "gemini-2.5-pro",
            "tokens": {"input": 50, "output": 100},
        }
        msg = GeminiMessage.from_dict(data)
        assert msg.msg_type == "gemini"
        assert len(msg.tool_calls) == 1
        assert msg.tool_calls[0]["name"] == "read_file"
        assert msg.thoughts == "thinking..."
        assert msg.model == "gemini-2.5-pro"

    def test_parse_invalid_timestamp_falls_back(self) -> None:
        data = {
            "id": "x",
            "timestamp": "not-a-timestamp",
            "type": "info",
            "content": "",
        }
        before = datetime.now()
        msg = GeminiMessage.from_dict(data)
        after = datetime.now()
        assert before <= msg.timestamp <= after

    def test_parse_empty_thoughts_becomes_none(self) -> None:
        data = {
            "id": "y",
            "timestamp": "2025-01-01T12:00:00Z",
            "type": "gemini",
            "content": "",
            "thoughts": "",
        }
        msg = GeminiMessage.from_dict(data)
        assert msg.thoughts is None


# ---------------------------------------------------------------------------
# GeminiSession tests
# ---------------------------------------------------------------------------


class TestGeminiSession:
    def test_parse_full_session(self, tmp_path: Path) -> None:
        session_data = _make_session_json()
        sf = tmp_path / "session.json"
        sf.write_text(json.dumps(session_data), encoding="utf-8")

        gs = GeminiSession.from_dict(session_data, source_file=sf)

        assert gs.session_id == "test-session-uuid"
        assert gs.project_hash == "aabbccdd"
        assert gs.start_time.year == 2025
        assert len(gs.messages) == 3

    def test_user_turn_count(self, tmp_path: Path) -> None:
        data = _make_session_json()
        gs = GeminiSession.from_dict(data, source_file=tmp_path / "s.json")
        assert gs.user_turn_count == 1

    def test_gemini_turn_count(self, tmp_path: Path) -> None:
        data = _make_session_json()
        gs = GeminiSession.from_dict(data, source_file=tmp_path / "s.json")
        assert gs.gemini_turn_count == 1

    def test_tool_call_count(self, tmp_path: Path) -> None:
        data = _make_session_json()
        gs = GeminiSession.from_dict(data, source_file=tmp_path / "s.json")
        assert gs.tool_call_count == 1

    def test_tool_names_used(self, tmp_path: Path) -> None:
        data = _make_session_json()
        gs = GeminiSession.from_dict(data, source_file=tmp_path / "s.json")
        assert gs.tool_names_used == ["read_file"]

    def test_first_user_prompt(self, tmp_path: Path) -> None:
        data = _make_session_json()
        gs = GeminiSession.from_dict(data, source_file=tmp_path / "s.json")
        assert gs.first_user_prompt == "Hello Gemini, help me write tests"

    def test_first_user_prompt_truncated(self, tmp_path: Path) -> None:
        long_prompt = "x" * 300
        data = _make_session_json(
            messages=[
                {
                    "id": "m1",
                    "timestamp": "2025-01-01T10:00:00Z",
                    "type": "user",
                    "content": long_prompt,
                }
            ]
        )
        gs = GeminiSession.from_dict(data, source_file=tmp_path / "s.json")
        assert gs.first_user_prompt is not None
        assert len(gs.first_user_prompt) <= 200

    def test_no_user_messages_returns_none(self, tmp_path: Path) -> None:
        data = _make_session_json(
            messages=[
                {
                    "id": "m1",
                    "timestamp": "2025-01-01T10:00:00Z",
                    "type": "info",
                    "content": "Update successful!",
                }
            ]
        )
        gs = GeminiSession.from_dict(data, source_file=tmp_path / "s.json")
        assert gs.first_user_prompt is None

    def test_tool_names_deduped(self, tmp_path: Path) -> None:
        data = _make_session_json(
            messages=[
                {
                    "id": "m1",
                    "timestamp": "2025-01-01T10:00:00Z",
                    "type": "gemini",
                    "content": "",
                    "toolCalls": [
                        {"id": "1", "name": "read_file", "args": {}, "result": []},
                        {"id": "2", "name": "read_file", "args": {}, "result": []},
                        {"id": "3", "name": "run_shell_command", "args": {}, "result": []},
                    ],
                }
            ]
        )
        gs = GeminiSession.from_dict(data, source_file=tmp_path / "s.json")
        assert gs.tool_names_used == ["read_file", "run_shell_command"]


# ---------------------------------------------------------------------------
# find_gemini_sessions tests
# ---------------------------------------------------------------------------


class TestFindGeminiSessions:
    def test_finds_session_files(self, tmp_gemini_dir: Path) -> None:
        files = find_gemini_sessions(base_path=tmp_gemini_dir)
        assert len(files) == 1
        assert files[0].suffix == ".json"
        assert "session-" in files[0].name

    def test_finds_multiple_sessions(self, tmp_gemini_dir_multi: Path) -> None:
        files = find_gemini_sessions(base_path=tmp_gemini_dir_multi)
        assert len(files) == 6

    def test_empty_directory_returns_empty(self, tmp_path: Path) -> None:
        files = find_gemini_sessions(base_path=tmp_path)
        assert files == []

    def test_nonexistent_path_returns_empty(self, tmp_path: Path) -> None:
        files = find_gemini_sessions(base_path=tmp_path / "does-not-exist")
        assert files == []

    def test_returns_sorted_by_mtime_newest_first(
        self, tmp_gemini_dir_multi: Path
    ) -> None:
        files = find_gemini_sessions(base_path=tmp_gemini_dir_multi)
        mtimes = [f.stat().st_mtime for f in files]
        assert mtimes == sorted(mtimes, reverse=True)


# ---------------------------------------------------------------------------
# parse_gemini_session tests
# ---------------------------------------------------------------------------


class TestParseGeminiSession:
    def test_parses_valid_file(self, tmp_gemini_dir: Path) -> None:
        files = find_gemini_sessions(base_path=tmp_gemini_dir)
        assert len(files) == 1
        gs = parse_gemini_session(files[0])
        assert gs is not None
        assert gs.session_id == "test-session-uuid"

    def test_returns_none_for_invalid_json(self, tmp_path: Path) -> None:
        bad_file = tmp_path / "bad.json"
        bad_file.write_text("{ not valid json }", encoding="utf-8")
        result = parse_gemini_session(bad_file)
        assert result is None

    def test_returns_none_for_missing_file(self, tmp_path: Path) -> None:
        result = parse_gemini_session(tmp_path / "nonexistent.json")
        assert result is None

    def test_parses_empty_messages(self, tmp_path: Path) -> None:
        data = _make_session_json(messages=[])
        sf = tmp_path / "empty.json"
        sf.write_text(json.dumps(data), encoding="utf-8")
        gs = parse_gemini_session(sf)
        assert gs is not None
        assert gs.messages == []
        assert gs.tool_call_count == 0
        assert gs.user_turn_count == 0
