package view

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/javiermolinar/parquetv/ui"
)

const (
	pageNavPanelWidth = 38 // must match model.navPanelWidth
	pageValueNumWidth = 7  // width for "#" column in value viewer
	pageValueColWidth = 24 // width for "value" column
	pageValueHexWidth = 50 // width for "hex" column
)

// RenderPageInspector renders the page inspector screen (Level 3).
func RenderPageInspector(vm ui.PageInspectorVM) string {
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

	// Chrome takes 3 lines (top=2, bottom=1).
	contentHeight := height - 3

	// Two-panel layout — same nav panel width as grid.
	leftWidth := pageNavPanelWidth
	rightWidth := width - leftWidth - 3 // divider + padding

	left := renderPageList(vm, leftWidth-1, contentHeight) // -1 for border
	right := renderValuePanel(vm, rightWidth-1, contentHeight) // -1 for PaddingLeft(1)

	leftPanel := leftPanelStyle.
		Width(leftWidth).
		Height(contentHeight).
		Render(left)

	rightPanel := lipgloss.NewStyle().
		Width(rightWidth).
		Height(contentHeight).
		PaddingLeft(1).
		Render(right)

	content := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)

	return lipgloss.JoinVertical(lipgloss.Left, topBar, content, bottomBar)
}

// pageEntryLines returns how many lines a page entry occupies.
func pageEntryLines(page ui.PageSummaryVM) int {
	lines := 3 // name + rows + size
	if page.MinValue != "" || page.MaxValue != "" {
		lines += 2 // min + max
	}
	return lines
}

// renderPageList renders the left panel with a windowed view of the page list.
func renderPageList(vm ui.PageInspectorVM, width, height int) string {
	var items []string

	// Hierarchy context + column chain.
	items = append(items, dimText.Render(fmt.Sprintf("Row Group %d", vm.RGIndex)))

	// Column navigation: prev ‹ [current] › next
	if vm.PrevColumn != "" {
		items = append(items, dimText.Render(truncate(vm.PrevColumn, width-2))+" ")
	}
	items = append(items, accentText.Render(fmt.Sprintf("[%s]", truncate(vm.ColumnPath, width-4))))
	if vm.NextColumn != "" {
		items = append(items, dimText.Render(truncate(vm.NextColumn, width-2)))
	}

	items = append(items, "")
	items = append(items, headerText.Render("Pages"))

	usedLines := len(items)

	for i, page := range vm.Pages {
		// Skip pages before the scroll offset.
		if i < vm.PageOffset {
			continue
		}

		// Stop if we'd exceed the available height.
		entryLines := pageEntryLines(page)
		if usedLines+entryLines > height {
			break
		}

		isSelected := page.Index == vm.SelectedPage

		// Page header line with optional pushdown badge.
		name := fmt.Sprintf("Page %d", page.Index)
		var badge string
		switch page.Pushdown {
		case "READ":
			badge = " " + pushdownReadStyle.Render("READ")
		case "SKIP":
			badge = " " + pushdownSkipStyle.Render("SKIP")
		}
		if isSelected {
			name = accentText.Render("▸ ") + selectedRow.Render(name) + badge
		} else {
			name = "  " + normalText.Render(name) + badge
		}
		items = append(items, name)
		usedLines++

		// Stats lines (indented).
		valLine := fmt.Sprintf("    rows: %s", FormatNumber(page.NumValues))
		items = append(items, dimText.Render(valLine))
		usedLines++

		if page.MinValue != "" || page.MaxValue != "" {
			minMax := fmt.Sprintf("    min: %s", truncate(page.MinValue, width-8))
			items = append(items, dimText.Render(minMax))
			maxLine := fmt.Sprintf("    max: %s", truncate(page.MaxValue, width-8))
			items = append(items, dimText.Render(maxLine))
			usedLines += 2
		}

		sizeLine := fmt.Sprintf("    size: %s", FormatBytes(page.CompressedSize))
		items = append(items, dimText.Render(sizeLine))
		usedLines++
	}

	// Scroll indicator if more pages below.
	rendered := 0
	testLines := usedLines
	for i := vm.PageOffset; i < len(vm.Pages); i++ {
		el := pageEntryLines(vm.Pages[i])
		if testLines+el > height {
			break
		}
		testLines += el
		rendered++
	}
	lastVisible := vm.PageOffset + rendered - 1
	if lastVisible < len(vm.Pages)-1 {
		remaining := len(vm.Pages) - lastVisible - 1
		items = append(items, dimText.Render(fmt.Sprintf("  ▼ %d more pages", remaining)))
	}

	// Pad to fill height.
	for len(items) < height {
		items = append(items, "")
	}

	return lipgloss.JoinVertical(lipgloss.Left, items...)
}

