package keeper

import (
	"fmt"
	"math/big"

	keepertypes "github.com/b2network/pulsar/keeper/types"
	"github.com/b2network/pulsar/modules/coin/types"
)

// SupplyKeeper manages coin supply operations
type SupplyKeeper struct {
	keeper *Keeper
}

// NewSupplyKeeper creates a new supply keeper
func NewSupplyKeeper(k *Keeper) *SupplyKeeper {
	return &SupplyKeeper{
		keeper: k,
	}
}

// GetSupply returns the supply of a specific denomination
func (k SupplyKeeper) GetSupply(ctx keepertypes.Context, denom string) types.Supply {
	store := k.keeper.GetKVStore(ctx)
	key := types.SupplyKey(denom)
	
	bz := store.Get(key)
	if bz == nil {
		// Return zero supply if not found
		return types.NewSupply(denom, "0")
	}
	
	var supply types.Supply
	if err := k.keeper.GetCodec().Unmarshal(bz, &supply); err != nil {
		// Return zero supply on unmarshal error
		return types.NewSupply(denom, "0")
	}
	
	return supply
}

// SetSupply sets the supply for a denomination
func (k SupplyKeeper) SetSupply(ctx keepertypes.Context, supply types.Supply) {
	store := k.keeper.GetKVStore(ctx)
	key := types.SupplyKey(supply.Denom)
	
	bz, err := k.keeper.GetCodec().Marshal(supply)
	if err != nil {
		panic(fmt.Errorf("failed to marshal supply: %w", err))
	}
	
	store.Set(key, bz)
}

// GetAllSupplies returns all coin supplies
func (k SupplyKeeper) GetAllSupplies(ctx keepertypes.Context) []types.Supply {
	var supplies []types.Supply
	
	store := k.keeper.GetKVStore(ctx)
	iterator := store.Iterator(types.SupplyPrefix, nil)
	defer iterator.Close()
	
	for ; iterator.Valid(); iterator.Next() {
		var supply types.Supply
		if err := k.keeper.GetCodec().Unmarshal(iterator.Value(), &supply); err != nil {
			continue // Skip invalid entries
		}
		supplies = append(supplies, supply)
	}
	
	return supplies
}

// MintCoins creates new coins and increases supply
func (k SupplyKeeper) MintCoins(ctx keepertypes.Context, moduleName string, coins []keepertypes.Coin) error {
	// Check permissions
	if !k.keeper.permissionKeeper.HasPermission(ctx, moduleName, types.PermissionMint) {
		return fmt.Errorf("module %s does not have mint permission", moduleName)
	}
	
	// Mint each coin
	for _, coin := range coins {
		if err := k.mintCoin(ctx, coin); err != nil {
			return fmt.Errorf("failed to mint %s: %w", coin.Denom, err)
		}
		
		// Emit mint event
		ctx.EventManager().EmitEvent(keepertypes.Event{
			Type: types.EventTypeCoinMint,
			Attributes: []keepertypes.Attribute{
				{Key: types.AttributeKeyMinter, Value: moduleName},
				{Key: types.AttributeKeyDenom, Value: coin.Denom},
				{Key: types.AttributeKeyAmount, Value: fmt.Sprintf("%d", coin.Amount)},
			},
		})
	}
	
	return nil
}

// BurnCoins destroys coins and decreases supply
func (k SupplyKeeper) BurnCoins(ctx keepertypes.Context, moduleName string, coins []keepertypes.Coin) error {
	// Check permissions
	if !k.keeper.permissionKeeper.HasPermission(ctx, moduleName, types.PermissionBurn) {
		return fmt.Errorf("module %s does not have burn permission", moduleName)
	}
	
	// Burn each coin
	for _, coin := range coins {
		if err := k.burnCoin(ctx, coin); err != nil {
			return fmt.Errorf("failed to burn %s: %w", coin.Denom, err)
		}
		
		// Emit burn event
		ctx.EventManager().EmitEvent(keepertypes.Event{
			Type: types.EventTypeCoinBurn,
			Attributes: []keepertypes.Attribute{
				{Key: types.AttributeKeyBurner, Value: moduleName},
				{Key: types.AttributeKeyDenom, Value: coin.Denom},
				{Key: types.AttributeKeyAmount, Value: fmt.Sprintf("%d", coin.Amount)},
			},
		})
	}
	
	return nil
}

// mintCoin increases the supply of a single coin
func (k SupplyKeeper) mintCoin(ctx keepertypes.Context, coin keepertypes.Coin) error {
	supply := k.GetSupply(ctx, coin.Denom)
	
	newSupply, err := supply.Add(fmt.Sprintf("%d", coin.Amount))
	if err != nil {
		return fmt.Errorf("failed to add to supply: %w", err)
	}
	
	k.SetSupply(ctx, newSupply)
	return nil
}

// burnCoin decreases the supply of a single coin
func (k SupplyKeeper) burnCoin(ctx keepertypes.Context, coin keepertypes.Coin) error {
	supply := k.GetSupply(ctx, coin.Denom)
	
	newSupply, err := supply.Sub(fmt.Sprintf("%d", coin.Amount))
	if err != nil {
		return fmt.Errorf("failed to subtract from supply: %w", err)
	}
	
	k.SetSupply(ctx, newSupply)
	return nil
}

// GetSupplyOf returns the supply amount of a specific denomination as string
func (k SupplyKeeper) GetSupplyOf(ctx keepertypes.Context, denom string) string {
	supply := k.GetSupply(ctx, denom)
	return supply.Amount
}

// InflateSupply increases the supply by a percentage
func (k SupplyKeeper) InflateSupply(ctx keepertypes.Context, denom string, inflationRate string) error {
	supply := k.GetSupply(ctx, denom)
	
	currentAmount, ok := new(big.Int).SetString(supply.Amount, 10)
	if !ok {
		return fmt.Errorf("invalid current supply amount: %s", supply.Amount)
	}
	
	rate, ok := new(big.Int).SetString(inflationRate, 10)
	if !ok {
		return fmt.Errorf("invalid inflation rate: %s", inflationRate)
	}
	
	// Calculate inflation: currentAmount * rate / 10000 (assuming rate is in basis points)
	inflation := new(big.Int).Mul(currentAmount, rate)
	inflation = inflation.Div(inflation, big.NewInt(10000))
	
	// Add inflation to supply
	newSupply, err := supply.Add(inflation.String())
	if err != nil {
		return fmt.Errorf("failed to add inflation to supply: %w", err)
	}
	
	k.SetSupply(ctx, newSupply)
	return nil
}