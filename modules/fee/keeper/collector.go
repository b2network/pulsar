package keeper

import (
	"fmt"

	keepertypes "github.com/b2network/pulsar/keeper/types"
	feetypes "github.com/b2network/pulsar/modules/fee/types"
	commontypes "github.com/b2network/pulsar/types"
)

// FeeCollector implements fee collection and distribution
type FeeCollector struct {
	keeper *Keeper
}

// NewFeeCollector creates a new FeeCollector
func NewFeeCollector(keeper *Keeper) feetypes.FeeCollector {
	return &FeeCollector{
		keeper: keeper,
	}
}

// CollectFees collects fees from a transaction
func (fc *FeeCollector) CollectFees(ctx keepertypes.Context, fees commontypes.Coin) error {
	params := fc.keeper.GetParams(ctx)
	if !params.FeeCollectionEnabled {
		return nil // Fee collection is disabled
	}

	// Validate fee denomination
	if !fc.keeper.IsValidFeeDenom(ctx, fees.Denom) {
		return fmt.Errorf("invalid fee denomination: %s", fees.Denom)
	}

	// Validate fee amount
	if fees.Amount == "" || fees.Amount == "0" {
		return nil // No fees to collect
	}

	// Transfer fees from fee payer to fee collector module
	// Note: In a real implementation, this would transfer from the user account
	// to the fee collector module account. For now, we'll assume the fees
	// are already collected and just track them.

	// Store collected fees for later distribution
	if err := fc.storeFeeCollection(ctx, fees); err != nil {
		return fmt.Errorf("failed to store fee collection: %w", err)
	}

	// Emit fee collection event
	ctx.EventManager().EmitEvent(convertEvent(commontypes.NewEvent(
		"fee_collected",
		commontypes.NewAttribute("amount", fees.Amount),
		commontypes.NewAttribute("denom", fees.Denom),
		commontypes.NewAttribute("collector", feetypes.FeeCollectorName),
	)))

	return nil
}

// DistributeFees distributes collected fees according to configuration
func (fc *FeeCollector) DistributeFees(ctx keepertypes.Context) error {
	params := fc.keeper.GetParams(ctx)
	if !params.FeeCollectionEnabled {
		return nil // Fee collection is disabled
	}

	// Get all fee denominations to process
	feeDenoms := fc.keeper.GetAllFeeDenoms(ctx)

	for _, feeDenom := range feeDenoms {
		if !feeDenom.IsEnabled() {
			continue
		}

		// Get collected fees for this denomination
		collectedFee, err := fc.GetCollectedFees(ctx, feeDenom.Denom)
		if err != nil {
			continue // Skip if we can't get collected fees
		}

		if collectedFee.Amount == "" || collectedFee.Amount == "0" {
			continue // No fees to distribute
		}

		// Distribute the collected fees
		if err := fc.distributeFeesByDenom(ctx, collectedFee); err != nil {
			// Log error but continue with other denominations
			ctx.Logger().Error("failed to distribute fees", "denom", feeDenom.Denom, "error", err)
			continue
		}

		// Clear collected fees after distribution
		if err := fc.clearCollectedFees(ctx, feeDenom.Denom); err != nil {
			ctx.Logger().Error("failed to clear collected fees", "denom", feeDenom.Denom, "error", err)
		}
	}

	return nil
}

// GetCollectedFees returns the amount of fees collected for a denomination
func (fc *FeeCollector) GetCollectedFees(ctx keepertypes.Context, denom string) (commontypes.Coin, error) {
	store := ctx.KVStore(fc.keeper.storeKey)
	key := feetypes.GetCollectedFeesKey(denom)

	bz := store.Get(key)
	if bz == nil {
		return commontypes.NewCoin(denom, "0"), nil
	}

	var collectedFee commontypes.Coin
	if err := commontypes.UnmarshalJSON(bz, &collectedFee); err != nil {
		return commontypes.Coin{}, fmt.Errorf("failed to unmarshal collected fees: %w", err)
	}

	return collectedFee, nil
}

