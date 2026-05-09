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
