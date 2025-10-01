package keeper

import (
	"fmt"
	"strconv"

	keepertypes "github.com/b2network/pulsar/keeper/types"
	feetypes "github.com/b2network/pulsar/modules/fee/types"
	commontypes "github.com/b2network/pulsar/types"
)

// FeeCalculator implements fee calculation for transactions
type FeeCalculator struct {
	keeper *Keeper
}

// NewFeeCalculator creates a new FeeCalculator
func NewFeeCalculator(keeper *Keeper) feetypes.FeeCalculator {
	return &FeeCalculator{
		keeper: keeper,
	}
}

// CalculateFee calculates the total fee for a transaction
func (fc *FeeCalculator) CalculateFee(ctx keepertypes.Context, req feetypes.FeeCalculationRequest) (feetypes.FeeCalculationResponse, error) {
	if err := req.ValidateBasic(); err != nil {
		return feetypes.FeeCalculationResponse{}, fmt.Errorf("invalid fee calculation request: %w", err)
	}

	// Check if the fee denomination is valid
	if !fc.keeper.IsValidFeeDenom(ctx, req.FeeDenom) {
		return feetypes.FeeCalculationResponse{}, fmt.Errorf("fee denomination %s is not supported", req.FeeDenom)
	}

	// Get fee denomination info
	feeDenom, found := fc.keeper.GetFeeDenom(ctx, req.FeeDenom)
	if !found {
		return feetypes.FeeCalculationResponse{}, fmt.Errorf("fee denomination %s not found", req.FeeDenom)
	}

	var estimatedGas uint64
	var err error

	// Calculate gas requirement
	if req.Simulation {
		// For simulation, provide a more accurate gas estimate
		estimatedGas, err = fc.keeper.EstimateGas(ctx, req.Messages)
		if err != nil {
			return feetypes.FeeCalculationResponse{}, fmt.Errorf("failed to estimate gas: %w", err)
		}
	} else {
		// Use provided gas limit
		estimatedGas = req.GasLimit
	}

	// Get gas price
	gasPrice, _, err := feetypes.ParseGasPrice(req.GasPrice)
	if err != nil {
		return feetypes.FeeCalculationResponse{}, fmt.Errorf("invalid gas price: %w", err)
	}

	// Calculate total fee amount
	totalFeeAmount := gasPrice * float64(estimatedGas)

	// Get recommended gas price
	recommendedGasPrice, err := fc.keeper.GetRecommendedGasPrice(ctx, req.FeeDenom)
	if err != nil {
		// Fallback to minimum gas price if recommendation fails
		recommendedGasPrice = feeDenom.MinGasPrice
	}

	// Create breakdown by module
	breakdown, err := fc.calculateGasBreakdown(ctx, req.Messages)
	if err != nil {
		// Continue without breakdown if calculation fails
		breakdown = []feetypes.GasBreakdown{}
	}

	// Format fee amount as string with appropriate precision
	feeAmountStr := fmt.Sprintf("%.0f", totalFeeAmount)

	response := feetypes.FeeCalculationResponse{
		EstimatedGas:           estimatedGas,
		MinGasPrice:           feeDenom.MinGasPrice,
		RecommendedGasPrice:   recommendedGasPrice,
		TotalFee:              commontypes.NewCoin(req.FeeDenom, feeAmountStr),
		BreakdownByModule:     breakdown,
	}

	return response, nil
}

// ValidateFee validates if the provided fee is sufficient
func (fc *FeeCalculator) ValidateFee(ctx keepertypes.Context, msgs []commontypes.Msg, fee commontypes.Coin, gasLimit uint64) error {
	// Check if fee denomination is supported
	if !fc.keeper.IsValidFeeDenom(ctx, fee.Denom) {
		return fmt.Errorf("fee denomination %s is not supported", fee.Denom)
	}

	// Get minimum fee required
	minFee, err := fc.GetMinimumFee(ctx, msgs, gasLimit, fee.Denom)
	if err != nil {
		return fmt.Errorf("failed to calculate minimum fee: %w", err)
	}

	// Convert fee amounts to float64 for comparison
	providedAmount, err := strconv.ParseFloat(fee.Amount, 64)
	if err != nil {
		return fmt.Errorf("invalid fee amount format: %w", err)
	}

	minAmount, err := strconv.ParseFloat(minFee.Amount, 64)
	if err != nil {
		return fmt.Errorf("invalid minimum fee amount format: %w", err)
	}

	// Check if provided fee is sufficient
	if providedAmount < minAmount {
		return fmt.Errorf("insufficient fee: provided %s%s, required at least %s%s",
			fee.Amount, fee.Denom, minFee.Amount, minFee.Denom)
	}

	return nil
}

