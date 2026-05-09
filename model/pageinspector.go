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
	pageListLinesPerPage = 5 // lines each page entry occupies in the list
	pageInspectorChrome  = 3 // top(2) + bottom(1)
	valueViewerColWidth  = 20 // width for value column
)

// PageInspectorModel handles the page inspector screen (Level 3).
type PageInspectorModel struct {
	file   *engine.File
	reader *engine.RowGroupReader

	rgIndex  int
	colIndex int

	detail engine.ColumnChunkDetail

	selectedPage int
	pageOffset   int // first visible page in left panel

	// Value viewer state.
	values        []engine.PageValueDetail
	totalPageVals int64
	valueOffset   int  // scroll offset in value list
	viewingValues bool // when true, keys control the value viewer

	width  int
	height int
}

// NewPageInspectorModel creates a page inspector for the given column chunk.
func NewPageInspectorModel(file *engine.File, reader *engine.RowGroupReader, rgIndex, colIndex int, width, height int) (PageInspectorModel, error) {
	detail, err := reader.ReadColumnChunkDetail(colIndex)
	if err != nil {
		return PageInspectorModel{}, err
	}

	m := PageInspectorModel{
		file:     file,
		reader:   reader,
		rgIndex:  rgIndex,
		colIndex: colIndex,
		detail:   detail,
		width:    width,
		height:   height,
	}

	// Load initial page values if pages exist.
	if len(detail.Pages) > 0 {
		m = m.loadPageValues()
	}

	return m, nil
}

// SetSize sets the terminal dimensions.
func (m *PageInspectorModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// valueViewerHeight returns how many value lines fit in the right panel.
func (m PageInspectorModel) valueViewerHeight() int {
	h := m.height - pageInspectorChrome - 3 // header(1) + sep(1) + padding(1)
	if h < 1 {
		h = 1
	}
	return h
}

// pageListHeight returns how many page entries fit in the left panel.
func (m PageInspectorModel) pageListHeight() int {
	// Available lines = total - chrome(3) - header(1) - blank(1)
	available := m.height - pageInspectorChrome - 2
	entries := available / pageListLinesPerPage
	if entries < 1 {
		entries = 1
	}
	return entries
}

func (m PageInspectorModel) Init() tea.Cmd { return nil }

func (m PageInspectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.viewingValues {
			return m.updateValueViewer(msg)
		}
		return m.updatePageList(msg)
	}
	return m, nil
}

// updatePageList handles keys in page list mode.
func (m PageInspectorModel) updatePageList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	numPages := len(m.detail.Pages)
	switch msg.String() {
	case "up", "k":
		if m.selectedPage > 0 {
			m.selectedPage--
			m = m.ensurePageVisible()
			m = m.loadPageValues()
		}
	case "down", "j":
		if m.selectedPage < numPages-1 {
			m.selectedPage++
			m = m.ensurePageVisible()
			m = m.loadPageValues()
		}
	case "g":
		m.selectedPage = 0
		m = m.ensurePageVisible()
		m = m.loadPageValues()
	case "G":
		if numPages > 0 {
			m.selectedPage = numPages - 1
			m = m.ensurePageVisible()
			m = m.loadPageValues()
		}
	case "left", "h":
		// Previous column chunk.
		if m.colIndex > 0 {
			return m, func() tea.Msg { return switchColumnMsg{colIndex: m.colIndex - 1} }
		}
	case "right", "l":
		// Next column chunk.
		totalCols := len(m.reader.Headers())
		if m.colIndex < totalCols-1 {
			return m, func() tea.Msg { return switchColumnMsg{colIndex: m.colIndex + 1} }
		}
	case "enter":
		if numPages > 0 {
			m.viewingValues = true
			m.valueOffset = 0
		}
	case "d":
		// Stub: dictionary view (Phase 8)
	case "f":
		// Stub: predicate simulation (Phase 8)
	case "esc":
		return m, func() tea.Msg { return backToGridMsg{} }
	}
	return m, nil
}

