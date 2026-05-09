package view

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/javiermolinar/parquetv/ui"
)

const dictBlockRows = 3 // rows used by the distribution blocks

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

	topBar := renderDictTopBar(vm, width)
	bottomBar := RenderBottomBar(ui.BottomBarData{
		Breadcrumb: fmt.Sprintf("Dictionary: %s", vm.ColumnPath),
		Shortcuts:  []string{"↑↓ scroll", "g/G jump", "esc back"},
	}, width)

	contentHeight := height - 3 // top(2) + bottom(1)

	var lines []string

	// Distribution blocks at the top.
	blockLines := renderDistributionBlocks(vm.AllEntries, width)
	lines = append(lines, blockLines...)
	lines = append(lines, "") // spacer

	// Table header.
	numW := 7
	valueW := width*2/5 - numW
	if valueW < 20 {
		valueW = 20
	}
	countW := 14
	pctW := 8

	header := lipgloss.JoinHorizontal(lipgloss.Top,
		dictNumStyle.Width(numW).Render("#"),
		dictHeaderStyle.Width(valueW).Render("value"),
		dictHeaderStyle.Width(countW).Align(lipgloss.Right).Render("count"),
		dictHeaderStyle.Width(pctW).Align(lipgloss.Right).Render("%"),
	)
	sep := dimText.Render(strings.Repeat("─", numW+valueW+countW+pctW))
	lines = append(lines, header, sep)

	// Table rows.
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

		lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Top, numCell, valCell, countCell, pctCell))
	}

	for len(lines) < contentHeight {
		lines = append(lines, "")
	}
	if len(lines) > contentHeight {
		lines = lines[:contentHeight]
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return lipgloss.JoinVertical(lipgloss.Left, topBar, content, bottomBar)
}

// renderDistributionBlocks renders proportional blocks showing the distribution.
// Each entry gets a block sized by its percentage, with the name inside.
func renderDistributionBlocks(entries []ui.DictEntryVM, width int) []string {
	if len(entries) == 0 {
		return nil
	}

	usable := width - 2 // leave margin

	// Assign character widths proportional to percentage.
	type block struct {
		label string
		pct   float64
		w     int
	}
	var blocks []block
	var othersCount int64
	var othersPct float64
	assigned := 0

	for _, e := range entries {
		charW := int(e.Percent / 100.0 * float64(usable))
		if charW < 1 && assigned < usable {
			charW = 1
		}
		if assigned+charW > usable {
			// Remaining entries go into "others".
			for _, r := range entries[len(blocks):] {
				othersCount += r.Count
				othersPct += r.Percent
			}
			break
		}
		blocks = append(blocks, block{
			label: e.Value,
			pct:   e.Percent,
			w:     charW,
		})
		assigned += charW
	}

	// Add "others" block if needed.
	if othersPct > 0 {
		remaining := usable - assigned
		if remaining < 1 {
			remaining = 1
		}
		blocks = append(blocks, block{
			label: fmt.Sprintf("+%d more", len(entries)-len(blocks)+1),
			pct:   othersPct,
			w:     remaining,
		})
	}

	// Pick colors for blocks (cycle through a palette).
	palette := []lipgloss.Color{
		lipgloss.Color("#22C55E"), // green (accent)
		lipgloss.Color("#3B82F6"), // blue
		lipgloss.Color("#F59E0B"), // amber
		lipgloss.Color("#EF4444"), // red
		lipgloss.Color("#8B5CF6"), // purple
		lipgloss.Color("#06B6D4"), // cyan
		lipgloss.Color("#F97316"), // orange
		lipgloss.Color("#EC4899"), // pink
	}
	othersColor := lipgloss.Color("#555555")

	// Render 3 rows: top border, label+pct, bottom border.
	var topParts, midParts, botParts []string

	for i, b := range blocks {
		if b.w < 1 {
			continue
		}
		color := palette[i%len(palette)]
		isOthers := i == len(blocks)-1 && othersPct > 0
		if isOthers {
			color = othersColor
		}

		blockStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(color).
			Bold(true)

		// Top: filled bar.
		topParts = append(topParts, blockStyle.Width(b.w).Render(strings.Repeat(" ", b.w)))

		// Middle: name + percentage (truncated to fit).
		label := b.label
		pctLabel := fmt.Sprintf(" %.0f%%", b.pct)
		maxLabel := b.w - len(pctLabel)
		if maxLabel < 0 {
			maxLabel = 0
		}
		if maxLabel < 3 {
			// Too narrow for name, just show percentage or nothing.
			if b.w >= 4 {
				midParts = append(midParts, blockStyle.Width(b.w).Render(fmt.Sprintf("%.0f%%", b.pct)))
			} else {
				midParts = append(midParts, blockStyle.Width(b.w).Render(strings.Repeat(" ", b.w)))
			}
		} else {
			text := truncate(label, maxLabel) + pctLabel
			midParts = append(midParts, blockStyle.Width(b.w).Render(text))
		}

		// Bottom: filled bar.
		botParts = append(botParts, blockStyle.Width(b.w).Render(strings.Repeat(" ", b.w)))
	}

	top := lipgloss.JoinHorizontal(lipgloss.Top, topParts...)
	mid := lipgloss.JoinHorizontal(lipgloss.Top, midParts...)
	bot := lipgloss.JoinHorizontal(lipgloss.Top, botParts...)

	return []string{top, mid, bot}
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
