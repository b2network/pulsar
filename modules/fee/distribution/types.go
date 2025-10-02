package distribution

import (
	"fmt"

	keepertypes "github.com/b2network/pulsar/keeper/types"
	"github.com/b2network/pulsar/modules/fee/types"
)

// DistributionEngine handles fee distribution across different recipients
type DistributionEngine struct {
	config           types.FeeDistributionConfig
	bankKeeper       BankKeeper
	stakingKeeper    StakingKeeper
	distributionKeeper DistributionKeeper
	validators       []ValidatorInfo
	pools            map[string]PoolInfo
}

// BankKeeper defines the expected bank keeper interface for distribution
type BankKeeper interface {
	SendCoinsFromModuleToModule(ctx keepertypes.Context, senderModule, recipientModule string, amt keepertypes.Coins) error
	SendCoinsFromModuleToAccount(ctx keepertypes.Context, senderModule string, recipientAddr []byte, amt keepertypes.Coins) error
	BurnCoins(ctx keepertypes.Context, moduleName string, amounts keepertypes.Coins) error
	GetAllBalances(ctx keepertypes.Context, addr []byte) keepertypes.Coins
	GetModuleAccount(ctx keepertypes.Context, moduleName string) []byte
}

// StakingKeeper defines the expected staking keeper interface
type StakingKeeper interface {
	GetAllValidators(ctx keepertypes.Context) []ValidatorInfo
	GetBondedValidatorsByPower(ctx keepertypes.Context) []ValidatorInfo
	GetValidator(ctx keepertypes.Context, addr []byte) (ValidatorInfo, bool)
	GetValidatorDelegations(ctx keepertypes.Context, valAddr []byte) []DelegationInfo
}

// DistributionKeeper defines the expected distribution keeper interface
type DistributionKeeper interface {
	AllocateTokensToValidator(ctx keepertypes.Context, val ValidatorInfo, tokens keepertypes.Coins) error
	GetCommunityPool(ctx keepertypes.Context) keepertypes.Coins
	FundCommunityPool(ctx keepertypes.Context, amount keepertypes.Coins, sender []byte) error
	GetFeePool(ctx keepertypes.Context) FeePool
	SetFeePool(ctx keepertypes.Context, feePool FeePool)
}

// ValidatorInfo represents validator information for distribution
type ValidatorInfo struct {
	OperatorAddress []byte  `json:"operator_address"`
	ConsensusPubkey []byte  `json:"consensus_pubkey"`
	Jailed          bool    `json:"jailed"`
	Status          int32   `json:"status"`
	Tokens          int64   `json:"tokens"`
	DelegatorShares int64   `json:"delegator_shares"`
	Commission      float64 `json:"commission"`
	MinSelfDelegation int64 `json:"min_self_delegation"`
}

// DelegationInfo represents delegation information
type DelegationInfo struct {
	DelegatorAddress []byte `json:"delegator_address"`
	ValidatorAddress []byte `json:"validator_address"`
	Shares           int64  `json:"shares"`
}

// FeePool represents the fee pool for distribution
type FeePool struct {
	CommunityPool keepertypes.Coins `json:"community_pool"`
	DecayPool     keepertypes.Coins `json:"decay_pool"`
}

// PoolInfo represents information about distribution pools
type PoolInfo struct {
	Name         string                `json:"name"`
	Address      []byte                `json:"address"`
	Balance      keepertypes.Coins     `json:"balance"`
	Allocation   float64               `json:"allocation"`
	Rules        DistributionRules     `json:"rules"`
	Recipients   []RecipientInfo       `json:"recipients"`
}

// DistributionRules defines rules for pool distribution
type DistributionRules struct {
	MinThreshold    int64             `json:"min_threshold"`
	MaxThreshold    int64             `json:"max_threshold"`
	DistributionCap int64             `json:"distribution_cap"`
	Frequency       DistributionFreq  `json:"frequency"`
	Conditions      []string          `json:"conditions"`
}

// DistributionFreq defines distribution frequency
type DistributionFreq string

