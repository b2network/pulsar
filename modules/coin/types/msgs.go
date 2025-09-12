package types

import (
	"fmt"
	
	keepertypes "github.com/b2network/pulsar/keeper/types"
)

// Message types for coin module
const (
	TypeMsgMintCoins      = "mint_coins"
	TypeMsgBurnCoins      = "burn_coins"
	TypeMsgSetMetadata    = "set_metadata"
	TypeMsgSetPermissions = "set_permissions"
)

// MsgMintCoins defines a message to mint coins
type MsgMintCoins struct {
	Authority string                `json:"authority"`
	Module    string                `json:"module"`
	Amount    keepertypes.Coins     `json:"amount"`
}

// NewMsgMintCoins creates a new MsgMintCoins instance
func NewMsgMintCoins(authority, module string, amount keepertypes.Coins) *MsgMintCoins {
	return &MsgMintCoins{
		Authority: authority,
		Module:    module,
		Amount:    amount,
	}
}

// Route returns the route for MsgMintCoins
func (msg MsgMintCoins) Route() string { return ModuleName }

// Type returns the type for MsgMintCoins
func (msg MsgMintCoins) Type() string { return TypeMsgMintCoins }

// ValidateBasic performs basic validation for MsgMintCoins
func (msg MsgMintCoins) ValidateBasic() error {
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if msg.Module == "" {
		return fmt.Errorf("module cannot be empty")
	}
	if len(msg.Amount) == 0 {
		return fmt.Errorf("amount cannot be empty")
	}
	for _, coin := range msg.Amount {
		if coin.Denom == "" {
			return fmt.Errorf("denom cannot be empty")
		}
		if coin.Amount <= 0 {
			return fmt.Errorf("amount must be positive")
		}
	}
	return nil
}

// GetSigners returns the signers for MsgMintCoins
func (msg MsgMintCoins) GetSigners() []string {
	return []string{msg.Authority}
}

// MsgBurnCoins defines a message to burn coins
type MsgBurnCoins struct {
	Authority string                `json:"authority"`
	Module    string                `json:"module"`
	Amount    keepertypes.Coins     `json:"amount"`
}

// NewMsgBurnCoins creates a new MsgBurnCoins instance
func NewMsgBurnCoins(authority, module string, amount keepertypes.Coins) *MsgBurnCoins {
	return &MsgBurnCoins{
		Authority: authority,
		Module:    module,
		Amount:    amount,
	}
}

// Route returns the route for MsgBurnCoins
func (msg MsgBurnCoins) Route() string { return ModuleName }

// Type returns the type for MsgBurnCoins
func (msg MsgBurnCoins) Type() string { return TypeMsgBurnCoins }

// ValidateBasic performs basic validation for MsgBurnCoins
func (msg MsgBurnCoins) ValidateBasic() error {
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if msg.Module == "" {
		return fmt.Errorf("module cannot be empty")
	}
	if len(msg.Amount) == 0 {
		return fmt.Errorf("amount cannot be empty")
	}
	for _, coin := range msg.Amount {
		if coin.Denom == "" {
			return fmt.Errorf("denom cannot be empty")
		}
		if coin.Amount <= 0 {
			return fmt.Errorf("amount must be positive")
		}
	}
	return nil
}

// GetSigners returns the signers for MsgBurnCoins
func (msg MsgBurnCoins) GetSigners() []string {
	return []string{msg.Authority}
}

// MsgSetMetadata defines a message to set coin metadata
type MsgSetMetadata struct {
	Authority string   `json:"authority"`
	Metadata  Metadata `json:"metadata"`
}

// NewMsgSetMetadata creates a new MsgSetMetadata instance
func NewMsgSetMetadata(authority string, metadata Metadata) *MsgSetMetadata {
	return &MsgSetMetadata{
		Authority: authority,
		Metadata:  metadata,
	}
}

// Route returns the route for MsgSetMetadata
func (msg MsgSetMetadata) Route() string { return ModuleName }

// Type returns the type for MsgSetMetadata
func (msg MsgSetMetadata) Type() string { return TypeMsgSetMetadata }

// ValidateBasic performs basic validation for MsgSetMetadata
func (msg MsgSetMetadata) ValidateBasic() error {
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	return msg.Metadata.ValidateBasic()
}

// GetSigners returns the signers for MsgSetMetadata
func (msg MsgSetMetadata) GetSigners() []string {
	return []string{msg.Authority}
}

// MsgSetPermissions defines a message to set module permissions
type MsgSetPermissions struct {
	Authority   string            `json:"authority"`
	ModuleName  string            `json:"module_name"`
	Permissions []string          `json:"permissions"`
}

// NewMsgSetPermissions creates a new MsgSetPermissions instance
func NewMsgSetPermissions(authority, moduleName string, permissions []string) *MsgSetPermissions {
	return &MsgSetPermissions{
		Authority:   authority,
		ModuleName:  moduleName,
		Permissions: permissions,
	}
}

// Route returns the route for MsgSetPermissions
func (msg MsgSetPermissions) Route() string { return ModuleName }

// Type returns the type for MsgSetPermissions
func (msg MsgSetPermissions) Type() string { return TypeMsgSetPermissions }

// ValidateBasic performs basic validation for MsgSetPermissions
func (msg MsgSetPermissions) ValidateBasic() error {
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if msg.ModuleName == "" {
		return fmt.Errorf("module name cannot be empty")
	}
	return nil
}

// GetSigners returns the signers for MsgSetPermissions
func (msg MsgSetPermissions) GetSigners() []string {
	return []string{msg.Authority}
}