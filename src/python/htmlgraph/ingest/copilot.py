from __future__ import annotations

"""GitHub Copilot CLI Session Ingester.

Parses GitHub Copilot CLI session files from ~/.copilot/session-state/*.jsonl
and creates/updates HtmlGraph session records using the SDK.

Copilot CLI stores sessions as JSONL files where each line is a JSON event:
    ~/.copilot/session-state/<session-uuid>.jsonl

Each line has the structure:
    {
        "type": "session.start" | "user.message" | "agent.message" |
                "tool.call" | "tool.result" | "session.error" | ...,
        "data": { ... },
        "id": "<event-uuid>",
        "timestamp": "<iso-timestamp>",
        "parentId": "<parent-event-uuid>" | null
    }

Notable event types:
    - session.start: Contains sessionId, version, copilotVersion, startTime
    - user.message: User input with content and attachments
    - agent.message: Copilot response with content
    - tool.call: Tool invocation with name and arguments
    - tool.result: Tool output
    - session.model_change: Model switched during session
    - session.error: Error events
"""

import json
import logging
from dataclasses import dataclass, field
from datetime import datetime
from pathlib import Path
from typing import Any

logger = logging.getLogger(__name__)

# Default Copilot CLI session storage locations (checked in order)
_COPILOT_DEFAULT_PATHS = [
    Path.home() / ".copilot" / "session-state",
]


@dataclass
class CopilotEvent:
    """A single event from a Copilot CLI session JSONL file."""

    event_id: str
    timestamp: datetime
    event_type: str  # "session.start", "user.message", "tool.call", etc.
    data: dict[str, Any]
    parent_id: str | None = None

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> CopilotEvent:
        """Parse an event dict from a Copilot session JSONL line."""
        ts_str = data.get("timestamp", "")
        try:
            timestamp = datetime.fromisoformat(ts_str.replace("Z", "+00:00"))
        except (ValueError, AttributeError):
            timestamp = datetime.now()

        return cls(
            event_id=data.get("id", ""),
            timestamp=timestamp,
            event_type=data.get("type", "unknown"),
            data=data.get("data", {}),
            parent_id=data.get("parentId") or None,
        )


@dataclass
class CopilotSession:
    """Parsed representation of a Copilot CLI session."""

    session_id: str
    start_time: datetime
    last_event_time: datetime
    events: list[CopilotEvent]
    source_file: Path
    copilot_version: str | None = None
    model: str | None = None

    @property
    def user_message_count(self) -> int:
        return sum(1 for e in self.events if e.event_type == "user.message")

    @property
    def agent_message_count(self) -> int:
        return sum(1 for e in self.events if e.event_type == "agent.message")

    @property
    def tool_call_count(self) -> int:
        return sum(1 for e in self.events if e.event_type == "tool.call")

    @property
    def tool_names_used(self) -> list[str]:
        names: list[str] = []
        for e in self.events:
            if e.event_type == "tool.call":
                name = e.data.get("name", e.data.get("toolName", ""))
                if name and name not in names:
                    names.append(name)
        return names

    @property
    def first_user_message(self) -> str | None:
        for e in self.events:
            if e.event_type == "user.message":
                content = e.data.get("content", "")
                if content:
                    return str(content)[:200]
        return None

    @property
    def error_count(self) -> int:
        return sum(1 for e in self.events if e.event_type == "session.error")


def find_copilot_sessions(base_path: Path | None = None) -> list[Path]:
    """Find all Copilot CLI session JSONL files.

    Searches the Copilot CLI session storage directory for session JSONL files.
    The structure is: <base_path>/<session-uuid>.jsonl

    Args:
        base_path: Path to search for sessions. If None, checks default locations.

    Returns:
        List of paths to session JSONL files, sorted by modification time (newest first).
    """
    search_paths: list[Path] = []

    if base_path is not None:
        search_paths = [Path(base_path)]
    else:
        for candidate in _COPILOT_DEFAULT_PATHS:
            if candidate.exists():
                search_paths.append(candidate)

    if not search_paths:
        logger.debug("No Copilot session directories found")
        return []

    session_files: list[Path] = []
    for search_path in search_paths:
        for session_file in search_path.glob("*.jsonl"):
            session_files.append(session_file)

    # Sort by modification time, newest first
    session_files.sort(key=lambda p: p.stat().st_mtime, reverse=True)
    return session_files


