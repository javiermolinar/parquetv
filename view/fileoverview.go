package view

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/javiermolinar/parquetv/ui"
)

// RenderFileOverview renders the file overview screen.
func RenderFileOverview(vm ui.FileOverviewVM) string {
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

	// Two-panel layout: row groups (left), footer (right).
	// Use same nav panel width as grid and page inspector.
	leftWidth := 38
	rightWidth := width - leftWidth - 3 // border + padding

	left := renderRowGroupList(vm.RowGroups, vm.Selected, leftWidth, contentHeight)
	right := renderFooterPanel(vm.FooterPanel, rightWidth, contentHeight)

	// Left panel gets a right border (the divider).
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

func renderRowGroupList(groups []ui.RowGroupSummary, selected, width, height int) string {
	var items []string

	items = append(items, headerText.Render("Row Groups"))

	for i, rg := range groups {
		name := fmt.Sprintf("Row Group %d", rg.Index)
		detail := fmt.Sprintf("  %s rows   %s",
			FormatNumber(rg.NumRows),
			FormatBytes(rg.TotalBytes),
		)
		if rg.AvgPagesCol > 0 {
			detail += fmt.Sprintf("   %d pages/col", rg.AvgPagesCol)
		}

		if i == selected {
			name = accentText.Render("▸ ") + selectedRow.Render(name)
		} else {
			name = "  " + normalText.Render(name)
		}

		entry := lipgloss.JoinVertical(lipgloss.Left,
			name,
			dimText.Render(detail),
		)
		items = append(items, entry)
	}

	return lipgloss.JoinVertical(lipgloss.Left, items...)
}

func renderFooterPanel(data ui.FooterData, width, height int) string {
	var sections []string

	sections = append(sections, headerText.Render("Footer"))

	// Schema info.
	sections = append(sections,
		normalText.Render(fmt.Sprintf("Schema: %d leaf columns", data.NumLeafColumns)),
	)
	if data.Format != "" {
		sections = append(sections,
			normalText.Render(fmt.Sprintf("Format: %s", data.Format)),
		)
	}

	// Top columns by size — use lipgloss table.
	if len(data.TopColumns) > 0 {
		sections = append(sections, "") // spacer
		sections = append(sections, headerText.Render("Top columns by size:"))

		rows := make([][]string, 0, len(data.TopColumns))
		for _, col := range data.TopColumns {
			rows = append(rows, []string{
				truncate(col.Path, width-12),
				FormatBytes(col.TotalBytes),
			})
		}

		t := table.New().
			Border(lipgloss.HiddenBorder()).
			Rows(rows...).
			StyleFunc(func(row, col int) lipgloss.Style {
				if col == 1 {
					return lipgloss.NewStyle().
						Foreground(colorAccent).
						Align(lipgloss.Right).
						PaddingLeft(2)
				}
				return lipgloss.NewStyle().
					Foreground(colorMuted).
					PaddingLeft(1)
			})

		sections = append(sections, t.Render())
	}

	// Key-value metadata.
	if len(data.KeyValues) > 0 {
		sections = append(sections, "") // spacer
		sections = append(sections, headerText.Render("Key-value metadata:"))

		for k, v := range data.KeyValues {
			entry := fmt.Sprintf("  %s = %s", k, truncate(v, width-len(k)-6))
			sections = append(sections, dimText.Render(entry))
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func truncate(s string, maxLen int) string {
	if maxLen < 4 {
		maxLen = 4
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-2] + ".."
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Indent adds prefix to the start of a multi-line string.
func indent(s string, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = prefix + l
	}
	return strings.Join(lines, "\n")
}
