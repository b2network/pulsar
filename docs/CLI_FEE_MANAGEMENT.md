# Fee Management CLI Guide

This guide covers the comprehensive fee management commands available in the Pulsar CLI.

## Overview

The Pulsar CLI now includes extensive fee management capabilities across three main command groups:

- **Transaction Commands** (`pulsarcli tx fee`): Manage fee configurations
- **Query Commands** (`pulsarcli query fee`): Query fee information
- **Utility Commands** (`pulsarcli fee-utils`): Advanced fee tools and analysis

## Transaction Commands

### Add Fee Denomination

Add a new fee denomination to the system:

```bash
pulsarcli tx fee add-denom [denom] [min-gas-price] --priority [priority] --from [authority]
```

**Examples:**
```bash
# Add ubtc as primary fee token
pulsarcli tx fee add-denom ubtc 0.01ubtc --priority 100 --from authority

# Add ueth as secondary fee token
pulsarcli tx fee add-denom ueth 100ueth --priority 90 --from authority

# Add disabled denomination for future use
pulsarcli tx fee add-denom utest 1utest --priority 50 --enabled=false --from authority
```

**Flags:**
- `--priority`: Priority level (higher = more preferred)
- `--enabled`: Whether denomination is enabled (default: true)

### Update Fee Denomination

Update existing fee denomination configuration:

```bash
pulsarcli tx fee update-denom [denom] [min-gas-price] --priority [priority] --from [authority]
```

**Examples:**
```bash
# Update minimum gas price
pulsarcli tx fee update-denom ubtc 0.02ubtc --priority 100 --from authority

# Disable a denomination
pulsarcli tx fee update-denom uold 0.01uold --enabled=false --from authority
```

### Remove Fee Denomination

Remove a fee denomination:

```bash
pulsarcli tx fee remove-denom [denom] --from [authority]
```

**Example:**
```bash
pulsarcli tx fee remove-denom uold --from authority
```

### Set Module Gas Configuration

Configure gas consumption for specific module message types:

```bash
pulsarcli tx fee set-gas-config [module] [message-type] [gas-amount] --from [authority]
```

**Examples:**
```bash
# Set bank transfer gas
pulsarcli tx fee set-gas-config bank MsgSend 25000 --from authority

# Set coin minting gas
pulsarcli tx fee set-gas-config coin MsgMint 40000 --from authority

# Set staking delegation gas
pulsarcli tx fee set-gas-config staking MsgDelegate 50000 --from authority
```

### Update Dynamic Gas Factors

Update multipliers for dynamic gas calculation:

```bash
pulsarcli tx fee update-gas-factors --size [factor] --complexity [factor] --network [factor] --storage [factor] --from [authority]
```

**Example:**
```bash
pulsarcli tx fee update-gas-factors --size 1.2 --complexity 1.1 --network 1.0 --storage 1.3 --from authority
```

### Set Fee Distribution

Configure how collected fees are distributed:

```bash
pulsarcli tx fee set-distribution --burn [rate] --validators [rate] --community [rate] --dev [rate] --from [authority]
```

**Example:**
```bash
# 50% burn, 30% validators, 15% community, 5% dev fund
pulsarcli tx fee set-distribution --burn 0.5 --validators 0.3 --community 0.15 --dev 0.05 --from authority
```

## Query Commands

### Query Module Parameters

Get current fee module parameters:

```bash
pulsarcli query fee params
```

### Query Fee Denominations

List all fee denominations:

```bash
pulsarcli query fee denominations
```

Query specific denomination:

```bash
pulsarcli query fee denomination [denom]
```

**Examples:**
```bash
pulsarcli query fee denominations
pulsarcli query fee denomination ubtc
```

### Query Gas Configurations

Get gas configuration for a module:

```bash
pulsarcli query fee gas-config [module]
```

Get all gas configurations:

```bash
pulsarcli query fee gas-configs
```

**Examples:**
```bash
pulsarcli query fee gas-config bank
pulsarcli query fee gas-configs
```

### Query Dynamic Factors

Get current dynamic gas factors:

```bash
pulsarcli query fee dynamic-factors
```

### Query Fee Distribution

Get fee distribution configuration:

```bash
pulsarcli query fee distribution
```

### Query Gas Price Statistics

Get gas price statistics for a denomination:

```bash
pulsarcli query fee gas-price-stats [denom]
```

**Example:**
```bash
pulsarcli query fee gas-price-stats ubtc
```

### Query Fee Estimates

Estimate fees for transactions:

```bash
pulsarcli query fee estimate [module] [message-type] --priority [level] --denom [denom]
```

**Examples:**
```bash
# Estimate bank transfer fees
pulsarcli query fee estimate bank MsgSend

# Estimate with high priority
pulsarcli query fee estimate coin MsgMint --priority high

# Estimate for specific denomination
pulsarcli query fee estimate staking MsgDelegate --denom ubtc
```

### Query Supported Denominations

List currently supported fee denominations:

```bash
pulsarcli query fee supported-denoms
```

## Utility Commands

### Calculate Fees

Calculate fees for specific scenarios:

```bash
pulsarcli fee-utils calculate [module] [message-type] [options]
```

**Examples:**
```bash
# Basic calculation
pulsarcli fee-utils calculate bank MsgSend --gas 30000 --priority high

# Compare across priorities
pulsarcli fee-utils calculate coin MsgMint --compare-all

# Custom multiplier
pulsarcli fee-utils calculate staking MsgDelegate --multiplier 1.5
```

**Flags:**
- `--gas`: Gas limit (0 = auto estimate)
- `--priority`: Fee priority level
- `--denom`: Specific denomination
- `--multiplier`: Fee multiplier
- `--compare-all`: Compare all priority levels

