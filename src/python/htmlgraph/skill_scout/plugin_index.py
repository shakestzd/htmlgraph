"""
Local cache management for the Skill Scout plugin index.

Caches discovered plugins to ~/.htmlgraph/plugin-cache/index.json with a
24-hour TTL to avoid hammering the GitHub API on every query.
"""

from __future__ import annotations

import json
import time
import urllib.request
from pathlib import Path
from urllib.error import HTTPError, URLError

from htmlgraph.skill_scout.github_search import PluginInfo, discover_plugins

CACHE_DIR = Path.home() / ".htmlgraph" / "plugin-cache"
CACHE_TTL = 86400  # 24 hours in seconds

_MARKETPLACE_URL = (
    "https://raw.githubusercontent.com/anthropics/"
    "claude-plugins-official/main/marketplace.json"
)


class PluginIndex:
    """Local cache of discovered Claude Code plugins."""

    def __init__(self, cache_dir: Path = CACHE_DIR) -> None:
        self.cache_dir = cache_dir
        self.cache_dir.mkdir(parents=True, exist_ok=True)

    # ------------------------------------------------------------------
    # Cache state
    # ------------------------------------------------------------------

    @property
    def _cache_file(self) -> Path:
        return self.cache_dir / "index.json"

    def is_stale(self) -> bool:
        """Return True if cache is missing or older than TTL."""
        if not self._cache_file.exists():
            return True
        age = time.time() - self._cache_file.stat().st_mtime
        return age > CACHE_TTL

    # ------------------------------------------------------------------
    # Refresh
    # ------------------------------------------------------------------

    def refresh(self, force: bool = False) -> int:
        """Refresh plugin index from GitHub and official marketplace.

        Skips network calls when cache is fresh, unless *force* is True.
        Returns the total number of cached plugins.
        """
        if not force and not self.is_stale():
            return len(self.load())

        github_plugins = discover_plugins(limit=200)
        marketplace_plugins = self._fetch_marketplace()
        merged = self._merge(github_plugins, marketplace_plugins)
        self._save(merged)
        return len(merged)

    # ------------------------------------------------------------------
    # Search
    # ------------------------------------------------------------------

    def search(self, query: str) -> list[PluginInfo]:
        """Search cached index by keyword, ranked by relevance."""
        plugins = self.load()
        query_lower = query.lower()
        scored: list[tuple[int, PluginInfo]] = []

        for plugin in plugins:
            score = _score_plugin(plugin, query_lower)
            if score > 0:
                scored.append((score, plugin))

        scored.sort(key=lambda x: x[0], reverse=True)
        return [p for _, p in scored]

    # ------------------------------------------------------------------
    # Persistence
    # ------------------------------------------------------------------

    def load(self) -> list[PluginInfo]:
        """Load plugin list from local cache. Returns [] if cache missing."""
        if not self._cache_file.exists():
            return []
        data = json.loads(self._cache_file.read_text())
        if not isinstance(data, list):
            return []
        plugins = []
        for item in data:
            if isinstance(item, dict):
                try:
                    plugins.append(PluginInfo(**item))  # type: ignore[arg-type]
                except Exception:
                    pass
        return plugins

    def _save(self, plugins: list[PluginInfo]) -> None:
        data = [p.model_dump() for p in plugins]
        self._cache_file.write_text(json.dumps(data, indent=2))

    # ------------------------------------------------------------------
    # Marketplace fetch
    # ------------------------------------------------------------------

    def _fetch_marketplace(self) -> list[PluginInfo]:
        """Fetch plugins from the official Claude plugins marketplace.

        Returns an empty list on any network or parse error so a refresh
        failure here never blocks the GitHub results.
        """
        try:
            req = urllib.request.Request(
                _MARKETPLACE_URL,
                headers={"Accept": "application/json"},
            )
            with urllib.request.urlopen(req, timeout=30) as resp:
                data = json.loads(resp.read())
        except (HTTPError, URLError, json.JSONDecodeError):
            return []

        plugins: list[PluginInfo] = []
        for entry in data.get("plugins", []):
            source = entry.get("source", {})
            plugins.append(
                PluginInfo(
                    name=entry.get("name", ""),
                    repo=source.get("repo", ""),
                    description=entry.get("description", ""),
                    version=entry.get("version", ""),
                    category=entry.get("category", ""),
                    keywords=entry.get("keywords", []),
                    homepage=entry.get("homepage", ""),
                    source_url=source.get("url", ""),
                )
            )
        return plugins

    # ------------------------------------------------------------------
    # Merge / dedup
    # ------------------------------------------------------------------

    def _merge(self, *sources: list[PluginInfo]) -> list[PluginInfo]:
        """Merge multiple plugin lists, deduplicating by repo (then name)."""
        seen: dict[str, PluginInfo] = {}
        for source in sources:
            for plugin in source:
                key = plugin.repo or plugin.name
                if key not in seen:
                    seen[key] = plugin
        return list(seen.values())


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _score_plugin(plugin: PluginInfo, query: str) -> int:
    """Return a relevance score for *plugin* against *query* (0 = no match)."""
    score = 0
    if query in plugin.name.lower():
        score += 10
    if query in plugin.description.lower():
        score += 5
    if any(query in kw.lower() for kw in plugin.keywords):
        score += 7
    if query in plugin.category.lower():
        score += 3
    return score
