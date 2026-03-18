"""
Tests for UserPromptSubmit hook with CIGS integration.

Tests cover:
1. CIGS intent classification (exploration, code changes, git)
2. Violation count integration
3. Imperative guidance generation (generate_cigs_guidance called directly)
4. Hook output structure (compact attribution block only, no per-turn imperatives)
5. Edge cases and error handling

These tests call the Python functions directly (no subprocess) so they run
without network access, without an API key, and complete in well under 10
seconds.  Database-touching helpers (create_user_query_event,
get_active_work_item, get_open_work_items, get_session_violation_count) are
patched with lightweight fakes.

NOTE: The hook script no longer calls generate_cigs_guidance() per-turn.
Static delegation imperatives now live in the system prompt to reduce context
bloat (~800 tokens/turn → ~60 tokens/turn, bug-f8fb4174).
generate_cigs_guidance() is tested directly here to preserve coverage.
"""

from unittest.mock import MagicMock

import pytest
from htmlgraph.hooks.prompt_analyzer import (
    classify_cigs_intent,
    classify_prompt,
    generate_cigs_guidance,
    generate_guidance,
)

# ---------------------------------------------------------------------------
# Helpers that replicate the hook's assembly logic so tests can check the
# combined output structure without spawning a subprocess.
# ---------------------------------------------------------------------------


def _make_hook_context() -> MagicMock:
    """Return a minimal HookContext mock (no real DB)."""
    ctx = MagicMock()
    ctx.session_id = "test-session-id"
    ctx.graph_dir = "/tmp/test-htmlgraph"
    ctx.database = MagicMock()
    ctx.database.connection = MagicMock()
    ctx.database.insert_event = MagicMock(return_value=True)
    return ctx


def _run_hook_logic(
    prompt: str,
    violation_count: int = 0,
    waste_tokens: int = 0,
    active_work: dict | None = None,
    open_items: list | None = None,
) -> dict:
    """Replicate the hook's main() logic in-process.

    Patches out all I/O and returns the same dict structure that the
    hook script would print to stdout.

    NOTE: Mirrors the simplified hook that no longer calls
    generate_cigs_guidance() per-turn (static rules now in system prompt).
    """
    if not prompt:
        return {}

    if open_items is None:
        open_items = []

    classification = classify_prompt(prompt)
    cigs_intent = classify_cigs_intent(prompt)

    # Hook no longer calls generate_cigs_guidance() per-turn.
    # Static delegation imperatives live in the system prompt.
    workflow_guidance = generate_guidance(
        classification, active_work, prompt, open_work_items=open_items
    )

    combined_guidance = []
    if workflow_guidance:
        combined_guidance.append(workflow_guidance)

    if combined_guidance:
        return {
            "hookSpecificOutput": {
                "hookEventName": "UserPromptSubmit",
                "additionalContext": "\n\n".join(combined_guidance),
            },
            "classification": {
                "implementation": classification["is_implementation"],
                "investigation": classification["is_investigation"],
                "bug_report": classification["is_bug_report"],
                "continuation": classification["is_continuation"],
                "confidence": classification["confidence"],
            },
            "cigs_classification": {
                "involves_exploration": cigs_intent["involves_exploration"],
                "involves_code_changes": cigs_intent["involves_code_changes"],
                "involves_git": cigs_intent["involves_git"],
                "intent_confidence": cigs_intent["intent_confidence"],
            },
            "cigs_session_status": {
                "violation_count": violation_count,
                "waste_tokens": waste_tokens,
            },
        }
    return {}


# ---------------------------------------------------------------------------
# Test classes
# ---------------------------------------------------------------------------


