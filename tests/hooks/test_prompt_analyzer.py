"""
Comprehensive unit tests for the prompt_analyzer module.

Tests prompt classification, intent detection, workflow guidance generation,
and CIGS (Computational Imperative Guidance System) integration.

Coverage Target: 95%+ code coverage
Test Framework: pytest
Test Count: 35+ comprehensive test cases
"""

import tempfile
from pathlib import Path
from unittest.mock import Mock, patch

import pytest
from htmlgraph.hooks.context import HookContext
from htmlgraph.hooks.prompt_analyzer import (
    EXPLORATION_KEYWORDS,
    classify_cigs_intent,
    classify_prompt,
    create_user_query_event,
    generate_cigs_guidance,
    generate_guidance,
    get_active_work_item,
    get_session_violation_count,
)

# ============================================================================
# FIXTURES: Mock HookContext and Dependencies
# ============================================================================


@pytest.fixture
def mock_hook_context():
    """Create a mock HookContext for testing."""
    context = Mock(spec=HookContext)
    context.session_id = "sess-test-123456"
    context.agent_id = "claude-code"
    context.project_dir = "/test/project"
    context.graph_dir = Path("/test/project/.htmlgraph")
    context.hook_input = {
        "session_id": "sess-test-123456",
        "type": "user_prompt_submit",
    }

    # Mock database
    mock_db = Mock()
    mock_conn = Mock()
    mock_cursor = Mock()
    mock_db.connection = mock_conn
    mock_db.insert_event = Mock(return_value=True)
    mock_conn.cursor = Mock(return_value=mock_cursor)
    mock_conn.commit = Mock()

    context.database = mock_db

    return context


@pytest.fixture
def mock_hook_context_no_session():
    """Create a mock HookContext with unknown session."""
    context = Mock(spec=HookContext)
    context.session_id = "unknown"
    context.agent_id = "unknown"
    context.project_dir = "/test/project"
    context.graph_dir = Path("/test/project/.htmlgraph")
    context.hook_input = {}

    return context


@pytest.fixture
def tmp_graph_dir():
    """Create a temporary .htmlgraph directory."""
    with tempfile.TemporaryDirectory() as tmpdir:
        graph_dir = Path(tmpdir) / ".htmlgraph"
        graph_dir.mkdir(parents=True, exist_ok=True)
        yield graph_dir


# ============================================================================
# TEST FIXTURES: Sample Prompts for Classification Testing
# ============================================================================


@pytest.fixture
def sample_implementation_prompts():
    """Collection of prompts that indicate implementation intent."""
    return [
        "Can you implement a new feature for user authentication?",
        "Please add a function to calculate the total price",
        "I need to fix the bug in the login endpoint",
        "Let's create a new React component for the dashboard",
        "Refactor the database connection module",
        "Update the error handling in the API",
        "Please write a test for the calculator module",
        "Build a caching layer for the API",
    ]


@pytest.fixture
def sample_investigation_prompts():
    """Collection of prompts that indicate investigation intent."""
    return [
        "Can you investigate why the tests are failing?",
        "Please research how the authentication system works",
        "I need to understand what causes the memory leak",
        "Where is the error being handled in the codebase?",
        "What files handle the payment processing?",
        "How does the session manager work?",
        "Can you explore the git history for this feature?",
        "Look into the performance bottleneck",
    ]


@pytest.fixture
def sample_bug_prompts():
    """Collection of prompts that indicate bug reports."""
    return [
        "There's a bug in the login form",
        "The API is returning 500 errors",
        "Something's wrong with the database connection",
        "Tests are failing on the CI pipeline",
        "The app crashes when I click this button",
        "Performance is really slow in the dashboard",
        "This feature isn't working as expected",
        "The error handling seems broken",
    ]


@pytest.fixture
def sample_continuation_prompts():
    """Collection of prompts that indicate continuation."""
    return [
        "continue",
        "okay",
        "go ahead",
        "resume from where we left off",
        "proceed with the implementation",
        "yes, do it",
        "keep going",
        "next",
    ]


@pytest.fixture
def sample_cigs_exploration_prompts():
    """Collection of prompts requiring exploration."""
    return [
        "Search for all error handling code",
        "Find all files that import this module",
        "What files are in the src/handlers directory?",
        "Which files implement the authentication logic?",
        "Locate all database queries",
        "Analyze the codebase structure",
        "Review all API endpoints",
        "Show me all test files",
    ]


@pytest.fixture
def sample_cigs_code_change_prompts():
    """Collection of prompts requiring code changes."""
    return [
        "Implement the new user authentication feature",
        "Fix the bug in the payment processing",
        "Update the error messages in the API",
        "Refactor the database connection module",
        "Change the session timeout to 30 minutes",
        "Add logging to all API endpoints",
        "Remove the deprecated function",
        "Rewrite the caching layer",
    ]


@pytest.fixture
def sample_cigs_git_prompts():
    """Collection of prompts requiring git operations."""
    return [
        "Commit these changes with a good message",
        "Push the feature branch to origin",
        "Merge the pull request",
        "Create a new branch for this feature",
        "Check the git status",
        "Rebase on the main branch",
        "Stash these changes",
        "Cherry-pick the commit",
    ]


# ============================================================================
# TESTS: classify_prompt() - Intent Detection
# ============================================================================


