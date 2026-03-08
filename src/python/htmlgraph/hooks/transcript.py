"""
Claude Code Conversation Transcript Reader.

Provides parallel-safe parent resolution by reading the Claude Code conversation
transcript instead of relying on env vars or timestamps.

Problem solved:
- HTMLGRAPH_PARENT_EVENT env var persists across Claude Code restarts and is
  wrong for resumed sessions.
- Timestamp-based staleness detection breaks for parallel sessions (multiple
  agents running simultaneously).

Solution:
- Claude Code writes conversation transcripts to:
    ~/.claude/projects/{PROJECT_HASH}/{SESSION_ID}.jsonl
- Each line is a JSON object with uuid, parentUuid, type, sessionId fields.
- Tool uses in assistant messages have type="tool_use" with id=toolu_XXX.
- We walk the parentUuid chain from that assistant message up to the nearest
  user turn to identify the originating UserQuery.

Transcript format (one JSON object per line):
  {
    "uuid": "5d490ebd-...",
    "parentUuid": "d43e905d-...",
    "type": "assistant",
    "sessionId": "175e9a56-...",
    "message": {
      "role": "assistant",
      "content": [
        {"type": "tool_use", "id": "toolu_01XYZ", "name": "Bash", "input": {...}}
      ]
    }
  }

  {
    "uuid": "7212300f-...",
    "parentUuid": "82560d73-...",
    "type": "user",
    "sessionId": "175e9a56-...",
    "message": {
      "role": "user",
      "content": "..."
    }
  }
"""

import json
import logging
from pathlib import Path

logger = logging.getLogger(__name__)

# Module-level cache: tool_use_id -> (session_id, user_turn_uuid)
# Avoids re-parsing the transcript for each hook call in the same process.
_cache: dict[str, tuple[str | None, str | None]] = {}


def _get_transcript_dir(project_dir: str) -> Path:
    """
    Compute the ~/.claude/projects/{project_hash}/ directory for a given project.

    The project hash is the absolute path with '/' replaced by '-':
      /Users/shakes/DevProjects/htmlgraph -> -Users-shakes-DevProjects-htmlgraph

    Args:
        project_dir: Absolute path to the project root directory.

    Returns:
        Path to the transcript directory (may not exist).
    """
    project_hash = project_dir.replace("/", "-")
    return Path.home() / ".claude" / "projects" / project_hash


def get_transcript_path(
    session_id: str, project_path: str | None = None
) -> Path | None:
    """
    Get the exact path to the Claude Code transcript for a given session_id.

    Constructs the canonical path directly from session_id rather than relying
    on mtime ordering, which picks the wrong file when multiple parallel sessions
    exist in the same project directory.

    Path format: ~/.claude/projects/{project_hash}/{session_id}.jsonl

    Args:
        session_id: Claude Code session ID (e.g. "175e9a56-fcdc-4b68-a466-a9145156bdfa").
        project_path: Absolute path to the project root. Defaults to cwd.

    Returns:
        Path to the .jsonl transcript file if it exists, otherwise None.
    """
    if project_path is None:
        project_path = str(Path.cwd())
    transcript_dir = _get_transcript_dir(project_path)
    candidate = transcript_dir / f"{session_id}.jsonl"
    if candidate.exists():
        return candidate
    logger.debug(f"Transcript not found at exact path: {candidate}")
    return None


def get_subagent_transcript_path(
    session_id: str, agent_id: str, project_path: str | None = None
) -> Path | None:
    """
    Get the path to a subagent's JSONL transcript file.

    Subagent transcripts live at:
      ~/.claude/projects/{project_hash}/{session_id}/subagents/agent-{agent_id}.jsonl

    Args:
        session_id: The parent session ID.
        agent_id: The subagent's agent ID (without the "agent-" prefix).
        project_path: Absolute path to the project root. Defaults to cwd.

    Returns:
        Path to the subagent .jsonl file if it exists, otherwise None.
    """
    session_path = get_transcript_path(session_id, project_path)
    if session_path is None:
        return None
    subagent_path = (
        session_path.parent / session_id / "subagents" / f"agent-{agent_id}.jsonl"
    )
    if subagent_path.exists():
        return subagent_path
    logger.debug(f"Subagent transcript not found at: {subagent_path}")
    return None


