package keeper

import (
	"fmt"
	"math/big"
	
	keepertypes "github.com/b2network/pulsar/keeper/types"
	"github.com/b2network/pulsar/modules/bank/types"
	commontypes "github.com/b2network/pulsar/types"
)

// SupplyKeeper manages coin supply
type SupplyKeeper struct {
	keeper *Keeper
}

// NewSupplyKeeper creates a new supply keeper
func NewSupplyKeeper(k *Keeper) *SupplyKeeper {
	return &SupplyKeeper{
		keeper: k,
	}
}

// GetSupply returns the total supply of a denomination
func (k *SupplyKeeper) GetSupply(ctx keepertypes.Context, denom string) *types.CoinSupply {
	key := fmt.Sprintf("supply/%s", denom)
	
	// TODO: Retrieve from KVStore
	_ = key
	
	// For now, return zero supply
	return &types.CoinSupply{
		Denom:  denom,
		Supply: "0",
	}
}

// GetTotalSupply returns the total supply of all denominations
func (k *SupplyKeeper) GetTotalSupply(ctx keepertypes.Context) []*types.CoinSupply {
	supplies := []*types.CoinSupply{}
	
	// TODO: Iterate through all supply records in KVStore
	// For now, return supplies for known denoms
	
	supplies = append(supplies, k.GetSupply(ctx, "ubtc"))
	supplies = append(supplies, k.GetSupply(ctx, "wei"))
	
	return supplies
}

// MintCoins creates new coins and adds them to supply
func (k *SupplyKeeper) MintCoins(ctx keepertypes.Context, moduleName string, coins []commontypes.Coin) error {
	// Check minting permissions
	if !k.HasMintPermission(ctx, moduleName) {
		return fmt.Errorf("module %s does not have mint permission", moduleName)
	}
	
	for _, coin := range coins {
		// Update supply
		if err := k.increaseSupply(ctx, coin.Denom, coin.Amount); err != nil {
			return err
		}
		
		// Emit event
		ctx.EventManager().EmitEvent(
			keepertypes.Event{
				Type: types.EventTypeCoinMint,
				Attributes: []keepertypes.Attribute{
					{Key: "minter", Value: moduleName},
					{Key: "denom", Value: coin.Denom},
					{Key: "amount", Value: coin.Amount},
				},
			},
		)
	}
	
	return nil
}

// BurnCoins destroys coins and removes them from supply
func (k *SupplyKeeper) BurnCoins(ctx keepertypes.Context, moduleName string, coins []commontypes.Coin) error {
	// Check burning permissions
	if !k.HasBurnPermission(ctx, moduleName) {
		return fmt.Errorf("module %s does not have burn permission", moduleName)
	}
	
	for _, coin := range coins {
		// Update supply
		if err := k.decreaseSupply(ctx, coin.Denom, coin.Amount); err != nil {
			return err
		}
		
		// Emit event
		ctx.EventManager().EmitEvent(
			keepertypes.Event{
				Type: types.EventTypeCoinBurn,
				Attributes: []keepertypes.Attribute{
					{Key: "burner", Value: moduleName},
					{Key: "denom", Value: coin.Denom},
					{Key: "amount", Value: coin.Amount},
				},
			},
		)
	}
	
	return nil
}

// increaseSupply increases the supply of a denomination
func (k *SupplyKeeper) increaseSupply(ctx keepertypes.Context, denom string, amount string) error {
	supply := k.GetSupply(ctx, denom)
	
	// Parse amounts
	currentSupply, ok := new(big.Int).SetString(supply.Supply, 10)
	if !ok {
		currentSupply = big.NewInt(0)
	}
	
	addAmount, ok := new(big.Int).SetString(amount, 10)
	if !ok {
		return fmt.Errorf("invalid amount: %s", amount)
	}
	
	// Calculate new supply
	newSupply := new(big.Int).Add(currentSupply, addAmount)
	
	// Update supply
	supply.Supply = newSupply.String()
	
	// Store updated supply
	key := fmt.Sprintf("supply/%s", denom)
	// TODO: Store in KVStore
	_ = key
	
	return nil
}

// decreaseSupply decreases the supply of a denomination
func (k *SupplyKeeper) decreaseSupply(ctx keepertypes.Context, denom string, amount string) error {
	supply := k.GetSupply(ctx, denom)
	
	// Parse amounts
	currentSupply, ok := new(big.Int).SetString(supply.Supply, 10)
	if !ok {
		currentSupply = big.NewInt(0)
	}
	
	subAmount, ok := new(big.Int).SetString(amount, 10)
	if !ok {
		return fmt.Errorf("invalid amount: %s", amount)
	}
	
	// Check for sufficient supply
	if currentSupply.Cmp(subAmount) < 0 {
		return fmt.Errorf("insufficient supply: have %s, trying to burn %s", supply.Supply, amount)
	}
	
	// Calculate new supply
	newSupply := new(big.Int).Sub(currentSupply, subAmount)
	
	// Update supply
	supply.Supply = newSupply.String()
	
	// Store updated supply
	key := fmt.Sprintf("supply/%s", denom)
	// TODO: Store in KVStore
	_ = key
	
	return nil
}

// HasMintPermission checks if a module has permission to mint coins
func (k *SupplyKeeper) HasMintPermission(ctx keepertypes.Context, moduleName string) bool {
	// TODO: Check permissions from configuration or governance
	// For now, allow specific modules
	switch moduleName {
	case "mint", "distribution", "staking":
		return true
	default:
		return false
	}
}

// HasBurnPermission checks if a module has permission to burn coins
func (k *SupplyKeeper) HasBurnPermission(ctx keepertypes.Context, moduleName string) bool {
	// TODO: Check permissions from configuration or governance
	// For now, allow specific modules
	switch moduleName {
	case "slashing", "governance":
		return true
	default:
		return false
	}
}

// SetSupply directly sets the supply of a denomination (admin only)
func (k *SupplyKeeper) SetSupply(ctx keepertypes.Context, supply *types.CoinSupply) error {
	// TODO: Check admin permissions
	
	key := fmt.Sprintf("supply/%s", supply.Denom)
	// TODO: Store in KVStore
	_ = key
	
	return nil
}