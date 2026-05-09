package view

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/javiermolinar/parquetv/ui"
)

const (
	gridRowNumWidth   = 7
	gridColPadLeft    = 2  // left padding inside each data column
	gridNavPanelWidth = 38 // must match model.navPanelWidth
)

// RenderRowGroupGrid renders the row group grid screen.
func RenderRowGroupGrid(vm ui.RowGroupGridVM) string {
	width := vm.Width
	if width < 40 {
		width = 80
	}
	height := vm.Height
	if height < 10 {
		height = 24
	}

	topBar := RenderTopBar(vm.TopBar, width)
	bottomBar := RenderBottomBar(vm.BottomBar, width)

	// Determine inspect panel height.
	inspectLines := 3
	if vm.Inspect.Focused {
		inspectLines = 8
	}

	// Two-panel area: nav tree (left) + grid (right).
	panelHeight := height - 2 - 1 - inspectLines - 1 // top(2) + statsSep(1) + inspect(N) + bottom(1)
	if panelHeight < 1 {
		panelHeight = 1
	}

	navWidth := gridNavPanelWidth
	navContent := renderGridNavPanel(vm, navWidth, panelHeight)
	leftPanel := leftPanelStyle.Width(navWidth).Height(panelHeight).Render(navContent)

	gridWidth := width - navWidth - 3
	headerLine := renderGridHeader(vm.Headers, vm.ColWidths, vm.SelectedCol)
	sepLine := renderGridSeparator(vm.ColWidths)
	dataLines := renderGridRows(vm)

	var gridParts []string
	gridParts = append(gridParts, headerLine, sepLine)
	gridParts = append(gridParts, dataLines...)

	for len(gridParts) < panelHeight {
		gridParts = append(gridParts, "")
	}
	if len(gridParts) > panelHeight {
		gridParts = gridParts[:panelHeight]
	}

	gridContent := lipgloss.JoinVertical(lipgloss.Left, gridParts...)
	rightPanel := lipgloss.NewStyle().Width(gridWidth).Height(panelHeight).PaddingLeft(1).Render(gridContent)

	middleSection := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)

	statsSep := dimText.Render(strings.Repeat("─", width))
	inspectBar := renderInspectPanel(vm.Inspect, vm.Stats, width)

	return lipgloss.JoinVertical(lipgloss.Left,
		topBar,
		middleSection,
		statsSep,
		inspectBar,
		bottomBar,
	)
}

// renderGridNavPanel renders the left navigation tree for the grid.
func renderGridNavPanel(vm ui.RowGroupGridVM, width, height int) string {
	var items []string

	items = append(items, accentText.Render(fmt.Sprintf("▸ Row Group %d", vm.RGIndex)))
	items = append(items, dimText.Render(fmt.Sprintf("    %s rows  %s",
		FormatNumber(vm.TotalRows), FormatBytes(vm.RGBytes))))
	items = append(items, "")
	items = append(items, dimText.Render(strings.Repeat("─", width-2)))

	cursorRow := vm.RowOffset + int64(vm.SelectedRow)
	items = append(items, normalText.Render(fmt.Sprintf("  row %s / %s",
		FormatNumber(cursorRow), FormatNumber(vm.TotalRows))))

	colName := ""
	if vm.SelectedCol >= 0 && vm.SelectedCol < len(vm.Headers) {
		colName = vm.Headers[vm.SelectedCol]
	}
	items = append(items, accentText.Render(fmt.Sprintf("  col: [%s]",
		truncate(colName, width-10))))
	items = append(items, "")

	if vm.Stats.Path != "" {
		items = append(items, dimText.Render(fmt.Sprintf("  %s", vm.Stats.Type)))
		items = append(items, dimText.Render(fmt.Sprintf("  %s  %d pages",
			FormatBytes(vm.Stats.TotalBytes), vm.Stats.NumPages)))
		items = append(items, dimText.Render(fmt.Sprintf("  %s values",
			FormatNumber(vm.Stats.NumValues))))
		items = append(items, dimText.Render(fmt.Sprintf("  per-row: %.1f",
			vm.Stats.ValuesPerRow)))
	}

	for len(items) < height {
		items = append(items, "")
	}

	return lipgloss.JoinVertical(lipgloss.Left, items...)
}