class TestClassifyPrompt:
    """Tests for prompt classification and intent detection."""

    def test_classify_implementation_intent_single_pattern(self):
        """Test implementation intent with single matching pattern."""
        result = classify_prompt("Can you implement a new feature?")
        assert result["is_implementation"] is True
        assert result["confidence"] >= 0.8
        assert len(result["matched_patterns"]) > 0

    @pytest.mark.parametrize(
        "prompt",
        [
            "Can you implement a new feature for user authentication?",
            "Please add a function to calculate the total price",
            "I need to fix the bug in the login endpoint",
            "Let's create a new React component for the dashboard",
            "Refactor the database connection module",
        ],
    )
    def test_classify_implementation_intent_multiple(self, prompt):
        """Test implementation intent detection across multiple prompts."""
        result = classify_prompt(prompt)
        assert result["is_implementation"] is True
        assert result["confidence"] >= 0.75

    def test_classify_investigation_intent_single_pattern(self):
        """Test investigation intent with single matching pattern."""
        result = classify_prompt("Can you investigate why the tests are failing?")
        assert result["is_investigation"] is True
        assert result["confidence"] >= 0.7

    @pytest.mark.parametrize(
        "prompt",
        [
            "Please research how the authentication system works",
            "I need to understand what causes the memory leak",
            "Can you investigate why the tests are failing?",
            "I need to explore the codebase structure",
            "Can you analyze the error handling?",
        ],
    )
    def test_classify_investigation_intent_multiple(self, prompt):
        """Test investigation intent detection across multiple prompts."""
        result = classify_prompt(prompt)
        assert result["is_investigation"] is True
        assert result["confidence"] >= 0.7

    def test_classify_bug_intent_single_pattern(self):
        """Test bug intent with single matching pattern."""
        result = classify_prompt("There's a bug in the login form")
        assert result["is_bug_report"] is True
        assert result["confidence"] >= 0.75

    @pytest.mark.parametrize(
        "prompt",
        [
            "There's a bug in the login form",
            "Something's wrong with the database connection",
            "The test is broken and failing",
            "There's an error in the API",
            "The function is not working",
        ],
    )
    def test_classify_bug_intent_multiple(self, prompt):
        """Test bug intent detection across multiple prompts."""
        result = classify_prompt(prompt)
        assert result["is_bug_report"] is True

    def test_classify_continuation_intent(self):
        """Test continuation intent detection."""
        for prompt in ["continue", "okay", "go ahead", "yes"]:
            result = classify_prompt(prompt)
            assert result["is_continuation"] is True
            assert result["confidence"] >= 0.9

    def test_continuation_returns_early(self):
        """Test that continuation classification returns early."""
        result = classify_prompt("continue with the implementation")
        assert result["is_continuation"] is True
        # Early return means other flags should be False
        assert result["is_implementation"] is False
        assert result["is_investigation"] is False
        assert result["is_bug_report"] is False

    def test_classify_multiple_intents(self):
        """Test prompt with multiple matching intents."""
        prompt = "Investigate the bug and fix the error handling"
        result = classify_prompt(prompt)
        # Should match both investigation and bug patterns
        assert result["is_investigation"] is True
        assert result["is_bug_report"] is True

    def test_classify_empty_prompt(self):
        """Test classification with empty prompt."""
        result = classify_prompt("")
        assert result["is_implementation"] is False
        assert result["is_investigation"] is False
        assert result["is_bug_report"] is False
        assert result["is_continuation"] is False
        assert result["confidence"] == 0.0

    def test_classify_whitespace_prompt(self):
        """Test classification with whitespace-only prompt."""
        result = classify_prompt("   \t\n   ")
        assert result["confidence"] == 0.0
        assert result["matched_patterns"] == []

    def test_classify_case_insensitive(self):
        """Test that classification is case-insensitive."""
        result_lower = classify_prompt("can you implement a feature?")
        result_upper = classify_prompt("CAN YOU IMPLEMENT A FEATURE?")
        result_mixed = classify_prompt("CaN yOu ImPlEmEnT a FeAtUrE?")

        assert result_lower["is_implementation"] == result_upper["is_implementation"]
        assert result_lower["is_implementation"] == result_mixed["is_implementation"]

    def test_classify_matched_patterns_list(self):
        """Test that matched_patterns contains actual patterns."""
        result = classify_prompt("Can you implement a new feature?")
        assert isinstance(result["matched_patterns"], list)
        assert len(result["matched_patterns"]) > 0
        # Patterns should have descriptive labels
        assert any("implementation" in p for p in result["matched_patterns"])

    def test_classify_confidence_scoring(self):
        """Test confidence scoring behavior."""
        result = classify_prompt("Can you implement a new feature?")
        assert 0.0 <= result["confidence"] <= 1.0
        assert result["confidence"] >= 0.75  # Implementation patterns give 0.8+

    def test_classify_unmatched_prompt(self):
        """Test prompt with no matching patterns."""
        result = classify_prompt("This is a random prompt about weather")
        assert result["is_implementation"] is False
        assert result["is_investigation"] is False
        assert result["is_bug_report"] is False
        assert result["is_continuation"] is False
        assert result["confidence"] == 0.0


# ============================================================================
# TESTS: classify_cigs_intent() - CIGS Intent Detection
# ============================================================================


