package keeper

import (
	"fmt"
	"sync"
	"time"

	keepertypes "github.com/b2network/pulsar/keeper/types"
	"github.com/b2network/pulsar/pre_execution/types"
	storetypes "github.com/b2network/pulsar/store/types"
)

// PreExecutionManager is the central coordinator for pre-execution transactions
// It manages the entire pre-execution lifecycle from initial execution to result application
type PreExecutionManager struct {
	mu sync.RWMutex

	// Core components
	cache      *types.PreExecutionCache // Caches pre-execution results
	sequencer  *TxSequencer             // Manages transaction ordering
	stateStore storetypes.MultiStore    // Access to blockchain state

	// Module registry
	modules map[string]types.PreExecutableModule // Registered modules that support pre-execution

	// Configuration
	config  *GlobalPreExecConfig // Global pre-execution configuration
	enabled bool                 // Whether pre-execution is globally enabled

	// Statistics and monitoring
	stats *PreExecStats // Performance and usage statistics

	// State management
	baseHeight int64                          // Current blockchain height baseline
	snapshots  map[int64]*types.StateSnapshot // State snapshots for rollback

	// Resource management
	gasTracker *GasTracker // Tracks gas usage across pre-executions

	// Validator information
	validatorID string // This node's validator ID
	isValidator bool   // Whether this node is a validator

	// Cleanup and maintenance
	cleanupTicker *time.Ticker  // Periodic cleanup timer
	stopCleanup   chan struct{} // Signal to stop cleanup
}

// GlobalPreExecConfig contains global configuration for pre-execution
type GlobalPreExecConfig struct {
	// Core settings
	Enabled            bool          `json:"enabled"`                // Global enable/disable
	MaxPreExecPerBlock uint32        `json:"max_pre_exec_per_block"` // Limit per block
	MaxTotalGas        uint64        `json:"max_total_gas"`          // Total gas limit for all pre-executions
	DefaultTimeout     time.Duration `json:"default_timeout"`        // Default execution timeout

	// Cache configuration
	CacheConfig types.CacheConfig `json:"cache_config"` // Cache settings

	// Sequencer configuration
	SequencerConfig SequencerConfig `json:"sequencer_config"` // Sequencer settings

	// Resource limits
	MaxMemoryUsage    uint64 `json:"max_memory_usage"`    // Maximum memory for pre-execution
	MaxConcurrentExec uint32 `json:"max_concurrent_exec"` // Maximum concurrent pre-executions

	// Monitoring
	MetricsEnabled  bool          `json:"metrics_enabled"`  // Enable metrics collection
	CleanupInterval time.Duration `json:"cleanup_interval"` // How often to run cleanup
}

// PreExecStats tracks pre-execution performance and usage statistics
type PreExecStats struct {
	mu sync.RWMutex

	// Execution counts
	TotalAttempts   uint64 `json:"total_attempts"`   // Total pre-execution attempts
	SuccessfulExecs uint64 `json:"successful_execs"` // Successful pre-executions
	FailedExecs     uint64 `json:"failed_execs"`     // Failed pre-executions
	CacheHits       uint64 `json:"cache_hits"`       // Cache hits
	CacheMisses     uint64 `json:"cache_misses"`     // Cache misses

	// Performance metrics
	AverageExecTime time.Duration `json:"average_exec_time"` // Average execution time
	TotalGasUsed    uint64        `json:"total_gas_used"`    // Total gas consumed
	AverageGasUsed  uint64        `json:"average_gas_used"`  // Average gas per execution

	// Resource usage
	PeakMemoryUsage    uint64 `json:"peak_memory_usage"`    // Peak memory usage
	CurrentMemoryUsage uint64 `json:"current_memory_usage"` // Current memory usage

	// Module breakdown
	ModuleStats map[string]*ModuleStats `json:"module_stats"` // Per-module statistics
}

// ModuleStats tracks statistics for individual modules
type ModuleStats struct {
	Attempts        uint64        `json:"attempts"`          // Attempts for this module
	Successes       uint64        `json:"successes"`         // Successful executions
	Failures        uint64        `json:"failures"`          // Failed executions
	AverageGasUsed  uint64        `json:"average_gas_used"`  // Average gas usage
	AverageExecTime time.Duration `json:"average_exec_time"` // Average execution time
}

