package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"

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
