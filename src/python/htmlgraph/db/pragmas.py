"""SQLite PRAGMA settings for HtmlGraph databases."""

# Standard PRAGMA settings applied to all HtmlGraph SQLite connections
PRAGMA_SETTINGS: dict[str, object] = {
    "journal_mode": "WAL",
    "synchronous": "NORMAL",
    "cache_size": -64000,  # 64MB cache
    "temp_store": "MEMORY",
    "mmap_size": 268435456,  # 256MB mmap
}
