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

// App is the root BubbleTea model. It manages the navigation stack.
type App struct {
	file     *engine.File
	overview FileOverviewModel
	grid     RowGroupGridModel
	level    int // 0=overview, 1=grid
	width    int
	height   int
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

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return a, tea.Quit
		}
	}

	// Delegate to active model.
	if a.level == 1 {
		updated, cmd := a.grid.Update(msg)
		a.grid = updated.(RowGroupGridModel)
		return a, cmd
	}

	updated, cmd := a.overview.Update(msg)
	a.overview = updated.(FileOverviewModel)
	return a, cmd
}

func (a App) View() string {
	if a.level == 1 {
		vm := a.grid.BuildViewModel()
		return view.RenderRowGroupGrid(vm)
	}
	vm := a.overview.BuildViewModel()
	return view.RenderFileOverview(vm)
}

// FileOverviewModel handles the file overview screen.
type FileOverviewModel struct {
	file     *engine.File
	selected int
	width    int
	height   int
}

// SetSize sets the terminal dimensions for rendering.
func (m *FileOverviewModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// NewFileOverviewModel creates a file overview model from an open file.
func NewFileOverviewModel(file *engine.File) FileOverviewModel {
	return FileOverviewModel{
		file:     file,
		selected: 0,
	}
}

func (m FileOverviewModel) Init() tea.Cmd {
	return nil
}

func (m FileOverviewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
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
		}
	}
	return m, nil
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
			Shortcuts:  []string{"enter open", "s schema", "q quit", "? help"},
		},
		RowGroups:   groups,
		Selected:    m.selected,
		FooterPanel: ui.FooterData{
			NumLeafColumns: info.NumColumns,
			Format:         format,
			TopColumns:     info.TopColumns,
			KeyValues:      info.KeyValues,
		},
		Width:  m.width,
		Height: m.height,
	}
}
