package types

import (
	"time"

	preexectypes "github.com/b2network/pulsar/pre_execution/types"
)

// Staking module message pre-execution extensions
// These extensions implement the PreExecutableMsg interface for staking messages

// MsgDelegate implements PreExecutableMsg interface
func (m *MsgDelegate) IsPreExecutable() bool {
	// Delegation supports pre-execution for faster staking operations
	// This improves user experience for liquid staking scenarios
	return true
}

func (m *MsgDelegate) GetPreExecutionHints() *preexectypes.PreExecHints {
	return &preexectypes.PreExecHints{
		// High priority as staking affects consensus security
		Priority: 200,

		// Higher gas limit due to complex staking calculations
		MaxGasForPreExec: 150000,

		// Delegation requires ordering to prevent double-spending
		RequiresOrdering: true,

		// Cache delegation results for 45 seconds
		CacheDuration: 45 * time.Second,

		// Estimate gas for delegation operations
		EstimatedGasUsage: 80000,

		// Delegation modifies validator and delegator state
		StateReadOnly: false,
	}
}

// MsgUndelegate implements PreExecutableMsg interface
func (m *MsgUndelegate) IsPreExecutable() bool {
	// Undelegation supports pre-execution but with longer validation
	// Pre-execution helps validate undelegation before commitment
	return true
}

func (m *MsgUndelegate) GetPreExecutionHints() *preexectypes.PreExecHints {
	return &preexectypes.PreExecHints{
		// High priority for undelegation operations
		Priority: 180,

		// Higher gas due to unbonding time calculations
		MaxGasForPreExec: 200000,

		// Strict ordering required for undelegation
		RequiresOrdering: true,

		// Shorter cache duration due to time-sensitive unbonding
		CacheDuration: 30 * time.Second,

		// Higher gas estimate for unbonding logic
		EstimatedGasUsage: 120000,

		// Undelegation modifies delegation and unbonding state
		StateReadOnly: false,
	}
}

// MsgBeginRedelegate implements PreExecutableMsg interface
func (m *MsgBeginRedelegate) IsPreExecutable() bool {
	// Redelegation supports pre-execution with complex validation
	// Helps users validate redelegation constraints in advance
	return true
}

func (m *MsgBeginRedelegate) GetPreExecutionHints() *preexectypes.PreExecHints {
	return &preexectypes.PreExecHints{
		// High priority for redelegation operations
		Priority: 190,

		// Highest gas limit due to complex redelegation logic
		MaxGasForPreExec: 250000,

		// Critical ordering for redelegation chains
		RequiresOrdering: true,

		// Medium cache duration
		CacheDuration: 40 * time.Second,

		// High gas estimate for redelegation complexity
		EstimatedGasUsage: 150000,

		// Redelegation affects multiple validators
		StateReadOnly: false,
	}
}

// MsgCreateValidator implements PreExecutableMsg interface
func (m *MsgCreateValidator) IsPreExecutable() bool {
	// Validator creation can be pre-executed for validation
	// Helps catch validator setup issues before block inclusion
	return true
}

func (m *MsgCreateValidator) GetPreExecutionHints() *preexectypes.PreExecHints {
	return &preexectypes.PreExecHints{
		// Highest priority for validator operations
		Priority: 250,

		// Very high gas limit for validator creation
		MaxGasForPreExec: 300000,

		// Strict ordering for validator creation
		RequiresOrdering: true,

		// Longer cache duration as validator creation is complex
		CacheDuration: 60 * time.Second,

		// High gas estimate for validator setup
		EstimatedGasUsage: 200000,

		// Creates new validator state
		StateReadOnly: false,
	}
}

// MsgEditValidator implements PreExecutableMsg interface
func (m *MsgEditValidator) IsPreExecutable() bool {
	// Validator editing supports pre-execution for quick validation
	return true
}

func (m *MsgEditValidator) GetPreExecutionHints() *preexectypes.PreExecHints {
	return &preexectypes.PreExecHints{
		// High priority for validator operations
		Priority: 220,

		// Moderate gas for validator editing
		MaxGasForPreExec: 100000,

		// Ordering required for validator consistency
		RequiresOrdering: true,

		// Standard cache duration
		CacheDuration: 45 * time.Second,

		// Moderate gas estimate
		EstimatedGasUsage: 60000,

		// Modifies validator metadata
		StateReadOnly: false,
	}
}

// MsgCancelUnbondingDelegation implements PreExecutableMsg interface
func (m *MsgCancelUnbondingDelegation) IsPreExecutable() bool {
	// Cancel unbonding supports pre-execution for validation
	return true
}

func (m *MsgCancelUnbondingDelegation) GetPreExecutionHints() *preexectypes.PreExecHints {
	return &preexectypes.PreExecHints{
		// Medium-high priority
		Priority: 160,

		// Moderate gas for cancellation logic
		MaxGasForPreExec: 120000,

		// Ordering required for unbonding consistency
		RequiresOrdering: true,

		// Short cache duration due to time sensitivity
		CacheDuration: 25 * time.Second,

		// Moderate gas estimate
		EstimatedGasUsage: 70000,

		// Modifies unbonding delegation state
		StateReadOnly: false,
	}
}

// Staking-specific pre-execution configuration
var StakingPreExecConfig = &preexectypes.ModulePreExecConfig{
	Enabled: true,
	SupportedMsgTypes: []string{
		"/cosmos.staking.v1beta1.MsgDelegate",
		"/cosmos.staking.v1beta1.MsgUndelegate",
		"/cosmos.staking.v1beta1.MsgBeginRedelegate",
		"/cosmos.staking.v1beta1.MsgCreateValidator",
		"/cosmos.staking.v1beta1.MsgEditValidator",
		"/cosmos.staking.v1beta1.MsgCancelUnbondingDelegation",
	},
	MaxPreExecPerBlock: 500,    // Conservative limit for staking operations
	MaxGasPerPreExec:   300000, // High gas limit for complex staking operations
	StateReadOnlyKeys: []string{
		"validators/",            // Validator information
		"delegations/",           // Delegation records
		"unbonding_delegations/", // Unbonding delegation records
		"redelegations/",         // Redelegation records
		"historical_info/",       // Historical validator info
		"params/",                // Staking module parameters
		"last_validator_power/",  // Last validator power
		"last_total_power/",      // Last total bonded power
	},
	CachePolicy: preexectypes.CachePolicy{
		MaxSize:            2000,             // Cache up to 2000 staking operations
		TTL:                90 * time.Second, // Cache for 1.5 minutes
		EvictionPolicy:     "LRU",            // Use LRU eviction
		CompressionEnabled: true,             // Compress complex staking data
	},
}
