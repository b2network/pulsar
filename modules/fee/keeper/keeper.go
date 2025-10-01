package keeper

import (
	"encoding/json"
	"fmt"
	"strconv"

	keepertypes "github.com/b2network/pulsar/keeper/types"
	feetypes "github.com/b2network/pulsar/modules/fee/types"
	storetypes "github.com/b2network/pulsar/store/types"
	commontypes "github.com/b2network/pulsar/types"
)

// Keeper manages fee configuration and calculation
type Keeper struct {
	storeKey   storetypes.StoreKey
	memKey     storetypes.StoreKey
	authority  string // Authority address for governance

	accountKeeper     feetypes.AccountKeeper
	bankKeeper        feetypes.BankKeeper
	stakingKeeper     feetypes.StakingKeeper
	distributionKeeper feetypes.DistributionKeeper

	// Internal state
	params          feetypes.Params
	gasEstimator    feetypes.GasEstimator
	feeCalculator   feetypes.FeeCalculator
	feeCollector    feetypes.FeeCollector
	gasPriceOracle  feetypes.GasPriceOracle
}

// NewKeeper creates a new fee Keeper
func NewKeeper(
	storeKey, memKey storetypes.StoreKey,
	authority string,
	accountKeeper feetypes.AccountKeeper,
	bankKeeper feetypes.BankKeeper,
	stakingKeeper feetypes.StakingKeeper,
	distributionKeeper feetypes.DistributionKeeper,
) *Keeper {
	k := &Keeper{
		storeKey:           storeKey,
		memKey:             memKey,
		authority:          authority,
		accountKeeper:      accountKeeper,
		bankKeeper:         bankKeeper,
		stakingKeeper:      stakingKeeper,
		distributionKeeper: distributionKeeper,
		params:             feetypes.DefaultParams(),
	}

	// Initialize sub-components
	k.gasEstimator = NewGasEstimator(k)
	k.feeCalculator = NewFeeCalculator(k)
	k.feeCollector = NewFeeCollector(k)
	k.gasPriceOracle = NewGasPriceOracle(k)

	return k
}

// GetAuthority returns the authority address
func (k Keeper) GetAuthority() string {
	return k.authority
}

// GetParams returns the module parameters
func (k Keeper) GetParams(ctx keepertypes.Context) feetypes.Params {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(feetypes.ParamsKey)
	if bz == nil {
		return feetypes.DefaultParams()
	}

	var params feetypes.Params
	if err := json.Unmarshal(bz, &params); err != nil {
		return feetypes.DefaultParams()
	}
	return params
}

// SetParams sets the module parameters
func (k Keeper) SetParams(ctx keepertypes.Context, params feetypes.Params) error {
	if err := params.ValidateBasic(); err != nil {
		return err
	}

	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(params)
	if err != nil {
		return err
	}

	store.Set(feetypes.ParamsKey, bz)
	k.params = params
	return nil
}

// Fee Denomination Management

// AddFeeDenom adds a new fee denomination
func (k Keeper) AddFeeDenom(ctx keepertypes.Context, feeDenom feetypes.FeeDenom) error {
	if err := feeDenom.ValidateBasic(); err != nil {
		return err
	}

	store := ctx.KVStore(k.storeKey)
	key := feetypes.GetFeeDenomKey(feeDenom.Denom)

	// Check if already exists
	if store.Has(key) {
		return fmt.Errorf("fee denomination %s already exists", feeDenom.Denom)
	}

	bz, err := json.Marshal(feeDenom)
	if err != nil {
		return err
	}

	store.Set(key, bz)

	// Emit event
	ctx.EventManager().EmitEvent(convertEvent(commontypes.NewEvent(
		"fee_denom_added",
		commontypes.NewAttribute("denom", feeDenom.Denom),
		commontypes.NewAttribute("min_gas_price", feeDenom.MinGasPrice),
		commontypes.NewAttribute("enabled", fmt.Sprintf("%t", feeDenom.Enabled)),
	)))

	return nil
}

