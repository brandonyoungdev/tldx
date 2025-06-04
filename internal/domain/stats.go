package domain

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type Stats struct {
	total        int
	available    int
	notAvailable int
	timedOut     int
	errored      int
}

var stats = Stats{}

func RenderStatsSummary() string {
	baseStyle := lipgloss.NewStyle().Bold(true)

	numberWidth := 4

	header := baseStyle.
		Foreground(lipgloss.Color("#00BFFF")). // DeepSkyBlue
		Render(fmt.Sprintf("%*d searched", numberWidth, stats.total))

	available := baseStyle.
		Foreground(lipgloss.Color("#00FF00")). // Bright green
		Render(fmt.Sprintf("%*d available", numberWidth, stats.available))

	notAvailable := baseStyle.
		Foreground(lipgloss.Color("#FF0000")). // Red
		Render(fmt.Sprintf("%*d taken", numberWidth, stats.notAvailable))

	timedOut := baseStyle.
		Foreground(lipgloss.Color("#F1C21B")). // Yellow
		Render(fmt.Sprintf("%*d timed out", numberWidth, stats.timedOut))

	errored := baseStyle.
		Foreground(lipgloss.Color("#FF832B")). // Orange
		Render(fmt.Sprintf("%*d errored", numberWidth, stats.errored))

	// Compose a single line
	content := strings.Join([]string{
		header,
		available,
		notAvailable,
		timedOut,
		errored,
	}, "  ")

	// Wrap in a border
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		MarginTop(1).
		MarginBottom(1).
		Align(lipgloss.Left).
		BorderForeground(lipgloss.Color("#5DADE2")) // Light blue

	return border.Render(content)
}
