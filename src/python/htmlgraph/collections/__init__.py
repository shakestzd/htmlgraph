"""
Collection classes for managing nodes by type.

Provides specialized collections for each node type with
common functionality inherited from BaseCollection.
"""

from htmlgraph.collections.base import BaseCollection
from htmlgraph.collections.bug import BugCollection
from htmlgraph.collections.feature import FeatureCollection
from htmlgraph.collections.session import SessionCollection
from htmlgraph.collections.spike import SpikeCollection

__all__ = [
    "BaseCollection",
    "FeatureCollection",
    "SpikeCollection",
    "BugCollection",
    "SessionCollection",
]
