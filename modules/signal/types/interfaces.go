package types

import (
	keepertypes "github.com/b2network/pulsar/keeper/types"
)

// MsgServer defines the signal module's message server interface
type MsgServer interface {
	// SubmitSignal handles signal submission messages
	SubmitSignal(ctx keepertypes.Context, msg *MsgSubmitSignal) (*MsgSubmitSignalResponse, error)

	// VerifyWork handles work verification messages
	VerifyWork(ctx keepertypes.Context, msg *MsgVerifyWork) (*MsgVerifyWorkResponse, error)

	// UpdateWorkType handles work type update messages
	UpdateWorkType(ctx keepertypes.Context, msg *MsgUpdateWorkType) (*MsgUpdateWorkTypeResponse, error)

	// ClaimReward handles reward claiming messages
	ClaimReward(ctx keepertypes.Context, msg *MsgClaimReward) (*MsgClaimRewardResponse, error)

	// BatchSubmit handles batch signal submission messages
	BatchSubmit(ctx keepertypes.Context, msg *MsgBatchSubmit) (*MsgBatchSubmitResponse, error)

	// UpdateParams handles parameter update messages
	UpdateParams(ctx keepertypes.Context, msg *MsgUpdateParams) (*MsgUpdateParamsResponse, error)

	// RegisterWorkType handles work type registration messages
	RegisterWorkType(ctx keepertypes.Context, msg *MsgRegisterWorkType) (*MsgRegisterWorkTypeResponse, error)
}

// Message response types

// MsgSubmitSignalResponse defines the response for MsgSubmitSignal
type MsgSubmitSignalResponse struct {
	SignalID string `json:"signal_id"`
}

// MsgVerifyWorkResponse defines the response for MsgVerifyWork
type MsgVerifyWorkResponse struct {
	VerificationScore uint64 `json:"verification_score"`
}

// MsgUpdateWorkTypeResponse defines the response for MsgUpdateWorkType
type MsgUpdateWorkTypeResponse struct{}

// MsgClaimRewardResponse defines the response for MsgClaimReward
type MsgClaimRewardResponse struct {
	RewardAmount keepertypes.Coins `json:"reward_amount"`
}

// MsgBatchSubmitResponse defines the response for MsgBatchSubmit
type MsgBatchSubmitResponse struct {
	BatchID          string   `json:"batch_id"`
	SubmittedSignals []string `json:"submitted_signals"`
	TotalScore       uint64   `json:"total_score"`
}

// MsgUpdateParamsResponse defines the response for MsgUpdateParams
type MsgUpdateParamsResponse struct{}

// MsgRegisterWorkTypeResponse defines the response for MsgRegisterWorkType
type MsgRegisterWorkTypeResponse struct{}

// Keeper interfaces for external dependencies

// BankKeeper defines the expected interface for the bank module
type BankKeeper interface {
	// SendCoinsFromModuleToModule transfers coins between module accounts
	SendCoinsFromModuleToModule(ctx keepertypes.Context, senderModule, recipientModule string, amt keepertypes.Coins) error

	// SendCoinsFromModuleToAccount transfers coins from module account to user account
	SendCoinsFromModuleToAccount(ctx keepertypes.Context, senderModule string, recipientAddr []byte, amt keepertypes.Coins) error

	// SendCoinsFromAccountToModule transfers coins from user account to module account
	SendCoinsFromAccountToModule(ctx keepertypes.Context, senderAddr []byte, recipientModule string, amt keepertypes.Coins) error

	// BurnCoins burns coins from a module account
	BurnCoins(ctx keepertypes.Context, moduleName string, amounts keepertypes.Coins) error

	// GetAllBalances returns all balances for an account
	GetAllBalances(ctx keepertypes.Context, addr []byte) keepertypes.Coins

	// GetBalance returns the balance of a specific denom for an account
	GetBalance(ctx keepertypes.Context, addr []byte, denom string) keepertypes.Coin

	// GetModuleAccount returns the module account for a given module name
	GetModuleAccount(ctx keepertypes.Context, moduleName string) []byte
}

// StakingKeeper defines the expected interface for the staking module
type StakingKeeper interface {
	// GetValidator returns a validator by operator address
	GetValidator(ctx keepertypes.Context, addr []byte) (Validator, bool)

	// GetAllValidators returns all validators
	GetAllValidators(ctx keepertypes.Context) []Validator

	// GetBondedValidatorsByPower returns bonded validators sorted by power
	GetBondedValidatorsByPower(ctx keepertypes.Context) []Validator

	// GetValidatorDelegations returns all delegations to a validator
	GetValidatorDelegations(ctx keepertypes.Context, valAddr []byte) []Delegation

	// Slash slashes a validator
	Slash(ctx keepertypes.Context, consAddr []byte, infractionHeight int64, power int64, slashFactor float64)

	// Jail jails a validator
	Jail(ctx keepertypes.Context, consAddr []byte)

	// Unjail unjails a validator
	Unjail(ctx keepertypes.Context, consAddr []byte)
}

