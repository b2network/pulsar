package types

import (
	"github.com/b2network/pulsar/types"
)

// MsgSend represents a message to send coins from one account to another
type MsgSend struct {
	FromAddress string      `json:"from_address"`
	ToAddress   string      `json:"to_address"`
	Amount      types.Coins `json:"amount"`
}

// ValidateBasic validates the MsgSend
func (m MsgSend) ValidateBasic() error {
	// Basic validation would go here
	return nil
}

// GetSigners returns the signers for MsgSend
func (m MsgSend) GetSigners() [][]byte {
	// Return signer addresses
	return [][]byte{}
}

// MsgMultiSend represents a message to send coins from multiple inputs to multiple outputs
type MsgMultiSend struct {
	Inputs  []Input     `json:"inputs"`
	Outputs []Output    `json:"outputs"`
	Amount  types.Coins `json:"amount"`
}

// Input represents an input for multi-send
type Input struct {
	Address string      `json:"address"`
	Coins   types.Coins `json:"coins"`
}

// Output represents an output for multi-send
type Output struct {
	Address string      `json:"address"`
	Coins   types.Coins `json:"coins"`
}

// ValidateBasic validates the MsgMultiSend
func (m MsgMultiSend) ValidateBasic() error {
	// Basic validation would go here
	return nil
}

// GetSigners returns the signers for MsgMultiSend
func (m MsgMultiSend) GetSigners() [][]byte {
	// Return signer addresses
	return [][]byte{}
}
