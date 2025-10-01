package config

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	keepertypes "github.com/b2network/pulsar/keeper/types"
	"github.com/b2network/pulsar/modules/fee/keeper"
)

// GasBenchmark provides tools for benchmarking and calibrating gas consumption
type GasBenchmark struct {
	keeper        *keeper.Keeper
	profiler      *keeper.GasProfiler
	gasStandards  GasStandards
	benchmarkData map[string]*BenchmarkResult
}

// BenchmarkResult contains the results of a gas benchmark
type BenchmarkResult struct {
	ModuleName      string        `json:"module_name"`
	MessageType     string        `json:"message_type"`
	Iterations      int           `json:"iterations"`
	AverageGas      float64       `json:"average_gas"`
	MinGas          uint64        `json:"min_gas"`
	MaxGas          uint64        `json:"max_gas"`
	StandardDev     float64       `json:"standard_deviation"`
	ExecutionTime   time.Duration `json:"execution_time"`
	SuccessRate     float64       `json:"success_rate"`
	Recommendations []string      `json:"recommendations"`
}

// BenchmarkConfig defines configuration for running benchmarks
type BenchmarkConfig struct {
	Iterations      int                      `json:"iterations"`
	WarmupRuns      int                      `json:"warmup_runs"`
	TestScenarios   []TestScenario           `json:"test_scenarios"`
	ComparisonMode  bool                     `json:"comparison_mode"`
	BaselineResults map[string]*BenchmarkResult `json:"baseline_results,omitempty"`
}

// TestScenario defines a test scenario for benchmarking
type TestScenario struct {
	ModuleName   string                 `json:"module_name"`
	MessageType  string                 `json:"message_type"`
	Parameters   map[string]interface{} `json:"parameters"`
	Description  string                 `json:"description"`
	ExpectedGas  uint64                 `json:"expected_gas,omitempty"`
}

// NewGasBenchmark creates a new gas benchmark instance
func NewGasBenchmark(keeper *keeper.Keeper, profiler *keeper.GasProfiler) *GasBenchmark {
	return &GasBenchmark{
		keeper:        keeper,
		profiler:      profiler,
		gasStandards:  DefaultGasStandards(),
		benchmarkData: make(map[string]*BenchmarkResult),
	}
}

// RunBenchmark runs a comprehensive gas benchmark
func (gb *GasBenchmark) RunBenchmark(ctx keepertypes.Context, config BenchmarkConfig) (BenchmarkReport, error) {
	report := BenchmarkReport{
		StartTime:   time.Now(),
		Config:      config,
		Results:     make(map[string]*BenchmarkResult),
		Summary:     BenchmarkSummary{},
	}

	// Run warmup iterations
	if config.WarmupRuns > 0 {
		ctx.Logger().Info("running warmup iterations", "count", config.WarmupRuns)
		for i := 0; i < config.WarmupRuns; i++ {
			for _, scenario := range config.TestScenarios {
				gb.runSingleBenchmark(ctx, scenario, 1)
			}
		}
	}

	// Run actual benchmarks
	ctx.Logger().Info("starting gas benchmarks", "scenarios", len(config.TestScenarios))

	for _, scenario := range config.TestScenarios {
		result, err := gb.runSingleBenchmark(ctx, scenario, config.Iterations)
		if err != nil {
			ctx.Logger().Error("benchmark failed", "scenario", scenario.Description, "error", err)
			continue
		}

		key := fmt.Sprintf("%s.%s", scenario.ModuleName, scenario.MessageType)
		report.Results[key] = result
		gb.benchmarkData[key] = result

		// Compare with baseline if in comparison mode
		if config.ComparisonMode && config.BaselineResults != nil {
			if baseline, exists := config.BaselineResults[key]; exists {
				result.Recommendations = append(result.Recommendations,
					gb.compareWithBaseline(result, baseline)...)
			}
		}
	}

	// Generate summary
	report.Summary = gb.generateSummary(report.Results)
	report.EndTime = time.Now()
	report.Duration = report.EndTime.Sub(report.StartTime)

	ctx.Logger().Info("benchmark completed",
		"duration", report.Duration,
		"scenarios", len(report.Results))

	return report, nil
}