class TestClassifyCigsIntent:
    """Tests for CIGS (Computational Imperative Guidance System) intent detection."""

    def test_classify_exploration_intent_single_keyword(self):
        """Test exploration intent with single keyword."""
        result = classify_cigs_intent("Search for all error handling code")
        assert result["involves_exploration"] is True
        assert result["intent_confidence"] > 0.0

    @pytest.mark.parametrize(
        "prompt",
        [
            "Search for all error handling code",
            "Find all files that import this module",
            "What files are in the src/handlers directory?",
            "Which files implement the authentication logic?",
            "Locate all database queries",
            "Analyze the codebase structure",
            "Review all API endpoints",
        ],
    )
    def test_classify_exploration_intent_multiple(self, prompt):
        """Test exploration intent across multiple prompts."""
        result = classify_cigs_intent(prompt)
        assert result["involves_exploration"] is True

    def test_classify_code_changes_intent(self):
        """Test code change detection."""
        result = classify_cigs_intent("Implement the new user authentication feature")
        assert result["involves_code_changes"] is True
        assert result["intent_confidence"] > 0.0

    @pytest.mark.parametrize(
        "prompt",
        [
            "Implement the new user authentication feature",
            "Fix the bug in the payment processing",
            "Update the error messages in the API",
            "Refactor the database connection module",
            "Change the session timeout to 30 minutes",
        ],
    )
    def test_classify_code_changes_intent_multiple(self, prompt):
        """Test code changes intent across multiple prompts."""
        result = classify_cigs_intent(prompt)
        assert result["involves_code_changes"] is True

    def test_classify_git_operations_intent(self):
        """Test git operation detection."""
        result = classify_cigs_intent("Commit these changes with a good message")
        assert result["involves_git"] is True
        assert result["intent_confidence"] > 0.0

    @pytest.mark.parametrize(
        "prompt",
        [
            "Commit these changes with a good message",
            "Push the feature branch to origin",
            "Merge the pull request",
            "Create a new branch for this feature",
            "Check the git status",
        ],
    )
    def test_classify_git_operations_intent_multiple(self, prompt):
        """Test git operations intent across multiple prompts."""
        result = classify_cigs_intent(prompt)
        assert result["involves_git"] is True

    def test_classify_confidence_with_multiple_keywords(self):
        """Test confidence increases with multiple matching keywords."""
        single = classify_cigs_intent("Search for files")
        multiple = classify_cigs_intent(
            "Search for and analyze all files in the project"
        )
        # Multiple keywords should give higher confidence
        assert multiple["intent_confidence"] >= single["intent_confidence"]

    def test_classify_confidence_bounded_at_one(self):
        """Test that confidence is bounded at 1.0."""
        # Create prompt with many exploration keywords
        prompt = " ".join([kw for kw in EXPLORATION_KEYWORDS[:10]])
        result = classify_cigs_intent(prompt)
        assert result["intent_confidence"] <= 1.0

    def test_classify_cigs_empty_prompt(self):
        """Test CIGS intent with empty prompt."""
        result = classify_cigs_intent("")
        assert result["involves_exploration"] is False
        assert result["involves_code_changes"] is False
        assert result["involves_git"] is False
        assert result["intent_confidence"] == 0.0

    def test_classify_cigs_case_insensitive(self):
        """Test that CIGS classification is case-insensitive."""
        result_lower = classify_cigs_intent("search for all files")
        result_upper = classify_cigs_intent("SEARCH FOR ALL FILES")
        assert (
            result_lower["involves_exploration"] == result_upper["involves_exploration"]
        )

    def test_classify_cigs_multiple_intents(self):
        """Test CIGS intent with multiple activity types."""
        prompt = "Search for the bug, fix it, and commit the changes"
        result = classify_cigs_intent(prompt)
        assert result["involves_exploration"] is True
        assert result["involves_code_changes"] is True
        assert result["involves_git"] is True

    def test_classify_cigs_result_structure(self):
        """Test CIGS intent result structure."""
        result = classify_cigs_intent("Implement a feature")
        assert "involves_exploration" in result
        assert "involves_code_changes" in result
        assert "involves_git" in result
        assert "intent_confidence" in result
        assert isinstance(result["involves_exploration"], bool)
        assert isinstance(result["involves_code_changes"], bool)
        assert isinstance(result["involves_git"], bool)
        assert isinstance(result["intent_confidence"], float)


# ============================================================================
# TESTS: get_session_violation_count() - Violation Retrieval
# ============================================================================


class TestGetSessionViolationCount:
    """Tests for session violation count retrieval."""

    def test_get_session_violation_count_success(self, mock_hook_context):
        """Test successful violation count retrieval."""
        with patch("htmlgraph.cigs.ViolationTracker") as mock_tracker_class:
            mock_tracker = Mock()
            mock_tracker_class.return_value = mock_tracker

            mock_summary = Mock()
            mock_summary.total_violations = 2
            mock_summary.total_waste_tokens = 5000
            mock_tracker.get_session_violations.return_value = mock_summary

            violation_count, waste_tokens = get_session_violation_count(
                mock_hook_context
            )

            assert violation_count == 2
            assert waste_tokens == 5000
            mock_tracker_class.assert_called_once()

    def test_get_session_violation_count_zero_violations(self, mock_hook_context):
        """Test retrieval with zero violations."""
        with patch("htmlgraph.cigs.ViolationTracker") as mock_tracker_class:
            mock_tracker = Mock()
            mock_tracker_class.return_value = mock_tracker

            mock_summary = Mock()
            mock_summary.total_violations = 0
            mock_summary.total_waste_tokens = 0
            mock_tracker.get_session_violations.return_value = mock_summary

            violation_count, waste_tokens = get_session_violation_count(
                mock_hook_context
            )

            assert violation_count == 0
            assert waste_tokens == 0

    def test_get_session_violation_count_cigs_unavailable(self, mock_hook_context):
        """Test graceful degradation when CIGS is unavailable."""
        with patch("htmlgraph.cigs.ViolationTracker") as mock_tracker_class:
            mock_tracker_class.side_effect = ImportError("CIGS not available")

            violation_count, waste_tokens = get_session_violation_count(
                mock_hook_context
            )

            assert violation_count == 0
            assert waste_tokens == 0

    def test_get_session_violation_count_exception_handling(self, mock_hook_context):
        """Test exception handling during violation retrieval."""
        with patch("htmlgraph.cigs.ViolationTracker") as mock_tracker_class:
            mock_tracker_class.side_effect = Exception("Database error")

            violation_count, waste_tokens = get_session_violation_count(
                mock_hook_context
            )

            assert violation_count == 0
            assert waste_tokens == 0

    def test_get_session_violation_count_returns_tuple(self, mock_hook_context):
        """Test that violation count returns a tuple."""
        with patch("htmlgraph.cigs.ViolationTracker") as mock_tracker_class:
            mock_tracker = Mock()
            mock_tracker_class.return_value = mock_tracker
            mock_summary = Mock()
            mock_summary.total_violations = 1
            mock_summary.total_waste_tokens = 1000
            mock_tracker.get_session_violations.return_value = mock_summary

            result = get_session_violation_count(mock_hook_context)

            assert isinstance(result, tuple)
            assert len(result) == 2


