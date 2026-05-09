package engine_test

import (
	"testing"

	"github.com/javiermolinar/parquetv/engine"
)

func TestReadDictionary(t *testing.T) {
	f, err := engine.Open("../testdata/small.parquet")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer f.Close()

	reader, err := f.NewRowGroupReader(0)
	if err != nil {
		t.Fatalf("NewRowGroupReader: %v", err)
	}

	// RootServiceName (col 5) — dict-encoded string column.
	result, err := reader.ReadDictionary(5)
	if err != nil {
		t.Fatalf("ReadDictionary(5): %v", err)
	}

	if result.Path != "RootServiceName" {
		t.Errorf("path = %q, want RootServiceName", result.Path)
	}
	if len(result.Entries) == 0 {
		t.Fatal("no dictionary entries")
	}
	if result.Total == 0 {
		t.Error("total = 0")
	}

	t.Logf("RootServiceName: %d entries, %d total values", len(result.Entries), result.Total)
	for i, e := range result.Entries {
		t.Logf("  [%d] %q  count=%d", i, e.Value, e.Count)
	}

	// All counts should sum to total.
	var sum int64
	for _, e := range result.Entries {
		sum += e.Count
	}
	if sum != result.Total {
		t.Errorf("sum of counts %d != total %d", sum, result.Total)
	}

	// Entries should be sorted by count descending.
	for i := 1; i < len(result.Entries); i++ {
		if result.Entries[i].Count > result.Entries[i-1].Count {
			t.Errorf("entries not sorted: [%d].Count=%d > [%d].Count=%d",
				i, result.Entries[i].Count, i-1, result.Entries[i-1].Count)
		}
	}
}

func TestReadDictionaryNonDict(t *testing.T) {
	f, err := engine.Open("../testdata/small.parquet")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer f.Close()

	reader, err := f.NewRowGroupReader(0)
	if err != nil {
		t.Fatalf("NewRowGroupReader: %v", err)
	}

	// DurationNano (col 4) — delta-encoded, not dict.
	_, err = reader.ReadDictionary(4)
	if err == nil {
		t.Error("expected error for non-dict column")
	}
	t.Logf("expected error: %v", err)
}
