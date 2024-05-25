package cmd

import (
	"fmt"
	"log/slog"
	"strconv"

	"github.com/brandutchmen/domitool/internal/domain"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(permutateCmd)
	permutateCmd.PersistentFlags().StringP("max-domain-length", "m", "32", "Maximum length of domain name")
	permutateCmd.PersistentFlags().StringSliceP("suffixes", "s", []string{""}, "List of suffixes to append to the keyword")
	permutateCmd.PersistentFlags().StringSliceP("prefixes", "p", []string{""}, "List of prefixes to prepend to the keyword")
	permutateCmd.PersistentFlags().StringSliceP("tlds", "t", []string{"com"}, "List of top-level domains")
}

var permutateCmd = &cobra.Command{
	Use:     "permutate",
	Aliases: []string{"perm", "p"},
	Short:   "Permutate keywords for domain availability",
	Args:    cobra.MinimumNArgs(1),
	Example: `domitool permutate google facebook twitter --max-domain-length 20 --suffixes app,web --prefixes get,buy --tlds com,net`,
	Run: func(cmd *cobra.Command, args []string) {

		maxDomainLengthString, err := cmd.Flags().GetString("max-domain-length")
		if err != nil {
			slog.Error(fmt.Sprintf("Error getting max-domain-length: %s", err))
		}
		maxDomainLength, err := strconv.Atoi(maxDomainLengthString)
		if err != nil {
			slog.Error(fmt.Sprintf("Error converting max-domain-length to int: %s", err))
		}

		suffixes, err := cmd.Flags().GetStringSlice("suffixes")
		if err != nil {
			slog.Error(fmt.Sprintf("Error: %s", err))
		}

		prefixes, err := cmd.Flags().GetStringSlice("prefixes")
		if err != nil {
			slog.Error(fmt.Sprintf("Error: %s", err))
		}

		tlds, err := cmd.Flags().GetStringSlice("tlds")
		if err != nil {
			slog.Error(fmt.Sprintf("Error: %s", err))
		}

		keywords := args

		var domains []string
		prefixes = append(prefixes, "")
		suffixes = append(suffixes, "")

		for _, keyword := range keywords {
			for _, prefix := range prefixes {
				for _, suffix := range suffixes {
					for _, tld := range tlds {
						newDomain := fmt.Sprintf("%s%s%s.%s", prefix, keyword, suffix, tld)
						if len(newDomain) <= maxDomainLength {
							domains = append(domains, newDomain)
						}
					}
				}
			}
		}
  
		gui, err := rootCmd.Flags().GetBool("gui")
		if err != nil {
			slog.Error(fmt.Sprintf("Error getting gui flag: %s", err))
		}
		if gui {
			domain.CheckAndList(domains)
		} else {
			domain.CheckAndPrint(domains)
		}
	},
}
