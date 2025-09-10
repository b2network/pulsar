package keeper

import (
	"time"

	keepertypes "github.com/b2network/pulsar/keeper/types"
	preexectypes "github.com/b2network/pulsar/pre_execution/types"
	storetypes "github.com/b2network/pulsar/store/types"
)

// PreExecutionContext implements the PreExecutionContext interface
// It wraps a regular context with additional state tracking capabilities
type PreExecutionContext struct {
	keepertypes.Context

	// State tracking
	readSet  []preexectypes.StateRead
	writeSet []preexectypes.StateWrite
	snapshot *preexectypes.StateSnapshot

	// Pre-execution metadata
	isPreExecution bool
	txHash         string

	// Store wrappers for state tracking
	stores map[storetypes.StoreKey]*TrackedStore
}

// NewPreExecutionContext creates a new pre-execution context wrapper
func NewPreExecutionContext(ctx keepertypes.Context, txHash string) *PreExecutionContext {
	preCtx := &PreExecutionContext{
		Context:        ctx,
		readSet:        make([]preexectypes.StateRead, 0),
		writeSet:       make([]preexectypes.StateWrite, 0),
		isPreExecution: true,
		txHash:         txHash,
		stores:         make(map[storetypes.StoreKey]*TrackedStore),
	}

	// Create state snapshot
	preCtx.snapshot = preCtx.createStateSnapshot()

	return preCtx
}

// GetStateSnapshot returns a snapshot of the current state
func (ctx *PreExecutionContext) GetStateSnapshot() *preexectypes.StateSnapshot {
	return ctx.snapshot
}

// TrackStateRead records a state read operation
func (ctx *PreExecutionContext) TrackStateRead(storeKey string, key, value []byte) {
	read := preexectypes.StateRead{
		StoreKey: storeKey,
		Key:      make([]byte, len(key)),
		Value:    make([]byte, len(value)),
		Version:  ctx.BlockHeight(),
	}

	copy(read.Key, key)
	copy(read.Value, value)

	ctx.readSet = append(ctx.readSet, read)
}

// TrackStateWrite records a state write operation
func (ctx *PreExecutionContext) TrackStateWrite(storeKey string, key, value []byte, isDelete bool) {
	write := preexectypes.StateWrite{
		StoreKey: storeKey,
		Key:      make([]byte, len(key)),
		IsDelete: isDelete,
	}

	copy(write.Key, key)

	if !isDelete && value != nil {
		write.Value = make([]byte, len(value))
		copy(write.Value, value)
	}

	ctx.writeSet = append(ctx.writeSet, write)
}

// GetReadSet returns all state reads performed in this context
func (ctx *PreExecutionContext) GetReadSet() []preexectypes.StateRead {
	result := make([]preexectypes.StateRead, len(ctx.readSet))
	copy(result, ctx.readSet)
	return result
}

// GetWriteSet returns all state writes performed in this context
func (ctx *PreExecutionContext) GetWriteSet() []preexectypes.StateWrite {
	result := make([]preexectypes.StateWrite, len(ctx.writeSet))
	copy(result, ctx.writeSet)
	return result
}

// IsPreExecution returns true if this is a pre-execution context
func (ctx *PreExecutionContext) IsPreExecution() bool {
	return ctx.isPreExecution
}

// KVStore returns a tracked KVStore that records read/write operations
func (ctx *PreExecutionContext) KVStore(key storetypes.StoreKey) storetypes.KVStore {
	if trackedStore, exists := ctx.stores[key]; exists {
		return trackedStore
	}

	// Get the original store from the underlying context
	originalStore := ctx.Context.KVStore(key)

	// Wrap it with tracking capabilities
	trackedStore := &TrackedStore{
		KVStore:  originalStore,
		context:  ctx,
		storeKey: key.Name(),
	}

	ctx.stores[key] = trackedStore
	return trackedStore
}

// createStateSnapshot creates a snapshot of the current blockchain state
func (ctx *PreExecutionContext) createStateSnapshot() *preexectypes.StateSnapshot {
	// This is a simplified implementation
	// In production, this would create a more comprehensive snapshot
	return &preexectypes.StateSnapshot{
		Version:     ctx.BlockHeight(),
		StoreStates: make(map[string]map[string][]byte),
		Timestamp:   time.Now(),
	}
}

// TrackedStore wraps a KVStore to track read and write operations
type TrackedStore struct {
	storetypes.KVStore
	context  *PreExecutionContext
	storeKey string
}

// Get reads a value and tracks the read operation
func (ts *TrackedStore) Get(key []byte) []byte {
	value := ts.KVStore.Get(key)

	// Track the read operation
	ts.context.TrackStateRead(ts.storeKey, key, value)

	return value
}

// Has checks if a key exists and tracks the read operation
func (ts *TrackedStore) Has(key []byte) bool {
	exists := ts.KVStore.Has(key)

	// Track the read operation (with nil value for existence check)
	var value []byte
	if exists {
		value = []byte{1} // Indicate existence
	} else {
		value = []byte{0} // Indicate non-existence
	}
	ts.context.TrackStateRead(ts.storeKey, key, value)

	return exists
}

