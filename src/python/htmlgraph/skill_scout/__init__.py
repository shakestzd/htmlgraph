"""
Skill Scout: project auditor, plugin discovery, and skill recommendation.

Analyses the current project's tech stack and work patterns, then recommends
Claude Code skills/plugins that would benefit the project.
"""

# Phase 1: Project auditing and work pattern analysis
# Phase 2: Plugin discovery and indexing
from htmlgraph.skill_scout.github_search import PluginInfo, discover_plugins
from htmlgraph.skill_scout.plugin_index import PluginIndex
from htmlgraph.skill_scout.project_analyzer import ProjectAnalysis, ProjectAnalyzer
from htmlgraph.skill_scout.work_patterns import ToolUsage, WorkPatternAnalyzer

__all__ = [
    # Phase 1 exports
    "ProjectAnalyzer",
    "ProjectAnalysis",
    "WorkPatternAnalyzer",
    "ToolUsage",
    # Phase 2 exports
    "PluginInfo",
    "discover_plugins",
    "PluginIndex",
]
