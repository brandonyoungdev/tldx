package mcpserver

import (
	"github.com/brandonyoungdev/tldx/internal/forsale"
	"github.com/brandonyoungdev/tldx/internal/resolver"
)

// StatusUnknown means the lookup failed; never report it as available.
const (
	StatusAvailable = "available"
	StatusTaken     = "taken"
	StatusUnknown   = "unknown"
)

// DomainCheck is separate from resolver.EncodableDomainResult, which backs the
// CLI's output formats.
type DomainCheck struct {
	Domain string `json:"domain" jsonschema_description:"The domain name this verdict is for."`
	Status string `json:"status" jsonschema_description:"One of \"available\", \"taken\", or \"unknown\". \"unknown\" means the lookup failed and says nothing about availability."`
	// Omitted when Status is "unknown", so a failed lookup is not read as free.
	Available *bool         `json:"available,omitempty" jsonschema_description:"Present only when status is \"available\" or \"taken\"."`
	Details   string        `json:"details,omitempty"`
	Error     string        `json:"error,omitempty"`
	Keyword   string        `json:"keyword,omitempty"`
	Prefix    string        `json:"prefix,omitempty"`
	Suffix    string        `json:"suffix,omitempty"`
	TLD       string        `json:"tld,omitempty"`
	ForSale   *forsale.Info `json:"for_sale,omitempty"`
}

type CheckResponse struct {
	Results []DomainCheck `json:"results"`
	// Every domain resolved, including ones filtered out of Results.
	Checked   int  `json:"checked"`
	Available int  `json:"available_count"`
	Taken     int  `json:"taken_count"`
	Errored   int  `json:"errored_count,omitempty"`
	ForSale   int  `json:"for_sale_count,omitempty"`
	Truncated bool `json:"truncated,omitempty"`

	DryRun  bool     `json:"dry_run,omitempty"`
	Planned int      `json:"planned,omitempty"`
	Domains []string `json:"domains,omitempty"`

	// Why results were cut short, and what to change on the next call.
	Note string `json:"note,omitempty"`
}

func fromResult(r resolver.DomainResult) DomainCheck {
	out := DomainCheck{
		Domain:  r.Domain,
		Details: r.Details,
		Keyword: r.Keyword,
		Prefix:  r.Prefix,
		Suffix:  r.Suffix,
		TLD:     r.TLD,
		ForSale: r.ForSale,
	}

	if r.Error != nil {
		out.Status = StatusUnknown
		out.Error = r.Error.Error()
		return out
	}

	available := r.Available
	out.Available = &available
	if available {
		out.Status = StatusAvailable
	} else {
		out.Status = StatusTaken
	}
	return out
}
