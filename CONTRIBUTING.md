# Contributing to HtmlGraph

Thank you for your interest in contributing to HtmlGraph!

## Branch Strategy

HtmlGraph uses a two-branch workflow:

### `main` - Production Branch
- **Purpose**: Stable, production-ready code
- **Protected**: Yes
- **Deploys to**: PyPI on tagged releases
- **CI/CD**: Full test suite + build + docs deployment
- **Merges**: Only from `dev` via pull request

### `dev` - Development Branch
- **Purpose**: Active development and integration
- **Protected**: Partially (require PR reviews)
- **CI/CD**: Full test suite + build + docs preview
- **Merges**: Feature branches via pull request

## Development Workflow

### 1. Create a Feature Branch

```bash
# Start from dev
git checkout dev
git pull origin dev

# Create feature branch
git checkout -b feature/your-feature-name
```

### 2. Make Your Changes

```bash
# Make changes
# Run tests locally
uv run pytest tests/

# Run linting
uv run ruff check src/python/htmlgraph/
uv run ruff format src/python/htmlgraph/

# Type checking
uv run mypy src/python/htmlgraph/
```

### 3. Commit Your Changes

```bash
git add .
git commit -m "feat: your feature description"

# Follow conventional commits:
# feat: New feature
# fix: Bug fix
# docs: Documentation changes
# test: Test changes
# chore: Maintenance tasks
```

### 4. Push and Create Pull Request

```bash
git push origin feature/your-feature-name

# Create PR to `dev` branch on GitHub
```

### 5. CI/CD Pipeline

Your PR will automatically trigger:
- ✅ Tests on Python 3.10, 3.11, 3.12
- ✅ Linting and type checking
- ✅ Package build validation
- ✅ Documentation build

### 6. Merge to Dev

Once approved and CI passes:
- PR is merged to `dev`
- `dev` CI runs again
- Docs are deployed to preview environment

### 7. Release Process (Maintainers Only)

When ready to release from `dev` to `main`:

```bash
# 1. Update version in all files
#    - pyproject.toml
#    - src/python/htmlgraph/__init__.py
#    - packages/claude-plugin/.claude-plugin/plugin.json
#    - packages/gemini-extension/gemini-extension.json

# 2. Create release notes
#    - RELEASE_NOTES_X.Y.Z.md

# 3. Commit version bump on dev
git add .
git commit -m "chore: bump version to X.Y.Z"
git push origin dev

# 4. Create PR from dev to main
# Title: "Release X.Y.Z"
# Body: Link to release notes

# 5. After merge, tag the release on main
git checkout main
git pull origin main
git tag vX.Y.Z
git push origin vX.Y.Z

# 6. GitHub Actions will automatically:
#    - Build the package
#    - Publish to PyPI
#    - Create GitHub release
#    - Deploy documentation
```

## Code Quality Standards

### Required Checks

All PRs must pass:
1. **Tests**: `uv run pytest tests/`
2. **Linting**: `uv run ruff check src/`
3. **Formatting**: `uv run ruff format src/`
4. **Type Checking**: `uv run mypy src/` (warnings ok)
5. **Package Build**: `uv build`

### Test Coverage

- Aim for >80% coverage
- All new features must have tests
- Bug fixes should include regression tests

### Code Style

- Follow PEP 8
- Use type hints for all functions
- Write docstrings for public APIs
- Keep functions small and focused

## Documentation

### Required Documentation

1. **Docstrings**: All public functions/classes
2. **README.md**: Update if adding major features
3. **AGENTS.md**: Update if changing AI agent workflows
4. **Type hints**: All function parameters and returns

### Documentation Format

```python
def example_function(param1: str, param2: int) -> bool:
    """
    Short description of what the function does.

    Args:
        param1: Description of param1
        param2: Description of param2

    Returns:
        Description of return value

    Example:
        >>> example_function("test", 42)
        True
    """
    pass
```

## Testing

### Running Tests

```bash
# All tests
uv run pytest tests/

# Specific test file
uv run pytest tests/test_sdk.py

# With coverage
uv run pytest tests/ --cov=htmlgraph --cov-report=html

# Watch mode (requires pytest-watch)
uv run ptw tests/
```

### Writing Tests

Place tests in `tests/` directory:
- `tests/test_cli.py` - CLI tests
- `tests/test_models.py` - Model tests
- `tests/test_api.py` - REST API tests

Use pytest fixtures and parametrize:

```python
import pytest
import subprocess

def test_feature_creation(tmp_path):
    result = subprocess.run(
        ["htmlgraph", "feature", "create", "Test Feature"],
        capture_output=True, text=True, cwd=tmp_path
    )
    assert result.returncode == 0
    assert "feat-" in result.stdout
```

## Common Tasks

### Local Development Setup

```bash
# Clone repository
git clone https://github.com/shakestzd/htmlgraph.git
cd htmlgraph

# Install dependencies
uv pip install -e ".[dev]"

# Run tests
uv run pytest tests/

# Start local server
uv run htmlgraph serve
```

### Running Linters

```bash
# Check code
uv run ruff check src/

# Format code
uv run ruff format src/

# Type check
uv run mypy src/
```

### Building Package Locally

```bash
# Build distributions
uv build

# Check package
uv run twine check dist/*

# Install locally
pip install -e .
```

### Updating Documentation

```bash
# Install docs dependencies
pip install mkdocs mkdocs-material mkdocstrings mkdocstrings-python

# Serve docs locally
mkdocs serve

# Build docs
mkdocs build
```

## Release Checklist

For maintainers preparing a release:

- [ ] All tests pass on `dev`
- [ ] Documentation is up to date
- [ ] Version bumped in all files
- [ ] Release notes created (RELEASE_NOTES_X.Y.Z.md)
- [ ] PR from `dev` to `main` created
- [ ] PR approved and merged
- [ ] Tag created: `git tag vX.Y.Z`
- [ ] Tag pushed: `git push origin vX.Y.Z`
- [ ] GitHub Actions successful
- [ ] Package published to PyPI
- [ ] GitHub release created
- [ ] Verify installation: `pip install htmlgraph==X.Y.Z`

## Getting Help

- **Issues**: https://github.com/shakestzd/htmlgraph/issues
- **Discussions**: https://github.com/shakestzd/htmlgraph/discussions
- **Documentation**: See `AGENTS.md` and `docs/`

## License

By contributing to HtmlGraph, you agree that your contributions will be licensed under the MIT License.

---

**Thank you for contributing to HtmlGraph!** 🎉
