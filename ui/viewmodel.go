// Package ui contains shared view model types used by both model and view layers.
// This breaks the import cycle: model → ui ← view, neither imports the other.
package ui

import "github.com/javiermolinar/parquetv/engine"

// TopBarData holds data for the top chrome bar.
type TopBarData struct {
	FileName     string
	FileSize     int64
	Format       string
	TotalRows    int64
	NumRowGroups int
	NumColumns   int
	Context      string // level-specific context line
}

// BottomBarData holds data for the bottom chrome bar.
type BottomBarData struct {
	Breadcrumb string
	Shortcuts  []string // e.g. ["enter open", "s schema", "q quit"]
}

// RowGroupSummary is the display data for one row group in Level 1.
type RowGroupSummary struct {
	Index       int
	NumRows     int64
	TotalBytes  int64
	AvgPagesCol int
}

// FooterData holds the right panel content for Level 1.
type FooterData struct {
	NumLeafColumns int
	Format         string
	TopColumns     []engine.ColumnInfo
	KeyValues      map[string]string
}

// SchemaNodeVM is a view-model node for the schema tree.
type SchemaNodeVM struct {
	Name     string
	Type     string
	Depth    int
	Leaf     bool
	Expanded bool
	Children []*SchemaNodeVM
}

// FileOverviewVM is the view model for the file overview screen.
type FileOverviewVM struct {
	TopBar      TopBarData
	BottomBar   BottomBarData
	RowGroups   []RowGroupSummary
	Selected    int
	FooterPanel FooterData
	SchemaTree  *SchemaNodeVM
	Width       int
	Height      int
}

// RowGroupGridVM is the view model for the row group grid screen.
type RowGroupGridVM struct {
	TopBar      TopBarData
	BottomBar   BottomBarData
	Headers     []string  // visible column paths (simplified)
	ColWidths   []int     // display width for each visible column
	Rows        []GridRow // visible rows (virtual scrolled)
	SelectedRow int       // cursor row within Rows slice
	SelectedCol int       // cursor col within Headers slice
	TotalRows   int64
	TotalCols   int
	RowOffset   int64 // absolute row index of first visible row
	Stats       ColumnStatsVM
	Inspect     CellInspectVM
	PageBounds  []int // viewport-relative row indices where a page boundary appears BEFORE the row
	Width       int
	Height      int
}

// GridRow is one row in the row group grid.
type GridRow struct {
	Index  int64    // absolute row index in the row group
	Values []string // values for visible columns only
}

// ColumnStatsVM holds display data for the selected column stats bar.
type ColumnStatsVM struct {
	Path         string
	Type         string
	TotalBytes   int64
	NumValues    int64
	NumPages     int
	ValuesPerRow float64
}

// CellInspectVM holds data for the always-visible cell inspect panel.
type CellInspectVM struct {
	ColumnPath string
	RowIndex   int64
	Value      string // full untruncated value
	HexDump    string // spaced hex: "03 29 be fd..."
	ByteLen    int
	RepCount   int // >1 for repeated columns
}