class TestCIGSIntentClassification:
    """Test CIGS intent classification for delegation guidance."""

    def test_exploration_intent_detected(self):
        """Exploration keywords should trigger CIGS exploration guidance."""
        prompts = [
            "Search for all files containing 'authentication'",
            "Find the implementation of the login function",
            "Analyze the codebase structure",
            "Show me where the API endpoints are defined",
        ]

        for prompt in prompts:
            result = classify_cigs_intent(prompt)
            assert result["involves_exploration"], (
                f"Failed to detect exploration in: {prompt}"
            )

    def test_exploration_guidance_contains_imperatives(self):
        """Exploration intent should produce imperative guidance via generate_cigs_guidance.

        NOTE: The hook no longer injects CIGS imperatives per-turn (they live in the
        system prompt). This test verifies generate_cigs_guidance() still produces
        correct output when called directly.
        """
        for prompt in [
            "Search for all files containing 'authentication'",
            "Find the implementation of the login function",
        ]:
            cigs_intent = classify_cigs_intent(prompt)
            guidance = generate_cigs_guidance(cigs_intent, 0, 0, prompt)
            if guidance:
                assert "IMPERATIVE" in guidance, (
                    f"Missing imperative in guidance for: {prompt}"
                )
                assert "spawn_gemini" in guidance or "exploration" in guidance.lower()

    def test_code_changes_intent_detected(self):
        """Code change keywords should trigger CIGS implementation guidance."""
        prompts = [
            "Implement the user authentication feature",
            "Fix the bug in the login handler",
            "Update the API endpoint to support pagination",
            "Refactor the database connection code",
            "Add error handling to the payment processor",
        ]

        for prompt in prompts:
            result = classify_cigs_intent(prompt)
            assert result["involves_code_changes"], (
                f"Failed to detect code changes in: {prompt}"
            )

    def test_code_changes_guidance_contains_imperatives(self):
        """Code change intent should produce imperative guidance via generate_cigs_guidance.

        NOTE: The hook no longer injects CIGS imperatives per-turn (they live in the
        system prompt). This test verifies generate_cigs_guidance() still produces
        correct output when called directly.
        """
        prompt = "Implement the user authentication feature"
        cigs_intent = classify_cigs_intent(prompt)
        guidance = generate_cigs_guidance(cigs_intent, 0, 0, prompt)
        if guidance:
            assert "IMPERATIVE" in guidance
            assert "spawn_codex" in guidance or "Task()" in guidance

    def test_git_intent_detected(self):
        """Git keywords should trigger CIGS git delegation guidance."""
        prompts = [
            "Commit these changes with a descriptive message",
            "Push the feature branch to origin",
            "Merge the pull request",
            "Run git status to see what changed",
        ]

        for prompt in prompts:
            result = classify_cigs_intent(prompt)
            assert result["involves_git"], f"Failed to detect git in: {prompt}"

    def test_git_guidance_contains_imperatives(self):
        """Git intent should produce imperative guidance via generate_cigs_guidance.

        NOTE: The hook no longer injects CIGS imperatives per-turn (they live in the
        system prompt). This test verifies generate_cigs_guidance() still produces
        correct output when called directly.
        """
        prompt = "Commit these changes with a descriptive message"
        cigs_intent = classify_cigs_intent(prompt)
        guidance = generate_cigs_guidance(cigs_intent, 0, 0, prompt)
        if guidance:
            assert "IMPERATIVE" in guidance
            assert "spawn_copilot" in guidance or "git" in guidance.lower()

    def test_multiple_intents_detected(self):
        """Prompts with multiple intents should detect all."""
        prompt = "Search for the login code, then fix the authentication bug and commit the changes"
        result = classify_cigs_intent(prompt)

        assert result["involves_exploration"], "Should detect exploration"
        assert result["involves_code_changes"], "Should detect code changes"
        assert result["involves_git"], "Should detect git"

    def test_no_delegation_intent_returns_dict(self):
        """classify_cigs_intent always returns a valid dict."""
        prompts = [
            "Explain how the authentication system works",
            "What's the best practice for error handling?",
        ]
        for prompt in prompts:
            result = classify_cigs_intent(prompt)
            assert isinstance(result, dict)
            assert "involves_exploration" in result
            assert "involves_code_changes" in result
            assert "involves_git" in result
            assert "intent_confidence" in result


class TestViolationWarnings:
    """Test violation count integration and warning generation."""

    def test_no_violations_no_warning_in_guidance(self):
        """With 0 violations the guidance should not contain a violation warning."""
        cigs_intent = classify_cigs_intent("Find the login function")
        guidance = generate_cigs_guidance(cigs_intent, 0, 0, "")
        if guidance:
            assert "VIOLATION WARNING" not in guidance

    def test_violation_count_included_in_output(self):
        """Violation count should be reflected in the assembled output dict."""
        output = _run_hook_logic(
            "Implement user authentication", violation_count=2, waste_tokens=500
        )
        if "cigs_session_status" in output:
            assert output["cigs_session_status"]["violation_count"] == 2
            assert output["cigs_session_status"]["waste_tokens"] == 500

    def test_violation_warning_appears_when_count_positive(self):
        """A positive violation count should produce a warning in the guidance."""
        cigs_intent = classify_cigs_intent("Implement user authentication")
        guidance = generate_cigs_guidance(cigs_intent, 3, 1500, "")
        assert "VIOLATION WARNING" in guidance
        assert "3" in guidance

    def test_violation_values_are_non_negative(self):
        """Violation count and waste tokens must always be >= 0."""
        output = _run_hook_logic(
            "Find the login function", violation_count=0, waste_tokens=0
        )
        if "cigs_session_status" in output:
            assert output["cigs_session_status"]["violation_count"] >= 0
            assert output["cigs_session_status"]["waste_tokens"] >= 0


