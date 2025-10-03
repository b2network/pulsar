package signal

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	keepertypes "github.com/b2network/pulsar/keeper/types"
	"github.com/b2network/pulsar/modules/signal/client/cli"
	"github.com/b2network/pulsar/modules/signal/keeper"
	"github.com/b2network/pulsar/modules/signal/types"
	storetypes "github.com/b2network/pulsar/store/types"
)

var (
	_ keepertypes.AppModule      = AppModule{}
	_ keepertypes.AppModuleBasic = AppModuleBasic{}
)

// AppModuleBasic implements the AppModuleBasic interface for the signal module
type AppModuleBasic struct{}

// Name returns the signal module's name
func (AppModuleBasic) Name() string {
	return types.ModuleName
}

// RegisterCodec registers the signal module's types to the codec
func (AppModuleBasic) RegisterCodec(cdc keepertypes.Codec) {
	types.RegisterCodec(cdc)
}

// DefaultGenesis returns the signal module's default genesis state
func (AppModuleBasic) DefaultGenesis(cdc keepertypes.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesisState())
}

// ValidateGenesis performs genesis state validation for the signal module
func (AppModuleBasic) ValidateGenesis(cdc keepertypes.JSONCodec, config keepertypes.TxEncodingConfig, bz json.RawMessage) error {
	var genState types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &genState); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}
	return types.ValidateGenesis(&genState)
}

// GetTxCmd returns the signal module's root tx command
func (a AppModuleBasic) GetTxCmd() *cobra.Command {
	return cli.GetTxCmd()
}

// GetQueryCmd returns the signal module's root query command
func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd()
}

// AppModule implements the AppModule interface for the signal module
type AppModule struct {
	AppModuleBasic

	keeper        keeper.Keeper
	accountKeeper types.BankKeeper      // Account keeper dependency
	bankKeeper    types.BankKeeper      // Bank keeper dependency
	stakingKeeper types.StakingKeeper   // Staking keeper dependency
	distKeeper    types.DistributionKeeper // Distribution keeper dependency
}

// NewAppModule creates a new AppModule object
func NewAppModule(
	keeper keeper.Keeper,
	accountKeeper types.BankKeeper,
	bankKeeper types.BankKeeper,
	stakingKeeper types.StakingKeeper,
	distKeeper types.DistributionKeeper,
) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{},
		keeper:         keeper,
		accountKeeper:  accountKeeper,
		bankKeeper:     bankKeeper,
		stakingKeeper:  stakingKeeper,
		distKeeper:     distKeeper,
	}
}

// Name returns the signal module's name
func (am AppModule) Name() string {
	return am.AppModuleBasic.Name()
}

// RegisterServices registers a gRPC query service to respond to the module-specific gRPC queries
func (am AppModule) RegisterServices(cfg keepertypes.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
	types.RegisterQueryServer(cfg.QueryServer(), am.keeper)
}

// InitGenesis performs the signal module's genesis initialization
func (am AppModule) InitGenesis(ctx keepertypes.Context, cdc keepertypes.JSONCodec, gs json.RawMessage) []keepertypes.ValidatorUpdate {
	var genState types.GenesisState
	cdc.MustUnmarshalJSON(gs, &genState)

	InitGenesis(ctx, am.keeper, genState)

	return []keepertypes.ValidatorUpdate{}
}

// ExportGenesis returns the signal module's exported genesis state as raw JSON bytes
func (am AppModule) ExportGenesis(ctx keepertypes.Context, cdc keepertypes.JSONCodec) json.RawMessage {
	genState := ExportGenesis(ctx, am.keeper)
	return cdc.MustMarshalJSON(genState)
}

// ConsensusVersion implements AppModule/ConsensusVersion
func (AppModule) ConsensusVersion() uint64 { return 1 }

// BeginBlock executes all ABCI BeginBlock logic respective to the signal module
func (am AppModule) BeginBlock(ctx keepertypes.Context, req keepertypes.RequestBeginBlock) {
	BeginBlocker(ctx, am.keeper)
}

// EndBlock executes all ABCI EndBlock logic respective to the signal module
func (am AppModule) EndBlock(ctx keepertypes.Context, req keepertypes.RequestEndBlock) []keepertypes.ValidatorUpdate {
	EndBlocker(ctx, am.keeper)
	return []keepertypes.ValidatorUpdate{}
}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface
func (am AppModule) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface
func (am AppModule) IsAppModule() {}

