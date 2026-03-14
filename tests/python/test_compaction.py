"""
Tests for semantic compaction (tiered Haiku archival).

Covers:
- Tier 1 compaction (30+ days old): AI summary replaces full content
- Tier 2 compaction (90+ days old): One-liner + metadata only
- Dry-run mode: lists items without modifying
- Restore: compact then restore, verify original restored
- Age threshold: young items skipped
- Idempotent: running twice doesn't re-compact
"""

from __future__ import annotations

import sqlite3
from datetime import datetime, timedelta, timezone
from pathlib import Path
from unittest.mock import MagicMock, patch

import pytest
from htmlgraph.archive.compactor import (
    CompactionResult,
    SemanticCompactor,
    _truncation_fallback,
)
from htmlgraph.models import Node

# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------


@pytest.fixture
def htmlgraph_dir(tmp_path: Path) -> Path:
    """Create a temporary .htmlgraph directory with entity subdirs."""
    hg = tmp_path / ".htmlgraph"
    hg.mkdir()
    (hg / "features").mkdir()
    (hg / "bugs").mkdir()
    (hg / "spikes").mkdir()
    (hg / "archive").mkdir()
    return hg


@pytest.fixture
def db_path(htmlgraph_dir: Path) -> Path:
    """Create an initialised test database with features table + compaction columns."""
    db = htmlgraph_dir / "htmlgraph.db"
    conn = sqlite3.connect(str(db))
    conn.execute("""
        CREATE TABLE IF NOT EXISTS features (
            id TEXT PRIMARY KEY,
            type TEXT NOT NULL DEFAULT 'feature',
            title TEXT NOT NULL DEFAULT '',
            status TEXT NOT NULL DEFAULT 'done',
            priority TEXT DEFAULT 'medium',
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            compacted_tier INTEGER DEFAULT 0,
            compacted_summary TEXT,
            compacted_at TEXT
        )
    """)
    conn.commit()
    conn.close()
    return db


def _write_feature_html(
    htmlgraph_dir: Path,
    item_id: str,
    title: str,
    content: str,
    updated: datetime,
    *,
    entity_type: str = "feature",
    status: str = "done",
) -> Path:
    """Write a minimal feature HTML file and return its path."""
    node = Node(
        id=item_id,
        title=title,
        type=entity_type,
        status=status,
        content=content,
        created=updated,
        updated=updated,
    )
    dir_name = f"{entity_type}s"
    entity_dir = htmlgraph_dir / dir_name
    entity_dir.mkdir(exist_ok=True)
    html_path = entity_dir / f"{item_id}.html"
    html_path.write_text(node.to_html(), encoding="utf-8")
    return html_path


def _insert_feature_row(db_path: Path, item_id: str, tier: int = 0) -> None:
    """Insert a feature row into the test database."""
    conn = sqlite3.connect(str(db_path))
    conn.execute(
        "INSERT OR REPLACE INTO features (id, type, title, compacted_tier) VALUES (?, 'feature', ?, ?)",
        (item_id, item_id, tier),
    )
    conn.commit()
    conn.close()


# ---------------------------------------------------------------------------
# CompactionResult dataclass tests
# ---------------------------------------------------------------------------


class TestCompactionResult:
    def test_empty_result(self) -> None:
        r = CompactionResult()
        assert r.total_compacted == 0
        assert r.tier1_compacted == []
        assert r.tier2_compacted == []
        assert r.skipped == []
        assert r.errors == []
        assert r.dry_run is False

    def test_total_compacted(self) -> None:
        r = CompactionResult(
            tier1_compacted=["a", "b"],
            tier2_compacted=["c"],
        )
        assert r.total_compacted == 3


# ---------------------------------------------------------------------------
# Truncation fallback tests
# ---------------------------------------------------------------------------


class TestTruncationFallback:
    def test_short_content_unchanged(self) -> None:
        assert _truncation_fallback("hello world") == "hello world"

    def test_long_content_truncated(self) -> None:
        content = "word " * 200  # 1000 chars
        result = _truncation_fallback(content, ratio=0.1)
        assert len(result) < len(content)
        assert result.endswith("[...]")


# ---------------------------------------------------------------------------
# Tier 1 compaction (30+ days)
# ---------------------------------------------------------------------------


