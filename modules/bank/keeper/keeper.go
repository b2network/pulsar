package keeper

import (
	"fmt"

	"github.com/b2network/pulsar/keeper/base"
	keepertypes "github.com/b2network/pulsar/keeper/types"
	"github.com/b2network/pulsar/modules/bank/types"
	storetypes "github.com/b2network/pulsar/store/types"
)

// Keeper implements the bank keeper
type Keeper struct {
	*base.KVStoreKeeper

	accountKeeper types.AccountKeeper

	// Module permissions
	maccPerms map[string][]string
	
	// Sub-keepers
	denomMetadata *DenomMetadataKeeper
	supply        *SupplyKeeper
}

// NewKeeper creates a new bank keeper
func NewKeeper(
	storeKey storetypes.StoreKey,
	codec keepertypes.Codec,
	accountKeeper types.AccountKeeper,
	maccPerms map[string][]string,
) *Keeper {
	// Ensure the module account permissions are valid
	if maccPerms == nil {
		maccPerms = make(map[string][]string)
	}

	keeper := &Keeper{
		KVStoreKeeper: base.NewKVStoreKeeper(storeKey, codec),
		accountKeeper: accountKeeper,
		maccPerms:     maccPerms,
	}
	
	// Initialize sub-keepers
	keeper.denomMetadata = NewDenomMetadataKeeper(keeper)
	keeper.supply = NewSupplyKeeper(keeper)

	return keeper
}

// Key prefixes for different types of data
var (
	BalancesPrefix      = []byte{0x02}
	SupplyPrefix        = []byte{0x00}
	DenomMetadataPrefix = []byte{0x1}
	SendEnabledPrefix   = []byte{0x03}
)

// GetBalance returns the balance of a specific denomination for an address
func (k Keeper) GetBalance(ctx keepertypes.Context, addr []byte, denom string) int64 {
	store := k.GetKVStore(ctx)
	key := createBalanceKey(addr, denom)

	bz := store.Get(key)
	if bz == nil {
		return 0
	}

	var balance types.Balance
	if err := k.GetCodec().Unmarshal(bz, &balance); err != nil {
		return 0
	}

	return balance.Amount
}

// SetBalance sets the balance of a specific denomination for an address
func (k Keeper) SetBalance(ctx keepertypes.Context, addr []byte, balance types.Balance) {
	store := k.GetKVStore(ctx)
	key := createBalanceKey(addr, balance.Denom)

	if balance.Amount <= 0 {
		store.Delete(key)
		return
	}

	bz, err := k.GetCodec().Marshal(balance)
	if err != nil {
		panic(fmt.Errorf("failed to marshal balance: %w", err))
	}

	store.Set(key, bz)

	// Emit balance change event
	k.EmitEvent(ctx, keepertypes.Event{
		Type: types.EventTypeCoinReceived,
		Attributes: []keepertypes.Attribute{
			{Key: types.AttributeKeyReceiver, Value: balance.Address},
			{Key: types.AttributeKeyAmount, Value: fmt.Sprintf("%d%s", balance.Amount, balance.Denom)},
		},
	})
}

// GetAllBalances returns all balances for an address
func (k Keeper) GetAllBalances(ctx keepertypes.Context, addr []byte) []types.Balance {
	var balances []types.Balance

	store := k.GetKVStore(ctx)
	prefix := createBalancePrefix(addr)

	iterator := store.Iterator(prefix, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var balance types.Balance
		if err := k.GetCodec().Unmarshal(iterator.Value(), &balance); err != nil {
			continue
		}
		balances = append(balances, balance)
	}

	return balances
}

// GetSupply returns the total supply of a denomination
func (k Keeper) GetSupply(ctx keepertypes.Context, denom string) int64 {
	store := k.GetKVStore(ctx)
	key := createSupplyKey(denom)

	bz := store.Get(key)
	if bz == nil {
		return 0
	}

	var supply types.Supply
	if err := k.GetCodec().Unmarshal(bz, &supply); err != nil {
		return 0
	}

	return supply.Amount
}

// SetSupply sets the total supply of a denomination
func (k Keeper) SetSupply(ctx keepertypes.Context, supply types.Supply) {
	store := k.GetKVStore(ctx)
	key := createSupplyKey(supply.Denom)

	if supply.Amount <= 0 {
		store.Delete(key)
		return
	}

	bz, err := k.GetCodec().Marshal(supply)
	if err != nil {
		panic(fmt.Errorf("failed to marshal supply: %w", err))
	}

	store.Set(key, bz)
}

