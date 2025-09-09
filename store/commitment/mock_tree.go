package commitment

import (
	"crypto/sha256"
	"sort"

	"github.com/b2network/pulsar/store/types"
)

// MockTree is a simple in-memory tree for testing purposes
// In production, this would be replaced with a real IAVL tree
type MockTree struct {
	data          map[string][]byte
	versions      map[int64]map[string][]byte
	version       int64
	savedVersions map[int64]bool
}

// NewMockTree creates a new mock tree
func NewMockTree() *MockTree {
	return &MockTree{
		data:          make(map[string][]byte),
		versions:      make(map[int64]map[string][]byte),
		version:       0,
		savedVersions: make(map[int64]bool),
	}
}

func (t *MockTree) Set(key, value []byte) (bool, error) {
	keyStr := string(key)
	existed := t.data[keyStr] != nil
	t.data[keyStr] = value
	return !existed, nil
}

func (t *MockTree) Remove(key []byte) ([]byte, bool, error) {
	keyStr := string(key)
	value, existed := t.data[keyStr]
	if existed {
		delete(t.data, keyStr)
	}
	return value, existed, nil
}

func (t *MockTree) Get(key []byte) ([]byte, error) {
	return t.data[string(key)], nil
}

func (t *MockTree) Has(key []byte) (bool, error) {
	_, exists := t.data[string(key)]
	return exists, nil
}

func (t *MockTree) Hash() ([]byte, error) {
	return t.calculateHash(t.data), nil
}

func (t *MockTree) Version() int64 {
	return t.version
}

func (t *MockTree) SaveVersion() ([]byte, int64, error) {
	t.version++

	// Make a copy of current data
	dataCopy := make(map[string][]byte)
	for k, v := range t.data {
		dataCopy[k] = v
	}
	t.versions[t.version] = dataCopy
	t.savedVersions[t.version] = true

	hash := t.calculateHash(dataCopy)
	return hash, t.version, nil
}

func (t *MockTree) LoadVersion(version int64) error {
	versionData, exists := t.versions[version]
	if !exists {
		return nil // Version doesn't exist, start with empty tree
	}

	// Replace current data with version data
	t.data = make(map[string][]byte)
	for k, v := range versionData {
		t.data[k] = v
	}

	t.version = version
	return nil
}

func (t *MockTree) DeleteVersion(version int64) error {
	delete(t.versions, version)
	delete(t.savedVersions, version)
	return nil
}

func (t *MockTree) GetProof(key []byte) (*types.Proof, error) {
	return t.generateMockProof(key, t.data), nil
}

func (t *MockTree) GetVersionedProof(key []byte, version int64) (*types.Proof, error) {
	versionData, exists := t.versions[version]
	if !exists {
		return nil, nil
	}

	return t.generateMockProof(key, versionData), nil
}

func (t *MockTree) VerifyProof(proof *types.Proof, rootHash []byte, key []byte, value []byte) bool {
	// Simplified verification - just check if proof exists
	return proof != nil && len(proof.Ops) > 0
}

func (t *MockTree) Iterator(start, end []byte, ascending bool) (types.Iterator, error) {
	return newMockTreeIterator(t.data, start, end, ascending), nil
}

func (t *MockTree) Close() error {
	t.data = nil
	t.versions = nil
	return nil
}

func (t *MockTree) calculateHash(data map[string][]byte) []byte {
	if len(data) == 0 {
		return make([]byte, 32)
	}

	// Sort keys for deterministic hashing
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	hasher := sha256.New()
	for _, k := range keys {
		hasher.Write([]byte(k))
		hasher.Write(data[k])
	}

	return hasher.Sum(nil)
}

func (t *MockTree) generateMockProof(key []byte, data map[string][]byte) *types.Proof {
	// Generate a simple mock proof
	return &types.Proof{
		Ops: []types.ProofOp{
			{
				Type: "iavl:v",
				Key:  key,
				Data: []byte("mock-proof-data"),
			},
		},
	}
}

// mockTreeIterator implements Iterator for the mock tree
type mockTreeIterator struct {
	data      map[string][]byte
	keys      []string
	start     []byte
	end       []byte
	ascending bool
	index     int
}

func newMockTreeIterator(data map[string][]byte, start, end []byte, ascending bool) *mockTreeIterator {
	iter := &mockTreeIterator{
		data:      data,
		start:     start,
		end:       end,
		ascending: ascending,
		index:     0,
	}

	// Filter and sort keys
	for k := range data {
		key := []byte(k)
		if iter.isInDomain(key) {
			iter.keys = append(iter.keys, k)
		}
	}

	sort.Strings(iter.keys)
	if !ascending {
		// Reverse for descending order
		for i, j := 0, len(iter.keys)-1; i < j; i, j = i+1, j-1 {
			iter.keys[i], iter.keys[j] = iter.keys[j], iter.keys[i]
		}
	}

	return iter
}

func (iter *mockTreeIterator) Domain() (start, end []byte) {
	return iter.start, iter.end
}

func (iter *mockTreeIterator) Valid() bool {
	return iter.index < len(iter.keys)
}

func (iter *mockTreeIterator) Next() {
	iter.index++
}

func (iter *mockTreeIterator) Key() []byte {
	if !iter.Valid() {
		return nil
	}
	return []byte(iter.keys[iter.index])
}

func (iter *mockTreeIterator) Value() []byte {
	if !iter.Valid() {
		return nil
	}
	return iter.data[iter.keys[iter.index]]
}

func (iter *mockTreeIterator) Close() error {
	iter.data = nil
	iter.keys = nil
	return nil
}

func (iter *mockTreeIterator) isInDomain(key []byte) bool {
	if iter.start != nil && string(key) < string(iter.start) {
		return false
	}
	if iter.end != nil && string(key) >= string(iter.end) {
		return false
	}
	return true
}
