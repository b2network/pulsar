package config

import (
	"fmt"

	feetypes "github.com/b2network/pulsar/modules/fee/types"
)

// GasStandards defines the standard gas consumption values for Pulsar blockchain
type GasStandards struct {
	// Base transaction costs
	BaseTxGas          uint64 `json:"base_tx_gas"`
	SignatureGas       uint64 `json:"signature_gas"`
	TxSizePerByteGas   uint64 `json:"tx_size_per_byte_gas"`

	// Module configurations
	BankGasConfig         BankGasConfig         `json:"bank"`
	CoinGasConfig         CoinGasConfig         `json:"coin"`
	AuthGasConfig         AuthGasConfig         `json:"auth"`
	FeeGasConfig          FeeGasConfig          `json:"fee"`
	StakingGasConfig      StakingGasConfig      `json:"staking"`
	DistributionGasConfig DistributionGasConfig `json:"distribution"`
	GovGasConfig          GovGasConfig          `json:"gov"`
	SlashingGasConfig     SlashingGasConfig     `json:"slashing"`
}

// BankGasConfig defines gas consumption for bank module operations
type BankGasConfig struct {
	BaseGas      uint64 `json:"base_gas"`
	Send         uint64 `json:"send"`
	MultiSend    uint64 `json:"multi_send"`
	// Additional costs based on complexity
	SendPerRecipient     uint64 `json:"send_per_recipient"`
	MultiSendPerInput    uint64 `json:"multi_send_per_input"`
	MultiSendPerOutput   uint64 `json:"multi_send_per_output"`
}

// CoinGasConfig defines gas consumption for coin module operations
type CoinGasConfig struct {
	BaseGas           uint64 `json:"base_gas"`
	Mint              uint64 `json:"mint"`
	Burn              uint64 `json:"burn"`
	SetPermissions    uint64 `json:"set_permissions"`
	SetMetadata       uint64 `json:"set_metadata"`
	CreateDenomination uint64 `json:"create_denomination"`
	UpdateDenomination uint64 `json:"update_denomination"`
	// Per-unit costs
	MintPerUnit       uint64 `json:"mint_per_unit"`
	BurnPerUnit       uint64 `json:"burn_per_unit"`
}

// AuthGasConfig defines gas consumption for auth module operations
type AuthGasConfig struct {
	BaseGas            uint64 `json:"base_gas"`
	CreateAccount      uint64 `json:"create_account"`
	UpdateAccount      uint64 `json:"update_account"`
	SetAccountParams   uint64 `json:"set_account_params"`
	VerifySignature    uint64 `json:"verify_signature"`
	// Signature verification costs by algorithm
	VerifyED25519      uint64 `json:"verify_ed25519"`
	VerifySecp256k1    uint64 `json:"verify_secp256k1"`
	VerifyMultisig     uint64 `json:"verify_multisig"`
}

// FeeGasConfig defines gas consumption for fee module operations
type FeeGasConfig struct {
	BaseGas                uint64 `json:"base_gas"`
	AddFeeDenom           uint64 `json:"add_fee_denom"`
	RemoveFeeDenom        uint64 `json:"remove_fee_denom"`
	UpdateFeeDenom        uint64 `json:"update_fee_denom"`
	SetModuleGasConfig    uint64 `json:"set_module_gas_config"`
	UpdateGasFactors      uint64 `json:"update_gas_factors"`
	SetFeeDistribution    uint64 `json:"set_fee_distribution"`
}

// StakingGasConfig defines gas consumption for staking module operations
type StakingGasConfig struct {
	BaseGas              uint64 `json:"base_gas"`
	CreateValidator      uint64 `json:"create_validator"`
	EditValidator        uint64 `json:"edit_validator"`
	Delegate             uint64 `json:"delegate"`
	Undelegate           uint64 `json:"undelegate"`
	Redelegate           uint64 `json:"redelegate"`
	CancelUnbonding      uint64 `json:"cancel_unbonding"`
	// Per-amount costs
	DelegatePerToken     uint64 `json:"delegate_per_token"`
	UndelegatePerToken   uint64 `json:"undelegate_per_token"`
	RedelegatePerToken   uint64 `json:"redelegate_per_token"`
}

