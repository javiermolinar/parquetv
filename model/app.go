package model

import (
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/javiermolinar/parquetv/engine"
	"github.com/javiermolinar/parquetv/ui"
	"github.com/javiermolinar/parquetv/view"
)

// Navigation messages.
type enterRowGroupMsg struct{ index int }
type backToOverviewMsg struct{}
type enterPageInspectorMsg struct{ colIndex int }
type backToGridMsg struct{}
type switchColumnMsg struct{ colIndex int }

// App is the root BubbleTea model. It manages the navigation stack.
type App struct {
	file          *engine.File
	overview      FileOverviewModel
	grid          RowGroupGridModel
	pageInspector PageInspectorModel
	level         int // 0=overview, 1=grid, 2=page inspector
	width         int
	height        int
}

// NewApp creates the root application model.
func NewApp(file *engine.File) App {
	return App{
		file:     file,
		overview: NewFileOverviewModel(file),
	}
}

func (a App) Init() tea.Cmd {
	return nil
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.overview.width = msg.Width
		a.overview.height = msg.Height
		if a.level == 1 {
			a.grid.width = msg.Width
			a.grid.height = msg.Height
		}
		if a.level == 2 {
			a.pageInspector.width = msg.Width
			a.pageInspector.height = msg.Height
		}
		return a, nil

	case enterRowGroupMsg:
		grid, err := NewRowGroupGridModel(a.file, msg.index, a.width, a.height)
		if err != nil {
			return a, nil
		}
		a.grid = grid
		a.level = 1
		return a, nil

	case backToOverviewMsg:
		a.level = 0
		return a, nil

	case enterPageInspectorMsg:
		pi, err := NewPageInspectorModel(a.file, a.grid.reader, a.grid.rgIndex, msg.colIndex, a.width, a.height)
		if err != nil {
			return a, nil
		}
		a.pageInspector = pi
		a.level = 2
		return a, nil

	case backToGridMsg:
		a.level = 1
		return a, nil

	case switchColumnMsg:
		pi, err := NewPageInspectorModel(a.file, a.grid.reader, a.grid.rgIndex, msg.colIndex, a.width, a.height)
		if err != nil {
			return a, nil
		}
		a.pageInspector = pi
		return a, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return a, tea.Quit
		}
	}

	// Delegate to active model.
	switch a.level {
	case 2:
		updated, cmd := a.pageInspector.Update(msg)
		a.pageInspector = updated.(PageInspectorModel)
		return a, cmd
	case 1:
		updated, cmd := a.grid.Update(msg)
		a.grid = updated.(RowGroupGridModel)
		return a, cmd
	default:
		updated, cmd := a.overview.Update(msg)
		a.overview = updated.(FileOverviewModel)
		return a, cmd
	}
}

func (a App) View() string {
	switch a.level {
	case 2:
		return a.pageInspector.View()
	case 1:
		vm := a.grid.BuildViewModel()
		return view.RenderRowGroupGrid(vm)
	default:
		vm := a.overview.BuildViewModel()
		return view.RenderFileOverview(vm)
	}
}

// FileOverviewModel handles the file overview screen.
type FileOverviewModel struct {
	file     *engine.File
	selected int

	// Schema tree focus.
	schemaFocused bool
	schemaCursor  int // cursor line in flattened tree
	schemaOffset  int // first visible line
	schemaLines   int // total flattened lines (computed once)

	width  int
	height int
}

// SetSize sets the terminal dimensions for rendering.
func (m *FileOverviewModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// NewFileOverviewModel creates a file overview model from an open file.
func NewFileOverviewModel(file *engine.File) FileOverviewModel {
	m := FileOverviewModel{
		file:     file,
		selected: 0,
	}
	// Precompute flattened schema line count.
	if info := file.Info(); info.Schema != nil {
		vm := buildSchemaTreeVM(info.Schema, 0)
		m.schemaLines = countSchemaLines(vm)
	}
	return m
}

func countSchemaLines(node *ui.SchemaNodeVM) int {
	if node == nil {
		return 0
	}
	n := 0
	for _, child := range node.Children {
		n++ // the node itself
		if !child.Leaf {
			n += countSchemaLines(child)
		}
	}
	return n
}

func (m FileOverviewModel) Init() tea.Cmd {
	return nil
}

func (m FileOverviewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.schemaFocused {
			return m.updateSchemaFocused(msg)
		}
		info := m.file.Info()
		switch msg.String() {
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < info.NumRowGroups-1 {
				m.selected++
			}
		case "g":
			m.selected = 0
		case "G":
			m.selected = info.NumRowGroups - 1
		case "enter":
			return m, func() tea.Msg { return enterRowGroupMsg{index: m.selected} }
		case "tab":
			if m.schemaLines > 0 {
				m.schemaFocused = true
			}
		}
	}
	return m, nil
}

