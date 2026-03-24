"""
GitHub search strategies for discovering Claude Code plugins.

Two strategies:
  A — gh CLI (preferred, respects user's auth)
  B — GitHub REST API fallback
"""

from __future__ import annotations

import json
import os
import subprocess
import time
import urllib.request
from typing import Any
from urllib.error import HTTPError, URLError

from pydantic import BaseModel

# ---------------------------------------------------------------------------
# Data model
# ---------------------------------------------------------------------------


class PluginInfo(BaseModel):
    """Metadata for a discovered Claude Code plugin."""

    name: str
    repo: str  # owner/repo
    description: str = ""
    version: str = ""
    category: str = ""
    keywords: list[str] = []
    homepage: str = ""
    source_url: str = ""


# ---------------------------------------------------------------------------
# Strategy A — gh CLI
# ---------------------------------------------------------------------------


def search_plugins_gh_cli(limit: int = 100) -> list[dict[str, Any]]:
    """Search GitHub for Claude Code plugins using gh CLI.

    Raises FileNotFoundError if gh is not installed.
    Raises RuntimeError on non-zero exit.
    """
    result = subprocess.run(
        [
            "gh",
            "search",
            "code",
            "filename:plugin.json",
            "--filename",
            "plugin.json",
            "--limit",
            str(limit),
            "--json",
            "path,repository,url,sha",
        ],
        capture_output=True,
        text=True,
        timeout=30,
    )
    if result.returncode != 0:
        raise RuntimeError(f"gh search failed: {result.stderr.strip()}")

    items: list[dict[str, Any]] = json.loads(result.stdout)
    return [i for i in items if ".claude-plugin" in i.get("path", "")]


# ---------------------------------------------------------------------------
# Strategy B — REST API fallback
# ---------------------------------------------------------------------------


def search_plugins_api(
    token: str | None = None,
    limit: int = 100,
) -> list[dict[str, Any]]:
    """Search GitHub REST API for Claude Code plugins.

    Falls back gracefully when no token is provided (lower rate limit).
    """
    headers = {
        "Accept": "application/vnd.github+json",
        "X-GitHub-Api-Version": "2022-11-28",
    }
    if token:
        headers["Authorization"] = f"Bearer {token}"

    per_page = min(limit, 30)  # search API max is 100, 30 is safe
    pages_needed = (limit + per_page - 1) // per_page
    results: list[dict[str, Any]] = []

    for page in range(1, pages_needed + 1):
        if len(results) >= limit:
            break
        url = (
            "https://api.github.com/search/code"
            f"?q=filename:plugin.json+path:.claude-plugin"
            f"&per_page={per_page}&page={page}"
        )
        req = urllib.request.Request(url, headers=headers)
        try:
            with urllib.request.urlopen(req, timeout=30) as resp:
                data = json.loads(resp.read())
                results.extend(data.get("items", []))
        except (HTTPError, URLError) as exc:
            raise RuntimeError(f"GitHub API search failed: {exc}") from exc

        # Respect rate limit: 10 requests/min for search API
        if page < pages_needed:
            time.sleep(6)

    return results[:limit]


# ---------------------------------------------------------------------------
# Normalisation
# ---------------------------------------------------------------------------


def _normalize_gh_cli_result(raw: dict[str, Any]) -> PluginInfo:
    """Convert a gh CLI search result into a PluginInfo."""
    repo_info = raw.get("repository", {})
    repo_name = repo_info.get("nameWithOwner", "")
    return PluginInfo(
        name=repo_info.get("name", repo_name.split("/")[-1] if repo_name else ""),
        repo=repo_name,
        description=repo_info.get("description") or "",
        homepage=repo_info.get("url", ""),
        source_url=raw.get("url", ""),
    )


def _normalize_api_result(raw: dict[str, Any]) -> PluginInfo:
    """Convert a REST API search item into a PluginInfo."""
    repo_info = raw.get("repository", {})
    full_name = repo_info.get("full_name", "")
    return PluginInfo(
        name=repo_info.get("name", full_name.split("/")[-1] if full_name else ""),
        repo=full_name,
        description=repo_info.get("description") or "",
        homepage=repo_info.get("html_url", ""),
        source_url=raw.get("html_url", ""),
    )


# ---------------------------------------------------------------------------
# Public entry point
# ---------------------------------------------------------------------------


def discover_plugins(limit: int = 100) -> list[PluginInfo]:
    """Discover Claude Code plugins.

    Tries gh CLI first; falls back to GitHub REST API if gh is not available.
    """
    try:
        raw = search_plugins_gh_cli(limit)
        return [_normalize_gh_cli_result(r) for r in raw]
    except (FileNotFoundError, RuntimeError):
        token = os.environ.get("GITHUB_TOKEN")
        raw_api = search_plugins_api(token, limit)
        return [_normalize_api_result(r) for r in raw_api]
