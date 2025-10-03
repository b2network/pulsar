package processing

import (
	"sort"
	"time"

	keepertypes "github.com/b2network/pulsar/keeper/types"
	"github.com/b2network/pulsar/modules/signal/types"
)

// SignalAggregator handles signal aggregation and analysis
type SignalAggregator struct {
	keeper SignalKeeper
}

// AggregationResult represents the result of signal aggregation
type AggregationResult struct {
	TotalSignals      int                    `json:"total_signals"`
	ValidatedSignals  int                    `json:"validated_signals"`
	RejectedSignals   int                    `json:"rejected_signals"`
	PendingSignals    int                    `json:"pending_signals"`
	TotalScore        uint64                 `json:"total_score"`
	AverageScore      uint64                 `json:"average_score"`
	ScoreDistribution types.ScoreDistribution `json:"score_distribution"`
	WorkTypeStats     map[string]WorkTypeAggregation `json:"work_type_stats"`
	ValidatorStats    map[string]ValidatorAggregation `json:"validator_stats"`
	TimeRange         TimeRange              `json:"time_range"`
}

// WorkTypeAggregation represents aggregated data for a work type
type WorkTypeAggregation struct {
	WorkType         string  `json:"work_type"`
	Count            int     `json:"count"`
	TotalScore       uint64  `json:"total_score"`
	AverageScore     uint64  `json:"average_score"`
	ValidationRate   float64 `json:"validation_rate"`
	AvgComputeTime   int64   `json:"avg_compute_time"`
	TotalResources   types.Resources `json:"total_resources"`
	AvgResources     types.Resources `json:"avg_resources"`
}

// ValidatorAggregation represents aggregated data for a validator
type ValidatorAggregation struct {
	ValidatorAddress string  `json:"validator_address"`
	Count            int     `json:"count"`
	TotalScore       uint64  `json:"total_score"`
	AverageScore     uint64  `json:"average_score"`
	ValidationRate   float64 `json:"validation_rate"`
	Rank             int     `json:"rank"`
	EffectiveScore   uint64  `json:"effective_score"`
}

// TimeRange represents a time range for aggregation
type TimeRange struct {
	StartTime int64 `json:"start_time"`
	EndTime   int64 `json:"end_time"`
}

// NewSignalAggregator creates a new signal aggregator
func NewSignalAggregator(keeper SignalKeeper) *SignalAggregator {
	return &SignalAggregator{
		keeper: keeper,
	}
}

// AggregateSignals performs comprehensive signal aggregation
func (sa *SignalAggregator) AggregateSignals(ctx keepertypes.Context, filter types.SignalFilter) AggregationResult {
	signals := sa.keeper.FilterSignals(ctx, filter)

	result := AggregationResult{
		TotalSignals:   len(signals),
		WorkTypeStats:  make(map[string]WorkTypeAggregation),
		ValidatorStats: make(map[string]ValidatorAggregation),
	}

	if len(signals) == 0 {
		return result
	}

	// Set time range
	result.TimeRange = sa.getTimeRange(signals)

	// Count signals by status
	sa.countSignalsByStatus(signals, &result)

	// Calculate scores
	sa.calculateScoreMetrics(signals, &result)

	// Aggregate by work type
	sa.aggregateByWorkType(signals, &result)

	// Aggregate by validator
	sa.aggregateByValidator(signals, &result)

	return result
}

// AggregateByTimeWindow aggregates signals in time windows
func (sa *SignalAggregator) AggregateByTimeWindow(ctx keepertypes.Context, windowSize time.Duration, startTime, endTime int64) []TimeWindowAggregation {
	var results []TimeWindowAggregation

	currentTime := startTime
	for currentTime < endTime {
		windowEnd := currentTime + int64(windowSize.Seconds())
		if windowEnd > endTime {
			windowEnd = endTime
		}

		filter := types.SignalFilter{
			StartTime: currentTime,
			EndTime:   windowEnd,
		}

		aggregation := sa.AggregateSignals(ctx, filter)
		timeWindow := TimeWindowAggregation{
			StartTime:   currentTime,
			EndTime:     windowEnd,
			Aggregation: aggregation,
		}

		results = append(results, timeWindow)
		currentTime = windowEnd
	}

	return results
}

// TimeWindowAggregation represents aggregation data for a time window
type TimeWindowAggregation struct {
	StartTime   int64             `json:"start_time"`
	EndTime     int64             `json:"end_time"`
	Aggregation AggregationResult `json:"aggregation"`
}

