"""
Collection classes for managing nodes by type.

Provides specialized collections for each node type with
common functionality inherited from BaseCollection.
"""

from htmlgraph.collections.base import BaseCollection
from htmlgraph.collections.bug import BugCollection
from htmlgraph.collections.chore import ChoreCollection
from htmlgraph.collections.epic import EpicCollection
from htmlgraph.collections.feature import FeatureCollection
from htmlgraph.collections.insight import InsightCollection
from htmlgraph.collections.metric import MetricCollection
from htmlgraph.collections.pattern import PatternCollection
from htmlgraph.collections.phase import PhaseCollection
from htmlgraph.collections.spike import SpikeCollection
from htmlgraph.collections.task_delegation import TaskDelegationCollection
from htmlgraph.collections.todo import TodoCollection
from htmlgraph.collections.wisp import Wisp, WispCollection

__all__ = [
    "BaseCollection",
    "FeatureCollection",
    "SpikeCollection",
    "BugCollection",
    "ChoreCollection",
    "EpicCollection",
    "PhaseCollection",
    "PatternCollection",
    "InsightCollection",
    "MetricCollection",
    "TodoCollection",
    "TaskDelegationCollection",
    "WispCollection",
    "Wisp",
]