// GetTotalSupply returns the total supply of all denominations
func (k Keeper) GetTotalSupply(ctx keepertypes.Context) []types.Supply {
	var supplies []types.Supply

	store := k.GetKVStore(ctx)
	iterator := store.Iterator(SupplyPrefix, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var supply types.Supply
		if err := k.GetCodec().Unmarshal(iterator.Value(), &supply); err != nil {
			continue
		}
		supplies = append(supplies, supply)
	}

	return supplies
}

// SendCoins transfers coins from one account to another
func (k Keeper) SendCoins(ctx keepertypes.Context, fromAddr, toAddr []byte, amount keepertypes.Coins) error {
	// Validate send is enabled for all coins
	for _, coin := range amount {
		if !k.IsSendEnabledCoin(ctx, coin) {
			return fmt.Errorf("send disabled for denomination: %s", coin.Denom)
		}
	}

	// Check sufficient balance
	for _, coin := range amount {
		balance := k.GetBalance(ctx, fromAddr, coin.Denom)
		if balance < coin.Amount {
			return fmt.Errorf("insufficient funds: need %d%s, have %d%s",
				coin.Amount, coin.Denom, balance, coin.Denom)
		}
	}

	// Subtract from sender
	for _, coin := range amount {
		balance := k.GetBalance(ctx, fromAddr, coin.Denom)
		newBalance := types.Balance{
			Address: string(fromAddr),
			Denom:   coin.Denom,
			Amount:  balance - coin.Amount,
		}
		k.SetBalance(ctx, fromAddr, newBalance)
	}

	// Add to recipient
	for _, coin := range amount {
		balance := k.GetBalance(ctx, toAddr, coin.Denom)
		newBalance := types.Balance{
			Address: string(toAddr),
			Denom:   coin.Denom,
			Amount:  balance + coin.Amount,
		}
		k.SetBalance(ctx, toAddr, newBalance)
	}

	// Emit transfer event
	k.EmitEvent(ctx, keepertypes.Event{
		Type: types.EventTypeTransfer,
		Attributes: []keepertypes.Attribute{
			{Key: types.AttributeKeySender, Value: string(fromAddr)},
			{Key: types.AttributeKeyRecipient, Value: string(toAddr)},
			{Key: types.AttributeKeyAmount, Value: formatCoins(amount)},
		},
	})

	return nil
}

// SendCoinsFromModuleToAccount transfers coins from module to account
func (k Keeper) SendCoinsFromModuleToAccount(ctx keepertypes.Context, senderModule string, recipientAddr []byte, amount keepertypes.Coins) error {
	// Handle nil account keeper for testing examples
	if k.accountKeeper == nil {
		return fmt.Errorf("account keeper not available in example")
	}

	senderAddr := k.accountKeeper.GetModuleAddress(senderModule)
	if senderAddr == nil {
		return fmt.Errorf("module address not found: %s", senderModule)
	}

	return k.SendCoins(ctx, senderAddr, recipientAddr, amount)
}

// SendCoinsFromAccountToModule transfers coins from account to module
func (k Keeper) SendCoinsFromAccountToModule(ctx keepertypes.Context, senderAddr []byte, recipientModule string, amount keepertypes.Coins) error {
	// Handle nil account keeper for testing examples
	if k.accountKeeper == nil {
		return fmt.Errorf("account keeper not available in example")
	}

	recipientAddr := k.accountKeeper.GetModuleAddress(recipientModule)
	if recipientAddr == nil {
		return fmt.Errorf("module address not found: %s", recipientModule)
	}

	return k.SendCoins(ctx, senderAddr, recipientAddr, amount)
}

// SendCoinsFromModuleToModule transfers coins from one module to another
func (k Keeper) SendCoinsFromModuleToModule(ctx keepertypes.Context, senderModule, recipientModule string, amount keepertypes.Coins) error {
	// Handle nil account keeper for testing examples
	if k.accountKeeper == nil {
		return fmt.Errorf("account keeper not available in example")
	}

	senderAddr := k.accountKeeper.GetModuleAddress(senderModule)
	recipientAddr := k.accountKeeper.GetModuleAddress(recipientModule)

	if senderAddr == nil {
		return fmt.Errorf("sender module address not found: %s", senderModule)
	}
	if recipientAddr == nil {
		return fmt.Errorf("recipient module address not found: %s", recipientModule)
	}

	return k.SendCoins(ctx, senderAddr, recipientAddr, amount)
}