// RemoveFeeDenom removes a fee denomination
func (k Keeper) RemoveFeeDenom(ctx keepertypes.Context, denom string) error {
	store := ctx.KVStore(k.storeKey)
	key := feetypes.GetFeeDenomKey(denom)

	if !store.Has(key) {
		return fmt.Errorf("fee denomination %s does not exist", denom)
	}

	store.Delete(key)

	// Emit event
	ctx.EventManager().EmitEvent(convertEvent(commontypes.NewEvent(
		"fee_denom_removed",
		commontypes.NewAttribute("denom", denom),
	)))

	return nil
}

// UpdateFeeDenom updates a fee denomination
func (k Keeper) UpdateFeeDenom(ctx keepertypes.Context, feeDenom feetypes.FeeDenom) error {
	if err := feeDenom.ValidateBasic(); err != nil {
		return err
	}

	store := ctx.KVStore(k.storeKey)
	key := feetypes.GetFeeDenomKey(feeDenom.Denom)

	if !store.Has(key) {
		return fmt.Errorf("fee denomination %s does not exist", feeDenom.Denom)
	}

	bz, err := json.Marshal(feeDenom)
	if err != nil {
		return err
	}

	store.Set(key, bz)

	// Emit event
	ctx.EventManager().EmitEvent(convertEvent(commontypes.NewEvent(
		"fee_denom_updated",
		commontypes.NewAttribute("denom", feeDenom.Denom),
		commontypes.NewAttribute("min_gas_price", feeDenom.MinGasPrice),
		commontypes.NewAttribute("enabled", fmt.Sprintf("%t", feeDenom.Enabled)),
	)))

	return nil
}

// GetFeeDenom returns a fee denomination
func (k Keeper) GetFeeDenom(ctx keepertypes.Context, denom string) (feetypes.FeeDenom, bool) {
	store := ctx.KVStore(k.storeKey)
	key := feetypes.GetFeeDenomKey(denom)

	bz := store.Get(key)
	if bz == nil {
		return feetypes.FeeDenom{}, false
	}

	var feeDenom feetypes.FeeDenom
	if err := json.Unmarshal(bz, &feeDenom); err != nil {
		return feetypes.FeeDenom{}, false
	}

	return feeDenom, true
}

// GetAllFeeDenoms returns all fee denominations
func (k Keeper) GetAllFeeDenoms(ctx keepertypes.Context) []feetypes.FeeDenom {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, feetypes.FeeDenomPrefix)
	defer iterator.Close()

	var feeDenoms []feetypes.FeeDenom
	for ; iterator.Valid(); iterator.Next() {
		var feeDenom feetypes.FeeDenom
		if err := json.Unmarshal(iterator.Value(), &feeDenom); err != nil {
			continue
		}
		feeDenoms = append(feeDenoms, feeDenom)
	}

	return feeDenoms
}

// IsValidFeeDenom checks if a denomination can be used for fee payment
func (k Keeper) IsValidFeeDenom(ctx keepertypes.Context, denom string) bool {
	feeDenom, found := k.GetFeeDenom(ctx, denom)
	return found && feeDenom.IsEnabled()
}

// Module Gas Configuration Management

// SetModuleGasConfig sets gas configuration for a module
func (k Keeper) SetModuleGasConfig(ctx keepertypes.Context, config feetypes.ModuleGasConfig) error {
	if err := config.ValidateBasic(); err != nil {
		return err
	}

	store := ctx.KVStore(k.storeKey)
	key := feetypes.GetModuleGasConfigKey(config.ModuleName)

	bz, err := json.Marshal(config)
	if err != nil {
		return err
	}

	store.Set(key, bz)

	// Emit event
	ctx.EventManager().EmitEvent(convertEvent(commontypes.NewEvent(
		"module_gas_config_set",
		commontypes.NewAttribute("module", config.ModuleName),
		commontypes.NewAttribute("base_gas", fmt.Sprintf("%d", config.BaseGas)),
		commontypes.NewAttribute("enabled", fmt.Sprintf("%t", config.Enabled)),
	)))

	return nil
}

