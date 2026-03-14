"""
Tests for Phase 1: Step-Level Attribution.

Every tool call event should know which step of which feature it belongs to.
Tests cover:
- Step model step_id field
- HTML serialization with data-step-id
- Parser extraction of data-step-id
- Feature creation generates step_ids
- Builder auto-assigns step_ids on save
- DB insert_event with step_id column
- resolve_active_step returns first incomplete step
- Backward compatibility for features without step_ids
"""

import sqlite3
import tempfile
from pathlib import Path
from unittest.mock import MagicMock, patch

from htmlgraph.models import Node, Step

# ---------------------------------------------------------------------------
# Step model tests
# ---------------------------------------------------------------------------


class TestStepModel:
    """Test Step model step_id field."""

    def test_step_model_has_step_id(self) -> None:
        """Step(step_id='step-feat-123-0') stores the step_id."""
        step = Step(description="Design approach", step_id="step-feat-123-0")
        assert step.step_id == "step-feat-123-0"

    def test_step_model_step_id_default_none(self) -> None:
        """Step without step_id defaults to None."""
        step = Step(description="Design approach")
        assert step.step_id is None

    def test_step_to_html_includes_step_id(self) -> None:
        """to_html() emits data-step-id attribute when step_id is set."""
        step = Step(
            description="Design approach",
            step_id="step-feat-abc-0",
        )
        html = step.to_html()
        assert 'data-step-id="step-feat-abc-0"' in html
        assert "Design approach" in html

    def test_step_to_html_without_step_id(self) -> None:
        """to_html() works without step_id (backward compat)."""
        step = Step(description="Design approach")
        html = step.to_html()
        assert "data-step-id" not in html
        assert "Design approach" in html
        assert 'data-completed="false"' in html

    def test_step_to_html_completed_with_step_id(self) -> None:
        """to_html() includes both completed and step_id attributes."""
        step = Step(
            description="Implement core",
            completed=True,
            agent="claude",
            step_id="step-feat-xyz-1",
        )
        html = step.to_html()
        assert 'data-completed="true"' in html
        assert 'data-agent="claude"' in html
        assert 'data-step-id="step-feat-xyz-1"' in html

    def test_step_to_context_includes_step_id(self) -> None:
        """to_context() includes step_id prefix when set."""
        step = Step(description="Add tests", step_id="step-feat-abc-2")
        ctx = step.to_context()
        assert ctx == "[step-feat-abc-2] [ ] Add tests"

    def test_step_to_context_without_step_id(self) -> None:
        """to_context() works without step_id prefix."""
        step = Step(description="Add tests")
        ctx = step.to_context()
        assert ctx == "[ ] Add tests"

    def test_step_to_context_completed(self) -> None:
        """to_context() shows completed status with step_id."""
        step = Step(description="Done", completed=True, step_id="step-f-0")
        ctx = step.to_context()
        assert ctx == "[step-f-0] [x] Done"

    def test_step_dict_access_step_id(self) -> None:
        """Backward-compatible dict-style access works for step_id."""
        step = Step(description="Test", step_id="step-f-0")
        assert step["step_id"] == "step-f-0"


# ---------------------------------------------------------------------------
# Parser tests
# ---------------------------------------------------------------------------


