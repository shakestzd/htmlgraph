"""
Skill Scout: project auditor, plugin discovery, and skill recommendation.

Analyses the current project's tech stack and work patterns, then recommends
Claude Code skills/plugins that would benefit the project.
"""

from htmlgraph.skill_scout.project_analyzer import ProjectAnalysis, ProjectAnalyzer
from htmlgraph.skill_scout.work_patterns import ToolUsage, WorkPatternAnalyzer

__all__ = [
    "ProjectAnalyzer",
    "ProjectAnalysis",
    "WorkPatternAnalyzer",
    "ToolUsage",
]