// DistributionKeeper defines the expected interface for the distribution module
type DistributionKeeper interface {
	// AllocateTokensToValidator allocates tokens to a validator
	AllocateTokensToValidator(ctx keepertypes.Context, val Validator, tokens keepertypes.Coins) error

	// GetCommunityPool returns the community pool balance
	GetCommunityPool(ctx keepertypes.Context) keepertypes.Coins

	// FundCommunityPool funds the community pool
	FundCommunityPool(ctx keepertypes.Context, amount keepertypes.Coins, sender []byte) error

	// DistributeFromFeePool distributes from the fee pool
	DistributeFromFeePool(ctx keepertypes.Context, amount keepertypes.Coins, receiveAddr []byte) error
}

// Simplified types for external dependencies (would be imported from actual modules)

// Validator represents a validator in the staking module
type Validator struct {
	OperatorAddress string                 `json:"operator_address"`
	ConsPubKey      []byte                 `json:"consensus_pubkey"`
	Jailed          bool                   `json:"jailed"`
	Status          int32                  `json:"status"`
	Tokens          int64                  `json:"tokens"`
	DelegatorShares int64                  `json:"delegator_shares"`
	Commission      Commission             `json:"commission"`
	Description     ValidatorDescription   `json:"description"`
}

// Commission defines commission parameters for a validator
type Commission struct {
	Rate          int64 `json:"rate"`
	MaxRate       int64 `json:"max_rate"`
	MaxChangeRate int64 `json:"max_change_rate"`
	UpdateTime    int64 `json:"update_time"`
}

// ValidatorDescription contains validator description information
type ValidatorDescription struct {
	Moniker         string `json:"moniker"`
	Identity        string `json:"identity"`
	Website         string `json:"website"`
	SecurityContact string `json:"security_contact"`
	Details         string `json:"details"`
}

// Delegation represents a delegation from a delegator to a validator
type Delegation struct {
	DelegatorAddress string `json:"delegator_address"`
	ValidatorAddress string `json:"validator_address"`
	Shares           int64  `json:"shares"`
}

// SignalKeeper defines the interface for signal keeper operations
type SignalKeeper interface {
	// Signal operations
	SubmitSignal(ctx keepertypes.Context, signal Signal) error
	ProcessSignal(ctx keepertypes.Context, signalID string) error
	GetSignal(ctx keepertypes.Context, signalID string) (Signal, bool)
	GetAllSignals(ctx keepertypes.Context) []Signal
	GetValidatorSignals(ctx keepertypes.Context, validatorAddr string) []Signal
	GetSignalsByStatus(ctx keepertypes.Context, status SignalStatus) []Signal
	FilterSignals(ctx keepertypes.Context, filter SignalFilter) []Signal

	// Score operations
	CalculateSignalScore(ctx keepertypes.Context, signal Signal) (SignalScore, error)
	GetSignalScore(ctx keepertypes.Context, signalID string) (SignalScore, bool)
	UpdateValidatorStats(ctx keepertypes.Context, validatorAddr string, signal Signal, score uint64) error
	GetValidatorStats(ctx keepertypes.Context, validatorAddr string) ValidatorSignalStats

	// Work type operations
	GetWorkType(ctx keepertypes.Context, workTypeID string) (AIWorkType, bool)
	GetAllWorkTypes(ctx keepertypes.Context) []AIWorkType
	StoreWorkType(ctx keepertypes.Context, workType AIWorkType) error

	// Parameter operations
	GetParams(ctx keepertypes.Context) Params
	SetParams(ctx keepertypes.Context, params Params) error

	// Verification operations
	StoreWorkVerification(ctx keepertypes.Context, verification WorkVerification) error
	GetWorkVerifications(ctx keepertypes.Context, signalID string) []WorkVerification

	// Utility
	Logger(ctx keepertypes.Context) keepertypes.Logger
	IsAuthorized(address string) bool
}