const (
	FreqImmediate DistributionFreq = "immediate"
	FreqDaily     DistributionFreq = "daily"
	FreqWeekly    DistributionFreq = "weekly"
	FreqMonthly   DistributionFreq = "monthly"
)

// RecipientInfo represents a distribution recipient
type RecipientInfo struct {
	Address     []byte  `json:"address"`
	Weight      float64 `json:"weight"`
	Type        string  `json:"type"`
	Active      bool    `json:"active"`
	LastReward  int64   `json:"last_reward"`
}

// DistributionResult represents the result of fee distribution
type DistributionResult struct {
	TotalDistributed keepertypes.Coins            `json:"total_distributed"`
	BurnedAmount     keepertypes.Coins            `json:"burned_amount"`
	Distributions    map[string]keepertypes.Coins `json:"distributions"`
	Recipients       []RecipientResult            `json:"recipients"`
	Errors           []string                     `json:"errors"`
	Timestamp        int64                        `json:"timestamp"`
}

// RecipientResult represents distribution result for a specific recipient
type RecipientResult struct {
	Address   []byte            `json:"address"`
	Amount    keepertypes.Coins `json:"amount"`
	Type      string            `json:"type"`
	Success   bool              `json:"success"`
	Error     string            `json:"error,omitempty"`
}

// DistributionStrategy defines different distribution strategies
type DistributionStrategy interface {
	Calculate(ctx keepertypes.Context, fees keepertypes.Coins, config types.FeeDistributionConfig) (DistributionPlan, error)
	Validate(plan DistributionPlan) error
	Execute(ctx keepertypes.Context, plan DistributionPlan) (DistributionResult, error)
}

// DistributionPlan represents a plan for fee distribution
type DistributionPlan struct {
	TotalFees       keepertypes.Coins              `json:"total_fees"`
	BurnAmount      keepertypes.Coins              `json:"burn_amount"`
	ValidatorReward keepertypes.Coins              `json:"validator_reward"`
	CommunityPool   keepertypes.Coins              `json:"community_pool"`
	DeveloperFund   keepertypes.Coins              `json:"developer_fund"`
	CustomPools     map[string]keepertypes.Coins   `json:"custom_pools"`
	Recipients      []PlannedDistribution          `json:"recipients"`
	Metadata        map[string]interface{}         `json:"metadata"`
}

// PlannedDistribution represents a planned distribution to a recipient
type PlannedDistribution struct {
	Recipient []byte            `json:"recipient"`
	Amount    keepertypes.Coins `json:"amount"`
	Type      string            `json:"type"`
	Priority  int               `json:"priority"`
	Reason    string            `json:"reason"`
}

// NewDistributionEngine creates a new distribution engine
func NewDistributionEngine(
	config types.FeeDistributionConfig,
	bankKeeper BankKeeper,
	stakingKeeper StakingKeeper,
	distributionKeeper DistributionKeeper,
) *DistributionEngine {
	return &DistributionEngine{
		config:             config,
		bankKeeper:         bankKeeper,
		stakingKeeper:      stakingKeeper,
		distributionKeeper: distributionKeeper,
		pools:              make(map[string]PoolInfo),
	}
}

// ValidateConfig validates the distribution configuration
func (de *DistributionEngine) ValidateConfig() error {
	total := de.config.BurnPercentage + de.config.ValidatorRewards +
			 de.config.CommunityPool + de.config.DeveloperFund

	if total != 1.0 {
		return fmt.Errorf("distribution percentages must sum to 1.0, got %.6f", total)
	}

	if de.config.BurnPercentage < 0 || de.config.BurnPercentage > 1 {
		return fmt.Errorf("burn percentage must be between 0 and 1, got %.6f", de.config.BurnPercentage)
	}

	if de.config.ValidatorRewards < 0 || de.config.ValidatorRewards > 1 {
		return fmt.Errorf("validator rewards percentage must be between 0 and 1, got %.6f", de.config.ValidatorRewards)
	}

	if de.config.CommunityPool < 0 || de.config.CommunityPool > 1 {
		return fmt.Errorf("community pool percentage must be between 0 and 1, got %.6f", de.config.CommunityPool)
	}

	if de.config.DeveloperFund < 0 || de.config.DeveloperFund > 1 {
		return fmt.Errorf("developer fund percentage must be between 0 and 1, got %.6f", de.config.DeveloperFund)
	}

	return nil
}

