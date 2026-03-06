from __future__ import annotations

"""
Claude Code Native Session Ingester.

Reads Claude Code's native session format (JSONL files in ~/.claude/projects/*/),
parses conversation turns, tool calls, and results, and creates HtmlGraph
session HTML files from the ingested data.

Claude Code stores sessions as JSONL files:
    ~/.claude/projects/[encoded-path]/[session-uuid].jsonl

Each line is a JSON object representing a message or event in the conversation.
"""

import logging
from dataclasses import dataclass, field
from datetime import datetime
from pathlib import Path
from typing import Any

logger = logging.getLogger(__name__)


@dataclass
class IngestResult:
    """Result of ingesting a single Claude Code session."""

    session_id: str
    """Claude Code session UUID."""

    htmlgraph_session_id: str
    """The HtmlGraph session ID that was created or updated."""

    imported: int = 0
    """Number of events imported into the HtmlGraph session."""

    skipped: int = 0
    """Number of entries skipped (non-user/tool types)."""

    total_entries: int = 0
    """Total entries in the source JSONL file."""

    was_existing: bool = False
    """Whether this session already existed in HtmlGraph (updated vs created)."""

    output_path: Path | None = None
    """Path to the written HtmlGraph session HTML file."""

    errors: list[str] = field(default_factory=list)
    """Any non-fatal errors encountered during ingestion."""

    @property
    def success(self) -> bool:
        """True if ingestion completed without fatal errors."""
        return self.imported >= 0

    def __str__(self) -> str:
        status = "updated" if self.was_existing else "created"
        return (
            f"Session {self.session_id[:8]}... → {self.htmlgraph_session_id} "
            f"({status}, {self.imported} events imported, {self.skipped} skipped)"
        )


@dataclass
class IngestSummary:
    """Summary of a batch ingestion run."""

    sessions_processed: int = 0
    sessions_created: int = 0
    sessions_updated: int = 0
    sessions_skipped: int = 0
    total_events_imported: int = 0
    errors: list[str] = field(default_factory=list)
    results: list[IngestResult] = field(default_factory=list)

    def __str__(self) -> str:
        lines = [
            f"Ingested {self.sessions_processed} Claude Code sessions:",
            f"  Created: {self.sessions_created}",
            f"  Updated: {self.sessions_updated}",
            f"  Skipped: {self.sessions_skipped}",
            f"  Total events: {self.total_events_imported}",
        ]
        if self.errors:
            lines.append(f"  Errors: {len(self.errors)}")
        return "\n".join(lines)


