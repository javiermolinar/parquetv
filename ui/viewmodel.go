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

	// Schema tree focus/scroll.
	SchemaFocused bool
	SchemaCursor  int // cursor line in flattened tree
	SchemaOffset  int // first visible line

	Width  int
	Height int
}

// RowGroupGridVM is the view model for the row group grid screen.
type RowGroupGridVM struct {
	TopBar      TopBarData
	BottomBar   BottomBarData
	Headers     []string  // visible column paths (simplified)
	ColWidths   []int     // display width for each visible column
	RightAlign  []bool    // true for numeric columns (right-align values)
	Rows        []GridRow // visible rows (virtual scrolled)
	SelectedRow int       // cursor row within Rows slice
	SelectedCol int       // cursor col within Headers slice
	TotalRows   int64
	TotalCols   int
	RowOffset   int64 // absolute row index of first visible row
	RGIndex     int   // row group index (for nav panel)
	RGBytes     int64 // row group total bytes (for nav panel)
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

// PageInspectorVM is the view model for the page inspector screen (Level 3).
type PageInspectorVM struct {
	TopBar     TopBarData
	BottomBar  BottomBarData
	RGIndex    int
	ColumnPath string
	ColumnType string
	Encoding   string
	Compression string
	TotalCompressed   int64
	TotalUncompressed int64
	NumValues  int64

	Pages        []PageSummaryVM
	SelectedPage int
	PageOffset   int // scroll offset in page list

	// Value viewer (right panel).
	Values          []PageValueVM
	ValueOffset     int   // scroll offset in value list
	SelectedValue   int   // cursor position within Values slice
	TotalPageValues int64 // total values in selected page
	ViewingValues   bool  // true when in value viewer mode (enter pressed)

	Width  int
	Height int
}

// PageSummaryVM is the display data for one page in the page list.
type PageSummaryVM struct {
	Index          int
	NumValues      int64
	FirstRowIndex  int64
	MinValue       string
	MaxValue       string
	CompressedSize int64
}

// PageValueVM is one value in the page value viewer.
type PageValueVM struct {
	Index   int    // absolute index within the page
	Value   string // formatted/decoded value
	HexDump string // spaced hex: "03 29 be fd..."
	ByteLen int    // raw byte count
}

// CellInspectVM holds data for the always-visible cell inspect panel.
type CellInspectVM struct {
	ColumnPath string
	RowIndex   int64
	Value      string   // first value (compact display)
	AllValues  []string // all values for repeated columns
	HexDump    string   // spaced hex: "03 29 be fd..."
	ByteLen    int
	RepCount   int // >1 for repeated columns

	// Focus mode (Tab to expand, j/k to scroll).
	Focused      bool
	ScrollOffset int
	VisibleLines int // value lines visible in focused panel
}