// GasTracker manages gas usage tracking for pre-executions
type GasTracker struct {
	mu sync.Mutex

	totalGasUsed  uint64        // Total gas used in current period
	maxTotalGas   uint64        // Maximum allowed total gas
	resetInterval time.Duration // How often to reset gas counter
	lastReset     time.Time     // When gas counter was last reset
}

// NewPreExecutionManager creates a new pre-execution manager
func NewPreExecutionManager(
	stateStore storetypes.MultiStore,
	config *GlobalPreExecConfig,
	validatorID string,
	isValidator bool,
) (*PreExecutionManager, error) {

	// Validate configuration
	if config == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}
	if stateStore == nil {
		return nil, fmt.Errorf("state store cannot be nil")
	}

	// Create cache
	cache := types.NewPreExecutionCache(config.CacheConfig)

	// Create sequencer
	sequencer := NewTxSequencer(0, config.SequencerConfig)

	// Create gas tracker
	gasTracker := &GasTracker{
		maxTotalGas:   config.MaxTotalGas,
		resetInterval: time.Hour, // Reset gas usage every hour
		lastReset:     time.Now(),
	}

	// Create manager
	manager := &PreExecutionManager{
		cache:       cache,
		sequencer:   sequencer,
		stateStore:  stateStore,
		modules:     make(map[string]types.PreExecutableModule),
		config:      config,
		enabled:     config.Enabled,
		stats:       &PreExecStats{ModuleStats: make(map[string]*ModuleStats)},
		snapshots:   make(map[int64]*types.StateSnapshot),
		gasTracker:  gasTracker,
		validatorID: validatorID,
		isValidator: isValidator,
		stopCleanup: make(chan struct{}),
	}

	// Start periodic cleanup if enabled
	if config.CleanupInterval > 0 {
		manager.cleanupTicker = time.NewTicker(config.CleanupInterval)
		go manager.runPeriodicCleanup()
	}

	return manager, nil
}

// RegisterModule registers a module for pre-execution support
func (m *PreExecutionManager) RegisterModule(module types.PreExecutableModule) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	moduleName := module.GetModuleName()
	if moduleName == "" {
		return fmt.Errorf("module name cannot be empty")
	}

	// Check if module is already registered
	if _, exists := m.modules[moduleName]; exists {
		return fmt.Errorf("module %s is already registered", moduleName)
	}

	// Register module
	m.modules[moduleName] = module

	// Initialize module statistics
	m.stats.ModuleStats[moduleName] = &ModuleStats{}

	return nil
}

// ShouldPreExecute determines if a transaction should be pre-executed
func (m *PreExecutionManager) ShouldPreExecute(msgs []types.PreExecutableMsg) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if pre-execution is globally enabled
	if !m.enabled {
		return false
	}

	// Check if this node is a validator (only validators pre-execute)
	if !m.isValidator {
		return false
	}

	// Check each message in the transaction
	for _, msg := range msgs {
		// Check if message supports pre-execution
		if !msg.IsPreExecutable() {
			return false
		}

		// Check if corresponding module is registered and enabled
		if !m.isModuleEnabled(getModuleFromMsg(msg)) {
			return false
		}

		// Check gas limits
		hints := msg.GetPreExecutionHints()
		if hints != nil && hints.MaxGasForPreExec > 0 {
			if !m.gasTracker.CanUseGas(hints.MaxGasForPreExec) {
				return false
			}
		}
	}

	return true
}

