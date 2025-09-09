# Pulsar - CometBFT-based Blockchain

Pulsar is a blockchain implementation based on CometBFT consensus engine.

## Prerequisites

- Go 1.22 or higher
- Make

## Quick Start

### Build
```bash
make build
```

### Initialize Node
```bash
make init
```

### Start Node
```bash
make start
```

### Reset (Clean all data)
```bash
make reset
```

### Fresh Start (Reset + Init + Start)
```bash
make fresh
```

## Architecture

The project follows a modular architecture inspired by Cosmos SDK:

- `abci/` - ABCI application implementation
- `cmd/pulsard/` - Node CLI commands
- `build/` - Compiled binaries

## Development

### Install Dependencies
```bash
make deps
```

### Run Tests
```bash
make test
```

## Configuration

The node stores its configuration and data in `~/.pulsar/` by default:
- `config/` - Configuration files
- `data/` - Blockchain data

You can specify a custom home directory:
```bash
./build/pulsard init my-node --home /custom/path
./build/pulsard start --home /custom/path
```