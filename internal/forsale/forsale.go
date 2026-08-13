// Package forsale reads the RFC 10023 "_for-sale" TXT records that let a
// domain holder advertise a registered domain for sale.
package forsale

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

const (
	NodeName     = "_for-sale"
	versionTag   = "v=FORSALE1"
	maxValueLen  = 239
	maxRecordLen = 255
)

type Info struct {
	URIs   []URI    `json:"uris,omitempty"`
	Texts  []string `json:"texts,omitempty"`
	Codes  []string `json:"codes,omitempty"`
	Prices []Price  `json:"prices,omitempty"`
}

// URI is a furi= value. An untrusted scheme is kept, but must not be shown as
// a followable link.
type URI struct {
	Value   string `json:"value"`
	Scheme  string `json:"scheme"`
	Trusted bool   `json:"trusted"`
}

// Price is an fval= asking price. Amount stays a string to avoid rounding.
type Price struct {
	Currency string `json:"currency"`
	Amount   string `json:"amount"`
}

func (p Price) String() string {
	return p.Currency + " " + p.Amount
}

// IsEmpty reports a version tag with no usable content. The domain should
// still be assumed for sale.
func (i Info) IsEmpty() bool {
	return len(i.URIs) == 0 && len(i.Texts) == 0 && len(i.Codes) == 0 && len(i.Prices) == 0
}

func (i Info) TrustedURIs() []string {
	var out []string
	for _, u := range i.URIs {
		if u.Trusted {
			out = append(out, u.Value)
		}
	}
	return out
}

func (i Info) UntrustedURIs() []string {
	var out []string
	for _, u := range i.URIs {
		if !u.Trusted {
			out = append(out, u.Value)
		}
	}
	return out
}

func Name(domain string) string {
	return NodeName + "." + strings.TrimSuffix(domain, ".")
}

// Lookup reports ok when a valid record was found.
func Lookup(ctx context.Context, domain string, lookupTXT func(context.Context, string) ([]string, error)) (Info, bool, error) {
	if lookupTXT == nil {
		return Info{}, false, errors.New("forsale: no TXT resolver provided")
	}

	txts, err := lookupTXT(ctx, Name(domain))
	if err != nil {
		return Info{}, false, err
	}

	info, ok := Parse(txts)
	return info, ok, nil
}

// Parse reports ok when at least one record carried a valid version tag. The
// returned Info may still be empty.
func Parse(txts []string) (Info, bool) {
	var (
		info Info
		ok   bool
		seen = make(map[string]bool, len(txts))
	)

	for _, txt := range txts {
		if len(txt) > maxRecordLen {
			continue
		}

		content, valid := trimVersion(txt)
		if !valid {
			continue
		}
		ok = true

		if seen[content] {
			continue
		}
		seen[content] = true

		// One pair per record, value running to the end: splitting on ";"
		// would corrupt URIs containing one.
		tag, value, found := strings.Cut(content, "=")
		if !found || len(value) < 1 || len(value) > maxValueLen {
			continue
		}

		switch tag {
		case "fcod":
			info.Codes = append(info.Codes, clean(value))
		case "ftxt":
			info.Texts = append(info.Texts, clean(value))
		case "furi":
			if u, valid := parseURI(value); valid {
				info.URIs = append(info.URIs, u)
			}
		case "fval":
			if p, valid := parsePrice(value); valid {
				info.Prices = append(info.Prices, p)
			}
		}
	}

	// An RRset has no inherent order, so sort for stable output.
	sort.Strings(info.Codes)
	sort.Strings(info.Texts)
	sort.Slice(info.URIs, func(a, b int) bool { return info.URIs[a].Value < info.URIs[b].Value })
	sort.Slice(info.Prices, func(a, b int) bool {
		if info.Prices[a].Currency != info.Prices[b].Currency {
			return info.Prices[a].Currency < info.Prices[b].Currency
		}
		return info.Prices[a].Amount < info.Prices[b].Amount
	})

	return info, ok
}

// trimVersion tolerates a missing trailing semicolon and spaces after it.
func trimVersion(txt string) (string, bool) {
	if !strings.HasPrefix(txt, versionTag) {
		return "", false
	}

	rest := txt[len(versionTag):]

	// Don't read a future "v=FORSALE10" as version 1.
	if rest != "" && rest[0] != ';' && rest[0] != ' ' {
		return "", false
	}

	rest = strings.TrimPrefix(rest, ";")
	return strings.TrimLeft(rest, " "), true
}

var trustedSchemes = map[string]bool{
	"http":   true,
	"https":  true,
	"mailto": true,
	"tel":    true,
}

func parseURI(value string) (URI, bool) {
	value = clean(value)

	u, err := url.Parse(value)
	if err != nil || u.Scheme == "" {
		return URI{}, false
	}

	scheme := strings.ToLower(u.Scheme)
	return URI{Value: value, Scheme: scheme, Trusted: trustedSchemes[scheme]}, true
}

// fval-value: an uppercase currency code followed by a decimal amount.
var fvalPattern = regexp.MustCompile(`^([A-Z]+)([0-9]+(?:\.[0-9]+)?)$`)

func parsePrice(value string) (Price, bool) {
	value = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(value), ";"))

	match := fvalPattern.FindStringSubmatch(value)
	if match == nil {
		return Price{}, false
	}

	return Price{Currency: match[1], Amount: match[2]}, true
}

// clean drops control characters. Values are authored by the domain holder, so
// they must not reach a terminal with ANSI escapes intact.
func clean(value string) string {
	value = strings.ToValidUTF8(value, "")
	return strings.Map(func(r rune) rune {
		switch {
		case r == '\t':
			return ' '
		case unicode.IsControl(r):
			return -1
		default:
			return r
		}
	}, value)
}
