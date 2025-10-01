package types

import (
	"encoding/json"
	"fmt"
)

// Message types for fee module
const (
	TypeMsgAddFeeDenom         = "add_fee_denom"
	TypeMsgRemoveFeeDenom      = "remove_fee_denom"
	TypeMsgUpdateFeeDenom      = "update_fee_denom"
	TypeMsgSetModuleGasConfig  = "set_module_gas_config"
	TypeMsgUpdateGasFactors    = "update_gas_factors"
	TypeMsgSetFeeDistribution  = "set_fee_distribution"
)

// MsgAddFeeDenom defines a message to add a new fee denomination
type MsgAddFeeDenom struct {
	Authority string    `json:"authority"`
	FeeDenom  FeeDenom  `json:"fee_denom"`
}

// NewMsgAddFeeDenom creates a new MsgAddFeeDenom instance
func NewMsgAddFeeDenom(authority string, feeDenom FeeDenom) *MsgAddFeeDenom {
	return &MsgAddFeeDenom{
		Authority: authority,
		FeeDenom:  feeDenom,
	}
}

// Route returns the route for MsgAddFeeDenom
func (msg MsgAddFeeDenom) Route() string { return RouterKey }

// Type returns the type for MsgAddFeeDenom
func (msg MsgAddFeeDenom) Type() string { return TypeMsgAddFeeDenom }

// ValidateBasic performs basic validation for MsgAddFeeDenom
func (msg MsgAddFeeDenom) ValidateBasic() error {
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	return msg.FeeDenom.ValidateBasic()
}

// GetSigners returns the signers for MsgAddFeeDenom
func (msg MsgAddFeeDenom) GetSigners() []string {
	return []string{msg.Authority}
}

// MsgRemoveFeeDenom defines a message to remove a fee denomination
type MsgRemoveFeeDenom struct {
	Authority string `json:"authority"`
	Denom     string `json:"denom"`
}

// NewMsgRemoveFeeDenom creates a new MsgRemoveFeeDenom instance
func NewMsgRemoveFeeDenom(authority, denom string) *MsgRemoveFeeDenom {
	return &MsgRemoveFeeDenom{
		Authority: authority,
		Denom:     denom,
	}
}

// Route returns the route for MsgRemoveFeeDenom
func (msg MsgRemoveFeeDenom) Route() string { return RouterKey }

// Type returns the type for MsgRemoveFeeDenom
func (msg MsgRemoveFeeDenom) Type() string { return TypeMsgRemoveFeeDenom }

// ValidateBasic performs basic validation for MsgRemoveFeeDenom
func (msg MsgRemoveFeeDenom) ValidateBasic() error {
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if msg.Denom == "" {
		return fmt.Errorf("denom cannot be empty")
	}
	return nil
}

// GetSigners returns the signers for MsgRemoveFeeDenom
func (msg MsgRemoveFeeDenom) GetSigners() []string {
	return []string{msg.Authority}
}

// MsgUpdateFeeDenom defines a message to update a fee denomination
type MsgUpdateFeeDenom struct {
	Authority string    `json:"authority"`
	FeeDenom  FeeDenom  `json:"fee_denom"`
}

// NewMsgUpdateFeeDenom creates a new MsgUpdateFeeDenom instance
func NewMsgUpdateFeeDenom(authority string, feeDenom FeeDenom) *MsgUpdateFeeDenom {
	return &MsgUpdateFeeDenom{
		Authority: authority,
		FeeDenom:  feeDenom,
	}
}

// Route returns the route for MsgUpdateFeeDenom
func (msg MsgUpdateFeeDenom) Route() string { return RouterKey }

// Type returns the type for MsgUpdateFeeDenom
func (msg MsgUpdateFeeDenom) Type() string { return TypeMsgUpdateFeeDenom }

// ValidateBasic performs basic validation for MsgUpdateFeeDenom
func (msg MsgUpdateFeeDenom) ValidateBasic() error {
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	return msg.FeeDenom.ValidateBasic()
}

