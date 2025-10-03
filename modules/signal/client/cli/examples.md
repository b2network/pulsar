# Signal Module CLI Examples

This document provides comprehensive examples of using the Signal module CLI commands.

## Transaction Commands

### 1. Submit Signal

Submit a new AI computation signal to the network:

```bash
# Basic signal submission
pulsarcli tx signal submit-signal "inference" "./input.json" "llm_inference" "cosmos1validator..." \
  --cpu-cycles 1000000 \
  --memory-mb 1024 \
  --computation-time 5000 \
  --from mykey \
  --chain-id pulsar-1

# Signal with GPU usage
pulsarcli tx signal submit-signal "generation" "./prompt.json" "image_generation" "cosmos1validator..." \
  --cpu-cycles 2000000 \
  --memory-mb 2048 \
  --gpu-time-ms 3000 \
  --computation-time 8000 \
  --verification-key "0x1234abcd..." \
  --nonce 12345 \
  --from mykey

# Model training signal
pulsarcli tx signal submit-signal "training" "./dataset.json" "model_training" "cosmos1validator..." \
  --cpu-cycles 50000000 \
  --memory-mb 8192 \
  --gpu-time-ms 60000 \
  --computation-time 120000 \
  --storage-bytes 1073741824 \
  --from mykey
```

### 2. Verify Work

Verify AI work for submitted signals:

```bash
# Cryptographic verification
pulsarcli tx signal verify-work "signal_abc123" "cryptographic" "true" \
  --verification-proof "0xabcd1234..." \
  --from validator_key

# Statistical verification
pulsarcli tx signal verify-work "signal_def456" "statistical" "false" \
  --verification-proof "0xef567890..." \
  --from validator_key

# Consensus verification
pulsarcli tx signal verify-work "signal_ghi789" "consensus" "true" \
  --verification-proof "0x12345678..." \
  --from validator_key
```

### 3. Batch Submit

Submit multiple signals in a single transaction:

```bash
# Batch submission from JSON file
pulsarcli tx signal batch-submit "./signals-batch.json" \
  --from mykey \
  --gas auto \
  --gas-adjustment 1.3
```

Example batch file (`signals-batch.json`):
```json
[
  {
    "creator": "cosmos1abc...",
    "signal_type": "inference",
    "payload": "aGVsbG8gd29ybGQ=",
    "work_proof": {
      "work_type": "llm_inference",
      "input_hash": "abcd1234...",
      "output_hash": "efgh5678...",
      "computation_time": 5000,
      "resources_used": {
        "cpu_cycles": 1000000,
        "memory_mb": 1024,
        "gpu_time_ms": 500
      },
      "nonce": 12345
    },
    "validator_address": "cosmos1validator..."
  }
]
```

### 4. Claim Rewards

Claim accumulated signal rewards:

```bash
pulsarcli tx signal claim-reward \
  --from validator_key \
  --chain-id pulsar-1
```

### 5. Work Type Management

Register or update AI work types (requires governance authority):

```bash
# Register new work type
pulsarcli tx signal register-work-type "./new-work-type.json" \
  --from authority_key

# Update existing work type
pulsarcli tx signal update-work-type "./updated-work-type.json" \
  --from authority_key
```

Example work type file:
```json
{
  "id": "custom_llm",
  "name": "Custom LLM Inference",
  "description": "Custom large language model inference",
  "category": "inference",
  "base_score": 150,
  "score_multiplier": 1.5,
  "verifier": "statistical",
  "min_resources": {
    "cpu_cycles": 1000000,
    "memory_mb": 512,
    "gpu_time_ms": 100
  },
  "max_resources": {
    "cpu_cycles": 100000000,
    "memory_mb": 16384,
    "gpu_time_ms": 10000
  },
  "required_proofs": ["input_hash", "output_hash", "model_hash"],
  "enabled": true
}
```

### 6. Update Parameters

Update module parameters (requires governance authority):

```bash
pulsarcli tx signal update-params "./params.json" \
  --from authority_key
```

## Query Commands

### 1. Signal Queries

```bash
# Query specific signal
pulsarcli query signal signal "signal_abc123"

# Query signals with filters
pulsarcli query signal signals \
  --creator cosmos1abc... \
  --limit 10 \
  --status validated

# Query signals by validator
pulsarcli query signal validator-signals cosmos1validator... \
  --limit 20

# Query signals by status
pulsarcli query signal signals-by-status validated \
  --limit 10

# Query signals by work type
pulsarcli query signal signals-by-work-type llm_inference \
  --limit 15

# Query recent signals
pulsarcli query signal recent-signals \
  --limit 20
```

