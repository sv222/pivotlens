# Product & Engineering Specification: `pivotlens` (clean-csv)
**The Blazing-Fast, DuckDB-Powered Terminal Data Wrangler & Pivot Tool in Go**

---

## 1. Executive Summary & Product Identity

### 1.1 Pitch & Positioning
- **Working Name:** `pivotlens` (alternatives: `duck-table`, `quiver`)
- **One-Liner:** "Instant interactive pivot tables, schema cleaning, and SQL-powered slicing for multi-gigabyte CSV, Parquet, and JSON files — right in your terminal."
- **Hacker News / Show HN Hook:**
  > *"Show HN: pivotlens – A fast TUI for instant pivot tables and data exploration on multi-GB files, powered by DuckDB and Go"*
- **Target Audience:** Data Engineers, Backend Developers, DevOps/SREs, Data Analysts, Financial Engineers, and command-line power users who work with raw dumps, logs, and datasets.

### 1.2 The Core Value Proposition
`pivotlens` eliminates the friction between discovering a 2 GB file and understanding what is inside it. 
Instead of waiting 45 seconds for Excel to crash, writing a disposable Python/Pandas script, or memorizing cryptic command-line flags, users type:
```bash
pivotlens dataset.parquet
```
Within **sub-150 milliseconds**, the schema is inferred, data is displayed in an interactive, keyboard-driven terminal table, and users can build multi-dimensional pivot aggregations with three keystrokes.

---

## 2. Market Context & Gap Analysis

```
                      Fast / Low Latency (Go/Rust/C)
                                   ▲
                                   │
                                   │       [ pivotlens ]
                                   │   (Interactive Pivot,
                                   │    Zero-SQL GUI, DuckDB)
                     qsv / xsv     │
                 (CLI batch only,  │
                  no interactive)  │
                                   │
◄──────────────────────────────────┼──────────────────────────────────►
Batch / Headless / Scripted        │         Interactive / TUI
                                   │
                                   │   VisiData (Python, lags >1GB)
                                   │   Harlequin (SQL IDE, not a wrangler)
                                   │   Tad (Electron desktop GUI)
                                   │
                                   ▼
                      Slow / Heavy Runtime (Python/Electron)
```

### 2.1 The Problem
1. **The Modern Data Format Shift:** CSV is no longer alone. Parquet, NDJSON, and compressed files (`.csv.zst`, `.json.gz`) are standard. Classic CLI utilities (`awk`, `cut`, `xsv`, `column`) struggle or cannot read them without conversion.
2. **The "Pivot Table" Penalty:** Pivot tables and cross-tabulations remain the primary way humans explore aggregated datasets. Currently, running a pivot requires either loading data into Excel/Google Sheets, opening a Jupyter Notebook, or writing multi-line SQL with `PIVOT ... ON ... USING ...`.
3. **The VisiData Barrier:** `VisiData` is powerful, but its Python engine slows down on multi-gigabyte files, and its keybinding learning curve is notoriously steep.
4. **The SQL Barrier:** The `duckdb` CLI and tools like `Harlequin` or `rainfrog` assume the user wants to hand-write SQL queries into a prompt. Most exploratory questions ("What are the distinct categories?", "What is the average latency by region?") shouldn't require typing a single `SELECT` statement.

### 2.2 Why Go as the Implementation Language?
While Rust is celebrated for CLI tools, Go provides strategic advantages for this specific product:
1. **The Charm Ecosystem (`bubbletea`, `lipgloss`, `bubbles`):** Go has the most mature, visually polished, and active TUI ecosystem in the open-source world.
2. **Concurrency & Event Streams:** Handling file I/O, asynchronous DuckDB queries, and terminal render cycles via Go channels and goroutines simplifies cancellation and streaming compared to complex `async-std`/`tokio` state machine dances.
3. **Developer Contribution Velocity:** A wider pool of backend and data engineers write Go. Lowering the barrier to contribution accelerates community-driven connectors and extensions.
4. **Single-Binary Portability:** With a targeted CGO build pipeline using `zig cc`, Go produces clean, self-contained binaries for macOS, Linux, and Windows.

---

## 3. Product Scope & User Experience (UX)

