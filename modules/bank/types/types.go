package types

import (
	keepertypes "github.com/b2network/pulsar/keeper/types"
)

// ModuleName is the name of the bank module
const ModuleName = "bank"

// StoreKey is the store key for the bank module
const StoreKey = ModuleName

// Events
const (
	EventTypeTransfer     = "transfer"
	EventTypeCoinSpent    = "coin_spent"
	EventTypeCoinReceived = "coin_received"
	EventTypeCoinMint     = "coin_mint"
	EventTypeCoinBurn     = "coin_burn"
)

// Event attribute keys
const (
	AttributeKeyRecipient = "recipient"
	AttributeKeySender    = "sender"
	AttributeKeyAmount    = "amount"
	AttributeKeySpender   = "spender"
	AttributeKeyReceiver  = "receiver"
	AttributeKeyMinter    = "minter"
	AttributeKeyBurner    = "burner"
)

// Balance represents an account balance for a specific denomination
type Balance struct {
	Address string `json:"address"`
	Denom   string `json:"denom"`
	Amount  int64  `json:"amount"`
}

// Supply represents the total supply of a denomination
type Supply struct {
	Denom  string `json:"denom"`
	Amount int64  `json:"amount"`
}

// Metadata represents metadata for a denomination
type Metadata struct {
	Description string      `json:"description"`
	DenomUnits  []DenomUnit `json:"denom_units"`
	Base        string      `json:"base"`
	Display     string      `json:"display"`
	Name        string      `json:"name"`
	Symbol      string      `json:"symbol"`
}


// SendEnabled represents whether transfers are enabled for a denom
type SendEnabled struct {
	Denom   string `json:"denom"`
	Enabled bool   `json:"enabled"`
}

// Params defines the parameters for the bank module
type Params struct {
	SendEnabled        []SendEnabled `json:"send_enabled"`
	DefaultSendEnabled bool          `json:"default_send_enabled"`
}

// DefaultParams returns default parameters
func DefaultParams() Params {
	return Params{
		SendEnabled:        []SendEnabled{},
		DefaultSendEnabled: true,
	}
}

// GenesisState defines the bank module genesis state
type GenesisState struct {
	Params      Params        `json:"params"`
	Balances    []Balance     `json:"balances"`
	Supply      []Supply      `json:"supply"`
	Metadata    []Metadata    `json:"denom_metadata"`
	SendEnabled []SendEnabled `json:"send_enabled"`
}

// DefaultGenesisState returns default genesis state
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:      DefaultParams(),
		Balances:    []Balance{},
		Supply:      []Supply{},
		Metadata:    []Metadata{},
		SendEnabled: []SendEnabled{},
	}
}

// BankKeeper defines the expected interface for the bank keeper
type BankKeeper interface {
	keepertypes.KVStoreKeeper

	// Balance operations
	GetBalance(ctx keepertypes.Context, addr []byte, denom string) int64
	SetBalance(ctx keepertypes.Context, addr []byte, balance Balance)
	GetAllBalances(ctx keepertypes.Context, addr []byte) []Balance

	// Supply operations
	GetSupply(ctx keepertypes.Context, denom string) int64
	SetSupply(ctx keepertypes.Context, supply Supply)
	GetTotalSupply(ctx keepertypes.Context) []Supply

	// Transfer operations
	SendCoins(ctx keepertypes.Context, fromAddr, toAddr []byte, amount keepertypes.Coins) error
	SendCoinsFromModuleToAccount(ctx keepertypes.Context, senderModule string, recipientAddr []byte, amount keepertypes.Coins) error
	SendCoinsFromAccountToModule(ctx keepertypes.Context, senderAddr []byte, recipientModule string, amount keepertypes.Coins) error
	SendCoinsFromModuleToModule(ctx keepertypes.Context, senderModule, recipientModule string, amount keepertypes.Coins) error

	// Mint/Burn operations
	MintCoins(ctx keepertypes.Context, moduleName string, amount keepertypes.Coins) error
	BurnCoins(ctx keepertypes.Context, moduleName string, amount keepertypes.Coins) error

	// Validation
	ValidateBalance(balance Balance) error
	IsSendEnabledCoin(ctx keepertypes.Context, coin keepertypes.Coin) bool

	// Parameters
	GetParams(ctx keepertypes.Context) Params
	SetParams(ctx keepertypes.Context, params Params)

	// Metadata
	GetDenomMetadata(ctx keepertypes.Context, denom string) (Metadata, bool)
	SetDenomMetadata(ctx keepertypes.Context, metadata Metadata)
	GetAllDenomMetadata(ctx keepertypes.Context) []Metadata
}

// AccountKeeper defines expected interface for account keeper
type AccountKeeper interface {
	GetAccount(ctx keepertypes.Context, addr []byte) keepertypes.Account
	SetAccount(ctx keepertypes.Context, acc keepertypes.Account)
	NewAccountWithAddress(ctx keepertypes.Context, addr []byte) keepertypes.Account
	GetModuleAddress(moduleName string) []byte
	GetModuleAccount(ctx keepertypes.Context, moduleName string) keepertypes.Account
}