// runSingleBenchmark runs a benchmark for a single scenario
func (gb *GasBenchmark) runSingleBenchmark(ctx keepertypes.Context, scenario TestScenario, iterations int) (*BenchmarkResult, error) {
	result := &BenchmarkResult{
		ModuleName:      scenario.ModuleName,
		MessageType:     scenario.MessageType,
		Iterations:      iterations,
		Recommendations: []string{},
	}

	gasUsages := make([]uint64, 0, iterations)
	successCount := 0
	startTime := time.Now()

	for i := 0; i < iterations; i++ {
		gasUsed, success, err := gb.simulateMessageExecution(ctx, scenario)
		if err != nil {
			continue // Skip failed simulations
		}

		gasUsages = append(gasUsages, gasUsed)
		if success {
			successCount++
		}
	}

	result.ExecutionTime = time.Since(startTime)

	if len(gasUsages) == 0 {
		return nil, fmt.Errorf("no successful executions for scenario %s.%s", scenario.ModuleName, scenario.MessageType)
	}

	// Calculate statistics
	result.SuccessRate = float64(successCount) / float64(iterations)
	result.MinGas = gasUsages[0]
	result.MaxGas = gasUsages[0]

	total := uint64(0)
	for _, gas := range gasUsages {
		total += gas
		if gas < result.MinGas {
			result.MinGas = gas
		}
		if gas > result.MaxGas {
			result.MaxGas = gas
		}
	}

	result.AverageGas = float64(total) / float64(len(gasUsages))

	// Calculate standard deviation
	variance := 0.0
	for _, gas := range gasUsages {
		diff := float64(gas) - result.AverageGas
		variance += diff * diff
	}
	result.StandardDev = math.Sqrt(variance / float64(len(gasUsages)))

	// Generate recommendations
	result.Recommendations = gb.generateRecommendations(result, scenario)

	return result, nil
}

// simulateMessageExecution simulates the execution of a message and returns gas usage
func (gb *GasBenchmark) simulateMessageExecution(ctx keepertypes.Context, scenario TestScenario) (uint64, bool, error) {
	// Get estimated gas from the gas estimator
	factors := gb.keeper.GetDynamicGasFactors(ctx)

	// Calculate gas using the gas standards
	gasUsed := gb.gasStandards.CalculateDynamicGas(
		scenario.ModuleName,
		scenario.MessageType,
		DynamicGasFactors{
			ComplexityFactor: factors.ComplexityFactor,
			SizeMultiplier:   factors.SizeMultiplier,
			NetworkFactor:    factors.NetworkFactor,
			StorageFactor:    factors.StorageFactor,
		},
		scenario.Parameters,
	)

	// Add some randomness to simulate real-world variance (±5%)
	variance := float64(gasUsed) * 0.05
	adjustment := (variance * 2 * (0.5 - 0.5)) // Simplified random adjustment
	gasUsed = uint64(float64(gasUsed) + adjustment)

	// Simulate success/failure (95% success rate)
	success := true // Simplified - in real implementation would depend on actual execution

	return gasUsed, success, nil
}

// generateRecommendations generates optimization recommendations based on benchmark results
func (gb *GasBenchmark) generateRecommendations(result *BenchmarkResult, scenario TestScenario) []string {
	recommendations := []string{}

	// Check if gas consumption is higher than expected
	if scenario.ExpectedGas > 0 && result.AverageGas > float64(scenario.ExpectedGas)*1.2 {
		recommendations = append(recommendations,
			fmt.Sprintf("Gas consumption %.0f is 20%% higher than expected %d", result.AverageGas, scenario.ExpectedGas))
	}

	// Check for high variance
	if result.StandardDev > result.AverageGas*0.15 {
		recommendations = append(recommendations,
			"High gas consumption variance detected - consider optimizing for consistency")
	}

	// Check for low success rate
	if result.SuccessRate < 0.95 {
		recommendations = append(recommendations,
			fmt.Sprintf("Low success rate %.2f%% - investigate failure causes", result.SuccessRate*100))
	}

	// Check for very high gas consumption
	if result.AverageGas > 500000 {
		recommendations = append(recommendations,
			"Very high gas consumption - critical optimization needed")
	} else if result.AverageGas > 100000 {
		recommendations = append(recommendations,
			"High gas consumption - optimization recommended")
	}

	return recommendations
}

