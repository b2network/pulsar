package pruning

import (
	"fmt"
	"sync"
	"time"

	"github.com/b2network/pulsar/store/types"
)

// Manager handles pruning of historical versions
type Manager struct {
	mu sync.RWMutex

	// Pruning configuration
	options types.PruningOptions

	// Components that can be pruned
	stateStorage    StateStoragePruner
	stateCommitment StateCommitmentPruner

	// Pruning state
	lastPruneHeight int64
	pruning         bool

	// Background pruning
	stopChan chan struct{}
	stopped  bool
}

// StateStoragePruner defines interface for pruning state storage
type StateStoragePruner interface {
	PruneVersions(versions []int64) error
}

// StateCommitmentPruner defines interface for pruning state commitment
type StateCommitmentPruner interface {
	PruneVersions(versions []int64) error
}

// Config defines configuration for the pruning manager
type Config struct {
	Options         types.PruningOptions
	StateStorage    StateStoragePruner
	StateCommitment StateCommitmentPruner
	PruningInterval time.Duration
}

// NewManager creates a new pruning manager
func NewManager(config Config) *Manager {
	return &Manager{
		options:         config.Options,
		stateStorage:    config.StateStorage,
		stateCommitment: config.StateCommitment,
		lastPruneHeight: 0,
		pruning:         false,
		stopChan:        make(chan struct{}),
		stopped:         false,
	}
}

// Start starts the background pruning process
func (m *Manager) Start() {
	if m.options.Strategy == types.PruningNothing {
		return // No pruning needed
	}

	go m.pruningLoop()
}

// Stop stops the background pruning process
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.stopped {
		close(m.stopChan)
		m.stopped = true
	}
}

// ShouldPrune returns whether the given height should be pruned
func (m *Manager) ShouldPrune(currentHeight, height int64) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.options.ShouldPrune(currentHeight, height)
}

// PruneHeight prunes data at a specific height
func (m *Manager) PruneHeight(height int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.pruning {
		return fmt.Errorf("pruning already in progress")
	}

	m.pruning = true
	defer func() {
		m.pruning = false
	}()

	versions := []int64{height}

	// Prune state storage
	if m.stateStorage != nil {
		if err := m.stateStorage.PruneVersions(versions); err != nil {
			return fmt.Errorf("failed to prune state storage: %w", err)
		}
	}

	// Prune state commitment
	if m.stateCommitment != nil {
		if err := m.stateCommitment.PruneVersions(versions); err != nil {
			return fmt.Errorf("failed to prune state commitment: %w", err)
		}
	}

	return nil
}

// PruneRange prunes a range of heights
func (m *Manager) PruneRange(fromHeight, toHeight int64) error {
	if fromHeight > toHeight {
		return fmt.Errorf("invalid range: from %d to %d", fromHeight, toHeight)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.pruning {
		return fmt.Errorf("pruning already in progress")
	}

	m.pruning = true
	defer func() {
		m.pruning = false
	}()

	// Collect versions to prune
	versions := make([]int64, 0, toHeight-fromHeight+1)
	for height := fromHeight; height <= toHeight; height++ {
		versions = append(versions, height)
	}

	// Prune state storage
	if m.stateStorage != nil {
		if err := m.stateStorage.PruneVersions(versions); err != nil {
			return fmt.Errorf("failed to prune state storage: %w", err)
		}
	}

	// Prune state commitment
	if m.stateCommitment != nil {
		if err := m.stateCommitment.PruneVersions(versions); err != nil {
			return fmt.Errorf("failed to prune state commitment: %w", err)
		}
	}

	m.lastPruneHeight = toHeight
	return nil
}

// GetPrunableHeights returns heights that can be pruned given the current height
func (m *Manager) GetPrunableHeights(currentHeight int64) []int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.options.Strategy == types.PruningNothing {
		return nil
	}

	var prunableHeights []int64

	// Calculate pruning range
	var startHeight int64 = 1
	if m.lastPruneHeight > 0 {
		startHeight = m.lastPruneHeight + 1
	}

	// Don't prune recent heights
	maxPruneHeight := currentHeight - int64(m.options.KeepRecent)
	if maxPruneHeight < startHeight {
		return nil
	}

	for height := startHeight; height <= maxPruneHeight; height++ {
		if m.options.ShouldPrune(currentHeight, height) {
			prunableHeights = append(prunableHeights, height)
		}
	}

	return prunableHeights
}

// SetOptions updates the pruning options
func (m *Manager) SetOptions(options types.PruningOptions) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.options = options
}

// GetOptions returns the current pruning options
func (m *Manager) GetOptions() types.PruningOptions {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.options
}

// IsPruning returns whether pruning is currently in progress
func (m *Manager) IsPruning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.pruning
}

// GetLastPruneHeight returns the last pruned height
func (m *Manager) GetLastPruneHeight() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.lastPruneHeight
}

func (m *Manager) pruningLoop() {
	if m.options.Interval == 0 {
		return // No interval pruning
	}

	ticker := time.NewTicker(time.Duration(m.options.Interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			// This would need access to current height from somewhere
			// For now, we'll skip automatic pruning
			continue
		}
	}
}

// AutoPrune performs automatic pruning based on current height
func (m *Manager) AutoPrune(currentHeight int64) error {
	if m.options.Strategy == types.PruningNothing {
		return nil
	}

	// Only prune if we're at an interval boundary
	if m.options.Interval > 0 && currentHeight%int64(m.options.Interval) != 0 {
		return nil
	}

	prunableHeights := m.GetPrunableHeights(currentHeight)
	if len(prunableHeights) == 0 {
		return nil
	}

	// Prune in batches to avoid large operations
	const batchSize = 100
	for i := 0; i < len(prunableHeights); i += batchSize {
		end := i + batchSize
		if end > len(prunableHeights) {
			end = len(prunableHeights)
		}

		batch := prunableHeights[i:end]
		fromHeight := batch[0]
		toHeight := batch[len(batch)-1]

		if err := m.PruneRange(fromHeight, toHeight); err != nil {
			return fmt.Errorf("failed to prune batch %d-%d: %w", fromHeight, toHeight, err)
		}
	}

	return nil
}
