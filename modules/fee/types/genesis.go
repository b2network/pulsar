package types

import (
	"encoding/json"
	"fmt"
)

// GenesisState defines the fee module's genesis state
type GenesisState struct {
	Params               Params                  `json:"params"`
	FeeDenoms            []FeeDenom              `json:"fee_denoms"`
	ModuleGasConfigs     []ModuleGasConfig       `json:"module_gas_configs"`
	DynamicGasFactors    DynamicGasFactors       `json:"dynamic_gas_factors"`
	FeeDistributionConfig FeeDistributionConfig  `json:"fee_distribution_config"`
	GasPriceStats        []GasPriceStats        `json:"gas_price_stats"`
}

// NewGenesisState creates a new genesis state
func NewGenesisState(
	params Params,
	feeDenoms []FeeDenom,
	moduleGasConfigs []ModuleGasConfig,
	dynamicGasFactors DynamicGasFactors,
	feeDistributionConfig FeeDistributionConfig,
	gasPriceStats []GasPriceStats,
) *GenesisState {
	return &GenesisState{
		Params:                params,
		FeeDenoms:             feeDenoms,
		ModuleGasConfigs:      moduleGasConfigs,
		DynamicGasFactors:     dynamicGasFactors,
		FeeDistributionConfig: feeDistributionConfig,
		GasPriceStats:         gasPriceStats,
	}
}

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:                DefaultParams(),
		FeeDenoms:             DefaultFeeDenoms(),
		ModuleGasConfigs:      DefaultModuleGasConfigs(),
		DynamicGasFactors:     DefaultDynamicGasFactors(),
		FeeDistributionConfig: DefaultFeeDistributionConfig(),
		GasPriceStats:         []GasPriceStats{},
	}
}

// DefaultFeeDenoms returns default fee denominations
func DefaultFeeDenoms() []FeeDenom {
	return []FeeDenom{
		NewFeeDenom("ubtc", "0.01ubtc", true, 100),    // Primary fee token
		NewFeeDenom("ueth", "100ueth", true, 90),      // Secondary fee token
		NewFeeDenom("upulse", "1000upulse", true, 80), // Native token
	}
}

// DefaultModuleGasConfigs returns default module gas configurations based on standards
func DefaultModuleGasConfigs() []ModuleGasConfig {
	// Bank module - handles token transfers
	bankConfig := NewModuleGasConfig("bank", 15000, true)
	bankConfig.SetMsgGasRate("MsgSend", 30000)
	bankConfig.SetMsgGasRate("MsgMultiSend", 50000)

	// Coin module - handles token minting, burning, and metadata
	coinConfig := NewModuleGasConfig("coin", 20000, true)
	coinConfig.SetMsgGasRate("MsgMint", 40000)
	coinConfig.SetMsgGasRate("MsgBurn", 35000)
	coinConfig.SetMsgGasRate("MsgSetPermissions", 25000)
	coinConfig.SetMsgGasRate("MsgSetMetadata", 20000)
	coinConfig.SetMsgGasRate("MsgCreateDenomination", 60000)
	coinConfig.SetMsgGasRate("MsgUpdateDenomination", 30000)

	// Auth module - handles account management
	authConfig := NewModuleGasConfig("auth", 5000, true)
	authConfig.SetMsgGasRate("MsgCreateAccount", 10000)
	authConfig.SetMsgGasRate("MsgUpdateAccount", 8000)
	authConfig.SetMsgGasRate("MsgSetAccountParams", 15000)

	// Fee module - handles fee management
	feeConfig := NewModuleGasConfig("fee", 10000, true)
	feeConfig.SetMsgGasRate("MsgAddFeeDenom", 15000)
	feeConfig.SetMsgGasRate("MsgUpdateFeeDenom", 12000)
	feeConfig.SetMsgGasRate("MsgRemoveFeeDenom", 10000)
	feeConfig.SetMsgGasRate("MsgSetModuleGasConfig", 18000)
	feeConfig.SetMsgGasRate("MsgUpdateGasFactors", 16000)
	feeConfig.SetMsgGasRate("MsgSetFeeDistribution", 14000)

	// Staking module - handles validator operations and delegation
	stakingConfig := NewModuleGasConfig("staking", 25000, true)
	stakingConfig.SetMsgGasRate("MsgCreateValidator", 100000)
	stakingConfig.SetMsgGasRate("MsgEditValidator", 30000)
	stakingConfig.SetMsgGasRate("MsgDelegate", 50000)
	stakingConfig.SetMsgGasRate("MsgUndelegate", 45000)
	stakingConfig.SetMsgGasRate("MsgRedelegate", 55000)
	stakingConfig.SetMsgGasRate("MsgCancelUnbonding", 20000)

	// Distribution module - handles reward distribution
	distributionConfig := NewModuleGasConfig("distribution", 15000, true)
	distributionConfig.SetMsgGasRate("MsgWithdrawDelegatorReward", 25000)
	distributionConfig.SetMsgGasRate("MsgWithdrawValidatorCommission", 20000)
	distributionConfig.SetMsgGasRate("MsgSetWithdrawAddress", 15000)
	distributionConfig.SetMsgGasRate("MsgFundCommunityPool", 10000)

	// Government module - handles governance proposals and voting
	govConfig := NewModuleGasConfig("gov", 30000, true)
	govConfig.SetMsgGasRate("MsgSubmitProposal", 75000)
	govConfig.SetMsgGasRate("MsgDeposit", 25000)
	govConfig.SetMsgGasRate("MsgVote", 30000)
	govConfig.SetMsgGasRate("MsgVoteWeighted", 40000)

	// Slashing module - handles validator punishment
	slashingConfig := NewModuleGasConfig("slashing", 20000, true)
	slashingConfig.SetMsgGasRate("MsgUnjail", 50000)
	slashingConfig.SetMsgGasRate("MsgUpdateParams", 30000)

	return []ModuleGasConfig{
		bankConfig,
		coinConfig,
		authConfig,
		feeConfig,
		stakingConfig,
		distributionConfig,
		govConfig,
		slashingConfig,
	}
}

