package keeper

import (
	"fmt"
	"strings"

	keepertypes "github.com/b2network/pulsar/keeper/types"
	feetypes "github.com/b2network/pulsar/modules/fee/types"
)

// GasConfigManager manages gas configurations for modules
type GasConfigManager struct {
	keeper *Keeper
}

// NewGasConfigManager creates a new gas configuration manager
func NewGasConfigManager(keeper *Keeper) *GasConfigManager {
	return &GasConfigManager{
		keeper: keeper,
	}
}

// LoadGasConfigFromFile loads gas configuration from a file
func (gcm *GasConfigManager) LoadGasConfigFromFile(ctx keepertypes.Context, configPath string) error {
	// In a real implementation, this would read from a config file
	// For now, we'll use hardcoded configurations
	configs := gcm.getBuiltinGasConfigs()

	for _, config := range configs {
		if err := gcm.keeper.SetModuleGasConfig(ctx, config); err != nil {
			return fmt.Errorf("failed to load gas config for module %s: %w", config.ModuleName, err)
		}
	}

	ctx.Logger().Info("loaded gas configurations from built-in configs",
		"path", configPath, "modules", len(configs))

	return nil
}

// getBuiltinGasConfigs returns built-in gas configurations
func (gcm *GasConfigManager) getBuiltinGasConfigs() []feetypes.ModuleGasConfig {
	// Bank module configuration
	bankConfig := feetypes.NewModuleGasConfig("bank", 15000, true)
	bankConfig.SetMsgGasRate("MsgSend", 30000)
	bankConfig.SetMsgGasRate("MsgMultiSend", 50000)

	// Coin module configuration
	coinConfig := feetypes.NewModuleGasConfig("coin", 20000, true)
	coinConfig.SetMsgGasRate("MsgMint", 40000)
	coinConfig.SetMsgGasRate("MsgBurn", 35000)
	coinConfig.SetMsgGasRate("MsgSetPermissions", 25000)
	coinConfig.SetMsgGasRate("MsgSetMetadata", 20000)

	// Fee module configuration
	feeConfig := feetypes.NewModuleGasConfig("fee", 10000, true)
	feeConfig.SetMsgGasRate("MsgAddFeeDenom", 15000)
	feeConfig.SetMsgGasRate("MsgUpdateFeeDenom", 12000)
	feeConfig.SetMsgGasRate("MsgRemoveFeeDenom", 10000)
	feeConfig.SetMsgGasRate("MsgSetModuleGasConfig", 18000)
	feeConfig.SetMsgGasRate("MsgUpdateGasFactors", 16000)
	feeConfig.SetMsgGasRate("MsgSetFeeDistribution", 14000)

	// Auth module configuration
	authConfig := feetypes.NewModuleGasConfig("auth", 5000, true)
	authConfig.SetMsgGasRate("MsgCreateAccount", 10000)
	authConfig.SetMsgGasRate("MsgUpdateAccount", 8000)

	// Staking module configuration (if exists in the future)
	stakingConfig := feetypes.NewModuleGasConfig("staking", 25000, true)
	stakingConfig.SetMsgGasRate("MsgDelegate", 50000)
	stakingConfig.SetMsgGasRate("MsgUndelegate", 45000)
	stakingConfig.SetMsgGasRate("MsgRedelegate", 55000)
	stakingConfig.SetMsgGasRate("MsgCreateValidator", 100000)
	stakingConfig.SetMsgGasRate("MsgEditValidator", 30000)

	// Distribution module configuration (if exists in the future)
	distributionConfig := feetypes.NewModuleGasConfig("distribution", 15000, true)
	distributionConfig.SetMsgGasRate("MsgWithdrawDelegatorReward", 25000)
	distributionConfig.SetMsgGasRate("MsgWithdrawValidatorCommission", 20000)
	distributionConfig.SetMsgGasRate("MsgSetWithdrawAddress", 15000)

	return []feetypes.ModuleGasConfig{
		bankConfig,
		coinConfig,
		feeConfig,
		authConfig,
		stakingConfig,
		distributionConfig,
	}
}

