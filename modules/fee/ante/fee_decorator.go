package ante

import (
	"fmt"

	keepertypes "github.com/b2network/pulsar/keeper/types"
	"github.com/b2network/pulsar/modules/fee/keeper"
	"github.com/b2network/pulsar/modules/fee/types"
)

// FeeDecorator is responsible for fee validation and deduction during transaction processing
type FeeDecorator struct {
	ak AccountKeeper
	bk BankKeeper
	fk keeper.Keeper
	// Configuration
	bypassMinFee bool
	maxGasWanted uint64
}

// NewFeeDecorator creates a new FeeDecorator
func NewFeeDecorator(ak AccountKeeper, bk BankKeeper, fk keeper.Keeper) FeeDecorator {
	return FeeDecorator{
		ak:           ak,
		bk:           bk,
		fk:           fk,
		bypassMinFee: false,
		maxGasWanted: 1000000, // Default max gas
	}
}

// NewFeeDecoratorWithOptions creates a new FeeDecorator with options
func NewFeeDecoratorWithOptions(
	ak AccountKeeper,
	bk BankKeeper,
	fk keeper.Keeper,
	bypassMinFee bool,
	maxGasWanted uint64,
) FeeDecorator {
	return FeeDecorator{
		ak:           ak,
		bk:           bk,
		fk:           fk,
		bypassMinFee: bypassMinFee,
		maxGasWanted: maxGasWanted,
	}
}

// AnteHandle validates and processes fees during transaction execution
func (fd FeeDecorator) AnteHandle(ctx keepertypes.Context, tx Tx, simulate bool, next AnteHandlerFunc) (newCtx keepertypes.Context, err error) {
	// Extract fee transaction
	feeTx, ok := tx.(FeeTx)
	if !ok {
		return ctx, fmt.Errorf("Tx must implement FeeTx interface")
	}

	// Skip fee checks during CheckTx if bypass is enabled
	extCtx := NewExtendedContext(ctx, NewInfiniteGasMeter())
	if extCtx.IsCheckTx() && fd.bypassMinFee {
		return next(ctx, tx, simulate)
	}

	// Validate basic fee requirements
	if err := fd.validateBasicFees(ctx, feeTx); err != nil {
		return ctx, err
	}

	// Get priority based on fee amount
	priority := fd.getPriority(ctx, feeTx)
	newCtx = extCtx.WithPriority(priority).(keepertypes.Context)

	// In simulation mode, skip actual fee deduction
	if simulate {
		// Set gas meter for simulation
		newCtx = fd.setGasMeter(newCtx, feeTx, simulate)
		return next(newCtx, tx, simulate)
	}

	// Process fees (validation, deduction, distribution)
	if err := fd.processFees(newCtx, feeTx); err != nil {
		return ctx, err
	}

	// Set gas meter for execution
	newCtx = fd.setGasMeter(newCtx, feeTx, simulate)

	// Continue to next handler
	return next(newCtx, tx, simulate)
}

// validateBasicFees performs basic fee validation
func (fd FeeDecorator) validateBasicFees(ctx keepertypes.Context, feeTx FeeTx) error {
	gas := feeTx.GetGas()
	fees := feeTx.GetFee()

	// Check gas limits
	if gas == 0 {
		return fmt.Errorf("gas limit cannot be zero")
	}

	if gas > fd.maxGasWanted {
		return fmt.Errorf(
			"gas limit %d exceeds maximum allowed %d",
			gas, fd.maxGasWanted,
		)
	}

	// Check fee amounts
	for _, fee := range fees {
		if fee.Amount < 0 {
			return fmt.Errorf("negative fees not allowed: %v", fees)
		}
	}

	// Validate fee denominations
	if !fd.bypassMinFee {
		if err := fd.validateFeeDenominations(ctx, fees); err != nil {
			return err
		}
	}

	return nil
}

// validateFeeDenominations validates that fees use accepted denominations
func (fd FeeDecorator) validateFeeDenominations(ctx keepertypes.Context, fees keepertypes.Coins) error {
	// Get accepted fee denominations
	acceptedDenoms := fd.fk.GetEnabledFeeDenoms(ctx)
	if len(acceptedDenoms) == 0 {
		return fmt.Errorf("no fee denominations configured")
	}

	// Check each fee coin
	for _, fee := range fees {
		accepted := false
		for _, denom := range acceptedDenoms {
			if fee.Denom == denom.Denom {
				accepted = true
				break
			}
		}
		if !accepted {
			return fmt.Errorf(
				"fee denomination %s not accepted",
				fee.Denom,
			)
		}
	}

	return nil
}

