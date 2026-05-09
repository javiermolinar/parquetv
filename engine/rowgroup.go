package engine

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/parquet-go/parquet-go"
)

// ColumnHeader describes a leaf column for grid display.
type ColumnHeader struct {
	Path  string // simplified path
	Type  string // parquet type
	Index int    // column index in schema
}

// RowGroupReader provides lazy row-level access to a row group.
type RowGroupReader struct {
	file    *File
	rgIndex int
	rg      parquet.RowGroup
	headers []ColumnHeader
}

// NewRowGroupReader creates a reader for the given row group.
func (f *File) NewRowGroupReader(rgIndex int) (*RowGroupReader, error) {
	rgs := f.pf.RowGroups()
	if rgIndex < 0 || rgIndex >= len(rgs) {
		return nil, fmt.Errorf("row group %d out of range [0, %d)", rgIndex, len(rgs))
	}

	rg := rgs[rgIndex]
	leafCols := f.pf.Schema().Columns()
	chunks := rg.ColumnChunks()

	headers := make([]ColumnHeader, len(leafCols))
	for i, path := range leafCols {
		typ := ""
		if i < len(chunks) {
			typ = chunks[i].Type().String()
		}
		headers[i] = ColumnHeader{
			Path:  SimplifyPath(strings.Join(path, ".")),
			Type:  typ,
			Index: i,
		}
	}

	return &RowGroupReader{
		file:    f,
		rgIndex: rgIndex,
		rg:      rg,
		headers: headers,
	}, nil
}

// Headers returns column headers.
func (r *RowGroupReader) Headers() []ColumnHeader { return r.headers }

// NumRows returns the total rows in this row group.
func (r *RowGroupReader) NumRows() int64 { return r.rg.NumRows() }

// ReadRows reads rows starting at offset, up to limit rows.
// Returns string-formatted values: one slice per row, one entry per leaf column.
// For repeated columns, shows the first value annotated with [+N].
func (r *RowGroupReader) ReadRows(offset, limit int64) ([][]string, error) {
	rows := r.rg.Rows()
	defer rows.Close()

	if err := rows.SeekToRow(offset); err != nil {
		return nil, fmt.Errorf("seek to row %d: %w", offset, err)
	}

	numCols := len(r.headers)
	buf := make([]parquet.Row, limit)
	n, err := rows.ReadRows(buf)
	if err != nil && n == 0 {
		return nil, fmt.Errorf("read rows at offset %d: %w", offset, err)
	}

	result := make([][]string, n)
	for i := 0; i < n; i++ {
		values := make([]string, numCols)
		counts := make([]int, numCols)
		seen := make([]bool, numCols)

		for _, v := range buf[i] {
			col := v.Column()
			if col < 0 || col >= numCols {
				continue
			}
			counts[col]++
			if !seen[col] {
				values[col] = FormatValue(v)
				seen[col] = true
			}
		}

		for j := range values {
			if counts[j] > 1 {
				values[j] = fmt.Sprintf("%s [+%d]", values[j], counts[j]-1)
			} else if counts[j] == 0 {
				values[j] = ""
			}
		}

		result[i] = values
	}

	return result, nil
}

// PageBoundaries returns the first row index for each page of the given column.
func (r *RowGroupReader) PageBoundaries(colIndex int) ([]int64, error) {
	chunks := r.rg.ColumnChunks()
	if colIndex < 0 || colIndex >= len(chunks) {
		return nil, fmt.Errorf("column %d out of range [0, %d)", colIndex, len(chunks))
	}

	oi, err := chunks[colIndex].OffsetIndex()
	if err != nil {
		return nil, err
	}

	bounds := make([]int64, oi.NumPages())
	for i := 0; i < oi.NumPages(); i++ {
		bounds[i] = oi.FirstRowIndex(i)
	}
	return bounds, nil
}

// FormatValue converts a parquet value to a display string.
func FormatValue(v parquet.Value) string {
	if v.IsNull() {
		return "null"
	}
	switch v.Kind() {
	case parquet.ByteArray:
		b := v.ByteArray()
		if isReadableString(b) {
			return string(b)
		}
		return formatHex(b)
	case parquet.FixedLenByteArray:
		return formatHex(v.ByteArray())
	case parquet.Int32:
		return fmt.Sprintf("%d", v.Int32())
	case parquet.Int64:
		return fmt.Sprintf("%d", v.Int64())
	case parquet.Float:
		return fmt.Sprintf("%.4g", v.Float())
	case parquet.Double:
		return fmt.Sprintf("%.4g", v.Double())
	case parquet.Boolean:
		if v.Boolean() {
			return "true"
		}
		return "false"
	default:
		return v.String()
	}
}

func formatHex(b []byte) string {
	if len(b) <= 16 {
		return hex.EncodeToString(b)
	}
	return hex.EncodeToString(b[:16]) + ".."
}

func isReadableString(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	for _, c := range b {
		if c < 0x20 || c > 0x7e {
			return false
		}
	}
	return true
}
