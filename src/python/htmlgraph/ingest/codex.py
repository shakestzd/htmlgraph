from __future__ import annotations

"""OpenAI Codex CLI Session Ingester.

Parses OpenAI Codex CLI session files from ~/.codex/sessions/*.json
and creates/updates HtmlGraph session records using the SDK.

Codex CLI stores sessions as JSON files with the structure:
    ~/.codex/sessions/rollout-<date>-<uuid>.json

Each file has the structure:
    {
        "session": {
            "timestamp": "<iso-timestamp>",
            "id": "<uuid>",
            "instructions": "<system prompt>"
        },
        "items": [
            {
                "role": "user",
                "content": [{"type": "input_text", "text": "..."}],
                "type": "message"
            },
            {
                "id": "<id>",
                "type": "reasoning",
                "summary": [...],
                "duration_ms": 1234
            },
            {
                "id": "<id>",
                "type": "function_call",
                "status": "completed",
                "arguments": "{...}",
                "call_id": "<call-id>",
                "name": "<tool-name>"
            },
            {
                "type": "function_call_output",
                "call_id": "<call-id>",
                "output": "{...}"
            },
            {
                "role": "assistant",
                "content": [{"type": "output_text", "text": "..."}],
                "type": "message"
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

# Default Codex CLI session storage locations (checked in order)
_CODEX_DEFAULT_PATHS = [
    Path.home() / ".codex" / "sessions",
]


@dataclass
class CodexItem:
    """A single item from a Codex CLI session."""

    item_type: str  # "message", "reasoning", "function_call", "function_call_output"
    role: str | None  # "user", "assistant" (for message types)
    content: str  # Extracted text content
    tool_name: str | None = None  # For function_call items
    tool_arguments: dict[str, Any] | None = None  # Parsed tool args
    duration_ms: int | None = None  # For reasoning items

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> CodexItem:
        """Parse an item dict from a Codex session JSON."""
        item_type = data.get("type", "unknown")
        role = data.get("role") or None
        content = ""
        tool_name: str | None = None
        tool_arguments: dict[str, Any] | None = None
        duration_ms: int | None = None

        if item_type == "message":
            content_list = data.get("content", [])
            parts: list[str] = []
            for part in content_list:
                if isinstance(part, dict):
                    text = part.get("text", "")
                    if text:
                        parts.append(str(text))
            content = " ".join(parts)
        elif item_type == "function_call":
            tool_name = data.get("name", "")
            content = f"tool_call: {tool_name}"
            raw_args = data.get("arguments", "{}")
            try:
                tool_arguments = (
                    json.loads(raw_args) if isinstance(raw_args, str) else raw_args
                )
            except (json.JSONDecodeError, TypeError):
                tool_arguments = {"raw": str(raw_args)[:200]}
        elif item_type == "function_call_output":
            output = data.get("output", "")
            content = f"tool_output: {str(output)[:200]}"
        elif item_type == "reasoning":
            summary = data.get("summary", [])
            duration_ms = data.get("duration_ms") or None
            if summary:
                content = str(summary[0])[:200] if summary else "reasoning"
            else:
                content = "reasoning"

        return cls(
            item_type=item_type,
            role=role,
            content=content,
            tool_name=tool_name,
            tool_arguments=tool_arguments,
            duration_ms=duration_ms,
        )


@dataclass
class CodexSession:
    """Parsed representation of a Codex CLI session."""

    session_id: str
    start_time: datetime
    items: list[CodexItem]
    source_file: Path
    instructions: str | None = None

    @property
    def user_message_count(self) -> int:
        return sum(
            1
            for item in self.items
            if item.item_type == "message" and item.role == "user"
        )

    @property
    def assistant_message_count(self) -> int:
        return sum(
            1
            for item in self.items
            if item.item_type == "message" and item.role == "assistant"
        )

    @property
    def tool_call_count(self) -> int:
        return sum(1 for item in self.items if item.item_type == "function_call")

    @property
    def reasoning_count(self) -> int:
        return sum(1 for item in self.items if item.item_type == "reasoning")

    @property
    def tool_names_used(self) -> list[str]:
        names: list[str] = []
        for item in self.items:
            if item.item_type == "function_call" and item.tool_name:
                if item.tool_name not in names:
                    names.append(item.tool_name)
        return names

    @property
    def first_user_message(self) -> str | None:
        for item in self.items:
            if item.item_type == "message" and item.role == "user" and item.content:
                return item.content[:200]
        return None

    @property
    def total_reasoning_ms(self) -> int:
        return sum(
            item.duration_ms
            for item in self.items
            if item.item_type == "reasoning" and item.duration_ms is not None
        )

    @classmethod
    def from_dict(cls, data: dict[str, Any], source_file: Path) -> CodexSession:
        """Parse a full Codex session JSON file."""
        session_meta = data.get("session", {})
        session_id = session_meta.get("id", source_file.stem)
        instructions = session_meta.get("instructions") or None

        ts_str = session_meta.get("timestamp", "")
        try:
            start_time = datetime.fromisoformat(ts_str.replace("Z", "+00:00"))
        except (ValueError, AttributeError):
            start_time = datetime.now()

        items = [CodexItem.from_dict(item) for item in data.get("items", [])]

        return cls(
            session_id=session_id,
            start_time=start_time,
            items=items,
            source_file=source_file,
            instructions=instructions,
        )


def find_codex_sessions(base_path: Path | None = None) -> list[Path]:
    """Find all Codex CLI session JSON files.

    Searches the Codex CLI session storage directory for session JSON files.
    The structure is: <base_path>/rollout-<date>-<uuid>.json

    Args:
        base_path: Path to search for sessions. If None, checks default locations.

    Returns:
        List of paths to session JSON files, sorted by modification time (newest first).
    """
    search_paths: list[Path] = []

    if base_path is not None:
        search_paths = [Path(base_path)]
    else:
        for candidate in _CODEX_DEFAULT_PATHS:
            if candidate.exists():
                search_paths.append(candidate)

    if not search_paths:
        logger.debug("No Codex session directories found")
        return []

    session_files: list[Path] = []
    for search_path in search_paths:
        for session_file in search_path.glob("*.json"):
            session_files.append(session_file)

    # Sort by modification time, newest first
    session_files.sort(key=lambda p: p.stat().st_mtime, reverse=True)
    return session_files


def parse_codex_session(session_file: Path) -> CodexSession | None:
    """Parse a single Codex CLI session JSON file.

    Args:
        session_file: Path to the session JSON file.

    Returns:
        Parsed CodexSession, or None if parsing fails.
    """
    try:
        with open(session_file, encoding="utf-8") as f:
            data = json.load(f)
        return CodexSession.from_dict(data, source_file=session_file)
    except (json.JSONDecodeError, OSError, KeyError) as e:
        logger.warning("Failed to parse Codex session %s: %s", session_file, e)
        return None


@dataclass
class IngestResult:
    """Result of ingesting Codex sessions into HtmlGraph."""

    ingested: int = 0
    skipped: int = 0
    errors: int = 0
    session_ids: list[str] = field(default_factory=list)
    error_files: list[str] = field(default_factory=list)


def ingest_codex_sessions(
    graph_dir: str | Path | None = None,
    agent: str = "codex",
    base_path: Path | None = None,
    limit: int | None = None,
    dry_run: bool = False,
) -> IngestResult:
    """Ingest OpenAI Codex CLI sessions into HtmlGraph.

    Discovers Codex CLI session files, parses them, and creates corresponding
    HtmlGraph session records. Sessions are identified by their Codex session ID
    and are idempotent - re-ingesting the same session will update it.

    Args:
        graph_dir: Path to .htmlgraph directory. Auto-discovered if None.
        agent: Agent name to attribute sessions to (default: "codex").
        base_path: Override for Codex session storage path. If None, uses defaults.
        limit: Maximum number of sessions to ingest. If None, ingest all.
        dry_run: If True, parse and report but do not write to HtmlGraph.

    Returns:
        IngestResult with counts of ingested, skipped, and errored sessions.
    """
    from htmlgraph import SDK

    result = IngestResult()

    # Find session files
    session_files = find_codex_sessions(base_path=base_path)
    if not session_files:
        logger.info("No Codex session files found")
        return result

    if limit is not None:
        session_files = session_files[:limit]

    logger.info("Found %d Codex session files", len(session_files))

    if dry_run:
        for sf in session_files:
            cs = parse_codex_session(sf)
            if cs is not None:
                result.ingested += 1
                result.session_ids.append(cs.session_id)
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
        cs = parse_codex_session(session_file)
        if cs is None:
            result.errors += 1
            result.error_files.append(str(session_file))
            continue

        try:
            _ingest_single_session(sdk, cs)
            result.ingested += 1
            result.session_ids.append(cs.session_id)
            logger.debug("Ingested Codex session %s", cs.session_id)
        except Exception as e:
            logger.warning("Failed to ingest Codex session %s: %s", cs.session_id, e)
            result.errors += 1
            result.error_files.append(str(session_file))

    return result


def _ingest_single_session(sdk: Any, cs: CodexSession) -> None:
    """Ingest a single CodexSession into HtmlGraph via the SDK.

    Uses start_session with the Codex session ID for idempotency.
    Attaches metadata as session title and notes.
    """
    # Build a descriptive title from the first user message
    title_parts: list[str] = ["codex"]
    if cs.first_user_message:
        prompt_preview = " ".join(cs.first_user_message.split())[:80]
        title_parts.append(prompt_preview)
    title = ": ".join(title_parts)

    # Build handoff notes summarising the session
    tool_names = cs.tool_names_used
    notes_parts = [
        f"Source: {cs.source_file.name}",
        f"Items: {len(cs.items)} "
        f"(user={cs.user_message_count}, assistant={cs.assistant_message_count})",
        f"Tool calls: {cs.tool_call_count}",
        f"Reasoning steps: {cs.reasoning_count}",
    ]
    if cs.total_reasoning_ms:
        notes_parts.append(f"Reasoning time: {cs.total_reasoning_ms}ms")
    if tool_names:
        notes_parts.append(f"Tools used: {', '.join(tool_names[:10])}")
    if cs.instructions:
        notes_parts.append(f"Instructions: {cs.instructions[:100]}")
    notes = " | ".join(notes_parts)

    # Use a deterministic session ID derived from the Codex session ID
    htmlgraph_session_id = f"codex-{cs.session_id}"

    session = sdk.session_manager.start_session(
        session_id=htmlgraph_session_id,
        agent="codex",
        title=title,
    )

    # Update session timing to match the original Codex session
    session.started_at = cs.start_time
    session.last_activity = cs.start_time
    session.handoff_notes = notes

    # Persist with updated metadata
    sdk.session_manager.session_converter.save(session)

    # Mark as done (ended) since these are imported historical sessions
    session.status = "done"
    if not session.ended_at:
        session.ended_at = cs.start_time
    sdk.session_manager.session_converter.save(session)