class TestParserStepId:
    """Test parser extraction of data-step-id from HTML."""

    def test_parser_extracts_step_id(self) -> None:
        """parser.get_steps() returns step_id from HTML."""
        from htmlgraph.parser import HtmlParser

        html = """
        <article id="feat-test">
            <section data-steps>
                <ol>
                    <li data-completed="false" data-step-id="step-feat-test-0">Design</li>
                    <li data-completed="true" data-step-id="step-feat-test-1">Implement</li>
                    <li data-completed="false" data-step-id="step-feat-test-2">Test</li>
                </ol>
            </section>
        </article>
        """
        parser = HtmlParser(html_content=html)
        steps = parser.get_steps()

        assert len(steps) == 3
        assert steps[0]["step_id"] == "step-feat-test-0"
        assert steps[0]["completed"] is False
        assert steps[1]["step_id"] == "step-feat-test-1"
        assert steps[1]["completed"] is True
        assert steps[2]["step_id"] == "step-feat-test-2"
        assert steps[2]["completed"] is False

    def test_parser_handles_missing_step_id(self) -> None:
        """parser.get_steps() works without data-step-id (backward compat)."""
        from htmlgraph.parser import HtmlParser

        html = """
        <article id="feat-old">
            <section data-steps>
                <ol>
                    <li data-completed="false">Old step without ID</li>
                    <li data-completed="true">Another old step</li>
                </ol>
            </section>
        </article>
        """
        parser = HtmlParser(html_content=html)
        steps = parser.get_steps()

        assert len(steps) == 2
        assert "step_id" not in steps[0]
        assert "step_id" not in steps[1]
        assert steps[0]["description"] == "Old step without ID"
        assert steps[1]["completed"] is True

    def test_parser_mixed_step_ids(self) -> None:
        """parser.get_steps() handles mix of steps with and without step_ids."""
        from htmlgraph.parser import HtmlParser

        html = """
        <article id="feat-mix">
            <section data-steps>
                <ol>
                    <li data-completed="false" data-step-id="step-feat-mix-0">Has ID</li>
                    <li data-completed="false">No ID</li>
                </ol>
            </section>
        </article>
        """
        parser = HtmlParser(html_content=html)
        steps = parser.get_steps()

        assert len(steps) == 2
        assert steps[0]["step_id"] == "step-feat-mix-0"
        assert "step_id" not in steps[1]


# ---------------------------------------------------------------------------
# Converter tests
# ---------------------------------------------------------------------------


class TestConverterStepId:
    """Test converter passes step_id from parsed data to Step models."""

    def test_converter_preserves_step_id(self) -> None:
        """html_to_node preserves step_id from HTML through to Step models."""
        from htmlgraph.converter import html_to_node

        with tempfile.NamedTemporaryFile(mode="w", suffix=".html", delete=False) as f:
            f.write("""<!DOCTYPE html>
<html>
<body>
<article id="feat-conv-test" data-type="feature" data-status="todo">
    <header><h1>Converter Test</h1></header>
    <section data-steps>
        <ol>
            <li data-completed="false" data-step-id="step-feat-conv-test-0">Step A</li>
            <li data-completed="true" data-step-id="step-feat-conv-test-1">Step B</li>
        </ol>
    </section>
</article>
</body>
</html>""")
            f.flush()
            node = html_to_node(f.name)

        assert len(node.steps) == 2
        assert node.steps[0].step_id == "step-feat-conv-test-0"
        assert node.steps[0].completed is False
        assert node.steps[1].step_id == "step-feat-conv-test-1"
        assert node.steps[1].completed is True

        # Clean up
        Path(f.name).unlink()

    def test_converter_handles_no_step_id(self) -> None:
        """html_to_node works for HTML without data-step-id (backward compat)."""
        from htmlgraph.converter import html_to_node

        with tempfile.NamedTemporaryFile(mode="w", suffix=".html", delete=False) as f:
            f.write("""<!DOCTYPE html>
<html>
<body>
<article id="feat-old-test" data-type="feature" data-status="todo">
    <header><h1>Old Feature</h1></header>
    <section data-steps>
        <ol>
            <li data-completed="false">Old step</li>
        </ol>
    </section>
</article>
</body>
</html>""")
            f.flush()
            node = html_to_node(f.name)

        assert len(node.steps) == 1
        assert node.steps[0].step_id is None
        assert node.steps[0].description == "Old step"

        Path(f.name).unlink()


# ---------------------------------------------------------------------------
# Feature creation tests
# ---------------------------------------------------------------------------


class TestFeatureCreationStepIds:
    """Test that feature creation generates step_ids."""

    def test_feature_creation_generates_step_ids(self) -> None:
        """New features created via FeatureWorkflow get step-{id}-{index} IDs."""
        from htmlgraph.sessions.features import FeatureWorkflow

        # Create a mock SessionManager
        mock_manager = MagicMock()
        mock_graph = MagicMock()
        mock_manager._get_graph.return_value = mock_graph

        # Capture what gets added to the graph
        added_nodes: list[Node] = []
        mock_graph.add.side_effect = lambda n: added_nodes.append(n)

        workflow = FeatureWorkflow(mock_manager)
        node = workflow.create_feature(
            title="Test Feature",
            steps=["Design", "Implement", "Test"],
        )

        assert len(node.steps) == 3
        assert node.steps[0].step_id == f"step-{node.id}-0"
        assert node.steps[1].step_id == f"step-{node.id}-1"
        assert node.steps[2].step_id == f"step-{node.id}-2"

    def test_feature_creation_no_steps(self) -> None:
        """Features with empty steps list get no step_ids."""
        from htmlgraph.sessions.features import FeatureWorkflow

        mock_manager = MagicMock()
        mock_graph = MagicMock()
        mock_manager._get_graph.return_value = mock_graph
        mock_graph.add.side_effect = lambda n: None

        workflow = FeatureWorkflow(mock_manager)
        node = workflow.create_feature(
            title="No Steps Feature",
            collection="bugs",
            steps=[],
        )

        assert len(node.steps) == 0