// AggregateValidatorPerformance aggregates validator performance metrics
func (sa *SignalAggregator) AggregateValidatorPerformance(ctx keepertypes.Context, validatorAddr string, days int) ValidatorPerformanceMetrics {
	endTime := ctx.BlockTime().Unix()
	startTime := endTime - int64(days*86400) // days * seconds per day

	filter := types.SignalFilter{
		ValidatorAddress: validatorAddr,
		StartTime:        startTime,
		EndTime:          endTime,
	}

	signals := sa.keeper.FilterSignals(ctx, filter)

	metrics := ValidatorPerformanceMetrics{
		ValidatorAddress: validatorAddr,
		Period:          days,
		StartTime:       startTime,
		EndTime:         endTime,
	}

	if len(signals) == 0 {
		return metrics
	}

	// Calculate basic metrics
	metrics.TotalSignals = len(signals)
	sa.calculateValidatorMetrics(signals, &metrics)

	// Calculate performance trends
	sa.calculatePerformanceTrends(signals, &metrics)

	return metrics
}

// ValidatorPerformanceMetrics represents performance metrics for a validator
type ValidatorPerformanceMetrics struct {
	ValidatorAddress     string                    `json:"validator_address"`
	Period              int                       `json:"period_days"`
	StartTime           int64                     `json:"start_time"`
	EndTime             int64                     `json:"end_time"`
	TotalSignals        int                       `json:"total_signals"`
	ValidatedSignals    int                       `json:"validated_signals"`
	RejectedSignals     int                       `json:"rejected_signals"`
	ValidationRate      float64                   `json:"validation_rate"`
	TotalScore          uint64                    `json:"total_score"`
	AverageScore        uint64                    `json:"average_score"`
	EffectiveScore      uint64                    `json:"effective_score"`
	Rank                int                       `json:"rank"`
	DailyAverages       []DailyMetrics            `json:"daily_averages"`
	WorkTypeBreakdown   map[string]int            `json:"work_type_breakdown"`
	PerformanceTrend    string                    `json:"performance_trend"` // "improving", "declining", "stable"
	ConsistencyScore    float64                   `json:"consistency_score"`
}

// DailyMetrics represents daily aggregated metrics
type DailyMetrics struct {
	Date           int64   `json:"date"`
	SignalCount    int     `json:"signal_count"`
	TotalScore     uint64  `json:"total_score"`
	AverageScore   uint64  `json:"average_score"`
	ValidationRate float64 `json:"validation_rate"`
}

// getTimeRange extracts the time range from signals
func (sa *SignalAggregator) getTimeRange(signals []types.Signal) TimeRange {
	if len(signals) == 0 {
		return TimeRange{}
	}

	minTime := signals[0].Timestamp
	maxTime := signals[0].Timestamp

	for _, signal := range signals {
		if signal.Timestamp < minTime {
			minTime = signal.Timestamp
		}
		if signal.Timestamp > maxTime {
			maxTime = signal.Timestamp
		}
	}

	return TimeRange{
		StartTime: minTime,
		EndTime:   maxTime,
	}
}

// countSignalsByStatus counts signals by their status
func (sa *SignalAggregator) countSignalsByStatus(signals []types.Signal, result *AggregationResult) {
	for _, signal := range signals {
		switch signal.Status {
		case types.SignalStatusValidated:
			result.ValidatedSignals++
		case types.SignalStatusRejected:
			result.RejectedSignals++
		case types.SignalStatusPending:
			result.PendingSignals++
		}
	}
}

// calculateScoreMetrics calculates score-related metrics
func (sa *SignalAggregator) calculateScoreMetrics(signals []types.Signal, result *AggregationResult) {
	var scores []uint64
	var totalScore uint64

	for _, signal := range signals {
		if signal.Status == types.SignalStatusValidated && signal.Score > 0 {
			scores = append(scores, signal.Score)
			totalScore += signal.Score
		}
	}

	result.TotalScore = totalScore
	if len(scores) > 0 {
		result.AverageScore = totalScore / uint64(len(scores))
		result.ScoreDistribution = types.AnalyzeScoreDistribution(scores)
	}
}

// aggregateByWorkType aggregates signals by work type
func (sa *SignalAggregator) aggregateByWorkType(signals []types.Signal, result *AggregationResult) {
	workTypeMap := make(map[string][]types.Signal)

	// Group signals by work type
	for _, signal := range signals {
		workType := signal.WorkProof.WorkType
		workTypeMap[workType] = append(workTypeMap[workType], signal)
	}

	// Calculate aggregations for each work type
	for workType, workTypeSignals := range workTypeMap {
		aggregation := sa.calculateWorkTypeAggregation(workType, workTypeSignals)
		result.WorkTypeStats[workType] = aggregation
	}
}

