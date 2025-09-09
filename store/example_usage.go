package store

import (
	"fmt"

	"github.com/b2network/pulsar/store/commitment"
	"github.com/b2network/pulsar/store/kvstore"
	"github.com/b2network/pulsar/store/pruning"
	"github.com/b2network/pulsar/store/rootstore"
	"github.com/b2network/pulsar/store/storage"
	"github.com/b2network/pulsar/store/storage/memory"
	"github.com/b2network/pulsar/store/types"
)

// ExampleUsage demonstrates how to use the Pulsar storage system
func ExampleUsage(dataDir string) error {
	// 1. Create Memory backend (for testing - in production use RocksDB)
	backend := memory.NewBackend()
	defer backend.Close()

	// 2. Create state storage layer
	stateStorage := storage.NewStateStorage(backend)

	// 3. Create commitment layer
	trees := map[string]commitment.Tree{
		"bank":    commitment.NewMockTree(),
		"staking": commitment.NewMockTree(),
		"gov":     commitment.NewMockTree(),
	}
	commitInfoStore := &mockCommitInfoStore{} // Implementation not shown for brevity
	stateCommitment := commitment.NewStateCommitment(trees, commitInfoStore)

	// 4. Create root store
	rootStoreConfig := rootstore.StoreConfig{
		StateStorage:    stateStorage,
		StateCommitment: stateCommitment,
		Pruning:         types.NewPruningOptions(types.PruningDefault),
		InitialVersion:  0,
	}
	rootStore := rootstore.NewStore(rootStoreConfig)

	// 5. Mount store keys
	bankKey := types.NewKVStoreKey("bank")
	stakingKey := types.NewKVStoreKey("staking")
	govKey := types.NewKVStoreKey("gov")

	rootStore.MountStoreWithDB(bankKey, types.StoreTypeIAVL, nil)
	rootStore.MountStoreWithDB(stakingKey, types.StoreTypeIAVL, nil)
	rootStore.MountStoreWithDB(govKey, types.StoreTypeIAVL, nil)

	// 6. Load latest version (will load version 0 for empty store)
	if err := rootStore.LoadLatestVersion(); err != nil {
		return fmt.Errorf("failed to load latest version: %w", err)
	}

	// 7. Create pruning manager
	pruningConfig := pruning.Config{
		Options:         types.NewPruningOptions(types.PruningDefault),
		StateStorage:    &mockPruner{}, // Implementation not shown
		StateCommitment: &mockPruner{}, // Implementation not shown
	}
	pruningManager := pruning.NewManager(pruningConfig)
	pruningManager.Start()
	defer pruningManager.Stop()

	// 8. Use the stores
	bankStore := rootStore.GetKVStore(bankKey)

	// Example: Store balance
	bankStore.Set([]byte("alice"), []byte("1000"))
	bankStore.Set([]byte("bob"), []byte("500"))

	// Create cache store for transaction
	cacheStore := rootStore.CacheMultiStore()
	cacheBankStore := cacheStore.GetKVStore(bankKey)

	// Modify in cache
	cacheBankStore.Set([]byte("alice"), []byte("900"))
	cacheBankStore.Set([]byte("bob"), []byte("600"))

	// Commit cache changes
	cacheStore.Write()

	// 9. Commit the changes
	commitID := rootStore.Commit()
	fmt.Printf("Committed version %d with hash %x\n", commitID.Version, commitID.Hash)

	// 10. Query data
	result, err := rootStore.Query(bankKey, commitID.Version, []byte("alice"), true)
	if err != nil {
		return err
	}
	fmt.Printf("Alice's balance: %s\n", string(result.Value))

	// 11. Use advanced features
	// Prefix store
	prefixStore := kvstore.NewPrefixStore(bankStore, []byte("balances/"))
	prefixStore.Set([]byte("charlie"), []byte("750"))

	// Gas metered store (requires gas meter implementation)
	gasMeter := &mockGasMeter{limit: 1000000, consumed: 0}
	gasConfig := kvstore.DefaultGasConfig()
	gasStore := kvstore.NewGasKVStore(bankStore, gasMeter, gasConfig)

	// This will consume gas
	gasStore.Get([]byte("alice"))
	fmt.Printf("Gas consumed: %d\n", gasMeter.GasConsumed())

	// 12. Auto-pruning
	if err := pruningManager.AutoPrune(commitID.Version); err != nil {
		return fmt.Errorf("failed to auto-prune: %w", err)
	}

	return nil
}

// Mock implementations for the example (not complete implementations)

type mockCommitInfoStore struct {
	commitInfos map[int64]*types.CommitInfo
}

func (s *mockCommitInfoStore) SaveCommitInfo(version int64, info *types.CommitInfo) error {
	if s.commitInfos == nil {
		s.commitInfos = make(map[int64]*types.CommitInfo)
	}
	s.commitInfos[version] = info
	return nil
}

func (s *mockCommitInfoStore) LoadCommitInfo(version int64) (*types.CommitInfo, error) {
	return s.commitInfos[version], nil
}

func (s *mockCommitInfoStore) GetLatestVersion() (int64, error) {
	var latest int64
	for version := range s.commitInfos {
		if version > latest {
			latest = version
		}
	}
	return latest, nil
}

func (s *mockCommitInfoStore) Close() error {
	return nil
}

type mockPruner struct{}

func (p *mockPruner) PruneVersions(versions []int64) error {
	return nil
}

type mockGasMeter struct {
	limit    uint64
	consumed uint64
}

func (g *mockGasMeter) ConsumeGas(amount uint64, descriptor string) {
	g.consumed += amount
}

func (g *mockGasMeter) GasConsumed() uint64 {
	return g.consumed
}

func (g *mockGasMeter) GasLimit() uint64 {
	return g.limit
}

func (g *mockGasMeter) OutOfGas() bool {
	return g.consumed >= g.limit
}