// PreExecuteTransaction executes a transaction in pre-execution mode
func (m *PreExecutionManager) PreExecuteTransaction(
	ctx keepertypes.Context,
	txHash string,
	msgs []types.PreExecutableMsg,
) (*types.TxPreExecResult, error) {

	startTime := time.Now()

	// Update statistics
	m.updateStats(func(stats *PreExecStats) {
		stats.TotalAttempts++
	})

	// Check if result is already cached
	if cachedResult, exists := m.cache.Get(txHash); exists {
		// Validate cached result is still valid
		if m.validateCachedResult(ctx, cachedResult) {
			m.updateStats(func(stats *PreExecStats) {
				stats.CacheHits++
			})
			return cachedResult, nil
		}
		// Remove invalid cached result
		m.cache.Remove(txHash)
	}

	m.updateStats(func(stats *PreExecStats) {
		stats.CacheMisses++
	})

	// Create pre-execution context with state isolation
	preExecCtx, err := m.createPreExecutionContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create pre-execution context: %w", err)
	}

	// Execute each message
	msgResults := make([]*types.PreExecResult, 0, len(msgs))
	var totalGasUsed uint64
	var allStateChanges *types.StateChangeSet

	for i, msg := range msgs {
		// Get the appropriate module
		moduleName := getModuleFromMsg(msg)
		module, exists := m.modules[moduleName]
		if !exists {
			return nil, types.NewPreExecError(
				types.ErrCodeMsgNotSupported,
				"module not registered",
				txHash, moduleName, getMessageType(msg), nil)
		}

		// Pre-execute the message
		result, err := module.PreExecuteMsg(preExecCtx, msg)
		if err != nil {
			m.updateModuleStats(moduleName, false, 0, time.Since(startTime))
			return nil, fmt.Errorf("failed to pre-execute message %d: %w", i, err)
		}

		msgResults = append(msgResults, result)
		totalGasUsed += result.GasUsed

		// Merge state changes
		if allStateChanges == nil {
			allStateChanges = result.StateChanges
		} else {
			allStateChanges = m.mergeStateChanges(allStateChanges, result.StateChanges)
		}

		// Update module statistics
		m.updateModuleStats(moduleName, true, result.GasUsed, time.Since(startTime))
	}

	// Record gas usage
	m.gasTracker.RecordGasUsage(totalGasUsed)

	// Get transaction priority for sequencing
	priority := m.calculateTransactionPriority(msgs)

	// Add to sequencer to get sequence number
	sequence, err := m.sequencer.AddTransaction(txHash, m.validatorID, priority)
	if err != nil {
		return nil, fmt.Errorf("failed to sequence transaction: %w", err)
	}

	// Create transaction result
	txResult := &types.TxPreExecResult{
		TxHash:       txHash,
		MsgResults:   msgResults,
		TotalGasUsed: totalGasUsed,
		Success:      true,
		StateChanges: allStateChanges,
		Priority:     priority,
		Sequence:     sequence,
		PreExecTime:  time.Now(),
		ValidUntil:   time.Now().Add(m.config.CacheConfig.TTL),
	}

	// Cache the result
	if !m.cache.Set(txHash, txResult) {
		// Cache full - this is not a fatal error, just log it
		// In production, you might want to emit a metric or log
	}

	// Update success statistics
	execTime := time.Since(startTime)
	m.updateStats(func(stats *PreExecStats) {
		stats.SuccessfulExecs++
		stats.TotalGasUsed += totalGasUsed

		// Update average execution time
		if stats.SuccessfulExecs == 1 {
			stats.AverageExecTime = execTime
		} else {
			stats.AverageExecTime = time.Duration(
				(int64(stats.AverageExecTime)*int64(stats.SuccessfulExecs-1) + int64(execTime)) /
					int64(stats.SuccessfulExecs))
		}

		// Update average gas usage
		stats.AverageGasUsed = stats.TotalGasUsed / stats.SuccessfulExecs
	})

	return txResult, nil
}

// ValidateCachedResult validates that a cached result is still applicable
func (m *PreExecutionManager) ValidateCachedResult(
	ctx keepertypes.Context,
	result *types.TxPreExecResult,
) error {
	// Check if result has expired
	if time.Now().After(result.ValidUntil) {
		return types.NewPreExecError(
			types.ErrCodeCacheExpired,
			"cached result has expired",
			result.TxHash, "", "", types.ErrCacheResultExpired)
	}

	// Validate each message result
	for _, msgResult := range result.MsgResults {
		// Check if the state reads are still valid
		for _, read := range msgResult.ReadSet {
			storeKey := storetypes.NewKVStoreKey(read.StoreKey)
			store := ctx.KVStore(storeKey)
			currentValue := store.Get(read.Key)

			// Compare current value with cached read value
			if !bytesEqual(currentValue, read.Value) {
				return types.NewPreExecError(
					types.ErrCodeCacheInvalid,
					"state has changed since pre-execution",
					result.TxHash, read.StoreKey, "", types.ErrCacheResultInvalid)
			}
		}
	}

	return nil
}

