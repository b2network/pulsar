package distribution

import (
	"fmt"
	"time"

	keepertypes "github.com/b2network/pulsar/keeper/types"
	"github.com/b2network/pulsar/modules/fee/types"
)

// DefaultDistributionStrategy implements the default fee distribution strategy
type DefaultDistributionStrategy struct {
	engine *DistributionEngine
}

// NewDefaultDistributionStrategy creates a new default distribution strategy
func NewDefaultDistributionStrategy(engine *DistributionEngine) *DefaultDistributionStrategy {
	return &DefaultDistributionStrategy{
		engine: engine,
	}
}

// Calculate creates a distribution plan based on the configuration
func (ds *DefaultDistributionStrategy) Calculate(
	ctx keepertypes.Context,
	fees keepertypes.Coins,
	config types.FeeDistributionConfig,
) (DistributionPlan, error) {
	if len(fees) == 0 {
		return DistributionPlan{}, fmt.Errorf("no fees to distribute")
	}

	plan := ds.engine.CalculateDistributionAmounts(fees)

	// Add metadata
	plan.Metadata["strategy"] = "default"
	plan.Metadata["calculated_at"] = time.Now().Unix()
	plan.Metadata["config_version"] = fmt.Sprintf("%.2f-%.2f-%.2f-%.2f",
		config.BurnPercentage, config.ValidatorRewards,
		config.CommunityPool, config.DeveloperFund)

	// Calculate validator-specific distributions
	validators, err := ds.engine.GetValidators(ctx)
	if err != nil {
		return plan, fmt.Errorf("failed to get validators: %w", err)
	}

	if len(validators) > 0 && len(plan.ValidatorReward) > 0 {
		validatorDistributions := ds.calculateValidatorDistributions(validators, plan.ValidatorReward)
		plan.Recipients = append(plan.Recipients, validatorDistributions...)
	}

	return plan, nil
}

// Validate checks if the distribution plan is valid
func (ds *DefaultDistributionStrategy) Validate(plan DistributionPlan) error {
	// Check that all amounts are non-negative
	for _, coin := range plan.BurnAmount {
		if coin.Amount < 0 {
			return fmt.Errorf("burn amount cannot be negative: %s", coin.Denom)
		}
	}

	for _, coin := range plan.ValidatorReward {
		if coin.Amount < 0 {
			return fmt.Errorf("validator reward amount cannot be negative: %s", coin.Denom)
		}
	}

	for _, coin := range plan.CommunityPool {
		if coin.Amount < 0 {
			return fmt.Errorf("community pool amount cannot be negative: %s", coin.Denom)
		}
	}

	for _, coin := range plan.DeveloperFund {
		if coin.Amount < 0 {
			return fmt.Errorf("developer fund amount cannot be negative: %s", coin.Denom)
		}
	}

	// Verify total amounts match
	totalCalculated := make(map[string]int64)

	for _, coin := range plan.BurnAmount {
		totalCalculated[coin.Denom] += coin.Amount
	}
	for _, coin := range plan.ValidatorReward {
		totalCalculated[coin.Denom] += coin.Amount
	}
	for _, coin := range plan.CommunityPool {
		totalCalculated[coin.Denom] += coin.Amount
	}
	for _, coin := range plan.DeveloperFund {
		totalCalculated[coin.Denom] += coin.Amount
	}

	for _, totalCoin := range plan.TotalFees {
		calculated, exists := totalCalculated[totalCoin.Denom]
		if !exists {
			return fmt.Errorf("missing distribution for denomination: %s", totalCoin.Denom)
		}

		// Allow small rounding differences (up to 1 unit)
		diff := totalCoin.Amount - calculated
		if diff < 0 {
			diff = -diff
		}
		if diff > 1 {
			return fmt.Errorf("distribution amounts don't match total for %s: total=%d, distributed=%d",
				totalCoin.Denom, totalCoin.Amount, calculated)
		}
	}

	return nil
}

