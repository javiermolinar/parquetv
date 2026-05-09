package model

import (
	"fmt"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/javiermolinar/parquetv/engine"
	"github.com/javiermolinar/parquetv/ui"
	"github.com/javiermolinar/parquetv/view"
)

const (
	rowNumColWidth = 7  // width for the "Row" number column
	minColWidth    = 14 // minimum data column width
	maxColWidth    = 24 // maximum data column width
	gridChromeRows = 8  // top(2) + header(1) + sep(1) + statsSep(1) + stats(2) + bottom(1)
)

// RowGroupGridModel handles the row group grid screen.
type RowGroupGridModel struct {
	file   *engine.File
	reader *engine.RowGroupReader

	rgIndex   int
	totalRows int64
	totalCols int

	cursorRow int64 // absolute row in row group (0-based)
	cursorCol int   // absolute column index (0-based)
	rowOffset int64 // first visible row
	colOffset int   // first visible column

	colWidths []int // precomputed display width per column

	// Cached row data for current viewport.
	cachedRows   [][]string
	cachedOffset int64

	width  int
	height int
}

// NewRowGroupGridModel creates a grid model for the given row group.
func NewRowGroupGridModel(file *engine.File, rgIndex int, width, height int) (RowGroupGridModel, error) {
	reader, err := file.NewRowGroupReader(rgIndex)
	if err != nil {
		return RowGroupGridModel{}, err
	}

	headers := reader.Headers()
	colWidths := make([]int, len(headers))
	for i, h := range headers {
		w := len(h.Path) + 2
		if w < minColWidth {
			w = minColWidth
		}
		if w > maxColWidth {
			w = maxColWidth
		}
		colWidths[i] = w
	}

	m := RowGroupGridModel{
		file:      file,
		reader:    reader,
		rgIndex:   rgIndex,
		totalRows: reader.NumRows(),
		totalCols: len(headers),
		colWidths: colWidths,
		width:     width,
		height:    height,
	}

	m.cachedRows, _ = m.reader.ReadRows(0, int64(m.gridHeight()))
	m.cachedOffset = 0
	return m, nil
}

// SetSize sets the terminal dimensions.
func (m *RowGroupGridModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// gridHeight returns the number of data rows that fit.
func (m RowGroupGridModel) gridHeight() int {
	h := m.height - gridChromeRows
	if h < 1 {
		h = 1
	}
	return h
}

// visibleColCount returns how many columns fit in the current width.
func (m RowGroupGridModel) visibleColCount() int {
	avail := m.width - rowNumColWidth
	count := 0
	used := 0
	for i := m.colOffset; i < m.totalCols; i++ {
		w := m.colWidths[i]
		if used+w > avail && count > 0 {
			break
		}
		used += w
		count++
	}
	if count == 0 && m.totalCols > 0 {
		count = 1
	}
	return count
}

func (m RowGroupGridModel) ensureCursorVisible() RowGroupGridModel {
	gh := int64(m.gridHeight())

	// Vertical: keep cursor within viewport.
	if m.cursorRow < m.rowOffset {
		m.rowOffset = m.cursorRow
	}
	if m.cursorRow >= m.rowOffset+gh {
		m.rowOffset = m.cursorRow - gh + 1
	}
	if m.rowOffset < 0 {
		m.rowOffset = 0
	}

	// Horizontal: iterate until stable because visibleColCount depends on colOffset
	// (variable-width columns mean shifting the offset changes how many fit).
	for i := 0; i < 10; i++ {
		visCols := m.visibleColCount()
		if m.cursorCol < m.colOffset {
			m.colOffset = m.cursorCol
		} else if m.cursorCol >= m.colOffset+visCols {
			m.colOffset = m.cursorCol - visCols + 1
		} else {
			break // cursor is visible
		}
	}
	if m.colOffset < 0 {
		m.colOffset = 0
	}

	return m
}

func (m RowGroupGridModel) reloadIfNeeded() RowGroupGridModel {
	gh := m.gridHeight()
	if m.cachedRows != nil && m.cachedOffset == m.rowOffset && len(m.cachedRows) >= gh {
		return m
	}

	limit := int64(gh)
	if m.rowOffset+limit > m.totalRows {
		limit = m.totalRows - m.rowOffset
	}
	if limit <= 0 {
		m.cachedRows = nil
		m.cachedOffset = m.rowOffset
		return m
	}

	rows, err := m.reader.ReadRows(m.rowOffset, limit)
	if err != nil {
		m.cachedRows = nil
		return m
	}
	m.cachedRows = rows
	m.cachedOffset = m.rowOffset
	return m
}

func (m RowGroupGridModel) Init() tea.Cmd { return nil }

func (m RowGroupGridModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursorRow > 0 {
				m.cursorRow--
				m = m.ensureCursorVisible()
				m = m.reloadIfNeeded()
			}
		case "down", "j":
			if m.cursorRow < m.totalRows-1 {
				m.cursorRow++
				m = m.ensureCursorVisible()
				m = m.reloadIfNeeded()
			}
		case "left", "h":
			if m.cursorCol > 0 {
				m.cursorCol--
				m = m.ensureCursorVisible()
			}
		case "right", "l":
			if m.cursorCol < m.totalCols-1 {
				m.cursorCol++
				m = m.ensureCursorVisible()
			}
		case "g":
			m.cursorRow = 0
			m = m.ensureCursorVisible()
			m = m.reloadIfNeeded()
		case "G":
			m.cursorRow = m.totalRows - 1
			m = m.ensureCursorVisible()
			m = m.reloadIfNeeded()
		case "ctrl+d":
			half := int64(m.gridHeight() / 2)
			m.cursorRow += half
			if m.cursorRow >= m.totalRows {
				m.cursorRow = m.totalRows - 1
			}
			m = m.ensureCursorVisible()
			m = m.reloadIfNeeded()
		case "ctrl+u":
			half := int64(m.gridHeight() / 2)
			m.cursorRow -= half
			if m.cursorRow < 0 {
				m.cursorRow = 0
			}
			m = m.ensureCursorVisible()
			m = m.reloadIfNeeded()
		case "esc":
			return m, func() tea.Msg { return backToOverviewMsg{} }
		case "enter":
			// Stub: page inspector (Phase 3)
		case "r":
			// Stub: row expansion (Phase 6)
		case "f":
			// Stub: column filter (Phase 4)
		}
	}
	return m, nil
}

