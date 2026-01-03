package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/brandonyoungdev/tldx/internal/config"
	"github.com/brandonyoungdev/tldx/internal/resolver"
)

type ResultOutput interface {
	Write(result resolver.DomainResult)
	Flush()
}

func GetOutputWriter(app *config.TldxContext) ResultOutput {
	switch app.Config.OutputFormat {
	case "json-stream":
		return &JSONStreamOutput{}
	case "json-array", "json":
		return NewJsonArrayOutput(os.Stdout)
	case "csv":
		return NewCSVOutput()
	case "text":
		return NewTextOutput(app)
	case "grouped":
		return NewGroupedOutput(app)
	case "grouped-tld":
		return NewGroupedByTLDOutput(app)
	default:
		// This is okay, since it'll output text by default.
		fmt.Println("Unknown output format. Defaulting to text.")
		return NewTextOutput(app)
	}
}

type TextOutput struct {
	app          *config.TldxContext
	styleService *StyleService
}

func NewTextOutput(app *config.TldxContext) *TextOutput {
	return &TextOutput{
		app:          app,
		styleService: NewStyleService(app),
	}
}

func (o *TextOutput) Write(result resolver.DomainResult) {
	switch {
	case result.Error != nil:
		if !o.app.Config.OnlyAvailable || o.app.Config.Verbose {
			fmt.Println(o.styleService.Errored(result.Domain, result.Error))
		}
	case result.Available:
		fmt.Println(o.styleService.Available(result))
	default:
		if !o.app.Config.OnlyAvailable {
			fmt.Println(o.styleService.NotAvailable(result))
		}
	}
}

func (o *TextOutput) Flush() {}

type CSVOutput struct {
	writer *csv.Writer
}

func NewCSVOutput() *CSVOutput {
	w := csv.NewWriter(os.Stdout)
	w.Write([]string{"domain", "available", "details", "error"})
	return &CSVOutput{writer: w}
}

func (o *CSVOutput) Write(result resolver.DomainResult) {
	errMsg := ""
	if result.Error != nil {
		errMsg = result.Error.Error()
	}

	record := []string{
		result.Domain,
		fmt.Sprintf("%v", result.Available),
		result.Details,
		errMsg,
	}

	if err := o.writer.Write(record); err != nil {
		fmt.Fprintf(os.Stderr, "error writing CSV record: %v\n", err)
	}
}

func (o *CSVOutput) Flush() {
	o.writer.Flush()
	if err := o.writer.Error(); err != nil {
		fmt.Fprintf(os.Stderr, "error flushing CSV writer: %v\n", err)
	}
}

type JSONStreamOutput struct{}

func (o *JSONStreamOutput) Write(result resolver.DomainResult) {
	json.NewEncoder(os.Stdout).Encode(result.AsEncodable())
}

func (o *JSONStreamOutput) Flush() {}

type JsonArrayOutput struct {
	results []resolver.EncodableDomainResult
	writer  io.Writer
}

func NewJsonArrayOutput(w io.Writer) *JsonArrayOutput {
	return &JsonArrayOutput{
		results: make([]resolver.EncodableDomainResult, 0, 100),
		writer:  w,
	}
}

func (o *JsonArrayOutput) Write(result resolver.DomainResult) {
	o.results = append(o.results, result.AsEncodable())
}

func (o *JsonArrayOutput) Flush() {
	enc := json.NewEncoder(o.writer)
	enc.SetIndent("", "  ")
	if err := enc.Encode(o.results); err != nil {
		fmt.Fprintf(os.Stderr, "error encoding JSON array: %v\n", err)
	}
}

type GroupedOutput struct {
	app          *config.TldxContext
	styleService *StyleService
	results      []resolver.DomainResult
}

func NewGroupedOutput(app *config.TldxContext) *GroupedOutput {
	return &GroupedOutput{
		app:          app,
		styleService: NewStyleService(app),
		results:      make([]resolver.DomainResult, 0, 100),
	}
}

func (o *GroupedOutput) Write(result resolver.DomainResult) {
	o.results = append(o.results, result)
}