// DistributionGasConfig defines gas consumption for distribution module operations
type DistributionGasConfig struct {
	BaseGas                       uint64 `json:"base_gas"`
	WithdrawDelegatorReward       uint64 `json:"withdraw_delegator_reward"`
	WithdrawValidatorCommission   uint64 `json:"withdraw_validator_commission"`
	SetWithdrawAddress           uint64 `json:"set_withdraw_address"`
	FundCommunityPool            uint64 `json:"fund_community_pool"`
	// Per-validator costs
	WithdrawRewardPerValidator   uint64 `json:"withdraw_reward_per_validator"`
}

// GovGasConfig defines gas consumption for governance module operations
type GovGasConfig struct {
	BaseGas           uint64 `json:"base_gas"`
	SubmitProposal    uint64 `json:"submit_proposal"`
	Deposit           uint64 `json:"deposit"`
	Vote              uint64 `json:"vote"`
	VoteWeighted      uint64 `json:"vote_weighted"`
	// Per-option costs for weighted voting
	VotePerOption     uint64 `json:"vote_per_option"`
	// Proposal type specific costs
	TextProposal      uint64 `json:"text_proposal"`
	ParameterChange   uint64 `json:"parameter_change"`
	SoftwareUpgrade   uint64 `json:"software_upgrade"`
}

// SlashingGasConfig defines gas consumption for slashing module operations
type SlashingGasConfig struct {
	BaseGas       uint64 `json:"base_gas"`
	Unjail        uint64 `json:"unjail"`
	UpdateParams  uint64 `json:"update_params"`
}

// DefaultGasStandards returns the default gas standards for Pulsar
func DefaultGasStandards() GasStandards {
	return GasStandards{
		// Base transaction costs
		BaseTxGas:        21000,  // Similar to Ethereum
		SignatureGas:     1000,   // Per signature verification
		TxSizePerByteGas: 10,     // Per byte of transaction data

		BankGasConfig: BankGasConfig{
			BaseGas:              15000,
			Send:                 30000,
			MultiSend:            50000,
			SendPerRecipient:     5000,
			MultiSendPerInput:    3000,
			MultiSendPerOutput:   3000,
		},

		CoinGasConfig: CoinGasConfig{
			BaseGas:            20000,
			Mint:               40000,
			Burn:               35000,
			SetPermissions:     25000,
			SetMetadata:        20000,
			CreateDenomination: 60000,
			UpdateDenomination: 30000,
			MintPerUnit:        10,
			BurnPerUnit:        8,
		},

		AuthGasConfig: AuthGasConfig{
			BaseGas:           5000,
			CreateAccount:     10000,
			UpdateAccount:     8000,
			SetAccountParams:  15000,
			VerifySignature:   1000,
			VerifyED25519:     590,
			VerifySecp256k1:   1000,
			VerifyMultisig:    1500,
		},

		FeeGasConfig: FeeGasConfig{
			BaseGas:             10000,
			AddFeeDenom:         15000,
			RemoveFeeDenom:      10000,
			UpdateFeeDenom:      12000,
			SetModuleGasConfig:  18000,
			UpdateGasFactors:    16000,
			SetFeeDistribution:  14000,
		},

		StakingGasConfig: StakingGasConfig{
			BaseGas:            25000,
			CreateValidator:    100000,
			EditValidator:      30000,
			Delegate:           50000,
			Undelegate:         45000,
			Redelegate:         55000,
			CancelUnbonding:    20000,
			DelegatePerToken:   1,
			UndelegatePerToken: 1,
			RedelegatePerToken: 2,
		},

		DistributionGasConfig: DistributionGasConfig{
			BaseGas:                     15000,
			WithdrawDelegatorReward:     25000,
			WithdrawValidatorCommission: 20000,
			SetWithdrawAddress:          15000,
			FundCommunityPool:           10000,
			WithdrawRewardPerValidator:  5000,
		},

		GovGasConfig: GovGasConfig{
			BaseGas:          30000,
			SubmitProposal:   75000,
			Deposit:          25000,
			Vote:             30000,
			VoteWeighted:     40000,
			VotePerOption:    5000,
			TextProposal:     50000,
			ParameterChange:  100000,
			SoftwareUpgrade: 150000,
		},

		SlashingGasConfig: SlashingGasConfig{
			BaseGas:      20000,
			Unjail:       50000,
			UpdateParams: 30000,
		},
	}
}

