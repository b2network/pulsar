package keeper

import (
	"fmt"

	keepertypes "github.com/b2network/pulsar/keeper/types"
	"github.com/b2network/pulsar/modules/coin/types"
)

// MetadataKeeper manages coin denomination metadata
type MetadataKeeper struct {
	keeper *Keeper
}

// NewMetadataKeeper creates a new metadata keeper
func NewMetadataKeeper(k *Keeper) *MetadataKeeper {
	return &MetadataKeeper{
		keeper: k,
	}
}

// GetMetadata returns metadata for a denomination
func (k MetadataKeeper) GetMetadata(ctx keepertypes.Context, denom string) (types.Metadata, bool) {
	store := k.keeper.GetKVStore(ctx)
	key := types.MetadataKey(denom)
	
	bz := store.Get(key)
	if bz == nil {
		return types.Metadata{}, false
	}
	
	var metadata types.Metadata
	if err := k.keeper.GetCodec().Unmarshal(bz, &metadata); err != nil {
		return types.Metadata{}, false
	}
	
	return metadata, true
}

// SetMetadata sets metadata for a denomination
func (k MetadataKeeper) SetMetadata(ctx keepertypes.Context, metadata types.Metadata) error {
	// Validate metadata
	if err := metadata.ValidateBasic(); err != nil {
		return fmt.Errorf("invalid metadata: %w", err)
	}
	
	store := k.keeper.GetKVStore(ctx)
	key := types.MetadataKey(metadata.Base)
	
	bz, err := k.keeper.GetCodec().Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	
	store.Set(key, bz)
	
	// Emit event
	ctx.EventManager().EmitEvent(keepertypes.Event{
		Type: types.EventTypeMetadataSet,
		Attributes: []keepertypes.Attribute{
			{Key: types.AttributeKeyDenom, Value: metadata.Base},
			{Key: types.AttributeKeyDisplay, Value: metadata.Display},
			{Key: types.AttributeKeyName, Value: metadata.Name},
			{Key: types.AttributeKeySymbol, Value: metadata.Symbol},
		},
	})
	
	return nil
}

// GetAllMetadata returns all denomination metadata
func (k MetadataKeeper) GetAllMetadata(ctx keepertypes.Context) []types.Metadata {
	var metadatas []types.Metadata
	
	store := k.keeper.GetKVStore(ctx)
	iterator := store.Iterator(types.MetadataPrefix, nil)
	defer iterator.Close()
	
	for ; iterator.Valid(); iterator.Next() {
		var metadata types.Metadata
		if err := k.keeper.GetCodec().Unmarshal(iterator.Value(), &metadata); err != nil {
			continue // Skip invalid entries
		}
		metadatas = append(metadatas, metadata)
	}
	
	return metadatas
}

// HasMetadata checks if metadata exists for a denomination
func (k MetadataKeeper) HasMetadata(ctx keepertypes.Context, denom string) bool {
	_, found := k.GetMetadata(ctx, denom)
	return found
}

// DeleteMetadata removes metadata for a denomination
func (k MetadataKeeper) DeleteMetadata(ctx keepertypes.Context, denom string) error {
	store := k.keeper.GetKVStore(ctx)
	key := types.MetadataKey(denom)
	
	// Check if metadata exists
	if !store.Has(key) {
		return fmt.Errorf("metadata not found for denom %s", denom)
	}
	
	store.Delete(key)
	
	// Emit event
	ctx.EventManager().EmitEvent(keepertypes.Event{
		Type: types.EventTypeMetadataDeleted,
		Attributes: []keepertypes.Attribute{
			{Key: types.AttributeKeyDenom, Value: denom},
		},
	})
	
	return nil
}

// GetDenomUnit returns the DenomUnit for a given denom
func (k MetadataKeeper) GetDenomUnit(ctx keepertypes.Context, baseDenom, unitDenom string) (types.DenomUnit, bool) {
	metadata, found := k.GetMetadata(ctx, baseDenom)
	if !found {
		return types.DenomUnit{}, false
	}
	
	return metadata.GetDenomUnit(unitDenom)
}

// UpdateMetadata updates existing metadata
func (k MetadataKeeper) UpdateMetadata(ctx keepertypes.Context, metadata types.Metadata) error {
	// Check if metadata exists
	if !k.HasMetadata(ctx, metadata.Base) {
		return fmt.Errorf("metadata not found for denom %s", metadata.Base)
	}
	
	return k.SetMetadata(ctx, metadata)
}

// SetDefaultMetadata sets default metadata for common denominations
func (k MetadataKeeper) SetDefaultMetadata(ctx keepertypes.Context) error {
	// Bitcoin metadata
	btcMetadata := types.NewMetadata(
		"Bitcoin on B2 Network",
		"ubtc", // base denom (micro bitcoin)
		"btc",  // display denom
		"Bitcoin",
		"BTC",
		[]types.DenomUnit{
			{Denom: "ubtc", Exponent: 0, Aliases: []string{"microbitcoin"}},
			{Denom: "mbtc", Exponent: 3, Aliases: []string{"millibitcoin"}},
			{Denom: "btc", Exponent: 6, Aliases: []string{"bitcoin"}},
		},
	)
	
	if err := k.SetMetadata(ctx, btcMetadata); err != nil {
		return fmt.Errorf("failed to set BTC metadata: %w", err)
	}
	
	// Ethereum metadata
	ethMetadata := types.NewMetadata(
		"Ethereum on B2 Network",
		"wei",  // base denom
		"eth",  // display denom
		"Ethereum",
		"ETH",
		[]types.DenomUnit{
			{Denom: "wei", Exponent: 0, Aliases: []string{}},
			{Denom: "gwei", Exponent: 9, Aliases: []string{"gigawei"}},
			{Denom: "eth", Exponent: 18, Aliases: []string{"ether"}},
		},
	)
	
	if err := k.SetMetadata(ctx, ethMetadata); err != nil {
		return fmt.Errorf("failed to set ETH metadata: %w", err)
	}
	
	return nil
}

// ValidateMetadataForSupply validates that metadata is consistent with supply
func (k MetadataKeeper) ValidateMetadataForSupply(ctx keepertypes.Context, metadata types.Metadata) error {
	// Check if supply exists for base denom
	supply := k.keeper.supplyKeeper.GetSupply(ctx, metadata.Base)
	if supply.IsZero() {
		return fmt.Errorf("no supply found for base denom %s", metadata.Base)
	}
	
	return nil
}