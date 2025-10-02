package ante

import (
	"fmt"

	keepertypes "github.com/b2network/pulsar/keeper/types"
	"github.com/b2network/pulsar/modules/fee/keeper"
	"github.com/b2network/pulsar/modules/fee/types"
	commontypes "github.com/b2network/pulsar/types"
)

// FeeAnteHandlerOptions defines the options for creating fee ante handlers
type FeeAnteHandlerOptions struct {
	AccountKeeper AccountKeeper
	BankKeeper    BankKeeper
	FeeKeeper     keeper.Keeper
	BypassMinFee  bool
	MaxGasWanted  uint64
}

// NewFeeAnteHandler creates a new ante handler chain for fee processing
func NewFeeAnteHandler(options FeeAnteHandlerOptions) AnteHandlerFunc {
	return ChainAnteDecorators(
		// Fee validation and deduction
		NewFeeDecorator(
			options.AccountKeeper,
			options.BankKeeper,
			options.FeeKeeper,
		),
		// Gas metering and tracking
		NewGasMeterDecorator(options.FeeKeeper),
		// Set gas consumption for specific message types
		NewSetGasConsumedDecorator(options.FeeKeeper),
		// Gas refund processing (applied last)
		NewRefundGasDecorator(
			options.AccountKeeper,
			options.BankKeeper,
			options.FeeKeeper,
		),
	)
}

// NewFeeAnteHandlerWithOptions creates a fee ante handler with custom options
func NewFeeAnteHandlerWithOptions(options FeeAnteHandlerOptions) AnteHandlerFunc {
	feeDecorator := NewFeeDecoratorWithOptions(
		options.AccountKeeper,
		options.BankKeeper,
		options.FeeKeeper,
		options.BypassMinFee,
		options.MaxGasWanted,
	)

	return ChainAnteDecorators(
		feeDecorator,
		NewGasMeterDecorator(options.FeeKeeper),
		NewSetGasConsumedDecorator(options.FeeKeeper),
		NewRefundGasDecorator(
			options.AccountKeeper,
			options.BankKeeper,
			options.FeeKeeper,
		),
	)
}

// SimpleAnteHandler creates a simple ante handler for basic fee processing
func SimpleAnteHandler(ak AccountKeeper, bk BankKeeper, fk keeper.Keeper) AnteHandlerFunc {
	return func(ctx keepertypes.Context, tx Tx, simulate bool) (newCtx keepertypes.Context, err error) {
		// Create ante handler with default options
		options := FeeAnteHandlerOptions{
			AccountKeeper: ak,
			BankKeeper:    bk,
			FeeKeeper:     fk,
			BypassMinFee:  false,
			MaxGasWanted:  1000000,
		}

		anteHandler := NewFeeAnteHandler(options)
		return anteHandler(ctx, tx, simulate)
	}
}

// ValidateTxFees validates transaction fees without processing them
func ValidateTxFees(ctx keepertypes.Context, tx Tx, fk keeper.Keeper) error {
	feeTx, ok := tx.(FeeTx)
	if !ok {
		return fmt.Errorf("Tx must implement FeeTx interface")
	}

	fees := feeTx.GetFee()
	gas := feeTx.GetGas()

	// Basic validations
	if gas == 0 {
		return fmt.Errorf("gas limit cannot be zero")
	}

	for _, fee := range fees {
		if fee.Amount < 0 {
			return fmt.Errorf("negative fees not allowed: %v", fees)
		}
	}

	// Validate fee denominations
	feeDenoms := fk.GetEnabledFeeDenoms(ctx)
	if len(feeDenoms) == 0 {
		return fmt.Errorf("no fee denominations configured")
	}

	for _, fee := range fees {
		accepted := false
		for _, denom := range feeDenoms {
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

// EstimateGasForTx estimates gas consumption for a transaction
func EstimateGasForTx(ctx keepertypes.Context, tx Tx, fk keeper.Keeper) (uint64, error) {
	msgs := tx.GetMsgs()
	if len(msgs) == 0 {
		return 0, fmt.Errorf("transaction has no messages")
	}

	totalGas := uint64(10000) // Base transaction gas

	for _, msg := range msgs {
		// Get message type information
		msgType := fmt.Sprintf("%T", msg)

		// Simple type parsing for module and message name
		var moduleName, msgName string = "unknown", "unknown"

		// Extract module and message type from the type string
		// This is a simplified version - in a real implementation
		// you would parse the type more carefully
		if len(msgType) > 0 {
			moduleName = "default"
			msgName = msgType
		}

		// Get configured gas for this message type
		msgGas := fk.GetMessageGas(ctx, moduleName, msgName)
		if msgGas == 0 {
			msgGas = 25000 // Default gas for unknown message types
		}

		totalGas += msgGas
	}

	// Apply dynamic gas factors
	factors := fk.GetDynamicGasFactors(ctx)

	// Apply size factor based on transaction size (simulated)
	// In a real implementation, you would get the actual tx bytes
	txSize := uint64(len(msgs) * 100) // Estimate based on number of messages
	sizeKB := float64(txSize) / 1024.0
	sizeGas := uint64(sizeKB * factors.SizeMultiplier * 1000)
	totalGas += sizeGas

	// Apply network congestion factor
	congestion := fk.GetNetworkCongestion(ctx)
	if congestion > 0 {
		congestionGas := uint64(congestion * factors.NetworkFactor * 1000)
		totalGas += congestionGas
	}

	return totalGas, nil
}

// CalculatePriority calculates transaction priority based on fees
func CalculatePriority(ctx keepertypes.Context, tx Tx, fk keeper.Keeper) int64 {
	feeTx, ok := tx.(FeeTx)
	if !ok {
		return 0
	}

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
		priority := getDenomPriority(ctx, fk, fee.Denom)
		weightedPrice := gasPrice * float64(priority)

		totalPrice += weightedPrice
	}

	// Convert to priority (higher price = higher priority)
	// Scale by 1000000 for precision
	priority := int64(totalPrice * 1000000)

	return priority
}

// getDenomPriority gets the priority multiplier for a denomination
func getDenomPriority(ctx keepertypes.Context, fk keeper.Keeper, denom string) uint32 {
	feeDenoms := fk.GetAllFeeDenoms(ctx)
	for _, feeDenom := range feeDenoms {
		if feeDenom.Denom == denom {
			return feeDenom.Priority
		}
	}
	return 1 // Default priority
}