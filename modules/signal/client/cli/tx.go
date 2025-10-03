package cli

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	keepertypes "github.com/b2network/pulsar/keeper/types"
	"github.com/b2network/pulsar/modules/signal/types"
)

// GetTxCmd returns the transaction commands for the signal module
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("%s transactions subcommands", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       keepertypes.ValidateCmd,
	}

	cmd.AddCommand(
		NewSubmitSignalCmd(),
		NewVerifyWorkCmd(),
		NewUpdateWorkTypeCmd(),
		NewClaimRewardCmd(),
		NewBatchSubmitCmd(),
		NewUpdateParamsCmd(),
		NewRegisterWorkTypeCmd(),
	)

	return cmd
}

// NewSubmitSignalCmd creates a command to submit a signal
func NewSubmitSignalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-signal [signal-type] [payload-file] [work-type] [validator-address]",
		Short: "Submit a new AI computation signal",
		Long: `Submit a new AI computation signal to the network.

The payload-file should contain the input data for the computation.
Work-type specifies the type of AI work performed (e.g., llm_inference, image_generation).
Validator-address is the address of the validator to associate with this signal.

Example:
$ pulsarcli tx signal submit-signal "inference" "./input.json" "llm_inference" "cosmos1validator..."`,
		Args: cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := keepertypes.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			signalType := args[0]
			payloadFile := args[1]
			workType := args[2]
			validatorAddr := args[3]

			// Read payload from file
			payload, err := readPayloadFile(payloadFile)
			if err != nil {
				return fmt.Errorf("failed to read payload file: %w", err)
			}

			// Get work proof flags
			workProof, err := parseWorkProofFromFlags(cmd, workType, payload)
			if err != nil {
				return fmt.Errorf("failed to parse work proof: %w", err)
			}

			msg := types.NewMsgSubmitSignal(
				clientCtx.GetFromAddress().String(),
				signalType,
				payload,
				workProof,
				validatorAddr,
			)

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return keepertypes.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	addWorkProofFlags(cmd)
	keepertypes.AddTxFlagsToCmd(cmd)

	return cmd
}

// NewVerifyWorkCmd creates a command to verify work
func NewVerifyWorkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify-work [signal-id] [verification-method] [is-valid]",
		Short: "Verify AI work for a signal",
		Long: `Verify the AI work performed for a specific signal.

verification-method can be: cryptographic, reproducible, statistical, consensus, hybrid
is-valid should be true or false based on verification result

Example:
$ pulsarcli tx signal verify-work "signal123" "cryptographic" "true" --verification-proof "0x..."`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := keepertypes.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			signalID := args[0]
			verificationMethod := types.VerificationMethod(args[1])
			isValid, err := strconv.ParseBool(args[2])
			if err != nil {
				return fmt.Errorf("invalid is-valid value: %w", err)
			}

			// Get verification proof from flag
			verificationProofStr, _ := cmd.Flags().GetString("verification-proof")
			verificationProof, err := hex.DecodeString(strings.TrimPrefix(verificationProofStr, "0x"))
			if err != nil {
				return fmt.Errorf("invalid verification proof hex: %w", err)
			}

			msg := types.NewMsgVerifyWork(
				clientCtx.GetFromAddress().String(),
				signalID,
				verificationMethod,
				verificationProof,
				isValid,
			)

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return keepertypes.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("verification-proof", "", "Hexadecimal verification proof data")
	keepertypes.AddTxFlagsToCmd(cmd)

	return cmd
}

// NewUpdateWorkTypeCmd creates a command to update work type
func NewUpdateWorkTypeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-work-type [work-type-file]",
		Short: "Update an AI work type configuration",
		Long: `Update an AI work type configuration from a JSON file.

The work-type-file should contain the complete work type definition in JSON format.

Example:
$ pulsarcli tx signal update-work-type "./work-type.json"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := keepertypes.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			workTypeFile := args[0]

			// Read and parse work type from file
			workType, err := readWorkTypeFile(workTypeFile)
			if err != nil {
				return fmt.Errorf("failed to read work type file: %w", err)
			}

			msg := types.NewMsgUpdateWorkType(
				clientCtx.GetFromAddress().String(),
				workType,
			)

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return keepertypes.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	keepertypes.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewClaimRewardCmd creates a command to claim rewards
func NewClaimRewardCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claim-reward",
		Short: "Claim signal rewards for validator",
		Long: `Claim accumulated signal rewards for the validator account.

Example:
$ pulsarcli tx signal claim-reward`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := keepertypes.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgClaimReward(
				clientCtx.GetFromAddress().String(),
			)

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return keepertypes.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	keepertypes.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewBatchSubmitCmd creates a command for batch signal submission
func NewBatchSubmitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "batch-submit [batch-file]",
		Short: "Submit multiple signals in a batch",
		Long: `Submit multiple signals in a single transaction from a JSON file.