// GetMinimumFee returns the minimum fee required for a transaction
func (fc *FeeCalculator) GetMinimumFee(ctx keepertypes.Context, msgs []commontypes.Msg, gasLimit uint64, feeDenom string) (commontypes.Coin, error) {
	// Check if fee denomination is supported
	feeDenomInfo, found := fc.keeper.GetFeeDenom(ctx, feeDenom)
	if !found {
		return commontypes.Coin{}, fmt.Errorf("fee denomination %s not found", feeDenom)
	}

	if !feeDenomInfo.IsEnabled() {
		return commontypes.Coin{}, fmt.Errorf("fee denomination %s is disabled", feeDenom)
	}

	// Parse minimum gas price
	minGasPrice, _, err := feetypes.ParseGasPrice(feeDenomInfo.MinGasPrice)
	if err != nil {
		return commontypes.Coin{}, fmt.Errorf("invalid minimum gas price: %w", err)
	}

	// Estimate gas if not provided
	var estimatedGas uint64
	if gasLimit == 0 {
		estimatedGas, err = fc.keeper.EstimateGas(ctx, msgs)
		if err != nil {
			return commontypes.Coin{}, fmt.Errorf("failed to estimate gas: %w", err)
		}
	} else {
		estimatedGas = gasLimit
	}

	// Calculate minimum fee
	minFeeAmount := minGasPrice * float64(estimatedGas)
	minFeeAmountStr := fmt.Sprintf("%.0f", minFeeAmount)

	return commontypes.NewCoin(feeDenom, minFeeAmountStr), nil
}

// calculateGasBreakdown calculates gas usage breakdown by module/operation
func (fc *FeeCalculator) calculateGasBreakdown(ctx keepertypes.Context, msgs []commontypes.Msg) ([]feetypes.GasBreakdown, error) {
	var breakdown []feetypes.GasBreakdown

	for _, msg := range msgs {
		// Extract module name from message route
		route := msg.Route()
		msgType := msg.Type()

		// Estimate base gas for this specific message
		baseGas, err := fc.keeper.gasEstimator.EstimateGasForModule(ctx, route, msgType, msg)
		if err != nil {
			// Use default estimation if module-specific fails
			msgs := []commontypes.Msg{msg}
			totalGas, err := fc.keeper.EstimateGas(ctx, msgs)
			if err != nil {
				continue // Skip this message if we can't estimate
			}
			baseGas = totalGas
		}

		// Estimate dynamic gas
		factors := fc.keeper.GetDynamicGasFactors(ctx)
		dynamicGas, err := fc.keeper.gasEstimator.EstimateDynamicGas(ctx, msg, factors)
		if err != nil {
			dynamicGas = 0 // No dynamic gas if estimation fails
		}

		// Calculate total gas for this message
		totalGas := baseGas
		if dynamicGas > baseGas {
			totalGas = dynamicGas
		}

		breakdown = append(breakdown, feetypes.GasBreakdown{
			ModuleName: route,
			Operation:  msgType,
			BaseGas:    baseGas,
			DynamicGas: dynamicGas - baseGas,
			TotalGas:   totalGas,
		})
	}

	return breakdown, nil
}

// CalculateFeeForGasAndPrice calculates fee for given gas and price
func (fc *FeeCalculator) CalculateFeeForGasAndPrice(gasUsed uint64, gasPrice, feeDenom string) (commontypes.Coin, error) {
	// Parse gas price
	price, denom, err := feetypes.ParseGasPrice(gasPrice)
	if err != nil {
		return commontypes.Coin{}, fmt.Errorf("invalid gas price: %w", err)
	}

	// Ensure denomination matches
	if denom != feeDenom {
		return commontypes.Coin{}, fmt.Errorf("gas price denomination %s does not match fee denomination %s", denom, feeDenom)
	}

	// Calculate fee amount
	feeAmount := price * float64(gasUsed)
	feeAmountStr := fmt.Sprintf("%.0f", feeAmount)

	return commontypes.NewCoin(feeDenom, feeAmountStr), nil
}

// GetEffectiveGasPrice returns the effective gas price considering network conditions
func (fc *FeeCalculator) GetEffectiveGasPrice(ctx keepertypes.Context, baseGasPrice string, feeDenom string) (string, error) {
	// Parse base gas price
	basePrice, _, err := feetypes.ParseGasPrice(baseGasPrice)
	if err != nil {
		return "", fmt.Errorf("invalid base gas price: %w", err)
	}

	// Get network congestion factor
	params := fc.keeper.GetParams(ctx)

	// Apply congestion multiplier (simplified implementation)
	// In a real system, this would analyze current network conditions
	congestionMultiplier := 1.0
	if basePrice > 0 {
		// Apply a small congestion factor
		congestionMultiplier = 1.0 + (params.NetworkCongestionThreshold * 0.1)
	}

	effectivePrice := basePrice * congestionMultiplier

	// Ensure effective price doesn't exceed maximum multiplier
	maxPrice := basePrice * params.MaxGasPriceMultiplier
	if effectivePrice > maxPrice {
		effectivePrice = maxPrice
	}

	// Ensure effective price meets minimum multiplier
	minPrice := basePrice * params.MinGasPriceMultiplier
	if effectivePrice < minPrice {
		effectivePrice = minPrice
	}

	return feetypes.FormatGasPrice(effectivePrice, feeDenom), nil
}