package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/b2network/pulsar/modules/fee/config"
	"github.com/b2network/pulsar/modules/fee/types"
)

// GetQueryCmd returns the query commands for the fee module
func GetQueryCmd() *cobra.Command {
	feeQueryCmd := &cobra.Command{
		Use:   "fee",
		Short: "Query fee module information",
		Long: `Query commands for the fee module, including fee denominations, gas configurations,
and fee distribution settings.

Examples:
  pulsarcli query fee params
  pulsarcli query fee denominations
  pulsarcli query fee gas-config bank
  pulsarcli query fee estimate bank MsgSend`,
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       nil,
	}

	feeQueryCmd.AddCommand(
		QueryParamsCmd(),
		QueryFeeDenominationsCmd(),
		QueryFeeDenomCmd(),
		QueryGasConfigCmd(),
		QueryGasConfigsCmd(),
		QueryDynamicFactorsCmd(),
		QueryFeeDistributionCmd(),
		QueryGasPriceStatsCmd(),
		QueryFeeEstimateCmd(),
		QuerySupportedDenomsCmd(),
	)

	return feeQueryCmd
}

// QueryParamsCmd queries the fee module parameters
func QueryParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query fee module parameters",
		Long: `Query the current parameters of the fee module.

Example:
  pulsarcli query fee params`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// In a real implementation, this would query the chain
			// For now, return default parameters
			params := types.DefaultParams()

			output, _ := cmd.Flags().GetString("output")
			return printOutput(params, output)
		},
	}

	cmd.Flags().String("output", "text", "Output format (text|json)")
	return cmd
}

// QueryFeeDenominationsCmd queries all fee denominations
func QueryFeeDenominationsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "denominations",
		Aliases: []string{"denoms"},
		Short:   "Query all fee denominations",
		Long: `Query all available fee denominations and their configurations.

Example:
  pulsarcli query fee denominations`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// In a real implementation, this would query the chain
			// For now, return default denominations
			denoms := types.DefaultFeeDenoms()

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return printOutput(denoms, output)
			}

			// Text format
			fmt.Println("Fee Denominations:")
			fmt.Println("==================")
			for i, denom := range denoms {
				fmt.Printf("%d. %s\n", i+1, denom.Denom)
				fmt.Printf("   Min Gas Price: %s\n", denom.MinGasPrice)
				fmt.Printf("   Enabled: %v\n", denom.IsEnabled())
				fmt.Printf("   Priority: %d\n", denom.Priority)
				if i < len(denoms)-1 {
					fmt.Println()
				}
			}

			return nil
		},
	}

	cmd.Flags().String("output", "text", "Output format (text|json)")
	return cmd
}

// QueryFeeDenomCmd queries a specific fee denomination
func QueryFeeDenomCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "denomination [denom]",
		Short: "Query a specific fee denomination",
		Long: `Query the configuration of a specific fee denomination.

Example:
  pulsarcli query fee denomination ubtc`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			denom := args[0]

			// In a real implementation, this would query the chain
			// For now, find in default denominations
			denoms := types.DefaultFeeDenoms()
			var targetDenom *types.FeeDenom
			for _, d := range denoms {
				if d.Denom == denom {
					targetDenom = &d
					break
				}
			}

			if targetDenom == nil {
				return fmt.Errorf("fee denomination %s not found", denom)
			}

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return printOutput(targetDenom, output)
			}

			// Text format
			fmt.Printf("Fee Denomination: %s\n", targetDenom.Denom)
			fmt.Printf("Min Gas Price: %s\n", targetDenom.MinGasPrice)
			fmt.Printf("Enabled: %v\n", targetDenom.IsEnabled())
			fmt.Printf("Priority: %d\n", targetDenom.Priority)

			return nil
		},
	}

	cmd.Flags().String("output", "text", "Output format (text|json)")
	return cmd
}

