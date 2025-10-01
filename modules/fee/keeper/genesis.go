package keeper

import (
	"encoding/json"
	"fmt"

	keepertypes "github.com/b2network/pulsar/keeper/types"
	feetypes "github.com/b2network/pulsar/modules/fee/types"
	storetypes "github.com/b2network/pulsar/store/types"
)

// InitGenesis initializes the fee module's state from genesis data
func (k Keeper) InitGenesis(ctx keepertypes.Context, genState feetypes.GenesisState) {
	// Set module parameters
	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(fmt.Sprintf("failed to set fee params during genesis: %s", err))
	}

	// Initialize fee denominations
	for _, feeDenom := range genState.FeeDenoms {
		if err := k.AddFeeDenom(ctx, feeDenom); err != nil {
			// Log warning but don't panic for duplicate denominations during genesis
			ctx.Logger().Error("failed to add fee denomination during genesis",
				"denom", feeDenom.Denom, "error", err)
		}
	}

	// Initialize module gas configurations
	for _, config := range genState.ModuleGasConfigs {
		if err := k.SetModuleGasConfig(ctx, config); err != nil {
			ctx.Logger().Error("failed to set module gas config during genesis",
				"module", config.ModuleName, "error", err)
		}
	}

	// Set dynamic gas factors
	if err := k.SetDynamicGasFactors(ctx, genState.DynamicGasFactors); err != nil {
		panic(fmt.Sprintf("failed to set dynamic gas factors during genesis: %s", err))
	}

	// Initialize gas price statistics
	for _, stats := range genState.GasPriceStats {
		if err := k.SetGasPriceStats(ctx, stats); err != nil {
			ctx.Logger().Error("failed to set gas price stats during genesis",
				"denom", stats.Denom, "error", err)
		}
	}

	// Set fee distribution configuration (we'll store this as module parameters)
	k.setFeeDistributionConfig(ctx, genState.FeeDistributionConfig)

	// Log successful initialization
	ctx.Logger().Info("fee module genesis initialized",
		"fee_denoms", len(genState.FeeDenoms),
		"module_configs", len(genState.ModuleGasConfigs),
		"gas_price_stats", len(genState.GasPriceStats))
}

// ExportGenesis exports the fee module's state to genesis data
func (k Keeper) ExportGenesis(ctx keepertypes.Context) *feetypes.GenesisState {
	return &feetypes.GenesisState{
		Params:                k.GetParams(ctx),
		FeeDenoms:             k.GetAllFeeDenoms(ctx),
		ModuleGasConfigs:      k.GetAllModuleGasConfigs(ctx),
		DynamicGasFactors:     k.GetDynamicGasFactors(ctx),
		FeeDistributionConfig: k.getFeeDistributionConfig(ctx),
		GasPriceStats:         k.getAllGasPriceStats(ctx),
	}
}

// setFeeDistributionConfig stores the fee distribution configuration
func (k Keeper) setFeeDistributionConfig(ctx keepertypes.Context, config feetypes.FeeDistributionConfig) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(config)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal fee distribution config: %s", err))
	}
	store.Set(feetypes.FeeDistributionConfigKey, bz)
}

// getFeeDistributionConfig retrieves the fee distribution configuration
func (k Keeper) getFeeDistributionConfig(ctx keepertypes.Context) feetypes.FeeDistributionConfig {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(feetypes.FeeDistributionConfigKey)
	if bz == nil {
		return feetypes.DefaultFeeDistributionConfig()
	}

	var config feetypes.FeeDistributionConfig
	if err := json.Unmarshal(bz, &config); err != nil {
		return feetypes.DefaultFeeDistributionConfig()
	}
	return config
}

// getAllGasPriceStats retrieves all gas price statistics
func (k Keeper) getAllGasPriceStats(ctx keepertypes.Context) []feetypes.GasPriceStats {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, feetypes.GasPriceStatsKey)
	defer iterator.Close()

	var allStats []feetypes.GasPriceStats
	for ; iterator.Valid(); iterator.Next() {
		var stats feetypes.GasPriceStats
		if err := json.Unmarshal(iterator.Value(), &stats); err != nil {
			continue
		}
		allStats = append(allStats, stats)
	}

	return allStats
}

// ValidateGasConfig validates gas configuration during runtime
func (k Keeper) ValidateGasConfig(ctx keepertypes.Context, moduleName string) error {
	config, found := k.GetModuleGasConfig(ctx, moduleName)
	if !found {
		return fmt.Errorf("no gas configuration found for module %s", moduleName)
	}

	if !config.Enabled {
		return fmt.Errorf("gas metering disabled for module %s", moduleName)
	}

	return config.ValidateBasic()
}