// Execute executes the distribution plan
func (ds *DefaultDistributionStrategy) Execute(
	ctx keepertypes.Context,
	plan DistributionPlan,
) (DistributionResult, error) {
	result := DistributionResult{
		Distributions: make(map[string]keepertypes.Coins),
		Timestamp:     time.Now().Unix(),
	}

	// Execute burn
	if len(plan.BurnAmount) > 0 {
		if err := ds.executeBurn(ctx, plan.BurnAmount); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("burn failed: %v", err))
		} else {
			result.BurnedAmount = plan.BurnAmount
			result.Distributions["burn"] = plan.BurnAmount
		}
	}

	// Execute validator rewards distribution
	if len(plan.ValidatorReward) > 0 {
		if err := ds.executeValidatorDistribution(ctx, plan); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("validator distribution failed: %v", err))
		} else {
			result.Distributions["validators"] = plan.ValidatorReward
		}
	}

	// Execute community pool distribution
	if len(plan.CommunityPool) > 0 {
		if err := ds.executeCommunityPoolDistribution(ctx, plan.CommunityPool); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("community pool distribution failed: %v", err))
		} else {
			result.Distributions["community_pool"] = plan.CommunityPool
		}
	}

	// Execute developer fund distribution
	if len(plan.DeveloperFund) > 0 {
		if err := ds.executeDeveloperFundDistribution(ctx, plan.DeveloperFund); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("developer fund distribution failed: %v", err))
		} else {
			result.Distributions["developer_fund"] = plan.DeveloperFund
		}
	}

	// Calculate total distributed
	for _, coins := range result.Distributions {
		for _, coin := range coins {
			found := false
			for i, totalCoin := range result.TotalDistributed {
				if totalCoin.Denom == coin.Denom {
					result.TotalDistributed[i].Amount += coin.Amount
					found = true
					break
				}
			}
			if !found {
				result.TotalDistributed = append(result.TotalDistributed, coin)
			}
		}
	}

	return result, nil
}

// calculateValidatorDistributions calculates individual validator distributions
func (ds *DefaultDistributionStrategy) calculateValidatorDistributions(
	validators []ValidatorInfo,
	totalReward keepertypes.Coins,
) []PlannedDistribution {
	var distributions []PlannedDistribution

	if len(validators) == 0 {
		return distributions
	}

	// Calculate total voting power
	totalPower := int64(0)
	for _, val := range validators {
		if !val.Jailed && val.Status == 3 { // BondStatusBonded
			totalPower += val.Tokens
		}
	}

	if totalPower == 0 {
		return distributions
	}

	// Distribute proportionally based on voting power
	for _, val := range validators {
		if val.Jailed || val.Status != 3 {
			continue
		}

		var valReward keepertypes.Coins
		for _, coin := range totalReward {
			amount := (coin.Amount * val.Tokens) / totalPower
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
				Type:      "validator_reward",
				Priority:  1,
				Reason:    fmt.Sprintf("Proportional reward based on voting power: %d", val.Tokens),
			})
		}
	}

	return distributions
}

// executeBurn burns the specified amount
func (ds *DefaultDistributionStrategy) executeBurn(
	ctx keepertypes.Context,
	burnAmount keepertypes.Coins,
) error {
	return ds.engine.bankKeeper.BurnCoins(ctx, types.FeeCollectorName, burnAmount)
}

