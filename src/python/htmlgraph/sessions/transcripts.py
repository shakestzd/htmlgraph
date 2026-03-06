"""
Transcript operations - link, import, and query transcript data.

Provides:
- TranscriptOps: link transcripts to sessions, import events, auto-link by branch

Uses SessionConverter for persistence (NOT graph queries).
"""

import logging
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

from htmlgraph.converter import SessionConverter
from htmlgraph.event_log import EventRecord, JsonlEventLog
from htmlgraph.models import ActivityEntry, Session

logger = logging.getLogger(__name__)


class TranscriptOps:
    """Handles transcript linking, import, and statistics."""

    def __init__(
        self,
        session_converter: SessionConverter,
        event_log: JsonlEventLog,
    ):
        self.session_converter = session_converter
        self.event_log = event_log

    def link_transcript(
        self,
        session: Session,
        transcript_id: str,
        transcript_path: str | None = None,
        git_branch: str | None = None,
    ) -> Session:
        """Link a Claude Code transcript to a session.

        Does nothing if the session already has a different transcript_id set,
        to prevent a new session's transcript from overwriting a completed
        session's correct transcript link.
        """
        if session.transcript_id:
            logger.debug(
                f"Session {session.id} already has transcript {session.transcript_id!r}; "
                f"refusing to overwrite with {transcript_id!r}"
            )
            return session

        session.transcript_id = transcript_id
        session.transcript_path = transcript_path
        session.transcript_synced_at = datetime.now()
        if git_branch:
            session.transcript_git_branch = git_branch
        self.session_converter.save(session)
        return session

    def find_session_by_transcript(self, transcript_id: str) -> Session | None:
        """Find a session linked to a transcript."""
        for session in self.session_converter.load_all():
            if session.transcript_id == transcript_id:
                return session
        return None

    def import_transcript_events(
        self,
        session: Session,
        session_id: str,
        transcript_session: Any,
        overwrite: bool = False,
    ) -> dict[str, int | str]:
        """Import events from a transcript into a session."""
        if overwrite:
            session.activity_log = []
            session.event_count = 0

        imported = 0
        skipped = 0

        for entry in transcript_session.entries:
            if entry.entry_type not in ("user", "tool_use"):
                skipped += 1
                continue

            if entry.entry_type == "user":
                activity = ActivityEntry(
                    id=f"tx-{entry.uuid[:8]}",
                    timestamp=entry.timestamp,
                    tool="UserQuery",
                    summary=entry.to_summary(),
                    success=True,
                    payload={
                        "source": "transcript",
                        "transcript_uuid": entry.uuid,
                        "message_content": entry.message_content,
                    },
                )
            elif entry.entry_type == "tool_use":
                activity = ActivityEntry(
                    id=f"tx-{entry.uuid[:8]}",
                    timestamp=entry.timestamp,
                    tool=entry.tool_name or "Unknown",
                    summary=entry.to_summary(),
                    success=True,
                    payload={
                        "source": "transcript",
                        "transcript_uuid": entry.uuid,
                        "tool_input": entry.tool_input,
                        "thinking": entry.thinking,
                    },
                )
            else:
                continue

            session.add_activity(activity)
            imported += 1

            try:
                from htmlgraph.work_type_utils import infer_work_type_from_id

                work_type = infer_work_type_from_id(activity.feature_id)
                self.event_log.append(
                    EventRecord(
                        event_id=activity.id or "",
                        timestamp=activity.timestamp,
                        session_id=session_id,
                        agent=session.agent,
                        tool=activity.tool,
                        summary=activity.summary,
                        success=activity.success,
                        feature_id=activity.feature_id,
                        drift_score=None,
                        start_commit=session.start_commit,
                        continued_from=session.continued_from,
                        work_type=work_type,
                        session_status=session.status,
                        payload=activity.payload
                        if isinstance(activity.payload, dict)
                        else None,
                    )
                )
            except Exception as e:
                logger.warning(f"Failed to append transcript event to event log: {e}")

        # Update transcript link
        session.transcript_id = transcript_session.session_id
        session.transcript_path = str(transcript_session.path)
        session.transcript_synced_at = datetime.now()
        if transcript_session.git_branch:
            session.transcript_git_branch = transcript_session.git_branch

        self.session_converter.save(session)

        return {
            "imported": imported,
            "skipped": skipped,
            "total_entries": len(transcript_session.entries),
        }

    def auto_link_by_branch(
        self,
        git_branch: str,
        graph_dir: Path,
        agent: str | None = None,
    ) -> list[tuple[str, str]]:
        """Auto-link sessions to transcripts based on git branch."""
        from htmlgraph.transcript import TranscriptReader

        linked: list[tuple[str, str]] = []
        reader = TranscriptReader()
        project_path = graph_dir.parent
        transcripts = reader.find_sessions_for_branch(git_branch, project_path)
        if not transcripts:
            return linked

        sessions = self.session_converter.load_all()
        if agent:
            sessions = [s for s in sessions if s.agent == agent]

        def normalize_dt(dt: datetime | None) -> datetime | None:
            if dt is None:
                return None
            if dt.tzinfo is not None:
                return dt.astimezone(timezone.utc).replace(tzinfo=None)
            return dt

        for transcript in transcripts:
            if not transcript.started_at:
                continue
            transcript_start = normalize_dt(transcript.started_at)
            transcript_end = normalize_dt(transcript.ended_at)

            for session in sessions:
                if session.transcript_id:
                    continue
                session_start = normalize_dt(session.started_at)
                session_end = normalize_dt(session.ended_at)

                if session_start and transcript_end:
                    if session_start > transcript_end:
                        continue
                if session_end and transcript_start:
                    if session_end < transcript_start:
                        continue

                self.link_transcript(
                    session=session,
                    transcript_id=transcript.session_id,
                    transcript_path=str(transcript.path),
                    git_branch=git_branch,
                )
                linked.append((session.id, transcript.session_id))
                break
        return linked

    def get_transcript_stats(self, session: Session) -> dict[str, Any] | None:
        """Get transcript statistics for a session."""
        if not session.transcript_id:
            return None

        from htmlgraph.transcript import TranscriptReader

        reader = TranscriptReader()
        transcript = reader.read_session(session.transcript_id)
        if not transcript:
            return {
                "transcript_id": session.transcript_id,
                "error": "transcript_not_found",
            }

        return {
            "transcript_id": session.transcript_id,
            "transcript_path": session.transcript_path,
            "synced_at": session.transcript_synced_at.isoformat()
            if session.transcript_synced_at
            else None,
            "git_branch": session.transcript_git_branch,
            "user_messages": transcript.user_message_count,
            "tool_calls": transcript.tool_call_count,
            "tool_breakdown": transcript.tool_breakdown,
            "duration_seconds": transcript.duration_seconds,
            "has_thinking_traces": transcript.has_thinking_traces(),
            "entry_count": len(transcript.entries),
        }
