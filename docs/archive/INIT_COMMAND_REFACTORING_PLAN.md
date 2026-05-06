# Init Command Refactoring Plan

## Executive Summary

The `InitCommand` in `/Users/shakes/DevProjects/htmlgraph/src/python/wipnote/cli/core.py` currently delegates to a **non-existent** `cli_legacy.py` file, causing runtime failures. This document provides a complete analysis and refactoring plan to extract and modernize the init functionality.

**Current Status:** ❌ **BROKEN** - `cli_legacy.py` has been deleted but `InitCommand.execute()` still imports it.

---

## 1. Analysis Summary: What Init Should Do

Based on analysis of existing Wipnote codebase, the `init` command should:

### Core Directory Structure Creation
- ✅ Create `.wipnote/` root directory
- ✅ Create subdirectories:
  - `features/` - Feature tracking HTML files
  - `sessions/` - Session HTML files
  - `events/` - Event JSONL files (gitignored)
  - `spikes/` - Spike investigation HTML files
  - `tracks/` - Track planning HTML files
  - `bugs/` - Bug tracking HTML files
  - `chores/` - Chore tracking HTML files
  - `archives/` - Archived items
  - `logs/errors/` - Error logs (gitignored)

### Database Initialization
- ✅ Create `wipnote.db` SQLite database with full schema
- ✅ Create `index.sqlite` analytics cache (gitignored)
- ✅ Initialize all tables:
  - `agent_events` - Tool calls and results
  - `features` - Work items
  - `sessions` - Agent sessions
  - `tracks` - Multi-feature initiatives
  - `agent_collaboration` - Handoffs
  - `graph_edges` - Relationships
  - `event_log_archive` - Historical events

### Configuration Files
- ✅ Create `.wipnote/hooks-config.json` (if `--install-hooks`)
- ✅ Create `.wipnote/agents.json` (agent registry)
- ✅ Update `.gitignore` with Wipnote patterns (unless `--no-update-gitignore`)

### Git Hooks Installation (Optional)
- ✅ Install git hooks if `--install-hooks` flag provided
- ✅ Supported hooks: `post-commit`, `post-checkout`, `post-merge`, `pre-push`
- ✅ Use symlinks from `.wipnote/hooks/` to `.git/hooks/`
- ✅ Backup existing hooks and chain them

### Validation
- ✅ Verify git repository exists (for hooks installation)
- ✅ Check directory permissions
- ✅ Validate database schema creation
- ✅ Test database connectivity

### User Feedback
- ✅ Print success message with next steps
- ✅ Show interactive wizard if `--interactive` flag
- ✅ Display errors with actionable guidance

---

## 2. Function Breakdown: Logical Components

Extract `cmd_init()` into these focused functions in `src/python/wipnote/cli/operations/initialization.py`:

### 2.1 Core Initialization Functions

```python
def create_directory_structure(
    base_dir: Path,
    include_events_keep: bool = True
) -> dict[str, Path]:
    """
    Create .wipnote directory structure.

    Args:
        base_dir: Base directory (project root)
        include_events_keep: Create .gitkeep in events/ directory

    Returns:
        Dictionary mapping collection names to created paths

    Raises:
        PermissionError: If directory creation fails
    """
    # Creates:
    # - .wipnote/
    # - .wipnote/features/
    # - .wipnote/sessions/
    # - .wipnote/events/
    # - .wipnote/spikes/
    # - .wipnote/tracks/
    # - .wipnote/bugs/
    # - .wipnote/chores/
    # - .wipnote/archives/
    # - .wipnote/logs/errors/
```

```python
def initialize_database(
    db_path: Path,
    skip_analytics_cache: bool = False
) -> tuple[bool, str]:
    """
    Initialize SQLite databases with full schema.

    Args:
        db_path: Path to wipnote.db
        skip_analytics_cache: Skip creating index.sqlite

    Returns:
        Tuple of (success, message)

    Raises:
        sqlite3.Error: If database creation fails
    """
    # Creates:
    # - wipnote.db (unified event database)
    # - index.sqlite (analytics cache, optional)
    # Initializes all tables via WipnoteDB.create_tables()
```

