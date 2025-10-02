package distribution

import (
	"fmt"
	"math"
	"sort"
	"time"

	keepertypes "github.com/b2network/pulsar/keeper/types"
	"github.com/b2network/pulsar/modules/fee/types"
)

// WeightedDistributionStrategy distributes fees based on weighted allocations
type WeightedDistributionStrategy struct {
	engine  *DistributionEngine
	weights map[string]float64
}

// NewWeightedDistributionStrategy creates a weighted distribution strategy
func NewWeightedDistributionStrategy(engine *DistributionEngine, weights map[string]float64) *WeightedDistributionStrategy {
	return &WeightedDistributionStrategy{
		engine:  engine,
		weights: weights,
	}
}

// Calculate implements weighted distribution calculation
func (ws *WeightedDistributionStrategy) Calculate(
	ctx keepertypes.Context,
	fees keepertypes.Coins,
	config types.FeeDistributionConfig,
) (DistributionPlan, error) {
	plan := DistributionPlan{
		TotalFees:   fees,
		CustomPools: make(map[string]keepertypes.Coins),
		Metadata:    make(map[string]interface{}),
	}

	// Apply weights to the base distribution
	for _, fee := range fees {
		burnWeight := ws.getWeight("burn", config.BurnPercentage)
		validatorWeight := ws.getWeight("validators", config.ValidatorRewards)
		communityWeight := ws.getWeight("community", config.CommunityPool)
		developerWeight := ws.getWeight("developer", config.DeveloperFund)

		totalWeight := burnWeight + validatorWeight + communityWeight + developerWeight

		// Normalize weights
		if totalWeight > 0 {
			burnAmount := int64(float64(fee.Amount) * burnWeight / totalWeight)
			validatorAmount := int64(float64(fee.Amount) * validatorWeight / totalWeight)
			communityAmount := int64(float64(fee.Amount) * communityWeight / totalWeight)
			developerAmount := int64(float64(fee.Amount) * developerWeight / totalWeight)

			if burnAmount > 0 {
				plan.BurnAmount = append(plan.BurnAmount, keepertypes.Coin{
					Denom: fee.Denom, Amount: burnAmount,
				})
			}
			if validatorAmount > 0 {
				plan.ValidatorReward = append(plan.ValidatorReward, keepertypes.Coin{
					Denom: fee.Denom, Amount: validatorAmount,
				})
			}
			if communityAmount > 0 {
				plan.CommunityPool = append(plan.CommunityPool, keepertypes.Coin{
					Denom: fee.Denom, Amount: communityAmount,
				})
			}
			if developerAmount > 0 {
				plan.DeveloperFund = append(plan.DeveloperFund, keepertypes.Coin{
					Denom: fee.Denom, Amount: developerAmount,
				})
			}
		}
	}

	plan.Metadata["strategy"] = "weighted"
	plan.Metadata["weights"] = ws.weights
	return plan, nil
}

// getWeight returns the weight for a given recipient type
func (ws *WeightedDistributionStrategy) getWeight(recipient string, baseAllocation float64) float64 {
	if weight, exists := ws.weights[recipient]; exists {
		return weight * baseAllocation
	}
	return baseAllocation
}

// Validate validates the weighted distribution plan
func (ws *WeightedDistributionStrategy) Validate(plan DistributionPlan) error {
	// Use default validation
	defaultStrategy := NewDefaultDistributionStrategy(ws.engine)
	return defaultStrategy.Validate(plan)
}

// Execute executes the weighted distribution plan
func (ws *WeightedDistributionStrategy) Execute(
	ctx keepertypes.Context,
	plan DistributionPlan,
) (DistributionResult, error) {
	// Use default execution
	defaultStrategy := NewDefaultDistributionStrategy(ws.engine)
	return defaultStrategy.Execute(ctx, plan)
}

// ThresholdDistributionStrategy distributes fees only when thresholds are met
type ThresholdDistributionStrategy struct {
	engine     *DistributionEngine
	thresholds map[string]int64
	accumulated map[string]keepertypes.Coins
}

// NewThresholdDistributionStrategy creates a threshold-based distribution strategy
func NewThresholdDistributionStrategy(engine *DistributionEngine, thresholds map[string]int64) *ThresholdDistributionStrategy {
	return &ThresholdDistributionStrategy{
		engine:      engine,
		thresholds:  thresholds,
		accumulated: make(map[string]keepertypes.Coins),
	}
}

