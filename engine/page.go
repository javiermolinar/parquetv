package engine

import (
	"fmt"
	"io"
	"strings"

	"github.com/parquet-go/parquet-go"
	"github.com/parquet-go/parquet-go/format"
)

// PageInfo describes a single data page within a column chunk.
type PageInfo struct {
	Index          int
	NumValues      int64
	FirstRowIndex  int64
	MinValue       string // formatted display string
	MaxValue       string // formatted display string
	MinRaw         []byte // raw bytes for comparison
	MaxRaw         []byte // raw bytes for comparison
	CompressedSize int64
}

// ColumnChunkDetail holds metadata about a column chunk for the page inspector.
type ColumnChunkDetail struct {
	Path              string
	Type              string
	Encoding          string
	Compression       string
	TotalCompressed   int64
	TotalUncompressed int64
	NumValues         int64
	Pages             []PageInfo
}

// ReadColumnChunkDetail reads page-level metadata for a column chunk.
// Uses column index and offset index for fast metadata without decompression.
func (r *RowGroupReader) ReadColumnChunkDetail(colIndex int) (ColumnChunkDetail, error) {
	chunks := r.rg.ColumnChunks()
	if colIndex < 0 || colIndex >= len(chunks) {
		return ColumnChunkDetail{}, fmt.Errorf("column %d out of range [0, %d)", colIndex, len(chunks))
	}

	cc := chunks[colIndex]

	path := ""
	if colIndex < len(r.headers) {
		path = r.headers[colIndex].Path
	}

	detail := ColumnChunkDetail{
		Path:      path,
		Type:      cc.Type().String(),
		NumValues: cc.NumValues(),
	}

	// Encoding and compression from format metadata.
	fmtMeta := r.file.pf.Metadata()
	if r.rgIndex < len(fmtMeta.RowGroups) {
		fmtRG := fmtMeta.RowGroups[r.rgIndex]
		if colIndex < len(fmtRG.Columns) {
			meta := fmtRG.Columns[colIndex].MetaData
			detail.Encoding = formatEncodings(meta.Encoding)
			detail.Compression = formatCompression(meta.Codec)
			detail.TotalCompressed = meta.TotalCompressedSize
			detail.TotalUncompressed = meta.TotalUncompressedSize
		}
	}

	// Page-level info from offset index and column index.
	oi, oiErr := cc.OffsetIndex()
	ci, ciErr := cc.ColumnIndex()

	if oiErr == nil && oi.NumPages() > 0 {
		numPages := oi.NumPages()
		detail.Pages = make([]PageInfo, numPages)

		for i := 0; i < numPages; i++ {
			pi := PageInfo{
				Index:          i,
				FirstRowIndex:  oi.FirstRowIndex(i),
				CompressedSize: oi.CompressedPageSize(i),
			}

			// Estimate value count from row index differences.
			// For non-repeated columns this equals value count exactly.
			// For repeated columns this is the row count (a lower bound).
			if i < numPages-1 {
				pi.NumValues = oi.FirstRowIndex(i+1) - oi.FirstRowIndex(i)
			} else {
				pi.NumValues = r.rg.NumRows() - oi.FirstRowIndex(i)
			}

			// Min/max from column index.
			if ciErr == nil && i < ci.NumPages() && !ci.NullPage(i) {
				minV := ci.MinValue(i)
				maxV := ci.MaxValue(i)
				pi.MinValue = FormatValue(minV)
				pi.MaxValue = FormatValue(maxV)
				pi.MinRaw = valueRawBytes(minV)
				pi.MaxRaw = valueRawBytes(maxV)
			}

			detail.Pages[i] = pi
		}
	}

	return detail, nil
}

// PageValueDetail holds a decoded page value with its raw byte representation.
type PageValueDetail struct {
	Formatted string // human-readable value
	HexDump   string // spaced hex: "03 29 be fd..."
	RawBytes  []byte // raw bytes for comparison
	ByteLen   int
}