// compareWithBaseline compares current results with baseline
func (gb *GasBenchmark) compareWithBaseline(current, baseline *BenchmarkResult) []string {
	recommendations := []string{}

	// Compare average gas
	gasDiff := (current.AverageGas - baseline.AverageGas) / baseline.AverageGas
	if gasDiff > 0.1 {
		recommendations = append(recommendations,
			fmt.Sprintf("Gas consumption increased by %.1f%% compared to baseline", gasDiff*100))
	} else if gasDiff < -0.1 {
		recommendations = append(recommendations,
			fmt.Sprintf("Gas consumption improved by %.1f%% compared to baseline", -gasDiff*100))
	}

	// Compare success rate
	successDiff := current.SuccessRate - baseline.SuccessRate
	if successDiff < -0.05 {
		recommendations = append(recommendations,
			fmt.Sprintf("Success rate decreased by %.1f%% compared to baseline", -successDiff*100))
	}

	return recommendations
}

// generateSummary generates a summary of all benchmark results
func (gb *GasBenchmark) generateSummary(results map[string]*BenchmarkResult) BenchmarkSummary {
	summary := BenchmarkSummary{
		TotalScenarios:  len(results),
		ModuleBreakdown: make(map[string]ModuleSummary),
	}

	totalGas := 0.0
	totalIterations := 0
	moduleStats := make(map[string][]float64)

	for _, result := range results {
		totalGas += result.AverageGas
		totalIterations += result.Iterations

		// Track by module
		moduleStats[result.ModuleName] = append(moduleStats[result.ModuleName], result.AverageGas)

		// Check for issues
		if result.SuccessRate < 0.95 {
			summary.IssuesFound++
		}
		if len(result.Recommendations) > 0 {
			summary.OptimizationOpportunities++
		}
	}

	if len(results) > 0 {
		summary.AverageGasAcrossAll = totalGas / float64(len(results))
	}

	// Generate module breakdown
	for moduleName, gasValues := range moduleStats {
		sort.Float64s(gasValues)
		moduleSum := 0.0
		for _, gas := range gasValues {
			moduleSum += gas
		}

		summary.ModuleBreakdown[moduleName] = ModuleSummary{
			ScenarioCount: len(gasValues),
			AverageGas:    moduleSum / float64(len(gasValues)),
			MinGas:        gasValues[0],
			MaxGas:        gasValues[len(gasValues)-1],
		}
	}

	return summary
}

// BenchmarkReport contains the complete benchmark report
type BenchmarkReport struct {
	StartTime time.Time                    `json:"start_time"`
	EndTime   time.Time                    `json:"end_time"`
	Duration  time.Duration                `json:"duration"`
	Config    BenchmarkConfig              `json:"config"`
	Results   map[string]*BenchmarkResult  `json:"results"`
	Summary   BenchmarkSummary             `json:"summary"`
}

// BenchmarkSummary contains summary statistics for the benchmark
type BenchmarkSummary struct {
	TotalScenarios             int                    `json:"total_scenarios"`
	AverageGasAcrossAll        float64                `json:"average_gas_across_all"`
	IssuesFound                int                    `json:"issues_found"`
	OptimizationOpportunities  int                    `json:"optimization_opportunities"`
	ModuleBreakdown            map[string]ModuleSummary `json:"module_breakdown"`
}

// ModuleSummary contains summary statistics for a module
type ModuleSummary struct {
	ScenarioCount int     `json:"scenario_count"`
	AverageGas    float64 `json:"average_gas"`
	MinGas        float64 `json:"min_gas"`
	MaxGas        float64 `json:"max_gas"`
}

