package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	
	bankcli "github.com/b2network/pulsar/modules/bank/client/cli"
	coincli "github.com/b2network/pulsar/modules/coin/client/cli"
)

var (
	// Version is set during build
	Version = "development"
	Commit  = "unknown"
)

func main() {
	rootCmd := NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// NewRootCmd creates the root command for pulsarcli
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "pulsarcli",
		Short: "Command line interface for Pulsar blockchain",
		Long: `Pulsar CLI is a command-line tool to interact with the Pulsar blockchain.

It allows you to:
- Query blockchain state
- Create and broadcast transactions
- Manage accounts and keys
- Interact with smart contracts`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Initialize any global configurations here
			return nil
		},
	}

	// Add sub-commands
	rootCmd.AddCommand(
		txCmd(),
		queryCmd(),
		keysCmd(),
		versionCmd(),
		configCmd(),
	)

	// Add global flags
	rootCmd.PersistentFlags().String("home", os.ExpandEnv("$HOME/.pulsar"), "Directory for config and data")
	rootCmd.PersistentFlags().String("chain-id", "", "The network chain ID")
	rootCmd.PersistentFlags().Bool("trace", false, "Print full stack trace on errors")
	rootCmd.PersistentFlags().String("log-level", "info", "Set the logging level (trace|debug|info|warn|error|fatal|panic)")
	rootCmd.PersistentFlags().String("log-format", "plain", "Set the logging format (json|plain)")

	return rootCmd
}

// txCmd returns the transaction command
func txCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tx",
		Short: "Construct and broadcast transactions",
		Long:  `Construct and broadcast transactions to the Pulsar network.`,
		RunE:  nil,
	}

	// Add module tx commands
	cmd.AddCommand(
		bankcli.GetTxCmd(),
		coincli.GetTxCmd(),
		// Future: Add other module tx commands here
		// stakingcli.GetTxCmd(),
		// govcli.GetTxCmd(),
	)

	return cmd
}

// queryCmd returns the query command
func queryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "query",
		Aliases: []string{"q"},
		Short:   "Query blockchain state",
		Long:    `Query blockchain state including accounts, balances, transactions, and more.`,
		RunE:    nil,
	}

	// Add module query commands
	cmd.AddCommand(
		bankcli.GetQueryCmd(),
		coincli.GetQueryCmd(),
		// Future: Add other module query commands here
		// stakingcli.GetQueryCmd(),
		// govcli.GetQueryCmd(),
	)

	// Add direct query commands
	cmd.AddCommand(
		queryTxCmd(),
		queryBlockCmd(),
	)

	return cmd
}

// keysCmd returns the key management command
func keysCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keys",
		Short: "Manage application keys",
		Long:  `Keys management commands to manage local keys for signing transactions.`,
	}

	cmd.AddCommand(
		keysAddCmd(),
		keysListCmd(),
		keysShowCmd(),
		keysDeleteCmd(),
		keysExportCmd(),
		keysImportCmd(),
	)

	return cmd
}

// versionCmd returns the version command
func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the application version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Pulsar CLI v%s\n", Version)
			fmt.Printf("Commit: %s\n", Commit)
			fmt.Printf("Go version: %s\n", getGoVersion())
			return nil
		},
	}
}

// configCmd returns the config command
func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage client configuration",
		Long:  `Manage client configuration including node endpoints, chain ID, and other settings.`,
	}

	cmd.AddCommand(
		configSetCmd(),
		configGetCmd(),
		configResetCmd(),
	)

	return cmd
}

// Query sub-commands
func queryTxCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tx [hash]",
		Short: "Query for a transaction by hash",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			txHash := args[0]
			fmt.Printf("Querying transaction: %s\n", txHash)
			// TODO: Implement actual query
			return nil
		},
	}
}

func queryBlockCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "block [height]",
		Short: "Query for a block by height",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			height := "latest"
			if len(args) > 0 {
				height = args[0]
			}
			fmt.Printf("Querying block at height: %s\n", height)
			// TODO: Implement actual query
			return nil
		},
	}
}

// Keys sub-commands
func keysAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add [name]",
		Short: "Add a new key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			fmt.Printf("Adding new key: %s\n", name)
			// TODO: Implement key generation
			return nil
		},
	}
}

func keysListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Listing all keys...")
			// TODO: Implement key listing
			return nil
		},
	}
}

func keysShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show [name]",
		Short: "Show key information",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			fmt.Printf("Showing key: %s\n", name)
			// TODO: Implement key display
			return nil
		},
	}
}

func keysDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete [name]",
		Short: "Delete a key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			fmt.Printf("Deleting key: %s\n", name)
			// TODO: Implement key deletion
			return nil
		},
	}
}

func keysExportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export [name]",
		Short: "Export a key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			fmt.Printf("Exporting key: %s\n", name)
			// TODO: Implement key export
			return nil
		},
	}
}

func keysImportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import [name] [file]",
		Short: "Import a key from file",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			file := args[1]
			fmt.Printf("Importing key %s from %s\n", name, file)
			// TODO: Implement key import
			return nil
		},
	}
}

// Config sub-commands
func configSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set [key] [value]",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			value := args[1]
			fmt.Printf("Setting %s = %s\n", key, value)
			// TODO: Implement config set
			return nil
		},
	}
}

func configGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get [key]",
		Short: "Get a configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			fmt.Printf("Getting config: %s\n", key)
			// TODO: Implement config get
			return nil
		},
	}
}

func configResetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: "Reset configuration to defaults",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Resetting configuration to defaults...")
			// TODO: Implement config reset
			return nil
		},
	}
}

// getGoVersion returns the Go version
func getGoVersion() string {
	// In production, this would use runtime.Version()
	return "go1.23.5"
}