// ToModuleGasConfigs converts GasStandards to a slice of ModuleGasConfig
func (gs GasStandards) ToModuleGasConfigs() []feetypes.ModuleGasConfig {
	configs := []feetypes.ModuleGasConfig{}

	// Bank module
	bankConfig := feetypes.NewModuleGasConfig("bank", gs.BankGasConfig.BaseGas, true)
	bankConfig.SetMsgGasRate("MsgSend", gs.BankGasConfig.Send)
	bankConfig.SetMsgGasRate("MsgMultiSend", gs.BankGasConfig.MultiSend)
	configs = append(configs, bankConfig)

	// Coin module
	coinConfig := feetypes.NewModuleGasConfig("coin", gs.CoinGasConfig.BaseGas, true)
	coinConfig.SetMsgGasRate("MsgMint", gs.CoinGasConfig.Mint)
	coinConfig.SetMsgGasRate("MsgBurn", gs.CoinGasConfig.Burn)
	coinConfig.SetMsgGasRate("MsgSetPermissions", gs.CoinGasConfig.SetPermissions)
	coinConfig.SetMsgGasRate("MsgSetMetadata", gs.CoinGasConfig.SetMetadata)
	coinConfig.SetMsgGasRate("MsgCreateDenomination", gs.CoinGasConfig.CreateDenomination)
	coinConfig.SetMsgGasRate("MsgUpdateDenomination", gs.CoinGasConfig.UpdateDenomination)
	configs = append(configs, coinConfig)

	// Auth module
	authConfig := feetypes.NewModuleGasConfig("auth", gs.AuthGasConfig.BaseGas, true)
	authConfig.SetMsgGasRate("MsgCreateAccount", gs.AuthGasConfig.CreateAccount)
	authConfig.SetMsgGasRate("MsgUpdateAccount", gs.AuthGasConfig.UpdateAccount)
	authConfig.SetMsgGasRate("MsgSetAccountParams", gs.AuthGasConfig.SetAccountParams)
	configs = append(configs, authConfig)

	// Fee module
	feeConfig := feetypes.NewModuleGasConfig("fee", gs.FeeGasConfig.BaseGas, true)
	feeConfig.SetMsgGasRate("MsgAddFeeDenom", gs.FeeGasConfig.AddFeeDenom)
	feeConfig.SetMsgGasRate("MsgRemoveFeeDenom", gs.FeeGasConfig.RemoveFeeDenom)
	feeConfig.SetMsgGasRate("MsgUpdateFeeDenom", gs.FeeGasConfig.UpdateFeeDenom)
	feeConfig.SetMsgGasRate("MsgSetModuleGasConfig", gs.FeeGasConfig.SetModuleGasConfig)
	feeConfig.SetMsgGasRate("MsgUpdateGasFactors", gs.FeeGasConfig.UpdateGasFactors)
	feeConfig.SetMsgGasRate("MsgSetFeeDistribution", gs.FeeGasConfig.SetFeeDistribution)
	configs = append(configs, feeConfig)

	// Staking module
	stakingConfig := feetypes.NewModuleGasConfig("staking", gs.StakingGasConfig.BaseGas, true)
	stakingConfig.SetMsgGasRate("MsgCreateValidator", gs.StakingGasConfig.CreateValidator)
	stakingConfig.SetMsgGasRate("MsgEditValidator", gs.StakingGasConfig.EditValidator)
	stakingConfig.SetMsgGasRate("MsgDelegate", gs.StakingGasConfig.Delegate)
	stakingConfig.SetMsgGasRate("MsgUndelegate", gs.StakingGasConfig.Undelegate)
	stakingConfig.SetMsgGasRate("MsgRedelegate", gs.StakingGasConfig.Redelegate)
	stakingConfig.SetMsgGasRate("MsgCancelUnbonding", gs.StakingGasConfig.CancelUnbonding)
	configs = append(configs, stakingConfig)

	// Distribution module
	distributionConfig := feetypes.NewModuleGasConfig("distribution", gs.DistributionGasConfig.BaseGas, true)
	distributionConfig.SetMsgGasRate("MsgWithdrawDelegatorReward", gs.DistributionGasConfig.WithdrawDelegatorReward)
	distributionConfig.SetMsgGasRate("MsgWithdrawValidatorCommission", gs.DistributionGasConfig.WithdrawValidatorCommission)
	distributionConfig.SetMsgGasRate("MsgSetWithdrawAddress", gs.DistributionGasConfig.SetWithdrawAddress)
	distributionConfig.SetMsgGasRate("MsgFundCommunityPool", gs.DistributionGasConfig.FundCommunityPool)
	configs = append(configs, distributionConfig)

	// Government module
	govConfig := feetypes.NewModuleGasConfig("gov", gs.GovGasConfig.BaseGas, true)
	govConfig.SetMsgGasRate("MsgSubmitProposal", gs.GovGasConfig.SubmitProposal)
	govConfig.SetMsgGasRate("MsgDeposit", gs.GovGasConfig.Deposit)
	govConfig.SetMsgGasRate("MsgVote", gs.GovGasConfig.Vote)
	govConfig.SetMsgGasRate("MsgVoteWeighted", gs.GovGasConfig.VoteWeighted)
	configs = append(configs, govConfig)

	// Slashing module
	slashingConfig := feetypes.NewModuleGasConfig("slashing", gs.SlashingGasConfig.BaseGas, true)
	slashingConfig.SetMsgGasRate("MsgUnjail", gs.SlashingGasConfig.Unjail)
	slashingConfig.SetMsgGasRate("MsgUpdateParams", gs.SlashingGasConfig.UpdateParams)
	configs = append(configs, slashingConfig)

	return configs
}