// Calculate implements threshold-based distribution
func (ts *ThresholdDistributionStrategy) Calculate(
	ctx keepertypes.Context,
	fees keepertypes.Coins,
	config types.FeeDistributionConfig,
) (DistributionPlan, error) {
	plan := DistributionPlan{
		TotalFees:   fees,
		CustomPools: make(map[string]keepertypes.Coins),
		Metadata:    make(map[string]interface{}),
	}

	// Accumulate fees
	for _, fee := range fees {
		ts.addToAccumulated("total", []keepertypes.Coin{fee})
	}

	// Check if thresholds are met
	for denomThreshold, threshold := range ts.thresholds {
		accumulated := ts.getAccumulatedForDenom(denomThreshold)

		for _, coin := range accumulated {
			if coin.Amount >= threshold {
				// Threshold met, proceed with distribution
				return ts.distributeAccumulated(ctx, config)
			}
		}
	}

	// Thresholds not met, return empty plan
	plan.Metadata["strategy"] = "threshold"
	plan.Metadata["status"] = "threshold_not_met"
	plan.Metadata["accumulated"] = ts.accumulated
	return plan, nil
}

// distributeAccumulated distributes all accumulated fees
func (ts *ThresholdDistributionStrategy) distributeAccumulated(
	ctx keepertypes.Context,
	config types.FeeDistributionConfig,
) (DistributionPlan, error) {
	totalAccumulated := ts.getTotalAccumulated()

	// Use default strategy to calculate distribution
	defaultStrategy := NewDefaultDistributionStrategy(ts.engine)
	plan, err := defaultStrategy.Calculate(ctx, totalAccumulated, config)
	if err != nil {
		return plan, err
	}

	// Clear accumulated amounts after distribution
	ts.accumulated = make(map[string]keepertypes.Coins)

	plan.Metadata["strategy"] = "threshold"
	plan.Metadata["status"] = "threshold_met"
	return plan, nil
}

// addToAccumulated adds coins to accumulated amounts
func (ts *ThresholdDistributionStrategy) addToAccumulated(key string, coins []keepertypes.Coin) {
	if _, exists := ts.accumulated[key]; !exists {
		ts.accumulated[key] = make(keepertypes.Coins, 0)
	}

	for _, coin := range coins {
		found := false
		for i, accCoin := range ts.accumulated[key] {
			if accCoin.Denom == coin.Denom {
				ts.accumulated[key][i].Amount += coin.Amount
				found = true
				break
			}
		}
		if !found {
			ts.accumulated[key] = append(ts.accumulated[key], coin)
		}
	}
}

// getAccumulatedForDenom returns accumulated coins for a specific denomination
func (ts *ThresholdDistributionStrategy) getAccumulatedForDenom(denom string) keepertypes.Coins {
	if coins, exists := ts.accumulated["total"]; exists {
		var result keepertypes.Coins
		for _, coin := range coins {
			if coin.Denom == denom {
				result = append(result, coin)
			}
		}
		return result
	}
	return keepertypes.Coins{}
}

// getTotalAccumulated returns all accumulated coins
func (ts *ThresholdDistributionStrategy) getTotalAccumulated() keepertypes.Coins {
	if coins, exists := ts.accumulated["total"]; exists {
		return coins
	}
	return keepertypes.Coins{}
}

// Validate validates the threshold distribution plan
func (ts *ThresholdDistributionStrategy) Validate(plan DistributionPlan) error {
	// Use default validation
	defaultStrategy := NewDefaultDistributionStrategy(ts.engine)
	return defaultStrategy.Validate(plan)
}

// Execute executes the threshold distribution plan
func (ts *ThresholdDistributionStrategy) Execute(
	ctx keepertypes.Context,
	plan DistributionPlan,
) (DistributionResult, error) {
	// Use default execution
	defaultStrategy := NewDefaultDistributionStrategy(ts.engine)
	return defaultStrategy.Execute(ctx, plan)
}

// TimeBasedDistributionStrategy distributes fees at specific intervals
type TimeBasedDistributionStrategy struct {
	engine       *DistributionEngine
	interval     time.Duration
	lastDistribution int64
	accumulated  keepertypes.Coins
}

// NewTimeBasedDistributionStrategy creates a time-based distribution strategy
func NewTimeBasedDistributionStrategy(engine *DistributionEngine, interval time.Duration) *TimeBasedDistributionStrategy {
	return &TimeBasedDistributionStrategy{
		engine:   engine,
		interval: interval,
		lastDistribution: 0,
		accumulated: make(keepertypes.Coins, 0),
	}
}