// UpdateGasConfigForModule updates gas configuration for a specific module
func (k Keeper) UpdateGasConfigForModule(ctx keepertypes.Context, moduleName string, updates map[string]uint64) error {
	config, found := k.GetModuleGasConfig(ctx, moduleName)
	if !found {
		return fmt.Errorf("module %s not found in gas configuration", moduleName)
	}

	// Apply updates
	for msgType, gasRate := range updates {
		config.SetMsgGasRate(msgType, gasRate)
	}

	// Validate and save
	if err := config.ValidateBasic(); err != nil {
		return fmt.Errorf("invalid gas configuration after update: %w", err)
	}

	return k.SetModuleGasConfig(ctx, config)
}

// GetEffectiveGasForMessage returns the effective gas cost for a specific message
func (k Keeper) GetEffectiveGasForMessage(ctx keepertypes.Context, moduleName, msgType string) (uint64, error) {
	config, found := k.GetModuleGasConfig(ctx, moduleName)
	if !found {
		return 0, fmt.Errorf("no gas configuration found for module %s", moduleName)
	}

	if !config.Enabled {
		return 0, fmt.Errorf("gas metering disabled for module %s", moduleName)
	}

	baseGas := config.GetMsgGasRate(msgType)

	// Apply dynamic gas factors
	factors := k.GetDynamicGasFactors(ctx)

	// Calculate effective gas with complexity factor
	effectiveGas := float64(baseGas) * factors.ComplexityFactor

	return uint64(effectiveGas), nil
}

// InitializeDefaultGasConfigs initializes default gas configurations for all modules
func (k Keeper) InitializeDefaultGasConfigs(ctx keepertypes.Context) error {
	defaultConfigs := feetypes.DefaultModuleGasConfigs()

	for _, config := range defaultConfigs {
		if err := k.SetModuleGasConfig(ctx, config); err != nil {
			return fmt.Errorf("failed to initialize gas config for module %s: %w", config.ModuleName, err)
		}
	}

	ctx.Logger().Info("initialized default gas configurations",
		"modules", len(defaultConfigs))

	return nil
}

// AutoTuneGasFactors automatically adjusts gas factors based on network conditions
func (k Keeper) AutoTuneGasFactors(ctx keepertypes.Context) error {
	params := k.GetParams(ctx)
	currentFactors := k.GetDynamicGasFactors(ctx)

	// Calculate network congestion level (simplified)
	congestionLevel := k.calculateNetworkCongestion(ctx)

	// Adjust network factor based on congestion
	newNetworkFactor := 1.0 + (congestionLevel * 0.5) // Max 50% increase

	// Apply smoothing to prevent rapid changes
	smoothingFactor := 0.1 // 10% weight to new value
	adjustedNetworkFactor := currentFactors.NetworkFactor*(1-smoothingFactor) + newNetworkFactor*smoothingFactor

	// Update factors if change is significant
	if abs(adjustedNetworkFactor-currentFactors.NetworkFactor) > 0.05 { // 5% threshold
		newFactors := currentFactors
		newFactors.NetworkFactor = adjustedNetworkFactor

		// Ensure factors stay within reasonable bounds
		if newFactors.NetworkFactor > params.MaxGasPriceMultiplier {
			newFactors.NetworkFactor = params.MaxGasPriceMultiplier
		}
		if newFactors.NetworkFactor < params.MinGasPriceMultiplier {
			newFactors.NetworkFactor = params.MinGasPriceMultiplier
		}

		if err := k.SetDynamicGasFactors(ctx, newFactors); err != nil {
			return fmt.Errorf("failed to update dynamic gas factors: %w", err)
		}

		ctx.Logger().Info("auto-tuned gas factors",
			"old_network_factor", currentFactors.NetworkFactor,
			"new_network_factor", newFactors.NetworkFactor,
			"congestion_level", congestionLevel)
	}

	return nil
}

// calculateNetworkCongestion calculates current network congestion (0.0 - 1.0)
func (k Keeper) calculateNetworkCongestion(ctx keepertypes.Context) float64 {
	// This is a simplified implementation
	// In a real system, this would analyze:
	// - Block gas usage vs gas limit
	// - Transaction pool size
	// - Recent block confirmation times
	// - Number of pending transactions

	params := k.GetParams(ctx)
	return params.NetworkCongestionThreshold * 0.6 // 60% of threshold as current congestion
}

// abs returns the absolute value of a float64
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}