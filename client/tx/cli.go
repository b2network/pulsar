package tx

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	commontypes "github.com/b2network/pulsar/types"
)

// TxCliConfig contains CLI configuration for transactions
type TxCliConfig struct {
	ChainID string
	HomeDir string
}

// PrepareAndExecuteTx prepares and executes a transaction
func PrepareAndExecuteTx(cmd *cobra.Command, msgs []commontypes.Msg, config TxCliConfig) error {
	// Get flags
	fromFlag, _ := cmd.Flags().GetString("from")
	feesFlag, _ := cmd.Flags().GetString("fees")
	gasFlag, _ := cmd.Flags().GetString("gas")
	gasPricesFlag, _ := cmd.Flags().GetString("gas-prices")
	memoFlag, _ := cmd.Flags().GetString("memo")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	generateOnly, _ := cmd.Flags().GetBool("generate-only")

	// Get home directory for keyring
	homeDir := config.HomeDir
	if homeDir == "" {
		homeDir, _ = cmd.Flags().GetString("home")
		if homeDir == "" {
			homeDir = os.ExpandEnv("$HOME/.pulsar")
		}
	}

	// Get chain ID
	chainID := config.ChainID
	if chainID == "" {
		chainID, _ = cmd.Flags().GetString("chain-id")
		if chainID == "" {
			chainID = "pulsar-1" // Default chain ID
		}
	}

	// Create transaction builder
	keyDir := filepath.Join(homeDir, "keys")
	txBuilder, err := NewTxBuilder(chainID, keyDir)
	if err != nil {
		return fmt.Errorf("failed to create transaction builder: %w", err)
	}

	// Validate from flag
	if fromFlag == "" {
		return fmt.Errorf("--from flag is required")
	}

	// Check if key exists
	keyring := txBuilder.GetKeyring()
	if !keyring.HasKey(fromFlag) {
		return fmt.Errorf("key %s not found. Use 'pulsarcli keys list' to see available keys", fromFlag)
	}

	// Get key info
	keyInfo, err := keyring.GetKey(fromFlag)
	if err != nil {
		return fmt.Errorf("failed to get key info: %w", err)
	}

	fmt.Printf("Using key: %s\n", keyInfo.Name)
	fmt.Printf("Address: %s\n", keyInfo.Address.String())

	// Estimate gas if auto
	var gasLimit uint64 = 200000 // Default
	if gasFlag == "auto" {
		estimated, err := txBuilder.EstimateGas(msgs)
		if err != nil {
			return fmt.Errorf("failed to estimate gas: %w", err)
		}
		gasLimit = estimated
		fmt.Printf("Estimated gas: %d\n", gasLimit)
	} else if gasFlag != "" {
		parsed, err := strconv.ParseUint(gasFlag, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid gas limit: %w", err)
		}
		gasLimit = parsed
	}

	// Build transaction config
	txConfig := TxConfig{
		From:         fromFlag,
		Fees:         feesFlag,
		Gas:          fmt.Sprintf("%d", gasLimit),
		GasPrices:    gasPricesFlag,
		Memo:         memoFlag,
		AccountNum:   0, // Would get from chain query in production
		Sequence:     0, // Would get from chain query in production
		TimeoutHeight: 0,
	}

	// If dry run, just validate and show what would be sent
	if dryRun {
		fmt.Println("\n--- DRY RUN ---")
		fmt.Printf("Chain ID: %s\n", chainID)
		fmt.Printf("From: %s (%s)\n", keyInfo.Name, keyInfo.Address.String())
		fmt.Printf("Gas: %d\n", gasLimit)
		fmt.Printf("Fees: %s\n", feesFlag)
		fmt.Printf("Memo: %s\n", memoFlag)
		fmt.Printf("Messages: %d\n", len(msgs))
		for i, msg := range msgs {
			fmt.Printf("  %d: %T\n", i+1, msg)
		}
		fmt.Println("--- END DRY RUN ---")
		return nil
	}

	// Get passphrase for signing
	passphrase, err := getPassphrase("Enter passphrase to unlock key:")
	if err != nil {
		return fmt.Errorf("failed to get passphrase: %w", err)
	}

	// Build and sign transaction
	fmt.Println("Building transaction...")
	tx, err := txBuilder.BuildAndSign(msgs, txConfig, passphrase)
	if err != nil {
		return fmt.Errorf("failed to build and sign transaction: %w", err)
	}

	fmt.Println("Transaction signed successfully!")

	// If generate only, just output the transaction
	if generateOnly {
		fmt.Println("\n--- GENERATED TRANSACTION ---")
		fmt.Printf("Chain ID: %s\n", chainID)
		fmt.Printf("Signer: %s\n", keyInfo.Address.String())
		fmt.Printf("Gas: %d\n", gasLimit)
		fmt.Printf("Signatures: %d\n", len(tx.Signatures))
		fmt.Println("--- END GENERATED TRANSACTION ---")
		return nil
	}

	// Verify transaction before broadcasting
	fmt.Println("Verifying transaction...")
	if err := txBuilder.VerifyTx(tx); err != nil {
		// Don't fail on verification error for now, just warn
		fmt.Printf("Warning: Transaction verification failed: %v\n", err)
	}

	// Ask for confirmation
	if !confirmBroadcast() {
		fmt.Println("Transaction cancelled.")
		return nil
	}

	// Broadcast transaction
	fmt.Println("Broadcasting transaction...")
	txHash, err := txBuilder.BroadcastTx(tx)
	if err != nil {
		return fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	fmt.Printf("Transaction broadcast successfully!\n")
	fmt.Printf("Transaction hash: %s\n", txHash)

	return nil
}

// getPassphrase securely reads a passphrase from stdin
func getPassphrase(prompt string) (string, error) {
	fmt.Print(prompt + " ")
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", err
	}
	fmt.Println()
	return string(bytePassword), nil
}

// confirmBroadcast asks user to confirm transaction broadcast
func confirmBroadcast() bool {
	fmt.Print("Do you want to broadcast this transaction? [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

// AddTxFlags adds common transaction flags to a command
func AddTxFlags(cmd *cobra.Command) {
	cmd.Flags().String("from", "", "Name or address of account that signs the transaction (required)")
	cmd.Flags().String("fees", "", "Fees to pay for the transaction (e.g., 100ubtc)")
	cmd.Flags().String("gas", "auto", "Gas limit to set per-transaction; set to 'auto' to calculate automatically")
	cmd.Flags().String("gas-prices", "", "Gas prices to determine the transaction fee (e.g., 0.1ubtc)")
	cmd.Flags().String("memo", "", "Memo to include in the transaction")
	cmd.Flags().Bool("dry-run", false, "Perform a dry run without broadcasting")
	cmd.Flags().Bool("generate-only", false, "Generate transaction without broadcasting")
	cmd.Flags().String("chain-id", "", "The network chain ID")
	cmd.Flags().String("home", "", "Directory for config and data")

	// Mark from as required
	cmd.MarkFlagRequired("from")
}

// ValidateBasicTxFlags validates basic transaction flags
func ValidateBasicTxFlags(cmd *cobra.Command) error {
	fromFlag, _ := cmd.Flags().GetString("from")
	if fromFlag == "" {
		return fmt.Errorf("--from flag is required")
	}

	gasFlag, _ := cmd.Flags().GetString("gas")
	if gasFlag != "auto" {
		if _, err := strconv.ParseUint(gasFlag, 10, 64); err != nil {
			return fmt.Errorf("invalid gas limit: %s", gasFlag)
		}
	}

	return nil
}