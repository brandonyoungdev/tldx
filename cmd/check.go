package cmd

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/brandutchmen/domitool/internal/domain"
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
	Run: func(cmd *cobra.Command, args []string) {
		raw, err := rootCmd.Flags().GetBool("raw")
		if err != nil {
			slog.Error(fmt.Sprintf("Error getting raw flag: %s", err))
		}
		if raw {
			domain.CheckAndPrint(args)
		} else {
			domain.CheckAndList(args)
		}
	},
}
