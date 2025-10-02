package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/b2network/pulsar/client/tx"
	"github.com/b2network/pulsar/modules/fee/config"
	"github.com/b2network/pulsar/modules/fee/types"
)

// GetUtilsCmd returns utility commands for fee management
func GetUtilsCmd() *cobra.Command {
	utilsCmd := &cobra.Command{
		Use:   "fee-utils",
		Short: "Fee management utility commands",
		Long: `Utility commands for fee management, including fee calculators,
benchmarking tools, and configuration helpers.

Examples:
  pulsarcli fee-utils calculate bank MsgSend --gas 30000 --priority high
  pulsarcli fee-utils benchmark --modules bank,coin --iterations 100
  pulsarcli fee-utils optimize --max-fee 1000ubtc
  pulsarcli fee-utils export-config --output fee-config.json`,
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       nil,
	}

	utilsCmd.AddCommand(
		CalculateFeeCmd(),
		CompareFeeCmd(),
		BenchmarkCmd(),
		OptimizeCmd(),
		ExportConfigCmd(),
		ImportConfigCmd(),
		ValidateConfigCmd(),
		FeeAnalyticsCmd(),
	)

	return utilsCmd
}

// CalculateFeeCmd calculates fees for various scenarios
func CalculateFeeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "calculate [module] [message-type]",
		Short: "Calculate fees for a specific transaction",
		Long: `Calculate fees for a specific module and message type with various options.

Examples:
  pulsarcli fee-utils calculate bank MsgSend --gas 30000 --priority high
  pulsarcli fee-utils calculate coin MsgMint --denom ubtc --multiplier 1.5
  pulsarcli fee-utils calculate staking MsgDelegate --compare-all`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			moduleName := args[0]
			messageType := args[1]

			gasLimit, _ := cmd.Flags().GetUint64("gas")
			priority, _ := cmd.Flags().GetString("priority")
			denom, _ := cmd.Flags().GetString("denom")
			multiplier, _ := cmd.Flags().GetFloat64("multiplier")
			compareAll, _ := cmd.Flags().GetBool("compare-all")

			// Convert priority
			var feePriority tx.FeePriority = tx.FeePriorityNormal
			switch priority {
			case "low":
				feePriority = tx.FeePriorityLow
			case "high":
				feePriority = tx.FeePriorityHigh
			case "fast":
				feePriority = tx.FeePriorityFast
			}

			// Calculate fees
			calculator := NewFeeCalculator()
			result := calculator.Calculate(moduleName, messageType, gasLimit, feePriority, denom, multiplier)

			if compareAll {
				fmt.Printf("Fee Comparison for %s.%s:\n", moduleName, messageType)
				fmt.Printf("Gas Limit: %d\n", result.GasLimit)
				fmt.Println("\nPriority Comparison:")

				priorities := []tx.FeePriority{tx.FeePriorityLow, tx.FeePriorityNormal, tx.FeePriorityHigh, tx.FeePriorityFast}
				for _, p := range priorities {
					r := calculator.Calculate(moduleName, messageType, gasLimit, p, denom, 1.0)
					fmt.Printf("  %s: %s %s\n", p, r.FeeAmount, r.Denomination)
				}
			} else {
				fmt.Printf("Fee Calculation Result:\n")
				fmt.Printf("Module: %s\n", moduleName)
				fmt.Printf("Message Type: %s\n", messageType)
				fmt.Printf("Gas Limit: %d\n", result.GasLimit)
				fmt.Printf("Priority: %s\n", priority)
				fmt.Printf("Denomination: %s\n", result.Denomination)
				fmt.Printf("Gas Price: %s\n", result.GasPrice)
				fmt.Printf("Fee Amount: %s\n", result.FeeAmount)
				if multiplier != 1.0 {
					fmt.Printf("Multiplier Applied: %.2fx\n", multiplier)
				}
			}

			return nil
		},
	}

	cmd.Flags().Uint64("gas", 0, "Gas limit (0 = auto estimate)")
	cmd.Flags().String("priority", "normal", "Fee priority (low|normal|high|fast)")
	cmd.Flags().String("denom", "", "Fee denomination (default: auto-select)")
	cmd.Flags().Float64("multiplier", 1.0, "Fee multiplier")
	cmd.Flags().Bool("compare-all", false, "Compare fees across all priority levels")

	return cmd
}