// QueryGasConfigCmd queries gas configuration for a module
func QueryGasConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gas-config [module]",
		Short: "Query gas configuration for a module",
		Long: `Query the gas configuration for a specific module.

Example:
  pulsarcli query fee gas-config bank
  pulsarcli query fee gas-config coin`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			moduleName := args[0]

			// In a real implementation, this would query the chain
			// For now, return from default genesis
			genesis := types.DefaultGenesis()
			var targetConfig *types.ModuleGasConfig
			for _, config := range genesis.ModuleGasConfigs {
				if config.ModuleName == moduleName {
					targetConfig = &config
					break
				}
			}

			if targetConfig == nil {
				return fmt.Errorf("gas configuration for module %s not found", moduleName)
			}

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return printOutput(targetConfig, output)
			}

			// Text format
			fmt.Printf("Gas Configuration for %s:\n", targetConfig.ModuleName)
			fmt.Printf("Base Gas: %d\n", targetConfig.BaseGas)
			fmt.Printf("Enabled: %v\n", targetConfig.Enabled)
			fmt.Println("Message Gas Rates:")
			for msgType, gasRate := range targetConfig.MsgGasRates {
				fmt.Printf("  %s: %d\n", msgType, gasRate)
			}

			return nil
		},
	}

	cmd.Flags().String("output", "text", "Output format (text|json)")
	return cmd
}

// QueryGasConfigsCmd queries all gas configurations
func QueryGasConfigsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gas-configs",
		Short: "Query all module gas configurations",
		Long: `Query gas configurations for all modules.

Example:
  pulsarcli query fee gas-configs`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// In a real implementation, this would query the chain
			genesis := types.DefaultGenesis()

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return printOutput(genesis.ModuleGasConfigs, output)
			}

			// Text format
			fmt.Println("Module Gas Configurations:")
			fmt.Println("=========================")
			for i, config := range genesis.ModuleGasConfigs {
				fmt.Printf("%d. Module: %s\n", i+1, config.ModuleName)
				fmt.Printf("   Base Gas: %d\n", config.BaseGas)
				fmt.Printf("   Enabled: %v\n", config.Enabled)
				fmt.Printf("   Message Types: %d\n", len(config.MsgGasRates))
				if i < len(genesis.ModuleGasConfigs)-1 {
					fmt.Println()
				}
			}

			return nil
		},
	}

	cmd.Flags().String("output", "text", "Output format (text|json)")
	return cmd
}

// QueryDynamicFactorsCmd queries dynamic gas factors
func QueryDynamicFactorsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dynamic-factors",
		Short: "Query dynamic gas factors",
		Long: `Query the current dynamic gas factors used for gas calculation.

Example:
  pulsarcli query fee dynamic-factors`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// In a real implementation, this would query the chain
			factors := types.DefaultDynamicGasFactors()

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return printOutput(factors, output)
			}

			// Text format
			fmt.Println("Dynamic Gas Factors:")
			fmt.Printf("Size Multiplier: %.2f\n", factors.SizeMultiplier)
			fmt.Printf("Complexity Factor: %.2f\n", factors.ComplexityFactor)
			fmt.Printf("Network Factor: %.2f\n", factors.NetworkFactor)
			fmt.Printf("Storage Factor: %.2f\n", factors.StorageFactor)

			return nil
		},
	}

	cmd.Flags().String("output", "text", "Output format (text|json)")
	return cmd
}

// QueryFeeDistributionCmd queries fee distribution configuration
func QueryFeeDistributionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "distribution",
		Short: "Query fee distribution configuration",
		Long: `Query how collected fees are distributed among different recipients.

Example:
  pulsarcli query fee distribution`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// In a real implementation, this would query the chain
			genesis := types.DefaultGenesis()
			config := genesis.FeeDistributionConfig

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return printOutput(config, output)
			}

			// Text format
			fmt.Println("Fee Distribution Configuration:")
			fmt.Printf("Burn Percentage: %.1f%%\n", config.BurnPercentage*100)
			fmt.Printf("Validator Rewards: %.1f%%\n", config.ValidatorRewards*100)
			fmt.Printf("Community Pool: %.1f%%\n", config.CommunityPool*100)
			fmt.Printf("Developer Fund: %.1f%%\n", config.DeveloperFund*100)

			total := config.BurnPercentage + config.ValidatorRewards + config.CommunityPool + config.DeveloperFund
			fmt.Printf("Total: %.1f%%\n", total*100)

			return nil
		},
	}

	cmd.Flags().String("output", "text", "Output format (text|json)")
	return cmd
}

