package domain

import (
	"fmt"
	"slices"
	"strings"

	"github.com/brandonyoungdev/tldx/internal/composer"
	"github.com/brandonyoungdev/tldx/internal/config"
	"github.com/brandonyoungdev/tldx/internal/output"
	"github.com/brandonyoungdev/tldx/internal/resolver"
	"github.com/charmbracelet/huh/spinner"
	"github.com/charmbracelet/lipgloss"
)

func Exec(app *config.TldxContext, domainsOrKeywords []string) {

	composerService := composer.NewComposerService(app)
	domains, warnings := composerService.Compile(domainsOrKeywords)
	styleService := output.NewStyleService(app)
	if warnings != nil && len(warnings) > 0 {
		for _, warning := range warnings {
			if !app.Config.OnlyAvailable && app.Config.OutputFormat == "text" {
				fmt.Println(styleService.Styled(warning.Error(), "11")) // Yellow
			}
		}
	}

	resolverService := resolver.NewResolverService(app)
	resultChan := resolverService.CheckDomainsStreaming(domains)

	outputWriter := output.GetOutputWriter(app)

	output.Stat.Total = len(domains)

	groupedResults := map[string][]resolver.DomainResult{}

	spinner.New().
		Title(" Checking domains...").
		Action(func() {
			for result := range resultChan {
				if result.Error != nil {
					output.Stat.Errored++
				} else if result.Available {
					output.Stat.Available++
				} else {
					output.Stat.NotAvailable++
				}
				if app.Config.OnlyAvailable && !result.Available {
					continue
				}

				s := strings.Split(result.Domain, ".")
				groupedResults[s[0]] = append(groupedResults[s[0]], result)
			}
		}).
		Type(spinner.MiniDot).
		Style(lipgloss.NewStyle().Foreground(lipgloss.Color("10"))).
		Run()

	for baseDomain, domains := range groupedResults {

		if app.Config.OutputFormat == "text" {
			fmt.Println(styleService.GroupTitle(baseDomain))
		}

		slices.SortFunc(domains, func(a, b resolver.DomainResult) int {
			return strings.Compare(a.Domain, b.Domain)
		})

		for _, domain := range domains {
			outputWriter.Write(domain)
		}

		fmt.Print("\n")
	}

	outputWriter.Flush()

	if app.Config.ShowStats && app.Config.OutputFormat == "text" {
		// TODO: pipe this out for non-text formats
		fmt.Println(output.RenderStatsSummary())
	}
}
