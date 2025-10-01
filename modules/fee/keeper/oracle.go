package keeper

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	keepertypes "github.com/b2network/pulsar/keeper/types"
	feetypes "github.com/b2network/pulsar/modules/fee/types"
	commontypes "github.com/b2network/pulsar/types"
)

// GasPriceOracle implements gas price oracle functionality
type GasPriceOracle struct {
	keeper *Keeper
}

// NewGasPriceOracle creates a new GasPriceOracle
func NewGasPriceOracle(keeper *Keeper) feetypes.GasPriceOracle {
	return &GasPriceOracle{
		keeper: keeper,
	}
}

// UpdateGasPrices updates gas price statistics based on recent transactions
func (gpo *GasPriceOracle) UpdateGasPrices(ctx keepertypes.Context) error {
	params := gpo.keeper.GetParams(ctx)

	// Get all enabled fee denominations
	feeDenoms := gpo.keeper.GetAllFeeDenoms(ctx)

	for _, feeDenom := range feeDenoms {
		if !feeDenom.IsEnabled() {
			continue
		}

		// Update gas price statistics for this denomination
		if err := gpo.updateGasPriceForDenom(ctx, feeDenom.Denom); err != nil {
			// Log error but continue with other denominations
			ctx.Logger().Error("failed to update gas price for denomination",
				"denom", feeDenom.Denom, "error", err)
			continue
		}
	}

	// Emit gas price update event
	ctx.EventManager().EmitEvent(convertEvent(commontypes.NewEvent(
		"gas_prices_updated",
		commontypes.NewAttribute("update_time", fmt.Sprintf("%d", ctx.BlockTime().Unix())),
		commontypes.NewAttribute("interval", params.GasPriceUpdateInterval.String()),
	)))

	return nil
}

// GetRecommendedGasPrice returns the recommended gas price for a denomination
func (gpo *GasPriceOracle) GetRecommendedGasPrice(ctx keepertypes.Context, denom string) (string, error) {
	// Get current gas price statistics
	stats, found := gpo.keeper.GetGasPriceStats(ctx, denom)
	if !found {
		// If no statistics available, use minimum gas price from fee denomination
		feeDenom, found := gpo.keeper.GetFeeDenom(ctx, denom)
		if !found {
			return "", fmt.Errorf("denomination %s not found", denom)
		}
		return feeDenom.MinGasPrice, nil
	}

	// Return the recommended price from statistics
	if stats.RecommendedPrice != "" {
		return stats.RecommendedPrice, nil
	}

	// Fallback to average price if recommended price is not available
	if stats.AveragePrice != "" {
		return stats.AveragePrice, nil
	}

	// Final fallback to minimum gas price
	feeDenom, found := gpo.keeper.GetFeeDenom(ctx, denom)
	if !found {
		return "", fmt.Errorf("denomination %s not found", denom)
	}

	return feeDenom.MinGasPrice, nil
}

// GetGasPriceStats returns gas price statistics for a denomination
func (gpo *GasPriceOracle) GetGasPriceStats(ctx keepertypes.Context, denom string) (feetypes.GasPriceStats, error) {
	stats, found := gpo.keeper.GetGasPriceStats(ctx, denom)
	if !found {
		return feetypes.GasPriceStats{}, fmt.Errorf("gas price statistics not found for denomination %s", denom)
	}

	return stats, nil
}

// updateGasPriceForDenom updates gas price statistics for a specific denomination
func (gpo *GasPriceOracle) updateGasPriceForDenom(ctx keepertypes.Context, denom string) error {
	// Get fee denomination info
	feeDenom, found := gpo.keeper.GetFeeDenom(ctx, denom)
	if !found {
		return fmt.Errorf("fee denomination %s not found", denom)
	}

	// In a real implementation, this would collect gas prices from recent transactions
	// For now, we'll simulate gas price data based on network conditions
	gasPrices := gpo.simulateRecentGasPrices(ctx, denom, feeDenom.MinGasPrice)

	// Calculate statistics
	stats, err := gpo.calculateGasPriceStats(denom, gasPrices)
	if err != nil {
		return fmt.Errorf("failed to calculate gas price statistics: %w", err)
	}

	// Store updated statistics
	return gpo.keeper.SetGasPriceStats(ctx, stats)
}

