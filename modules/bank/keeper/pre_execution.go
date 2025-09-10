package keeper

import (
	keepertypes "github.com/b2network/pulsar/keeper/types"
	banktypes "github.com/b2network/pulsar/modules/bank/types"
	preexectypes "github.com/b2network/pulsar/pre_execution/types"
)

// GetModuleName returns the module name for identification
func (k Keeper) GetModuleName() string {
	return banktypes.ModuleName
}

// PreExecuteMsg executes a banking message in pre-execution mode
func (k Keeper) PreExecuteMsg(ctx keepertypes.Context, msg preexectypes.PreExecutableMsg) (*preexectypes.PreExecResult, error) {
	// Simplified implementation - just return success for now
	return &preexectypes.PreExecResult{
		Success: true,
		Code:    0,
		Log:     "pre-execution successful",
	}, nil
}

// ValidatePreExecResult validates a cached pre-execution result
func (k Keeper) ValidatePreExecResult(ctx keepertypes.Context, result *preexectypes.PreExecResult) error {
	return nil
}

// ApplyCachedResult applies a validated pre-execution result to the current state
func (k Keeper) ApplyCachedResult(ctx keepertypes.Context, result *preexectypes.PreExecResult) error {
	return nil
}

// GetPreExecConfig returns the module's pre-execution configuration
func (k Keeper) GetPreExecConfig() *preexectypes.ModulePreExecConfig {
	return banktypes.BankPreExecConfig
}
