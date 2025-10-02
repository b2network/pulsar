package cli

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/b2network/pulsar/client/tx"
	"github.com/b2network/pulsar/modules/fee/types"
	commontypes "github.com/b2network/pulsar/types"
)

// GetTxCmd returns the transaction commands for the fee module
func GetTxCmd() *cobra.Command {
	feeCmd := &cobra.Command{
		Use:   "fee",
		Short: "Fee management transaction commands",
		Long: `Commands for managing fees, gas configurations, and fee denominations.

Examples:
  pulsarcli tx fee add-denom ubtc 0.01ubtc --from mykey
  pulsarcli tx fee update-gas-config bank MsgSend 25000 --from authority
  pulsarcli tx fee set-distribution 0.5 0.3 0.15 0.05 --from authority`,
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       nil,
	}

	feeCmd.AddCommand(
		AddFeeDenomCmd(),
		UpdateFeeDenomCmd(),
		RemoveFeeDenomCmd(),
		SetModuleGasConfigCmd(),
		UpdateGasFactorsCmd(),
		SetFeeDistributionCmd(),
		EstimateFeeCmd(),
	)

	return feeCmd
}

// AddFeeDenomCmd adds a new fee denomination
func AddFeeDenomCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-denom [denom] [min-gas-price] --priority [priority]",
		Short: "Add a new fee denomination",
		Long: `Add a new fee denomination to the system.

The denomination will be enabled by default and can be used for paying transaction fees.

Examples:
  pulsarcli tx fee add-denom ubtc 0.01ubtc --priority 100 --from authority
  pulsarcli tx fee add-denom ueth 100ueth --priority 90 --from authority`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			denom := args[0]
			minGasPrice := args[1]

			priority, _ := cmd.Flags().GetUint32("priority")
			enabled, _ := cmd.Flags().GetBool("enabled")

			// Create the fee denomination
			feeDenom := types.NewFeeDenom(denom, minGasPrice, enabled, priority)

			// Create the message
			msg := types.NewMsgAddFeeDenom(getAuthorityFromFlag(cmd), feeDenom)

			// Validate the message
			if err := msg.ValidateBasic(); err != nil {
				return fmt.Errorf("message validation failed: %w", err)
			}

			// Prepare and execute the transaction
			return tx.PrepareAndExecuteTx(cmd, []commontypes.Msg{msg}, tx.TxCliConfig{})
		},
	}

	cmd.Flags().Uint32("priority", 100, "Priority level for this denomination (higher = more preferred)")
	cmd.Flags().Bool("enabled", true, "Whether this denomination is enabled")
	tx.AddEnhancedTxFlags(cmd)

	return cmd
}

// UpdateFeeDenomCmd updates an existing fee denomination
func UpdateFeeDenomCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-denom [denom] [min-gas-price] --priority [priority]",
		Short: "Update an existing fee denomination",
		Long: `Update the configuration of an existing fee denomination.

Examples:
  pulsarcli tx fee update-denom ubtc 0.02ubtc --priority 100 --from authority
  pulsarcli tx fee update-denom ueth 150ueth --enabled=false --from authority`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			denom := args[0]
			minGasPrice := args[1]

			priority, _ := cmd.Flags().GetUint32("priority")
			enabled, _ := cmd.Flags().GetBool("enabled")

			// Create the fee denomination
			feeDenom := types.NewFeeDenom(denom, minGasPrice, enabled, priority)

			// Create the message
			msg := types.NewMsgUpdateFeeDenom(getAuthorityFromFlag(cmd), feeDenom)

			// Validate the message
			if err := msg.ValidateBasic(); err != nil {
				return fmt.Errorf("message validation failed: %w", err)
			}

			// Prepare and execute the transaction
			return tx.PrepareAndExecuteTx(cmd, []commontypes.Msg{msg}, tx.TxCliConfig{})
		},
	}

	cmd.Flags().Uint32("priority", 100, "Priority level for this denomination")
	cmd.Flags().Bool("enabled", true, "Whether this denomination is enabled")
	tx.AddEnhancedTxFlags(cmd)

	return cmd
}

// RemoveFeeDenomCmd removes a fee denomination
func RemoveFeeDenomCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove-denom [denom]",
		Short: "Remove a fee denomination",
		Long: `Remove a fee denomination from the system.

Warning: This will disable the denomination for fee payments immediately.

Examples:
  pulsarcli tx fee remove-denom uold --from authority`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			denom := args[0]

			// Create the message
			msg := types.NewMsgRemoveFeeDenom(getAuthorityFromFlag(cmd), denom)

			// Validate the message
			if err := msg.ValidateBasic(); err != nil {
				return fmt.Errorf("message validation failed: %w", err)
			}

			// Prepare and execute the transaction
			return tx.PrepareAndExecuteTx(cmd, []commontypes.Msg{msg}, tx.TxCliConfig{})
		},
	}

	tx.AddEnhancedTxFlags(cmd)
	return cmd
}

