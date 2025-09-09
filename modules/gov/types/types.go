package types

import (
	keepertypes "github.com/b2network/pulsar/keeper/types"
)

// ModuleName is the name of the governance module
const ModuleName = "gov"

// StoreKey is the store key for the governance module
const StoreKey = ModuleName

// Events
const (
	EventTypeSubmitProposal   = "submit_proposal"
	EventTypeProposalDeposit  = "proposal_deposit"
	EventTypeProposalVote     = "proposal_vote"
	EventTypeInactiveProposal = "inactive_proposal"
	EventTypeActiveProposal   = "active_proposal"
)

// Event attribute keys
const (
	AttributeKeyProposalID       = "proposal_id"
	AttributeKeyProposalType     = "proposal_type"
	AttributeKeyProposalResult   = "proposal_result"
	AttributeKeyVoter            = "voter"
	AttributeKeyDepositor        = "depositor"
	AttributeKeyOption           = "option"
	AttributeKeyProposalMessages = "proposal_messages"
)

// ProposalStatus represents the status of a governance proposal
type ProposalStatus int32

const (
	StatusNil           ProposalStatus = 0
	StatusDepositPeriod ProposalStatus = 1
	StatusVotingPeriod  ProposalStatus = 2
	StatusPassed        ProposalStatus = 3
	StatusRejected      ProposalStatus = 4
	StatusFailed        ProposalStatus = 5
)

// VoteOption represents a vote option
type VoteOption int32

const (
	OptionEmpty      VoteOption = 0
	OptionYes        VoteOption = 1
	OptionAbstain    VoteOption = 2
	OptionNo         VoteOption = 3
	OptionNoWithVeto VoteOption = 4
)

// Proposal represents a governance proposal
type Proposal struct {
	ID               uint64             `json:"id"`
	Messages         []ProposalMessage  `json:"messages"`
	Status           ProposalStatus     `json:"status"`
	FinalTallyResult TallyResult        `json:"final_tally_result"`
	SubmitTime       int64              `json:"submit_time"`
	DepositEndTime   int64              `json:"deposit_end_time"`
	TotalDeposit     []keepertypes.Coin `json:"total_deposit"`
	VotingStartTime  int64              `json:"voting_start_time"`
	VotingEndTime    int64              `json:"voting_end_time"`
	Title            string             `json:"title"`
	Summary          string             `json:"summary"`
	Proposer         string             `json:"proposer"`
}

// ProposalMessage represents a message contained in a proposal
type ProposalMessage struct {
	TypeURL string `json:"type_url"`
	Value   []byte `json:"value"`
}

// TallyResult represents the tally result of a proposal
type TallyResult struct {
	YesCount        int64 `json:"yes_count"`
	AbstainCount    int64 `json:"abstain_count"`
	NoCount         int64 `json:"no_count"`
	NoWithVetoCount int64 `json:"no_with_veto_count"`
}

// Vote represents a vote on a proposal
type Vote struct {
	ProposalID uint64       `json:"proposal_id"`
	Voter      string       `json:"voter"`
	Options    []VoteOption `json:"options"`
	Metadata   string       `json:"metadata"`
}

// Deposit represents a deposit on a proposal
type Deposit struct {
	ProposalID uint64             `json:"proposal_id"`
	Depositor  string             `json:"depositor"`
	Amount     []keepertypes.Coin `json:"amount"`
}

// DepositParams defines the parameters for proposal deposits
type DepositParams struct {
	MinDeposit       []keepertypes.Coin `json:"min_deposit"`
	MaxDepositPeriod int64              `json:"max_deposit_period"`
}

// VotingParams defines the parameters for voting
type VotingParams struct {
	VotingPeriod int64 `json:"voting_period"`
}

// TallyParams defines the parameters for tallying
type TallyParams struct {
	Quorum        int64 `json:"quorum"`         // Minimum percentage of total stake needed to vote for a result to be considered valid
	Threshold     int64 `json:"threshold"`      // Minimum proportion of Yes votes for proposal to pass
	VetoThreshold int64 `json:"veto_threshold"` // Minimum value of Veto votes to Total votes ratio for proposal to be vetoed
}

