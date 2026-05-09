package view

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/javiermolinar/parquetv/ui"
)

// RenderDictionary renders the dictionary view: table (left) + treemap (right).
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

	// Treemap on top, scrollable table below.
	mapHeight := contentHeight / 3
	if mapHeight < 4 {
		mapHeight = 4
	}
	tableHeight := contentHeight - mapHeight - 1 // 1 for separator

	mapContent := renderTreemap(vm.AllEntries, width, mapHeight)
	sep := dimText.Render(strings.Repeat("─", width))
	tableContent := renderDictTable(vm, width, tableHeight)

	content := lipgloss.JoinVertical(lipgloss.Left, mapContent, sep, tableContent)

	return lipgloss.JoinVertical(lipgloss.Left, topBar, content, bottomBar)
}

// renderDictTable renders the scrollable entry table.
func renderDictTable(vm ui.DictionaryVM, width, height int) string {
	numW := 6
	pctW := 7
	countW := 12
	valueW := width - numW - pctW - countW
	if valueW < 12 {
		valueW = 12
	}

	var lines []string

	header := lipgloss.JoinHorizontal(lipgloss.Top,
		dictNumStyle.Width(numW).Render("#"),
		dictHeaderStyle.Width(valueW).Render("value"),
		dictHeaderStyle.Width(countW).Align(lipgloss.Right).Render("count"),
		dictHeaderStyle.Width(pctW).Align(lipgloss.Right).Render("%"),
	)
	sep := dimText.Render(strings.Repeat("─", numW+valueW+countW+pctW))
	lines = append(lines, header, sep)

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

		lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Top,
			nStyle.Width(numW).Render(fmt.Sprintf("%d", e.Index)),
			vStyle.Width(valueW).Render(truncate(e.Value, valueW-2)),
			cStyle.Width(countW).Align(lipgloss.Right).Render(FormatNumber(e.Count)),
			pStyle.Width(pctW).Align(lipgloss.Right).Render(fmt.Sprintf("%.1f", e.Percent)),
		))
	}

	for len(lines) < height {
		lines = append(lines, "")
	}
	if len(lines) > height {
		lines = lines[:height]
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// --- Treemap rendering ---

// treemapRect represents a rectangle in the terminal grid.
type treemapRect struct {
	x, y, w, h int
	label       string
	pct         float64
	colorIdx    int
}

// renderTreemap renders a squarified treemap into a character grid.
func renderTreemap(entries []ui.DictEntryVM, width, height int) string {
	if len(entries) == 0 || width < 4 || height < 2 {
		return ""
	}

	// Build treemap rectangles using squarified layout.
	rects := squarify(entries, 0, 0, width, height)

	// Render into a 2D character grid.
	grid := make([][]rune, height)
	colors := make([][]int, height) // color index per cell (-1 = none)
	for y := 0; y < height; y++ {
		grid[y] = make([]rune, width)
		colors[y] = make([]int, width)
		for x := 0; x < width; x++ {
			grid[y][x] = ' '
			colors[y][x] = -1
		}
	}

	// Fill rectangles into grid.
	for _, r := range rects {
		fillRect(grid, colors, r)
	}

	// Render grid to styled string.
	palette := []lipgloss.Color{
		lipgloss.Color("#22C55E"), // green
		lipgloss.Color("#3B82F6"), // blue
		lipgloss.Color("#F59E0B"), // amber
		lipgloss.Color("#EF4444"), // red
		lipgloss.Color("#8B5CF6"), // purple
		lipgloss.Color("#06B6D4"), // cyan
		lipgloss.Color("#F97316"), // orange
		lipgloss.Color("#EC4899"), // pink
	}

	var rows []string
	for y := 0; y < height; y++ {
		var parts []string
		runStart := 0
		for runStart < width {
			// Find run of same color.
			ci := colors[y][runStart]
			runEnd := runStart + 1
			for runEnd < width && colors[y][runEnd] == ci {
				runEnd++
			}

			text := string(grid[y][runStart:runEnd])

			if ci >= 0 && ci < len(palette) {
				style := lipgloss.NewStyle().
					Foreground(lipgloss.Color("#000000")).
					Background(palette[ci])
				parts = append(parts, style.Render(text))
			} else {
				parts = append(parts, dimText.Render(text))
			}
			runStart = runEnd
		}
		rows = append(rows, strings.Join(parts, ""))
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// fillRect draws a rectangle with label into the character grid.
func fillRect(grid [][]rune, colors [][]int, r treemapRect) {
	if r.w < 1 || r.h < 1 {
		return
	}
	h := len(grid)
	w := 0
	if h > 0 {
		w = len(grid[0])
	}

	// Fill area.
	for y := r.y; y < r.y+r.h && y < h; y++ {
		for x := r.x; x < r.x+r.w && x < w; x++ {
			grid[y][x] = ' '
			colors[y][x] = r.colorIdx
		}
	}

	// Draw label centered in the rectangle.
	if r.w >= 3 && r.h >= 1 {
		label := r.label
		pctStr := fmt.Sprintf("%.0f%%", r.pct)

		// Try to fit label on one line, percentage on next.
		if r.h >= 2 {
			// Line 1: label.
			line1 := truncate(label, r.w-2)
			cy := r.y + (r.h-2)/2
			cx := r.x + (r.w-len(line1))/2
			writeText(grid, cx, cy, line1, w, h)

			// Line 2: percentage.
			cx2 := r.x + (r.w-len(pctStr))/2
			writeText(grid, cx2, cy+1, pctStr, w, h)
		} else {
			// Single row: try label+pct, or just pct.
			combined := truncate(label, r.w-len(pctStr)-2) + " " + pctStr
			if len(combined) > r.w-1 {
				combined = pctStr
			}
			if len(combined) > r.w-1 {
				combined = ""
			}
			cy := r.y
			cx := r.x + (r.w-len(combined))/2
			writeText(grid, cx, cy, combined, w, h)
		}
	}
}

func writeText(grid [][]rune, x, y int, text string, maxW, maxH int) {
	for i, ch := range text {
		px := x + i
		if px >= 0 && px < maxW && y >= 0 && y < maxH {
			grid[y][px] = ch
		}
	}
}

// Terminal chars are ~2x taller than wide. We scale the layout to
// compensate so "square" blocks look square on screen.
const charAspect = 2.0

// squarify implements the squarified treemap algorithm.
// Accounts for terminal character aspect ratio so blocks look square.
func squarify(entries []ui.DictEntryVM, x, y, w, h int) []treemapRect {
	if len(entries) == 0 || w <= 0 || h <= 0 {
		return nil
	}

	totalPct := 0.0
	for _, e := range entries {
		totalPct += e.Percent
	}
	if totalPct <= 0 {
		return nil
	}

	// Work in aspect-corrected space: scale height by charAspect.
	scaledH := float64(h) * charAspect
	totalArea := float64(w) * scaledH

	var result []treemapRect
	remaining := entries
	rx, ry := float64(x), float64(y)*charAspect
	rw, rh := float64(w), scaledH

	for len(remaining) > 0 && rw > 0.5 && rh > 0.5 {
		isWide := rw >= rh
		rowItems, rest := layoutRowF(remaining, totalPct, totalArea, rx, ry, rw, rh, isWide)

		// Convert from scaled space back to character space.
		for _, r := range rowItems {
			cr := treemapRect{
				x: int(math.Round(r.fx)), y: int(math.Round(r.fy / charAspect)),
				w: int(math.Round(r.fw)), h: int(math.Round(r.fh / charAspect)),
				label: r.label, pct: r.pct, colorIdx: r.colorIdx,
			}
			if cr.w < 1 {
				cr.w = 1
			}
			if cr.h < 1 {
				cr.h = 1
			}
			if cr.x+cr.w > x+w {
				cr.w = x + w - cr.x
			}
			if cr.y+cr.h > y+h {
				cr.h = y + h - cr.y
			}
			if cr.w > 0 && cr.h > 0 {
				result = append(result, cr)
			}
		}

		if len(rowItems) > 0 {
			if isWide {
				usedW := rowItems[0].fw
				rx += usedW
				rw -= usedW
			} else {
				usedH := rowItems[0].fh
				ry += usedH
				rh -= usedH
			}
		}
		remaining = rest
	}

	return result
}

// floatRect is a treemapRect in float (scaled) space.
type floatRect struct {
	fx, fy, fw, fh float64
	label          string
	pct            float64
	colorIdx       int
}

// layoutRowF works in float space for the squarified algorithm.
func layoutRowF(entries []ui.DictEntryVM, totalPct, totalArea, fx, fy, fw, fh float64, vertical bool) ([]floatRect, []ui.DictEntryVM) {
	if len(entries) == 0 {
		return nil, nil
	}

	best := math.MaxFloat64
	bestN := 1

	for n := 1; n <= len(entries); n++ {
		worst := worstAspectF(entries[:n], totalPct, totalArea, fw, fh, vertical)
		if worst <= best {
			best = worst
			bestN = n
		} else {
			break
		}
	}

	rowEntries := entries[:bestN]
	rowPct := 0.0
	for _, e := range rowEntries {
		rowPct += e.Percent
	}
	rowArea := rowPct / totalPct * totalArea

	var rects []floatRect

	if vertical {
		stripW := rowArea / fh
		if stripW < 1 {
			stripW = 1
		}
		cy := fy
		for i, e := range rowEntries {
			cellH := e.Percent / rowPct * fh
			if i == len(rowEntries)-1 {
				cellH = fh - (cy - fy)
			}
			rects = append(rects, floatRect{
				fx: fx, fy: cy, fw: stripW, fh: cellH,
				label: e.Value, pct: e.Percent, colorIdx: e.Index % 8,
			})
			cy += cellH
		}
	} else {
		stripH := rowArea / fw
		if stripH < 1 {
			stripH = 1
		}
		cx := fx
		for i, e := range rowEntries {
			cellW := e.Percent / rowPct * fw
			if i == len(rowEntries)-1 {
				cellW = fw - (cx - fx)
			}
			rects = append(rects, floatRect{
				fx: cx, fy: fy, fw: cellW, fh: stripH,
				label: e.Value, pct: e.Percent, colorIdx: e.Index % 8,
			})
			cx += cellW
		}
	}

	return rects, entries[bestN:]
}

func worstAspectF(entries []ui.DictEntryVM, totalPct, totalArea, fw, fh float64, vertical bool) float64 {
	rowPct := 0.0
	for _, e := range entries {
		rowPct += e.Percent
	}
	rowArea := rowPct / totalPct * totalArea

	var stripLen float64
	if vertical {
		stripLen = fh
	} else {
		stripLen = fw
	}

	stripThick := rowArea / stripLen
	if stripThick < 1 {
		stripThick = 1
	}

	worst := 0.0
	for _, e := range entries {
		cellArea := e.Percent / totalPct * totalArea
		cellLen := cellArea / stripThick
		if cellLen < 1 {
			cellLen = 1
		}
		aspect := math.Max(cellLen/stripThick, stripThick/cellLen)
		if aspect > worst {
			worst = aspect
		}
	}
	return worst
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
