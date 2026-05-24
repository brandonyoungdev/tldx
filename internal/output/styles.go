package output

import (
	"fmt"
	"os"

	"github.com/brandonyoungdev/tldx/internal/config"
	"github.com/brandonyoungdev/tldx/internal/resolver"
	"github.com/charmbracelet/lipgloss"
)

type StyleService struct {
	app     *config.TldxContext
	noColor bool
}

func NewStyleService(app *config.TldxContext) *StyleService {
	noColor := app.Config.NoColor
	if !noColor {
		_, noColor = os.LookupEnv("NO_COLOR")
	}
	if !noColor {
		if fi, err := os.Stdout.Stat(); err == nil {
			noColor = fi.Mode()&os.ModeCharDevice == 0
		}
	}
	return &StyleService{app: app, noColor: noColor}
}

// NewStyleServiceDirect creates a StyleService with an explicit noColor override.
// Useful for testing color rendering paths without TTY detection.
func NewStyleServiceDirect(app *config.TldxContext, noColor bool) *StyleService {
	return &StyleService{app: app, noColor: noColor}
}

func (s *StyleService) Available(domain resolver.DomainResult) string {
	text := fmt.Sprintf("✅ %s is available", domain.Domain)
	if s.app.Config.Verbose {
		text = fmt.Sprintf("%s - %v", text, domain.Details)
	}
	return s.Styled(text, "10") // green
}

func (s *StyleService) NotAvailable(domain resolver.DomainResult) string {
	text := fmt.Sprintf("❌ %s is not available", domain.Domain)
	if s.app.Config.Verbose {
		text = fmt.Sprintf("%s - %v", text, domain.Details)
	}
	return s.Styled(text, "9") // red
}

func (s *StyleService) Errored(domain string, err error) string {
	text := fmt.Sprintf("🟡 %s errored", domain)
	if s.app.Config.Verbose {
		text = fmt.Sprintf("%s - %s", text, err)
	}
	return s.Styled(text, "11") // Yellow
}

func (s *StyleService) Styled(text string, color string) string {
	if s.IsNoColor() {
		return text
	}

	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(color)).
		PaddingLeft(2)

	return style.Render(text)
}

func (s *StyleService) IsNoColor() bool {
	return s.noColor
}

func (s *StyleService) GroupHeader(text string) string {
	if s.IsNoColor() {
		return text
	}

	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("14")). // cyan
		PaddingLeft(2)

	return style.Render(text)
}
