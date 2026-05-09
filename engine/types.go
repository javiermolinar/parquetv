package engine

// RowGroupInfo contains metadata for a single row group.
type RowGroupInfo struct {
	Index       int
	NumRows     int64
	NumColumns  int
	TotalBytes  int64
	ColumnInfos []ColumnInfo
}

// ColumnInfo contains metadata for a column within a row group.
type ColumnInfo struct {
	Path            string
	Type            string
	Encoding        string
	Compression     string
	TotalBytes      int64
	CompressedBytes int64
	NumValues       int64
	NumPages        int
}

// FileInfo contains top-level file metadata.
type FileInfo struct {
	Path         string
	Size         int64
	NumRows      int64
	NumRowGroups int
	NumColumns   int
	Schema       *SchemaNode
	KeyValues    map[string]string
	RowGroups    []RowGroupInfo
	TopColumns   []ColumnInfo // sorted by size descending
}

// SchemaNode represents a node in the Parquet schema tree.
type SchemaNode struct {
	Name     string
	Type     string // empty for group nodes
	Children []*SchemaNode
	Leaf     bool
}