func (m RowGroupGridModel) View() string {
	vm := m.BuildViewModel()
	return view.RenderRowGroupGrid(vm)
}

// BuildViewModel converts model state into a RowGroupGridVM.
func (m RowGroupGridModel) BuildViewModel() ui.RowGroupGridVM {
	info := m.file.Info()
	rgInfo := info.RowGroups[m.rgIndex]

	visCols := m.visibleColCount()
	allHeaders := m.reader.Headers()

	headers := make([]string, visCols)
	colWidths := make([]int, visCols)
	for i := 0; i < visCols; i++ {
		ci := m.colOffset + i
		if ci < len(allHeaders) {
			headers[i] = allHeaders[ci].Path
			colWidths[i] = m.colWidths[ci]
		}
	}

	// Build visible rows with only visible columns.
	var gridRows []ui.GridRow
	if m.cachedRows != nil {
		for i, row := range m.cachedRows {
			absRow := m.cachedOffset + int64(i)
			vals := make([]string, visCols)
			for j := 0; j < visCols; j++ {
				ci := m.colOffset + j
				if ci < len(row) {
					vals[j] = row[ci]
				}
			}
			gridRows = append(gridRows, ui.GridRow{
				Index:  absRow,
				Values: vals,
			})
		}
	}

	// Page boundaries for the selected column.
	var pageBounds []int
	bounds, err := m.reader.PageBoundaries(m.cursorCol)
	if err == nil {
		for _, b := range bounds {
			relRow := int(b - m.rowOffset)
			// Only show boundaries between rows (not at row 0 of viewport).
			if relRow > 0 && relRow < len(gridRows) {
				pageBounds = append(pageBounds, relRow)
			}
		}
	}

	// Column stats for the selected column.
	var stats ui.ColumnStatsVM
	if m.cursorCol < len(rgInfo.ColumnInfos) {
		ci := rgInfo.ColumnInfos[m.cursorCol]
		vpr := float64(0)
		if rgInfo.NumRows > 0 {
			vpr = float64(ci.NumValues) / float64(rgInfo.NumRows)
		}
		stats = ui.ColumnStatsVM{
			Path:         ci.Path,
			Type:         ci.Type,
			TotalBytes:   ci.CompressedBytes,
			NumValues:    ci.NumValues,
			NumPages:     ci.NumPages,
			ValuesPerRow: vpr,
		}
	}

	// Top bar context for Level 2.
	context := fmt.Sprintf("Row Group %d  %s rows  %s",
		m.rgIndex,
		formatNumber(rgInfo.NumRows),
		formatBytes(rgInfo.TotalBytes),
	)

	breadcrumb := fmt.Sprintf("%s › RG %d", filepath.Base(info.Path), m.rgIndex)

	return ui.RowGroupGridVM{
		TopBar: ui.TopBarData{
			FileName:     filepath.Base(info.Path),
			FileSize:     info.Size,
			TotalRows:    info.NumRows,
			NumRowGroups: info.NumRowGroups,
			NumColumns:   info.NumColumns,
			Context:      context,
		},
		BottomBar: ui.BottomBarData{
			Breadcrumb: breadcrumb,
			Shortcuts:  []string{"↑↓ rows", "◂▸ cols", "enter pages", "f filter", "r expand", "esc back"},
		},
		Headers:     headers,
		ColWidths:   colWidths,
		Rows:        gridRows,
		SelectedRow: int(m.cursorRow - m.rowOffset),
		SelectedCol: m.cursorCol - m.colOffset,
		TotalRows:   m.totalRows,
		TotalCols:   m.totalCols,
		RowOffset:   m.rowOffset,
		Stats:       stats,
		PageBounds:  pageBounds,
		Width:       m.width,
		Height:      m.height,
	}
}

// Formatting helpers — keep in model to avoid importing view.

func formatBytes(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func formatNumber(n int64) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var parts []string
	for i := len(s); i > 0; i -= 3 {
		start := i - 3
		if start < 0 {
			start = 0
		}
		parts = append([]string{s[start:i]}, parts...)
	}
	return joinStrings(parts, ",")
}

func joinStrings(parts []string, sep string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += sep
		}
		result += p
	}
	return result
}
