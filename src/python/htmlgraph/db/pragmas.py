"""SQLite PRAGMA settings for HtmlGraph databases."""

import logging
import sqlite3

logger = logging.getLogger(__name__)

# Standard PRAGMA settings applied to all HtmlGraph SQLite connections
PRAGMA_SETTINGS: dict[str, object] = {
    "journal_mode": "WAL",
    "synchronous": "NORMAL",
    "cache_size": -64000,  # 64MB cache
    "temp_store": "MEMORY",
    "mmap_size": 268435456,  # 256MB mmap
}


def apply_sync_pragmas(conn: sqlite3.Connection) -> None:
    """Apply standard PRAGMAs to a synchronous SQLite connection."""
    for pragma, value in PRAGMA_SETTINGS.items():
        conn.execute(f"PRAGMA {pragma} = {value}")


async def apply_async_pragmas(conn: object) -> None:
    """Apply standard PRAGMAs to an async aiosqlite connection."""
    for pragma, value in PRAGMA_SETTINGS.items():
        await conn.execute(f"PRAGMA {pragma} = {value}")  # type: ignore[attr-defined]


def check_integrity(conn: sqlite3.Connection) -> bool:
    """Run integrity_check and foreign_key_check. Returns True if all pass."""
    ic = conn.execute("PRAGMA integrity_check").fetchone()
    fkc = conn.execute("PRAGMA foreign_key_check").fetchall()
    ok = (ic[0] == "ok") and (len(fkc) == 0)
    if not ok:
        logger.critical(
            "SQLite integrity check failed: integrity=%s fk_violations=%d",
            ic[0],
            len(fkc),
        )
    return ok
