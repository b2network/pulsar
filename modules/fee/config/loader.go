package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	feetypes "github.com/b2network/pulsar/modules/fee/types"
)

// ConfigLoader handles loading and saving gas configurations
type ConfigLoader struct {
	configDir string
}

// NewConfigLoader creates a new configuration loader
func NewConfigLoader(configDir string) *ConfigLoader {
	return &ConfigLoader{
		configDir: configDir,
	}
}

// LoadGasConfigs loads gas configurations from files
func (cl *ConfigLoader) LoadGasConfigs() ([]feetypes.ModuleGasConfig, error) {
	configPath := filepath.Join(cl.configDir, "gas-configs.json")

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Return default configurations if file doesn't exist
		return cl.getDefaultConfigs(), nil
	}

	// Read config file
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read gas config file: %w", err)
	}

	var configData GasConfigFile
	if err := json.Unmarshal(data, &configData); err != nil {
		return nil, fmt.Errorf("failed to parse gas config file: %w", err)
	}

	// Validate configurations
	if err := cl.validateConfigs(configData.ModuleConfigs); err != nil {
		return nil, fmt.Errorf("invalid gas configurations: %w", err)
	}

	return configData.ModuleConfigs, nil
}

// SaveGasConfigs saves gas configurations to file
func (cl *ConfigLoader) SaveGasConfigs(configs []feetypes.ModuleGasConfig) error {
	configPath := filepath.Join(cl.configDir, "gas-configs.json")

	// Ensure config directory exists
	if err := os.MkdirAll(cl.configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configData := GasConfigFile{
		Version:       "1.0",
		ModuleConfigs: configs,
	}

	data, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal gas configurations: %w", err)
	}

	if err := ioutil.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write gas config file: %w", err)
	}

	return nil
}

// LoadFeeDenoms loads fee denomination configurations
func (cl *ConfigLoader) LoadFeeDenoms() ([]feetypes.FeeDenom, error) {
	configPath := filepath.Join(cl.configDir, "fee-denoms.json")

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Return default fee denominations if file doesn't exist
		return cl.getDefaultFeeDenoms(), nil
	}

	// Read config file
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read fee denoms config file: %w", err)
	}

	var configData FeeDenomConfigFile
	if err := json.Unmarshal(data, &configData); err != nil {
		return nil, fmt.Errorf("failed to parse fee denoms config file: %w", err)
	}

	// Validate configurations
	if err := cl.validateFeeDenoms(configData.FeeDenoms); err != nil {
		return nil, fmt.Errorf("invalid fee denomination configurations: %w", err)
	}

	return configData.FeeDenoms, nil
}

// SaveFeeDenoms saves fee denomination configurations to file
func (cl *ConfigLoader) SaveFeeDenoms(feeDenoms []feetypes.FeeDenom) error {
	configPath := filepath.Join(cl.configDir, "fee-denoms.json")

	// Ensure config directory exists
	if err := os.MkdirAll(cl.configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configData := FeeDenomConfigFile{
		Version:   "1.0",
		FeeDenoms: feeDenoms,
	}

	data, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal fee denomination configurations: %w", err)
	}

	if err := ioutil.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write fee denoms config file: %w", err)
	}

	return nil
}

// LoadDynamicGasFactors loads dynamic gas factors configuration
func (cl *ConfigLoader) LoadDynamicGasFactors() (feetypes.DynamicGasFactors, error) {
	configPath := filepath.Join(cl.configDir, "dynamic-gas-factors.json")

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Return default factors if file doesn't exist
		return cl.getDefaultDynamicGasFactors(), nil
	}

	// Read config file
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return feetypes.DynamicGasFactors{}, fmt.Errorf("failed to read dynamic gas factors config file: %w", err)
	}

	var configData DynamicGasFactorsConfigFile
	if err := json.Unmarshal(data, &configData); err != nil {
		return feetypes.DynamicGasFactors{}, fmt.Errorf("failed to parse dynamic gas factors config file: %w", err)
	}

	// Validate configuration
	if err := configData.DynamicGasFactors.ValidateBasic(); err != nil {
		return feetypes.DynamicGasFactors{}, fmt.Errorf("invalid dynamic gas factors configuration: %w", err)
	}

	return configData.DynamicGasFactors, nil
}

// SaveDynamicGasFactors saves dynamic gas factors configuration to file
func (cl *ConfigLoader) SaveDynamicGasFactors(factors feetypes.DynamicGasFactors) error {
	configPath := filepath.Join(cl.configDir, "dynamic-gas-factors.json")

	// Ensure config directory exists
	if err := os.MkdirAll(cl.configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configData := DynamicGasFactorsConfigFile{
		Version:           "1.0",
		DynamicGasFactors: factors,
	}

	data, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal dynamic gas factors configuration: %w", err)
	}

	if err := ioutil.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write dynamic gas factors config file: %w", err)
	}

	return nil
}

