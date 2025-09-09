package rootstore

import (
	"bytes"
	"sort"

	"github.com/b2network/pulsar/store/types"
)

// cacheKVStore implements a cache wrapper around a KVStore
type cacheKVStore struct {
	parent   types.KVStore
	cache    map[string]cacheValue
	deleted  map[string]bool
	sortedKeys []string
	dirty    bool
}

type cacheValue struct {
	value []byte
	dirty bool
}

func newCacheKVStore(parent types.KVStore) *cacheKVStore {
	return &cacheKVStore{
		parent:  parent,
		cache:   make(map[string]cacheValue),
		deleted: make(map[string]bool),
		dirty:   false,
	}
}

func (s *cacheKVStore) GetStoreType() types.StoreType {
	return s.parent.GetStoreType()
}

func (s *cacheKVStore) CacheWrap() types.CacheWrap {
	return newCacheKVStore(s)
}

func (s *cacheKVStore) Get(key []byte) []byte {
	keyStr := string(key)
	
	// Check if deleted
	if s.deleted[keyStr] {
		return nil
	}
	
	// Check cache first
	if cached, exists := s.cache[keyStr]; exists {
		return cached.value
	}
	
	// Get from parent
	value := s.parent.Get(key)
	
	// Cache the result
	s.cache[keyStr] = cacheValue{value: value, dirty: false}
	
	return value
}

func (s *cacheKVStore) Has(key []byte) bool {
	return s.Get(key) != nil
}

func (s *cacheKVStore) Set(key, value []byte) {
	keyStr := string(key)
	s.cache[keyStr] = cacheValue{value: value, dirty: true}
	delete(s.deleted, keyStr)
	s.dirty = true
	s.invalidateSortedKeys()
}

func (s *cacheKVStore) Delete(key []byte) {
	keyStr := string(key)
	s.deleted[keyStr] = true
	delete(s.cache, keyStr)
	s.dirty = true
	s.invalidateSortedKeys()
}

func (s *cacheKVStore) Iterator(start, end []byte) types.Iterator {
	return s.iterator(start, end, false)
}

func (s *cacheKVStore) ReverseIterator(start, end []byte) types.Iterator {
	return s.iterator(start, end, true)
}

func (s *cacheKVStore) iterator(start, end []byte, reverse bool) types.Iterator {
	// This is a simplified implementation
	// A full implementation would merge parent iterator with cache changes
	
	// Get all keys from cache and parent
	allKeys := make(map[string][]byte)
	
	// Add parent keys
	parentIter := s.parent.Iterator(start, end)
	defer parentIter.Close()
	
	for ; parentIter.Valid(); parentIter.Next() {
		key := string(parentIter.Key())
		if !s.deleted[key] {
			if cached, exists := s.cache[key]; exists {
				allKeys[key] = cached.value
			} else {
				allKeys[key] = parentIter.Value()
			}
		}
	}
	
	// Add cache-only keys
	for keyStr, cached := range s.cache {
		key := []byte(keyStr)
		if !s.deleted[keyStr] && isInDomain(key, start, end) {
			allKeys[keyStr] = cached.value
		}
	}
	
	return newMapIterator(allKeys, start, end, reverse)
}

func (s *cacheKVStore) Write() {
	if !s.dirty {
		return
	}
	
	// Write cached changes to parent
	for keyStr, cached := range s.cache {
		if cached.dirty {
			s.parent.Set([]byte(keyStr), cached.value)
		}
	}
	
	// Apply deletions
	for keyStr := range s.deleted {
		s.parent.Delete([]byte(keyStr))
	}
	
	// Clear cache
	s.cache = make(map[string]cacheValue)
	s.deleted = make(map[string]bool)
	s.dirty = false
	s.invalidateSortedKeys()
}

func (s *cacheKVStore) invalidateSortedKeys() {
	s.sortedKeys = nil
}

