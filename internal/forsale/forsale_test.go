package forsale_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/brandonyoungdev/tldx/internal/forsale"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		txts     []string
		wantOK   bool
		expected forsale.Info
	}{
		{
			name:   "no records at all",
			txts:   nil,
			wantOK: false,
		},
		{
			name:   "unrelated txt record",
			txts:   []string{"v=spf1 include:_spf.example.com ~all"},
			wantOK: false,
		},
		{
			// RFC 10023 §3.6: a bare version tag still means "for sale".
			name:     "version tag only",
			txts:     []string{"v=FORSALE1;"},
			wantOK:   true,
			expected: forsale.Info{},
		},
		{
			name:     "version tag without trailing semicolon",
			txts:     []string{"v=FORSALE1"},
			wantOK:   true,
			expected: forsale.Info{},
		},
		{
			name:     "spaces after the version tag",
			txts:     []string{"v=FORSALE1;   fval=USD750"},
			wantOK:   true,
			expected: forsale.Info{Prices: []forsale.Price{{Currency: "USD", Amount: "750"}}},
		},
		{
			name:     "fcod from the RFC",
			txts:     []string{"v=FORSALE1;fcod=XX-aHR0cHM...wbGUuY29t"},
			wantOK:   true,
			expected: forsale.Info{Codes: []string{"XX-aHR0cHM...wbGUuY29t"}},
		},
		{
			name:     "ftxt from the RFC",
			txts:     []string{"v=FORSALE1;ftxt=Eligibility criteria apply."},
			wantOK:   true,
			expected: forsale.Info{Texts: []string{"Eligibility criteria apply."}},
		},
		{
			name:   "furi from the RFC",
			txts:   []string{"v=FORSALE1;furi=https://example.com/fs?d=eHl6"},
			wantOK: true,
			expected: forsale.Info{URIs: []forsale.URI{
				{Value: "https://example.com/fs?d=eHl6", Scheme: "https", Trusted: true},
			}},
		},
		{
			name:     "fval from the RFC",
			txts:     []string{"v=FORSALE1;fval=USD750"},
			wantOK:   true,
			expected: forsale.Info{Prices: []forsale.Price{{Currency: "USD", Amount: "750"}}},
		},
		{
			name: "combined rrset from the RFC",
			txts: []string{
				"v=FORSALE1;furi=https://fs.example.com/",
				"v=FORSALE1;ftxt=This domain name is for sale",
				"v=FORSALE1;fval=EUR500",
				"v=FORSALE1;fcod=EXCO-ZGVhZGJlZWYx",
			},
			wantOK: true,
			expected: forsale.Info{
				URIs:   []forsale.URI{{Value: "https://fs.example.com/", Scheme: "https", Trusted: true}},
				Texts:  []string{"This domain name is for sale"},
				Codes:  []string{"EXCO-ZGVhZGJlZWYx"},
				Prices: []forsale.Price{{Currency: "EUR", Amount: "500"}},
			},
		},
		{
			name:     "fractional crypto amount keeps full precision",
			txts:     []string{"v=FORSALE1;fval=BTC0.000010"},
			wantOK:   true,
			expected: forsale.Info{Prices: []forsale.Price{{Currency: "BTC", Amount: "0.000010"}}},
		},
		{
			name:   "mailto and tel are trusted schemes",
			txts:   []string{"v=FORSALE1;furi=mailto:sales@example.com", "v=FORSALE1;furi=tel:+15551234567"},
			wantOK: true,
			expected: forsale.Info{URIs: []forsale.URI{
				{Value: "mailto:sales@example.com", Scheme: "mailto", Trusted: true},
				{Value: "tel:+15551234567", Scheme: "tel", Trusted: true},
			}},
		},
		{
			name:   "javascript uri is parsed but not trusted",
			txts:   []string{"v=FORSALE1;furi=javascript:alert(1)"},
			wantOK: true,
			expected: forsale.Info{URIs: []forsale.URI{
				{Value: "javascript:alert(1)", Scheme: "javascript", Trusted: false},
			}},
		},
		{
			name:   "relative uri is dropped",
			txts:   []string{"v=FORSALE1;furi=/buy-this-domain"},
			wantOK: true,
		},
		{
			name:   "uri containing a semicolon is not split",
			txts:   []string{"v=FORSALE1;furi=https://example.com/fs?a=1;b=2"},
			wantOK: true,
			expected: forsale.Info{URIs: []forsale.URI{
				{Value: "https://example.com/fs?a=1;b=2", Scheme: "https", Trusted: true},
			}},
		},
		{
			name:   "duplicate pairs across the rrset are collapsed",
			txts:   []string{"v=FORSALE1;fval=USD750", "v=FORSALE1;fval=USD750"},
			wantOK: true,
			expected: forsale.Info{
				Prices: []forsale.Price{{Currency: "USD", Amount: "750"}},
			},
		},
		{
			name:   "output order is stable regardless of rrset order",
			txts:   []string{"v=FORSALE1;fval=USD750", "v=FORSALE1;fval=EUR500"},
			wantOK: true,
			expected: forsale.Info{Prices: []forsale.Price{
				{Currency: "EUR", Amount: "500"},
				{Currency: "USD", Amount: "750"},
			}},
		},
		{
			name:     "control characters are stripped from free text",
			txts:     []string{"v=FORSALE1;ftxt=Buy \x1b[31mnow\x1b[0m\x07 please"},
			wantOK:   true,
			expected: forsale.Info{Texts: []string{"Buy [31mnow[0m please"}},
		},
		{
			name:   "unknown content tag is ignored",
			txts:   []string{"v=FORSALE1;fzzz=something"},
			wantOK: true,
		},
		{
			name:   "version tag is case sensitive",
			txts:   []string{"V=FORSALE1;fval=USD750", "v=forsale1;fval=USD750"},
			wantOK: false,
		},
		{
			name:   "content tags are case sensitive",
			txts:   []string{"v=FORSALE1;FVAL=USD750"},
			wantOK: true,
		},
		{
			name:   "a future version is not read as version 1",
			txts:   []string{"v=FORSALE10;fval=USD750"},
			wantOK: false,
		},
		{
			name:   "version tag must lead the record",
			txts:   []string{"fval=USD750;v=FORSALE1;"},
			wantOK: false,
		},
		{
			name:   "lowercase currency is rejected",
			txts:   []string{"v=FORSALE1;fval=usd750"},
			wantOK: true,
		},
		{
			name:   "price without an amount is rejected",
			txts:   []string{"v=FORSALE1;fval=USD"},
			wantOK: true,
		},
		{
			name:   "price with a trailing dot is rejected",
			txts:   []string{"v=FORSALE1;fval=USD750."},
			wantOK: true,
		},
		{
			name:   "empty value is rejected",
			txts:   []string{"v=FORSALE1;ftxt="},
			wantOK: true,
		},
		{
			// The 16-octet prefix puts the 239-octet value bound and the
			// 255-octet record bound on the same byte.
			name:     "value of exactly 239 octets is kept",
			txts:     []string{"v=FORSALE1;ftxt=" + strings.Repeat("a", 239)},
			wantOK:   true,
			expected: forsale.Info{Texts: []string{strings.Repeat("a", 239)}},
		},
		{
			name:   "record longer than 255 octets is ignored entirely",
			txts:   []string{"v=FORSALE1;ftxt=" + strings.Repeat("a", 240)},
			wantOK: false,
		},
		{
			name:   "invalid records do not suppress valid ones",
			txts:   []string{"garbage", "v=FORSALE1;fval=USD750", "v=spf1 ~all"},
			wantOK: true,
			expected: forsale.Info{
				Prices: []forsale.Price{{Currency: "USD", Amount: "750"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, ok := forsale.Parse(tt.txts)
			assert.Equal(t, tt.wantOK, ok, "for-sale signal")
			assert.Equal(t, tt.expected, info)
		})
	}
}

func TestInfoHelpers(t *testing.T) {
	empty, ok := forsale.Parse([]string{"v=FORSALE1;"})
	require.True(t, ok)
	assert.True(t, empty.IsEmpty())

	info, ok := forsale.Parse([]string{
		"v=FORSALE1;furi=https://fs.example.com/",
		"v=FORSALE1;furi=gopher://example.com/",
		"v=FORSALE1;fval=EUR500",
	})
	require.True(t, ok)

	assert.False(t, info.IsEmpty())
	assert.Equal(t, []string{"https://fs.example.com/"}, info.TrustedURIs())
	assert.Equal(t, []string{"gopher://example.com/"}, info.UntrustedURIs())
	assert.Equal(t, "EUR 500", info.Prices[0].String())
}

func TestName(t *testing.T) {
	assert.Equal(t, "_for-sale.example.com", forsale.Name("example.com"))
	assert.Equal(t, "_for-sale.example.com", forsale.Name("example.com."))
}

func TestLookup(t *testing.T) {
	t.Run("queries the for-sale node and parses the result", func(t *testing.T) {
		var queried string
		lookup := func(_ context.Context, name string) ([]string, error) {
			queried = name
			return []string{"v=FORSALE1;fval=USD750"}, nil
		}

		info, ok, err := forsale.Lookup(t.Context(), "example.com", lookup)

		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "_for-sale.example.com", queried)
		assert.Equal(t, []forsale.Price{{Currency: "USD", Amount: "750"}}, info.Prices)
	})

	t.Run("propagates resolver errors", func(t *testing.T) {
		lookup := func(_ context.Context, _ string) ([]string, error) {
			return nil, errors.New("no such host")
		}

		_, ok, err := forsale.Lookup(t.Context(), "example.com", lookup)

		require.Error(t, err)
		assert.False(t, ok)
	})

	t.Run("requires a resolver", func(t *testing.T) {
		_, ok, err := forsale.Lookup(t.Context(), "example.com", nil)

		require.Error(t, err)
		assert.False(t, ok)
	})
}