// DefaultDynamicGasFactors returns default dynamic gas factors
func DefaultDynamicGasFactors() DynamicGasFactors {
	return DynamicGasFactors{
		SizeMultiplier:   1.0,
		ComplexityFactor: 1.0,
		NetworkFactor:    1.0,
		StorageFactor:    1.2,
	}
}

// ValidateGenesis validates the genesis state
func ValidateGenesis(data GenesisState) error {
	if err := data.Params.ValidateBasic(); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}

	// Validate fee denominations
	denomMap := make(map[string]bool)
	for _, feeDenom := range data.FeeDenoms {
		if err := feeDenom.ValidateBasic(); err != nil {
			return fmt.Errorf("invalid fee denomination %s: %w", feeDenom.Denom, err)
		}
		if denomMap[feeDenom.Denom] {
			return fmt.Errorf("duplicate fee denomination: %s", feeDenom.Denom)
		}
		denomMap[feeDenom.Denom] = true
	}

	// Validate module gas configs
	moduleMap := make(map[string]bool)
	for _, config := range data.ModuleGasConfigs {
		if err := config.ValidateBasic(); err != nil {
			return fmt.Errorf("invalid module gas config for %s: %w", config.ModuleName, err)
		}
		if moduleMap[config.ModuleName] {
			return fmt.Errorf("duplicate module gas config: %s", config.ModuleName)
		}
		moduleMap[config.ModuleName] = true
	}

	// Validate dynamic gas factors
	if err := data.DynamicGasFactors.ValidateBasic(); err != nil {
		return fmt.Errorf("invalid dynamic gas factors: %w", err)
	}

	// Validate fee distribution config
	if err := data.FeeDistributionConfig.ValidateBasic(); err != nil {
		return fmt.Errorf("invalid fee distribution config: %w", err)
	}

	// Validate gas price stats
	for _, stats := range data.GasPriceStats {
		if err := stats.ValidateBasic(); err != nil {
			return fmt.Errorf("invalid gas price stats for %s: %w", stats.Denom, err)
		}
	}

	return nil
}

// GetGenesisStateFromAppState gets the genesis state from the app state
func GetGenesisStateFromAppState(cdc interface{}, appState map[string]json.RawMessage) *GenesisState {
	var genesisState GenesisState

	if appState[ModuleName] != nil {
		// In a real implementation, we would use the codec to unmarshal
		// For now, we'll use json.Unmarshal
		json.Unmarshal(appState[ModuleName], &genesisState)
	}

	return &genesisState
}