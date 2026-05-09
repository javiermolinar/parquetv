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

// Schema tree styles.
var (
	schemaCursorStyle = lipgloss.NewStyle().
				Foreground(colorAccent).
				Background(colorSelected).
				Bold(true)
)

// Dictionary view styles.
var (
	dictNumStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Align(lipgloss.Right)

	dictHeaderStyle = lipgloss.NewStyle().
				Foreground(colorDim).
				Bold(true).
				PaddingLeft(2)

	dictValueStyle = lipgloss.NewStyle().
				Foreground(colorPrimary).
				PaddingLeft(2)

	dictCountStyle = lipgloss.NewStyle().
				Foreground(colorDim)

	dictPctStyle = lipgloss.NewStyle().
			Foreground(colorDim)

	// Cursor row.
	dictCursorNum = lipgloss.NewStyle().
				Foreground(colorAccent).
				Background(colorSelected).
				Align(lipgloss.Right)

	dictCursorValue = lipgloss.NewStyle().
				Foreground(colorAccent).
				Background(colorSelected).
				Bold(true).
				PaddingLeft(2)

	dictCursorCount = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Background(colorSelected)

	dictCursorPct = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Background(colorSelected)
)

// Page inspector styles (Level 3).
var (
	pageValueNumStyle = lipgloss.NewStyle().
				Foreground(colorMuted).
				Align(lipgloss.Right)

	pageValueHeaderStyle = lipgloss.NewStyle().
					Foreground(colorDim).
					Bold(true).
					PaddingLeft(2)

	pageValueCellStyle = lipgloss.NewStyle().
				Foreground(colorPrimary).
				PaddingLeft(2)

	pageValueHexHeaderStyle = lipgloss.NewStyle().
					Foreground(colorDim).
					Bold(true).
					PaddingLeft(1)

	pageValueHexStyle = lipgloss.NewStyle().
				Foreground(colorMuted).
				PaddingLeft(1)

	// Cursor row (value viewer active).
	pageValueCursorNum = lipgloss.NewStyle().
				Foreground(colorAccent).
				Background(colorSelected).
				Align(lipgloss.Right)

	pageValueCursorCell = lipgloss.NewStyle().
				Foreground(colorAccent).
				Background(colorSelected).
				Bold(true).
				PaddingLeft(2)

	pageValueCursorHex = lipgloss.NewStyle().
				Foreground(colorDim).
				Background(colorSelected).
				PaddingLeft(1)
)
