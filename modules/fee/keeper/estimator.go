package keeper

import (
	"fmt"
	"math"

	keepertypes "github.com/b2network/pulsar/keeper/types"
	feetypes "github.com/b2network/pulsar/modules/fee/types"
	commontypes "github.com/b2network/pulsar/types"
)

// GasEstimator implements gas estimation for transactions
type GasEstimator struct {
	keeper *Keeper
}

// NewGasEstimator creates a new GasEstimator
func NewGasEstimator(keeper *Keeper) feetypes.GasEstimator {
	return &GasEstimator{
		keeper: keeper,
	}
}

// EstimateGas estimates the gas consumption for a set of messages
func (ge *GasEstimator) EstimateGas(ctx keepertypes.Context, msgs []commontypes.Msg) (uint64, error) {
	if len(msgs) == 0 {
		return 0, fmt.Errorf("no messages provided")
	}

	params := ge.keeper.GetParams(ctx)
	totalGas := uint64(0)

	// Base gas cost for transaction
	baseGas := uint64(21000) // Similar to Ethereum base transaction cost

	// Add signature verification costs
	sigCost := params.SigVerifyCostSecp256k1 // Assuming secp256k1 signatures

	// Add transaction size cost
	txSize := ge.keeper.CalculateTransactionSize(msgs)
	sizeCost := txSize * params.TxSizeCostPerByte

	totalGas = baseGas + sigCost + sizeCost

	// Estimate gas for each message
	for _, msg := range msgs {
		msgGas, err := ge.estimateMessageGas(ctx, msg)
		if err != nil {
			return 0, fmt.Errorf("failed to estimate gas for message: %w", err)
		}
		totalGas += msgGas
	}

	// Apply safety margin to prevent out-of-gas errors
	safetyMargin := uint64(float64(totalGas) * 0.1) // 10% safety margin
	totalGas += safetyMargin

	// Ensure we don't exceed maximum gas wanted
	if totalGas > params.MaxGasWanted {
		totalGas = params.MaxGasWanted
	}

	return totalGas, nil
}

// EstimateDynamicGas estimates gas with dynamic factors applied
func (ge *GasEstimator) EstimateDynamicGas(ctx keepertypes.Context, msg commontypes.Msg, factors feetypes.DynamicGasFactors) (uint64, error) {
	// Get base gas estimation
	baseGas, err := ge.EstimateGas(ctx, []commontypes.Msg{msg})
	if err != nil {
		return 0, err
	}

	// Apply dynamic factors
	dynamicGas := float64(baseGas)

	// Apply size multiplier
	dynamicGas *= factors.SizeMultiplier

	// Apply complexity factor based on message type
	complexityMultiplier := ge.getComplexityMultiplier(msg)
	dynamicGas *= complexityMultiplier * factors.ComplexityFactor

	// Apply network congestion factor
	networkLoad := ge.calculateNetworkLoad(ctx)
	congestionMultiplier := 1.0 + (networkLoad * factors.NetworkFactor)
	dynamicGas *= congestionMultiplier

	// Apply storage factor for state-changing operations
	if ge.isStorageOperation(msg) {
		dynamicGas *= factors.StorageFactor
	}

	return uint64(math.Ceil(dynamicGas)), nil
}

// EstimateGasForModule estimates gas for a specific module operation
func (ge *GasEstimator) EstimateGasForModule(ctx keepertypes.Context, moduleName, msgType string, msg commontypes.Msg) (uint64, error) {
	// Get module gas configuration
	config, found := ge.keeper.GetModuleGasConfig(ctx, moduleName)
	if !found {
		// Use default estimation if no specific config found
		return ge.EstimateGas(ctx, []commontypes.Msg{msg})
	}

	if !config.Enabled {
		return 0, fmt.Errorf("gas metering disabled for module %s", moduleName)
	}

	// Get specific gas rate for message type
	msgGas := config.GetMsgGasRate(msgType)

	// Add base gas for the module
	totalGas := config.BaseGas + msgGas

	// Apply dynamic factors
	factors := ge.keeper.GetDynamicGasFactors(ctx)
	dynamicGas, err := ge.EstimateDynamicGas(ctx, msg, factors)
	if err != nil {
		// Fallback to static calculation
		return totalGas, nil
	}

	// Use the higher of static or dynamic estimation
	if dynamicGas > totalGas {
		return dynamicGas, nil
	}

	return totalGas, nil
}

// estimateMessageGas estimates gas for a single message
func (ge *GasEstimator) estimateMessageGas(ctx keepertypes.Context, msg commontypes.Msg) (uint64, error) {
	// Default gas costs based on message types
	switch msg.Type() {
	case "bank/MsgSend":
		return 30000, nil
	case "bank/MsgMultiSend":
		return 50000, nil
	case "coin/MsgMint":
		return 40000, nil
	case "coin/MsgBurn":
		return 35000, nil
	case "coin/MsgSetPermissions":
		return 25000, nil
	case "coin/MsgSetMetadata":
		return 20000, nil
	case "fee/MsgAddFeeDenom":
		return 15000, nil
	case "fee/MsgUpdateFeeDenom":
		return 12000, nil
	case "fee/MsgRemoveFeeDenom":
		return 10000, nil
	case "fee/MsgSetModuleGasConfig":
		return 18000, nil
	case "fee/MsgUpdateGasFactors":
		return 16000, nil
	case "fee/MsgSetFeeDistribution":
		return 14000, nil
	default:
		// Default gas for unknown message types
		return 25000, nil
	}
}

// getComplexityMultiplier returns a complexity multiplier based on message type
func (ge *GasEstimator) getComplexityMultiplier(msg commontypes.Msg) float64 {
	switch msg.Type() {
	case "bank/MsgMultiSend":
		return 1.5 // Multi-send is more complex
	case "coin/MsgMint", "coin/MsgBurn":
		return 1.3 // Minting/burning requires additional validation
	case "fee/MsgSetModuleGasConfig":
		return 1.2 // Configuration changes are moderately complex
	default:
		return 1.0 // Standard complexity
	}
}

// calculateNetworkLoad estimates current network congestion (0.0 - 1.0)
func (ge *GasEstimator) calculateNetworkLoad(ctx keepertypes.Context) float64 {
	// This is a simplified implementation
	// In a real system, this would analyze recent block utilization,
	// pending transaction pool size, etc.

	params := ge.keeper.GetParams(ctx)

	// For now, return a static value based on the congestion threshold
	// This could be enhanced to use actual network metrics
	return params.NetworkCongestionThreshold * 0.5 // 50% of threshold as current load
}

// isStorageOperation determines if a message involves significant storage operations
func (ge *GasEstimator) isStorageOperation(msg commontypes.Msg) bool {
	switch msg.Type() {
	case "coin/MsgSetMetadata":
		return true // Metadata storage
	case "fee/MsgAddFeeDenom":
		return true // Adding new denomination
	case "fee/MsgSetModuleGasConfig":
		return true // Configuration storage
	case "fee/MsgSetFeeDistribution":
		return true // Distribution config storage
	default:
		return false
	}
}