// simulateRecentGasPrices simulates recent gas prices for testing purposes
// In a real implementation, this would query actual transaction data
func (gpo *GasPriceOracle) simulateRecentGasPrices(ctx keepertypes.Context, denom, minGasPrice string) []float64 {
	params := gpo.keeper.GetParams(ctx)

	// Parse minimum gas price
	minPrice, err := strconv.ParseFloat(minGasPrice, 64)
	if err != nil {
		minPrice = 0.01 // Default minimum
	}

	// Generate simulated gas prices based on network congestion
	var gasPrices []float64

	// Simulate 50 recent transactions
	sampleSize := 50
	basePrice := minPrice

	// Apply network congestion factor
	congestionFactor := params.NetworkCongestionThreshold
	averagePrice := basePrice * (1.0 + congestionFactor)

	// Generate price samples with some variance
	for i := 0; i < sampleSize; i++ {
		// Add random variance (+/- 20%)
		variance := 0.8 + (float64(i%41))/100.0 // Simple pseudo-random variance
		price := averagePrice * variance

		// Ensure price doesn't go below minimum
		if price < minPrice {
			price = minPrice
		}

		// Apply maximum multiplier constraint
		maxPrice := basePrice * params.MaxGasPriceMultiplier
		if price > maxPrice {
			price = maxPrice
		}

		gasPrices = append(gasPrices, price)
	}

	return gasPrices
}

// calculateGasPriceStats calculates statistics from a set of gas prices
func (gpo *GasPriceOracle) calculateGasPriceStats(denom string, gasPrices []float64) (feetypes.GasPriceStats, error) {
	if len(gasPrices) == 0 {
		return feetypes.GasPriceStats{}, fmt.Errorf("no gas prices provided")
	}

	// Sort prices for median calculation
	sortedPrices := make([]float64, len(gasPrices))
	copy(sortedPrices, gasPrices)
	sort.Float64s(sortedPrices)

	// Calculate statistics
	minPrice := sortedPrices[0]
	maxPrice := sortedPrices[len(sortedPrices)-1]

	// Calculate average
	sum := 0.0
	for _, price := range gasPrices {
		sum += price
	}
	averagePrice := sum / float64(len(gasPrices))

	// Calculate median
	var medianPrice float64
	mid := len(sortedPrices) / 2
	if len(sortedPrices)%2 == 0 {
		medianPrice = (sortedPrices[mid-1] + sortedPrices[mid]) / 2
	} else {
		medianPrice = sortedPrices[mid]
	}

	// Calculate recommended price (e.g., 75th percentile for faster confirmation)
	percentile75Index := int(float64(len(sortedPrices)) * 0.75)
	if percentile75Index >= len(sortedPrices) {
		percentile75Index = len(sortedPrices) - 1
	}
	recommendedPrice := sortedPrices[percentile75Index]

	// Create gas price statistics
	stats := feetypes.GasPriceStats{
		Denom:            denom,
		MinPrice:         feetypes.FormatGasPrice(minPrice, denom),
		MaxPrice:         feetypes.FormatGasPrice(maxPrice, denom),
		AveragePrice:     feetypes.FormatGasPrice(averagePrice, denom),
		MedianPrice:      feetypes.FormatGasPrice(medianPrice, denom),
		RecommendedPrice: feetypes.FormatGasPrice(recommendedPrice, denom),
		UpdateTime:       time.Now().Unix(),
		SampleSize:       uint64(len(gasPrices)),
	}

	return stats, nil
}

