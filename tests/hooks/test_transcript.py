"""
Tests for htmlgraph.hooks.transcript — Claude Code transcript reader.

Covers:
- Project hash computation (directory -> transcript dir path)
- Transcript file selection (most recently modified .jsonl)
- Parsing: tool_use_id found in assistant message
- Parsing: parentUuid chain walk to user message
- Parsing: skips tool_result messages in chain
- Parsing: handles malformed JSON lines gracefully
- Parsing: handles cycle in parentUuid chain
- Parsing: tool_use_id not present in file
- Module-level cache: repeated calls don't re-parse
- find_parent_user_query: returns (None, None) when transcript dir missing
- find_parent_user_query: returns (session_id, uuid) on success
"""

import json
import time
from pathlib import Path
from unittest import mock

import pytest
from htmlgraph.hooks.transcript import (
    _find_transcript_file,
    _get_transcript_dir,
    _parse_transcript,
    clear_cache,
    find_parent_user_query,
)

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _write_jsonl(path: Path, messages: list[dict]) -> None:
    """Write a list of message dicts as JSONL."""
    with open(path, "w", encoding="utf-8") as fh:
        for msg in messages:
            fh.write(json.dumps(msg) + "\n")


def _make_user_msg(
    uuid: str, parent_uuid: str | None, session_id: str, text: str = "hello"
) -> dict:
    return {
        "type": "user",
        "uuid": uuid,
        "parentUuid": parent_uuid,
        "sessionId": session_id,
        "message": {"role": "user", "content": text},
    }


def _make_tool_result_msg(
    uuid: str, parent_uuid: str | None, session_id: str, tool_use_id: str
) -> dict:
    """A user message that is actually a tool_result (should be skipped when walking chain)."""
    return {
        "type": "user",
        "uuid": uuid,
        "parentUuid": parent_uuid,
        "sessionId": session_id,
        "message": {
            "role": "user",
            "content": [
                {"type": "tool_result", "tool_use_id": tool_use_id, "content": "done"}
            ],
        },
    }


def _make_assistant_tool_use(
    uuid: str,
    parent_uuid: str | None,
    session_id: str,
    tool_use_id: str,
    tool_name: str = "Bash",
) -> dict:
    return {
        "type": "assistant",
        "uuid": uuid,
        "parentUuid": parent_uuid,
        "sessionId": session_id,
        "message": {
            "role": "assistant",
            "content": [
                {
                    "type": "tool_use",
                    "id": tool_use_id,
                    "name": tool_name,
                    "input": {"command": "ls"},
                },
            ],
        },
    }


# ---------------------------------------------------------------------------
# _get_transcript_dir
# ---------------------------------------------------------------------------


def test_get_transcript_dir_replaces_slashes():
    result = _get_transcript_dir("/Users/shakes/DevProjects/htmlgraph")
    expected = (
        Path.home() / ".claude" / "projects" / "-Users-shakes-DevProjects-htmlgraph"
    )
    assert result == expected


def test_get_transcript_dir_single_segment():
    result = _get_transcript_dir("/myproject")
    expected = Path.home() / ".claude" / "projects" / "-myproject"
    assert result == expected


# ---------------------------------------------------------------------------
# _find_transcript_file
# ---------------------------------------------------------------------------


def test_find_transcript_file_returns_none_for_missing_dir(tmp_path):
    missing = tmp_path / "nonexistent"
    assert _find_transcript_file(missing) is None


def test_find_transcript_file_returns_none_when_no_jsonl(tmp_path):
    (tmp_path / "some.txt").write_text("hello")
    assert _find_transcript_file(tmp_path) is None


def test_find_transcript_file_returns_most_recent(tmp_path):
    old = tmp_path / "old.jsonl"
    new = tmp_path / "new.jsonl"
    old.write_text("{}")
    time.sleep(0.01)  # ensure different mtime
    new.write_text("{}")
    result = _find_transcript_file(tmp_path)
    assert result == new


def test_find_transcript_file_single_file(tmp_path):
    f = tmp_path / "sess.jsonl"
    f.write_text("{}")
    assert _find_transcript_file(tmp_path) == f


# ---------------------------------------------------------------------------
# _parse_transcript — basic success case
# ---------------------------------------------------------------------------


def test_parse_transcript_finds_tool_use_and_walks_to_user(tmp_path):
    """
    Simple linear chain:
      user_msg -> assistant_with_tool_use
    """
    transcript = tmp_path / "session.jsonl"
    session_id = "sess-abc123"
    tool_use_id = "toolu_01XYZ"

    messages = [
        _make_user_msg("user-uuid-1", None, session_id, "Do something"),
        _make_assistant_tool_use("asst-uuid-1", "user-uuid-1", session_id, tool_use_id),
    ]
    _write_jsonl(transcript, messages)

    result_session, result_uuid = _parse_transcript(transcript, tool_use_id)
    assert result_session == session_id
    assert result_uuid == "user-uuid-1"


