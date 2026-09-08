# Changelog

All notable changes to this project are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project uses [Semantic Versioning](https://semver.org/).

## [Unreleased]

No tags exist yet. Everything below is unreleased history.

### Added
- Open a CSV, Parquet or NDJSON file and browse it as a scrollable, virtualized grid.
- Sort by any column, ascending or descending.
- Filter rows with a live, debounced text search across all visible columns.
- Undo the last applied sort or filter.
- Cell values with newlines or other control characters render safely on one line instead of breaking the grid.
- MIT license.

### Fixed
- Rendering, undo-granularity and version-metadata issues found in review.