# ---------------------------------------------------------------------------
# Builder tests
# ---------------------------------------------------------------------------


class TestBuilderStepIds:
    """Test builder auto-assigns step_ids on save."""

    def test_builder_generates_step_ids_on_save(self) -> None:
        """Builder auto-assigns step_ids to steps that don't have them."""
        from htmlgraph.builders.base import BaseBuilder

        mock_sdk = MagicMock()
        mock_sdk.tracks.all.return_value = []

        builder = BaseBuilder(sdk=mock_sdk, title="Builder Test")
        builder.node_type = "bug"  # Avoid track_id requirement for features
        builder.add_step("Step A")
        builder.add_step("Step B")
        builder.add_step("Step C")

        # Override _data id for predictability
        builder._data["id"] = "bug-test-123"

        node = builder.save()

        assert len(node.steps) == 3
        assert node.steps[0].step_id == "step-bug-test-123-0"
        assert node.steps[1].step_id == "step-bug-test-123-1"
        assert node.steps[2].step_id == "step-bug-test-123-2"

    def test_builder_preserves_explicit_step_id(self) -> None:
        """Builder preserves explicitly provided step_ids."""
        from htmlgraph.builders.base import BaseBuilder

        mock_sdk = MagicMock()
        mock_sdk.tracks.all.return_value = []

        builder = BaseBuilder(sdk=mock_sdk, title="Explicit IDs")
        builder.node_type = "bug"
        builder.add_step("Step A", step_id="custom-step-0")
        builder.add_step("Step B")  # No explicit ID

        builder._data["id"] = "bug-explicit-456"

        node = builder.save()

        assert node.steps[0].step_id == "custom-step-0"
        assert node.steps[1].step_id == "step-bug-explicit-456-1"


# ---------------------------------------------------------------------------
# Database tests
# ---------------------------------------------------------------------------


