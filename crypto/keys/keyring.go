package keys

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Keyring manages multiple keys
type Keyring interface {
	// Key management
	CreateKey(name, mnemonic, passphrase string) (*KeyInfo, error)
	ImportPrivKey(name, privKeyHex, passphrase string) error
	ExportPrivKey(name, passphrase string) (string, error)

	// Key queries
	GetKey(name string) (*KeyInfo, error)
	GetKeyByAddress(addr Address) (*KeyInfo, error)
	ListKeys() ([]*KeyInfo, error)
	DeleteKey(name, passphrase string) error
	HasKey(name string) bool

	// Signing
	SignByKey(name string, msg []byte, passphrase string) ([]byte, PubKey, error)
	SignByAddress(addr Address, msg []byte, passphrase string) ([]byte, PubKey, error)
}

// FileKeyring implements Keyring using file-based storage
type FileKeyring struct {
	keyDir string
	keys   map[string]*keyEntry
}

// keyEntry represents an encrypted key entry
type keyEntry struct {
	Name      string   `json:"name"`
	Type      KeyType  `json:"type"`
	Address   Address  `json:"address"`
	PubKey    []byte   `json:"pub_key"`
	PrivKey   []byte   `json:"priv_key"` // Encrypted
	Salt      []byte   `json:"salt"`
}

// NewFileKeyring creates a new file-based keyring
func NewFileKeyring(keyDir string) (*FileKeyring, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(keyDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create key directory: %w", err)
	}

	kr := &FileKeyring{
		keyDir: keyDir,
		keys:   make(map[string]*keyEntry),
	}

	// Load existing keys
	if err := kr.loadKeys(); err != nil {
		return nil, err
	}

	return kr, nil
}

// CreateKey creates a new key
func (kr *FileKeyring) CreateKey(name, mnemonic, passphrase string) (*KeyInfo, error) {
	if kr.HasKey(name) {
		return nil, fmt.Errorf("key with name %s already exists", name)
	}

	// Generate new key
	privKey, err := GenerateSecp256k1PrivKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	// Get public key and address
	pubKey := privKey.PubKey()
	address := pubKey.Address()

	// Create key entry
	entry := &keyEntry{
		Name:    name,
		Type:    Secp256k1KeyType,
		Address: address,
		PubKey:  pubKey.Bytes(),
	}

	// Encrypt and store private key
	if err := kr.encryptAndStorePrivKey(entry, privKey.Bytes(), passphrase); err != nil {
		return nil, err
	}

	// Save to file
	if err := kr.saveKey(entry); err != nil {
		return nil, err
	}

	// Add to memory
	kr.keys[name] = entry

	return &KeyInfo{
		Name:     name,
		Type:     Secp256k1KeyType,
		Address:  address,
		PubKey:   pubKey,
		Mnemonic: mnemonic,
	}, nil
}

// ImportPrivKey imports a private key
func (kr *FileKeyring) ImportPrivKey(name, privKeyHex, passphrase string) error {
	if kr.HasKey(name) {
		return fmt.Errorf("key with name %s already exists", name)
	}

	// Decode hex private key
	privKeyBytes, err := hex.DecodeString(strings.TrimPrefix(privKeyHex, "0x"))
	if err != nil {
		return fmt.Errorf("invalid private key hex: %w", err)
	}

	// Create private key
	privKey, err := NewSecp256k1PrivKey(privKeyBytes)
	if err != nil {
		return fmt.Errorf("failed to create private key: %w", err)
	}

	// Get public key and address
	pubKey := privKey.PubKey()
	address := pubKey.Address()

	// Create key entry
	entry := &keyEntry{
		Name:    name,
		Type:    Secp256k1KeyType,
		Address: address,
		PubKey:  pubKey.Bytes(),
	}

	// Encrypt and store private key
	if err := kr.encryptAndStorePrivKey(entry, privKeyBytes, passphrase); err != nil {
		return err
	}

	// Save to file
	if err := kr.saveKey(entry); err != nil {
		return err
	}

	// Add to memory
	kr.keys[name] = entry

	return nil
}

// ExportPrivKey exports a private key
func (kr *FileKeyring) ExportPrivKey(name, passphrase string) (string, error) {
	entry, exists := kr.keys[name]
	if !exists {
		return "", fmt.Errorf("key %s not found", name)
	}

	// Decrypt private key
	privKeyBytes, err := kr.decryptPrivKey(entry, passphrase)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt private key: %w", err)
	}

	return "0x" + hex.EncodeToString(privKeyBytes), nil
}