def _find_transcript_file(
    transcript_dir: Path, session_id: str | None = None
) -> Path | None:
    """
    Locate the .jsonl transcript file for the current session.

    When session_id is provided, constructs the exact path directly (preferred).
    Falls back to the most recently modified .jsonl only when session_id is not
    known, preserving backward compatibility.

    Args:
        transcript_dir: Directory containing .jsonl transcript files.
        session_id: Optional session ID for direct file lookup.

    Returns:
        Path to the transcript .jsonl file, or None if not found.
    """
    if not transcript_dir.exists():
        logger.debug(f"Transcript dir does not exist: {transcript_dir}")
        return None

    # Preferred: construct exact path from session_id — avoids mtime races
    # when multiple parallel sessions exist in the same project directory.
    if session_id:
        candidate = transcript_dir / f"{session_id}.jsonl"
        if candidate.exists():
            return candidate
        logger.debug(
            f"Transcript not found at exact path {candidate}; "
            "falling back to mtime heuristic"
        )

    # Fallback: pick the most recently modified .jsonl (legacy behaviour for
    # callers that don't have a session_id available).
    jsonl_files = list(transcript_dir.glob("*.jsonl"))
    if not jsonl_files:
        logger.debug(f"No .jsonl files in {transcript_dir}")
        return None

    # Sort by modification time descending; pick the most recently touched file.
    jsonl_files.sort(key=lambda p: p.stat().st_mtime, reverse=True)
    return jsonl_files[0]


def _parse_transcript(
    transcript_path: Path, tool_use_id: str
) -> tuple[str | None, str | None]:
    """
    Parse a JSONL transcript file and resolve the originating user turn.

    Strategy:
    1. Stream through the file, indexing each message by uuid.
    2. When an assistant message containing tool_use_id is found, stop reading.
    3. Walk the parentUuid chain upward until a 'user' type message is reached.
    4. Return (session_id, user_message_uuid).

    The function stops reading the file early once the target tool_use is found,
    making it efficient even for large transcripts.

    Args:
        transcript_path: Path to the .jsonl transcript file.
        tool_use_id: The tool_use id to locate (e.g. "toolu_01XYZ").

    Returns:
        (session_id, user_turn_uuid) or (None, None) if not found.
    """
    # Index of uuid -> message dict, built while scanning.
    messages_by_uuid: dict[str, dict] = {}
    target_session_id: str | None = None
    target_assistant_uuid: str | None = None

    try:
        with open(transcript_path, encoding="utf-8") as fh:
            for raw_line in fh:
                raw_line = raw_line.strip()
                if not raw_line:
                    continue

                try:
                    msg = json.loads(raw_line)
                except json.JSONDecodeError:
                    # Skip malformed lines gracefully.
                    continue

                msg_uuid = msg.get("uuid")
                if not msg_uuid:
                    # Some entries (file-history-snapshot, queue-operation, etc.)
                    # don't have a uuid; skip them for the index.
                    continue

                messages_by_uuid[msg_uuid] = msg

                # Check if this is the assistant message containing our tool_use_id.
                if msg.get("type") == "assistant":
                    content = msg.get("message", {}).get("content", [])
                    if isinstance(content, list):
                        for block in content:
                            if (
                                isinstance(block, dict)
                                and block.get("type") == "tool_use"
                                and block.get("id") == tool_use_id
                            ):
                                target_assistant_uuid = msg_uuid
                                target_session_id = msg.get("sessionId")
                                break

                if target_assistant_uuid is not None:
                    # Found our target message; stop reading the file.
                    break

    except OSError as e:
        logger.debug(f"Could not read transcript {transcript_path}: {e}")
        return None, None

    if target_assistant_uuid is None:
        logger.debug(f"tool_use_id={tool_use_id} not found in {transcript_path.name}")
        return None, None

    # Walk the parentUuid chain upward to find the nearest user-type message.
    current_uuid: str | None = target_assistant_uuid
    visited: set[str] = set()

    while current_uuid is not None:
        if current_uuid in visited:
            # Cycle guard.
            logger.debug(f"Cycle detected in parentUuid chain at {current_uuid}")
            break
        visited.add(current_uuid)

        msg = messages_by_uuid.get(current_uuid)
        if msg is None:
            # Parent not indexed (may be before the start of our window or missing).
            logger.debug(f"Parent uuid {current_uuid} not found in indexed messages")
            break

        if msg.get("type") == "user":
            # Check it's a real user turn (not a meta/tool_result message).
            content = msg.get("message", {}).get("content", "")
            is_tool_result = False
            if isinstance(content, list):
                for block in content:
                    if isinstance(block, dict) and block.get("type") == "tool_result":
                        is_tool_result = True
                        break
            if not is_tool_result:
                # This is the originating user message.
                user_uuid = current_uuid
                session_id = target_session_id or msg.get("sessionId")
                logger.debug(
                    f"Resolved tool_use_id={tool_use_id} -> "
                    f"session={session_id}, user_turn={user_uuid}"
                )
                return session_id, user_uuid

        current_uuid = msg.get("parentUuid")

    logger.debug(
        f"Could not walk parentUuid chain to a user message for "
        f"tool_use_id={tool_use_id}"
    )
    return target_session_id, None


