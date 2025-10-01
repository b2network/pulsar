package config

import (
	"fmt"

	feetypes "github.com/b2network/pulsar/modules/fee/types"
)

// ConfigValidator provides validation for fee module configurations
type ConfigValidator struct {
	// Known module names for validation
	knownModules map[string]bool
	// Known message types for validation
	knownMessageTypes map[string][]string
}

// NewConfigValidator creates a new configuration validator
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{
		knownModules:      getKnownModules(),
		knownMessageTypes: getKnownMessageTypes(),
	}
}

// ValidateGasConfig validates a single gas configuration
func (cv *ConfigValidator) ValidateGasConfig(config feetypes.ModuleGasConfig) error {
	// Basic validation
	if err := config.ValidateBasic(); err != nil {
		return err
	}

	// Check if module is known
	if !cv.knownModules[config.ModuleName] {
		return fmt.Errorf("unknown module: %s", config.ModuleName)
	}

	// Validate base gas is reasonable
	if err := cv.validateBaseGas(config.ModuleName, config.BaseGas); err != nil {
		return err
	}

	// Validate message-specific gas rates
	if err := cv.validateMessageGasRates(config.ModuleName, config.MsgGasRates); err != nil {
		return err
	}

	return nil
}

// ValidateAllGasConfigs validates a collection of gas configurations
func (cv *ConfigValidator) ValidateAllGasConfigs(configs []feetypes.ModuleGasConfig) error {
	moduleNames := make(map[string]bool)

	for _, config := range configs {
		// Validate individual config
		if err := cv.ValidateGasConfig(config); err != nil {
			return fmt.Errorf("invalid config for module %s: %w", config.ModuleName, err)
		}

		// Check for duplicates
		if moduleNames[config.ModuleName] {
			return fmt.Errorf("duplicate module configuration: %s", config.ModuleName)
		}
		moduleNames[config.ModuleName] = true
	}

	// Validate that required modules are present
	if err := cv.validateRequiredModules(configs); err != nil {
		return err
	}

	return nil
}

// ValidateFeeDenom validates a fee denomination
func (cv *ConfigValidator) ValidateFeeDenom(feeDenom feetypes.FeeDenom) error {
	// Basic validation
	if err := feeDenom.ValidateBasic(); err != nil {
		return err
	}

	// Validate denomination format
	if err := cv.validateDenomFormat(feeDenom.Denom); err != nil {
		return err
	}

	// Validate gas price format
	if err := cv.validateGasPriceFormat(feeDenom.MinGasPrice); err != nil {
		return err
	}

	// Validate priority is reasonable
	if err := cv.validatePriority(feeDenom.Priority); err != nil {
		return err
	}

	return nil
}

// ValidateAllFeeDenoms validates a collection of fee denominations
func (cv *ConfigValidator) ValidateAllFeeDenoms(feeDenoms []feetypes.FeeDenom) error {
	denomNames := make(map[string]bool)
	priorities := make(map[uint32]bool)

	for _, feeDenom := range feeDenoms {
		// Validate individual fee denomination
		if err := cv.ValidateFeeDenom(feeDenom); err != nil {
			return fmt.Errorf("invalid fee denomination %s: %w", feeDenom.Denom, err)
		}

		// Check for duplicate denominations
		if denomNames[feeDenom.Denom] {
			return fmt.Errorf("duplicate fee denomination: %s", feeDenom.Denom)
		}
		denomNames[feeDenom.Denom] = true

		// Check for duplicate priorities (only for enabled denoms)
		if feeDenom.Enabled {
			if priorities[feeDenom.Priority] {
				return fmt.Errorf("duplicate priority %d for enabled fee denomination", feeDenom.Priority)
			}
			priorities[feeDenom.Priority] = true
		}
	}

	// Ensure at least one fee denomination is enabled
	hasEnabledDenom := false
	for _, feeDenom := range feeDenoms {
		if feeDenom.Enabled {
			hasEnabledDenom = true
			break
		}
	}

	if !hasEnabledDenom {
		return fmt.Errorf("at least one fee denomination must be enabled")
	}

	return nil
}

// ValidateDynamicGasFactors validates dynamic gas factors
func (cv *ConfigValidator) ValidateDynamicGasFactors(factors feetypes.DynamicGasFactors) error {
	// Basic validation
	if err := factors.ValidateBasic(); err != nil {
		return err
	}

	// Validate factor ranges
	if err := cv.validateFactorRanges(factors); err != nil {
		return err
	}

	return nil
}