The batch-file should contain an array of signal submission data.

Example:
$ pulsarcli tx signal batch-submit "./signals-batch.json"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := keepertypes.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			batchFile := args[0]

			// Read and parse batch from file
			signals, err := readSignalBatchFile(batchFile)
			if err != nil {
				return fmt.Errorf("failed to read batch file: %w", err)
			}

			msg := types.NewMsgBatchSubmit(
				clientCtx.GetFromAddress().String(),
				signals,
			)

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return keepertypes.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	keepertypes.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewUpdateParamsCmd creates a command to update module parameters
func NewUpdateParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-params [params-file]",
		Short: "Update signal module parameters",
		Long: `Update signal module parameters from a JSON file.

The params-file should contain the complete module parameters in JSON format.
This command requires governance authority.

Example:
$ pulsarcli tx signal update-params "./params.json"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := keepertypes.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			paramsFile := args[0]

			// Read and parse params from file
			params, err := readParamsFile(paramsFile)
			if err != nil {
				return fmt.Errorf("failed to read params file: %w", err)
			}

			msg := types.NewMsgUpdateParams(
				clientCtx.GetFromAddress().String(),
				params,
			)

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return keepertypes.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	keepertypes.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewRegisterWorkTypeCmd creates a command to register new work type
func NewRegisterWorkTypeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-work-type [work-type-file]",
		Short: "Register a new AI work type",
		Long: `Register a new AI work type from a JSON file.

The work-type-file should contain the complete work type definition.
This command requires governance authority.

Example:
$ pulsarcli tx signal register-work-type "./new-work-type.json"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := keepertypes.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			workTypeFile := args[0]

			// Read and parse work type from file
			workType, err := readWorkTypeFile(workTypeFile)
			if err != nil {
				return fmt.Errorf("failed to read work type file: %w", err)
			}

			msg := types.NewMsgRegisterWorkType(
				clientCtx.GetFromAddress().String(),
				workType,
			)

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return keepertypes.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	keepertypes.AddTxFlagsToCmd(cmd)
	return cmd
}

// Helper functions

// addWorkProofFlags adds work proof related flags to command
func addWorkProofFlags(cmd *cobra.Command) {
	cmd.Flags().String("input-hash", "", "Input data hash (hex)")
	cmd.Flags().String("output-hash", "", "Output data hash (hex)")
	cmd.Flags().Uint64("computation-time", 0, "Computation time in milliseconds")
	cmd.Flags().Uint64("cpu-cycles", 0, "CPU cycles used")
	cmd.Flags().Uint64("memory-mb", 0, "Memory used in MB")
	cmd.Flags().Uint64("gpu-time-ms", 0, "GPU time used in milliseconds")
	cmd.Flags().Uint64("network-bandwidth", 0, "Network bandwidth used in bytes")
	cmd.Flags().Uint64("storage-bytes", 0, "Storage used in bytes")
	cmd.Flags().String("verification-key", "", "Verification key (hex)")
	cmd.Flags().Uint64("nonce", 0, "Proof of work nonce")
}

// parseWorkProofFromFlags parses work proof from command flags
func parseWorkProofFromFlags(cmd *cobra.Command, workType string, payload []byte) (types.WorkProof, error) {
	inputHashStr, _ := cmd.Flags().GetString("input-hash")
	outputHashStr, _ := cmd.Flags().GetString("output-hash")
	computationTime, _ := cmd.Flags().GetUint64("computation-time")
	cpuCycles, _ := cmd.Flags().GetUint64("cpu-cycles")
	memoryMB, _ := cmd.Flags().GetUint64("memory-mb")
	gpuTimeMs, _ := cmd.Flags().GetUint64("gpu-time-ms")
	networkBandwidth, _ := cmd.Flags().GetUint64("network-bandwidth")
	storageBytes, _ := cmd.Flags().GetUint64("storage-bytes")
	verificationKeyStr, _ := cmd.Flags().GetString("verification-key")
	nonce, _ := cmd.Flags().GetUint64("nonce")

	// Parse hashes
	var inputHash, outputHash []byte
	var err error

	if inputHashStr != "" {
		inputHash, err = hex.DecodeString(strings.TrimPrefix(inputHashStr, "0x"))
		if err != nil {
			return types.WorkProof{}, fmt.Errorf("invalid input hash: %w", err)
		}
	} else {
		// Generate input hash from payload
		inputHash = types.HashPayload(payload)
	}

	if outputHashStr != "" {
		outputHash, err = hex.DecodeString(strings.TrimPrefix(outputHashStr, "0x"))
		if err != nil {
			return types.WorkProof{}, fmt.Errorf("invalid output hash: %w", err)
		}
	} else {
		// Generate dummy output hash (in practice, this would be the actual computation result)
		outputHash = types.HashPayload(append(payload, []byte("output")...))
	}

	var verificationKey []byte
	if verificationKeyStr != "" {
		verificationKey, err = hex.DecodeString(strings.TrimPrefix(verificationKeyStr, "0x"))
		if err != nil {
			return types.WorkProof{}, fmt.Errorf("invalid verification key: %w", err)
		}
	}

	workProof := types.WorkProof{
		WorkType:        workType,
		InputHash:       inputHash,
		OutputHash:      outputHash,
		ComputationTime: int64(computationTime),
		ResourcesUsed: types.Resources{
			CPUCycles:        cpuCycles,
			MemoryMB:         memoryMB,
			GPUTimeMs:        gpuTimeMs,
			NetworkBandwidth: networkBandwidth,
			StorageBytes:     storageBytes,
		},
		VerificationKey: verificationKey,
		Nonce:           nonce,
	}

	return workProof, nil
}

