package rootstore

import (
	"fmt"
	"sync"

	"github.com/b2network/pulsar/store/types"
)

// Store implements the RootStore interface
type Store struct {
	mu sync.RWMutex

	// Core components
	stateStorage    types.StateStorage
	stateCommitment types.StateCommitment

	// Store management
	stores     map[types.StoreKey]types.CommitKVStore
	keysByName map[string]types.StoreKey
	pruning    types.PruningOptions

	// Version tracking
	lastCommitInfo *types.CommitInfo
	initialVersion int64

	// Cache layer
	cacheStore types.CacheMultiStore

	// Tracing
	traceWriter types.TraceWriter
	tracing     bool
}

// StoreConfig defines configuration for creating a new Store
type StoreConfig struct {
	StateStorage    types.StateStorage
	StateCommitment types.StateCommitment
	Pruning         types.PruningOptions
	InitialVersion  int64
}

// NewStore creates a new RootStore
func NewStore(config StoreConfig) *Store {
	if config.StateStorage == nil {
		panic("state storage cannot be nil")
	}
	if config.StateCommitment == nil {
		panic("state commitment cannot be nil")
	}

	return &Store{
		stateStorage:    config.StateStorage,
		stateCommitment: config.StateCommitment,
		stores:          make(map[types.StoreKey]types.CommitKVStore),
		keysByName:      make(map[string]types.StoreKey),
		pruning:         config.Pruning,
		initialVersion:  config.InitialVersion,
		lastCommitInfo:  &types.CommitInfo{},
	}
}

// GetStateStorage returns the state storage interface
func (s *Store) GetStateStorage() types.StateStorage {
	return s.stateStorage
}

// GetStateCommitment returns the state commitment interface
func (s *Store) GetStateCommitment() types.StateCommitment {
	return s.stateCommitment
}

// SetInitialVersion sets the initial version for the store
func (s *Store) SetInitialVersion(version int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.initialVersion = version
}

// GetStoreType returns the store type
func (s *Store) GetStoreType() types.StoreType {
	return types.StoreTypeMulti
}

// MountStoreWithDB mounts a store with the given database
func (s *Store) MountStoreWithDB(key types.StoreKey, typ types.StoreType, db types.DB) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stores[key] != nil {
		panic(fmt.Sprintf("store duplicate store key: %v", key))
	}

	if s.keysByName[key.Name()] != nil {
		panic(fmt.Sprintf("store duplicate store key name: %s", key.Name()))
	}

	// Create appropriate store based on type
	var store types.CommitKVStore
	switch typ {
	case types.StoreTypeIAVL:
		store = newIAVLStore(key, db, s)
	case types.StoreTypeDB:
		store = newDBStore(key, db, s)
	case types.StoreTypeTransient:
		store = newTransientStore(key)
	case types.StoreTypeMemory:
		store = newMemoryStore(key)
	default:
		panic(fmt.Sprintf("unsupported store type: %v", typ))
	}

	s.stores[key] = store
	s.keysByName[key.Name()] = key
}

// LoadLatestVersion loads the latest version of the store
func (s *Store) LoadLatestVersion() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	version, err := s.stateStorage.GetLatestVersion()
	if err != nil {
		return fmt.Errorf("failed to get latest version: %w", err)
	}

	return s.loadVersionInternal(version)
}

// LoadVersion loads a specific version of the store
func (s *Store) LoadVersion(version int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.loadVersionInternal(version)
}

func (s *Store) loadVersionInternal(version int64) error {
	// Validate version
	if version < s.initialVersion {
		return fmt.Errorf("cannot load version %d, initial version is %d", version, s.initialVersion)
	}

	// Load commit info for this version
	commitInfo, err := s.stateCommitment.GetCommitInfo(version)
	if err != nil {
		return fmt.Errorf("failed to get commit info for version %d: %w", version, err)
	}

	s.lastCommitInfo = commitInfo

	// Initialize all mounted stores
	for key, store := range s.stores {
		if err := s.loadStoreVersion(key, store, version); err != nil {
			return fmt.Errorf("failed to load store %s: %w", key.Name(), err)
		}
	}

	return nil
}

func (s *Store) loadStoreVersion(key types.StoreKey, store types.CommitKVStore, version int64) error {
	// For now, we'll assume stores can load themselves
	// In a full implementation, this would coordinate with the storage layers
	return nil
}

