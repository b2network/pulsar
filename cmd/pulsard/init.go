package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	banktypes "github.com/b2network/pulsar/modules/bank/types"
	govtypes "github.com/b2network/pulsar/modules/gov/types"
	stakingtypes "github.com/b2network/pulsar/modules/staking/types"
	"github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/privval"
	"github.com/cometbft/cometbft/types"
	"github.com/spf13/cobra"
)

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init [moniker]",
		Short: "Initialize the Pulsar node",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			moniker := args[0]
			homeDir, _ := cmd.Flags().GetString("home")

			return initNode(homeDir, moniker)
		},
	}
}

func initNode(homeDir, moniker string) error {
	configDir := filepath.Join(homeDir, "config")
	dataDir := filepath.Join(homeDir, "data")

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	cfg := config.DefaultConfig()
	cfg.SetRoot(homeDir)
	cfg.Moniker = moniker
	cfg.ProxyApp = "tcp://127.0.0.1:26658"
	cfg.RPC.ListenAddress = "tcp://127.0.0.1:26657"
	cfg.P2P.ListenAddress = "tcp://0.0.0.0:26656"

	config.WriteConfigFile(filepath.Join(configDir, "config.toml"), cfg)

	pvKeyFile := filepath.Join(configDir, "priv_validator_key.json")
	pvStateFile := filepath.Join(dataDir, "priv_validator_state.json")

	pv := privval.GenFilePV(pvKeyFile, pvStateFile)
	pv.Save()

	nodeKeyFile := filepath.Join(configDir, "node_key.json")
	nodeKey, err := p2p.LoadOrGenNodeKey(nodeKeyFile)
	if err != nil {
		return fmt.Errorf("failed to load or generate node key: %w", err)
	}

	pubKey, err := pv.GetPubKey()
	if err != nil {
		return fmt.Errorf("failed to get public key: %w", err)
	}

	// Create application genesis state
	appState := make(map[string]json.RawMessage)

	// Bank module genesis
	bankGenesis := banktypes.DefaultGenesisState()
	bankGenesisBytes, err := json.Marshal(bankGenesis)
	if err != nil {
		return fmt.Errorf("failed to marshal bank genesis: %w", err)
	}
	appState[banktypes.ModuleName] = bankGenesisBytes

	// Staking module genesis
	stakingGenesis := stakingtypes.DefaultGenesisState()
	stakingGenesisBytes, err := json.Marshal(stakingGenesis)
	if err != nil {
		return fmt.Errorf("failed to marshal staking genesis: %w", err)
	}
	appState[stakingtypes.ModuleName] = stakingGenesisBytes

	// Gov module genesis
	govGenesis := govtypes.DefaultGenesisState()
	govGenesisBytes, err := json.Marshal(govGenesis)
	if err != nil {
		return fmt.Errorf("failed to marshal gov genesis: %w", err)
	}
	appState[govtypes.ModuleName] = govGenesisBytes

	appStateBytes, err := json.Marshal(appState)
	if err != nil {
		return fmt.Errorf("failed to marshal app state: %w", err)
	}

	genDoc := &types.GenesisDoc{
		ChainID:         "pulsar-1",
		GenesisTime:     time.Now(),
		ConsensusParams: types.DefaultConsensusParams(),
		AppState:        appStateBytes,
		Validators: []types.GenesisValidator{
			{
				Address: pubKey.Address(),
				PubKey:  pubKey,
				Power:   10,
			},
		},
	}

	if err := genDoc.SaveAs(filepath.Join(configDir, "genesis.json")); err != nil {
		return fmt.Errorf("failed to save genesis file: %w", err)
	}

	fmt.Printf("Node initialized successfully:\n")
	fmt.Printf("  Home directory: %s\n", homeDir)
	fmt.Printf("  Node ID: %s\n", nodeKey.ID())
	fmt.Printf("  Validator pubkey: %s\n", pubKey)

	return nil
}
