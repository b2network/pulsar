package fee

import (
	"encoding/json"
	"fmt"

	keepertypes "github.com/b2network/pulsar/keeper/types"
	"github.com/b2network/pulsar/modules/fee/config"
	"github.com/b2network/pulsar/modules/fee/keeper"
	feetypes "github.com/b2network/pulsar/modules/fee/types"
	storetypes "github.com/b2network/pulsar/store/types"
)

// AppModule interface definition
type AppModule interface {
	Name() string
	RegisterInterfaces(registry InterfaceRegistry)
	DefaultGenesis() json.RawMessage
	ValidateGenesis(bz json.RawMessage) error
	InitGenesis(ctx keepertypes.Context, data json.RawMessage)
	ExportGenesis(ctx keepertypes.Context) json.RawMessage
	BeginBlock(ctx keepertypes.Context)
	EndBlock(ctx keepertypes.Context)

	// Module-specific methods
	IsOnePerModuleType()
	IsAppModule()
}

// InterfaceRegistry defines the interface for registering interfaces
type InterfaceRegistry interface {
	RegisterInterface(protoName string, iface interface{}, impls ...interface{})
	RegisterImplementations(iface interface{}, impls ...interface{})
}

// FeeAppModule represents the fee module
type FeeAppModule struct {
	keeper        keeper.Keeper
	configManager *keeper.GasConfigManager
	configLoader  *config.ConfigLoader
	validator     *config.ConfigValidator
}

// NewAppModule creates a new AppModule object
func NewAppModule(
	storeKey, memKey storetypes.StoreKey,
	authority string,
	accountKeeper feetypes.AccountKeeper,
	bankKeeper feetypes.BankKeeper,
	stakingKeeper feetypes.StakingKeeper,
	distributionKeeper feetypes.DistributionKeeper,
	configDir string,
) FeeAppModule {
	k := keeper.NewKeeper(
		storeKey, memKey,
		authority,
		accountKeeper,
		bankKeeper,
		stakingKeeper,
		distributionKeeper,
	)

	return FeeAppModule{
		keeper:        *k,
		configManager: keeper.NewGasConfigManager(k),
		configLoader:  config.NewConfigLoader(configDir),
		validator:     config.NewConfigValidator(),
	}
}

// Name returns the fee module's name
func (FeeAppModule) Name() string {
	return feetypes.ModuleName
}

// RegisterInterfaces registers the module's interface types
func (am FeeAppModule) RegisterInterfaces(registry InterfaceRegistry) {
	feetypes.RegisterInterfaces(registry)
}

// DefaultGenesis returns the fee module's default genesis state
func (FeeAppModule) DefaultGenesis() json.RawMessage {
	return json.RawMessage(mustMarshalJSON(feetypes.DefaultGenesis()))
}

// ValidateGenesis performs genesis state validation for the fee module
func (FeeAppModule) ValidateGenesis(bz json.RawMessage) error {
	var genState feetypes.GenesisState
	if err := json.Unmarshal(bz, &genState); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", feetypes.ModuleName, err)
	}
	return feetypes.ValidateGenesis(genState)
}

// InitGenesis performs the fee module's genesis initialization
func (am FeeAppModule) InitGenesis(ctx keepertypes.Context, data json.RawMessage) {
	var genState feetypes.GenesisState
	if err := json.Unmarshal(data, &genState); err != nil {
		panic(fmt.Sprintf("failed to unmarshal %s genesis state: %s", feetypes.ModuleName, err))
	}

	am.keeper.InitGenesis(ctx, genState)

	// Initialize gas configuration system
	if err := am.initializeGasConfiguration(ctx); err != nil {
		ctx.Logger().Error("failed to initialize gas configuration system", "error", err)
	}
}

// ExportGenesis returns the fee module's exported genesis state
func (am FeeAppModule) ExportGenesis(ctx keepertypes.Context) json.RawMessage {
	genState := am.keeper.ExportGenesis(ctx)
	return json.RawMessage(mustMarshalJSON(genState))
}

// BeginBlock executes all ABCI BeginBlock logic respective to the fee module
func (am FeeAppModule) BeginBlock(ctx keepertypes.Context) {
	// Update gas prices periodically
	if am.keeper.ShouldUpdateGasPrices(ctx) {
		if err := am.keeper.UpdateGasPrices(ctx); err != nil {
			ctx.Logger().Error("failed to update gas prices", "error", err)
		}
	}

	// Auto-tune gas factors based on network conditions
	if err := am.keeper.AutoTuneGasFactors(ctx); err != nil {
		ctx.Logger().Error("failed to auto-tune gas factors", "error", err)
	}
}

// EndBlock executes all ABCI EndBlock logic respective to the fee module
func (am FeeAppModule) EndBlock(ctx keepertypes.Context) {
	// Distribute collected fees
	if err := am.keeper.DistributeFees(ctx); err != nil {
		ctx.Logger().Error("failed to distribute fees", "error", err)
	}
}

// GetKeeper returns the fee module's keeper
func (am FeeAppModule) GetKeeper() keeper.Keeper {
	return am.keeper
}

// GetConfigManager returns the gas configuration manager
func (am FeeAppModule) GetConfigManager() *keeper.GasConfigManager {
	return am.configManager
}

// GetConfigLoader returns the configuration loader
func (am FeeAppModule) GetConfigLoader() *config.ConfigLoader {
	return am.configLoader
}

// GetValidator returns the configuration validator
func (am FeeAppModule) GetValidator() *config.ConfigValidator {
	return am.validator
}

