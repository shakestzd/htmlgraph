# Browser-Native Query Interface - Phase 1 Implementation Plan

**Track**: trk-991c5af1
**Status**: Planning
**Target Scope**: Quick Wins - Snapshot & Refs
**Deadline**: Next 3-4 sprints

---

## Overview

Phase 1 introduces the foundational ref system and snapshot capabilities needed for browser-native queries. Instead of long UUIDs like `feat-a1b2c3d4`, users get short stable refs like `@f1`, `@t1`, `@b5`. This enables:
- **AI-friendly snapshots** - agents can reason about short refs
- **Browser queries** - agents can ask "show @f1 @f2 @t1"
- **Command simplicity** - `htmlgraph snapshot` shows all work with refs
- **Foundation for Phase 2** - semantic queries build on top of this

---

## Architecture Design

### 1. RefManager Class

**File**: `src/python/htmlgraph/refs.py`

**Purpose**: Maintain persistent mapping of short refs (@f1, @t2) to full node IDs.

```python
class RefManager:
    """Manages short references (@f1, @t1, @b5, etc.) for graph nodes."""

    def __init__(self, graph_dir: Path):
        self.graph_dir = graph_dir
        self.refs_file = graph_dir / "refs.json"
        self._refs = {}  # Maps: "@f1" -> "feat-a1b2c3d4"
        self._reverse_refs = {}  # Maps: "feat-a1b2c3d4" -> "@f1"
        self._load()

    # Core Methods

    def generate_ref(self, node_id: str) -> str:
        """Generate a short ref for a node ID (auto-saved).

        Args:
            node_id: Full node ID like "feat-a1b2c3d4"

        Returns:
            Short ref like "@f1"

        Raises:
            ValueError: If node_id already has a ref (use get_ref)
        """
        # Parse prefix from node_id
        # Check if already has ref (idempotent)
        # Generate next available: @f1, @f2, @f3, etc.
        # Save to refs.json
        # Return ref

    def get_ref(self, node_id: str) -> str | None:
        """Get existing ref for a node ID (create if not exist).

        Returns None only if node_id invalid.
        """

    def resolve_ref(self, short_ref: str) -> str | None:
        """Resolve short ref to full node ID.

        Args:
            short_ref: "@f1", "@t2", etc.

        Returns:
            Full node ID or None if not found
        """

    def get_all_refs(self) -> dict[str, str]:
        """Return all refs. Maps: "@f1" -> "feat-a1b2c3d4"."""

    def get_refs_by_type(self, node_type: str) -> list[tuple[str, str]]:
        """Get all refs for a specific type.

        Args:
            node_type: "feature", "track", "bug", "spike", "chore", "epic"

        Returns:
            List of (short_ref, full_id) tuples sorted by ref number
        """

    def rebuild_refs(self):
        """Rebuild refs from all .htmlgraph/ files (recovery tool).

        - Scans all features/, tracks/, bugs/, spikes/, etc.
        - Rebuilds refs.json from scratch
        - Idempotent (preserves existing refs where possible)
        """

    # Internal Methods

    def _load(self):
        """Load refs.json into memory."""

    def _save(self):
        """Save refs to refs.json."""

    def _next_ref_number(self, node_type: str) -> int:
        """Get next available ref number for a type."""
```

**ref.json Format**:
```json
{
  "refs": {
    "@f1": "feat-a1b2c3d4",
    "@f2": "feat-b2c3d4e5",
    "@f3": "feat-c3d4e5f6",
    "@t1": "trk-123abc45",
    "@b1": "bug-456def78",
    "@s1": "spk-789ghi01",
    "@c1": "chr-abc1234d",
    "@e1": "epc-def5678e"
  },
  "version": 1,
  "regenerated_at": "2026-01-13T12:00:00Z"
}
```

**Ref Format Rules**:
- Format: `@{prefix}{number}` (e.g., @f1, @t5, @b10)
- Prefixes: f=feature, t=track, b=bug, s=spike, c=chore, e=epic, d=todo
- Numbers: Sequential per type (1, 2, 3, ...)
- Stable: Once created, never changes for that node

---

### 2. SDK Integration

**File**: `src/python/htmlgraph/sdk.py` (add to existing SDK class)