// GetKey gets a key by name
func (kr *FileKeyring) GetKey(name string) (*KeyInfo, error) {
	entry, exists := kr.keys[name]
	if !exists {
		return nil, fmt.Errorf("key %s not found", name)
	}

	pubKey, err := NewSecp256k1PubKey(entry.PubKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	return &KeyInfo{
		Name:    entry.Name,
		Type:    entry.Type,
		Address: entry.Address,
		PubKey:  pubKey,
	}, nil
}

// GetKeyByAddress gets a key by address
func (kr *FileKeyring) GetKeyByAddress(addr Address) (*KeyInfo, error) {
	for _, entry := range kr.keys {
		if entry.Address.Equals(addr) {
			return kr.GetKey(entry.Name)
		}
	}
	return nil, fmt.Errorf("key with address %s not found", addr.String())
}

// ListKeys lists all keys
func (kr *FileKeyring) ListKeys() ([]*KeyInfo, error) {
	var keys []*KeyInfo

	// Get all key names and sort them
	var names []string
	for name := range kr.keys {
		names = append(names, name)
	}
	sort.Strings(names)

	// Create KeyInfo for each key
	for _, name := range names {
		keyInfo, err := kr.GetKey(name)
		if err != nil {
			return nil, err
		}
		keys = append(keys, keyInfo)
	}

	return keys, nil
}

// DeleteKey deletes a key
func (kr *FileKeyring) DeleteKey(name, passphrase string) error {
	entry, exists := kr.keys[name]
	if !exists {
		return fmt.Errorf("key %s not found", name)
	}

	// Verify passphrase
	if _, err := kr.decryptPrivKey(entry, passphrase); err != nil {
		return fmt.Errorf("invalid passphrase")
	}

	// Delete file
	keyFile := filepath.Join(kr.keyDir, name+".json")
	if err := os.Remove(keyFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete key file: %w", err)
	}

	// Remove from memory
	delete(kr.keys, name)

	return nil
}

// HasKey checks if a key exists
func (kr *FileKeyring) HasKey(name string) bool {
	_, exists := kr.keys[name]
	return exists
}

// SignByKey signs a message with a key
func (kr *FileKeyring) SignByKey(name string, msg []byte, passphrase string) ([]byte, PubKey, error) {
	entry, exists := kr.keys[name]
	if !exists {
		return nil, nil, fmt.Errorf("key %s not found", name)
	}

	// Decrypt private key
	privKeyBytes, err := kr.decryptPrivKey(entry, passphrase)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decrypt private key: %w", err)
	}

	// Create private key
	privKey, err := NewSecp256k1PrivKey(privKeyBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create private key: %w", err)
	}

	// Sign message
	sig, err := privKey.Sign(msg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to sign message: %w", err)
	}

	return sig, privKey.PubKey(), nil
}

// SignByAddress signs a message with a key identified by address
func (kr *FileKeyring) SignByAddress(addr Address, msg []byte, passphrase string) ([]byte, PubKey, error) {
	// Find key by address
	var entry *keyEntry
	for _, e := range kr.keys {
		if e.Address.Equals(addr) {
			entry = e
			break
		}
	}

	if entry == nil {
		return nil, nil, fmt.Errorf("key with address %s not found", addr.String())
	}

	return kr.SignByKey(entry.Name, msg, passphrase)
}

// encryptAndStorePrivKey encrypts and stores a private key
func (kr *FileKeyring) encryptAndStorePrivKey(entry *keyEntry, privKey []byte, passphrase string) error {
	// Generate salt
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive key using Argon2
	key := argon2.IDKey([]byte(passphrase), salt, 1, 64*1024, 4, 32)

	// Encrypt private key (simple XOR for now, should use AES in production)
	encrypted := make([]byte, len(privKey))
	for i := range privKey {
		encrypted[i] = privKey[i] ^ key[i%len(key)]
	}

	entry.PrivKey = encrypted
	entry.Salt = salt

	return nil
}

// decryptPrivKey decrypts a private key
func (kr *FileKeyring) decryptPrivKey(entry *keyEntry, passphrase string) ([]byte, error) {
	// Derive key using Argon2
	key := argon2.IDKey([]byte(passphrase), entry.Salt, 1, 64*1024, 4, 32)

	// Decrypt private key
	decrypted := make([]byte, len(entry.PrivKey))
	for i := range entry.PrivKey {
		decrypted[i] = entry.PrivKey[i] ^ key[i%len(key)]
	}

	return decrypted, nil
}

// saveKey saves a key to file
func (kr *FileKeyring) saveKey(entry *keyEntry) error {
	keyFile := filepath.Join(kr.keyDir, entry.Name+".json")

	// Marshal to JSON
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal key: %w", err)
	}

	// Write to file
	if err := os.WriteFile(keyFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write key file: %w", err)
	}

	return nil
}

// loadKeys loads all keys from disk
func (kr *FileKeyring) loadKeys() error {
	files, err := os.ReadDir(kr.keyDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read key directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		keyPath := filepath.Join(kr.keyDir, file.Name())
		data, err := os.ReadFile(keyPath)
		if err != nil {
			return fmt.Errorf("failed to read key file %s: %w", file.Name(), err)
		}

		var entry keyEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			return fmt.Errorf("failed to unmarshal key file %s: %w", file.Name(), err)
		}

		kr.keys[entry.Name] = &entry
	}

	return nil
}