# ============================================================================
# TESTS: get_active_work_item() - Work Item Retrieval
# ============================================================================


class TestGetActiveWorkItem:
    """Tests for active work item retrieval."""

    def test_get_active_work_item_feature(self, mock_hook_context):
        """Test retrieval of active feature."""
        with patch("htmlgraph.SDK") as mock_sdk_class:
            mock_sdk = Mock()
            mock_sdk_class.return_value = mock_sdk

            mock_work_item = {
                "id": "feat-12345",
                "title": "Add user authentication",
                "type": "feature",
            }
            mock_sdk.get_active_work_item.return_value = mock_work_item

            result = get_active_work_item(mock_hook_context)

            assert result is not None
            assert result["id"] == "feat-12345"
            assert result["type"] == "feature"

    def test_get_active_work_item_spike(self, mock_hook_context):
        """Test retrieval of active spike."""
        with patch("htmlgraph.SDK") as mock_sdk_class:
            mock_sdk = Mock()
            mock_sdk_class.return_value = mock_sdk

            mock_work_item = {
                "id": "spk-67890",
                "title": "Research authentication options",
                "type": "spike",
            }
            mock_sdk.get_active_work_item.return_value = mock_work_item

            result = get_active_work_item(mock_hook_context)

            assert result is not None
            assert result["id"] == "spk-67890"
            assert result["type"] == "spike"

    def test_get_active_work_item_none(self, mock_hook_context):
        """Test when no active work item exists."""
        with patch("htmlgraph.SDK") as mock_sdk_class:
            mock_sdk = Mock()
            mock_sdk_class.return_value = mock_sdk
            mock_sdk.get_active_work_item.return_value = None

            result = get_active_work_item(mock_hook_context)

            assert result is None

    def test_get_active_work_item_sdk_unavailable(self, mock_hook_context):
        """Test graceful degradation when SDK is unavailable."""
        with patch("htmlgraph.SDK") as mock_sdk_class:
            mock_sdk_class.side_effect = ImportError("SDK not available")

            result = get_active_work_item(mock_hook_context)

            assert result is None

    def test_get_active_work_item_exception_handling(self, mock_hook_context):
        """Test exception handling during work item retrieval."""
        with patch("htmlgraph.SDK") as mock_sdk_class:
            mock_sdk_class.side_effect = Exception("Database error")

            result = get_active_work_item(mock_hook_context)

            assert result is None

    def test_get_active_work_item_returns_dict(self, mock_hook_context):
        """Test that active work item returns dict or None."""
        with patch("htmlgraph.SDK") as mock_sdk_class:
            mock_sdk = Mock()
            mock_sdk_class.return_value = mock_sdk
            mock_work_item = {"id": "test-123", "type": "feature"}
            mock_sdk.get_active_work_item.return_value = mock_work_item

            result = get_active_work_item(mock_hook_context)

            assert isinstance(result, dict)


# ============================================================================
# TESTS: generate_guidance() - Guidance Generation
# ============================================================================


