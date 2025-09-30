package signature

import (
	"fmt"

	"github.com/b2network/pulsar/crypto/keys"
)

// Verifier handles signature verification
type Verifier struct {
	chainID string
}

// NewVerifier creates a new signature verifier
func NewVerifier(chainID string) *Verifier {
	return &Verifier{
		chainID: chainID,
	}
}

// VerifySignature verifies a signature against a message
func (v *Verifier) VerifySignature(msg []byte, sig *Signature) bool {
	return sig.Verify(msg)
}

// VerifyTransaction verifies a transaction signature
func (v *Verifier) VerifyTransaction(
	sig *Signature,
	accountNumber uint64,
	sequence uint64,
	fee Fee,
	messages []Message,
	memo string,
	timeoutHeight uint64,
) bool {
	// Create sign document
	signDoc := &SignDoc{
		ChainID:       v.chainID,
		AccountNumber: accountNumber,
		Sequence:      sequence,
		Fee:           fee,
		Messages:      messages,
		Memo:          memo,
		TimeoutHeight: timeoutHeight,
	}

	// Get canonical bytes
	signBytes, err := signDoc.GetSignBytes()
	if err != nil {
		return false
	}

	// Verify signature
	return sig.Verify(signBytes)
}

// VerifyMultiSignature verifies a multi-signature
func (v *Verifier) VerifyMultiSignature(
	multiSig *MultiSignature,
	accountNumber uint64,
	sequence uint64,
	fee Fee,
	messages []Message,
	memo string,
	timeoutHeight uint64,
) bool {
	// Create sign document
	signDoc := &SignDoc{
		ChainID:       v.chainID,
		AccountNumber: accountNumber,
		Sequence:      sequence,
		Fee:           fee,
		Messages:      messages,
		Memo:          memo,
		TimeoutHeight: timeoutHeight,
	}

	// Get canonical bytes
	signBytes, err := signDoc.GetSignBytes()
	if err != nil {
		return false
	}

	// Verify multi-signature
	return multiSig.Verify(signBytes)
}

// VerifyStdSignature verifies a standard signature
func (v *Verifier) VerifyStdSignature(msg []byte, stdSig *StdSignature) bool {
	pubKey, err := stdSig.GetPubKey()
	if err != nil {
		return false
	}

	return pubKey.VerifySignature(msg, stdSig.Signature)
}

// RecoverSigner recovers the signer's address from a signature
func (v *Verifier) RecoverSigner(msg []byte, sig []byte) (keys.Address, error) {
	return RecoverSignerAddress(msg, sig)
}

// ValidateSignature validates signature format and components
func (v *Verifier) ValidateSignature(sig *Signature) error {
	if sig == nil {
		return fmt.Errorf("signature is nil")
	}

	if sig.PubKey == nil {
		return fmt.Errorf("public key is nil")
	}

	if len(sig.Signature) == 0 {
		return fmt.Errorf("signature bytes are empty")
	}

	// Validate signature length (65 bytes for secp256k1)
	if sig.PubKey.Type() == string(keys.Secp256k1KeyType) && len(sig.Signature) != keys.Secp256k1SignatureSize {
		return fmt.Errorf("invalid signature length: expected %d, got %d",
			keys.Secp256k1SignatureSize, len(sig.Signature))
	}

	return nil
}

// ValidateStdSignature validates standard signature format
func (v *Verifier) ValidateStdSignature(stdSig *StdSignature) error {
	if stdSig == nil {
		return fmt.Errorf("signature is nil")
	}

	if len(stdSig.Signature) == 0 {
		return fmt.Errorf("signature bytes are empty")
	}

	if len(stdSig.PubKey) > 0 {
		_, err := keys.NewSecp256k1PubKey(stdSig.PubKey)
		if err != nil {
			return fmt.Errorf("invalid public key: %w", err)
		}
	}

	return nil
}

// ValidateMultiSignature validates multi-signature format
func (v *Verifier) ValidateMultiSignature(multiSig *MultiSignature) error {
	if multiSig == nil {
		return fmt.Errorf("multi-signature is nil")
	}

	if multiSig.Threshold == 0 {
		return fmt.Errorf("threshold must be greater than 0")
	}

	if len(multiSig.Signatures) == 0 {
		return fmt.Errorf("no signatures provided")
	}

	if uint32(len(multiSig.Signatures)) < multiSig.Threshold {
		return fmt.Errorf("insufficient signatures: need %d, got %d",
			multiSig.Threshold, len(multiSig.Signatures))
	}

	// Validate each signature
	for i, sig := range multiSig.Signatures {
		if err := v.ValidateStdSignature(&sig); err != nil {
			return fmt.Errorf("invalid signature at index %d: %w", i, err)
		}
	}

	return nil
}

// BatchVerify verifies multiple signatures efficiently
func (v *Verifier) BatchVerify(messages [][]byte, sigs []*Signature) []bool {
	if len(messages) != len(sigs) {
		return nil
	}

	results := make([]bool, len(messages))
	for i := range messages {
		results[i] = v.VerifySignature(messages[i], sigs[i])
	}

	return results
}

// GetChainID returns the chain ID
func (v *Verifier) GetChainID() string {
	return v.chainID
}