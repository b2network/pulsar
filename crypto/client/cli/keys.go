package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/b2network/pulsar/crypto/keys"
)

const (
	flagKeyName     = "name"
	flagKeyType     = "keyring-backend"
	flagMnemonic    = "mnemonic"
	flagInteractive = "interactive"
	flagIndex       = "index"
	flagRecover     = "recover"
	flagNoBackup    = "no-backup"
	flagDryRun      = "dry-run"
	flagShowMnemonic = "show-mnemonic"
)

// GetKeysCmd returns the keys command
func GetKeysCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keys",
		Short: "Manage your application's keys",
		Long: `Keys allow you to manage your local keystore for signing transactions.
These keys may be in any format supported by the Pulsar keyring.`,
	}

	cmd.AddCommand(
		AddKeyCmd(),
		ListKeysCmd(),
		ShowKeyCmd(),
		DeleteKeyCmd(),
		ExportKeyCmd(),
		ImportKeyCmd(),
	)

	return cmd
}

// AddKeyCmd returns a command to add a key
func AddKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add an encrypted private key (either newly generated or recovered)",
		Long: `Derive a new private key and encrypt to disk.
Optionally specify a BIP39 mnemonic, a BIP39 passphrase to further secure the mnemonic,
and a bip32 HD path to derive a specific account. The key will be stored under the given name
and encrypted with the given password. The only input that is required is the encryption password.

If run with -i, it will prompt the user for BIP44 path, BIP39 mnemonic, and passphrase.
The flag --recover allows one to recover a key from a seed passphrase.
If run with --dry-run, a key would be generated (or recovered) but not stored to the
local keystore.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// Get keyring directory
			homeDir, _ := cmd.Flags().GetString("home")
			keyDir := filepath.Join(homeDir, "keys")

			// Create keyring
			keyring, err := keys.NewFileKeyring(keyDir)
			if err != nil {
				return fmt.Errorf("failed to create keyring: %w", err)
			}

			// Check if key already exists
			if keyring.HasKey(name) {
				return fmt.Errorf("key with name %s already exists", name)
			}

			// Get flags
			recover, _ := cmd.Flags().GetBool(flagRecover)
			noBackup, _ := cmd.Flags().GetBool(flagNoBackup)
			dryRun, _ := cmd.Flags().GetBool(flagDryRun)

			var mnemonic string
			var passphrase string

			if recover {
				// Recovery mode
				fmt.Print("Enter your bip39 mnemonic: ")
				reader := bufio.NewReader(os.Stdin)
				mnemonic, err = reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("failed to read mnemonic: %w", err)
				}
				mnemonic = strings.TrimSpace(mnemonic)
			}

			if !dryRun {
				// Get passphrase for encryption
				passphrase, err = getPassphrase("Enter keyring passphrase:")
				if err != nil {
					return err
				}

				confirmPassphrase, err := getPassphrase("Re-enter keyring passphrase:")
				if err != nil {
					return err
				}

				if passphrase != confirmPassphrase {
					return fmt.Errorf("passphrases do not match")
				}
			}

			// Create the key
			keyInfo, err := keyring.CreateKey(name, mnemonic, passphrase)
			if err != nil {
				return fmt.Errorf("failed to create key: %w", err)
			}

			// Print result
			fmt.Printf("\n- name: %s\n", keyInfo.Name)
			fmt.Printf("  type: %s\n", keyInfo.Type)
			fmt.Printf("  address: %s\n", keys.FormatAddress(keyInfo.Address, true))
			fmt.Printf("  pubkey: '%s'\n", "0x"+fmt.Sprintf("%x", keyInfo.PubKey.Bytes()))

			if !noBackup && mnemonic != "" {
				fmt.Printf("  mnemonic: \"%s\"\n", mnemonic)
				fmt.Println("\n**Important** write this mnemonic phrase in a safe place.")
				fmt.Println("It is the only way to recover your account if you ever forget your password.")
			}

			return nil
		},
	}

	cmd.Flags().Bool(flagRecover, false, "Provide seed phrase to recover existing key instead of creating")
	cmd.Flags().Bool(flagInteractive, false, "Interactively prompt user for BIP39 passphrase and mnemonic")
	cmd.Flags().Bool(flagNoBackup, false, "Don't print out seed phrase (if others are watching the terminal)")
	cmd.Flags().Bool(flagDryRun, false, "Perform action, but don't add key to local keystore")
	cmd.Flags().String("home", "", "The application home directory")

	return cmd
}

// ListKeysCmd returns a command to list keys
func ListKeysCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all keys",
		Long:  "Return a list of all public keys stored by this keyring.",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get keyring directory
			homeDir, _ := cmd.Flags().GetString("home")
			keyDir := filepath.Join(homeDir, "keys")

			// Create keyring
			keyring, err := keys.NewFileKeyring(keyDir)
			if err != nil {
				return fmt.Errorf("failed to create keyring: %w", err)
			}

			// List keys
			keyInfos, err := keyring.ListKeys()
			if err != nil {
				return fmt.Errorf("failed to list keys: %w", err)
			}

			if len(keyInfos) == 0 {
				fmt.Println("No keys found.")
				return nil
			}

			for _, keyInfo := range keyInfos {
				fmt.Printf("- name: %s\n", keyInfo.Name)
				fmt.Printf("  type: %s\n", keyInfo.Type)
				fmt.Printf("  address: %s\n", keys.FormatAddress(keyInfo.Address, true))
				fmt.Printf("  pubkey: '%s'\n", "0x"+fmt.Sprintf("%x", keyInfo.PubKey.Bytes()))
				fmt.Println()
			}

			return nil
		},
	}

	cmd.Flags().String("home", "", "The application home directory")
	return cmd
}

// ShowKeyCmd returns a command to show key info
func ShowKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <name>",
		Short: "Retrieve key information by name or address",
		Long:  "Display keys details. If multiple keys are selected, then an ephemeral multisig key will be created.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// Get keyring directory
			homeDir, _ := cmd.Flags().GetString("home")
			keyDir := filepath.Join(homeDir, "keys")

			// Create keyring
			keyring, err := keys.NewFileKeyring(keyDir)
			if err != nil {
				return fmt.Errorf("failed to create keyring: %w", err)
			}

			// Get key
			keyInfo, err := keyring.GetKey(name)
			if err != nil {
				return fmt.Errorf("failed to get key: %w", err)
			}

			// Print key info
			fmt.Printf("- name: %s\n", keyInfo.Name)
			fmt.Printf("  type: %s\n", keyInfo.Type)
			fmt.Printf("  address: %s\n", keys.FormatAddress(keyInfo.Address, true))
			fmt.Printf("  pubkey: '%s'\n", "0x"+fmt.Sprintf("%x", keyInfo.PubKey.Bytes()))

			return nil
		},
	}

	cmd.Flags().String("home", "", "The application home directory")
	return cmd
}

// DeleteKeyCmd returns a command to delete a key
func DeleteKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete the given keys",
		Long:  "Delete a key from the store.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// Get keyring directory
			homeDir, _ := cmd.Flags().GetString("home")
			keyDir := filepath.Join(homeDir, "keys")

			// Create keyring
			keyring, err := keys.NewFileKeyring(keyDir)
			if err != nil {
				return fmt.Errorf("failed to create keyring: %w", err)
			}

			// Check if key exists
			if !keyring.HasKey(name) {
				return fmt.Errorf("key %s not found", name)
			}

			// Confirm deletion
			fmt.Printf("Are you sure you want to delete key %s? [y/N]: ", name)
			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read response: %w", err)
			}

			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				fmt.Println("Cancelled.")
				return nil
			}

			// Get passphrase
			passphrase, err := getPassphrase("Enter keyring passphrase:")
			if err != nil {
				return err
			}

			// Delete key
			err = keyring.DeleteKey(name, passphrase)
			if err != nil {
				return fmt.Errorf("failed to delete key: %w", err)
			}

			fmt.Printf("Key %s deleted successfully.\n", name)
			return nil
		},
	}

	cmd.Flags().String("home", "", "The application home directory")
	return cmd
}

// ExportKeyCmd returns a command to export a private key
func ExportKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export <name>",
		Short: "Export private keys",
		Long:  "Export a private key from the local keyring in encrypted format.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// Get keyring directory
			homeDir, _ := cmd.Flags().GetString("home")
			keyDir := filepath.Join(homeDir, "keys")

			// Create keyring
			keyring, err := keys.NewFileKeyring(keyDir)
			if err != nil {
				return fmt.Errorf("failed to create keyring: %w", err)
			}

			// Check if key exists
			if !keyring.HasKey(name) {
				return fmt.Errorf("key %s not found", name)
			}

			// Get passphrase
			passphrase, err := getPassphrase("Enter keyring passphrase:")
			if err != nil {
				return err
			}

			// Export key
			privKeyHex, err := keyring.ExportPrivKey(name, passphrase)
			if err != nil {
				return fmt.Errorf("failed to export key: %w", err)
			}

			fmt.Printf("Private key: %s\n", privKeyHex)
			fmt.Println("\n**WARNING**: Never share your private key with anyone!")
			return nil
		},
	}

	cmd.Flags().String("home", "", "The application home directory")
	return cmd
}

// ImportKeyCmd returns a command to import a private key
func ImportKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import <name> <keyfile>",
		Short: "Import private keys into the local keyring",
		Long:  "Import a ASCII armored private key into the local keyring.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			keyfile := args[1]

			// Get keyring directory
			homeDir, _ := cmd.Flags().GetString("home")
			keyDir := filepath.Join(homeDir, "keys")

			// Create keyring
			keyring, err := keys.NewFileKeyring(keyDir)
			if err != nil {
				return fmt.Errorf("failed to create keyring: %w", err)
			}

			// Check if key already exists
			if keyring.HasKey(name) {
				return fmt.Errorf("key with name %s already exists", name)
			}

			// Read private key from file or input
			var privKeyHex string
			if keyfile == "-" {
				// Read from stdin
				fmt.Print("Enter private key: ")
				reader := bufio.NewReader(os.Stdin)
				privKeyHex, err = reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("failed to read private key: %w", err)
				}
				privKeyHex = strings.TrimSpace(privKeyHex)
			} else {
				// Read from file
				data, err := os.ReadFile(keyfile)
				if err != nil {
					return fmt.Errorf("failed to read key file: %w", err)
				}
				privKeyHex = strings.TrimSpace(string(data))
			}

			// Get passphrase
			passphrase, err := getPassphrase("Enter passphrase to encrypt the imported key:")
			if err != nil {
				return err
			}

			// Import key
			err = keyring.ImportPrivKey(name, privKeyHex, passphrase)
			if err != nil {
				return fmt.Errorf("failed to import key: %w", err)
			}

			// Get key info to display
			keyInfo, err := keyring.GetKey(name)
			if err != nil {
				return fmt.Errorf("failed to get imported key info: %w", err)
			}

			fmt.Printf("Key %s imported successfully.\n", name)
			fmt.Printf("Address: %s\n", keys.FormatAddress(keyInfo.Address, true))
			return nil
		},
	}

	cmd.Flags().String("home", "", "The application home directory")
	return cmd
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

// outputJSON outputs the given value as JSON
func outputJSON(v interface{}) error {
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}