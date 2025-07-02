package output

import (
	"fmt"
	"strings"

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

	numberWidth := 4

	header := baseStyle.
		Foreground(lipgloss.Color("14")). // Bright Blue
		Render(fmt.Sprintf("%*d searched", numberWidth, Stat.Total))

	available := baseStyle.
		Foreground(lipgloss.Color("10")). // Bright green
		Render(fmt.Sprintf("%*d available", numberWidth, Stat.Available))

	notAvailable := baseStyle.
		Foreground(lipgloss.Color("9")). // Red
		Render(fmt.Sprintf("%*d taken", numberWidth, Stat.NotAvailable))

	timedOut := baseStyle.
		Foreground(lipgloss.Color("12")). // Intense Yellow
		Render(fmt.Sprintf("%*d timed out", numberWidth, Stat.TimedOut))

	errored := baseStyle.
		Foreground(lipgloss.Color("3")). // Standard Yellow
		Render(fmt.Sprintf("%*d errored", numberWidth, Stat.Errored))

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
		BorderForeground(lipgloss.Color("14"))

	return border.Render(content)
}
