package keys

import (
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/crypto/sha3"
)

// IsValidAddress checks if a string is a valid Ethereum address
func IsValidAddress(addr string) bool {
	_, err := HexToAddress(addr)
	return err == nil
}

// IsZeroAddress checks if an address is the zero address
func IsZeroAddress(addr Address) bool {
	return addr == Address{}
}

// ChecksumAddress returns an EIP-55 compliant checksummed address
func ChecksumAddress(addr Address) string {
	hex := hex.EncodeToString(addr[:])

	// Hash the lowercase address
	hash := sha3.NewLegacyKeccak256()
	hash.Write([]byte(hex))
	digest := hash.Sum(nil)

	// Apply checksum
	result := make([]byte, len(hex))
	for i := 0; i < len(hex); i++ {
		hashByte := digest[i/2]
		if i%2 == 0 {
			hashByte = hashByte >> 4
		} else {
			hashByte &= 0xf
		}

		if hex[i] >= 'a' && hex[i] <= 'f' && hashByte >= 8 {
			result[i] = hex[i] - 32 // Convert to uppercase
		} else {
			result[i] = hex[i]
		}
	}

	return "0x" + string(result)
}

// ValidateChecksum validates an EIP-55 checksummed address
func ValidateChecksum(addr string) bool {
	if !strings.HasPrefix(addr, "0x") {
		return false
	}

	// Parse the address
	address, err := HexToAddress(addr)
	if err != nil {
		return false
	}

	// Get the expected checksum
	expected := ChecksumAddress(address)

	return addr == expected
}

// AddressFromPrivKey derives an address from a private key
func AddressFromPrivKey(privKey PrivKey) Address {
	return privKey.PubKey().Address()
}

// AddressFromPubKey derives an address from a public key
func AddressFromPubKey(pubKey PubKey) Address {
	return pubKey.Address()
}

// AddressFromPubKeyBytes derives an address from public key bytes
func AddressFromPubKeyBytes(pubKeyBytes []byte) (Address, error) {
	pubKey, err := NewSecp256k1PubKey(pubKeyBytes)
	if err != nil {
		return Address{}, fmt.Errorf("invalid public key: %w", err)
	}
	return pubKey.Address(), nil
}

// MustHexToAddress converts a hex string to an Address, panics on error
func MustHexToAddress(s string) Address {
	addr, err := HexToAddress(s)
	if err != nil {
		panic(fmt.Sprintf("invalid address: %s", err))
	}
	return addr
}

// FormatAddress formats an address with optional checksum
func FormatAddress(addr Address, checksum bool) string {
	if checksum {
		return ChecksumAddress(addr)
	}
	return addr.String()
}

// HashData hashes data using Keccak256
func HashData(data []byte) []byte {
	hash := sha3.NewLegacyKeccak256()
	hash.Write(data)
	return hash.Sum(nil)
}

// CommonAddress represents commonly used addresses
var (
	// ZeroAddress is the all-zero address
	ZeroAddress = Address{}

	// DeadAddress is the commonly used burn address
	DeadAddress = MustHexToAddress("0x000000000000000000000000000000000000dEaD")
)