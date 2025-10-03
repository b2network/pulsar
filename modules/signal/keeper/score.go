package keeper

import (
	"fmt"

	keepertypes "github.com/b2network/pulsar/keeper/types"
	"github.com/b2network/pulsar/modules/signal/types"
)

// CalculateSignalScore calculates the score for a signal
func (k Keeper) CalculateSignalScore(ctx keepertypes.Context, signal types.Signal) (types.SignalScore, error) {
	params := k.GetParams(ctx)

	// Get work type for score calculation
	workType, found := k.GetWorkType(ctx, signal.WorkProof.WorkType)
	if !found {
		return types.SignalScore{}, types.ErrWorkTypeNotFound
	}

	// Calculate score using the scoring algorithm
	score := types.CalculateSignalScore(
		signal,
		workType,
		params.ScoreParams,
		ctx.BlockTime().Unix(),
	)

	// Apply work type weight from params
	weight := params.GetWorkTypeWeight(signal.WorkProof.WorkType)
	score.FinalScore = uint64(float64(score.FinalScore) * weight)

	// Apply minimum threshold
	if score.FinalScore < params.MinSignalScore {
		score.FinalScore = 0
	}

	return score, nil
}

// StoreSignalScore stores a signal score
func (k Keeper) StoreSignalScore(ctx keepertypes.Context, score types.SignalScore) error {
	store := k.GetKVStore(ctx)

	bz, err := k.GetCodec().Marshal(&score)
	if err != nil {
		return fmt.Errorf("failed to marshal signal score: %w", err)
	}

	key := types.GetSignalScoreKey(score.SignalID)
	store.Set(key, bz)

	return nil
}

// GetSignalScore retrieves a signal score by signal ID
func (k Keeper) GetSignalScore(ctx keepertypes.Context, signalID string) (types.SignalScore, bool) {
	store := k.GetKVStore(ctx)
	key := types.GetSignalScoreKey(signalID)

	bz := store.Get(key)
	if bz == nil {
		return types.SignalScore{}, false
	}

	var score types.SignalScore
	if err := k.GetCodec().Unmarshal(bz, &score); err != nil {
		k.Logger(ctx).Error("failed to unmarshal signal score", "error", err, "signal_id", signalID)
		return types.SignalScore{}, false
	}

	return score, true
}

// DeleteSignalScore removes a signal score
func (k Keeper) DeleteSignalScore(ctx keepertypes.Context, signalID string) {
	store := k.GetKVStore(ctx)
	key := types.GetSignalScoreKey(signalID)
	store.Delete(key)
}

// GetAllSignalScores returns all signal scores
func (k Keeper) GetAllSignalScores(ctx keepertypes.Context) []types.SignalScore {
	store := k.GetKVStore(ctx)
	iterator := store.Iterator(types.SignalScoreKey, append(types.SignalScoreKey, 0xFF))
	defer iterator.Close()

	var scores []types.SignalScore
	for ; iterator.Valid(); iterator.Next() {
		var score types.SignalScore
		if err := k.GetCodec().Unmarshal(iterator.Value(), &score); err != nil {
			k.Logger(ctx).Error("failed to unmarshal signal score", "error", err)
			continue
		}
		scores = append(scores, score)
	}

	return scores
}

// UpdateValidatorStats updates the statistics for a validator
func (k Keeper) UpdateValidatorStats(ctx keepertypes.Context, validatorAddr string, signal types.Signal, score uint64) error {
	stats := k.GetValidatorStats(ctx, validatorAddr)

	// Update stats with the new signal
	types.UpdateValidatorStats(&stats, signal, score)

	// Recalculate effective score
	params := k.GetParams(ctx)
	stats.EffectiveScore = types.AggregateValidatorScore(stats, params.ScoreParams, ctx.BlockTime().Unix())

	// Store updated stats
	if err := k.SetValidatorStats(ctx, stats); err != nil {
		return fmt.Errorf("failed to store validator stats: %w", err)
	}

	// Emit stats update event
	types.EmitSignalEvent(ctx, types.NewStatsUpdatedEvent(stats))

	return nil
}