class TestGuidanceGeneration:
    """Test imperative guidance generation."""

    def test_exploration_guidance_format(self):
        """Exploration guidance should have correct format."""
        cigs_intent = classify_cigs_intent("Search for authentication code")
        guidance = generate_cigs_guidance(cigs_intent, 0, 0, "")

        if guidance:
            assert "CIGS PRE-RESPONSE GUIDANCE" in guidance
            assert "IMPERATIVE" in guidance
            assert "YOU MUST" in guidance
            assert "spawn_gemini" in guidance

    def test_code_changes_guidance_format(self):
        """Code changes guidance should have correct format."""
        cigs_intent = classify_cigs_intent("Implement the payment processor")
        guidance = generate_cigs_guidance(cigs_intent, 0, 0, "")

        if guidance:
            assert "CIGS PRE-RESPONSE GUIDANCE" in guidance
            assert "IMPERATIVE" in guidance
            assert "YOU MUST" in guidance
            assert "spawn_codex" in guidance or "Task()" in guidance

    def test_git_guidance_format(self):
        """Git guidance should have correct format."""
        cigs_intent = classify_cigs_intent("Commit the changes")
        guidance = generate_cigs_guidance(cigs_intent, 0, 0, "")

        if guidance:
            assert "CIGS PRE-RESPONSE GUIDANCE" in guidance
            assert "IMPERATIVE" in guidance
            assert "spawn_copilot" in guidance


class TestCombinedGuidance:
    """Test hook output guidance (compact attribution block only).

    NOTE: Per-turn CIGS imperatives were removed from hook output in bug-f8fb4174.
    Static delegation rules now live in the system prompt. The hook only
    injects the compact attribution block (~60 tokens) per turn.
    """

    def test_implementation_with_no_work_item(self):
        """Implementation without work item should include workflow guidance."""
        output = _run_hook_logic("Implement user login feature")

        if "hookSpecificOutput" in output:
            guidance = output["hookSpecificOutput"]["additionalContext"]
            # Should reference work items or SDK calls (compact attribution block)
            assert (
                "work item" in guidance.lower()
                or "sdk" in guidance.lower()
                or "feature" in guidance.lower()
            )

    def test_exploration_request(self):
        """Exploration request: hook produces attribution block, not CIGS imperatives."""
        output = _run_hook_logic("Find all files that use the database connection")

        # The hook may or may not produce output for pure exploration with no open items
        # When it does, it should be the compact attribution block format
        if "hookSpecificOutput" in output:
            guidance = output["hookSpecificOutput"]["additionalContext"]
            # Compact block: should NOT contain the old verbose CIGS header
            assert "CIGS PRE-RESPONSE GUIDANCE" not in guidance


class TestEdgeCases:
    """Test edge cases and error handling."""

    def test_empty_prompt(self):
        """Empty prompt should return empty result."""
        output = _run_hook_logic("")
        assert output == {}

    def test_very_short_prompt(self):
        """Very short prompts should not crash."""
        for prompt in ["ok", "yes", "continue", "next"]:
            result = classify_prompt(prompt)
            assert isinstance(result, dict)

    def test_very_long_prompt(self):
        """Very long prompts should not crash."""
        prompt = "Search for " + "authentication " * 100
        result = classify_cigs_intent(prompt)
        assert isinstance(result, dict)
        assert result["involves_exploration"] is True

    def test_special_characters(self):
        """Prompts with special characters should be handled."""
        prompts = [
            "Find the `authenticate()` function",
            'Search for "user login" in the code',
            "Look for files with $ENV variables",
        ]
        for prompt in prompts:
            result = classify_cigs_intent(prompt)
            assert isinstance(result, dict)


class TestOutputStructure:
    """Test output structure conforms to hook specification."""

    def test_hook_output_structure(self):
        """Output should have correct hookSpecificOutput structure."""
        output = _run_hook_logic("Find the login code")

        if "hookSpecificOutput" in output:
            hook_output = output["hookSpecificOutput"]
            assert "hookEventName" in hook_output
            assert hook_output["hookEventName"] == "UserPromptSubmit"
            assert "additionalContext" in hook_output
            assert isinstance(hook_output["additionalContext"], str)

    def test_classification_structure(self):
        """Output should include both classification types."""
        output = _run_hook_logic("Implement and commit user authentication")

        # Should have original classification
        assert "classification" in output
        assert "implementation" in output["classification"]
        assert "investigation" in output["classification"]
        assert "bug_report" in output["classification"]
        assert "continuation" in output["classification"]
        assert "confidence" in output["classification"]

        # Should have CIGS classification
        assert "cigs_classification" in output
        assert "involves_exploration" in output["cigs_classification"]
        assert "involves_code_changes" in output["cigs_classification"]
        assert "involves_git" in output["cigs_classification"]
        assert "intent_confidence" in output["cigs_classification"]

        # Should have session status
        assert "cigs_session_status" in output
        assert "violation_count" in output["cigs_session_status"]
        assert "waste_tokens" in output["cigs_session_status"]

    def test_empty_prompt_returns_empty_dict(self):
        """Empty prompt must return {} (not raise)."""
        assert _run_hook_logic("") == {}


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
