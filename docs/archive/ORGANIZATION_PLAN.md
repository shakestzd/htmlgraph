# HtmlGraph Project Organization Plan

**Date:** 2026-01-12
**Status:** Ready for execution

---

## Executive Summary

**Current State:**
- 45 Python files in root (should be 0-2)
- 162 Markdown files in root (should be ~7-10)
- Most files are misplaced - tests, demos, analysis scripts, and documentation reports

**Target State:**
- Clean root directory with only essential user-facing files
- All tests in tests/
- All utilities in scripts/
- All documentation in docs/ with clear structure
- Comprehensive reference documentation

---

## Part 1: Python Files Organization

### Current Inventory

**Root Python Files:** 45 total

1. **Test Files (22)** - Move to `tests/integration/`
   - test_codex_spawner.py
   - test_complexity_assessment.py
   - test_copilot_spawner_tracking.py
   - test_delegation_events.py
   - test_discoverability.py
   - test_edge_index_bug.py
   - test_features.py
   - test_gemini_spawner_diagnosis.py
   - test_new_features.py
   - test_orchestration_tab.py
   - test_orchestration_with_playwright.py
   - test_orchestrator_pattern.py
   - test_phase1_userquery.py
   - test_phase4_delegation_events.py
   - test_real_workitem_integration.py
   - test_redesigned_dashboard.py
   - test_spawner_live_events.py
   - test_spawner_routing_live.py
   - test_task_delegation.py
   - test_user_query_events.py
   - test_websocket_realtime.py
   - test_websocket_streaming.py

2. **Demo/Example Files (4)** - Move to `examples/`
   - demo_agent_planning.py
   - demo_real_project_analytics.py
   - demo_sdk_operations.py
   - example_query_compilation.py

3. **Analysis/Utility Files (7)** - Move to `scripts/`
   - analyze_features.py
   - analyze_orchestrator_impact_v2.py
   - analyze_orchestrator_impact.py
   - generate_real_events.py
   - verify_htmx_dashboard.py
   - verify_new_dashboard.py
   - verify_spawner_tracking.py

4. **Setup/Cleanup Files (6)** - Move to `scripts/`
   - cleanup_wip.py
   - create_delegation_test_features.py
   - create_integrity_spike.py
   - create_spike_report.py
   - create_spike.py
   - setup_features.py

5. **Other Files (6)** - Needs individual assessment
   - delegation_analysis.py → scripts/
   - link_features_to_track.py → scripts/
   - record_orchestration_verification.py → scripts/
   - start_api_server.py → scripts/ (dev utility)
   - test-hook-input.py → tests/manual/
   - update_phase2_feature.py → scripts/

### Action Plan for Python Files

**Move Operations:**
```bash
# Create directories
mkdir -p tests/integration
mkdir -p tests/manual
mkdir -p docs/reports
mkdir -p docs/analysis
mkdir -p docs/investigations

# Move test files
git mv test_*.py tests/integration/

# Move demo files
git mv demo_*.py examples/
git mv example_query_compilation.py examples/

# Move analysis/utility files
git mv analyze_*.py scripts/
git mv verify_*.py scripts/
git mv generate_real_events.py scripts/

# Move setup/cleanup files
git mv cleanup_wip.py scripts/
git mv create_*.py scripts/
git mv setup_features.py scripts/

# Move other files
git mv delegation_analysis.py scripts/
git mv link_features_to_track.py scripts/
git mv record_orchestration_verification.py scripts/
git mv start_api_server.py scripts/
git mv test-hook-input.py tests/manual/
git mv update_phase2_feature.py scripts/
```

---

## Part 2: Markdown Files Organization

### Current Inventory

**Root Markdown Files:** 162 total

**Categories:**

1. **Essential (Keep in Root - 7 files)**
   - README.md
   - CONTRIBUTING.md
   - CHANGELOG.md
   - CLAUDE.md
   - AGENTS.md
   - GEMINI.md
   - PRD.md

2. **Implementation Reports (~14 files)** - Move to `docs/implementation/`
   - *_IMPLEMENTATION*.md files
   - IMPLEMENTATION_*.md files