// GetValidatorStats retrieves validator statistics
func (k Keeper) GetValidatorStats(ctx keepertypes.Context, validatorAddr string) types.ValidatorSignalStats {
	store := k.GetKVStore(ctx)
	key := types.GetValidatorStatsKey(validatorAddr)

	bz := store.Get(key)
	if bz == nil {
		// Return empty stats if not found
		return types.ValidatorSignalStats{
			ValidatorAddress: validatorAddr,
			TotalSignals:     0,
			ValidSignals:     0,
			RejectedSignals:  0,
			TotalScore:       0,
			AverageScore:     0,
			LastSignalTime:   0,
			ActiveSignals:    []string{},
			RecentScores:     []uint64{},
			ScoreDecayRate:   k.GetParams(ctx).ScoreDecayRate,
			EffectiveScore:   0,
			Rank:             0,
		}
	}

	var stats types.ValidatorSignalStats
	if err := k.GetCodec().Unmarshal(bz, &stats); err != nil {
		k.Logger(ctx).Error("failed to unmarshal validator stats", "error", err, "validator", validatorAddr)
		// Return empty stats on error
		return types.ValidatorSignalStats{
			ValidatorAddress: validatorAddr,
			TotalSignals:     0,
			ValidSignals:     0,
			RejectedSignals:  0,
			TotalScore:       0,
			AverageScore:     0,
			LastSignalTime:   0,
			ActiveSignals:    []string{},
			RecentScores:     []uint64{},
			ScoreDecayRate:   k.GetParams(ctx).ScoreDecayRate,
			EffectiveScore:   0,
			Rank:             0,
		}
	}

	return stats
}

// SetValidatorStats stores validator statistics
func (k Keeper) SetValidatorStats(ctx keepertypes.Context, stats types.ValidatorSignalStats) error {
	store := k.GetKVStore(ctx)

	bz, err := k.GetCodec().Marshal(&stats)
	if err != nil {
		return fmt.Errorf("failed to marshal validator stats: %w", err)
	}

	key := types.GetValidatorStatsKey(stats.ValidatorAddress)
	store.Set(key, bz)

	return nil
}

// GetAllValidatorStats returns statistics for all validators
func (k Keeper) GetAllValidatorStats(ctx keepertypes.Context) map[string]*types.ValidatorSignalStats {
	store := k.GetKVStore(ctx)
	iterator := store.Iterator(types.ValidatorStatsKey, append(types.ValidatorStatsKey, 0xFF))
	defer iterator.Close()

	statsMap := make(map[string]*types.ValidatorSignalStats)
	for ; iterator.Valid(); iterator.Next() {
		var stats types.ValidatorSignalStats
		if err := k.GetCodec().Unmarshal(iterator.Value(), &stats); err != nil {
			k.Logger(ctx).Error("failed to unmarshal validator stats", "error", err)
			continue
		}
		statsMap[stats.ValidatorAddress] = &stats
	}

	return statsMap
}

// GetLeaderboard generates a leaderboard of validators by signal score
func (k Keeper) GetLeaderboard(ctx keepertypes.Context, limit int) types.SignalLeaderboard {
	statsMap := k.GetAllValidatorStats(ctx)

	// Update effective scores for all validators
	params := k.GetParams(ctx)
	currentTime := ctx.BlockTime().Unix()

	for _, stats := range statsMap {
		stats.EffectiveScore = types.AggregateValidatorScore(*stats, params.ScoreParams, currentTime)
	}

	return types.CreateLeaderboard(statsMap, limit)
}

// UpdateLeaderboard updates the stored leaderboard
func (k Keeper) UpdateLeaderboard(ctx keepertypes.Context) error {
	leaderboard := k.GetLeaderboard(ctx, 100) // Top 100

	store := k.GetKVStore(ctx)
	bz, err := k.GetCodec().Marshal(&leaderboard)
	if err != nil {
		return fmt.Errorf("failed to marshal leaderboard: %w", err)
	}

	store.Set(types.LeaderboardKey, bz)

	// Emit leaderboard update event if there are entries
	if len(leaderboard.Entries) > 0 {
		topEntry := leaderboard.Entries[0]
		types.EmitSignalEvent(ctx, types.NewLeaderboardUpdateEvent(
			topEntry.ValidatorAddress, topEntry.Score))
	}

	return nil
}

