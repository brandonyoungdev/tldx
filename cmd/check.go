package cmd

import (
	"fmt"
	"log/slog"

	"github.com/brandutchmen/domitool/internal/domain"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(checkCmd)
}

var checkCmd = &cobra.Command{
	Use:     "check",
	Aliases: []string{"c"},
	Short:   "Check the availability of a domain",
	Long: `
Check the availability of domain or multiple domains
  `,
	Example: `domitool check google.com facebook.com twitter.com`,
	Args:    cobra.MinimumNArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		gui, err := rootCmd.Flags().GetBool("gui")
		if err != nil {
			slog.Error(fmt.Sprintf("Error getting gui flag: %s", err))
		}
		if gui {
			domain.CheckAndList(args)
		} else {
			domain.CheckAndPrint(args)
		}
	},
}
