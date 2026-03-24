"""Skill Scout: Plugin/Agent Discovery & Recommendation Engine."""

from htmlgraph.skill_scout.github_search import PluginInfo, discover_plugins
from htmlgraph.skill_scout.plugin_index import PluginIndex
from htmlgraph.skill_scout.project_analyzer import analyze_project
from htmlgraph.skill_scout.work_patterns import analyze_work_patterns

__all__ = [
    "PluginInfo",
    "discover_plugins",
    "PluginIndex",
    "analyze_project",
    "analyze_work_patterns",
]