// GetModuleGasConfig returns gas configuration for a module
func (k Keeper) GetModuleGasConfig(ctx keepertypes.Context, moduleName string) (feetypes.ModuleGasConfig, bool) {
	store := ctx.KVStore(k.storeKey)
	key := feetypes.GetModuleGasConfigKey(moduleName)

	bz := store.Get(key)
	if bz == nil {
		return feetypes.ModuleGasConfig{}, false
	}

	var config feetypes.ModuleGasConfig
	if err := json.Unmarshal(bz, &config); err != nil {
		return feetypes.ModuleGasConfig{}, false
	}

	return config, true
}

// GetAllModuleGasConfigs returns all module gas configurations
func (k Keeper) GetAllModuleGasConfigs(ctx keepertypes.Context) []feetypes.ModuleGasConfig {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, feetypes.ModuleGasConfigPrefix)
	defer iterator.Close()

	var configs []feetypes.ModuleGasConfig
	for ; iterator.Valid(); iterator.Next() {
		var config feetypes.ModuleGasConfig
		if err := json.Unmarshal(iterator.Value(), &config); err != nil {
			continue
		}
		configs = append(configs, config)
	}

	return configs
}

// Dynamic Gas Factors Management

// SetDynamicGasFactors sets the dynamic gas factors
func (k Keeper) SetDynamicGasFactors(ctx keepertypes.Context, factors feetypes.DynamicGasFactors) error {
	if err := factors.ValidateBasic(); err != nil {
		return err
	}

	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(factors)
	if err != nil {
		return err
	}

	store.Set(feetypes.DynamicGasFactorsKey, bz)

	// Emit event
	ctx.EventManager().EmitEvent(convertEvent(commontypes.NewEvent(
		"dynamic_gas_factors_updated",
		commontypes.NewAttribute("size_multiplier", fmt.Sprintf("%.2f", factors.SizeMultiplier)),
		commontypes.NewAttribute("complexity_factor", fmt.Sprintf("%.2f", factors.ComplexityFactor)),
		commontypes.NewAttribute("network_factor", fmt.Sprintf("%.2f", factors.NetworkFactor)),
		commontypes.NewAttribute("storage_factor", fmt.Sprintf("%.2f", factors.StorageFactor)),
	)))

	return nil
}

// GetDynamicGasFactors returns the current dynamic gas factors
func (k Keeper) GetDynamicGasFactors(ctx keepertypes.Context) feetypes.DynamicGasFactors {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(feetypes.DynamicGasFactorsKey)
	if bz == nil {
		return feetypes.NewDynamicGasFactors()
	}

	var factors feetypes.DynamicGasFactors
	if err := json.Unmarshal(bz, &factors); err != nil {
		return feetypes.NewDynamicGasFactors()
	}

	return factors
}

// Gas Price Statistics Management

// SetGasPriceStats sets gas price statistics for a denomination
func (k Keeper) SetGasPriceStats(ctx keepertypes.Context, stats feetypes.GasPriceStats) error {
	if err := stats.ValidateBasic(); err != nil {
		return err
	}

	store := ctx.KVStore(k.storeKey)
	key := append(feetypes.GasPriceStatsKey, []byte(stats.Denom)...)

	bz, err := json.Marshal(stats)
	if err != nil {
		return err
	}

	store.Set(key, bz)
	return nil
}

// GetGasPriceStats returns gas price statistics for a denomination
func (k Keeper) GetGasPriceStats(ctx keepertypes.Context, denom string) (feetypes.GasPriceStats, bool) {
	store := ctx.KVStore(k.storeKey)
	key := append(feetypes.GasPriceStatsKey, []byte(denom)...)

	bz := store.Get(key)
	if bz == nil {
		return feetypes.GasPriceStats{}, false
	}

	var stats feetypes.GasPriceStats
	if err := json.Unmarshal(bz, &stats); err != nil {
		return feetypes.GasPriceStats{}, false
	}

	return stats, true
}

// Interface implementations

// EstimateGas implements GasEstimator interface
func (k Keeper) EstimateGas(ctx keepertypes.Context, msgs []commontypes.Msg) (uint64, error) {
	return k.gasEstimator.EstimateGas(ctx, msgs)
}