// Calculate implements time-based distribution
func (tbs *TimeBasedDistributionStrategy) Calculate(
	ctx keepertypes.Context,
	fees keepertypes.Coins,
	config types.FeeDistributionConfig,
) (DistributionPlan, error) {
	// Add fees to accumulation
	for _, fee := range fees {
		found := false
		for i, accCoin := range tbs.accumulated {
			if accCoin.Denom == fee.Denom {
				tbs.accumulated[i].Amount += fee.Amount
				found = true
				break
			}
		}
		if !found {
			tbs.accumulated = append(tbs.accumulated, fee)
		}
	}

	currentTime := time.Now().Unix()

	// Check if enough time has passed
	if currentTime-tbs.lastDistribution < int64(tbs.interval.Seconds()) {
		// Not time yet, return empty plan
		plan := DistributionPlan{
			TotalFees: fees,
			Metadata: map[string]interface{}{
				"strategy": "time_based",
				"status": "waiting",
				"next_distribution": tbs.lastDistribution + int64(tbs.interval.Seconds()),
				"accumulated": tbs.accumulated,
			},
		}
		return plan, nil
	}

	// Time to distribute
	defaultStrategy := NewDefaultDistributionStrategy(tbs.engine)
	plan, err := defaultStrategy.Calculate(ctx, tbs.accumulated, config)
	if err != nil {
		return plan, err
	}

	// Update last distribution time and clear accumulated
	tbs.lastDistribution = currentTime
	tbs.accumulated = make(keepertypes.Coins, 0)

	plan.Metadata["strategy"] = "time_based"
	plan.Metadata["status"] = "distributing"
	plan.Metadata["distribution_time"] = currentTime
	return plan, nil
}

// Validate validates the time-based distribution plan
func (tbs *TimeBasedDistributionStrategy) Validate(plan DistributionPlan) error {
	defaultStrategy := NewDefaultDistributionStrategy(tbs.engine)
	return defaultStrategy.Validate(plan)
}

// Execute executes the time-based distribution plan
func (tbs *TimeBasedDistributionStrategy) Execute(
	ctx keepertypes.Context,
	plan DistributionPlan,
) (DistributionResult, error) {
	defaultStrategy := NewDefaultDistributionStrategy(tbs.engine)
	return defaultStrategy.Execute(ctx, plan)
}

// ProportionalDistributionStrategy distributes fees proportionally to validator performance
type ProportionalDistributionStrategy struct {
	engine *DistributionEngine
	performanceMetrics map[string]float64
}

// NewProportionalDistributionStrategy creates a performance-based distribution strategy
func NewProportionalDistributionStrategy(
	engine *DistributionEngine,
	performanceMetrics map[string]float64,
) *ProportionalDistributionStrategy {
	return &ProportionalDistributionStrategy{
		engine: engine,
		performanceMetrics: performanceMetrics,
	}
}

// Calculate implements performance-based distribution
func (ps *ProportionalDistributionStrategy) Calculate(
	ctx keepertypes.Context,
	fees keepertypes.Coins,
	config types.FeeDistributionConfig,
) (DistributionPlan, error) {
	// Start with default calculation
	defaultStrategy := NewDefaultDistributionStrategy(ps.engine)
	plan, err := defaultStrategy.Calculate(ctx, fees, config)
	if err != nil {
		return plan, err
	}

	// Adjust validator distributions based on performance
	validators, err := ps.engine.GetValidators(ctx)
	if err != nil {
		return plan, err
	}

	adjustedDistributions := ps.adjustValidatorRewards(validators, plan.ValidatorReward)
	plan.Recipients = adjustedDistributions

	plan.Metadata["strategy"] = "proportional"
	plan.Metadata["performance_metrics"] = ps.performanceMetrics
	return plan, nil
}