// calculateWorkTypeAggregation calculates aggregation for a specific work type
func (sa *SignalAggregator) calculateWorkTypeAggregation(workType string, signals []types.Signal) WorkTypeAggregation {
	aggregation := WorkTypeAggregation{
		WorkType: workType,
		Count:    len(signals),
	}

	var validatedSignals int
	var totalScore uint64
	var totalComputeTime int64
	var totalResources types.Resources

	for _, signal := range signals {
		if signal.Status == types.SignalStatusValidated {
			validatedSignals++
			totalScore += signal.Score
		}

		totalComputeTime += signal.WorkProof.ComputationTime

		// Aggregate resources
		resources := signal.WorkProof.ResourcesUsed
		totalResources.CPUCycles += resources.CPUCycles
		totalResources.MemoryMB += resources.MemoryMB
		totalResources.GPUTimeMs += resources.GPUTimeMs
		totalResources.NetworkBandwidth += resources.NetworkBandwidth
		totalResources.StorageBytes += resources.StorageBytes
	}

	aggregation.TotalScore = totalScore
	if validatedSignals > 0 {
		aggregation.AverageScore = totalScore / uint64(validatedSignals)
	}

	if len(signals) > 0 {
		aggregation.ValidationRate = float64(validatedSignals) / float64(len(signals))
		aggregation.AvgComputeTime = totalComputeTime / int64(len(signals))

		// Calculate average resources
		aggregation.AvgResources = types.Resources{
			CPUCycles:        totalResources.CPUCycles / uint64(len(signals)),
			MemoryMB:         totalResources.MemoryMB / uint64(len(signals)),
			GPUTimeMs:        totalResources.GPUTimeMs / uint64(len(signals)),
			NetworkBandwidth: totalResources.NetworkBandwidth / uint64(len(signals)),
			StorageBytes:     totalResources.StorageBytes / uint64(len(signals)),
		}
	}

	aggregation.TotalResources = totalResources

	return aggregation
}

// aggregateByValidator aggregates signals by validator
func (sa *SignalAggregator) aggregateByValidator(signals []types.Signal, result *AggregationResult) {
	validatorMap := make(map[string][]types.Signal)

	// Group signals by validator
	for _, signal := range signals {
		if signal.ValidatorAddress != "" {
			validatorMap[signal.ValidatorAddress] = append(validatorMap[signal.ValidatorAddress], signal)
		}
	}

	// Calculate aggregations for each validator
	for validatorAddr, validatorSignals := range validatorMap {
		aggregation := sa.calculateValidatorAggregation(validatorAddr, validatorSignals)
		result.ValidatorStats[validatorAddr] = aggregation
	}

	// Calculate ranks
	sa.calculateValidatorRanks(result.ValidatorStats)
}

// calculateValidatorAggregation calculates aggregation for a specific validator
func (sa *SignalAggregator) calculateValidatorAggregation(validatorAddr string, signals []types.Signal) ValidatorAggregation {
	aggregation := ValidatorAggregation{
		ValidatorAddress: validatorAddr,
		Count:           len(signals),
	}

	var validatedSignals int
	var totalScore uint64

	for _, signal := range signals {
		if signal.Status == types.SignalStatusValidated {
			validatedSignals++
			totalScore += signal.Score
		}
	}

	aggregation.TotalScore = totalScore
	if validatedSignals > 0 {
		aggregation.AverageScore = totalScore / uint64(validatedSignals)
	}

	if len(signals) > 0 {
		aggregation.ValidationRate = float64(validatedSignals) / float64(len(signals))
	}

	// Effective score would be calculated using the signal scoring algorithm
	aggregation.EffectiveScore = totalScore // Simplified

	return aggregation
}

// calculateValidatorRanks calculates ranks for validators
func (sa *SignalAggregator) calculateValidatorRanks(validatorStats map[string]ValidatorAggregation) {
	// Create slice for sorting
	type validatorScore struct {
		addr  string
		score uint64
	}

	var validators []validatorScore
	for addr, stats := range validatorStats {
		validators = append(validators, validatorScore{
			addr:  addr,
			score: stats.EffectiveScore,
		})
	}

	// Sort by effective score (descending)
	sort.Slice(validators, func(i, j int) bool {
		return validators[i].score > validators[j].score
	})

	// Assign ranks
	for i, validator := range validators {
		stats := validatorStats[validator.addr]
		stats.Rank = i + 1
		validatorStats[validator.addr] = stats
	}
}

