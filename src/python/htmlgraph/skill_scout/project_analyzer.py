"""
Project Auditor — detect languages, frameworks, and structural signals.

Reads manifest files (pyproject.toml, package.json, mix.exs, go.mod, Cargo.toml,
etc.) from a project directory to produce a structured summary of the tech stack.
"""

from __future__ import annotations

import json
import logging
from dataclasses import dataclass, field
from pathlib import Path

logger = logging.getLogger(__name__)

# Manifest filenames that indicate a language/ecosystem
_MANIFEST_LANGS: dict[str, str] = {
    "pyproject.toml": "python",
    "setup.py": "python",
    "setup.cfg": "python",
    "requirements.txt": "python",
    "package.json": "javascript",
    "yarn.lock": "javascript",
    "pnpm-lock.yaml": "javascript",
    "mix.exs": "elixir",
    "go.mod": "go",
    "Cargo.toml": "rust",
    "pom.xml": "java",
    "build.gradle": "java",
    "build.gradle.kts": "java",
    "Gemfile": "ruby",
    "composer.json": "php",
    "pubspec.yaml": "dart",
    "CMakeLists.txt": "cpp",
}

# Framework signals: (manifest, key_in_deps_or_file) -> framework name
_FRAMEWORK_SIGNALS: list[tuple[str, str, str]] = [
    ("pyproject.toml", "fastapi", "FastAPI"),
    ("pyproject.toml", "flask", "Flask"),
    ("pyproject.toml", "django", "Django"),
    ("pyproject.toml", "phoenix", "Phoenix"),
    ("pyproject.toml", "pytest", "pytest"),
    ("package.json", "react", "React"),
    ("package.json", "vue", "Vue"),
    ("package.json", "next", "Next.js"),
    ("package.json", "svelte", "Svelte"),
    ("package.json", "typescript", "TypeScript"),
    ("mix.exs", "phoenix", "Phoenix"),
    ("mix.exs", "ecto", "Ecto"),
]


@dataclass
class ProjectAnalysis:
    """Structured summary of a project's tech stack."""

    root: Path
    languages: list[str] = field(default_factory=list)
    frameworks: list[str] = field(default_factory=list)
    has_tests: bool = False
    has_ci: bool = False
    has_docker: bool = False
    has_htmlgraph: bool = False
    manifest_files: list[str] = field(default_factory=list)

    def primary_language(self) -> str | None:
        """Return the most prominent language, or None if none detected."""
        return self.languages[0] if self.languages else None


class ProjectAnalyzer:
    """Detect tech stack signals from a project directory."""

    def __init__(self, root: str | Path) -> None:
        self.root = Path(root).resolve()

    def analyze(self) -> ProjectAnalysis:
        """Scan the project root and return a ProjectAnalysis."""
        analysis = ProjectAnalysis(root=self.root)

        self._detect_languages(analysis)
        self._detect_frameworks(analysis)
        self._detect_structural_signals(analysis)

        logger.debug(
            "ProjectAnalyzer: root=%s languages=%s frameworks=%s",
            self.root,
            analysis.languages,
            analysis.frameworks,
        )
        return analysis

    # ------------------------------------------------------------------
    # Private helpers
    # ------------------------------------------------------------------

    def _detect_languages(self, analysis: ProjectAnalysis) -> None:
        seen: set[str] = set()
        for manifest, lang in _MANIFEST_LANGS.items():
            if (self.root / manifest).exists():
                analysis.manifest_files.append(manifest)
                if lang not in seen:
                    analysis.languages.append(lang)
                    seen.add(lang)

    def _detect_frameworks(self, analysis: ProjectAnalysis) -> None:
        seen: set[str] = set()
        for manifest, keyword, framework in _FRAMEWORK_SIGNALS:
            manifest_path = self.root / manifest
            if not manifest_path.exists():
                continue
            try:
                content = manifest_path.read_text(encoding="utf-8", errors="replace")
            except OSError:
                continue
            if keyword.lower() in content.lower() and framework not in seen:
                analysis.frameworks.append(framework)
                seen.add(framework)

    def _detect_structural_signals(self, analysis: ProjectAnalysis) -> None:
        root = self.root

        # Tests
        analysis.has_tests = any(
            (root / d).is_dir() for d in ("tests", "test", "spec", "__tests__")
        )

        # CI
        ci_paths = [
            root / ".github" / "workflows",
            root / ".gitlab-ci.yml",
            root / ".circleci",
            root / "Jenkinsfile",
        ]
        analysis.has_ci = any(p.exists() for p in ci_paths)

        # Docker
        analysis.has_docker = any(
            (root / f).exists()
            for f in ("Dockerfile", "docker-compose.yml", "docker-compose.yaml")
        )

        # HtmlGraph
        analysis.has_htmlgraph = (root / ".htmlgraph").is_dir()

        # package.json dependency scan for JS frameworks not yet detected
        self._scan_package_json(analysis)

    def _scan_package_json(self, analysis: ProjectAnalysis) -> None:
        pkg_path = self.root / "package.json"
        if not pkg_path.exists():
            return
        try:
            data = json.loads(pkg_path.read_text(encoding="utf-8"))
        except (OSError, json.JSONDecodeError):
            return
        deps: set[str] = set()
        for key in ("dependencies", "devDependencies", "peerDependencies"):
            deps.update(data.get(key, {}).keys())
        mapping = {
            "typescript": "TypeScript",
            "react": "React",
            "vue": "Vue",
            "@angular/core": "Angular",
            "svelte": "Svelte",
            "next": "Next.js",
            "nuxt": "Nuxt.js",
        }
        for dep, framework in mapping.items():
            if dep in deps and framework not in analysis.frameworks:
                analysis.frameworks.append(framework)
