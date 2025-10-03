package types

import (
	"fmt"

	keepertypes "github.com/b2network/pulsar/keeper/types"
)

// Params defines the parameters for the signal module
type Params struct {
	// Fee for submitting signals
	SignalSubmissionFee keepertypes.Coins `json:"signal_submission_fee"`

	// Minimum score threshold for signals to be considered valid
	MinSignalScore uint64 `json:"min_signal_score"`

	// Maximum number of signals that can be processed in a single block
	MaxSignalsPerBlock uint32 `json:"max_signals_per_block"`

	// How long to retain signals in storage (in seconds)
	SignalRetentionPeriod int64 `json:"signal_retention_period"`

	// Rate at which signal scores decay over time
	ScoreDecayRate float64 `json:"score_decay_rate"`

	// Weights for different work types
	WorkTypeWeights map[string]float64 `json:"work_type_weights"`

	// Minimum number of validators required for verification
	VerificationThreshold uint32 `json:"verification_threshold"`

	// Percentage of fees allocated to reward pool
	RewardPoolPercentage float64 `json:"reward_pool_percentage"`

	// Enable automatic signal validation
	AutoValidationEnabled bool `json:"auto_validation_enabled"`

	// Maximum age of signals that can be submitted (in seconds)
	MaxSignalAge int64 `json:"max_signal_age"`

	// Slashing percentage for invalid signals
	InvalidSignalSlashRate float64 `json:"invalid_signal_slash_rate"`

	// Bonus multiplier for verified signals
	VerifiedSignalBonus float64 `json:"verified_signal_bonus"`

	// Maximum number of active signals per validator
	MaxSignalsPerValidator uint32 `json:"max_signals_per_validator"`

	// Cooldown period between signal submissions (in seconds)
	SignalSubmissionCooldown int64 `json:"signal_submission_cooldown"`

	// Enable signal batching for efficiency
	BatchProcessingEnabled bool `json:"batch_processing_enabled"`

	// Maximum batch size for signal processing
	MaxBatchSize uint32 `json:"max_batch_size"`

	// Score calculation parameters
	ScoreParams SignalScoreParams `json:"score_params"`
}

// DefaultParams returns default parameters for the signal module
func DefaultParams() Params {
	return Params{
		SignalSubmissionFee: keepertypes.Coins{
			{Denom: "stake", Amount: 100},
		},
		MinSignalScore:           10,
		MaxSignalsPerBlock:       100,
		SignalRetentionPeriod:    2592000,  // 30 days
		ScoreDecayRate:           0.1,
		WorkTypeWeights:          defaultWorkTypeWeights(),
		VerificationThreshold:    3,
		RewardPoolPercentage:     0.5,       // 50% of fees go to rewards
		AutoValidationEnabled:    true,
		MaxSignalAge:             3600,      // 1 hour
		InvalidSignalSlashRate:   0.01,      // 1% slash for invalid signals
		VerifiedSignalBonus:      1.5,       // 50% bonus for verified signals
		MaxSignalsPerValidator:   1000,
		SignalSubmissionCooldown: 10,        // 10 seconds
		BatchProcessingEnabled:   true,
		MaxBatchSize:            50,
		ScoreParams:             DefaultSignalScoreParams(),
	}
}

// defaultWorkTypeWeights returns default weights for work types
func defaultWorkTypeWeights() map[string]float64 {
	return map[string]float64{
		"llm_inference":      1.5,
		"image_generation":   2.0,
		"model_training":     3.0,
		"data_analysis":      1.2,
		"proof_verification": 1.0,
		"optimization":       1.8,
	}
}

