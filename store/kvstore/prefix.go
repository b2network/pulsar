package kvstore

import (
	"bytes"

	"github.com/b2network/pulsar/store/types"
)

// PrefixStore wraps a KVStore and adds a prefix to all keys
type PrefixStore struct {
	parent types.KVStore
	prefix []byte
}

// NewPrefixStore creates a new PrefixStore
func NewPrefixStore(parent types.KVStore, prefix []byte) *PrefixStore {
	return &PrefixStore{
		parent: parent,
		prefix: prefix,
	}
}

func (s *PrefixStore) GetStoreType() types.StoreType {
	return s.parent.GetStoreType()
}

func (s *PrefixStore) CacheWrap() types.CacheWrap {
	return NewPrefixStore(s.parent.CacheWrap().(types.KVStore), s.prefix)
}

func (s *PrefixStore) Write() {
	if cacheWrap, ok := s.parent.(types.CacheWrap); ok {
		cacheWrap.Write()
	}
}

func (s *PrefixStore) Get(key []byte) []byte {
	prefixedKey := s.prefixKey(key)
	return s.parent.Get(prefixedKey)
}

func (s *PrefixStore) Has(key []byte) bool {
	prefixedKey := s.prefixKey(key)
	return s.parent.Has(prefixedKey)
}

func (s *PrefixStore) Set(key, value []byte) {
	prefixedKey := s.prefixKey(key)
	s.parent.Set(prefixedKey, value)
}

func (s *PrefixStore) Delete(key []byte) {
	prefixedKey := s.prefixKey(key)
	s.parent.Delete(prefixedKey)
}

func (s *PrefixStore) Iterator(start, end []byte) types.Iterator {
	prefixedStart := s.prefixKey(start)
	prefixedEnd := s.prefixKey(end)
	
	parentIter := s.parent.Iterator(prefixedStart, prefixedEnd)
	return newPrefixIterator(parentIter, s.prefix, start, end)
}

func (s *PrefixStore) ReverseIterator(start, end []byte) types.Iterator {
	prefixedStart := s.prefixKey(start)
	prefixedEnd := s.prefixKey(end)
	
	parentIter := s.parent.ReverseIterator(prefixedStart, prefixedEnd)
	return newPrefixIterator(parentIter, s.prefix, start, end)
}

func (s *PrefixStore) prefixKey(key []byte) []byte {
	if key == nil {
		return nil
	}
	prefixedKey := make([]byte, len(s.prefix)+len(key))
	copy(prefixedKey, s.prefix)
	copy(prefixedKey[len(s.prefix):], key)
	return prefixedKey
}

// prefixIterator wraps an iterator and strips the prefix from keys
type prefixIterator struct {
	parent types.Iterator
	prefix []byte
	start  []byte
	end    []byte
}

func newPrefixIterator(parent types.Iterator, prefix, start, end []byte) types.Iterator {
	return &prefixIterator{
		parent: parent,
		prefix: prefix,
		start:  start,
		end:    end,
	}
}

func (iter *prefixIterator) Domain() (start, end []byte) {
	return iter.start, iter.end
}

func (iter *prefixIterator) Valid() bool {
	if !iter.parent.Valid() {
		return false
	}
	
	// Check if the key still has our prefix
	key := iter.parent.Key()
	return bytes.HasPrefix(key, iter.prefix)
}

func (iter *prefixIterator) Next() {
	iter.parent.Next()
}

func (iter *prefixIterator) Key() []byte {
	if !iter.Valid() {
		return nil
	}
	
	key := iter.parent.Key()
	if !bytes.HasPrefix(key, iter.prefix) {
		return nil
	}
	
	return key[len(iter.prefix):]
}

func (iter *prefixIterator) Value() []byte {
	if !iter.Valid() {
		return nil
	}
	return iter.parent.Value()
}

func (iter *prefixIterator) Close() error {
	return iter.parent.Close()
}