class ClaudeCodeIngester:
    """
    Ingests Claude Code native JSONL session files into HtmlGraph.

    Reads ~/.claude/projects/[encoded-path]/*.jsonl files, parses each
    conversation turn (user messages, tool calls, results), and creates
    or updates HtmlGraph session HTML files.

    Usage::

        ingester = ClaudeCodeIngester(graph_dir=Path(".htmlgraph"))

        # Ingest sessions for the current project
        summary = ingester.ingest_project(project_path=Path.cwd())

        # Ingest from a specific directory
        summary = ingester.ingest_from_path(Path("~/.claude/projects/my-project"))

        # Ingest a single JSONL file
        result = ingester.ingest_file(Path("session.jsonl"))
    """

    def __init__(
        self,
        graph_dir: Path | str,
        agent: str = "claude-code",
        overwrite: bool = False,
    ) -> None:
        """
        Initialize the ingester.

        Args:
            graph_dir: Path to .htmlgraph directory.
            agent: Agent name to assign to ingested sessions.
            overwrite: If True, re-ingest sessions that already exist.
        """
        self.graph_dir = Path(graph_dir)
        self.agent = agent
        self.overwrite = overwrite

        self._sessions_dir = self.graph_dir / "sessions"
        self._sessions_dir.mkdir(parents=True, exist_ok=True)

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------

    def ingest_project(
        self,
        project_path: Path | str | None = None,
        limit: int | None = None,
        since: datetime | None = None,
    ) -> IngestSummary:
        """
        Ingest all Claude Code sessions for a project.

        Args:
            project_path: Path to project directory. Defaults to cwd.
            limit: Maximum number of sessions to ingest (newest first).
            since: Only ingest sessions started after this datetime.

        Returns:
            IngestSummary with counts and per-session results.
        """
        from htmlgraph.transcript import TranscriptReader

        if project_path is None:
            project_path = Path.cwd()

        project_path = Path(project_path)
        reader = TranscriptReader()

        project_dir = reader.find_project_dir(project_path)
        if project_dir is None:
            logger.warning(f"No Claude Code sessions found for project: {project_path}")
            return IngestSummary()

        return self._ingest_directory(project_dir, limit=limit, since=since)

    def ingest_from_path(
        self,
        path: Path | str,
        limit: int | None = None,
        since: datetime | None = None,
    ) -> IngestSummary:
        """
        Ingest all Claude Code sessions from a directory of JSONL files.

        Args:
            path: Directory containing *.jsonl session files, or a single
                  *.jsonl file.
            limit: Maximum number of sessions to ingest (newest first).
            since: Only ingest sessions started after this datetime.

        Returns:
            IngestSummary with counts and per-session results.
        """
        path = Path(path).expanduser()

        if path.is_file():
            result = self.ingest_file(path)
            summary = IngestSummary()
            self._add_result_to_summary(summary, result)
            return summary

        if not path.is_dir():
            return IngestSummary(
                errors=[f"Path does not exist or is not a directory: {path}"]
            )

        return self._ingest_directory(path, limit=limit, since=since)

    def ingest_file(self, jsonl_path: Path | str) -> IngestResult:
        """
        Ingest a single Claude Code JSONL session file.

        Args:
            jsonl_path: Path to the .jsonl session file.

        Returns:
            IngestResult describing what was imported.
        """
        from htmlgraph.transcript import TranscriptReader

        jsonl_path = Path(jsonl_path)
        reader = TranscriptReader()

        transcript = reader.read_transcript(jsonl_path)
        return self._ingest_transcript(transcript)

    # ------------------------------------------------------------------
    # Internal helpers
    # ------------------------------------------------------------------

    def _ingest_directory(
        self,
        directory: Path,
        limit: int | None = None,
        since: datetime | None = None,
    ) -> IngestSummary:
        """Ingest all JSONL files in a directory."""
        from htmlgraph.transcript import TranscriptReader

        reader = TranscriptReader()
        summary = IngestSummary()

        # Collect and sort JSONL files by modification time (newest first)
        jsonl_files = sorted(
            directory.glob("*.jsonl"),
            key=lambda p: p.stat().st_mtime,
            reverse=True,
        )

        if not jsonl_files:
            logger.info(f"No JSONL files found in: {directory}")
            return summary

        processed = 0
        for jsonl_file in jsonl_files:
            if limit is not None and processed >= limit:
                break

            try:
                transcript = reader.read_transcript(jsonl_file)

                # Filter by time if requested
                if since is not None and transcript.started_at is not None:
                    from datetime import timezone

                    def _to_utc(dt: datetime) -> datetime:
                        if dt.tzinfo is None:
                            return dt.replace(tzinfo=timezone.utc)
                        return dt.astimezone(timezone.utc)

                    if _to_utc(transcript.started_at) < _to_utc(since):
                        summary.sessions_skipped += 1
                        continue

                result = self._ingest_transcript(transcript)
                self._add_result_to_summary(summary, result)
                processed += 1

            except Exception as e:
                msg = f"Failed to ingest {jsonl_file.name}: {e}"
                logger.error(msg)
                summary.errors.append(msg)

        return summary

    def _ingest_transcript(self, transcript: Any) -> IngestResult:
        """
        Ingest a parsed TranscriptSession into HtmlGraph.

        Creates a new HtmlGraph session HTML file (or updates an existing one
        if overwrite=True and the session was previously ingested).

        Args:
            transcript: A TranscriptSession from TranscriptReader.

        Returns:
            IngestResult describing what was created/updated.
        """
        from htmlgraph.converter import SessionConverter
        from htmlgraph.event_log import EventRecord, JsonlEventLog
        from htmlgraph.ids import generate_id
        from htmlgraph.models import ActivityEntry, Session

        session_id = transcript.session_id
        converter = SessionConverter(self._sessions_dir)

        # Check whether we already have an HtmlGraph session linked to this
        # Claude Code session transcript
        existing_session = self._find_existing_session(session_id, converter)

        if existing_session is not None and not self.overwrite:
            logger.debug(
                f"Session {session_id[:8]}... already ingested as "
                f"{existing_session.id}; skipping (use overwrite=True to re-ingest)"
            )
            return IngestResult(
                session_id=session_id,
                htmlgraph_session_id=existing_session.id,
                was_existing=True,
                total_entries=len(transcript.entries),
                skipped=len(transcript.entries),
                output_path=self._sessions_dir / f"{existing_session.id}.html",
            )

        # Determine HtmlGraph session ID
        if existing_session is not None:
            htmlgraph_id = existing_session.id
        else:
            # Generate a short deterministic-ish ID from the Claude session UUID
            short = session_id.replace("-", "")[:8]
            htmlgraph_id = f"sess-{short}"

            # Avoid collisions: if this ID already exists (for a different transcript),
            # generate a fresh one
            if converter.exists(htmlgraph_id):
                htmlgraph_id = generate_id("sess")

        # Build Session model
        session = Session(
            id=htmlgraph_id,
            agent=self.agent,
            status="ended",
            started_at=transcript.started_at or datetime.now(),
            ended_at=transcript.ended_at,
            last_activity=transcript.ended_at
            or transcript.started_at
            or datetime.now(),
            # Link back to source transcript
            transcript_id=session_id,
            transcript_path=str(transcript.path),
            transcript_synced_at=datetime.now(),
            transcript_git_branch=transcript.git_branch,
        )

        if existing_session is not None:
            # Preserve fields from the existing session that we should not overwrite
            session.title = existing_session.title
            session.worked_on = existing_session.worked_on
            session.handoff_notes = existing_session.handoff_notes
            session.recommended_next = existing_session.recommended_next
            session.blockers = existing_session.blockers

        # Import events from transcript entries
        imported = 0
        skipped = 0
        result_errors: list[str] = []

        events_dir = self.graph_dir / "events"
        event_log = JsonlEventLog(events_dir)

        for entry in transcript.entries:
            if entry.entry_type not in ("user", "tool_use"):
                skipped += 1
                continue

            try:
                if entry.entry_type == "user":
                    activity = ActivityEntry(
                        id=f"tx-{entry.uuid[:8]}",
                        timestamp=entry.timestamp,
                        tool="UserQuery",
                        summary=entry.to_summary(),
                        success=True,
                        payload={
                            "source": "claude-code-ingest",
                            "transcript_uuid": entry.uuid,
                            "message_content": (entry.message_content or "")[:500],
                        },
                    )
                else:  # tool_use
                    activity = ActivityEntry(
                        id=f"tx-{entry.uuid[:8]}",
                        timestamp=entry.timestamp,
                        tool=entry.tool_name or "Unknown",
                        summary=entry.to_summary(),
                        success=True,
                        payload={
                            "source": "claude-code-ingest",
                            "transcript_uuid": entry.uuid,
                            "tool_input": _truncate_tool_input(entry.tool_input),
                        },
                    )

                session.add_activity(activity)
                imported += 1

                # Write to event log
                try:
                    event_log.append(
                        EventRecord(
                            event_id=activity.id or "",
                            timestamp=activity.timestamp,
                            session_id=htmlgraph_id,
                            agent=self.agent,
                            tool=activity.tool,
                            summary=activity.summary,
                            success=True,
                            feature_id=None,
                            drift_score=None,
                            start_commit=None,
                            continued_from=None,
                            work_type=None,
                            session_status="ended",
                            payload=activity.payload
                            if isinstance(activity.payload, dict)
                            else None,
                        )
                    )
                except Exception as e:
                    logger.debug(f"Failed to write event log entry: {e}")

            except Exception as e:
                msg = f"Entry {entry.uuid[:8]}: {e}"
                logger.warning(msg)
                result_errors.append(msg)
                skipped += 1

        # Persist the session HTML file
        output_path = converter.save(session)

        return IngestResult(
            session_id=session_id,
            htmlgraph_session_id=htmlgraph_id,
            imported=imported,
            skipped=skipped,
            total_entries=len(transcript.entries),
            was_existing=existing_session is not None,
            output_path=output_path,
            errors=result_errors,
        )

    def _find_existing_session(self, transcript_id: str, converter: Any) -> Any | None:
        """Find an HtmlGraph session already linked to a transcript ID."""
        try:
            for session in converter.load_all():
                if session.transcript_id == transcript_id:
                    return session
        except Exception as e:
            logger.debug(f"Error searching for existing session: {e}")
        return None

    @staticmethod
    def _add_result_to_summary(summary: IngestSummary, result: IngestResult) -> None:
        """Update summary counts from a single result."""
        summary.sessions_processed += 1
        summary.results.append(result)

        if result.was_existing:
            if result.imported > 0:
                summary.sessions_updated += 1
            else:
                summary.sessions_skipped += 1
        else:
            summary.sessions_created += 1

        summary.total_events_imported += result.imported
        summary.errors.extend(result.errors)


def _truncate_tool_input(
    tool_input: dict[str, Any] | None, max_value_len: int = 200
) -> dict[str, Any] | None:
    """Truncate long string values in tool_input to avoid huge HTML files."""
    if tool_input is None:
        return None
    truncated: dict[str, Any] = {}
    for k, v in tool_input.items():
        if isinstance(v, str) and len(v) > max_value_len:
            truncated[k] = v[:max_value_len] + "...[truncated]"
        else:
            truncated[k] = v
    return truncated
