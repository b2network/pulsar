package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/b2network/pulsar/app"
	abciserver "github.com/cometbft/cometbft/abci/server"
	cfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/libs/log"
	nm "github.com/cometbft/cometbft/node"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/privval"
	"github.com/cometbft/cometbft/proxy"
	"github.com/spf13/cobra"
)

func startCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start the Pulsar node",
		RunE: func(cmd *cobra.Command, args []string) error {
			homeDir, _ := cmd.Flags().GetString("home")
			return startNode(homeDir)
		},
	}
}

func startNode(homeDir string) error {
	config := cfg.DefaultConfig()
	config.SetRoot(homeDir)

	configFile := filepath.Join(homeDir, "config", "config.toml")
	if _, err := os.Stat(configFile); err == nil {
		viper := cfg.DefaultConfig()
		viper.SetRoot(homeDir)
		if err := viper.ValidateBasic(); err != nil {
			return fmt.Errorf("config is invalid: %w", err)
		}
		config = viper
	}

	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))

	// Create Pulsar application
	pulsarApp := app.NewPulsarApp(logger, homeDir)

	server := abciserver.NewSocketServer("tcp://127.0.0.1:26658", pulsarApp)
	server.SetLogger(logger.With("module", "abci-server"))

	if err := server.Start(); err != nil {
		return fmt.Errorf("failed to start ABCI server: %w", err)
	}
	defer server.Stop()

	nodeKeyFile := filepath.Join(homeDir, "config", "node_key.json")
	nodeKey, err := p2p.LoadNodeKey(nodeKeyFile)
	if err != nil {
		return fmt.Errorf("failed to load node key: %w", err)
	}

	pvKeyFile := filepath.Join(homeDir, "config", "priv_validator_key.json")
	pvStateFile := filepath.Join(homeDir, "data", "priv_validator_state.json")
	pv := privval.LoadFilePV(pvKeyFile, pvStateFile)

	node, err := nm.NewNode(
		config,
		pv,
		nodeKey,
		proxy.NewLocalClientCreator(pulsarApp),
		nm.DefaultGenesisDocProviderFunc(config),
		cfg.DefaultDBProvider,
		nm.DefaultMetricsProvider(config.Instrumentation),
		logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create node: %w", err)
	}

	if err := node.Start(); err != nil {
		return fmt.Errorf("failed to start node: %w", err)
	}
	defer func() {
		node.Stop()
		node.Wait()
	}()

	fmt.Printf("\nNode started successfully:\n")
	fmt.Printf("  Node ID: %s\n", nodeKey.ID())
	fmt.Printf("  RPC: %s\n", config.RPC.ListenAddress)
	fmt.Printf("  P2P: %s\n", config.P2P.ListenAddress)
	fmt.Printf("\nPress Ctrl+C to stop the node\n")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	fmt.Println("\nShutting down...")
	return nil
}
