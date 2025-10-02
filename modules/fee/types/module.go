package types

// Store keys for different data types
var (
	// FeeDenomPrefix is the prefix for fee denomination storage
	FeeDenomPrefix = []byte("fee_denom")

	// ModuleGasConfigPrefix is the prefix for module gas configuration storage
	ModuleGasConfigPrefix = []byte("module_gas_config")

	// DynamicGasFactorsKeyModule is the key for dynamic gas factors (module-specific)
	DynamicGasFactorsKeyModule = []byte("dynamic_gas_factors")

	// ParamsKey is the key for module parameters
	ParamsKey = []byte("params")

	// GasPriceStatsKey is the key for gas price statistics
	GasPriceStatsKey = []byte("gas_price_stats")

	// CollectedFeesKey is the key for collected fees storage
	CollectedFeesKey = []byte("collected_fees")

	// FeeDistributionConfigKeyModule is the key for fee distribution configuration (module-specific)
	FeeDistributionConfigKeyModule = []byte("fee_distribution_config")
)

// GetFeeDenomKey returns the store key for a specific fee denomination
func GetFeeDenomKey(denom string) []byte {
	return append(FeeDenomPrefix, []byte(denom)...)
}

// GetModuleGasConfigKey returns the store key for a specific module's gas configuration
func GetModuleGasConfigKey(moduleName string) []byte {
	return append(ModuleGasConfigPrefix, []byte(moduleName)...)
}

// GetGasPriceStatsKey returns the key for gas price statistics
func GetGasPriceStatsKey(denom string) []byte {
	return append(GasPriceStatsKey, []byte(denom)...)
}

// GetCollectedFeesKey returns the key for collected fees by denomination
func GetCollectedFeesKey(denom string) []byte {
	return append(CollectedFeesKey, []byte(denom)...)
}