// InitGenesis initializes the signal module's state from a provided genesis state
func InitGenesis(ctx keepertypes.Context, keeper keeper.Keeper, genState types.GenesisState) {
	// Validate genesis state
	if err := types.ValidateGenesis(&genState); err != nil {
		panic(fmt.Errorf("failed to validate %s genesis state: %w", types.ModuleName, err))
	}

	// Set module parameters
	if err := keeper.SetParams(ctx, genState.Params); err != nil {
		panic(fmt.Errorf("failed to set signal module params: %w", err))
	}

	// Initialize work registry
	if err := keeper.InitializeWorkRegistry(ctx); err != nil {
		panic(fmt.Errorf("failed to initialize work registry: %w", err))
	}

	// Import signals
	for _, signal := range genState.Signals {
		if err := keeper.StoreSignal(ctx, signal); err != nil {
			keeper.Logger(ctx).Error("failed to import signal", "signal_id", signal.ID, "error", err)
		}
	}

	// Import signal scores
	for _, score := range genState.SignalScores {
		if err := keeper.StoreSignalScore(ctx, score); err != nil {
			keeper.Logger(ctx).Error("failed to import signal score", "signal_id", score.SignalID, "error", err)
		}
	}

	// Import work types
	for _, workType := range genState.WorkTypes {
		if err := keeper.StoreWorkType(ctx, workType); err != nil {
			keeper.Logger(ctx).Error("failed to import work type", "work_type_id", workType.ID, "error", err)
		}
	}

	// Import validator stats
	for addr, stats := range genState.ValidatorStats {
		if err := keeper.SetValidatorStats(ctx, *stats); err != nil {
			keeper.Logger(ctx).Error("failed to import validator stats", "validator", addr, "error", err)
		}
	}

	// Set work registry if provided
	if genState.WorkRegistry != nil {
		if err := keeper.SetWorkRegistry(ctx, genState.WorkRegistry); err != nil {
			keeper.Logger(ctx).Error("failed to set work registry", "error", err)
		}
	}

	keeper.Logger(ctx).Info("signal module genesis initialized",
		"signals_count", len(genState.Signals),
		"work_types_count", len(genState.WorkTypes),
		"validator_stats_count", len(genState.ValidatorStats))
}

// ExportGenesis returns the signal module's exported genesis
func ExportGenesis(ctx keepertypes.Context, keeper keeper.Keeper) *types.GenesisState {
	genesis := types.NewGenesisState(keeper.GetParams(ctx))

	// Export signals
	genesis.Signals = keeper.GetAllSignals(ctx)

	// Export signal scores
	genesis.SignalScores = keeper.GetAllSignalScores(ctx)

	// Export work types
	genesis.WorkTypes = keeper.GetAllWorkTypes(ctx)

	// Export validator stats
	genesis.ValidatorStats = keeper.GetAllValidatorStats(ctx)

	// Export work registry
	if registry, found := keeper.GetWorkRegistry(ctx); found {
		genesis.WorkRegistry = registry
	}

	// Export signal sequence
	genesis.SignalSequence = keeper.GetSignalSequence(ctx)

	return genesis
}

// BeginBlocker called at the beginning of every block
func BeginBlocker(ctx keepertypes.Context, keeper keeper.Keeper) {
	// Clean up expired signals
	if err := keeper.CleanupExpiredSignals(ctx); err != nil {
		keeper.Logger(ctx).Error("failed to cleanup expired signals", "error", err)
	}

	// Update leaderboard
	if err := keeper.UpdateLeaderboard(ctx); err != nil {
		keeper.Logger(ctx).Error("failed to update leaderboard", "error", err)
	}
}

// EndBlocker called at the end of every block
func EndBlocker(ctx keepertypes.Context, keeper keeper.Keeper) {
	// Process any pending signals
	params := keeper.GetParams(ctx)
	if params.AutoValidationEnabled {
		// Get pending signals
		pendingSignals := keeper.GetSignalsByStatus(ctx, types.SignalStatusPending)

		// Process a limited number of signals per block
		processCount := 0
		maxProcess := int(params.MaxSignalsPerBlock / 2) // Process half of max per block

		for _, signal := range pendingSignals {
			if processCount >= maxProcess {
				break
			}

			// Process signal
			if err := keeper.ProcessSignal(ctx, signal.ID); err != nil {
				keeper.Logger(ctx).Debug("failed to process signal in end blocker",
					"signal_id", signal.ID,
					"error", err)
			} else {
				processCount++
			}
		}

		if processCount > 0 {
			keeper.Logger(ctx).Debug("processed signals in end blocker",
				"processed_count", processCount,
				"pending_count", len(pendingSignals))
		}
	}
}

// Module initialization helpers

// ProvideModule provides the signal module for dependency injection
func ProvideModule(
	storeKey storetypes.StoreKey,
	codec keepertypes.Codec,
	bankKeeper types.BankKeeper,
	stakingKeeper types.StakingKeeper,
	distKeeper types.DistributionKeeper,
	authority string,
) (keepertypes.AppModule, keeper.Keeper) {
	k := keeper.NewKeeper(
		storeKey,
		codec,
		bankKeeper,
		stakingKeeper,
		distKeeper,
		authority,
	)

	m := NewAppModule(
		k,
		bankKeeper,
		bankKeeper,
		stakingKeeper,
		distKeeper,
	)

	return m, k
}

// ModuleConfiguration represents configuration for the signal module
type ModuleConfiguration struct {
	EnableAutoValidation bool                   `json:"enable_auto_validation"`
	ProcessingInterval   int64                  `json:"processing_interval"`
	MaxConcurrentProcess int                    `json:"max_concurrent_process"`
	WorkTypeWeights      map[string]float64     `json:"work_type_weights"`
}

// DefaultModuleConfiguration returns default module configuration
func DefaultModuleConfiguration() ModuleConfiguration {
	return ModuleConfiguration{
		EnableAutoValidation: true,
		ProcessingInterval:   10, // seconds
		MaxConcurrentProcess: 10,
		WorkTypeWeights: map[string]float64{
			"llm_inference":      1.5,
			"image_generation":   2.0,
			"model_training":     3.0,
			"data_analysis":      1.2,
			"proof_verification": 1.0,
			"optimization":       1.8,
		},
	}
}