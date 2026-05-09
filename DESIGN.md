# parquetv — Interactive Parquet File Explorer

Design Doc

| Field | Value |
|---|---|
| Author(s) | Javi Molina |
| Created | May 2026 |
| Status | Draft |
| Language | Go |
| Dependencies | parquet-go, bubbletea, lipgloss, bubbles |

## 1. Background

Parquet is a columnar format that stores data very differently from how humans think about it. A single trace becomes 68+ leaf columns. Nested structures like span attributes become flat value lists with definition and repetition levels. Pages split column data into compressed blocks with min/max statistics that enable predicate pushdown.

This physical reality is invisible when using SQL tools like DuckDB or dump tools like `parquet-tools`. They reconstruct the logical row view and hide the columnar mechanics entirely. There is no interactive tool that shows both the logical data and the physical storage simultaneously.

For Tempo specifically, understanding how vParquet5 stores traces is critical for:
- Writing efficient TraceQL queries (why scoped attributes are faster, why dedicated columns matter)
- Debugging query performance (which columns are expensive, why pushdown didn't help)
- Validating block structure (dedicated column assignments, row group sizes, attribute cardinality)
- Onboarding new contributors to the storage engine

## 2. Goals

Build `parquetv`, a terminal UI for exploring Parquet files that shows both the logical (rows) and physical (columns, pages, encodings) views linked together.

### Must have
- Open any Parquet file (not Tempo-specific)
- Three-level navigation: row groups → row/column grid → pages/values
- Per-column filters with visible cost reporting (page skips, dict lookups, values scanned)
- Generic attribute KV filters (key scan + value lookup with cost breakdown)
- See the physical column/page structure for any selected column
- Show page-level statistics (min/max, size, encoding, compression)
- Display definition and repetition levels with decoded nesting context
- Keyboard-driven navigation

### Nice to have
- Tempo-aware mode: interpret vParquet5 schema (trace IDs, span trees, NSM bounds)
- Dictionary inspection: show dictionary entries for string columns
- Column size comparison: sort columns by size to find bloat
- Object storage support: read blocks directly from S3/GCS via URL

### Non-goals
- Write/edit Parquet files
- Full SQL query engine (use DuckDB for that)
- Web UI

## 3. Design

### 3.1 Architecture layers

Strict separation between three layers. No layer imports from a layer above it.

```
┌──────────────────────────────────────────────────────────────┐
│  UI Layer (view/)                                            │
│  Pure rendering. Receives a ViewModel, returns a string.     │
│  lipgloss styles, layout math, no state, no side effects.    │
│  One file per level: level1.go, level2.go, level3.go         │
│  Shared: chrome.go (top bar, bottom bar, help overlay)       │
├──────────────────────────────────────────────────────────────┤
│  Model Layer (model/)                                        │
│  BubbleTea models. Owns state, handles Update(msg),          │
│  produces ViewModels for the UI layer.                       │
│  Navigation stack, cursor position, filter state,            │
│  viewport offset. Calls engine for data.                     │
├──────────────────────────────────────────────────────────────┤
│  Engine Layer (engine/)                                      │
│  Parquet reading. No BubbleTea imports, no lipgloss.         │
│  Opens files, reads footer, decodes pages, resolves          │
│  nesting, evaluates filters. Returns plain Go structs.       │
│  Testable without a terminal.                                │
└──────────────────────────────────────────────────────────────┘
```

**UI layer** — stateless rendering functions. Each takes a `ViewModel` struct and terminal dimensions, returns a styled string. All lipgloss usage lives here. No `tea.Model`, no `Update()`. This means:
- Views are pure functions — same input always produces same output
- Styling changes never touch model or engine code
- Golden file tests call the view function directly with a test ViewModel

**Model layer** — BubbleTea `tea.Model` implementations. Owns the navigation stack, cursor positions, filter state, viewport offsets, and inspect panel toggle. On each `Update()`, it may call engine methods to fetch data, then builds a `ViewModel` struct for the UI layer. The `View()` method just delegates to the UI layer:

```go
func (m Level2Model) View() string {
    vm := m.buildViewModel() // model → ViewModel
    return view.RenderLevel2(vm, m.width, m.height) // pure render
}
```

**Engine layer** — plain Go. Opens Parquet files, reads metadata, decodes pages, resolves def/rep levels into nesting context, evaluates filter predicates, computes column stats. Returns structs like `RowGroupInfo`, `PageInfo`, `CellValue`, `FilterResult`. Zero dependency on BubbleTea or lipgloss. Fully testable with `go test` against real Parquet files.

The `ViewModel` structs are the contract between model and view:

```go
// Shared across levels
type Chrome struct {
    TopBar     TopBarData
    BottomBar  BottomBarData
}

type Level1ViewModel struct {
    Chrome
    RowGroups    []RowGroupSummary
    Selected     int
    FooterPanel  FooterData
    SchemaTree   *SchemaNode // nil if panel closed
}

type Level2ViewModel struct {
    Chrome
    Headers      []ColumnHeader
    Rows         [][]CellValue   // only visible rows (virtual scroll)
    SelectedRow  int
    SelectedCol  int
    Filters      []ActiveFilter
    ColumnStats  ColumnStatsData // for selected column
    InspectPanel *InspectData    // nil if panel closed
    PageBounds   []int           // row indices where page boundaries fall
}
```

This separation ensures:
- Engine can be reused outside the TUI (CLI mode, tests, library)
- UI can be restyled without touching logic
- Models are testable by calling `Update()` + checking the ViewModel
- 60fps rendering — views are pure string builders with no I/O

#### Directory layout

```
parquetv/
├── main.go              # CLI entry, arg parsing, tea.NewProgram
├── engine/              # Parquet reading — no TUI imports
│   ├── file.go          # Open file, read footer, schema
│   ├── rowgroup.go      # Row group metadata, column stats
│   ├── page.go          # Page reading, decompression, value decoding
│   ├── filter.go        # Filter evaluation, page skip, dict shortcut
│   ├── nesting.go       # Def/rep level → nesting context resolver
│   └── types.go         # Shared structs: RowGroupInfo, PageInfo, CellValue, etc.
├── model/               # BubbleTea models — state + Update()
│   ├── app.go           # Root model, navigation stack, level switching
│   ├── level1.go        # File overview model
│   ├── level2.go        # Row group grid model
│   ├── level3.go        # Page inspector model
│   └── viewmodel.go     # ViewModel structs (contract with view layer)
├── view/                # Pure rendering — no state, no side effects
│   ├── chrome.go        # Top bar, bottom bar, help overlay
│   ├── level1.go        # File overview rendering
│   ├── level2.go        # Row group grid rendering
│   ├── level3.go        # Page inspector rendering
│   ├── inspect.go       # Content/inspect panel rendering
│   └── styles.go        # All lipgloss styles in one place
├── testdata/            # Committed test fixtures
│   ├── small.parquet    # 17-row block for golden files
│   └── small.meta.json
├── Makefile
└── go.mod
```

Import rule: `view/` imports nothing from `model/` or `engine/` — it only receives ViewModel structs. `model/` imports `engine/` but not `view/` (it calls view functions in `View()` only). `engine/` imports neither.

### 3.2 Three-level navigation

The UI follows the physical Parquet structure exactly: file → row group → column chunk → pages. Three zoom levels, each entered with `enter` and exited with `esc`.

```
Level 1: File Overview    → enter →  Level 2: Row Group    → enter →  Level 3: Pages
(row groups + footer)                (rows × columns grid)            (page list + values)
```

#### Level 1 — File overview

The entry screen. Shows the row groups as a navigable list on the left and the footer metadata on the right. You see the file as Parquet sees it: a sequence of horizontal partitions plus a footer.

```
╭─ parquetv ─── block.parquet ─────────────────────────────────────────────────╮
│  148.2 MB  vParquet4  4,500 rows  3 row groups  68 columns                    │
╰─────────────────────────────────────────────────────────────────────────────╯

  Row Groups                     │  Footer
  ──────────                     │  ──────
  ▸ Row Group 0                  │  Schema: 68 leaf columns
      1,502 rows   49.1 MB      │  Format: vParquet4
      12 pages/col avg           │
                                 │  Top columns by size:
    Row Group 1                  │    rs.ss.Spans.Attrs.Value  12.1 MB
      1,498 rows   50.3 MB      │    rs.ss.Spans.Attrs.Key     8.2 MB
      12 pages/col avg           │    rs.ss.Spans.SpanID        8.2 MB
                                 │    rs.ss.Spans.ParentSpanID  7.9 MB
    Row Group 2                  │    rs.ss.Spans.DedicatedA..  5.1 MB
      1,500 rows   48.8 MB      │
      11 pages/col avg           │  Key-value metadata:
                                 │    tempo.block.format = vParquet4
                                 │    tempo.block.tenant = my-tenant
                                 │
 block.parquet                                     enter open  s schema  q quit  ? help
```

Pressing `s` shows the full schema tree as a side panel.

#### Level 2 — Row group (rows × columns)

Press `enter` on a row group and you're inside it. The grid shows rows vertically and columns horizontally — the logical view of the data. Navigate with arrow keys. The currently selected column header is highlighted.

```
╭─ parquetv ─── block.parquet ─────────────────────────────────────────────────╮
│  Row Group 0  1,502 rows  49.1 MB  12 pages/col                               │
╰─────────────────────────────────────────────────────────────────────────────╯

  Row   TraceID        DurationNano [RootServiceName] RootSpanName
  ───── ────────────── ───────────── ──────────────── ──────────────────
     0  a8c2f1d3e5..   250000000    checkout          POST /order
     1  a9d3e2f4a6..    50000000    api-gateway       GET /health
     2  ab44f3b5c7..  1200000000    payment           charge
     3  ac55a4c6d8..   340000000    auth              POST /login
     4  ad66b5d7e9..    89000000    checkout          GET /cart
     5  ae77c6e8fa..  5200000000    payment           refund
    ...
  ──────────────────────────────────────────────────────────────────────
  RootServiceName                     │
  type: string  enc: dict+snappy      │ dict: 12 unique strings
  size: 18 KB   values: 1,502         │ top: checkout(412) api-gateway(389)
  per-row: 1.0                        │      payment(301) auth(198) ...

 block.parquet › RG 0                   ↑↓ rows  ◂▸ cols  enter pages  f filter  r expand  esc back
```

The area above the bottom bar shows stats for the selected column (marked with `[]` in the header). The `per-row` value is the teaching moment: 1.0 for trace-level, ~20 for span-level, ~275 for `Attrs.Key`.

Pressing `r` on a row expands it inline to show the full nested structure (ResourceSpans → Spans → Attrs), like a tree that unfolds within the table.

#### Level 3 — Pages (column chunk detail)

Press `enter` on a column and you zoom into its column chunk for this row group. Left side shows the page list. Right side shows values for the selected page.

```
╭─ parquetv ─── Row Group 0 ── DurationNano ───────────────────────────────────╮
│  block.parquet › RG 0 › DurationNano                                 esc back│
╰─────────────────────────────────────────────────────────────────────────────╯

  Pages                          │  Page 1 values
  ─────                          │
    Page 0                       │  #     value           row
      values: 501                │  ───── ─────────────── ──────────────
      min: 12000000     (12ms)   │   501  812000000       ae77c6..
      max: 198000000    (198ms)  │   502  845000000       af88d7..
      size: 8.1 KB → 3.2 KB     │   503  912000000       b099e8..
      encoding: delta + snappy   │   504  1042000000      b1aaf9..
                                 │   505  1200000000      b2bb0a..
  ▸ Page 1                       │   506  1201000000      b3cc1b..
      values: 501                │   507  1890000000      b4dd2c..
      min: 812000000    (812ms)  │   508  2340000000      b5ee3d..
      max: 48200000000  (48.2s)  │   509  5100000000      b6ff4e..
      size: 9.4 KB → 3.8 KB     │   510  48200000000     b700f5..
      encoding: delta + snappy   │   ...
                                 │
    Page 2                       │  Delta encoding:
      values: 500                │  base: 812000000
      min: 15000000     (15ms)   │  deltas: +33M +67M +130M +158M
      max: 31200000000  (31.2s)  │          +1M +689M +450M +2.76B
      ...                        │

 ↑↓ pages  enter values  d dictionary  f filter (simulate pushdown)  esc back
```

For span-level columns like `rs.ss.Spans.StatusCode`, the value list adds def/rep columns and a decoded nesting context:

```
  Page 0 values                                               │
                                                               │
  #     val  def  rep  context                                 │
  ───── ──── ──── ──── ─────────────────────────────────────── │
     0   0    3    0   trace a8c2f1.. › rs0 › s0 › span 0     │
     1   2    3    3   trace a8c2f1.. › rs0 › s0 › span 1     │
     2   0    3    3   trace a8c2f1.. › rs0 › s0 › span 2     │
     3   0    3    0   trace a9d3e2.. › rs0 › s0 › span 0     │
     4   0    3    3   trace a9d3e2.. › rs0 › s0 › span 1     │
     5   2    3    3   trace a9d3e2.. › rs0 › s0 › span 2     │
    ...                                                        │
                                                               │
  Legend: def=3 span present  rep=0 new trace  rep=3 new span  │
```

Pressing `f` opens a filter input. Type `> 1s` and pages highlight:

```
    Page 0   min: 12ms   max: 198ms   SKIP (max < 1s)
  ▸ Page 1   min: 812ms  max: 48.2s   READ (might match)
    Page 2   min: 15ms   max: 31.2s   READ (might match)

  Predicate pushdown: 1 of 3 pages skipped (33%)
```

### 3.3 Content panel

A toggleable side panel (press `i` for inspect) that shows the raw representation of the value under the cursor. Updates live as you navigate.

```
  Row   TraceID        DurationNano  RootSvc     │  Inspect
  ───── ────────────── ───────────── ──────────── │
     0  a8c2f1d3e5..   250000000    checkout      │  TraceID (row 0)
     1  a9d3e2f4a6..    50000000    api-gateway   │
  ▸  2  ab44f3b5c7..  1200000000    payment       │  Hex
     3  ac55a4c6d8..   340000000    auth          │  ab 44 f3 b5 c7 d8 e9 fa
     4  ad66b5d7e9..    89000000    checkout      │  0b 1c 2d 3e 4f 50 61 72
                                                  │
                                                  │  Binary
                                                  │  10101011 01000100 11110011
                                                  │  10110101 11000111 ...
                                                  │
                                                  │  Decoded
                                                  │  ab44f3b5c7d8e9fa0b1c2d3e4f506172
                                                  │
                                                  │  Type: []byte (16 bytes)
                                                  │  Encoding: plain
                                                  │  Page: 0 (offset 2)
```

The panel adapts to the value type:

| Value type | Inspect shows |
|---|---|
| `[]byte` (TraceID, SpanID) | Hex dump, binary, decoded hex string |
| `uint64` (DurationNano) | Raw nanos, human-readable (`1.2s`), binary |
| `string` | Raw value, byte length, dictionary code if dict-encoded |
| `int` (StatusCode) | Raw int, binary, Tempo-aware hint if known enum (`2 = ERROR`) |
| `[]string` (Attrs.Value) | Array elements listed, total size |
| `float64` | Raw, scientific notation, binary IEEE 754 |

For duration values specifically:

```
                                                  │  Inspect
                                                  │
                                                  │  DurationNano (row 2)
                                                  │
                                                  │  Raw:   1200000000
                                                  │  Human: 1.2s
                                                  │  Hex:   00 00 00 00 47 86 8C 00
                                                  │  Binary: 01000111 10000110 ...
                                                  │
                                                  │  Delta from prev: +950000000
                                                  │  Type: uint64
                                                  │  Encoding: delta + snappy
                                                  │  Page: 0 (offset 2)
                                                  │  Dict: n/a
```

The `Delta from prev` line is a nice touch for delta-encoded columns — you see the actual delta the encoder stored, reinforcing how the encoding works.

For dict-encoded strings:

```
                                                  │  Inspect
                                                  │
                                                  │  RootServiceName (row 2)
                                                  │
                                                  │  Value: "payment"
                                                  │  Bytes: 7
                                                  │  Hex:   70 61 79 6D 65 6E 74
                                                  │
                                                  │  Dict code: 7
                                                  │  Dict size: 12 entries
                                                  │  Frequency: 301 / 1,502 (20.0%)
                                                  │
                                                  │  Type: string
                                                  │  Encoding: dict + snappy
                                                  │  Page: 0 (offset 2)
```

The dictionary code and frequency make you understand why dict encoding works — 301 occurrences of "payment" stored as the integer `7` instead of the 7-byte string.

Toggle with `i`. Works in Level 2 (grid cells) and Level 3 (page values). The panel takes ~30 columns on the right side, so the grid narrows. On small terminals, it could be a bottom panel instead.

### 3.4 Column filters (querying without a query language)

No query language. Instead, filters are per-column: one column + one operator + one value. Press `f` on any column in the Level 2 grid to open a filter input.

```
f  (on DurationNano)

┌─ Filter: DurationNano ──────────────────────────────┐
│                                                      │
│  > 1000000000                                        │
│                                                      │
│  Operators: =  !=  >  >=  <  <=  ~(regex)           │
│  Duration shorthand: 1s  500ms  2m                   │
│                                                      │
└──────────────────────────────────────────────────────┘
```

Press enter and the grid filters in place — non-matching rows grey out or hide. Multiple filters stack as AND. Active filters show as pills:

```
  Filters: DurationNano > 1s  ×   RootServiceName = "payment"  ×

  Row   TraceID        DurationNano  RootServiceName  RootSpanName
  ───── ────────────── ───────────── ──────────────── ──────────────
     2  ab44f3b5c7..  1200000000    payment           charge
     5  ae77c6e8fa..  5200000000    payment           refund
    23  d4ee1ac56f..  3400000000    payment           webhook-retry
    ...

  3 of 1,502 rows match (2 filters active)  c clear all
```

The status bar shows work done, making the cost model visible:

```
  DurationNano: 2 of 4 pages skipped (min/max pushdown), 1,001 values scanned
  RootServiceName: dict lookup "payment" = code 7, 1,502 values scanned
  Total: 3 matches
```

#### Evaluation strategy

No expression parser or query engine needed. Each filter uses what Parquet already provides:

1. **Page-level skip** — check page min/max stats against the filter. Skip pages that can't contain matches. This is the same predicate pushdown the Level 3 simulation shows, but now it actually filters rows.
2. **Dictionary shortcut** — for dict-encoded string columns, check if the filter value exists in the dictionary. If not, skip the entire column chunk (zero matches guaranteed).
3. **Value scan** — for non-skipped pages, decompress and check values one by one.

#### Filtering on generic attributes (Attrs.Key + Value)

For the generic key-value columns, the filter input has two fields:

```
f  (on rs.ss.Spans.Attrs)

┌─ Filter: rs.ss.Spans.Attrs ─────────────────────────┐
│                                                      │
│  Key:   http.method                                  │
│  Value: = GET                                        │
│                                                      │
│  (filters to traces containing a span with this KV)  │
└──────────────────────────────────────────────────────┘
```

The row grid filters to traces containing a matching span. The status bar shows the full cost:

```
  Attrs.Key: dict lookup "http.method" = code 3 (present)
  Scanned 412,340 key entries across 4 pages, matched 1,204 positions
  Attrs.Value: read 1,204 values at matched positions, 389 = "GET"
  Result: 389 traces
```

This is the teaching moment. Compare filtering `DurationNano > 1s` (skips 2 pages, scans 1,001 values) against filtering `span.http.method = "GET"` (scans 412,340 key entries, then 1,204 values). The cost difference is visceral, not theoretical.

### 3.5 Lazy loading and virtual scrolling

A block can have 149K rows and 412K attribute entries. The tool never loads everything upfront.

#### Read strategy

| What | When loaded |
|---|---|
| Footer (schema, row group metadata) | On file open — the only eager read |
| Column indexes (min/max per page) | On demand when column selected in Level 2 stats bar or Level 3 |
| Pages (compressed data) | On demand when scrolling into a page's row range or entering value viewer |
| Row data for grid | On demand, one page-worth of rows at a time |

This is natural with `parquet-go`'s API which works via `io.ReaderAt` — random access reads at specific offsets. No sequential scan needed.

#### Virtual scrolling

The Level 2 grid and Level 3 value viewer use virtual scrolling — only the visible rows exist in memory. The model tracks:

```go
type GridModel struct {
    totalRows   int       // from row group metadata (known from footer)
    viewport    Viewport  // visible window: offset + height
    cache       *PageCache // decoded rows for currently visible pages
}
```

Scrolling `↑↓` moves the viewport offset. When the viewport crosses a page boundary, the new page is decoded and the previous page can be evicted. `g` (jump to first) and `G` (jump to last) seek directly to the target page without reading intermediate pages.

For the 149K row block: the viewport shows ~30 rows. Only the page containing those 30 rows is decoded. Scrolling through the entire block reads pages sequentially but never holds more than 2-3 decoded pages in memory.

#### Jump-to navigation

Press `:` to open a jump prompt (vim-style). Direct access without scrolling through intermediate data:

```
Level 1:  :2        → jump to Row Group 2
Level 2:  :1000     → jump to row 1000
Level 2:  :Attrs    → jump to column matching "Attrs"
Level 3:  :5        → jump to Page 5
```

The jump reads only the target page — no lazy loading of everything in between. For row jumps, the row group index maps row numbers to pages, so the seek is O(1).

`/` (search) scans lazily: it reads pages sequentially from the current position, decoding one at a time, and stops at the first match. It doesn't preload the entire column.

### 3.6 Nesting resolver (Level 3)

The most complex component. Given a value's position, definition level, and repetition level, it produces a human-readable context string like:

```
trace a8c2f1.. › rs0 › scope0 › span 2
```

Implementation: maintain a stack of counters, one per nesting level. When repetition level drops to level N, increment the counter at level N and reset all deeper counters. The trace ID is resolved by cross-referencing with the TraceID column's value at the corresponding trace-level row number.

## 4. Navigation and controls

### Chrome: top bar and bottom bar

Two persistent bars frame every level. The content adapts to the current navigation state.

**Top bar** — file identity + context stats. Always visible, never scrolls.

```
Level 1:
╭─ parquetv ─── block.parquet ─────────────────────────────────────────────────╮
│  148.2 MB  vParquet4  4,500 rows  3 row groups  68 columns                    │
╰─────────────────────────────────────────────────────────────────────────────╯

Level 2:
╭─ parquetv ─── block.parquet ─────────────────────────────────────────────────╮
│  Row Group 0  1,502 rows  49.1 MB  12 pages/col                               │
╰─────────────────────────────────────────────────────────────────────────────╯

Level 3:
╭─ parquetv ─── block.parquet ─────────────────────────────────────────────────╮
│  DurationNano  uint64  delta+snappy  102.4 KB  4,500 values                    │
╰─────────────────────────────────────────────────────────────────────────────╯
```

The file name is always in the top bar. The stats change to describe what you're looking at: file-level at L1, row-group-level at L2, column-level at L3.

**Bottom bar** — breadcrumb path + contextual shortcuts. Always visible, never scrolls.

```
Level 1:
 block.parquet                                     enter open  s schema  q quit  ? help

Level 2:
 block.parquet › RG 0                  ↑↓ rows  ◂▸ cols  enter pages  f filter  r expand  esc back

Level 2 with filters active:
 block.parquet › RG 0  [142/1,502]     ↑↓ rows  ◂▸ cols  f filter  c clear  esc back

Level 3:
 block.parquet › RG 0 › DurationNano    ↑↓ pages  ◂▸ column  enter values  f simulate  d dict  esc back

Level 3 value viewer:
 block.parquet › RG 0 › DurationNano › Page 1    ↑↓ scroll  y yank  esc back
```

The breadcrumb on the left shows exactly where you are in the drill-down. The shortcuts on the right show only what's available at the current state — no full keybinding dump, just the relevant actions. `?` opens a full help overlay if needed.

When filters are active in Level 2, the breadcrumb shows the match count (`[142/1,502]`).

### Global

| Key | Action |
|---|---|
| `enter` | Drill into selected item (next level) |
| `esc` | Back to previous level |
| `q` | Quit |
| `?` | Help overlay (show all shortcuts for current level) |
| `↑` `↓` / `j` `k` | Move cursor up/down |
| `g` / `G` | Jump to first / last item |
| `ctrl+d` / `ctrl+u` | Half-page down / up |
| `:` | Jump prompt (`:1000` row, `:Attrs` column, `:5` page) |
| `/` | Search from current position (lazy, stops at first match) |

### Level 1 — File overview

| Key | Action |
|---|---|
| `↑` `↓` | Select row group |
| `enter` | Open selected row group (go to Level 2) |
| `s` | Toggle schema tree side panel |

### Level 2 — Row group grid

| Key | Action |
|---|---|
| `↑` `↓` | Scroll rows |
| `◂` `▸` / `h` `l` | Select column (highlights header, updates stats bar) |
| `enter` | Open selected column's pages (go to Level 3) |
| `r` | Expand/collapse selected row inline (nested tree) |
| `f` | Add filter on selected column |
| `c` | Clear all filters |
| `s` | Cycle column sort in stats bar (path / size / values-per-row) |
| `/` | Search rows by value |
| `y` | Yank (copy) selected cell value to clipboard |
| `i` | Toggle inspect panel (hex, binary, decoded, encoding details) |

The bottom stats bar always shows metadata for the selected column: type, encoding, size, value count, values-per-row, and dictionary cardinality if applicable.

Active filters show as pills below the header. Each pill shows `column operator value ×`. Press `c` to clear all, or navigate to a pill and press `x` to remove one.

### Level 3 — Pages

| Key | Action |
|---|---|
| `↑` `↓` | Select page (left panel) |
| `enter` | Show values for selected page (right panel) |
| `◂` `▸` | Switch to prev/next column chunk (same row group) |
| `f` | Predicate simulation — type a filter, pages annotate SKIP/READ |
| `d` | Dictionary view — show unique values, frequencies, integer codes |
| `y` | Yank selected value to clipboard |
| `i` | Toggle inspect panel |

The value viewer for nested columns shows definition level, repetition level, and decoded nesting context alongside each value. A legend at the bottom explains the level meanings.

### Filtering — end to end

Filtering is the primary query mechanism. No query language — just column + operator + value.

#### Simple column filter

1. In Level 2 grid, use `◂▸` to select a column (e.g. `DurationNano`)
2. Press `f` — filter input opens:

```
┌─ Filter: DurationNano ──────────────────────────────┐
│                                                      │
│  [>]  1s                                             │
│                                                      │
│  tab cycle op: =  !=  >  >=  <  <=  ~(regex)       │
│  Shorthand: 1s  500ms  2m  (converted to nanos)     │
│  enter apply  esc cancel                             │
└──────────────────────────────────────────────────────┘
```

3. `tab` cycles the operator. Type the value. Duration shorthand (`1s`, `500ms`, `2m`) converts to nanoseconds automatically.
4. Press `enter` — filter applies. Grid updates:

```
  Filters: DurationNano > 1s  ×

  Row   TraceID        DurationNano  RootServiceName  RootSpanName
  ───── ────────────── ───────────── ──────────────── ──────────────
     2  ab44f3b5c7..  1200000000    payment           charge
     5  ae77c6e8fa..  5200000000    payment           refund
    23  d4ee1ac56f..  3400000000    payment           webhook-retry

  142 of 1,502 rows match │ DurationNano: 2/4 pages skipped, 1,001 values scanned
```

5. Select another column, press `f` again to stack a second filter (AND).
6. Press `c` to clear all filters, or `◂▸` to a filter pill and `x` to remove one.

#### Generic attribute filter (Attrs.Key + Value)

When the selected column is `rs.ss.Spans.Attrs` (or any generic attribute column), the filter input shows two fields:

```
┌─ Filter: rs.ss.Spans.Attrs ─────────────────────────┐
│                                                      │
│  Key:   http.method                                  │
│  Value: [=]  GET                                     │
│                                                      │
│  tab switch field  enter apply  esc cancel            │
└──────────────────────────────────────────────────────┘
```

`tab` switches between Key and Value fields. The operator applies to the Value only. Press `enter` and the grid filters to traces containing a span with that KV pair:

```
  Filters: Attrs[http.method] = "GET"  ×

  389 of 1,502 rows match
  Cost: dict lookup "http.method" = code 3 (present)
        scanned 412,340 key entries, 1,204 matched
        read 1,204 values, 389 = "GET"
```

The cost line is the teaching moment — you see the full key scan cost compared to a simple column filter.

#### Predicate simulation (Level 3, read-only)

Press `f` from the Level 3 page list to simulate a filter without applying it to the grid. Pages annotate with SKIP/READ:

```
  Page 0   min: 12ms    max: 198ms   SKIP (max < 1s)
  Page 1   min: 812ms   max: 48.2s   READ (might match)
  Page 2   min: 15ms    max: 31.2s   READ (might match)

  Pushdown: 1 of 3 pages skipped (33%)
```

This previews what a filter would do at the page level without changing the grid state. Useful for exploring predicate pushdown behavior before committing.

#### How filtering works internally

No expression parser or query engine. Each filter evaluates using what Parquet already provides:

1. **Page-level skip** — check page min/max stats against the filter value. Skip pages that can't contain matches.
2. **Dictionary shortcut** — for dict-encoded string columns, check if the value exists in the dictionary. If absent, skip the entire column chunk.
3. **Value scan** — decompress non-skipped pages, check values one by one.

For generic attribute filters, step 1-3 apply to `Attrs.Key` first (scan all entries for matching key), then step 3 applies to the corresponding `Attrs.Value*` column at matched positions only.

Every step reports its work in the status line so the user sees exactly what happened.

### Visual feedback

- **Selected column**: header highlighted in the grid, stats bar updates
- **Page boundaries**: dashed lines between rows in the grid, shift when selecting a different column
- **Active filters**: pill badges below header with match count
- **Filter cost**: status line shows pages skipped, entries scanned, matches found
- **Predicate simulation**: pages annotated with SKIP (greyed) or READ (normal) + skip ratio

## 5. Development methodology

### Build loop

BubbleTea uses the Elm architecture: `Model` + `Update(msg)` + `View() string`. The `View()` function returns a plain string — no terminal needed to test it. This enables a tight golden-file-based development loop:

1. Create model with test data (real parquet file)
2. Send keystrokes via `model.Update(tea.KeyMsg{...})`
3. Call `model.View()` → get rendered string
4. Compare against golden file → fail with diff if changed

```go
// level1_test.go
func TestLevel1View(t *testing.T) {
    f, _ := parquet.OpenFile(...)
    model := NewLevel1Model(f)
    got := model.View()

    golden := "testdata/level1.golden"
    if *update {
        os.WriteFile(golden, []byte(got), 0644)
        return
    }
    want, _ := os.ReadFile(golden)
    if got != string(want) {
        t.Errorf("diff:\n%s", diff(got, string(want)))
    }
}

// Test navigation
func TestLevel1EnterRowGroup(t *testing.T) {
    model := NewLevel1Model(f)
    model, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
    model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
    got := model.View()
    // assert it switched to Level 2 for row group 1
}
```

For full program integration tests, `charmbracelet/x/exp/teatest` spins up the TUI in a pseudo-terminal, sends keystrokes, and snapshots frames against golden files.

The file-watching loop:

```makefile
test:
	go test ./... -count=1

update-golden:
	go test ./... -update

dev:
	find . -name '*.go' | entr -cr make test

run:
	go run . testdata/small.parquet

live:
	find . -name '*.go' | entr -cr go run . testdata/small.parquet
```

`make dev` in one terminal, edit code in another. Every save runs tests and shows diffs against golden rendered output. No terminal needed to see what the UI looks like.

For debugging (since BubbleTea owns stdout):

```go
f, _ := tea.LogToFile("debug.log", "parquetv")
defer f.Close()
log.Println("selected row group:", m.selected)
```

Tail it in a split pane: `tail -f debug.log`.

### Test data

Real Tempo blocks available locally at `/private/tmp/tempo-block/` and `/private/tmp/tempo-block-1026056/`.

| File | Source | Rows | Row Groups | Size | Use |
|---|---|---|---|---|---|
| `testdata/small.parquet` | `tempo-block/c2ff3990../data.parquet` | 17 | 1 | <1 MB | Golden files — full grid fits in a snapshot |
| `testdata/small.meta.json` | Same block's `meta.json` | — | — | — | Dedicated column mapping for test assertions |
| `/private/tmp/tempo-block-1026056/3dd7c298..` | — | 358,864 | 2 | 85 MB | Level 1 testing — the only block with 2 row groups |
| `/private/tmp/tempo-block/fe87871b..` | — | 149,895 | 1 | 42 MB | Manual testing — lazy loading, scrolling, large Attrs.Key |

The 17-row block is committed to the repo as `testdata/`. Larger blocks stay local for manual testing.

All blocks are vParquet4 with 10-20 dedicated columns, real tenant data from tenant `1000087` and `1026056`.

### Golden files as documentation

Golden files serve triple duty:
- **Tests** — CI fails if rendered output changes unintentionally
- **Documentation** — reviewers see exactly what the UI looks like as text diffs in PRs
- **Regression** — visual regressions show up as golden file changes

Update goldens intentionally with `go test -update` when changing the UI.

## 6. Implementation plan

| Phase | Scope | Level | Effort | Status |
|---|---|---|---|---|
| 0+1 | Scaffold + file overview: open file, row group list, footer panel, top columns table, chrome bars | L1 | 2 days | **Done** |
| 2 | Row group grid: row × column table, horizontal scroll, column stats bar, enter/esc navigation | L2 | 2 days | Next |
| 3 | Page inspector: page list + value viewer for trace-level columns | L3 | 2 days | |
| 4 | Column filters with page skip + dict shortcut + cost reporting | L2 | 2-3 days | |
| 5 | Def/rep levels + nesting resolver for nested columns | L3 | 2-3 days | |
| 6 | Row expansion (nested tree inline) | L2 | 1 day | |
| 7 | Generic attribute KV filter (key scan + value lookup) | L2 | 1-2 days | |
| 8 | Dictionary view, predicate simulation | L3 | 1-2 days | |
| 9 | Schema tree side panel | L1 | 1 day | |
| 10 | Content/inspect panel (hex, binary, decoded, encoding details) | L2/L3 | 1-2 days | |
| 11 | Object storage reader (S3/GCS) | - | 1-2 days | |

**Total: ~16-21 days** for a complete tool. Phases 0-3 produce a usable MVP covering all three levels for simple columns. Phase 4 adds filtering.

## 7. Implementation rules

Hard-won rules from Phase 0+1. Follow these in every phase.

### lipgloss

- **Inherit(parentStyle)** on every inner styled element when composing inside a background-colored container. `Render()` emits ANSI reset sequences that punch holes in the parent's background otherwise.
- **Never join pre-rendered ANSI strings with plain spaces** for gap filling. The spaces won't have the background color. Use a styled `Width()` gap element or measure with `len()` on plain text first.
- **Use `lipgloss/table`** for any aligned columnar data. Manual space padding breaks on varying terminal widths.
- **Use `lipgloss.JoinVertical`/`JoinHorizontal`** for composition, not string concatenation with `\n`.
- **Use style `.Width()`, `.Height()`, `.Padding()`, `.Align()`** for layout, not manual spaces.

### Architecture

- **Import graph**: `model → ui ← view`. View models live in `ui/` package. `view/` never imports `model/`, `model/` never imports `view/`. `engine/` imports neither.
- **View functions are pure**: `func Render*(vm SomeVM) string`. No state, no side effects, no I/O. All lipgloss usage lives in `view/`.
- **Model.View() just delegates**: builds a ViewModel, passes it to the view layer. No rendering logic in model.
- **Meaningful names**: `FileOverview`, `RowGroupGrid`, `PageInspector`. Not Level1/2/3.

### parquet-go

- **Strip column paths**: parquet-go returns `rs.list.element.ss.list.element.Spans.list.element.Name`. Use `engine.SimplifyPath()` to strip `list` and `element` segments.
- **`ColumnChunk.Column()` returns `int`** (column index), not `*Column`. Get column metadata from `Schema.Columns()` instead.
- **`OffsetIndex()` and `ColumnIndex()` return `(T, error)`** — always handle the error.

### Testing

- **Render tests in `view/`**: golden file comparison with `-update` flag.
- **State tests in `model/`**: navigation, filter state, cursor bounds.
- **Engine tests in `engine/`**: Parquet reading against real test blocks.
- **All use `_test` packages** (black-box, exported API only).
- **Golden files are documentation**: reviewers see exactly what the UI looks like as text diffs.

## 8. Technology choices

| Component | Choice | Rationale |
|---|---|---|
| Language | Go | Same as Tempo, same Parquet library, single binary distribution |
| Parquet library | `parquet-go/parquet-go` | The library Tempo uses. Full low-level access to pages, dictionaries, def/rep levels |
| TUI framework | `charmbracelet/bubbletea` | Mature, well-documented, good component library (bubbles) |
| Styling | `charmbracelet/lipgloss` | Pairs with bubbletea, handles terminal colors and layout |
| Distribution | Single binary via `go install` | No runtime dependencies |

## 9. Risks

| Risk | Mitigation |
|---|---|
| Large files cause slow startup | Lazy loading — only footer is read on open |
| Very wide schemas (68+ columns) don't fit in terminal | Horizontal scroll in RowTable, collapsible columns |
| Nesting resolver is complex for deep schemas | Incremental implementation: trace-level first, then resource, then span |
| parquet-go API changes | Pin version, the API is stable |

## 10. Success criteria

- A Tempo contributor can open a vParquet5 block and understand within 2 minutes why `Attrs.Key` is expensive and dedicated columns help
- An operator can identify which columns dominate a block's size without running `tempo-cli analyse`
- The tool works on any Parquet file, not just Tempo blocks
