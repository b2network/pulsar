package keeper

import (
	keepertypes "github.com/b2network/pulsar/keeper/types"
	"github.com/b2network/pulsar/keeper/base"
	"github.com/b2network/pulsar/modules/coin/types"
	storetypes "github.com/b2network/pulsar/store/types"
)

// Keeper manages coin supply, metadata and permissions
type Keeper struct {
	*base.KVStoreKeeper
	
	// Sub-keepers
	supplyKeeper     *SupplyKeeper
	metadataKeeper   *MetadataKeeper
	permissionKeeper *PermissionKeeper
}

// NewKeeper creates a new coin keeper
func NewKeeper(
	storeKey storetypes.StoreKey,
	codec keepertypes.Codec,
) *Keeper {
	keeper := &Keeper{
		KVStoreKeeper: base.NewKVStoreKeeper(storeKey, codec),
	}
	
	// Initialize sub-keepers
	keeper.supplyKeeper = NewSupplyKeeper(keeper)
	keeper.metadataKeeper = NewMetadataKeeper(keeper)
	keeper.permissionKeeper = NewPermissionKeeper(keeper)
	
	return keeper
}

// Supply returns the supply keeper
func (k Keeper) Supply() *SupplyKeeper {
	return k.supplyKeeper
}

// Metadata returns the metadata keeper
func (k Keeper) Metadata() *MetadataKeeper {
	return k.metadataKeeper
}

// Permission returns the permission keeper
func (k Keeper) Permission() *PermissionKeeper {
	return k.permissionKeeper
}

// InitGenesis initializes the coin module from genesis state
func (k Keeper) InitGenesis(ctx keepertypes.Context, genState *types.GenesisState) {
	// Initialize supplies
	for _, supply := range genState.Supplies {
		k.supplyKeeper.SetSupply(ctx, supply)
	}
	
	// Initialize metadata
	for _, metadata := range genState.Metadatas {
		k.metadataKeeper.SetMetadata(ctx, metadata)
	}
	
	// Initialize permissions
	k.permissionKeeper.SetPermissionSet(ctx, genState.Permissions)
}

// ExportGenesis exports the coin module state to genesis
func (k Keeper) ExportGenesis(ctx keepertypes.Context) *types.GenesisState {
	return &types.GenesisState{
		Supplies:    k.supplyKeeper.GetAllSupplies(ctx),
		Metadatas:   k.metadataKeeper.GetAllMetadata(ctx),
		Permissions: k.permissionKeeper.GetPermissionSet(ctx),
	}
}

// CoinKeeper defines the expected interface for interacting with coins
type CoinKeeper interface {
	// Supply operations
	GetSupply(ctx keepertypes.Context, denom string) types.Supply
	SetSupply(ctx keepertypes.Context, supply types.Supply)
	GetTotalSupply(ctx keepertypes.Context) []types.Supply
	
	// Mint/Burn operations
	MintCoins(ctx keepertypes.Context, moduleName string, coins []keepertypes.Coin) error
	BurnCoins(ctx keepertypes.Context, moduleName string, coins []keepertypes.Coin) error
	
	// Metadata operations
	GetMetadata(ctx keepertypes.Context, denom string) (types.Metadata, bool)
	SetMetadata(ctx keepertypes.Context, metadata types.Metadata) error
	GetAllMetadata(ctx keepertypes.Context) []types.Metadata
	
	// Permission operations
	HasPermission(ctx keepertypes.Context, moduleName, permission string) bool
	SetModulePermissions(ctx keepertypes.Context, mp types.ModulePermissions) error
}

// Ensure Keeper implements CoinKeeper interface
var _ CoinKeeper = (*Keeper)(nil)

// Interface delegation to sub-keepers
func (k Keeper) GetSupply(ctx keepertypes.Context, denom string) types.Supply {
	return k.supplyKeeper.GetSupply(ctx, denom)
}

func (k Keeper) SetSupply(ctx keepertypes.Context, supply types.Supply) {
	k.supplyKeeper.SetSupply(ctx, supply)
}

func (k Keeper) GetTotalSupply(ctx keepertypes.Context) []types.Supply {
	return k.supplyKeeper.GetAllSupplies(ctx)
}

func (k Keeper) MintCoins(ctx keepertypes.Context, moduleName string, coins []keepertypes.Coin) error {
	return k.supplyKeeper.MintCoins(ctx, moduleName, coins)
}

func (k Keeper) BurnCoins(ctx keepertypes.Context, moduleName string, coins []keepertypes.Coin) error {
	return k.supplyKeeper.BurnCoins(ctx, moduleName, coins)
}

func (k Keeper) GetMetadata(ctx keepertypes.Context, denom string) (types.Metadata, bool) {
	return k.metadataKeeper.GetMetadata(ctx, denom)
}

func (k Keeper) SetMetadata(ctx keepertypes.Context, metadata types.Metadata) error {
	return k.metadataKeeper.SetMetadata(ctx, metadata)
}

func (k Keeper) GetAllMetadata(ctx keepertypes.Context) []types.Metadata {
	return k.metadataKeeper.GetAllMetadata(ctx)
}

func (k Keeper) HasPermission(ctx keepertypes.Context, moduleName, permission string) bool {
	return k.permissionKeeper.HasPermission(ctx, moduleName, permission)
}

func (k Keeper) SetModulePermissions(ctx keepertypes.Context, mp types.ModulePermissions) error {
	return k.permissionKeeper.SetModulePermissions(ctx, mp)
}

// Querier returns a new querier for the coin module
func (k Keeper) Querier() keepertypes.ModuleQuerier {
	return NewQuerier(&k)
}