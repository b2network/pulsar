package cli

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	
	"github.com/b2network/pulsar/modules/coin/types"
	keepertypes "github.com/b2network/pulsar/keeper/types"
)

// GetTxCmd returns the transaction commands for the coin module
func GetTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Coin transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       nil,
	}

	txCmd.AddCommand(
		NewMintCoinsCmd(),
		NewBurnCoinsCmd(),
		NewSetMetadataCmd(),
		NewSetPermissionsCmd(),
	)

	return txCmd
}

// NewMintCoinsCmd returns a CLI command handler for creating a MsgMintCoins transaction
func NewMintCoinsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mint [authority] [module] [amount]",
		Short: "Mint coins for a module",
		Long: `Mint coins for a specific module account.
Example:
$ pulsarcli tx coin mint cosmos1... staking 1000000ubtc
$ pulsarcli tx coin mint cosmos1... mint 500000ubtc,1000000wei
`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			authority := args[0]
			module := args[1]
			amountStr := args[2]

			// Parse coins
			coins, err := parseCoins(amountStr)
			if err != nil {
				return fmt.Errorf("invalid amount: %w", err)
			}

			// Create message
			msg := types.NewMsgMintCoins(authority, module, coins)

			// Validate message
			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			// TODO: Sign and broadcast transaction
			fmt.Printf("Minting %s coins for module %s\n", amountStr, module)
			fmt.Printf("Authority: %s\n", authority)
			fmt.Printf("Message created: %+v\n", msg)

			return nil
		},
	}

	// Add common flags
	addTxFlags(cmd)

	return cmd
}

// NewBurnCoinsCmd returns a CLI command handler for creating a MsgBurnCoins transaction
func NewBurnCoinsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "burn [authority] [module] [amount]",
		Short: "Burn coins from a module",
		Long: `Burn coins from a specific module account.
Example:
$ pulsarcli tx coin burn cosmos1... staking 1000000ubtc
$ pulsarcli tx coin burn cosmos1... mint 500000ubtc,1000000wei
`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			authority := args[0]
			module := args[1]
			amountStr := args[2]

			// Parse coins
			coins, err := parseCoins(amountStr)
			if err != nil {
				return fmt.Errorf("invalid amount: %w", err)
			}

			// Create message
			msg := types.NewMsgBurnCoins(authority, module, coins)

			// Validate message
			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			// TODO: Sign and broadcast transaction
			fmt.Printf("Burning %s coins from module %s\n", amountStr, module)
			fmt.Printf("Authority: %s\n", authority)
			fmt.Printf("Message created: %+v\n", msg)

			return nil
		},
	}

	// Add common flags
	addTxFlags(cmd)

	return cmd
}

// NewSetMetadataCmd returns a CLI command handler for creating a MsgSetMetadata transaction
func NewSetMetadataCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-metadata [authority] [metadata-json]",
		Short: "Set metadata for a coin denomination",
		Long: `Set metadata for a coin denomination using JSON input.
Example:
$ pulsarcli tx coin set-metadata cosmos1... '{"description":"Bitcoin","base":"ubtc","display":"btc","name":"Bitcoin","symbol":"BTC","denom_units":[{"denom":"ubtc","exponent":0},{"denom":"btc","exponent":6}]}'
`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			authority := args[0]
			metadataJSON := args[1]

			// Parse metadata
			var metadata types.Metadata
			if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
				return fmt.Errorf("invalid metadata JSON: %w", err)
			}

			// Create message
			msg := types.NewMsgSetMetadata(authority, metadata)

			// Validate message
			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			// TODO: Sign and broadcast transaction
			fmt.Printf("Setting metadata for %s\n", metadata.Base)
			fmt.Printf("Authority: %s\n", authority)
			fmt.Printf("Message created: %+v\n", msg)

			return nil
		},
	}

	// Add common flags
	addTxFlags(cmd)

	return cmd
}

// NewSetPermissionsCmd returns a CLI command handler for creating a MsgSetPermissions transaction
func NewSetPermissionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-permissions [authority] [module] [permissions]",
		Short: "Set permissions for a module",
		Long: `Set permissions for a specific module.
Example:
$ pulsarcli tx coin set-permissions cosmos1... staking mint,burn
$ pulsarcli tx coin set-permissions cosmos1... gov mint
`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			authority := args[0]
			moduleName := args[1]
			permissionsStr := args[2]

			// Parse permissions
			permissions := strings.Split(permissionsStr, ",")
			for i, perm := range permissions {
				permissions[i] = strings.TrimSpace(perm)
			}

			// Create message
			msg := types.NewMsgSetPermissions(authority, moduleName, permissions)

			// Validate message
			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			// TODO: Sign and broadcast transaction
			fmt.Printf("Setting permissions for module %s: %v\n", moduleName, permissions)
			fmt.Printf("Authority: %s\n", authority)
			fmt.Printf("Message created: %+v\n", msg)

			return nil
		},
	}

	// Add common flags
	addTxFlags(cmd)

	return cmd
}

// parseCoins parses a coins string into Coins
func parseCoins(coinsStr string) (keepertypes.Coins, error) {
	coinsStr = strings.TrimSpace(coinsStr)
	if len(coinsStr) == 0 {
		return nil, fmt.Errorf("empty coins string")
	}

	coinStrs := strings.Split(coinsStr, ",")
	coins := make(keepertypes.Coins, 0, len(coinStrs))

	for _, coinStr := range coinStrs {
		coinStr = strings.TrimSpace(coinStr)
		
		// Find the split between numeric amount and denom
		i := 0
		for i < len(coinStr) && (coinStr[i] >= '0' && coinStr[i] <= '9') {
			i++
		}
		
		if i == 0 || i == len(coinStr) {
			return nil, fmt.Errorf("invalid coin format: %s", coinStr)
		}
		
		amountStr := coinStr[:i]
		denom := coinStr[i:]
		
		amount, err := strconv.ParseInt(amountStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid amount %s: %w", amountStr, err)
		}

		coins = append(coins, keepertypes.Coin{
			Denom:  denom,
			Amount: amount,
		})
	}

	return coins, nil
}

// addTxFlags adds common transaction flags
func addTxFlags(cmd *cobra.Command) {
	cmd.Flags().String("from", "", "Name or address of account that signs the transaction")
	cmd.Flags().String("fees", "", "Fees to pay for the transaction")
	cmd.Flags().String("gas", "auto", "Gas limit to set per-transaction")
	cmd.Flags().String("gas-prices", "", "Gas prices to determine the transaction fee")
	cmd.Flags().String("memo", "", "Memo to include in the transaction")
	cmd.Flags().Bool("dry-run", false, "Perform a dry run without broadcasting")
	cmd.Flags().Bool("generate-only", false, "Generate transaction without broadcasting")
}