// Set writes a value and tracks the write operation
func (ts *TrackedStore) Set(key, value []byte) {
	ts.KVStore.Set(key, value)

	// Track the write operation
	ts.context.TrackStateWrite(ts.storeKey, key, value, false)
}

// Delete removes a key and tracks the delete operation
func (ts *TrackedStore) Delete(key []byte) {
	ts.KVStore.Delete(key)

	// Track the delete operation
	ts.context.TrackStateWrite(ts.storeKey, key, nil, true)
}

// Iterator returns a tracked iterator that records all read operations
func (ts *TrackedStore) Iterator(start, end []byte) storetypes.Iterator {
	iter := ts.KVStore.Iterator(start, end)
	return &TrackedIterator{
		Iterator: iter,
		store:    ts,
	}
}

// ReverseIterator returns a tracked reverse iterator
func (ts *TrackedStore) ReverseIterator(start, end []byte) storetypes.Iterator {
	iter := ts.KVStore.ReverseIterator(start, end)
	return &TrackedIterator{
		Iterator: iter,
		store:    ts,
	}
}

// TrackedIterator wraps an iterator to track read operations
type TrackedIterator struct {
	storetypes.Iterator
	store *TrackedStore
}

// Value returns the current value and tracks the read
func (ti *TrackedIterator) Value() []byte {
	value := ti.Iterator.Value()
	key := ti.Iterator.Key()

	// Track the read operation
	ti.store.context.TrackStateRead(ti.store.storeKey, key, value)

	return value
}

// Key returns the current key (no tracking needed for key access)
func (ti *TrackedIterator) Key() []byte {
	key := ti.Iterator.Key()

	// Optionally track key access as well
	// ti.store.context.TrackStateRead(ti.store.storeKey, key, []byte("key_access"))

	return key
}

// PreExecutionContextWithLogger creates a pre-execution context with enhanced logging
type PreExecutionContextWithLogger struct {
	*PreExecutionContext
	logger keepertypes.Logger
}

// NewPreExecutionContextWithLogger creates a pre-execution context with logging capabilities
func NewPreExecutionContextWithLogger(ctx keepertypes.Context, txHash string, logger keepertypes.Logger) *PreExecutionContextWithLogger {
	preCtx := NewPreExecutionContext(ctx, txHash)

	return &PreExecutionContextWithLogger{
		PreExecutionContext: preCtx,
		logger:              logger,
	}
}

// Logger returns the context logger
func (ctx *PreExecutionContextWithLogger) Logger() keepertypes.Logger {
	return ctx.logger
}

// LogStateRead logs state read operations for debugging
func (ctx *PreExecutionContextWithLogger) TrackStateRead(storeKey string, key, value []byte) {
	ctx.PreExecutionContext.TrackStateRead(storeKey, key, value)

	ctx.logger.Debug("Pre-execution state read",
		"store", storeKey,
		"key", string(key),
		"value_len", len(value),
		"tx_hash", ctx.txHash,
	)
}

// LogStateWrite logs state write operations for debugging
func (ctx *PreExecutionContextWithLogger) TrackStateWrite(storeKey string, key, value []byte, isDelete bool) {
	ctx.PreExecutionContext.TrackStateWrite(storeKey, key, value, isDelete)

	operation := "set"
	if isDelete {
		operation = "delete"
	}

	ctx.logger.Debug("Pre-execution state write",
		"store", storeKey,
		"key", string(key),
		"operation", operation,
		"value_len", len(value),
		"tx_hash", ctx.txHash,
	)
}

// GetTxHash returns the transaction hash associated with this context
func (ctx *PreExecutionContext) GetTxHash() string {
	return ctx.txHash
}

// GetReadSetSummary returns a summary of read operations for logging
func (ctx *PreExecutionContext) GetReadSetSummary() map[string]int {
	summary := make(map[string]int)
	for _, read := range ctx.readSet {
		summary[read.StoreKey]++
	}
	return summary
}

// GetWriteSetSummary returns a summary of write operations for logging
func (ctx *PreExecutionContext) GetWriteSetSummary() map[string]int {
	summary := make(map[string]int)
	for _, write := range ctx.writeSet {
		summary[write.StoreKey]++
	}
	return summary
}

// Reset clears all tracked state operations (useful for reusing context)
func (ctx *PreExecutionContext) Reset(txHash string) {
	ctx.readSet = ctx.readSet[:0]   // Clear slice but keep capacity
	ctx.writeSet = ctx.writeSet[:0] // Clear slice but keep capacity
	ctx.txHash = txHash
	ctx.stores = make(map[storetypes.StoreKey]*TrackedStore) // Reset store wrappers
	ctx.snapshot = ctx.createStateSnapshot()                 // Create new snapshot
}