// updateValueViewer handles keys in value viewer mode.
func (m PageInspectorModel) updateValueViewer(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	visHeight := m.valueViewerHeight()
	maxOffset := int(m.totalPageVals) - visHeight
	if maxOffset < 0 {
		maxOffset = 0
	}

	switch msg.String() {
	case "up", "k":
		if m.valueOffset > 0 {
			m.valueOffset--
			m = m.loadPageValues()
		}
	case "down", "j":
		if m.valueOffset < maxOffset {
			m.valueOffset++
			m = m.loadPageValues()
		}
	case "g":
		m.valueOffset = 0
		m = m.loadPageValues()
	case "G":
		m.valueOffset = maxOffset
		m = m.loadPageValues()
	case "ctrl+d":
		half := visHeight / 2
		m.valueOffset += half
		if m.valueOffset > maxOffset {
			m.valueOffset = maxOffset
		}
		m = m.loadPageValues()
	case "ctrl+u":
		half := visHeight / 2
		m.valueOffset -= half
		if m.valueOffset < 0 {
			m.valueOffset = 0
		}
		m = m.loadPageValues()
	case "esc":
		m.viewingValues = false
		m.valueOffset = 0
	}
	return m, nil
}

func (m PageInspectorModel) ensurePageVisible() PageInspectorModel {
	visible := m.pageListHeight()
	if m.selectedPage < m.pageOffset {
		m.pageOffset = m.selectedPage
	}
	if m.selectedPage >= m.pageOffset+visible {
		m.pageOffset = m.selectedPage - visible + 1
	}
	if m.pageOffset < 0 {
		m.pageOffset = 0
	}
	return m
}

func (m PageInspectorModel) loadPageValues() PageInspectorModel {
	if len(m.detail.Pages) == 0 {
		m.values = nil
		m.totalPageVals = 0
		return m
	}

	limit := m.valueViewerHeight()
	vals, total, err := m.reader.ReadPageValues(m.colIndex, m.selectedPage, m.valueOffset, limit)
	if err != nil {
		m.values = nil
		m.totalPageVals = 0
		return m
	}
	m.values = vals
	m.totalPageVals = total
	return m
}

func (m PageInspectorModel) View() string {
	vm := m.BuildViewModel()
	return view.RenderPageInspector(vm)
}

// BuildViewModel converts model state into a PageInspectorVM.
func (m PageInspectorModel) BuildViewModel() ui.PageInspectorVM {
	info := m.file.Info()

	// Top bar context for Level 3.
	context := fmt.Sprintf("%s  %s  %s  %s  %s values",
		m.detail.Path,
		m.detail.Type,
		m.detail.Encoding,
		formatBytes(m.detail.TotalCompressed),
		formatNumber(m.detail.NumValues),
	)

	breadcrumb := fmt.Sprintf("%s › RG %d › %s",
		filepath.Base(info.Path), m.rgIndex, m.detail.Path)
	if m.viewingValues {
		breadcrumb += fmt.Sprintf(" › Page %d", m.selectedPage)
	}

	var shortcuts []string
	if m.viewingValues {
		shortcuts = []string{"↑↓ scroll", "ctrl+d/u half-page", "g/G jump", "esc back"}
	} else {
		shortcuts = []string{"↑↓ pages", "◂▸ column", "enter values", "d dict", "f simulate", "esc back"}
	}

	// Build page summaries.
	pages := make([]ui.PageSummaryVM, len(m.detail.Pages))
	for i, p := range m.detail.Pages {
		pages[i] = ui.PageSummaryVM{
			Index:          p.Index,
			NumValues:      p.NumValues,
			FirstRowIndex:  p.FirstRowIndex,
			MinValue:       p.MinValue,
			MaxValue:       p.MaxValue,
			CompressedSize: p.CompressedSize,
		}
	}

	// Build value list.
	var values []ui.PageValueVM
	for i, v := range m.values {
		values = append(values, ui.PageValueVM{
			Index:   m.valueOffset + i,
			Value:   v.Formatted,
			HexDump: v.HexDump,
			ByteLen: v.ByteLen,
		})
	}

	return ui.PageInspectorVM{
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
			Shortcuts:  shortcuts,
		},
		ColumnPath:      m.detail.Path,
		ColumnType:      m.detail.Type,
		Encoding:        m.detail.Encoding,
		Compression:     m.detail.Compression,
		TotalCompressed: m.detail.TotalCompressed,
		TotalUncompressed: m.detail.TotalUncompressed,
		NumValues:       m.detail.NumValues,
		Pages:           pages,
		SelectedPage:    m.selectedPage,
		PageOffset:      m.pageOffset,
		Values:          values,
		ValueOffset:     m.valueOffset,
		TotalPageValues: m.totalPageVals,
		ViewingValues:   m.viewingValues,
		Width:           m.width,
		Height:          m.height,
	}
}
