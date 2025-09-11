package types

import (
	"fmt"
	"strings"
)

// DenomUnit represents a denomination unit with display info
type DenomUnit struct {
	Denom    string   `json:"denom"`
	Exponent uint32   `json:"exponent"`
	Aliases  []string `json:"aliases"`
}

// Metadata represents coin denomination metadata
type Metadata struct {
	Description string      `json:"description"`
	DenomUnits  []DenomUnit `json:"denom_units"`
	Base        string      `json:"base"`        // Base denom (e.g., "uatom")
	Display     string      `json:"display"`     // Display denom (e.g., "atom")
	Name        string      `json:"name"`        // Long form name (e.g., "Cosmos Hub Atom")
	Symbol      string      `json:"symbol"`      // Token symbol (e.g., "ATOM")
	URI         string      `json:"uri"`         // URI to icon/logo (optional)
	URIHash     string      `json:"uri_hash"`    // Hash of URI content (optional)
}

// NewMetadata creates a new Metadata
func NewMetadata(description, base, display, name, symbol string, denomUnits []DenomUnit) Metadata {
	return Metadata{
		Description: description,
		DenomUnits:  denomUnits,
		Base:        base,
		Display:     display,
		Name:        name,
		Symbol:      symbol,
	}
}

// ValidateBasic performs basic validation on Metadata
func (m Metadata) ValidateBasic() error {
	if m.Base == "" {
		return fmt.Errorf("base denom cannot be empty")
	}
	
	if m.Display == "" {
		return fmt.Errorf("display denom cannot be empty")
	}
	
	if m.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	
	if m.Symbol == "" {
		return fmt.Errorf("symbol cannot be empty")
	}
	
	// Validate denom units
	if len(m.DenomUnits) == 0 {
		return fmt.Errorf("denom units cannot be empty")
	}
	
	// Check for base denom in denom units
	baseFound := false
	displayFound := false
	
	for i, unit := range m.DenomUnits {
		if unit.Denom == "" {
			return fmt.Errorf("denom unit at index %d has empty denom", i)
		}
		
		if unit.Denom == m.Base {
			baseFound = true
			if unit.Exponent != 0 {
				return fmt.Errorf("base denom %s must have exponent 0, got %d", m.Base, unit.Exponent)
			}
		}
		
		if unit.Denom == m.Display {
			displayFound = true
		}
		
		// Check for duplicate denoms
		for j := i + 1; j < len(m.DenomUnits); j++ {
			if unit.Denom == m.DenomUnits[j].Denom {
				return fmt.Errorf("duplicate denom %s in denom units", unit.Denom)
			}
		}
		
		// Validate aliases don't conflict with other denoms
		for _, alias := range unit.Aliases {
			if alias == "" {
				return fmt.Errorf("empty alias in denom unit %s", unit.Denom)
			}
			for _, otherUnit := range m.DenomUnits {
				if alias == otherUnit.Denom {
					return fmt.Errorf("alias %s conflicts with denom %s", alias, otherUnit.Denom)
				}
			}
		}
	}
	
	if !baseFound {
		return fmt.Errorf("base denom %s not found in denom units", m.Base)
	}
	
	if !displayFound {
		return fmt.Errorf("display denom %s not found in denom units", m.Display)
	}
	
	return nil
}

// GetBaseDenom returns the base denomination
func (m Metadata) GetBaseDenom() string {
	return m.Base
}

// GetDisplayDenom returns the display denomination
func (m Metadata) GetDisplayDenom() string {
	return m.Display
}

// GetDenomUnit returns the DenomUnit for a given denom
func (m Metadata) GetDenomUnit(denom string) (DenomUnit, bool) {
	for _, unit := range m.DenomUnits {
		if unit.Denom == denom {
			return unit, true
		}
		// Check aliases
		for _, alias := range unit.Aliases {
			if alias == denom {
				return unit, true
			}
		}
	}
	return DenomUnit{}, false
}

// String returns a string representation of the metadata
func (m Metadata) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Metadata: %s (%s)\n", m.Name, m.Symbol))
	sb.WriteString(fmt.Sprintf("  Base: %s, Display: %s\n", m.Base, m.Display))
	sb.WriteString(fmt.Sprintf("  Description: %s\n", m.Description))
	sb.WriteString("  Units:\n")
	for _, unit := range m.DenomUnits {
		sb.WriteString(fmt.Sprintf("    %s (10^%d)", unit.Denom, unit.Exponent))
		if len(unit.Aliases) > 0 {
			sb.WriteString(fmt.Sprintf(" aliases: %s", strings.Join(unit.Aliases, ", ")))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// MetadataI defines the interface for coin metadata
type MetadataI interface {
	GetBaseDenom() string
	GetDisplayDenom() string
	ValidateBasic() error
}

// Ensure Metadata implements MetadataI
var _ MetadataI = (*Metadata)(nil)