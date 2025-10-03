package types

import (
	"fmt"

	keepertypes "github.com/b2network/pulsar/keeper/types"
)

// Event types for the signal module
const (
	EventTypeSignalSubmitted   = "signal_submitted"
	EventTypeSignalValidated   = "signal_validated"
	EventTypeSignalRejected    = "signal_rejected"
	EventTypeSignalExpired     = "signal_expired"
	EventTypeWorkVerified      = "work_verified"
	EventTypeScoreCalculated   = "score_calculated"
	EventTypeBatchProcessed    = "batch_processed"
	EventTypeRewardDistributed = "reward_distributed"
	EventTypeSlashApplied      = "slash_applied"
	EventTypeWorkTypeUpdated   = "work_type_updated"
	EventTypeStatsUpdated      = "stats_updated"
	EventTypeLeaderboardUpdate = "leaderboard_update"
)

// Attribute keys for events
const (
	AttributeKeySignalID         = "signal_id"
	AttributeKeyCreator          = "creator"
	AttributeKeyValidator        = "validator"
	AttributeKeySignalType       = "signal_type"
	AttributeKeyWorkType         = "work_type"
	AttributeKeyScore            = "score"
	AttributeKeyStatus           = "status"
	AttributeKeyTimestamp        = "timestamp"
	AttributeKeyBatchID          = "batch_id"
	AttributeKeyBatchSize        = "batch_size"
	AttributeKeyRewardAmount     = "reward_amount"
	AttributeKeySlashAmount      = "slash_amount"
	AttributeKeyVerifier         = "verifier"
	AttributeKeyVerificationResult = "verification_result"
	AttributeKeyRank             = "rank"
	AttributeKeyTotalSignals     = "total_signals"
	AttributeKeyEffectiveScore   = "effective_score"
	AttributeKeyReason           = "reason"
	AttributeKeyComputationTime  = "computation_time"
	AttributeKeyResourcesUsed    = "resources_used"
)

// NewSignalSubmittedEvent creates a new signal submitted event
func NewSignalSubmittedEvent(signal Signal) keepertypes.Event {
	return keepertypes.Event{
		Type: EventTypeSignalSubmitted,
		Attributes: []keepertypes.Attribute{
			{Key: AttributeKeySignalID, Value: signal.ID},
			{Key: AttributeKeyCreator, Value: signal.Creator},
			{Key: AttributeKeyValidator, Value: signal.ValidatorAddress},
			{Key: AttributeKeySignalType, Value: signal.Type},
			{Key: AttributeKeyWorkType, Value: signal.WorkProof.WorkType},
			{Key: AttributeKeyTimestamp, Value: fmt.Sprintf("%d", signal.Timestamp)},
			{Key: AttributeKeyStatus, Value: signal.Status.String()},
		},
	}
}

// NewSignalValidatedEvent creates a new signal validated event
func NewSignalValidatedEvent(signalID string, score uint64, validatorAddr string) keepertypes.Event {
	return keepertypes.Event{
		Type: EventTypeSignalValidated,
		Attributes: []keepertypes.Attribute{
			{Key: AttributeKeySignalID, Value: signalID},
			{Key: AttributeKeyValidator, Value: validatorAddr},
			{Key: AttributeKeyScore, Value: fmt.Sprintf("%d", score)},
			{Key: AttributeKeyStatus, Value: SignalStatusValidated.String()},
		},
	}
}

// NewSignalRejectedEvent creates a new signal rejected event
func NewSignalRejectedEvent(signalID string, reason string, validatorAddr string) keepertypes.Event {
	return keepertypes.Event{
		Type: EventTypeSignalRejected,
		Attributes: []keepertypes.Attribute{
			{Key: AttributeKeySignalID, Value: signalID},
			{Key: AttributeKeyValidator, Value: validatorAddr},
			{Key: AttributeKeyStatus, Value: SignalStatusRejected.String()},
			{Key: AttributeKeyReason, Value: reason},
		},
	}
}