class TestTier1Compaction:
    @patch("htmlgraph.archive.compactor._summarize_with_haiku")
    def test_tier1_compaction(
        self, mock_haiku: MagicMock, htmlgraph_dir: Path, db_path: Path
    ) -> None:
        """Item >30 days old gets tier-1 summary."""
        mock_haiku.return_value = "AI-generated tier-1 summary of the content."

        old_date = datetime.now(timezone.utc) - timedelta(days=45)
        original_content = "A " * 100  # Substantial content to compact
        _write_feature_html(
            htmlgraph_dir, "feat-old1", "Old Feature", original_content, old_date
        )
        _insert_feature_row(db_path, "feat-old1")

        compactor = SemanticCompactor(db_path=db_path, htmlgraph_dir=htmlgraph_dir)
        result = compactor.compact(tier1_days=30, tier2_days=90)

        assert "feat-old1" in result.tier1_compacted
        assert "feat-old1" not in result.tier2_compacted
        assert result.total_compacted == 1

        # Verify archive was created
        archive_path = htmlgraph_dir / "archive" / "feat-old1.html"
        assert archive_path.exists()

        # Verify the live file was modified
        live_path = htmlgraph_dir / "features" / "feat-old1.html"
        live_content = live_path.read_text(encoding="utf-8")
        assert "AI-generated tier-1 summary" in live_content

        # Verify DB metadata updated
        conn = sqlite3.connect(str(db_path))
        row = conn.execute(
            "SELECT compacted_tier, compacted_summary FROM features WHERE id = ?",
            ("feat-old1",),
        ).fetchone()
        conn.close()
        assert row is not None
        assert row[0] == 1
        assert "AI-generated tier-1 summary" in row[1]


# ---------------------------------------------------------------------------
# Tier 2 compaction (90+ days)
# ---------------------------------------------------------------------------


class TestTier2Compaction:
    @patch("htmlgraph.archive.compactor._summarize_with_haiku")
    def test_tier2_compaction(
        self, mock_haiku: MagicMock, htmlgraph_dir: Path, db_path: Path
    ) -> None:
        """Item >90 days old gets tier-2 one-liner."""
        mock_haiku.return_value = "One-liner summary of very old feature."

        ancient_date = datetime.now(timezone.utc) - timedelta(days=120)
        original_content = "Detailed content " * 50
        _write_feature_html(
            htmlgraph_dir,
            "feat-ancient",
            "Ancient Feature",
            original_content,
            ancient_date,
        )
        _insert_feature_row(db_path, "feat-ancient")

        compactor = SemanticCompactor(db_path=db_path, htmlgraph_dir=htmlgraph_dir)
        result = compactor.compact(tier1_days=30, tier2_days=90)

        assert "feat-ancient" in result.tier2_compacted
        assert "feat-ancient" not in result.tier1_compacted
        assert result.total_compacted == 1

        # Verify archive exists
        assert (htmlgraph_dir / "archive" / "feat-ancient.html").exists()

        # Verify DB metadata
        conn = sqlite3.connect(str(db_path))
        row = conn.execute(
            "SELECT compacted_tier FROM features WHERE id = ?",
            ("feat-ancient",),
        ).fetchone()
        conn.close()
        assert row is not None
        assert row[0] == 2


# ---------------------------------------------------------------------------
# Dry-run mode
# ---------------------------------------------------------------------------


class TestDryRun:
    def test_dry_run_no_changes(self, htmlgraph_dir: Path, db_path: Path) -> None:
        """Dry-run lists items without modifying them."""
        old_date = datetime.now(timezone.utc) - timedelta(days=45)
        original_content = "Original content " * 20
        html_path = _write_feature_html(
            htmlgraph_dir, "feat-dry", "Dry Run Feature", original_content, old_date
        )
        _insert_feature_row(db_path, "feat-dry")
        original_html = html_path.read_text(encoding="utf-8")

        compactor = SemanticCompactor(
            db_path=db_path, htmlgraph_dir=htmlgraph_dir, dry_run=True
        )
        result = compactor.compact(tier1_days=30, tier2_days=90)

        assert "feat-dry" in result.tier1_compacted
        assert result.dry_run is True

        # File should NOT be modified
        assert html_path.read_text(encoding="utf-8") == original_html

        # Archive should NOT be created
        assert not (htmlgraph_dir / "archive" / "feat-dry.html").exists()

        # DB should NOT be updated
        conn = sqlite3.connect(str(db_path))
        row = conn.execute(
            "SELECT compacted_tier FROM features WHERE id = ?",
            ("feat-dry",),
        ).fetchone()
        conn.close()
        assert row is not None
        assert row[0] == 0