// extractKeyword extracts the keyword from a domain by removing prefixes, suffixes, and TLD
func (o *GroupedOutput) extractKeyword(domain string) string {
	// Remove TLD (everything after last dot)
	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return domain
	}
	baseName := strings.Join(parts[:len(parts)-1], ".")

	// Try to remove known prefixes
	for _, prefix := range o.app.Config.Prefixes {
		if strings.HasPrefix(baseName, prefix) {
			baseName = strings.TrimPrefix(baseName, prefix)
			break
		}
	}

	// Try to remove known suffixes
	for _, suffix := range o.app.Config.Suffixes {
		if strings.HasSuffix(baseName, suffix) {
			baseName = strings.TrimSuffix(baseName, suffix)
			break
		}
	}

	return baseName
}

func (o *GroupedOutput) Flush() {
	// Group domains by keyword
	grouped := make(map[string][]resolver.DomainResult)
	for _, result := range o.results {
		keyword := o.extractKeyword(result.Domain)
		grouped[keyword] = append(grouped[keyword], result)
	}

	// Sort keywords
	sortedKeywords := make([]string, 0, len(grouped))
	for keyword := range grouped {
		sortedKeywords = append(sortedKeywords, keyword)
	}
	sort.Strings(sortedKeywords)

	// Output grouped and sorted domains
	for _, keyword := range sortedKeywords {
		domains := grouped[keyword]
		// Sort domains within each keyword
		sort.Slice(domains, func(i, j int) bool {
			return domains[i].Domain < domains[j].Domain
		})

		fmt.Printf("\n%s\n", o.styleService.GroupHeader(strings.ToLower(keyword)))
		for _, result := range domains {
			switch {
			case result.Error != nil:
				if !o.app.Config.OnlyAvailable || o.app.Config.Verbose {
					fmt.Println(o.styleService.Errored(result.Domain, result.Error))
				}
			case result.Available:
				fmt.Println(o.styleService.Available(result))
			default:
				if !o.app.Config.OnlyAvailable {
					fmt.Println(o.styleService.NotAvailable(result))
				}
			}
		}
	}
}

type GroupedByTLDOutput struct {
	app          *config.TldxContext
	styleService *StyleService
	results      []resolver.DomainResult
}

func NewGroupedByTLDOutput(app *config.TldxContext) *GroupedByTLDOutput {
	return &GroupedByTLDOutput{
		app:          app,
		styleService: NewStyleService(app),
		results:      make([]resolver.DomainResult, 0, 100),
	}
}

func (o *GroupedByTLDOutput) Write(result resolver.DomainResult) {
	o.results = append(o.results, result)
}

func (o *GroupedByTLDOutput) Flush() {
	// Group domains by TLD
	grouped := make(map[string][]resolver.DomainResult)
	for _, result := range o.results {
		// Extract TLD from domain (everything after the last dot)
		parts := strings.Split(result.Domain, ".")
		if len(parts) > 0 {
			tld := parts[len(parts)-1]
			grouped[tld] = append(grouped[tld], result)
		}
	}

	// Sort TLDs
	tlds := make([]string, 0, len(grouped))
	for tld := range grouped {
		tlds = append(tlds, tld)
	}
	sort.Strings(tlds)

	// Output grouped and sorted domains
	for _, tld := range tlds {
		domains := grouped[tld]
		// Sort domains within each TLD
		sort.Slice(domains, func(i, j int) bool {
			return domains[i].Domain < domains[j].Domain
		})

		fmt.Printf("\n%s\n", o.styleService.GroupHeader(fmt.Sprintf(".%s", tld)))
		for _, result := range domains {
			switch {
			case result.Error != nil:
				if !o.app.Config.OnlyAvailable || o.app.Config.Verbose {
					fmt.Println(o.styleService.Errored(result.Domain, result.Error))
				}
			case result.Available:
				fmt.Println(o.styleService.Available(result))
			default:
				if !o.app.Config.OnlyAvailable {
					fmt.Println(o.styleService.NotAvailable(result))
				}
			}
		}
	}
}
