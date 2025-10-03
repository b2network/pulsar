package types

import (
	"fmt"
	"math"
	"sort"
	"time"
)

// SignalScore represents the calculated score for a signal
type SignalScore struct {
	SignalID        string  `json:"signal_id"`
	BaseScore       uint64  `json:"base_score"`
	WorkMultiplier  float64 `json:"work_multiplier"`
	TimeDecayFactor float64 `json:"time_decay_factor"`
	QualityFactor   float64 `json:"quality_factor"`
	FinalScore      uint64  `json:"final_score"`
	CalculatedAt    int64   `json:"calculated_at"`
	Components      ScoreComponents `json:"components"`
}

// ScoreComponents breaks down the score calculation
type ScoreComponents struct {
	ResourceScore     uint64  `json:"resource_score"`
	ComplexityScore   uint64  `json:"complexity_score"`
	AccuracyScore     uint64  `json:"accuracy_score"`
	TimelinessScore   uint64  `json:"timeliness_score"`
	VerificationBonus uint64  `json:"verification_bonus"`
}

// ValidatorSignalStats tracks signal statistics for a validator
type ValidatorSignalStats struct {
	ValidatorAddress   string   `json:"validator_address"`
	TotalSignals       uint64   `json:"total_signals"`
	ValidSignals       uint64   `json:"valid_signals"`
	RejectedSignals    uint64   `json:"rejected_signals"`
	TotalScore         uint64   `json:"total_score"`
	AverageScore       uint64   `json:"average_score"`
	LastSignalTime     int64    `json:"last_signal_time"`
	ActiveSignals      []string `json:"active_signals"`
	RecentScores       []uint64 `json:"recent_scores"`
	ScoreDecayRate     float64  `json:"score_decay_rate"`
	EffectiveScore     uint64   `json:"effective_score"`
	Rank               uint32   `json:"rank"`
}

// SignalScoreParams defines parameters for score calculation
type SignalScoreParams struct {
	BaseScoreWeight       float64 `json:"base_score_weight"`
	ResourceWeight        float64 `json:"resource_weight"`
	ComplexityWeight      float64 `json:"complexity_weight"`
	AccuracyWeight        float64 `json:"accuracy_weight"`
	TimeDecayHalfLife     int64   `json:"time_decay_half_life"`     // in seconds
	MaxScoreAge           int64   `json:"max_score_age"`            // in seconds
	VerificationBonusRate float64 `json:"verification_bonus_rate"`
	MinScoreThreshold     uint64  `json:"min_score_threshold"`
}

// DefaultSignalScoreParams returns default scoring parameters
func DefaultSignalScoreParams() SignalScoreParams {
	return SignalScoreParams{
		BaseScoreWeight:       0.3,
		ResourceWeight:        0.25,
		ComplexityWeight:      0.25,
		AccuracyWeight:        0.2,
		TimeDecayHalfLife:     86400,  // 1 day
		MaxScoreAge:           604800,  // 7 days
		VerificationBonusRate: 0.1,
		MinScoreThreshold:     10,
	}
}

// CalculateSignalScore calculates the final score for a signal
func CalculateSignalScore(
	signal Signal,
	workType AIWorkType,
	params SignalScoreParams,
	currentTime int64,
) SignalScore {
	score := SignalScore{
		SignalID:       signal.ID,
		CalculatedAt:   currentTime,
		WorkMultiplier: workType.ScoreMultiplier,
	}

	// Calculate base score from work type
	score.BaseScore = workType.BaseScore

	// Calculate component scores
	components := ScoreComponents{}

	// Resource score based on computational resources used
	components.ResourceScore = signal.WorkProof.ResourcesUsed.CalculateResourceScore()

	// Complexity score based on computation time and resource diversity
	components.ComplexityScore = calculateComplexityScore(signal.WorkProof)

	// Accuracy score (would be determined by verification)
	components.AccuracyScore = 100 // Default, updated after verification

	// Timeliness score based on how recent the signal is
	components.TimelinessScore = calculateTimelinessScore(signal.Timestamp, currentTime)

	// Verification bonus if signal has been verified
	if signal.Status == SignalStatusValidated {
		components.VerificationBonus = uint64(float64(score.BaseScore) * params.VerificationBonusRate)
	}

	score.Components = components

	// Calculate time decay factor
	age := currentTime - signal.Timestamp
	if age > params.MaxScoreAge {
		score.TimeDecayFactor = 0
	} else {
		halfLives := float64(age) / float64(params.TimeDecayHalfLife)
		score.TimeDecayFactor = math.Pow(0.5, halfLives)
	}

	// Quality factor (default to 1.0, can be adjusted based on verification)
	score.QualityFactor = 1.0

	// Calculate final score
	weightedScore := float64(score.BaseScore)*params.BaseScoreWeight +
		float64(components.ResourceScore)*params.ResourceWeight +
		float64(components.ComplexityScore)*params.ComplexityWeight +
		float64(components.AccuracyScore)*params.AccuracyWeight +
		float64(components.VerificationBonus)

	score.FinalScore = uint64(weightedScore * score.WorkMultiplier * score.TimeDecayFactor * score.QualityFactor)

	// Apply minimum threshold
	if score.FinalScore < params.MinScoreThreshold {
		score.FinalScore = 0
	}

	return score
}