### 2. Score and Statistics

```bash
# Query signal score
pulsarcli query signal signal-score "signal_abc123"

# Query validator statistics
pulsarcli query signal validator-stats cosmos1validator...

# Query validator rank
pulsarcli query signal validator-rank cosmos1validator...

# Query leaderboard
pulsarcli query signal leaderboard \
  --limit 10

# Query top validators
pulsarcli query signal top-validators \
  --limit 5

# Query overall signal statistics
pulsarcli query signal signal-stats

# Query score distribution
pulsarcli query signal score-distribution
```

### 3. Work Type Queries

```bash
# Query all work types
pulsarcli query signal work-types

# Query enabled work types only
pulsarcli query signal work-types \
  --enabled-only

# Query specific work type
pulsarcli query signal work-type "llm_inference"
```

### 4. Verification Queries

```bash
# Query work verifications for a signal
pulsarcli query signal verifications "signal_abc123"
```

### 5. Module Parameters

```bash
# Query module parameters
pulsarcli query signal params
```

## Advanced Query Examples

### Filter Combinations

```bash
# Query validated signals from specific validator in time range
pulsarcli query signal signals \
  --validator cosmos1validator... \
  --status validated \
  --start-time 1640995200 \
  --end-time 1641081600 \
  --limit 50

# Query high-scoring signals
pulsarcli query signal signals \
  --min-score 500 \
  --limit 20

# Query signals by creator with score range
pulsarcli query signal signals \
  --creator cosmos1abc... \
  --min-score 100 \
  --max-score 1000

# Query signals with pagination
pulsarcli query signal signals \
  --limit 25 \
  --offset 50
```

### Output Formatting

```bash
# JSON output
pulsarcli query signal signal "signal_abc123" \
  --output json

# Table output (default)
pulsarcli query signal leaderboard \
  --limit 10 \
  --output table

# YAML output
pulsarcli query signal validator-stats cosmos1validator... \
  --output yaml
```

## Scripting Examples

### Monitor Validator Performance

```bash
#!/bin/bash
VALIDATOR="cosmos1validator..."

echo "=== Validator Signal Performance ==="
pulsarcli query signal validator-stats $VALIDATOR

echo -e "\n=== Recent Signals ==="
pulsarcli query signal validator-signals $VALIDATOR --limit 5

echo -e "\n=== Current Rank ==="
pulsarcli query signal validator-rank $VALIDATOR
```

### Submit Multiple Signals

```bash
#!/bin/bash
VALIDATOR="cosmos1validator..."
FROM_KEY="mykey"

for i in {1..5}; do
  echo "Submitting signal $i..."
  pulsarcli tx signal submit-signal \
    "inference" \
    "./inputs/input_$i.json" \
    "llm_inference" \
    $VALIDATOR \
    --cpu-cycles $((1000000 + i * 100000)) \
    --memory-mb $((1024 + i * 256)) \
    --computation-time $((5000 + i * 1000)) \
    --from $FROM_KEY \
    --yes

  sleep 2
done
```

### Check Signal Processing Status

```bash
#!/bin/bash
SIGNAL_ID="$1"

if [ -z "$SIGNAL_ID" ]; then
  echo "Usage: $0 <signal_id>"
  exit 1
fi

echo "Checking signal: $SIGNAL_ID"
pulsarcli query signal signal $SIGNAL_ID

echo -e "\nSignal score:"
pulsarcli query signal signal-score $SIGNAL_ID

echo -e "\nVerifications:"
pulsarcli query signal verifications $SIGNAL_ID
```

## Useful Tips

### 1. Gas Estimation

Always use gas estimation for transactions:
```bash
--gas auto --gas-adjustment 1.3
```

### 2. Dry Run

Test transactions without submitting:
```bash
--dry-run
```

### 3. Broadcasting

For better reliability, use sync mode:
```bash
--broadcast-mode sync
```

### 4. JSON Files

Keep JSON files properly formatted and validated before use.

### 5. Key Management

Use appropriate keys for different operations:
- Regular users: Use personal keys for signal submission
- Validators: Use validator keys for work verification
- Authority: Use governance keys for parameter updates

### 6. Monitoring

Regularly check:
- Signal processing status
- Validator statistics and rank
- Module parameters
- Overall network statistics

This comprehensive CLI reference enables effective interaction with the Signal module for AI computation verification and PoSg consensus participation.