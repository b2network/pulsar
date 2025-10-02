package keeper

import (
	"fmt"

	keepertypes "github.com/b2network/pulsar/keeper/types"
	"github.com/b2network/pulsar/modules/fee/distribution"
	feetypes "github.com/b2network/pulsar/modules/fee/types"
)

// Distribution-related methods for the fee keeper

// GetFeeDistributionConfig returns the current fee distribution configuration
func (k *Keeper) GetFeeDistributionConfig(ctx keepertypes.Context) feetypes.FeeDistributionConfig {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(feetypes.FeeDistributionConfigKey)
	if bz == nil {
		return feetypes.DefaultFeeDistributionConfig()
	}

	var config feetypes.FeeDistributionConfig
	if err := k.cdc.UnmarshalJSON(bz, &config); err != nil {
		return feetypes.DefaultFeeDistributionConfig()
	}
	return config
}

// SetFeeDistributionConfig sets the fee distribution configuration
func (k *Keeper) SetFeeDistributionConfig(ctx keepertypes.Context, config feetypes.FeeDistributionConfig) error {
	if err := config.ValidateBasic(); err != nil {
		return err
	}

	store := ctx.KVStore(k.storeKey)
	bz, err := k.cdc.MarshalJSON(&config)
	if err != nil {
		return err
	}

	store.Set(feetypes.FeeDistributionConfigKey, bz)

	// Emit event
	ctx.EventManager().EmitEvent(convertEvent(feetypes.NewEvent(
		feetypes.EventTypeFeeDistribution,
		feetypes.NewAttribute("action", "config_updated"),
		feetypes.NewAttribute("burn_percentage", fmt.Sprintf("%.6f", config.BurnPercentage)),
		feetypes.NewAttribute("validator_rewards", fmt.Sprintf("%.6f", config.ValidatorRewards)),
		feetypes.NewAttribute("community_pool", fmt.Sprintf("%.6f", config.CommunityPool)),
		feetypes.NewAttribute("developer_fund", fmt.Sprintf("%.6f", config.DeveloperFund)),
	)))

	return nil
}

// InitializeDistributionManager initializes the distribution manager
func (k *Keeper) InitializeDistributionManager() error {
	// Create distribution engine
	engine := distribution.NewDistributionEngine(
		k.GetFeeDistributionConfig(keepertypes.Context{}), // We'll need a proper context
		&BankKeeperWrapper{keeper: k.bankKeeper},
		&StakingKeeperWrapper{keeper: k.stakingKeeper},
		&DistributionKeeperWrapper{keeper: k.distributionKeeper},
	)

	// Create distribution manager with default config
	config := distribution.DefaultManagerConfig()
	k.distributionManager = distribution.NewDistributionManager(engine, config)

	return nil
}

// DistributeFees distributes collected fees using the distribution manager
func (k *Keeper) DistributeFees(ctx keepertypes.Context, fees keepertypes.Coins) (distribution.DistributionResult, error) {
	if k.distributionManager == nil {
		return distribution.DistributionResult{}, fmt.Errorf("distribution manager not initialized")
	}

	return k.distributionManager.DistributeFeesAuto(ctx, fees)
}

// DistributeFeesWithStrategy distributes fees using a specific strategy
func (k *Keeper) DistributeFeesWithStrategy(
	ctx keepertypes.Context,
	fees keepertypes.Coins,
	strategy string,
) (distribution.DistributionResult, error) {
	if k.distributionManager == nil {
		return distribution.DistributionResult{}, fmt.Errorf("distribution manager not initialized")
	}

	return k.distributionManager.DistributeFees(ctx, fees, strategy)
}

// GetDistributionStats returns distribution statistics
func (k *Keeper) GetDistributionStats() distribution.DistributionManagerStats {
	if k.distributionManager == nil {
		return distribution.DistributionManagerStats{}
	}

	return k.distributionManager.GetDistributionStats()
}

// GetDistributionHistory returns distribution history
func (k *Keeper) GetDistributionHistory(limit int) []distribution.DistributionRecord {
	if k.distributionManager == nil {
		return []distribution.DistributionRecord{}
	}

	return k.distributionManager.GetDistributionHistory(limit)
}

// RegisterDistributionStrategy registers a custom distribution strategy
func (k *Keeper) RegisterDistributionStrategy(name string, strategy distribution.DistributionStrategy) error {
	if k.distributionManager == nil {
		return fmt.Errorf("distribution manager not initialized")
	}

	return k.distributionManager.RegisterStrategy(name, strategy)
}

// GetAvailableDistributionStrategies returns available distribution strategies
func (k *Keeper) GetAvailableDistributionStrategies() []string {
	if k.distributionManager == nil {
		return []string{}
	}

	return k.distributionManager.ListStrategies()
}