### 3.1 Terminal Workflows

```
┌───────────────────────────────────────────────────────────────────────────────────────┐
│ File Discovery               Exploration Mode                  Action / Export        │
├───────────────────────┐      ┌─────────────────────────┐      ┌───────────────────────┤
│ $ pivotlens large.parquet│ ───► │ Interactive Grid View   │ ───► │ Instant Markdown /    │
│ or:                   │      │ - Dynamic Column Resiz  │      │ Parquet / CSV Output  │
│ $ cat log.json |      │      │ - Sorting & Filtering   │      │                       │
│     pivotlens            │      └───────────┬─────────────┘      └───────────────────────┘
└───────────────────────┘                  │
                                           ▼ (Press 'p')
                               ┌─────────────────────────┐
                               │ Interactive Pivot Modal │
                               │ - Row: region           │
                               │ - Col: status           │
                               │ - Val: count(request_id)│
                               └─────────────────────────┘
```

#### Primary User Actions
1. **Instant Inspection:** Schema auto-detection, null-count summary, and memory-efficient data preview.
2. **Visual Filtering & Sorting:** Vim-style search (`/`), instant column sorting (`s` -> choose column -> ASC/DESC), and visual filtering without manual SQL syntax.
3. **Two-Key Pivot Table Generation:**
   - User hits `p`.
   - A modal lets the user pick:
     - **Row Group(s):** e.g., `country`, `device`
     - **Pivot Column:** e.g., `subscription_tier`
     - **Aggregate Measure:** e.g., `sum(revenue)`, `avg(session_length)`, `count(*)`
   - Hitting `Enter` compiles and executes a native DuckDB `PIVOT` statement and presents the resulting matrix view in real time.
4. **Data Cleansing / Transformation Presets:**
   - Trim whitespace.
   - Drop nulls or replace null values with defaults.
   - Type coercion (e.g., string date to `TIMESTAMP`, float string to `DECIMAL`).
5. **Frictionless Export:**
   - Press `ctrl+y`: Copy current view or SQL statement to system clipboard as Markdown, JSON, or TSV.
   - Press `ctrl+s`: Export the mutated/filtered/pivoted dataset directly to `.parquet` or `.csv`.

### 3.2 TUI Mockup & Layout Specification

```
 pivotlens v0.1.0 │ dataset_prod_orders.parquet │ 4.2 GB │ 12,450,892 rows (filtered: 84,102)
────────────────────────────────────────────────────────────────────────────────────────────
 [1] order_id    │ [2] created_at       │ [3] customer_id │ [4] region │ [5] amount_usd ↕  
────────────────────────────────────────────────────────────────────────────────────────────
 #8941029        │ 2026-09-01 14:02:11  │ C-991823        │ US-EAST    │ $1,420.50         
 #8941030        │ 2026-09-01 14:02:15  │ C-102931        │ EU-CENT    │ $84.00            
 #8941031        │ 2026-09-01 14:03:02  │ C-884120        │ AP-SOUTH   │ $312.25           
 #8941032        │ 2026-09-01 14:03:22  │ C-441029        │ US-WEST    │ $9,120.00         
 #8941033        │ 2026-09-01 14:03:45  │ C-102931        │ EU-CENT    │ $12.50            
────────────────────────────────────────────────────────────────────────────────────────────
 [Pivot Modal: 'p']                                                                         
 ┌────────────────────────────────────────────────────────────────────────────────────────┐ 
 │ PIVOT BUILDER (Native DuckDB)                                                          │ 
 │                                                                                        │ 
 │ [Rows Group by]  : region                                                              │ 
 │ [Pivot Column]   : status (COMPLETED, FAILED, PENDING)                                 │ 
 │ [Aggregation]    : SUM(amount_usd)                                                     │ 
 │                                                                                        │ 
 │ Generated SQL: PIVOT (SELECT region, status, amount_usd FROM src)                      │ 
 │                ON status USING SUM(amount_usd) GROUP BY region                         │ 
 │                                                                                        │ 
 │   <Enter> Apply Pivot    <Tab> Switch Field    <Esc> Cancel                            │ 
 └────────────────────────────────────────────────────────────────────────────────────────┘ 
 Query: 84ms │ Memory: 184MB │ Keys: [?] Help  [/] Filter  [s] Sort  [p] Pivot  [e] Export 
```

