package keeper

import (
	"fmt"

	keepertypes "github.com/b2network/pulsar/keeper/types"
	"github.com/b2network/pulsar/modules/signal/types"
)

// QuerySignal handles signal queries
func (k Keeper) QuerySignal(ctx keepertypes.Context, signalID string) (types.Signal, error) {
	signal, found := k.GetSignal(ctx, signalID)
	if !found {
		return types.Signal{}, types.ErrSignalNotFound
	}
	return signal, nil
}

// QuerySignalScore handles signal score queries
func (k Keeper) QuerySignalScore(ctx keepertypes.Context, signalID string) (types.SignalScore, error) {
	score, found := k.GetSignalScore(ctx, signalID)
	if !found {
		return types.SignalScore{}, fmt.Errorf("signal score not found for signal %s", signalID)
	}
	return score, nil
}

// QueryValidatorSignals handles validator signals queries
func (k Keeper) QueryValidatorSignals(ctx keepertypes.Context, validatorAddr string, limit, offset int) ([]types.Signal, error) {
	allSignals := k.GetValidatorSignals(ctx, validatorAddr)

	// Apply pagination
	if offset >= len(allSignals) {
		return []types.Signal{}, nil
	}

	end := offset + limit
	if end > len(allSignals) {
		end = len(allSignals)
	}

	return allSignals[offset:end], nil
}

// QuerySignalsByStatus handles queries for signals by status
func (k Keeper) QuerySignalsByStatus(ctx keepertypes.Context, status types.SignalStatus, limit, offset int) ([]types.Signal, error) {
	allSignals := k.GetSignalsByStatus(ctx, status)

	// Apply pagination
	if offset >= len(allSignals) {
		return []types.Signal{}, nil
	}

	end := offset + limit
	if end > len(allSignals) {
		end = len(allSignals)
	}

	return allSignals[offset:end], nil
}

// QuerySignalsByTimeRange handles time range queries
func (k Keeper) QuerySignalsByTimeRange(ctx keepertypes.Context, startTime, endTime int64, limit, offset int) ([]types.Signal, error) {
	allSignals := k.GetSignalsByTimeRange(ctx, startTime, endTime)

	// Apply pagination
	if offset >= len(allSignals) {
		return []types.Signal{}, nil
	}

	end := offset + limit
	if end > len(allSignals) {
		end = len(allSignals)
	}

	return allSignals[offset:end], nil
}

// QueryWorkTypes handles work type queries
func (k Keeper) QueryWorkTypes(ctx keepertypes.Context, enabledOnly bool) ([]types.AIWorkType, error) {
	if enabledOnly {
		return k.GetEnabledWorkTypes(ctx), nil
	}
	return k.GetAllWorkTypes(ctx), nil
}

// QueryWorkType handles single work type queries
func (k Keeper) QueryWorkType(ctx keepertypes.Context, workTypeID string) (types.AIWorkType, error) {
	workType, found := k.GetWorkType(ctx, workTypeID)
	if !found {
		return types.AIWorkType{}, types.ErrWorkTypeNotFound
	}
	return workType, nil
}

// QueryValidatorStats handles validator statistics queries
func (k Keeper) QueryValidatorStats(ctx keepertypes.Context, validatorAddr string) (types.ValidatorSignalStats, error) {
	stats := k.GetValidatorStats(ctx, validatorAddr)

	// Update effective score with current time
	params := k.GetParams(ctx)
	stats.EffectiveScore = types.AggregateValidatorScore(stats, params.ScoreParams, ctx.BlockTime().Unix())

	return stats, nil
}

// QueryLeaderboard handles leaderboard queries
func (k Keeper) QueryLeaderboard(ctx keepertypes.Context, limit int) (types.SignalLeaderboard, error) {
	return k.GetLeaderboard(ctx, limit), nil
}

// QueryParams handles parameter queries
func (k Keeper) QueryParams(ctx keepertypes.Context) (types.Params, error) {
	return k.GetParams(ctx), nil
}

// QuerySignalStats handles signal statistics queries
func (k Keeper) QuerySignalStats(ctx keepertypes.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total signals
	totalSignals := k.GetSignalCount(ctx)
	stats["total_signals"] = totalSignals

	// Signals by status
	pendingSignals := k.GetSignalCountByStatus(ctx, types.SignalStatusPending)
	validatedSignals := k.GetSignalCountByStatus(ctx, types.SignalStatusValidated)
	rejectedSignals := k.GetSignalCountByStatus(ctx, types.SignalStatusRejected)
	expiredSignals := k.GetSignalCountByStatus(ctx, types.SignalStatusExpired)

	stats["pending_signals"] = pendingSignals
	stats["validated_signals"] = validatedSignals
	stats["rejected_signals"] = rejectedSignals
	stats["expired_signals"] = expiredSignals

	// Validation rate
	if totalSignals > 0 {
		stats["validation_rate"] = float64(validatedSignals) / float64(totalSignals)
		stats["rejection_rate"] = float64(rejectedSignals) / float64(totalSignals)
	} else {
		stats["validation_rate"] = 0.0
		stats["rejection_rate"] = 0.0
	}

	// Score distribution
	scoreDistribution := k.CalculateScoreDistribution(ctx)
	stats["score_distribution"] = scoreDistribution

	// Current sequence
	stats["current_sequence"] = k.GetSignalSequence(ctx)

	return stats, nil
}

