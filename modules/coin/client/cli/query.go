package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	
	"github.com/b2network/pulsar/modules/coin/types"
)

// GetQueryCmd returns the query commands for the coin module
func GetQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the coin module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       nil,
	}

	queryCmd.AddCommand(
		GetSupplyCmd(),
		GetTotalSupplyCmd(),
		GetMetadataCmd(),
		GetAllMetadataCmd(),
		GetPermissionsCmd(),
		GetModulePermissionsCmd(),
	)

	return queryCmd
}

// GetSupplyCmd returns a CLI command handler for querying coin supply
func GetSupplyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "supply [denom]",
		Short: "Query the supply of a specific coin denomination",
		Long: `Query the current supply of a specific coin denomination.
Example:
$ pulsarcli query coin supply ubtc
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			denom := args[0]

			// TODO: Query from actual state via keeper
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

// GetTotalSupplyCmd returns a CLI command handler for querying total supply
func GetTotalSupplyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "total-supply",
		Short: "Query the total supply of all coin denominations",
		Long: `Query the total supply of all coin denominations.
Example:
$ pulsarcli query coin total-supply
`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Query from actual state via keeper
			// For now, return mock data
			supplies := map[string]interface{}{
				"supplies": []map[string]interface{}{
					{"denom": "ubtc", "amount": "21000000000000"},
					{"denom": "wei", "amount": "1000000000000000000000"},
				},
				"pagination": map[string]interface{}{
					"total": 2,
				},
			}

			return printJSON(supplies)
		},
	}

	addQueryFlags(cmd)
	return cmd
}

// GetMetadataCmd returns a CLI command handler for querying denom metadata
func GetMetadataCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metadata [denom]",
		Short: "Query the metadata of a specific coin denomination",
		Long: `Query the metadata of a specific coin denomination.
Example:
$ pulsarcli query coin metadata ubtc
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			denom := args[0]

			// TODO: Query from actual state via keeper
			// For now, return mock data
			metadata := map[string]interface{}{
				"description": "Bitcoin on B2 Network",
				"base":        denom,
				"display":     "btc",
				"name":        "Bitcoin",
				"symbol":      "BTC",
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

// GetAllMetadataCmd returns a CLI command handler for querying all denoms metadata
func GetAllMetadataCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "all-metadata",
		Short: "Query the metadata of all coin denominations",
		Long: `Query the metadata of all coin denominations.
Example:
$ pulsarcli query coin all-metadata
`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Query from actual state via keeper
			// For now, return mock data
			metadatas := map[string]interface{}{
				"metadatas": []map[string]interface{}{
					{
						"description": "Bitcoin on B2 Network",
						"base":        "ubtc",
						"display":     "btc",
						"name":        "Bitcoin",
						"symbol":      "BTC",
						"denom_units": []map[string]interface{}{
							{"denom": "ubtc", "exponent": 0},
							{"denom": "btc", "exponent": 6},
						},
					},
					{
						"description": "Ethereum on B2 Network",
						"base":        "wei",
						"display":     "eth",
						"name":        "Ethereum",
						"symbol":      "ETH",
						"denom_units": []map[string]interface{}{
							{"denom": "wei", "exponent": 0},
							{"denom": "eth", "exponent": 18},
						},
					},
				},
				"pagination": map[string]interface{}{
					"total": 2,
				},
			}

			return printJSON(metadatas)
		},
	}

	addQueryFlags(cmd)
	return cmd
}

// GetPermissionsCmd returns a CLI command handler for querying all module permissions
func GetPermissionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "permissions",
		Short: "Query all module permissions",
		Long: `Query all module permissions in the coin module.
Example:
$ pulsarcli query coin permissions
`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Query from actual state via keeper
			// For now, return mock data
			permissions := map[string]interface{}{
				"permissions": []map[string]interface{}{
					{
						"module_name":  "mint",
						"permissions": []string{"mint"},
					},
					{
						"module_name":  "staking", 
						"permissions": []string{"mint", "burn"},
					},
					{
						"module_name":  "gov",
						"permissions": []string{"mint"},
					},
				},
			}

			return printJSON(permissions)
		},
	}

	addQueryFlags(cmd)
	return cmd
}

// GetModulePermissionsCmd returns a CLI command handler for querying permissions of a specific module
func GetModulePermissionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "module-permissions [module-name]",
		Short: "Query permissions for a specific module",
		Long: `Query permissions for a specific module.
Example:
$ pulsarcli query coin module-permissions staking
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			moduleName := args[0]

			// TODO: Query from actual state via keeper
			// For now, return mock data based on module name
			var permissions []string
			switch moduleName {
			case "mint":
				permissions = []string{"mint"}
			case "staking":
				permissions = []string{"mint", "burn"}
			case "gov":
				permissions = []string{"mint"}
			default:
				permissions = []string{}
			}

			modulePerms := map[string]interface{}{
				"module_name":  moduleName,
				"permissions": permissions,
			}

			return printJSON(modulePerms)
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