// SetModuleGasConfigCmd sets gas configuration for a module
func SetModuleGasConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-gas-config [module] [message-type] [gas-amount]",
		Short: "Set gas configuration for a module message type",
		Long: `Set the gas consumption configuration for a specific module and message type.

Examples:
  pulsarcli tx fee set-gas-config bank MsgSend 25000 --from authority
  pulsarcli tx fee set-gas-config coin MsgMint 40000 --from authority
  pulsarcli tx fee set-gas-config staking MsgDelegate 50000 --from authority`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			moduleName := args[0]
			messageType := args[1]
			gasAmountStr := args[2]

			gasAmount, err := strconv.ParseUint(gasAmountStr, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid gas amount: %w", err)
			}

			enabled, _ := cmd.Flags().GetBool("enabled")

			// Create the module gas config
			config := types.NewModuleGasConfig(moduleName, gasAmount, enabled)
			config.SetMsgGasRate(messageType, gasAmount)

			// Create the message
			msg := types.NewMsgSetModuleGasConfig(getAuthorityFromFlag(cmd), config)

			// Validate the message
			if err := msg.ValidateBasic(); err != nil {
				return fmt.Errorf("message validation failed: %w", err)
			}

			// Prepare and execute the transaction
			return tx.PrepareAndExecuteTx(cmd, []commontypes.Msg{msg}, tx.TxCliConfig{})
		},
	}

	cmd.Flags().Bool("enabled", true, "Whether this gas configuration is enabled")
	tx.AddEnhancedTxFlags(cmd)

	return cmd
}

// UpdateGasFactorsCmd updates dynamic gas factors
func UpdateGasFactorsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-gas-factors --size [factor] --complexity [factor] --network [factor] --storage [factor]",
		Short: "Update dynamic gas factors",
		Long: `Update the dynamic gas factors used for gas calculation.

These factors are multipliers applied to base gas amounts based on transaction characteristics.

Examples:
  pulsarcli tx fee update-gas-factors --size 1.2 --complexity 1.1 --network 1.0 --storage 1.3 --from authority`,
		RunE: func(cmd *cobra.Command, args []string) error {
			sizeMultiplier, _ := cmd.Flags().GetFloat64("size")
			complexityFactor, _ := cmd.Flags().GetFloat64("complexity")
			networkFactor, _ := cmd.Flags().GetFloat64("network")
			storageFactor, _ := cmd.Flags().GetFloat64("storage")

			// Create the dynamic gas factors
			factors := types.DynamicGasFactors{
				SizeMultiplier:   sizeMultiplier,
				ComplexityFactor: complexityFactor,
				NetworkFactor:    networkFactor,
				StorageFactor:    storageFactor,
			}

			// Create the message
			msg := types.NewMsgUpdateGasFactors(getAuthorityFromFlag(cmd), factors)

			// Validate the message
			if err := msg.ValidateBasic(); err != nil {
				return fmt.Errorf("message validation failed: %w", err)
			}

			// Prepare and execute the transaction
			return tx.PrepareAndExecuteTx(cmd, []commontypes.Msg{msg}, tx.TxCliConfig{})
		},
	}

	cmd.Flags().Float64("size", 1.0, "Size multiplier factor")
	cmd.Flags().Float64("complexity", 1.0, "Complexity factor")
	cmd.Flags().Float64("network", 1.0, "Network factor")
	cmd.Flags().Float64("storage", 1.0, "Storage factor")
	tx.AddEnhancedTxFlags(cmd)

	return cmd
}

// SetFeeDistributionCmd sets fee distribution configuration
func SetFeeDistributionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-distribution --burn [rate] --validators [rate] --community [rate] --dev [rate]",
		Short: "Set fee distribution configuration",
		Long: `Set how collected fees are distributed among different recipients.

All rates should be decimal values between 0.0 and 1.0, and must sum to 1.0.

Examples:
  pulsarcli tx fee set-distribution --burn 0.5 --validators 0.3 --community 0.15 --dev 0.05 --from authority`,
		RunE: func(cmd *cobra.Command, args []string) error {
			burnRate, _ := cmd.Flags().GetFloat64("burn")
			validatorRate, _ := cmd.Flags().GetFloat64("validators")
			communityRate, _ := cmd.Flags().GetFloat64("community")
			devRate, _ := cmd.Flags().GetFloat64("dev")

			// Validate rates sum to 1.0
			total := burnRate + validatorRate + communityRate + devRate
			if total != 1.0 {
				return fmt.Errorf("distribution rates must sum to 1.0, got: %f", total)
			}

			// Create the fee distribution config
			config := types.FeeDistributionConfig{
				BurnPercentage:   burnRate,
				ValidatorRewards: validatorRate,
				CommunityPool:    communityRate,
				DeveloperFund:    devRate,
			}

			// Create the message
			msg := types.NewMsgSetFeeDistribution(getAuthorityFromFlag(cmd), config)

			// Validate the message
			if err := msg.ValidateBasic(); err != nil {
				return fmt.Errorf("message validation failed: %w", err)
			}

			// Prepare and execute the transaction
			return tx.PrepareAndExecuteTx(cmd, []commontypes.Msg{msg}, tx.TxCliConfig{})
		},
	}

	cmd.Flags().Float64("burn", 0.5, "Percentage of fees to burn (0.0-1.0)")
	cmd.Flags().Float64("validators", 0.3, "Percentage of fees to validators (0.0-1.0)")
	cmd.Flags().Float64("community", 0.15, "Percentage of fees to community pool (0.0-1.0)")
	cmd.Flags().Float64("dev", 0.05, "Percentage of fees to developer fund (0.0-1.0)")
	tx.AddEnhancedTxFlags(cmd)

	return cmd
}

