# Changelog

All notable changes to this project are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project uses [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.3.0] - 2026-11-03

### Added
- Several value fields in one pivot: after picking a value, `add value` in the pivot builder adds another aggregate and value field, and `esc` on that step removes the last one. On the command line, `values=` takes a comma-separated list of `field:agg` items such as `values=amount:sum,amount:avg`; `*:count` counts rows. A bare field uses `agg=`, so `values=amount,agg=sum` still works.
- Totals: `t` on a pivot view toggles a grand total row, labeled `Total` and always last, and, when there is a column field, a total column per value. Totals are real aggregates over all rows, computed by DuckDB, and are included in exports. On the command line, `totals=true`.
- Percent: `%` on a pivot view cycles values through percent of row, percent of column, percent of grand total, and back to plain values. On the command line, `percent=row|col|total`. Values are rounded to 2 decimals. Percent needs numeric values (count, or sum, avg, min or max of a numeric field); otherwise the pivot is refused with a message.
- Export to NDJSON: `e` in the grid and `-o` with `--pivot` accept `.ndjson` besides `.csv` and `.parquet`, for normal views and pivots.
- The pivot view's status bar lists its keys, `t` and `%` included.

## [0.2.0] - 2026-11-03

### Added
- Pivot tables: `p` opens a pivot builder. Pick a row field, an optional column field, an aggregate (count, sum, avg, min, max) and a value field. DuckDB computes the result, which opens as a grid you can scroll, sort and export like the main view; `u` goes back.
- `--pivot rows=...,cols=...,values=...,agg=...` prints a pivot table as CSV without opening the grid; `-o` writes it to a `.csv` or `.parquet` file instead.
- A column field may have at most 50 distinct values; past that the pivot is refused with a clear message.
- Typed per-column filters: `f` on a column picks an operator and a value, `F` lists and removes active filters.
- Hide, reorder and rename columns without leaving the grid (`c`).
- Export the current view - filter, hidden columns, aliases and sort - to CSV or Parquet (`e`).
- Per-column statistics over the filtered view: rows, nulls, distinct, min, max, and avg and sum for numeric columns (`i`).

### Fixed
- Renaming columns in the columns panel no longer drops or orphans sort and filter entries when a column is renamed onto a hidden column's display name or an already-aliased column is renamed again.
- Confirming the filters panel with a toggled-then-untoggled row no longer commits an identical spec and triggers a spurious reload; deleting the last typed filter now leaves `Filters` nil.
- Selecting an existing `is null` / `is not null` filter in the typed filter panel now applies immediately instead of dead-ending on a value prompt.
- The filter panel no longer drops the text input's cursor-blink command while typing.

## [0.1.0] - 2026-09-08

### Added
- Open a CSV, Parquet or NDJSON file and browse it as a scrollable, virtualized grid.
- Sort by any column, ascending or descending.
- Filter rows with a live, debounced text search across all visible columns.
- Undo the last applied sort or filter.
- Cell values with newlines or other control characters render safely on one line instead of breaking the grid.
- MIT license.

### Fixed
- Rendering, undo-granularity and version-metadata issues found in review.
