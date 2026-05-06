# Changelog

> **Note:** This changelog covers early releases. For recent changes (v0.34.x), see the [GitHub Releases](https://github.com/shakestzd/wipnote/releases) page.

All notable changes to Wipnote will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.4] - 2024-12-22

### Added

- **Hash-Based ID System**: Collision-resistant IDs for multi-agent collaboration
  - Short, readable format: `{prefix}-{hash}` (e.g., `feat-a1b2c3d4`)
  - Content-addressable with entropy for collision resistance
  - Hierarchical sub-task support (e.g., `feat-a1b2c3d4.1.2`)
  - Type-specific prefixes for visual identification
  - Inspired by [Beads](https://github.com/steveyegge/beads)
- **New `wipnote.ids` module**: `generate_id()`, `parse_id()`, `is_valid_id()`, etc.
- **ID Generation API documentation**: Complete guide with examples

### Changed

- SDK `FeatureBuilder` now generates hash-based IDs
- CLI `feature create` now generates hash-based IDs
- Server API creates hash-based IDs for new nodes
- Session IDs now use hash-based format

### Fixed

- mkdocstrings configuration updated for newer versions (inventories option)

## [0.3.0] - 2024-12-22

### Added

- **TrackBuilder Fluent API**: Deterministic track creation with specs and plans
- **Multi-Agent Collaboration**: Agent assignment, handoff notes, feature claiming
- **Comprehensive Documentation**: Complete docs site with MkDocs Material theme
- **Agent Integration Docs**: TrackBuilder documentation in Claude, Codex, and Gemini skills
- **Time Estimation Parsing**: Automatic parsing of `(Xh)` time estimates in task descriptions
- **Track Validation**: Pydantic validation for all track components

### Changed

- Updated all agent integration documentation with TrackBuilder examples
- Improved session start hook with track creation quick reference
- Enhanced feature creation decision framework

### Fixed

- Track ID persistence in HTML files
- Drift detection for new modules and directories

## [0.2.2] - 2024-12-20

### Added

- **Drift Detection**: Automatic detection of activity misalignment with features
- **Session Validation**: Validate attribution with `session validate-attribution`
- **Enhanced Activity Logging**: Richer activity log with agent attribution

### Changed

- Improved session end summaries with next steps
- Better feature completion tracking
- Enhanced dashboard with session history view

### Fixed

- Session continuity across conversation compacts
- Activity attribution edge cases
- Dashboard refresh issues

## [0.2.0] - 2024-12-18

### Added

- **Tracks**: Multi-feature project support with specs and plans
- **Spec Model**: Requirements and acceptance criteria
- **Plan Model**: Phased implementation planning
- **Track HTML Files**: index.html, spec.html, plan.html per track
- **Feature-Track Linking**: `track_id` field on features

### Changed

- Enhanced SDK with tracks API
- Updated CLI with track commands
- Improved dashboard with track views

### Fixed

- Feature status transitions
- Session event ordering

## [0.1.3] - 2024-12-16

### Added

- **Session Management**: Automatic session tracking via hooks
- **Activity Logging**: Comprehensive activity tracking in sessions
- **Dashboard**: Interactive Kanban board and graph views

### Changed

- Improved feature query performance
- Enhanced HTML file structure
- Better CSS styling for nodes

### Fixed

- Feature step completion tracking
- Edge relationship types
- SQLite index synchronization

## [0.1.2] - 2024-12-15

### Added

- **CLI Commands**: `wipnote` command-line interface
- **Dashboard Server**: `wipnote serve` for local development
- **Feature Relationships**: Typed edges (blocks, related, etc.)

### Changed

- Simplified SDK initialization
- Improved error messages
- Better type hints

### Fixed

- Feature file path generation
- CSS selector query edge cases

## [0.1.1] - 2024-12-14

### Added

- **Pydantic Models**: Type-safe Feature, Track, Session models
- **HTML Conversion**: Automatic HTML ↔ Pydantic conversion
- **CSS Selector Queries**: Query features with CSS selectors

### Changed

- Refactored graph operations
- Improved HTML file format
- Enhanced documentation

### Fixed

- HTML parsing edge cases
- Property serialization

## [0.1.0] - 2024-12-13

### Added

- Initial release
- **Core SDK**: Feature creation and management
- **HTML Files**: Features as HTML files on disk
- **Basic Queries**: Filter features by status and priority
- **Git Integration**: Text-based storage for version control
- **Minimal Dependencies**: Pure Python with justhtml

[0.3.0]: https://github.com/shakestzd/wipnote/compare/v0.2.2...v0.3.0
[0.2.2]: https://github.com/shakestzd/wipnote/compare/v0.2.0...v0.2.2
[0.2.0]: https://github.com/shakestzd/wipnote/compare/v0.1.3...v0.2.0
[0.1.3]: https://github.com/shakestzd/wipnote/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/shakestzd/wipnote/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/shakestzd/wipnote/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/shakestzd/wipnote/releases/tag/v0.1.0
