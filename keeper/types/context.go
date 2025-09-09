package types

import (
	"github.com/b2network/pulsar/store/types"
)

// PulsarContext implements the Context interface
type PulsarContext struct {
	multiStore   types.MultiStore
	blockHeight  int64
	chainID      string
	logger       Logger
	eventManager EventManager
}

// NewPulsarContext creates a new PulsarContext
func NewPulsarContext(
	multiStore types.MultiStore,
	blockHeight int64,
	chainID string,
	logger Logger,
) *PulsarContext {
	return &PulsarContext{
		multiStore:   multiStore,
		blockHeight:  blockHeight,
		chainID:      chainID,
		logger:       logger,
		eventManager: NewEventManager(),
	}
}

// KVStore returns the KVStore for the given key
func (ctx *PulsarContext) KVStore(key types.StoreKey) types.KVStore {
	return ctx.multiStore.GetKVStore(key)
}

// MultiStore returns the multi store
func (ctx *PulsarContext) MultiStore() types.MultiStore {
	return ctx.multiStore
}

// BlockHeight returns the current block height
func (ctx *PulsarContext) BlockHeight() int64 {
	return ctx.blockHeight
}

// ChainID returns the chain ID
func (ctx *PulsarContext) ChainID() string {
	return ctx.chainID
}

// Logger returns a logger for this context
func (ctx *PulsarContext) Logger() Logger {
	return ctx.logger
}

// EventManager returns the event manager
func (ctx *PulsarContext) EventManager() EventManager {
	return ctx.eventManager
}

// WithLogger returns a new context with the given logger
func (ctx *PulsarContext) WithLogger(logger Logger) Context {
	newCtx := *ctx
	newCtx.logger = logger
	return &newCtx
}

// WithEventManager returns a new context with the given event manager
func (ctx *PulsarContext) WithEventManager(em EventManager) Context {
	newCtx := *ctx
	newCtx.eventManager = em
	return &newCtx
}

// WithBlockHeight returns a new context with the given block height
func (ctx *PulsarContext) WithBlockHeight(height int64) Context {
	newCtx := *ctx
	newCtx.blockHeight = height
	return &newCtx
}

// WithChainID returns a new context with the given chain ID
func (ctx *PulsarContext) WithChainID(chainID string) Context {
	newCtx := *ctx
	newCtx.chainID = chainID
	return &newCtx
}

// BasicEventManager implements EventManager interface
type BasicEventManager struct {
	events Events
}

// NewEventManager creates a new BasicEventManager
func NewEventManager() EventManager {
	return &BasicEventManager{
		events: make(Events, 0),
	}
}

// EmitEvent emits a single event
func (em *BasicEventManager) EmitEvent(event Event) {
	em.events = append(em.events, event)
}

// EmitEvents emits multiple events
func (em *BasicEventManager) EmitEvents(events Events) {
	em.events = append(em.events, events...)
}

// Events returns all emitted events
func (em *BasicEventManager) Events() Events {
	return em.events
}

// Clear clears all events
func (em *BasicEventManager) Clear() {
	em.events = em.events[:0]
}

// BasicLogger implements Logger interface
type BasicLogger struct {
	prefix string
}

// NewBasicLogger creates a new BasicLogger
func NewBasicLogger(prefix string) Logger {
	return &BasicLogger{prefix: prefix}
}

// Debug logs a debug message
func (l *BasicLogger) Debug(msg string, keyvals ...interface{}) {
	// In a real implementation, this would use a proper logging library
}

// Info logs an info message
func (l *BasicLogger) Info(msg string, keyvals ...interface{}) {
	// In a real implementation, this would use a proper logging library
}

// Error logs an error message
func (l *BasicLogger) Error(msg string, keyvals ...interface{}) {
	// In a real implementation, this would use a proper logging library
}

// BasicAuthority implements Authority interface
type BasicAuthority struct {
	authorizedAddresses map[string]bool
}

// NewBasicAuthority creates a new BasicAuthority
func NewBasicAuthority(addresses []string) Authority {
	auth := &BasicAuthority{
		authorizedAddresses: make(map[string]bool),
	}

	for _, addr := range addresses {
		auth.authorizedAddresses[addr] = true
	}

	return auth
}

// GetAuthority returns the authority address (returns first authorized address)
func (auth *BasicAuthority) GetAuthority() string {
	for addr := range auth.authorizedAddresses {
		return addr
	}
	return ""
}

// HasAuthority checks if the given address has authority
func (auth *BasicAuthority) HasAuthority(address string) bool {
	return auth.authorizedAddresses[address]
}

// AddAuthority adds an address to the authority list
func (auth *BasicAuthority) AddAuthority(address string) {
	auth.authorizedAddresses[address] = true
}

// RemoveAuthority removes an address from the authority list
func (auth *BasicAuthority) RemoveAuthority(address string) {
	delete(auth.authorizedAddresses, address)
}