3. **Test/Verification Reports (~16 files)** - Move to `docs/verification/`
   - *_TEST*.md files
   - *_VERIFICATION*.md files
   - TEST_*.md files
   - VERIFICATION_*.md files

4. **Bug/Investigation Reports (~22 files)** - Move to `docs/investigations/`
   - *_BUG*.md files
   - BUG_*.md files
   - *_INVESTIGATION*.md files
   - INVESTIGATION_*.md files

5. **Phase/Release Documentation (~13 files)** - Move to `docs/phases/` and `docs/releases/`
   - PHASE*.md → docs/phases/
   - RELEASE*.md → docs/releases/

6. **Analysis/Research (~14 files)** - Move to `docs/analysis/`
   - *_ANALYSIS*.md files
   - ANALYSIS_*.md files
   - *_RESEARCH*.md files
   - RESEARCH_*.md files

7. **Architecture/Design Docs** - Move to `docs/architecture/`
   - ARCHITECTURE*.md
   - DESIGN*.md
   - *_ARCHITECTURE*.md

8. **Quick Start/Guides** - Move to `docs/guides/`
   - *_QUICK_START*.md
   - QUICK_START_*.md
   - *_GUIDE*.md

9. **Index/Summary Docs** - Move to `docs/`
   - *_INDEX*.md
   - INDEX_*.md
   - *_SUMMARY*.md
   - SUMMARY_*.md

10. **Deprecated/Old Docs** - Move to `docs/deprecated/`
    - Files not referenced anywhere
    - Old/superseded documentation

### Action Plan for Markdown Files

**Create Directory Structure:**
```bash
mkdir -p docs/implementation
mkdir -p docs/verification
mkdir -p docs/investigations
mkdir -p docs/phases
mkdir -p docs/releases
mkdir -p docs/analysis
mkdir -p docs/architecture
mkdir -p docs/guides
mkdir -p docs/deprecated
```

**Move Operations:** (See execution section for detailed commands)

---

## Part 3: Scripts Directory Cleanup

**Current scripts/ contents:** 14 Python files + 1 Markdown + shell scripts

**Action:**
1. Create scripts/README.md documenting each utility
2. Group by purpose:
   - Migration utilities
   - Analysis tools
   - Build/deploy tools
   - Development utilities

---

## Part 4: Documentation Structure

**New docs/ Structure:**

```
docs/
├── INDEX.md                    # Master table of contents
├── architecture/               # System design documents
├── guides/                     # User guides and quick starts
├── implementation/             # Implementation reports
├── verification/               # Test and verification reports
├── investigations/             # Bug investigations and analysis
├── analysis/                   # Project analysis documents
├── phases/                     # Development phase documentation
├── releases/                   # Release notes and changelogs
└── deprecated/                 # Archived old documentation
```

---

## Part 5: Quality Assurance

**After Organization:**

1. **Verify all imports still work**
   ```bash
   uv run pytest
   uv run mypy src/
   ```

2. **Check all links in documentation**
   - Search for relative links
   - Update paths where needed

3. **Update .gitignore if needed**
   - Ensure test artifacts are ignored

4. **Create reference documents**
   - PYTHON_SCRIPTS_REFERENCE.md
   - DOCUMENTATION_STRUCTURE.md

---

## Success Criteria

- [ ] Root directory has ≤10 files (essentials only)
- [ ] All tests in tests/ directory
- [ ] All utilities in scripts/ with README
- [ ] All docs in docs/ with clear structure
- [ ] docs/INDEX.md provides complete navigation
- [ ] All tests pass after reorganization
- [ ] All links work correctly
- [ ] Reference documentation created

---

## Execution Order

1. Create all necessary directories
2. Move Python files (tests, examples, scripts)
3. Move Markdown files by category
4. Create scripts/README.md
5. Create docs/INDEX.md
6. Create PYTHON_SCRIPTS_REFERENCE.md
7. Create DOCUMENTATION_STRUCTURE.md
8. Run tests and verify
9. Fix any broken links
10. Final quality check
