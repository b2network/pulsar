package types

import (
	"time"

	preexectypes "github.com/b2network/pulsar/pre_execution/types"
)

// Bank module message pre-execution extensions
// These extensions implement the PreExecutableMsg interface for bank messages

// MsgSend implements PreExecutableMsg interface
func (m *MsgSend) IsPreExecutable() bool {
	// MsgSend supports pre-execution as it's a simple balance transfer
	// Pre-execution is beneficial for high-frequency transfer operations
	return true
}

func (m *MsgSend) GetPreExecutionHints() *preexectypes.PreExecHints {
	return &preexectypes.PreExecHints{
		// Normal priority for regular transfers
		Priority: 100,

		// Conservative gas estimate for balance transfers
		MaxGasForPreExec: 50000,

		// Transfers require ordering to prevent double-spending
		RequiresOrdering: true,

		// Cache transfer results for 30 seconds
		CacheDuration: 30 * time.Second,

		// Estimate typical gas usage for transfers
		EstimatedGasUsage: 25000,

		// Transfers modify account balances (not read-only)
		StateReadOnly: false,
	}
}

// MsgMultiSend implements PreExecutableMsg interface
func (m *MsgMultiSend) IsPreExecutable() bool {
	// Multi-send supports pre-execution but with more complexity
	// Enable only for smaller multi-send transactions to avoid gas limits
	return len(m.Inputs) <= 10 && len(m.Outputs) <= 10
}

func (m *MsgMultiSend) GetPreExecutionHints() *preexectypes.PreExecHints {
	// Calculate dynamic gas based on number of transfers
	baseGas := uint64(30000)
	transferGas := uint64(15000) * uint64(len(m.Inputs)+len(m.Outputs))
	maxGas := baseGas + transferGas

	return &preexectypes.PreExecHints{
		// Higher priority due to batch nature
		Priority: 150,

		// Dynamic gas limit based on transfer count
		MaxGasForPreExec: maxGas,

		// Multi-send definitely requires strict ordering
		RequiresOrdering: true,

		// Shorter cache duration due to complexity
		CacheDuration: 15 * time.Second,

		// Estimate gas usage
		EstimatedGasUsage: maxGas * 80 / 100, // 80% of max

		// Multi-send modifies multiple account balances
		StateReadOnly: false,
	}
}

// Banking-specific pre-execution configuration
var BankPreExecConfig = &preexectypes.ModulePreExecConfig{
	Enabled: true,
	SupportedMsgTypes: []string{
		"/cosmos.bank.v1beta1.MsgSend",
		"/cosmos.bank.v1beta1.MsgMultiSend",
	},
	MaxPreExecPerBlock: 1000,   // Allow many banking pre-executions per block
	MaxGasPerPreExec:   100000, // Conservative gas limit per banking operation
	StateReadOnlyKeys: []string{
		"balances/",       // Account balance prefixes
		"supply/",         // Token supply information
		"denom_metadata/", // Denomination metadata
	},
	CachePolicy: preexectypes.CachePolicy{
		MaxSize:            5000,             // Cache up to 5000 banking operations
		TTL:                60 * time.Second, // Cache for 1 minute
		EvictionPolicy:     "LRU",            // Use LRU eviction
		CompressionEnabled: false,            // Don't compress simple balance data
	},
}