### 3.3 Keyboard Ergonomics (Zero-Friction Vim Paradigm)
- `j` / `k` or `Down` / `Up`: Scroll rows.
- `h` / `l` or `Left` / `Right`: Scroll columns.
- `g` / `G`: Jump to Top / Bottom.
- `/`: Open interactive fuzzy-filter input.
- `s`: Column sort selector.
- `p`: Open Pivot Builder.
- `c`: Column manager (toggle visibility, rename, re-type).
- `e`: Export dialog (Format: Parquet, CSV, Markdown Table, SQL query).
- `y`: Copy selected cell / range / view to clipboard.
- `q` / `ctrl+c`: Instant clean exit.

---

## 4. Technical Architecture & Go Implementation

### 4.1 High-Level Component Graph

```
                   ┌────────────────────────────────────────┐
                   │               pivotlens CLI               │
                   │           (Flags, Args, Stdin)         │
                   └───────────────────┬────────────────────┘
                                       │
                    ┌──────────────────┴──────────────────┐
                    ▼                                     ▼
        ┌───────────────────────┐             ┌───────────────────────┐
        │     TUI Subsystem     │             │    Engine Subsystem   │
        │   (Bubble Tea MVU)    │◄──Messages──│   (DuckDB Go Driver)  │
        ├───────────────────────┤             ├───────────────────────┤
        │ - Grid Viewport       │──Commands──►│ - Virtual Memory DB   │
        │ - Pivot Builder View  │             │ - Auto File Scanner   │
        │ - Status/Filter Bar   │             │ - Windowed Paginated  │
        │ - Lipgloss Themes     │             │   Cursor Queries      │
        └───────────────────────┘             └───────────────────────┘
```

### 4.2 The Engine Core (`engine/duckdb.go`)
We interact with DuckDB through `github.com/marcboeker/go-duckdb` using `database/sql`. DuckDB is initialized as an in-memory database instance (`:memory:`) or with an optional spill-to-disk directory for constrained environments.

```go
package engine

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"

	_ "github.com/marcboeker/go-duckdb"
)

type Session struct {
	db        *sql.DB
	sourceRef string
	rowCount  int64
	columns   []ColumnMeta
}

type ColumnMeta struct {
	Name         string
	DatabaseType string
}

func NewSession(ctx context.Context, filePath string) (*Session, error) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize duckdb: %w", err)
	}

	// Performance Tuning Pragma configurations
	pragmas := []string{
		"SET preserve_insertion_order = false;",
		"SET threads TO 4;",
		"SET max_memory = '4GB';",
	}
	for _, pragma := range pragmas {
		if _, err := db.ExecContext(ctx, pragma); err != nil {
			return nil, fmt.Errorf("failed to exec pragma %s: %w", pragma, err)
		}
	}

	s := &Session{db: db}
	if err := s.loadSource(ctx, filePath); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Session) loadSource(ctx context.Context, filePath string) error {
	ext := strings.ToLower(filepath.Ext(filePath))
	var scanQuery string

	switch ext {
	case ".parquet":
		scanQuery = fmt.Sprintf("CREATE VIEW src AS SELECT * FROM read_parquet('%s');", filePath)
	case ".csv", ".tsv":
		scanQuery = fmt.Sprintf("CREATE VIEW src AS SELECT * FROM read_csv_auto('%s', header=true, sample_size=20480);", filePath)
	case ".json", ".ndjson":
		scanQuery = fmt.Sprintf("CREATE VIEW src AS SELECT * FROM read_json_auto('%s');", filePath)
	default:
		// Fallback to duckdb's sniffer
		scanQuery = fmt.Sprintf("CREATE VIEW src AS SELECT * FROM '%s';", filePath)
	}

	if _, err := s.db.ExecContext(ctx, scanQuery); err != nil {
		return fmt.Errorf("failed to scan input file: %w", err)
	}

	// Read column schema
	rows, err := s.db.QueryContext(ctx, "DESCRIBE src;")
	if err != nil {
		return fmt.Errorf("failed to read schema: %w", err)
	}
	defer rows.Close()

	s.columns = make([]ColumnMeta, 0)
	for rows.Next() {
		var col, colType, null, key, def, extra sql.NullString
		if err := rows.Scan(&col, &colType, &null, &key, &def, &extra); err != nil {
			return err
		}
		s.columns = append(s.columns, ColumnMeta{
			Name:         col.String,
			DatabaseType: colType.String,
		})
	}

	// Calculate initial row count asynchronously or fast estimate
	var count int64
	row := s.db.QueryRowContext(ctx, "SELECT count(*) FROM src;")
	if err := row.Scan(&count); err != nil {
		return err
	}
	s.rowCount = count
	return nil
}
```

