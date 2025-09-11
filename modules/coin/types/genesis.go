package types

import "fmt"

// GenesisState defines the coin module's genesis state
type GenesisState struct {
	Supplies    []Supply            `json:"supplies"`
	Metadatas   []Metadata          `json:"metadatas"`
	Permissions PermissionSet       `json:"permissions"`
}

// NewGenesisState creates a new genesis state
func NewGenesisState(supplies []Supply, metadatas []Metadata, permissions PermissionSet) *GenesisState {
	return &GenesisState{
		Supplies:    supplies,
		Metadatas:   metadatas,
		Permissions: permissions,
	}
}

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Supplies:    []Supply{},
		Metadatas:   []Metadata{},
		Permissions: DefaultPermissionSet(),
	}
}

// ValidateGenesis validates the coin module genesis state
func ValidateGenesis(data *GenesisState) error {
	// Validate supplies
	for i, supply := range data.Supplies {
		if err := supply.ValidateBasic(); err != nil {
			return fmt.Errorf("invalid supply at index %d: %w", i, err)
		}
	}
	
	// Validate metadatas
	seenDenoms := make(map[string]bool)
	for i, metadata := range data.Metadatas {
		if err := metadata.ValidateBasic(); err != nil {
			return fmt.Errorf("invalid metadata at index %d: %w", i, err)
		}
		
		// Check for duplicate base denoms
		if seenDenoms[metadata.Base] {
			return fmt.Errorf("duplicate metadata for denom %s", metadata.Base)
		}
		seenDenoms[metadata.Base] = true
	}
	
	// Validate permissions
	if err := data.Permissions.ValidateBasic(); err != nil {
		return fmt.Errorf("invalid permissions: %w", err)
	}
	
	return nil
}