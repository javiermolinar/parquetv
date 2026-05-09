package engine

import (
	"fmt"
	"testing"
)

func TestOpenSmallBlock(t *testing.T) {
	f, err := Open("../testdata/small.parquet")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer f.Close()

	info := f.Info()

	if info.NumRows == 0 {
		t.Error("expected rows > 0")
	}
	t.Logf("rows=%d rowGroups=%d columns=%d size=%d",
		info.NumRows, info.NumRowGroups, info.NumColumns, info.Size)

	if info.NumRowGroups == 0 {
		t.Error("expected row groups > 0")
	}
	if info.NumColumns == 0 {
		t.Error("expected columns > 0")
	}

	// Schema tree should exist.
	if info.Schema == nil {
		t.Fatal("schema is nil")
	}
	if len(info.Schema.Children) == 0 {
		t.Error("schema has no children")
	}
	t.Logf("schema root has %d children", len(info.Schema.Children))

	// Row group info.
	for _, rg := range info.RowGroups {
		t.Logf("  RG %d: rows=%d cols=%d bytes=%d",
			rg.Index, rg.NumRows, rg.NumColumns, rg.TotalBytes)
	}

	// Top columns (simplified paths).
	for _, c := range info.TopColumns {
		t.Logf("  top col: %-45s %s", c.Path, formatTestBytes(c.TotalBytes))
	}
}

func TestSimplifyPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"rs.list.element.ss.list.element.Spans.list.element.Name", "rs.ss.Spans.Name"},
		{"TraceID", "TraceID"},
		{"rs.list.element.Resource.ServiceName", "rs.Resource.ServiceName"},
		{"rs.list.element.ss.list.element.Spans.list.element.Attrs.list.element.Key", "rs.ss.Spans.Attrs.Key"},
	}
	for _, tt := range tests {
		got := SimplifyPath(tt.input)
		if got != tt.want {
			t.Errorf("SimplifyPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func formatTestBytes(b int64) string {
	switch {
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