// QueryGasPriceStatsCmd queries gas price statistics
func QueryGasPriceStatsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gas-price-stats [denom]",
		Short: "Query gas price statistics for a denomination",
		Long: `Query gas price statistics including min, max, average, and recommended prices.

Example:
  pulsarcli query fee gas-price-stats ubtc
  pulsarcli query fee gas-price-stats ueth`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			denom := args[0]

			// In a real implementation, this would query the chain
			// For now, simulate gas price stats
			stats := simulateGasPriceStats(denom)

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return printOutput(stats, output)
			}

			// Text format
			fmt.Printf("Gas Price Statistics for %s:\n", stats.Denom)
			fmt.Printf("Min Price: %s\n", stats.MinPrice)
			fmt.Printf("Max Price: %s\n", stats.MaxPrice)
			fmt.Printf("Average Price: %s\n", stats.AveragePrice)
			fmt.Printf("Median Price: %s\n", stats.MedianPrice)
			fmt.Printf("Recommended Price: %s\n", stats.RecommendedPrice)
			fmt.Printf("Sample Size: %d\n", stats.SampleSize)
			fmt.Printf("Last Updated: %d\n", stats.UpdateTime)

			return nil
		},
	}

	cmd.Flags().String("output", "text", "Output format (text|json)")
	return cmd
}

// QueryFeeEstimateCmd estimates fees for a transaction
func QueryFeeEstimateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "estimate [module] [message-type]",
		Short: "Estimate fees for a transaction",
		Long: `Estimate fees for a specific module and message type.

Example:
  pulsarcli query fee estimate bank MsgSend
  pulsarcli query fee estimate coin MsgMint --priority high`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			moduleName := args[0]
			messageType := args[1]

			priority, _ := cmd.Flags().GetString("priority")
			denom, _ := cmd.Flags().GetString("denom")

			// Simulate fee estimation using gas standards
			gasStandards := config.DefaultGasStandards()
			estimate := simulateFeeEstimate(gasStandards, moduleName, messageType, priority, denom)

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return printOutput(estimate, output)
			}

			// Text format
			fmt.Printf("Fee Estimate for %s.%s:\n", moduleName, messageType)
			fmt.Printf("Gas Limit: %d\n", estimate.GasLimit)
			fmt.Printf("Priority: %s\n", priority)

			if denom != "" {
				// Show estimate for specific denomination
				for _, est := range estimate.Estimates {
					if est.Denom == denom {
						fmt.Printf("Denomination: %s\n", est.Denom)
						fmt.Printf("Amount: %s\n", est.Amount)
						fmt.Printf("Gas Price: %s\n", est.GasPrice)
						break
					}
				}
			} else {
				// Show all estimates
				fmt.Printf("Recommended: %s\n", estimate.Recommended)
				fmt.Println("All Denominations:")
				for _, est := range estimate.Estimates {
					fmt.Printf("  %s: %s (gas price: %s, priority: %d)\n",
						est.Denom, est.Amount, est.GasPrice, est.Priority)
				}
			}

			return nil
		},
	}

	cmd.Flags().String("priority", "normal", "Fee priority level (low|normal|high|fast)")
	cmd.Flags().String("denom", "", "Specific denomination to estimate (optional)")
	cmd.Flags().String("output", "text", "Output format (text|json)")
	return cmd
}

// QuerySupportedDenomsCmd queries supported fee denominations
func QuerySupportedDenomsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "supported-denoms",
		Short: "Query supported fee denominations",
		Long: `Query the list of currently supported fee denominations.

Example:
  pulsarcli query fee supported-denoms`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// In a real implementation, this would query the chain
			denoms := types.DefaultFeeDenoms()

			// Filter enabled denominations
			var supportedDenoms []string
			for _, denom := range denoms {
				if denom.IsEnabled() {
					supportedDenoms = append(supportedDenoms, denom.Denom)
				}
			}

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return printOutput(supportedDenoms, output)
			}

			// Text format
			fmt.Println("Supported Fee Denominations:")
			for i, denom := range supportedDenoms {
				fmt.Printf("%d. %s\n", i+1, denom)
			}

			return nil
		},
	}

	cmd.Flags().String("output", "text", "Output format (text|json)")
	return cmd
}