class TestDatabaseStepId:
    """Test step_id column in agent_events table."""

    def _create_test_db(self) -> tuple[sqlite3.Connection, str]:
        """Create a temporary test database with schema."""
        tmp = tempfile.mkdtemp()
        db_path = str(Path(tmp) / "test.db")

        from htmlgraph.db.schema import HtmlGraphDB

        db = HtmlGraphDB(db_path)
        assert db.connection is not None
        return db.connection, db_path

    def test_step_id_column_exists(self) -> None:
        """step_id column exists in agent_events table after migration."""
        conn, db_path = self._create_test_db()
        cursor = conn.cursor()
        cursor.execute("PRAGMA table_info(agent_events)")
        columns = {row[1] for row in cursor.fetchall()}
        assert "step_id" in columns
        conn.close()

    def test_step_id_index_exists(self) -> None:
        """Index on step_id column exists."""
        conn, db_path = self._create_test_db()
        cursor = conn.cursor()
        cursor.execute(
            "SELECT name FROM sqlite_master WHERE type='index' AND name='idx_agent_events_step_id'"
        )
        row = cursor.fetchone()
        assert row is not None
        conn.close()

    def test_insert_event_with_step_id(self) -> None:
        """DB insert_event includes step_id column."""
        from htmlgraph.db.schema import HtmlGraphDB

        tmp = tempfile.mkdtemp()
        db_path = str(Path(tmp) / "test.db")
        db = HtmlGraphDB(db_path)

        # Insert a session first (foreign key)
        db.insert_session(
            session_id="test-session",
            agent_assigned="claude",
        )

        # Insert event with step_id
        success = db.insert_event(
            event_id="evt-001",
            agent_id="claude",
            event_type="tool_call",
            session_id="test-session",
            tool_name="Read",
            step_id="step-feat-test-0",
        )
        assert success is True

        # Verify step_id was stored
        assert db.connection is not None
        cursor = db.connection.cursor()
        cursor.execute("SELECT step_id FROM agent_events WHERE event_id = 'evt-001'")
        row = cursor.fetchone()
        assert row is not None
        assert row[0] == "step-feat-test-0"

        db.close()

    def test_insert_event_without_step_id(self) -> None:
        """DB insert_event works without step_id (backward compat)."""
        from htmlgraph.db.schema import HtmlGraphDB

        tmp = tempfile.mkdtemp()
        db_path = str(Path(tmp) / "test.db")
        db = HtmlGraphDB(db_path)

        db.insert_session(
            session_id="test-session",
            agent_assigned="claude",
        )

        success = db.insert_event(
            event_id="evt-002",
            agent_id="claude",
            event_type="tool_call",
            session_id="test-session",
            tool_name="Read",
        )
        assert success is True

        assert db.connection is not None
        cursor = db.connection.cursor()
        cursor.execute("SELECT step_id FROM agent_events WHERE event_id = 'evt-002'")
        row = cursor.fetchone()
        assert row is not None
        assert row[0] is None

        db.close()

    def test_migration_adds_step_id_to_existing_db(self) -> None:
        """Migration adds step_id column to an existing agent_events table."""
        tmp = tempfile.mkdtemp()
        db_path = str(Path(tmp) / "migrate.db")

        # Create a bare DB without step_id column
        conn = sqlite3.connect(db_path)
        conn.execute("""
            CREATE TABLE agent_events (
                event_id TEXT PRIMARY KEY,
                agent_id TEXT NOT NULL,
                event_type TEXT NOT NULL,
                timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
                session_id TEXT NOT NULL
            )
        """)
        conn.execute("""
            CREATE TABLE sessions (
                session_id TEXT PRIMARY KEY,
                agent_assigned TEXT NOT NULL,
                status TEXT DEFAULT 'active',
                created_at DATETIME DEFAULT CURRENT_TIMESTAMP
            )
        """)
        conn.commit()

        # Verify step_id doesn't exist yet
        cursor = conn.cursor()
        cursor.execute("PRAGMA table_info(agent_events)")
        columns = {row[1] for row in cursor.fetchall()}
        assert "step_id" not in columns
        conn.close()

        # Now open with HtmlGraphDB (runs migrations)
        from htmlgraph.db.schema import HtmlGraphDB

        db = HtmlGraphDB(db_path)

        # Verify step_id column was added
        assert db.connection is not None
        cursor = db.connection.cursor()
        cursor.execute("PRAGMA table_info(agent_events)")
        columns = {row[1] for row in cursor.fetchall()}
        assert "step_id" in columns

        db.close()


# ---------------------------------------------------------------------------
# resolve_active_step tests
# ---------------------------------------------------------------------------