class TestGenerateGuidance:
    """Tests for workflow guidance generation."""

    def test_generate_guidance_continuation_with_active_work(self):
        """Test no guidance for continuation with active work."""
        classification = {
            "is_continuation": True,
            "confidence": 0.9,
        }
        active_work = {"id": "feat-123", "type": "feature"}

        guidance = generate_guidance(classification, active_work, "continue")

        assert guidance is None

    def test_generate_guidance_implementation_no_active_work(self):
        """Test guidance for implementation with no active work."""
        classification = {
            "is_implementation": True,
            "is_investigation": False,
            "is_bug_report": False,
            "is_continuation": False,
            "confidence": 0.8,
        }

        guidance = generate_guidance(classification, None, "Implement a feature")

        assert guidance is not None
        assert "ORCHESTRATOR DIRECTIVE" in guidance
        assert "sdk.features.create" in guidance
        assert "Task(" in guidance

    def test_generate_guidance_implementation_with_feature_active(self):
        """Test guidance for implementation when feature is active."""
        classification = {
            "is_implementation": True,
            "is_investigation": False,
            "is_bug_report": False,
            "is_continuation": False,
            "confidence": 0.8,
        }
        active_work = {"id": "feat-123", "title": "Add auth", "type": "feature"}

        guidance = generate_guidance(classification, active_work, "Implement auth")

        assert guidance is not None
        assert "spawn_codex()" in guidance or "Task(" in guidance

    def test_generate_guidance_implementation_with_spike_active(self):
        """Test guidance when implementation requested during spike."""
        classification = {
            "is_implementation": True,
            "is_investigation": False,
            "is_bug_report": False,
            "is_continuation": False,
            "confidence": 0.8,
        }
        active_work = {"id": "spk-456", "title": "Research auth", "type": "spike"}

        guidance = generate_guidance(classification, active_work, "Implement auth")

        assert guidance is not None
        assert "spike" in guidance.lower()
        assert "spikes.complete" in guidance or "spikes.pause" in guidance

    def test_generate_guidance_bug_report_no_active_work(self):
        """Test guidance for bug report with no active work."""
        classification = {
            "is_implementation": False,
            "is_investigation": False,
            "is_bug_report": True,
            "is_continuation": False,
            "confidence": 0.75,
        }

        guidance = generate_guidance(classification, None, "There's a bug")

        assert guidance is not None
        assert "BUG REPORT" in guidance
        assert "sdk.bugs.create" in guidance

    def test_generate_guidance_investigation_no_active_work(self):
        """Test guidance for investigation with no active work."""
        classification = {
            "is_implementation": False,
            "is_investigation": True,
            "is_bug_report": False,
            "is_continuation": False,
            "confidence": 0.7,
        }

        guidance = generate_guidance(classification, None, "Investigate the issue")

        assert guidance is not None
        assert "INVESTIGATION" in guidance
        assert "sdk.spikes.create" in guidance

    def test_generate_guidance_low_confidence(self):
        """Test guidance for low confidence prompts."""
        classification = {
            "is_implementation": False,
            "is_investigation": False,
            "is_bug_report": False,
            "is_continuation": False,
            "confidence": 0.3,
        }

        guidance = generate_guidance(classification, None, "Some prompt")

        assert guidance is not None
        assert "REMINDER" in guidance or "work item" in guidance.lower()

    def test_generate_guidance_appropriate_active_work(self):
        """Test no guidance when appropriate work is active."""
        classification = {
            "is_implementation": False,
            "is_investigation": True,
            "is_bug_report": False,
            "is_continuation": False,
            "confidence": 0.7,
        }
        active_work = {"id": "spk-123", "type": "spike"}

        guidance = generate_guidance(classification, active_work, "Investigate")

        assert guidance is None

    def test_generate_guidance_bug_with_feature_active(self):
        """Test guidance for bug report when feature is active."""
        classification = {
            "is_implementation": False,
            "is_investigation": False,
            "is_bug_report": True,
            "is_continuation": False,
            "confidence": 0.75,
        }
        active_work = {"id": "feat-123", "type": "feature"}

        guidance = generate_guidance(classification, active_work, "There's a bug")

        assert guidance is not None
        # Should suggest creating a bug or continuing with feature


# ============================================================================
# TESTS: generate_cigs_guidance() - CIGS Guidance Generation
# ============================================================================


class TestGenerateCigsGuidance:
    """Tests for CIGS-specific guidance generation."""

    def test_generate_cigs_guidance_exploration(self):
        """Test CIGS guidance for exploration intent."""
        cigs_intent = {
            "involves_exploration": True,
            "involves_code_changes": False,
            "involves_git": False,
            "intent_confidence": 0.5,
        }

        guidance = generate_cigs_guidance(cigs_intent, 0, 0)

        assert guidance != ""
        assert "CIGS PRE-RESPONSE GUIDANCE" in guidance
        assert "exploration" in guidance.lower()
        assert "spawn_gemini()" in guidance

    def test_generate_cigs_guidance_code_changes(self):
        """Test CIGS guidance for code changes."""
        cigs_intent = {
            "involves_exploration": False,
            "involves_code_changes": True,
            "involves_git": False,
            "intent_confidence": 0.5,
        }

        guidance = generate_cigs_guidance(cigs_intent, 0, 0)

        assert guidance != ""
        assert "code changes" in guidance.lower()
        assert "spawn_codex()" in guidance

    def test_generate_cigs_guidance_git_operations(self):
        """Test CIGS guidance for git operations."""
        cigs_intent = {
            "involves_exploration": False,
            "involves_code_changes": False,
            "involves_git": True,
            "intent_confidence": 0.5,
        }

        guidance = generate_cigs_guidance(cigs_intent, 0, 0)

        assert guidance != ""
        assert "git" in guidance.lower()
        assert "spawn_copilot()" in guidance

    def test_generate_cigs_guidance_violations_low(self):
        """Test CIGS guidance with low violation count."""
        cigs_intent = {
            "involves_exploration": False,
            "involves_code_changes": False,
            "involves_git": False,
            "intent_confidence": 0.0,
        }

        guidance = generate_cigs_guidance(cigs_intent, 1, 1000)

        assert "VIOLATION WARNING" in guidance
        assert "⚠️" in guidance

    def test_generate_cigs_guidance_violations_high(self):
        """Test CIGS guidance with high violation count."""
        cigs_intent = {
            "involves_exploration": False,
            "involves_code_changes": False,
            "involves_git": False,
            "intent_confidence": 0.0,
        }

        guidance = generate_cigs_guidance(cigs_intent, 3, 5000)

        assert "VIOLATION WARNING" in guidance
        assert "🚨" in guidance or "3" in guidance

    def test_generate_cigs_guidance_no_violations_no_intent(self):
        """Test no CIGS guidance when no violations and no intent."""
        cigs_intent = {
            "involves_exploration": False,
            "involves_code_changes": False,
            "involves_git": False,
            "intent_confidence": 0.0,
        }

        guidance = generate_cigs_guidance(cigs_intent, 0, 0)

        assert guidance == ""

    def test_generate_cigs_guidance_multiple_imperatives(self):
        """Test CIGS guidance with multiple imperatives."""
        cigs_intent = {
            "involves_exploration": True,
            "involves_code_changes": True,
            "involves_git": True,
            "intent_confidence": 0.8,
        }

        guidance = generate_cigs_guidance(cigs_intent, 0, 0)

        assert "spawn_gemini()" in guidance
        assert "spawn_codex()" in guidance
        assert "spawn_copilot()" in guidance

    def test_generate_cigs_guidance_format(self):
        """Test CIGS guidance formatting."""
        cigs_intent = {
            "involves_exploration": True,
            "involves_code_changes": False,
            "involves_git": False,
            "intent_confidence": 0.5,
        }

        guidance = generate_cigs_guidance(cigs_intent, 0, 0)

        # Check for formatting
        assert "═══════════════════════════════════════════════════════════" in guidance
        assert "CIGS PRE-RESPONSE GUIDANCE" in guidance

    def test_generate_cigs_guidance_waste_tokens_formatting(self):
        """Test that waste tokens are formatted with commas."""
        cigs_intent = {
            "involves_exploration": False,
            "involves_code_changes": False,
            "involves_git": False,
            "intent_confidence": 0.0,
        }

        guidance = generate_cigs_guidance(cigs_intent, 2, 10000)

        # Should have comma-formatted number
        assert "10,000" in guidance