// executeValidatorDistribution distributes rewards to validators
func (ds *DefaultDistributionStrategy) executeValidatorDistribution(
	ctx keepertypes.Context,
	plan DistributionPlan,
) error {
	// Send total validator rewards to staking rewards module
	err := ds.engine.bankKeeper.SendCoinsFromModuleToModule(
		ctx,
		types.FeeCollectorName,
		types.StakingRewardsName,
		plan.ValidatorReward,
	)
	if err != nil {
		return fmt.Errorf("failed to send rewards to staking module: %w", err)
	}

	// Distribute to individual validators
	for _, distribution := range plan.Recipients {
		if distribution.Type == "validator_reward" {
			val, found := ds.engine.stakingKeeper.GetValidator(ctx, distribution.Recipient)
			if found && !val.Jailed {
				err := ds.engine.distributionKeeper.AllocateTokensToValidator(ctx, val, distribution.Amount)
				if err != nil {
					// Log error but don't fail entire distribution
					ctx.Logger().Error("Failed to allocate tokens to validator",
						"validator", fmt.Sprintf("%x", distribution.Recipient),
						"error", err)
				}
			}
		}
	}

	return nil
}

// executeCommunityPoolDistribution sends funds to community pool
func (ds *DefaultDistributionStrategy) executeCommunityPoolDistribution(
	ctx keepertypes.Context,
	amount keepertypes.Coins,
) error {
	return ds.engine.bankKeeper.SendCoinsFromModuleToModule(
		ctx,
		types.FeeCollectorName,
		types.CommunityPoolName,
		amount,
	)
}

// executeDeveloperFundDistribution sends funds to developer fund
func (ds *DefaultDistributionStrategy) executeDeveloperFundDistribution(
	ctx keepertypes.Context,
	amount keepertypes.Coins,
) error {
	return ds.engine.bankKeeper.SendCoinsFromModuleToModule(
		ctx,
		types.FeeCollectorName,
		types.DeveloperFundName,
		amount,
	)
}

// DistributeFees is the main entry point for fee distribution
func (de *DistributionEngine) DistributeFees(
	ctx keepertypes.Context,
	fees keepertypes.Coins,
	strategy DistributionStrategy,
) (DistributionResult, error) {
	if len(fees) == 0 {
		return DistributionResult{}, fmt.Errorf("no fees to distribute")
	}

	// Validate configuration
	if err := de.ValidateConfig(); err != nil {
		return DistributionResult{}, fmt.Errorf("invalid distribution config: %w", err)
	}

	// Calculate distribution plan
	plan, err := strategy.Calculate(ctx, fees, de.config)
	if err != nil {
		return DistributionResult{}, fmt.Errorf("failed to calculate distribution: %w", err)
	}

	// Validate plan
	if err := strategy.Validate(plan); err != nil {
		return DistributionResult{}, fmt.Errorf("invalid distribution plan: %w", err)
	}

	// Execute distribution
	result, err := strategy.Execute(ctx, plan)
	if err != nil {
		return result, fmt.Errorf("failed to execute distribution: %w", err)
	}

	return result, nil
}

// GetDistributionHistory returns historical distribution data
func (de *DistributionEngine) GetDistributionHistory(
	ctx keepertypes.Context,
	limit int,
) ([]DistributionResult, error) {
	// This would typically read from stored distribution history
	// For now, return empty slice as this requires persistent storage
	return []DistributionResult{}, nil
}

// GetDistributionStats returns distribution statistics
func (de *DistributionEngine) GetDistributionStats(
	ctx keepertypes.Context,
) (DistributionStats, error) {
	stats := DistributionStats{
		TotalDistributed: make(map[string]int64),
		RecipientStats:   make(map[string]RecipientStats),
	}

	// This would calculate stats from historical data
	// For now, return empty stats
	return stats, nil
}

// DistributionStats represents distribution statistics
type DistributionStats struct {
	TotalDistributed map[string]int64            `json:"total_distributed"`
	TotalBurned      map[string]int64            `json:"total_burned"`
	RecipientStats   map[string]RecipientStats   `json:"recipient_stats"`
	LastDistribution int64                       `json:"last_distribution"`
	DistributionCount int64                      `json:"distribution_count"`
}

// RecipientStats represents statistics for a specific recipient
type RecipientStats struct {
	TotalReceived map[string]int64 `json:"total_received"`
	LastReward    int64            `json:"last_reward"`
	RewardCount   int64            `json:"reward_count"`
}