// GetStoredLeaderboard retrieves the stored leaderboard
func (k Keeper) GetStoredLeaderboard(ctx keepertypes.Context) (types.SignalLeaderboard, bool) {
	store := k.GetKVStore(ctx)
	bz := store.Get(types.LeaderboardKey)

	if bz == nil {
		return types.SignalLeaderboard{}, false
	}

	var leaderboard types.SignalLeaderboard
	if err := k.GetCodec().Unmarshal(bz, &leaderboard); err != nil {
		k.Logger(ctx).Error("failed to unmarshal leaderboard", "error", err)
		return types.SignalLeaderboard{}, false
	}

	return leaderboard, true
}

// GetTopValidators returns the top N validators by signal score
func (k Keeper) GetTopValidators(ctx keepertypes.Context, limit int) []types.SignalLeaderboardEntry {
	leaderboard := k.GetLeaderboard(ctx, limit)
	return leaderboard.Entries
}

// GetValidatorRank returns the rank of a specific validator
func (k Keeper) GetValidatorRank(ctx keepertypes.Context, validatorAddr string) uint32 {
	leaderboard := k.GetLeaderboard(ctx, 0) // Get all

	for _, entry := range leaderboard.Entries {
		if entry.ValidatorAddress == validatorAddr {
			return entry.Rank
		}
	}

	return 0 // Not found or no rank
}

// CalculateScoreDistribution analyzes the distribution of signal scores
func (k Keeper) CalculateScoreDistribution(ctx keepertypes.Context) types.ScoreDistribution {
	scores := k.GetAllSignalScores(ctx)

	scoreValues := make([]uint64, len(scores))
	for i, score := range scores {
		scoreValues[i] = score.FinalScore
	}

	return types.AnalyzeScoreDistribution(scoreValues)
}

// GetValidatorSignalScore returns the aggregate signal score for a validator
func (k Keeper) GetValidatorSignalScore(ctx keepertypes.Context, validatorAddr string) uint64 {
	stats := k.GetValidatorStats(ctx, validatorAddr)
	return stats.EffectiveScore
}

// RecalculateAllScores recalculates scores for all signals (useful for param updates)
func (k Keeper) RecalculateAllScores(ctx keepertypes.Context) error {
	signals := k.GetAllSignals(ctx)
	params := k.GetParams(ctx)

	for _, signal := range signals {
		if signal.Status != types.SignalStatusValidated {
			continue
		}

		workType, found := k.GetWorkType(ctx, signal.WorkProof.WorkType)
		if !found {
			continue
		}

		// Recalculate score
		score := types.CalculateSignalScore(
			signal,
			workType,
			params.ScoreParams,
			ctx.BlockTime().Unix(),
		)

		// Apply work type weight
		weight := params.GetWorkTypeWeight(signal.WorkProof.WorkType)
		score.FinalScore = uint64(float64(score.FinalScore) * weight)

		// Update signal and score
		signal.Score = score.FinalScore
		if err := k.StoreSignal(ctx, signal); err != nil {
			k.Logger(ctx).Error("failed to update signal score", "error", err, "signal_id", signal.ID)
			continue
		}
		if err := k.StoreSignalScore(ctx, score); err != nil {
			k.Logger(ctx).Error("failed to store signal score", "error", err, "signal_id", signal.ID)
			continue
		}
	}

	// Recalculate validator stats
	return k.recalculateValidatorStats(ctx)
}

// recalculateValidatorStats recalculates statistics for all validators
func (k Keeper) recalculateValidatorStats(ctx keepertypes.Context) error {
	params := k.GetParams(ctx)
	currentTime := ctx.BlockTime().Unix()

	// Get all validators with stats
	statsMap := k.GetAllValidatorStats(ctx)

	for validatorAddr, stats := range statsMap {
		// Recalculate effective score
		stats.EffectiveScore = types.AggregateValidatorScore(*stats, params.ScoreParams, currentTime)

		// Store updated stats
		if err := k.SetValidatorStats(ctx, *stats); err != nil {
			k.Logger(ctx).Error("failed to update validator stats", "error", err, "validator", validatorAddr)
			continue
		}
	}

	// Update leaderboard
	return k.UpdateLeaderboard(ctx)
}