// GetConfig returns the current distribution configuration
func (de *DistributionEngine) GetConfig() types.FeeDistributionConfig {
	return de.config
}

// UpdateConfig updates the distribution configuration
func (de *DistributionEngine) UpdateConfig(config types.FeeDistributionConfig) error {
	oldConfig := de.config
	de.config = config

	if err := de.ValidateConfig(); err != nil {
		// Restore old config on validation failure
		de.config = oldConfig
		return fmt.Errorf("invalid configuration: %w", err)
	}

	return nil
}

// AddPool adds a custom distribution pool
func (de *DistributionEngine) AddPool(name string, pool PoolInfo) error {
	if _, exists := de.pools[name]; exists {
		return fmt.Errorf("pool %s already exists", name)
	}

	if pool.Allocation < 0 || pool.Allocation > 1 {
		return fmt.Errorf("pool allocation must be between 0 and 1, got %.6f", pool.Allocation)
	}

	de.pools[name] = pool
	return nil
}

// RemovePool removes a custom distribution pool
func (de *DistributionEngine) RemovePool(name string) error {
	if _, exists := de.pools[name]; !exists {
		return fmt.Errorf("pool %s does not exist", name)
	}

	delete(de.pools, name)
	return nil
}

// GetPool returns information about a specific pool
func (de *DistributionEngine) GetPool(name string) (PoolInfo, bool) {
	pool, exists := de.pools[name]
	return pool, exists
}

// ListPools returns all configured pools
func (de *DistributionEngine) ListPools() map[string]PoolInfo {
	pools := make(map[string]PoolInfo)
	for name, pool := range de.pools {
		pools[name] = pool
	}
	return pools
}

// GetValidators retrieves current validator set for distribution
func (de *DistributionEngine) GetValidators(ctx keepertypes.Context) ([]ValidatorInfo, error) {
	validators := de.stakingKeeper.GetBondedValidatorsByPower(ctx)
	if len(validators) == 0 {
		return nil, fmt.Errorf("no bonded validators found")
	}

	de.validators = validators
	return validators, nil
}

// CalculateDistributionAmounts calculates how fees should be distributed
func (de *DistributionEngine) CalculateDistributionAmounts(fees keepertypes.Coins) DistributionPlan {
	plan := DistributionPlan{
		TotalFees:   fees,
		CustomPools: make(map[string]keepertypes.Coins),
		Metadata:    make(map[string]interface{}),
	}

	for _, fee := range fees {
		// Calculate burn amount
		burnAmount := int64(float64(fee.Amount) * de.config.BurnPercentage)
		if burnAmount > 0 {
			plan.BurnAmount = append(plan.BurnAmount, keepertypes.Coin{
				Denom:  fee.Denom,
				Amount: burnAmount,
			})
		}

		// Calculate validator rewards
		validatorAmount := int64(float64(fee.Amount) * de.config.ValidatorRewards)
		if validatorAmount > 0 {
			plan.ValidatorReward = append(plan.ValidatorReward, keepertypes.Coin{
				Denom:  fee.Denom,
				Amount: validatorAmount,
			})
		}

		// Calculate community pool amount
		communityAmount := int64(float64(fee.Amount) * de.config.CommunityPool)
		if communityAmount > 0 {
			plan.CommunityPool = append(plan.CommunityPool, keepertypes.Coin{
				Denom:  fee.Denom,
				Amount: communityAmount,
			})
		}

		// Calculate developer fund amount
		developerAmount := int64(float64(fee.Amount) * de.config.DeveloperFund)
		if developerAmount > 0 {
			plan.DeveloperFund = append(plan.DeveloperFund, keepertypes.Coin{
				Denom:  fee.Denom,
				Amount: developerAmount,
			})
		}
	}

	return plan
}