### 4.3 Windowed Virtual Table Strategy
Loading 10 million rows into Go memory will trigger garbage collection pauses, ruin terminal responsiveness, and defeat the purpose of using DuckDB.

**The Solution:** The TUI operates as a **virtualized viewport**. The Go application maintains an in-memory buffer of only **visible rows + lookahead buffer** (e.g., 200 rows). When the user scrolls past the threshold, a background goroutine queries the next slice using indexed windowing.

```go
// FetchSlice executes a fast windowed query
func (s *Session) FetchSlice(ctx context.Context, offset, limit int, sortCol string, ascending bool) ([][]string, error) {
	orderClause := ""
	if sortCol != "" {
		direction := "ASC"
		if !ascending {
			direction = "DESC"
		}
		orderClause = fmt.Sprintf("ORDER BY \"%s\" %s", sortCol, direction)
	}

	query := fmt.Sprintf("SELECT * FROM src %s LIMIT %d OFFSET %d;", orderClause, limit, offset)
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	data := make([][]string, 0, limit)

	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}

		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}

		rowStrings := make([]string, len(cols))
		for i, val := range vals {
			if val == nil {
				rowStrings[i] = "NULL"
			} else {
				rowStrings[i] = fmt.Sprintf("%v", val)
			}
		}
		data = append(data, rowStrings)
	}

	return data, nil
}
```

### 4.4 The Dynamic Pivot Query Generator
DuckDB has built-in, vectorized `PIVOT` syntax. `pivotlens` translates UI modal inputs directly into DuckDB's SQL syntax:

```go
type PivotConfig struct {
	SourceTable string
	Rows        []string // e.g., []string{"region"}
	PivotOn     string   // e.g., "status"
	UsingAgg    string   // e.g., "sum(amount_usd)"
}

func (s *Session) BuildPivotView(ctx context.Context, cfg PivotConfig) error {
	rowsList := strings.Join(cfg.Rows, ", ")
	
	// Native DuckDB SQL Pivot syntax
	pivotSQL := fmt.Sprintf(`
		CREATE OR REPLACE VIEW src_pivoted AS 
		PIVOT %s 
		ON "%s" 
		USING %s 
		GROUP BY %s;
	`, cfg.SourceTable, cfg.PivotOn, cfg.UsingAgg, rowsList)

	_, err := s.db.ExecContext(ctx, pivotSQL)
	if err != nil {
		return fmt.Errorf("pivot execution failed: %w", err)
	}
	return nil
}
```

### 4.5 The Bubble Tea UI Architecture
`pivotlens` follows the Model-View-Update (Elm architecture) pattern using `charmbracelet/bubbletea`.

```
                        ┌─────────────────────────────────┐
                        │        Init() / Terminal        │
                        └────────────────┬────────────────┘
                                         │
                                         ▼
                                ┌─────────────────┐
                   ┌───────────►│  Update(msg)    │◄───────────┐
                   │            └────────┬────────┘            │
                   │                     │                     │
            tea.KeyMsg /                 │ Returns             │ tea.Cmd (Async)
           Custom Events                 ▼                     │ (DuckDB Queries)
                   │             (New Model, Cmd)              │
                   │                     │                     │
                   │                     ▼                     │
                   │            ┌─────────────────┐            │
                   └────────────│  View() string  │────────────┘
                                └─────────────────┘
```

