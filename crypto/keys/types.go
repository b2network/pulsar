package keys

import (
	"crypto/subtle"
	"encoding/hex"
	"fmt"
)

// Address represents a 20-byte Ethereum-compatible address
type Address [20]byte

// String returns the hex-encoded address with 0x prefix
func (a Address) String() string {
	return "0x" + hex.EncodeToString(a[:])
}

// Bytes returns the raw address bytes
func (a Address) Bytes() []byte {
	return a[:]
}

// Empty returns true if the address is all zeros
func (a Address) Empty() bool {
	return a == Address{}
}

// Equals checks if two addresses are the same
func (a Address) Equals(other Address) bool {
	return subtle.ConstantTimeCompare(a[:], other[:]) == 1
}

// MarshalJSON marshals the address to JSON
func (a Address) MarshalJSON() ([]byte, error) {
	return []byte(`"` + a.String() + `"`), nil
}

// UnmarshalJSON unmarshals the address from JSON
func (a *Address) UnmarshalJSON(data []byte) error {
	if len(data) < 2 || data[0] != '"' || data[len(data)-1] != '"' {
		return fmt.Errorf("invalid address format")
	}

	addrStr := string(data[1 : len(data)-1])
	addr, err := HexToAddress(addrStr)
	if err != nil {
		return err
	}

	*a = addr
	return nil
}

// HexToAddress converts a hex string to an Address
func HexToAddress(s string) (Address, error) {
	if len(s) >= 2 && s[:2] == "0x" {
		s = s[2:]
	}

	if len(s) != 40 {
		return Address{}, fmt.Errorf("invalid address length: expected 40 hex chars, got %d", len(s))
	}

	bytes, err := hex.DecodeString(s)
	if err != nil {
		return Address{}, fmt.Errorf("invalid hex address: %w", err)
	}

	var addr Address
	copy(addr[:], bytes)
	return addr, nil
}

// BytesToAddress converts bytes to an Address
func BytesToAddress(b []byte) Address {
	var addr Address
	if len(b) > 20 {
		copy(addr[:], b[len(b)-20:])
	} else {
		copy(addr[20-len(b):], b)
	}
	return addr
}

// PrivKey defines the interface for private keys
type PrivKey interface {
	// Bytes returns the raw private key bytes
	Bytes() []byte

	// Sign signs the given message
	Sign(msg []byte) ([]byte, error)

	// PubKey returns the corresponding public key
	PubKey() PubKey

	// Type returns the key type
	Type() string

	// Equals checks if two private keys are the same
	Equals(PrivKey) bool
}

// PubKey defines the interface for public keys
type PubKey interface {
	// Address returns the address derived from the public key
	Address() Address

	// Bytes returns the raw public key bytes
	Bytes() []byte

	// VerifySignature verifies a signature on the given message
	VerifySignature(msg []byte, sig []byte) bool

	// Type returns the key type
	Type() string

	// Equals checks if two public keys are the same
	Equals(PubKey) bool
}

// KeyType represents the type of cryptographic key
type KeyType string

const (
	// Secp256k1KeyType represents secp256k1 keys (Ethereum compatible)
	Secp256k1KeyType KeyType = "secp256k1"
)

// KeyInfo contains information about a key
type KeyInfo struct {
	Name      string   `json:"name"`
	Type      KeyType  `json:"type"`
	Address   Address  `json:"address"`
	PubKey    PubKey   `json:"-"` // Don't expose in JSON
	Mnemonic  string   `json:"-"` // Don't expose in JSON
}

// GetAddress returns the address for this key
func (ki *KeyInfo) GetAddress() Address {
	return ki.Address
}

// GetName returns the name for this key
func (ki *KeyInfo) GetName() string {
	return ki.Name
}

// GetType returns the type for this key
func (ki *KeyInfo) GetType() KeyType {
	return ki.Type
}

// GetPubKey returns the public key
func (ki *KeyInfo) GetPubKey() PubKey {
	return ki.PubKey
}