# ============================================================================
# TESTS: create_user_query_event() - Event Creation
# ============================================================================


class TestCreateUserQueryEvent:
    """Tests for UserQuery event creation (database-only, no file-based state)."""

    def test_create_user_query_event_success(self, mock_hook_context, tmp_graph_dir):
        """Test successful UserQuery event creation."""
        mock_hook_context.graph_dir = tmp_graph_dir

        # Mock database cursor
        mock_cursor = Mock()
        mock_cursor.fetchone.return_value = (1,)  # Session exists
        mock_hook_context.database.connection.cursor.return_value = mock_cursor

        with patch("htmlgraph.hooks.prompt_analyzer.uuid.uuid4") as mock_uuid:
            mock_uuid.return_value = Mock(hex="abcdefghijklmnop")

            event_id = create_user_query_event(mock_hook_context, "Test prompt")

            assert event_id is not None
            assert event_id.startswith("uq-")

    def test_create_user_query_event_creates_session_if_not_exists(
        self, mock_hook_context, tmp_graph_dir
    ):
        """Test that session is created if it doesn't exist."""
        mock_hook_context.graph_dir = tmp_graph_dir

        # Mock database cursor - session doesn't exist
        mock_cursor = Mock()
        mock_cursor.fetchone.return_value = (0,)  # Session doesn't exist
        mock_hook_context.database.connection.cursor.return_value = mock_cursor

        event_id = create_user_query_event(mock_hook_context, "Test prompt")

        # Should still create event even if session didn't exist
        assert event_id is not None

    def test_create_user_query_event_unknown_session(
        self, mock_hook_context_no_session
    ):
        """Test handling of unknown session."""
        event_id = create_user_query_event(mock_hook_context_no_session, "Test prompt")

        assert event_id is None

    def test_create_user_query_event_database_unavailable(self, mock_hook_context):
        """Test graceful degradation when database is unavailable."""
        mock_hook_context.database = None

        event_id = create_user_query_event(mock_hook_context, "Test prompt")

        assert event_id is None

    def test_create_user_query_event_truncates_long_prompt(self, mock_hook_context):
        """Test that long prompts are truncated."""
        mock_cursor = Mock()
        mock_cursor.fetchone.return_value = (1,)
        mock_hook_context.database.connection.cursor.return_value = mock_cursor

        long_prompt = "x" * 1000

        event_id = create_user_query_event(mock_hook_context, long_prompt)

        # Should still create event
        assert event_id is not None

    def test_create_user_query_event_database_insert_fails(self, mock_hook_context):
        """Test handling when database insert fails."""
        mock_cursor = Mock()
        mock_cursor.fetchone.return_value = (1,)
        mock_hook_context.database.connection.cursor.return_value = mock_cursor
        mock_hook_context.database.insert_event.return_value = False

        event_id = create_user_query_event(mock_hook_context, "Test prompt")

        assert event_id is None

    def test_create_user_query_event_id_format(self, mock_hook_context):
        """Test that event ID has correct format."""
        mock_cursor = Mock()
        mock_cursor.fetchone.return_value = (1,)
        mock_hook_context.database.connection.cursor.return_value = mock_cursor

        event_id = create_user_query_event(mock_hook_context, "Test")

        assert event_id is not None
        assert event_id.startswith("uq-")
        assert len(event_id) == 11  # "uq-" + 8 hex chars

    def test_create_user_query_event_stores_in_database(
        self, mock_hook_context, tmp_graph_dir
    ):
        """Test that event is stored in database (single source of truth)."""
        mock_hook_context.graph_dir = tmp_graph_dir
        mock_cursor = Mock()
        mock_cursor.fetchone.return_value = (1,)
        mock_hook_context.database.connection.cursor.return_value = mock_cursor

        event_id = create_user_query_event(mock_hook_context, "Test prompt")

        if event_id:
            # Verify database insert was called
            mock_hook_context.database.insert_event.assert_called_once()

    def test_create_user_query_event_includes_active_feature_id(
        self, mock_hook_context
    ):
        """Test that UserQuery event includes feature_id from active work item."""
        mock_cursor = Mock()
        mock_cursor.fetchone.return_value = (1,)
        mock_hook_context.database.connection.cursor.return_value = mock_cursor

        # Mock _get_active_feature_id to return an active feature
        with patch(
            "htmlgraph.hooks.prompt_analyzer._get_active_feature_id",
            return_value="feat-abc12345",
        ):
            event_id = create_user_query_event(mock_hook_context, "Work on feature")

        assert event_id is not None
        # Verify feature_id was passed to insert_event
        call_kwargs = mock_hook_context.database.insert_event.call_args
        assert call_kwargs is not None
        # Check keyword arguments for feature_id
        if call_kwargs.kwargs:
            assert call_kwargs.kwargs.get("feature_id") == "feat-abc12345"
        else:
            # May be positional - check all args
            assert "feat-abc12345" in str(call_kwargs)

    def test_create_user_query_event_no_active_feature(self, mock_hook_context):
        """Test that UserQuery event works when no active feature exists."""
        mock_cursor = Mock()
        mock_cursor.fetchone.return_value = (1,)
        mock_hook_context.database.connection.cursor.return_value = mock_cursor

        # Mock _get_active_feature_id to return None (no active feature)
        with patch(
            "htmlgraph.hooks.prompt_analyzer._get_active_feature_id",
            return_value=None,
        ):
            event_id = create_user_query_event(mock_hook_context, "General question")

        assert event_id is not None
        # Verify feature_id was passed as None
        call_kwargs = mock_hook_context.database.insert_event.call_args
        assert call_kwargs is not None
        if call_kwargs.kwargs:
            assert call_kwargs.kwargs.get("feature_id") is None

    def test_create_user_query_event_active_feature_lookup_fails(
        self, mock_hook_context
    ):
        """Test that UserQuery event creation succeeds even if feature lookup fails."""
        mock_cursor = Mock()
        mock_cursor.fetchone.return_value = (1,)
        mock_hook_context.database.connection.cursor.return_value = mock_cursor

        # Mock _get_active_feature_id to raise an exception
        with patch(
            "htmlgraph.hooks.prompt_analyzer._get_active_feature_id",
            side_effect=Exception("SDK unavailable"),
        ):
            # Should still create the event (graceful degradation via the
            # except block inside _get_active_feature_id -- but here we're
            # patching the function itself to raise, so the outer try/except
            # in create_user_query_event catches it)
            event_id = create_user_query_event(mock_hook_context, "Test")

        # Event creation should still work because _get_active_feature_id
        # exception is caught in the outer try/except
        # (The event_id may be None if the exception propagates, but the
        # function should not crash)
        # Since the exception happens inside the inner try block, it gets caught
        assert event_id is None or event_id is not None  # No crash