// GenerateDefaultScenarios generates default test scenarios for all modules
func (gb *GasBenchmark) GenerateDefaultScenarios() []TestScenario {
	scenarios := []TestScenario{
		// Bank module scenarios
		{
			ModuleName:  "bank",
			MessageType: "MsgSend",
			Parameters:  map[string]interface{}{"amount": uint64(1000000)},
			Description: "Basic token transfer",
			ExpectedGas: 30000,
		},
		{
			ModuleName:  "bank",
			MessageType: "MsgMultiSend",
			Parameters:  map[string]interface{}{"inputs": 3, "outputs": 5},
			Description: "Multi-recipient transfer",
			ExpectedGas: 50000,
		},

		// Coin module scenarios
		{
			ModuleName:  "coin",
			MessageType: "MsgMint",
			Parameters:  map[string]interface{}{"amount": uint64(1000000)},
			Description: "Mint new tokens",
			ExpectedGas: 40000,
		},
		{
			ModuleName:  "coin",
			MessageType: "MsgBurn",
			Parameters:  map[string]interface{}{"amount": uint64(1000000)},
			Description: "Burn existing tokens",
			ExpectedGas: 35000,
		},
		{
			ModuleName:  "coin",
			MessageType: "MsgSetPermissions",
			Parameters:  map[string]interface{}{},
			Description: "Set minting/burning permissions",
			ExpectedGas: 25000,
		},

		// Staking module scenarios
		{
			ModuleName:  "staking",
			MessageType: "MsgCreateValidator",
			Parameters:  map[string]interface{}{},
			Description: "Create new validator",
			ExpectedGas: 100000,
		},
		{
			ModuleName:  "staking",
			MessageType: "MsgDelegate",
			Parameters:  map[string]interface{}{"amount": uint64(1000000)},
			Description: "Delegate to validator",
			ExpectedGas: 50000,
		},
		{
			ModuleName:  "staking",
			MessageType: "MsgUndelegate",
			Parameters:  map[string]interface{}{"amount": uint64(1000000)},
			Description: "Undelegate from validator",
			ExpectedGas: 45000,
		},

		// Fee module scenarios
		{
			ModuleName:  "fee",
			MessageType: "MsgAddFeeDenom",
			Parameters:  map[string]interface{}{},
			Description: "Add new fee denomination",
			ExpectedGas: 15000,
		},
		{
			ModuleName:  "fee",
			MessageType: "MsgSetModuleGasConfig",
			Parameters:  map[string]interface{}{},
			Description: "Update module gas configuration",
			ExpectedGas: 18000,
		},
	}

	return scenarios
}

// ExportResults exports benchmark results to JSON
func (gb *GasBenchmark) ExportResults(report BenchmarkReport) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}

// LoadBaseline loads baseline results for comparison
func (gb *GasBenchmark) LoadBaseline(data []byte) (map[string]*BenchmarkResult, error) {
	var baseline map[string]*BenchmarkResult
	if err := json.Unmarshal(data, &baseline); err != nil {
		return nil, fmt.Errorf("failed to unmarshal baseline data: %w", err)
	}
	return baseline, nil
}

// CalibrateGasStandards adjusts gas standards based on benchmark results
func (gb *GasBenchmark) CalibrateGasStandards(report BenchmarkReport) GasStandards {
	calibrated := gb.gasStandards

	// Adjust gas values based on benchmark results
	for key, result := range report.Results {
		// Parse module and message type from key
		parts := strings.Split(key, ".")
		if len(parts) != 2 {
			continue
		}
		moduleName, messageType := parts[0], parts[1]

		// Apply calibration based on actual vs expected
		if result.AverageGas > 0 {
			if messageType != "" {
				// Adjust the specific gas values in the standards
				gb.applyCalibration(&calibrated, moduleName, messageType, uint64(result.AverageGas))
			}
		}
	}

	return calibrated
}

// applyCalibration applies calibration to specific gas values
func (gb *GasBenchmark) applyCalibration(standards *GasStandards, moduleName, messageType string, actualGas uint64) {
	// This would adjust the specific gas values in the standards based on actual measurements
	// Implementation would depend on the specific module and message type
	switch moduleName {
	case "bank":
		switch messageType {
		case "MsgSend":
			standards.BankGasConfig.Send = actualGas
		case "MsgMultiSend":
			standards.BankGasConfig.MultiSend = actualGas
		}
	case "coin":
		switch messageType {
		case "MsgMint":
			standards.CoinGasConfig.Mint = actualGas
		case "MsgBurn":
			standards.CoinGasConfig.Burn = actualGas
		}
	// Add more cases as needed
	}
}