```python
class SDK:
    def __init__(self, agent="claude", directory=None):
        # ... existing __init__ code ...

        # Add ref manager
        self.refs = RefManager(self._directory)
        self.features.set_ref_manager(self.refs)  # Each collection knows about refs
        self.tracks.set_ref_manager(self.refs)
        self.bugs.set_ref_manager(self.refs)
        self.spikes.set_ref_manager(self.refs)
        # ... etc for all collections

    def ref(self, short_ref: str) -> Node | None:
        """Resolve a short ref to a Node object.

        Args:
            short_ref: "@f1", "@t2", "@b5", etc.

        Returns:
            Node object or None if not found

        Example:
            feature = sdk.ref("@f1")
            if feature:
                print(feature.title)
        """
        full_id = self.refs.resolve_ref(short_ref)
        if not full_id:
            return None

        # Determine type from ref prefix and fetch from appropriate collection
        prefix = short_ref[1]  # Get letter after @
        if prefix == 'f':
            return self.features.get(full_id)
        elif prefix == 't':
            return self.tracks.get(full_id)
        # ... etc for other types

        return None

    def snapshot(self) -> dict:
        """Return current graph state (used by snapshot command).

        Returns structured dict with all work items organized by type and status.
        See SnapshotCommand for output format.
        """
```

**Collection Integration** (`src/python/htmlgraph/collections/base.py`):
```python
class BaseCollection:
    def __init__(self, sdk):
        # ... existing code ...
        self._ref_manager = None

    def set_ref_manager(self, ref_manager):
        """Called by SDK during init."""
        self._ref_manager = ref_manager

    def get_ref(self, node_id: str) -> str | None:
        """Convenience method to get ref for a node in this collection."""
        if self._ref_manager:
            return self._ref_manager.get_ref(node_id)
        return None
```

---

### 3. Snapshot Command

**File**: `src/python/htmlgraph/cli/work/snapshot.py`

**Purpose**: Output current graph state in a structured, AI-readable format.

```python
class SnapshotCommand(BaseCommand):
    """
    Generate and output a snapshot of the current graph state.

    Usage:
        htmlgraph snapshot                    # Human-readable
        htmlgraph snapshot --format json      # JSON
        htmlgraph snapshot --format refs      # With short refs (default)
        htmlgraph snapshot --type feature     # Only features
        htmlgraph snapshot --status todo      # Only todo items
    """

    def __init__(self, *,
                 format: str = "refs",        # refs, json, text
                 node_type: str | None = None, # feature, track, bug, spike, chore, epic, all
                 status: str | None = None):   # todo, in_progress, blocked, done, all
        self.format = format
        self.node_type = node_type
        self.status = status

    @classmethod
    def from_args(cls, args: argparse.Namespace) -> 'SnapshotCommand':
        return cls(
            format=args.format,
            node_type=args.type,
            status=args.status
        )

    def execute(self) -> CommandResult:
        sdk = self.get_sdk()

        # Gather all work items
        items = self._gather_items(sdk)

        # Format output
        if self.format == "json":
            output = self._format_json(items)
        elif self.format == "refs":
            output = self._format_refs(items, include_refs=True)
        else:  # text
            output = self._format_text(items)

        return CommandResult(
            success=True,
            data={"snapshot": output},
            message=f"Snapshot: {len(items)} items"
        )

    def _gather_items(self, sdk) -> list[dict]:
        """Gather all relevant items from SDK."""
        items = []

        # Collect from each collection
        for collection_name in ["features", "tracks", "bugs", "spikes", "chores", "epics"]:
            if self.node_type and self.node_type != "all" and self.node_type != collection_name:
                continue

            collection = getattr(sdk, collection_name)
            nodes = collection.all()

            for node in nodes:
                if self.status and self.status != "all" and node.status != self.status:
                    continue

                items.append(self._node_to_dict(sdk, node))

        return sorted(items, key=lambda x: (x["type"], x["status"], x["ref"] or ""))

    def _node_to_dict(self, sdk, node) -> dict:
        """Convert Node to dict with ref."""
        ref = sdk.refs.get_ref(node.id) if hasattr(sdk, 'refs') else None

        return {
            "ref": ref,
            "id": node.id,
            "type": node.type,
            "title": node.title,
            "status": node.status,
            "priority": getattr(node, "priority", None),
            "assigned_to": getattr(node, "agent_assigned", None),
            "track_id": getattr(node, "track_id", None),
        }

    def _format_refs(self, items: list[dict], include_refs: bool) -> str:
        """Format as readable list with refs."""
        output = []
        output.append("SNAPSHOT - Current Graph State")
        output.append("=" * 50)

        by_type = {}
        for item in items:
            t = item["type"]
            if t not in by_type:
                by_type[t] = []
            by_type[t].append(item)

        for node_type in ["feature", "track", "bug", "spike", "chore", "epic"]:
            if node_type not in by_type:
                continue

            output.append(f"\n{node_type.upper()}S ({len(by_type[node_type])})")
            output.append("-" * 40)

            # Group by status
            by_status = {}
            for item in by_type[node_type]:
                status = item["status"]
                if status not in by_status:
                    by_status[status] = []
                by_status[status].append(item)

            for status in ["todo", "in_progress", "blocked", "done"]:
                if status not in by_status:
                    continue

                output.append(f"\n  {status.upper()}:")
                for item in by_status[status]:
                    ref_str = f"{item['ref']:4s}" if item['ref'] else "    "
                    prio = item['priority'] or "-"
                    output.append(f"    {ref_str} | {item['title'][:40]:40s} | {prio}")

        return "\n".join(output)

    def _format_json(self, items: list[dict]) -> str:
        """Format as JSON."""
        import json
        return json.dumps(items, indent=2, default=str)

    def _format_text(self, items: list[dict]) -> str:
        """Format as simple text (no refs)."""
        output = []
        for item in items:
            output.append(f"{item['type']:8s} | {item['title']:40s} | {item['status']}")
        return "\n".join(output)
```

