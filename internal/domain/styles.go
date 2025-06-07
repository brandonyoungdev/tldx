package domain

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func Available(domain DomainResult) string {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00FF00")). // Light green color
		PaddingLeft(2).
		Render

	if Config.Verbose {
		return style(fmt.Sprintf("✅ %s is available - %v", domain.Domain, domain.Details))
	}

	// Use the style to format the output
	return style(fmt.Sprintf("✅ %s is available", domain.Domain))
}

func NotAvailable(domain DomainResult) string {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF0000")). // Light red color
		PaddingLeft(2).
		Render

	if Config.Verbose {
		return style(fmt.Sprintf("❌ %s is not available - %v", domain.Domain, domain.Details))
	}

	// Use the style to format the output
	return style(fmt.Sprintf("❌ %s is not available", domain.Domain))
}

func Errored(domain string, err error) string {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFF00")). // Yellow color
		PaddingLeft(2).
		Render
	// Use the style to format the output
	emoji := "⚠️"

	return style(fmt.Sprintf("%s  %s: %s", emoji, domain, err))

}