### Compare Fees

Compare fees across configurations:

```bash
pulsarcli fee-utils compare [module] [message-type] --denoms [list]
```

**Examples:**
```bash
# Compare all denominations
pulsarcli fee-utils compare bank MsgSend

# Compare specific denominations
pulsarcli fee-utils compare coin MsgMint --denoms ubtc,ueth,upulse
```

### Run Benchmarks

Benchmark fee calculation performance:

```bash
pulsarcli fee-utils benchmark [options]
```

**Examples:**
```bash
# Benchmark specific modules
pulsarcli fee-utils benchmark --modules bank,coin --iterations 1000

# Full benchmark
pulsarcli fee-utils benchmark --full --output results.json
```

### Optimize Fees

Get fee optimization suggestions:

```bash
pulsarcli fee-utils optimize [options]
```

**Examples:**
```bash
# Budget optimization
pulsarcli fee-utils optimize --max-fee 1000ubtc

# Time optimization
pulsarcli fee-utils optimize --target-time 5s --priority high
```

### Configuration Management

Export current configuration:

```bash
pulsarcli fee-utils export-config --output config.json
```

Import configuration:

```bash
pulsarcli fee-utils import-config config.json
```

Validate configuration:

```bash
pulsarcli fee-utils validate-config config.json
```

### Fee Analytics

Generate fee analytics and reports:

```bash
pulsarcli fee-utils analytics [options]
```

**Examples:**
```bash
# 30-day analytics
pulsarcli fee-utils analytics --days 30

# Module-specific analytics
pulsarcli fee-utils analytics --module bank --output report.json
```

## Enhanced Transaction Flags

All transaction commands now support enhanced fee options:

### Basic Enhanced Flags

```bash
--auto-fee              # Enable automatic intelligent fee calculation
--fee-priority [level]  # Set priority: low, normal, high, fast
--fee-denom [denom]     # Preferred fee denomination
--max-fee [amount]      # Maximum fee limit
--fee-multiplier [num]  # Fee multiplier (default: 1.0)
```

### Examples with Enhanced Fees

```bash
# Auto fee with high priority
pulsarcli tx bank send addr1 addr2 100ubtc --from mykey --auto-fee --fee-priority high

# Budget-limited transaction
pulsarcli tx coin mint addr1 1000000ubtc --from mykey --auto-fee --max-fee 500ubtc

# Custom fee scaling
pulsarcli tx staking delegate val1 1000000ubtc --from mykey --auto-fee --fee-multiplier 1.5

# Dry run with fee suggestions
pulsarcli tx bank send addr1 addr2 100ubtc --from mykey --auto-fee --dry-run
```

## Fee Priority Levels

| Priority | Description | Multiplier | Use Case |
|----------|-------------|------------|----------|
| `low` | Economical fees, slower confirmation | 1.0x | Cost-conscious transactions |
| `normal` | Standard fees, typical confirmation | 1.5x | Regular transactions (default) |
| `high` | Higher fees, faster confirmation | 2.0x | Important transactions |
| `fast` | Premium fees, fastest confirmation | 3.0x | Urgent transactions |

## Fee Denominations

| Denomination | Priority | Min Gas Price | Use Case |
|--------------|----------|---------------|----------|
| `ubtc` | 100 | 0.01ubtc | Primary fee token |
| `ueth` | 90 | 100ueth | Secondary fee token |
| `upulse` | 80 | 1000upulse | Native token |

## Output Formats

Most query commands support multiple output formats:

```bash
--output text    # Human-readable text (default)
--output json    # JSON format for scripts
```

## Examples and Use Cases

### Setting Up Fee System

```bash
# 1. Add fee denominations
pulsarcli tx fee add-denom ubtc 0.01ubtc --priority 100 --from authority
pulsarcli tx fee add-denom ueth 100ueth --priority 90 --from authority

# 2. Configure gas for modules
pulsarcli tx fee set-gas-config bank MsgSend 25000 --from authority
pulsarcli tx fee set-gas-config coin MsgMint 40000 --from authority

# 3. Set fee distribution
pulsarcli tx fee set-distribution --burn 0.5 --validators 0.3 --community 0.15 --dev 0.05 --from authority
```

### User Transaction Examples

```bash
# Automatic fee calculation
pulsarcli tx bank send addr1 addr2 100ubtc --from mykey --auto-fee

# High priority transaction
pulsarcli tx bank send addr1 addr2 100ubtc --from mykey --auto-fee --fee-priority high

# Budget-conscious transaction
pulsarcli tx bank send addr1 addr2 100ubtc --from mykey --auto-fee --fee-priority low --max-fee 50ubtc
```

### Analysis and Monitoring

```bash
# Check current fee rates
pulsarcli query fee denominations

# Estimate transaction costs
pulsarcli query fee estimate bank MsgSend --priority normal

# Generate usage reports
pulsarcli fee-utils analytics --days 7 --output weekly-report.json

# Benchmark performance
pulsarcli fee-utils benchmark --modules bank,coin --iterations 1000
```

## Troubleshooting

### Common Issues

1. **Authority Required**: Most configuration commands require authority permissions
2. **Invalid Gas Price Format**: Use format like "0.01ubtc" or "100ueth"
3. **Distribution Must Sum to 1.0**: Fee distribution rates must total exactly 1.0
4. **Denomination Not Found**: Ensure denomination exists before using

### Getting Help

```bash
# Command-specific help
pulsarcli tx fee --help
pulsarcli query fee --help
pulsarcli fee-utils --help

# Subcommand help
pulsarcli tx fee add-denom --help
pulsarcli query fee estimate --help
```

---

This comprehensive fee management system provides complete control over transaction costs, gas configurations, and fee optimization in the Pulsar blockchain.