// storeFeeCollection stores collected fees in the state
func (fc *FeeCollector) storeFeeCollection(ctx keepertypes.Context, fees commontypes.Coin) error {
	// Get current collected fees
	currentFees, err := fc.GetCollectedFees(ctx, fees.Denom)
	if err != nil {
		return err
	}

	// Add new fees to current total
	newTotal, err := fc.addCoins(currentFees, fees)
	if err != nil {
		return fmt.Errorf("failed to add fees: %w", err)
	}

	// Store updated total
	store := ctx.KVStore(fc.keeper.storeKey)
	key := feetypes.GetCollectedFeesKey(fees.Denom)

	bz, err := commontypes.MarshalJSON(newTotal)
	if err != nil {
		return fmt.Errorf("failed to marshal collected fees: %w", err)
	}

	store.Set(key, bz)
	return nil
}

// distributeFeesByDenom distributes fees for a specific denomination
func (fc *FeeCollector) distributeFeesByDenom(ctx keepertypes.Context, fees commontypes.Coin) error {
	// Get fee distribution configuration
	// For now, use default configuration - this could be made configurable
	config := feetypes.DefaultFeeDistributionConfig()

	// Calculate distribution amounts
	totalAmount, err := commontypes.ParseCoinAmount(fees.Amount)
	if err != nil {
		return fmt.Errorf("invalid fee amount: %w", err)
	}

	// Calculate each distribution share
	validatorShare := totalAmount * config.ValidatorRewards
	communityShare := totalAmount * config.CommunityPool
	burnShare := totalAmount * config.BurnPercentage
	developerShare := totalAmount * config.DeveloperFund

	// Distribute to validators (through distribution module)
	if validatorShare > 0 {
		validatorCoin := commontypes.NewCoin(fees.Denom, fmt.Sprintf("%.0f", validatorShare))
		if err := fc.distributeToValidators(ctx, validatorCoin); err != nil {
			return fmt.Errorf("failed to distribute to validators: %w", err)
		}
	}

	// Add to community pool
	if communityShare > 0 {
		communityCoin := commontypes.NewCoin(fees.Denom, fmt.Sprintf("%.0f", communityShare))
		if err := fc.addToCommunityPool(ctx, communityCoin); err != nil {
			return fmt.Errorf("failed to add to community pool: %w", err)
		}
	}

	// Burn tokens
	if burnShare > 0 {
		burnCoin := commontypes.NewCoin(fees.Denom, fmt.Sprintf("%.0f", burnShare))
		if err := fc.burnFees(ctx, burnCoin); err != nil {
			return fmt.Errorf("failed to burn fees: %w", err)
		}
	}

	// Send to developer fund
	if developerShare > 0 {
		developerCoin := commontypes.NewCoin(fees.Denom, fmt.Sprintf("%.0f", developerShare))
		if err := fc.sendToDeveloperFund(ctx, developerCoin); err != nil {
			return fmt.Errorf("failed to send to developer fund: %w", err)
		}
	}

	// Emit distribution event
	ctx.EventManager().EmitEvent(convertEvent(commontypes.NewEvent(
		"fee_distributed",
		commontypes.NewAttribute("total_amount", fees.Amount),
		commontypes.NewAttribute("denom", fees.Denom),
		commontypes.NewAttribute("validator_share", fmt.Sprintf("%.0f", validatorShare)),
		commontypes.NewAttribute("community_share", fmt.Sprintf("%.0f", communityShare)),
		commontypes.NewAttribute("burn_share", fmt.Sprintf("%.0f", burnShare)),
		commontypes.NewAttribute("developer_share", fmt.Sprintf("%.0f", developerShare)),
	)))

	return nil
}

// clearCollectedFees clears collected fees for a denomination
func (fc *FeeCollector) clearCollectedFees(ctx keepertypes.Context, denom string) error {
	store := ctx.KVStore(fc.keeper.storeKey)
	key := feetypes.GetCollectedFeesKey(denom)
	store.Delete(key)
	return nil
}

