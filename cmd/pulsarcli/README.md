# Pulsar CLI

Pulsar CLI is a command-line interface for interacting with the Pulsar blockchain.

## Installation

```bash
make build
# Or install globally
make install
```

## Usage

### Basic Commands

```bash
# Show version
pulsarcli version

# Get help
pulsarcli --help
pulsarcli tx --help
pulsarcli query --help
```

### Bank Module Commands

#### Transaction Commands

```bash
# Send coins from one account to another
pulsarcli tx bank send [from_address] [to_address] [amount]

# Examples:
pulsarcli tx bank send addr1 addr2 100ubtc
pulsarcli tx bank send addr1 addr2 100ubtc,50wei

# Multi-send transaction
pulsarcli tx bank multi-send [inputs] [outputs]

# Example:
pulsarcli tx bank multi-send addr1:100ubtc,addr2:50ubtc addr3:80ubtc,addr4:70ubtc
```

#### Query Commands

```bash
# Query balance for a specific denomination
pulsarcli query bank balance [address] [denom]
pulsarcli query bank balance addr1 ubtc

# Query all balances of an account
pulsarcli query bank balances [address]
pulsarcli query bank balances addr1

# Query total supply of all coins
pulsarcli query bank total

# Query supply of a specific denomination
pulsarcli query bank supply-of [denom]
pulsarcli query bank supply-of ubtc

# Query denomination metadata
pulsarcli query bank denom-metadata [denom]
pulsarcli query bank denom-metadata ubtc

# Query all denominations metadata
pulsarcli query bank denoms-metadata
```

### Key Management

```bash
# Add a new key
pulsarcli keys add [name]

# List all keys
pulsarcli keys list

# Show key information
pulsarcli keys show [name]

# Delete a key
pulsarcli keys delete [name]

# Export a key
pulsarcli keys export [name]

# Import a key
pulsarcli keys import [name] [file]
```

### Configuration

```bash
# Set configuration value
pulsarcli config set [key] [value]
pulsarcli config set node tcp://localhost:26657
pulsarcli config set chain-id pulsar-1

# Get configuration value
pulsarcli config get [key]
pulsarcli config get node

# Reset configuration to defaults
pulsarcli config reset
```

### Query Blockchain State

```bash
# Query transaction by hash
pulsarcli query tx [hash]

# Query block by height
pulsarcli query block [height]
pulsarcli query block 100
pulsarcli query block  # Latest block
```

## Global Flags

```bash
--home string         # Directory for config and data (default "$HOME/.pulsar")
--chain-id string     # The network chain ID
--trace              # Print full stack trace on errors
--log-level string   # Set logging level (trace|debug|info|warn|error|fatal|panic)
--log-format string  # Set logging format (json|plain)
```

## Transaction Flags

```bash
--from string        # Name or address of account that signs the transaction
--fees string        # Fees to pay for the transaction
--gas string         # Gas limit to set per-transaction (default "auto")
--gas-prices string  # Gas prices to determine the transaction fee
--memo string        # Memo to include in the transaction
--dry-run           # Perform a dry run without broadcasting
--generate-only     # Generate transaction without broadcasting
```

## Query Flags

```bash
--output string     # Output format (json|text) (default "json")
--node string       # RPC endpoint (default "tcp://localhost:26657")
--height int64      # Use a specific height to query state at
```

## Module Structure

The CLI is organized by modules:

```
pulsarcli
├── tx                  # Transaction commands
│   ├── bank           # Bank module transactions
│   ├── staking        # Staking module transactions (future)
│   └── gov            # Governance module transactions (future)
├── query              # Query commands
│   ├── bank           # Bank module queries
│   ├── staking        # Staking module queries (future)
│   ├── gov            # Governance module queries (future)
│   ├── tx             # Query transaction by hash
│   └── block          # Query block by height
├── keys               # Key management
├── config             # Configuration management
└── version            # Version information
```

## Development

To add new module commands:

1. Create CLI package in your module: `modules/[module]/client/cli/`
2. Implement `GetTxCmd()` and `GetQueryCmd()` functions
3. Register in `cmd/pulsarcli/main.go`:

```go
// In txCmd() function
cmd.AddCommand(
    bankcli.GetTxCmd(),
    yourmodulecli.GetTxCmd(), // Add your module here
)

// In queryCmd() function
cmd.AddCommand(
    bankcli.GetQueryCmd(),
    yourmodulecli.GetQueryCmd(), // Add your module here
)
```

## Future Enhancements

- [ ] Actual transaction signing and broadcasting
- [ ] Key encryption and management
- [ ] Configuration persistence
- [ ] Transaction history
- [ ] Account management
- [ ] Validator operations
- [ ] Governance proposals
- [ ] IBC operations