// GetCurrentNetworkCongestion returns current network congestion level (0.0 - 1.0)
func (gpo *GasPriceOracle) GetCurrentNetworkCongestion(ctx keepertypes.Context) float64 {
	// In a real implementation, this would analyze:
	// - Current block gas usage vs gas limit
	// - Transaction pool size
	// - Recent block confirmation times
	// - Validator response times

	params := gpo.keeper.GetParams(ctx)

	// For simulation, return a value based on configuration
	// This could be enhanced with actual network metrics
	return params.NetworkCongestionThreshold * 0.6 // 60% of threshold
}

// ShouldUpdateGasPrices determines if gas prices should be updated
func (gpo *GasPriceOracle) ShouldUpdateGasPrices(ctx keepertypes.Context) bool {
	params := gpo.keeper.GetParams(ctx)

	// Get all fee denominations to check last update time
	feeDenoms := gpo.keeper.GetAllFeeDenoms(ctx)

	for _, feeDenom := range feeDenoms {
		if !feeDenom.IsEnabled() {
			continue
		}

		stats, found := gpo.keeper.GetGasPriceStats(ctx, feeDenom.Denom)
		if !found {
			// If no statistics exist, we should update
			return true
		}

		// Check if enough time has passed since last update
		lastUpdate := time.Unix(stats.UpdateTime, 0)
		timeSinceUpdate := ctx.BlockTime().Sub(lastUpdate)

		if timeSinceUpdate >= params.GasPriceUpdateInterval {
			return true
		}
	}

	return false
}

// GetFastGasPrice returns a gas price for fast transaction confirmation
func (gpo *GasPriceOracle) GetFastGasPrice(ctx keepertypes.Context, denom string) (string, error) {
	stats, found := gpo.keeper.GetGasPriceStats(ctx, denom)
	if !found {
		// Fallback to recommended price
		return gpo.GetRecommendedGasPrice(ctx, denom)
	}

	// Use the 90th percentile for fast confirmation
	if stats.MaxPrice != "" {
		maxPrice, _, err := feetypes.ParseGasPrice(stats.MaxPrice)
		if err == nil {
			fastPrice := maxPrice * 0.9 // 90% of max price
			return feetypes.FormatGasPrice(fastPrice, denom), nil
		}
	}

	// Fallback to recommended price
	return gpo.GetRecommendedGasPrice(ctx, denom)
}

// GetSlowGasPrice returns a gas price for economical transaction confirmation
func (gpo *GasPriceOracle) GetSlowGasPrice(ctx keepertypes.Context, denom string) (string, error) {
	stats, found := gpo.keeper.GetGasPriceStats(ctx, denom)
	if !found {
		// Get minimum price from fee denomination
		feeDenom, found := gpo.keeper.GetFeeDenom(ctx, denom)
		if !found {
			return "", fmt.Errorf("denomination %s not found", denom)
		}
		return feeDenom.MinGasPrice, nil
	}

	// Use the 25th percentile for slow/economical confirmation
	if stats.MinPrice != "" && stats.AveragePrice != "" {
		minPrice, _, err1 := feetypes.ParseGasPrice(stats.MinPrice)
		avgPrice, _, err2 := feetypes.ParseGasPrice(stats.AveragePrice)

		if err1 == nil && err2 == nil {
			slowPrice := minPrice + (avgPrice-minPrice)*0.25 // 25th percentile approximation
			return feetypes.FormatGasPrice(slowPrice, denom), nil
		}
	}

	// Fallback to minimum price
	feeDenom, found := gpo.keeper.GetFeeDenom(ctx, denom)
	if !found {
		return "", fmt.Errorf("denomination %s not found", denom)
	}
	return feeDenom.MinGasPrice, nil
}

// ShouldUpdateGasPrices determines if gas prices should be updated
func (k Keeper) ShouldUpdateGasPrices(ctx keepertypes.Context) bool {
	oracle := k.gasPriceOracle.(*GasPriceOracle)
	return oracle.ShouldUpdateGasPrices(ctx)
}