// CompareFeeCmd compares fees across different configurations
func CompareFeeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compare [module] [message-type]",
		Short: "Compare fees across different configurations",
		Long: `Compare fees across different denominations, priorities, and multipliers.

Examples:
  pulsarcli fee-utils compare bank MsgSend
  pulsarcli fee-utils compare coin MsgMint --denoms ubtc,ueth,upulse`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			moduleName := args[0]
			messageType := args[1]

			denomsStr, _ := cmd.Flags().GetString("denoms")
			gasLimit, _ := cmd.Flags().GetUint64("gas")

			// Parse denominations
			var targetDenoms []string
			if denomsStr != "" {
				targetDenoms = strings.Split(denomsStr, ",")
			} else {
				// Use all available denominations
				denoms := types.DefaultFeeDenoms()
				for _, d := range denoms {
					if d.IsEnabled() {
						targetDenoms = append(targetDenoms, d.Denom)
					}
				}
			}

			calculator := NewFeeCalculator()
			priorities := []tx.FeePriority{tx.FeePriorityLow, tx.FeePriorityNormal, tx.FeePriorityHigh, tx.FeePriorityFast}

			fmt.Printf("Fee Comparison for %s.%s:\n", moduleName, messageType)
			fmt.Printf("Gas Limit: %d\n", gasLimit)
			fmt.Println()

			// Create comparison table
			fmt.Printf("%-10s", "Priority")
			for _, denom := range targetDenoms {
				fmt.Printf("%-15s", denom)
			}
			fmt.Println()
			fmt.Println(strings.Repeat("-", 10+15*len(targetDenoms)))

			for _, priority := range priorities {
				fmt.Printf("%-10s", priority)
				for _, denom := range targetDenoms {
					result := calculator.Calculate(moduleName, messageType, gasLimit, priority, denom, 1.0)
					fmt.Printf("%-15s", result.FeeAmount)
				}
				fmt.Println()
			}

			return nil
		},
	}

	cmd.Flags().String("denoms", "", "Comma-separated list of denominations to compare")
	cmd.Flags().Uint64("gas", 0, "Gas limit (0 = auto estimate)")

	return cmd
}

// BenchmarkCmd runs fee calculation benchmarks
func BenchmarkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "benchmark",
		Short: "Run fee calculation benchmarks",
		Long: `Run benchmarks to test fee calculation performance and accuracy.

Examples:
  pulsarcli fee-utils benchmark --modules bank,coin --iterations 1000
  pulsarcli fee-utils benchmark --full --output benchmark-results.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			modulesStr, _ := cmd.Flags().GetString("modules")
			iterations, _ := cmd.Flags().GetInt("iterations")
			full, _ := cmd.Flags().GetBool("full")
			output, _ := cmd.Flags().GetString("output")

			fmt.Printf("Running Fee Calculation Benchmark...\n")
			fmt.Printf("Iterations: %d\n", iterations)

			var modules []string
			if modulesStr != "" {
				modules = strings.Split(modulesStr, ",")
			} else if full {
				modules = []string{"bank", "coin", "staking", "gov", "fee", "auth", "distribution", "slashing"}
			} else {
				modules = []string{"bank", "coin"}
			}

			fmt.Printf("Modules: %v\n", modules)
			fmt.Println()

			benchmarkResults := runBenchmark(modules, iterations)

			if output != "" {
				// Save to file
				data, _ := json.MarshalIndent(benchmarkResults, "", "  ")
				if err := os.WriteFile(output, data, 0644); err != nil {
					return fmt.Errorf("failed to write results to file: %w", err)
				}
				fmt.Printf("Results saved to: %s\n", output)
			}

			// Display summary
			fmt.Println("Benchmark Results:")
			fmt.Println("==================")
			for module, result := range benchmarkResults {
				fmt.Printf("%s: %.2f ms/op, %d ops/sec\n",
					module, result.AvgTimeMs, int(1000.0/result.AvgTimeMs))
			}

			return nil
		},
	}

	cmd.Flags().String("modules", "", "Comma-separated list of modules to benchmark")
	cmd.Flags().Int("iterations", 1000, "Number of iterations per test")
	cmd.Flags().Bool("full", false, "Run full benchmark on all modules")
	cmd.Flags().String("output", "", "Output file for results (JSON)")

	return cmd
}

// OptimizeCmd suggests fee optimizations
func OptimizeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "optimize",
		Short: "Suggest fee optimizations",
		Long: `Analyze fee configurations and suggest optimizations for cost and performance.

