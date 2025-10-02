package ante

import (
	"fmt"

	keepertypes "github.com/b2network/pulsar/keeper/types"
	"github.com/b2network/pulsar/modules/fee/keeper"
	"github.com/b2network/pulsar/modules/fee/types"
	commontypes "github.com/b2network/pulsar/types"
)

// GasMeterDecorator tracks and validates gas consumption during transaction execution
type GasMeterDecorator struct {
	fk keeper.Keeper
}

// NewGasMeterDecorator creates a new GasMeterDecorator
func NewGasMeterDecorator(fk keeper.Keeper) GasMeterDecorator {
	return GasMeterDecorator{
		fk: fk,
	}
}

// AnteHandle tracks gas consumption and applies dynamic gas adjustments
func (gd GasMeterDecorator) AnteHandle(ctx keepertypes.Context, tx Tx, simulate bool, next AnteHandlerFunc) (newCtx keepertypes.Context, err error) {
	// Create extended context with gas meter
	extCtx := NewExtendedContext(ctx, NewInfiniteGasMeter())
	gasConsumedBefore := extCtx.GasMeter().GasConsumed()

	// Defer function to handle panics from gas consumption
	defer func() {
		if r := recover(); r != nil {
			switch rType := r.(type) {
			case ErrorOutOfGas:
				err = fmt.Errorf(
					"out of gas in location: %v; gas consumed: %d",
					rType.Descriptor, extCtx.GasMeter().GasConsumed(),
				)
			default:
				panic(r)
			}
		}
	}()

	// Apply dynamic gas factors to context
	newCtx = gd.applyDynamicGasFactors(extCtx.(keepertypes.Context), tx)

	// Track message-specific gas consumption
	msgs := tx.GetMsgs()
	for i, msg := range msgs {
		// Get module and message type
		moduleName, messageType := gd.getMessageInfo(msg)

		// Get configured gas for this message type
		configuredGas := gd.fk.GetMessageGas(newCtx, moduleName, messageType)

		// Apply pre-message gas consumption
		if configuredGas > 0 && !simulate {
			extNewCtx := NewExtendedContext(newCtx, NewGasMeter(1000000))
			extNewCtx.GasMeter().ConsumeGas(configuredGas, fmt.Sprintf("msg[%d] %s.%s", i, moduleName, messageType))
			newCtx = extNewCtx.(keepertypes.Context)
		}

		// Track gas consumption for profiling
		if gd.fk.IsProfilingEnabled(newCtx) {
			extNewCtx := NewExtendedContext(newCtx, NewGasMeter(1000000))
			gasBeforeMsg := extNewCtx.GasMeter().GasConsumed()
			defer func(idx int, module, msgType string, gasBefore uint64) {
				gasAfterMsg := extNewCtx.GasMeter().GasConsumed()
				gasUsed := gasAfterMsg - gasBefore

				// Record gas usage for profiling
				gd.fk.RecordGasUsage(
					newCtx,
					module,
					msgType,
					gasUsed,
					extNewCtx.GasMeter().Limit(),
					err == nil,
					gd.extractMessageParameters(msg),
				)
			}(i, moduleName, messageType, gasBeforeMsg)
		}
	}

	// Call next handler
	newCtx, err = next(newCtx, tx, simulate)

	// Track total gas consumed
	extNewCtx := NewExtendedContext(newCtx, NewGasMeter(1000000))
	gasConsumedAfter := extNewCtx.GasMeter().GasConsumed()
	totalGasUsed := gasConsumedAfter - gasConsumedBefore

	// Update gas consumption metrics
	if !simulate {
		gd.updateGasMetrics(newCtx, totalGasUsed, msgs)
	}

	return newCtx, err
}