```python
def create_default_config_files(
    wipnote_dir: Path,
    install_hooks: bool = False
) -> dict[str, Path]:
    """
    Create default configuration files.

    Args:
        wipnote_dir: Path to .wipnote directory
        install_hooks: Whether to create hooks-config.json

    Returns:
        Dictionary mapping config file names to paths
    """
    # Creates:
    # - agents.json (empty agent registry)
    # - hooks-config.json (if install_hooks=True)
```

### 2.2 Git Integration Functions

```python
def update_gitignore(
    project_dir: Path,
    patterns: list[str] | None = None
) -> tuple[bool, str]:
    """
    Update .gitignore with Wipnote patterns.

    Args:
        project_dir: Project root directory
        patterns: Custom patterns (uses defaults if None)

    Returns:
        Tuple of (updated, message)

    Default patterns:
        - .wipnote/index.sqlite*
        - .wipnote/sessions/*.jsonl
        - .wipnote/events/*.jsonl
        - .wipnote/parent-activity.json
        - .wipnote/logs/errors/
    """
```

```python
def install_git_hooks(
    project_dir: Path,
    force: bool = False,
    dry_run: bool = False
) -> dict[str, tuple[bool, str]]:
    """
    Install git hooks for event logging.

    Uses existing HookInstaller class.

    Args:
        project_dir: Project root directory
        force: Force overwrite existing hooks
        dry_run: Show what would be done

    Returns:
        Dictionary mapping hook names to (success, message) tuples
    """
```

### 2.3 Validation Functions

```python
def validate_init_prerequisites(
    base_dir: Path,
    install_hooks: bool = False
) -> list[str]:
    """
    Validate prerequisites before initialization.

    Args:
        base_dir: Base directory to initialize
        install_hooks: Check git repository if True

    Returns:
        List of error messages (empty if valid)

    Checks:
        - Directory exists and is writable
        - Git repository exists (if install_hooks)
        - No conflicting .wipnote directory
    """
```

```python
def verify_initialization(
    wipnote_dir: Path,
    check_database: bool = True
) -> tuple[bool, list[str]]:
    """
    Verify initialization completed successfully.

    Args:
        wipnote_dir: Path to .wipnote directory
        check_database: Verify database schema

    Returns:
        Tuple of (success, list of issues)

    Verifies:
        - All directories exist
        - Database files exist
        - Database schema is correct
        - Config files are valid JSON
    """
```

### 2.4 Interactive Wizard

```python
def run_interactive_wizard(
    base_dir: Path
) -> InitConfig:
    """
    Run interactive initialization wizard.

    Args:
        base_dir: Base directory to initialize

    Returns:
        InitConfig with user selections

    Prompts for:
        - Install git hooks? (y/n)
        - Update .gitignore? (y/n)
        - Skip analytics cache? (y/n)
    """
```

### 2.5 Orchestration Function

```python
def initialize_wipnote(
    config: InitConfig,
    verbose: bool = False
) -> CommandResult:
    """
    Main orchestration function for initialization.

    Args:
        config: InitConfig with all settings
        verbose: Print detailed progress

    Returns:
        CommandResult with success status and message

    Steps:
        1. Validate prerequisites
        2. Create directory structure
        3. Initialize databases
        4. Create config files
        5. Update .gitignore (if enabled)
        6. Install git hooks (if enabled)
        7. Verify initialization
        8. Return results
    """
```

---

## 3. InitConfig Model (ALREADY EXISTS!)

The `InitConfig` model already exists in `/Users/shakes/DevProjects/htmlgraph/src/python/wipnote/cli/models.py` (lines 120-138):

```python
class InitConfig(BaseModel):
    """Configuration for wipnote init command.

    Attributes:
        dir: Directory to initialize (default: .)
        install_hooks: Install Git hooks for event logging
        interactive: Interactive setup wizard
        no_index: Do not create the analytics cache (index.sqlite)
        no_update_gitignore: Do not update/create .gitignore for cache files
        no_events_keep: Do not create .wipnote/events/.gitkeep
    """

    dir: str = Field(default=".")
    install_hooks: bool = Field(default=False)
    interactive: bool = Field(default=False)
    no_index: bool = Field(default=False)
    no_update_gitignore: bool = Field(default=False)
    no_events_keep: bool = Field(default=False)
```

**Status:** ✅ **COMPLETE** - No changes needed to the model.

---

## 4. Dependencies