**Register in** `src/python/htmlgraph/cli/work/__init__.py`:
```python
def register_commands(subparsers):
    # ... existing registrations ...

    # Snapshot command
    snapshot_parser = subparsers.add_parser(
        "snapshot",
        help="Snapshot current graph state with refs"
    )
    snapshot_parser.add_argument(
        "--format",
        choices=["refs", "json", "text"],
        default="refs",
        help="Output format (default: refs)"
    )
    snapshot_parser.add_argument(
        "--type",
        choices=["feature", "track", "bug", "spike", "chore", "epic", "all"],
        default="all",
        help="Filter by type"
    )
    snapshot_parser.add_argument(
        "--status",
        choices=["todo", "in_progress", "blocked", "done", "all"],
        default="all",
        help="Filter by status"
    )
    snapshot_parser.set_defaults(func=SnapshotCommand.from_args)
```

---

### 4. Browse Command

**File**: `src/python/htmlgraph/cli/work/browse.py`

**Purpose**: Open the dashboard in default browser (foundation for later browser automation).

```python
class BrowseCommand(BaseCommand):
    """
    Open the HtmlGraph dashboard in your default browser.

    Usage:
        htmlgraph browse                      # Open dashboard
        htmlgraph browse --port 8080          # Custom port
        htmlgraph browse --query-type feature # Show only features
    """

    def __init__(self, *,
                 port: int = 8080,
                 query_type: str | None = None,
                 query_status: str | None = None):
        self.port = port
        self.query_type = query_type
        self.query_status = query_status

    @classmethod
    def from_args(cls, args: argparse.Namespace) -> 'BrowseCommand':
        return cls(
            port=args.port,
            query_type=args.query_type,
            query_status=args.query_status
        )

    def execute(self) -> CommandResult:
        import webbrowser

        sdk = self.get_sdk()

        # Build URL with query params
        url = f"http://localhost:{self.port}"

        params = []
        if self.query_type:
            params.append(f"type={self.query_type}")
        if self.query_status:
            params.append(f"status={self.query_status}")

        if params:
            url += "?" + "&".join(params)

        # Check if server is running, suggest starting if not
        try:
            import requests
            requests.head("http://localhost:8080", timeout=1)
        except:
            return CommandResult(
                success=False,
                message=f"Dashboard server not running. Start with: htmlgraph serve"
            )

        # Open browser
        webbrowser.open(url)

        return CommandResult(
            success=True,
            data={"url": url},
            message=f"Opening dashboard at {url}"
        )
```