// Helper methods for fee distribution

// distributeToValidators distributes fees to validators through the distribution module
func (fc *FeeCollector) distributeToValidators(ctx keepertypes.Context, fees commontypes.Coin) error {
	// Get all validators
	validators := fc.keeper.stakingKeeper.GetValidators(ctx, 100) // Get up to 100 validators

	if len(validators) == 0 {
		return fmt.Errorf("no validators found for fee distribution")
	}

	// Calculate total bonded tokens
	totalBonded := fc.keeper.stakingKeeper.TotalBondedTokens(ctx)
	if totalBonded == 0 {
		return fmt.Errorf("no bonded tokens found")
	}

	// Distribute proportionally to each validator based on their stake
	feeAmount, err := commontypes.ParseCoinAmount(fees.Amount)
	if err != nil {
		return err
	}

	for _, validator := range validators {
		// Calculate this validator's share
		validatorTokens := validator.GetTokens()
		share := float64(validatorTokens) / float64(totalBonded)
		validatorFee := feeAmount * share

		if validatorFee > 0 {
			validatorCoin := commontypes.NewCoin(fees.Denom, fmt.Sprintf("%.0f", validatorFee))
			coinsToDistribute := commontypes.Coins{validatorCoin}
			fc.keeper.distributionKeeper.AllocateTokensToValidator(ctx, validator, coinsToDistribute)
		}
	}

	return nil
}

// addToCommunityPool adds fees to the community pool
func (fc *FeeCollector) addToCommunityPool(ctx keepertypes.Context, fees commontypes.Coin) error {
	feePool := fc.keeper.distributionKeeper.GetFeePool(ctx)
	currentPool := feePool.GetCommunityPool()

	// Convert fees to keepertypes.Coins
	keeperCoin := keepertypes.Coin{
		Denom:  fees.Denom,
		Amount: 0, // Will parse below
	}

	// Parse amount from string to int64
	if amount, err := commontypes.ParseCoinAmount(fees.Amount); err == nil {
		keeperCoin.Amount = int64(amount)
	}

	// Add the new fees to the community pool
	newPool := currentPool.Add(keeperCoin)
	feePool.SetCommunityPool(newPool)
	fc.keeper.distributionKeeper.SetFeePool(ctx, feePool)

	return nil
}

// burnFees burns the specified fee amount
func (fc *FeeCollector) burnFees(ctx keepertypes.Context, fees commontypes.Coin) error {
	// Burn fees by sending them to the burn module or simply removing them from supply
	coinsToburn := commontypes.Coins{fees}
	return fc.keeper.bankKeeper.BurnCoins(ctx, feetypes.FeeCollectorName, coinsToburn)
}

// sendToDeveloperFund sends fees to a developer fund address
func (fc *FeeCollector) sendToDeveloperFund(ctx keepertypes.Context, fees commontypes.Coin) error {
	// This would send to a specific developer fund address
	// For now, we'll add it to the community pool as a placeholder
	return fc.addToCommunityPool(ctx, fees)
}

// addCoins adds two coins of the same denomination
func (fc *FeeCollector) addCoins(coin1, coin2 commontypes.Coin) (commontypes.Coin, error) {
	if coin1.Denom != coin2.Denom {
		return commontypes.Coin{}, fmt.Errorf("cannot add coins of different denominations: %s and %s", coin1.Denom, coin2.Denom)
	}

	amount1, err := commontypes.ParseCoinAmount(coin1.Amount)
	if err != nil {
		return commontypes.Coin{}, err
	}

	amount2, err := commontypes.ParseCoinAmount(coin2.Amount)
	if err != nil {
		return commontypes.Coin{}, err
	}

	total := amount1 + amount2
	return commontypes.NewCoin(coin1.Denom, fmt.Sprintf("%.0f", total)), nil
}

