package app

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"

	keepertypes "github.com/b2network/pulsar/keeper/types"
	"github.com/b2network/pulsar/store/commitment"
	"github.com/b2network/pulsar/store/rootstore"
	"github.com/b2network/pulsar/store/storage/memory"

	// Pre-execution imports
	preexeckeeper "github.com/b2network/pulsar/pre_execution/keeper"

	// Module keepers
	bankkeeper "github.com/b2network/pulsar/modules/bank/keeper"
	banktypes "github.com/b2network/pulsar/modules/bank/types"
	govkeeper "github.com/b2network/pulsar/modules/gov/keeper"
	govtypes "github.com/b2network/pulsar/modules/gov/types"
	stakingkeeper "github.com/b2network/pulsar/modules/staking/keeper"
	stakingtypes "github.com/b2network/pulsar/modules/staking/types"

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

	// Pre-execution components
	PreExecManager *preexeckeeper.PreExecutionManager

	// Query router
	queryRouter *keepertypes.QueryRouter

	// Store keys
	keys map[string]storetypes.StoreKey

	// Configuration
	config AppConfig
}

// AppConfig contains configuration for the Pulsar application
type AppConfig struct {
	// Pre-execution settings
	PreExecEnabled   bool   `json:"pre_exec_enabled"`
	PreExecCacheSize int    `json:"pre_exec_cache_size"`
	PreExecTTL       string `json:"pre_exec_ttl"`

	// Validator settings
	ValidatorID string `json:"validator_id"`
}

// NewPulsarApp creates a new PulsarApp instance
func NewPulsarApp(
	logger log.Logger,
	dataDir string,
	config AppConfig,
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
		config:    config,
	}

	// Initialize context
	app.ctx = keepertypes.NewPulsarContext(
		cms,
		0,          // initial block height
		"pulsar-1", // chain ID
		logger,
	)

	// Initialize keepers
	app.initKeepers()

	// Initialize pre-execution if enabled
	if config.PreExecEnabled {
		app.initPreExecution()
	}

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
		app.BankKeeper,       // bank keeper for token operations
		nil,                  // slashing keeper - simplified for now
		banktypes.ModuleName, // authority
	)

	// Gov keeper
	app.GovKeeper = *govkeeper.NewKeeper(
		app.keys[govtypes.StoreKey],
		app.codec,
		app.BankKeeper,      // bank keeper for deposits
		app.StakingKeeper,   // staking keeper for voting power
		govtypes.ModuleName, // authority
	)

	// Initialize query router
	app.initQueryRouter()
}

// initQueryRouter initializes the query router with module queriers
func (app *PulsarApp) initQueryRouter() {
	app.queryRouter = keepertypes.NewQueryRouter()
	
	// Register module queriers
	app.queryRouter.
		AddRoute(banktypes.ModuleName, app.BankKeeper.Querier()).
		AddRoute(stakingtypes.ModuleName, app.StakingKeeper.Querier()).
		AddRoute(govtypes.ModuleName, app.GovKeeper.Querier())
}

// initPreExecution sets up the pre-execution system
func (app *PulsarApp) initPreExecution() {
	// Parse TTL duration
	ttl, err := time.ParseDuration(app.config.PreExecTTL)
	if err != nil {
		ttl = 60 * time.Second // Default to 1 minute
		app.logger.Info("⚠️  Invalid TTL, using default", "ttl", ttl)
	}

	// Create simple configuration for pre-execution manager
	globalConfig := &preexeckeeper.GlobalPreExecConfig{}

	// Initialize pre-execution manager with simplified parameters
	var err2 error
	app.PreExecManager, err2 = preexeckeeper.NewPreExecutionManager(
		nil, // state store
		globalConfig,
		"validator1", // validator ID
		true,         // is validator
	)

	if err2 != nil {
		app.logger.Error("❌ Failed to initialize pre-execution manager", "error", err2)
		return
	}

	// Register modules with pre-execution support
	app.registerPreExecModules()

	app.logger.Info("✅ Pre-execution system initialized")
}

// registerPreExecModules registers all modules that support pre-execution
func (app *PulsarApp) registerPreExecModules() {
	// Register bank module
	app.PreExecManager.RegisterModule(&app.BankKeeper)
	app.logger.Info("📦 Registered module for pre-execution", "module", banktypes.ModuleName)

	// Register staking module
	app.PreExecManager.RegisterModule(&app.StakingKeeper)
	app.logger.Info("📦 Registered module for pre-execution", "module", stakingtypes.ModuleName)

	// Register governance module
	app.PreExecManager.RegisterModule(&app.GovKeeper)
	app.logger.Info("📦 Registered module for pre-execution", "module", govtypes.ModuleName)

	app.logger.Info("✅ Registered modules for pre-execution", "count", 3)
}

// GetPreExecutionStats returns statistics about pre-execution performance
func (app *PulsarApp) GetPreExecutionStats() *PreExecStats {
	if app.PreExecManager == nil {
		return &PreExecStats{
			Enabled: false,
		}
	}

	// Simplified stats return
	return &PreExecStats{
		Enabled:           true,
		TotalPreExecuted:  0,
		CacheHits:         0,
		CacheMisses:       0,
		AverageGasUsed:    0,
		AverageExecTime:   0,
		PendingTxs:        0,
		RegisteredModules: 0,
	}
}

// PreExecStats contains statistics about pre-execution performance
type PreExecStats struct {
	Enabled           bool          `json:"enabled"`
	TotalPreExecuted  uint64        `json:"total_pre_executed"`
	CacheHits         uint64        `json:"cache_hits"`
	CacheMisses       uint64        `json:"cache_misses"`
	AverageGasUsed    uint64        `json:"average_gas_used"`
	AverageExecTime   time.Duration `json:"average_exec_time"`
	PendingTxs        int           `json:"pending_txs"`
	RegisteredModules int           `json:"registered_modules"`
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
	// Use the query router to handle all queries
	if app.queryRouter != nil {
		return app.queryRouter.Route(app.ctx, req.Path, *req)
	}
	
	// Fallback if router is not initialized
	return &abcitypes.ResponseQuery{
		Code: 1,
		Log:  "query router not initialized",
	}, nil
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