// processFees handles the complete fee processing pipeline
func (fd FeeDecorator) processFees(ctx keepertypes.Context, feeTx FeeTx) error {
	fees := feeTx.GetFee()
	gas := feeTx.GetGas()
	feePayer := feeTx.FeePayer()
	feeGranter := feeTx.FeeGranter()

	// Determine who pays the fees
	deductFrom := feePayer
	if feeGranter != nil {
		// If fee granter is specified, use it
		deductFrom = feeGranter
	}

	// Calculate minimum required fees
	minFees := fd.calculateMinimumFees(ctx, gas)

	// Validate fees meet minimum requirements
	if !fd.feesAreGreaterOrEqual(fees, minFees) {
		return fmt.Errorf(
			"insufficient fees; got: %v, required: %v",
			fees, minFees,
		)
	}

	// Check account balance
	if err := fd.checkBalance(ctx, deductFrom, fees); err != nil {
		return err
	}

	// Deduct fees from account
	if err := fd.deductFeesFromAccount(ctx, deductFrom, fees); err != nil {
		return err
	}

	// Distribute collected fees
	if err := fd.distributeFees(ctx, fees); err != nil {
		return err
	}

	// Emit fee events
	fd.emitFeeEvents(ctx, deductFrom, fees, gas)

	// Update fee statistics
	fd.updateFeeStatistics(ctx, fees, gas)

	return nil
}

// calculateMinimumFees calculates the minimum required fees based on gas
func (fd FeeDecorator) calculateMinimumFees(ctx keepertypes.Context, gasLimit uint64) keepertypes.Coins {
	var minFees keepertypes.Coins

	// Get all enabled fee denominations
	feeDenoms := fd.fk.GetEnabledFeeDenoms(ctx)

	// Calculate minimum for each denomination
	for _, denom := range feeDenoms {
		price, denomStr, err := types.ParseGasPrice(denom.MinGasPrice)
		if err != nil {
			continue
		}

		minFee := int64(price * float64(gasLimit))
		coin := keepertypes.Coin{Denom: denomStr, Amount: minFee}
		minFees = append(minFees, coin)
	}

	return minFees
}

