<p align="center">
  <img src="docs/logo.png" alt="PivotLens logo" width="180">
</p>

# PivotLens

A command-line viewer for CSV, Parquet and NDJSON files. Browse, sort and
filter millions of rows in the terminal, no SQL required.

[![ci](https://github.com/sv222/pivotlens/actions/workflows/ci.yml/badge.svg)](https://github.com/sv222/pivotlens/actions/workflows/ci.yml)
[![release](https://img.shields.io/github/v/release/sv222/pivotlens)](https://github.com/sv222/pivotlens/releases)
[![license](https://img.shields.io/github/license/sv222/pivotlens)](LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/sv222/pivotlens.svg)](https://pkg.go.dev/github.com/sv222/pivotlens)

## What it is

pivotlens is an open source CSV, Parquet and NDJSON viewer for the terminal:
an interactive grid backed by an embedded DuckDB. It is for anyone who needs
to check a data file - a data engineer, an analyst, a backend developer
looking at a log export - without opening a spreadsheet or writing a script.
Point it at a file and you get a scrollable grid with sort, filter and
pivot tables over millions of rows, no SQL required.

It is early, but opening a file, scrolling, sorting, filtering, column
management, pivot tables and export all work today.

## The problem

A CSV that is a few hundred MB opens slowly in a spreadsheet, if it opens at
all, and a multi-gigabyte one will not open. Excel caps a worksheet at
1,048,576 rows; past that, extra rows are silently dropped. A Parquet or
NDJSON file cannot be opened in a spreadsheet at all without converting it
first. The usual fallback is a disposable Python script or a raw `duckdb`
SQL shell, just to see what is in the file.

## Install

```bash
go install github.com/sv222/pivotlens@latest
```

Requires Go and, because `pivotlens` embeds DuckDB through cgo, a C compiler
on `PATH`. To skip both, download a prebuilt binary from the
[Releases page](https://github.com/sv222/pivotlens/releases) instead: every
tagged release ships Linux and macOS builds for amd64 and arm64, a Windows
amd64 build, and a `checksums.txt` covering all of them.

Build from source:

```bash
git clone https://github.com/sv222/pivotlens.git
cd pivotlens
go build .
```

On Windows, `go-duckdb`'s prebuilt library needs a specific GCC version:
GCC 16.x on MSYS2 defaults to native TLS, which fails to link against
DuckDB's emulated-TLS objects. Use GCC 15.x or older, and point `CC` at
your own install of it, for example an MSYS2 `ucrt64` toolchain:

```powershell
$env:CC = "C:/msys64/ucrt64/bin/gcc.exe"
$env:CGO_ENABLED = "1"
go build .
```

Linux and macOS build with the system's default C compiler.

## Quick start

```
$ pivotlens testdata/tiny.csv
```

```
pivotlens │ testdata/tiny.csv │ 5 rows

region   │ status    │ amount │ created_at
US-EAST  │ COMPLETED │ 1420.5 │ 2026-09-01 14:02:11
EU-CENT  │ FAILED    │ 84     │ 2026-09-01 14:02:15
AP-SOUTH │ COMPLETED │ 312.25 │ 2026-09-01 14:03:02
US-WEST  │ PENDING   │ 9120   │ 2026-09-01 14:03:22
EU-CENT  │ COMPLETED │ 12.5   │ 2026-09-01 14:03:45

row 1  │  [/] filter  [s] sort  [u] undo  [q] quit
```

Press `/` and type to filter, `s` to sort the current column, `u` to undo
the last sort or filter, `q` to quit.

```
$ pivotlens --version
pivotlens dev

$ pivotlens
usage: pivotlens <file.csv|file.parquet|file.ndjson>
```

## Usage and flags

| Flag | Effect |
|---|---|
| `-v`, `--version` | print the version and exit |
| `--pivot spec` | print a pivot table as CSV and exit, see [Pivot tables](#pivot-tables) |
| `-o`, `--output file` | with `--pivot`, write to a `.csv`, `.parquet` or `.ndjson` file instead of stdout |

| Key | Effect |
|---|---|
| `j` / `down`, `k` / `up` | move the cursor one row |
| `ctrl+d`, `ctrl+u` | move the cursor one page down or up |
| `g`, `G` | jump to the first or last row |
| `h` / `left`, `l` / `right` | scroll one column left or right |
| `/` | open the filter, `enter` applies it, `esc` cancels |
| `f` | open a typed filter on the current column: pick an operator, type a value, `enter` applies |
| `F` | list active filters, `d` marks one for deletion, `enter` applies, `esc` cancels |
| `s` | open sort on the current column, `a` ascending, `d` descending, `x` clears, `esc` cancels |
| `u` | undo the last applied sort or filter |
| `c` | manage columns: `space` hides, `J`/`K` reorder, `r` renames, `enter` applies, `esc` discards |
| `e` | export the current view to a `.csv`, `.parquet` or `.ndjson` file |
| `i` | show statistics for the current column over the filtered view |
| `p` | build a pivot table: pick rows, columns, aggregates and values with `j`/`k` and `enter`, `add value` adds another value, `esc` steps back |
| `t` | on a pivot view, toggle the grand total row and total columns |
| `%` | on a pivot view, cycle values as percent of row, of column, of grand total, then plain |
| `q`, `ctrl+c` | quit |

## Pivot tables

Press `p` in the grid and pick, one step at a time:

1. a row field, such as `region`
2. a column field, such as `status`, or `(none)` for one value per row
3. an aggregate: `count`, `sum`, `avg`, `min` or `max`
4. a value field, such as `amount` (`count` also accepts `(none)` to count rows)
5. `done`, or `add value` to repeat steps 3 and 4 for another value; `esc`
   on this step removes the last value

The pivot opens as a new grid. Scroll, sort (`s`) and export (`e`) work as
in the main view; `u` goes back to the data. Active filters apply to the
pivot, so filter first to pivot a subset. With a column field and several
values, columns are named `<column value> <label>`, such as
`COMPLETED sum(amount)` and `COMPLETED count`.

`t` toggles totals: a `Total` row that always stays last, and, when there
is a column field, a total column per value. Totals are aggregates over all
rows, so an average total is the average of every row. `%` cycles the
values through percent of row, percent of column, percent of grand total,
and back to plain values, rounded to 2 decimals. Percent works with
`count` and `sum` only, and `sum` needs a numeric field; otherwise pivotlens
refuses with a message.

The same pivot from the command line, for scripts:

```
$ pivotlens --pivot rows=region,cols=status,values=amount,agg=sum testdata/tiny.csv
region,COMPLETED,FAILED,PENDING
AP-SOUTH,312.25,,
EU-CENT,12.5,84,
US-EAST,1420.5,,
US-WEST,,,9120
```

With several values and totals:

```
$ pivotlens --pivot rows=region,cols=status,values=amount:sum,*:count,totals=true testdata/tiny.csv
region,COMPLETED sum(amount),COMPLETED count,FAILED sum(amount),FAILED count,PENDING sum(amount),PENDING count,Total sum(amount),Total count
AP-SOUTH,312.25,1,,0,,0,312.25,1
EU-CENT,12.5,1,84,1,,0,96.5,2
US-EAST,1420.5,1,,0,,0,1420.5,1
US-WEST,,0,,0,9120,1,9120,1
Total,1745.25,3,84,1,9120,1,10949.25,5
```

`values=field:agg,...` lists the value fields with their aggregates, and
`*:count` counts rows. A bare field in `values` uses `agg=`. `totals=true`
adds the `Total` row and columns, and `percent=row|col|total` shows values
as a percent of row, of column or of the grand total. `cols`, `values`,
`totals` and `percent` are optional, and `agg` defaults to `count`. Add
`-o pivot.csv`, `-o pivot.parquet` or `-o pivot.ndjson` to write a file
instead of printing.

DuckDB computes the pivot, totals included, in SQL, so it stays fast on
millions of rows. A column field may have at most 50 distinct values;
past that, pivotlens refuses and asks for a field with fewer values. Rows
or columns whose field is NULL show up as `NULL` in the grid and as an empty field in CSV
export. An empty string in the column field is labeled `(empty)`.
With totals on, the first row field is exported as text, so the `Total`
label fits in the same column.

## How it works

Every view - the open file, an applied sort, a typed filter, a pivot - is one
`QuerySpec` value. A pure function compiles that struct into a single SQL
statement; DuckDB runs it. The grid never loads a whole file into memory:
it fetches the visible rows in 200-row pages as you scroll. Sorting and
filtering each queue their own DuckDB query, tagged with a request id, so
a fast keystroke never lets a slow, stale query overwrite a newer one.

- Engine: [`go-duckdb`](https://github.com/marcboeker/go-duckdb), embedded, no separate DuckDB install
- TUI: [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lipgloss](https://github.com/charmbracelet/lipgloss)
- File formats: CSV, Parquet, NDJSON, and the compressed variants DuckDB reads natively (`.gz`, `.zst`)

## Comparison

If you know `less`, the idea is the same - page through a file in place,
without loading or converting it first - except the unit is a row and a
column rather than a line, and you get sort and filter with it.

| Tool | Interactive TUI | Multi-GB files | No SQL needed | Formats |
|---|---|---|---|---|
| Excel / Google Sheets | yes | no, 1,048,576-row cap | yes | spreadsheet formats, csv |
| `visidata` | yes | yes | yes | csv, tsv, json, xlsx, and more via plugins |
| `csvlens` | yes | yes | yes | csv only |
| `duckdb` CLI | no, SQL shell | yes | no, SQL required | csv, parquet, json, and more |
| pivotlens | yes, with pivot tables | yes | yes | csv, parquet, ndjson |

## FAQ

**How do I view a large CSV file in the terminal?**
Run `pivotlens yourfile.csv`. It opens instantly because DuckDB reads the
file in a virtualized window instead of loading it whole; scrolling fetches
more rows as needed.

**What do I do when a CSV is too big to open in Excel or Google Sheets?**
Excel stops at 1,048,576 rows per worksheet and drops the rest. `pivotlens`
has no row limit of its own: DuckDB streams the rows the grid actually
displays.

**Can I view a Parquet file without pandas or PyArrow?**
Yes. `pivotlens yourfile.parquet` opens it directly; no Python environment is
needed.

**Is there a `visidata` or `csvlens` alternative written in Go?**
`pivotlens` is a single Go binary with the same kind of terminal grid, backed
by an embedded DuckDB instead of an in-process CSV parser. Unlike
`csvlens`, it also reads Parquet and NDJSON.

**How do I sort or filter a CSV without opening a spreadsheet?**
Press `s` to sort the current column, or `/` to type a filter. Both apply
instantly and can be undone with `u`. Press `f` on a column for a typed
filter such as `amount > 1000` or `status = FAILED`; `F` lists every active
filter for review or removal.

**How do I make a pivot table from a CSV without Excel?**
Press `p` in `pivotlens`, pick a row field, a column field, an aggregate
and a value field. For a script, run
`pivotlens --pivot rows=region,cols=status,values=amount,agg=sum data.csv`
and it prints the pivot as CSV. It works the same on Parquet and NDJSON.

**How do I inspect NDJSON or JSON Lines logs from the command line?**
`pivotlens yourlogs.ndjson` reads the file as a table, one row per line, so
you can sort by timestamp or filter down to a single request id without
piping it through `jq`. Compressed `.gz` and `.zst` logs open the same way.

**Is there a CSV and Parquet viewer for the Windows terminal?**
Yes. `pivotlens` is a single binary and behaves the same on Windows, Linux
and macOS. Download the Windows build from the Releases page; nothing else
needs installing, because DuckDB is embedded in the binary.

## Contributing and license

Issues and pull requests are welcome. Licensed under the [MIT License](LICENSE).
