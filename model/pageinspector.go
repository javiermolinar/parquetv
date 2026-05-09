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
	valueCursor   int  // absolute value index (0-based within page)
	valueOffset   int  // first visible value index
	viewingValues bool // when true, keys control the value viewer

	// Dictionary overlay state.
	showingDict  bool
	dictResult   *engine.DictionaryResult
	dictCursor   int
	dictOffset   int

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
		if m.showingDict {
			return m.updateDictView(msg)
		}
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
			m.valueCursor = 0
			m.valueOffset = 0
			m = m.loadPageValues()
		}
	case "d":
		if m.dictResult == nil {
			result, err := m.reader.ReadDictionary(m.colIndex)
			if err != nil {
				break // not dict-encoded, ignore
			}
			m.dictResult = &result
		}
		m.showingDict = true
		m.dictCursor = 0
		m.dictOffset = 0
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
	maxCursor := int(m.totalPageVals) - 1
	if maxCursor < 0 {
		maxCursor = 0
	}

	switch msg.String() {
	case "up", "k":
		if m.valueCursor > 0 {
			m.valueCursor--
			m = m.ensureValueVisible()
			m = m.loadPageValues()
		}
	case "down", "j":
		if m.valueCursor < maxCursor {
			m.valueCursor++
			m = m.ensureValueVisible()
			m = m.loadPageValues()
		}
	case "g":
		m.valueCursor = 0
		m = m.ensureValueVisible()
		m = m.loadPageValues()
	case "G":
		m.valueCursor = maxCursor
		m = m.ensureValueVisible()
		m = m.loadPageValues()
	case "ctrl+d":
		half := visHeight / 2
		m.valueCursor += half
		if m.valueCursor > maxCursor {
			m.valueCursor = maxCursor
		}
		m = m.ensureValueVisible()
		m = m.loadPageValues()
	case "ctrl+u":
		half := visHeight / 2
		m.valueCursor -= half
		if m.valueCursor < 0 {
			m.valueCursor = 0
		}
		m = m.ensureValueVisible()
		m = m.loadPageValues()
	case "esc":
		m.viewingValues = false
		m.valueCursor = 0
		m.valueOffset = 0
	}
	return m, nil
}

// updateDictView handles keys when the dictionary overlay is showing.
func (m PageInspectorModel) updateDictView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.dictResult == nil {
		m.showingDict = false
		return m, nil
	}

	visHeight := m.dictVisibleLines()
	maxCursor := len(m.dictResult.Entries) - 1
	if maxCursor < 0 {
		maxCursor = 0
	}

	switch msg.String() {
	case "down", "j":
		if m.dictCursor < maxCursor {
			m.dictCursor++
			m = m.ensureDictVisible()
		}
	case "up", "k":
		if m.dictCursor > 0 {
			m.dictCursor--
			m = m.ensureDictVisible()
		}
	case "g":
		m.dictCursor = 0
		m = m.ensureDictVisible()
	case "G":
		m.dictCursor = maxCursor
		m = m.ensureDictVisible()
	case "ctrl+d":
		m.dictCursor += visHeight / 2
		if m.dictCursor > maxCursor {
			m.dictCursor = maxCursor
		}
		m = m.ensureDictVisible()
	case "ctrl+u":
		m.dictCursor -= visHeight / 2
		if m.dictCursor < 0 {
			m.dictCursor = 0
		}
		m = m.ensureDictVisible()
	case "esc":
		m.showingDict = false
	}
	return m, nil
}

func (m PageInspectorModel) ensureDictVisible() PageInspectorModel {
	vis := m.dictVisibleLines()
	if m.dictCursor < m.dictOffset {
		m.dictOffset = m.dictCursor
	}
	if m.dictCursor >= m.dictOffset+vis {
		m.dictOffset = m.dictCursor - vis + 1
	}
	if m.dictOffset < 0 {
		m.dictOffset = 0
	}
	return m
}

func (m PageInspectorModel) dictVisibleLines() int {
	// height minus chrome(3) minus header(3: title + colheaders + sep)
	vis := m.height - 3 - 3
	if vis < 1 {
		vis = 1
	}
	return vis
}

func (m PageInspectorModel) ensureValueVisible() PageInspectorModel {
	vis := m.valueViewerHeight()
	if m.valueCursor < m.valueOffset {
		m.valueOffset = m.valueCursor
	}
	if m.valueCursor >= m.valueOffset+vis {
		m.valueOffset = m.valueCursor - vis + 1
	}
	if m.valueOffset < 0 {
		m.valueOffset = 0
	}
	return m
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
	if m.showingDict {
		vm := m.BuildDictionaryVM()
		return view.RenderDictionary(vm)
	}
	vm := m.BuildViewModel()
	return view.RenderPageInspector(vm)
}

// BuildDictionaryVM creates the dictionary overlay view model.
func (m PageInspectorModel) BuildDictionaryVM() ui.DictionaryVM {
	if m.dictResult == nil {
		return ui.DictionaryVM{}
	}

	visible := m.dictVisibleLines()
	end := m.dictOffset + visible
	if end > len(m.dictResult.Entries) {
		end = len(m.dictResult.Entries)
	}

	entries := make([]ui.DictEntryVM, 0, end-m.dictOffset)
	for i := m.dictOffset; i < end; i++ {
		e := m.dictResult.Entries[i]
		pct := float64(0)
		if m.dictResult.Total > 0 {
			pct = float64(e.Count) / float64(m.dictResult.Total) * 100
		}
		entries = append(entries, ui.DictEntryVM{
			Index:   i,
			Value:   e.Value,
			Count:   e.Count,
			Percent: pct,
		})
	}

	// Build all entries for the distribution chart.
	allEntries := make([]ui.DictEntryVM, len(m.dictResult.Entries))
	for i, e := range m.dictResult.Entries {
		pct := float64(0)
		if m.dictResult.Total > 0 {
			pct = float64(e.Count) / float64(m.dictResult.Total) * 100
		}
		allEntries[i] = ui.DictEntryVM{
			Index:   i,
			Value:   e.Value,
			Count:   e.Count,
			Percent: pct,
		}
	}

	return ui.DictionaryVM{
		ColumnPath: m.dictResult.Path,
		Entries:    entries,
		AllEntries: allEntries,
		Total:      m.dictResult.Total,
		NumEntries: len(m.dictResult.Entries),
		Cursor:     m.dictCursor - m.dictOffset,
		Offset:     m.dictOffset,
		Width:      m.width,
		Height:     m.height,
	}
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
		RGIndex:         m.rgIndex,
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
		SelectedValue:   m.valueCursor - m.valueOffset,
		TotalPageValues: m.totalPageVals,
		ViewingValues:   m.viewingValues,
		Width:           m.width,
		Height:          m.height,
	}
}