// feesAreGreaterOrEqual checks if fees are greater than or equal to minimum
func (fd FeeDecorator) feesAreGreaterOrEqual(fees, minFees keepertypes.Coins) bool {
	for _, minFee := range minFees {
		found := false
		for _, fee := range fees {
			if fee.Denom == minFee.Denom && fee.Amount >= minFee.Amount {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// checkBalance checks if account has sufficient balance for fees
func (fd FeeDecorator) checkBalance(ctx keepertypes.Context, addr []byte, fees keepertypes.Coins) error {
	balance := fd.bk.GetAllBalances(ctx, addr)

	for _, fee := range fees {
		var hasBalance bool
		for _, bal := range balance {
			if bal.Denom == fee.Denom && bal.Amount >= fee.Amount {
				hasBalance = true
				break
			}
		}
		if !hasBalance {
			return fmt.Errorf(
				"insufficient balance to pay fees; balance: %v, fees: %v",
				balance, fees,
			)
		}
	}
	return nil
}

// deductFeesFromAccount deducts fees from the specified account
func (fd FeeDecorator) deductFeesFromAccount(ctx keepertypes.Context, addr []byte, fees keepertypes.Coins) error {
	// Send fees to fee collector module
	err := fd.bk.SendCoinsFromAccountToModule(ctx, addr, types.FeeCollectorName, fees)
	if err != nil {
		return fmt.Errorf("failed to deduct fees from account %x: %w", addr, err)
	}
	return nil
}

// distributeFees distributes collected fees according to configuration
func (fd FeeDecorator) distributeFees(ctx keepertypes.Context, fees keepertypes.Coins) error {
	// Get distribution configuration
	config := fd.fk.GetFeeDistributionConfig(ctx)

	// Process each fee coin
	for _, fee := range fees {
		burnAmount := int64(float64(fee.Amount) * config.BurnPercentage)
		validatorAmount := int64(float64(fee.Amount) * config.ValidatorRewards)
		communityAmount := int64(float64(fee.Amount) * config.CommunityPool)
		developerAmount := int64(float64(fee.Amount) * config.DeveloperFund)

		// Execute distribution
		if burnAmount > 0 {
			burnCoins := keepertypes.Coins{{Denom: fee.Denom, Amount: burnAmount}}
			if err := fd.bk.BurnCoins(ctx, types.FeeCollectorName, burnCoins); err != nil {
				return fmt.Errorf("failed to burn fees: %w", err)
			}
		}

		if validatorAmount > 0 {
			validatorCoins := keepertypes.Coins{{Denom: fee.Denom, Amount: validatorAmount}}
			if err := fd.bk.SendCoinsFromModuleToModule(
				ctx, types.FeeCollectorName, types.StakingRewardsName, validatorCoins,
			); err != nil {
				return fmt.Errorf("failed to send fees to staking rewards: %w", err)
			}
		}

		if communityAmount > 0 {
			communityCoins := keepertypes.Coins{{Denom: fee.Denom, Amount: communityAmount}}
			if err := fd.bk.SendCoinsFromModuleToModule(
				ctx, types.FeeCollectorName, types.CommunityPoolName, communityCoins,
			); err != nil {
				return fmt.Errorf("failed to send fees to community pool: %w", err)
			}
		}

		if developerAmount > 0 {
			developerCoins := keepertypes.Coins{{Denom: fee.Denom, Amount: developerAmount}}
			if err := fd.bk.SendCoinsFromModuleToModule(
				ctx, types.FeeCollectorName, types.DeveloperFundName, developerCoins,
			); err != nil {
				return fmt.Errorf("failed to send fees to developer fund: %w", err)
			}
		}
	}

	return nil
}

// getPriority returns transaction priority based on fee amount
func (fd FeeDecorator) getPriority(ctx keepertypes.Context, feeTx FeeTx) int64 {
	fees := feeTx.GetFee()
	gas := feeTx.GetGas()

	if gas == 0 || len(fees) == 0 {
		return 0
	}

	// Calculate average gas price across all fees
	var totalPrice float64
	for _, fee := range fees {
		gasPrice := float64(fee.Amount) / float64(gas)

		// Get denomination priority multiplier
		priority := fd.getDenomPriority(ctx, fee.Denom)
		weightedPrice := gasPrice * float64(priority)

		totalPrice += weightedPrice
	}

	// Convert to priority (higher price = higher priority)
	// Scale by 1000000 for precision
	priority := int64(totalPrice * 1000000)

	return priority
}

// getDenomPriority gets the priority multiplier for a denomination
func (fd FeeDecorator) getDenomPriority(ctx keepertypes.Context, denom string) uint32 {
	feeDenoms := fd.fk.GetAllFeeDenoms(ctx)
	for _, feeDenom := range feeDenoms {
		if feeDenom.Denom == denom {
			return feeDenom.Priority
		}
	}
	return 1 // Default priority
}

// setGasMeter sets the gas meter for transaction execution
func (fd FeeDecorator) setGasMeter(ctx keepertypes.Context, feeTx FeeTx, simulate bool) keepertypes.Context {
	gasLimit := feeTx.GetGas()

	var gasMeter GasMeter
	// In simulation mode, use infinite gas meter
	if simulate {
		gasMeter = NewInfiniteGasMeter()
	} else {
		// Set gas meter with the specified limit
		gasMeter = NewGasMeter(gasLimit)
	}

	return NewExtendedContext(ctx, gasMeter).(keepertypes.Context)
}

// emitFeeEvents emits events related to fee payment
func (fd FeeDecorator) emitFeeEvents(ctx keepertypes.Context, feePayer []byte, fees keepertypes.Coins, gas uint64) {
	feePayerStr := fmt.Sprintf("%x", feePayer)
	feesStr := ""
	for i, fee := range fees {
		if i > 0 {
			feesStr += ","
		}
		feesStr += fmt.Sprintf("%d%s", fee.Amount, fee.Denom)
	}

	events := []keepertypes.Event{
		{
			Type: types.EventTypeFeePayment,
			Attributes: []keepertypes.Attribute{
				{Key: types.AttributeKeyFeePayer, Value: feePayerStr},
				{Key: types.AttributeKeyFees, Value: feesStr},
				{Key: types.AttributeKeyGasLimit, Value: fmt.Sprintf("%d", gas)},
			},
		},
	}

	ctx.EventManager().EmitEvents(events)
}

// updateFeeStatistics updates fee-related statistics
func (fd FeeDecorator) updateFeeStatistics(ctx keepertypes.Context, fees keepertypes.Coins, gas uint64) {
	// Update gas price statistics for each fee denomination
	for _, fee := range fees {
		if gas > 0 {
			gasPrice := float64(fee.Amount) / float64(gas)

			// Update statistics in keeper
			fd.fk.UpdateGasPriceStatistics(ctx, fee.Denom, gasPrice)
		}
	}

	// Update total fees collected
	fd.fk.IncrementTotalFeesCollected(ctx, fees)
}