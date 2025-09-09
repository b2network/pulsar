package types

import (
	keepertypes "github.com/b2network/pulsar/keeper/types"
)

// ModuleName is the name of the staking module
const ModuleName = "staking"

// StoreKey is the store key for the staking module
const StoreKey = ModuleName

// Events
const (
	EventTypeDelegate   = "delegate"
	EventTypeUndelegate = "undelegate"
	EventTypeRedelegate = "redelegate"
	EventTypeSlash      = "slash"
)

// Event attribute keys
const (
	AttributeKeyValidator    = "validator"
	AttributeKeyDelegator    = "delegator"
	AttributeKeyAmount       = "amount"
	AttributeKeyShares       = "shares"
	AttributeKeySrcValidator = "source_validator"
	AttributeKeyDstValidator = "destination_validator"
)

// BondStatus represents the status of a validator
type BondStatus int32

const (
	Unbonded  BondStatus = 1
	Unbonding BondStatus = 2
	Bonded    BondStatus = 3
)

// Validator represents a validator in the staking module
type Validator struct {
	OperatorAddress   string      `json:"operator_address"`
	ConsPubKey        []byte      `json:"consensus_pubkey"`
	Jailed            bool        `json:"jailed"`
	Status            BondStatus  `json:"status"`
	Tokens            int64       `json:"tokens"`
	DelegatorShares   int64       `json:"delegator_shares"`
	Description       Description `json:"description"`
	UnbondingHeight   int64       `json:"unbonding_height"`
	UnbondingTime     int64       `json:"unbonding_time"`
	Commission        Commission  `json:"commission"`
	MinSelfDelegation int64       `json:"min_self_delegation"`
}

// Description contains validator description information
type Description struct {
	Moniker         string `json:"moniker"`
	Identity        string `json:"identity"`
	Website         string `json:"website"`
	SecurityContact string `json:"security_contact"`
	Details         string `json:"details"`
}

// Commission defines commission parameters for a validator
type Commission struct {
	Rate          int64 `json:"rate"`            // Commission rate charged to delegators
	MaxRate       int64 `json:"max_rate"`        // Maximum commission rate
	MaxChangeRate int64 `json:"max_change_rate"` // Maximum daily increase of the validator commission
	UpdateTime    int64 `json:"update_time"`     // Last time the commission rate was changed
}

// Delegation represents a delegation from a delegator to a validator
type Delegation struct {
	DelegatorAddress string `json:"delegator_address"`
	ValidatorAddress string `json:"validator_address"`
	Shares           int64  `json:"shares"`
}

// UnbondingDelegation represents an unbonding delegation
type UnbondingDelegation struct {
	DelegatorAddress string           `json:"delegator_address"`
	ValidatorAddress string           `json:"validator_address"`
	Entries          []UnbondingEntry `json:"entries"`
}

// UnbondingEntry represents a single unbonding entry
type UnbondingEntry struct {
	CreationHeight int64 `json:"creation_height"`
	CompletionTime int64 `json:"completion_time"`
	InitialBalance int64 `json:"initial_balance"`
	Balance        int64 `json:"balance"`
}

// Redelegation represents a redelegation from one validator to another
type Redelegation struct {
	DelegatorAddress    string              `json:"delegator_address"`
	ValidatorSrcAddress string              `json:"validator_src_address"`
	ValidatorDstAddress string              `json:"validator_dst_address"`
	Entries             []RedelegationEntry `json:"entries"`
}

// RedelegationEntry represents a single redelegation entry
type RedelegationEntry struct {
	CreationHeight int64 `json:"creation_height"`
	CompletionTime int64 `json:"completion_time"`
	InitialBalance int64 `json:"initial_balance"`
	SharesDst      int64 `json:"shares_dst"`
}

// Params defines the parameters for the staking module
type Params struct {
	UnbondingTime     int64  `json:"unbonding_time"`
	MaxValidators     uint32 `json:"max_validators"`
	MaxEntries        uint32 `json:"max_entries"`
	HistoricalEntries uint32 `json:"historical_entries"`
	BondDenom         string `json:"bond_denom"`
	MinCommissionRate int64  `json:"min_commission_rate"`
}

