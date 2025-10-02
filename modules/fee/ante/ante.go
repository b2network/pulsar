package ante

import (
	"fmt"

	keepertypes "github.com/b2network/pulsar/keeper/types"
	"github.com/b2network/pulsar/modules/fee/keeper"
	"github.com/b2network/pulsar/modules/fee/types"
	commontypes "github.com/b2network/pulsar/types"
)

// AnteHandler is the fee module's AnteHandler for processing fees at runtime
type AnteHandler struct {
	ak           AccountKeeper
	bk           BankKeeper
	fk           keeper.Keeper
	bypassMinFee bool
}

// NewAnteHandler creates a new AnteHandler for fee processing
func NewAnteHandler(ak AccountKeeper, bk BankKeeper, fk keeper.Keeper, bypassMinFee bool) AnteHandler {
	return AnteHandler{
		ak:           ak,
		bk:           bk,
		fk:           fk,
		bypassMinFee: bypassMinFee,
	}
}

// AnteHandle processes fees during transaction execution
func (ah AnteHandler) AnteHandle(ctx keepertypes.Context, tx Tx, simulate bool, next AnteHandlerFunc) (newCtx keepertypes.Context, err error) {
	// Extract fee transaction
	feeTx, ok := tx.(FeeTx)
	if !ok {
		return ctx, fmt.Errorf("Tx must be a FeeTx")
	}

	// Skip fee deduction in simulation mode
	if simulate && !ah.bypassMinFee {
		return next(ctx, tx, simulate)
	}

	// Get transaction fees
	fees := feeTx.GetFee()
	gas := feeTx.GetGas()
	feePayer := feeTx.FeePayer()
	feeGranter := feeTx.FeeGranter()

	// Determine actual fee payer
	deductFrom := feePayer
	if feeGranter != nil {
		deductFrom = feeGranter
	}

	// Validate fees
	if err := ah.validateFees(ctx, fees, gas, tx); err != nil {
		return ctx, err
	}

	// Deduct fees from account
	if len(fees) > 0 {
		if err := ah.deductFees(ctx, deductFrom, fees); err != nil {
			return ctx, err
		}

		// Process fee distribution
		if err := ah.processFeeDistribution(ctx, fees); err != nil {
			return ctx, err
		}

		// Record fee payment event
		ah.emitFeeEvent(ctx, deductFrom, fees, gas)
	}

	// Update gas meter with actual gas limit
	extCtx := NewExtendedContext(ctx, NewGasMeter(gas))
	newCtx = extCtx.(keepertypes.Context)

	// Call next handler
	return next(newCtx, tx, simulate)
}

// validateFees validates that fees meet minimum requirements
func (ah AnteHandler) validateFees(ctx keepertypes.Context, fees keepertypes.Coins, gasLimit uint64, tx Tx) error {
	if ah.bypassMinFee {
		return nil
	}

	// Get all enabled fee denominations
	feeDenoms := ah.fk.GetAllFeeDenoms(ctx)

	// Check if fees are in accepted denominations
	validDenom := false
	var minRequired keepertypes.Coins

	for _, feeDenom := range feeDenoms {
		if !feeDenom.Enabled {
			continue
		}

		// Check if fee is paid in this denomination
		var feeAmount int64
		for _, fee := range fees {
			if fee.Denom == feeDenom.Denom {
				feeAmount = fee.Amount
				break
			}
		}

		if feeAmount > 0 {
			validDenom = true

			// Parse minimum gas price from string
			price, denom, err := types.ParseGasPrice(feeDenom.MinGasPrice)
			if err != nil || denom != feeDenom.Denom {
				continue
			}

			// Calculate minimum required fee
			minFee := int64(price * float64(gasLimit))
			minCoin := keepertypes.Coin{Denom: feeDenom.Denom, Amount: minFee}

			// Check if provided fee meets minimum
			if feeAmount < minFee {
				return fmt.Errorf(
					"insufficient fee: got %v, minimum required %v",
					fees, minCoin,
				)
			}

			minRequired = minRequired.Add(minCoin)
			break
		}
	}

	if !validDenom && len(fees) > 0 {
		return fmt.Errorf(
			"fee denomination not accepted: %v",
			fees,
		)
	}

	// For transactions with no fees, check if zero fees are allowed
	if len(fees) == 0 && !ah.isZeroFeeAllowed(ctx, tx) {
		return fmt.Errorf("zero fees not allowed for this transaction")
	}

	return nil
}