// initializeGasConfiguration initializes the gas configuration system
func (am FeeAppModule) initializeGasConfiguration(ctx keepertypes.Context) error {
	// Load gas configurations from files
	gasConfigs, err := am.configLoader.LoadGasConfigs()
	if err != nil {
		ctx.Logger().Error("failed to load gas configurations from file, using defaults", "error", err)
		// Use built-in defaults
		gasConfigs = feetypes.DefaultModuleGasConfigs()
	}

	// Validate configurations
	if err := am.validator.ValidateAllGasConfigs(gasConfigs); err != nil {
		return fmt.Errorf("invalid gas configurations: %w", err)
	}

	// Apply configurations
	for _, config := range gasConfigs {
		if err := am.keeper.SetModuleGasConfig(ctx, config); err != nil {
			ctx.Logger().Error("failed to set gas config for module",
				"module", config.ModuleName, "error", err)
		}
	}

	// Load fee denominations
	feeDenoms, err := am.configLoader.LoadFeeDenoms()
	if err != nil {
		ctx.Logger().Error("failed to load fee denominations from file, using defaults", "error", err)
		feeDenoms = feetypes.DefaultFeeDenoms()
	}

	// Validate fee denominations
	if err := am.validator.ValidateAllFeeDenoms(feeDenoms); err != nil {
		return fmt.Errorf("invalid fee denominations: %w", err)
	}

	// Apply fee denominations
	for _, feeDenom := range feeDenoms {
		if err := am.keeper.AddFeeDenom(ctx, feeDenom); err != nil {
			// Log but don't fail - denomination might already exist
			ctx.Logger().Info("fee denomination already exists or failed to add",
				"denom", feeDenom.Denom, "error", err)
		}
	}

	// Load dynamic gas factors
	dynamicFactors, err := am.configLoader.LoadDynamicGasFactors()
	if err != nil {
		ctx.Logger().Error("failed to load dynamic gas factors from file, using defaults", "error", err)
		dynamicFactors = feetypes.DefaultDynamicGasFactors()
	}

	// Validate dynamic gas factors
	if err := am.validator.ValidateDynamicGasFactors(dynamicFactors); err != nil {
		return fmt.Errorf("invalid dynamic gas factors: %w", err)
	}

	// Apply dynamic gas factors
	if err := am.keeper.SetDynamicGasFactors(ctx, dynamicFactors); err != nil {
		return fmt.Errorf("failed to set dynamic gas factors: %w", err)
	}

	ctx.Logger().Info("gas configuration system initialized successfully",
		"gas_configs", len(gasConfigs),
		"fee_denoms", len(feeDenoms))

	return nil
}

// ValidateConfiguration validates the current configuration
func (am FeeAppModule) ValidateConfiguration(ctx keepertypes.Context) error {
	gasConfigs := am.keeper.GetAllModuleGasConfigs(ctx)
	feeDenoms := am.keeper.GetAllFeeDenoms(ctx)
	dynamicFactors := am.keeper.GetDynamicGasFactors(ctx)

	report := am.validator.GenerateValidationReport(gasConfigs, feeDenoms, dynamicFactors)

	if !report.Valid {
		return fmt.Errorf("configuration validation failed: %v", report.Errors)
	}

	if len(report.Warnings) > 0 {
		ctx.Logger().Info("configuration validation warnings", "warnings", report.Warnings)
	}

	return nil
}

// ReloadConfiguration reloads configuration from files
func (am FeeAppModule) ReloadConfiguration(ctx keepertypes.Context) error {
	ctx.Logger().Info("reloading fee module configuration")

	// Reinitialize gas configuration system
	if err := am.initializeGasConfiguration(ctx); err != nil {
		return fmt.Errorf("failed to reload gas configuration: %w", err)
	}

	// Validate the new configuration
	if err := am.ValidateConfiguration(ctx); err != nil {
		return fmt.Errorf("validation failed after reload: %w", err)
	}

	ctx.Logger().Info("fee module configuration reloaded successfully")
	return nil
}

// SaveConfiguration saves current configuration to files
func (am FeeAppModule) SaveConfiguration(ctx keepertypes.Context) error {
	gasConfigs := am.keeper.GetAllModuleGasConfigs(ctx)
	feeDenoms := am.keeper.GetAllFeeDenoms(ctx)
	dynamicFactors := am.keeper.GetDynamicGasFactors(ctx)

	// Save gas configurations
	if err := am.configLoader.SaveGasConfigs(gasConfigs); err != nil {
		return fmt.Errorf("failed to save gas configurations: %w", err)
	}

	// Save fee denominations
	if err := am.configLoader.SaveFeeDenoms(feeDenoms); err != nil {
		return fmt.Errorf("failed to save fee denominations: %w", err)
	}

	// Save dynamic gas factors
	if err := am.configLoader.SaveDynamicGasFactors(dynamicFactors); err != nil {
		return fmt.Errorf("failed to save dynamic gas factors: %w", err)
	}

	ctx.Logger().Info("fee module configuration saved successfully")
	return nil
}

// GenerateExampleConfigs generates example configuration files
func (am FeeAppModule) GenerateExampleConfigs() error {
	return am.configLoader.GenerateExampleConfigs()
}

// AppModule interface implementations

// IsOnePerModuleType implements the depinject.OnePerModuleType interface
func (am FeeAppModule) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface
func (am FeeAppModule) IsAppModule() {}

// mustMarshalJSON marshals JSON and panics on error
func mustMarshalJSON(v interface{}) []byte {
	bz, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal JSON: %s", err))
	}
	return bz
}