package engine

import (
	"fmt"
	"sort"

	"github.com/parquet-go/parquet-go"
)

// DictEntry holds one dictionary entry with its frequency.
type DictEntry struct {
	Index int    // dictionary index
	Value string // formatted value
	Count int64  // number of occurrences across all pages
}

// DictionaryResult holds the full dictionary for a column chunk.
type DictionaryResult struct {
	Path    string
	Entries []DictEntry // sorted by count descending
	Total   int64       // total values in the column chunk
}

// ReadDictionary reads the dictionary and value frequencies for a column chunk.
// Returns an error if the column is not dictionary-encoded.
func (r *RowGroupReader) ReadDictionary(colIndex int) (DictionaryResult, error) {
	chunks := r.rg.ColumnChunks()
	if colIndex < 0 || colIndex >= len(chunks) {
		return DictionaryResult{}, fmt.Errorf("column %d out of range [0, %d)", colIndex, len(chunks))
	}

	cc := chunks[colIndex]
	pages := cc.Pages()
	defer pages.Close()

	// Read the first page to get the dictionary.
	firstPage, err := pages.ReadPage()
	if err != nil {
		return DictionaryResult{}, fmt.Errorf("read first page: %w", err)
	}

	dict := firstPage.Dictionary()
	if dict == nil {
		parquet.Release(firstPage)
		return DictionaryResult{}, fmt.Errorf("column %q is not dictionary-encoded", r.headers[colIndex].Path)
	}

	dictLen := dict.Len()

	// Build entry list from dictionary values.
	entries := make([]DictEntry, dictLen)
	for i := 0; i < dictLen; i++ {
		entries[i] = DictEntry{
			Index: i,
			Value: FormatValue(dict.Index(int32(i))),
		}
	}

	// Build value→index lookup for frequency counting.
	valueIndex := make(map[string]int, dictLen)
	for i := range entries {
		valueIndex[entries[i].Value] = i
	}

	// Count frequencies by reading decoded values from all pages.
	var total int64
	countPage := func(p parquet.Page) {
		vr := p.Values()
		buf := make([]parquet.Value, 1024)
		for {
			n, err := vr.ReadValues(buf)
			for i := 0; i < n; i++ {
				total++
				formatted := FormatValue(buf[i])
				if idx, ok := valueIndex[formatted]; ok {
					entries[idx].Count++
				}
			}
			if n == 0 || err != nil {
				break
			}
		}
	}

	countPage(firstPage)
	parquet.Release(firstPage)

	// Read remaining pages.
	for {
		p, err := pages.ReadPage()
		if err != nil {
			break
		}
		countPage(p)
		parquet.Release(p)
	}

	// Sort by count descending.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Count > entries[j].Count
	})

	path := ""
	if colIndex < len(r.headers) {
		path = r.headers[colIndex].Path
	}

	return DictionaryResult{
		Path:    path,
		Entries: entries,
		Total:   total,
	}, nil
}
