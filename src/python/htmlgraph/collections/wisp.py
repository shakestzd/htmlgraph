from __future__ import annotations

"""
Wisp collection — ephemeral, TTL-based coordination signals.

Wisps let agents publish short-lived signals that are visible to other
agents via CIGS guidance.  They are stored in a dedicated SQLite table
(not as HTML files) because they are ephemeral by design.

Example:
    >>> sdk = SDK(agent="sonnet-coder")
    >>> sdk.wisps.signal("starting DB migration, avoid writes", ttl=600, category="coordination")
    >>> active = sdk.wisps.active()
    >>> for w in active:
    ...     print(w.agent_id, w.message)
"""

import logging
import sqlite3
import uuid
from dataclasses import dataclass, field
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from htmlgraph.sdk import SDK

logger = logging.getLogger(__name__)


@dataclass
class Wisp:
    """An ephemeral coordination signal between agents."""

    id: str
    agent_id: str
    message: str
    category: str = "general"
    created_at: datetime = field(default_factory=lambda: datetime.now(timezone.utc))
    expires_at: datetime = field(
        default_factory=lambda: datetime.now(timezone.utc) + timedelta(seconds=3600)
    )

    def is_expired(self) -> bool:
        """Check whether this wisp has passed its expiry time."""
        now = datetime.now(timezone.utc)
        exp = self.expires_at
        if exp.tzinfo is None:
            exp = exp.replace(tzinfo=timezone.utc)
        return now >= exp

    def __repr__(self) -> str:
        return (
            f"Wisp(id={self.id!r}, agent={self.agent_id!r}, "
            f"category={self.category!r}, message={self.message[:40]!r})"
        )