// Helper functions

// printOutput prints the output in the specified format
func printOutput(data interface{}, format string) error {
	if format == "json" {
		output, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(output))
	} else {
		// Text format would be handled by individual commands
		output, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(output))
	}
	return nil
}

// simulateGasPriceStats simulates gas price statistics for testing
func simulateGasPriceStats(denom string) types.GasPriceStats {
	// Base price varies by denomination
	var basePrice float64
	switch denom {
	case "ubtc":
		basePrice = 0.01
	case "ueth":
		basePrice = 100
	case "upulse":
		basePrice = 1000
	default:
		basePrice = 1
	}

	return types.GasPriceStats{
		Denom:            denom,
		MinPrice:         types.FormatGasPrice(basePrice*0.8, denom),
		MaxPrice:         types.FormatGasPrice(basePrice*2.0, denom),
		AveragePrice:     types.FormatGasPrice(basePrice*1.2, denom),
		MedianPrice:      types.FormatGasPrice(basePrice*1.1, denom),
		RecommendedPrice: types.FormatGasPrice(basePrice*1.5, denom),
		UpdateTime:       1640995200, // Timestamp
		SampleSize:       50,
	}
}

// simulateFeeEstimate simulates fee estimation for testing
func simulateFeeEstimate(gasStandards config.GasStandards, module, msgType, priority, denom string) FeeEstimateResult {
	// Get base gas for message type
	var baseGas uint64 = 21000 // Default base gas

	switch module {
	case "bank":
		switch msgType {
		case "MsgSend":
			baseGas = gasStandards.BankGasConfig.Send
		case "MsgMultiSend":
			baseGas = gasStandards.BankGasConfig.MultiSend
		}
	case "coin":
		switch msgType {
		case "MsgMint":
			baseGas = gasStandards.CoinGasConfig.Mint
		case "MsgBurn":
			baseGas = gasStandards.CoinGasConfig.Burn
		}
	case "staking":
		switch msgType {
		case "MsgDelegate":
			baseGas = gasStandards.StakingGasConfig.Delegate
		case "MsgUndelegate":
			baseGas = gasStandards.StakingGasConfig.Undelegate
		}
	}

	// Priority multipliers
	var multiplier float64 = 1.5 // Normal
	switch priority {
	case "low":
		multiplier = 1.0
	case "normal":
		multiplier = 1.5
	case "high":
		multiplier = 2.0
	case "fast":
		multiplier = 3.0
	}

	// Generate estimates for all denominations
	denoms := types.DefaultFeeDenoms()
	estimates := make([]FeeEstimate, 0, len(denoms))

	for _, d := range denoms {
		if !d.IsEnabled() {
			continue
		}

		// Parse base price
		basePrice := 0.01 // Default
		switch d.Denom {
		case "ubtc":
			basePrice = 0.01
		case "ueth":
			basePrice = 100
		case "upulse":
			basePrice = 1000
		}

		gasPrice := basePrice * multiplier
		feeAmount := gasPrice * float64(baseGas)

		estimates = append(estimates, FeeEstimate{
			Denom:    d.Denom,
			Amount:   fmt.Sprintf("%.0f", feeAmount),
			GasPrice: fmt.Sprintf("%.6f", gasPrice),
			Priority: d.Priority,
		})
	}

	return FeeEstimateResult{
		GasLimit:    baseGas,
		Estimates:   estimates,
		Recommended: "ubtc", // Highest priority denomination
	}
}

// Helper types for estimates
type FeeEstimate struct {
	Denom    string `json:"denom"`
	Amount   string `json:"amount"`
	GasPrice string `json:"gas_price"`
	Priority uint32 `json:"priority"`
}

type FeeEstimateResult struct {
	GasLimit    uint64        `json:"gas_limit"`
	Estimates   []FeeEstimate `json:"estimates"`
	Recommended string        `json:"recommended"`
}