// EstimateDynamicGas implements GasEstimator interface
func (k Keeper) EstimateDynamicGas(ctx keepertypes.Context, msg commontypes.Msg, factors feetypes.DynamicGasFactors) (uint64, error) {
	return k.gasEstimator.EstimateDynamicGas(ctx, msg, factors)
}

// EstimateGasForModule implements GasEstimator interface
func (k Keeper) EstimateGasForModule(ctx keepertypes.Context, moduleName, msgType string, msg commontypes.Msg) (uint64, error) {
	return k.gasEstimator.EstimateGasForModule(ctx, moduleName, msgType, msg)
}

// CalculateFee implements FeeCalculator interface
func (k Keeper) CalculateFee(ctx keepertypes.Context, req feetypes.FeeCalculationRequest) (feetypes.FeeCalculationResponse, error) {
	return k.feeCalculator.CalculateFee(ctx, req)
}

// ValidateFee implements FeeCalculator interface
func (k Keeper) ValidateFee(ctx keepertypes.Context, msgs []commontypes.Msg, fee commontypes.Coin, gasLimit uint64) error {
	return k.feeCalculator.ValidateFee(ctx, msgs, fee, gasLimit)
}

// GetMinimumFee implements FeeCalculator interface
func (k Keeper) GetMinimumFee(ctx keepertypes.Context, msgs []commontypes.Msg, gasLimit uint64, feeDenom string) (commontypes.Coin, error) {
	return k.feeCalculator.GetMinimumFee(ctx, msgs, gasLimit, feeDenom)
}

// CollectFees implements FeeCollector interface
func (k Keeper) CollectFees(ctx keepertypes.Context, fees commontypes.Coin) error {
	return k.feeCollector.CollectFees(ctx, fees)
}

// DistributeFees implements FeeCollector interface
func (k Keeper) DistributeFees(ctx keepertypes.Context) error {
	return k.feeCollector.DistributeFees(ctx)
}

// GetCollectedFees implements FeeCollector interface
func (k Keeper) GetCollectedFees(ctx keepertypes.Context, denom string) (commontypes.Coin, error) {
	return k.feeCollector.GetCollectedFees(ctx, denom)
}

// UpdateGasPrices implements GasPriceOracle interface
func (k Keeper) UpdateGasPrices(ctx keepertypes.Context) error {
	return k.gasPriceOracle.UpdateGasPrices(ctx)
}

// GetRecommendedGasPrice implements GasPriceOracle interface
func (k Keeper) GetRecommendedGasPrice(ctx keepertypes.Context, denom string) (string, error) {
	return k.gasPriceOracle.GetRecommendedGasPrice(ctx, denom)
}

// Utility functions

// ValidateAuthority checks if the given address has authority
func (k Keeper) ValidateAuthority(address string) error {
	if address != k.authority {
		return fmt.Errorf("unauthorized: expected %s, got %s", k.authority, address)
	}
	return nil
}

// CalculateTransactionSize calculates the size of a transaction in bytes
func (k Keeper) CalculateTransactionSize(msgs []commontypes.Msg) uint64 {
	// Simple estimation based on JSON marshaling
	// In production, this would use the actual transaction encoding
	totalSize := uint64(0)
	for _, msg := range msgs {
		if bz, err := json.Marshal(msg); err == nil {
			totalSize += uint64(len(bz))
		}
	}
	return totalSize
}

// Helper function to convert strings to uint64
func stringToUint64(s string) uint64 {
	if val, err := strconv.ParseUint(s, 10, 64); err == nil {
		return val
	}
	return 0
}

// Helper function to convert strings to float64
func stringToFloat64(s string) float64 {
	if val, err := strconv.ParseFloat(s, 64); err == nil {
		return val
	}
	return 0.0
}

// convertEvent converts a commontypes.Event to keepertypes.Event
func convertEvent(event commontypes.Event) keepertypes.Event {
	attributes := make([]keepertypes.Attribute, len(event.Attributes))
	for i, attr := range event.Attributes {
		attributes[i] = keepertypes.Attribute{
			Key:   attr.Key,
			Value: attr.Value,
		}
	}
	return keepertypes.Event{
		Type:       event.Type,
		Attributes: attributes,
	}
}