// applyDynamicGasFactors applies dynamic gas multipliers to the context
func (gd GasMeterDecorator) applyDynamicGasFactors(ctx keepertypes.Context, tx Tx) keepertypes.Context {
	// Get dynamic gas factors
	factors := gd.fk.GetDynamicGasFactors(ctx)

	// Create extended context for gas operations
	extCtx := NewExtendedContext(ctx, NewGasMeter(1000000))

	// Calculate transaction size factor (simulated)
	txSize := uint64(len(tx.GetMsgs()) * 100) // Estimate based on number of messages
	if txSize > 0 {
		// Apply size multiplier based on transaction size
		// Every 1KB increases gas by sizeMultiplier
		sizeKB := float64(txSize) / 1024.0
		sizeGas := uint64(sizeKB * factors.SizeMultiplier * 1000)
		extCtx.GasMeter().ConsumeGas(sizeGas, "tx size multiplier")
	}

	// Apply network congestion factor
	congestion := gd.fk.GetNetworkCongestion(ctx)
	if congestion > 0 {
		congestionGas := uint64(congestion * factors.NetworkFactor * 1000)
		extCtx.GasMeter().ConsumeGas(congestionGas, "network congestion")
	}

	// Store factors in context for message handlers to use
	newCtx := extCtx.WithValue(types.ContextKeyDynamicFactors, factors)

	return newCtx.(keepertypes.Context)
}

// getMessageInfo extracts module and message type information
func (gd GasMeterDecorator) getMessageInfo(msg commontypes.Msg) (string, string) {
	// Get message type string
	msgType := fmt.Sprintf("%T", msg)

	// Parse module and message type from string
	// This is a simplified implementation
	// In a real implementation, you would have proper type registration

	// Simple parsing - extract the last part as message name
	moduleName := "default"
	msgName := "unknown"

	if len(msgType) > 0 {
		// Try to extract meaningful information from type string
		// This is a basic implementation
		msgName = msgType

		// Check for common module patterns
		if msgType == "*bank.MsgSend" {
			moduleName = "bank"
			msgName = "MsgSend"
		} else if msgType == "*coin.MsgMint" {
			moduleName = "coin"
			msgName = "MsgMint"
		} else if msgType == "*staking.MsgDelegate" {
			moduleName = "staking"
			msgName = "MsgDelegate"
		} else if msgType == "*distribution.MsgWithdrawDelegatorReward" {
			moduleName = "distribution"
			msgName = "MsgWithdrawDelegatorReward"
		} else if msgType == "*gov.MsgVote" {
			moduleName = "gov"
			msgName = "MsgVote"
		} else if msgType == "*fee.MsgAddFeeDenom" {
			moduleName = "fee"
			msgName = "MsgAddFeeDenom"
		}
	}

	return moduleName, msgName
}

// extractMessageParameters extracts relevant parameters from message for profiling
func (gd GasMeterDecorator) extractMessageParameters(msg commontypes.Msg) map[string]interface{} {
	params := make(map[string]interface{})

	// Extract common parameters based on message type
	msgType := fmt.Sprintf("%T", msg)
	params["type"] = msgType

	// In a real implementation, you would have type-specific parameter extraction
	// This is a simplified version that just records the type

	return params
}

// updateGasMetrics updates gas consumption metrics
func (gd GasMeterDecorator) updateGasMetrics(ctx keepertypes.Context, gasUsed uint64, msgs []commontypes.Msg) {
	// Update total gas consumed
	gd.fk.IncrementTotalGasConsumed(ctx, gasUsed)

	// Update per-message type metrics
	for _, msg := range msgs {
		moduleName, messageType := gd.getMessageInfo(msg)
		gd.fk.UpdateMessageGasMetrics(ctx, moduleName, messageType, gasUsed/uint64(len(msgs)))
	}

	// Update average gas per transaction
	gd.fk.UpdateAverageGasPerTx(ctx, gasUsed)
}

// SetGasConsumedDecorator sets the gas consumed for specific message types
type SetGasConsumedDecorator struct {
	fk keeper.Keeper
}

// NewSetGasConsumedDecorator creates a new SetGasConsumedDecorator
func NewSetGasConsumedDecorator(fk keeper.Keeper) SetGasConsumedDecorator {
	return SetGasConsumedDecorator{
		fk: fk,
	}
}

// AnteHandle sets gas consumption based on message types
func (sgc SetGasConsumedDecorator) AnteHandle(ctx keepertypes.Context, tx Tx, simulate bool, next AnteHandlerFunc) (keepertypes.Context, error) {
	// Skip gas consumption in simulation
	if simulate {
		return next(ctx, tx, simulate)
	}

	// Create extended context for gas operations
	extCtx := NewExtendedContext(ctx, NewGasMeter(1000000))

	// Apply gas consumption for each message
	msgs := tx.GetMsgs()
	for i, msg := range msgs {
		gasToConsume := sgc.getGasForMessage(extCtx.(keepertypes.Context), msg)
		if gasToConsume > 0 {
			extCtx.GasMeter().ConsumeGas(
				gasToConsume,
				fmt.Sprintf("message %d gas consumption", i),
			)
		}
	}

	return next(extCtx.(keepertypes.Context), tx, simulate)
}

