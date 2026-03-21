"""
Tests for scripts/pre-commit-systematic-check.py

Covers:
- extract_replacements(): diff-based rename detection
- extract_rename_pairs_from_message(): commit-message-based detection
- detect_systematic_keywords(): keyword signal detection
- find_remaining_occurrences(): search result parsing
- main() integration: empty diff produces no output, multiple replacements detected
"""

import importlib.util
from pathlib import Path
from unittest.mock import MagicMock, patch

# ---------------------------------------------------------------------------
# Import the script as a module (it lives in scripts/, not a package)
# ---------------------------------------------------------------------------

_SCRIPT_PATH = (
    Path(__file__).parent.parent.parent / "scripts" / "pre-commit-systematic-check.py"
)


def _load_module():
    spec = importlib.util.spec_from_file_location(
        "pre_commit_systematic_check", _SCRIPT_PATH
    )
    assert spec is not None, f"Could not load spec from {_SCRIPT_PATH}"
    assert spec.loader is not None
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)  # type: ignore[union-attr]
    return mod


_mod = _load_module()

extract_replacements = _mod.extract_replacements
extract_rename_pairs_from_message = _mod.extract_rename_pairs_from_message
detect_systematic_keywords = _mod.detect_systematic_keywords
find_remaining_occurrences = _mod.find_remaining_occurrences
main = _mod.main


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _make_diff(old_line: str, new_line: str, filename: str = "src/foo.py") -> str:
    """Build a minimal unified diff replacing one line."""
    return (
        f"diff --git a/{filename} b/{filename}\n"
        f"--- a/{filename}\n"
        f"+++ b/{filename}\n"
        f"@@ -1,1 +1,1 @@\n"
        f"-{old_line}\n"
        f"+{new_line}\n"
    )


# ---------------------------------------------------------------------------
# extract_replacements — diff-based detection
# ---------------------------------------------------------------------------


class TestExtractReplacements:
    def test_simple_rename_detected(self):
        diff = _make_diff(
            "def get_old_handler(request):",
            "def get_new_handler(request):",
        )
        pairs = extract_replacements(diff)
        assert ("get_old_handler", "get_new_handler") in pairs

    def test_empty_diff_returns_empty(self):
        assert extract_replacements("") == []

    def test_no_change_lines_returns_empty(self):
        diff = (
            "diff --git a/src/foo.py b/src/foo.py\n"
            "--- a/src/foo.py\n"
            "+++ b/src/foo.py\n"
            "@@ -1,1 +1,1 @@\n"
            " unchanged_function()\n"
        )
        assert extract_replacements(diff) == []

    def test_multiple_renames_in_one_diff(self):
        diff = (
            "diff --git a/src/foo.py b/src/foo.py\n"
            "--- a/src/foo.py\n"
            "+++ b/src/foo.py\n"
            "@@ -1,2 +1,2 @@\n"
            "-old_service = OldService()\n"
            "+new_service = NewService()\n"
        )
        pairs = extract_replacements(diff)
        assert ("old_service", "new_service") in pairs
        assert ("OldService", "NewService") in pairs

    def test_short_tokens_ignored(self):
        # Tokens shorter than _MIN_IDENT_LEN (4) should be skipped
        diff = _make_diff("foo = bar()", "baz = qux()")
        pairs = extract_replacements(diff)
        # "foo", "bar", "baz", "qux" are all 3 chars — none should appear
        assert pairs == []

    def test_deduplicated_pairs(self):
        # Same rename appearing in two hunks should yield one pair
        diff = (
            "diff --git a/src/foo.py b/src/foo.py\n"
            "--- a/src/foo.py\n"
            "+++ b/src/foo.py\n"
            "@@ -1,1 +1,1 @@\n"
            "-old_handler()\n"
            "+new_handler()\n"
            "@@ -10,1 +10,1 @@\n"
            "-old_handler()\n"
            "+new_handler()\n"
        )
        pairs = extract_replacements(diff)
        assert pairs.count(("old_handler", "new_handler")) == 1

    def test_only_added_lines_no_false_positive(self):
        # Pure additions (no corresponding removals) should yield nothing
        diff = (
            "diff --git a/src/new_file.py b/src/new_file.py\n"
            "--- /dev/null\n"
            "+++ b/src/new_file.py\n"
            "@@ -0,0 +1,3 @@\n"
            "+def brand_new_function():\n"
            "+    pass\n"
        )
        pairs = extract_replacements(diff)
        assert pairs == []

    def test_class_rename_detected(self):
        diff = _make_diff(
            "class OldRepository(BaseRepo):",
            "class NewRepository(BaseRepo):",
        )
        pairs = extract_replacements(diff)
        assert ("OldRepository", "NewRepository") in pairs

    def test_variable_rename_in_import(self):
        diff = _make_diff(
            "from htmlgraph.sdk import old_client as client",
            "from htmlgraph.sdk import new_client as client",
        )
        pairs = extract_replacements(diff)
        assert ("old_client", "new_client") in pairs

    def test_multiple_files_in_diff(self):
        diff = (
            "diff --git a/src/a.py b/src/a.py\n"
            "--- a/src/a.py\n"
            "+++ b/src/a.py\n"
            "@@ -1,1 +1,1 @@\n"
            "-alpha_service()\n"
            "+beta_service()\n"
            "diff --git a/src/b.py b/src/b.py\n"
            "--- a/src/b.py\n"
            "+++ b/src/b.py\n"
            "@@ -1,1 +1,1 @@\n"
            "-gamma_helper()\n"
            "+delta_helper()\n"
        )
        pairs = extract_replacements(diff)
        assert ("alpha_service", "beta_service") in pairs
        assert ("gamma_helper", "delta_helper") in pairs


