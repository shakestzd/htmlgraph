"""
Semantic Compaction for HtmlGraph work items.

Two-tier in-place content shrinking using AI-powered summarization:
- Tier 1 (30+ days): AI summary replaces full content (~30% of original)
- Tier 2 (90+ days): One-liner + metadata only

Originals are archived to .htmlgraph/archive/{item_id}.html before compaction.
"""

from __future__ import annotations

import logging
import shutil
import sqlite3
from dataclasses import dataclass, field
from datetime import datetime, timedelta, timezone
from pathlib import Path

from htmlgraph.converter import html_to_node
from htmlgraph.models import Node

logger = logging.getLogger(__name__)


# ---------------------------------------------------------------------------
# Haiku summarization (optional dependency)
# ---------------------------------------------------------------------------

_HAIKU_MODEL = "claude-haiku-4-5-20251001"

_TIER1_SYSTEM_PROMPT = (
    "You are a technical summarizer. Condense the following work-item content "
    "to roughly 30% of its original length. Preserve key decisions, outcomes, "
    "and any unresolved issues. Output plain text only, no markdown headers."
)

_TIER2_SYSTEM_PROMPT = (
    "You are a technical summarizer. Produce a single sentence (max 120 chars) "
    "that captures what this work item was about and its outcome. "
    "Output plain text only."
)


def _summarize_with_haiku(content: str, system_prompt: str) -> str | None:
    """Call Claude Haiku for summarization. Returns None if unavailable."""
    try:
        import anthropic  # type: ignore[import-untyped]
    except ImportError:
        return None

    try:
        client = anthropic.Anthropic()
        message = client.messages.create(
            model=_HAIKU_MODEL,
            max_tokens=1024,
            system=system_prompt,
            messages=[{"role": "user", "content": content}],
        )
        # Extract text from the response
        if message.content and len(message.content) > 0:
            text = getattr(message.content[0], "text", None)
            if isinstance(text, str):
                return text
    except Exception as exc:
        logger.warning("Haiku summarization failed: %s", exc)

    return None


def _truncation_fallback(content: str, *, ratio: float = 0.3) -> str:
    """Truncate content to approximately *ratio* of its original length."""
    target_len = max(int(len(content) * ratio), 80)
    if len(content) <= target_len:
        return content
    return content[:target_len].rsplit(" ", 1)[0] + " [...]"


# ---------------------------------------------------------------------------
# CompactionResult
# ---------------------------------------------------------------------------


@dataclass
class CompactionResult:
    """Result of a compaction run."""

    tier1_compacted: list[str] = field(default_factory=list)
    tier2_compacted: list[str] = field(default_factory=list)
    skipped: list[str] = field(default_factory=list)
    errors: list[str] = field(default_factory=list)
    dry_run: bool = False

    @property
    def total_compacted(self) -> int:
        return len(self.tier1_compacted) + len(self.tier2_compacted)


# ---------------------------------------------------------------------------
# SemanticCompactor
# ---------------------------------------------------------------------------


