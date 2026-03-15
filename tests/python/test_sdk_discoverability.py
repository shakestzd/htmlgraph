"""
Tests for SDK discoverability improvements.

Verifies that help(), __dir__, and documentation are working correctly
to make the SDK easier for AI agents to discover and use.
"""

from pathlib import Path

import pytest
from htmlgraph import SDK


@pytest.fixture
def sdk(isolated_graph_dir_full: Path, isolated_db: Path):
    """Create a temporary SDK instance for testing."""
    # isolated_graph_dir_full already has all required subdirectories
    return SDK(
        directory=isolated_graph_dir_full, agent="test-agent", db_path=str(isolated_db)
    )


class TestSDKHelp:
    """Test sdk.help() method for discoverability."""

    def test_help_returns_all_topics(self, sdk: SDK, isolated_db):
        """help() without args returns topic index with available topics."""
        result = sdk.help()
        # Check for key sections in help output
        assert "COLLECTIONS" in result
        assert "METHODS" in result or "CORE METHODS" in result
        # Check that major collection types are listed
        assert "features" in result.lower()
        assert "bugs" in result.lower()
        assert "sessions" in result.lower()

    def test_help_features_topic(self, sdk: SDK, isolated_db):
        """help('features') returns feature-specific documentation."""
        result = sdk.help("features")
        # Should contain feature-specific info
        assert "feature" in result.lower()
        assert "create" in result.lower()
        # Should mention builder pattern or methods
        assert "builder" in result.lower() or "get" in result.lower()

    def test_help_orchestrate_topic(self, sdk: SDK, isolated_db):
        """help('orchestration') returns orchestration docs."""
        result = sdk.help("orchestration")
        # Should contain orchestration-specific methods
        assert "spawn" in result.lower() or "orchestrat" in result.lower()
        # Should mention explorer or coder
        assert "explorer" in result.lower() or "coder" in result.lower()

    def test_help_unknown_topic(self, sdk: SDK, isolated_db):
        """help('unknown') returns helpful message for unknown topics."""
        result = sdk.help("nonexistent_topic_xyz")
        # Should indicate topic wasn't found or show available topics
        assert result is not None
        assert len(result) > 0
        # Reasonable behavior: return empty string or helpful message
        # The implementation might return "" or a "not found" message

    def test_help_planning_topic(self, sdk: SDK, isolated_db):
        """help('planning') returns planning workflow docs."""
        result = sdk.help("planning")
        # Should contain planning-specific methods
        assert "plan" in result.lower() or "spike" in result.lower()

    def test_help_sessions_topic(self, sdk: SDK, isolated_db):
        """help('sessions') returns session management docs."""
        result = sdk.help("sessions")
        # Should contain session-specific methods
        assert "session" in result.lower()


class TestSDKDir:
    """Test __dir__ priority ordering for better tab completion."""

    def test_dir_returns_list(self, sdk: SDK, isolated_db):
        """__dir__ returns a list of strings."""
        attrs = dir(sdk)
        assert isinstance(attrs, list)
        assert all(isinstance(a, str) for a in attrs)

    def test_priority_collections_exist(self, sdk: SDK, isolated_db):
        """Key collection attributes should be in dir()."""
        attrs = dir(sdk)
        priority_collections = [
            "features",
            "bugs",
            "spikes",
            "sessions",
        ]
        for collection in priority_collections:
            assert collection in attrs, f"{collection} should be in dir(sdk)"

    def test_priority_orchestration_methods_exist(self, sdk: SDK, isolated_db):
        """Orchestration methods should be in dir()."""
        attrs = dir(sdk)
        orchestration_methods = ["spawn_explorer", "spawn_coder", "orchestrate"]
        for method in orchestration_methods:
            assert method in attrs, f"{method} should be in dir(sdk)"

    def test_help_method_exists(self, sdk: SDK, isolated_db):
        """help method should be in dir()."""
        attrs = dir(sdk)
        assert "help" in attrs

    def test_custom_dir_method_exists(self, sdk: SDK, isolated_db):
        """SDK should have custom __dir__ implementation."""
        assert hasattr(sdk, "__dir__")
        # Should return priority items when called directly
        direct = sdk.__dir__()
        assert isinstance(direct, list)
        assert len(direct) > 0

    def test_priority_items_come_first(self, sdk: SDK, isolated_db):
        """Priority items should appear before private methods."""
        # Use __dir__() directly to get custom ordering
        attrs = sdk.__dir__()

        # Find index of first priority item and first private method
        features_idx = attrs.index("features") if "features" in attrs else -1
        help_idx = attrs.index("help") if "help" in attrs else -1

        # Find first private method (starts with _)
        private_indices = [i for i, a in enumerate(attrs) if a.startswith("_")]
        first_private_idx = private_indices[0] if private_indices else len(attrs)

        # Priority items should come before private methods
        assert features_idx >= 0, "features should be in dir()"
        assert help_idx >= 0, "help should be in dir()"
        assert features_idx < first_private_idx, (
            "features should come before private methods"
        )
        assert help_idx < first_private_idx, "help should come before private methods"


