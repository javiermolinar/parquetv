package view

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/javiermolinar/parquetv/ui"
)

// RenderDictionary renders the dictionary view as a full-screen overlay.
func RenderDictionary(vm ui.DictionaryVM) string {
	width := vm.Width
	if width < 40 {
		width = 80
	}
	height := vm.Height
	if height < 10 {
		height = 24
	}

	// Top bar.
	topBar := renderDictTopBar(vm, width)

	// Bottom bar.
	bottomBar := RenderBottomBar(ui.BottomBarData{
		Breadcrumb: fmt.Sprintf("Dictionary: %s", vm.ColumnPath),
		Shortcuts:  []string{"↑↓ scroll", "g/G jump", "esc back"},
	}, width)

	// Content area.
	contentHeight := height - 3 // top(2) + bottom(1)

	// Column header.
	numW := 7
	valueW := width/2 - numW
	if valueW < 20 {
		valueW = 20
	}
	countW := 14
	pctW := 8
	barW := width - numW - valueW - countW - pctW
	if barW < 0 {
		barW = 0
	}

	header := lipgloss.JoinHorizontal(lipgloss.Top,
		dictNumStyle.Width(numW).Render("#"),
		dictHeaderStyle.Width(valueW).Render("value"),
		dictHeaderStyle.Width(countW).Align(lipgloss.Right).Render("count"),
		dictHeaderStyle.Width(pctW).Align(lipgloss.Right).Render("%"),
		dictHeaderStyle.Width(barW).PaddingLeft(2).Render(""),
	)

	sep := dimText.Render(strings.Repeat("─", width))

	var lines []string
	lines = append(lines, header, sep)

	// Find max count for bar scaling.
	maxCount := int64(1)
	if len(vm.Entries) > 0 {
		maxCount = vm.Entries[0].Count // already sorted desc by model
	}

	for i, e := range vm.Entries {
		isCursor := i == vm.Cursor

		nStyle := dictNumStyle
		vStyle := dictValueStyle
		cStyle := dictCountStyle
		pStyle := dictPctStyle
		if isCursor {
			nStyle = dictCursorNum
			vStyle = dictCursorValue
			cStyle = dictCursorCount
			pStyle = dictCursorPct
		}

		numCell := nStyle.Width(numW).Render(fmt.Sprintf("%d", e.Index))
		valCell := vStyle.Width(valueW).Render(truncate(e.Value, valueW-2))
		countCell := cStyle.Width(countW).Align(lipgloss.Right).Render(FormatNumber(e.Count))
		pctCell := pStyle.Width(pctW).Align(lipgloss.Right).Render(fmt.Sprintf("%.1f", e.Percent))

		// Frequency bar.
		var barCell string
		if barW > 2 {
			barLen := int(float64(e.Count) / float64(maxCount) * float64(barW-2))
			if barLen < 0 {
				barLen = 0
			}
			bar := strings.Repeat("█", barLen)
			if isCursor {
				barCell = dictCursorBar.Width(barW).PaddingLeft(2).Render(bar)
			} else {
				barCell = dictBarStyle.Width(barW).PaddingLeft(2).Render(bar)
			}
		}

		lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Top, numCell, valCell, countCell, pctCell, barCell))
	}

	// Pad.
	for len(lines) < contentHeight {
		lines = append(lines, "")
	}
	if len(lines) > contentHeight {
		lines = lines[:contentHeight]
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

	return lipgloss.JoinVertical(lipgloss.Left, topBar, content, bottomBar)
}

func renderDictTopBar(vm ui.DictionaryVM, width int) string {
	title := topBarTitle.Inherit(topBarStyle).Render("parquetv")
	sep := lipgloss.NewStyle().Inherit(topBarStyle).Foreground(colorMuted).Render(" ─── ")
	name := lipgloss.NewStyle().Inherit(topBarStyle).Foreground(colorPrimary).Render("Dictionary")

	line1 := topBarStyle.Width(width).Render(" " + title + sep + name)

	context := fmt.Sprintf("%s  %d entries  %s total values",
		vm.ColumnPath, vm.NumEntries, FormatNumber(vm.Total))
	line2 := topBarStyle.Width(width).Render(" " + topBarDim.Inherit(topBarStyle).Render(context))

	return lipgloss.JoinVertical(lipgloss.Left, line1, line2)
}