class SemanticCompactor:
    """Two-tier semantic compaction for HtmlGraph work items.

    Tier 1 (default 30+ days old):
        Full content replaced by an AI-generated summary (~30% of original).
        Original archived to ``archive/{item_id}.html``.

    Tier 2 (default 90+ days old):
        Content reduced to a single-line summary + metadata only.
        Full content in ``archive/{item_id}.html``.

    If the ``anthropic`` package is not installed, a simple truncation
    fallback is used instead of Haiku summarization.
    """

    def __init__(
        self,
        db_path: str | Path,
        htmlgraph_dir: str | Path,
        *,
        dry_run: bool = False,
    ) -> None:
        self.db_path = Path(db_path)
        self.htmlgraph_dir = Path(htmlgraph_dir)
        self.archive_dir = self.htmlgraph_dir / "archive"
        self.dry_run = dry_run

        # Ensure archive dir exists
        if not self.dry_run:
            self.archive_dir.mkdir(parents=True, exist_ok=True)

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------

    def compact(
        self,
        tier1_days: int = 30,
        tier2_days: int = 90,
    ) -> CompactionResult:
        """Run semantic compaction across all eligible work items.

        Args:
            tier1_days: Items older than this get tier-1 compaction.
            tier2_days: Items older than this get tier-2 compaction.

        Returns:
            CompactionResult with lists of affected item IDs.
        """
        result = CompactionResult(dry_run=self.dry_run)
        now = datetime.now(timezone.utc)
        tier1_cutoff = now - timedelta(days=tier1_days)
        tier2_cutoff = now - timedelta(days=tier2_days)

        # Scan entity directories for eligible HTML files
        entity_dirs = ["features", "bugs", "spikes"]
        for dir_name in entity_dirs:
            entity_dir = self.htmlgraph_dir / dir_name
            if not entity_dir.is_dir():
                continue

            for html_file in sorted(entity_dir.glob("*.html")):
                try:
                    self._process_item(
                        html_file,
                        tier1_cutoff=tier1_cutoff,
                        tier2_cutoff=tier2_cutoff,
                        result=result,
                    )
                except Exception as exc:
                    item_id = html_file.stem
                    logger.warning("Error compacting %s: %s", item_id, exc)
                    result.errors.append(f"{item_id}: {exc}")

        return result

    def restore(self, item_id: str) -> bool:
        """Restore a compacted item from its archive.

        Copies the original HTML from ``archive/{item_id}.html`` back to
        its entity directory and resets compaction metadata in SQLite.

        Args:
            item_id: The ID of the work item to restore.

        Returns:
            True if restored successfully, False if archive not found.
        """
        archive_path = self.archive_dir / f"{item_id}.html"
        if not archive_path.exists():
            logger.warning("Archive not found for %s", item_id)
            return False

        # Find the current (compacted) file
        target_path = self._find_item_file(item_id)
        if target_path is None:
            # Item was removed; determine type from archived node
            try:
                node = html_to_node(archive_path)
                entity_type = node.type
            except Exception:
                entity_type = "feature"
            target_dir = self.htmlgraph_dir / f"{entity_type}s"
            target_dir.mkdir(parents=True, exist_ok=True)
            target_path = target_dir / f"{item_id}.html"

        # Copy archive back as the live file
        shutil.copy2(archive_path, target_path)

        # Reset compaction metadata in SQLite
        self._reset_compaction_metadata(item_id)

        # Remove archive copy
        archive_path.unlink(missing_ok=True)

        logger.info("Restored %s from archive", item_id)
        return True

    # ------------------------------------------------------------------
    # Internal helpers
    # ------------------------------------------------------------------

    def _process_item(
        self,
        html_file: Path,
        *,
        tier1_cutoff: datetime,
        tier2_cutoff: datetime,
        result: CompactionResult,
    ) -> None:
        """Evaluate and optionally compact a single item."""
        node = html_to_node(html_file)
        item_id = node.id

        # Determine item age — use updated timestamp, make tz-aware if needed
        item_time = node.updated
        if item_time.tzinfo is None:
            item_time = item_time.replace(tzinfo=timezone.utc)

        # Already compacted at this tier or higher? Skip.
        current_tier = self._get_current_tier(item_id)

        # Determine target tier
        if item_time <= tier2_cutoff:
            target_tier = 2
        elif item_time <= tier1_cutoff:
            target_tier = 1
        else:
            result.skipped.append(item_id)
            return

        # Skip if already at target tier or higher
        if current_tier >= target_tier:
            result.skipped.append(item_id)
            return

        # Skip items with no meaningful content to compact
        content = node.content or ""
        if len(content.strip()) < 50:
            result.skipped.append(item_id)
            return

        if self.dry_run:
            if target_tier == 2:
                result.tier2_compacted.append(item_id)
            else:
                result.tier1_compacted.append(item_id)
            return

        # Archive the original before modifying
        if current_tier == 0:
            # First compaction — archive the pristine original
            self._archive_original(html_file, item_id)

        # Generate summary
        if target_tier == 2:
            summary = self._summarize_tier2(content)
            result.tier2_compacted.append(item_id)
        else:
            summary = self._summarize_tier1(content)
            result.tier1_compacted.append(item_id)

        # Write compacted content back to the HTML file
        self._write_compacted(html_file, node, summary, target_tier)

        # Update SQLite metadata
        self._update_compaction_metadata(item_id, target_tier, summary)

    def _summarize_tier1(self, content: str) -> str:
        """Generate tier-1 summary (~30% of original)."""
        summary = _summarize_with_haiku(content, _TIER1_SYSTEM_PROMPT)
        if summary is None:
            summary = _truncation_fallback(content, ratio=0.3)
        return summary

    def _summarize_tier2(self, content: str) -> str:
        """Generate tier-2 summary (one-liner)."""
        summary = _summarize_with_haiku(content, _TIER2_SYSTEM_PROMPT)
        if summary is None:
            summary = _truncation_fallback(content, ratio=0.05)
            # Ensure single line
            summary = summary.replace("\n", " ").strip()
            if len(summary) > 120:
                summary = summary[:117] + "..."
        return summary

    def _archive_original(self, html_file: Path, item_id: str) -> None:
        """Copy original HTML to the archive directory."""
        dest = self.archive_dir / f"{item_id}.html"
        shutil.copy2(html_file, dest)
        logger.debug("Archived original: %s -> %s", html_file, dest)

    def _write_compacted(
        self, html_file: Path, node: Node, summary: str, tier: int
    ) -> None:
        """Overwrite the HTML file with compacted content."""
        node.content = summary
        node.properties["compacted_tier"] = str(tier)
        node.properties["compacted_at"] = datetime.now(timezone.utc).isoformat()
        html_content = node.to_html()
        html_file.write_text(html_content, encoding="utf-8")

    def _find_item_file(self, item_id: str) -> Path | None:
        """Find the live HTML file for an item across entity directories."""
        for dir_name in ["features", "bugs", "spikes"]:
            path = self.htmlgraph_dir / dir_name / f"{item_id}.html"
            if path.exists():
                return path
        return None

    # ------------------------------------------------------------------
    # SQLite metadata
    # ------------------------------------------------------------------

    def _get_db_connection(self) -> sqlite3.Connection:
        """Open a connection to the HtmlGraph database."""
        conn = sqlite3.connect(str(self.db_path), check_same_thread=False)
        conn.execute("PRAGMA journal_mode=WAL")
        return conn

    def _get_current_tier(self, item_id: str) -> int:
        """Read current compacted_tier from SQLite. Returns 0 if not set."""
        if not self.db_path.exists():
            return 0
        try:
            conn = self._get_db_connection()
            cursor = conn.execute(
                "SELECT compacted_tier FROM features WHERE id = ?", (item_id,)
            )
            row = cursor.fetchone()
            conn.close()
            if row and row[0] is not None:
                return int(row[0])
        except (sqlite3.OperationalError, sqlite3.DatabaseError) as exc:
            logger.debug("Could not read compacted_tier for %s: %s", item_id, exc)
        return 0

    def _update_compaction_metadata(
        self, item_id: str, tier: int, summary: str
    ) -> None:
        """Write compaction metadata to SQLite."""
        if not self.db_path.exists():
            return
        try:
            conn = self._get_db_connection()
            conn.execute(
                """
                UPDATE features
                SET compacted_tier = ?,
                    compacted_summary = ?,
                    compacted_at = ?
                WHERE id = ?
                """,
                (tier, summary, datetime.now(timezone.utc).isoformat(), item_id),
            )
            conn.commit()
            conn.close()
        except (sqlite3.OperationalError, sqlite3.DatabaseError) as exc:
            logger.warning(
                "Could not update compaction metadata for %s: %s", item_id, exc
            )

    def _reset_compaction_metadata(self, item_id: str) -> None:
        """Clear compaction metadata in SQLite (used during restore)."""
        if not self.db_path.exists():
            return
        try:
            conn = self._get_db_connection()
            conn.execute(
                """
                UPDATE features
                SET compacted_tier = 0,
                    compacted_summary = NULL,
                    compacted_at = NULL
                WHERE id = ?
                """,
                (item_id,),
            )
            conn.commit()
            conn.close()
        except (sqlite3.OperationalError, sqlite3.DatabaseError) as exc:
            logger.warning(
                "Could not reset compaction metadata for %s: %s", item_id, exc
            )