// adjustValidatorRewards adjusts validator rewards based on performance metrics
func (ps *ProportionalDistributionStrategy) adjustValidatorRewards(
	validators []ValidatorInfo,
	totalReward keepertypes.Coins,
) []PlannedDistribution {
	var distributions []PlannedDistribution

	// Calculate performance-adjusted weights
	totalWeight := 0.0
	validatorWeights := make(map[string]float64)

	for _, val := range validators {
		if val.Jailed || val.Status != 3 {
			continue
		}

		valAddr := fmt.Sprintf("%x", val.OperatorAddress)
		performance := ps.getPerformanceScore(valAddr)
		votingPower := float64(val.Tokens)

		// Combine voting power with performance
		weight := votingPower * performance
		validatorWeights[valAddr] = weight
		totalWeight += weight
	}

	if totalWeight == 0 {
		return distributions
	}

	// Distribute based on adjusted weights
	for _, val := range validators {
		if val.Jailed || val.Status != 3 {
			continue
		}

		valAddr := fmt.Sprintf("%x", val.OperatorAddress)
		weight := validatorWeights[valAddr]

		if weight > 0 {
			var valReward keepertypes.Coins
			for _, coin := range totalReward {
				amount := int64(float64(coin.Amount) * weight / totalWeight)
				if amount > 0 {
					valReward = append(valReward, keepertypes.Coin{
						Denom:  coin.Denom,
						Amount: amount,
					})
				}
			}

			if len(valReward) > 0 {
				distributions = append(distributions, PlannedDistribution{
					Recipient: val.OperatorAddress,
					Amount:    valReward,
					Type:      "validator_reward_performance",
					Priority:  1,
					Reason:    fmt.Sprintf("Performance-adjusted reward (score: %.2f, power: %d)",
						ps.getPerformanceScore(valAddr), val.Tokens),
				})
			}
		}
	}

	return distributions
}

// getPerformanceScore returns the performance score for a validator
func (ps *ProportionalDistributionStrategy) getPerformanceScore(validatorAddr string) float64 {
	if score, exists := ps.performanceMetrics[validatorAddr]; exists {
		// Ensure score is between 0.1 and 2.0 to prevent extreme penalties/rewards
		return math.Max(0.1, math.Min(2.0, score))
	}
	return 1.0 // Default neutral performance
}

// Validate validates the proportional distribution plan
func (ps *ProportionalDistributionStrategy) Validate(plan DistributionPlan) error {
	defaultStrategy := NewDefaultDistributionStrategy(ps.engine)
	return defaultStrategy.Validate(plan)
}

// Execute executes the proportional distribution plan
func (ps *ProportionalDistributionStrategy) Execute(
	ctx keepertypes.Context,
	plan DistributionPlan,
) (DistributionResult, error) {
	defaultStrategy := NewDefaultDistributionStrategy(ps.engine)
	return defaultStrategy.Execute(ctx, plan)
}

// CreateStrategyChain creates a chain of strategies that execute in order
func CreateStrategyChain(strategies ...DistributionStrategy) *StrategyChain {
	return &StrategyChain{
		strategies: strategies,
	}
}

// StrategyChain executes multiple strategies in sequence
type StrategyChain struct {
	strategies []DistributionStrategy
}

// Calculate executes all strategies and combines their results
func (sc *StrategyChain) Calculate(
	ctx keepertypes.Context,
	fees keepertypes.Coins,
	config types.FeeDistributionConfig,
) (DistributionPlan, error) {
	if len(sc.strategies) == 0 {
		return DistributionPlan{}, fmt.Errorf("no strategies in chain")
	}

	// Execute first strategy
	plan, err := sc.strategies[0].Calculate(ctx, fees, config)
	if err != nil {
		return plan, err
	}

	// Execute remaining strategies and combine results
	for i := 1; i < len(sc.strategies); i++ {
		nextPlan, err := sc.strategies[i].Calculate(ctx, fees, config)
		if err != nil {
			return plan, err
		}

		// Combine plans (this is simplified - in practice you'd need more sophisticated merging)
		plan = sc.mergePlans(plan, nextPlan)
	}

	plan.Metadata["strategy"] = "chain"
	plan.Metadata["chain_length"] = len(sc.strategies)
	return plan, nil
}

// mergePlans combines two distribution plans
func (sc *StrategyChain) mergePlans(plan1, plan2 DistributionPlan) DistributionPlan {
	// This is a simplified merge - in practice you'd need more sophisticated logic
	merged := plan1

	// Combine recipients
	merged.Recipients = append(merged.Recipients, plan2.Recipients...)

	// Sort recipients by priority
	sort.Slice(merged.Recipients, func(i, j int) bool {
		return merged.Recipients[i].Priority < merged.Recipients[j].Priority
	})

	return merged
}

// Validate validates the strategy chain
func (sc *StrategyChain) Validate(plan DistributionPlan) error {
	for _, strategy := range sc.strategies {
		if err := strategy.Validate(plan); err != nil {
			return err
		}
	}
	return nil
}

// Execute executes the strategy chain
func (sc *StrategyChain) Execute(
	ctx keepertypes.Context,
	plan DistributionPlan,
) (DistributionResult, error) {
	// Use the last strategy for execution (or implement custom execution logic)
	if len(sc.strategies) > 0 {
		return sc.strategies[len(sc.strategies)-1].Execute(ctx, plan)
	}
	return DistributionResult{}, fmt.Errorf("no strategies to execute")
}