### External Packages (Standard Library Only)
- `pathlib` - Directory/file operations
- `sqlite3` - Database creation
- `json` - Config file generation
- `shutil` - File copying (hooks)
- `subprocess` - Git command execution

### Internal Wipnote Modules
- `wipnote.db.schema.WipnoteDB` - Database initialization
- `wipnote.hooks.installer.HookInstaller` - Git hooks installation
- `wipnote.hooks.installer.HookConfig` - Hook configuration
- `wipnote.config.get_database_path` - Database path resolution
- `wipnote.config.get_analytics_cache_path` - Cache path resolution
- `wipnote.cli.models.InitConfig` - Configuration model (exists)
- `wipnote.cli.base.CommandResult` - Return type

### Git Hooks Source
- Hooks are distributed with the Python package
- Located in `wipnote/hooks/` directory
- Copied to `.wipnote/hooks/` during initialization
- Symlinked from `.git/hooks/` to `.wipnote/hooks/`

---

## 5. Directory Structure

### New File: `src/python/wipnote/cli/operations/initialization.py`

```
src/python/wipnote/cli/operations/
├── __init__.py           # Export public functions
└── initialization.py     # All init logic (new file, ~300 lines)
```

### Functions Exported from `operations/__init__.py`:

```python
from wipnote.cli.operations.initialization import (
    initialize_wipnote,  # Main entry point
    create_directory_structure,
    initialize_database,
    create_default_config_files,
    update_gitignore,
    install_git_hooks,
    validate_init_prerequisites,
    verify_initialization,
    run_interactive_wizard,
)

__all__ = [
    "initialize_wipnote",
    "create_directory_structure",
    "initialize_database",
    "create_default_config_files",
    "update_gitignore",
    "install_git_hooks",
    "validate_init_prerequisites",
    "verify_initialization",
    "run_interactive_wizard",
]
```

---

## 6. Error Handling

### Error Categories

1. **Permission Errors**
   - Directory creation fails
   - File write fails
   - Git hooks installation fails

2. **Validation Errors**
   - Not a git repository (when `--install-hooks` used)
   - .wipnote already exists (conflict)
   - Invalid directory path

3. **Database Errors**
   - SQLite connection fails
   - Schema creation fails
   - Database corruption

4. **Configuration Errors**
   - Invalid JSON in config files
   - Malformed .gitignore

### Error Recovery Strategies

```python
# Example: Graceful degradation for git hooks
try:
    install_git_hooks(project_dir, force=False)
except NotAGitRepositoryError:
    logger.warning("Not a git repository - skipping hook installation")
    # Continue with rest of initialization
except PermissionError as e:
    logger.error(f"Cannot install hooks: {e}")
    # Continue with rest of initialization
```

### Error Messages

All errors should provide:
- ✅ Clear description of what failed
- ✅ Explanation of why it failed
- ✅ Actionable next steps

Example:
```
❌ Failed to install git hooks: Not a git repository

This directory is not a git repository (.git directory not found).
Git hooks can only be installed in git repositories.

Next steps:
  1. Initialize git: git init
  2. Re-run: wipnote init --install-hooks

Alternatively, skip hooks:
  wipnote init
```

---

## 7. Testing Strategy

### Unit Tests (Priority 1)

Test each function in isolation:

