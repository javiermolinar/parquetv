# parquetv Agent Guide

Interactive Parquet file explorer TUI built with Go, BubbleTea, and lipgloss.

## Key files

- `DESIGN.md` — full design doc with UX mockups, architecture, keybindings
- `PLAN.md` — implementation roadmap with completed/pending phases
- `DESIGN.md` section 7 — **mandatory** implementation rules (lipgloss gotchas, import graph, testing)

## Architecture

```
model → ui ← view       ViewModel structs live in ui/
model → engine           data access
view  → ui               render from ViewModels
engine → nothing          pure Parquet reading
```

- **view/** — pure functions: `func Render*(vm SomeVM) string`. No state, no I/O. All lipgloss here.
- **model/** — BubbleTea models. Owns state, handles Update(msg), builds ViewModels.
- **engine/** — Parquet reading. No BubbleTea, no lipgloss. Testable without a terminal.
- **ui/** — ViewModel structs only. The contract between model and view.

## lipgloss rules

These cause silent rendering bugs if violated:

1. **`Inherit(parentStyle)`** on every inner element inside a background-colored container. `Render()` emits ANSI resets that punch holes in the parent background.
2. **Never join ANSI strings with plain spaces** for gaps. Spaces won't carry the background. Use styled `Width()` elements.
3. **Use `lipgloss/table`** for aligned columnar data. Manual padding breaks on varying widths.
4. **Use `JoinVertical`/`JoinHorizontal`** for composition, not `\n` concatenation.
5. **Use `.Width()`, `.PaddingLeft()`, `.Align()`** for layout, not manual spaces.
6. **`PaddingLeft` is included in `Width`**. `Width(18).PaddingLeft(2)` = 16 content chars + 2 padding.

## Testing

```
make test                    # all tests
make dev                     # file-watching test loop
go test ./view/ -update      # regenerate golden files
```

- **view/** — golden file comparison with `-update` flag. Render tests only.
- **model/** — navigation, state, cursor bounds. No rendering.
- **engine/** — Parquet reading against real test files in `testdata/`.
- All tests use `_test` packages (black-box, exported API only).
- Golden files are documentation — reviewers see exactly what the UI looks like.

## TUI visual testing

Use VHS to capture screenshots without a terminal:

```bash
# Quick capture
bash ~/.agents/skills/tui-preview/tui-capture.sh -w 1400 -h 700 -s 3 "/tmp/parquetv testdata/small.parquet"

# With keystrokes (enter grid, navigate)
# Use a custom .tape file for multi-step interactions — see tui-preview skill.
```

Always build first: `go build -o /tmp/parquetv .`

VHS timing matters — use `Sleep 2s` between launch and first Enter, `Sleep 1s` between navigation steps.

## Test data

| File | Rows | RGs | Use |
|------|------|-----|-----|
| `testdata/small.parquet` | 17 | 1 | Golden files — full grid fits |
| `/private/tmp/tempo-block-1026056/3dd7c298..` | 358,864 | 2 | Multi-RG, virtual scroll |
| `/private/tmp/tempo-block/fe87871b..` | 149,895 | 1 | Large single-RG |

## Common patterns

### Adding a new screen/level

1. Engine: add data reading methods to `engine/` (no TUI imports)
2. Types: add ViewModel struct to `ui/viewmodel.go`
3. Model: new model file in `model/` with `Init/Update/View/BuildViewModel`
4. View: new render function in `view/` — pure, takes VM returns string
5. Navigation: add message types + routing in `model/app.go`
6. Tests: golden file in `view/`, navigation in `model/`, data in `engine/`

### Adding a keybinding

1. Handle in the model's `Update(tea.KeyMsg)` switch
2. Update `BottomBar.Shortcuts` in `BuildViewModel` (contextual hints)
3. Test: send `tea.KeyMsg` in model test, check state change

### Modifying the grid layout

1. Change view rendering in `view/rowgroupgrid.go`
2. If chrome height changes, update `gridChromeFixed` or `inspectHeight()` in model
3. Regenerate golden files: `go test ./view/ -update`
4. VHS capture to verify visual result