def find_parent_user_query(
    tool_use_id: str, project_dir: str, session_id: str | None = None
) -> tuple[str | None, str | None]:
    """
    Find the originating user turn for a tool_use_id by reading the Claude Code
    conversation transcript.

    This provides parallel-safe, restart-safe parent resolution. It does not
    depend on environment variables or timestamp comparisons.

    Given a tool_use_id from a PostToolUse or PreToolUse hook, this function:
    1. Locates the transcript directory for the current project.
    2. When session_id is supplied, opens the exact {session_id}.jsonl file
       (avoids mtime races with parallel sessions). Otherwise falls back to
       the most recently modified .jsonl.
    3. Parses the transcript to find the assistant message containing tool_use_id.
    4. Walks the parentUuid chain to the nearest user (non-tool-result) message.
    5. Returns (session_id, user_turn_uuid).

    Results are cached per tool_use_id within the same process to avoid repeated
    file I/O when multiple hook calls reference the same tool use.

    Args:
        tool_use_id: The Claude Code tool use ID (e.g. "toolu_01XYZ").
        project_dir: Absolute path to the project root directory, used to
                     compute the transcript directory hash.
        session_id: Optional session ID for direct file lookup. When provided,
                    the exact {session_id}.jsonl is opened instead of using the
                    mtime heuristic. Callers with session_id available should
                    always pass it to ensure correct parallel-session behaviour.

    Returns:
        (session_id, user_turn_uuid): Both strings if found, otherwise
        (None, None). session_id may be returned without user_turn_uuid if the
        assistant message was found but the parent chain walk failed.

    Example:
        session_id, user_turn_uuid = find_parent_user_query(
            "toolu_01XYZ", "/Users/shakes/DevProjects/myproject",
            session_id="175e9a56-fcdc-4b68-a466-a9145156bdfa",
        )
        if session_id:
            # Use session_id for authoritative session resolution.
            # Use user_turn_uuid to look up the matching UserQuery event in DB.
    """
    # Return cached result if available.
    if tool_use_id in _cache:
        return _cache[tool_use_id]

    result: tuple[str | None, str | None] = (None, None)

    try:
        transcript_dir = _get_transcript_dir(project_dir)
        # Pass session_id so _find_transcript_file opens the exact file rather
        # than falling back to mtime ordering (which breaks with parallel sessions).
        transcript_path = _find_transcript_file(transcript_dir, session_id=session_id)

        if transcript_path is None:
            logger.debug(
                f"No transcript found for project_dir={project_dir}; "
                "falling back to existing parent resolution"
            )
        else:
            logger.debug(
                f"Reading transcript {transcript_path.name} for "
                f"tool_use_id={tool_use_id}"
            )
            result = _parse_transcript(transcript_path, tool_use_id)

    except Exception as e:
        logger.debug(f"Transcript lookup failed for tool_use_id={tool_use_id}: {e}")

    _cache[tool_use_id] = result
    return result


def clear_cache() -> None:
    """
    Clear the module-level result cache.

    Intended for use in tests that need isolated state.
    """
    _cache.clear()


__all__ = [
    "find_parent_user_query",
    "get_transcript_path",
    "get_subagent_transcript_path",
    "clear_cache",
]
