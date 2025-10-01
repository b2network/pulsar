package types

import (
	"encoding/json"
	"fmt"
	"strconv"
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

// Add adds two coins collections together
func (coins Coins) Add(coinsB ...Coin) Coins {
	result := make(Coins, len(coins))
	copy(result, coins)

	for _, coinB := range coinsB {
		result = result.add(coinB)
	}

	return result
}

// add adds a single coin to the collection
func (coins Coins) add(coin Coin) Coins {
	for i, existingCoin := range coins {
		if existingCoin.Denom == coin.Denom {
			// Add amounts together
			amount1, _ := strconv.ParseFloat(existingCoin.Amount, 64)
			amount2, _ := strconv.ParseFloat(coin.Amount, 64)
			total := amount1 + amount2
			coins[i].Amount = fmt.Sprintf("%.0f", total)
			return coins
		}
	}

	// Add new coin if denomination not found
	return append(coins, coin)
}

// MarshalJSON marshals coins to JSON
func MarshalJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// UnmarshalJSON unmarshals JSON to the provided interface
func UnmarshalJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// ParseCoinAmount parses a coin amount string to float64
func ParseCoinAmount(amount string) (float64, error) {
	if amount == "" {
		return 0, nil
	}
	return strconv.ParseFloat(amount, 64)
}