def parse_copilot_session(session_file: Path) -> CopilotSession | None:
    """Parse a single Copilot CLI session JSONL file.

    Args:
        session_file: Path to the session JSONL file.

    Returns:
        Parsed CopilotSession, or None if parsing fails.
    """
    try:
        events: list[CopilotEvent] = []
        with open(session_file, encoding="utf-8") as f:
            for line in f:
                line = line.strip()
                if not line:
                    continue
                try:
                    data = json.loads(line)
                    events.append(CopilotEvent.from_dict(data))
                except json.JSONDecodeError as e:
                    logger.debug("Skipping malformed line in %s: %s", session_file, e)

        if not events:
            logger.debug("No events found in %s", session_file)
            return None

        # Extract session metadata from session.start event
        session_id = session_file.stem  # Use filename as fallback session ID
        start_time = events[0].timestamp
        copilot_version: str | None = None
        model: str | None = None

        for event in events:
            if event.event_type == "session.start":
                session_id = event.data.get("sessionId", session_id)
                copilot_version = event.data.get("copilotVersion") or None
                start_str = event.data.get("startTime", "")
                if start_str:
                    try:
                        start_time = datetime.fromisoformat(
                            start_str.replace("Z", "+00:00")
                        )
                    except (ValueError, AttributeError):
                        pass
            elif event.event_type == "session.model_change":
                model = event.data.get("newModel") or model

        last_event_time = events[-1].timestamp if events else start_time

        return CopilotSession(
            session_id=session_id,
            start_time=start_time,
            last_event_time=last_event_time,
            events=events,
            source_file=session_file,
            copilot_version=copilot_version,
            model=model,
        )
    except (OSError, KeyError) as e:
        logger.warning("Failed to parse Copilot session %s: %s", session_file, e)
        return None


@dataclass
class IngestResult:
    """Result of ingesting Copilot sessions into HtmlGraph."""

    ingested: int = 0
    skipped: int = 0
    errors: int = 0
    session_ids: list[str] = field(default_factory=list)
    error_files: list[str] = field(default_factory=list)


def ingest_copilot_sessions(
    graph_dir: str | Path | None = None,
    agent: str = "copilot",
    base_path: Path | None = None,
    limit: int | None = None,
    dry_run: bool = False,
) -> IngestResult:
    """Ingest GitHub Copilot CLI sessions into HtmlGraph.

    Discovers Copilot CLI session files, parses them, and creates corresponding
    HtmlGraph session records. Sessions are identified by their Copilot session ID
    and are idempotent - re-ingesting the same session will update it.

    Args:
        graph_dir: Path to .htmlgraph directory. Auto-discovered if None.
        agent: Agent name to attribute sessions to (default: "copilot").
        base_path: Override for Copilot session storage path. If None, uses defaults.
        limit: Maximum number of sessions to ingest. If None, ingest all.
        dry_run: If True, parse and report but do not write to HtmlGraph.

    Returns:
        IngestResult with counts of ingested, skipped, and errored sessions.
    """
    from htmlgraph import SDK

    result = IngestResult()

    # Find session files
    session_files = find_copilot_sessions(base_path=base_path)
    if not session_files:
        logger.info("No Copilot session files found")
        return result

    if limit is not None:
        session_files = session_files[:limit]

    logger.info("Found %d Copilot session files", len(session_files))

    if dry_run:
        for sf in session_files:
            cs = parse_copilot_session(sf)
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
        cs = parse_copilot_session(session_file)
        if cs is None:
            result.errors += 1
            result.error_files.append(str(session_file))
            continue

        try:
            _ingest_single_session(sdk, cs)
            result.ingested += 1
            result.session_ids.append(cs.session_id)
            logger.debug("Ingested Copilot session %s", cs.session_id)
        except Exception as e:
            logger.warning("Failed to ingest Copilot session %s: %s", cs.session_id, e)
            result.errors += 1
            result.error_files.append(str(session_file))

    return result


def _ingest_single_session(sdk: Any, cs: CopilotSession) -> None:
    """Ingest a single CopilotSession into HtmlGraph via the SDK.

    Uses start_session with the Copilot session ID for idempotency.
    Attaches metadata as session title and notes.
    """
    # Build a descriptive title from the first user message
    title_parts: list[str] = ["copilot"]
    if cs.first_user_message:
        prompt_preview = " ".join(cs.first_user_message.split())[:80]
        title_parts.append(prompt_preview)
    title = ": ".join(title_parts)

    # Build handoff notes summarising the session
    tool_names = cs.tool_names_used
    notes_parts = [
        f"Source: {cs.source_file.name}",
        f"Events: {len(cs.events)} "
        f"(user={cs.user_message_count}, agent={cs.agent_message_count})",
        f"Tool calls: {cs.tool_call_count}",
    ]
    if cs.model:
        notes_parts.append(f"Model: {cs.model}")
    if cs.copilot_version:
        notes_parts.append(f"Copilot version: {cs.copilot_version}")
    if cs.error_count:
        notes_parts.append(f"Errors: {cs.error_count}")
    if tool_names:
        notes_parts.append(f"Tools used: {', '.join(tool_names[:10])}")
    notes = " | ".join(notes_parts)

    # Use a deterministic session ID derived from the Copilot session ID
    htmlgraph_session_id = f"copilot-{cs.session_id}"

    session = sdk.session_manager.start_session(
        session_id=htmlgraph_session_id,
        agent="copilot",
        title=title,
    )

    # Update session timing to match the original Copilot session
    session.started_at = cs.start_time
    session.last_activity = cs.last_event_time
    session.handoff_notes = notes

    # Persist with updated metadata
    sdk.session_manager.session_converter.save(session)

    # Mark as done (ended) since these are imported historical sessions
    session.status = "done"
    if not session.ended_at:
        session.ended_at = cs.last_event_time
    sdk.session_manager.session_converter.save(session)