// getGasForMessage returns the gas to consume for a specific message
func (sgc SetGasConsumedDecorator) getGasForMessage(ctx keepertypes.Context, msg commontypes.Msg) uint64 {
	// Get message type information
	msgType := fmt.Sprintf("%T", msg)

	// Look up configured gas for this message type
	// This is a simplified lookup - in a real implementation
	// you would have proper message type registration
	gasConfig := sgc.fk.GetGasConfigForMessageType(ctx, msgType)
	if gasConfig > 0 {
		return gasConfig
	}

	// Default gas consumption
	return 10000
}

// RefundGasDecorator handles gas refunds for unused gas
type RefundGasDecorator struct {
	ak AccountKeeper
	bk BankKeeper
	fk keeper.Keeper
}

// NewRefundGasDecorator creates a new RefundGasDecorator
func NewRefundGasDecorator(ak AccountKeeper, bk BankKeeper, fk keeper.Keeper) RefundGasDecorator {
	return RefundGasDecorator{
		ak: ak,
		bk: bk,
		fk: fk,
	}
}

// AnteHandle processes gas refunds after transaction execution
func (rgd RefundGasDecorator) AnteHandle(ctx keepertypes.Context, tx Tx, simulate bool, next AnteHandlerFunc) (keepertypes.Context, error) {
	// Execute the transaction
	newCtx, err := next(ctx, tx, simulate)

	// Skip refund in simulation or on error
	if simulate || err != nil {
		return newCtx, err
	}

	// Check if refunds are enabled
	if !rgd.fk.IsGasRefundEnabled(newCtx) {
		return newCtx, err
	}

	// Calculate refund
	feeTx, ok := tx.(FeeTx)
	if !ok {
		return newCtx, err
	}

	gasLimit := feeTx.GetGas()
	extCtx := NewExtendedContext(newCtx, NewGasMeter(gasLimit))
	gasUsed := extCtx.GasMeter().GasConsumed()

	if gasUsed >= gasLimit {
		// No refund if all gas was consumed
		return newCtx, err
	}

	// Calculate refund amount
	unusedGas := gasLimit - gasUsed
	refundRatio := float64(unusedGas) / float64(gasLimit)

	fees := feeTx.GetFee()
	var refundAmount keepertypes.Coins

	for _, fee := range fees {
		refund := int64(float64(fee.Amount) * refundRatio)
		if refund > 0 {
			refundAmount = append(refundAmount, keepertypes.Coin{Denom: fee.Denom, Amount: refund})
		}
	}

	// Process refund
	if len(refundAmount) > 0 {
		feePayer := feeTx.FeePayer()
		if feeTx.FeeGranter() != nil {
			feePayer = feeTx.FeeGranter()
		}

		// Send refund from fee collector to fee payer
		err := rgd.bk.SendCoinsFromModuleToAccount(
			newCtx,
			types.FeeCollectorName,
			feePayer,
			refundAmount,
		)
		if err != nil {
			// Log refund error but don't fail the transaction
			newCtx.Logger().Error("failed to process gas refund", "error", err)
		} else {
			// Emit refund event
			refundAmountStr := ""
			for i, coin := range refundAmount {
				if i > 0 {
					refundAmountStr += ","
				}
				refundAmountStr += fmt.Sprintf("%d%s", coin.Amount, coin.Denom)
			}

			event := keepertypes.Event{
				Type: types.EventTypeGasRefund,
				Attributes: []keepertypes.Attribute{
					{Key: types.AttributeKeyRefundRecipient, Value: fmt.Sprintf("%x", feePayer)},
					{Key: types.AttributeKeyRefundAmount, Value: refundAmountStr},
					{Key: types.AttributeKeyUnusedGas, Value: fmt.Sprintf("%d", unusedGas)},
				},
			}

			newCtx.EventManager().EmitEvent(event)
		}
	}

	return newCtx, err
}