package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/b2network/pulsar/client/tx"
	"github.com/b2network/pulsar/modules/bank/types"
	commontypes "github.com/b2network/pulsar/types"
)

// GetTxCmd returns the transaction commands for the bank module
func GetTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Bank transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       nil,
	}

	txCmd.AddCommand(
		NewSendTxCmd(),
		NewMultiSendTxCmd(),
	)

	return txCmd
}

// NewSendTxCmd returns a CLI command handler for creating a MsgSend transaction
func NewSendTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send [from_address] [to_address] [amount]",
		Short: "Send coins from one account to another",
		Long: `Send coins from one account to another.
Example:
$ pulsarcli tx bank send addr1 addr2 100ubtc
$ pulsarcli tx bank send addr1 addr2 100ubtc,50wei
`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			fromAddr := args[0]
			toAddr := args[1]
			amountStr := args[2]

			// Parse coins
			coins, err := parseCoins(amountStr)
			if err != nil {
				return fmt.Errorf("invalid amount: %w", err)
			}

			// Create message
			msg := &types.MsgSend{
				FromAddress: fromAddr,
				ToAddress:   toAddr,
				Amount:      coins,
			}

			// Validate message
			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			// Prepare and execute transaction
			msgs := []commontypes.Msg{msg}
			txConfig := tx.TxCliConfig{
				ChainID: "", // Will be derived from flags
				HomeDir: "", // Will be derived from flags
			}

			return tx.PrepareAndExecuteTx(cmd, msgs, txConfig)
		},
	}

	// Add common flags
	addTxFlags(cmd)

	return cmd
}

// NewMultiSendTxCmd returns a CLI command handler for creating a MsgMultiSend transaction
func NewMultiSendTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "multi-send [inputs] [outputs]",
		Short: "Send coins from multiple inputs to multiple outputs",
		Long: `Send coins from multiple inputs to multiple outputs.
Example:
$ pulsarcli tx bank multi-send addr1:100ubtc,addr2:50ubtc addr3:80ubtc,addr4:70ubtc
`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			inputsStr := args[0]
			outputsStr := args[1]

			// Parse inputs
			inputs, err := parseInputs(inputsStr)
			if err != nil {
				return fmt.Errorf("invalid inputs: %w", err)
			}

			// Parse outputs
			outputs, err := parseOutputs(outputsStr)
			if err != nil {
				return fmt.Errorf("invalid outputs: %w", err)
			}

			// Calculate total amount
			totalCoins := calculateTotalCoins(inputs, outputs)

			// Create message
			msg := &types.MsgMultiSend{
				Inputs:  inputs,
				Outputs: outputs,
				Amount:  totalCoins,
			}

			// Validate message
			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			// Prepare and execute transaction
			msgs := []commontypes.Msg{msg}
			txConfig := tx.TxCliConfig{
				ChainID: "", // Will be derived from flags
				HomeDir: "", // Will be derived from flags
			}

			return tx.PrepareAndExecuteTx(cmd, msgs, txConfig)
		},
	}

	// Add common flags
	addTxFlags(cmd)

	return cmd
}

// parseCoins parses a coins string into Coins
func parseCoins(coinsStr string) (commontypes.Coins, error) {
	coinsStr = strings.TrimSpace(coinsStr)
	if len(coinsStr) == 0 {
		return nil, fmt.Errorf("empty coins string")
	}

	coinStrs := strings.Split(coinsStr, ",")
	coins := make(commontypes.Coins, 0, len(coinStrs))

	for _, coinStr := range coinStrs {
		// Simple parsing: extract amount and denom
		// Format: 100ubtc
		amount, denom, err := types.ParseDenomAmount(coinStr)
		if err != nil {
			return nil, err
		}

		coins = append(coins, commontypes.Coin{
			Denom:  denom,
			Amount: amount,
		})
	}

	return coins, nil
}

// parseInputs parses input string into Input slice
func parseInputs(inputsStr string) ([]types.Input, error) {
	// Format: addr1:100ubtc,addr2:50ubtc
	parts := strings.Split(inputsStr, ",")
	inputs := make([]types.Input, 0, len(parts))

	for _, part := range parts {
		addrAmount := strings.Split(part, ":")
		if len(addrAmount) != 2 {
			return nil, fmt.Errorf("invalid input format: %s", part)
		}

		coins, err := parseCoins(addrAmount[1])
		if err != nil {
			return nil, err
		}

		inputs = append(inputs, types.Input{
			Address: addrAmount[0],
			Coins:   coins,
		})
	}

	return inputs, nil
}

// parseOutputs parses output string into Output slice
func parseOutputs(outputsStr string) ([]types.Output, error) {
	// Format: addr3:80ubtc,addr4:70ubtc
	parts := strings.Split(outputsStr, ",")
	outputs := make([]types.Output, 0, len(parts))

	for _, part := range parts {
		addrAmount := strings.Split(part, ":")
		if len(addrAmount) != 2 {
			return nil, fmt.Errorf("invalid output format: %s", part)
		}

		coins, err := parseCoins(addrAmount[1])
		if err != nil {
			return nil, err
		}

		outputs = append(outputs, types.Output{
			Address: addrAmount[0],
			Coins:   coins,
		})
	}

	return outputs, nil
}

// calculateTotalCoins calculates total coins from inputs and outputs
func calculateTotalCoins(inputs []types.Input, outputs []types.Output) commontypes.Coins {
	// For simplicity, just return empty coins
	// In production, this would calculate the actual totals
	return commontypes.Coins{}
}

// addTxFlags adds common transaction flags
func addTxFlags(cmd *cobra.Command) {
	tx.AddTxFlags(cmd)
}