// EstimateFeeCmd estimates fees for a transaction
func EstimateFeeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "estimate [module] [message-type]",
		Short: "Estimate fees for a transaction",
		Long: `Estimate fees for a specific module and message type.

This command provides fee estimates across all supported denominations and priority levels.

Examples:
  pulsarcli tx fee estimate bank MsgSend
  pulsarcli tx fee estimate coin MsgMint --priority high
  pulsarcli tx fee estimate staking MsgDelegate --denom ubtc`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			moduleName := args[0]
			messageType := args[1]

			priority, _ := cmd.Flags().GetString("priority")
			denom, _ := cmd.Flags().GetString("denom")
			verbose, _ := cmd.Flags().GetBool("verbose")

			// Convert priority string to enum
			var feePriority tx.FeePriority = tx.FeePriorityNormal
			switch priority {
			case "low":
				feePriority = tx.FeePriorityLow
			case "normal":
				feePriority = tx.FeePriorityNormal
			case "high":
				feePriority = tx.FeePriorityHigh
			case "fast":
				feePriority = tx.FeePriorityFast
			}

			// Create a mock transaction builder to estimate fees
			homeDir, _ := cmd.Flags().GetString("home")
			chainID, _ := cmd.Flags().GetString("chain-id")
			if chainID == "" {
				chainID = "pulsar-1"
			}

			txBuilder, err := tx.NewTxBuilder(chainID, homeDir)
			if err != nil {
				return fmt.Errorf("failed to create transaction builder: %w", err)
			}

			// Create mock message for estimation
			mockMsgs := []commontypes.Msg{} // Would create based on module/message type

			if denom != "" {
				// Get estimate for specific denomination
				estimate, err := txBuilder.GetFeeEstimateForDenom(mockMsgs, denom, feePriority)
				if err != nil {
					return fmt.Errorf("failed to estimate fee: %w", err)
				}

				fmt.Printf("Fee Estimate for %s.%s:\n", moduleName, messageType)
				fmt.Printf("Denomination: %s\n", estimate.Denom)
				fmt.Printf("Amount: %s\n", estimate.Amount)
				fmt.Printf("Gas Price: %s\n", estimate.GasPrice)
				fmt.Printf("Priority: %d\n", estimate.Priority)
			} else {
				// Get estimates for all denominations
				estimate, err := txBuilder.SuggestOptimalFee(mockMsgs, feePriority)
				if err != nil {
					return fmt.Errorf("failed to estimate fees: %w", err)
				}

				fmt.Printf("Fee Estimates for %s.%s:\n", moduleName, messageType)
				fmt.Printf("Gas Limit: %d\n", estimate.GasLimit)
				fmt.Printf("Recommended: %s\n", estimate.Recommended)
				fmt.Println("\nAll Denominations:")

				for _, est := range estimate.Estimates {
					fmt.Printf("  %s: %s (gas price: %s, priority: %d)\n",
						est.Denom, est.Amount, est.GasPrice, est.Priority)
				}

				if verbose {
					fmt.Println("\nEstimate Details:")
					estimateJSON, _ := json.MarshalIndent(estimate, "", "  ")
					fmt.Println(string(estimateJSON))
				}
			}

			return nil
		},
	}

	cmd.Flags().String("priority", "normal", "Fee priority level (low|normal|high|fast)")
	cmd.Flags().String("denom", "", "Specific denomination to estimate (optional)")
	cmd.Flags().Bool("verbose", false, "Show detailed estimation information")
	cmd.Flags().String("home", "", "Directory for config and data")
	cmd.Flags().String("chain-id", "", "The network chain ID")

	return cmd
}

// getAuthorityFromFlag gets the authority address from the --from flag
func getAuthorityFromFlag(cmd *cobra.Command) string {
	authority, _ := cmd.Flags().GetString("from")
	if authority == "" {
		// In production, this would be the governance module address
		authority = "cosmos1gov1234567890abcdefghijklmnopqrstuvwxyz"
	}
	return authority
}