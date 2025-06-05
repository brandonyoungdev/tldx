package domain

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func Available(domain string) string {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00FF00")). // Light green color
		PaddingLeft(2).
		Render

	// Use the style to format the output
	return style(fmt.Sprintf("✅ %s is available", domain))
}

func NotAvailable(domain string) string {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF0000")). // Light red color
		PaddingLeft(2).
		Render

	// Use the style to format the output
	return style(fmt.Sprintf("❌ %s is not available", domain))
}

func Errored(domain string, err error) string {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFF00")). // Yellow color
		PaddingLeft(2).
		Render
	// Use the style to format the output
	emoji := "⚠️"

	if Config.Verbose {
		return style(fmt.Sprintf("%s  %s: %s", emoji, domain, err))
	}

	return style(fmt.Sprintf("%s  %s errored", emoji, domain))
}