**Register in** `src/python/htmlgraph/cli/work/__init__.py`:
```python
# Browse command
browse_parser = subparsers.add_parser(
    "browse",
    help="Open dashboard in browser"
)
browse_parser.add_argument(
    "--port",
    type=int,
    default=8080,
    help="Server port (default: 8080)"
)
browse_parser.add_argument(
    "--query-type",
    help="Filter by type (feature, track, bug, spike, chore, epic)"
)
browse_parser.add_argument(
    "--query-status",
    help="Filter by status (todo, in_progress, blocked, done)"
)
browse_parser.set_defaults(func=BrowseCommand.from_args)
```

---

## Implementation Order

### Step 1: RefManager Class (2.0 hours)
- [ ] Create `src/python/htmlgraph/refs.py`
- [ ] Implement RefManager class with all methods
- [ ] Create tests in `tests/python/test_refs.py`
- [ ] Integration test: generate refs, persist, reload

### Step 2: SDK Integration (1.0 hour)
- [ ] Add ref manager to SDK.__init__()
- [ ] Add sdk.ref() method
- [ ] Add set_ref_manager() to BaseCollection
- [ ] Test: `sdk.ref("@f1")` returns correct Feature

### Step 3: Snapshot Command (2.0 hours)
- [ ] Create `src/python/htmlgraph/cli/work/snapshot.py`
- [ ] Implement SnapshotCommand class
- [ ] Register in work/__init__.py
- [ ] Test: `htmlgraph snapshot` shows all items with refs
- [ ] Test: `htmlgraph snapshot --format json` outputs valid JSON
- [ ] Test: `htmlgraph snapshot --type feature` filters correctly

### Step 4: Browse Command (1.0 hour)
- [ ] Create `src/python/htmlgraph/cli/work/browse.py`
- [ ] Implement BrowseCommand class
- [ ] Register in work/__init__.py
- [ ] Test: Opens correct URL in browser

### Step 5: Integration Tests (2.0 hours)
- [ ] Create `tests/python/test_snapshot_and_refs.py`
- [ ] Test ref generation and resolution
- [ ] Test snapshot output formats
- [ ] Test SDL.ref() integration
- [ ] Test command integration

---

## Testing Strategy

### Unit Tests
```python
# tests/python/test_refs.py
def test_ref_generation()
def test_ref_resolution()
def test_ref_persistence()
def test_ref_collision_detection()
def test_rebuild_refs()

# tests/python/test_snapshot.py
def test_snapshot_command_refs_format()
def test_snapshot_command_json_format()
def test_snapshot_command_type_filter()
def test_snapshot_command_status_filter()
```

### Integration Tests
```python
# tests/python/test_snapshot_and_refs.py
def test_end_to_end_snapshot_with_refs()
def test_sdk_ref_method()
def test_browse_command_opens_dashboard()
```

### Acceptance Criteria (from Track)
1. ✅ htmlgraph snapshot outputs parseable refs and graph state
2. ✅ Short refs resolve correctly to full entity IDs
3. ✅ sdk.ref('@f1') returns Feature object
4. ✅ All linting, type checking, and tests pass

---

## Phase 1 Completion Checklist

- [ ] RefManager class complete and tested
- [ ] SDK.ref() method working
- [ ] `htmlgraph snapshot` command functional
- [ ] `htmlgraph browse` command functional
- [ ] All unit tests passing
- [ ] All integration tests passing
- [ ] All type checks passing (mypy)
- [ ] All lint checks passing (ruff)
- [ ] Documentation updated in track
- [ ] Commit to branch with message "feat: Phase 1 - snapshot and ref system"

---

## Notes

### Design Decisions

1. **RefManager as separate class**: Decoupled from SDK for testability and reusability
2. **Persistent refs.json**: Short refs are stable across sessions
3. **Auto-generation on access**: Calling sdk.ref() generates ref if needed
4. **Rebuild capability**: Can recover refs from file system if refs.json corrupted
5. **Type-specific numbering**: @f1-@f99 for features, @t1-@t99 for tracks, etc.

### Future Considerations for Phase 2

- Semantic query DSL will use these refs: `find("features").where(ref=["@f1", "@f2"]).execute()`
- Browser integration will parse snapshots and navigate by refs
- HTTP API will support ref-based queries: `/api/features/@f1`