// ValidateBasic performs basic validation on module parameters
func (p Params) ValidateBasic() error {
	if len(p.SignalSubmissionFee) == 0 {
		return fmt.Errorf("signal submission fee cannot be empty")
	}

	for _, coin := range p.SignalSubmissionFee {
		if coin.Amount <= 0 {
			return fmt.Errorf("signal submission fee amount must be positive")
		}
	}

	if p.MinSignalScore == 0 {
		return fmt.Errorf("minimum signal score must be positive")
	}

	if p.MaxSignalsPerBlock == 0 {
		return fmt.Errorf("max signals per block must be positive")
	}

	if p.SignalRetentionPeriod <= 0 {
		return fmt.Errorf("signal retention period must be positive")
	}

	if p.ScoreDecayRate < 0 || p.ScoreDecayRate > 1 {
		return fmt.Errorf("score decay rate must be between 0 and 1")
	}

	if p.VerificationThreshold == 0 {
		return fmt.Errorf("verification threshold must be positive")
	}

	if p.RewardPoolPercentage < 0 || p.RewardPoolPercentage > 1 {
		return fmt.Errorf("reward pool percentage must be between 0 and 1")
	}

	if p.MaxSignalAge <= 0 {
		return fmt.Errorf("max signal age must be positive")
	}

	if p.InvalidSignalSlashRate < 0 || p.InvalidSignalSlashRate > 1 {
		return fmt.Errorf("invalid signal slash rate must be between 0 and 1")
	}

	if p.VerifiedSignalBonus < 1 {
		return fmt.Errorf("verified signal bonus must be at least 1")
	}

	if p.MaxSignalsPerValidator == 0 {
		return fmt.Errorf("max signals per validator must be positive")
	}

	if p.SignalSubmissionCooldown < 0 {
		return fmt.Errorf("signal submission cooldown cannot be negative")
	}

	if p.BatchProcessingEnabled && p.MaxBatchSize == 0 {
		return fmt.Errorf("max batch size must be positive when batch processing is enabled")
	}

	// Validate work type weights
	for workType, weight := range p.WorkTypeWeights {
		if weight <= 0 {
			return fmt.Errorf("work type weight for %s must be positive", workType)
		}
	}

	// Validate score params
	if err := p.ScoreParams.ValidateBasic(); err != nil {
		return fmt.Errorf("invalid score params: %w", err)
	}

	return nil
}

// ValidateBasic performs basic validation on score parameters
func (sp SignalScoreParams) ValidateBasic() error {
	if sp.BaseScoreWeight < 0 || sp.BaseScoreWeight > 1 {
		return fmt.Errorf("base score weight must be between 0 and 1")
	}

	if sp.ResourceWeight < 0 || sp.ResourceWeight > 1 {
		return fmt.Errorf("resource weight must be between 0 and 1")
	}

	if sp.ComplexityWeight < 0 || sp.ComplexityWeight > 1 {
		return fmt.Errorf("complexity weight must be between 0 and 1")
	}

	if sp.AccuracyWeight < 0 || sp.AccuracyWeight > 1 {
		return fmt.Errorf("accuracy weight must be between 0 and 1")
	}

	totalWeight := sp.BaseScoreWeight + sp.ResourceWeight + sp.ComplexityWeight + sp.AccuracyWeight
	if totalWeight < 0.99 || totalWeight > 1.01 { // Allow small floating point error
		return fmt.Errorf("score weights must sum to 1.0, got %f", totalWeight)
	}

	if sp.TimeDecayHalfLife <= 0 {
		return fmt.Errorf("time decay half life must be positive")
	}

	if sp.MaxScoreAge <= 0 {
		return fmt.Errorf("max score age must be positive")
	}

	if sp.VerificationBonusRate < 0 || sp.VerificationBonusRate > 1 {
		return fmt.Errorf("verification bonus rate must be between 0 and 1")
	}

	return nil
}

// GetWorkTypeWeight returns the weight for a specific work type
func (p Params) GetWorkTypeWeight(workType string) float64 {
	if weight, exists := p.WorkTypeWeights[workType]; exists {
		return weight
	}
	return 1.0 // Default weight
}

// ShouldProcessSignal determines if a signal should be processed based on params
func (p Params) ShouldProcessSignal(signal Signal, currentTime int64) bool {
	// Check if signal is too old
	age := currentTime - signal.Timestamp
	if age > p.MaxSignalAge {
		return false
	}

	// Check if signal has expired
	if signal.IsExpired(currentTime) {
		return false
	}

	// Additional checks can be added here
	return true
}

// CalculateFeeReward calculates the reward amount from submission fees
func (p Params) CalculateFeeReward(totalFees keepertypes.Coins) keepertypes.Coins {
	rewards := make(keepertypes.Coins, len(totalFees))
	for i, coin := range totalFees {
		rewardAmount := int64(float64(coin.Amount) * p.RewardPoolPercentage)
		rewards[i] = keepertypes.Coin{
			Denom:  coin.Denom,
			Amount: rewardAmount,
		}
	}
	return rewards
}

// CalculateSlashAmount calculates the slash amount for invalid signals
func (p Params) CalculateSlashAmount(validatorStake int64) int64 {
	return int64(float64(validatorStake) * p.InvalidSignalSlashRate)
}

// ParamChangeProposal defines a proposal for changing module parameters
type ParamChangeProposal struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Changes     []ParamChange `json:"changes"`
}

// ParamChange defines a single parameter change
type ParamChange struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	OldValue string `json:"old_value,omitempty"`
}