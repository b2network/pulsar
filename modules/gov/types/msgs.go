package types

import (
	"github.com/b2network/pulsar/types"
)

// MsgSubmitProposal represents a message to submit a governance proposal
type MsgSubmitProposal struct {
	Messages       []types.Msg `json:"messages"`
	InitialDeposit types.Coins `json:"initial_deposit"`
	Proposer       string      `json:"proposer"`
	Metadata       string      `json:"metadata"`
	Title          string      `json:"title"`
	Summary        string      `json:"summary"`
	Expedited      bool        `json:"expedited"`
}

// ValidateBasic validates the MsgSubmitProposal
func (m MsgSubmitProposal) ValidateBasic() error {
	return nil
}

// GetSigners returns the signers for MsgSubmitProposal
func (m MsgSubmitProposal) GetSigners() [][]byte {
	return [][]byte{}
}

// MsgDeposit represents a message to deposit coins to a governance proposal
type MsgDeposit struct {
	ProposalId uint64      `json:"proposal_id"`
	Depositor  string      `json:"depositor"`
	Amount     types.Coins `json:"amount"`
}

// ValidateBasic validates the MsgDeposit
func (m MsgDeposit) ValidateBasic() error {
	return nil
}

// GetSigners returns the signers for MsgDeposit
func (m MsgDeposit) GetSigners() [][]byte {
	return [][]byte{}
}

// MsgVote represents a message to vote on a governance proposal
type MsgVote struct {
	ProposalId uint64 `json:"proposal_id"`
	Voter      string `json:"voter"`
	Option     int32  `json:"option"`
	Metadata   string `json:"metadata"`
}

// ValidateBasic validates the MsgVote
func (m MsgVote) ValidateBasic() error {
	return nil
}

// GetSigners returns the signers for MsgVote
func (m MsgVote) GetSigners() [][]byte {
	return [][]byte{}
}

// MsgVoteWeighted represents a message to vote with weighted options on a governance proposal
type MsgVoteWeighted struct {
	ProposalId uint64               `json:"proposal_id"`
	Voter      string               `json:"voter"`
	Options    []WeightedVoteOption `json:"options"`
	Metadata   string               `json:"metadata"`
}

// WeightedVoteOption represents a weighted vote option
type WeightedVoteOption struct {
	Option int32  `json:"option"`
	Weight string `json:"weight"`
}

// ValidateBasic validates the MsgVoteWeighted
func (m MsgVoteWeighted) ValidateBasic() error {
	return nil
}

// GetSigners returns the signers for MsgVoteWeighted
func (m MsgVoteWeighted) GetSigners() [][]byte {
	return [][]byte{}
}

// MsgExecLegacyContent represents a message to execute legacy content
type MsgExecLegacyContent struct {
	Content   types.Msg `json:"content"`
	Authority string    `json:"authority"`
}

// ValidateBasic validates the MsgExecLegacyContent
func (m MsgExecLegacyContent) ValidateBasic() error {
	return nil
}

// GetSigners returns the signers for MsgExecLegacyContent
func (m MsgExecLegacyContent) GetSigners() [][]byte {
	return [][]byte{}
}
