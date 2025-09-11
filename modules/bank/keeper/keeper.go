package keeper

import (
	"fmt"

	abcitypes "github.com/cometbft/cometbft/abci/types"

	"github.com/b2network/pulsar/keeper/base"
	keepertypes "github.com/b2network/pulsar/keeper/types"
	"github.com/b2network/pulsar/modules/bank/types"
	coinkeeper "github.com/b2network/pulsar/modules/coin/keeper"
	cointypes "github.com/b2network/pulsar/modules/coin/types"
	storetypes "github.com/b2network/pulsar/store/types"
)

// RefactoredKeeper implements the bank keeper with coin module integration
type RefactoredKeeper struct {
	*base.KVStoreKeeper

	accountKeeper types.AccountKeeper
	
	// Coin keeper for supply and metadata management
	coinKeeper coinkeeper.CoinKeeper
}

// NewRefactoredKeeper creates a new refactored bank keeper
func NewRefactoredKeeper(
	storeKey storetypes.StoreKey,
	codec keepertypes.Codec,
	accountKeeper types.AccountKeeper,
	coinKeeper coinkeeper.CoinKeeper,
) *RefactoredKeeper {
	keeper := &RefactoredKeeper{
		KVStoreKeeper: base.NewKVStoreKeeper(storeKey, codec),
		accountKeeper: accountKeeper,
		coinKeeper:    coinKeeper,
	}

	return keeper
}

// Key prefixes for bank-specific data (balances and send enabled)
var (
	BalancesPrefix      = []byte{0x02}
	SendEnabledPrefix   = []byte{0x03}
)

// Balance operations (core bank functionality)

