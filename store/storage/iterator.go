package storage

import (
	"bytes"
	"sort"

	"github.com/b2network/pulsar/store/types"
)

// bufferIterator combines backend iterator with buffered changes
type bufferIterator struct {
	backend     types.Iterator
	buffer      *StoreBuffer
	storePrefix string
	start       []byte
	end         []byte
	reverse     bool

	// Merged iteration state
	allKeys     []string
	keyIndex    int
	initialized bool
}

func newBufferIterator(backend types.Iterator, buffer *StoreBuffer, storePrefix string, start, end []byte, reverse bool) *bufferIterator {
	return &bufferIterator{
		backend:     backend,
		buffer:      buffer,
		storePrefix: storePrefix,
		start:       start,
		end:         end,
		reverse:     reverse,
		keyIndex:    0,
		initialized: false,
	}
}

func (iter *bufferIterator) Domain() (start, end []byte) {
	return iter.start, iter.end
}

func (iter *bufferIterator) Valid() bool {
	iter.ensureInitialized()
	return iter.keyIndex < len(iter.allKeys)
}

func (iter *bufferIterator) Next() {
	iter.ensureInitialized()
	iter.keyIndex++
}

func (iter *bufferIterator) Key() []byte {
	if !iter.Valid() {
		return nil
	}
	return []byte(iter.allKeys[iter.keyIndex])
}

func (iter *bufferIterator) Value() []byte {
	if !iter.Valid() {
		return nil
	}

	key := iter.allKeys[iter.keyIndex]

	// Check buffer first
	if iter.buffer != nil {
		if value, exists := iter.buffer.sets[key]; exists {
			return value
		}
	}

	// This is a simplified implementation - in practice we'd need to
	// maintain a mapping from keys to their backend values
	return nil
}

func (iter *bufferIterator) Close() error {
	if iter.backend != nil {
		return iter.backend.Close()
	}
	return nil
}

func (iter *bufferIterator) ensureInitialized() {
	if iter.initialized {
		return
	}

	keySet := make(map[string]bool)

	// Add keys from backend iterator
	if iter.backend != nil {
		for ; iter.backend.Valid(); iter.backend.Next() {
			key := iter.backend.Key()
			if iter.isInDomain(key) {
				// Remove store prefix to get the actual key
				actualKey := iter.removePrefix(key)
				keySet[string(actualKey)] = true
			}
		}
	}

	// Add/remove keys from buffer
	if iter.buffer != nil {
		// Add buffered sets
		for key := range iter.buffer.sets {
			keyBytes := []byte(key)
			if iter.isInDomain(keyBytes) {
				keySet[key] = true
			}
		}

		// Remove buffered deletes
		for key := range iter.buffer.deletes {
			delete(keySet, key)
		}
	}

	// Convert to sorted slice
	iter.allKeys = make([]string, 0, len(keySet))
	for key := range keySet {
		iter.allKeys = append(iter.allKeys, key)
	}

	sort.Strings(iter.allKeys)

	if iter.reverse {
		// Reverse the slice
		for i, j := 0, len(iter.allKeys)-1; i < j; i, j = i+1, j-1 {
			iter.allKeys[i], iter.allKeys[j] = iter.allKeys[j], iter.allKeys[i]
		}
	}

	iter.initialized = true
}

func (iter *bufferIterator) isInDomain(key []byte) bool {
	if iter.start != nil && bytes.Compare(key, iter.start) < 0 {
		return false
	}
	if iter.end != nil && bytes.Compare(key, iter.end) >= 0 {
		return false
	}
	return true
}

func (iter *bufferIterator) removePrefix(key []byte) []byte {
	prefix := iter.storePrefix + "/"
	prefixBytes := []byte(prefix)

	if bytes.HasPrefix(key, prefixBytes) {
		return key[len(prefixBytes):]
	}
	return key
}
