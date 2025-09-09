package commitment

import (
	"github.com/b2network/pulsar/store/types"
)

// Tree defines the interface for commitment trees
type Tree interface {
	// Set stores a key-value pair in the tree
	Set(key, value []byte) (bool, error)

	// Remove deletes a key from the tree
	Remove(key []byte) ([]byte, bool, error)

	// Get retrieves a value by key
	Get(key []byte) ([]byte, error)

	// Has checks if a key exists
	Has(key []byte) (bool, error)

	// Hash returns the root hash of the tree
	Hash() ([]byte, error)

	// Version returns the current version of the tree
	Version() int64

	// SaveVersion saves the current state and returns the hash and version
	SaveVersion() ([]byte, int64, error)

	// LoadVersion loads a specific version of the tree
	LoadVersion(version int64) error

	// DeleteVersion deletes a specific version
	DeleteVersion(version int64) error

	// GetProof generates a proof for a key
	GetProof(key []byte) (*types.Proof, error)

	// GetVersionedProof generates a proof for a key at a specific version
	GetVersionedProof(key []byte, version int64) (*types.Proof, error)

	// VerifyProof verifies a proof against the root hash
	VerifyProof(proof *types.Proof, rootHash []byte, key []byte, value []byte) bool

	// Iterator creates an iterator for the tree
	Iterator(start, end []byte, ascending bool) (types.Iterator, error)

	// Close closes the tree
	Close() error
}

// StateCommitmentImpl implements the StateCommitment interface
type StateCommitmentImpl struct {
	// Map of store names to their commitment trees
	trees map[string]Tree

	// Store for commitment info metadata
	commitInfoStore CommitInfoStore

	// Current version being worked on
	workingVersion int64

	// Latest committed version
	latestVersion int64
}

// CommitInfoStore defines interface for storing commit information
type CommitInfoStore interface {
	// SaveCommitInfo saves commit information for a version
	SaveCommitInfo(version int64, info *types.CommitInfo) error

	// LoadCommitInfo loads commit information for a version
	LoadCommitInfo(version int64) (*types.CommitInfo, error)

	// GetLatestVersion returns the latest committed version
	GetLatestVersion() (int64, error)

	// Close closes the commit info store
	Close() error
}

// NewStateCommitment creates a new StateCommitment implementation
func NewStateCommitment(trees map[string]Tree, commitInfoStore CommitInfoStore) *StateCommitmentImpl {
	return &StateCommitmentImpl{
		trees:           trees,
		commitInfoStore: commitInfoStore,
		workingVersion:  0,
		latestVersion:   0,
	}
}

// WriteChangeset writes a changeset to the commitment layer
func (s *StateCommitmentImpl) WriteChangeset(cs *types.ChangeSet) error {
	// Apply changes to respective trees
	for _, pair := range cs.Pairs {
		// Extract store name from key (assuming format: "store_name/actual_key")
		storeName, actualKey := s.parseKey(pair.Key)

		tree := s.trees[storeName]
		if tree == nil {
			// Create new tree for this store if it doesn't exist
			var err error
			tree, err = s.createTree(storeName)
			if err != nil {
				return err
			}
			s.trees[storeName] = tree
		}

		if pair.Delete {
			_, _, err := tree.Remove(actualKey)
			if err != nil {
				return err
			}
		} else {
			_, err := tree.Set(actualKey, pair.Value)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Commit commits the current changeset and returns the root hash
func (s *StateCommitmentImpl) Commit(version int64) ([]byte, error) {
	storeInfos := make([]types.StoreInfo, 0, len(s.trees))

	// Save each tree and collect store infos
	for storeName, tree := range s.trees {
		hash, treeVersion, err := tree.SaveVersion()
		if err != nil {
			return nil, err
		}

		storeInfos = append(storeInfos, types.StoreInfo{
			Name: storeName,
			CommitId: types.CommitID{
				Version: treeVersion,
				Hash:    hash,
			},
		})
	}

	// Create commit info
	commitInfo := &types.CommitInfo{
		Version:    version,
		StoreInfos: storeInfos,
	}

	// Save commit info
	if err := s.commitInfoStore.SaveCommitInfo(version, commitInfo); err != nil {
		return nil, err
	}

	s.latestVersion = version
	s.workingVersion = version + 1

	// Calculate overall root hash (simplified - just hash of all store hashes)
	return s.calculateRootHash(storeInfos), nil
}

// GetCommitInfo returns commit information for a specific version
func (s *StateCommitmentImpl) GetCommitInfo(version int64) (*types.CommitInfo, error) {
	return s.commitInfoStore.LoadCommitInfo(version)
}

// GetProof generates a proof for a key at a specific version
func (s *StateCommitmentImpl) GetProof(storeKey types.StoreKey, version int64, key []byte) (*types.Proof, error) {
	tree := s.trees[storeKey.Name()]
	if tree == nil {
		return nil, nil
	}

	return tree.GetVersionedProof(key, version)
}

// GetLatestVersion returns the latest committed version
func (s *StateCommitmentImpl) GetLatestVersion() (int64, error) {
	return s.latestVersion, nil
}

func (s *StateCommitmentImpl) parseKey(key []byte) (string, []byte) {
	// Simple implementation - splits on first '/'
	keyStr := string(key)
	for i, char := range keyStr {
		if char == '/' {
			return keyStr[:i], []byte(keyStr[i+1:])
		}
	}
	return "default", key
}

func (s *StateCommitmentImpl) createTree(storeName string) (Tree, error) {
	// This would create an actual IAVL tree
	// For now, return a mock tree
	return NewMockTree(), nil
}

func (s *StateCommitmentImpl) calculateRootHash(storeInfos []types.StoreInfo) []byte {
	// Simplified root hash calculation
	// In practice, this would be a proper merkle tree of store hashes
	hash := make([]byte, 32)
	for i, info := range storeInfos {
		if len(info.CommitId.Hash) > 0 {
			hash[i%32] ^= info.CommitId.Hash[0]
		}
	}
	return hash
}
