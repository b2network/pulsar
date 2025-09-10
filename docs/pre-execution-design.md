# Pre-execution Transaction Design and Implementation Plan

## Table of Contents
1. [Overview](#overview)
2. [Architecture Design](#architecture-design)
3. [Core Components](#core-components)
4. [Module Integration](#module-integration)
5. [Implementation Plan](#implementation-plan)
6. [Technical Specifications](#technical-specifications)
7. [Configuration](#configuration)
8. [Testing Strategy](#testing-strategy)

## Overview

### Background
Pulsar utilizes an advanced versioned state storage system that maintains multiple versions of blockchain state data, enabling efficient state management and rollback capabilities. Traditional blockchain transaction throughput (TPS) is limited by block time - users must wait for the next block to be produced (typically 2-3 seconds) before their transactions are confirmed. To improve transaction TPS and confirmation time, we propose a pre-execution transaction mechanism that leverages our versioned storage architecture to provide instant transaction feedback while maintaining security and consistency.

### Core Concept
- **Fast Transaction Submission**: Transactions are submitted to mempool
- **Immediate Pre-execution**: Next block's validator executes transactions immediately upon receipt
- **Instant Result Return**: Users receive execution results without waiting for block confirmation
- **State Application**: Pre-executed states are directly written to store during block packing

### Key Benefits
- Reduce transaction confirmation time from seconds to milliseconds
- Improve user experience with instant feedback
- Increase effective TPS without changing consensus mechanism
- Maintain complete state consistency through serial execution

## Architecture Design

### Design Principles

1. **Modular Extensibility**: Each module independently decides pre-execution support
2. **Serial Execution**: Maintain transaction serial execution to avoid concurrency conflicts
3. **Backward Compatibility**: Pre-execution is an optional feature that doesn't affect existing functionality
4. **State Consistency**: Leverage our advanced versioned storage system to ensure correct state management

### System Architecture

```
User → Submit Tx → Mempool → Validator Pre-execution → Return Result
                                    ↓
                            Cache Execution State
                                    ↓
                        Apply Cached Results in Block
```

### Transaction States

```go
type TxStatus int

const (
    TxStatusPending      TxStatus = iota  // In mempool waiting
    TxStatusPreConfirm                    // Pre-executed (new state)
    TxStatusConfirmed                     // On-chain confirmed
    TxStatusFailed                        // Execution failed
)
```

## Core Components

### 1. Pre-execution Manager

```go
type PreExecutionManager struct {
    modules         map[string]PreExecutableModule
    cache           *PreExecutionCache
    sequencer       *TxSequencer
    config          *GlobalPreExecConfig
    stateStore      types.MultiStore
}
```

**Responsibilities:**
- Determine if transactions should be pre-executed
- Manage pre-execution cache
- Coordinate module pre-execution
- Handle state snapshots and rollbacks

### 2. Pre-execution Cache

```go
type PreExecutionCache struct {
    mu              sync.RWMutex
    baseHeight      int64
    executedTxs     map[string]*CachedExecution
    orderedList     []string
    stateChanges    *BufferedStateChanges
    maxSize         int
}

type CachedExecution struct {
    TxHash          string
    Sequence        uint32
    Result          *ExecutionResult
    StateRoot       []byte
    GasUsed         uint64
    Events          []Event
}
```

**Features:**
- LRU cache strategy
- Automatic cleanup after block confirmation
- Memory usage limits
- State change buffering

### 3. Transaction Sequencer

```go
type PreExecutionSequence struct {
    baseHeight   int64
    sequences    map[string]uint32
    orderedTxs   []string
    nextSeq      uint32
}

type PreExecutionOrder struct {
    BaseBlock    int64
    TxHash       string
    Sequence     uint32
    Timestamp    time.Time
    ValidatorID  string
}
```

**Purpose:**
- Maintain transaction execution order
- Synchronize order across nodes
- Ensure deterministic execution

## Module Integration

### Extensible Module Interface

```go
// Base message interface with pre-execution support
type PreExecutableMsg interface {
    Msg
    IsPreExecutable() bool
    GetPreExecutionHints() *ExecHints
}

// Module interface with pre-execution capabilities
type PreExecutableModule interface {
    Module
    
    // Pre-execution methods
    PreExecuteMsg(ctx Context, msg Msg) (*PreExecResult, error)
    ValidatePreExecResult(ctx Context, result *PreExecResult) error
    ApplyCachedResult(ctx Context, result *PreExecResult) error
    
    // Configuration
    GetPreExecConfig() *ModulePreExecConfig
}

// Pre-execution hints for optimization
type ExecHints struct {
    Priority        uint32
    MaxGasForPreExec uint64
    RequiresOrdering bool
    CacheDuration   time.Duration
}
```

### Bank Module Implementation

```go
// bank/types/msgs.go
type MsgSend struct {
    FromAddress string
    ToAddress   string
    Amount      Coins
    PreExecute  bool  // User-selectable pre-execution
}

func (msg *MsgSend) IsPreExecutable() bool {
    return msg.PreExecute && msg.Amount.IsValid()
}

func (msg *MsgSend) GetPreExecutionHints() *ExecHints {
    return &ExecHints{
        Priority:         10,  // High priority for transfers
        MaxGasForPreExec: 100000,
        RequiresOrdering: false,  // Simple transfers don't need strict ordering
        CacheDuration:    30 * time.Second,
    }
}

// bank/keeper/pre_execution.go
func (k Keeper) PreExecuteMsg(ctx Context, msg Msg) (*PreExecResult, error) {
    switch msg := msg.(type) {
    case *MsgSend:
        return k.preExecuteSend(ctx, msg)
    case *MsgMultiSend:
        return k.preExecuteMultiSend(ctx, msg)
    default:
        return nil, ErrMsgTypeNotSupported
    }
}
```

### Staking Module Implementation

```go
// staking/types/msgs.go
type MsgDelegate struct {
    DelegatorAddress string
    ValidatorAddress string
    Amount           Coin
    PreExecute       bool
}

func (msg *MsgDelegate) IsPreExecutable() bool {
    return msg.PreExecute && msg.Amount.IsPositive()
}

func (msg *MsgDelegate) GetPreExecutionHints() *ExecHints {
    return &ExecHints{
        Priority:         5,  // Medium priority
        MaxGasForPreExec: 200000,
        RequiresOrdering: true,  // Delegation needs order guarantee
        CacheDuration:    10 * time.Second,
    }
}
```

### Governance Module Implementation

```go
// gov/types/msgs.go
type MsgVote struct {
    ProposalID uint64
    Voter      string
    Option     VoteOption
    PreExecute bool
}

func (msg *MsgVote) IsPreExecutable() bool {
    return msg.PreExecute  // Voting can be pre-executed
}

type MsgSubmitProposal struct {
    Content   Content
    Proposer  string
    PreExecute bool
}

func (msg *MsgSubmitProposal) IsPreExecutable() bool {
    return false  // Proposal submission typically doesn't need pre-execution
}
```

## Implementation Plan

### Phase 1: Core Framework

#### Directory Structure
```
pre_execution/
├── types/
│   ├── interfaces.go      # Pre-execution interfaces
│   ├── msgs.go            # Pre-execution message types
│   ├── cache.go           # Cache data structures
│   └── errors.go          # Error definitions
├── keeper/
│   ├── keeper.go          # Pre-execution manager
│   ├── cache.go           # Cache management
│   └── sequencer.go       # Transaction sequencing
└── config/
    └── config.go          # Pre-execution configuration
```

#### Key Tasks
- Define pre-execution interfaces
- Implement pre-execution cache with versioned storage integration
- Create transaction sequencer
- Build pre-execution manager

### Phase 2: Module Integration

#### Bank Module
```
modules/bank/
├── types/
│   └── msgs_preexec.go    # Pre-execution message extensions
├── keeper/
│   └── pre_execution.go   # Pre-execution logic
└── preexec/
    └── handler.go         # Pre-execution handler
```

#### Staking Module
```
modules/staking/
├── types/
│   └── msgs_preexec.go
├── keeper/
│   └── pre_execution.go
└── preexec/
    └── handler.go
```

#### Governance Module
```
modules/gov/
├── types/
│   └── msgs_preexec.go
├── keeper/
│   └── pre_execution.go
└── preexec/
    └── handler.go
```

### Phase 3: App Layer Integration

#### Modify app/app.go
```go
type PulsarApp struct {
    // Existing fields...
    
    // Pre-execution additions
    preExecManager  *preexecution.Manager
    preExecCache    *preexecution.Cache
    preExecConfig   *preexecution.Config
}
```

#### CheckTx Integration
```go
func (app *PulsarApp) CheckTx(ctx context.Context, req *RequestCheckTx) (*ResponseCheckTx, error) {
    tx, err := app.txDecoder(req.Tx)
    if err != nil {
        return &ResponseCheckTx{Code: 1}, err
    }
    
    // Check if pre-execution is needed
    if app.preExecManager.ShouldPreExecute(tx) {
        result, err := app.preExecManager.PreExecuteTx(app.ctx, tx)
        if err == nil {
            // Cache pre-execution result
            app.preExecManager.CacheResult(result)
            
            // Return pre-execution status
            return &ResponseCheckTx{
                Code: 0,
                Data: result.Encode(),
                Info: "pre-executed",
                GasUsed: result.GasUsed,
            }, nil
        }
    }
    
    // Regular check
    return &ResponseCheckTx{Code: 0}, nil
}
```

#### PrepareProposal Optimization
```go
func (app *PulsarApp) PrepareProposal(ctx context.Context, req *RequestPrepareProposal) (*ResponsePrepareProposal, error) {
    // Use pre-executed transactions from cache
    preExecutedTxs := app.preExecCache.GetOrderedTransactions()
    
    // Apply cached results
    for _, cachedTx := range preExecutedTxs {
        if app.preExecManager.ValidateCachedResult(app.ctx, cachedTx) {
            // Use cached execution result
            app.preExecManager.ApplyCachedResult(app.ctx, cachedTx)
        }
    }
    
    return &ResponsePrepareProposal{
        Txs: preExecutedTxs,
    }, nil
}
```

### Phase 4: P2P Synchronization

#### Message Definitions
```protobuf
// p2p/types/pre_exec_msgs.proto
message PreExecutionOrder {
    int64  base_block = 1;
    string tx_hash = 2;
    uint32 sequence = 3;
    int64  timestamp = 4;
    string validator_id = 5;
}

message PreExecutionSync {
    repeated PreExecutionOrder orders = 1;
}
```

#### Synchronization Logic
```go
// p2p/sync/pre_exec_sync.go
type PreExecSyncManager struct {
    orders    *PreExecutionSequence
    peers     map[string]Peer
    broadcast chan PreExecutionOrder
}

func (m *PreExecSyncManager) BroadcastOrder(order PreExecutionOrder) {
    for _, peer := range m.peers {
        peer.Send(order)
    }
}

func (m *PreExecSyncManager) HandleOrderReceived(order PreExecutionOrder) error {
    // Validate order
    if err := m.validateOrder(order); err != nil {
        return err
    }
    
    // Update local sequence
    m.orders.AddOrder(order)
    
    return nil
}
```

### Phase 5: Monitoring and Optimization

#### Metrics Collection
```go
type PreExecMetrics struct {
    TotalPreExecuted   uint64
    SuccessfulPreExec  uint64
    CacheHits          uint64
    CacheMisses        uint64
    AveragePreExecTime time.Duration
    MemoryUsage        uint64
}
```

#### Performance Optimization
- Dynamic cache size adjustment
- Smart pre-execution selection based on success rate
- Memory pressure handling with graceful degradation

## Technical Specifications

### Execution Flow

1. **Transaction Submission**
   ```
   User → Submit Tx → Mempool(pendingTxs)
   ```

2. **Pre-execution Phase (Validator)**
   ```
   CheckTx → Pre-execute → Update to PreConfirm → 
   Move to preExecCache → Broadcast order(N-txhash-seq)
   ```

3. **Synchronization Phase (Other Nodes)**
   ```
   Receive order → Validate → Update local sequence
   ```

4. **Block Packing Phase**
   ```
   PrepareProposal → Get from preExecCache in order → 
   Use cached results → Pack block
   ```

5. **Confirmation Phase**
   ```
   FinalizeBlock → Apply state changes → 
   Clear preExecCache → Update to Confirmed
   ```

### State Management

```go
// Pre-execution context with state snapshot
type PreExecContext struct {
    Context
    snapshot    StateSnapshot
    gasLimit    uint64
    stateWrites map[string][]byte
    stateReads  map[string][]byte
}

// State snapshot for rollback
type StateSnapshot struct {
    version     int64
    storeStates map[string][]byte
}
```

### Memory Management

```go
type CachePolicy struct {
    MaxSize         int
    TTL             time.Duration
    EvictionPolicy  string  // "LRU", "LFU", "FIFO"
    CompressionEnabled bool
}

func (c *PreExecutionCache) enforceMemoryLimit() {
    if c.memoryUsage() > c.maxMemory {
        c.evictOldest()
    }
}
```

## Configuration

### YAML Configuration Example

```yaml
# config/pre_execution.yaml
pre_execution:
  enabled: true
  max_cache_size: 10000
  cache_ttl: 30s
  max_pre_exec_gas: 1000000
  memory_limit: 1GB
  
  sequencer:
    broadcast_interval: 100ms
    sync_timeout: 5s
    max_pending_orders: 5000
  
  modules:
    bank:
      enabled: true
      msgs:
        MsgSend:
          enabled: true
          max_gas: 100000
          priority: 10
        MsgMultiSend:
          enabled: true
          max_gas: 200000
          priority: 8
    
    staking:
      enabled: true
      msgs:
        MsgDelegate:
          enabled: true
          max_gas: 150000
          priority: 5
          requires_ordering: true
        MsgUndelegate:
          enabled: false
    
    gov:
      enabled: true
      msgs:
        MsgVote:
          enabled: true
          max_gas: 50000
          priority: 3
        MsgSubmitProposal:
          enabled: false
```

### Runtime Configuration

```go
// Dynamic configuration updates
type PreExecConfigUpdate struct {
    Module   string
    MsgType  string
    Enabled  *bool
    MaxGas   *uint64
    Priority *uint32
}

func (m *PreExecutionManager) UpdateConfig(update PreExecConfigUpdate) error {
    // Apply configuration update without restart
    return m.config.ApplyUpdate(update)
}
```

## Testing Strategy

### Unit Tests

#### Cache Testing
```go
func TestPreExecutionCache(t *testing.T) {
    // Test cache operations
    // Test eviction policies
    // Test memory limits
    // Test TTL expiration
}
```

#### Module Pre-execution Testing
```go
func TestBankPreExecution(t *testing.T) {
    // Test MsgSend pre-execution
    // Test balance validation
    // Test error handling
}
```

#### State Rollback Testing
```go
func TestStateRollback(t *testing.T) {
    // Test snapshot creation
    // Test rollback on error
    // Test state consistency
}
```

### Integration Tests

#### End-to-End Flow
```go
func TestPreExecutionE2E(t *testing.T) {
    // Submit transaction
    // Verify pre-execution
    // Check cache
    // Verify block inclusion
    // Confirm state changes
}
```

#### Multi-node Synchronization
```go
func TestPreExecSync(t *testing.T) {
    // Start multiple nodes
    // Submit transactions
    // Verify order synchronization
    // Check consensus on execution order
}
```

#### Failure Recovery
```go
func TestFailureRecovery(t *testing.T) {
    // Test validator switch
    // Test cache corruption
    // Test network partition
    // Verify graceful degradation
}
```

### Performance Tests

#### Throughput Testing
```go
func BenchmarkPreExecution(b *testing.B) {
    // Measure pre-execution TPS
    // Compare with regular execution
    // Test under various loads
}
```

#### Memory Pressure Testing
```go
func TestMemoryPressure(t *testing.T) {
    // Fill cache to limit
    // Test eviction behavior
    // Measure memory usage
    // Test degradation strategy
}
```

#### Network Latency Impact
```go
func TestNetworkLatency(t *testing.T) {
    // Simulate various network conditions
    // Measure synchronization delay
    // Test timeout handling
}
```

## Expected Outcomes

### Performance Improvements
- **Confirmation Time**: Reduced from 2-3 seconds to 100-200ms
- **TPS Increase**: 30-50% improvement in effective throughput
- **Success Rate**: >95% pre-execution success rate

### System Requirements
- **Additional Memory**: <1GB for pre-execution cache
- **CPU Overhead**: <10% additional CPU usage
- **Network Bandwidth**: Minimal increase for order synchronization

### Security Guarantees
- **State Consistency**: Maintained through serial execution
- **Determinism**: Guaranteed through ordered execution
- **Fault Tolerance**: Automatic fallback to regular execution

## Conclusion

This pre-execution transaction design provides a scalable, extensible solution for improving transaction throughput and user experience in Pulsar. By leveraging our advanced versioned storage system and maintaining serial execution, we ensure state consistency while significantly reducing confirmation times. The modular design allows each module to independently opt-in to pre-execution support, ensuring backward compatibility and gradual adoption.