def test_parse_transcript_skips_tool_result_messages(tmp_path):
    """
    Chain: user_original -> assistant_1 -> tool_result_msg -> assistant_2(with target tool)
    The tool_result_msg should be skipped; the walk should reach user_original.
    """
    transcript = tmp_path / "session.jsonl"
    session_id = "sess-abc123"
    prev_tool_id = "toolu_prev"
    target_tool_id = "toolu_target"

    messages = [
        _make_user_msg("user-uuid-1", None, session_id, "original question"),
        _make_assistant_tool_use(
            "asst-uuid-1", "user-uuid-1", session_id, prev_tool_id
        ),
        _make_tool_result_msg("tr-uuid-1", "asst-uuid-1", session_id, prev_tool_id),
        _make_assistant_tool_use(
            "asst-uuid-2", "tr-uuid-1", session_id, target_tool_id
        ),
    ]
    _write_jsonl(transcript, messages)

    result_session, result_uuid = _parse_transcript(transcript, target_tool_id)
    assert result_session == session_id
    assert result_uuid == "user-uuid-1"


def test_parse_transcript_tool_use_not_found(tmp_path):
    """Returns (None, None) when tool_use_id is not in the transcript."""
    transcript = tmp_path / "session.jsonl"
    messages = [
        _make_user_msg("user-1", None, "sess-1", "hello"),
        _make_assistant_tool_use("asst-1", "user-1", "sess-1", "toolu_OTHER"),
    ]
    _write_jsonl(transcript, messages)

    result = _parse_transcript(transcript, "toolu_NOT_PRESENT")
    assert result == (None, None)


def test_parse_transcript_handles_malformed_json_lines(tmp_path):
    """Malformed lines are skipped; valid lines are still parsed."""
    transcript = tmp_path / "session.jsonl"
    session_id = "sess-abc"
    tool_use_id = "toolu_good"

    with open(transcript, "w") as fh:
        fh.write("THIS IS NOT JSON\n")
        fh.write("{incomplete json\n")
        fh.write(json.dumps(_make_user_msg("user-1", None, session_id, "hi")) + "\n")
        fh.write(
            json.dumps(
                _make_assistant_tool_use("asst-1", "user-1", session_id, tool_use_id)
            )
            + "\n"
        )

    result_session, result_uuid = _parse_transcript(transcript, tool_use_id)
    assert result_session == session_id
    assert result_uuid == "user-1"


def test_parse_transcript_handles_missing_file(tmp_path):
    """Returns (None, None) gracefully for a non-existent file."""
    missing = tmp_path / "nonexistent.jsonl"
    result = _parse_transcript(missing, "toolu_01")
    assert result == (None, None)


def test_parse_transcript_handles_cycle_in_parent_chain(tmp_path):
    """If parentUuid forms a cycle, the function breaks and returns session_id without uuid."""
    transcript = tmp_path / "session.jsonl"
    session_id = "sess-cycle"
    tool_use_id = "toolu_cycle"

    # Create a cycle: asst -> msg-a -> msg-b -> msg-a (cycle)
    messages = [
        {
            "type": "assistant",
            "uuid": "asst-1",
            "parentUuid": "msg-a",
            "sessionId": session_id,
            "message": {
                "role": "assistant",
                "content": [
                    {"type": "tool_use", "id": tool_use_id, "name": "Bash", "input": {}}
                ],
            },
        },
        {
            "type": "assistant",  # not a user msg, but part of chain
            "uuid": "msg-a",
            "parentUuid": "msg-b",
            "sessionId": session_id,
            "message": {"role": "assistant", "content": []},
        },
        {
            "type": "assistant",
            "uuid": "msg-b",
            "parentUuid": "msg-a",  # cycle back to msg-a
            "sessionId": session_id,
            "message": {"role": "assistant", "content": []},
        },
    ]
    _write_jsonl(transcript, messages)

    result_session, result_uuid = _parse_transcript(transcript, tool_use_id)
    # Should return the session_id (from assistant message) but no user uuid
    assert result_session == session_id
    assert result_uuid is None


def test_parse_transcript_entries_without_uuid_are_skipped(tmp_path):
    """Messages without uuid (e.g. file-history-snapshot) don't break parsing."""
    transcript = tmp_path / "session.jsonl"
    session_id = "sess-abc"
    tool_use_id = "toolu_real"

    with open(transcript, "w") as fh:
        # A system entry without uuid
        fh.write(json.dumps({"type": "file-history-snapshot", "snapshot": {}}) + "\n")
        fh.write(
            json.dumps(_make_user_msg("user-1", None, session_id, "question")) + "\n"
        )
        fh.write(
            json.dumps(
                _make_assistant_tool_use("asst-1", "user-1", session_id, tool_use_id)
            )
            + "\n"
        )

    result_session, result_uuid = _parse_transcript(transcript, tool_use_id)
    assert result_session == session_id
    assert result_uuid == "user-1"


# ---------------------------------------------------------------------------
# find_parent_user_query — integration with caching
# ---------------------------------------------------------------------------


@pytest.fixture(autouse=True)
def reset_cache():
    """Ensure the module cache is clear before and after each test."""
    clear_cache()
    yield
    clear_cache()