```python
# tests/cli/operations/test_initialization.py

def test_create_directory_structure_creates_all_dirs(tmp_path):
    """Test directory structure creation."""
    dirs = create_directory_structure(tmp_path, include_events_keep=True)

    assert (tmp_path / ".wipnote").exists()
    assert (tmp_path / ".wipnote" / "features").exists()
    assert (tmp_path / ".wipnote" / "sessions").exists()
    assert (tmp_path / ".wipnote" / "events").exists()
    assert (tmp_path / ".wipnote" / "spikes").exists()
    assert len(dirs) == 9  # All expected directories

def test_initialize_database_creates_schema(tmp_path):
    """Test database initialization."""
    db_path = tmp_path / "wipnote.db"
    success, message = initialize_database(db_path, skip_analytics_cache=False)

    assert success is True
    assert db_path.exists()
    # Verify tables exist
    import sqlite3
    conn = sqlite3.connect(db_path)
    cursor = conn.cursor()
    cursor.execute("SELECT name FROM sqlite_master WHERE type='table'")
    tables = {row[0] for row in cursor.fetchall()}
    assert "agent_events" in tables
    assert "sessions" in tables
    assert "features" in tables
    conn.close()

def test_update_gitignore_appends_patterns(tmp_path):
    """Test .gitignore update."""
    gitignore = tmp_path / ".gitignore"
    gitignore.write_text("# Existing content\n*.pyc\n")

    updated, message = update_gitignore(tmp_path)

    assert updated is True
    content = gitignore.read_text()
    assert ".wipnote/index.sqlite" in content
    assert "*.pyc" in content  # Preserves existing

def test_validate_init_prerequisites_detects_conflicts(tmp_path):
    """Test prerequisite validation."""
    # Create conflicting .wipnote directory
    (tmp_path / ".wipnote").mkdir()
    (tmp_path / ".wipnote" / "existing.txt").write_text("conflict")

    errors = validate_init_prerequisites(tmp_path, install_hooks=False)

    # Should warn but not fail (idempotent)
    assert len(errors) == 0  # Initialization is idempotent

def test_verify_initialization_detects_incomplete_setup(tmp_path):
    """Test initialization verification."""
    # Create partial structure
    (tmp_path / ".wipnote").mkdir()
    (tmp_path / ".wipnote" / "features").mkdir()
    # Missing database

    success, issues = verify_initialization(tmp_path / ".wipnote")

    assert success is False
    assert any("database" in issue.lower() for issue in issues)
```

### Integration Tests (Priority 2)

Test full initialization workflow:

```python
def test_initialize_wipnote_full_workflow(tmp_path):
    """Test complete initialization."""
    config = InitConfig(
        dir=str(tmp_path),
        install_hooks=False,
        interactive=False,
        no_index=False,
        no_update_gitignore=False,
        no_events_keep=False,
    )

    result = initialize_wipnote(config, verbose=True)

    assert result.success is True
    assert (tmp_path / ".wipnote").exists()
    assert (tmp_path / ".wipnote" / "wipnote.db").exists()
    assert (tmp_path / ".wipnote" / "index.sqlite").exists()
    assert (tmp_path / ".gitignore").exists()

def test_initialize_wipnote_with_hooks(tmp_path, git_repo):
    """Test initialization with git hooks."""
    config = InitConfig(
        dir=str(git_repo),
        install_hooks=True,
        interactive=False,
        no_index=False,
        no_update_gitignore=False,
        no_events_keep=False,
    )

    result = initialize_wipnote(config, verbose=True)

    assert result.success is True
    assert (git_repo / ".git" / "hooks" / "post-commit").exists()
```

### Backward Compatibility Tests (Priority 3)

Ensure migration from legacy:

```python
def test_init_command_matches_legacy_behavior(tmp_path):
    """Verify new implementation matches old behavior."""
    config = InitConfig(dir=str(tmp_path))
    result = initialize_wipnote(config)

    # Check that all directories match legacy structure
    expected_dirs = [
        ".wipnote/features",
        ".wipnote/sessions",
        ".wipnote/events",
        ".wipnote/spikes",
        ".wipnote/tracks",
        ".wipnote/bugs",
        ".wipnote/chores",
        ".wipnote/archives",
    ]

    for dir_path in expected_dirs:
        assert (tmp_path / dir_path).exists()
```

---

## 8. Migration Checklist

### Phase 1: Create New Implementation
- [ ] Create `src/python/wipnote/cli/operations/` directory
- [ ] Create `src/python/wipnote/cli/operations/__init__.py`
- [ ] Create `src/python/wipnote/cli/operations/initialization.py`
- [ ] Implement all 9 functions listed in Section 2
- [ ] Add comprehensive docstrings with type hints
- [ ] Add logging statements for debugging

### Phase 2: Write Tests
- [ ] Create `tests/cli/operations/` directory
- [ ] Create `tests/cli/operations/test_initialization.py`
- [ ] Implement all unit tests from Section 7
- [ ] Implement integration tests
- [ ] Implement backward compatibility tests
- [ ] Achieve >90% code coverage

### Phase 3: Update InitCommand
- [ ] Modify `src/python/wipnote/cli/core.py`
- [ ] Remove import of `cli_legacy` (lines 400)
- [ ] Import `initialize_wipnote` from operations
- [ ] Update `InitCommand.execute()` to call new function
- [ ] Handle interactive mode if needed