// CalculateDynamicGas calculates dynamic gas cost based on complexity factors
func (gs GasStandards) CalculateDynamicGas(moduleName, msgType string, factors DynamicGasFactors, additionalParams map[string]interface{}) uint64 {
	baseGas := gs.getBaseGasForMessage(moduleName, msgType)

	// Apply complexity factors
	complexityMultiplier := factors.ComplexityFactor
	sizeMultiplier := factors.SizeMultiplier
	networkMultiplier := factors.NetworkFactor
	storageMultiplier := factors.StorageFactor

	// Apply additional parameter-based costs
	additionalGas := gs.calculateAdditionalGas(moduleName, msgType, additionalParams)

	// Calculate final gas cost
	finalGas := float64(baseGas+additionalGas) * complexityMultiplier * sizeMultiplier * networkMultiplier

	// Apply storage multiplier for state-changing operations
	if gs.isStorageOperation(moduleName, msgType) {
		finalGas *= storageMultiplier
	}

	return uint64(finalGas)
}

// DynamicGasFactors represents factors for dynamic gas calculation
type DynamicGasFactors struct {
	ComplexityFactor float64 `json:"complexity_factor"`
	SizeMultiplier   float64 `json:"size_multiplier"`
	NetworkFactor    float64 `json:"network_factor"`
	StorageFactor    float64 `json:"storage_factor"`
}

// getBaseGasForMessage returns the base gas cost for a specific message
func (gs GasStandards) getBaseGasForMessage(moduleName, msgType string) uint64 {
	switch moduleName {
	case "bank":
		switch msgType {
		case "MsgSend":
			return gs.BankGasConfig.Send
		case "MsgMultiSend":
			return gs.BankGasConfig.MultiSend
		default:
			return gs.BankGasConfig.BaseGas
		}
	case "coin":
		switch msgType {
		case "MsgMint":
			return gs.CoinGasConfig.Mint
		case "MsgBurn":
			return gs.CoinGasConfig.Burn
		case "MsgSetPermissions":
			return gs.CoinGasConfig.SetPermissions
		case "MsgSetMetadata":
			return gs.CoinGasConfig.SetMetadata
		default:
			return gs.CoinGasConfig.BaseGas
		}
	case "staking":
		switch msgType {
		case "MsgCreateValidator":
			return gs.StakingGasConfig.CreateValidator
		case "MsgDelegate":
			return gs.StakingGasConfig.Delegate
		case "MsgUndelegate":
			return gs.StakingGasConfig.Undelegate
		case "MsgRedelegate":
			return gs.StakingGasConfig.Redelegate
		default:
			return gs.StakingGasConfig.BaseGas
		}
	default:
		return 25000 // Default gas for unknown modules
	}
}

