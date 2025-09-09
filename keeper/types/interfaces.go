package types

import (
	"github.com/b2network/pulsar/store/types"
)

// Keeper defines the base interface that all keepers must implement
type Keeper interface {
	// GetStoreKey returns the store key for this keeper
	GetStoreKey() types.StoreKey

	// GetCodec returns the codec used by this keeper
	GetCodec() Codec
}

// KVStoreKeeper extends Keeper with key-value store operations
type KVStoreKeeper interface {
	Keeper

	// GetKVStore returns the KVStore for this keeper with the provided context
	GetKVStore(ctx Context) types.KVStore

	// Get retrieves a value by key
	Get(ctx Context, key []byte) []byte

	// Set stores a key-value pair
	Set(ctx Context, key []byte, value []byte)

	// Delete removes a key
	Delete(ctx Context, key []byte)

	// Has checks if a key exists
	Has(ctx Context, key []byte) bool

	// Iterator creates an iterator over a key range
	Iterator(ctx Context, start, end []byte) types.Iterator

	// GetObject retrieves and deserializes an object by key
	GetObject(ctx Context, key []byte, obj interface{}) error

	// SetObject serializes and stores an object by key
	SetObject(ctx Context, key []byte, obj interface{}) error
}

// Codec defines the interface for encoding/decoding data
type Codec interface {
	// Marshal encodes an object to bytes
	Marshal(obj interface{}) ([]byte, error)

	// Unmarshal decodes bytes to an object
	Unmarshal(bz []byte, obj interface{}) error

	// MarshalJSON encodes an object to JSON
	MarshalJSON(obj interface{}) ([]byte, error)

	// UnmarshalJSON decodes JSON to an object
	UnmarshalJSON(bz []byte, obj interface{}) error
}

// Context defines the interface for execution context
type Context interface {
	// KVStore returns the KVStore for the given key
	KVStore(key types.StoreKey) types.KVStore

	// MultiStore returns the multi store
	MultiStore() types.MultiStore

	// BlockHeight returns the current block height
	BlockHeight() int64

	// ChainID returns the chain ID
	ChainID() string

	// Logger returns a logger for this context
	Logger() Logger

	// EventManager returns the event manager
	EventManager() EventManager

	// WithLogger returns a new context with the given logger
	WithLogger(logger Logger) Context

	// WithEventManager returns a new context with the given event manager
	WithEventManager(em EventManager) Context
}

// Logger defines the logging interface
type Logger interface {
	Debug(msg string, keyvals ...interface{})
	Info(msg string, keyvals ...interface{})
	Error(msg string, keyvals ...interface{})
}

// EventManager defines the event management interface
type EventManager interface {
	EmitEvent(event Event)
	EmitEvents(events Events)
	Events() Events
}

// Event represents an event that occurred during execution
type Event struct {
	Type       string      `json:"type"`
	Attributes []Attribute `json:"attributes"`
}

// Events is a slice of Event objects
type Events []Event

// Attribute represents a key-value pair in an event
type Attribute struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Authority defines the interface for authorization
type Authority interface {
	// GetAuthority returns the authority address
	GetAuthority() string

	// HasAuthority checks if the given address has authority
	HasAuthority(address string) bool
}

// Querier defines the interface for query handling
type Querier interface {
	// Query processes a query request
	Query(ctx Context, path []string, req []byte) ([]byte, error)
}

// Migrator defines the interface for state migration
type Migrator interface {
	// Migrate1to2 migrates state from version 1 to 2
	Migrate1to2(ctx Context) error

	// Migrate2to3 migrates state from version 2 to 3
	Migrate2to3(ctx Context) error
}

// ValidatorSet defines the interface for validator set operations
type ValidatorSet interface {
	// GetValidator returns a validator by operator address
	GetValidator(ctx Context, addr []byte) (Validator, bool)

	// IterateValidators iterates over all validators
	IterateValidators(ctx Context, fn func(index int64, validator Validator) bool)

	// Jail jails a validator
	Jail(ctx Context, addr []byte)

	// Unjail unjails a validator
	Unjail(ctx Context, addr []byte)
}

// Validator defines the validator interface
type Validator interface {
	GetOperator() []byte
	GetPubKey() []byte
	GetTokens() int64
	GetStatus() int32
	IsJailed() bool
}

// Delegation defines the delegation interface
type Delegation interface {
	GetDelegatorAddr() []byte
	GetValidatorAddr() []byte
	GetShares() int64
}

// AccountKeeper defines expected interface for account operations
type AccountKeeper interface {
	GetAccount(ctx Context, addr []byte) Account
	SetAccount(ctx Context, acc Account)
	NewAccount(ctx Context, acc Account) Account
	NewAccountWithAddress(ctx Context, addr []byte) Account
	GetNextAccountNumber(ctx Context) uint64
}

// Account defines the account interface
type Account interface {
	GetAddress() []byte
	GetPubKey() []byte
	GetAccountNumber() uint64
	GetSequence() uint64
	SetSequence(uint64)
}

// BankKeeper defines expected interface for bank operations
type BankKeeper interface {
	GetSupply(ctx Context, denom string) int64
	GetBalance(ctx Context, addr []byte, denom string) int64
	SendCoins(ctx Context, fromAddr, toAddr []byte, amount Coins) error
	MintCoins(ctx Context, moduleName string, amount Coins) error
	BurnCoins(ctx Context, moduleName string, amount Coins) error
}

// Coin represents a token with denomination and amount
type Coin struct {
	Denom  string `json:"denom"`
	Amount int64  `json:"amount"`
}

// Coins represents a collection of coins
type Coins []Coin

// Distribution defines expected interface for distribution operations
type DistributionKeeper interface {
	AllocateTokensToValidator(ctx Context, val Validator, tokens int64)
	WithdrawDelegationRewards(ctx Context, delAddr, valAddr []byte) (Coins, error)
}

// Slashing defines expected interface for slashing operations
type SlashingKeeper interface {
	Slash(ctx Context, consAddr []byte, fraction int64) int64
	SlashWithInfractionReason(ctx Context, consAddr []byte, fraction int64, reason string) int64
}

// Governance defines expected interface for governance operations
type GovKeeper interface {
	GetProposal(ctx Context, proposalID uint64) (Proposal, bool)
	SetProposal(ctx Context, proposal Proposal)
	GetVote(ctx Context, proposalID uint64, voterAddr []byte) (Vote, bool)
	SetVote(ctx Context, vote Vote)
}

// Proposal defines the governance proposal interface
type Proposal interface {
	GetProposalID() uint64
	GetTitle() string
	GetDescription() string
	GetStatus() int32
}

// Vote defines the governance vote interface
type Vote interface {
	GetProposalID() uint64
	GetVoter() []byte
	GetOptions() []VoteOption
}

// VoteOption defines a vote option
type VoteOption struct {
	Option int32  `json:"option"`
	Weight string `json:"weight"`
}
