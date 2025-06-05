package domain

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type ResultOutput interface {
	Write(result DomainResult)
	Flush()
}

func GetOutputWriter(format string) ResultOutput {
	switch format {
	case "json-stream":
		return &JSONStreamOutput{}
	case "json-array", "json":
		return NewJsonArrayOutput(os.Stdout)
	case "csv":
		return NewCSVOutput()
	case "text":
		return &TextOutput{}
	default:
		fmt.Println("Unknown output format. Defaulting to text.")
		return &TextOutput{}
	}
}

type TextOutput struct{}

func (o *TextOutput) Write(result DomainResult) {
	switch {
	case result.Error != nil:
		if !Config.OnlyAvailable || Config.Verbose {
			fmt.Println(Errored(result.Domain, result.Error))
		}
	case result.Available:
		fmt.Println(Available(result.Domain))
	default:
		if !Config.OnlyAvailable {
			fmt.Println(NotAvailable(result.Domain))
		}
	}
}

func (o *TextOutput) Flush() {}

type CSVOutput struct {
	writer *csv.Writer
}

func NewCSVOutput() *CSVOutput {
	w := csv.NewWriter(os.Stdout)
	w.Write([]string{"domain", "available", "error"})
	return &CSVOutput{writer: w}
}

func (o *CSVOutput) Write(result DomainResult) {
	errMsg := ""
	if result.Error != nil {
		errMsg = result.Error.Error()
	}
	o.writer.Write([]string{result.Domain, fmt.Sprintf("%v", result.Available), errMsg})
}

func (o *CSVOutput) Flush() {
	o.writer.Flush()
}

type JSONStreamOutput struct{}

func (o *JSONStreamOutput) Write(result DomainResult) {
	json.NewEncoder(os.Stdout).Encode(result)
}

func (o *JSONStreamOutput) Flush() {}

type JsonArrayOutput struct {
	results []DomainResult
	writer  io.Writer
}

func NewJsonArrayOutput(w io.Writer) *JsonArrayOutput {
	return &JsonArrayOutput{
		results: make([]DomainResult, 0, 100),
		writer:  w,
	}
}

func (o *JsonArrayOutput) Write(result DomainResult) {
	o.results = append(o.results, result)
}

func (o *JsonArrayOutput) Flush() {
	enc := json.NewEncoder(o.writer)
	enc.SetIndent("", "  ")
	if err := enc.Encode(o.results); err != nil {
		fmt.Fprintf(os.Stderr, "error encoding JSON array: %v\n", err)
	}
}