// cacheMultiStore implements a cache wrapper around MultiStore
type cacheMultiStore struct {
	parent types.MultiStore
	stores map[types.StoreKey]types.CacheWrap
}

func newCacheMultiStore(parent types.MultiStore) *cacheMultiStore {
	return &cacheMultiStore{
		parent: parent,
		stores: make(map[types.StoreKey]types.CacheWrap),
	}
}

func (s *cacheMultiStore) GetStoreType() types.StoreType {
	return s.parent.GetStoreType()
}

func (s *cacheMultiStore) CacheWrap() types.CacheWrap {
	panic("cannot CacheWrap a CacheMultiStore")
}

func (s *cacheMultiStore) CacheMultiStore() types.CacheMultiStore {
	return newCacheMultiStore(s)
}

func (s *cacheMultiStore) GetStore(key types.StoreKey) types.Store {
	if cached, exists := s.stores[key]; exists {
		return cached.(types.Store)
	}
	
	store := s.parent.GetStore(key)
	if store == nil {
		return nil
	}
	
	cached := store.CacheWrap()
	s.stores[key] = cached
	return cached.(types.Store)
}

func (s *cacheMultiStore) GetKVStore(key types.StoreKey) types.KVStore {
	return s.GetStore(key).(types.KVStore)
}

func (s *cacheMultiStore) TracingEnabled() bool {
	return s.parent.TracingEnabled()
}

func (s *cacheMultiStore) SetTracer(w types.TraceWriter) types.MultiStore {
	s.parent.SetTracer(w)
	return s
}

func (s *cacheMultiStore) Write() {
	for _, store := range s.stores {
		store.Write()
	}
}

// mapIterator implements Iterator over a map
type mapIterator struct {
	data    map[string][]byte
	keys    []string
	start   []byte
	end     []byte
	reverse bool
	index   int
}

func newMapIterator(data map[string][]byte, start, end []byte, reverse bool) types.Iterator {
	iter := &mapIterator{
		data:    data,
		start:   start,
		end:     end,
		reverse: reverse,
		index:   0,
	}
	
	// Filter and sort keys
	for keyStr := range data {
		key := []byte(keyStr)
		if isInDomain(key, start, end) {
			iter.keys = append(iter.keys, keyStr)
		}
	}
	
	sort.Strings(iter.keys)
	if reverse {
		// Reverse the slice
		for i, j := 0, len(iter.keys)-1; i < j; i, j = i+1, j-1 {
			iter.keys[i], iter.keys[j] = iter.keys[j], iter.keys[i]
		}
	}
	
	return iter
}

func (iter *mapIterator) Domain() (start, end []byte) {
	return iter.start, iter.end
}

func (iter *mapIterator) Valid() bool {
	return iter.index < len(iter.keys)
}

func (iter *mapIterator) Next() {
	iter.index++
}

func (iter *mapIterator) Key() []byte {
	if !iter.Valid() {
		return nil
	}
	return []byte(iter.keys[iter.index])
}

func (iter *mapIterator) Value() []byte {
	if !iter.Valid() {
		return nil
	}
	return iter.data[iter.keys[iter.index]]
}

func (iter *mapIterator) Close() error {
	iter.keys = nil
	iter.data = nil
	return nil
}

// emptyIterator is an iterator with no elements
type emptyIterator struct{}

func (iter *emptyIterator) Domain() (start, end []byte) {
	return nil, nil
}

func (iter *emptyIterator) Valid() bool {
	return false
}

func (iter *emptyIterator) Next() {}

func (iter *emptyIterator) Key() []byte {
	return nil
}

func (iter *emptyIterator) Value() []byte {
	return nil
}

func (iter *emptyIterator) Close() error {
	return nil
}

// isInDomain checks if key is within [start, end) domain
func isInDomain(key, start, end []byte) bool {
	if start != nil && bytes.Compare(key, start) < 0 {
		return false
	}
	if end != nil && bytes.Compare(key, end) >= 0 {
		return false
	}
	return true
}