// renderGridHeader renders the column header line.
func renderGridHeader(headers []string, colWidths []int, selectedCol int) string {
	cells := []string{
		gridRowNumStyle.Width(gridRowNumWidth).Render("Row"),
	}
	for i, h := range headers {
		w := gridRowNumWidth // fallback
		if i < len(colWidths) {
			w = colWidths[i]
		}
		if i == selectedCol {
			text := "[" + truncate(h, w-gridColPadLeft-4) + "]"
			cells = append(cells, gridSelectedHeader.Width(w).PaddingLeft(gridColPadLeft).Render(text))
		} else {
			cells = append(cells, gridHeaderStyle.Width(w).PaddingLeft(gridColPadLeft).Render(truncate(h, w-gridColPadLeft-1)))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, cells...)
}

// renderGridSeparator renders the separator line under headers.
func renderGridSeparator(colWidths []int) string {
	cells := []string{
		dimText.Width(gridRowNumWidth).Render(strings.Repeat("─", gridRowNumWidth-1)),
	}
	for _, w := range colWidths {
		cells = append(cells, dimText.Width(w).Render(strings.Repeat("─", w-1)))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, cells...)
}

// renderGridRows renders data rows with page boundaries interleaved.
func renderGridRows(vm ui.RowGroupGridVM) []string {
	// Build a set of viewport-relative indices that have page boundaries before them.
	pageBoundSet := make(map[int]bool, len(vm.PageBounds))
	for _, pb := range vm.PageBounds {
		pageBoundSet[pb] = true
	}

	var lines []string
	for i, row := range vm.Rows {
		// Insert page boundary dashed line before this row if needed.
		if pageBoundSet[i] {
			lines = append(lines, renderPageBoundaryLine(vm.ColWidths))
		}

		isSelectedRow := i == vm.SelectedRow
		lines = append(lines, renderGridDataRow(row, vm.ColWidths, vm.RightAlign, vm.SelectedCol, isSelectedRow))
	}
	return lines
}

// renderGridDataRow renders a single data row.
func renderGridDataRow(row ui.GridRow, colWidths []int, rightAlign []bool, selectedCol int, isSelectedRow bool) string {
	// Row number cell.
	rowNumStyle := gridRowNumStyle.Width(gridRowNumWidth)
	if isSelectedRow {
		rowNumStyle = rowNumStyle.Background(colorSelected)
	}
	cells := []string{
		rowNumStyle.Render(fmt.Sprintf("%d", row.Index)),
	}

	for i, v := range row.Values {
		w := gridRowNumWidth // fallback
		if i < len(colWidths) {
			w = colWidths[i]
		}
		text := truncate(v, w-gridColPadLeft-1)

		var style lipgloss.Style
		switch {
		case isSelectedRow && i == selectedCol:
			style = gridCursorCell.Width(w).PaddingLeft(gridColPadLeft)
		case isSelectedRow:
			style = gridSelectedRowCell.Width(w).PaddingLeft(gridColPadLeft)
		case i == selectedCol:
			style = gridSelectedColCell.Width(w).PaddingLeft(gridColPadLeft)
		default:
			style = gridCellStyle.Width(w).PaddingLeft(gridColPadLeft)
		}
		if i < len(rightAlign) && rightAlign[i] {
			style = style.Align(lipgloss.Right).PaddingRight(1)
		}
		cells = append(cells, style.Render(text))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, cells...)
}

// renderPageBoundaryLine renders a dashed separator at a page boundary.
func renderPageBoundaryLine(colWidths []int) string {
	totalWidth := gridRowNumWidth
	for _, w := range colWidths {
		totalWidth += w
	}
	dash := " ┄"
	repeated := strings.Repeat(dash, totalWidth/len(dash))
	return pageBoundaryStyle.Render(truncate(repeated, totalWidth))
}

// renderInspectPanel renders the combined cell inspect + column stats area.
// Compact (3 lines) when unfocused; expanded (8 lines) with scrollable values when focused.
func renderInspectPanel(inspect ui.CellInspectVM, stats ui.ColumnStatsVM, width int) string {
	if stats.Path == "" {
		return strings.Repeat("\n", 2)
	}

	if inspect.Focused {
		return renderInspectFocused(inspect, stats, width)
	}
	return renderInspectCompact(inspect, stats, width)
}

// renderInspectCompact renders the 3-line compact inspect panel.
func renderInspectCompact(inspect ui.CellInspectVM, stats ui.ColumnStatsVM, width int) string {
	rightWidth := 45
	leftWidth := width - rightWidth - 3
	if leftWidth < 30 {
		leftWidth = width - 5
		rightWidth = 0
	}

	divider := dimText.Render(" │ ")

	// Line 1: path (row N)
	headerText := fmt.Sprintf("%s (row %d)", inspect.ColumnPath, inspect.RowIndex)
	if inspect.RepCount > 1 {
		headerText += fmt.Sprintf("  %d values", inspect.RepCount)
	}
	headerLeft := statsPathStyle.Render(headerText)

	// Line 2: full value
	valDisplay := inspect.Value
	if valDisplay == "" {
		valDisplay = "null"
	}
	valueLeft := normalText.Render(truncate(valDisplay, leftWidth-1))

	// Line 3: hex dump
	hexLeft := dimText.Render(truncate(inspect.HexDump, leftWidth-1))

	if rightWidth == 0 {
		return lipgloss.JoinVertical(lipgloss.Left, headerLeft, valueLeft, hexLeft)
	}

	statsLine1 := statsDetailStyle.Render(
		fmt.Sprintf("type: %s  size: %s", stats.Type, FormatBytes(stats.TotalBytes)),
	)
	statsLine2 := statsDetailStyle.Render(
		fmt.Sprintf("values: %s  pages: %d  per-row: %.1f",
			FormatNumber(stats.NumValues), stats.NumPages, stats.ValuesPerRow),
	)

	bytesInfo := ""
	if inspect.ByteLen > 0 {
		bytesInfo = fmt.Sprintf("%d bytes", inspect.ByteLen)
	}
	if inspect.RepCount > 1 {
		bytesInfo += "  Tab ↹ expand"
	}
	statsLine3 := statsDetailStyle.Render(bytesInfo)

	joinLine := func(left string, right string) string {
		return lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(leftWidth).Render(left),
			divider, right,
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		joinLine(headerLeft, statsLine1),
		joinLine(valueLeft, statsLine2),
		joinLine(hexLeft, statsLine3),
	)
}

// renderInspectFocused renders the expanded inspect panel with a scrollable value list.
func renderInspectFocused(inspect ui.CellInspectVM, stats ui.ColumnStatsVM, width int) string {
	rightWidth := 45
	leftWidth := width - rightWidth - 3
	if leftWidth < 30 {
		leftWidth = width - 5
		rightWidth = 0
	}

	divider := dimText.Render(" │ ")

	joinLine := func(left string, right string) string {
		if rightWidth == 0 {
			return left
		}
		return lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(leftWidth).Render(left),
			divider, right,
		)
	}

	// Line 1: header + stats
	headerText := fmt.Sprintf("%s (row %d)  %d values", inspect.ColumnPath, inspect.RowIndex, inspect.RepCount)
	headerLeft := statsPathStyle.Render(headerText)
	statsRight := statsDetailStyle.Render(
		fmt.Sprintf("type: %s  size: %s  per-row: %.1f",
			stats.Type, FormatBytes(stats.TotalBytes), stats.ValuesPerRow),
	)

	var lines []string
	lines = append(lines, joinLine(headerLeft, statsRight))

	// Value lines (scrollable).
	visible := inspect.VisibleLines
	if visible < 1 {
		visible = 7
	}

	for i := 0; i < visible; i++ {
		idx := inspect.ScrollOffset + i
		if idx < len(inspect.AllValues) {
			num := fmt.Sprintf("%3d  ", idx+1)
			val := truncate(inspect.AllValues[idx], leftWidth-7)
			line := accentText.Render(num) + normalText.Render(val)
			lines = append(lines, joinLine(line, ""))
		} else {
			lines = append(lines, joinLine("", ""))
		}
	}

	// Show scroll position on the last value line's right side.
	if rightWidth > 0 && len(lines) > 1 {
		remaining := len(inspect.AllValues) - inspect.ScrollOffset - visible
		if remaining < 0 {
			remaining = 0
		}
		scrollHint := ""
		if remaining > 0 {
			scrollHint = fmt.Sprintf("▼ %d more", remaining)
		}
		if inspect.ScrollOffset > 0 {
			if scrollHint != "" {
				scrollHint = fmt.Sprintf("▲ %d above  %s", inspect.ScrollOffset, scrollHint)
			} else {
				scrollHint = fmt.Sprintf("▲ %d above", inspect.ScrollOffset)
			}
		}
		// Replace last line to include scroll hint on the right.
		lastIdx := len(lines) - 1
		lastLeft := ""
		idx := inspect.ScrollOffset + visible - 1
		if idx < len(inspect.AllValues) {
			num := fmt.Sprintf("%3d  ", idx+1)
			val := truncate(inspect.AllValues[idx], leftWidth-7)
			lastLeft = accentText.Render(num) + normalText.Render(val)
		}
		lines[lastIdx] = joinLine(lastLeft, dimText.Render(scrollHint))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}
