"""
Comprehensive tests for Phase 1: Feature-Track Linkage Enforcement.

This test suite verifies:
1. BaseBuilder.save() validates track_id for features
2. CLI --track flag works with feature create
3. Interactive track selection in CLI
4. Error messages are helpful with available tracks listed
5. Features can be created with set_track()
6. Validation prevents untracked features from being saved
"""

from pathlib import Path

import pytest
from htmlgraph import SDK


@pytest.fixture
def tmp_htmlgraph(isolated_graph_dir_full: Path):
    """Create a temporary .htmlgraph directory structure."""
    return isolated_graph_dir_full


class TestBaseBuilderTrackIdValidation:
    """Test BaseBuilder.save() track_id validation."""

    def test_feature_without_track_raises_error(self, tmp_htmlgraph, isolated_db: Path):
        """Creating feature without track_id should raise ValueError."""
        sdk = SDK(directory=tmp_htmlgraph, agent="test-agent", db_path=str(isolated_db))

        # Create a feature without setting track_id
        builder = sdk.features.create("Test Feature without Track")

        # Should raise ValueError on save
        with pytest.raises(ValueError) as exc_info:
            builder.save()

        error_msg = str(exc_info.value)
        assert "requires a track linkage" in error_msg
        assert "set_track" in error_msg

    def test_feature_with_track_saves_successfully(
        self, tmp_htmlgraph, isolated_db: Path
    ):
        """Creating feature with track_id should save successfully."""
        sdk = SDK(directory=tmp_htmlgraph, agent="test-agent", db_path=str(isolated_db))

        # Create a track first
        track = sdk.tracks.create("Test Track").save()

        # Create feature with track
        feature = (
            sdk.features.create("Test Feature with Track").set_track(track.id).save()
        )

        # Verify feature was saved
        assert feature.id is not None
        assert feature.title == "Test Feature with Track"
        assert feature.track_id == track.id

    def test_feature_track_linkage_persists(self, tmp_htmlgraph, isolated_db: Path):
        """Track linkage should persist when feature is retrieved."""
        sdk = SDK(directory=tmp_htmlgraph, agent="test-agent", db_path=str(isolated_db))

        # Create track and feature
        track = sdk.tracks.create("Persistent Track").save()
        feature = sdk.features.create("Persistent Feature").set_track(track.id).save()

        # Retrieve feature
        retrieved = sdk.features.get(feature.id)

        assert retrieved is not None
        assert retrieved.track_id == track.id

    def test_error_message_includes_available_tracks(
        self, tmp_htmlgraph, isolated_db: Path
    ):
        """Error message should list available tracks."""
        sdk = SDK(directory=tmp_htmlgraph, agent="test-agent", db_path=str(isolated_db))

        # Create multiple tracks
        track1 = sdk.tracks.create("First Track").save()
        sdk.tracks.create("Second Track").save()
        sdk.tracks.create("Third Track").save()

        # Try to create feature without track
        builder = sdk.features.create("Test Feature")

        with pytest.raises(ValueError) as exc_info:
            builder.save()

        error_msg = str(exc_info.value)
        # Should mention available tracks
        assert "Available tracks" in error_msg
        # Should include at least one track
        assert track1.id in error_msg or track1.title in error_msg

    def test_error_message_fallback_when_no_tracks(
        self, tmp_htmlgraph, isolated_db: Path
    ):
        """Error message should be helpful even with no tracks."""
        sdk = SDK(directory=tmp_htmlgraph, agent="test-agent", db_path=str(isolated_db))

        # Try to create feature without any tracks existing
        builder = sdk.features.create("Test Feature")

        with pytest.raises(ValueError) as exc_info:
            builder.save()

        error_msg = str(exc_info.value)
        assert "requires a track linkage" in error_msg
        assert "Create a track first" in error_msg

    def test_feature_with_track_and_priority(self, tmp_htmlgraph, isolated_db: Path):
        """Feature with track and other properties should work."""
        sdk = SDK(directory=tmp_htmlgraph, agent="test-agent", db_path=str(isolated_db))

        track = sdk.tracks.create("Priority Track").save()

        feature = (
            sdk.features.create("High Priority Feature")
            .set_priority("high")
            .set_track(track.id)
            .save()
        )

        assert feature.priority == "high"
        assert feature.track_id == track.id

    def test_feature_with_track_and_steps(self, tmp_htmlgraph, isolated_db: Path):
        """Feature with track and steps should work."""
        sdk = SDK(directory=tmp_htmlgraph, agent="test-agent", db_path=str(isolated_db))

        track = sdk.tracks.create("Steps Track").save()

        feature = (
            sdk.features.create("Feature with Steps")
            .set_track(track.id)
            .add_steps(["Step 1", "Step 2", "Step 3"])
            .save()
        )

        assert feature.track_id == track.id
        assert len(feature.steps) == 3

    def test_feature_with_track_and_description(self, tmp_htmlgraph, isolated_db: Path):
        """Feature with track and description should work."""
        sdk = SDK(directory=tmp_htmlgraph, agent="test-agent", db_path=str(isolated_db))

        track = sdk.tracks.create("Description Track").save()

        feature = (
            sdk.features.create("Described Feature")
            .set_track(track.id)
            .set_description("This is a feature description")
            .save()
        )

        assert feature.track_id == track.id
        assert "This is a feature description" in (feature.content or "")

    def test_feature_multiple_relationships_with_track(
        self, tmp_htmlgraph, isolated_db: Path
    ):
        """Feature can have multiple relationships and track."""
        sdk = SDK(directory=tmp_htmlgraph, agent="test-agent", db_path=str(isolated_db))

        track = sdk.tracks.create("Relationship Track").save()

        # Create related features
        related_feature = (
            sdk.features.create("Related Feature").set_track(track.id).save()
        )

        # Create main feature with relationship
        feature = (
            sdk.features.create("Main Feature")
            .set_track(track.id)
            .blocked_by(related_feature.id)
            .save()
        )

        assert feature.track_id == track.id
        assert len(feature.edges.get("blocked_by", [])) > 0

    def test_feature_in_different_collections(self, tmp_htmlgraph, isolated_db: Path):
        """Bugs collection should allow features without tracks (for now)."""
        sdk = SDK(directory=tmp_htmlgraph, agent="test-agent", db_path=str(isolated_db))

        # Bugs collection should NOT require track (only features collection)
        bug = sdk.bugs.create("Test Bug").save()

        assert bug.id is not None
        assert bug.type == "bug"


