from __future__ import annotations

"""Gemini CLI Native Session Ingester.

Parses Gemini CLI session files from ~/.gemini/tmp/<project-hash>/chats/
and creates/updates HtmlGraph session records using the SDK.

Gemini stores sessions as JSON files with the structure:
    {
        "sessionId": "<uuid>",
        "projectHash": "<sha256>",
        "startTime": "<iso-timestamp>",
        "lastUpdated": "<iso-timestamp>",
        "messages": [
            {
                "id": "<uuid>",
                "timestamp": "<iso-timestamp>",
                "type": "user" | "gemini" | "info",
                "content": "<text>",
                # For type=="gemini" with tool calls:
                "toolCalls": [...],
                "thoughts": "...",
                "model": "...",
                "tokens": {...}
            }
        ]
    }
"""

import json
import logging
from dataclasses import dataclass, field
from datetime import datetime
from pathlib import Path
from typing import Any

logger = logging.getLogger(__name__)

# Default Gemini CLI session storage locations (checked in order)
_GEMINI_DEFAULT_PATHS = [
    Path.home() / ".gemini" / "tmp",
    Path.home() / ".config" / "gemini" / "tmp",
]


@dataclass
class GeminiMessage:
    """A single message from a Gemini session file."""

    id: str
    timestamp: datetime
    msg_type: str  # "user", "gemini", "info"
    content: str
    tool_calls: list[dict[str, Any]] = field(default_factory=list)
    thoughts: str | None = None
    model: str | None = None
    tokens: dict[str, Any] | None = None

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> GeminiMessage:
        """Parse a message dict from a Gemini session JSON file."""
        ts_str = data.get("timestamp", "")
        try:
            timestamp = datetime.fromisoformat(ts_str.replace("Z", "+00:00"))
        except (ValueError, AttributeError):
            timestamp = datetime.now()

        return cls(
            id=data.get("id", ""),
            timestamp=timestamp,
            msg_type=data.get("type", "unknown"),
            content=data.get("content", ""),
            tool_calls=data.get("toolCalls", []),
            thoughts=data.get("thoughts") or None,
            model=data.get("model") or None,
            tokens=data.get("tokens") or None,
        )


@dataclass
class GeminiSession:
    """Parsed representation of a Gemini CLI session."""

    session_id: str
    project_hash: str
    start_time: datetime
    last_updated: datetime
    messages: list[GeminiMessage]
    source_file: Path

    @property
    def user_turn_count(self) -> int:
        return sum(1 for m in self.messages if m.msg_type == "user")

    @property
    def gemini_turn_count(self) -> int:
        return sum(1 for m in self.messages if m.msg_type == "gemini")

    @property
    def tool_call_count(self) -> int:
        return sum(len(m.tool_calls) for m in self.messages)

    @property
    def tool_names_used(self) -> list[str]:
        names: list[str] = []
        for m in self.messages:
            for tc in m.tool_calls:
                name = tc.get("name", "")
                if name and name not in names:
                    names.append(name)
        return names

    @property
    def first_user_prompt(self) -> str | None:
        for m in self.messages:
            if m.msg_type == "user" and m.content:
                return m.content[:200]
        return None

    @classmethod
    def from_dict(cls, data: dict[str, Any], source_file: Path) -> GeminiSession:
        """Parse a full Gemini session JSON file."""
        start_time_str = data.get("startTime", "")
        last_updated_str = data.get("lastUpdated", "")
        try:
            start_time = datetime.fromisoformat(start_time_str.replace("Z", "+00:00"))
        except (ValueError, AttributeError):
            start_time = datetime.now()
        try:
            last_updated = datetime.fromisoformat(
                last_updated_str.replace("Z", "+00:00")
            )
        except (ValueError, AttributeError):
            last_updated = datetime.now()

        messages = [GeminiMessage.from_dict(m) for m in data.get("messages", [])]

        return cls(
            session_id=data.get("sessionId", ""),
            project_hash=data.get("projectHash", ""),
            start_time=start_time,
            last_updated=last_updated,
            messages=messages,
            source_file=source_file,
        )


def find_gemini_sessions(base_path: Path | None = None) -> list[Path]:
    """Find all Gemini CLI session JSON files.

    Searches the Gemini CLI session storage directory for session JSON files.
    The structure is: <base_path>/<project-hash>/chats/session-*.json

    Args:
        base_path: Path to search for sessions. If None, checks default locations.

    Returns:
        List of paths to session JSON files, sorted by modification time (newest first).
    """
    search_paths: list[Path] = []

    if base_path is not None:
        search_paths = [Path(base_path)]
    else:
        for candidate in _GEMINI_DEFAULT_PATHS:
            if candidate.exists():
                search_paths.append(candidate)

    if not search_paths:
        logger.debug("No Gemini session directories found")
        return []

    session_files: list[Path] = []
    for search_path in search_paths:
        # Pattern: <search_path>/<project-hash>/chats/session-*.json
        for session_file in search_path.glob("*/chats/session-*.json"):
            session_files.append(session_file)

    # Sort by modification time, newest first
    session_files.sort(key=lambda p: p.stat().st_mtime, reverse=True)
    return session_files


