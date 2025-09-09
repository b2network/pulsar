package rootstore

import (
	"github.com/b2network/pulsar/store/types"
)

// Basic store implementations - these are placeholder implementations
// In a full implementation, these would be more sophisticated

// iavlStore implements a CommitKVStore backed by IAVL
type iavlStore struct {
	key    types.StoreKey
	db     types.DB
	root   *Store
	
	// IAVL-specific fields would go here
	// tree *iavl.MutableTree
}

func newIAVLStore(key types.StoreKey, db types.DB, root *Store) types.CommitKVStore {
	return &iavlStore{
		key:  key,
		db:   db,
		root: root,
	}
}

func (s *iavlStore) GetStoreType() types.StoreType {
	return types.StoreTypeIAVL
}

func (s *iavlStore) CacheWrap() types.CacheWrap {
	// Return a cache wrapper for this store
	return newCacheKVStore(s)
}

func (s *iavlStore) Get(key []byte) []byte {
	// Get from the current version via state storage
	value, err := s.root.stateStorage.Get(s.key, key, s.root.lastCommitInfo.Version)
	if err != nil {
		return nil
	}
	return value
}

func (s *iavlStore) Has(key []byte) bool {
	has, err := s.root.stateStorage.Has(s.key, key, s.root.lastCommitInfo.Version)
	if err != nil {
		return false
	}
	return has
}

func (s *iavlStore) Set(key, value []byte) {
	s.root.stateStorage.Set(s.key, key, value)
}

func (s *iavlStore) Delete(key []byte) {
	s.root.stateStorage.Delete(s.key, key)
}

func (s *iavlStore) Iterator(start, end []byte) types.Iterator {
	iter, err := s.root.stateStorage.Iterator(s.key, start, end, s.root.lastCommitInfo.Version)
	if err != nil {
		return &emptyIterator{}
	}
	return iter
}

func (s *iavlStore) ReverseIterator(start, end []byte) types.Iterator {
	iter, err := s.root.stateStorage.ReverseIterator(s.key, start, end, s.root.lastCommitInfo.Version)
	if err != nil {
		return &emptyIterator{}
	}
	return iter
}

func (s *iavlStore) Commit() types.CommitID {
	// IAVL store commit logic would go here
	var version int64
	if s.root.lastCommitInfo != nil {
		version = s.root.lastCommitInfo.Version + 1
	} else {
		version = 1
	}
	return types.CommitID{
		Version: version,
		Hash:    []byte("mock-hash"),
	}
}

func (s *iavlStore) LastCommitID() types.CommitID {
	var version int64
	if s.root.lastCommitInfo != nil {
		version = s.root.lastCommitInfo.Version
	}
	return types.CommitID{
		Version: version,
		Hash:    []byte("mock-hash"),
	}
}

func (s *iavlStore) SetPruning(pruning types.PruningOptions) {
	// Set pruning for this specific store
}

func (s *iavlStore) GetPruning() types.PruningOptions {
	return s.root.pruning
}

// dbStore implements a simple CommitKVStore backed by a database
type dbStore struct {
	key  types.StoreKey
	db   types.DB
	root *Store
}

func newDBStore(key types.StoreKey, db types.DB, root *Store) types.CommitKVStore {
	return &dbStore{
		key:  key,
		db:   db,
		root: root,
	}
}

func (s *dbStore) GetStoreType() types.StoreType {
	return types.StoreTypeDB
}

func (s *dbStore) CacheWrap() types.CacheWrap {
	return newCacheKVStore(s)
}

func (s *dbStore) Get(key []byte) []byte {
	value, err := s.root.stateStorage.Get(s.key, key, s.root.lastCommitInfo.Version)
	if err != nil {
		return nil
	}
	return value
}

func (s *dbStore) Has(key []byte) bool {
	has, err := s.root.stateStorage.Has(s.key, key, s.root.lastCommitInfo.Version)
	if err != nil {
		return false
	}
	return has
}

func (s *dbStore) Set(key, value []byte) {
	s.root.stateStorage.Set(s.key, key, value)
}

func (s *dbStore) Delete(key []byte) {
	s.root.stateStorage.Delete(s.key, key)
}

func (s *dbStore) Iterator(start, end []byte) types.Iterator {
	iter, err := s.root.stateStorage.Iterator(s.key, start, end, s.root.lastCommitInfo.Version)
	if err != nil {
		return &emptyIterator{}
	}
	return iter
}