# ---------------------------------------------------------------------------
# extract_rename_pairs_from_message — commit message parsing
# ---------------------------------------------------------------------------


class TestExtractRenameFromMessage:
    def test_rename_verb_pattern(self):
        pairs = extract_rename_pairs_from_message("rename old_handler to new_handler")
        assert ("old_handler", "new_handler") in pairs

    def test_replace_with_pattern(self):
        pairs = extract_rename_pairs_from_message("replace OldService with NewService")
        assert ("OldService", "NewService") in pairs

    def test_sed_style_pattern(self):
        pairs = extract_rename_pairs_from_message("refactor: s/old_client/new_client/g")
        assert ("old_client", "new_client") in pairs

    def test_arrow_pattern(self):
        pairs = extract_rename_pairs_from_message("old_model -> new_model migration")
        assert ("old_model", "new_model") in pairs

    def test_empty_message_returns_empty(self):
        assert extract_rename_pairs_from_message("") == []

    def test_no_rename_pattern_returns_empty(self):
        assert extract_rename_pairs_from_message("fix: update docstring typo") == []

    def test_short_names_filtered(self):
        # "foo" and "bar" are 3 chars, below MIN_NAME_LEN=4
        pairs = extract_rename_pairs_from_message("rename foo to bar")
        assert pairs == []

    def test_deduplicated_output(self):
        msg = "rename old_thing to new_thing; old_thing -> new_thing"
        pairs = extract_rename_pairs_from_message(msg)
        assert pairs.count(("old_thing", "new_thing")) == 1


# ---------------------------------------------------------------------------
# detect_systematic_keywords
# ---------------------------------------------------------------------------


class TestDetectSystematicKeywords:
    def test_rename_keyword(self):
        assert detect_systematic_keywords("rename foo_bar to baz_qux") is True

    def test_replace_keyword(self):
        assert detect_systematic_keywords("replace all usages") is True

    def test_migrate_keyword(self):
        assert detect_systematic_keywords("migrate database schema") is True

    def test_refactor_keyword(self):
        assert detect_systematic_keywords("refactor user service") is True

    def test_sed_keyword(self):
        assert detect_systematic_keywords("s/old/new/g in all files") is True

    def test_no_keyword(self):
        assert detect_systematic_keywords("fix typo in readme") is False

    def test_case_insensitive(self):
        assert detect_systematic_keywords("RENAME OldThing TO NewThing") is True


# ---------------------------------------------------------------------------
# find_remaining_occurrences — result parsing
# ---------------------------------------------------------------------------


class TestFindRemainingOccurrences:
    def test_returns_list_of_tuples(self):
        mock_output = (
            "src/foo.py:10:    old_handler()\nsrc/bar.py:20:    old_handler = None\n"
        )
        with patch("subprocess.run") as mock_run:
            mock_result = MagicMock()
            mock_result.stdout = mock_output
            mock_run.return_value = mock_result
            hits = find_remaining_occurrences("old_handler", set())
        assert isinstance(hits, list)
        for item in hits:
            assert len(item) == 3
            filepath, lineno, content = item
            assert isinstance(filepath, str)
            assert isinstance(lineno, int)
            assert isinstance(content, str)

    def test_excludes_staged_files(self):
        mock_output = (
            "src/foo.py:10:    old_handler()\nsrc/bar.py:20:    old_handler = None\n"
        )
        with patch("subprocess.run") as mock_run:
            mock_result = MagicMock()
            mock_result.stdout = mock_output
            mock_run.return_value = mock_result
            hits = find_remaining_occurrences("old_handler", {"src/foo.py"})
        paths = [h[0] for h in hits]
        assert "src/foo.py" not in paths
        assert "src/bar.py" in paths

    def test_empty_search_output(self):
        with patch("subprocess.run") as mock_run:
            mock_result = MagicMock()
            mock_result.stdout = ""
            mock_run.return_value = mock_result
            hits = find_remaining_occurrences("nonexistent_symbol", set())
        assert hits == []

    def test_max_10_results(self):
        lines = "\n".join(f"src/file{i}.py:1:    old_name()" for i in range(20))
        with patch("subprocess.run") as mock_run:
            mock_result = MagicMock()
            mock_result.stdout = lines + "\n"
            mock_run.return_value = mock_result
            hits = find_remaining_occurrences("old_name", set())
        assert len(hits) <= 10


