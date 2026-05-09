package model_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/javiermolinar/parquetv/engine"
	"github.com/javiermolinar/parquetv/model"
)

func newGridModel(t *testing.T) model.RowGroupGridModel {
	t.Helper()
	f := openTestFile(t)
	m, err := model.NewRowGroupGridModel(f, 0, 120, 30)
	if err != nil {
		t.Fatalf("NewRowGroupGridModel: %v", err)
	}
	return m
}

func sendKey(m tea.Model, key string) tea.Model {
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
	return updated
}

func sendSpecialKey(m tea.Model, key tea.KeyType) tea.Model {
	updated, _ := m.Update(tea.KeyMsg{Type: key})
	return updated
}

func TestRowGroupGridInitialState(t *testing.T) {
	m := newGridModel(t)
	vm := m.BuildViewModel()

	if vm.SelectedRow != 0 {
		t.Errorf("initial SelectedRow = %d, want 0", vm.SelectedRow)
	}
	if vm.SelectedCol != 0 {
		t.Errorf("initial SelectedCol = %d, want 0", vm.SelectedCol)
	}
	if vm.TotalRows != 17 {
		t.Errorf("TotalRows = %d, want 17", vm.TotalRows)
	}
	if vm.TotalCols != 103 {
		t.Errorf("TotalCols = %d, want 103", vm.TotalCols)
	}
	if len(vm.Rows) != 17 {
		t.Errorf("visible Rows = %d, want 17 (all fit at height 30)", len(vm.Rows))
	}
	if len(vm.Headers) == 0 {
		t.Error("no visible headers")
	}
}

func TestRowGroupGridVerticalNav(t *testing.T) {
	m := newGridModel(t)

	// Move down.
	m = sendKey(m, "j").(model.RowGroupGridModel)
	vm := m.BuildViewModel()
	if vm.SelectedRow != 1 {
		t.Errorf("after j, SelectedRow = %d, want 1", vm.SelectedRow)
	}

	// Move up.
	m = sendKey(m, "k").(model.RowGroupGridModel)
	vm = m.BuildViewModel()
	if vm.SelectedRow != 0 {
		t.Errorf("after k, SelectedRow = %d, want 0", vm.SelectedRow)
	}

	// Up at top should stay at 0.
	m = sendKey(m, "k").(model.RowGroupGridModel)
	vm = m.BuildViewModel()
	if vm.SelectedRow != 0 {
		t.Errorf("after k at top, SelectedRow = %d, want 0", vm.SelectedRow)
	}

	// Jump to last.
	m = sendKey(m, "G").(model.RowGroupGridModel)
	vm = m.BuildViewModel()
	if vm.SelectedRow != 16 {
		t.Errorf("after G, SelectedRow = %d, want 16", vm.SelectedRow)
	}

	// Down at bottom should stay.
	m = sendKey(m, "j").(model.RowGroupGridModel)
	vm = m.BuildViewModel()
	if vm.SelectedRow != 16 {
		t.Errorf("after j at bottom, SelectedRow = %d, want 16", vm.SelectedRow)
	}

	// Jump to first.
	m = sendKey(m, "g").(model.RowGroupGridModel)
	vm = m.BuildViewModel()
	if vm.SelectedRow != 0 {
		t.Errorf("after g, SelectedRow = %d, want 0", vm.SelectedRow)
	}
}

func TestRowGroupGridHorizontalNav(t *testing.T) {
	m := newGridModel(t)

	// Move right.
	m = sendKey(m, "l").(model.RowGroupGridModel)
	vm := m.BuildViewModel()
	if vm.SelectedCol != 1 {
		t.Errorf("after l, SelectedCol = %d, want 1", vm.SelectedCol)
	}

	// Move left.
	m = sendKey(m, "h").(model.RowGroupGridModel)
	vm = m.BuildViewModel()
	if vm.SelectedCol != 0 {
		t.Errorf("after h, SelectedCol = %d, want 0", vm.SelectedCol)
	}

	// Left at leftmost should stay.
	m = sendKey(m, "h").(model.RowGroupGridModel)
	vm = m.BuildViewModel()
	if vm.SelectedCol != 0 {
		t.Errorf("after h at left edge, SelectedCol = %d, want 0", vm.SelectedCol)
	}
}

func TestRowGroupGridHorizontalScroll(t *testing.T) {
	m := newGridModel(t)
	vm := m.BuildViewModel()
	initialVisCols := len(vm.Headers)

	// Scroll right past the visible range — headers should shift.
	for i := 0; i < initialVisCols+2; i++ {
		m = sendKey(m, "l").(model.RowGroupGridModel)
	}
	vm = m.BuildViewModel()

	// The first visible header should no longer be TraceID.
	if vm.Headers[0] == "TraceID" {
		t.Error("after scrolling right past visible range, first header should have shifted")
	}
	// Selected col should be within the visible range.
	if vm.SelectedCol < 0 || vm.SelectedCol >= len(vm.Headers) {
		t.Errorf("SelectedCol=%d out of visible range [0, %d)", vm.SelectedCol, len(vm.Headers))
	}
}

