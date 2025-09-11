package types

import (
	"fmt"
	"math/big"

	commontypes "github.com/b2network/pulsar/types"
)

// Supply represents the supply of a coin denomination
type Supply struct {
	Denom  string `json:"denom"`
	Amount string `json:"amount"`
}

// NewSupply creates a new Supply
func NewSupply(denom, amount string) Supply {
	return Supply{
		Denom:  denom,
		Amount: amount,
	}
}

// ValidateBasic performs basic validation on Supply
func (s Supply) ValidateBasic() error {
	if s.Denom == "" {
		return fmt.Errorf("denom cannot be empty")
	}
	
	// Validate amount is a valid big integer
	_, ok := new(big.Int).SetString(s.Amount, 10)
	if !ok {
		return fmt.Errorf("invalid amount: %s", s.Amount)
	}
	
	return nil
}

// IsZero checks if supply is zero
func (s Supply) IsZero() bool {
	amount, ok := new(big.Int).SetString(s.Amount, 10)
	if !ok {
		return true
	}
	return amount.Sign() == 0
}

// Add adds the given amount to the supply
func (s Supply) Add(amount string) (Supply, error) {
	currentAmount, ok := new(big.Int).SetString(s.Amount, 10)
	if !ok {
		return Supply{}, fmt.Errorf("invalid current amount: %s", s.Amount)
	}
	
	addAmount, ok := new(big.Int).SetString(amount, 10)
	if !ok {
		return Supply{}, fmt.Errorf("invalid add amount: %s", amount)
	}
	
	newAmount := new(big.Int).Add(currentAmount, addAmount)
	return Supply{
		Denom:  s.Denom,
		Amount: newAmount.String(),
	}, nil
}

// Sub subtracts the given amount from the supply
func (s Supply) Sub(amount string) (Supply, error) {
	currentAmount, ok := new(big.Int).SetString(s.Amount, 10)
	if !ok {
		return Supply{}, fmt.Errorf("invalid current amount: %s", s.Amount)
	}
	
	subAmount, ok := new(big.Int).SetString(amount, 10)
	if !ok {
		return Supply{}, fmt.Errorf("invalid sub amount: %s", amount)
	}
	
	if currentAmount.Cmp(subAmount) < 0 {
		return Supply{}, fmt.Errorf("insufficient supply: have %s, trying to subtract %s", s.Amount, amount)
	}
	
	newAmount := new(big.Int).Sub(currentAmount, subAmount)
	return Supply{
		Denom:  s.Denom,
		Amount: newAmount.String(),
	}, nil
}

// SupplyI defines the interface for coin supply
type SupplyI interface {
	GetDenom() string
	GetAmount() string
	ValidateBasic() error
}

// Ensure Supply implements SupplyI
var _ SupplyI = (*Supply)(nil)

func (s Supply) GetDenom() string  { return s.Denom }
func (s Supply) GetAmount() string { return s.Amount }

// MintRequest represents a request to mint coins
type MintRequest struct {
	Module string              `json:"module"`
	Coins  []commontypes.Coin `json:"coins"`
}

// BurnRequest represents a request to burn coins
type BurnRequest struct {
	Module string              `json:"module"`
	Coins  []commontypes.Coin `json:"coins"`
}