// ApplyCachedResult applies a validated cached result to the current state
func (m *PreExecutionManager) ApplyCachedResult(
	ctx keepertypes.Context,
	result *types.TxPreExecResult,
) error {
	// Apply state changes from the cached result
	if result.StateChanges != nil {
		for storeName, storeChanges := range result.StateChanges.StoreChanges {
			storeKey := storetypes.NewKVStoreKey(storeName)
			store := ctx.KVStore(storeKey)

			// Apply sets
			for keyStr, value := range storeChanges.Sets {
				store.Set([]byte(keyStr), value)
			}

			// Apply deletes
			for _, keyStr := range storeChanges.Deletes {
				store.Delete([]byte(keyStr))
			}
		}
	}

	// Emit events from the cached result
	eventManager := ctx.EventManager()
	for _, msgResult := range result.MsgResults {
		for _, event := range msgResult.Events {
			eventManager.EmitEvent(event)
		}
	}

	return nil
}

// GetCachedResult retrieves a cached pre-execution result
func (m *PreExecutionManager) GetCachedResult(txHash string) (*types.TxPreExecResult, bool) {
	return m.cache.Get(txHash)
}

// GetOrderedTransactions returns transactions in execution order
func (m *PreExecutionManager) GetOrderedTransactions() []string {
	return m.sequencer.GetOrderedTransactions()
}

// RemoveTransaction removes a transaction from cache and sequencer
func (m *PreExecutionManager) RemoveTransaction(txHash string) {
	m.cache.Remove(txHash)
	m.sequencer.RemoveTransaction(txHash)
}

// UpdateBaseHeight updates the base height for sequencing
func (m *PreExecutionManager) UpdateBaseHeight(newHeight int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.baseHeight = newHeight
	m.sequencer.UpdateBaseHeight(newHeight)

	// Clean up old snapshots
	for height := range m.snapshots {
		if height < newHeight-10 { // Keep last 10 block snapshots
			delete(m.snapshots, height)
		}
	}
}

// GetStats returns current pre-execution statistics
func (m *PreExecutionManager) GetStats() *PreExecStats {
	m.stats.mu.RLock()
	defer m.stats.mu.RUnlock()

	// Return a deep copy to prevent external modification
	statsCopy := *m.stats
	statsCopy.ModuleStats = make(map[string]*ModuleStats)
	for k, v := range m.stats.ModuleStats {
		moduleCopy := *v
		statsCopy.ModuleStats[k] = &moduleCopy
	}

	// Add cache statistics
	cacheStats := m.cache.GetStats()
	statsCopy.CurrentMemoryUsage = cacheStats.MemoryUsage

	return &statsCopy
}

// Close shuts down the pre-execution manager and releases resources
func (m *PreExecutionManager) Close() {
	// Stop cleanup goroutine
	if m.cleanupTicker != nil {
		m.cleanupTicker.Stop()
		close(m.stopCleanup)
	}

	// Close cache
	if m.cache != nil {
		m.cache.Close()
	}
}

// Helper methods

func (m *PreExecutionManager) validateCachedResult(
	ctx keepertypes.Context,
	result *types.TxPreExecResult,
) bool {
	return m.ValidateCachedResult(ctx, result) == nil
}

func (m *PreExecutionManager) isModuleEnabled(moduleName string) bool {
	module, exists := m.modules[moduleName]
	if !exists {
		return false
	}

	config := module.GetPreExecConfig()
	return config != nil && config.Enabled
}

func (m *PreExecutionManager) createPreExecutionContext(
	ctx keepertypes.Context,
) (types.PreExecutionContext, error) {
	// TODO: Implement pre-execution context with state isolation
	// This would wrap the regular context with tracking capabilities
	// For now, return nil as placeholder
	return nil, fmt.Errorf("pre-execution context not implemented")
}