// UpdateGasConfig updates gas configuration for a module
func (gcm *GasConfigManager) UpdateGasConfig(ctx keepertypes.Context, moduleName string, updates map[string]uint64) error {
	config, found := gcm.keeper.GetModuleGasConfig(ctx, moduleName)
	if !found {
		return fmt.Errorf("module %s not found in gas configuration", moduleName)
	}

	// Create a copy to avoid modifying the original
	updatedConfig := config
	for msgType, gasRate := range updates {
		updatedConfig.SetMsgGasRate(msgType, gasRate)
	}

	// Validate the updated configuration
	if err := updatedConfig.ValidateBasic(); err != nil {
		return fmt.Errorf("invalid gas configuration after update: %w", err)
	}

	// Save the updated configuration
	if err := gcm.keeper.SetModuleGasConfig(ctx, updatedConfig); err != nil {
		return fmt.Errorf("failed to save updated gas configuration: %w", err)
	}

	ctx.Logger().Info("updated gas configuration",
		"module", moduleName, "updates", len(updates))

	return nil
}

// EnableModule enables gas metering for a module
func (gcm *GasConfigManager) EnableModule(ctx keepertypes.Context, moduleName string) error {
	config, found := gcm.keeper.GetModuleGasConfig(ctx, moduleName)
	if !found {
		return fmt.Errorf("module %s not found in gas configuration", moduleName)
	}

	if config.Enabled {
		return nil // Already enabled
	}

	config.Enabled = true
	return gcm.keeper.SetModuleGasConfig(ctx, config)
}

// DisableModule disables gas metering for a module
func (gcm *GasConfigManager) DisableModule(ctx keepertypes.Context, moduleName string) error {
	config, found := gcm.keeper.GetModuleGasConfig(ctx, moduleName)
	if !found {
		return fmt.Errorf("module %s not found in gas configuration", moduleName)
	}

	if !config.Enabled {
		return nil // Already disabled
	}

	config.Enabled = false
	return gcm.keeper.SetModuleGasConfig(ctx, config)
}

// GetModuleStatus returns the status of gas metering for a module
func (gcm *GasConfigManager) GetModuleStatus(ctx keepertypes.Context, moduleName string) (bool, error) {
	config, found := gcm.keeper.GetModuleGasConfig(ctx, moduleName)
	if !found {
		return false, fmt.Errorf("module %s not found in gas configuration", moduleName)
	}

	return config.Enabled, nil
}

// ListModules returns all configured modules
func (gcm *GasConfigManager) ListModules(ctx keepertypes.Context) []string {
	configs := gcm.keeper.GetAllModuleGasConfigs(ctx)
	modules := make([]string, len(configs))
	for i, config := range configs {
		modules[i] = config.ModuleName
	}
	return modules
}

// GetGasConfigSummary returns a summary of gas configurations
func (gcm *GasConfigManager) GetGasConfigSummary(ctx keepertypes.Context) GasConfigSummary {
	configs := gcm.keeper.GetAllModuleGasConfigs(ctx)

	summary := GasConfigSummary{
		TotalModules:   len(configs),
		EnabledModules: 0,
		ModuleConfigs:  make(map[string]ModuleConfigSummary),
	}

	for _, config := range configs {
		if config.Enabled {
			summary.EnabledModules++
		}

		summary.ModuleConfigs[config.ModuleName] = ModuleConfigSummary{
			ModuleName:    config.ModuleName,
			BaseGas:       config.BaseGas,
			Enabled:       config.Enabled,
			MessageTypes:  len(config.MsgGasRates),
		}
	}

	return summary
}

// GasConfigSummary represents a summary of gas configurations
type GasConfigSummary struct {
	TotalModules   int                            `json:"total_modules"`
	EnabledModules int                            `json:"enabled_modules"`
	ModuleConfigs  map[string]ModuleConfigSummary `json:"module_configs"`
}

// ModuleConfigSummary represents a summary of a module's gas configuration
type ModuleConfigSummary struct {
	ModuleName   string `json:"module_name"`
	BaseGas      uint64 `json:"base_gas"`
	Enabled      bool   `json:"enabled"`
	MessageTypes int    `json:"message_types"`
}