// DefaultParams returns default staking parameters
func DefaultParams() Params {
	return Params{
		UnbondingTime:     1814400, // 3 weeks in seconds
		MaxValidators:     100,
		MaxEntries:        7,
		HistoricalEntries: 10000,
		BondDenom:         "stake",
		MinCommissionRate: 0,
	}
}

// GenesisState defines the staking module genesis state
type GenesisState struct {
	Params               Params                `json:"params"`
	LastTotalPower       int64                 `json:"last_total_power"`
	LastValidatorPowers  []LastValidatorPower  `json:"last_validator_powers"`
	Validators           []Validator           `json:"validators"`
	Delegations          []Delegation          `json:"delegations"`
	UnbondingDelegations []UnbondingDelegation `json:"unbonding_delegations"`
	Redelegations        []Redelegation        `json:"redelegations"`
	Exported             bool                  `json:"exported"`
}

// LastValidatorPower represents the last validator power
type LastValidatorPower struct {
	Address string `json:"address"`
	Power   int64  `json:"power"`
}

// DefaultGenesisState returns default genesis state
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:               DefaultParams(),
		LastTotalPower:       0,
		LastValidatorPowers:  []LastValidatorPower{},
		Validators:           []Validator{},
		Delegations:          []Delegation{},
		UnbondingDelegations: []UnbondingDelegation{},
		Redelegations:        []Redelegation{},
		Exported:             false,
	}
}

// StakingKeeper defines the expected interface for the staking keeper
type StakingKeeper interface {
	keepertypes.KVStoreKeeper

	// Validator operations
	GetValidator(ctx keepertypes.Context, addr []byte) (Validator, bool)
	SetValidator(ctx keepertypes.Context, validator Validator)
	GetAllValidators(ctx keepertypes.Context) []Validator
	GetBondedValidatorsByPower(ctx keepertypes.Context) []Validator

	// Delegation operations
	GetDelegation(ctx keepertypes.Context, delAddr []byte, valAddr []byte) (Delegation, bool)
	SetDelegation(ctx keepertypes.Context, delegation Delegation)
	GetAllDelegations(ctx keepertypes.Context) []Delegation

	// Staking operations
	Delegate(ctx keepertypes.Context, delAddr []byte, valAddr []byte, amount int64) error
	Undelegate(ctx keepertypes.Context, delAddr []byte, valAddr []byte, shares int64) error
	BeginRedelegate(ctx keepertypes.Context, delAddr []byte, valSrcAddr []byte, valDstAddr []byte, shares int64) error

	// Power operations
	GetLastTotalPower(ctx keepertypes.Context) int64
	SetLastTotalPower(ctx keepertypes.Context, power int64)
	GetLastValidatorPower(ctx keepertypes.Context, valAddr []byte) int64
	SetLastValidatorPower(ctx keepertypes.Context, valAddr []byte, power int64)

	// Parameters
	GetParams(ctx keepertypes.Context) Params
	SetParams(ctx keepertypes.Context, params Params)
}

// BankKeeper defines expected interface for bank keeper
type BankKeeper interface {
	SendCoins(ctx keepertypes.Context, fromAddr, toAddr []byte, amount keepertypes.Coins) error
	SendCoinsFromModuleToAccount(ctx keepertypes.Context, senderModule string, recipientAddr []byte, amount keepertypes.Coins) error
	SendCoinsFromAccountToModule(ctx keepertypes.Context, senderAddr []byte, recipientModule string, amount keepertypes.Coins) error
	MintCoins(ctx keepertypes.Context, moduleName string, amount keepertypes.Coins) error
	BurnCoins(ctx keepertypes.Context, moduleName string, amount keepertypes.Coins) error
}

// SlashingKeeper defines expected interface for slashing keeper
type SlashingKeeper interface {
	Slash(ctx keepertypes.Context, consAddr []byte, fraction int64, power int64, distributionHeight int64)
	Jail(ctx keepertypes.Context, consAddr []byte)
	Unjail(ctx keepertypes.Context, consAddr []byte)
}
