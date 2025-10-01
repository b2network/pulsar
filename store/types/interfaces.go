package types

import (
	"fmt"
)

// StoreType defines the type of store
type StoreType int

const (
	StoreTypeMulti StoreType = iota
	StoreTypeDB
	StoreTypeIAVL
	StoreTypeTransient
	StoreTypeMemory
	StoreTypePersistent
)

func (st StoreType) String() string {
	switch st {
	case StoreTypeMulti:
		return "StoreTypeMulti"
	case StoreTypeDB:
		return "StoreTypeDB"
	case StoreTypeIAVL:
		return "StoreTypeIAVL"
	case StoreTypeTransient:
		return "StoreTypeTransient"
	case StoreTypeMemory:
		return "StoreTypeMemory"
	case StoreTypePersistent:
		return "StoreTypePersistent"
	default:
		return "StoreTypeUnknown"
	}
}

// CommitID defines the commitment information of a store
type CommitID struct {
	Version int64  `json:"version"`
	Hash    []byte `json:"hash"`
}

func (cid CommitID) IsZero() bool {
	return cid.Version == 0 && len(cid.Hash) == 0
}

func (cid CommitID) String() string {
	return fmt.Sprintf("CommitID{%v:%X}", cid.Version, cid.Hash)
}

// CommitInfo defines commit information used by the multi-store when committing a version
type CommitInfo struct {
	Version    int64       `json:"version"`
	StoreInfos []StoreInfo `json:"store_infos"`
}

// StoreInfo defines store-specific commit information
type StoreInfo struct {
	Name     string   `json:"name"`
	CommitId CommitID `json:"commit_id"`
}

// ChangeSet defines a set of changes to be applied to a store
type ChangeSet struct {
	Pairs []KVPair `json:"pairs"`
}

// KVPair defines a key-value pair with operation type
type KVPair struct {
	Key    []byte `json:"key"`
	Value  []byte `json:"value"`
	Delete bool   `json:"delete"`
}

// Iterator defines an interface for iterating over key-value pairs
type Iterator interface {
	Domain() (start, end []byte)
	Valid() bool
	Next()
	Key() []byte
	Value() []byte
	Close() error
}

// Proof defines a proof for a key-value pair
type Proof struct {
	Ops []ProofOp `json:"ops"`
}

// ProofOp defines a single proof operation
type ProofOp struct {
	Type string `json:"type"`
	Key  []byte `json:"key"`
	Data []byte `json:"data"`
}

// Store defines the basic store interface
type Store interface {
	GetStoreType() StoreType
	CacheWrapper
}

// CacheWrapper defines an interface for cache wrapping
type CacheWrapper interface {
	CacheWrap() CacheWrap
}

// CacheWrap defines an interface for cache operations
type CacheWrap interface {
	Write()
	CacheWrapper
}

// Committer defines an interface for committing changes
type Committer interface {
	Commit() CommitID
	LastCommitID() CommitID
	SetPruning(pruning PruningOptions)
	GetPruning() PruningOptions
}

// CommitStore defines a store that can be committed
type CommitStore interface {
	Committer
	Store
}

// KVStore defines a key-value store interface
type KVStore interface {
	Store

	Get(key []byte) []byte
	Has(key []byte) bool
	Set(key, value []byte)
	Delete(key []byte)

	Iterator(start, end []byte) Iterator
	ReverseIterator(start, end []byte) Iterator
}

// CommitKVStore defines a committable key-value store
type CommitKVStore interface {
	Committer
	KVStore
}

// MultiStore defines an interface for managing multiple stores
type MultiStore interface {
	Store

	CacheMultiStore() CacheMultiStore
	GetStore(StoreKey) Store
	GetKVStore(StoreKey) KVStore

	TracingEnabled() bool
	SetTracer(w TraceWriter) MultiStore
}

// CommitMultiStore defines a committable multi-store
type CommitMultiStore interface {
	Committer
	MultiStore

	MountStoreWithDB(key StoreKey, typ StoreType, db DB)
	LoadLatestVersion() error
	LoadVersion(ver int64) error

	GetCommitKVStore(key StoreKey) CommitKVStore
	GetCommitStore(key StoreKey) CommitStore
}

// CacheMultiStore defines a cached multi-store
type CacheMultiStore interface {
	MultiStore
	Write()
}

// RootStore defines the root store interface for Pulsar
type RootStore interface {
	CommitMultiStore

	GetStateStorage() StateStorage
	GetStateCommitment() StateCommitment

	Query(storeKey StoreKey, version int64, key []byte, prove bool) (QueryResult, error)
	SetInitialVersion(version int64)
}

// StateStorage defines the state storage interface
type StateStorage interface {
	Get(storeKey StoreKey, key []byte, version int64) ([]byte, error)
	Set(storeKey StoreKey, key, value []byte)
	Delete(storeKey StoreKey, key []byte)
	Has(storeKey StoreKey, key []byte, version int64) (bool, error)

	Iterator(storeKey StoreKey, start, end []byte, version int64) (Iterator, error)
	ReverseIterator(storeKey StoreKey, start, end []byte, version int64) (Iterator, error)

	ApplyChangeset(version int64, cs *ChangeSet) error
	GetLatestVersion() (int64, error)
}

// StateCommitment defines the state commitment interface
type StateCommitment interface {
	WriteChangeset(cs *ChangeSet) error
	Commit(version int64) ([]byte, error)
	GetCommitInfo(version int64) (*CommitInfo, error)

	GetProof(storeKey StoreKey, version int64, key []byte) (*Proof, error)
	GetLatestVersion() (int64, error)
}

// QueryResult defines the result of a query operation
type QueryResult struct {
	Key    []byte
	Value  []byte
	Height int64
	Proof  *Proof
}

// TraceWriter defines an interface for writing store traces
type TraceWriter interface {
	Write([]byte) (int, error)
}

// DB defines a generic database interface
type DB interface {
	Get([]byte) ([]byte, error)
	Has([]byte) (bool, error)
	Set([]byte, []byte) error
	SetSync([]byte, []byte) error
	Delete([]byte) error
	DeleteSync([]byte) error
	Iterator(start, end []byte) (Iterator, error)
	ReverseIterator(start, end []byte) (Iterator, error)
	Close() error
	NewBatch() Batch
}

// Batch defines a database batch interface
type Batch interface {
	Set(key, value []byte) error
	Delete(key []byte) error
	Write() error
	WriteSync() error
	Close() error
}

// KVStorePrefixIterator creates an iterator that iterates over all keys with a given prefix
func KVStorePrefixIterator(store KVStore, prefix []byte) Iterator {
	return store.Iterator(prefix, PrefixEndBytes(prefix))
}

// PrefixEndBytes returns the end bytes for a prefix
func PrefixEndBytes(prefix []byte) []byte {
	if len(prefix) == 0 {
		return nil
	}

	end := make([]byte, len(prefix))
	copy(end, prefix)

	for i := len(end) - 1; i >= 0; i-- {
		if end[i] < 0xff {
			end[i]++
			return end[:i+1]
		}
	}

	return nil
}