# ---------------------------------------------------------------------------
# Restore from archive
# ---------------------------------------------------------------------------


class TestRestore:
    @patch("htmlgraph.archive.compactor._summarize_with_haiku")
    def test_restore_after_compact(
        self, mock_haiku: MagicMock, htmlgraph_dir: Path, db_path: Path
    ) -> None:
        """Compact then restore — verify original is recovered."""
        mock_haiku.return_value = "Compacted summary."

        old_date = datetime.now(timezone.utc) - timedelta(days=45)
        original_content = "This is the original full content " * 10
        _write_feature_html(
            htmlgraph_dir, "feat-restore", "Restorable", original_content, old_date
        )
        _insert_feature_row(db_path, "feat-restore")

        compactor = SemanticCompactor(db_path=db_path, htmlgraph_dir=htmlgraph_dir)

        # Compact first
        compactor.compact(tier1_days=30, tier2_days=90)
        assert (htmlgraph_dir / "archive" / "feat-restore.html").exists()

        # Now restore
        restored = compactor.restore("feat-restore")
        assert restored is True

        # Archive should be removed after restore
        assert not (htmlgraph_dir / "archive" / "feat-restore.html").exists()

        # Live file should contain original content
        live_path = htmlgraph_dir / "features" / "feat-restore.html"
        live_content = live_path.read_text(encoding="utf-8")
        assert "This is the original full content" in live_content

        # DB metadata should be reset
        conn = sqlite3.connect(str(db_path))
        row = conn.execute(
            "SELECT compacted_tier, compacted_summary, compacted_at FROM features WHERE id = ?",
            ("feat-restore",),
        ).fetchone()
        conn.close()
        assert row is not None
        assert row[0] == 0  # tier reset
        assert row[1] is None  # summary cleared
        assert row[2] is None  # timestamp cleared

    def test_restore_missing_archive(self, htmlgraph_dir: Path, db_path: Path) -> None:
        """Restore returns False when no archive exists."""
        compactor = SemanticCompactor(db_path=db_path, htmlgraph_dir=htmlgraph_dir)
        assert compactor.restore("nonexistent-id") is False


# ---------------------------------------------------------------------------
# Age threshold — young items skipped
# ---------------------------------------------------------------------------


class TestAgeThreshold:
    def test_young_items_skipped(self, htmlgraph_dir: Path, db_path: Path) -> None:
        """Items younger than tier1_days are skipped."""
        recent_date = datetime.now(timezone.utc) - timedelta(days=5)
        _write_feature_html(
            htmlgraph_dir,
            "feat-young",
            "Young Feature",
            "Some content " * 20,
            recent_date,
        )
        _insert_feature_row(db_path, "feat-young")

        compactor = SemanticCompactor(db_path=db_path, htmlgraph_dir=htmlgraph_dir)
        result = compactor.compact(tier1_days=30, tier2_days=90)

        assert "feat-young" in result.skipped
        assert result.total_compacted == 0

    def test_items_between_tiers(self, htmlgraph_dir: Path, db_path: Path) -> None:
        """Item 45 days old gets tier 1, not tier 2."""
        mid_date = datetime.now(timezone.utc) - timedelta(days=45)
        _write_feature_html(
            htmlgraph_dir,
            "feat-mid",
            "Mid-Age Feature",
            "Some substantial content " * 20,
            mid_date,
        )
        _insert_feature_row(db_path, "feat-mid")

        compactor = SemanticCompactor(db_path=db_path, htmlgraph_dir=htmlgraph_dir)

        with patch("htmlgraph.archive.compactor._summarize_with_haiku") as mock_haiku:
            mock_haiku.return_value = "Tier 1 summary."
            result = compactor.compact(tier1_days=30, tier2_days=90)

        assert "feat-mid" in result.tier1_compacted
        assert "feat-mid" not in result.tier2_compacted


# ---------------------------------------------------------------------------
# Idempotent — running twice doesn't re-compact
# ---------------------------------------------------------------------------


