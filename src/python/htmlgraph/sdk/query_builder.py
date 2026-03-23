"""Semantic query builder for expressive work item queries."""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any


@dataclass
class QueryFilter:
    """A single filter condition."""

    field: str
    operator: str  # eq, ne, contains, gt, lt, in, not_in
    value: Any


@dataclass
class SemanticQuery:
    """Compiled semantic query."""

    filters: list[QueryFilter] = field(default_factory=list)
    sort_by: str | None = None
    sort_order: str = "desc"
    limit: int | None = None
    offset: int = 0
    include_related: bool = False


class SemanticQueryBuilder:
    """Fluent query builder for work items.

    Usage:
        results = (sdk.query()
            .features()
            .where_status("in-progress")
            .where_priority("high")
            .with_tag("backend")
            .created_after("2026-03-01")
            .sort_by("updated", "desc")
            .limit(10)
            .execute())
    """

    def __init__(self, sdk: Any) -> None:
        self._sdk = sdk
        self._collection: str | None = None
        self._filters: list[QueryFilter] = []
        self._sort_by: str | None = None
        self._sort_order: str = "desc"
        self._limit: int | None = None
        self._offset: int = 0
        self._include_related: bool = False

    # Collection selectors

    def features(self) -> SemanticQueryBuilder:
        """Select the features collection."""
        self._collection = "features"
        return self

    def bugs(self) -> SemanticQueryBuilder:
        """Select the bugs collection."""
        self._collection = "bugs"
        return self

    def spikes(self) -> SemanticQueryBuilder:
        """Select the spikes collection."""
        self._collection = "spikes"
        return self

    def all_items(self) -> SemanticQueryBuilder:
        """Select all work item collections (currently uses features)."""
        self._collection = "all"
        return self

    # Status filters

    def where_status(self, status: str) -> SemanticQueryBuilder:
        """Filter by exact status value."""
        self._filters.append(QueryFilter("status", "eq", status))
        return self

    def active(self) -> SemanticQueryBuilder:
        """Shorthand for where_status('in-progress')."""
        return self.where_status("in-progress")

    def todo(self) -> SemanticQueryBuilder:
        """Shorthand for where_status('todo')."""
        return self.where_status("todo")

    def done(self) -> SemanticQueryBuilder:
        """Shorthand for where_status('done')."""
        return self.where_status("done")

    # Priority filters

    def where_priority(self, priority: str) -> SemanticQueryBuilder:
        """Filter by exact priority value."""
        self._filters.append(QueryFilter("priority", "eq", priority))
        return self

    def high_priority(self) -> SemanticQueryBuilder:
        """Shorthand for where_priority('high')."""
        return self.where_priority("high")

    # Generic field filters

    def where(self, field: str, operator: str, value: Any) -> SemanticQueryBuilder:
        """Add a generic filter condition.

        Args:
            field: Attribute name on the node
            operator: One of: eq, ne, contains, gt, lt, in, not_in
            value: Value to compare against
        """
        self._filters.append(QueryFilter(field, operator, value))
        return self

    def where_track(self, track_id: str) -> SemanticQueryBuilder:
        """Filter by track_id."""
        self._filters.append(QueryFilter("track_id", "eq", track_id))
        return self

    def with_tag(self, tag: str) -> SemanticQueryBuilder:
        """Filter to items that contain the given tag."""
        self._filters.append(QueryFilter("tags", "contains", tag))
        return self

    def created_after(self, date: str) -> SemanticQueryBuilder:
        """Filter to items created after the given ISO date string."""
        self._filters.append(QueryFilter("created", "gt", date))
        return self

    def created_before(self, date: str) -> SemanticQueryBuilder:
        """Filter to items created before the given ISO date string."""
        self._filters.append(QueryFilter("created", "lt", date))
        return self

    # Sorting and pagination

    def sort_by(self, field: str, order: str = "desc") -> SemanticQueryBuilder:
        """Sort results by the given field.

        Args:
            field: Attribute name to sort by
            order: 'asc' or 'desc' (default: 'desc')
        """
        self._sort_by = field
        self._sort_order = order
        return self

    def limit(self, n: int) -> SemanticQueryBuilder:
        """Limit the number of results returned."""
        self._limit = n
        return self

    def offset(self, n: int) -> SemanticQueryBuilder:
        """Skip the first n results."""
        self._offset = n
        return self

    # Options

    def with_related(self) -> SemanticQueryBuilder:
        """Include related items in results (future use)."""
        self._include_related = True
        return self

    # Build and execute

    def build(self) -> SemanticQuery:
        """Compile the builder into a SemanticQuery object."""
        return SemanticQuery(
            filters=list(self._filters),
            sort_by=self._sort_by,
            sort_order=self._sort_order,
            limit=self._limit,
            offset=self._offset,
            include_related=self._include_related,
        )

    def execute(self) -> list[Any]:
        """Execute the query and return matching work items."""
        query = self.build()
        collection = self._get_collection()
        if collection is None:
            return []

        # Separate eq filters (handled by .where()) from the rest
        eq_kwargs: dict[str, Any] = {}
        non_eq_filters: list[QueryFilter] = []
        for f in query.filters:
            if f.operator == "eq":
                eq_kwargs[f.field] = f.value
            else:
                non_eq_filters.append(f)

        results: list[Any] = (
            list(collection.where(**eq_kwargs)) if eq_kwargs else list(collection.all())
        )

        # Apply remaining filters in memory
        for f in non_eq_filters:
            if f.operator == "contains":
                results = [
                    r for r in results if f.value in (getattr(r, f.field, None) or [])
                ]
            elif f.operator == "gt":
                results = [
                    r for r in results if (getattr(r, f.field, None) or "") > f.value
                ]
            elif f.operator == "lt":
                results = [
                    r for r in results if (getattr(r, f.field, None) or "") < f.value
                ]
            elif f.operator == "ne":
                results = [r for r in results if getattr(r, f.field, None) != f.value]
            elif f.operator == "in":
                results = [r for r in results if getattr(r, f.field, None) in f.value]
            elif f.operator == "not_in":
                results = [
                    r for r in results if getattr(r, f.field, None) not in f.value
                ]

        # Sort
        if query.sort_by:
            sort_field = query.sort_by
            results.sort(
                key=lambda r: getattr(r, sort_field, None) or "",
                reverse=(query.sort_order == "desc"),
            )

        # Pagination
        if query.offset:
            results = results[query.offset :]
        if query.limit is not None:
            results = results[: query.limit]

        return results

    def count(self) -> int:
        """Return the count of matching items."""
        return len(self.execute())

    def first(self) -> Any | None:
        """Return the first matching item, or None."""
        results = self.limit(1).execute()
        return results[0] if results else None

    def _get_collection(self) -> Any | None:
        """Resolve the target collection from the SDK."""
        if self._collection == "features":
            return self._sdk.features
        elif self._collection == "bugs":
            return self._sdk.bugs
        elif self._collection == "spikes":
            return self._sdk.spikes
        elif self._collection == "all":
            # TODO: union across all collections
            return self._sdk.features
        return None
