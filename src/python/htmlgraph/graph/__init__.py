"""
Graph operations for HtmlGraph.

Provides:
- File-based graph management
- CSS selector queries
- Graph algorithms (BFS, shortest path, dependency analysis)
- Bottleneck detection
- Transaction/snapshot support for concurrency

The graph package is organized into:
- core: Main HtmlGraph class with CRUD operations
- queries: CSS selectors, filtering, and find API
- algorithms: Graph traversal and analysis algorithms
"""

# Import core classes
from .core import GraphSnapshot, HtmlGraph

# Import query-related classes
from .queries import CompiledQuery

# Re-export everything for backward compatibility
__all__ = [
    "HtmlGraph",
    "GraphSnapshot",
    "CompiledQuery",
]