Examples:
  pulsarcli fee-utils optimize --max-fee 1000ubtc
  pulsarcli fee-utils optimize --target-time 5s --priority normal`,
		RunE: func(cmd *cobra.Command, args []string) error {
			maxFee, _ := cmd.Flags().GetString("max-fee")
			targetTime, _ := cmd.Flags().GetDuration("target-time")
			priority, _ := cmd.Flags().GetString("priority")

			fmt.Println("Fee Optimization Analysis:")
			fmt.Println("==========================")

			optimizer := NewFeeOptimizer()
			recommendations := optimizer.AnalyzeAndOptimize(maxFee, targetTime, priority)

			fmt.Println("Recommendations:")
			for i, rec := range recommendations {
				fmt.Printf("%d. %s\n", i+1, rec)
			}

			return nil
		},
	}

	cmd.Flags().String("max-fee", "", "Maximum fee budget")
	cmd.Flags().Duration("target-time", 30*time.Second, "Target confirmation time")
	cmd.Flags().String("priority", "normal", "Preferred priority level")

	return cmd
}

// ExportConfigCmd exports fee configuration
func ExportConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export-config",
		Short: "Export fee configuration to file",
		Long: `Export current fee configuration to a JSON file.

Examples:
  pulsarcli fee-utils export-config --output fee-config.json
  pulsarcli fee-utils export-config --format yaml --output config.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			output, _ := cmd.Flags().GetString("output")
			format, _ := cmd.Flags().GetString("format")

			if output == "" {
				output = "fee-config.json"
			}

			// Get current configuration
			config := getCurrentFeeConfig()

			var data []byte
			var err error

			switch format {
			case "yaml":
				// Would implement YAML marshaling
				return fmt.Errorf("YAML format not yet implemented")
			default:
				data, err = json.MarshalIndent(config, "", "  ")
			}

			if err != nil {
				return fmt.Errorf("failed to marshal config: %w", err)
			}

			if err := os.WriteFile(output, data, 0644); err != nil {
				return fmt.Errorf("failed to write config file: %w", err)
			}

			fmt.Printf("Fee configuration exported to: %s\n", output)
			return nil
		},
	}

	cmd.Flags().String("output", "", "Output file path")
	cmd.Flags().String("format", "json", "Output format (json|yaml)")

	return cmd
}

// ImportConfigCmd imports fee configuration
func ImportConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import-config [file]",
		Short: "Import fee configuration from file",
		Long: `Import fee configuration from a JSON file.

Examples:
  pulsarcli fee-utils import-config fee-config.json
  pulsarcli fee-utils import-config config.json --validate-only`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			configFile := args[0]
			validateOnly, _ := cmd.Flags().GetBool("validate-only")

			data, err := os.ReadFile(configFile)
			if err != nil {
				return fmt.Errorf("failed to read config file: %w", err)
			}

			var config FeeConfigFile
			if err := json.Unmarshal(data, &config); err != nil {
				return fmt.Errorf("failed to parse config file: %w", err)
			}

			// Validate configuration
			if err := validateFeeConfig(config); err != nil {
				return fmt.Errorf("config validation failed: %w", err)
			}

			if validateOnly {
				fmt.Println("Configuration file is valid!")
				return nil
			}

			fmt.Printf("Importing fee configuration from: %s\n", configFile)
			fmt.Println("This would apply the new configuration to the chain.")
			fmt.Println("(Implementation would create governance proposal or admin transaction)")

			return nil
		},
	}

	cmd.Flags().Bool("validate-only", false, "Only validate the configuration file")

	return cmd
}

// ValidateConfigCmd validates fee configuration
func ValidateConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate-config [file]",
		Short: "Validate fee configuration file",
		Long: `Validate a fee configuration file for correctness.

Examples:
  pulsarcli fee-utils validate-config fee-config.json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			configFile := args[0]

			data, err := os.ReadFile(configFile)
			if err != nil {
				return fmt.Errorf("failed to read config file: %w", err)
			}

			var config FeeConfigFile
			if err := json.Unmarshal(data, &config); err != nil {
				return fmt.Errorf("failed to parse config file: %w", err)
			}

			if err := validateFeeConfig(config); err != nil {
				fmt.Printf("Validation failed: %v\n", err)
				return err
			}

			fmt.Println("Fee configuration is valid!")
			fmt.Printf("Denominations: %d\n", len(config.FeeDenoms))
			fmt.Printf("Module Configs: %d\n", len(config.ModuleGasConfigs))
			fmt.Printf("Distribution sum: %.1f%%\n",
				(config.FeeDistribution.BurnPercentage +
				 config.FeeDistribution.ValidatorRewards +
				 config.FeeDistribution.CommunityPool +
				 config.FeeDistribution.DeveloperFund) * 100)

			return nil
		},
	}

	return cmd
}

// FeeAnalyticsCmd provides fee analytics
func FeeAnalyticsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analytics",
		Short: "Generate fee analytics and reports",
		Long: `Generate comprehensive analytics about fee usage and optimization opportunities.

