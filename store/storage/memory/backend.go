package memory

import (
	"bytes"
	"sort"
	"sync"

	"github.com/b2network/pulsar/store/types"
)

// Backend implements an in-memory storage backend for testing
type Backend struct {
	mu            sync.RWMutex
	data          map[string]map[int64][]byte // key -> version -> value
	deletions     map[string]map[int64]bool   // key -> version -> deleted
	buffer        map[string][]byte           // buffered changes
	bufferDeletes map[string]bool
	latestVersion int64
}

// NewBackend creates a new memory backend
func NewBackend() *Backend {
	return &Backend{
		data:          make(map[string]map[int64][]byte),
		deletions:     make(map[string]map[int64]bool),
		buffer:        make(map[string][]byte),
		bufferDeletes: make(map[string]bool),
		latestVersion: 0,
	}
}

func (b *Backend) Get(key []byte, version int64) ([]byte, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	keyStr := string(key)

	// Check buffer first
	if b.bufferDeletes[keyStr] {
		return nil, nil
	}
	if value, exists := b.buffer[keyStr]; exists {
		return value, nil
	}

	// Check versioned data
	versions, exists := b.data[keyStr]
	if !exists {
		return nil, nil
	}

	// Find the latest version <= requested version
	var latestValue []byte
	var found bool
	for v, value := range versions {
		if v <= version {
			if !found || v > version {
				// Check if this version was deleted
				if deletions, exists := b.deletions[keyStr]; exists && deletions[v] {
					continue
				}
				latestValue = value
				found = true
			}
		}
	}

	if !found {
		return nil, nil
	}

	return latestValue, nil
}

func (b *Backend) Set(key []byte, value []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()

	keyStr := string(key)
	b.buffer[keyStr] = value
	delete(b.bufferDeletes, keyStr)
}

func (b *Backend) Delete(key []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()

	keyStr := string(key)
	b.bufferDeletes[keyStr] = true
	delete(b.buffer, keyStr)
}

func (b *Backend) Has(key []byte, version int64) (bool, error) {
	value, err := b.Get(key, version)
	if err != nil {
		return false, err
	}
	return value != nil, nil
}

func (b *Backend) Iterator(start, end []byte, version int64) (types.Iterator, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return newMemoryIterator(b, start, end, version, false), nil
}

func (b *Backend) ReverseIterator(start, end []byte, version int64) (types.Iterator, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return newMemoryIterator(b, start, end, version, true), nil
}

func (b *Backend) Flush(version int64) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Apply buffered changes
	for keyStr, value := range b.buffer {
		if b.data[keyStr] == nil {
			b.data[keyStr] = make(map[int64][]byte)
		}
		b.data[keyStr][version] = value
	}

	// Apply buffered deletions
	for keyStr := range b.bufferDeletes {
		if b.deletions[keyStr] == nil {
			b.deletions[keyStr] = make(map[int64]bool)
		}
		b.deletions[keyStr][version] = true
	}

	// Clear buffers
	b.buffer = make(map[string][]byte)
	b.bufferDeletes = make(map[string]bool)

	// Update latest version
	if version > b.latestVersion {
		b.latestVersion = version
	}

	return nil
}

func (b *Backend) GetLatestVersion() (int64, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.latestVersion, nil
}

func (b *Backend) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.data = nil
	b.deletions = nil
	b.buffer = nil
	b.bufferDeletes = nil
	return nil
}

// memoryIterator implements Iterator for the memory backend
type memoryIterator struct {
	backend *Backend
	keys    []string
	values  [][]byte
	start   []byte
	end     []byte
	version int64
	reverse bool
	index   int
}

func newMemoryIterator(backend *Backend, start, end []byte, version int64, reverse bool) *memoryIterator {
	iter := &memoryIterator{
		backend: backend,
		start:   start,
		end:     end,
		version: version,
		reverse: reverse,
		index:   0,
	}

	// Collect all valid keys and values
	keyValuePairs := make([]struct {
		key   string
		value []byte
	}, 0)

	// Add buffered data first
	for keyStr, value := range backend.buffer {
		key := []byte(keyStr)
		if iter.isInDomain(key) && !backend.bufferDeletes[keyStr] {
			keyValuePairs = append(keyValuePairs, struct {
				key   string
				value []byte
			}{keyStr, value})
		}
	}

	// Add versioned data
	for keyStr, versions := range backend.data {
		key := []byte(keyStr)
		if !iter.isInDomain(key) {
			continue
		}

		// Skip if in buffer (already added above)
		if _, exists := backend.buffer[keyStr]; exists {
			continue
		}
		if backend.bufferDeletes[keyStr] {
			continue
		}

		// Find latest version <= requested version
		var latestValue []byte
		var found bool
		for v, value := range versions {
			if v <= version {
				if !found || v > version {
					// Check if this version was deleted
					if deletions, exists := backend.deletions[keyStr]; exists && deletions[v] {
						continue
					}
					latestValue = value
					found = true
				}
			}
		}

		if found {
			keyValuePairs = append(keyValuePairs, struct {
				key   string
				value []byte
			}{keyStr, latestValue})
		}
	}

	// Sort by key
	sort.Slice(keyValuePairs, func(i, j int) bool {
		return keyValuePairs[i].key < keyValuePairs[j].key
	})

	if reverse {
		// Reverse for descending order
		for i, j := 0, len(keyValuePairs)-1; i < j; i, j = i+1, j-1 {
			keyValuePairs[i], keyValuePairs[j] = keyValuePairs[j], keyValuePairs[i]
		}
	}

	// Extract keys and values
	iter.keys = make([]string, len(keyValuePairs))
	iter.values = make([][]byte, len(keyValuePairs))
	for i, kv := range keyValuePairs {
		iter.keys[i] = kv.key
		iter.values[i] = kv.value
	}

	return iter
}

func (iter *memoryIterator) Domain() (start, end []byte) {
	return iter.start, iter.end
}

func (iter *memoryIterator) Valid() bool {
	return iter.index < len(iter.keys)
}

func (iter *memoryIterator) Next() {
	iter.index++
}

func (iter *memoryIterator) Key() []byte {
	if !iter.Valid() {
		return nil
	}
	return []byte(iter.keys[iter.index])
}

func (iter *memoryIterator) Value() []byte {
	if !iter.Valid() {
		return nil
	}
	return iter.values[iter.index]
}

func (iter *memoryIterator) Close() error {
	iter.keys = nil
	iter.values = nil
	return nil
}

func (iter *memoryIterator) isInDomain(key []byte) bool {
	if iter.start != nil && bytes.Compare(key, iter.start) < 0 {
		return false
	}
	if iter.end != nil && bytes.Compare(key, iter.end) >= 0 {
		return false
	}
	return true
}