def parse_gemini_session(session_file: Path) -> GeminiSession | None:
    """Parse a single Gemini session JSON file.

    Args:
        session_file: Path to the session JSON file.

    Returns:
        Parsed GeminiSession, or None if parsing fails.
    """
    try:
        with open(session_file, encoding="utf-8") as f:
            data = json.load(f)
        return GeminiSession.from_dict(data, source_file=session_file)
    except (json.JSONDecodeError, OSError, KeyError) as e:
        logger.warning("Failed to parse Gemini session %s: %s", session_file, e)
        return None


@dataclass
class IngestResult:
    """Result of ingesting Gemini sessions into HtmlGraph."""

    ingested: int = 0
    skipped: int = 0
    errors: int = 0
    session_ids: list[str] = field(default_factory=list)
    error_files: list[str] = field(default_factory=list)


def ingest_gemini_sessions(
    graph_dir: str | Path | None = None,
    agent: str = "gemini",
    base_path: Path | None = None,
    limit: int | None = None,
    dry_run: bool = False,
) -> IngestResult:
    """Ingest Gemini CLI sessions into HtmlGraph.

    Discovers Gemini CLI session files, parses them, and creates corresponding
    HtmlGraph session records. Sessions are identified by their Gemini session ID
    and are idempotent - re-ingesting the same session will update it.

    Args:
        graph_dir: Path to .htmlgraph directory. Auto-discovered if None.
        agent: Agent name to attribute sessions to (default: "gemini").
        base_path: Override for Gemini session storage path. If None, uses defaults.
        limit: Maximum number of sessions to ingest. If None, ingest all.
        dry_run: If True, parse and report but do not write to HtmlGraph.

    Returns:
        IngestResult with counts of ingested, skipped, and errored sessions.
    """
    from htmlgraph import SDK

    result = IngestResult()

    # Find session files
    session_files = find_gemini_sessions(base_path=base_path)
    if not session_files:
        logger.info("No Gemini session files found")
        return result

    if limit is not None:
        session_files = session_files[:limit]

    logger.info("Found %d Gemini session files", len(session_files))

    if dry_run:
        for sf in session_files:
            gs = parse_gemini_session(sf)
            if gs is not None:
                result.ingested += 1
                result.session_ids.append(gs.session_id)
            else:
                result.errors += 1
                result.error_files.append(str(sf))
        return result

    # Initialize SDK
    try:
        sdk = SDK(agent=agent, directory=graph_dir)
    except Exception as e:
        logger.error("Failed to initialize HtmlGraph SDK: %s", e)
        result.errors += 1
        return result

    for session_file in session_files:
        gs = parse_gemini_session(session_file)
        if gs is None:
            result.errors += 1
            result.error_files.append(str(session_file))
            continue

        try:
            _ingest_single_session(sdk, gs)
            result.ingested += 1
            result.session_ids.append(gs.session_id)
            logger.debug("Ingested session %s", gs.session_id)
        except Exception as e:
            logger.warning("Failed to ingest session %s: %s", gs.session_id, e)
            result.errors += 1
            result.error_files.append(str(session_file))

    return result


def _ingest_single_session(sdk: Any, gs: GeminiSession) -> None:
    """Ingest a single GeminiSession into HtmlGraph via the SDK.

    Uses start_session with the Gemini session ID for idempotency.
    Attaches metadata as session title and notes.
    """
    # Build a descriptive title from the first user prompt
    title_parts: list[str] = ["gemini"]
    if gs.first_user_prompt:
        # Trim and normalize whitespace
        prompt_preview = " ".join(gs.first_user_prompt.split())[:80]
        title_parts.append(prompt_preview)
    title = ": ".join(title_parts)

    # Build handoff notes summarising the session
    tool_names = gs.tool_names_used
    notes_parts = [
        f"Source: {gs.source_file.name}",
        f"Project hash: {gs.project_hash[:12]}",
        f"Messages: {len(gs.messages)} "
        f"(user={gs.user_turn_count}, gemini={gs.gemini_turn_count})",
        f"Tool calls: {gs.tool_call_count}",
    ]
    if tool_names:
        notes_parts.append(f"Tools used: {', '.join(tool_names[:10])}")
    notes = " | ".join(notes_parts)

    # Use a deterministic session ID derived from the Gemini session ID
    htmlgraph_session_id = f"gemini-{gs.session_id}"

    session = sdk.session_manager.start_session(
        session_id=htmlgraph_session_id,
        agent="gemini",
        title=title,
    )

    # Update session timing to match the original Gemini session
    session.started_at = gs.start_time
    session.last_activity = gs.last_updated
    session.handoff_notes = notes

    # End the session since Gemini sessions are historical
    sdk.session_manager.session_converter.save(session)

    # Mark as done (ended) since these are imported historical sessions
    session.status = "done"
    if not session.ended_at:
        session.ended_at = gs.last_updated
    sdk.session_manager.session_converter.save(session)