// GetSigners returns the signers for MsgUpdateFeeDenom
func (msg MsgUpdateFeeDenom) GetSigners() []string {
	return []string{msg.Authority}
}

// MsgSetModuleGasConfig defines a message to set gas configuration for a module
type MsgSetModuleGasConfig struct {
	Authority string            `json:"authority"`
	Config    ModuleGasConfig   `json:"config"`
}

// NewMsgSetModuleGasConfig creates a new MsgSetModuleGasConfig instance
func NewMsgSetModuleGasConfig(authority string, config ModuleGasConfig) *MsgSetModuleGasConfig {
	return &MsgSetModuleGasConfig{
		Authority: authority,
		Config:    config,
	}
}

// Route returns the route for MsgSetModuleGasConfig
func (msg MsgSetModuleGasConfig) Route() string { return RouterKey }

// Type returns the type for MsgSetModuleGasConfig
func (msg MsgSetModuleGasConfig) Type() string { return TypeMsgSetModuleGasConfig }

// ValidateBasic performs basic validation for MsgSetModuleGasConfig
func (msg MsgSetModuleGasConfig) ValidateBasic() error {
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	return msg.Config.ValidateBasic()
}

// GetSigners returns the signers for MsgSetModuleGasConfig
func (msg MsgSetModuleGasConfig) GetSigners() []string {
	return []string{msg.Authority}
}

// MsgUpdateGasFactors defines a message to update dynamic gas factors
type MsgUpdateGasFactors struct {
	Authority string            `json:"authority"`
	Factors   DynamicGasFactors `json:"factors"`
}

// NewMsgUpdateGasFactors creates a new MsgUpdateGasFactors instance
func NewMsgUpdateGasFactors(authority string, factors DynamicGasFactors) *MsgUpdateGasFactors {
	return &MsgUpdateGasFactors{
		Authority: authority,
		Factors:   factors,
	}
}

// Route returns the route for MsgUpdateGasFactors
func (msg MsgUpdateGasFactors) Route() string { return RouterKey }

// Type returns the type for MsgUpdateGasFactors
func (msg MsgUpdateGasFactors) Type() string { return TypeMsgUpdateGasFactors }

// ValidateBasic performs basic validation for MsgUpdateGasFactors
func (msg MsgUpdateGasFactors) ValidateBasic() error {
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	return msg.Factors.ValidateBasic()
}

// GetSigners returns the signers for MsgUpdateGasFactors
func (msg MsgUpdateGasFactors) GetSigners() []string {
	return []string{msg.Authority}
}

// MsgSetFeeDistribution defines a message to set fee distribution configuration
type MsgSetFeeDistribution struct {
	Authority string                 `json:"authority"`
	Config    FeeDistributionConfig  `json:"config"`
}

// NewMsgSetFeeDistribution creates a new MsgSetFeeDistribution instance
func NewMsgSetFeeDistribution(authority string, config FeeDistributionConfig) *MsgSetFeeDistribution {
	return &MsgSetFeeDistribution{
		Authority: authority,
		Config:    config,
	}
}

// Route returns the route for MsgSetFeeDistribution
func (msg MsgSetFeeDistribution) Route() string { return RouterKey }

// Type returns the type for MsgSetFeeDistribution
func (msg MsgSetFeeDistribution) Type() string { return TypeMsgSetFeeDistribution }

// ValidateBasic performs basic validation for MsgSetFeeDistribution
func (msg MsgSetFeeDistribution) ValidateBasic() error {
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	return msg.Config.ValidateBasic()
}

// GetSigners returns the signers for MsgSetFeeDistribution
func (msg MsgSetFeeDistribution) GetSigners() []string {
	return []string{msg.Authority}
}

// GetSignBytes returns the raw bytes for a message
func GetSignBytes(msg interface{}) ([]byte, error) {
	return json.Marshal(msg)
}