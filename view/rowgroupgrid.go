package view

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/javiermolinar/parquetv/ui"
)

const (
	gridRowNumWidth = 7
	gridColPadLeft  = 2 // left padding inside each data column
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

	// Header line.
	headerLine := renderGridHeader(vm.Headers, vm.ColWidths, vm.SelectedCol)

	// Separator.
	sepLine := renderGridSeparator(vm.ColWidths)

	// Data rows (with page boundary lines interleaved).
	dataLines := renderGridRows(vm)

	// Stats bar.
	statsSep := dimText.Render(strings.Repeat("─", width))
	inspectBar := renderInspectPanel(vm.Inspect, vm.Stats, width)

	// Assemble content: header + sep + data rows.
	var contentParts []string
	contentParts = append(contentParts, headerLine, sepLine)
	contentParts = append(contentParts, dataLines...)

	// Pad to fill available height.
	contentHeight := height - 2 - 1 - 1 - 3 // top(2) + statsSep+inspect(3) + bottom(1)
	for len(contentParts) < contentHeight {
		contentParts = append(contentParts, "")
	}

	// Trim if too tall (page boundary lines might push over).
	if len(contentParts) > contentHeight {
		contentParts = contentParts[:contentHeight]
	}

	content := lipgloss.JoinVertical(lipgloss.Left, contentParts...)

	return lipgloss.JoinVertical(lipgloss.Left,
		topBar,
		content,
		statsSep,
		inspectBar,
		bottomBar,
	)
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
		lines = append(lines, renderGridDataRow(row, vm.ColWidths, vm.SelectedCol, isSelectedRow))
	}
	return lines
}

// renderGridDataRow renders a single data row.
func renderGridDataRow(row ui.GridRow, colWidths []int, selectedCol int, isSelectedRow bool) string {
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

// renderInspectPanel renders the combined cell inspect + column stats area (3 lines).
func renderInspectPanel(inspect ui.CellInspectVM, stats ui.ColumnStatsVM, width int) string {
	if stats.Path == "" {
		return strings.Repeat("\n", 2)
	}

	// Two-column layout: left = cell value, right = column stats.
	rightWidth := 45
	leftWidth := width - rightWidth - 3 // 3 for " │ " divider
	if leftWidth < 30 {
		leftWidth = width - 5
		rightWidth = 0
	}

	divider := dimText.Render(" │ ")

	// Line 1: path (row N) │ type + size
	headerLeft := statsPathStyle.Render(
		fmt.Sprintf("%s (row %d)", inspect.ColumnPath, inspect.RowIndex),
	)

	// Line 2: full value │ values + pages + per-row
	valDisplay := inspect.Value
	if valDisplay == "" {
		valDisplay = "null"
	}
	valueLeft := normalText.Render(truncate(valDisplay, leftWidth-1))

	// Line 3: hex dump │ byte count
	hexLeft := dimText.Render(truncate(inspect.HexDump, leftWidth-1))

	if rightWidth == 0 {
		// Narrow terminal: no right column.
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
		bytesInfo += fmt.Sprintf("  (%d values)", inspect.RepCount)
	}
	statsLine3 := statsDetailStyle.Render(bytesInfo)

	line1 := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(leftWidth).Render(headerLeft),
		divider,
		statsLine1,
	)
	line2 := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(leftWidth).Render(valueLeft),
		divider,
		statsLine2,
	)
	line3 := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(leftWidth).Render(hexLeft),
		divider,
		statsLine3,
	)

	return lipgloss.JoinVertical(lipgloss.Left, line1, line2, line3)
}