class TestResolveActiveStep:
    """Test resolve_active_step returns first incomplete step's ID."""

    def _create_feature_html(
        self, tmpdir: Path, feature_id: str, steps: list[dict]
    ) -> Path:
        """Create a feature HTML file with given steps."""
        features_dir = tmpdir / ".htmlgraph" / "features"
        features_dir.mkdir(parents=True, exist_ok=True)

        step_items = []
        for s in steps:
            attrs = f'data-completed="{str(s["completed"]).lower()}"'
            if s.get("step_id"):
                attrs += f' data-step-id="{s["step_id"]}"'
            step_items.append(f"<li {attrs}>{s['description']}</li>")

        html = f"""<!DOCTYPE html>
<html><body>
<article id="{feature_id}" data-type="feature" data-status="in-progress">
    <header><h1>Test Feature</h1></header>
    <section data-steps>
        <ol>
            {"".join(step_items)}
        </ol>
    </section>
</article>
</body></html>"""

        filepath = features_dir / f"{feature_id}.html"
        filepath.write_text(html, encoding="utf-8")
        return filepath

    def test_resolve_active_step(self) -> None:
        """Returns first incomplete step's ID."""
        from htmlgraph.hooks.event_tracker import resolve_active_step

        with tempfile.TemporaryDirectory() as tmpdir:
            tmpdir_path = Path(tmpdir)
            self._create_feature_html(
                tmpdir_path,
                "feat-test-001",
                [
                    {
                        "description": "Done",
                        "completed": True,
                        "step_id": "step-feat-test-001-0",
                    },
                    {
                        "description": "Active",
                        "completed": False,
                        "step_id": "step-feat-test-001-1",
                    },
                    {
                        "description": "Pending",
                        "completed": False,
                        "step_id": "step-feat-test-001-2",
                    },
                ],
            )

            with patch("os.getcwd", return_value=str(tmpdir_path)):
                # Also patch Path.cwd since resolve_active_step uses Path.cwd()
                with patch.object(Path, "cwd", return_value=tmpdir_path):
                    result = resolve_active_step("feat-test-001")

            assert result == "step-feat-test-001-1"

    def test_resolve_active_step_all_complete(self) -> None:
        """Returns None when all steps are done."""
        from htmlgraph.hooks.event_tracker import resolve_active_step

        with tempfile.TemporaryDirectory() as tmpdir:
            tmpdir_path = Path(tmpdir)
            self._create_feature_html(
                tmpdir_path,
                "feat-done-001",
                [
                    {
                        "description": "Done1",
                        "completed": True,
                        "step_id": "step-feat-done-001-0",
                    },
                    {
                        "description": "Done2",
                        "completed": True,
                        "step_id": "step-feat-done-001-1",
                    },
                ],
            )

            with patch.object(Path, "cwd", return_value=tmpdir_path):
                result = resolve_active_step("feat-done-001")

            assert result is None

    def test_resolve_active_step_no_feature(self) -> None:
        """Returns None when feature_id is None."""
        from htmlgraph.hooks.event_tracker import resolve_active_step

        result = resolve_active_step(None)
        assert result is None

    def test_resolve_active_step_missing_file(self) -> None:
        """Returns None when feature HTML file doesn't exist."""
        from htmlgraph.hooks.event_tracker import resolve_active_step

        with tempfile.TemporaryDirectory() as tmpdir:
            tmpdir_path = Path(tmpdir)
            # Create the .htmlgraph/features directory but no feature file
            (tmpdir_path / ".htmlgraph" / "features").mkdir(parents=True)

            with patch.object(Path, "cwd", return_value=tmpdir_path):
                result = resolve_active_step("feat-nonexistent")

            assert result is None

    def test_resolve_active_step_no_step_ids(self) -> None:
        """Returns None for old features without step_ids."""
        from htmlgraph.hooks.event_tracker import resolve_active_step

        with tempfile.TemporaryDirectory() as tmpdir:
            tmpdir_path = Path(tmpdir)
            self._create_feature_html(
                tmpdir_path,
                "feat-old-001",
                [
                    {"description": "Old step", "completed": False},
                ],
            )

            with patch.object(Path, "cwd", return_value=tmpdir_path):
                result = resolve_active_step("feat-old-001")

            assert result is None

    def test_resolve_active_step_first_step_incomplete(self) -> None:
        """Returns first step_id when first step is incomplete."""
        from htmlgraph.hooks.event_tracker import resolve_active_step

        with tempfile.TemporaryDirectory() as tmpdir:
            tmpdir_path = Path(tmpdir)
            self._create_feature_html(
                tmpdir_path,
                "feat-first-001",
                [
                    {
                        "description": "First",
                        "completed": False,
                        "step_id": "step-feat-first-001-0",
                    },
                    {
                        "description": "Second",
                        "completed": False,
                        "step_id": "step-feat-first-001-1",
                    },
                ],
            )

            with patch.object(Path, "cwd", return_value=tmpdir_path):
                result = resolve_active_step("feat-first-001")

            assert result == "step-feat-first-001-0"


# ---------------------------------------------------------------------------
# Backward compatibility tests
# ---------------------------------------------------------------------------