// calculateComplexityScore calculates a complexity score based on work proof
func calculateComplexityScore(wp WorkProof) uint64 {
	// Consider computation time and resource diversity
	timeScore := uint64(wp.ComputationTime / 100) // Convert ms to scoring units

	// Count number of different resource types used
	resourceDiversity := 0
	if wp.ResourcesUsed.CPUCycles > 0 {
		resourceDiversity++
	}
	if wp.ResourcesUsed.GPUTimeMs > 0 {
		resourceDiversity++
	}
	if wp.ResourcesUsed.NetworkBandwidth > 0 {
		resourceDiversity++
	}
	if wp.ResourcesUsed.StorageBytes > 0 {
		resourceDiversity++
	}

	diversityBonus := uint64(resourceDiversity * 25)

	return timeScore + diversityBonus
}

// calculateTimelinessScore calculates score based on signal recency
func calculateTimelinessScore(signalTime, currentTime int64) uint64 {
	age := currentTime - signalTime
	if age <= 0 {
		return 100 // Maximum score for very recent signals
	}

	// Decay over 24 hours
	hoursPassed := age / 3600
	if hoursPassed >= 24 {
		return 0
	}

	return uint64(100 - (hoursPassed * 4)) // Lose 4 points per hour
}

// AggregateValidatorScore calculates the aggregate score for a validator
func AggregateValidatorScore(stats ValidatorSignalStats, params SignalScoreParams, currentTime int64) uint64 {
	if stats.TotalSignals == 0 {
		return 0
	}

	// Calculate average of recent scores with time decay
	var weightedSum float64
	var weightSum float64

	for i, score := range stats.RecentScores {
		// More recent scores have higher weight
		recencyWeight := 1.0 / float64(i+1)
		weightedSum += float64(score) * recencyWeight
		weightSum += recencyWeight
	}

	avgScore := uint64(0)
	if weightSum > 0 {
		avgScore = uint64(weightedSum / weightSum)
	}

	// Apply decay based on last signal time
	timeSinceLastSignal := currentTime - stats.LastSignalTime
	decayFactor := 1.0
	if timeSinceLastSignal > 0 {
		halfLives := float64(timeSinceLastSignal) / float64(params.TimeDecayHalfLife)
		decayFactor = math.Pow(0.5, halfLives)
	}

	// Factor in signal validity rate
	validityRate := float64(stats.ValidSignals) / float64(stats.TotalSignals)

	// Calculate effective score
	effectiveScore := uint64(float64(avgScore) * decayFactor * validityRate)

	return effectiveScore
}