// renderValuePanel renders the right panel with page values in hex-editor style.
// Shows: index | decoded value | hex dump.
func renderValuePanel(vm ui.PageInspectorVM, width, height int) string {
	if len(vm.Pages) == 0 {
		return headerText.Render("No pages")
	}

	var items []string

	// Header.
	title := fmt.Sprintf("Page %d values", vm.SelectedPage)
	if vm.TotalPageValues > 0 {
		title += fmt.Sprintf("  (%s total)", FormatNumber(vm.TotalPageValues))
	}
	items = append(items, headerText.Render(title))

	// Compute column widths based on available space.
	hexW := width - pageValueNumWidth - pageValueColWidth
	if hexW < 20 {
		hexW = 0 // hide hex column on narrow terminals
	}
	valW := pageValueColWidth
	if hexW == 0 {
		valW = width - pageValueNumWidth // give all space to value
	}

	// Column headers.
	numHeader := pageValueNumStyle.Width(pageValueNumWidth).Render("#")
	valHeader := pageValueHeaderStyle.Width(valW).Render("value")
	if hexW > 0 {
		hexHeader := pageValueHexHeaderStyle.Width(hexW).Render("hex")
		items = append(items, lipgloss.JoinHorizontal(lipgloss.Top, numHeader, valHeader, hexHeader))
	} else {
		items = append(items, lipgloss.JoinHorizontal(lipgloss.Top, numHeader, valHeader))
	}

	// Separator.
	sepWidth := pageValueNumWidth + valW + hexW
	if sepWidth > width {
		sepWidth = width
	}
	items = append(items, dimText.Render(strings.Repeat("─", sepWidth)))

	// Value rows.
	for i, v := range vm.Values {
		isCursor := vm.ViewingValues && i == vm.SelectedValue

		// Pick styles based on cursor.
		numStyle := pageValueNumStyle
		vStyle := pageValueCellStyle
		hStyle := pageValueHexStyle
		if isCursor {
			numStyle = pageValueCursorNum
			vStyle = pageValueCursorCell
			hStyle = pageValueCursorHex
		}

		numCell := numStyle.Width(pageValueNumWidth).Render(fmt.Sprintf("%d", v.Index))
		valText := truncate(v.Value, valW-2)
		valCell := vStyle.Width(valW).Render(valText)

		if hexW > 0 {
			hexText := truncate(v.HexDump, hexW-2)
			hexCell := hStyle.Width(hexW).Render(hexText)
			items = append(items, lipgloss.JoinHorizontal(lipgloss.Top, numCell, valCell, hexCell))
		} else {
			items = append(items, lipgloss.JoinHorizontal(lipgloss.Top, numCell, valCell))
		}
	}

	// Scroll indicator.
	if vm.TotalPageValues > 0 && vm.ViewingValues {
		shown := vm.ValueOffset + len(vm.Values)
		remaining := int(vm.TotalPageValues) - shown
		if remaining > 0 {
			items = append(items, dimText.Render(fmt.Sprintf("  ▼ %s more", FormatNumber(int64(remaining)))))
		}
	}

	// Pad to fill height.
	for len(items) < height {
		items = append(items, "")
	}

	return lipgloss.JoinVertical(lipgloss.Left, items...)
}
