package types

import (
	"time"

	keepertypes "github.com/b2network/pulsar/keeper/types"
)

// PreExecutableMsg defines the interface for messages that support pre-execution
// This interface extends the base Msg interface with pre-execution capabilities
type PreExecutableMsg interface {
	// IsPreExecutable returns true if this message type supports pre-execution
	// Modules can implement custom logic to determine pre-execution eligibility
	IsPreExecutable() bool

	// GetPreExecutionHints provides optimization hints for pre-execution
	// These hints help the pre-execution engine optimize performance and resource usage
	GetPreExecutionHints() *PreExecHints

	// ValidateBasic performs basic validation checks on the message
	ValidateBasic() error

	// GetSigners returns the addresses that must sign this message
	GetSigners() [][]byte
}

// PreExecHints provides optimization hints and constraints for pre-execution
type PreExecHints struct {
	// Priority determines execution order (higher values = higher priority)
	Priority uint32 `json:"priority"`

	// MaxGasForPreExec sets the maximum gas limit for pre-execution
	// This prevents resource exhaustion during pre-execution
	MaxGasForPreExec uint64 `json:"max_gas_for_pre_exec"`

	// RequiresOrdering indicates if this transaction requires strict ordering
	// Transactions with strict ordering are executed in mempool order
	RequiresOrdering bool `json:"requires_ordering"`

	// CacheDuration specifies how long pre-execution results should be cached
	// Longer durations reduce re-execution but increase memory usage
	CacheDuration time.Duration `json:"cache_duration"`

	// EstimatedGasUsage provides an estimate of gas consumption
	// This helps with resource planning and scheduling
	EstimatedGasUsage uint64 `json:"estimated_gas_usage"`

	// StateReadOnly indicates if this transaction only reads state (no writes)
	// Read-only transactions can be optimized and cached more aggressively
	StateReadOnly bool `json:"state_read_only"`
}

// PreExecutableModule defines the interface for modules that support pre-execution
// Modules implement this interface to provide pre-execution capabilities
type PreExecutableModule interface {
	// GetModuleName returns the unique name of this module
	GetModuleName() string

	// PreExecuteMsg executes a message in pre-execution mode
	// This should perform the same logic as normal execution but with state isolation
	PreExecuteMsg(ctx keepertypes.Context, msg PreExecutableMsg) (*PreExecResult, error)

	// ValidatePreExecResult validates a cached pre-execution result is still valid
	// This checks if the underlying state has changed since pre-execution
	ValidatePreExecResult(ctx keepertypes.Context, result *PreExecResult) error

	// ApplyCachedResult applies a validated pre-execution result to the current state
	// This efficiently applies pre-computed state changes during block execution
	ApplyCachedResult(ctx keepertypes.Context, result *PreExecResult) error

	// GetPreExecConfig returns the module's pre-execution configuration
	GetPreExecConfig() *ModulePreExecConfig
}

// PreExecResult contains the results of pre-executing a transaction
type PreExecResult struct {
	// TxHash uniquely identifies the transaction
	TxHash string `json:"tx_hash"`

	// MsgIndex identifies which message in the transaction this result is for
	MsgIndex uint32 `json:"msg_index"`

	// Success indicates if pre-execution completed successfully
	Success bool `json:"success"`

	// Code contains the execution result code (0 for success)
	Code uint32 `json:"code"`

	// Log contains execution log messages
	Log string `json:"log"`

	// GasUsed records the amount of gas consumed during pre-execution
	GasUsed uint64 `json:"gas_used"`

	// Events contains the events emitted during pre-execution
	Events []keepertypes.Event `json:"events"`

	// StateChanges contains all state modifications made during pre-execution
	// This allows efficient application of changes during block execution
	StateChanges *StateChangeSet `json:"state_changes"`

	// ReadSet contains all state keys read during execution
	// This is used for conflict detection and validation
	ReadSet []StateRead `json:"read_set"`

	// WriteSet contains all state keys written during execution
	// This is used for conflict detection and state application
	WriteSet []StateWrite `json:"write_set"`

	// Timestamp records when pre-execution occurred
	Timestamp time.Time `json:"timestamp"`

	// ValidUntil specifies when this result expires
	ValidUntil time.Time `json:"valid_until"`
}

// StateChangeSet represents a collection of state changes from pre-execution
type StateChangeSet struct {
	// StoreChanges maps store names to their respective changes
	StoreChanges map[string]*StoreChangeSet `json:"store_changes"`

	// Version is the state version when these changes were made
	Version int64 `json:"version"`
}

// StoreChangeSet represents changes for a specific store
type StoreChangeSet struct {
	// Sets contains key-value pairs to be set
	Sets map[string][]byte `json:"sets"`

	// Deletes contains keys to be deleted
	Deletes []string `json:"deletes"`
}