```python
# Updated InitCommand.execute()
def execute(self) -> CommandResult:
    """Initialize the .wipnote directory."""
    from wipnote.cli.operations import initialize_wipnote

    # Run interactive wizard if requested
    if self.interactive:
        from wipnote.cli.operations import run_interactive_wizard
        config = run_interactive_wizard(Path(self.dir))
    else:
        config = InitConfig(
            dir=self.dir,
            install_hooks=self.install_hooks,
            interactive=self.interactive,
            no_index=self.no_index,
            no_update_gitignore=self.no_update_gitignore,
            no_events_keep=self.no_events_keep,
        )

    return initialize_wipnote(config, verbose=True)
```

### Phase 4: Verify & Validate
- [ ] Run all tests: `uv run pytest tests/cli/operations/`
- [ ] Run type checking: `uv run mypy src/python/wipnote/cli/operations/`
- [ ] Run linting: `uv run ruff check --fix`
- [ ] Test in real project: `uv run wipnote init`
- [ ] Test with flags: `uv run wipnote init --install-hooks`
- [ ] Test interactive mode: `uv run wipnote init --interactive`

### Phase 5: Documentation
- [ ] Update CLI help text in `core.py`
- [ ] Add docstrings to all new functions
- [ ] Update AGENTS.md with init command examples
- [ ] Add troubleshooting section for common errors

### Phase 6: Cleanup
- [ ] Remove `cli_legacy` references from codebase
- [ ] Update imports in other files (if any)
- [ ] Add deprecation notice (if needed)
- [ ] Run final test suite

---

## 9. Success Criteria

### Functional Requirements
✅ Creates all required directories
✅ Initializes databases with correct schema
✅ Updates .gitignore appropriately
✅ Installs git hooks (when requested)
✅ Provides clear user feedback
✅ Handles errors gracefully
✅ Supports interactive mode
✅ Idempotent (safe to run multiple times)

### Quality Requirements
✅ >90% test coverage
✅ Zero mypy type errors
✅ Zero ruff lint warnings
✅ Comprehensive docstrings
✅ Clear error messages
✅ Performance: <1 second for basic init
✅ Performance: <3 seconds with hooks

### Backward Compatibility
✅ Creates same directory structure as legacy
✅ Uses same database schema
✅ Generates same .gitignore patterns
✅ Installs same git hooks

---

## 10. Risk Assessment

### High Risk
- ❌ **Database schema changes** - Could break existing installations
  - **Mitigation:** Use `WipnoteDB.create_tables()` which handles migrations

### Medium Risk
- ⚠️  **Hook installation conflicts** - Overwriting user hooks
  - **Mitigation:** Backup existing hooks, chain them

- ⚠️  **.gitignore conflicts** - Malformed .gitignore files
  - **Mitigation:** Validate before append, handle parse errors

### Low Risk
- ✅ **Directory creation** - Standard mkdir operations
- ✅ **Config file generation** - Simple JSON writes

---

## 11. Timeline Estimate

### Time Breakdown
- **Phase 1 (Implementation):** 4-6 hours
  - Create directory structure: 1 hour
  - Database initialization: 1 hour
  - Git integration: 2 hours
  - Validation & verification: 1 hour
  - Interactive wizard: 1 hour

- **Phase 2 (Testing):** 3-4 hours
  - Unit tests: 2 hours
  - Integration tests: 1 hour
  - Manual testing: 1 hour

- **Phase 3 (Integration):** 1-2 hours
  - Update InitCommand: 30 minutes
  - Fix imports: 30 minutes
  - Verify CLI works: 1 hour

- **Phase 4 (Documentation):** 1-2 hours
  - Docstrings: 30 minutes
  - AGENTS.md updates: 30 minutes
  - Troubleshooting guide: 1 hour

**Total:** 9-14 hours (1-2 days of focused work)

---

## 12. Next Steps