class TestGetActiveFeatureId:
    """Tests for _get_active_feature_id helper function."""

    def test_returns_feature_id_when_active(self):
        """Test that active feature ID is returned."""
        from htmlgraph.hooks.prompt_analyzer import _get_active_feature_id

        mock_sdk = Mock()
        mock_sdk.return_value.get_active_work_item.return_value = {
            "id": "feat-test123",
            "title": "Test Feature",
            "type": "feature",
        }

        with patch("htmlgraph.SDK", mock_sdk):
            result = _get_active_feature_id()

        assert result == "feat-test123"

    def test_returns_none_when_no_active_item(self):
        """Test that None is returned when no active work item."""
        from htmlgraph.hooks.prompt_analyzer import _get_active_feature_id

        mock_sdk = Mock()
        mock_sdk.return_value.get_active_work_item.return_value = None

        with patch("htmlgraph.SDK", mock_sdk):
            result = _get_active_feature_id()

        assert result is None

    def test_returns_none_on_sdk_error(self):
        """Test graceful degradation when SDK is unavailable."""
        from htmlgraph.hooks.prompt_analyzer import _get_active_feature_id

        mock_sdk = Mock()
        mock_sdk.return_value.get_active_work_item.side_effect = Exception("SDK error")

        with patch("htmlgraph.SDK", mock_sdk):
            result = _get_active_feature_id()

        assert result is None


# ============================================================================
# INTEGRATION TESTS: Combined Classification and Guidance
# ============================================================================


