package mcpserver

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/brandonyoungdev/tldx/internal/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func isolatedService(t *testing.T) *Service {
	t.Helper()
	t.Setenv("TLDX_CONFIG", filepath.Join(t.TempDir(), "config.toml"))
	return NewService()
}

// specsFor builds count specs spread over the given keywords and TLDs, so the
// dimensions overBudgetMessage reads back are exactly as stated.
func specsFor(count, keywords, tlds int) []resolver.DomainSpec {
	specs := make([]resolver.DomainSpec, 0, count)
	for i := range count {
		specs = append(specs, resolver.DomainSpec{
			Domain:  fmt.Sprintf("d%d.example", i),
			Keyword: fmt.Sprintf("keyword%d", i%keywords),
			TLD:     fmt.Sprintf("tld%d", i%tlds),
		})
	}
	return specs
}

func TestOverBudgetMessage_BlamesPrefixesAndSuffixes(t *testing.T) {
	msg := overBudgetMessage(specsFor(1200, 1, 1), "")

	assert.Contains(t, msg, "1200 domains exceeds")
	assert.Contains(t, msg, "prefixes and suffixes multiply")
	assert.NotContains(t, msg, "tld_preset", "no preset was used")
}

func TestOverBudgetMessage_FallsBackWhenNoDimensionDominates(t *testing.T) {
	msg := overBudgetMessage(specsFor(1, 1, 1), "")

	assert.Contains(t, msg, "Every dimension is already small")
	assert.Contains(t, msg, "limit=", "the escape hatch is always offered")
}

func TestCollect_ReportsAnEarlyStopWithoutALimit(t *testing.T) {
	s := isolatedService(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	resp := s.collect(ctx, s.context(), specsFor(3, 1, 1), 0)

	require.True(t, resp.Truncated)
	assert.Zero(t, resp.Checked)
	assert.Contains(t, resp.Note, "Stopped early after 0 of 3 domains")
}