class TestSDKDocstrings:
    """Test that key methods have docstrings for introspection."""

    def test_sdk_class_has_docstring(self, sdk: SDK, isolated_db):
        """SDK class should have comprehensive docstring."""
        assert SDK.__doc__ is not None
        assert len(SDK.__doc__) > 100
        # Should describe the purpose
        assert "agent" in SDK.__doc__.lower() or "sdk" in SDK.__doc__.lower()

    def test_spawn_explorer_has_docstring(self, sdk: SDK, isolated_db):
        """spawn_explorer should have docstring with see-also."""
        doc = sdk.spawn_explorer.__doc__
        assert doc is not None
        assert len(doc) > 50
        # Should mention it's for exploration/discovery
        assert "explorer" in doc.lower() or "discover" in doc.lower()
        # Should have see-also section
        assert "see also" in doc.lower()

    def test_spawn_coder_has_docstring(self, sdk: SDK, isolated_db):
        """spawn_coder should have docstring with see-also."""
        doc = sdk.spawn_coder.__doc__
        assert doc is not None
        assert len(doc) > 50
        # Should mention it's for implementation
        assert "coder" in doc.lower() or "implement" in doc.lower()
        # Should have see-also section
        assert "see also" in doc.lower()

    def test_help_method_has_docstring(self, sdk: SDK, isolated_db):
        """help method should have docstring."""
        doc = sdk.help.__doc__
        assert doc is not None
        assert "topic" in doc.lower() or "help" in doc.lower()

    def test_orchestrate_has_docstring(self, sdk: SDK, isolated_db):
        """orchestrate method should have docstring."""
        doc = sdk.orchestrate.__doc__
        assert doc is not None
        assert len(doc) > 50
        # Should mention orchestration
        assert "orchestrat" in doc.lower()

    def test_features_collection_has_docstring(self, sdk: SDK, isolated_db):
        """Features collection should be accessible and documented."""
        assert hasattr(sdk, "features")
        assert sdk.features is not None
        # Features should have a create method
        assert hasattr(sdk.features, "create")


class TestSDKDiscoverabilityIntegration:
    """Integration tests for discoverability features working together."""

    def test_help_lists_dir_priority_methods(self, sdk: SDK, isolated_db):
        """help() should list methods that appear in priority dir()."""
        help_text = sdk.help()
        dir_attrs = dir(sdk)

        # Key priority items should be in both
        assert "features" in help_text.lower()
        assert "features" in dir_attrs

        assert "spawn_explorer" in help_text or "orchestrat" in help_text.lower()
        assert "spawn_explorer" in dir_attrs

    def test_docstrings_align_with_help_topics(self, sdk: SDK, isolated_db):
        """Docstrings should provide details that align with help topics."""
        # Get orchestration help
        orch_help = sdk.help("orchestration")

        # Get spawn_explorer docstring
        explorer_doc = sdk.spawn_explorer.__doc__

        # Both should mention explorer/spawning
        assert "explorer" in orch_help.lower()
        assert "explorer" in explorer_doc.lower()

    def test_all_priority_methods_are_callable(self, sdk: SDK, isolated_db):
        """All priority methods in __dir__ should be callable or accessible."""
        priority_items = [
            "features",
            "bugs",
            "spikes",
            "spawn_explorer",
            "spawn_coder",
            "help",
        ]

        for item in priority_items:
            assert hasattr(sdk, item), f"SDK should have {item}"
            attr = getattr(sdk, item)
            # Should be callable or a collection object
            assert callable(attr) or hasattr(attr, "create")