func (m *PreExecutionManager) mergeStateChanges(
	base, additional *types.StateChangeSet,
) *types.StateChangeSet {
	if base == nil {
		return additional
	}
	if additional == nil {
		return base
	}

	merged := &types.StateChangeSet{
		StoreChanges: make(map[string]*types.StoreChangeSet),
		Version:      base.Version,
	}

	// Copy base changes
	for storeName, storeChanges := range base.StoreChanges {
		merged.StoreChanges[storeName] = &types.StoreChangeSet{
			Sets:    make(map[string][]byte),
			Deletes: make([]string, len(storeChanges.Deletes)),
		}
		for k, v := range storeChanges.Sets {
			merged.StoreChanges[storeName].Sets[k] = v
		}
		copy(merged.StoreChanges[storeName].Deletes, storeChanges.Deletes)
	}

	// Merge additional changes
	for storeName, storeChanges := range additional.StoreChanges {
		if merged.StoreChanges[storeName] == nil {
			merged.StoreChanges[storeName] = &types.StoreChangeSet{
				Sets:    make(map[string][]byte),
				Deletes: make([]string, 0),
			}
		}

		// Merge sets (additional overwrites base)
		for k, v := range storeChanges.Sets {
			merged.StoreChanges[storeName].Sets[k] = v
		}

		// Merge deletes
		merged.StoreChanges[storeName].Deletes = append(
			merged.StoreChanges[storeName].Deletes,
			storeChanges.Deletes...)
	}

	return merged
}

func (m *PreExecutionManager) calculateTransactionPriority(msgs []types.PreExecutableMsg) uint32 {
	var totalPriority uint32
	for _, msg := range msgs {
		hints := msg.GetPreExecutionHints()
		if hints != nil {
			totalPriority += hints.Priority
		}
	}
	return totalPriority / uint32(len(msgs)) // Average priority
}

func (m *PreExecutionManager) updateStats(updater func(*PreExecStats)) {
	m.stats.mu.Lock()
	defer m.stats.mu.Unlock()
	updater(m.stats)
}

func (m *PreExecutionManager) updateModuleStats(
	moduleName string,
	success bool,
	gasUsed uint64,
	execTime time.Duration,
) {
	m.stats.mu.Lock()
	defer m.stats.mu.Unlock()

	moduleStats := m.stats.ModuleStats[moduleName]
	if moduleStats == nil {
		moduleStats = &ModuleStats{}
		m.stats.ModuleStats[moduleName] = moduleStats
	}

	moduleStats.Attempts++
	if success {
		moduleStats.Successes++

		// Update average gas usage
		if moduleStats.Successes == 1 {
			moduleStats.AverageGasUsed = gasUsed
		} else {
			moduleStats.AverageGasUsed = (moduleStats.AverageGasUsed*(moduleStats.Successes-1) + gasUsed) / moduleStats.Successes
		}

		// Update average execution time
		if moduleStats.Successes == 1 {
			moduleStats.AverageExecTime = execTime
		} else {
			moduleStats.AverageExecTime = time.Duration(
				(int64(moduleStats.AverageExecTime)*int64(moduleStats.Successes-1) + int64(execTime)) /
					int64(moduleStats.Successes))
		}
	} else {
		moduleStats.Failures++
	}
}

func (m *PreExecutionManager) runPeriodicCleanup() {
	ticker := m.cleanupTicker
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Process pending orders
			m.sequencer.ProcessPendingOrders()

			// Reset gas tracker if needed
			m.gasTracker.ResetIfNeeded()

		case <-m.stopCleanup:
			return
		}
	}
}

// Helper functions

func getModuleFromMsg(msg types.PreExecutableMsg) string {
	// TODO: Implement logic to determine module name from message type
	// This would typically involve type reflection or a message registry
	return "unknown"
}

func getMessageType(msg types.PreExecutableMsg) string {
	// TODO: Implement logic to get message type name
	return "unknown"
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// GasTracker methods

func (g *GasTracker) CanUseGas(amount uint64) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.resetIfNeeded()
	return g.totalGasUsed+amount <= g.maxTotalGas
}

func (g *GasTracker) RecordGasUsage(amount uint64) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.resetIfNeeded()
	g.totalGasUsed += amount
}

func (g *GasTracker) ResetIfNeeded() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.resetIfNeeded()
}

func (g *GasTracker) resetIfNeeded() {
	if time.Since(g.lastReset) >= g.resetInterval {
		g.totalGasUsed = 0
		g.lastReset = time.Now()
	}
}
