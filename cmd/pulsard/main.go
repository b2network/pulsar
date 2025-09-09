package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	Version = "dev"
	Commit  = "unknown"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "pulsard",
		Short: "Pulsar blockchain node",
		Long:  "Pulsar is a blockchain network based on CometBFT consensus",
	}

	homeDir := os.ExpandEnv("$HOME/.pulsar")
	rootCmd.PersistentFlags().String("home", homeDir, "Home directory for config and data")

	rootCmd.AddCommand(
		initCmd(),
		startCmd(),
		versionCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version info",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Pulsar %s\n", Version)
			fmt.Printf("Commit: %s\n", Commit)
			fmt.Printf("Go: %s\n", runtime.Version())
			fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		},
	}
}
