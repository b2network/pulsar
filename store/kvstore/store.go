package kvstore

import (
	"github.com/b2network/pulsar/store/types"
)

// Store implements a basic KVStore
type Store struct {
	parent types.KVStore
}

// NewStore creates a new KVStore wrapper
func NewStore(parent types.KVStore) *Store {
	return &Store{parent: parent}
}

func (s *Store) GetStoreType() types.StoreType {
	return s.parent.GetStoreType()
}

func (s *Store) CacheWrap() types.CacheWrap {
	return s.parent.CacheWrap()
}

func (s *Store) Get(key []byte) []byte {
	return s.parent.Get(key)
}

func (s *Store) Has(key []byte) bool {
	return s.parent.Has(key)
}

func (s *Store) Set(key, value []byte) {
	s.parent.Set(key, value)
}

func (s *Store) Delete(key []byte) {
	s.parent.Delete(key)
}

func (s *Store) Iterator(start, end []byte) types.Iterator {
	return s.parent.Iterator(start, end)
}

func (s *Store) ReverseIterator(start, end []byte) types.Iterator {
	return s.parent.ReverseIterator(start, end)
}
