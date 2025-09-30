package signature

import (
	"encoding/json"
	"fmt"

	"github.com/b2network/pulsar/crypto/keys"
)

// Signature represents a cryptographic signature
type Signature struct {
	PubKey    keys.PubKey `json:"pub_key"`
	Signature []byte      `json:"signature"`
	Sequence  uint64      `json:"sequence"`
}

// NewSignature creates a new signature
func NewSignature(pubKey keys.PubKey, sig []byte, sequence uint64) *Signature {
	return &Signature{
		PubKey:    pubKey,
		Signature: sig,
		Sequence:  sequence,
	}
}

// Verify verifies the signature against a message
func (s *Signature) Verify(msg []byte) bool {
	return s.PubKey.VerifySignature(msg, s.Signature)
}

// GetAddress returns the address associated with this signature
func (s *Signature) GetAddress() keys.Address {
	return s.PubKey.Address()
}

// SignDoc represents the document to be signed
type SignDoc struct {
	ChainID       string      `json:"chain_id"`
	AccountNumber uint64      `json:"account_number"`
	Sequence      uint64      `json:"sequence"`
	Fee           Fee         `json:"fee"`
	Messages      []Message   `json:"msgs"`
	Memo          string      `json:"memo"`
	TimeoutHeight uint64      `json:"timeout_height"`
}

// Message represents a transaction message for signing
type Message struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

// Fee represents transaction fees
type Fee struct {
	Amount []Coin `json:"amount"`
	Gas    uint64 `json:"gas"`
}

// Coin represents a fee coin
type Coin struct {
	Denom  string `json:"denom"`
	Amount string `json:"amount"`
}

// GetSignBytes returns the canonical bytes for signing
func (sd *SignDoc) GetSignBytes() ([]byte, error) {
	return json.Marshal(sd)
}

// StdSignature represents a standard signature with minimal data
type StdSignature struct {
	Signature []byte `json:"signature"`
	PubKey    []byte `json:"pub_key,omitempty"`
}

// NewStdSignature creates a new standard signature
func NewStdSignature(sig []byte, pubKey keys.PubKey) *StdSignature {
	var pubKeyBytes []byte
	if pubKey != nil {
		pubKeyBytes = pubKey.Bytes()
	}

	return &StdSignature{
		Signature: sig,
		PubKey:    pubKeyBytes,
	}
}

// GetPubKey returns the public key
func (s *StdSignature) GetPubKey() (keys.PubKey, error) {
	if len(s.PubKey) == 0 {
		return nil, fmt.Errorf("no public key in signature")
	}

	return keys.NewSecp256k1PubKey(s.PubKey)
}

// MultiSignature represents a multi-signature
type MultiSignature struct {
	Signatures []StdSignature `json:"signatures"`
	Threshold  uint32         `json:"threshold"`
}

// NewMultiSignature creates a new multi-signature
func NewMultiSignature(threshold uint32) *MultiSignature {
	return &MultiSignature{
		Signatures: make([]StdSignature, 0),
		Threshold:  threshold,
	}
}

// AddSignature adds a signature to the multi-signature
func (ms *MultiSignature) AddSignature(sig StdSignature) {
	ms.Signatures = append(ms.Signatures, sig)
}

// IsComplete returns true if enough signatures have been collected
func (ms *MultiSignature) IsComplete() bool {
	return uint32(len(ms.Signatures)) >= ms.Threshold
}

// Verify verifies all signatures in the multi-signature
func (ms *MultiSignature) Verify(msg []byte) bool {
	if !ms.IsComplete() {
		return false
	}

	validSigs := 0
	for _, sig := range ms.Signatures {
		pubKey, err := sig.GetPubKey()
		if err != nil {
			continue
		}

		if pubKey.VerifySignature(msg, sig.Signature) {
			validSigs++
		}
	}

	return uint32(validSigs) >= ms.Threshold
}