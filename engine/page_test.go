package engine_test

import (
	"testing"

	"github.com/javiermolinar/parquetv/engine"
)

func TestReadColumnChunkDetail(t *testing.T) {
	f, err := engine.Open("../testdata/small.parquet")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer f.Close()

	reader, err := f.NewRowGroupReader(0)
	if err != nil {
		t.Fatalf("NewRowGroupReader: %v", err)
	}

	// Test TraceID column (col 0).
	detail, err := reader.ReadColumnChunkDetail(0)
	if err != nil {
		t.Fatalf("ReadColumnChunkDetail(0): %v", err)
	}

	if detail.Path != "TraceID" {
		t.Errorf("path = %q, want TraceID", detail.Path)
	}
	if detail.Type == "" {
		t.Error("type is empty")
	}
	if detail.Encoding == "" {
		t.Error("encoding is empty")
	}
	if detail.Compression == "" {
		t.Error("compression is empty")
	}
	if detail.NumValues == 0 {
		t.Error("numValues = 0")
	}
	if len(detail.Pages) == 0 {
		t.Fatal("no pages")
	}

	t.Logf("TraceID: type=%s enc=%s comp=%s values=%d compressed=%d uncompressed=%d pages=%d",
		detail.Type, detail.Encoding, detail.Compression, detail.NumValues,
		detail.TotalCompressed, detail.TotalUncompressed, len(detail.Pages))

	// Small file: 17 rows, 1 RG, 1 page per column.
	if len(detail.Pages) != 1 {
		t.Errorf("pages = %d, expected 1 for small file", len(detail.Pages))
	}

	page := detail.Pages[0]
	if page.FirstRowIndex != 0 {
		t.Errorf("page 0 firstRowIndex = %d, want 0", page.FirstRowIndex)
	}
	if page.NumValues != 17 {
		t.Errorf("page 0 numValues = %d, want 17", page.NumValues)
	}
	if page.CompressedSize <= 0 {
		t.Errorf("page 0 compressedSize = %d, expected > 0", page.CompressedSize)
	}

	t.Logf("  Page 0: rows=%d min=%q max=%q size=%d",
		page.NumValues, page.MinValue, page.MaxValue, page.CompressedSize)
}

func TestReadColumnChunkDetailOutOfRange(t *testing.T) {
	f, err := engine.Open("../testdata/small.parquet")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer f.Close()

	reader, err := f.NewRowGroupReader(0)
	if err != nil {
		t.Fatalf("NewRowGroupReader: %v", err)
	}

	_, err = reader.ReadColumnChunkDetail(-1)
	if err == nil {
		t.Error("expected error for negative column index")
	}

	_, err = reader.ReadColumnChunkDetail(999)
	if err == nil {
		t.Error("expected error for out-of-range column index")
	}
}

func TestReadColumnChunkDetailStringColumn(t *testing.T) {
	f, err := engine.Open("../testdata/small.parquet")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer f.Close()

	reader, err := f.NewRowGroupReader(0)
	if err != nil {
		t.Fatalf("NewRowGroupReader: %v", err)
	}

	// RootServiceName (col 5) — string column with dict encoding.
	detail, err := reader.ReadColumnChunkDetail(5)
	if err != nil {
		t.Fatalf("ReadColumnChunkDetail(5): %v", err)
	}

	if detail.Path != "RootServiceName" {
		t.Errorf("path = %q, want RootServiceName", detail.Path)
	}

	t.Logf("RootServiceName: type=%s enc=%s comp=%s values=%d pages=%d",
		detail.Type, detail.Encoding, detail.Compression,
		detail.NumValues, len(detail.Pages))

	// Should have min/max for string column.
	if len(detail.Pages) > 0 {
		p := detail.Pages[0]
		t.Logf("  Page 0: rows=%d min=%q max=%q", p.NumValues, p.MinValue, p.MaxValue)
	}
}

func TestReadPageValues(t *testing.T) {
	f, err := engine.Open("../testdata/small.parquet")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer f.Close()

	reader, err := f.NewRowGroupReader(0)
	if err != nil {
		t.Fatalf("NewRowGroupReader: %v", err)
	}

	// Read values from TraceID column, page 0.
	vals, total, err := reader.ReadPageValues(0, 0, 0, 5)
	if err != nil {
		t.Fatalf("ReadPageValues: %v", err)
	}

	if total == 0 {
		t.Error("total values = 0")
	}
	if len(vals) != 5 {
		t.Errorf("got %d values, want 5", len(vals))
	}

	for i, v := range vals {
		if v.Formatted == "" {
			t.Errorf("value %d formatted is empty", i)
		}
		if v.HexDump == "" {
			t.Errorf("value %d hex is empty", i)
		}
		t.Logf("  val[%d] = %-34s  hex=%s  (%d bytes)", i, v.Formatted, v.HexDump, v.ByteLen)
	}
	t.Logf("total values in page = %d", total)
}

func TestReadPageValuesWithOffset(t *testing.T) {
	f, err := engine.Open("../testdata/small.parquet")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer f.Close()

	reader, err := f.NewRowGroupReader(0)
	if err != nil {
		t.Fatalf("NewRowGroupReader: %v", err)
	}

	// Read all values.
	allVals, _, err := reader.ReadPageValues(0, 0, 0, 100)
	if err != nil {
		t.Fatalf("ReadPageValues all: %v", err)
	}

	// Read with offset 5.
	offsetVals, _, err := reader.ReadPageValues(0, 0, 5, 3)
	if err != nil {
		t.Fatalf("ReadPageValues offset: %v", err)
	}

	if len(offsetVals) != 3 {
		t.Fatalf("got %d values, want 3", len(offsetVals))
	}

	// Values at offset 5 should match.
	for i, v := range offsetVals {
		if i+5 < len(allVals) && v.Formatted != allVals[i+5].Formatted {
			t.Errorf("offset val[%d] = %q, want %q", i, v.Formatted, allVals[i+5].Formatted)
		}
	}
}

func TestReadPageValuesOutOfRange(t *testing.T) {
	f, err := engine.Open("../testdata/small.parquet")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer f.Close()

	reader, err := f.NewRowGroupReader(0)
	if err != nil {
		t.Fatalf("NewRowGroupReader: %v", err)
	}

	_, _, err = reader.ReadPageValues(-1, 0, 0, 5)
	if err == nil {
		t.Error("expected error for negative column index")
	}

	_, _, err = reader.ReadPageValues(0, 99, 0, 5)
	if err == nil {
		t.Error("expected error for out-of-range page index")
	}
}

func TestReadPageValuesRootServiceName(t *testing.T) {
	f, err := engine.Open("../testdata/small.parquet")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer f.Close()

	reader, err := f.NewRowGroupReader(0)
	if err != nil {
		t.Fatalf("NewRowGroupReader: %v", err)
	}

	// RootServiceName (col 5) — string values.
	vals, total, err := reader.ReadPageValues(5, 0, 0, 5)
	if err != nil {
		t.Fatalf("ReadPageValues: %v", err)
	}

	t.Logf("RootServiceName page 0: total=%d", total)
	for i, v := range vals {
		t.Logf("  val[%d] = %-20s  hex=%s  (%d bytes)", i, v.Formatted, v.HexDump, v.ByteLen)
	}

	if len(vals) == 0 {
		t.Error("no values returned")
	}

	// String values should have readable hex.
	if vals[0].HexDump == "" {
		t.Error("string value hex is empty")
	}
	if vals[0].ByteLen == 0 {
		t.Error("string value byte length is 0")
	}
}