// AutoConfigureModule automatically configures gas for a new module
func (gcm *GasConfigManager) AutoConfigureModule(ctx keepertypes.Context, moduleName string, messageTypes []string) error {
	// Check if module already exists
	_, found := gcm.keeper.GetModuleGasConfig(ctx, moduleName)
	if found {
		return fmt.Errorf("module %s already has gas configuration", moduleName)
	}

	// Create default configuration
	baseGas := gcm.calculateDefaultBaseGas(moduleName)
	config := feetypes.NewModuleGasConfig(moduleName, baseGas, true)

	// Set default gas rates for message types
	for _, msgType := range messageTypes {
		gasRate := gcm.calculateDefaultGasRate(msgType)
		config.SetMsgGasRate(msgType, gasRate)
	}

	// Save the configuration
	if err := gcm.keeper.SetModuleGasConfig(ctx, config); err != nil {
		return fmt.Errorf("failed to save auto-configured gas config: %w", err)
	}

	ctx.Logger().Info("auto-configured gas for module",
		"module", moduleName, "base_gas", baseGas, "message_types", len(messageTypes))

	return nil
}

// calculateDefaultBaseGas calculates default base gas for a module
func (gcm *GasConfigManager) calculateDefaultBaseGas(moduleName string) uint64 {
	// Default base gas based on module type
	switch {
	case strings.Contains(moduleName, "bank"):
		return 15000
	case strings.Contains(moduleName, "coin"):
		return 20000
	case strings.Contains(moduleName, "staking"):
		return 25000
	case strings.Contains(moduleName, "gov"):
		return 30000
	case strings.Contains(moduleName, "auth"):
		return 5000
	default:
		return 10000 // Default for unknown modules
	}
}

// calculateDefaultGasRate calculates default gas rate for a message type
func (gcm *GasConfigManager) calculateDefaultGasRate(msgType string) uint64 {
	// Default gas rates based on message type patterns
	switch {
	case strings.Contains(msgType, "Send"):
		return 30000
	case strings.Contains(msgType, "Transfer"):
		return 25000
	case strings.Contains(msgType, "Mint"):
		return 40000
	case strings.Contains(msgType, "Burn"):
		return 35000
	case strings.Contains(msgType, "Create"):
		return 50000
	case strings.Contains(msgType, "Update"):
		return 25000
	case strings.Contains(msgType, "Delete"):
		return 20000
	case strings.Contains(msgType, "Delegate"):
		return 50000
	case strings.Contains(msgType, "Vote"):
		return 30000
	default:
		return 20000 // Default for unknown message types
	}
}

// ValidateConfiguration validates all gas configurations
func (gcm *GasConfigManager) ValidateConfiguration(ctx keepertypes.Context) error {
	configs := gcm.keeper.GetAllModuleGasConfigs(ctx)

	for _, config := range configs {
		if err := config.ValidateBasic(); err != nil {
			return fmt.Errorf("invalid configuration for module %s: %w", config.ModuleName, err)
		}
	}

	ctx.Logger().Info("validated all gas configurations",
		"modules", len(configs))

	return nil
}

// ExportConfiguration exports current gas configuration
func (gcm *GasConfigManager) ExportConfiguration(ctx keepertypes.Context) map[string]feetypes.ModuleGasConfig {
	configs := gcm.keeper.GetAllModuleGasConfigs(ctx)
	configMap := make(map[string]feetypes.ModuleGasConfig)

	for _, config := range configs {
		configMap[config.ModuleName] = config
	}

	return configMap
}

// ImportConfiguration imports gas configuration
func (gcm *GasConfigManager) ImportConfiguration(ctx keepertypes.Context, configs map[string]feetypes.ModuleGasConfig) error {
	// Validate all configurations first
	for moduleName, config := range configs {
		if err := config.ValidateBasic(); err != nil {
			return fmt.Errorf("invalid configuration for module %s: %w", moduleName, err)
		}
	}

	// Import all configurations
	for _, config := range configs {
		if err := gcm.keeper.SetModuleGasConfig(ctx, config); err != nil {
			return fmt.Errorf("failed to import configuration for module %s: %w", config.ModuleName, err)
		}
	}

	ctx.Logger().Info("imported gas configurations",
		"modules", len(configs))

	return nil
}