package engine_test

import (
	"testing"

	"github.com/javiermolinar/parquetv/engine"
)

func TestRowGroupReader(t *testing.T) {
	f, err := engine.Open("../testdata/small.parquet")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer f.Close()

	reader, err := f.NewRowGroupReader(0)
	if err != nil {
		t.Fatalf("NewRowGroupReader: %v", err)
	}

	// Headers should match the number of columns.
	headers := reader.Headers()
	info := f.Info()
	if len(headers) != info.NumColumns {
		t.Errorf("headers count = %d, want %d", len(headers), info.NumColumns)
	}

	// First few columns should have known paths.
	if headers[0].Path != "TraceID" {
		t.Errorf("col 0 path = %q, want TraceID", headers[0].Path)
	}
	if headers[1].Path != "TraceIDText" {
		t.Errorf("col 1 path = %q, want TraceIDText", headers[1].Path)
	}
	if headers[5].Path != "RootServiceName" {
		t.Errorf("col 5 path = %q, want RootServiceName", headers[5].Path)
	}

	// NumRows should match.
	if reader.NumRows() != info.RowGroups[0].NumRows {
		t.Errorf("NumRows = %d, want %d", reader.NumRows(), info.RowGroups[0].NumRows)
	}
}

func TestReadRows(t *testing.T) {
	f, err := engine.Open("../testdata/small.parquet")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer f.Close()

	reader, err := f.NewRowGroupReader(0)
	if err != nil {
		t.Fatalf("NewRowGroupReader: %v", err)
	}

	// Read first 3 rows.
	rows, err := reader.ReadRows(0, 3)
	if err != nil {
		t.Fatalf("ReadRows: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows))
	}

	numCols := len(reader.Headers())
	for i, row := range rows {
		if len(row) != numCols {
			t.Errorf("row %d: got %d cols, want %d", i, len(row), numCols)
		}
	}

	// TraceIDText (col 1) should be a non-empty hex-like string.
	if rows[0][1] == "" {
		t.Error("row 0 TraceIDText is empty")
	}
	t.Logf("row 0 TraceIDText = %s", rows[0][1])
	t.Logf("row 0 DurationNano = %s", rows[0][4])
	t.Logf("row 0 RootServiceName = %s", rows[0][5])

	// Read all rows.
	allRows, err := reader.ReadRows(0, reader.NumRows())
	if err != nil {
		t.Fatalf("ReadRows all: %v", err)
	}
	if int64(len(allRows)) != reader.NumRows() {
		t.Errorf("got %d rows, want %d", len(allRows), reader.NumRows())
	}

	// Read with offset.
	offsetRows, err := reader.ReadRows(10, 5)
	if err != nil {
		t.Fatalf("ReadRows offset: %v", err)
	}
	if len(offsetRows) != 5 {
		t.Fatalf("got %d rows, want 5", len(offsetRows))
	}
	// Row at offset 10 should match allRows[10].
	if offsetRows[0][1] != allRows[10][1] {
		t.Errorf("offset row 0 TraceIDText = %q, want %q", offsetRows[0][1], allRows[10][1])
	}
}

func TestPageBoundaries(t *testing.T) {
	f, err := engine.Open("../testdata/small.parquet")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer f.Close()

	reader, err := f.NewRowGroupReader(0)
	if err != nil {
		t.Fatalf("NewRowGroupReader: %v", err)
	}

	// Small file should have 1 page per column → boundary at row 0.
	bounds, err := reader.PageBoundaries(0)
	if err != nil {
		t.Fatalf("PageBoundaries: %v", err)
	}
	if len(bounds) == 0 {
		t.Fatal("expected at least 1 page boundary")
	}
	if bounds[0] != 0 {
		t.Errorf("first page boundary = %d, want 0", bounds[0])
	}
	t.Logf("col 0 page boundaries: %v", bounds)
}

func TestNewRowGroupReaderOutOfRange(t *testing.T) {
	f, err := engine.Open("../testdata/small.parquet")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer f.Close()

	_, err = f.NewRowGroupReader(5)
	if err == nil {
		t.Error("expected error for out-of-range row group")
	}
}