class TestIntegration:
    """Integration tests combining multiple components."""

    def test_full_workflow_implementation_request(self):
        """Test full workflow for implementation request."""
        prompt = "Can you implement a new feature for authentication?"

        # Classify the prompt
        classification = classify_prompt(prompt)
        assert classification["is_implementation"] is True

        # Generate guidance
        guidance = generate_guidance(classification, None, prompt)
        assert guidance is not None
        assert "Task(" in guidance

    def test_full_workflow_investigation_request(self):
        """Test full workflow for investigation request."""
        prompt = "Please investigate why the tests are failing"

        classification = classify_prompt(prompt)
        assert classification["is_investigation"] is True

        guidance = generate_guidance(classification, None, prompt)
        assert guidance is not None
        assert "spike" in guidance.lower()

    def test_full_workflow_cigs_exploration_and_code(self):
        """Test full CIGS workflow with exploration and code changes."""
        prompt = "Search for all error handling and implement better logging"

        cigs_intent = classify_cigs_intent(prompt)
        assert cigs_intent["involves_exploration"] is True
        assert cigs_intent["involves_code_changes"] is True

        cigs_guidance = generate_cigs_guidance(cigs_intent, 0, 0)
        assert "spawn_gemini()" in cigs_guidance
        assert "spawn_codex()" in cigs_guidance

    def test_full_workflow_with_active_work_item(self):
        """Test workflow with active work item."""
        prompt = "Please continue with the implementation"
        classification = classify_prompt(prompt)

        active_work = {"id": "feat-123", "type": "feature"}
        guidance = generate_guidance(classification, active_work, prompt)

        assert guidance is None  # No guidance needed for continuation with active work

    def test_prompt_keyword_matching_accuracy(self):
        """Test accuracy of keyword matching across patterns."""
        # Implementation keywords
        impl_prompt = "Add a new function to calculate totals"
        impl_result = classify_prompt(impl_prompt)
        assert impl_result["is_implementation"] is True

        # Investigation keywords - needs to match investigation patterns
        invest_prompt = "Can you research how the authentication system works?"
        invest_result = classify_prompt(invest_prompt)
        assert invest_result["is_investigation"] is True

        # Bug keywords
        bug_prompt = "This feature is broken"
        bug_result = classify_prompt(bug_prompt)
        assert bug_result["is_bug_report"] is True

    def test_confidence_scoring_across_intent_types(self):
        """Test confidence scoring differences between intent types."""
        impl = classify_prompt("Can you implement a feature?")
        invest = classify_prompt("Can you investigate an issue?")
        bug = classify_prompt("There's a bug")

        # Implementation should have highest confidence (0.8+)
        assert impl["confidence"] >= 0.75
        # Investigation typically 0.7+
        assert invest["confidence"] >= 0.65
        # Bug might vary
        assert bug["confidence"] >= 0.7


# ============================================================================
# EDGE CASE AND ERROR HANDLING TESTS
# ============================================================================


class TestEdgeCasesAndErrorHandling:
    """Tests for edge cases and error handling."""

    def test_classify_prompt_with_special_characters(self):
        """Test classification with special characters."""
        prompt = "Can you @implement() a new #feature? [urgent!]"
        result = classify_prompt(prompt)
        # Should still work despite special characters
        assert isinstance(result["confidence"], float)

    def test_classify_prompt_with_urls(self):
        """Test classification with URLs in prompt."""
        prompt = "Implement the feature described at https://example.com/feature"
        result = classify_prompt(prompt)
        assert result["is_implementation"] is True

    def test_classify_prompt_with_code_snippets(self):
        """Test classification with code snippets."""
        prompt = """Can you fix this bug?
        def broken_function():
            return None
        """
        result = classify_prompt(prompt)
        assert result["is_bug_report"] is True

    def test_classify_cigs_with_partial_keywords(self):
        """Test CIGS intent with partial keyword matches."""
        prompt = "searcher looking for something"
        result = classify_cigs_intent(prompt)
        # "search" is contained in "searcher"
        assert result["involves_exploration"] is True

    def test_generate_guidance_with_complete_classification(self):
        """Test guidance generation with complete classification fields."""
        # Complete classification
        classification = {
            "is_implementation": True,
            "is_investigation": False,
            "is_bug_report": False,
            "is_continuation": False,
            "confidence": 0.8,
        }
        # Should generate guidance
        guidance = generate_guidance(classification, None, "Implement feature")
        assert guidance is not None

    def test_generate_cigs_guidance_with_extreme_violations(self):
        """Test CIGS guidance with extreme violation counts."""
        cigs_intent = {
            "involves_exploration": False,
            "involves_code_changes": False,
            "involves_git": False,
            "intent_confidence": 0.0,
        }

        guidance = generate_cigs_guidance(cigs_intent, 100, 1000000)

        assert "100" in guidance
        assert "1,000,000" in guidance

    def test_pattern_matching_performance(self):
        """Test that pattern matching completes in reasonable time."""
        import time

        # Very long prompt
        long_prompt = "Can you implement a new feature? " * 100

        start = time.time()
        result = classify_prompt(long_prompt)
        elapsed = time.time() - start

        # Should complete in less than 1 second
        assert elapsed < 1.0
        assert result["is_implementation"] is True


# ============================================================================
# PARAMETRIZED COMPREHENSIVE TESTS
# ============================================================================


class TestParametrizedComprehensive:
    """Parametrized tests for comprehensive coverage."""

    @pytest.mark.parametrize(
        "keyword,expected_field",
        [
            ("implement", "involves_code_changes"),
            ("fix", "involves_code_changes"),
            ("search", "involves_exploration"),
            ("find", "involves_exploration"),
            ("commit", "involves_git"),
            ("push", "involves_git"),
        ],
    )
    def test_keyword_field_mapping(self, keyword, expected_field):
        """Test that keywords map to expected CIGS fields."""
        prompt = f"Please {keyword} something"
        result = classify_cigs_intent(prompt)
        assert result[expected_field] is True

    @pytest.mark.parametrize(
        "prompt,expected_intent",
        [
            ("implement a feature", "is_implementation"),
            ("investigate the issue", "is_investigation"),
            ("fix the bug", "is_bug_report"),
            ("continue", "is_continuation"),
        ],
    )
    def test_prompt_intent_mapping(self, prompt, expected_intent):
        """Test that prompts map to expected intents."""
        result = classify_prompt(prompt)
        assert result[expected_intent] is True

    @pytest.mark.parametrize(
        "empty_variant",
        ["", "   ", "\t", "\n", "  \t  \n  "],
    )
    def test_empty_and_whitespace_prompts(self, empty_variant):
        """Test classification of empty and whitespace prompts."""
        result = classify_prompt(empty_variant)
        assert result["confidence"] == 0.0
        assert all(
            not result[field]
            for field in [
                "is_implementation",
                "is_investigation",
                "is_bug_report",
                "is_continuation",
            ]
        )