func (s *dbStore) ReverseIterator(start, end []byte) types.Iterator {
	iter, err := s.root.stateStorage.ReverseIterator(s.key, start, end, s.root.lastCommitInfo.Version)
	if err != nil {
		return &emptyIterator{}
	}
	return iter
}

func (s *dbStore) Commit() types.CommitID {
	var version int64
	if s.root.lastCommitInfo != nil {
		version = s.root.lastCommitInfo.Version + 1
	} else {
		version = 1
	}
	return types.CommitID{
		Version: version,
		Hash:    []byte("mock-hash"),
	}
}

func (s *dbStore) LastCommitID() types.CommitID {
	var version int64
	if s.root.lastCommitInfo != nil {
		version = s.root.lastCommitInfo.Version
	}
	return types.CommitID{
		Version: version,
		Hash:    []byte("mock-hash"),
	}
}

func (s *dbStore) SetPruning(pruning types.PruningOptions) {}

func (s *dbStore) GetPruning() types.PruningOptions {
	return s.root.pruning
}

// transientStore implements a transient store (cleared on each commit)
type transientStore struct {
	key  types.StoreKey
	data map[string][]byte
}

func newTransientStore(key types.StoreKey) types.CommitKVStore {
	return &transientStore{
		key:  key,
		data: make(map[string][]byte),
	}
}

func (s *transientStore) GetStoreType() types.StoreType {
	return types.StoreTypeTransient
}

func (s *transientStore) CacheWrap() types.CacheWrap {
	return newCacheKVStore(s)
}

func (s *transientStore) Get(key []byte) []byte {
	return s.data[string(key)]
}

func (s *transientStore) Has(key []byte) bool {
	_, exists := s.data[string(key)]
	return exists
}

func (s *transientStore) Set(key, value []byte) {
	s.data[string(key)] = value
}

func (s *transientStore) Delete(key []byte) {
	delete(s.data, string(key))
}

func (s *transientStore) Iterator(start, end []byte) types.Iterator {
	return newMapIterator(s.data, start, end, false)
}

func (s *transientStore) ReverseIterator(start, end []byte) types.Iterator {
	return newMapIterator(s.data, start, end, true)
}

func (s *transientStore) Commit() types.CommitID {
	// Clear transient data on commit
	s.data = make(map[string][]byte)
	return types.CommitID{}
}

func (s *transientStore) LastCommitID() types.CommitID {
	return types.CommitID{}
}

func (s *transientStore) SetPruning(pruning types.PruningOptions) {}

func (s *transientStore) GetPruning() types.PruningOptions {
	return types.NewPruningOptions(types.PruningNothing)
}

// memoryStore implements an in-memory store
type memoryStore struct {
	key  types.StoreKey
	data map[string][]byte
}

func newMemoryStore(key types.StoreKey) types.CommitKVStore {
	return &memoryStore{
		key:  key,
		data: make(map[string][]byte),
	}
}

func (s *memoryStore) GetStoreType() types.StoreType {
	return types.StoreTypeMemory
}

func (s *memoryStore) CacheWrap() types.CacheWrap {
	return newCacheKVStore(s)
}

func (s *memoryStore) Get(key []byte) []byte {
	return s.data[string(key)]
}

func (s *memoryStore) Has(key []byte) bool {
	_, exists := s.data[string(key)]
	return exists
}

func (s *memoryStore) Set(key, value []byte) {
	s.data[string(key)] = value
}

func (s *memoryStore) Delete(key []byte) {
	delete(s.data, string(key))
}

func (s *memoryStore) Iterator(start, end []byte) types.Iterator {
	return newMapIterator(s.data, start, end, false)
}

func (s *memoryStore) ReverseIterator(start, end []byte) types.Iterator {
	return newMapIterator(s.data, start, end, true)
}

func (s *memoryStore) Commit() types.CommitID {
	return types.CommitID{}
}

func (s *memoryStore) LastCommitID() types.CommitID {
	return types.CommitID{}
}

func (s *memoryStore) SetPruning(pruning types.PruningOptions) {}

func (s *memoryStore) GetPruning() types.PruningOptions {
	return types.NewPruningOptions(types.PruningNothing)
}