package engine

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/parquet-go/parquet-go"
	"github.com/parquet-go/parquet-go/format"
)

// File wraps a parquet file with lazy access to row groups and pages.
type File struct {
	path string
	file *os.File
	pf   *parquet.File
	info *FileInfo
}

// Open opens a Parquet file and reads the footer metadata.
func Open(path string) (*File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}

	stat, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}

	pf, err := parquet.OpenFile(f, stat.Size())
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("open parquet %s: %w", path, err)
	}

	file := &File{path: path, file: f, pf: pf}
	file.info = file.buildFileInfo(stat.Size())
	return file, nil
}

// Info returns the file metadata (built from footer on open).
func (f *File) Info() *FileInfo {
	return f.info
}

// Close closes the underlying file.
func (f *File) Close() error {
	return f.file.Close()
}

func (f *File) buildFileInfo(size int64) *FileInfo {
	schema := f.pf.Schema()
	rowGroups := f.pf.RowGroups()
	leafColumns := collectLeafColumns(schema)

	info := &FileInfo{
		Path:         f.path,
		Size:         size,
		NumRows:      f.pf.NumRows(),
		NumRowGroups: len(rowGroups),
		NumColumns:   len(leafColumns),
		Schema:       buildSchemaTree(schema),
		KeyValues:    extractKeyValues(f.pf.Metadata()),
		RowGroups:    make([]RowGroupInfo, len(rowGroups)),
	}

	// Accumulate column sizes across all row groups for top columns.
	columnSizes := make(map[string]int64)

	for i, rg := range rowGroups {
		rgInfo := buildRowGroupInfo(i, rg, leafColumns)
		info.RowGroups[i] = rgInfo
		for _, ci := range rgInfo.ColumnInfos {
			columnSizes[ci.Path] += ci.TotalBytes
		}
	}

	info.TopColumns = buildTopColumns(columnSizes, 10)
	return info
}

// collectLeafColumns returns the schema leaf columns in order.
func collectLeafColumns(schema *parquet.Schema) [][]string {
	return schema.Columns()
}

// SimplifyPath removes Parquet list/element noise from column paths.
// e.g. "rs.list.element.ss.list.element.Spans.list.element.Name" → "rs.ss.Spans.Name"
func SimplifyPath(path string) string {
	parts := strings.Split(path, ".")
	var clean []string
	for _, p := range parts {
		if p == "list" || p == "element" {
			continue
		}
		clean = append(clean, p)
	}
	return strings.Join(clean, ".")
}

func buildRowGroupInfo(index int, rg parquet.RowGroup, leafColumns [][]string) RowGroupInfo {
	chunks := rg.ColumnChunks()
	rgInfo := RowGroupInfo{
		Index:       index,
		NumRows:     rg.NumRows(),
		NumColumns:  len(chunks),
		ColumnInfos: make([]ColumnInfo, len(chunks)),
	}

	for i, cc := range chunks {
		path := ""
		if i < len(leafColumns) {
			path = SimplifyPath(strings.Join(leafColumns[i], "."))
		}
		ci := buildColumnInfo(cc, path)
		rgInfo.ColumnInfos[i] = ci
		rgInfo.TotalBytes += ci.TotalBytes
	}

	return rgInfo
}

func buildColumnInfo(cc parquet.ColumnChunk, path string) ColumnInfo {
	ci := ColumnInfo{
		Path:      path,
		Type:      cc.Type().String(),
		NumValues: cc.NumValues(),
	}

	// Read page count and compressed sizes from offset index.
	oi, err := cc.OffsetIndex()
	if err == nil && oi.NumPages() > 0 {
		ci.NumPages = oi.NumPages()
		for j := 0; j < oi.NumPages(); j++ {
			ci.CompressedBytes += oi.CompressedPageSize(j)
		}
	}
	ci.TotalBytes = ci.CompressedBytes

	return ci
}

func buildSchemaTree(schema *parquet.Schema) *SchemaNode {
	root := &SchemaNode{Name: "root"}
	for _, f := range schema.Fields() {
		root.Children = append(root.Children, buildFieldNode(f))
	}
	return root
}

func buildFieldNode(field parquet.Field) *SchemaNode {
	node := &SchemaNode{
		Name: field.Name(),
		Leaf: field.Leaf(),
	}
	if field.Leaf() {
		node.Type = field.Type().String()
	}
	for _, child := range field.Fields() {
		node.Children = append(node.Children, buildFieldNode(child))
	}
	return node
}

func extractKeyValues(meta *format.FileMetaData) map[string]string {
	kv := make(map[string]string)
	for _, pair := range meta.KeyValueMetadata {
		kv[pair.Key] = pair.Value
	}
	return kv
}

func buildTopColumns(sizes map[string]int64, limit int) []ColumnInfo {
	cols := make([]ColumnInfo, 0, len(sizes))
	for path, size := range sizes {
		cols = append(cols, ColumnInfo{Path: path, TotalBytes: size})
	}
	sort.Slice(cols, func(i, j int) bool {
		return cols[i].TotalBytes > cols[j].TotalBytes
	})
	if len(cols) > limit {
		cols = cols[:limit]
	}
	return cols
}