// GetStore returns a store by key
func (s *Store) GetStore(key types.StoreKey) types.Store {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.stores[key]
}

// GetKVStore returns a KVStore by key
func (s *Store) GetKVStore(key types.StoreKey) types.KVStore {
	s.mu.RLock()
	defer s.mu.RUnlock()

	store := s.stores[key]
	if store == nil {
		panic(fmt.Sprintf("store does not exist for key: %s", key.Name()))
	}

	return store
}

// GetCommitKVStore returns a CommitKVStore by key
func (s *Store) GetCommitKVStore(key types.StoreKey) types.CommitKVStore {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.stores[key]
}

// GetCommitStore returns a CommitStore by key
func (s *Store) GetCommitStore(key types.StoreKey) types.CommitStore {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.stores[key]
}

// Commit commits the current state and returns commit info
func (s *Store) Commit() types.CommitID {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get the next version
	var version int64
	if s.lastCommitInfo != nil {
		version = s.lastCommitInfo.Version + 1
	} else {
		version = s.initialVersion
	}

	if version < s.initialVersion {
		version = s.initialVersion
	}

	// Collect changes from all stores
	changeset := &types.ChangeSet{}
	storeInfos := make([]types.StoreInfo, 0, len(s.stores))

	for key, store := range s.stores {
		// Commit each store and collect its changes
		commitID := store.Commit()

		storeInfos = append(storeInfos, types.StoreInfo{
			Name:     key.Name(),
			CommitId: commitID,
		})

		// TODO: Collect actual changes from store
		// This would involve tracking modifications made to each store
	}

	// Apply changeset to storage layer
	if err := s.stateStorage.ApplyChangeset(version, changeset); err != nil {
		panic(fmt.Sprintf("failed to apply changeset: %v", err))
	}

	// Commit to commitment layer
	if err := s.stateCommitment.WriteChangeset(changeset); err != nil {
		panic(fmt.Sprintf("failed to write changeset to commitment layer: %v", err))
	}

	hash, err := s.stateCommitment.Commit(version)
	if err != nil {
		panic(fmt.Sprintf("failed to commit to commitment layer: %v", err))
	}

	// Update commit info
	commitInfo := &types.CommitInfo{
		Version:    version,
		StoreInfos: storeInfos,
	}

	s.lastCommitInfo = commitInfo

	return types.CommitID{
		Version: version,
		Hash:    hash,
	}
}

// LastCommitID returns the last commit ID
func (s *Store) LastCommitID() types.CommitID {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.lastCommitInfo == nil {
		return types.CommitID{}
	}

	return types.CommitID{
		Version: s.lastCommitInfo.Version,
		Hash:    nil, // TODO: Get actual hash from commitment layer
	}
}

// SetPruning sets the pruning options
func (s *Store) SetPruning(pruning types.PruningOptions) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruning = pruning
}

// GetPruning returns the current pruning options
func (s *Store) GetPruning() types.PruningOptions {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.pruning
}

// TracingEnabled returns whether tracing is enabled
func (s *Store) TracingEnabled() bool {
	return s.tracing
}

// SetTracer sets the trace writer
func (s *Store) SetTracer(w types.TraceWriter) types.MultiStore {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.traceWriter = w
	s.tracing = w != nil

	return s
}

// CacheWrap creates a cache wrapper - not implemented for root store
func (s *Store) CacheWrap() types.CacheWrap {
	panic("cannot cache wrap root store")
}

// CacheMultiStore returns a cached multi-store
func (s *Store) CacheMultiStore() types.CacheMultiStore {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cacheStore == nil {
		s.cacheStore = newCacheMultiStore(s)
	}

	return s.cacheStore
}

// Query performs a query on the store
func (s *Store) Query(storeKey types.StoreKey, version int64, key []byte, prove bool) (types.QueryResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Get value from storage layer
	value, err := s.stateStorage.Get(storeKey, key, version)
	if err != nil {
		return types.QueryResult{}, fmt.Errorf("failed to get value: %w", err)
	}

	result := types.QueryResult{
		Key:    key,
		Value:  value,
		Height: version,
	}

	// Get proof if requested
	if prove {
		proof, err := s.stateCommitment.GetProof(storeKey, version, key)
		if err != nil {
			return types.QueryResult{}, fmt.Errorf("failed to get proof: %w", err)
		}
		result.Proof = proof
	}

	return result, nil
}