// validateConfigs validates gas configurations
func (cl *ConfigLoader) validateConfigs(configs []feetypes.ModuleGasConfig) error {
	moduleNames := make(map[string]bool)

	for _, config := range configs {
		if err := config.ValidateBasic(); err != nil {
			return fmt.Errorf("invalid config for module %s: %w", config.ModuleName, err)
		}

		if moduleNames[config.ModuleName] {
			return fmt.Errorf("duplicate module configuration: %s", config.ModuleName)
		}
		moduleNames[config.ModuleName] = true
	}

	return nil
}

// validateFeeDenoms validates fee denomination configurations
func (cl *ConfigLoader) validateFeeDenoms(feeDenoms []feetypes.FeeDenom) error {
	denomNames := make(map[string]bool)

	for _, feeDenom := range feeDenoms {
		if err := feeDenom.ValidateBasic(); err != nil {
			return fmt.Errorf("invalid fee denomination %s: %w", feeDenom.Denom, err)
		}

		if denomNames[feeDenom.Denom] {
			return fmt.Errorf("duplicate fee denomination: %s", feeDenom.Denom)
		}
		denomNames[feeDenom.Denom] = true
	}

	return nil
}

// getDefaultConfigs returns default gas configurations
func (cl *ConfigLoader) getDefaultConfigs() []feetypes.ModuleGasConfig {
	return feetypes.DefaultModuleGasConfigs()
}

// getDefaultFeeDenoms returns default fee denominations
func (cl *ConfigLoader) getDefaultFeeDenoms() []feetypes.FeeDenom {
	return feetypes.DefaultFeeDenoms()
}

// getDefaultDynamicGasFactors returns default dynamic gas factors
func (cl *ConfigLoader) getDefaultDynamicGasFactors() feetypes.DynamicGasFactors {
	return feetypes.DefaultDynamicGasFactors()
}

// GasConfigFile represents the structure of the gas configuration file
type GasConfigFile struct {
	Version       string                       `json:"version"`
	Description   string                       `json:"description,omitempty"`
	ModuleConfigs []feetypes.ModuleGasConfig   `json:"module_configs"`
}

// FeeDenomConfigFile represents the structure of the fee denomination configuration file
type FeeDenomConfigFile struct {
	Version   string               `json:"version"`
	FeeDenoms []feetypes.FeeDenom  `json:"fee_denoms"`
}

// DynamicGasFactorsConfigFile represents the structure of the dynamic gas factors configuration file
type DynamicGasFactorsConfigFile struct {
	Version           string                      `json:"version"`
	DynamicGasFactors feetypes.DynamicGasFactors  `json:"dynamic_gas_factors"`
}

// GenerateExampleConfigs generates example configuration files
func (cl *ConfigLoader) GenerateExampleConfigs() error {
	// Ensure config directory exists
	if err := os.MkdirAll(cl.configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Generate example gas configs
	if err := cl.generateExampleGasConfigs(); err != nil {
		return fmt.Errorf("failed to generate example gas configs: %w", err)
	}

	// Generate example fee denoms
	if err := cl.generateExampleFeeDenoms(); err != nil {
		return fmt.Errorf("failed to generate example fee denoms: %w", err)
	}

	// Generate example dynamic gas factors
	if err := cl.generateExampleDynamicGasFactors(); err != nil {
		return fmt.Errorf("failed to generate example dynamic gas factors: %w", err)
	}

	return nil
}

// generateExampleGasConfigs generates an example gas configuration file
func (cl *ConfigLoader) generateExampleGasConfigs() error {
	configs := cl.getDefaultConfigs()
	configData := GasConfigFile{
		Version:     "1.0",
		Description: "Example gas configuration for Pulsar blockchain modules",
		ModuleConfigs: configs,
	}

	data, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		return err
	}

	examplePath := filepath.Join(cl.configDir, "gas-configs.example.json")
	return ioutil.WriteFile(examplePath, data, 0644)
}

// generateExampleFeeDenoms generates an example fee denomination configuration file
func (cl *ConfigLoader) generateExampleFeeDenoms() error {
	feeDenoms := cl.getDefaultFeeDenoms()
	configData := FeeDenomConfigFile{
		Version:   "1.0",
		FeeDenoms: feeDenoms,
	}

	data, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		return err
	}

	examplePath := filepath.Join(cl.configDir, "fee-denoms.example.json")
	return ioutil.WriteFile(examplePath, data, 0644)
}

// generateExampleDynamicGasFactors generates an example dynamic gas factors configuration file
func (cl *ConfigLoader) generateExampleDynamicGasFactors() error {
	factors := cl.getDefaultDynamicGasFactors()
	configData := DynamicGasFactorsConfigFile{
		Version:           "1.0",
		DynamicGasFactors: factors,
	}

	data, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		return err
	}

	examplePath := filepath.Join(cl.configDir, "dynamic-gas-factors.example.json")
	return ioutil.WriteFile(examplePath, data, 0644)
}