// UpdateDistributionConfig updates the distribution configuration
func (k *Keeper) UpdateDistributionConfig(ctx keepertypes.Context, config feetypes.FeeDistributionConfig) error {
	// Update stored config
	if err := k.SetFeeDistributionConfig(ctx, config); err != nil {
		return err
	}

	// Update distribution engine config if manager exists
	if k.distributionManager != nil {
		// We would need to get the engine from the manager and update its config
		// This is a simplified approach - in practice you'd need proper engine access
	}

	return nil
}

// BankKeeperWrapper wraps the bank keeper to implement the distribution interface
type BankKeeperWrapper struct {
	keeper feetypes.BankKeeper
}

func (bk *BankKeeperWrapper) SendCoinsFromModuleToModule(
	ctx keepertypes.Context,
	senderModule, recipientModule string,
	amt keepertypes.Coins,
) error {
	// Convert keepertypes.Coins to the format expected by the bank keeper
	// This is a simplified conversion - you'd need proper type conversion
	return bk.keeper.SendCoinsFromModuleToModule(ctx, senderModule, recipientModule, amt)
}

func (bk *BankKeeperWrapper) SendCoinsFromModuleToAccount(
	ctx keepertypes.Context,
	senderModule string,
	recipientAddr []byte,
	amt keepertypes.Coins,
) error {
	return bk.keeper.SendCoinsFromModuleToAccount(ctx, senderModule, recipientAddr, amt)
}

func (bk *BankKeeperWrapper) BurnCoins(
	ctx keepertypes.Context,
	moduleName string,
	amounts keepertypes.Coins,
) error {
	return bk.keeper.BurnCoins(ctx, moduleName, amounts)
}

func (bk *BankKeeperWrapper) GetAllBalances(
	ctx keepertypes.Context,
	addr []byte,
) keepertypes.Coins {
	return bk.keeper.GetAllBalances(ctx, addr)
}

func (bk *BankKeeperWrapper) GetModuleAccount(
	ctx keepertypes.Context,
	moduleName string,
) []byte {
	// This would get the module account address
	// Simplified implementation
	return []byte(moduleName)
}

// StakingKeeperWrapper wraps the staking keeper
type StakingKeeperWrapper struct {
	keeper feetypes.StakingKeeper
}

func (sk *StakingKeeperWrapper) GetAllValidators(ctx keepertypes.Context) []distribution.ValidatorInfo {
	// This would convert from the staking keeper's validator format to distribution.ValidatorInfo
	// Simplified implementation
	return []distribution.ValidatorInfo{}
}

func (sk *StakingKeeperWrapper) GetBondedValidatorsByPower(ctx keepertypes.Context) []distribution.ValidatorInfo {
	// This would get bonded validators sorted by power
	// Simplified implementation
	return []distribution.ValidatorInfo{}
}

func (sk *StakingKeeperWrapper) GetValidator(
	ctx keepertypes.Context,
	addr []byte,
) (distribution.ValidatorInfo, bool) {
	// This would get a specific validator
	// Simplified implementation
	return distribution.ValidatorInfo{}, false
}

func (sk *StakingKeeperWrapper) GetValidatorDelegations(
	ctx keepertypes.Context,
	valAddr []byte,
) []distribution.DelegationInfo {
	// This would get validator delegations
	// Simplified implementation
	return []distribution.DelegationInfo{}
}

// DistributionKeeperWrapper wraps the distribution keeper
type DistributionKeeperWrapper struct {
	keeper feetypes.DistributionKeeper
}

func (dk *DistributionKeeperWrapper) AllocateTokensToValidator(
	ctx keepertypes.Context,
	val distribution.ValidatorInfo,
	tokens keepertypes.Coins,
) error {
	// This would allocate tokens to a validator
	// Simplified implementation
	return nil
}

func (dk *DistributionKeeperWrapper) GetCommunityPool(ctx keepertypes.Context) keepertypes.Coins {
	// This would get the community pool balance
	// Simplified implementation
	return keepertypes.Coins{}
}

func (dk *DistributionKeeperWrapper) FundCommunityPool(
	ctx keepertypes.Context,
	amount keepertypes.Coins,
	sender []byte,
) error {
	// This would fund the community pool
	// Simplified implementation
	return nil
}

func (dk *DistributionKeeperWrapper) GetFeePool(ctx keepertypes.Context) distribution.FeePool {
	// This would get the fee pool
	// Simplified implementation
	return distribution.FeePool{}
}

func (dk *DistributionKeeperWrapper) SetFeePool(ctx keepertypes.Context, feePool distribution.FeePool) {
	// This would set the fee pool
	// Simplified implementation
}

// Add distribution manager to Keeper struct (need to update keeper.go)
// distributionManager *distribution.DistributionManager

// Helper function to convert events
func convertEvent(event feetypes.Event) keepertypes.Event {
	attrs := make([]keepertypes.Attribute, len(event.Attributes))
	for i, attr := range event.Attributes {
		attrs[i] = keepertypes.Attribute{
			Key:   attr.Key,
			Value: attr.Value,
		}
	}
	return keepertypes.Event{
		Type:       event.Type,
		Attributes: attrs,
	}
}