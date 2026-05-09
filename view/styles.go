package view

import "github.com/charmbracelet/lipgloss"

// Colors — single palette, all defined here.
var (
	colorPrimary  = lipgloss.Color("#FFFFFF")
	colorDim      = lipgloss.Color("#808080")
	colorAccent   = lipgloss.Color("#22C55E")
	colorMuted    = lipgloss.Color("#555555")
	colorBarBg    = lipgloss.Color("#2A2A2A")
	colorSelected = lipgloss.Color("#333333")
)

// Top bar styles.
var (
	topBarStyle = lipgloss.NewStyle().
			Background(colorBarBg).
			Foreground(colorPrimary).
			Bold(true).
			Padding(0, 1)

	topBarTitle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	topBarDim = lipgloss.NewStyle().
			Foreground(colorDim)
)

// Bottom bar styles.
var (
	bottomBarStyle = lipgloss.NewStyle().
			Background(colorBarBg).
			Foreground(colorDim).
			Padding(0, 1)

	breadcrumbText = lipgloss.NewStyle().
			Foreground(colorPrimary)

	shortcutKey = lipgloss.NewStyle().
			Foreground(colorAccent)

	shortcutDesc = lipgloss.NewStyle().
			Foreground(colorDim)
)

// Content styles.
var (
	headerText = lipgloss.NewStyle().
			Foreground(colorDim).
			Bold(true)

	selectedRow = lipgloss.NewStyle().
			Background(colorSelected).
			Foreground(colorPrimary)

	normalText = lipgloss.NewStyle().
			Foreground(colorPrimary)

	dimText = lipgloss.NewStyle().
		Foreground(colorMuted)

	accentText = lipgloss.NewStyle().
			Foreground(colorAccent)

	// Left panel with right border for the divider.
	leftPanelStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, true, false, false).
			BorderForeground(colorMuted)
)

// Grid styles (Level 2).
var (
	gridHeaderStyle = lipgloss.NewStyle().
				Foreground(colorDim).
				Bold(true)

	gridSelectedHeader = lipgloss.NewStyle().
					Foreground(colorAccent).
					Bold(true)

	gridRowNumStyle = lipgloss.NewStyle().
				Foreground(colorMuted).
				Align(lipgloss.Right)

	gridCellStyle = lipgloss.NewStyle().
				Foreground(colorPrimary)

	gridSelectedColCell = lipgloss.NewStyle().
					Foreground(colorPrimary).
					Background(colorSelected)

	gridSelectedRowCell = lipgloss.NewStyle().
					Foreground(colorPrimary).
					Background(colorSelected)

	gridCursorCell = lipgloss.NewStyle().
				Foreground(colorAccent).
				Background(colorSelected).
				Bold(true)

	statsPathStyle = lipgloss.NewStyle().
				Foreground(colorAccent).
				Bold(true)

	statsDetailStyle = lipgloss.NewStyle().
					Foreground(colorDim)

	pageBoundaryStyle = lipgloss.NewStyle().
				  Foreground(colorMuted)
)
