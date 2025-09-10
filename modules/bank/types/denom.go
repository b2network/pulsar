package types

import (
	"fmt"
	"strings"
)

// DenomMetadata represents the metadata of a coin denomination
type DenomMetadata struct {
	// Basic Information
	Denom       string `json:"denom"`       // Base denomination (e.g., ubtc, wei)
	Display     string `json:"display"`     // Display denomination (e.g., btc, eth)
	Name        string `json:"name"`        // Full name (e.g., Bitcoin, Ethereum)
	Symbol      string `json:"symbol"`      // Symbol (e.g., BTC, ETH)
	Description string `json:"description"` // Description of the coin

	// Units and Precision
	DenomUnits []DenomUnit `json:"denom_units"` // Different units of the coin
	Base       string      `json:"base"`        // Base unit denom
	
	// Display preferences
	DisplayUnit string `json:"display_unit"` // Default display unit
}

// DenomUnit represents a unit of a denomination
type DenomUnit struct {
	Denom    string   `json:"denom"`    // Unit name (e.g., btc, satoshi)
	Exponent uint32   `json:"exponent"` // Power of 10 exponent (e.g., 8 for btc->satoshi)
	Aliases  []string `json:"aliases"`  // Alternative names
}

// CoinSupply tracks the supply of a specific denomination
type CoinSupply struct {
	Denom  string `json:"denom"`  // Coin denomination
	Supply string `json:"supply"` // Current total supply
}

// DenomPermissions defines permissions for a denomination
type DenomPermissions struct {
	Denom  string `json:"denom"`  // Coin denomination
	Owner  string `json:"owner"`  // Owner address (can modify metadata)
	
	// Permission lists
	CanMint   []string `json:"can_mint"`   // Addresses that can mint
	CanBurn   []string `json:"can_burn"`   // Addresses that can burn
	CanFreeze []string `json:"can_freeze"` // Addresses that can freeze accounts
	
	// Flags
	Mintable bool `json:"mintable"` // Whether minting is allowed
	Burnable bool `json:"burnable"` // Whether burning is allowed
}

// ValidateDenomMetadata validates denomination metadata
func ValidateDenomMetadata(metadata *DenomMetadata) error {
	if metadata.Denom == "" {
		return fmt.Errorf("denom cannot be empty")
	}
	
	if metadata.Base == "" {
		metadata.Base = metadata.Denom
	}
	
	// Validate denom format (lowercase alphanumeric)
	if !isValidDenom(metadata.Denom) {
		return fmt.Errorf("invalid denom format: %s", metadata.Denom)
	}
	
	// Validate units
	if len(metadata.DenomUnits) == 0 {
		return fmt.Errorf("at least one denom unit is required")
	}
	
	return nil
}

// isValidDenom checks if a denomination string is valid
func isValidDenom(denom string) bool {
	if denom == "" {
		return false
	}
	
	// Check for valid characters (lowercase letters, numbers, '/')
	for _, r := range denom {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '/') {
			return false
		}
	}
	
	// Must start with a letter
	if denom[0] < 'a' || denom[0] > 'z' {
		return false
	}
	
	return true
}

// ParseDenomAmount parses amount with denomination
func ParseDenomAmount(str string) (amount string, denom string, err error) {
	str = strings.TrimSpace(str)
	if str == "" {
		return "", "", fmt.Errorf("empty string")
	}
	
	// Find where the number ends and denom begins
	var i int
	for i = 0; i < len(str); i++ {
		if !((str[i] >= '0' && str[i] <= '9') || str[i] == '.') {
			break
		}
	}
	
	if i == 0 {
		return "", "", fmt.Errorf("missing amount")
	}
	
	if i == len(str) {
		return "", "", fmt.Errorf("missing denomination")
	}
	
	amount = str[:i]
	denom = str[i:]
	
	return amount, denom, nil
}

// FormatCoinWithMetadata formats a coin using its metadata
func FormatCoinWithMetadata(amount string, metadata *DenomMetadata) string {
	if metadata == nil || metadata.Display == "" {
		return amount + metadata.Denom
	}
	
	// TODO: Convert to display unit based on metadata
	// For now, just return with display denom
	return amount + " " + metadata.Display
}