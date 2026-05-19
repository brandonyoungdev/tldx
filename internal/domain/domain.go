package domain

import (
	"context"
	"fmt"

	"github.com/brandonyoungdev/tldx/internal/composer"
	"github.com/brandonyoungdev/tldx/internal/config"
	"github.com/brandonyoungdev/tldx/internal/output"
	"github.com/brandonyoungdev/tldx/internal/resolver"
)

func Exec(ctx context.Context, app *config.TldxContext, domainsOrKeywords []string) bool {

	composerService := composer.NewComposerService(app)
	specs, warnings := composerService.Compile(domainsOrKeywords)
	styleService := output.NewStyleService(app)
	if warnings != nil && len(warnings) > 0 {
		for _, warning := range warnings {
			if !app.Config.OnlyAvailable && app.Config.OutputFormat == "text" {
				fmt.Println(styleService.Styled(warning.Error(), "11")) // Yellow
			}
		}
	}

	if app.Config.DryRun {
		fmt.Printf("Would check %d domain(s):\n", len(specs))
		for _, spec := range specs {
			fmt.Printf("  %s\n", spec.Domain)
		}
		return false
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	resolverService := resolver.NewResolverService(app)
	resultChan := resolverService.CheckDomainsStreaming(ctx, specs)

	outputWriter := output.GetOutputWriter(app)

	output.Stat.Total = len(specs)
	foundAvailable := false
	availableCount := 0

	for result := range resultChan {
		select {
		case <-ctx.Done():
			if app.Config.Verbose {
				fmt.Println(styleService.Styled("\\nOperation cancelled", "11"))
			}
			outputWriter.Flush()
			if app.Config.ShowStats && app.Config.OutputFormat == "text" {
				fmt.Println(output.RenderStatsSummary())
			}
			return foundAvailable
		default:
		}

		if result.Error != nil {
			output.Stat.Errored++
		} else if result.Available {
			output.Stat.Available++
			foundAvailable = true
			availableCount++
		} else {
			output.Stat.NotAvailable++
		}
		if app.Config.OnlyAvailable && !result.Available {
			continue
		}
		outputWriter.Write(result)

		if app.Config.Limit > 0 && availableCount >= app.Config.Limit {
			cancel()
			break
		}
	}

	outputWriter.Flush()

	if app.Config.ShowStats && app.Config.OutputFormat == "text" {
		fmt.Println(output.RenderStatsSummary())
	}

	return foundAvailable
}
