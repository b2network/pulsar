package types

import (
	"github.com/b2network/pulsar/types"
)

// MsgDelegate represents a message to delegate coins to a validator
type MsgDelegate struct {
	DelegatorAddress string     `json:"delegator_address"`
	ValidatorAddress string     `json:"validator_address"`
	Amount           types.Coin `json:"amount"`
}

// ValidateBasic validates the MsgDelegate
func (m MsgDelegate) ValidateBasic() error {
	return nil
}

// GetSigners returns the signers for MsgDelegate
func (m MsgDelegate) GetSigners() [][]byte {
	return [][]byte{}
}

// MsgUndelegate represents a message to undelegate coins from a validator
type MsgUndelegate struct {
	DelegatorAddress string     `json:"delegator_address"`
	ValidatorAddress string     `json:"validator_address"`
	Amount           types.Coin `json:"amount"`
}

// ValidateBasic validates the MsgUndelegate
func (m MsgUndelegate) ValidateBasic() error {
	return nil
}

// GetSigners returns the signers for MsgUndelegate
func (m MsgUndelegate) GetSigners() [][]byte {
	return [][]byte{}
}

// MsgBeginRedelegate represents a message to redelegate coins between validators
type MsgBeginRedelegate struct {
	DelegatorAddress    string     `json:"delegator_address"`
	ValidatorSrcAddress string     `json:"validator_src_address"`
	ValidatorDstAddress string     `json:"validator_dst_address"`
	Amount              types.Coin `json:"amount"`
}

// ValidateBasic validates the MsgBeginRedelegate
func (m MsgBeginRedelegate) ValidateBasic() error {
	return nil
}

// GetSigners returns the signers for MsgBeginRedelegate
func (m MsgBeginRedelegate) GetSigners() [][]byte {
	return [][]byte{}
}

// MsgCreateValidator represents a message to create a new validator
type MsgCreateValidator struct {
	ValidatorAddress  string     `json:"validator_address"`
	Moniker           string     `json:"moniker"`
	Commission        string     `json:"commission"`
	MinSelfDelegation string     `json:"min_self_delegation"`
	DelegatorAddress  string     `json:"delegator_address"`
	Value             types.Coin `json:"value"`
}

// ValidateBasic validates the MsgCreateValidator
func (m MsgCreateValidator) ValidateBasic() error {
	return nil
}

// GetSigners returns the signers for MsgCreateValidator
func (m MsgCreateValidator) GetSigners() [][]byte {
	return [][]byte{}
}

// MsgEditValidator represents a message to edit validator information
type MsgEditValidator struct {
	ValidatorAddress  string `json:"validator_address"`
	Description       string `json:"description"`
	CommissionRate    string `json:"commission_rate"`
	MinSelfDelegation string `json:"min_self_delegation"`
}

// ValidateBasic validates the MsgEditValidator
func (m MsgEditValidator) ValidateBasic() error {
	return nil
}

// GetSigners returns the signers for MsgEditValidator
func (m MsgEditValidator) GetSigners() [][]byte {
	return [][]byte{}
}

// MsgCancelUnbondingDelegation represents a message to cancel unbonding delegation
type MsgCancelUnbondingDelegation struct {
	DelegatorAddress string     `json:"delegator_address"`
	ValidatorAddress string     `json:"validator_address"`
	Amount           types.Coin `json:"amount"`
	CreationHeight   int64      `json:"creation_height"`
}

// ValidateBasic validates the MsgCancelUnbondingDelegation
func (m MsgCancelUnbondingDelegation) ValidateBasic() error {
	return nil
}

// GetSigners returns the signers for MsgCancelUnbondingDelegation
func (m MsgCancelUnbondingDelegation) GetSigners() [][]byte {
	return [][]byte{}
}
