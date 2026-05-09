package view_test

import (
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/javiermolinar/parquetv/model"
	"github.com/javiermolinar/parquetv/view"
)

func newPageInspector(t *testing.T, colIndex int) model.PageInspectorModel {
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

func TestPageInspectorRender(t *testing.T) {
	m := newPageInspector(t, 0)
	vm := m.BuildViewModel()
	got := view.RenderPageInspector(vm)

	golden := "../testdata/pageinspector.golden"
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

func TestPageInspectorRenderValueViewer(t *testing.T) {
	m := newPageInspector(t, 5) // RootServiceName

	// Enter value viewer mode.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model.PageInspectorModel)

	vm := m.BuildViewModel()
	got := view.RenderPageInspector(vm)

	golden := "../testdata/pageinspector_values.golden"
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
