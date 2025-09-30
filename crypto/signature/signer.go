package signature

import (
	"encoding/json"
	"fmt"

	"github.com/b2network/pulsar/crypto/keys"
	"golang.org/x/crypto/sha3"
)

// Signer handles transaction signing
type Signer struct {
	chainID string
	keyring keys.Keyring
}

// NewSigner creates a new transaction signer
func NewSigner(chainID string, keyring keys.Keyring) *Signer {
	return &Signer{
		chainID: chainID,
		keyring: keyring,
	}
}

// SignTransaction signs a transaction with the specified key
func (s *Signer) SignTransaction(
	keyName string,
	passphrase string,
	accountNumber uint64,
	sequence uint64,
	fee Fee,
	messages []Message,
	memo string,
	timeoutHeight uint64,
) (*Signature, error) {
	// Create sign document
	signDoc := &SignDoc{
		ChainID:       s.chainID,
		AccountNumber: accountNumber,
		Sequence:      sequence,
		Fee:           fee,
		Messages:      messages,
		Memo:          memo,
		TimeoutHeight: timeoutHeight,
	}

	// Get canonical bytes for signing
	signBytes, err := signDoc.GetSignBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to get sign bytes: %w", err)
	}

	// Sign the document
	sig, pubKey, err := s.keyring.SignByKey(keyName, signBytes, passphrase)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	return NewSignature(pubKey, sig, sequence), nil
}

// SignTransactionByAddress signs a transaction with the key at the specified address
func (s *Signer) SignTransactionByAddress(
	address keys.Address,
	passphrase string,
	accountNumber uint64,
	sequence uint64,
	fee Fee,
	messages []Message,
	memo string,
	timeoutHeight uint64,
) (*Signature, error) {
	// Create sign document
	signDoc := &SignDoc{
		ChainID:       s.chainID,
		AccountNumber: accountNumber,
		Sequence:      sequence,
		Fee:           fee,
		Messages:      messages,
		Memo:          memo,
		TimeoutHeight: timeoutHeight,
	}

	// Get canonical bytes for signing
	signBytes, err := signDoc.GetSignBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to get sign bytes: %w", err)
	}

	// Sign the document
	sig, pubKey, err := s.keyring.SignByAddress(address, signBytes, passphrase)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	return NewSignature(pubKey, sig, sequence), nil
}

// SignMessage signs an arbitrary message
func (s *Signer) SignMessage(keyName string, passphrase string, msg []byte) (*StdSignature, error) {
	// Sign the message
	sig, pubKey, err := s.keyring.SignByKey(keyName, msg, passphrase)
	if err != nil {
		return nil, fmt.Errorf("failed to sign message: %w", err)
	}

	return NewStdSignature(sig, pubKey), nil
}

// SignMessageByAddress signs an arbitrary message with the key at the specified address
func (s *Signer) SignMessageByAddress(address keys.Address, passphrase string, msg []byte) (*StdSignature, error) {
	// Sign the message
	sig, pubKey, err := s.keyring.SignByAddress(address, msg, passphrase)
	if err != nil {
		return nil, fmt.Errorf("failed to sign message: %w", err)
	}

	return NewStdSignature(sig, pubKey), nil
}

// CreateMultiSig creates a multi-signature transaction
func (s *Signer) CreateMultiSig(
	threshold uint32,
	keyNames []string,
	passphrase string,
	accountNumber uint64,
	sequence uint64,
	fee Fee,
	messages []Message,
	memo string,
	timeoutHeight uint64,
) (*MultiSignature, error) {
	if threshold == 0 || threshold > uint32(len(keyNames)) {
		return nil, fmt.Errorf("invalid threshold: %d (must be 1-%d)", threshold, len(keyNames))
	}

	multiSig := NewMultiSignature(threshold)

	// Create sign document
	signDoc := &SignDoc{
		ChainID:       s.chainID,
		AccountNumber: accountNumber,
		Sequence:      sequence,
		Fee:           fee,
		Messages:      messages,
		Memo:          memo,
		TimeoutHeight: timeoutHeight,
	}

	// Get canonical bytes for signing
	signBytes, err := signDoc.GetSignBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to get sign bytes: %w", err)
	}

	// Sign with each key
	for _, keyName := range keyNames {
		sig, pubKey, err := s.keyring.SignByKey(keyName, signBytes, passphrase)
		if err != nil {
			return nil, fmt.Errorf("failed to sign with key %s: %w", keyName, err)
		}

		stdSig := NewStdSignature(sig, pubKey)
		multiSig.AddSignature(*stdSig)
	}

	return multiSig, nil
}

// GetChainID returns the chain ID
func (s *Signer) GetChainID() string {
	return s.chainID
}

// RecoverSignerAddress recovers the signer address from a signature
func RecoverSignerAddress(msg []byte, sig []byte) (keys.Address, error) {
	// Hash the message
	hash := sha3.NewLegacyKeccak256()
	hash.Write(msg)
	digest := hash.Sum(nil)

	// Recover public key
	pubKey, err := keys.RecoverPubKey(digest, sig)
	if err != nil {
		return keys.Address{}, fmt.Errorf("failed to recover public key: %w", err)
	}

	return pubKey.Address(), nil
}

// HashMessage hashes a message for signing (Ethereum compatible)
func HashMessage(msg []byte) []byte {
	hash := sha3.NewLegacyKeccak256()
	hash.Write(msg)
	return hash.Sum(nil)
}

// SignDocHash returns the hash of a sign document
func SignDocHash(signDoc *SignDoc) ([]byte, error) {
	signBytes, err := signDoc.GetSignBytes()
	if err != nil {
		return nil, err
	}
	return HashMessage(signBytes), nil
}

// CreateSignDoc creates a sign document from transaction components
func CreateSignDoc(
	chainID string,
	accountNumber uint64,
	sequence uint64,
	fee Fee,
	messages []Message,
	memo string,
	timeoutHeight uint64,
) *SignDoc {
	return &SignDoc{
		ChainID:       chainID,
		AccountNumber: accountNumber,
		Sequence:      sequence,
		Fee:           fee,
		Messages:      messages,
		Memo:          memo,
		TimeoutHeight: timeoutHeight,
	}
}

// VerifySignDoc verifies a sign document signature
func VerifySignDoc(signDoc *SignDoc, sig *Signature) bool {
	signBytes, err := signDoc.GetSignBytes()
	if err != nil {
		return false
	}

	return sig.Verify(signBytes)
}

// MarshalSignDoc marshals a sign document to JSON bytes
func MarshalSignDoc(signDoc *SignDoc) ([]byte, error) {
	return json.Marshal(signDoc)
}

// UnmarshalSignDoc unmarshals a sign document from JSON bytes
func UnmarshalSignDoc(data []byte) (*SignDoc, error) {
	var signDoc SignDoc
	err := json.Unmarshal(data, &signDoc)
	return &signDoc, err
}