// MintCoins creates new coins and adds them to a module account
func (k Keeper) MintCoins(ctx keepertypes.Context, moduleName string, amount keepertypes.Coins) error {
	// Check module has minting permission
	if !k.hasPermission(moduleName, "minter") {
		return fmt.Errorf("module %s does not have minting permission", moduleName)
	}

	// Handle nil account keeper for testing examples
	if k.accountKeeper == nil {
		// For examples, simulate minting by updating supply only
		for _, coin := range amount {
			// Update total supply
			supply := k.GetSupply(ctx, coin.Denom)
			newSupply := types.Supply{
				Denom:  coin.Denom,
				Amount: supply + coin.Amount,
			}
			k.SetSupply(ctx, newSupply)
		}
		return nil
	}

	moduleAddr := k.accountKeeper.GetModuleAddress(moduleName)
	if moduleAddr == nil {
		return fmt.Errorf("module address not found: %s", moduleName)
	}

	// Add coins to module account
	for _, coin := range amount {
		balance := k.GetBalance(ctx, moduleAddr, coin.Denom)
		newBalance := types.Balance{
			Address: string(moduleAddr),
			Denom:   coin.Denom,
			Amount:  balance + coin.Amount,
		}
		k.SetBalance(ctx, moduleAddr, newBalance)

		// Update total supply
		supply := k.GetSupply(ctx, coin.Denom)
		newSupply := types.Supply{
			Denom:  coin.Denom,
			Amount: supply + coin.Amount,
		}
		k.SetSupply(ctx, newSupply)
	}

	// Emit mint event
	k.EmitEvent(ctx, keepertypes.Event{
		Type: types.EventTypeCoinMint,
		Attributes: []keepertypes.Attribute{
			{Key: types.AttributeKeyMinter, Value: moduleName},
			{Key: types.AttributeKeyAmount, Value: formatCoins(amount)},
		},
	})

	return nil
}

// BurnCoins removes coins from a module account
func (k Keeper) BurnCoins(ctx keepertypes.Context, moduleName string, amount keepertypes.Coins) error {
	// Check module has burning permission
	if !k.hasPermission(moduleName, "burner") {
		return fmt.Errorf("module %s does not have burning permission", moduleName)
	}

	// Handle nil account keeper for testing examples
	if k.accountKeeper == nil {
		// For examples, simulate burning by updating supply only
		for _, coin := range amount {
			// Update total supply
			supply := k.GetSupply(ctx, coin.Denom)
			newSupply := types.Supply{
				Denom:  coin.Denom,
				Amount: supply - coin.Amount,
			}
			k.SetSupply(ctx, newSupply)
		}
		return nil
	}

	moduleAddr := k.accountKeeper.GetModuleAddress(moduleName)
	if moduleAddr == nil {
		return fmt.Errorf("module address not found: %s", moduleName)
	}

	// Check sufficient balance
	for _, coin := range amount {
		balance := k.GetBalance(ctx, moduleAddr, coin.Denom)
		if balance < coin.Amount {
			return fmt.Errorf("insufficient funds to burn: need %d%s, have %d%s",
				coin.Amount, coin.Denom, balance, coin.Denom)
		}
	}

	// Remove coins from module account
	for _, coin := range amount {
		balance := k.GetBalance(ctx, moduleAddr, coin.Denom)
		newBalance := types.Balance{
			Address: string(moduleAddr),
			Denom:   coin.Denom,
			Amount:  balance - coin.Amount,
		}
		k.SetBalance(ctx, moduleAddr, newBalance)

		// Update total supply
		supply := k.GetSupply(ctx, coin.Denom)
		newSupply := types.Supply{
			Denom:  coin.Denom,
			Amount: supply - coin.Amount,
		}
		k.SetSupply(ctx, newSupply)
	}

	// Emit burn event
	k.EmitEvent(ctx, keepertypes.Event{
		Type: types.EventTypeCoinBurn,
		Attributes: []keepertypes.Attribute{
			{Key: types.AttributeKeyBurner, Value: moduleName},
			{Key: types.AttributeKeyAmount, Value: formatCoins(amount)},
		},
	})

	return nil
}

// ValidateBalance validates a balance object
func (k Keeper) ValidateBalance(balance types.Balance) error {
	if balance.Address == "" {
		return fmt.Errorf("address cannot be empty")
	}
	if balance.Denom == "" {
		return fmt.Errorf("denom cannot be empty")
	}
	if balance.Amount < 0 {
		return fmt.Errorf("amount cannot be negative")
	}
	return nil
}