// validateBaseGas validates base gas for a module
func (cv *ConfigValidator) validateBaseGas(moduleName string, baseGas uint64) error {
	// Define reasonable ranges for different modules
	ranges := map[string]struct{ min, max uint64 }{
		"bank":         {10000, 50000},
		"coin":         {15000, 60000},
		"fee":          {5000, 30000},
		"auth":         {3000, 20000},
		"staking":      {20000, 100000},
		"distribution": {10000, 50000},
		"gov":          {25000, 150000},
	}

	if r, exists := ranges[moduleName]; exists {
		if baseGas < r.min || baseGas > r.max {
			return fmt.Errorf("base gas %d for module %s is outside reasonable range [%d, %d]",
				baseGas, moduleName, r.min, r.max)
		}
	} else {
		// Default range for unknown modules
		if baseGas < 5000 || baseGas > 100000 {
			return fmt.Errorf("base gas %d for module %s is outside default range [5000, 100000]",
				baseGas, moduleName)
		}
	}

	return nil
}

// validateMessageGasRates validates message-specific gas rates
func (cv *ConfigValidator) validateMessageGasRates(moduleName string, msgGasRates map[string]uint64) error {
	knownMsgTypes, exists := cv.knownMessageTypes[moduleName]
	if !exists {
		// For unknown modules, just validate basic ranges
		for msgType, gasRate := range msgGasRates {
			if gasRate < 5000 || gasRate > 500000 {
				return fmt.Errorf("gas rate %d for message type %s is outside reasonable range [5000, 500000]",
					gasRate, msgType)
			}
		}
		return nil
	}

	// Validate known message types
	for msgType, gasRate := range msgGasRates {
		if !contains(knownMsgTypes, msgType) {
			return fmt.Errorf("unknown message type %s for module %s", msgType, moduleName)
		}

		if gasRate < 5000 || gasRate > 500000 {
			return fmt.Errorf("gas rate %d for message type %s is outside reasonable range [5000, 500000]",
				gasRate, msgType)
		}
	}

	return nil
}

// validateRequiredModules ensures required modules are configured
func (cv *ConfigValidator) validateRequiredModules(configs []feetypes.ModuleGasConfig) error {
	requiredModules := []string{"bank", "coin", "fee"}
	configuredModules := make(map[string]bool)

	for _, config := range configs {
		configuredModules[config.ModuleName] = true
	}

	for _, required := range requiredModules {
		if !configuredModules[required] {
			return fmt.Errorf("required module %s is not configured", required)
		}
	}

	return nil
}

// validateDenomFormat validates denomination format
func (cv *ConfigValidator) validateDenomFormat(denom string) error {
	if len(denom) < 3 || len(denom) > 16 {
		return fmt.Errorf("denomination length must be between 3 and 16 characters")
	}

	// Check if denomination contains only valid characters (lowercase letters and numbers)
	for _, char := range denom {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9')) {
			return fmt.Errorf("denomination must contain only lowercase letters and numbers")
		}
	}

	// Check if denomination starts with a letter
	if denom[0] < 'a' || denom[0] > 'z' {
		return fmt.Errorf("denomination must start with a lowercase letter")
	}

	return nil
}

// validateGasPriceFormat validates gas price format
func (cv *ConfigValidator) validateGasPriceFormat(gasPrice string) error {
	price, denom, err := feetypes.ParseGasPrice(gasPrice)
	if err != nil {
		return fmt.Errorf("invalid gas price format: %w", err)
	}

	if price <= 0 {
		return fmt.Errorf("gas price must be positive")
	}

	if err := cv.validateDenomFormat(denom); err != nil {
		return fmt.Errorf("invalid denomination in gas price: %w", err)
	}

	return nil
}

// validatePriority validates priority value
func (cv *ConfigValidator) validatePriority(priority uint32) error {
	if priority < 1 || priority > 1000 {
		return fmt.Errorf("priority must be between 1 and 1000")
	}
	return nil
}

// validateFactorRanges validates dynamic gas factor ranges
func (cv *ConfigValidator) validateFactorRanges(factors feetypes.DynamicGasFactors) error {
	// Define reasonable ranges for factors
	factorChecks := []struct {
		name  string
		value float64
		min   float64
		max   float64
	}{
		{"SizeMultiplier", factors.SizeMultiplier, 0.1, 10.0},
		{"ComplexityFactor", factors.ComplexityFactor, 0.1, 5.0},
		{"NetworkFactor", factors.NetworkFactor, 0.1, 5.0},
		{"StorageFactor", factors.StorageFactor, 0.1, 10.0},
	}

	for _, check := range factorChecks {
		if check.value < check.min || check.value > check.max {
			return fmt.Errorf("%s %.2f is outside reasonable range [%.1f, %.1f]",
				check.name, check.value, check.min, check.max)
		}
	}

	return nil
}

