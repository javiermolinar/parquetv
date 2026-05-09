package view_test

import (
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/javiermolinar/parquetv/engine"
	"github.com/javiermolinar/parquetv/model"
	"github.com/javiermolinar/parquetv/view"
)

var update = flag.Bool("update", false, "update golden files")

func openTestFile(t *testing.T) *engine.File {
	t.Helper()
	f, err := engine.Open("../testdata/small.parquet")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

func TestFileOverviewRender(t *testing.T) {
	f := openTestFile(t)
	m := model.NewFileOverviewModel(f)
	m.SetSize(100, 30)

	vm := m.BuildViewModel()
	got := view.RenderFileOverview(vm)

	golden := "../testdata/fileoverview.golden"
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