// StateRead represents a state read operation during pre-execution
type StateRead struct {
	// StoreKey identifies which store was read
	StoreKey string `json:"store_key"`

	// Key is the specific key that was read
	Key []byte `json:"key"`

	// Value is the value that was read (for validation)
	Value []byte `json:"value"`

	// Version is the state version when the read occurred
	Version int64 `json:"version"`
}

// StateWrite represents a state write operation during pre-execution
type StateWrite struct {
	// StoreKey identifies which store was written to
	StoreKey string `json:"store_key"`

	// Key is the specific key that was written
	Key []byte `json:"key"`

	// Value is the new value (nil for deletions)
	Value []byte `json:"value"`

	// IsDelete indicates if this is a deletion operation
	IsDelete bool `json:"is_delete"`
}

// ModulePreExecConfig contains configuration for a module's pre-execution behavior
type ModulePreExecConfig struct {
	// Enabled determines if pre-execution is enabled for this module
	Enabled bool `json:"enabled"`

	// SupportedMsgTypes lists the message types that support pre-execution
	SupportedMsgTypes []string `json:"supported_msg_types"`

	// MaxPreExecPerBlock limits the number of pre-executions per block
	MaxPreExecPerBlock uint32 `json:"max_pre_exec_per_block"`

	// MaxGasPerPreExec limits gas usage for individual pre-executions
	MaxGasPerPreExec uint64 `json:"max_gas_per_pre_exec"`

	// StateReadOnlyKeys lists state keys that are frequently read
	// These can be cached more aggressively for performance
	StateReadOnlyKeys []string `json:"state_read_only_keys"`

	// CachePolicy defines how pre-execution results should be cached
	CachePolicy CachePolicy `json:"cache_policy"`
}

// CachePolicy defines caching behavior for pre-execution results
type CachePolicy struct {
	// MaxSize limits the number of cached results
	MaxSize int `json:"max_size"`

	// TTL defines how long results remain valid
	TTL time.Duration `json:"ttl"`

	// EvictionPolicy determines how to remove old entries ("LRU", "LFU", "FIFO")
	EvictionPolicy string `json:"eviction_policy"`

	// CompressionEnabled determines if cached results should be compressed
	CompressionEnabled bool `json:"compression_enabled"`
}

// PreExecutionContext provides an isolated execution context for pre-execution
// This wraps the regular context with additional tracking capabilities
type PreExecutionContext interface {
	keepertypes.Context

	// GetStateSnapshot returns a snapshot of the current state
	GetStateSnapshot() *StateSnapshot

	// TrackStateRead records a state read operation
	TrackStateRead(storeKey string, key, value []byte)

	// TrackStateWrite records a state write operation
	TrackStateWrite(storeKey string, key, value []byte, isDelete bool)

	// GetReadSet returns all state reads performed in this context
	GetReadSet() []StateRead

	// GetWriteSet returns all state writes performed in this context
	GetWriteSet() []StateWrite

	// IsPreExecution returns true if this is a pre-execution context
	IsPreExecution() bool
}

// StateSnapshot represents a point-in-time snapshot of blockchain state
type StateSnapshot struct {
	// Version is the state version of this snapshot
	Version int64 `json:"version"`

	// StoreStates maps store names to their state data
	StoreStates map[string]map[string][]byte `json:"store_states"`

	// Timestamp records when this snapshot was created
	Timestamp time.Time `json:"timestamp"`
}

// TxPreExecResult contains the complete pre-execution results for a transaction
type TxPreExecResult struct {
	// TxHash uniquely identifies the transaction
	TxHash string `json:"tx_hash"`

	// MsgResults contains results for each message in the transaction
	MsgResults []*PreExecResult `json:"msg_results"`

	// TotalGasUsed is the sum of gas used by all messages
	TotalGasUsed uint64 `json:"total_gas_used"`

	// Success indicates if all messages executed successfully
	Success bool `json:"success"`

	// StateChanges contains all state changes from the transaction
	StateChanges *StateChangeSet `json:"state_changes"`

	// Priority is the execution priority for this transaction
	Priority uint32 `json:"priority"`

	// Sequence is the execution order sequence number
	Sequence uint32 `json:"sequence"`

	// PreExecTime records when pre-execution occurred
	PreExecTime time.Time `json:"pre_exec_time"`

	// ValidUntil specifies when this result expires
	ValidUntil time.Time `json:"valid_until"`
}

// PreExecutionOrder represents an order message for transaction sequencing
// This is broadcast to other nodes to maintain consistent execution order
type PreExecutionOrder struct {
	// BaseBlock is the reference block height (N)
	BaseBlock int64 `json:"base_block"`

	// TxHash uniquely identifies the transaction
	TxHash string `json:"tx_hash"`

	// Sequence is the order number for this transaction after BaseBlock
	Sequence uint32 `json:"sequence"`

	// Timestamp records when this order was created
	Timestamp time.Time `json:"timestamp"`

	// ValidatorID identifies the validator that created this order
	ValidatorID string `json:"validator_id"`

	// Priority can be used for order resolution in case of conflicts
	Priority uint32 `json:"priority"`
}
