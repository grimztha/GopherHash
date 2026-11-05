# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.0] - 2026-11-05

### Added
- Unified frameless window title bar integrating application brand identity, version indicator, and native Windows 11 style controls (minimize, maximize/restore, close).
- Double-click on draggable title bar region to toggle window maximize/restore.
- Dynamic maximize/restore button icon and accessibility label synchronization with window state.

### Changed
- Moved the Dark/Light theme toggle button to the secondary navigation menu bar, aligned cleanly to the right.
- Redesigned the navigation tab bar into a modern segmented control with glowing active indicators.
- Removed redundant native Windows title bar and stacked app header in favor of a sleek, modern desktop aesthetic.

## [1.0.0] - 2026-09-07

### Added
- High-performance cryptographic file hashing supporting SHA-256, BLAKE3, MD5, SHA-1, SHA-512, and CRC32.
- Two-file side-by-side checksum comparison with visual diff highlighting.
- Batch checksum verification for manifest files (`.sha256`, `.md5`, `.sfv`, etc.).
- Single-file checksum verification against expected hash values.
- Dark and Light theme support with system default integration.
- Cross-platform packaging support for Windows (Portable & NSIS installer) and macOS (Universal app & DMG).