### Immediate Actions (DO THIS NOW)
1. ✅ **Fix broken import** - Remove `cli_legacy` import from `core.py`
2. ✅ **Add temporary stub** - Make init command not crash:
   ```python
   def execute(self) -> CommandResult:
       """Initialize the .wipnote directory."""
       from wipnote.config import WipnoteConfig
       from wipnote.db.schema import WipnoteDB

       # Temporary implementation until full refactor
       config = WipnoteConfig(graph_dir=Path(self.dir) / ".wipnote")
       config.ensure_directories()

       db = WipnoteDB(str(config.graph_dir / "wipnote.db"))

       return CommandResult(
           success=True,
           text=f"Initialized .wipnote at {self.dir}"
       )
   ```

### Short-term (This Week)
1. Create `operations/initialization.py` with full implementation
2. Write comprehensive test suite
3. Update `InitCommand.execute()` to use new implementation
4. Run full test suite and verify

### Long-term (Next Sprint)
1. Add interactive wizard
2. Improve error messages with rich formatting
3. Add progress indicators for long-running operations
4. Consider adding `wipnote init --upgrade` for migrations

---

## Appendix A: File Structure Reference

### Current State (BROKEN)
```
src/python/wipnote/cli/
├── core.py              # InitCommand.execute() imports cli_legacy ❌
├── models.py            # InitConfig exists ✅
└── base.py             # CommandResult base class ✅
```

### Target State (WORKING)
```
src/python/wipnote/cli/
├── core.py              # InitCommand.execute() calls initialize_wipnote() ✅
├── models.py            # InitConfig (no changes) ✅
├── base.py             # CommandResult (no changes) ✅
└── operations/
    ├── __init__.py      # Export public functions ✅
    └── initialization.py # All init logic (~300 lines) ✅
```

### Dependencies Flow
```
InitCommand.execute()
    ↓
initialize_wipnote(config)
    ↓
├── validate_init_prerequisites()
├── create_directory_structure()
├── initialize_database()
│   └── WipnoteDB.create_tables()
├── create_default_config_files()
├── update_gitignore()
├── install_git_hooks()
│   └── HookInstaller.install_all_hooks()
└── verify_initialization()
```

---

## Appendix B: Default .gitignore Patterns

```gitignore
# Wipnote analytics index (rebuildable cache)
.wipnote/index.sqlite
.wipnote/index.sqlite-wal
.wipnote/index.sqlite-shm

# Wipnote session tracking artifacts (regenerable observability data)
.wipnote/sessions/*.jsonl
.wipnote/events/*.jsonl
.wipnote/parent-activity.json
.wipnote/logs/errors/

# Keep these (source of truth):
# - .wipnote/features/
# - .wipnote/bugs/
# - .wipnote/chores/
# - .wipnote/spikes/
# - .wipnote/agents.json
```

---

## Appendix C: Example User Output

### Successful Initialization
```bash
$ wipnote init --install-hooks

🚀 Initializing Wipnote...

✅ Created directory structure
   - .wipnote/features
   - .wipnote/sessions
   - .wipnote/events
   - .wipnote/spikes
   - .wipnote/tracks
   - .wipnote/bugs
   - .wipnote/chores
   - .wipnote/archives

✅ Initialized databases
   - wipnote.db (unified event database)
   - index.sqlite (analytics cache)

✅ Updated .gitignore
   - Added 8 Wipnote patterns

✅ Installed git hooks
   - post-commit → .wipnote/hooks/post-commit.sh
   - post-checkout → .wipnote/hooks/post-checkout.sh
   - post-merge → .wipnote/hooks/post-merge.sh
   - pre-push → .wipnote/hooks/pre-push.sh

🎉 Wipnote initialized successfully!

Next steps:
  1. Start tracking: git commit (hooks will track automatically)
  2. View dashboard: wipnote serve
  3. Check status: wipnote status

Documentation: https://github.com/shakestzd/wipnote
```

### Error Handling Example
```bash
$ wipnote init --install-hooks

🚀 Initializing Wipnote...

✅ Created directory structure
✅ Initialized databases

❌ Failed to install git hooks: Not a git repository

This directory is not a git repository (.git directory not found).
Git hooks can only be installed in git repositories.

Next steps:
  1. Initialize git: git init
  2. Re-run: wipnote init --install-hooks

Alternatively, skip hooks:
  wipnote init

⚠️  Wipnote initialized (without hooks)
```

---

## Document Version

- **Version:** 1.0
- **Created:** 2026-01-11
- **Status:** Ready for implementation
- **Estimated Effort:** 9-14 hours (1-2 days)