// IsSendEnabledCoin checks if transfers are enabled for a coin
func (k Keeper) IsSendEnabledCoin(ctx keepertypes.Context, coin keepertypes.Coin) bool {
	store := k.GetKVStore(ctx)
	key := createSendEnabledKey(coin.Denom)

	bz := store.Get(key)
	if bz == nil {
		// If not set, use default from params
		params := k.GetParams(ctx)
		return params.DefaultSendEnabled
	}

	var sendEnabled types.SendEnabled
	if err := k.GetCodec().Unmarshal(bz, &sendEnabled); err != nil {
		return false
	}

	return sendEnabled.Enabled
}

// GetParams returns the parameters for the bank module
func (k Keeper) GetParams(ctx keepertypes.Context) types.Params {
	var params types.Params
	if err := k.GetObject(ctx, []byte("params"), &params); err != nil {
		return types.DefaultParams()
	}
	return params
}

// SetParams sets the parameters for the bank module
func (k Keeper) SetParams(ctx keepertypes.Context, params types.Params) {
	if err := k.SetObject(ctx, []byte("params"), params); err != nil {
		panic(fmt.Errorf("failed to set params: %w", err))
	}
}

// GetDenomMetadata returns metadata for a denomination
func (k Keeper) GetDenomMetadata(ctx keepertypes.Context, denom string) (types.Metadata, bool) {
	store := k.GetKVStore(ctx)
	key := createDenomMetadataKey(denom)

	bz := store.Get(key)
	if bz == nil {
		return types.Metadata{}, false
	}

	var metadata types.Metadata
	if err := k.GetCodec().Unmarshal(bz, &metadata); err != nil {
		return types.Metadata{}, false
	}

	return metadata, true
}

// SetDenomMetadata sets metadata for a denomination
func (k Keeper) SetDenomMetadata(ctx keepertypes.Context, metadata types.Metadata) {
	store := k.GetKVStore(ctx)
	key := createDenomMetadataKey(metadata.Base)

	bz, err := k.GetCodec().Marshal(metadata)
	if err != nil {
		panic(fmt.Errorf("failed to marshal metadata: %w", err))
	}

	store.Set(key, bz)
}

// GetAllDenomMetadata returns all denomination metadata
func (k Keeper) GetAllDenomMetadata(ctx keepertypes.Context) []types.Metadata {
	var metadata []types.Metadata

	store := k.GetKVStore(ctx)
	iterator := store.Iterator(DenomMetadataPrefix, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var meta types.Metadata
		if err := k.GetCodec().Unmarshal(iterator.Value(), &meta); err != nil {
			continue
		}
		metadata = append(metadata, meta)
	}

	return metadata
}

// Helper functions

func createBalanceKey(addr []byte, denom string) []byte {
	return append(createBalancePrefix(addr), []byte(denom)...)
}

func createBalancePrefix(addr []byte) []byte {
	return append(BalancesPrefix, addr...)
}

func createSupplyKey(denom string) []byte {
	return append(SupplyPrefix, []byte(denom)...)
}

func createDenomMetadataKey(denom string) []byte {
	return append(DenomMetadataPrefix, []byte(denom)...)
}

func createSendEnabledKey(denom string) []byte {
	return append(SendEnabledPrefix, []byte(denom)...)
}

func (k Keeper) hasPermission(moduleName, permission string) bool {
	perms, exists := k.maccPerms[moduleName]
	if !exists {
		return false
	}

	for _, perm := range perms {
		if perm == permission {
			return true
		}
	}
	return false
}

func formatCoins(coins keepertypes.Coins) string {
	result := ""
	for i, coin := range coins {
		if i > 0 {
			result += ","
		}
		result += fmt.Sprintf("%d%s", coin.Amount, coin.Denom)
	}
	return result
}

// KVStorePrefixIterator creates a prefix iterator (placeholder implementation)
func KVStorePrefixIterator(store storetypes.KVStore, prefix []byte) storetypes.Iterator {
	// In a real implementation, this would use our prefix store
	return store.Iterator(prefix, nil)
}

// DenomMetadata returns the denomination metadata keeper
func (k Keeper) DenomMetadata() *DenomMetadataKeeper {
	return k.denomMetadata
}

// Supply returns the supply keeper
func (k Keeper) Supply() *SupplyKeeper {
	return k.supply
}

// Querier returns a new querier for the bank module
func (k Keeper) Querier() keepertypes.ModuleQuerier {
	return NewQuerier(&k)
}
