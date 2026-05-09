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
	right := renderFooterPanel(vm, rightWidth, contentHeight)

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

func renderFooterPanel(vm ui.FileOverviewVM, width, height int) string {
	data := vm.FooterPanel
	schema := vm.SchemaTree

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
	schemaContent := renderSchemaPanel(schema, vm, schemaWidth, height)
	schemaPanel := lipgloss.NewStyle().Width(schemaWidth).Height(height).PaddingLeft(1).Render(schemaContent)

	return lipgloss.JoinHorizontal(lipgloss.Top, metaPanel, schemaPanel)
}

// renderSchemaPanel renders the schema tree with optional focus/scroll/cursor.
func renderSchemaPanel(schema *ui.SchemaNodeVM, vm ui.FileOverviewVM, width, height int) string {
	if schema == nil || len(schema.Children) == 0 {
		return ""
	}

	var items []string
	items = append(items, headerText.Render("Schema"))

	// Flatten the tree into lines.
	var treeLines []string
	for _, child := range schema.Children {
		flattenSchemaNode(child, width, &treeLines)
	}

	visible := height - 1 // minus header
	if visible < 1 {
		visible = 1
	}

	if vm.SchemaFocused {
		// Windowed view with cursor.
		for i := 0; i < visible; i++ {
			idx := vm.SchemaOffset + i
			if idx >= len(treeLines) {
				items = append(items, "")
				continue
			}
			if idx == vm.SchemaCursor {
				items = append(items, schemaCursorStyle.Width(width).Render(stripAnsi(treeLines[idx])))
			} else {
				items = append(items, treeLines[idx])
			}
		}
	} else {
		// Static view, capped.
		if len(treeLines) > visible {
			treeLines = treeLines[:visible-1]
			treeLines = append(treeLines, dimText.Render("  ..."))
		}
		items = append(items, treeLines...)
	}

	return lipgloss.JoinVertical(lipgloss.Left, items...)
}

// flattenSchemaNode recursively flattens a schema tree into styled lines.
func flattenSchemaNode(node *ui.SchemaNodeVM, width int, lines *[]string) {
	indent := strings.Repeat("  ", node.Depth)

	if node.Leaf {
		typStr := dimText.Render(node.Type)
		nameStr := normalText.Render(truncate(node.Name, width-node.Depth*2-15))
		*lines = append(*lines, fmt.Sprintf("%s  %s  %s", indent, nameStr, typStr))
	} else {
		nameStr := accentText.Render("▼ " + node.Name)
		*lines = append(*lines, indent+nameStr)
		for _, child := range node.Children {
			flattenSchemaNode(child, width, lines)
		}
	}
}

// stripAnsi removes ANSI escape sequences for re-styling cursor lines.
func stripAnsi(s string) string {
	var out strings.Builder
	inEsc := false
	for _, r := range s {
		if r == '\033' {
			inEsc = true
			continue
		}
		if inEsc {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
			continue
		}
		out.WriteRune(r)
	}
	return out.String()
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
