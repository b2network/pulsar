package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cometbft/cometbft/libs/log"
	abcitypes "github.com/cometbft/cometbft/abci/types"

	"github.com/b2network/pulsar/store/rootstore"
	"github.com/b2network/pulsar/store/storage/memory"
	"github.com/b2network/pulsar/store/commitment"
	keepertypes "github.com/b2network/pulsar/keeper/types"
	
	// Module keepers
	bankkeeper "github.com/b2network/pulsar/modules/bank/keeper"
	banktypes "github.com/b2network/pulsar/modules/bank/types"
	stakingkeeper "github.com/b2network/pulsar/modules/staking/keeper"
	stakingtypes "github.com/b2network/pulsar/modules/staking/types"
	govkeeper "github.com/b2network/pulsar/modules/gov/keeper"
	govtypes "github.com/b2network/pulsar/modules/gov/types"
	
	storetypes "github.com/b2network/pulsar/store/types"
)

const (
	appName = "PulsarApp"
)

// PulsarApp represents the main application
type PulsarApp struct {
	// Core components
	logger    log.Logger
	cms       storetypes.CommitMultiStore
	ctx       keepertypes.Context
	codec     keepertypes.Codec
	authority keepertypes.Authority

	// Module keepers
	BankKeeper    bankkeeper.Keeper
	StakingKeeper stakingkeeper.Keeper
	GovKeeper     govkeeper.Keeper

	// Store keys
	keys map[string]storetypes.StoreKey
}

// NewPulsarApp creates a new PulsarApp instance
func NewPulsarApp(
	logger log.Logger,
	dataDir string,
) *PulsarApp {
	// Create codec
	codec := keepertypes.NewJSONCodec()
	
	// Create authority with default authorized address
	authority := keepertypes.NewBasicAuthority([]string{"pulsar1admin"})

	// Initialize store keys
	keys := map[string]storetypes.StoreKey{
		banktypes.StoreKey:    storetypes.NewKVStoreKey(banktypes.StoreKey),
		stakingtypes.StoreKey: storetypes.NewKVStoreKey(stakingtypes.StoreKey),
		govtypes.StoreKey:     storetypes.NewKVStoreKey(govtypes.StoreKey),
	}

	// Create storage backend - use memory backend for simplicity
	memBackend := memory.NewBackend()
	storageBackend := memory.NewStateStorageAdapter(memBackend)

	// Create commitment backend
	commitmentBackend := commitment.NewMockTreeBackend()

	// Create commit multi store
	cms := rootstore.NewStore(rootstore.StoreConfig{
		StateStorage:    storageBackend,
		StateCommitment: commitmentBackend,
		Pruning:         storetypes.PruningOptions{},
		InitialVersion:  1,
	})

	// Mount stores
	for _, key := range keys {
		cms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, nil)
	}

	// Load latest version
	if err := cms.LoadLatestVersion(); err != nil {
		panic(fmt.Sprintf("failed to load latest version: %v", err))
	}

	app := &PulsarApp{
		logger:    logger,
		cms:       cms,
		codec:     codec,
		authority: authority,
		keys:      keys,
	}

	// Initialize context
	app.ctx = keepertypes.NewPulsarContext(
		cms,
		0, // initial block height
		"pulsar-1", // chain ID
		logger,
	)

	// Initialize keepers
	app.initKeepers()

	return app
}

// initKeepers initializes all module keepers
func (app *PulsarApp) initKeepers() {
	// Bank keeper
	app.BankKeeper = *bankkeeper.NewKeeper(
		app.keys[banktypes.StoreKey],
		app.codec,
		nil, // account keeper - simplified for now
		nil, // module account permissions
	)

	// Staking keeper
	app.StakingKeeper = *stakingkeeper.NewKeeper(
		app.keys[stakingtypes.StoreKey],
		app.codec,
		app.BankKeeper, // bank keeper for token operations
		nil,           // slashing keeper - simplified for now
		banktypes.ModuleName, // authority
	)

	// Gov keeper
	app.GovKeeper = *govkeeper.NewKeeper(
		app.keys[govtypes.StoreKey],
		app.codec,
		app.BankKeeper,    // bank keeper for deposits
		app.StakingKeeper, // staking keeper for voting power
		govtypes.ModuleName, // authority
	)
}

// Name returns the app name
func (app *PulsarApp) Name() string {
	return appName
}

// ABCI Interface Implementation

// Info implements ABCI Info method
func (app *PulsarApp) Info(ctx context.Context, req *abcitypes.RequestInfo) (*abcitypes.ResponseInfo, error) {
	version := app.cms.LastCommitID()
	return &abcitypes.ResponseInfo{
		Data:             appName,
		Version:          req.Version,
		AppVersion:       1,
		LastBlockHeight:  version.Version,
		LastBlockAppHash: version.Hash,
	}, nil
}

