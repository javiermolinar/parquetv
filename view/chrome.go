package view

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/javiermolinar/parquetv/ui"
)

// RenderTopBar renders the persistent top chrome.
func RenderTopBar(data ui.TopBarData, width int) string {
	// Inner styles inherit bar background so it fills the full width.
	title := topBarTitle.Inherit(topBarStyle).Render("parquetv")
	fileName := lipgloss.NewStyle().Inherit(topBarStyle).Foreground(colorPrimary).Render(data.FileName)
	sep := lipgloss.NewStyle().Inherit(topBarStyle).Foreground(colorMuted).Render(" ─── ")

	line1 := topBarStyle.Width(width).Render(" " + title + sep + fileName)

	context := data.Context
	if context == "" {
		context = fmt.Sprintf("%s  %s  %s rows  %d row groups  %d columns",
			FormatBytes(data.FileSize),
			formatOrDefault(data.Format, "parquet"),
			FormatNumber(data.TotalRows),
			data.NumRowGroups,
			data.NumColumns,
		)
	}
	line2 := topBarStyle.Width(width).Render(" " + topBarDim.Inherit(topBarStyle).Render(context))

	return lipgloss.JoinVertical(lipgloss.Left, line1, line2)
}

// RenderBottomBar renders the persistent bottom chrome.
// Simple approach: build plain text for measurement, then colorize.
func RenderBottomBar(data ui.BottomBarData, width int) string {
	// Build the shortcuts as plain text first for accurate width measurement.
	var shortcutPlain []string
	for _, s := range data.Shortcuts {
		shortcutPlain = append(shortcutPlain, s)
	}
	rightText := strings.Join(shortcutPlain, "  ")
	leftText := data.Breadcrumb

	gap := width - len(leftText) - len(rightText) - 4 // 4 for padding
	if gap < 1 {
		gap = 1
	}

	// Now build the colored version with exact same widths.
	var colored strings.Builder
	colored.WriteString(" ")
	colored.WriteString(breadcrumbText.Render(leftText))
	colored.WriteString(strings.Repeat(" ", gap))
	for i, s := range data.Shortcuts {
		if i > 0 {
			colored.WriteString("  ")
		}
		parts := strings.SplitN(s, " ", 2)
		if len(parts) == 2 {
			colored.WriteString(shortcutKey.Render(parts[0]))
			colored.WriteString(" ")
			colored.WriteString(shortcutDesc.Render(parts[1]))
		}
	}
	colored.WriteString(" ")

	return bottomBarStyle.Width(width).Render(colored.String())
}

// FormatBytes formats bytes as human-readable.
func FormatBytes(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

// FormatNumber formats an int64 with comma separators.
func FormatNumber(n int64) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var parts []string
	for i := len(s); i > 0; i -= 3 {
		start := i - 3
		if start < 0 {
			start = 0
		}
		parts = append([]string{s[start:i]}, parts...)
	}
	return strings.Join(parts, ",")
}

func formatOrDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