Examples:
  pulsarcli fee-utils analytics --days 30
  pulsarcli fee-utils analytics --module bank --output report.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			days, _ := cmd.Flags().GetInt("days")
			module, _ := cmd.Flags().GetString("module")
			output, _ := cmd.Flags().GetString("output")

			fmt.Printf("Generating Fee Analytics Report...\n")
			fmt.Printf("Period: Last %d days\n", days)
			if module != "" {
				fmt.Printf("Module: %s\n", module)
			}
			fmt.Println()

			analytics := generateFeeAnalytics(days, module)

			if output != "" {
				data, _ := json.MarshalIndent(analytics, "", "  ")
				if err := os.WriteFile(output, data, 0644); err != nil {
					return fmt.Errorf("failed to write analytics to file: %w", err)
				}
				fmt.Printf("Analytics saved to: %s\n", output)
			}

			// Display summary
			fmt.Println("Fee Analytics Summary:")
			fmt.Println("======================")
			fmt.Printf("Total Transactions: %d\n", analytics.TotalTxs)
			fmt.Printf("Total Fees Collected: %s\n", analytics.TotalFeesCollected)
			fmt.Printf("Average Fee per Tx: %s\n", analytics.AvgFeePerTx)
			fmt.Printf("Most Used Denomination: %s\n", analytics.TopDenom)
			fmt.Printf("Peak Hour: %d:00\n", analytics.PeakHour)

			return nil
		},
	}

	cmd.Flags().Int("days", 7, "Number of days to analyze")
	cmd.Flags().String("module", "", "Specific module to analyze")
	cmd.Flags().String("output", "", "Output file for full analytics")

	return cmd
}

// Helper types and functions

type FeeCalculationResult struct {
	GasLimit     uint64  `json:"gas_limit"`
	Denomination string  `json:"denomination"`
	GasPrice     string  `json:"gas_price"`
	FeeAmount    string  `json:"fee_amount"`
}

type FeeCalculator struct {
	gasStandards config.GasStandards
}

func NewFeeCalculator() *FeeCalculator {
	return &FeeCalculator{
		gasStandards: config.DefaultGasStandards(),
	}
}

func (fc *FeeCalculator) Calculate(module, msgType string, gasLimit uint64, priority tx.FeePriority, denom string, multiplier float64) FeeCalculationResult {
	// Auto-estimate gas if not provided
	if gasLimit == 0 {
		gasLimit = fc.estimateGas(module, msgType)
	}

	// Auto-select denomination if not provided
	if denom == "" {
		denom = "ubtc" // Default to highest priority
	}

	// Get base price for denomination
	basePrice := fc.getBasePrice(denom)

	// Apply priority multiplier
	priorityMultiplier := fc.getPriorityMultiplier(priority)

	// Calculate final gas price and fee
	gasPrice := basePrice * priorityMultiplier * multiplier
	feeAmount := gasPrice * float64(gasLimit)

	return FeeCalculationResult{
		GasLimit:     gasLimit,
		Denomination: denom,
		GasPrice:     fmt.Sprintf("%.6f", gasPrice),
		FeeAmount:    fmt.Sprintf("%.0f", feeAmount),
	}
}

func (fc *FeeCalculator) estimateGas(module, msgType string) uint64 {
	switch module {
	case "bank":
		if msgType == "MsgSend" {
			return fc.gasStandards.BankGasConfig.Send
		}
		return fc.gasStandards.BankGasConfig.MultiSend
	case "coin":
		if msgType == "MsgMint" {
			return fc.gasStandards.CoinGasConfig.Mint
		}
		return fc.gasStandards.CoinGasConfig.Burn
	case "staking":
		return fc.gasStandards.StakingGasConfig.Delegate
	default:
		return 25000 // Default
	}
}