# ---------------------------------------------------------------------------
# main() integration tests
# ---------------------------------------------------------------------------


class TestMainIntegration:
    def test_empty_diff_exits_zero_no_output(self, capsys):
        with (
            patch.object(_mod, "get_staged_diff", return_value=""),
            patch.object(_mod, "get_commit_message", return_value=""),
            patch.object(_mod, "get_staged_files", return_value=set()),
        ):
            rc = main([])
        assert rc == 0
        captured = capsys.readouterr()
        assert captured.out == ""

    def test_rename_in_diff_with_no_remaining_exits_zero_no_warning(self, capsys):
        diff = _make_diff("old_service()", "new_service()")
        with (
            patch.object(_mod, "get_staged_diff", return_value=diff),
            patch.object(_mod, "get_commit_message", return_value=""),
            patch.object(_mod, "get_staged_files", return_value=set()),
            patch.object(_mod, "find_remaining_occurrences", return_value=[]),
        ):
            rc = main([])
        assert rc == 0
        captured = capsys.readouterr()
        assert "Warning" not in captured.out

    def test_rename_in_diff_with_remaining_prints_warning(self, capsys):
        diff = _make_diff("old_service()", "new_service()")
        hits = [("src/other.py", 5, "    old_service()")]
        with (
            patch.object(_mod, "get_staged_diff", return_value=diff),
            patch.object(_mod, "get_commit_message", return_value=""),
            patch.object(_mod, "get_staged_files", return_value=set()),
            patch.object(_mod, "find_remaining_occurrences", return_value=hits),
        ):
            rc = main([])
        assert rc == 0
        captured = capsys.readouterr()
        assert "Warning" in captured.out
        assert "old_service" in captured.out

    def test_multiple_replacements_all_warned(self, capsys):
        diff = (
            "--- a/src/foo.py\n+++ b/src/foo.py\n"
            "@@ -1,2 +1,2 @@\n"
            "-alpha_service()\n+beta_service()\n"
            "@@ -5,1 +5,1 @@\n"
            "-gamma_helper()\n+delta_helper()\n"
        )
        hits_alpha = [("src/bar.py", 3, "    alpha_service()")]
        hits_gamma = [("src/baz.py", 7, "    gamma_helper()")]

        def fake_find(old_name, exclude):
            if old_name == "alpha_service":
                return hits_alpha
            if old_name == "gamma_helper":
                return hits_gamma
            return []

        with (
            patch.object(_mod, "get_staged_diff", return_value=diff),
            patch.object(_mod, "get_commit_message", return_value=""),
            patch.object(_mod, "get_staged_files", return_value=set()),
            patch.object(_mod, "find_remaining_occurrences", side_effect=fake_find),
        ):
            rc = main([])
        assert rc == 0
        captured = capsys.readouterr()
        assert "alpha_service" in captured.out
        assert "gamma_helper" in captured.out

    def test_message_flag_overrides_commit_editmsg(self, capsys):
        with (
            patch.object(_mod, "get_staged_diff", return_value=""),
            patch.object(_mod, "get_staged_files", return_value=set()),
            patch.object(_mod, "find_remaining_occurrences", return_value=[]),
        ):
            rc = main(["--message", "rename old_thing to new_thing"])
        assert rc == 0

    def test_always_exits_zero_even_with_warnings(self, capsys):
        diff = _make_diff("old_handler()", "new_handler()")
        hits = [("src/x.py", 1, "old_handler()")]
        with (
            patch.object(_mod, "get_staged_diff", return_value=diff),
            patch.object(_mod, "get_commit_message", return_value=""),
            patch.object(_mod, "get_staged_files", return_value=set()),
            patch.object(_mod, "find_remaining_occurrences", return_value=hits),
        ):
            rc = main([])
        assert rc == 0