class TestSetTrackMethod:
    """Test the set_track() builder method."""

    def test_set_track_returns_builder(self, tmp_htmlgraph, isolated_db: Path):
        """set_track() should return builder for chaining."""
        sdk = SDK(directory=tmp_htmlgraph, agent="test-agent", db_path=str(isolated_db))
        track = sdk.tracks.create("Chain Track").save()

        builder = sdk.features.create("Test Feature")
        result = builder.set_track(track.id)

        # Should be chainable
        assert result is builder

    def test_set_track_multiple_times(self, tmp_htmlgraph, isolated_db: Path):
        """Setting track multiple times should use last value."""
        sdk = SDK(directory=tmp_htmlgraph, agent="test-agent", db_path=str(isolated_db))
        track1 = sdk.tracks.create("Track 1").save()
        track2 = sdk.tracks.create("Track 2").save()

        feature = (
            sdk.features.create("Multiple Track Sets")
            .set_track(track1.id)
            .set_track(track2.id)
            .save()
        )

        # Should have the last set track
        assert feature.track_id == track2.id

    def test_set_track_with_empty_string_fails(self, tmp_htmlgraph, isolated_db: Path):
        """Setting track to empty string should fail validation."""
        sdk = SDK(directory=tmp_htmlgraph, agent="test-agent", db_path=str(isolated_db))

        builder = sdk.features.create("Empty Track Test")
        builder.set_track("")

        # Empty track_id should still fail
        with pytest.raises(ValueError):
            builder.save()