#### State Definitions
```go
package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"pivotlens/internal/engine"
)

type Mode int
const (
	ModeGrid Mode = iota
	ModeFilter
	ModePivotModal
	ModeExport
)

type Model struct {
	session    *engine.Session
	mode       Mode
	width      int
	height     int
	cursorRow  int
	cursorCol  int
	offsetRow  int
	visibleData [][]string
	sortCol    string
	sortAsc    bool
	styles     Styles
}

func NewModel(session *engine.Session) Model {
	return Model{
		session: session,
		mode:    ModeGrid,
		styles:  DefaultStyles(),
	}
}

func (m Model) Init() tea.Cmd {
	return m.fetchVisibleDataCmd()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, m.fetchVisibleDataCmd()

	case tea.KeyMsg:
		switch m.mode {
		case ModeGrid:
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "j", "down":
				m.cursorRow++
				return m, m.checkScrollDown()
			case "k", "up":
				if m.cursorRow > 0 {
					m.cursorRow--
				}
				return m, m.checkScrollUp()
			case "p":
				m.mode = ModePivotModal
				return m, nil
			case "/":
				m.mode = ModeFilter
				return m, nil
			}
		case ModePivotModal:
			// Handle modal keys (Esc to exit, Tab to change fields)
		}
	}
	return m, nil
}
```

---

## 5. Overcoming Technical Pitfalls & Build Engineering

### 5.1 The CGO / Static Binary Problem
Because DuckDB is written in C++, `go-duckdb` requires CGO. A naive `go build` creates dynamic dependencies on `glibc`, breaking portability across different Linux distributions and Alpine environments.

#### The Production Solution: `zig cc` Cross-Compilation
We use `zig` as a drop-in C/C++ cross-compiler to build static musl binaries:

```makefile
# Makefile for truly static release binaries
BINARY_NAME=pivotlens
VERSION=0.1.0

build-linux-amd64:
	CC="zig cc -target x86_64-linux-musl" \
	CXX="zig c++ -target x86_64-linux-musl" \
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
	go build -ldflags="-w -s -extldflags '-static'" -o bin/$(BINARY_NAME)-linux-amd64 main.go

build-darwin-arm64:
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 \
	go build -ldflags="-w -s" -o bin/$(BINARY_NAME)-darwin-arm64 main.go

build-windows-amd64:
	CC="zig cc -target x86_64-windows-gnu" \
	CXX="zig c++ -target x86_64-windows-gnu" \
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 \
	go build -ldflags="-w -s" -o bin/$(BINARY_NAME)-windows-amd64.exe main.go
```

### 5.2 Terminal Width Constraints on Large Pivot Results
Pivot tables often produce wide outputs (e.g., pivoting on 50 states or 30 status codes creates 30+ columns).
- **Sticky Headers & Pinned Columns:** The row labels (e.g., `region`) remain frozen on the left while horizontal arrow keys (`h`/`l`) pan across pivot values.
- **Auto-Compaction:** If numbers are large, numbers are formatted using SI suffixes ($1.4M instead of $1,402,102.22) with a toggle (`#`) to show raw precision.

---

## 6. MVP Implementation Roadmap (4 Weeks to Launch)

```
 Week 1: Core Engine Integration
 ├── Set up repository, CGO build matrix, and CI pipeline (zig cc)
 ├── Implement DuckDB in-memory session wrapper
 └── Add multi-format file readers (CSV auto-sniff, Parquet, JSON)

 Week 2: Interactive Grid & Virtual Pagination
 ├── Implement Bubble Tea basic layout (Header, Grid, Status Bar)
 ├── Implement virtualized viewport and windowed DuckDB queries
 └── Wire up arrow keys, Vim motions (j/k/h/l), and column auto-sizing

 Week 3: Pivot Builder & Search/Filter
 ├── Build the Interactive Pivot Modal (Select Row, Column, Aggregate)
 ├── Hook UI model into native DuckDB PIVOT generation
 └── Implement real-time fuzzy filtering (generating DuckDB WHERE clauses)

 Week 4: Polish, Exports & Launch Packaging
 ├── Add clipboard copy (Markdown/TSV) and export to Parquet/CSV
 ├── Record demo GIF with VHS (`charmbracelet/vhs`)
 ├── Author Homebrew formula, Go install path, and GitHub Release automation
 └── Execute Hacker News & Reddit launch
```