// deductFees deducts fees from the fee payer's account
func (ah AnteHandler) deductFees(ctx keepertypes.Context, feePayer []byte, fees keepertypes.Coins) error {
	// Check if account has sufficient balance
	balance := ah.bk.GetAllBalances(ctx, feePayer)

	// Check if balance is sufficient for each fee coin
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
				"insufficient funds to pay fees: balance %v, fees %v",
				balance, fees,
			)
		}
	}

	// Transfer fees from account to fee collector module
	err := ah.bk.SendCoinsFromAccountToModule(ctx, feePayer, types.FeeCollectorName, fees)
	if err != nil {
		return fmt.Errorf(
			"failed to deduct fees: %s",
			err,
		)
	}

	return nil
}

// processFeeDistribution distributes collected fees according to configuration
func (ah AnteHandler) processFeeDistribution(ctx keepertypes.Context, fees keepertypes.Coins) error {
	// Get fee distribution configuration
	feeDistribution := ah.fk.GetFeeDistributionConfig(ctx)

	// Calculate distribution amounts for each coin
	for _, fee := range fees {
		burnAmount := int64(float64(fee.Amount) * feeDistribution.BurnPercentage)
		validatorAmount := int64(float64(fee.Amount) * feeDistribution.ValidatorRewards)
		communityAmount := int64(float64(fee.Amount) * feeDistribution.CommunityPool)
		developerAmount := int64(float64(fee.Amount) * feeDistribution.DeveloperFund)

		// Burn portion of fees
		if burnAmount > 0 {
			burnCoins := keepertypes.Coins{keepertypes.Coin{Denom: fee.Denom, Amount: burnAmount}}
			if err := ah.bk.BurnCoins(ctx, types.FeeCollectorName, burnCoins); err != nil {
				return fmt.Errorf("failed to burn fees: %w", err)
			}
		}

		// Distribute to validators (staking rewards)
		if validatorAmount > 0 {
			validatorCoins := keepertypes.Coins{keepertypes.Coin{Denom: fee.Denom, Amount: validatorAmount}}
			if err := ah.bk.SendCoinsFromModuleToModule(
				ctx, types.FeeCollectorName, types.StakingRewardsName, validatorCoins,
			); err != nil {
				return fmt.Errorf("failed to distribute to validators: %w", err)
			}
		}

		// Distribute to community pool
		if communityAmount > 0 {
			communityCoins := keepertypes.Coins{keepertypes.Coin{Denom: fee.Denom, Amount: communityAmount}}
			if err := ah.bk.SendCoinsFromModuleToModule(
				ctx, types.FeeCollectorName, types.CommunityPoolName, communityCoins,
			); err != nil {
				return fmt.Errorf("failed to distribute to community pool: %w", err)
			}
		}

		// Distribute to developer fund
		if developerAmount > 0 {
			developerCoins := keepertypes.Coins{keepertypes.Coin{Denom: fee.Denom, Amount: developerAmount}}
			if err := ah.bk.SendCoinsFromModuleToModule(
				ctx, types.FeeCollectorName, types.DeveloperFundName, developerCoins,
			); err != nil {
				return fmt.Errorf("failed to distribute to developer fund: %w", err)
			}
		}
	}

	return nil
}

// isZeroFeeAllowed checks if zero fees are allowed for specific transaction types
func (ah AnteHandler) isZeroFeeAllowed(ctx keepertypes.Context, tx Tx) bool {
	// Get messages from transaction
	msgs := tx.GetMsgs()
	if len(msgs) == 0 {
		return false
	}

	// Check if any message type allows zero fees
	for _, msg := range msgs {
		msgType := fmt.Sprintf("%T", msg)

		// System messages that might allow zero fees
		systemMessages := []string{
			"*gov.MsgVote",
			"*staking.MsgCreateValidator",
			"*crisis.MsgVerifyInvariant",
		}

		for _, sysMsg := range systemMessages {
			if msgType == sysMsg {
				return true
			}
		}
	}

	return false
}

// emitFeeEvent emits a fee payment event
func (ah AnteHandler) emitFeeEvent(ctx keepertypes.Context, feePayer []byte, fees keepertypes.Coins, gas uint64) {
	feePayerStr := fmt.Sprintf("%x", feePayer)
	feesStr := ""
	for i, fee := range fees {
		if i > 0 {
			feesStr += ","
		}
		feesStr += fmt.Sprintf("%d%s", fee.Amount, fee.Denom)
	}

	event := keepertypes.Event{
		Type: types.EventTypeFeePayment,
		Attributes: []keepertypes.Attribute{
			{Key: types.AttributeKeyFeePayer, Value: feePayerStr},
			{Key: types.AttributeKeyFees, Value: feesStr},
			{Key: types.AttributeKeyGasLimit, Value: fmt.Sprintf("%d", gas)},
		},
	}

	ctx.EventManager().EmitEvent(event)
}