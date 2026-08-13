package output

import (
	"fmt"
	"os"
	"strings"

	"github.com/brandonyoungdev/tldx/internal/config"
	"github.com/brandonyoungdev/tldx/internal/forsale"
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

func (s *StyleService) ForSale(domain resolver.DomainResult) string {
	text := fmt.Sprintf("💰 %s is taken but for sale", domain.Domain)

	if details := s.forSaleDetails(domain.ForSale); details != "" {
		text = fmt.Sprintf("%s — %s", text, details)
	}

	return s.Styled(text, "13") // magenta
}

func (s *StyleService) forSaleDetails(info *forsale.Info) string {
	if info == nil {
		return ""
	}

	var parts []string
	for _, price := range info.Prices {
		parts = append(parts, price.String())
	}
	parts = append(parts, info.TrustedURIs()...)

	if s.app.Config.Verbose {
		parts = append(parts, info.Texts...)
		parts = append(parts, info.Codes...)
		for _, uri := range info.UntrustedURIs() {
			parts = append(parts, fmt.Sprintf("unverified scheme: %s", uri))
		}
	}

	return strings.Join(parts, " · ")
}

// Render formats a result for the text output modes. The second return value
// reports whether the line should be printed at all.
func (s *StyleService) Render(result resolver.DomainResult) (string, bool) {
	switch {
	case result.Error != nil:
		if s.app.Config.OnlyAvailable && !s.app.Config.Verbose {
			return "", false
		}
		return s.Errored(result.Domain, result.Error), true
	case result.Available:
		return s.Available(result), true
	case result.ForSale != nil:
		return s.ForSale(result), true
	default:
		if s.app.Config.OnlyAvailable || s.app.Config.OnlyForSale {
			return "", false
		}
		return s.NotAvailable(result), true
	}
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
