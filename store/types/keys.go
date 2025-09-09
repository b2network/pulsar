package types

import (
	"fmt"
)

// StoreKey defines an interface for store keys
type StoreKey interface {
	Name() string
	String() string
}

// KVStoreKey defines a key for accessing KVStores
type KVStoreKey struct {
	name string
}

// NewKVStoreKey creates a new KVStoreKey
func NewKVStoreKey(name string) *KVStoreKey {
	if name == "" {
		panic("store key name cannot be empty")
	}
	return &KVStoreKey{name: name}
}

func (key *KVStoreKey) Name() string {
	return key.name
}

func (key *KVStoreKey) String() string {
	return fmt.Sprintf("KVStoreKey{%s}", key.name)
}

// TransientStoreKey defines a key for accessing transient stores
type TransientStoreKey struct {
	name string
}

// NewTransientStoreKey creates a new TransientStoreKey
func NewTransientStoreKey(name string) *TransientStoreKey {
	if name == "" {
		panic("store key name cannot be empty")
	}
	return &TransientStoreKey{name: name}
}

func (key *TransientStoreKey) Name() string {
	return key.name
}

func (key *TransientStoreKey) String() string {
	return fmt.Sprintf("TransientStoreKey{%s}", key.name)
}

// MemoryStoreKey defines a key for accessing memory stores
type MemoryStoreKey struct {
	name string
}

// NewMemoryStoreKey creates a new MemoryStoreKey
func NewMemoryStoreKey(name string) *MemoryStoreKey {
	if name == "" {
		panic("store key name cannot be empty")
	}
	return &MemoryStoreKey{name: name}
}

func (key *MemoryStoreKey) Name() string {
	return key.name
}

func (key *MemoryStoreKey) String() string {
	return fmt.Sprintf("MemoryStoreKey{%s}", key.name)
}

// StoreKeys is a slice of StoreKey
type StoreKeys []StoreKey

// NewKVStoreKeys creates multiple KVStoreKeys
func NewKVStoreKeys(names ...string) []StoreKey {
	keys := make([]StoreKey, len(names))
	for i, name := range names {
		keys[i] = NewKVStoreKey(name)
	}
	return keys
}

// NewTransientStoreKeys creates multiple TransientStoreKeys
func NewTransientStoreKeys(names ...string) []StoreKey {
	keys := make([]StoreKey, len(names))
	for i, name := range names {
		keys[i] = NewTransientStoreKey(name)
	}
	return keys
}

// NewMemoryStoreKeys creates multiple MemoryStoreKeys
func NewMemoryStoreKeys(names ...string) []StoreKey {
	keys := make([]StoreKey, len(names))
	for i, name := range names {
		keys[i] = NewMemoryStoreKey(name)
	}
	return keys
}