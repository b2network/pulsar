package storage

import (
	"github.com/b2network/pulsar/store/types"
)

// Backend defines the interface for storage backends
type Backend interface {
	// Get retrieves a value by key and version
	Get(key []byte, version int64) ([]byte, error)

	// Set stores a key-value pair (buffered until flush)
	Set(key []byte, value []byte)

	// Delete marks a key for deletion (buffered until flush)
	Delete(key []byte)

	// Has checks if a key exists at the given version
	Has(key []byte, version int64) (bool, error)

	// Iterator creates an iterator over a key range at a specific version
	Iterator(start, end []byte, version int64) (types.Iterator, error)

	// ReverseIterator creates a reverse iterator over a key range at a specific version
	ReverseIterator(start, end []byte, version int64) (types.Iterator, error)

	// Flush writes all buffered changes to the storage backend
	Flush(version int64) error

	// GetLatestVersion returns the latest committed version
	GetLatestVersion() (int64, error)

	// Close closes the backend
	Close() error
}

// StateStorageImpl implements the StateStorage interface using a Backend
type StateStorageImpl struct {
	backend Backend

	// Per-store buffers for batching changes
	storeBuffers map[string]*StoreBuffer
}

// StoreBuffer buffers changes for a single store
type StoreBuffer struct {
	sets    map[string][]byte
	deletes map[string]bool
}

// NewStateStorage creates a new StateStorage implementation
func NewStateStorage(backend Backend) *StateStorageImpl {
	return &StateStorageImpl{
		backend:      backend,
		storeBuffers: make(map[string]*StoreBuffer),
	}
}

// Get retrieves a value from a specific store at a given version
func (s *StateStorageImpl) Get(storeKey types.StoreKey, key []byte, version int64) ([]byte, error) {
	prefixedKey := s.prefixKey(storeKey, key)

	// Check buffer first
	if buffer := s.getStoreBuffer(storeKey.Name()); buffer != nil {
		keyStr := string(key)
		if buffer.deletes[keyStr] {
			return nil, nil
		}
		if value, exists := buffer.sets[keyStr]; exists {
			return value, nil
		}
	}

	return s.backend.Get(prefixedKey, version)
}

// Set stores a key-value pair in the buffer
func (s *StateStorageImpl) Set(storeKey types.StoreKey, key, value []byte) {
	buffer := s.getOrCreateStoreBuffer(storeKey.Name())
	keyStr := string(key)

	buffer.sets[keyStr] = value
	delete(buffer.deletes, keyStr)
}

// Delete marks a key for deletion in the buffer
func (s *StateStorageImpl) Delete(storeKey types.StoreKey, key []byte) {
	buffer := s.getOrCreateStoreBuffer(storeKey.Name())
	keyStr := string(key)

	buffer.deletes[keyStr] = true
	delete(buffer.sets, keyStr)
}

// Has checks if a key exists
func (s *StateStorageImpl) Has(storeKey types.StoreKey, key []byte, version int64) (bool, error) {
	value, err := s.Get(storeKey, key, version)
	if err != nil {
		return false, err
	}
	return value != nil, nil
}

// Iterator creates an iterator for a store
func (s *StateStorageImpl) Iterator(storeKey types.StoreKey, start, end []byte, version int64) (types.Iterator, error) {
	prefixedStart := s.prefixKey(storeKey, start)
	prefixedEnd := s.prefixKey(storeKey, end)

	backendIter, err := s.backend.Iterator(prefixedStart, prefixedEnd, version)
	if err != nil {
		return nil, err
	}

	// Wrap with buffer overlay
	return newBufferIterator(backendIter, s.getStoreBuffer(storeKey.Name()), storeKey.Name(), start, end, false), nil
}

// ReverseIterator creates a reverse iterator for a store
func (s *StateStorageImpl) ReverseIterator(storeKey types.StoreKey, start, end []byte, version int64) (types.Iterator, error) {
	prefixedStart := s.prefixKey(storeKey, start)
	prefixedEnd := s.prefixKey(storeKey, end)

	backendIter, err := s.backend.ReverseIterator(prefixedStart, prefixedEnd, version)
	if err != nil {
		return nil, err
	}

	// Wrap with buffer overlay
	return newBufferIterator(backendIter, s.getStoreBuffer(storeKey.Name()), storeKey.Name(), start, end, true), nil
}

// ApplyChangeset applies a changeset to the storage backend
func (s *StateStorageImpl) ApplyChangeset(version int64, cs *types.ChangeSet) error {
	// Apply buffered changes to backend
	for storeName, buffer := range s.storeBuffers {
		// Apply sets
		for keyStr, value := range buffer.sets {
			key := []byte(keyStr)
			prefixedKey := []byte(storeName + "/" + string(key))
			s.backend.Set(prefixedKey, value)
		}

		// Apply deletes
		for keyStr := range buffer.deletes {
			key := []byte(keyStr)
			prefixedKey := []byte(storeName + "/" + string(key))
			s.backend.Delete(prefixedKey)
		}
	}

	// Apply external changeset
	for _, pair := range cs.Pairs {
		if pair.Delete {
			s.backend.Delete(pair.Key)
		} else {
			s.backend.Set(pair.Key, pair.Value)
		}
	}

	// Flush to storage
	if err := s.backend.Flush(version); err != nil {
		return err
	}

	// Clear buffers
	s.storeBuffers = make(map[string]*StoreBuffer)

	return nil
}

// GetLatestVersion returns the latest committed version
func (s *StateStorageImpl) GetLatestVersion() (int64, error) {
	return s.backend.GetLatestVersion()
}

func (s *StateStorageImpl) prefixKey(storeKey types.StoreKey, key []byte) []byte {
	if key == nil {
		return []byte(storeKey.Name() + "/")
	}
	prefix := storeKey.Name() + "/"
	result := make([]byte, len(prefix)+len(key))
	copy(result, prefix)
	copy(result[len(prefix):], key)
	return result
}

func (s *StateStorageImpl) getStoreBuffer(storeName string) *StoreBuffer {
	return s.storeBuffers[storeName]
}

func (s *StateStorageImpl) getOrCreateStoreBuffer(storeName string) *StoreBuffer {
	buffer := s.storeBuffers[storeName]
	if buffer == nil {
		buffer = &StoreBuffer{
			sets:    make(map[string][]byte),
			deletes: make(map[string]bool),
		}
		s.storeBuffers[storeName] = buffer
	}
	return buffer
}