// getKnownModules returns a map of known module names
func getKnownModules() map[string]bool {
	return map[string]bool{
		"bank":         true,
		"coin":         true,
		"fee":          true,
		"auth":         true,
		"staking":      true,
		"distribution": true,
		"gov":          true,
		"slashing":     true,
		"upgrade":      true,
	}
}

// getKnownMessageTypes returns known message types for each module
func getKnownMessageTypes() map[string][]string {
	return map[string][]string{
		"bank": {
			"MsgSend",
			"MsgMultiSend",
		},
		"coin": {
			"MsgMint",
			"MsgBurn",
			"MsgSetPermissions",
			"MsgSetMetadata",
		},
		"fee": {
			"MsgAddFeeDenom",
			"MsgUpdateFeeDenom",
			"MsgRemoveFeeDenom",
			"MsgSetModuleGasConfig",
			"MsgUpdateGasFactors",
			"MsgSetFeeDistribution",
		},
		"auth": {
			"MsgCreateAccount",
			"MsgUpdateAccount",
		},
		"staking": {
			"MsgDelegate",
			"MsgUndelegate",
			"MsgRedelegate",
			"MsgCreateValidator",
			"MsgEditValidator",
		},
		"distribution": {
			"MsgWithdrawDelegatorReward",
			"MsgWithdrawValidatorCommission",
			"MsgSetWithdrawAddress",
		},
		"gov": {
			"MsgSubmitProposal",
			"MsgVote",
			"MsgVoteWeighted",
			"MsgDeposit",
		},
	}
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ValidationReport represents a validation report
type ValidationReport struct {
	Valid   bool     `json:"valid"`
	Errors  []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// GenerateValidationReport generates a comprehensive validation report
func (cv *ConfigValidator) GenerateValidationReport(
	gasConfigs []feetypes.ModuleGasConfig,
	feeDenoms []feetypes.FeeDenom,
	dynamicFactors feetypes.DynamicGasFactors,
) ValidationReport {
	report := ValidationReport{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// Validate gas configurations
	if err := cv.ValidateAllGasConfigs(gasConfigs); err != nil {
		report.Valid = false
		report.Errors = append(report.Errors, fmt.Sprintf("Gas configs: %s", err))
	}

	// Validate fee denominations
	if err := cv.ValidateAllFeeDenoms(feeDenoms); err != nil {
		report.Valid = false
		report.Errors = append(report.Errors, fmt.Sprintf("Fee denoms: %s", err))
	}

	// Validate dynamic gas factors
	if err := cv.ValidateDynamicGasFactors(dynamicFactors); err != nil {
		report.Valid = false
		report.Errors = append(report.Errors, fmt.Sprintf("Dynamic factors: %s", err))
	}

	// Generate warnings
	report.Warnings = cv.generateWarnings(gasConfigs, feeDenoms, dynamicFactors)

	return report
}

// generateWarnings generates validation warnings
func (cv *ConfigValidator) generateWarnings(
	gasConfigs []feetypes.ModuleGasConfig,
	feeDenoms []feetypes.FeeDenom,
	dynamicFactors feetypes.DynamicGasFactors,
) []string {
	warnings := []string{}

	// Check for disabled modules
	for _, config := range gasConfigs {
		if !config.Enabled {
			warnings = append(warnings, fmt.Sprintf("Module %s has gas metering disabled", config.ModuleName))
		}
	}

	// Check for disabled fee denominations
	for _, feeDenom := range feeDenoms {
		if !feeDenom.Enabled {
			warnings = append(warnings, fmt.Sprintf("Fee denomination %s is disabled", feeDenom.Denom))
		}
	}

	// Check for extreme dynamic factors
	if dynamicFactors.SizeMultiplier > 3.0 {
		warnings = append(warnings, "Size multiplier is very high (>3.0)")
	}
	if dynamicFactors.ComplexityFactor > 2.0 {
		warnings = append(warnings, "Complexity factor is very high (>2.0)")
	}
	if dynamicFactors.NetworkFactor > 2.0 {
		warnings = append(warnings, "Network factor is very high (>2.0)")
	}
	if dynamicFactors.StorageFactor > 3.0 {
		warnings = append(warnings, "Storage factor is very high (>3.0)")
	}

	return warnings
}