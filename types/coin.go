package types

import (
	"fmt"
	"strings"
)

// Coin represents a single coin with denomination and amount
type Coin struct {
	Denom  string `json:"denom"`
	Amount string `json:"amount"`
}

// NewCoin creates a new coin with given denomination and amount
func NewCoin(denom, amount string) Coin {
	return Coin{
		Denom:  denom,
		Amount: amount,
	}
}

// IsValid checks if the coin is valid
func (c Coin) IsValid() bool {
	return c.Denom != "" && c.Amount != ""
}

// String returns string representation of the coin
func (c Coin) String() string {
	return fmt.Sprintf("%s%s", c.Amount, c.Denom)
}

// Coins represents a collection of coins
type Coins []Coin

// NewCoins creates a new coins collection
func NewCoins(coins ...Coin) Coins {
	return Coins(coins)
}

// IsValid checks if all coins are valid
func (coins Coins) IsValid() bool {
	for _, coin := range coins {
		if !coin.IsValid() {
			return false
		}
	}
	return true
}

// String returns string representation of the coins
func (coins Coins) String() string {
	if len(coins) == 0 {
		return ""
	}

	var parts []string
	for _, coin := range coins {
		parts = append(parts, coin.String())
	}
	return strings.Join(parts, ",")
}