func TestRowGroupGridColumnStats(t *testing.T) {
	m := newGridModel(t)
	vm := m.BuildViewModel()

	// Stats should reflect the first column (TraceID).
	if vm.Stats.Path != "TraceID" {
		t.Errorf("initial stats path = %q, want TraceID", vm.Stats.Path)
	}
	if vm.Stats.NumValues == 0 {
		t.Error("stats NumValues = 0")
	}

	// Move to column 5 (RootServiceName) and check stats update.
	for i := 0; i < 5; i++ {
		m = sendKey(m, "l").(model.RowGroupGridModel)
	}
	vm = m.BuildViewModel()
	if vm.Stats.Path != "RootServiceName" {
		t.Errorf("stats at col 5 = %q, want RootServiceName", vm.Stats.Path)
	}
	if vm.Stats.ValuesPerRow < 0.9 || vm.Stats.ValuesPerRow > 1.1 {
		t.Errorf("RootServiceName per-row = %.1f, want ~1.0", vm.Stats.ValuesPerRow)
	}
}

func TestRowGroupGridSpanLevelStats(t *testing.T) {
	m := newGridModel(t)

	// Navigate to a span-level column (e.g. col 47 = rs.ss.Spans.SpanID).
	// per-row should be > 1.0 since it's repeated.
	for i := 0; i < 47; i++ {
		m = sendKey(m, "l").(model.RowGroupGridModel)
	}
	vm := m.BuildViewModel()
	if vm.Stats.Path != "rs.ss.Spans.SpanID" {
		t.Errorf("stats at col 47 = %q, want rs.ss.Spans.SpanID", vm.Stats.Path)
	}
	if vm.Stats.ValuesPerRow <= 1.0 {
		t.Errorf("SpanID per-row = %.1f, expected > 1.0 (span-level column)", vm.Stats.ValuesPerRow)
	}
	t.Logf("SpanID per-row = %.1f", vm.Stats.ValuesPerRow)
}

func TestRowGroupGridEscProducesBackMsg(t *testing.T) {
	f := openTestFile(t)
	m, err := model.NewRowGroupGridModel(f, 0, 120, 30)
	if err != nil {
		t.Fatalf("NewRowGroupGridModel: %v", err)
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("esc should produce a command")
	}
}

func TestRowGroupGridBreadcrumb(t *testing.T) {
	m := newGridModel(t)
	vm := m.BuildViewModel()

	if vm.BottomBar.Breadcrumb != "small.parquet › RG 0" {
		t.Errorf("breadcrumb = %q, want %q", vm.BottomBar.Breadcrumb, "small.parquet › RG 0")
	}
}

func TestRowGroupGridArrowKeys(t *testing.T) {
	m := newGridModel(t)

	m = sendSpecialKey(m, tea.KeyDown).(model.RowGroupGridModel)
	vm := m.BuildViewModel()
	if vm.SelectedRow != 1 {
		t.Errorf("after down, SelectedRow = %d, want 1", vm.SelectedRow)
	}

	m = sendSpecialKey(m, tea.KeyUp).(model.RowGroupGridModel)
	vm = m.BuildViewModel()
	if vm.SelectedRow != 0 {
		t.Errorf("after up, SelectedRow = %d, want 0", vm.SelectedRow)
	}

	m = sendSpecialKey(m, tea.KeyRight).(model.RowGroupGridModel)
	vm = m.BuildViewModel()
	if vm.SelectedCol != 1 {
		t.Errorf("after right, SelectedCol = %d, want 1", vm.SelectedCol)
	}

	m = sendSpecialKey(m, tea.KeyLeft).(model.RowGroupGridModel)
	vm = m.BuildViewModel()
	if vm.SelectedCol != 0 {
		t.Errorf("after left, SelectedCol = %d, want 0", vm.SelectedCol)
	}
}

func TestAppEnterRowGroup(t *testing.T) {
	f, err := engine.Open("../testdata/small.parquet")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { f.Close() })

	app := model.NewApp(f)
	updatedModel, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	app = updatedModel.(model.App)

	// Press enter to open row group 0.
	updated, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = updated.(model.App)

	if cmd == nil {
		t.Fatal("enter should produce a command")
	}

	// Execute the command to get the enterRowGroupMsg.
	msg := cmd()
	updated, _ = app.Update(msg)
	app = updated.(model.App)

	// App should now render the grid view.
	output := app.View()
	if output == "" {
		t.Error("grid view is empty")
	}
	if !containsStr(output, "RG 0") {
		t.Error("grid view should contain 'RG 0' breadcrumb")
	}
}

func TestAppEnterAndEsc(t *testing.T) {
	f, err := engine.Open("../testdata/small.parquet")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { f.Close() })

	app := model.NewApp(f)
	updatedModel, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	app = updatedModel.(model.App)

	// Enter grid.
	updated, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = updated.(model.App)
	msg := cmd()
	updated, _ = app.Update(msg)
	app = updated.(model.App)

	// Should be in grid view.
	gridOutput := app.View()
	if !containsStr(gridOutput, "RG 0") {
		t.Fatal("expected grid view")
	}

	// Press esc to go back.
	updated, cmd = app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	app = updated.(model.App)
	if cmd != nil {
		msg = cmd()
		updated, _ = app.Update(msg)
		app = updated.(model.App)
	}

	// Should be back to overview.
	overviewOutput := app.View()
	if !containsStr(overviewOutput, "Row Groups") {
		t.Error("expected file overview after esc")
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
