"""
Work Pattern Analyzer — mine SQLite event history for tool usage and capability gaps.

Queries the HtmlGraph SQLite database to surface:
- Which tools are used most/least frequently
- Capability gaps (tools never used despite likely need)
- Delegation rate (Task tool usage vs direct tool usage)
"""

from __future__ import annotations

import logging
import sqlite3
from dataclasses import dataclass, field
from pathlib import Path

logger = logging.getLogger(__name__)

# Tools that signal good delegation hygiene when present
_DELEGATION_TOOLS = {"Task", "Agent"}

# Capability categories and their representative tools
_CAPABILITY_CATEGORIES: dict[str, list[str]] = {
    "file_operations": ["Read", "Write", "Edit", "MultiEdit"],
    "search": ["Bash", "Grep", "Glob"],
    "web": ["WebFetch", "WebSearch"],
    "delegation": ["Task", "Agent"],
    "version_control": ["Bash"],  # git via Bash
    "testing": ["Bash"],  # pytest/test runners via Bash
}


@dataclass
class ToolUsage:
    """Frequency data for a single tool."""

    tool_name: str
    call_count: int
    error_count: int = 0

    @property
    def error_rate(self) -> float:
        if self.call_count == 0:
            return 0.0
        return self.error_count / self.call_count


@dataclass
class WorkPatternSummary:
    """Aggregated work pattern analysis for a project."""

    tool_usage: list[ToolUsage] = field(default_factory=list)
    delegation_rate: float = 0.0
    total_tool_calls: int = 0
    capability_gaps: list[str] = field(default_factory=list)

    def most_used_tools(self, n: int = 5) -> list[ToolUsage]:
        """Return the top-n most frequently used tools."""
        return sorted(self.tool_usage, key=lambda t: t.call_count, reverse=True)[:n]

    def least_used_tools(self, n: int = 5) -> list[ToolUsage]:
        """Return the n least-used tools (excluding zero-count tools)."""
        nonzero = [t for t in self.tool_usage if t.call_count > 0]
        return sorted(nonzero, key=lambda t: t.call_count)[:n]


class WorkPatternAnalyzer:
    """Mine HtmlGraph SQLite for tool usage frequency and capability gaps."""

    def __init__(self, db_path: str | Path) -> None:
        self.db_path = Path(db_path)

    def analyze(self) -> WorkPatternSummary:
        """Return a WorkPatternSummary for the project's event history."""
        summary = WorkPatternSummary()

        if not self.db_path.exists():
            logger.debug("WorkPatternAnalyzer: db not found at %s", self.db_path)
            return summary

        try:
            tool_rows = self._query_tool_counts()
            error_rows = self._query_error_counts()
        except sqlite3.Error as exc:
            logger.warning("WorkPatternAnalyzer: sqlite error: %s", exc)
            return summary

        error_map: dict[str, int] = {row[0]: row[1] for row in error_rows}
        usage_list: list[ToolUsage] = []
        total = 0

        for tool_name, count in tool_rows:
            usage = ToolUsage(
                tool_name=tool_name,
                call_count=count,
                error_count=error_map.get(tool_name, 0),
            )
            usage_list.append(usage)
            total += count

        summary.tool_usage = usage_list
        summary.total_tool_calls = total
        summary.delegation_rate = self._compute_delegation_rate(usage_list, total)
        summary.capability_gaps = self._find_capability_gaps(usage_list)

        return summary

    # ------------------------------------------------------------------
    # Private helpers
    # ------------------------------------------------------------------

    def _query_tool_counts(self) -> list[tuple[str, int]]:
        sql = """
            SELECT tool_name, COUNT(*) AS cnt
            FROM events
            WHERE tool_name IS NOT NULL
            GROUP BY tool_name
            ORDER BY cnt DESC
        """
        with sqlite3.connect(str(self.db_path)) as conn:
            return conn.execute(sql).fetchall()

    def _query_error_counts(self) -> list[tuple[str, int]]:
        sql = """
            SELECT tool_name, COUNT(*) AS cnt
            FROM events
            WHERE tool_name IS NOT NULL
              AND (is_error = 1 OR error IS NOT NULL)
            GROUP BY tool_name
        """
        with sqlite3.connect(str(self.db_path)) as conn:
            try:
                return conn.execute(sql).fetchall()
            except sqlite3.OperationalError:
                # Older schema may not have is_error / error columns
                return []

    @staticmethod
    def _compute_delegation_rate(usage: list[ToolUsage], total: int) -> float:
        if total == 0:
            return 0.0
        delegation_calls = sum(
            u.call_count for u in usage if u.tool_name in _DELEGATION_TOOLS
        )
        return delegation_calls / total

    @staticmethod
    def _find_capability_gaps(usage: list[ToolUsage]) -> list[str]:
        used_tools: set[str] = {u.tool_name for u in usage if u.call_count > 0}
        gaps: list[str] = []

        # If web tools never appear but Bash is heavily used, flag web tools
        web_tools = set(_CAPABILITY_CATEGORIES["web"])
        if not used_tools & web_tools:
            gaps.append("web_browsing")

        # If delegation tools never appear
        if not used_tools & _DELEGATION_TOOLS:
            gaps.append("delegation")

        return gaps
