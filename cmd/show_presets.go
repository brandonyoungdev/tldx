package cmd

import (
	"sort"
	"strings"

	"github.com/brandonyoungdev/tldx/internal/presets"
	"github.com/spf13/cobra"
)

func NewShowPresetsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show-tld-presets",
		Short: "Show available TLD presets",
		Run: func(cmd *cobra.Command, args []string) {
			store := presets.NewTypedStore("tld", presets.DefaultTLDPresets)
			presets.ShowAllPresets(store, func(v []string) string {
				// Sort TLDs by length, then alphabetically
				sorted := make([]string, len(v))
				copy(sorted, v)
				sort.Slice(sorted, func(i, j int) bool {
					if len(sorted[i]) != len(sorted[j]) {
						return len(sorted[i]) < len(sorted[j])
					}
					return sorted[i] < sorted[j]
				})

				return strings.Join(sorted, " ")
			})
		},
	}
}