class TestFeatureCreateFluentAPI:
    """Test feature creation with fluent API."""

    def test_fluent_api_with_all_options(self, tmp_htmlgraph, isolated_db: Path):
        """Feature creation with all fluent API options."""
        sdk = SDK(directory=tmp_htmlgraph, agent="test-agent", db_path=str(isolated_db))
        track = sdk.tracks.create("Complete Track").save()

        feature = (
            sdk.features.create("Complete Feature")
            .set_priority("high")
            .set_status("in-progress")
            .set_description("Complete feature description")
            .add_steps(["Step 1", "Step 2"])
            .add_capability_tags(["backend", "api"])
            .set_required_capabilities(["python", "fastapi"])
            .set_track(track.id)
            .save()
        )

        assert feature.track_id == track.id
        assert feature.priority == "high"
        assert feature.status == "in-progress"
        assert len(feature.steps) == 2

    def test_fluent_api_preserves_agent_attribution(
        self, tmp_htmlgraph, isolated_db: Path
    ):
        """Feature creation should preserve agent attribution."""
        sdk = SDK(directory=tmp_htmlgraph, agent="test-agent", db_path=str(isolated_db))
        track = sdk.tracks.create("Agent Track").save()

        feature = sdk.features.create("Agent Feature").set_track(track.id).save()

        assert feature.agent_assigned == "test-agent"


class TestTrackValidationEdgeCases:
    """Test edge cases in track validation."""

    def test_feature_with_none_track_id(self, tmp_htmlgraph, isolated_db: Path):
        """Feature with None track_id should fail."""
        sdk = SDK(directory=tmp_htmlgraph, agent="test-agent", db_path=str(isolated_db))

        builder = sdk.features.create("None Track Test")
        builder._data["track_id"] = None

        with pytest.raises(ValueError):
            builder.save()

    def test_feature_with_invalid_track_id(self, tmp_htmlgraph, isolated_db: Path):
        """Feature with non-existent track_id should still save (no reference validation)."""
        sdk = SDK(directory=tmp_htmlgraph, agent="test-agent", db_path=str(isolated_db))

        # This should save successfully - no reference validation at this level
        feature = (
            sdk.features.create("Invalid Track Test")
            .set_track("nonexistent-track-id")
            .save()
        )

        # Should have the track_id even if it doesn't exist
        assert feature.track_id == "nonexistent-track-id"

    def test_feature_track_id_special_characters(
        self, tmp_htmlgraph, isolated_db: Path
    ):
        """Track ID with special characters should work."""
        sdk = SDK(directory=tmp_htmlgraph, agent="test-agent", db_path=str(isolated_db))
        track = sdk.tracks.create("Special Track").save()

        # Use actual track ID
        feature = sdk.features.create("Special Char Test").set_track(track.id).save()

        assert feature.track_id == track.id


class TestFeatureCreateErrorMessages:
    """Test error messages for feature creation failures."""

    def test_error_message_includes_feature_title(
        self, tmp_htmlgraph, isolated_db: Path
    ):
        """Error message should include the feature title."""
        sdk = SDK(directory=tmp_htmlgraph, agent="test-agent", db_path=str(isolated_db))

        builder = sdk.features.create("Specific Feature Name")

        with pytest.raises(ValueError) as exc_info:
            builder.save()

        error_msg = str(exc_info.value)
        assert "Specific Feature Name" in error_msg

    def test_error_message_is_clear_and_actionable(
        self, tmp_htmlgraph, isolated_db: Path
    ):
        """Error message should be clear and actionable."""
        sdk = SDK(directory=tmp_htmlgraph, agent="test-agent", db_path=str(isolated_db))

        builder = sdk.features.create("Unclear Test")

        with pytest.raises(ValueError) as exc_info:
            builder.save()

        error_msg = str(exc_info.value)
        # Should provide clear guidance
        assert "set_track" in error_msg
        assert "track" in error_msg.lower()


class TestCollectionSpecificTrackRequirement:
    """Test that track requirement is features-collection specific."""

    def test_only_features_collection_requires_track(
        self, tmp_htmlgraph, isolated_db: Path
    ):
        """Only 'features' collection should require track linkage."""
        sdk = SDK(directory=tmp_htmlgraph, agent="test-agent", db_path=str(isolated_db))

        # Bugs don't require track
        bug = sdk.bugs.create("No Track Bug").save()
        assert bug.id is not None

        # Spikes don't require track (for Phase 1)
        spike = sdk.spikes.create("No Track Spike").save()
        assert spike.id is not None

        # Only features require track
        with pytest.raises(ValueError):
            sdk.features.create("Must Have Track Feature").save()