// Query implements ABCI Query method
func (app *PulsarApp) Query(ctx context.Context, req *abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	path := req.Path
	
	switch path {
	case "/bank/balance":
		// Query bank balance
		var addr []byte
		if err := app.codec.Unmarshal(req.Data, &addr); err != nil {
			return &abcitypes.ResponseQuery{
				Code: 1,
				Log:  fmt.Sprintf("failed to unmarshal address: %v", err),
			}, nil
		}
		
		balance := app.BankKeeper.GetBalance(app.ctx, addr, "stake")
		data, err := app.codec.Marshal(balance)
		if err != nil {
			return &abcitypes.ResponseQuery{
				Code: 1,
				Log:  fmt.Sprintf("failed to marshal balance: %v", err),
			}, nil
		}
		
		return &abcitypes.ResponseQuery{
			Code:  0,
			Value: data,
		}, nil
		
	case "/staking/validator":
		// Query validator info
		var addr []byte
		if err := app.codec.Unmarshal(req.Data, &addr); err != nil {
			return &abcitypes.ResponseQuery{
				Code: 1,
				Log:  fmt.Sprintf("failed to unmarshal validator address: %v", err),
			}, nil
		}
		
		validator, found := app.StakingKeeper.GetValidator(app.ctx, addr)
		if !found {
			return &abcitypes.ResponseQuery{
				Code: 1,
				Log:  "validator not found",
			}, nil
		}
		
		data, err := app.codec.Marshal(validator)
		if err != nil {
			return &abcitypes.ResponseQuery{
				Code: 1,
				Log:  fmt.Sprintf("failed to marshal validator: %v", err),
			}, nil
		}
		
		return &abcitypes.ResponseQuery{
			Code:  0,
			Value: data,
		}, nil
		
	case "/gov/proposal":
		// Query proposal
		var proposalID uint64
		if err := app.codec.Unmarshal(req.Data, &proposalID); err != nil {
			return &abcitypes.ResponseQuery{
				Code: 1,
				Log:  fmt.Sprintf("failed to unmarshal proposal ID: %v", err),
			}, nil
		}
		
		proposal, found := app.GovKeeper.GetProposal(app.ctx, proposalID)
		if !found {
			return &abcitypes.ResponseQuery{
				Code: 1,
				Log:  "proposal not found",
			}, nil
		}
		
		data, err := app.codec.Marshal(proposal)
		if err != nil {
			return &abcitypes.ResponseQuery{
				Code: 1,
				Log:  fmt.Sprintf("failed to marshal proposal: %v", err),
			}, nil
		}
		
		return &abcitypes.ResponseQuery{
			Code:  0,
			Value: data,
		}, nil
		
	default:
		return &abcitypes.ResponseQuery{
			Code: 1,
			Log:  fmt.Sprintf("unknown query path: %s", path),
		}, nil
	}
}

// CheckTx implements ABCI CheckTx method
func (app *PulsarApp) CheckTx(ctx context.Context, req *abcitypes.RequestCheckTx) (*abcitypes.ResponseCheckTx, error) {
	// Basic transaction validation
	if len(req.Tx) == 0 {
		return &abcitypes.ResponseCheckTx{
			Code: 1,
			Log:  "empty transaction",
		}, nil
	}
	
	return &abcitypes.ResponseCheckTx{
		Code: 0,
		Log:  "transaction passed check",
	}, nil
}

// InitChain implements ABCI InitChain method
func (app *PulsarApp) InitChain(ctx context.Context, req *abcitypes.RequestInitChain) (*abcitypes.ResponseInitChain, error) {
	// Initialize chain state
	app.ctx = keepertypes.NewPulsarContext(
		app.cms,
		req.InitialHeight,
		req.ChainId,
		app.logger,
	)
	
	// Initialize genesis state for each module
	var genesisState map[string]json.RawMessage
	if len(req.AppStateBytes) > 0 {
		if err := json.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
			return nil, fmt.Errorf("failed to unmarshal genesis state: %w", err)
		}
	}
	
	// TODO: Initialize modules with genesis state
	// For now, we skip detailed genesis initialization
	
	return &abcitypes.ResponseInitChain{}, nil
}

// FinalizeBlock implements ABCI FinalizeBlock method (ABCI++ 1.0)
func (app *PulsarApp) FinalizeBlock(ctx context.Context, req *abcitypes.RequestFinalizeBlock) (*abcitypes.ResponseFinalizeBlock, error) {
	// Update context with block info
	app.ctx = keepertypes.NewPulsarContext(
		app.cms,
		req.Height,
		app.ctx.ChainID(),
		app.logger,
	)
	
	// Process transactions
	txResults := make([]*abcitypes.ExecTxResult, len(req.Txs))
	for i, tx := range req.Txs {
		result := app.processTx(tx)
		txResults[i] = result
	}
	
	// End block processing
	events := app.endBlock()
	
	return &abcitypes.ResponseFinalizeBlock{
		TxResults: txResults,
		Events:    events,
	}, nil
}

// processTx processes a single transaction
func (app *PulsarApp) processTx(tx []byte) *abcitypes.ExecTxResult {
	if len(tx) == 0 {
		return &abcitypes.ExecTxResult{
			Code: 1,
			Log:  "empty transaction",
		}
	}
	
	// Simple transaction processing - in a real implementation,
	// this would decode and route transactions to appropriate modules
	return &abcitypes.ExecTxResult{
		Code: 0,
		Log:  "transaction executed",
	}
}

