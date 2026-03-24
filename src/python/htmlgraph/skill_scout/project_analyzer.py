"""Project Auditor: detect tech stack and structural signals from manifest files."""

from __future__ import annotations

import json
import sys
from pathlib import Path

from pydantic import BaseModel

if sys.version_info >= (3, 11):
    import tomllib
else:
    import tomli as tomllib  # type: ignore[no-redef]

# Manifest file → primary language
_MANIFEST_LANGUAGES: dict[str, str] = {
    "pyproject.toml": "python",
    "setup.py": "python",
    "setup.cfg": "python",
    "requirements.txt": "python",
    "package.json": "javascript",
    "mix.exs": "elixir",
    "Cargo.toml": "rust",
    "go.mod": "go",
    "Gemfile": "ruby",
    "pom.xml": "java",
    "build.gradle": "java",
    "composer.json": "php",
}

# Python packages → framework names
_PYTHON_FRAMEWORKS: dict[str, str] = {
    "fastapi": "fastapi",
    "django": "django",
    "flask": "flask",
    "pytest": "pytest",
    "sqlalchemy": "sqlalchemy",
    "pydantic": "pydantic",
}

# JS packages → framework names
_JS_FRAMEWORKS: dict[str, str] = {
    "react": "react",
    "vue": "vue",
    "next": "nextjs",
    "typescript": "typescript",
    "@types/react": "react",
}

# Elixir deps → framework names
_ELIXIR_FRAMEWORKS: dict[str, str] = {
    "phoenix": "phoenix",
    "ecto": "ecto",
}

# File extensions to count
_COUNT_EXTENSIONS = {".py", ".js", ".ts", ".ex", ".exs", ".rs", ".go"}


class ProjectSignals(BaseModel):
    languages: list[str]
    frameworks: list[str]
    has_tests: bool
    has_ci: bool
    has_docker: bool
    manifest_files: list[str]
    existing_plugins: list[str]
    file_counts: dict[str, int]


def analyze_project(root: Path) -> ProjectSignals:
    """Detect a project's tech stack and structural signals."""
    languages: set[str] = set()
    frameworks: set[str] = set()
    manifest_files: list[str] = []

    for manifest, lang in _MANIFEST_LANGUAGES.items():
        if (root / manifest).exists():
            languages.add(lang)
            manifest_files.append(manifest)

    frameworks.update(_detect_python_frameworks(root))
    frameworks.update(_detect_js_frameworks(root))
    frameworks.update(_detect_elixir_frameworks(root))

    return ProjectSignals(
        languages=sorted(languages),
        frameworks=sorted(frameworks),
        has_tests=_has_tests(root),
        has_ci=_has_ci(root),
        has_docker=_has_docker(root),
        manifest_files=sorted(manifest_files),
        existing_plugins=_detect_plugins(root),
        file_counts=_count_files(root),
    )


def _detect_python_frameworks(root: Path) -> list[str]:
    pyproject = root / "pyproject.toml"
    if not pyproject.exists():
        return []
    try:
        with open(pyproject, "rb") as f:
            data = tomllib.load(f)
    except Exception:
        return []
    deps = data.get("project", {}).get("dependencies", [])
    found = []
    for dep in deps:
        name = dep.split("[")[0].split(">=")[0].split("==")[0].strip().lower()
        if name in _PYTHON_FRAMEWORKS:
            found.append(_PYTHON_FRAMEWORKS[name])
    return found


def _detect_js_frameworks(root: Path) -> list[str]:
    pkg = root / "package.json"
    if not pkg.exists():
        return []
    try:
        data = json.loads(pkg.read_text())
    except Exception:
        return []
    all_deps = {**data.get("dependencies", {}), **data.get("devDependencies", {})}
    return [_JS_FRAMEWORKS[k] for k in all_deps if k in _JS_FRAMEWORKS]


def _detect_elixir_frameworks(root: Path) -> list[str]:
    mix = root / "mix.exs"
    if not mix.exists():
        return []
    content = mix.read_text()
    return [v for k, v in _ELIXIR_FRAMEWORKS.items() if f":{k}" in content]


def _has_tests(root: Path) -> bool:
    return any((root / d).is_dir() for d in ("tests", "test", "spec", "__tests__"))


def _has_ci(root: Path) -> bool:
    return (root / ".github" / "workflows").is_dir() or (
        root / ".gitlab-ci.yml"
    ).exists()


def _has_docker(root: Path) -> bool:
    return (root / "Dockerfile").exists() or (root / "docker-compose.yml").exists()


def _detect_plugins(root: Path) -> list[str]:
    settings = root / ".claude" / "settings.json"
    if not settings.exists():
        return []
    try:
        data = json.loads(settings.read_text())
        plugins = data.get("enabledPlugins", [])
        return [p for p in plugins if isinstance(p, str)]
    except Exception:
        return []


def _count_files(root: Path) -> dict[str, int]:
    counts: dict[str, int] = {ext: 0 for ext in _COUNT_EXTENSIONS}
    try:
        for path in root.rglob("*"):
            if path.suffix in counts and path.is_file():
                counts[path.suffix] += 1
    except Exception:
        pass
    return {ext: count for ext, count in counts.items() if count > 0}