class TestIdempotent:
    @patch("htmlgraph.archive.compactor._summarize_with_haiku")
    def test_compaction_idempotent(
        self, mock_haiku: MagicMock, htmlgraph_dir: Path, db_path: Path
    ) -> None:
        """Running compaction twice does not re-compact already-compacted items."""
        mock_haiku.return_value = "Summary from first run."

        old_date = datetime.now(timezone.utc) - timedelta(days=45)
        _write_feature_html(
            htmlgraph_dir,
            "feat-idem",
            "Idempotent Feature",
            "Content to compact " * 20,
            old_date,
        )
        _insert_feature_row(db_path, "feat-idem")

        compactor = SemanticCompactor(db_path=db_path, htmlgraph_dir=htmlgraph_dir)

        # First run
        result1 = compactor.compact(tier1_days=30, tier2_days=90)
        assert "feat-idem" in result1.tier1_compacted

        # Second run — same item should be skipped
        mock_haiku.reset_mock()
        result2 = compactor.compact(tier1_days=30, tier2_days=90)
        assert "feat-idem" in result2.skipped
        assert result2.total_compacted == 0
        # Haiku should NOT be called on the second run
        mock_haiku.assert_not_called()


# ---------------------------------------------------------------------------
# Haiku fallback — uses truncation when anthropic unavailable
# ---------------------------------------------------------------------------


class TestHaikuFallback:
    @patch("htmlgraph.archive.compactor._summarize_with_haiku", return_value=None)
    def test_fallback_to_truncation(
        self, mock_haiku: MagicMock, htmlgraph_dir: Path, db_path: Path
    ) -> None:
        """When Haiku returns None, truncation fallback is used."""
        old_date = datetime.now(timezone.utc) - timedelta(days=45)
        original_content = "Fallback content word " * 50
        _write_feature_html(
            htmlgraph_dir,
            "feat-fallback",
            "Fallback Feature",
            original_content,
            old_date,
        )
        _insert_feature_row(db_path, "feat-fallback")

        compactor = SemanticCompactor(db_path=db_path, htmlgraph_dir=htmlgraph_dir)
        result = compactor.compact(tier1_days=30, tier2_days=90)

        assert "feat-fallback" in result.tier1_compacted

        # Live file should have truncated content (ends with [...])
        live_path = htmlgraph_dir / "features" / "feat-fallback.html"
        live_content = live_path.read_text(encoding="utf-8")
        assert "[...]" in live_content


# ---------------------------------------------------------------------------
# Multiple entity types
# ---------------------------------------------------------------------------


class TestMultipleEntityTypes:
    @patch("htmlgraph.archive.compactor._summarize_with_haiku")
    def test_compacts_bugs_and_spikes(
        self, mock_haiku: MagicMock, htmlgraph_dir: Path, db_path: Path
    ) -> None:
        """Compaction scans features, bugs, and spikes directories."""
        mock_haiku.return_value = "Summarized."

        old_date = datetime.now(timezone.utc) - timedelta(days=45)

        _write_feature_html(
            htmlgraph_dir,
            "bug-old1",
            "Old Bug",
            "Bug description " * 20,
            old_date,
            entity_type="bug",
        )
        _insert_feature_row(db_path, "bug-old1")

        _write_feature_html(
            htmlgraph_dir,
            "spk-old1",
            "Old Spike",
            "Spike findings " * 20,
            old_date,
            entity_type="spike",
        )
        _insert_feature_row(db_path, "spk-old1")

        compactor = SemanticCompactor(db_path=db_path, htmlgraph_dir=htmlgraph_dir)
        result = compactor.compact(tier1_days=30, tier2_days=90)

        assert "bug-old1" in result.tier1_compacted
        assert "spk-old1" in result.tier1_compacted
        assert result.total_compacted == 2


# ---------------------------------------------------------------------------
# Short content skipped
# ---------------------------------------------------------------------------


class TestShortContentSkipped:
    def test_minimal_content_skipped(self, htmlgraph_dir: Path, db_path: Path) -> None:
        """Items with very short content (<50 chars) are skipped."""
        old_date = datetime.now(timezone.utc) - timedelta(days=45)
        _write_feature_html(
            htmlgraph_dir,
            "feat-tiny",
            "Tiny Feature",
            "Short",
            old_date,
        )
        _insert_feature_row(db_path, "feat-tiny")

        compactor = SemanticCompactor(db_path=db_path, htmlgraph_dir=htmlgraph_dir)
        result = compactor.compact(tier1_days=30, tier2_days=90)

        assert "feat-tiny" in result.skipped
        assert result.total_compacted == 0