// Params defines the parameters for the governance module
type Params struct {
	MinDeposit       []keepertypes.Coin `json:"min_deposit"`
	MaxDepositPeriod int64              `json:"max_deposit_period"`
	VotingPeriod     int64              `json:"voting_period"`
	Quorum           int64              `json:"quorum"`
	Threshold        int64              `json:"threshold"`
	VetoThreshold    int64              `json:"veto_threshold"`
}

// DefaultParams returns default governance parameters
func DefaultParams() Params {
	return Params{
		MinDeposit: []keepertypes.Coin{{
			Denom:  "stake",
			Amount: 10000000, // 10 tokens
		}},
		MaxDepositPeriod: 172800, // 2 days in seconds
		VotingPeriod:     172800, // 2 days in seconds
		Quorum:           334000, // 33.4%
		Threshold:        500000, // 50%
		VetoThreshold:    334000, // 33.4%
	}
}

// GenesisState defines the governance module genesis state
type GenesisState struct {
	StartingProposalID uint64     `json:"starting_proposal_id"`
	Deposits           []Deposit  `json:"deposits"`
	Votes              []Vote     `json:"votes"`
	Proposals          []Proposal `json:"proposals"`
	Params             Params     `json:"params"`
}

// DefaultGenesisState returns default genesis state
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		StartingProposalID: 1,
		Deposits:           []Deposit{},
		Votes:              []Vote{},
		Proposals:          []Proposal{},
		Params:             DefaultParams(),
	}
}

// GovKeeper defines the expected interface for the governance keeper
type GovKeeper interface {
	keepertypes.KVStoreKeeper

	// Proposal operations
	GetProposal(ctx keepertypes.Context, proposalID uint64) (Proposal, bool)
	SetProposal(ctx keepertypes.Context, proposal Proposal)
	GetProposals(ctx keepertypes.Context) []Proposal
	GetProposalID(ctx keepertypes.Context) uint64
	SetProposalID(ctx keepertypes.Context, proposalID uint64)

	// Submit and activate proposals
	SubmitProposal(ctx keepertypes.Context, messages []ProposalMessage, title, summary, proposer string) (uint64, error)
	ActivateVotingPeriod(ctx keepertypes.Context, proposal Proposal)

	// Deposit operations
	GetDeposit(ctx keepertypes.Context, proposalID uint64, depositorAddr []byte) (Deposit, bool)
	SetDeposit(ctx keepertypes.Context, deposit Deposit)
	GetDeposits(ctx keepertypes.Context, proposalID uint64) []Deposit
	AddDeposit(ctx keepertypes.Context, proposalID uint64, depositorAddr []byte, depositAmount []keepertypes.Coin) (bool, error)

	// Vote operations
	GetVote(ctx keepertypes.Context, proposalID uint64, voterAddr []byte) (Vote, bool)
	SetVote(ctx keepertypes.Context, vote Vote)
	GetVotes(ctx keepertypes.Context, proposalID uint64) []Vote
	AddVote(ctx keepertypes.Context, proposalID uint64, voterAddr []byte, options []VoteOption, metadata string) error

	// Tally operations
	Tally(ctx keepertypes.Context, proposal Proposal) (passes bool, burnDeposits bool, tallyResults TallyResult)

	// Parameters
	GetParams(ctx keepertypes.Context) Params
	SetParams(ctx keepertypes.Context, params Params)
}

// BankKeeper defines expected interface for bank keeper
type BankKeeper interface {
	SendCoinsFromAccountToModule(ctx keepertypes.Context, senderAddr []byte, recipientModule string, amount keepertypes.Coins) error
	SendCoinsFromModuleToAccount(ctx keepertypes.Context, senderModule string, recipientAddr []byte, amount keepertypes.Coins) error
	BurnCoins(ctx keepertypes.Context, moduleName string, amount keepertypes.Coins) error
}

// StakingKeeper defines expected interface for staking keeper
type StakingKeeper interface {
	GetBondedValidatorsByPower(ctx keepertypes.Context) []interface{} // Simplified interface
	GetLastTotalPower(ctx keepertypes.Context) int64
}

// String returns the string representation of ProposalStatus
func (status ProposalStatus) String() string {
	switch status {
	case StatusDepositPeriod:
		return "DepositPeriod"
	case StatusVotingPeriod:
		return "VotingPeriod"
	case StatusPassed:
		return "Passed"
	case StatusRejected:
		return "Rejected"
	case StatusFailed:
		return "Failed"
	default:
		return "Nil"
	}
}