// endBlock performs end-block processing
func (app *PulsarApp) endBlock() []abcitypes.Event {
	var events []abcitypes.Event
	
	// End block processing for each module would go here
	// For now, return empty events
	
	return events
}

// Commit implements ABCI Commit method
func (app *PulsarApp) Commit(ctx context.Context, req *abcitypes.RequestCommit) (*abcitypes.ResponseCommit, error) {
	// Commit the multi store
	_ = app.cms.Commit()
	
	return &abcitypes.ResponseCommit{
		RetainHeight: 0,
	}, nil
}

// Export exports the application state
func (app *PulsarApp) Export(forZeroHeight bool, jailAllowedAddrs []string, modulesToExport []string) (json.RawMessage, error) {
	// Create genesis state
	genesisState := make(map[string]json.RawMessage)
	
	// TODO: Export modules state
	// For now, return default genesis states
	bankGenesis := banktypes.DefaultGenesisState()
	bankData, _ := json.Marshal(bankGenesis)
	genesisState[banktypes.ModuleName] = bankData
	
	stakingGenesis := stakingtypes.DefaultGenesisState()
	stakingData, _ := json.Marshal(stakingGenesis)
	genesisState[stakingtypes.ModuleName] = stakingData
	
	govGenesis := govtypes.DefaultGenesisState()
	govData, _ := json.Marshal(govGenesis)
	genesisState[govtypes.ModuleName] = govData
	
	return json.Marshal(genesisState)
}

// Close closes the application
func (app *PulsarApp) Close() error {
	// Close any resources if needed
	return nil
}

// GetStoreKeys returns the store keys used by the app
func (app *PulsarApp) GetStoreKeys() map[string]storetypes.StoreKey {
	return app.keys
}

// GetKeepers returns all module keepers
func (app *PulsarApp) GetKeepers() (bankkeeper.Keeper, stakingkeeper.Keeper, govkeeper.Keeper) {
	return app.BankKeeper, app.StakingKeeper, app.GovKeeper
}

// Additional ABCI methods required by interface

// ExtendVote implements ABCI ExtendVote method (ABCI++ 2.0)
func (app *PulsarApp) ExtendVote(ctx context.Context, req *abcitypes.RequestExtendVote) (*abcitypes.ResponseExtendVote, error) {
	return &abcitypes.ResponseExtendVote{}, nil
}

// VerifyVoteExtension implements ABCI VerifyVoteExtension method (ABCI++ 2.0)
func (app *PulsarApp) VerifyVoteExtension(ctx context.Context, req *abcitypes.RequestVerifyVoteExtension) (*abcitypes.ResponseVerifyVoteExtension, error) {
	return &abcitypes.ResponseVerifyVoteExtension{
		Status: abcitypes.ResponseVerifyVoteExtension_ACCEPT,
	}, nil
}

// PrepareProposal implements ABCI PrepareProposal method (ABCI++ 1.0)
func (app *PulsarApp) PrepareProposal(ctx context.Context, req *abcitypes.RequestPrepareProposal) (*abcitypes.ResponsePrepareProposal, error) {
	return &abcitypes.ResponsePrepareProposal{
		Txs: req.Txs,
	}, nil
}

// ProcessProposal implements ABCI ProcessProposal method (ABCI++ 1.0)
func (app *PulsarApp) ProcessProposal(ctx context.Context, req *abcitypes.RequestProcessProposal) (*abcitypes.ResponseProcessProposal, error) {
	return &abcitypes.ResponseProcessProposal{
		Status: abcitypes.ResponseProcessProposal_ACCEPT,
	}, nil
}

// ListSnapshots implements ABCI ListSnapshots method
func (app *PulsarApp) ListSnapshots(ctx context.Context, req *abcitypes.RequestListSnapshots) (*abcitypes.ResponseListSnapshots, error) {
	return &abcitypes.ResponseListSnapshots{}, nil
}

// OfferSnapshot implements ABCI OfferSnapshot method
func (app *PulsarApp) OfferSnapshot(ctx context.Context, req *abcitypes.RequestOfferSnapshot) (*abcitypes.ResponseOfferSnapshot, error) {
	return &abcitypes.ResponseOfferSnapshot{
		Result: abcitypes.ResponseOfferSnapshot_REJECT,
	}, nil
}

// LoadSnapshotChunk implements ABCI LoadSnapshotChunk method
func (app *PulsarApp) LoadSnapshotChunk(ctx context.Context, req *abcitypes.RequestLoadSnapshotChunk) (*abcitypes.ResponseLoadSnapshotChunk, error) {
	return &abcitypes.ResponseLoadSnapshotChunk{}, nil
}

// ApplySnapshotChunk implements ABCI ApplySnapshotChunk method
func (app *PulsarApp) ApplySnapshotChunk(ctx context.Context, req *abcitypes.RequestApplySnapshotChunk) (*abcitypes.ResponseApplySnapshotChunk, error) {
	return &abcitypes.ResponseApplySnapshotChunk{
		Result: abcitypes.ResponseApplySnapshotChunk_UNKNOWN,
	}, nil
}