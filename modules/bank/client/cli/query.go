package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	
	"github.com/b2network/pulsar/modules/bank/types"
)

// GetQueryCmd returns the query commands for the bank module
func GetQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the bank module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       nil,
	}

	queryCmd.AddCommand(
		GetBalanceCmd(),
		GetAllBalancesCmd(),
		GetTotalSupplyCmd(),
		GetSupplyOfCmd(),
		GetDenomMetadataCmd(),
		GetDenomsMetadataCmd(),
	)

	return queryCmd
}

// GetBalanceCmd returns a CLI command handler for querying account balance
func GetBalanceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "balance [address] [denom]",
		Short: "Query an account balance by denomination",
		Long: `Query the balance of an account for a specific denomination.
Example:
$ pulsarcli query bank balance addr1 ubtc
`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			address := args[0]
			denom := args[1]

			// TODO: Query from actual state
			// For now, return mock data
			balance := map[string]interface{}{
				"address": address,
				"denom":   denom,
				"amount":  "1000000",
			}

			return printJSON(balance)
		},
	}

	addQueryFlags(cmd)
	return cmd
}

// GetAllBalancesCmd returns a CLI command handler for querying all account balances
func GetAllBalancesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "balances [address]",
		Short: "Query all balances of an account",
		Long: `Query all coin balances of an account.
Example:
$ pulsarcli query bank balances addr1
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			address := args[0]

			// TODO: Query from actual state
			// For now, return mock data
			balances := map[string]interface{}{
				"address": address,
				"balances": []map[string]string{
					{"denom": "ubtc", "amount": "1000000"},
					{"denom": "wei", "amount": "5000000000"},
				},
			}

			return printJSON(balances)
		},
	}

	addQueryFlags(cmd)
	return cmd
}

// GetTotalSupplyCmd returns a CLI command handler for querying total supply
func GetTotalSupplyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "total",
		Short: "Query the total supply of all coins",
		Long: `Query the total supply of all coins in circulation.
Example:
$ pulsarcli query bank total
`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Query from actual state
			// For now, return mock data
			supply := map[string]interface{}{
				"supply": []map[string]string{
					{"denom": "ubtc", "amount": "21000000000000"},
					{"denom": "wei", "amount": "1000000000000000000000"},
				},
			}

			return printJSON(supply)
		},
	}

	addQueryFlags(cmd)
	return cmd
}

// GetSupplyOfCmd returns a CLI command handler for querying supply of a denom
func GetSupplyOfCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "supply-of [denom]",
		Short: "Query the supply of a single coin denomination",
		Long: `Query the current supply of a specific coin denomination.
Example:
$ pulsarcli query bank supply-of ubtc
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			denom := args[0]

			// TODO: Query from actual state
			// For now, return mock data
			supply := map[string]interface{}{
				"denom":  denom,
				"amount": "21000000000000",
			}

			return printJSON(supply)
		},
	}

	addQueryFlags(cmd)
	return cmd
}

// GetDenomMetadataCmd returns a CLI command handler for querying denom metadata
func GetDenomMetadataCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "denom-metadata [denom]",
		Short: "Query the metadata of a single coin denomination",
		Long: `Query the metadata of a specific coin denomination.
Example:
$ pulsarcli query bank denom-metadata ubtc
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			denom := args[0]

			// TODO: Query from actual state
			// For now, return mock data
			metadata := map[string]interface{}{
				"denom":       denom,
				"display":     "btc",
				"name":        "Bitcoin",
				"symbol":      "BTC",
				"description": "Bitcoin on B2 Network",
				"denom_units": []map[string]interface{}{
					{"denom": "ubtc", "exponent": 0, "aliases": []string{"microbitcoin"}},
					{"denom": "mbtc", "exponent": 3, "aliases": []string{"millibitcoin"}},
					{"denom": "btc", "exponent": 6, "aliases": []string{"bitcoin"}},
				},
			}

			return printJSON(metadata)
		},
	}

	addQueryFlags(cmd)
	return cmd
}

// GetDenomsMetadataCmd returns a CLI command handler for querying all denoms metadata
func GetDenomsMetadataCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "denoms-metadata",
		Short: "Query the metadata of all coin denominations",
		Long: `Query the metadata of all coin denominations.
Example:
$ pulsarcli query bank denoms-metadata
`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Query from actual state
			// For now, return mock data
			metadatas := map[string]interface{}{
				"metadatas": []map[string]interface{}{
					{
						"denom":       "ubtc",
						"display":     "btc",
						"name":        "Bitcoin",
						"symbol":      "BTC",
						"description": "Bitcoin on B2 Network",
					},
					{
						"denom":       "wei",
						"display":     "eth",
						"name":        "Ethereum",
						"symbol":      "ETH",
						"description": "Ethereum on B2 Network",
					},
				},
			}

			return printJSON(metadatas)
		},
	}

	addQueryFlags(cmd)
	return cmd
}

// addQueryFlags adds common query flags
func addQueryFlags(cmd *cobra.Command) {
	cmd.Flags().String("output", "json", "Output format (json|text)")
	cmd.Flags().String("node", "tcp://localhost:26657", "RPC endpoint")
	cmd.Flags().Int64("height", 0, "Use a specific height to query state at")
}

// printJSON prints data as formatted JSON
func printJSON(data interface{}) error {
	bz, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(bz))
	return nil
}