// readPayloadFile reads payload data from file
func readPayloadFile(filename string) ([]byte, error) {
	// In a real implementation, this would read from the actual file system
	// For now, return dummy data
	return []byte(fmt.Sprintf("payload_from_%s", filename)), nil
}

// readWorkTypeFile reads work type from JSON file
func readWorkTypeFile(filename string) (types.AIWorkType, error) {
	// In a real implementation, this would read from the actual file system
	// For now, return a dummy work type
	return types.AIWorkType{
		ID:              "custom_work_type",
		Name:            "Custom Work Type",
		Description:     "A custom AI work type",
		Category:        "custom",
		BaseScore:       100,
		ScoreMultiplier: 1.0,
		Verifier:        "cryptographic",
		MinResources: types.Resources{
			CPUCycles: 1000,
			MemoryMB:  256,
		},
		MaxResources: types.Resources{
			CPUCycles: 1000000,
			MemoryMB:  4096,
		},
		RequiredProofs: []string{"input_hash", "output_hash"},
		Enabled:        true,
	}, nil
}

// readSignalBatchFile reads signal batch from JSON file
func readSignalBatchFile(filename string) ([]types.MsgSubmitSignal, error) {
	// In a real implementation, this would read from the actual file system
	// For now, return dummy batch
	return []types.MsgSubmitSignal{
		{
			Creator:     "creator1",
			SignalType:  "inference",
			Payload:     []byte("payload1"),
			WorkProof:   types.WorkProof{WorkType: "llm_inference"},
			ValidatorAddress: "validator1",
		},
	}, nil
}

// readParamsFile reads module parameters from JSON file
func readParamsFile(filename string) (types.Params, error) {
	// In a real implementation, this would read from the actual file system
	// For now, return default params
	return types.DefaultParams(), nil
}

// JSON file format examples for documentation

// Example work type JSON:
/*
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
    "gpu_time_ms": 100,
    "network_bandwidth": 0,
    "storage_bytes": 0
  },
  "max_resources": {
    "cpu_cycles": 100000000,
    "memory_mb": 16384,
    "gpu_time_ms": 10000,
    "network_bandwidth": 1048576,
    "storage_bytes": 1073741824
  },
  "required_proofs": ["input_hash", "output_hash", "model_hash"],
  "enabled": true
}
*/

// Example batch submit JSON:
/*
[
  {
    "creator": "cosmos1...",
    "signal_type": "inference",
    "payload": "base64_encoded_payload",
    "work_proof": {
      "work_type": "llm_inference",
      "input_hash": "hex_hash",
      "output_hash": "hex_hash",
      "computation_time": 5000,
      "resources_used": {
        "cpu_cycles": 1000000,
        "memory_mb": 1024,
        "gpu_time_ms": 500,
        "network_bandwidth": 0,
        "storage_bytes": 0
      },
      "verification_key": "hex_key",
      "nonce": 12345
    },
    "validator_address": "cosmos1validator..."
  }
]
*/

// Example params JSON:
/*
{
  "signal_submission_fee": [{"denom": "stake", "amount": "100"}],
  "min_signal_score": 10,
  "max_signals_per_block": 100,
  "signal_retention_period": 2592000,
  "score_decay_rate": 0.1,
  "work_type_weights": {
    "llm_inference": 1.5,
    "image_generation": 2.0,
    "model_training": 3.0
  },
  "verification_threshold": 3,
  "reward_pool_percentage": 0.5,
  "auto_validation_enabled": true,
  "max_signal_age": 3600,
  "invalid_signal_slash_rate": 0.01,
  "verified_signal_bonus": 1.5,
  "max_signals_per_validator": 1000,
  "signal_submission_cooldown": 10,
  "batch_processing_enabled": true,
  "max_batch_size": 50
}
*/