func (m FileOverviewModel) updateSchemaFocused(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	visible := m.schemaVisibleLines()
	maxCursor := m.schemaLines - 1
	if maxCursor < 0 {
		maxCursor = 0
	}

	switch msg.String() {
	case "down", "j":
		if m.schemaCursor < maxCursor {
			m.schemaCursor++
			m = m.ensureSchemaVisible()
		}
	case "up", "k":
		if m.schemaCursor > 0 {
			m.schemaCursor--
			m = m.ensureSchemaVisible()
		}
	case "g":
		m.schemaCursor = 0
		m = m.ensureSchemaVisible()
	case "G":
		m.schemaCursor = maxCursor
		m = m.ensureSchemaVisible()
	case "ctrl+d":
		m.schemaCursor += visible / 2
		if m.schemaCursor > maxCursor {
			m.schemaCursor = maxCursor
		}
		m = m.ensureSchemaVisible()
	case "ctrl+u":
		m.schemaCursor -= visible / 2
		if m.schemaCursor < 0 {
			m.schemaCursor = 0
		}
		m = m.ensureSchemaVisible()
	case "esc", "tab":
		m.schemaFocused = false
	}
	return m, nil
}

func (m FileOverviewModel) ensureSchemaVisible() FileOverviewModel {
	vis := m.schemaVisibleLines()
	if m.schemaCursor < m.schemaOffset {
		m.schemaOffset = m.schemaCursor
	}
	if m.schemaCursor >= m.schemaOffset+vis {
		m.schemaOffset = m.schemaCursor - vis + 1
	}
	if m.schemaOffset < 0 {
		m.schemaOffset = 0
	}
	return m
}

func (m FileOverviewModel) schemaVisibleLines() int {
	// height minus chrome(3) minus 1 for the "Schema" header.
	vis := m.height - 3 - 1
	if vis < 1 {
		vis = 1
	}
	return vis
}

func (m FileOverviewModel) View() string {
	vm := m.BuildViewModel()
	return view.RenderFileOverview(vm)
}

// BuildViewModel converts model state into a view model.
func (m FileOverviewModel) BuildViewModel() ui.FileOverviewVM {
	info := m.file.Info()

	format := ""
	for k, v := range info.KeyValues {
		if k == "tempo.block.format" || k == "parquet.format" {
			format = v
			break
		}
	}

	groups := make([]ui.RowGroupSummary, len(info.RowGroups))
	for i, rg := range info.RowGroups {
		avgPages := 0
		if rg.NumColumns > 0 {
			total := 0
			for _, ci := range rg.ColumnInfos {
				total += ci.NumPages
			}
			avgPages = total / rg.NumColumns
		}
		groups[i] = ui.RowGroupSummary{
			Index:       rg.Index,
			NumRows:     rg.NumRows,
			TotalBytes:  rg.TotalBytes,
			AvgPagesCol: avgPages,
		}
	}

	shortcuts := []string{"enter open", "Tab schema", "q quit"}
	if m.schemaFocused {
		shortcuts = []string{"↑↓ scroll", "g/G jump", "Tab/esc back"}
	}

	return ui.FileOverviewVM{
		TopBar: ui.TopBarData{
			FileName:     filepath.Base(info.Path),
			FileSize:     info.Size,
			Format:       format,
			TotalRows:    info.NumRows,
			NumRowGroups: info.NumRowGroups,
			NumColumns:   info.NumColumns,
		},
		BottomBar: ui.BottomBarData{
			Breadcrumb: filepath.Base(info.Path),
			Shortcuts:  shortcuts,
		},
		RowGroups:   groups,
		Selected:    m.selected,
		FooterPanel: ui.FooterData{
			NumLeafColumns: info.NumColumns,
			Format:         format,
			TopColumns:     info.TopColumns,
			KeyValues:      info.KeyValues,
		},
		SchemaTree:    buildSchemaTreeVM(info.Schema, 0),
		SchemaFocused: m.schemaFocused,
		SchemaCursor:  m.schemaCursor,
		SchemaOffset:  m.schemaOffset,
		Width:         m.width,
		Height:        m.height,
	}
}

// buildSchemaTreeVM converts engine schema nodes to view model nodes.
// Skips list/element wrapper nodes (Parquet nested encoding noise).
func buildSchemaTreeVM(node *engine.SchemaNode, depth int) *ui.SchemaNodeVM {
	if node == nil {
		return nil
	}
	vm := &ui.SchemaNodeVM{
		Name:     node.Name,
		Type:     node.Type,
		Depth:    depth,
		Leaf:     node.Leaf,
		Expanded: true,
	}
	for _, child := range node.Children {
		for _, resolved := range resolveSchemaChildren(child) {
			vm.Children = append(vm.Children, buildSchemaTreeVM(resolved, depth+1))
		}
	}
	return vm
}

// resolveSchemaChildren skips list/element wrapper nodes and returns
// their meaningful children, flattening the Parquet nesting noise.
func resolveSchemaChildren(node *engine.SchemaNode) []*engine.SchemaNode {
	if node.Name == "list" || node.Name == "element" {
		var result []*engine.SchemaNode
		for _, child := range node.Children {
			result = append(result, resolveSchemaChildren(child)...)
		}
		return result
	}
	return []*engine.SchemaNode{node}
}