// NewWorkVerifiedEvent creates a new work verified event
func NewWorkVerifiedEvent(verification WorkVerification) keepertypes.Event {
	return keepertypes.Event{
		Type: EventTypeWorkVerified,
		Attributes: []keepertypes.Attribute{
			{Key: AttributeKeySignalID, Value: verification.SignalID},
			{Key: AttributeKeyVerifier, Value: verification.VerifierAddress},
			{Key: AttributeKeyVerificationResult, Value: fmt.Sprintf("%t", verification.IsValid)},
			{Key: AttributeKeyScore, Value: fmt.Sprintf("%d", verification.VerificationScore)},
		},
	}
}

// NewScoreCalculatedEvent creates a new score calculated event
func NewScoreCalculatedEvent(score SignalScore) keepertypes.Event {
	return keepertypes.Event{
		Type: EventTypeScoreCalculated,
		Attributes: []keepertypes.Attribute{
			{Key: AttributeKeySignalID, Value: score.SignalID},
			{Key: AttributeKeyScore, Value: fmt.Sprintf("%d", score.FinalScore)},
			{Key: AttributeKeyTimestamp, Value: fmt.Sprintf("%d", score.CalculatedAt)},
		},
	}
}

// NewBatchProcessedEvent creates a new batch processed event
func NewBatchProcessedEvent(batchID string, batchSize int, totalScore uint64) keepertypes.Event {
	return keepertypes.Event{
		Type: EventTypeBatchProcessed,
		Attributes: []keepertypes.Attribute{
			{Key: AttributeKeyBatchID, Value: batchID},
			{Key: AttributeKeyBatchSize, Value: fmt.Sprintf("%d", batchSize)},
			{Key: AttributeKeyScore, Value: fmt.Sprintf("%d", totalScore)},
		},
	}
}

// NewRewardDistributedEvent creates a new reward distributed event
func NewRewardDistributedEvent(validatorAddr string, amount keepertypes.Coins) keepertypes.Event {
	return keepertypes.Event{
		Type: EventTypeRewardDistributed,
		Attributes: []keepertypes.Attribute{
			{Key: AttributeKeyValidator, Value: validatorAddr},
			{Key: AttributeKeyRewardAmount, Value: amount.String()},
		},
	}
}

// NewSlashAppliedEvent creates a new slash applied event
func NewSlashAppliedEvent(validatorAddr string, amount int64, reason string) keepertypes.Event {
	return keepertypes.Event{
		Type: EventTypeSlashApplied,
		Attributes: []keepertypes.Attribute{
			{Key: AttributeKeyValidator, Value: validatorAddr},
			{Key: AttributeKeySlashAmount, Value: fmt.Sprintf("%d", amount)},
			{Key: AttributeKeyReason, Value: reason},
		},
	}
}

// NewStatsUpdatedEvent creates a new stats updated event
func NewStatsUpdatedEvent(stats ValidatorSignalStats) keepertypes.Event {
	return keepertypes.Event{
		Type: EventTypeStatsUpdated,
		Attributes: []keepertypes.Attribute{
			{Key: AttributeKeyValidator, Value: stats.ValidatorAddress},
			{Key: AttributeKeyTotalSignals, Value: fmt.Sprintf("%d", stats.TotalSignals)},
			{Key: AttributeKeyEffectiveScore, Value: fmt.Sprintf("%d", stats.EffectiveScore)},
			{Key: AttributeKeyRank, Value: fmt.Sprintf("%d", stats.Rank)},
		},
	}
}

// NewLeaderboardUpdateEvent creates a new leaderboard update event
func NewLeaderboardUpdateEvent(topValidator string, topScore uint64) keepertypes.Event {
	return keepertypes.Event{
		Type: EventTypeLeaderboardUpdate,
		Attributes: []keepertypes.Attribute{
			{Key: AttributeKeyValidator, Value: topValidator},
			{Key: AttributeKeyScore, Value: fmt.Sprintf("%d", topScore)},
			{Key: AttributeKeyRank, Value: "1"},
		},
	}
}

// EmitSignalEvent is a helper function to emit signal-related events
func EmitSignalEvent(ctx keepertypes.Context, event keepertypes.Event) {
	ctx.EventManager().EmitEvent(event)
}