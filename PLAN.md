# parquetv — Implementation Plan

Interactive TUI for exploring Parquet files. Shows both the logical (rows) and
physical (columns, pages, encodings) views linked together.

Design doc: `tempo-brain/sources/design/parquetv-tui-explorer.md`

## Completed

### Phase 0+1 — Scaffold + File Overview (407ee4a)
- [x] engine/ reads Parquet footer, builds schema tree, column stats
- [x] model/ FileOverviewModel with j/k/g/G navigation
- [x] view/ pure render functions, lipgloss/table
- [x] ui/ shared ViewModel structs (breaks import cycle)
- [x] Two-panel layout: row group list + footer panel (top columns, KV metadata)
- [x] Chrome: top bar (file stats), bottom bar (breadcrumb + shortcuts)
- [x] Golden file tests in view/, navigation tests in model/

### Phase 2 — Row Group Grid (873c6f1..cd3475b)
- [x] Rows × columns grid with j/k rows, h/l columns, g/G/ctrl+d/u
- [x] Virtual scrolling (13ms seek on 358K rows)
- [x] Selected column `[brackets]`, cursor cell accent, row highlight
- [x] Column stats bar: type, size, values, pages, per-row ratio
- [x] Page boundaries as dashed lines between rows
- [x] enter/esc navigation between Level 1 ↔ Level 2
- [x] enter/r/f stubs wired for future phases
- [x] Always-visible inspect panel: full value + hex dump + byte count
- [x] Tab-to-expand for repeated values (lazygit-style scrollable list)
- [x] ×N prefix badge on repeated cells for discoverability
- [x] Right-aligned numeric columns, tighter max column width (18)

### Phase 3 — Page Inspector
Press enter on a column → show pages + values for that column chunk.

- [x] engine: read page headers (min/max, size, encoding, compression)
- [x] engine: read page values via ReadPageValues (offset/limit)
- [x] model: PageInspectorModel with page list + value viewer modes
- [x] view: two-panel layout (page list left, values right)
- [x] Page stats: min/max, compressed size, encoding, compression
- [x] ◂▸ to switch column chunks within the same row group
- [x] esc → back to grid (two-level: esc from values → page list, esc from pages → grid)
- [x] enter → value viewer mode with scroll controls (j/k/g/G/ctrl+d/u)
- [x] d/f key stubs wired for future phases (dictionary + predicate simulation)
- [x] Paginated page list with scroll indicator for large files (11+ pages)
- [x] Golden file tests (page inspector + value viewer)
- [x] Model tests (navigation, column switch, enter/esc, breadcrumb)
- [x] Engine tests (page metadata, page values, offset/range validation)
- [x] VHS visual verification on 358K-row multi-page file
- [x] Hex editor style value viewer: decoded value + spaced hex side by side
- [x] Cursor highlight in value viewer (j/k moves, accent+background row)
- [x] Persistent left navigation panel (38 chars) across all three levels:
  - Level 1: row group list (standardized width)
  - Level 2: RG context + cursor row/col + column stats
  - Level 3: hierarchy header (RG → Column) + page list

### Phase 9 — Schema Tree (2183d2b)
- [x] Always-visible schema tree in Level 1 right panel
- [x] Strip list/element wrapper nodes (clean Parquet nesting)
- [x] Group nodes (▼) in accent color, leaf nodes with types
- [x] Height-capped to remaining panel space with ... truncation
- [x] No toggle — fills available space below top columns and KV metadata

### Phase 8 — Dictionary View (7d29059)
- [x] d key in page inspector opens dictionary overlay
- [x] Shows entries sorted by frequency: value, count, %, visual bar
- [x] Cursor navigation (j/k/g/G/ctrl+d/u), esc to close
- [x] Lazy loaded on first d press, cached for column
- [x] Silently ignored for non-dict-encoded columns
- [x] Engine: ReadDictionary with frequency counting across all pages
- [x] Verified on 337K-row file (48 RootServiceName entries)

## Next

### Phase 4 — Column Filters
Per-column filters with page skip + dict shortcut + cost reporting.

- [ ] Filter input overlay (f key): operator + value
- [ ] Operators: = != > >= < <= ~(regex)
- [ ] Duration shorthand: 1s, 500ms, 2m → nanoseconds
- [ ] Page-level skip using min/max stats
- [ ] Dictionary shortcut for string columns
- [ ] Value scan for non-skipped pages
- [ ] Filter pills below header, match count in breadcrumb
- [ ] Cost reporting: pages skipped, values scanned, matches
- [ ] c to clear all filters, x to remove one

### Phase 5 — Def/Rep Levels + Nesting Resolver
Decode nested structure context for span-level columns.

- [ ] Nesting resolver: def/rep levels → human-readable context
- [ ] Value viewer shows: value, def, rep, context columns
- [ ] Legend explaining level meanings
- [ ] Cross-reference TraceID for trace context

### Phase 6 — Row Expansion
Press r on a row → inline nested tree (ResourceSpans → Spans → Attrs).

- [ ] Expand/collapse inline within the grid
- [ ] Tree rendering for nested structure

### Phase 7 — Generic Attribute KV Filter
Two-field filter for Attrs.Key + Value with cost breakdown.

- [ ] Key scan + value lookup at matched positions
- [ ] Cost reporting showing the full key scan cost

### Phase 8b — Predicate Simulation
- [ ] f in Level 3: simulate pushdown, annotate pages SKIP/READ

### Phase 10 — Content/Inspect Panel (Level 3)
- [ ] i key: detailed hex, binary, decoded, encoding details
- [ ] Adapts per value type (see design doc section 3.3)

### Phase 11 — Object Storage Reader
- [ ] Read blocks directly from S3/GCS via URL

## Architecture

```
model → ui ← view       (ViewModel structs in ui/)
model → engine           (data access)
view → ui                (render from ViewModels)
engine → nothing         (pure Parquet reading)
```

## Test Data

| File | Rows | RGs | Use |
|------|------|-----|-----|
| `testdata/small.parquet` | 17 | 1 | Golden files |
| `/private/tmp/tempo-block-1026056/3dd7c298../data.parquet` | 358,864 | 2 | Multi-RG testing |
| `/private/tmp/tempo-block/fe87871b../data.parquet` | 149,895 | 1 | Large single-RG |

## Commands

```
make run          # small test block
make test         # all tests
make dev          # file-watching test loop
go test ./view/ -update   # regenerate golden files
```