// calculateAdditionalGas calculates additional gas based on message parameters
func (gs GasStandards) calculateAdditionalGas(moduleName, msgType string, params map[string]interface{}) uint64 {
	additionalGas := uint64(0)

	switch moduleName {
	case "bank":
		if msgType == "MsgMultiSend" {
			if inputs, ok := params["inputs"].(int); ok {
				additionalGas += uint64(inputs) * gs.BankGasConfig.MultiSendPerInput
			}
			if outputs, ok := params["outputs"].(int); ok {
				additionalGas += uint64(outputs) * gs.BankGasConfig.MultiSendPerOutput
			}
		}
	case "coin":
		if msgType == "MsgMint" || msgType == "MsgBurn" {
			if amount, ok := params["amount"].(uint64); ok {
				if msgType == "MsgMint" {
					additionalGas += amount * gs.CoinGasConfig.MintPerUnit
				} else {
					additionalGas += amount * gs.CoinGasConfig.BurnPerUnit
				}
			}
		}
	case "staking":
		if amount, ok := params["amount"].(uint64); ok {
			switch msgType {
			case "MsgDelegate":
				additionalGas += amount * gs.StakingGasConfig.DelegatePerToken
			case "MsgUndelegate":
				additionalGas += amount * gs.StakingGasConfig.UndelegatePerToken
			case "MsgRedelegate":
				additionalGas += amount * gs.StakingGasConfig.RedelegatePerToken
			}
		}
	case "distribution":
		if msgType == "MsgWithdrawDelegatorReward" {
			if validators, ok := params["validators"].(int); ok {
				additionalGas += uint64(validators) * gs.DistributionGasConfig.WithdrawRewardPerValidator
			}
		}
	case "gov":
		if msgType == "MsgVoteWeighted" {
			if options, ok := params["options"].(int); ok {
				additionalGas += uint64(options) * gs.GovGasConfig.VotePerOption
			}
		}
	}

	return additionalGas
}

// isStorageOperation determines if an operation involves significant storage
func (gs GasStandards) isStorageOperation(moduleName, msgType string) bool {
	storageOperations := map[string][]string{
		"coin": {"MsgSetMetadata", "MsgCreateDenomination", "MsgSetPermissions"},
		"fee":  {"MsgAddFeeDenom", "MsgSetModuleGasConfig", "MsgSetFeeDistribution"},
		"staking": {"MsgCreateValidator", "MsgEditValidator"},
		"auth": {"MsgCreateAccount", "MsgSetAccountParams"},
		"gov":  {"MsgSubmitProposal"},
	}

	if operations, exists := storageOperations[moduleName]; exists {
		for _, op := range operations {
			if op == msgType {
				return true
			}
		}
	}

	return false
}

// ValidateGasStandards validates the gas standards configuration
func (gs GasStandards) ValidateGasStandards() error {
	// Validate base costs
	if gs.BaseTxGas == 0 {
		return fmt.Errorf("base transaction gas cannot be zero")
	}
	if gs.SignatureGas == 0 {
		return fmt.Errorf("signature gas cannot be zero")
	}

	// Validate module configurations
	moduleConfigs := []struct {
		name   string
		config interface{ ValidateBasic() error }
	}{
		// Note: These would implement ValidateBasic if needed
		// {"bank", gs.BankGasConfig},
		// {"coin", gs.CoinGasConfig},
		// Add more as needed
	}

	for _, config := range moduleConfigs {
		if err := config.config.ValidateBasic(); err != nil {
			return fmt.Errorf("invalid %s configuration: %w", config.name, err)
		}
	}

	return nil
}

// GetGasCostSummary returns a summary of gas costs for different operations
func (gs GasStandards) GetGasCostSummary() map[string]map[string]uint64 {
	summary := make(map[string]map[string]uint64)

	summary["base"] = map[string]uint64{
		"base_tx":         gs.BaseTxGas,
		"signature":       gs.SignatureGas,
		"tx_size_per_byte": gs.TxSizePerByteGas,
	}

	summary["bank"] = map[string]uint64{
		"base":      gs.BankGasConfig.BaseGas,
		"send":      gs.BankGasConfig.Send,
		"multisend": gs.BankGasConfig.MultiSend,
	}

	summary["coin"] = map[string]uint64{
		"base":         gs.CoinGasConfig.BaseGas,
		"mint":         gs.CoinGasConfig.Mint,
		"burn":         gs.CoinGasConfig.Burn,
		"permissions":  gs.CoinGasConfig.SetPermissions,
		"metadata":     gs.CoinGasConfig.SetMetadata,
	}

	summary["staking"] = map[string]uint64{
		"base":            gs.StakingGasConfig.BaseGas,
		"create_validator": gs.StakingGasConfig.CreateValidator,
		"delegate":        gs.StakingGasConfig.Delegate,
		"undelegate":      gs.StakingGasConfig.Undelegate,
		"redelegate":      gs.StakingGasConfig.Redelegate,
	}

	summary["gov"] = map[string]uint64{
		"base":            gs.GovGasConfig.BaseGas,
		"submit_proposal": gs.GovGasConfig.SubmitProposal,
		"vote":            gs.GovGasConfig.Vote,
		"deposit":         gs.GovGasConfig.Deposit,
	}

	return summary
}