// ReadPageValues reads values from a specific page of a column.
// Returns both formatted strings and raw hex for hex-editor style display.
// offset and limit control pagination within the page.
func (r *RowGroupReader) ReadPageValues(colIndex, pageIndex int, offset, limit int) ([]PageValueDetail, int64, error) {
	chunks := r.rg.ColumnChunks()
	if colIndex < 0 || colIndex >= len(chunks) {
		return nil, 0, fmt.Errorf("column %d out of range [0, %d)", colIndex, len(chunks))
	}

	pages := chunks[colIndex].Pages()
	defer pages.Close()

	// Read through pages to reach the target.
	var targetPage parquet.Page
	for i := 0; i <= pageIndex; i++ {
		p, err := pages.ReadPage()
		if err != nil {
			return nil, 0, fmt.Errorf("read page %d: %w", i, err)
		}
		if i < pageIndex {
			// Release non-target pages.
			parquet.Release(p)
		} else {
			targetPage = p
		}
	}
	defer parquet.Release(targetPage)

	totalValues := targetPage.NumValues()
	vr := targetPage.Values()

	// Skip to offset.
	buf := make([]parquet.Value, 256)
	skipped := 0
	for skipped < offset {
		toRead := offset - skipped
		if toRead > len(buf) {
			toRead = len(buf)
		}
		n, err := vr.ReadValues(buf[:toRead])
		skipped += n
		if n == 0 || (err != nil && err != io.EOF) {
			break
		}
	}

	// Read requested values.
	result := make([]PageValueDetail, 0, limit)
	remaining := limit
	for remaining > 0 {
		toRead := remaining
		if toRead > len(buf) {
			toRead = len(buf)
		}
		n, err := vr.ReadValues(buf[:toRead])
		for i := 0; i < n; i++ {
			raw := valueRawBytes(buf[i])
			result = append(result, PageValueDetail{
				Formatted: FormatValue(buf[i]),
				HexDump:   FormatHexDump(raw, 16),
				RawBytes:  raw,
				ByteLen:   len(raw),
			})
		}
		remaining -= n
		if n == 0 || err != nil {
			break
		}
	}

	return result, totalValues, nil
}

func formatEncodings(encs []format.Encoding) string {
	if len(encs) == 0 {
		return "unknown"
	}
	// Deduplicate and format encoding names.
	seen := make(map[string]bool)
	var names []string
	for _, e := range encs {
		name := formatEncodingName(e)
		if !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	return strings.Join(names, ", ")
}

func formatEncodingName(e format.Encoding) string {
	switch e {
	case format.Plain:
		return "PLAIN"
	case format.PlainDictionary:
		return "PLAIN_DICTIONARY"
	case format.RLE:
		return "RLE"
	case format.BitPacked:
		return "BIT_PACKED"
	case format.DeltaBinaryPacked:
		return "DELTA"
	case format.DeltaLengthByteArray:
		return "DELTA_LENGTH"
	case format.DeltaByteArray:
		return "DELTA_BYTE_ARRAY"
	case format.RLEDictionary:
		return "RLE_DICTIONARY"
	case format.ByteStreamSplit:
		return "BYTE_STREAM_SPLIT"
	default:
		return e.String()
	}
}

func formatCompression(c format.CompressionCodec) string {
	switch c {
	case format.Uncompressed:
		return "UNCOMPRESSED"
	case format.Snappy:
		return "SNAPPY"
	case format.Gzip:
		return "GZIP"
	case format.Lz4Raw:
		return "LZ4"
	case format.Zstd:
		return "ZSTD"
	default:
		return c.String()
	}
}

// CompareRawBytes compares two raw byte slices for ordering.
// Works correctly for big-endian integers and lexicographic byte arrays.
// Returns -1, 0, or 1.
func CompareRawBytes(a, b []byte) int {
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}
	for i := 0; i < minLen; i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	if len(a) < len(b) {
		return -1
	}
	if len(a) > len(b) {
		return 1
	}
	return 0
}