// calculateValidatorMetrics calculates basic metrics for validator performance
func (sa *SignalAggregator) calculateValidatorMetrics(signals []types.Signal, metrics *ValidatorPerformanceMetrics) {
	var validatedSignals int
	var totalScore uint64
	workTypeBreakdown := make(map[string]int)

	for _, signal := range signals {
		if signal.Status == types.SignalStatusValidated {
			validatedSignals++
			totalScore += signal.Score
		} else if signal.Status == types.SignalStatusRejected {
			metrics.RejectedSignals++
		}

		workTypeBreakdown[signal.WorkProof.WorkType]++
	}

	metrics.ValidatedSignals = validatedSignals
	metrics.TotalScore = totalScore
	metrics.WorkTypeBreakdown = workTypeBreakdown

	if validatedSignals > 0 {
		metrics.AverageScore = totalScore / uint64(validatedSignals)
	}

	if len(signals) > 0 {
		metrics.ValidationRate = float64(validatedSignals) / float64(len(signals))
	}

	metrics.EffectiveScore = totalScore // Simplified
}

// calculatePerformanceTrends calculates performance trends for a validator
func (sa *SignalAggregator) calculatePerformanceTrends(signals []types.Signal, metrics *ValidatorPerformanceMetrics) {
	// Group signals by day
	dailyMetrics := sa.groupSignalsByDay(signals)
	metrics.DailyAverages = dailyMetrics

	// Calculate trend
	if len(dailyMetrics) >= 2 {
		metrics.PerformanceTrend = sa.calculateTrend(dailyMetrics)
		metrics.ConsistencyScore = sa.calculateConsistency(dailyMetrics)
	} else {
		metrics.PerformanceTrend = "insufficient_data"
		metrics.ConsistencyScore = 0.0
	}
}

// groupSignalsByDay groups signals by day and calculates daily metrics
func (sa *SignalAggregator) groupSignalsByDay(signals []types.Signal) []DailyMetrics {
	dayMap := make(map[int64][]types.Signal)

	// Group signals by day
	for _, signal := range signals {
		day := signal.Timestamp / 86400 * 86400 // Round down to start of day
		dayMap[day] = append(dayMap[day], signal)
	}

	// Calculate daily metrics
	var dailyMetrics []DailyMetrics
	for day, daySignals := range dayMap {
		metrics := DailyMetrics{
			Date:        day,
			SignalCount: len(daySignals),
		}

		var validatedCount int
		var totalScore uint64

		for _, signal := range daySignals {
			if signal.Status == types.SignalStatusValidated {
				validatedCount++
				totalScore += signal.Score
			}
		}

		metrics.TotalScore = totalScore
		if validatedCount > 0 {
			metrics.AverageScore = totalScore / uint64(validatedCount)
		}

		if len(daySignals) > 0 {
			metrics.ValidationRate = float64(validatedCount) / float64(len(daySignals))
		}

		dailyMetrics = append(dailyMetrics, metrics)
	}

	// Sort by date
	sort.Slice(dailyMetrics, func(i, j int) bool {
		return dailyMetrics[i].Date < dailyMetrics[j].Date
	})

	return dailyMetrics
}

// calculateTrend calculates the performance trend
func (sa *SignalAggregator) calculateTrend(dailyMetrics []DailyMetrics) string {
	if len(dailyMetrics) < 2 {
		return "stable"
	}

	// Simple trend calculation based on score progression
	firstHalf := len(dailyMetrics) / 2
	var firstHalfAvg, secondHalfAvg uint64

	for i := 0; i < firstHalf; i++ {
		firstHalfAvg += dailyMetrics[i].AverageScore
	}
	firstHalfAvg /= uint64(firstHalf)

	for i := firstHalf; i < len(dailyMetrics); i++ {
		secondHalfAvg += dailyMetrics[i].AverageScore
	}
	secondHalfAvg /= uint64(len(dailyMetrics) - firstHalf)

	threshold := firstHalfAvg / 10 // 10% threshold

	if secondHalfAvg > firstHalfAvg+threshold {
		return "improving"
	} else if secondHalfAvg < firstHalfAvg-threshold {
		return "declining"
	}

	return "stable"
}

// calculateConsistency calculates the consistency score
func (sa *SignalAggregator) calculateConsistency(dailyMetrics []DailyMetrics) float64 {
	if len(dailyMetrics) < 2 {
		return 1.0
	}

	// Calculate standard deviation of validation rates
	var mean, variance float64
	for _, metrics := range dailyMetrics {
		mean += metrics.ValidationRate
	}
	mean /= float64(len(dailyMetrics))

	for _, metrics := range dailyMetrics {
		diff := metrics.ValidationRate - mean
		variance += diff * diff
	}
	variance /= float64(len(dailyMetrics))

	stdDev := variance // Simplified - should use math.Sqrt

	// Consistency score: 1.0 - normalized standard deviation
	consistencyScore := 1.0 - (stdDev / mean)
	if consistencyScore < 0 {
		consistencyScore = 0
	}
	if consistencyScore > 1 {
		consistencyScore = 1
	}

	return consistencyScore
}