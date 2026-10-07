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
- Typed per-column filters: `f` on a column picks an operator and a value, `F` lists and removes active filters.
- Undo the last applied sort or filter.
- Hide, reorder and rename columns without leaving the grid (`c`).
- Export the current view - filter, hidden columns, aliases and sort - to CSV or Parquet (`e`).
- Per-column statistics over the filtered view: rows, nulls, distinct, min, max, and avg and sum for numeric columns (`i`).
- Cell values with newlines or other control characters render safely on one line instead of breaking the grid.
- MIT license.

### Fixed
- Rendering, undo-granularity and version-metadata issues found in review.
- Renaming columns in the columns panel no longer drops or orphans sort and filter entries when a column is renamed onto a hidden column's display name or an already-aliased column is renamed again.
- Confirming the filters panel with a toggled-then-untoggled row no longer commits an identical spec and triggers a spurious reload; deleting the last typed filter now leaves `Filters` nil.
- Selecting an existing `is null` / `is not null` filter in the typed filter panel now applies immediately instead of dead-ending on a value prompt.
- The filter panel no longer drops the text input's cursor-blink command while typing.