class TestBackwardCompat:
    """Test backward compatibility for features without step_ids."""

    def test_backward_compat_no_step_id(self) -> None:
        """Features without step_ids still work through the full pipeline."""
        # Create a node with steps that have no step_id
        node = Node(
            id="feat-old-compat",
            title="Old Feature",
            type="feature",
            steps=[
                Step(description="Step 1"),
                Step(description="Step 2", completed=True),
            ],
        )

        # to_html should work without step_ids
        html = node.to_html()
        assert "Step 1" in html
        assert "Step 2" in html
        assert "data-step-id" not in html

        # to_context should work without step_ids
        for step in node.steps:
            ctx = step.to_context()
            assert step.description in ctx

    def test_node_next_step_with_step_id(self) -> None:
        """Node.next_step returns step with step_id."""
        node = Node(
            id="feat-next",
            title="Next Step Test",
            steps=[
                Step(description="Done", completed=True, step_id="step-feat-next-0"),
                Step(description="Active", completed=False, step_id="step-feat-next-1"),
                Step(
                    description="Pending", completed=False, step_id="step-feat-next-2"
                ),
            ],
        )

        next_step = node.next_step
        assert next_step is not None
        assert next_step.step_id == "step-feat-next-1"
        assert next_step.description == "Active"

    def test_roundtrip_html_with_step_ids(self) -> None:
        """Step IDs survive HTML serialization and deserialization roundtrip."""
        from htmlgraph.converter import html_to_node

        node = Node(
            id="feat-roundtrip",
            title="Roundtrip Test",
            type="feature",
            steps=[
                Step(description="Step A", step_id="step-feat-roundtrip-0"),
                Step(
                    description="Step B",
                    completed=True,
                    step_id="step-feat-roundtrip-1",
                ),
                Step(description="Step C", step_id="step-feat-roundtrip-2"),
            ],
        )

        # Serialize to HTML
        html = node.to_html()

        # Write to temp file and parse back
        with tempfile.NamedTemporaryFile(mode="w", suffix=".html", delete=False) as f:
            f.write(html)
            f.flush()
            parsed_node = html_to_node(f.name)

        assert len(parsed_node.steps) == 3
        assert parsed_node.steps[0].step_id == "step-feat-roundtrip-0"
        assert parsed_node.steps[0].completed is False
        assert parsed_node.steps[1].step_id == "step-feat-roundtrip-1"
        assert parsed_node.steps[1].completed is True
        assert parsed_node.steps[2].step_id == "step-feat-roundtrip-2"

        Path(f.name).unlink()


# ---------------------------------------------------------------------------
# Integration: record_event_to_sqlite with step_id
# ---------------------------------------------------------------------------


class TestRecordEventStepId:
    """Test record_event_to_sqlite passes step_id through."""

    def test_record_event_with_step_id(self) -> None:
        """record_event_to_sqlite stores step_id in the database."""
        from htmlgraph.db.schema import HtmlGraphDB
        from htmlgraph.hooks.event_tracker import record_event_to_sqlite

        tmp = tempfile.mkdtemp()
        db_path = str(Path(tmp) / "test.db")
        db = HtmlGraphDB(db_path)

        db.insert_session(
            session_id="test-session",
            agent_assigned="claude",
        )

        event_id = record_event_to_sqlite(
            db=db,
            session_id="test-session",
            tool_name="Edit",
            tool_input={"file": "test.py"},
            tool_response={"content": "OK"},
            is_error=False,
            feature_id="feat-test-rec",
            step_id="step-feat-test-rec-1",
        )

        assert event_id is not None

        # Verify step_id was stored
        assert db.connection is not None
        cursor = db.connection.cursor()
        cursor.execute(
            "SELECT step_id FROM agent_events WHERE event_id = ?",
            (event_id,),
        )
        row = cursor.fetchone()
        assert row is not None
        assert row[0] == "step-feat-test-rec-1"

        db.close()

    def test_record_event_without_step_id(self) -> None:
        """record_event_to_sqlite works without step_id."""
        from htmlgraph.db.schema import HtmlGraphDB
        from htmlgraph.hooks.event_tracker import record_event_to_sqlite

        tmp = tempfile.mkdtemp()
        db_path = str(Path(tmp) / "test.db")
        db = HtmlGraphDB(db_path)

        db.insert_session(
            session_id="test-session",
            agent_assigned="claude",
        )

        event_id = record_event_to_sqlite(
            db=db,
            session_id="test-session",
            tool_name="Read",
            tool_input={"file": "test.py"},
            tool_response={"content": "data"},
            is_error=False,
        )

        assert event_id is not None

        assert db.connection is not None
        cursor = db.connection.cursor()
        cursor.execute(
            "SELECT step_id FROM agent_events WHERE event_id = ?",
            (event_id,),
        )
        row = cursor.fetchone()
        assert row is not None
        assert row[0] is None

        db.close()