// GetBalance returns the balance of a specific denomination for an address
func (k RefactoredKeeper) GetBalance(ctx keepertypes.Context, addr []byte, denom string) int64 {
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

// SetBalance sets the balance for an address and denomination
func (k RefactoredKeeper) SetBalance(ctx keepertypes.Context, addr []byte, balance types.Balance) {
	if err := k.ValidateBalance(balance); err != nil {
		panic(fmt.Errorf("invalid balance: %w", err))
	}

	store := k.GetKVStore(ctx)
	key := createBalanceKey(addr, balance.Denom)

	if balance.Amount == 0 {
		store.Delete(key)
		return
	}

	bz, err := k.GetCodec().Marshal(balance)
	if err != nil {
		panic(fmt.Errorf("failed to marshal balance: %w", err))
	}

	store.Set(key, bz)
}

// GetAllBalances returns all balances for an address
func (k RefactoredKeeper) GetAllBalances(ctx keepertypes.Context, addr []byte) []types.Balance {
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

// Transfer operations

// SendCoins transfers coins between two addresses
func (k RefactoredKeeper) SendCoins(ctx keepertypes.Context, fromAddr, toAddr []byte, amount keepertypes.Coins) error {
	// Validate send enabled for all coins
	for _, coin := range amount {
		if !k.IsSendEnabledCoin(ctx, coin) {
			return fmt.Errorf("transfers are not enabled for %s", coin.Denom)
		}
	}

	// Check sufficient balances
	for _, coin := range amount {
		balance := k.GetBalance(ctx, fromAddr, coin.Denom)
		if balance < coin.Amount {
			return fmt.Errorf("insufficient funds: need %d%s, have %d%s",
				coin.Amount, coin.Denom, balance, coin.Denom)
		}
	}

	// Perform the transfer
	for _, coin := range amount {
		// Subtract from sender
		senderBalance := k.GetBalance(ctx, fromAddr, coin.Denom)
		k.SetBalance(ctx, fromAddr, types.Balance{
			Address: string(fromAddr),
			Denom:   coin.Denom,
			Amount:  senderBalance - coin.Amount,
		})

		// Add to recipient
		recipientBalance := k.GetBalance(ctx, toAddr, coin.Denom)
		k.SetBalance(ctx, toAddr, types.Balance{
			Address: string(toAddr),
			Denom:   coin.Denom,
			Amount:  recipientBalance + coin.Amount,
		})
	}

	// Emit transfer event
	k.EmitEvent(ctx, keepertypes.Event{
		Type: types.EventTypeTransfer,
		Attributes: []keepertypes.Attribute{
			{Key: types.AttributeKeyRecipient, Value: string(toAddr)},
			{Key: types.AttributeKeySender, Value: string(fromAddr)},
			{Key: types.AttributeKeyAmount, Value: formatCoins(amount)},
		},
	})

	return nil
}

// Mint and Burn operations (delegated to coin module)

// MintCoins creates new coins and adds them to a module account
func (k RefactoredKeeper) MintCoins(ctx keepertypes.Context, moduleName string, amount keepertypes.Coins) error {
	// Delegate to coin keeper for supply management
	if err := k.coinKeeper.MintCoins(ctx, moduleName, amount); err != nil {
		return err
	}

	// Handle account keeper operations if available
	if k.accountKeeper != nil {
		moduleAddr := k.accountKeeper.GetModuleAddress(moduleName)
		if moduleAddr == nil {
			return fmt.Errorf("module address not found: %s", moduleName)
		}

		// Add minted coins to module account balance
		for _, coin := range amount {
			balance := k.GetBalance(ctx, moduleAddr, coin.Denom)
			k.SetBalance(ctx, moduleAddr, types.Balance{
				Address: string(moduleAddr),
				Denom:   coin.Denom,
				Amount:  balance + coin.Amount,
			})
		}
	}

	return nil
}

// BurnCoins removes coins from a module account
func (k RefactoredKeeper) BurnCoins(ctx keepertypes.Context, moduleName string, amount keepertypes.Coins) error {
	// Handle account keeper operations if available
	if k.accountKeeper != nil {
		moduleAddr := k.accountKeeper.GetModuleAddress(moduleName)
		if moduleAddr == nil {
			return fmt.Errorf("module address not found: %s", moduleName)
		}

		// Check sufficient balance and remove from module account
		for _, coin := range amount {
			balance := k.GetBalance(ctx, moduleAddr, coin.Denom)
			if balance < coin.Amount {
				return fmt.Errorf("insufficient funds to burn: need %d%s, have %d%s",
					coin.Amount, coin.Denom, balance, coin.Denom)
			}

			k.SetBalance(ctx, moduleAddr, types.Balance{
				Address: string(moduleAddr),
				Denom:   coin.Denom,
				Amount:  balance - coin.Amount,
			})
		}
	}

	// Delegate to coin keeper for supply management
	return k.coinKeeper.BurnCoins(ctx, moduleName, amount)
}

// Supply operations (delegated to coin module)

// GetSupply returns the total supply of a denomination
func (k RefactoredKeeper) GetSupply(ctx keepertypes.Context, denom string) int64 {
	supply := k.coinKeeper.GetSupply(ctx, denom)
	// Convert coin module supply to int64 for backward compatibility
	// In a real implementation, we'd want to handle big.Int properly
	// For now, assuming the amount can be parsed as int64
	if amount, err := fmt.Sscanf(supply.Amount, "%d"); err == nil && amount > 0 {
		return int64(amount)
	}
	return 0
}

// SetSupply sets the total supply of a denomination
func (k RefactoredKeeper) SetSupply(ctx keepertypes.Context, supply types.Supply) {
	coinSupply := cointypes.NewSupply(supply.Denom, fmt.Sprintf("%d", supply.Amount))
	k.coinKeeper.SetSupply(ctx, coinSupply)
}

// Metadata operations (delegated to coin module)

// GetDenomMetadata returns metadata for a denomination
func (k RefactoredKeeper) GetDenomMetadata(ctx keepertypes.Context, denom string) (types.Metadata, bool) {
	coinMetadata, found := k.coinKeeper.GetMetadata(ctx, denom)
	if !found {
		return types.Metadata{}, false
	}

	// Convert coin module metadata to bank module metadata
	bankMetadata := types.Metadata{
		Description: coinMetadata.Description,
		Base:        coinMetadata.Base,
		Display:     coinMetadata.Display,
		Name:        coinMetadata.Name,
		Symbol:      coinMetadata.Symbol,
		DenomUnits:  make([]types.DenomUnit, len(coinMetadata.DenomUnits)),
	}

	for i, unit := range coinMetadata.DenomUnits {
		bankMetadata.DenomUnits[i] = types.DenomUnit{
			Denom:    unit.Denom,
			Exponent: unit.Exponent,
			Aliases:  unit.Aliases,
		}
	}

	return bankMetadata, true
}

// SetDenomMetadata sets metadata for a denomination
func (k RefactoredKeeper) SetDenomMetadata(ctx keepertypes.Context, metadata types.Metadata) error {
	// Convert bank module metadata to coin module metadata
	coinMetadata := cointypes.NewMetadata(
		metadata.Description,
		metadata.Base,
		metadata.Display,
		metadata.Name,
		metadata.Symbol,
		make([]cointypes.DenomUnit, len(metadata.DenomUnits)),
	)

	for i, unit := range metadata.DenomUnits {
		coinMetadata.DenomUnits[i] = cointypes.DenomUnit{
			Denom:    unit.Denom,
			Exponent: unit.Exponent,
			Aliases:  unit.Aliases,
		}
	}

	return k.coinKeeper.SetMetadata(ctx, coinMetadata)
}

// GetAllDenomMetadata returns all denomination metadata
func (k RefactoredKeeper) GetAllDenomMetadata(ctx keepertypes.Context) []types.Metadata {
	coinMetadatas := k.coinKeeper.GetAllMetadata(ctx)
	bankMetadatas := make([]types.Metadata, len(coinMetadatas))

	for i, coinMetadata := range coinMetadatas {
		bankMetadatas[i] = types.Metadata{
			Description: coinMetadata.Description,
			Base:        coinMetadata.Base,
			Display:     coinMetadata.Display,
			Name:        coinMetadata.Name,
			Symbol:      coinMetadata.Symbol,
			DenomUnits:  make([]types.DenomUnit, len(coinMetadata.DenomUnits)),
		}

		for j, unit := range coinMetadata.DenomUnits {
			bankMetadatas[i].DenomUnits[j] = types.DenomUnit{
				Denom:    unit.Denom,
				Exponent: unit.Exponent,
				Aliases:  unit.Aliases,
			}
		}
	}

	return bankMetadatas
}

// Bank-specific functionality (send enabled)

// IsSendEnabledCoin checks if transfers are enabled for a coin
func (k RefactoredKeeper) IsSendEnabledCoin(ctx keepertypes.Context, coin keepertypes.Coin) bool {
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
func (k RefactoredKeeper) GetParams(ctx keepertypes.Context) types.Params {
	var params types.Params
	if err := k.GetObject(ctx, []byte("params"), &params); err != nil {
		return types.DefaultParams()
	}
	return params
}

// SetParams sets the parameters for the bank module
func (k RefactoredKeeper) SetParams(ctx keepertypes.Context, params types.Params) {
	if err := k.SetObject(ctx, []byte("params"), params); err != nil {
		panic(fmt.Errorf("failed to set params: %w", err))
	}
}

// ValidateBalance validates a balance object
func (k RefactoredKeeper) ValidateBalance(balance types.Balance) error {
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

// Querier returns a new querier for the bank module
func (k RefactoredKeeper) Querier() keepertypes.ModuleQuerier {
	// For now, create a simple querier that delegates to the coin module
	// In a full implementation, we'd create a proper bank querier
	return &bankRefactoredQuerier{keeper: &k}
}

// bankRefactoredQuerier implements basic bank queries
type bankRefactoredQuerier struct {
	keeper *RefactoredKeeper
}

func (q *bankRefactoredQuerier) Query(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	if len(path) == 0 {
		return keepertypes.QueryError(1, "no query specified"), nil
	}

	switch path[0] {
	case "balance":
		if len(path) < 3 {
			return keepertypes.QueryError(1, "address and denom required"), nil
		}
		addr := []byte(path[1])
		denom := path[2]
		balance := q.keeper.GetBalance(ctx, addr, denom)
		
		data, err := q.keeper.GetCodec().Marshal(map[string]interface{}{
			"address": string(addr),
			"denom":   denom,
			"amount":  balance,
		})
		if err != nil {
			return keepertypes.QueryError(1, "failed to marshal balance: %v", err), nil
		}
		
		return keepertypes.QuerySuccess(data), nil
	default:
		return keepertypes.QueryError(1, "unknown bank query: %s", path[0]), nil
	}
}

func (q *bankRefactoredQuerier) RegisterQueryRoutes() map[string]keepertypes.QueryHandler {
	return map[string]keepertypes.QueryHandler{
		"balance": q.handleBalance,
	}
}

func (q *bankRefactoredQuerier) handleBalance(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	return q.Query(ctx, path, req)
}

// Helper functions
func createBalanceKey(addr []byte, denom string) []byte {
	return append(createBalancePrefix(addr), []byte(denom)...)
}

func createBalancePrefix(addr []byte) []byte {
	return append(BalancesPrefix, addr...)
}

func createSendEnabledKey(denom string) []byte {
	return append(SendEnabledPrefix, []byte(denom)...)
}

// SendCoinsFromAccountToModule transfers coins from an account to a module account
func (k RefactoredKeeper) SendCoinsFromAccountToModule(ctx keepertypes.Context, senderAddr []byte, recipientModule string, amt keepertypes.Coins) error {
	if k.accountKeeper == nil {
		return fmt.Errorf("account keeper not available")
	}
	
	moduleAddr := k.accountKeeper.GetModuleAddress(recipientModule)
	if moduleAddr == nil {
		return fmt.Errorf("module address not found: %s", recipientModule)
	}
	
	return k.SendCoins(ctx, senderAddr, moduleAddr, amt)
}

// SendCoinsFromModuleToAccount transfers coins from a module account to an account
func (k RefactoredKeeper) SendCoinsFromModuleToAccount(ctx keepertypes.Context, senderModule string, recipientAddr []byte, amt keepertypes.Coins) error {
	if k.accountKeeper == nil {
		return fmt.Errorf("account keeper not available")
	}
	
	moduleAddr := k.accountKeeper.GetModuleAddress(senderModule)
	if moduleAddr == nil {
		return fmt.Errorf("module address not found: %s", senderModule)
	}
	
	return k.SendCoins(ctx, moduleAddr, recipientAddr, amt)
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