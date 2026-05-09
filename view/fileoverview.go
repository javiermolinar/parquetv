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
	right := renderFooterPanel(vm.FooterPanel, vm.SchemaTree, rightWidth, contentHeight)

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

func renderFooterPanel(data ui.FooterData, schema *ui.SchemaNodeVM, width, height int) string {
	// Split the right panel into two sub-columns: metadata (left) and schema tree (right).
	metaWidth := width / 2
	schemaWidth := width - metaWidth - 1 // 1 for gap

	// --- Metadata sub-column ---
	var metaSections []string

	metaSections = append(metaSections, headerText.Render("Footer"))
	metaSections = append(metaSections,
		normalText.Render(fmt.Sprintf("Schema: %d leaf columns", data.NumLeafColumns)),
	)
	if data.Format != "" {
		metaSections = append(metaSections,
			normalText.Render(fmt.Sprintf("Format: %s", data.Format)),
		)
	}

	if len(data.TopColumns) > 0 {
		metaSections = append(metaSections, "") // spacer
		metaSections = append(metaSections, headerText.Render("Top columns by size:"))

		rows := make([][]string, 0, len(data.TopColumns))
		for _, col := range data.TopColumns {
			rows = append(rows, []string{
				truncate(col.Path, metaWidth-12),
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

		metaSections = append(metaSections, t.Render())
	}

	if len(data.KeyValues) > 0 {
		metaSections = append(metaSections, "") // spacer
		metaSections = append(metaSections, headerText.Render("Key-value metadata:"))

		for k, v := range data.KeyValues {
			entry := fmt.Sprintf("  %s = %s", k, truncate(v, metaWidth-len(k)-6))
			metaSections = append(metaSections, dimText.Render(entry))
		}
	}

	metaContent := lipgloss.JoinVertical(lipgloss.Left, metaSections...)
	metaPanel := lipgloss.NewStyle().Width(metaWidth).Height(height).Render(metaContent)

	// --- Schema tree sub-column ---
	var schemaSections []string

	if schema != nil && len(schema.Children) > 0 {
		schemaSections = append(schemaSections, headerText.Render("Schema"))

		var treeLines []string
		for _, child := range schema.Children {
			renderSchemaNode(child, schemaWidth, &treeLines)
		}

		// Cap to available height.
		max := height - 2
		if max < 0 {
			max = 0
		}
		if len(treeLines) > max {
			treeLines = treeLines[:max]
			treeLines = append(treeLines, dimText.Render("  ..."))
		}
		schemaSections = append(schemaSections, treeLines...)
	}

	schemaContent := lipgloss.JoinVertical(lipgloss.Left, schemaSections...)
	schemaPanel := lipgloss.NewStyle().Width(schemaWidth).Height(height).PaddingLeft(1).Render(schemaContent)

	return lipgloss.JoinHorizontal(lipgloss.Top, metaPanel, schemaPanel)
}

// renderSchemaNode recursively renders a schema tree node.
func renderSchemaNode(node *ui.SchemaNodeVM, width int, lines *[]string) {
	indent := strings.Repeat("  ", node.Depth)

	if node.Leaf {
		// Leaf: show name + type.
		typStr := dimText.Render(node.Type)
		nameStr := normalText.Render(truncate(node.Name, width-node.Depth*2-15))
		*lines = append(*lines, fmt.Sprintf("%s  %s  %s", indent, nameStr, typStr))
	} else {
		// Group: show name with marker.
		nameStr := accentText.Render("▼ " + node.Name)
		*lines = append(*lines, indent+nameStr)
		for _, child := range node.Children {
			renderSchemaNode(child, width, lines)
		}
	}
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
