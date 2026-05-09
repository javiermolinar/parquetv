package view_test

import (
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/javiermolinar/parquetv/model"
	"github.com/javiermolinar/parquetv/view"
)

func keyMsg(key string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
}

func TestRowGroupGridRender(t *testing.T) {
	f := openTestFile(t)
	m, err := model.NewRowGroupGridModel(f, 0, 120, 30)
	if err != nil {
		t.Fatalf("NewRowGroupGridModel: %v", err)
	}

	vm := m.BuildViewModel()
	got := view.RenderRowGroupGrid(vm)

	golden := "../testdata/rowgroupgrid.golden"
	if *update {
		os.WriteFile(golden, []byte(got), 0644)
		t.Log("updated golden file")
		return
	}

	want, err := os.ReadFile(golden)
	if err != nil {
		os.WriteFile(golden, []byte(got), 0644)
		t.Log("created golden file (first run)")
		return
	}

	if got != string(want) {
		gotLines := strings.Split(got, "\n")
		wantLines := strings.Split(string(want), "\n")
		for i := 0; i < len(gotLines) || i < len(wantLines); i++ {
			g, w := "", ""
			if i < len(gotLines) {
				g = gotLines[i]
			}
			if i < len(wantLines) {
				w = wantLines[i]
			}
			if g != w {
				t.Errorf("line %d:\n  got:  %q\n  want: %q", i+1, g, w)
			}
		}
	}
}

func TestRowGroupGridRenderWithColScroll(t *testing.T) {
	f := openTestFile(t)
	m, err := model.NewRowGroupGridModel(f, 0, 120, 30)
	if err != nil {
		t.Fatalf("NewRowGroupGridModel: %v", err)
	}

	// Scroll right to column 5 (RootServiceName).
	for i := 0; i < 5; i++ {
		updated, _ := m.Update(keyMsg("l"))
		m = updated.(model.RowGroupGridModel)
	}

	vm := m.BuildViewModel()
	got := view.RenderRowGroupGrid(vm)

	golden := "../testdata/rowgroupgrid_colscroll.golden"
	if *update {
		os.WriteFile(golden, []byte(got), 0644)
		t.Log("updated golden file")
		return
	}

	want, err := os.ReadFile(golden)
	if err != nil {
		os.WriteFile(golden, []byte(got), 0644)
		t.Log("created golden file (first run)")
		return
	}

	if got != string(want) {
		gotLines := strings.Split(got, "\n")
		wantLines := strings.Split(string(want), "\n")
		for i := 0; i < len(gotLines) || i < len(wantLines); i++ {
			g, w := "", ""
			if i < len(gotLines) {
				g = gotLines[i]
			}
			if i < len(wantLines) {
				w = wantLines[i]
			}
			if g != w {
				t.Errorf("line %d:\n  got:  %q\n  want: %q", i+1, g, w)
			}
		}
	}
}

func TestRowGroupGridRenderWithSelection(t *testing.T) {
	f := openTestFile(t)
	m, err := model.NewRowGroupGridModel(f, 0, 120, 30)
	if err != nil {
		t.Fatalf("NewRowGroupGridModel: %v", err)
	}

	// Move cursor to row 3, col 1.
	for i := 0; i < 3; i++ {
		updated, _ := m.Update(keyMsg("j"))
		m = updated.(model.RowGroupGridModel)
	}
	updated, _ := m.Update(keyMsg("l"))
	m = updated.(model.RowGroupGridModel)

	vm := m.BuildViewModel()
	got := view.RenderRowGroupGrid(vm)

	golden := "../testdata/rowgroupgrid_selected.golden"
	if *update {
		os.WriteFile(golden, []byte(got), 0644)
		t.Log("updated golden file")
		return
	}

	want, err := os.ReadFile(golden)
	if err != nil {
		os.WriteFile(golden, []byte(got), 0644)
		t.Log("created golden file (first run)")
		return
	}

	if got != string(want) {
		gotLines := strings.Split(got, "\n")
		wantLines := strings.Split(string(want), "\n")
		for i := 0; i < len(gotLines) || i < len(wantLines); i++ {
			g, w := "", ""
			if i < len(gotLines) {
				g = gotLines[i]
			}
			if i < len(wantLines) {
				w = wantLines[i]
			}
			if g != w {
				t.Errorf("line %d:\n  got:  %q\n  want: %q", i+1, g, w)
			}
		}
	}
}
