package output

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

type Stats struct {
	Total        int
	Available    int
	NotAvailable int
	TimedOut     int
	Errored      int
}

var Stat = Stats{}

func RenderStatsSummary() string {
	baseStyle := lipgloss.NewStyle().Bold(true)

	// Widths for number and label padding
	numberWidth := 2
	labelWidth := 14

	// Color helper
	color := func(c string) lipgloss.Style {
		return baseStyle.Foreground(lipgloss.Color(c))
	}

	type statRow struct {
		emoji string
		count int
		label string
		color string
	}

	stats := []statRow{
		{"🔍", Stat.Total, "searched", "14"},      // Bright Blue
		{"✅", Stat.Available, "available", "10"}, // Bright Green
		{"❌", Stat.NotAvailable, "taken", "9"},   // Red
		{"⏳", Stat.TimedOut, "timed out", "12"},  // Intense Yellow
		{"🟡", Stat.Errored, "errored", "3"},      // Yellow
	}

	var blocks []string
	for _, stat := range stats {
		// emoji + space + padded number + space + padded label
		formatted := fmt.Sprintf(
			"%s%*d %-*s",
			stat.emoji,
			numberWidth,
			stat.count,
			labelWidth,
			stat.label,
		)
		blocks = append(blocks, color(stat.color).Render(formatted))
	}

	content := lipgloss.JoinHorizontal(lipgloss.Top, blocks...)

	// Wrap in border
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		MarginTop(1).
		MarginBottom(1).
		Align(lipgloss.Left).
		BorderForeground(lipgloss.Color("14"))

	return border.Render(content)
}