def test_find_parent_user_query_no_transcript_dir(tmp_path):
    """Returns (None, None) when the project has no transcript directory."""
    fake_project = str(tmp_path / "fake_project")

    with mock.patch("htmlgraph.hooks.transcript._get_transcript_dir") as mock_dir:
        mock_dir.return_value = tmp_path / "nonexistent_dir"
        result = find_parent_user_query("toolu_01XYZ", fake_project)

    assert result == (None, None)


def test_find_parent_user_query_success(tmp_path):
    """Happy path: transcript exists and tool_use_id is found."""
    session_id = "sess-happy"
    tool_use_id = "toolu_happy"

    transcript_dir = tmp_path / "transcript_dir"
    transcript_dir.mkdir()
    transcript = transcript_dir / "session.jsonl"
    messages = [
        _make_user_msg("user-1", None, session_id, "do the thing"),
        _make_assistant_tool_use("asst-1", "user-1", session_id, tool_use_id),
    ]
    _write_jsonl(transcript, messages)

    with mock.patch("htmlgraph.hooks.transcript._get_transcript_dir") as mock_dir:
        mock_dir.return_value = transcript_dir
        result_session, result_uuid = find_parent_user_query(
            tool_use_id, "/fake/project"
        )

    assert result_session == session_id
    assert result_uuid == "user-1"


def test_find_parent_user_query_caches_result(tmp_path):
    """Second call with same tool_use_id uses cached result without re-parsing."""
    session_id = "sess-cache"
    tool_use_id = "toolu_cached"

    transcript_dir = tmp_path / "transcript_dir"
    transcript_dir.mkdir()
    transcript = transcript_dir / "session.jsonl"
    messages = [
        _make_user_msg("user-1", None, session_id, "hello"),
        _make_assistant_tool_use("asst-1", "user-1", session_id, tool_use_id),
    ]
    _write_jsonl(transcript, messages)

    with mock.patch("htmlgraph.hooks.transcript._get_transcript_dir") as mock_dir:
        mock_dir.return_value = transcript_dir
        result1 = find_parent_user_query(tool_use_id, "/fake/project")

    # Delete transcript so second call would fail if it re-parsed.
    transcript.unlink()

    with mock.patch("htmlgraph.hooks.transcript._get_transcript_dir") as mock_dir:
        mock_dir.return_value = transcript_dir
        result2 = find_parent_user_query(tool_use_id, "/fake/project")

    assert result1 == result2
    assert result1 == (session_id, "user-1")


def test_find_parent_user_query_different_tool_ids_not_cached(tmp_path):
    """Each tool_use_id has its own cache entry."""
    session_id = "sess-multi"
    transcript_dir = tmp_path / "td"
    transcript_dir.mkdir()
    transcript = transcript_dir / "session.jsonl"

    messages = [
        _make_user_msg("user-1", None, session_id, "first"),
        _make_assistant_tool_use("asst-1", "user-1", session_id, "toolu_A"),
        _make_tool_result_msg("tr-1", "asst-1", session_id, "toolu_A"),
        _make_user_msg("user-2", "tr-1", session_id, "second"),
        _make_assistant_tool_use("asst-2", "user-2", session_id, "toolu_B"),
    ]
    _write_jsonl(transcript, messages)

    with mock.patch("htmlgraph.hooks.transcript._get_transcript_dir") as mock_dir:
        mock_dir.return_value = transcript_dir

        res_a = find_parent_user_query("toolu_A", "/fake")
        res_b = find_parent_user_query("toolu_B", "/fake")

    assert res_a == (session_id, "user-1")
    assert res_b == (session_id, "user-2")


def test_find_parent_user_query_exception_returns_none_none(tmp_path):
    """If an unexpected exception occurs, returns (None, None) gracefully."""
    with mock.patch(
        "htmlgraph.hooks.transcript._get_transcript_dir",
        side_effect=RuntimeError("unexpected"),
    ):
        result = find_parent_user_query("toolu_01", "/fake/project")

    assert result == (None, None)


def test_clear_cache_resets_state(tmp_path):
    """clear_cache() causes the next call to re-parse."""
    session_id = "sess-reset"
    tool_use_id = "toolu_reset"

    transcript_dir = tmp_path / "td"
    transcript_dir.mkdir()
    transcript = transcript_dir / "session.jsonl"
    messages = [
        _make_user_msg("user-1", None, session_id, "hi"),
        _make_assistant_tool_use("asst-1", "user-1", session_id, tool_use_id),
    ]
    _write_jsonl(transcript, messages)

    with mock.patch(
        "htmlgraph.hooks.transcript._get_transcript_dir", return_value=transcript_dir
    ):
        res1 = find_parent_user_query(tool_use_id, "/fake")

    clear_cache()

    # Remove the transcript; after clear_cache, the next call re-parses and finds nothing.
    transcript.unlink()

    with mock.patch(
        "htmlgraph.hooks.transcript._get_transcript_dir", return_value=transcript_dir
    ):
        res2 = find_parent_user_query(tool_use_id, "/fake")

    assert res1 == (session_id, "user-1")
    assert res2 == (None, None)  # re-parsed, file gone
