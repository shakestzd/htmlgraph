---
id: task-1
priority: high
status: completed
dependencies: []
labels:
  - parallel-execution
  - auto-created
  - priority-high
  - feat-130780b2
---

# Deployment Script Generalization

## 🎯 Objective

Package HtmlGraph's deployment automation (`deploy-all.sh`) as a reusable pattern for all Python projects. Provide both shell-based interface (primary) and optional Python entry points via Invoke for CI/CD automation.

## 🛠️ Implementation Approach

**Dual Interface Strategy (Shell + Python):**
- Keep `deploy-all.sh` as primary interface (portable, familiar to users)
- Add **Invoke** tasks as optional Python-native alternative
- Package both via `pyproject.toml` entry points
- Create template for users to adapt to their projects

**Libraries:**
- `invoke>=2.2` - Task automation framework (optional dev dependency)
- `tomllib` (Python 3.11+) - Built-in TOML parser for metadata updates

**Pattern to follow:**
- **File:** `scripts/deploy-all.sh:1-50`
- **Description:** Current 7-step workflow (git push → build → publish → install → update plugins). Generalize by:
  1. Extracting project-specific vars to config section
  2. Making PyPI/plugin steps optional (flags: `--skip-pypi`, `--skip-plugins`)
  3. Adding template generation: `htmlgraph init-deploy` creates deploy script for user's project

## 📁 Files to Touch

**Modify:**
- `scripts/deploy-all.sh`
  - Add config section at top (project-specific variables)
  - Add usage documentation header
  - Improve error handling (fail fast on errors)

- `pyproject.toml`
  - Add `[project.scripts]` entry points:
    - `htmlgraph-deploy = "htmlgraph.scripts.deploy:main"`
  - Add `[project.optional-dependencies]` dev group:
    - `invoke>=2.2`

**Create:**
- `scripts/tasks.py` - Invoke task equivalents (deploy, build, publish)
- `src/python/htmlgraph/scripts/deploy.py` - Python entry point wrapping shell script
- `scripts/templates/deploy-template.sh` - User-customizable template
- `scripts/README.md` - Documentation for deployment automation
- `tests/python/test_deploy.py` - Unit tests for deploy script logic

## 🧪 Tests Required

**Unit:**
- [ ] Test version extraction from pyproject.toml
- [ ] Test dry-run mode (no actual publish)
- [ ] Test flag parsing (`--docs-only`, `--build-only`, `--skip-pypi`)
- [ ] Test error handling (invalid version, missing credentials)
- [ ] Test template generation (`htmlgraph init-deploy`)

**Integration:**
- [ ] Test full deploy workflow on clean virtualenv
- [ ] Test `invoke deploy --version=0.8.0` equivalence to shell script
- [ ] Verify packaged entry point works: `htmlgraph-deploy 0.8.0`

## ✅ Acceptance Criteria

- [ ] All tests pass (`uv run pytest tests/python/test_deploy.py`)
- [ ] Deploy script works on fresh clone (no hardcoded paths)
- [ ] Invoke tasks provide identical functionality to shell script
- [ ] Template generates customizable deploy script for other projects
- [ ] Documentation added to `scripts/README.md`
- [ ] Entry points registered in `pyproject.toml`
- [ ] No breaking changes to existing `./scripts/deploy-all.sh` usage

## ⚠️ Potential Conflicts

**Files:**
- `pyproject.toml` - Task 2 might add optional dependencies
  - **Mitigation:** Task 1 uses `[project.scripts]` section, Task 2 uses `[project.optional-dependencies]`. No overlap.

## 📝 Notes

**Design Decision:** Shell script remains primary interface (don't force users into Python workflows). Invoke tasks are opt-in enhancement for developers who prefer programmatic control.

**Distribution Strategy:**
```bash
# Default (shell script)
./scripts/deploy-all.sh 0.8.0

# Python package (after install)
htmlgraph-deploy 0.8.0
# or
invoke deploy --version=0.8.0
```

**Future Enhancement:** Add `htmlgraph init-deploy --template=pypi` to generate deployment scripts for different ecosystems (npm, cargo, etc.).

---

**Worktree:** `worktree/task-1-deploy`
**Branch:** `feature/task-1`

🤖 Auto-created via Contextune parallel execution
