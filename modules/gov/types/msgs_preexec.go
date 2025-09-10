package types

import (
	"time"

	preexectypes "github.com/b2network/pulsar/pre_execution/types"
)

// Gov module message pre-execution extensions
// These extensions implement the PreExecutableMsg interface for governance messages

// MsgSubmitProposal implements PreExecutableMsg interface
func (m *MsgSubmitProposal) IsPreExecutable() bool {
	// Proposal submission supports pre-execution for validation
	// This helps validate proposals before committing to blockchain
	return true
}

func (m *MsgSubmitProposal) GetPreExecutionHints() *preexectypes.PreExecHints {
	return &preexectypes.PreExecHints{
		// High priority for governance operations
		Priority: 180,

		// High gas limit due to proposal validation complexity
		MaxGasForPreExec: 200000,

		// Proposal submission requires ordering
		RequiresOrdering: true,

		// Cache proposal validation for 60 seconds
		CacheDuration: 60 * time.Second,

		// Estimate gas for proposal processing
		EstimatedGasUsage: 100000,

		// Proposal submission modifies governance state
		StateReadOnly: false,
	}
}

// MsgDeposit implements PreExecutableMsg interface
func (m *MsgDeposit) IsPreExecutable() bool {
	// Deposits support pre-execution for faster governance participation
	// Pre-execution validates deposit requirements and balances
	return true
}

func (m *MsgDeposit) GetPreExecutionHints() *preexectypes.PreExecHints {
	return &preexectypes.PreExecHints{
		// Medium-high priority for deposits
		Priority: 160,

		// Moderate gas for deposit operations
		MaxGasForPreExec: 100000,

		// Deposits require ordering to prevent double-spending
		RequiresOrdering: true,

		// Cache deposit validation for 45 seconds
		CacheDuration: 45 * time.Second,

		// Estimate gas for deposit processing
		EstimatedGasUsage: 60000,

		// Deposits modify proposal and account state
		StateReadOnly: false,
	}
}

// MsgVote implements PreExecutableMsg interface
func (m *MsgVote) IsPreExecutable() bool {
	// Voting supports pre-execution for immediate feedback
	// Helps validate voting eligibility before block inclusion
	return true
}

func (m *MsgVote) GetPreExecutionHints() *preexectypes.PreExecHints {
	return &preexectypes.PreExecHints{
		// High priority for democratic participation
		Priority: 200,

		// Lower gas limit for simple voting
		MaxGasForPreExec: 80000,

		// Votes require ordering to prevent manipulation
		RequiresOrdering: true,

		// Shorter cache duration due to voting period sensitivity
		CacheDuration: 30 * time.Second,

		// Lower gas estimate for voting
		EstimatedGasUsage: 40000,

		// Voting modifies vote records
		StateReadOnly: false,
	}
}

// MsgVoteWeighted implements PreExecutableMsg interface
func (m *MsgVoteWeighted) IsPreExecutable() bool {
	// Weighted voting supports pre-execution with validation
	// More complex than regular voting due to weight calculations
	return true
}

func (m *MsgVoteWeighted) GetPreExecutionHints() *preexectypes.PreExecHints {
	return &preexectypes.PreExecHints{
		// High priority for governance participation
		Priority: 190,

		// Higher gas due to weighted vote complexity
		MaxGasForPreExec: 120000,

		// Strict ordering for weighted votes
		RequiresOrdering: true,

		// Standard cache duration
		CacheDuration: 35 * time.Second,

		// Higher gas estimate for weight calculations
		EstimatedGasUsage: 70000,

		// Weighted voting modifies vote state
		StateReadOnly: false,
	}
}

// MsgExecLegacyContent implements PreExecutableMsg interface
func (m *MsgExecLegacyContent) IsPreExecutable() bool {
	// Legacy content execution can be pre-executed for validation
	// Helps catch execution issues before proposal passes
	return false // Disabled by default due to complexity and security concerns
}

func (m *MsgExecLegacyContent) GetPreExecutionHints() *preexectypes.PreExecHints {
	return &preexectypes.PreExecHints{
		// Low priority due to security concerns
		Priority: 50,

		// Very high gas limit for legacy content execution
		MaxGasForPreExec: 1000000,

		// Critical ordering for execution
		RequiresOrdering: true,

		// Very short cache duration
		CacheDuration: 10 * time.Second,

		// High gas estimate for execution
		EstimatedGasUsage: 500000,

		// Legacy execution modifies arbitrary state
		StateReadOnly: false,
	}
}

// Gov-specific pre-execution configuration
var GovPreExecConfig = &preexectypes.ModulePreExecConfig{
	Enabled: true,
	SupportedMsgTypes: []string{
		"/cosmos.gov.v1.MsgSubmitProposal",
		"/cosmos.gov.v1beta1.MsgSubmitProposal",
		"/cosmos.gov.v1.MsgDeposit",
		"/cosmos.gov.v1beta1.MsgDeposit",
		"/cosmos.gov.v1.MsgVote",
		"/cosmos.gov.v1beta1.MsgVote",
		"/cosmos.gov.v1.MsgVoteWeighted",
		"/cosmos.gov.v1beta1.MsgVoteWeighted",
		// Note: MsgExecLegacyContent not included for security reasons
	},
	MaxPreExecPerBlock: 300,    // Moderate limit for governance operations
	MaxGasPerPreExec:   200000, // Conservative gas limit for governance
	StateReadOnlyKeys: []string{
		"proposals/",    // Proposal data
		"deposits/",     // Deposit records
		"votes/",        // Vote records
		"params/",       // Governance parameters
		"constitution/", // Constitution text
		"proposal_id/",  // Next proposal ID
	},
	CachePolicy: preexectypes.CachePolicy{
		MaxSize:            1000,              // Cache up to 1000 gov operations
		TTL:                120 * time.Second, // Cache for 2 minutes
		EvictionPolicy:     "LFU",             // Use LFU for governance (less frequent updates)
		CompressionEnabled: true,              // Compress proposal data
	},
}
