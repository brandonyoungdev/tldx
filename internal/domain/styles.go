package domain

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
)

func Available(domain DomainResult) string {
	text := fmt.Sprintf("✅ %s is available", domain.Domain)
	if Config.Verbose {
		text = fmt.Sprintf("✅ %s is available - %v", domain.Domain, domain.Details)
	}
	return Styled(text, "#00FF00")
}

func NotAvailable(domain DomainResult) string {
	text := fmt.Sprintf("❌ %s is not available", domain.Domain)
	if Config.Verbose {
		text = fmt.Sprintf("❌ %s is not available - %v", domain.Domain, domain.Details)
	}
	return Styled(text, "#FF0000")
}

func Errored(domain string, err error) string {
	text := fmt.Sprintf("⚠️  %s: %s", domain, err)
	return Styled(text, "#FFFF00")
}

func Styled(text string, color string) string {
	if IsNoColor() {
		return text
	}

	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(color)).
		PaddingLeft(2)

	return style.Render(text)
}

func IsNoColor() bool {
	if Config.NoColor {
		return true
	}
	_, exists := os.LookupEnv("NO_COLOR")
	return exists
}
