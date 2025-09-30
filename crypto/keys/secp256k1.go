package keys

import (
	"crypto/subtle"
	"fmt"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"golang.org/x/crypto/sha3"
)

const (
	// Secp256k1PrivKeySize is the size of the Secp256k1 private key
	Secp256k1PrivKeySize = 32

	// Secp256k1PubKeySize is the size of the Secp256k1 public key (compressed)
	Secp256k1PubKeyCompressedSize = 33

	// Secp256k1PubKeySize is the size of the Secp256k1 public key (uncompressed)
	Secp256k1PubKeyUncompressedSize = 65

	// Secp256k1SignatureSize is the size of the Secp256k1 signature
	Secp256k1SignatureSize = 65 // 32 + 32 + 1 (r + s + v)
)

// Secp256k1PrivKey implements PrivKey interface for secp256k1
type Secp256k1PrivKey struct {
	key *secp256k1.PrivateKey
}

// NewSecp256k1PrivKey creates a new Secp256k1 private key from bytes
func NewSecp256k1PrivKey(bz []byte) (*Secp256k1PrivKey, error) {
	if len(bz) != Secp256k1PrivKeySize {
		return nil, fmt.Errorf("invalid private key size: expected %d, got %d",
			Secp256k1PrivKeySize, len(bz))
	}

	privKey := secp256k1.PrivKeyFromBytes(bz)
	return &Secp256k1PrivKey{key: privKey}, nil
}

// GenerateSecp256k1PrivKey generates a new random Secp256k1 private key
func GenerateSecp256k1PrivKey() (*Secp256k1PrivKey, error) {
	privKey, err := secp256k1.GeneratePrivateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	return &Secp256k1PrivKey{key: privKey}, nil
}

// Bytes returns the raw private key bytes
func (k *Secp256k1PrivKey) Bytes() []byte {
	return k.key.Serialize()
}

// Sign signs the given message
func (k *Secp256k1PrivKey) Sign(msg []byte) ([]byte, error) {
	// Hash the message using Keccak256 (Ethereum compatible)
	hash := sha3.NewLegacyKeccak256()
	hash.Write(msg)
	digest := hash.Sum(nil)

	// Sign the hash
	sig := Sign(k.key, digest)
	return sig, nil
}

// PubKey returns the corresponding public key
func (k *Secp256k1PrivKey) PubKey() PubKey {
	return &Secp256k1PubKey{
		key: k.key.PubKey(),
	}
}

// Type returns the key type
func (k *Secp256k1PrivKey) Type() string {
	return string(Secp256k1KeyType)
}

// Equals checks if two private keys are the same
func (k *Secp256k1PrivKey) Equals(other PrivKey) bool {
	if other.Type() != k.Type() {
		return false
	}

	otherSecp, ok := other.(*Secp256k1PrivKey)
	if !ok {
		return false
	}

	return subtle.ConstantTimeCompare(k.Bytes(), otherSecp.Bytes()) == 1
}

// Secp256k1PubKey implements PubKey interface for secp256k1
type Secp256k1PubKey struct {
	key *secp256k1.PublicKey
}

// NewSecp256k1PubKey creates a new Secp256k1 public key from bytes
func NewSecp256k1PubKey(bz []byte) (*Secp256k1PubKey, error) {
	if len(bz) != Secp256k1PubKeyCompressedSize && len(bz) != Secp256k1PubKeyUncompressedSize {
		return nil, fmt.Errorf("invalid public key size: expected %d or %d, got %d",
			Secp256k1PubKeyCompressedSize, Secp256k1PubKeyUncompressedSize, len(bz))
	}

	pubKey, err := secp256k1.ParsePubKey(bz)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	return &Secp256k1PubKey{key: pubKey}, nil
}

// Address returns the Ethereum-compatible address
func (k *Secp256k1PubKey) Address() Address {
	// Get uncompressed public key bytes (65 bytes: 0x04 + 32 bytes X + 32 bytes Y)
	uncompressed := k.key.SerializeUncompressed()

	// Remove the 0x04 prefix
	if len(uncompressed) > 1 && uncompressed[0] == 0x04 {
		uncompressed = uncompressed[1:]
	}

	// Hash the public key using Keccak256
	hash := sha3.NewLegacyKeccak256()
	hash.Write(uncompressed)
	digest := hash.Sum(nil)

	// Take the last 20 bytes as the address
	return BytesToAddress(digest[12:])
}

// Bytes returns the compressed public key bytes
func (k *Secp256k1PubKey) Bytes() []byte {
	return k.key.SerializeCompressed()
}

// BytesUncompressed returns the uncompressed public key bytes
func (k *Secp256k1PubKey) BytesUncompressed() []byte {
	return k.key.SerializeUncompressed()
}

// VerifySignature verifies a signature on the given message
func (k *Secp256k1PubKey) VerifySignature(msg []byte, sig []byte) bool {
	if len(sig) != Secp256k1SignatureSize {
		return false
	}

	// Hash the message using Keccak256
	hash := sha3.NewLegacyKeccak256()
	hash.Write(msg)
	digest := hash.Sum(nil)

	// Verify the signature
	return Verify(k.key, digest, sig)
}

// Type returns the key type
func (k *Secp256k1PubKey) Type() string {
	return string(Secp256k1KeyType)
}

// Equals checks if two public keys are the same
func (k *Secp256k1PubKey) Equals(other PubKey) bool {
	if other.Type() != k.Type() {
		return false
	}

	otherSecp, ok := other.(*Secp256k1PubKey)
	if !ok {
		return false
	}

	return k.key.IsEqual(otherSecp.key)
}

// Sign signs a message with the private key (Ethereum compatible)
func Sign(privKey *secp256k1.PrivateKey, msg []byte) []byte {
	// Create signature
	sig := signCompact(privKey, msg, true)
	return sig
}

// Verify verifies a signature (simplified implementation)
func Verify(pubKey *secp256k1.PublicKey, msg []byte, sigBytes []byte) bool {
	if len(sigBytes) != Secp256k1SignatureSize {
		return false
	}

	// For now, we'll implement a simplified verification
	// In production, this should be properly implemented with recovery
	// The signature format is r(32) + s(32) + v(1)
	r := sigBytes[:32]
	s := sigBytes[32:64]

	// Create signature components using the proper API
	var rScalar secp256k1.ModNScalar
	overflow := rScalar.SetByteSlice(r)
	if overflow {
		return false
	}

	var sScalar secp256k1.ModNScalar
	overflow = sScalar.SetByteSlice(s)
	if overflow {
		return false
	}

	// Create signature using the correct constructor
	signature := ecdsa.NewSignature(&rScalar, &sScalar)

	// Verify
	return signature.Verify(msg, pubKey)
}

// signCompact creates a compact signature with recovery ID
func signCompact(privKey *secp256k1.PrivateKey, msg []byte, compressed bool) []byte {
	// Sign the message using ecdsa compact signing
	sig := ecdsa.SignCompact(privKey, msg, compressed)
	return sig
}

// RecoverPubKey recovers the public key from signature
func RecoverPubKey(msg []byte, sig []byte) (*Secp256k1PubKey, error) {
	if len(sig) != Secp256k1SignatureSize {
		return nil, fmt.Errorf("invalid signature size")
	}

	// Recover the public key using ecdsa.RecoverCompact
	pubKey, _, err := ecdsa.RecoverCompact(sig, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to recover public key: %w", err)
	}

	return &Secp256k1PubKey{key: pubKey}, nil
}