// UpdateValidatorStats updates validator statistics with a new signal
func UpdateValidatorStats(stats *ValidatorSignalStats, signal Signal, score uint64) {
	stats.TotalSignals++

	if signal.Status == SignalStatusValidated {
		stats.ValidSignals++
	} else if signal.Status == SignalStatusRejected {
		stats.RejectedSignals++
	}

	stats.TotalScore += score
	stats.AverageScore = stats.TotalScore / stats.TotalSignals
	stats.LastSignalTime = signal.Timestamp

	// Update active signals
	stats.ActiveSignals = append(stats.ActiveSignals, signal.ID)
	if len(stats.ActiveSignals) > 100 { // Keep only last 100
		stats.ActiveSignals = stats.ActiveSignals[len(stats.ActiveSignals)-100:]
	}

	// Update recent scores
	stats.RecentScores = append(stats.RecentScores, score)
	if len(stats.RecentScores) > 20 { // Keep only last 20 scores
		stats.RecentScores = stats.RecentScores[len(stats.RecentScores)-20:]
	}
}

// SignalLeaderboard represents a leaderboard of validators by signal score
type SignalLeaderboard struct {
	UpdatedAt  int64                  `json:"updated_at"`
	TotalScore uint64                 `json:"total_score"`
	Entries    []SignalLeaderboardEntry `json:"entries"`
}

// SignalLeaderboardEntry represents an entry in the signal leaderboard
type SignalLeaderboardEntry struct {
	ValidatorAddress string `json:"validator_address"`
	Score            uint64 `json:"score"`
	SignalCount      uint64 `json:"signal_count"`
	Rank             uint32 `json:"rank"`
	PercentOfTotal   float64 `json:"percent_of_total"`
}

// CreateLeaderboard creates a leaderboard from validator stats
func CreateLeaderboard(statsMap map[string]*ValidatorSignalStats, limit int) SignalLeaderboard {
	leaderboard := SignalLeaderboard{
		UpdatedAt: time.Now().Unix(),
		Entries:   make([]SignalLeaderboardEntry, 0),
	}

	// Convert map to slice for sorting
	entries := make([]SignalLeaderboardEntry, 0, len(statsMap))
	for addr, stats := range statsMap {
		entries = append(entries, SignalLeaderboardEntry{
			ValidatorAddress: addr,
			Score:           stats.EffectiveScore,
			SignalCount:     stats.TotalSignals,
		})
		leaderboard.TotalScore += stats.EffectiveScore
	}

	// Sort by score (descending)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Score > entries[j].Score
	})

	// Assign ranks and calculate percentages
	for i, entry := range entries {
		entry.Rank = uint32(i + 1)
		if leaderboard.TotalScore > 0 {
			entry.PercentOfTotal = float64(entry.Score) / float64(leaderboard.TotalScore) * 100
		}
		entries[i] = entry

		if limit > 0 && i >= limit-1 {
			break
		}
	}

	if limit > 0 && len(entries) > limit {
		leaderboard.Entries = entries[:limit]
	} else {
		leaderboard.Entries = entries
	}

	return leaderboard
}

// ScoreDistribution analyzes the distribution of scores
type ScoreDistribution struct {
	Min      uint64  `json:"min"`
	Max      uint64  `json:"max"`
	Mean     uint64  `json:"mean"`
	Median   uint64  `json:"median"`
	StdDev   float64 `json:"std_dev"`
	Quartiles [3]uint64 `json:"quartiles"` // Q1, Q2 (median), Q3
}

// AnalyzeScoreDistribution analyzes the distribution of signal scores
func AnalyzeScoreDistribution(scores []uint64) ScoreDistribution {
	if len(scores) == 0 {
		return ScoreDistribution{}
	}

	dist := ScoreDistribution{}

	// Sort scores for percentile calculations
	sorted := make([]uint64, len(scores))
	copy(sorted, scores)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	// Min and Max
	dist.Min = sorted[0]
	dist.Max = sorted[len(sorted)-1]

	// Mean
	var sum uint64
	for _, score := range scores {
		sum += score
	}
	dist.Mean = sum / uint64(len(scores))

	// Median and Quartiles
	dist.Median = sorted[len(sorted)/2]
	dist.Quartiles[0] = sorted[len(sorted)/4]     // Q1
	dist.Quartiles[1] = dist.Median               // Q2
	dist.Quartiles[2] = sorted[3*len(sorted)/4]   // Q3

	// Standard Deviation
	var variance float64
	meanFloat := float64(dist.Mean)
	for _, score := range scores {
		diff := float64(score) - meanFloat
		variance += diff * diff
	}
	variance /= float64(len(scores))
	dist.StdDev = math.Sqrt(variance)

	return dist
}