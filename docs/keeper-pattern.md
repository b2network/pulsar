# Keeper Pattern in Pulsar

The Keeper pattern is a fundamental design pattern in Pulsar that provides secure, modular access control to blockchain state. This pattern is inspired by the Cosmos SDK design but implemented specifically for Pulsar's architecture.

## Overview

The Keeper pattern implements the **object-capabilities security model**, where modules can only access state through well-defined interfaces and capabilities. This ensures:

- **Module Isolation**: Each module can only access its own state store
- **Interface-Based Access**: Inter-module communication happens through explicit interfaces
- **Permission Management**: Operations are controlled by explicit capabilities
- **Separation of Concerns**: Each keeper has a specific responsibility

## Architecture

### Core Components

1. **Base Keeper** (`keeper/base/keeper.go`)
   - Provides fundamental functionality for all keepers
   - Handles store access, codec operations, and basic CRUD operations
   - Implements authority validation and event emission

2. **KV Store Keeper** 
   - Extends BaseKeeper with key-value store operations
   - Provides typed object storage and retrieval
   - Handles store iteration and prefix operations

3. **Module-Specific Keepers**
   - Bank Keeper: Balance and transfer operations
   - Staking Keeper: Delegation and validator management  
   - Governance Keeper: Proposal and voting operations

### Key Interfaces

```go
// Core keeper interface
type KVStoreKeeper interface {
    GetKVStore(ctx context.Context) types.KVStore
    Get(ctx context.Context, key []byte) []byte
    Set(ctx context.Context, key []byte, value []byte)
    Delete(ctx context.Context, key []byte)
    Has(ctx context.Context, key []byte) bool
}

// Context interface for state access
type Context interface {
    KVStore(key types.StoreKey) types.KVStore
    BlockHeight() int64
    ChainID() string
    Logger() Logger
    EventManager() EventManager
}
```

## Module Implementation

### Bank Keeper

The Bank keeper manages account balances and token transfers:

```go
type Keeper struct {
    *base.KVStoreKeeper
    accountKeeper types.AccountKeeper
    maccPerms map[string][]string // Module account permissions
}
```

**Key Operations:**
- `GetBalance(ctx, addr, denom)`: Retrieve account balance
- `SetBalance(ctx, addr, balance)`: Update account balance  
- `SendCoins(ctx, from, to, amount)`: Transfer tokens between accounts
- `MintCoins(ctx, module, amount)`: Create new tokens (requires minter permission)
- `BurnCoins(ctx, module, amount)`: Destroy tokens (requires burner permission)

### Staking Keeper

The Staking keeper manages validators and delegations:

```go
type Keeper struct {
    *base.KVStoreKeeper
    bankKeeper types.BankKeeper
    slashingKeeper types.SlashingKeeper
    bondDenom string
}
```

**Key Operations:**
- `GetValidator(ctx, addr)`: Retrieve validator information
- `Delegate(ctx, delegator, validator, amount)`: Delegate tokens to validator
- `Undelegate(ctx, delegator, validator, shares)`: Start unbonding process
- `BeginRedelegate(ctx, del, valSrc, valDst, shares)`: Redelegate between validators

### Governance Keeper

The Governance keeper manages proposals and voting:

```go
type Keeper struct {
    *base.KVStoreKeeper
    bankKeeper types.BankKeeper
    stakingKeeper types.StakingKeeper
    authority string
}
```

**Key Operations:**
- `SubmitProposal(ctx, messages, title, summary, proposer)`: Submit new proposal
- `AddDeposit(ctx, proposalID, depositor, amount)`: Add deposit to proposal
- `AddVote(ctx, proposalID, voter, options, metadata)`: Vote on proposal
- `Tally(ctx, proposal)`: Calculate voting results

## Security Features

### Object Capabilities Model

1. **Store Isolation**: Each keeper has its own `StoreKey` and can only access its designated store
2. **Interface Boundaries**: Modules interact through well-defined interfaces, not direct access
3. **Permission System**: Module accounts have specific permissions (minter, burner, etc.)
4. **Authority Checks**: Critical operations require proper authorization

### Example Security Implementation

```go
// Bank keeper can only access bank store
func (k Keeper) GetKVStore(ctx keepertypes.Context) types.KVStore {
    return ctx.KVStore(k.storeKey) // Only bank store key works
}

// Minting requires explicit permission
func (k Keeper) MintCoins(ctx keepertypes.Context, moduleName string, amount keepertypes.Coins) error {
    if !k.hasPermission(moduleName, "minter") {
        return fmt.Errorf("module %s does not have minting permission", moduleName)
    }
    // ... minting logic
}
```

## Inter-Module Communication

Modules communicate through dependency injection of interfaces:

```go
// Staking keeper depends on bank keeper interface
type Keeper struct {
    bankKeeper types.BankKeeper // Interface, not concrete type
}

// Governance keeper depends on both bank and staking
type Keeper struct {
    bankKeeper    types.BankKeeper
    stakingKeeper types.StakingKeeper
}
```

This design ensures:
- Loose coupling between modules
- Testability through interface mocking
- Clear dependency relationships
- Controlled access to other module functionality

## Usage Examples

### Basic Operations

```go
// Create keepers with proper dependencies
bankKeeper := bankkeeper.NewKeeper(bankStoreKey, codec, accountKeeper, permissions)
stakingKeeper := stakingkeeper.NewKeeper(stakingStoreKey, codec, bankKeeper, slashingKeeper, "stake")
govKeeper := govkeeper.NewKeeper(govStoreKey, codec, bankKeeper, stakingKeeper, authority)

// Use keepers for operations
balance := bankKeeper.GetBalance(ctx, addr, "stake")
err := stakingKeeper.Delegate(ctx, delegator, validator, amount)
proposalID, err := govKeeper.SubmitProposal(ctx, messages, title, summary, proposer)
```

### Cross-Module Operations

```go
// Governance proposal affecting staking parameters
proposalID, err := govKeeper.SubmitProposal(ctx, stakingParamChangeMessages, title, summary, proposer)

// Deposit requires bank transfer
activated, err := govKeeper.AddDeposit(ctx, proposalID, depositor, depositAmount)

// Voting requires staking-based voting power (handled internally by governance keeper)
err = govKeeper.AddVote(ctx, proposalID, voter, options, metadata)
```

## Testing

The keeper pattern enables comprehensive testing through:

1. **Interface Mocking**: Mock dependencies for isolated testing
2. **Memory Stores**: Use in-memory stores for fast testing
3. **Integration Tests**: Test cross-module interactions
4. **Security Tests**: Verify access control and permissions

## Best Practices

1. **Interface Design**: Define minimal, focused interfaces for inter-module communication
2. **Permission Management**: Use explicit permissions for sensitive operations
3. **Event Emission**: Emit events for important state changes
4. **Error Handling**: Provide clear error messages for unauthorized operations
5. **Store Key Management**: Use unique store keys for each module
6. **Parameter Management**: Store module parameters in a structured way

## Implementation Files

- **Base Keeper**: `keeper/base/keeper.go`
- **Keeper Types**: `keeper/types/interfaces.go`
- **Bank Module**: `modules/bank/keeper/keeper.go`
- **Staking Module**: `modules/staking/keeper/keeper.go`  
- **Governance Module**: `modules/gov/keeper/keeper.go`
- **Integration Examples**: `examples/keeper_integration.go`
- **Tests**: `examples/keeper_test.go`

This implementation provides a secure, modular foundation for blockchain state management while maintaining the flexibility needed for complex cross-module interactions.