class WispCollection:
    """
    Collection interface for ephemeral coordination signals (wisps).

    Wisps are short-lived messages stored in SQLite that agents publish to
    coordinate with each other.  They expire automatically via a TTL and are
    visible in CIGS guidance while active.

    Unlike other collections this class does *not* extend BaseCollection
    because wisps are purely DB-backed (no HTML files) and have a distinct
    API surface (signal / active / expire rather than create / where / edit).

    Example:
        >>> sdk = SDK(agent="claude")
        >>> sdk.wisps.signal("migrating DB — avoid writes", ttl=300)
        >>> for w in sdk.wisps.active():
        ...     print(w.agent_id, w.message)
    """

    def __init__(self, sdk: SDK) -> None:
        self._sdk = sdk

    # ------------------------------------------------------------------
    # Internal helpers
    # ------------------------------------------------------------------

    def _db_path(self) -> Path:
        return self._sdk._directory / "htmlgraph.db"

    def _connect(self) -> sqlite3.Connection:
        conn = sqlite3.connect(
            str(self._db_path()), timeout=5.0, check_same_thread=False
        )
        conn.row_factory = sqlite3.Row
        return conn

    def _ensure_table(self, conn: sqlite3.Connection) -> None:
        """Create wisps table if not yet present (idempotent)."""
        cursor = conn.cursor()
        cursor.execute("""
            CREATE TABLE IF NOT EXISTS wisps (
                id TEXT PRIMARY KEY,
                agent_id TEXT NOT NULL,
                message TEXT NOT NULL,
                category TEXT NOT NULL DEFAULT 'general',
                created_at TEXT NOT NULL,
                expires_at TEXT NOT NULL
            )
        """)
        cursor.execute(
            "CREATE INDEX IF NOT EXISTS idx_wisps_expires ON wisps(expires_at)"
        )
        cursor.execute(
            "CREATE INDEX IF NOT EXISTS idx_wisps_category ON wisps(category)"
        )
        conn.commit()

    def _row_to_wisp(self, row: sqlite3.Row) -> Wisp:
        def _parse_dt(val: str) -> datetime:
            try:
                dt = datetime.fromisoformat(val)
            except ValueError:
                dt = datetime.strptime(val, "%Y-%m-%d %H:%M:%S")
            if dt.tzinfo is None:
                dt = dt.replace(tzinfo=timezone.utc)
            return dt

        return Wisp(
            id=row["id"],
            agent_id=row["agent_id"],
            message=row["message"],
            category=row["category"] or "general",
            created_at=_parse_dt(row["created_at"]),
            expires_at=_parse_dt(row["expires_at"]),
        )

    @staticmethod
    def _to_db_ts(dt: datetime) -> str:
        """Normalise a datetime to the plain UTC string SQLite expects.

        SQLite's ``datetime('now')`` returns ``YYYY-MM-DD HH:MM:SS`` (no ``T``,
        no timezone suffix).  Storing timestamps with a ``+00:00`` suffix causes
        string comparisons like ``expires_at < datetime('now')`` to break because
        ``+`` (ASCII 43) sorts *after* a space (ASCII 32), so
        ``'2026-03-13 10:00:00+00:00' < '2026-03-13 10:00:01'`` evaluates to
        ``False``.  Normalising to ``YYYY-MM-DD HH:MM:SS.ffffff`` (space
        separator, no offset) makes the comparison work correctly.
        """
        utc = dt.astimezone(timezone.utc).replace(tzinfo=None)
        return utc.isoformat(sep=" ")

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------

    def _wisp_to_dict(self, wisp: Wisp) -> dict:
        """Convert a Wisp dataclass to a plain dict for consistent API returns."""
        return {
            "id": wisp.id,
            "agent_id": wisp.agent_id,
            "message": wisp.message,
            "category": wisp.category,
            "created_at": wisp.created_at.isoformat(),
            "expires_at": wisp.expires_at.isoformat(),
        }

    def signal(
        self,
        message: str,
        ttl: int = 3600,
        category: str = "general",
    ) -> dict:
        """
        Broadcast an ephemeral coordination signal.

        Args:
            message: The coordination message to broadcast.
            ttl: Time-to-live in seconds (default: 3600 = 1 hour).
            category: Signal category for filtering (default: 'general').

        Returns:
            Dict with wisp fields: id, agent_id, message, category,
            created_at, expires_at.

        Example:
            >>> sdk.wisps.signal("Entering critical section", ttl=300, category="locks")
        """
        now = datetime.now(timezone.utc)
        expires_at = now + timedelta(seconds=ttl)
        wisp = Wisp(
            id=f"wisp-{uuid.uuid4().hex[:8]}",
            agent_id=self._sdk._agent_id or "unknown",
            message=message,
            category=category,
            created_at=now,
            expires_at=expires_at,
        )
        try:
            conn = self._connect()
            try:
                self._ensure_table(conn)
                cursor = conn.cursor()
                cursor.execute(
                    """
                    INSERT INTO wisps (id, agent_id, message, category, created_at, expires_at)
                    VALUES (?, ?, ?, ?, ?, ?)
                    """,
                    (
                        wisp.id,
                        wisp.agent_id,
                        wisp.message,
                        wisp.category,
                        self._to_db_ts(wisp.created_at),
                        self._to_db_ts(wisp.expires_at),
                    ),
                )
                conn.commit()
                logger.debug(
                    f"Created wisp {wisp.id} (ttl={ttl}s, category={category!r})"
                )
            finally:
                conn.close()
        except sqlite3.Error as exc:
            logger.error(f"Failed to persist wisp: {exc}")
        return self._wisp_to_dict(wisp)

    def active(self, category: str | None = None) -> list[dict]:
        """
        Return all non-expired wisps, running lazy expiry first.

        Expired wisps are deleted from the database before the result
        set is computed so this method also acts as a cleanup sweep.

        Args:
            category: Optional category filter.  If None, all categories
                      are returned.

        Returns:
            List of dicts (id, agent_id, message, category, created_at,
            expires_at) for active (non-expired) wisps, oldest first.

        Example:
            >>> for w in sdk.wisps.active(category="coordination"):
            ...     print(w["message"])
        """
        try:
            conn = self._connect()
            try:
                self._ensure_table(conn)
                cursor = conn.cursor()

                # Lazy expiry: delete rows whose expires_at has passed
                cursor.execute(
                    "DELETE FROM wisps WHERE datetime(expires_at) < datetime('now')"
                )
                conn.commit()

                # Query remaining wisps
                if category is not None:
                    cursor.execute(
                        "SELECT * FROM wisps WHERE category = ? ORDER BY created_at ASC",
                        (category,),
                    )
                else:
                    cursor.execute("SELECT * FROM wisps ORDER BY created_at ASC")

                rows = cursor.fetchall()
                wisps = [self._row_to_wisp(r) for r in rows]
                # Secondary Python-side guard for clock skew
                active = [w for w in wisps if not w.is_expired()]
                return [self._wisp_to_dict(w) for w in active]
            finally:
                conn.close()
        except sqlite3.Error as exc:
            logger.error(f"Failed to query wisps: {exc}")
            return []

    def expire(self) -> int:
        """
        Delete all wisps whose TTL has elapsed.

        Called automatically by active() (lazy expiry) but can also be
        triggered manually for eager cleanup.

        Returns:
            Number of wisps deleted.

        Example:
            >>> deleted = sdk.wisps.expire()
        """
        try:
            conn = self._connect()
            try:
                self._ensure_table(conn)
                cursor = conn.cursor()
                cursor.execute(
                    "DELETE FROM wisps WHERE datetime(expires_at) < datetime('now')"
                )
                deleted = cursor.rowcount
                conn.commit()
                if deleted:
                    logger.debug(f"Expired {deleted} wisp(s)")
                return deleted
            finally:
                conn.close()
        except sqlite3.Error as exc:
            logger.error(f"Failed to expire wisps: {exc}")
            return 0

    def clear(self, category: str | None = None) -> int:
        """
        Delete all wisps, optionally filtered by category.

        Args:
            category: If provided, only wisps in this category are removed.
                      If None, all wisps are removed.

        Returns:
            Number of rows deleted.

        Example:
            >>> sdk.wisps.clear()          # clear all
            >>> sdk.wisps.clear("locks")   # clear one category
        """
        try:
            conn = self._connect()
            try:
                self._ensure_table(conn)
                cursor = conn.cursor()
                if category is not None:
                    cursor.execute("DELETE FROM wisps WHERE category = ?", (category,))
                else:
                    cursor.execute("DELETE FROM wisps")
                deleted = cursor.rowcount
                conn.commit()
                logger.debug(f"Cleared {deleted} wisp(s) (category={category!r})")
                return deleted
            finally:
                conn.close()
        except sqlite3.Error as exc:
            logger.error(f"Failed to clear wisps: {exc}")
            return 0
