package model_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/javiermolinar/parquetv/engine"
	"github.com/javiermolinar/parquetv/model"
)

func openTestFile(t *testing.T) *engine.File {
	t.Helper()
	f, err := engine.Open("../testdata/small.parquet")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

func TestFileOverviewNavigation(t *testing.T) {
	f := openTestFile(t)
	m := model.NewFileOverviewModel(f)
	m.SetSize(100, 30)

	// Should start at row group 0.
	vm := m.BuildViewModel()
	if vm.Selected != 0 {
		t.Errorf("initial selected = %d, want 0", vm.Selected)
	}

	// Down should not go below 0 (only 1 row group in small.parquet).
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(model.FileOverviewModel)
	vm = m.BuildViewModel()
	if vm.Selected != 0 {
		t.Errorf("after down, selected = %d, want 0 (single RG)", vm.Selected)
	}

	// Up from 0 should stay at 0.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(model.FileOverviewModel)
	vm = m.BuildViewModel()
	if vm.Selected != 0 {
		t.Errorf("after up from 0, selected = %d, want 0", vm.Selected)
	}
}