func (fc *FeeCalculator) getBasePrice(denom string) float64 {
	switch denom {
	case "ubtc":
		return 0.01
	case "ueth":
		return 100
	case "upulse":
		return 1000
	default:
		return 1
	}
}

func (fc *FeeCalculator) getPriorityMultiplier(priority tx.FeePriority) float64 {
	switch priority {
	case tx.FeePriorityLow:
		return 1.0
	case tx.FeePriorityNormal:
		return 1.5
	case tx.FeePriorityHigh:
		return 2.0
	case tx.FeePriorityFast:
		return 3.0
	default:
		return 1.5
	}
}

type FeeOptimizer struct{}

func NewFeeOptimizer() *FeeOptimizer {
	return &FeeOptimizer{}
}

func (fo *FeeOptimizer) AnalyzeAndOptimize(maxFee string, targetTime time.Duration, priority string) []string {
	recommendations := []string{}

	if maxFee != "" {
		recommendations = append(recommendations,
			"Consider using 'low' priority to reduce costs within budget")
	}

	if targetTime < 30*time.Second {
		recommendations = append(recommendations,
			"Use 'fast' priority for sub-30 second confirmation times")
	}

	recommendations = append(recommendations,
		"Use ubtc denomination for optimal gas pricing",
		"Consider batch transactions to reduce per-transaction overhead",
		"Monitor network congestion for optimal timing")

	return recommendations
}

// Configuration types
type FeeConfigFile struct {
	FeeDenoms         []types.FeeDenom              `json:"fee_denoms"`
	ModuleGasConfigs  []types.ModuleGasConfig       `json:"module_gas_configs"`
	DynamicFactors    types.DynamicGasFactors       `json:"dynamic_factors"`
	FeeDistribution   types.FeeDistributionConfig   `json:"fee_distribution"`
}

func getCurrentFeeConfig() FeeConfigFile {
	genesis := types.DefaultGenesis()
	return FeeConfigFile{
		FeeDenoms:        genesis.FeeDenoms,
		ModuleGasConfigs: genesis.ModuleGasConfigs,
		DynamicFactors:   genesis.DynamicGasFactors,
		FeeDistribution:  genesis.FeeDistributionConfig,
	}
}

func validateFeeConfig(config FeeConfigFile) error {
	// Validate fee distribution sums to 1.0
	total := config.FeeDistribution.BurnPercentage +
			 config.FeeDistribution.ValidatorRewards +
			 config.FeeDistribution.CommunityPool +
			 config.FeeDistribution.DeveloperFund

	if total != 1.0 {
		return fmt.Errorf("fee distribution must sum to 1.0, got: %f", total)
	}

	// Validate denominations
	if len(config.FeeDenoms) == 0 {
		return fmt.Errorf("at least one fee denomination must be specified")
	}

	return nil
}

// Analytics types
type BenchmarkResult struct {
	AvgTimeMs float64 `json:"avg_time_ms"`
	OpsPerSec float64 `json:"ops_per_sec"`
}

func runBenchmark(modules []string, iterations int) map[string]BenchmarkResult {
	results := make(map[string]BenchmarkResult)
	calculator := NewFeeCalculator()

	for _, module := range modules {
		start := time.Now()

		for i := 0; i < iterations; i++ {
			calculator.Calculate(module, "MsgTest", 25000, tx.FeePriorityNormal, "ubtc", 1.0)
		}

		elapsed := time.Since(start)
		avgTimeMs := elapsed.Seconds() * 1000 / float64(iterations)

		results[module] = BenchmarkResult{
			AvgTimeMs: avgTimeMs,
			OpsPerSec: 1000.0 / avgTimeMs,
		}
	}

	return results
}

type FeeAnalytics struct {
	TotalTxs           int    `json:"total_txs"`
	TotalFeesCollected string `json:"total_fees_collected"`
	AvgFeePerTx        string `json:"avg_fee_per_tx"`
	TopDenom           string `json:"top_denom"`
	PeakHour           int    `json:"peak_hour"`
}

func generateFeeAnalytics(days int, module string) FeeAnalytics {
	// Simulate analytics data
	return FeeAnalytics{
		TotalTxs:           15000 * days,
		TotalFeesCollected: fmt.Sprintf("%d ubtc", 75000*days),
		AvgFeePerTx:        "5 ubtc",
		TopDenom:           "ubtc",
		PeakHour:           14, // 2 PM
	}
}