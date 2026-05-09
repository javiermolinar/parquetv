package model_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/javiermolinar/parquetv/engine"
	"github.com/javiermolinar/parquetv/model"
)

func newPageInspectorModel(t *testing.T, colIndex int) model.PageInspectorModel {
	t.Helper()
	f := openTestFile(t)
	reader, err := f.NewRowGroupReader(0)
	if err != nil {
		t.Fatalf("NewRowGroupReader: %v", err)
	}
	m, err := model.NewPageInspectorModel(f, reader, 0, colIndex, 120, 30)
	if err != nil {
		t.Fatalf("NewPageInspectorModel: %v", err)
	}
	return m
}

func TestPageInspectorInitialState(t *testing.T) {
	m := newPageInspectorModel(t, 0)
	vm := m.BuildViewModel()

	if vm.ColumnPath != "TraceID" {
		t.Errorf("ColumnPath = %q, want TraceID", vm.ColumnPath)
	}
	if vm.SelectedPage != 0 {
		t.Errorf("SelectedPage = %d, want 0", vm.SelectedPage)
	}
	if len(vm.Pages) == 0 {
		t.Error("no pages in view model")
	}
	if vm.ViewingValues {
		t.Error("should not be viewing values initially")
	}
	if len(vm.Values) == 0 {
		t.Error("should have preview values loaded")
	}
	if vm.Encoding == "" {
		t.Error("encoding is empty")
	}
	if vm.Compression == "" {
		t.Error("compression is empty")
	}
}

func TestPageInspectorPageNavigation(t *testing.T) {
	// Use DurationNano (col 4) which should have 1 page in small file.
	m := newPageInspectorModel(t, 4)
	vm := m.BuildViewModel()

	// Only 1 page in small file — down should not move.
	m = sendKey(m, "j").(model.PageInspectorModel)
	vm = m.BuildViewModel()
	if vm.SelectedPage != 0 {
		t.Errorf("after j with 1 page, SelectedPage = %d, want 0", vm.SelectedPage)
	}

	// Up from 0 should stay.
	m = sendKey(m, "k").(model.PageInspectorModel)
	vm = m.BuildViewModel()
	if vm.SelectedPage != 0 {
		t.Errorf("after k at page 0, SelectedPage = %d, want 0", vm.SelectedPage)
	}
}

func TestPageInspectorEnterValueViewer(t *testing.T) {
	m := newPageInspectorModel(t, 0)

	// Press enter to go into value viewer mode.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model.PageInspectorModel)
	vm := m.BuildViewModel()

	if !vm.ViewingValues {
		t.Error("should be viewing values after enter")
	}
	if len(vm.Values) == 0 {
		t.Error("no values in viewer")
	}

	// Breadcrumb should include page number.
	if !containsStr(vm.BottomBar.Breadcrumb, "Page 0") {
		t.Errorf("breadcrumb = %q, should contain 'Page 0'", vm.BottomBar.Breadcrumb)
	}

	// Esc should go back to page list mode.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = updated.(model.PageInspectorModel)
	vm = m.BuildViewModel()

	if vm.ViewingValues {
		t.Error("should not be viewing values after esc")
	}
}

func TestPageInspectorEscToGrid(t *testing.T) {
	m := newPageInspectorModel(t, 0)

	// Esc from page list should go back to grid.
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("esc should produce a command")
	}
}

func TestPageInspectorColumnSwitch(t *testing.T) {
	// Start at col 1 (TraceIDText).
	m := newPageInspectorModel(t, 1)
	vm := m.BuildViewModel()

	if vm.ColumnPath != "TraceIDText" {
		t.Errorf("ColumnPath = %q, want TraceIDText", vm.ColumnPath)
	}

	// Press left — should produce switchColumnMsg for col 0.
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if cmd == nil {
		t.Fatal("left should produce a command for column switch")
	}

	// Press right — should produce switchColumnMsg for col 2.
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	if cmd == nil {
		t.Fatal("right should produce a command for column switch")
	}
}

func TestPageInspectorLeftEdge(t *testing.T) {
	// Start at col 0 — left should not produce a command.
	m := newPageInspectorModel(t, 0)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if cmd != nil {
		t.Error("left at col 0 should not produce a command")
	}
}

func TestPageInspectorTopBarContext(t *testing.T) {
	m := newPageInspectorModel(t, 5)
	vm := m.BuildViewModel()

	// Top bar context should contain column info.
	if !containsStr(vm.TopBar.Context, "RootServiceName") {
		t.Errorf("top bar context = %q, should contain column name", vm.TopBar.Context)
	}
}

func TestPageInspectorBreadcrumb(t *testing.T) {
	m := newPageInspectorModel(t, 0)
	vm := m.BuildViewModel()

	expected := "small.parquet › RG 0 › TraceID"
	if vm.BottomBar.Breadcrumb != expected {
		t.Errorf("breadcrumb = %q, want %q", vm.BottomBar.Breadcrumb, expected)
	}
}

func TestAppGridToPageInspector(t *testing.T) {
	f, err := engine.Open("../testdata/small.parquet")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { f.Close() })

	app := model.NewApp(f)
	updatedModel, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	app = updatedModel.(model.App)

	// Enter grid (Level 1).
	updated, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = updated.(model.App)
	msg := cmd()
	updated, _ = app.Update(msg)
	app = updated.(model.App)

	// Press enter to open page inspector (Level 3).
	updated, cmd = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = updated.(model.App)
	if cmd == nil {
		t.Fatal("enter in grid should produce command")
	}
	msg = cmd()
	updated, _ = app.Update(msg)
	app = updated.(model.App)

	// Should render page inspector.
	output := app.View()
	if !containsStr(output, "Page 0") {
		t.Error("page inspector should contain 'Page 0'")
	}
	if !containsStr(output, "TraceID") {
		t.Error("page inspector should show TraceID column")
	}

	// Press esc to go back to grid.
	updated, cmd = app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	app = updated.(model.App)
	if cmd != nil {
		msg = cmd()
		updated, _ = app.Update(msg)
		app = updated.(model.App)
	}

	// Should be back in grid.
	output = app.View()
	if !containsStr(output, "RG 0") {
		t.Error("should be back in grid after esc")
	}
}
