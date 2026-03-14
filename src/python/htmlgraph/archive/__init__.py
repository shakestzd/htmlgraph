"""
Archive management system for HtmlGraph.

Provides three-tier optimized search:
- Tier 1: Bloom filters (skip 70-90% of archives)
- Tier 2: SQLite FTS5 with BM25 ranking
- Tier 3: Snippet extraction and highlighting

Plus semantic compaction:
- Tier 1 compaction (30+ days): AI summary replaces full content
- Tier 2 compaction (90+ days): One-liner + metadata only

Target: 67x faster than naive multi-file search.
"""

from htmlgraph.archive.bloom import BloomFilter
from htmlgraph.archive.compactor import CompactionResult, SemanticCompactor
from htmlgraph.archive.fts import ArchiveFTS5Index
from htmlgraph.archive.manager import ArchiveConfig, ArchiveManager
from htmlgraph.archive.search import ArchiveSearchEngine, SearchResult

__all__ = [
    "ArchiveManager",
    "ArchiveConfig",
    "ArchiveSearchEngine",
    "SearchResult",
    "BloomFilter",
    "ArchiveFTS5Index",
    "CompactionResult",
    "SemanticCompactor",
]