// QueryWorkTypeStats handles work type statistics queries
func (k Keeper) QueryWorkTypeStats(ctx keepertypes.Context, workTypeID string) (map[string]interface{}, error) {
	return k.GetWorkTypeStats(ctx, workTypeID)
}

// QueryWorkVerifications handles work verification queries
func (k Keeper) QueryWorkVerifications(ctx keepertypes.Context, signalID string) ([]types.WorkVerification, error) {
	return k.GetWorkVerifications(ctx, signalID), nil
}

// QueryTopValidators handles top validators queries
func (k Keeper) QueryTopValidators(ctx keepertypes.Context, limit int) ([]types.SignalLeaderboardEntry, error) {
	return k.GetTopValidators(ctx, limit), nil
}

// QueryValidatorRank handles validator rank queries
func (k Keeper) QueryValidatorRank(ctx keepertypes.Context, validatorAddr string) (uint32, error) {
	rank := k.GetValidatorRank(ctx, validatorAddr)
	return rank, nil
}

// QuerySignalsWithFilter handles filtered signal queries
func (k Keeper) QuerySignalsWithFilter(ctx keepertypes.Context, filter types.SignalFilter, limit, offset int) ([]types.Signal, error) {
	allSignals := k.FilterSignals(ctx, filter)

	// Apply pagination
	if offset >= len(allSignals) {
		return []types.Signal{}, nil
	}

	end := offset + limit
	if end > len(allSignals) {
		end = len(allSignals)
	}

	return allSignals[offset:end], nil
}

// QueryRecentSignals handles recent signals queries
func (k Keeper) QueryRecentSignals(ctx keepertypes.Context, limit int) ([]types.Signal, error) {
	// Get signals from the last 24 hours
	currentTime := ctx.BlockTime().Unix()
	startTime := currentTime - 86400 // 24 hours ago

	signals := k.GetSignalsByTimeRange(ctx, startTime, currentTime)

	// Sort by timestamp (most recent first) and limit
	if len(signals) > limit {
		signals = signals[len(signals)-limit:]
	}

	return signals, nil
}

// QueryValidatorSignalScore handles validator signal score queries
func (k Keeper) QueryValidatorSignalScore(ctx keepertypes.Context, validatorAddr string) (uint64, error) {
	score := k.GetValidatorSignalScore(ctx, validatorAddr)
	return score, nil
}

// QueryWorkTypesByCategory handles work types by category queries
func (k Keeper) QueryWorkTypesByCategory(ctx keepertypes.Context, category string) ([]types.AIWorkType, error) {
	return k.GetWorkTypeByCategory(ctx, category), nil
}

// Query response types for structured responses

// SignalQueryResponse represents a signal query response
type SignalQueryResponse struct {
	Signal types.Signal      `json:"signal"`
	Score  *types.SignalScore `json:"score,omitempty"`
}

// SignalsQueryResponse represents a signals list query response
type SignalsQueryResponse struct {
	Signals []types.Signal `json:"signals"`
	Total   int            `json:"total"`
	Limit   int            `json:"limit"`
	Offset  int            `json:"offset"`
}

// ValidatorStatsQueryResponse represents a validator stats query response
type ValidatorStatsQueryResponse struct {
	Stats types.ValidatorSignalStats `json:"stats"`
	Rank  uint32                     `json:"rank"`
}

// LeaderboardQueryResponse represents a leaderboard query response
type LeaderboardQueryResponse struct {
	Leaderboard types.SignalLeaderboard `json:"leaderboard"`
	UpdatedAt   int64                   `json:"updated_at"`
}

// QuerySignalWithScore returns signal with its score
func (k Keeper) QuerySignalWithScore(ctx keepertypes.Context, signalID string) (SignalQueryResponse, error) {
	signal, found := k.GetSignal(ctx, signalID)
	if !found {
		return SignalQueryResponse{}, types.ErrSignalNotFound
	}

	response := SignalQueryResponse{
		Signal: signal,
	}

	// Add score if available
	if score, found := k.GetSignalScore(ctx, signalID); found {
		response.Score = &score
	}

	return response, nil
}

// QuerySignalsWithPagination returns paginated signals with metadata
func (k Keeper) QuerySignalsWithPagination(ctx keepertypes.Context, filter types.SignalFilter, limit, offset int) (SignalsQueryResponse, error) {
	allSignals := k.FilterSignals(ctx, filter)
	total := len(allSignals)

	// Apply pagination
	signals := []types.Signal{}
	if offset < total {
		end := offset + limit
		if end > total {
			end = total
		}
		signals = allSignals[offset:end]
	}

	return SignalsQueryResponse{
		Signals: signals,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	}, nil
}

// QueryValidatorStatsWithRank returns validator stats with rank
func (k Keeper) QueryValidatorStatsWithRank(ctx keepertypes.Context, validatorAddr string) (ValidatorStatsQueryResponse, error) {
	stats := k.GetValidatorStats(ctx, validatorAddr)

	// Update effective score
	params := k.GetParams(ctx)
	stats.EffectiveScore = types.AggregateValidatorScore(stats, params.ScoreParams, ctx.BlockTime().Unix())

	rank := k.GetValidatorRank(ctx, validatorAddr)

	return ValidatorStatsQueryResponse{
		Stats: stats,
		Rank:  rank,
	}, nil
}