// QueryServer defines the signal module's query server interface
type QueryServer interface {
	// Signal queries
	Signal(ctx keepertypes.Context, req *QuerySignalRequest) (*QuerySignalResponse, error)
	Signals(ctx keepertypes.Context, req *QuerySignalsRequest) (*QuerySignalsResponse, error)
	ValidatorSignals(ctx keepertypes.Context, req *QueryValidatorSignalsRequest) (*QueryValidatorSignalsResponse, error)

	// Score queries
	SignalScore(ctx keepertypes.Context, req *QuerySignalScoreRequest) (*QuerySignalScoreResponse, error)
	ValidatorStats(ctx keepertypes.Context, req *QueryValidatorStatsRequest) (*QueryValidatorStatsResponse, error)
	Leaderboard(ctx keepertypes.Context, req *QueryLeaderboardRequest) (*QueryLeaderboardResponse, error)

	// Work type queries
	WorkType(ctx keepertypes.Context, req *QueryWorkTypeRequest) (*QueryWorkTypeResponse, error)
	WorkTypes(ctx keepertypes.Context, req *QueryWorkTypesRequest) (*QueryWorkTypesResponse, error)

	// Statistics queries
	SignalStats(ctx keepertypes.Context, req *QuerySignalStatsRequest) (*QuerySignalStatsResponse, error)

	// Parameter queries
	Params(ctx keepertypes.Context, req *QueryParamsRequest) (*QueryParamsResponse, error)
}

// Query request and response types

// QuerySignalRequest is the request type for the Signal query
type QuerySignalRequest struct {
	SignalID string `json:"signal_id"`
}

// QuerySignalResponse is the response type for the Signal query
type QuerySignalResponse struct {
	Signal Signal      `json:"signal"`
	Score  *SignalScore `json:"score,omitempty"`
}

// QuerySignalsRequest is the request type for the Signals query
type QuerySignalsRequest struct {
	Filter SignalFilter `json:"filter"`
	Limit  int          `json:"limit"`
	Offset int          `json:"offset"`
}

// QuerySignalsResponse is the response type for the Signals query
type QuerySignalsResponse struct {
	Signals []Signal `json:"signals"`
	Total   int      `json:"total"`
}

// QueryValidatorSignalsRequest is the request type for the ValidatorSignals query
type QueryValidatorSignalsRequest struct {
	ValidatorAddress string `json:"validator_address"`
	Limit           int    `json:"limit"`
	Offset          int    `json:"offset"`
}

// QueryValidatorSignalsResponse is the response type for the ValidatorSignals query
type QueryValidatorSignalsResponse struct {
	Signals []Signal `json:"signals"`
	Total   int      `json:"total"`
}

// QuerySignalScoreRequest is the request type for the SignalScore query
type QuerySignalScoreRequest struct {
	SignalID string `json:"signal_id"`
}

// QuerySignalScoreResponse is the response type for the SignalScore query
type QuerySignalScoreResponse struct {
	Score SignalScore `json:"score"`
}

// QueryValidatorStatsRequest is the request type for the ValidatorStats query
type QueryValidatorStatsRequest struct {
	ValidatorAddress string `json:"validator_address"`
}

// QueryValidatorStatsResponse is the response type for the ValidatorStats query
type QueryValidatorStatsResponse struct {
	Stats ValidatorSignalStats `json:"stats"`
	Rank  uint32              `json:"rank"`
}

// QueryLeaderboardRequest is the request type for the Leaderboard query
type QueryLeaderboardRequest struct {
	Limit int `json:"limit"`
}

// QueryLeaderboardResponse is the response type for the Leaderboard query
type QueryLeaderboardResponse struct {
	Leaderboard SignalLeaderboard `json:"leaderboard"`
}

// QueryWorkTypeRequest is the request type for the WorkType query
type QueryWorkTypeRequest struct {
	WorkTypeID string `json:"work_type_id"`
}

// QueryWorkTypeResponse is the response type for the WorkType query
type QueryWorkTypeResponse struct {
	WorkType AIWorkType `json:"work_type"`
}

// QueryWorkTypesRequest is the request type for the WorkTypes query
type QueryWorkTypesRequest struct {
	EnabledOnly bool `json:"enabled_only"`
}

// QueryWorkTypesResponse is the response type for the WorkTypes query
type QueryWorkTypesResponse struct {
	WorkTypes []AIWorkType `json:"work_types"`
}

// QuerySignalStatsRequest is the request type for the SignalStats query
type QuerySignalStatsRequest struct{}

// QuerySignalStatsResponse is the response type for the SignalStats query
type QuerySignalStatsResponse struct {
	Stats map[string]interface{} `json:"stats"`
}

// QueryParamsRequest is the request type for the Params query
type QueryParamsRequest struct{}

// QueryParamsResponse is the response type for the Params query
type QueryParamsResponse struct {
	Params Params `json:"params"`
}