---

## 7. The GitHub Stars & Launch Playbook

To hit **1,000–3,000+ stars** in week one, the repository presentation and launch sequencing must be executed precisely.

### 7.1 Visual Assets (The Rule of the 3-Second Hook)
- **Do not use static screenshots.** Terminal utilities need motion.
- Use `vhs` (by Charm) to generate a high-framerate, crisp terminal GIF/MP4 showcasing:
  1. Terminal prompt: `pivotlens 5gb_sales_dump.parquet`
  2. The table loads **instantly** (sub-200ms).
  3. The user types `/` -> filters to `FAILED` orders.
  4. The user hits `p` -> selects `region` and `SUM(amount)` -> hits `Enter`.
  5. An aggregated pivot table appears immediately.
  6. The user copies the result as a clean Markdown table and exits.
- Place this GIF at the very top of the `README.md` before any text.

### 7.2 The Show HN Launch Plan

#### Title Formulation
> **Show HN: pivotlens – Fast TUI for instant pivot tables on multi-GB files (Go + DuckDB)**

#### First Comment Template (Post Immediately After Submission)
```markdown
Hey HN,

I spend a lot of my day staring at massive CSVs, Parquet files, and log dumps. 

Excel crashes on anything over a few hundred megabytes, writing disposable Python/Pandas scripts breaks flow state, and running the standard DuckDB CLI requires remembering and typing full `PIVOT ... ON ... USING` SQL statements every time I just want to glance at an aggregation.

I built `pivotlens` to bridge this gap.

What it does:
- Single static binary (Go + embedded DuckDB, no Python runtime or external database required).
- Opens CSV, Parquet, and NDJSON files instantly via memory-mapped windowing.
- Interactive Pivot Builder: press 'p', pick your rows, columns, and metric, and it runs a native DuckDB vectorized pivot underneath.
- Exports results in one keystroke to Markdown, Parquet, or your system clipboard.

Architecture:
- TUI built using Charm's Bubble Tea and Lipgloss.
- Engine uses `go-duckdb`. The TUI renders a virtualized window buffer so it never loads the entire file into Go memory; DuckDB handles the heavy analytical lifting.
- Cross-compiled as static musl binaries using `zig cc`.

Code is open-source under MIT: https://github.com/yourusername/pivotlens

I'd love feedback on the keyboard shortcuts, pivot UX, and what formats you'd like to see supported next!
```

### 7.3 Targeted Subreddit Distribution Strategy
Post with different tailored angles within 48 hours of the HN launch:
1. **`r/dataengineering`:** Focus on the pain of quickly checking Parquet files in S3/local data lakes without spinning up PySpark or writing Athena queries.
2. **`r/golang`:** Focus on the technical implementation: How Bubble Tea virtualizes DuckDB query slices, and how `zig cc` solved static musl builds with CGO.
3. **`r/commandline` and `r/unixporn`:** Focus on the beauty of the TUI, themes, and Vim keybindings.

### 7.4 Package Manager Availability on Day 1
Developers will not manually compile code to test a tool:
- **Homebrew:** `brew install yourusername/tap/pivotlens`
- **Go Install:** `go install github.com/yourusername/pivotlens@latest`
- **Arch User Repository (AUR):** `yay -S pivotlens-bin`
- **GitHub Releases:** Pre-compiled static binaries for `linux-amd64`, `linux-arm64`, `darwin-amd64`, `darwin-arm64`, and `windows-amd64`.

---

## 8. Summary Checklist Before Public Release

- [ ] Does `pivotlens data.csv` open a 2 GB file in under 250ms without swapping memory?
- [ ] Does resizing the terminal window cleanly re-render without tearing or breaking column alignments?
- [ ] Does pressing `p` on a 10M-row file return the pivoted aggregation in under 1 second?
- [ ] Is there an automated GitHub Actions pipeline generating signed release assets for all 5 major platforms?
- [ ] Is the README GIF under 4 MB, auto-playing, and demonstrating the complete workflow within 15 seconds?

With this specification, `pivotlens` is designed to be technically robust, straightforward to build for a single developer in 4 weeks, and laser-targeted at developer pain points to maximize organic traction, GitHub stars, and community adoption.