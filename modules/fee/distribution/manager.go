package distribution

import (
	"fmt"
	"sync"
	"time"

	keepertypes "github.com/b2network/pulsar/keeper/types"
	"github.com/b2network/pulsar/modules/fee/types"
)

// DistributionManager manages fee distribution operations
type DistributionManager struct {
	engine     *DistributionEngine
	strategies map[string]DistributionStrategy
	config     ManagerConfig
	mu         sync.RWMutex

	// Statistics tracking
	totalDistributed map[string]int64
	distributionHistory []DistributionRecord
	lastDistribution time.Time
}

// ManagerConfig configures the distribution manager
type ManagerConfig struct {
	DefaultStrategy      string                 `json:"default_strategy"`
	AutoDistribution     bool                   `json:"auto_distribution"`
	DistributionInterval time.Duration          `json:"distribution_interval"`
	MinDistributionAmount int64                 `json:"min_distribution_amount"`
	MaxRetries           int                    `json:"max_retries"`
	RetryDelay           time.Duration          `json:"retry_delay"`
	EnableMetrics        bool                   `json:"enable_metrics"`
	StrategyConfig       map[string]interface{} `json:"strategy_config"`
}

// DistributionRecord represents a historical distribution record
type DistributionRecord struct {
	ID               string                     `json:"id"`
	Timestamp        time.Time                  `json:"timestamp"`
	Strategy         string                     `json:"strategy"`
	TotalFees        keepertypes.Coins          `json:"total_fees"`
	Result           DistributionResult         `json:"result"`
	ExecutionTime    time.Duration              `json:"execution_time"`
	Success          bool                       `json:"success"`
	ErrorMessage     string                     `json:"error_message,omitempty"`
}

// NewDistributionManager creates a new distribution manager
func NewDistributionManager(
	engine *DistributionEngine,
	config ManagerConfig,
) *DistributionManager {
	manager := &DistributionManager{
		engine:              engine,
		strategies:          make(map[string]DistributionStrategy),
		config:              config,
		totalDistributed:    make(map[string]int64),
		distributionHistory: make([]DistributionRecord, 0),
	}

	// Register default strategies
	manager.registerDefaultStrategies()

	return manager
}

// registerDefaultStrategies registers the built-in distribution strategies
func (dm *DistributionManager) registerDefaultStrategies() {
	// Default strategy
	dm.strategies["default"] = NewDefaultDistributionStrategy(dm.engine)

	// Weighted strategy (if weights are configured)
	if weights, ok := dm.config.StrategyConfig["weights"].(map[string]float64); ok {
		dm.strategies["weighted"] = NewWeightedDistributionStrategy(dm.engine, weights)
	}

	// Threshold strategy (if thresholds are configured)
	if thresholds, ok := dm.config.StrategyConfig["thresholds"].(map[string]int64); ok {
		dm.strategies["threshold"] = NewThresholdDistributionStrategy(dm.engine, thresholds)
	}

	// Time-based strategy
	if dm.config.DistributionInterval > 0 {
		dm.strategies["time_based"] = NewTimeBasedDistributionStrategy(dm.engine, dm.config.DistributionInterval)
	}

	// Performance-based strategy (if metrics are configured)
	if metrics, ok := dm.config.StrategyConfig["performance_metrics"].(map[string]float64); ok {
		dm.strategies["proportional"] = NewProportionalDistributionStrategy(dm.engine, metrics)
	}
}

// RegisterStrategy registers a custom distribution strategy
func (dm *DistributionManager) RegisterStrategy(name string, strategy DistributionStrategy) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if _, exists := dm.strategies[name]; exists {
		return fmt.Errorf("strategy %s already exists", name)
	}

	dm.strategies[name] = strategy
	return nil
}

// UnregisterStrategy removes a distribution strategy
func (dm *DistributionManager) UnregisterStrategy(name string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if name == dm.config.DefaultStrategy {
		return fmt.Errorf("cannot unregister default strategy")
	}

	if _, exists := dm.strategies[name]; !exists {
		return fmt.Errorf("strategy %s does not exist", name)
	}

	delete(dm.strategies, name)
	return nil
}

// GetStrategy returns a distribution strategy by name
func (dm *DistributionManager) GetStrategy(name string) (DistributionStrategy, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	strategy, exists := dm.strategies[name]
	if !exists {
		return nil, fmt.Errorf("strategy %s not found", name)
	}

	return strategy, nil
}

// ListStrategies returns all available strategy names
func (dm *DistributionManager) ListStrategies() []string {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	strategies := make([]string, 0, len(dm.strategies))
	for name := range dm.strategies {
		strategies = append(strategies, name)
	}

	return strategies
}

// DistributeFees distributes fees using the specified strategy
func (dm *DistributionManager) DistributeFees(
	ctx keepertypes.Context,
	fees keepertypes.Coins,
	strategyName string,
) (DistributionResult, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	// Validate minimum distribution amount
	if dm.shouldSkipDistribution(fees) {
		return DistributionResult{}, fmt.Errorf("fees below minimum distribution threshold")
	}

	// Get strategy
	if strategyName == "" {
		strategyName = dm.config.DefaultStrategy
	}

	strategy, exists := dm.strategies[strategyName]
	if !exists {
		return DistributionResult{}, fmt.Errorf("strategy %s not found", strategyName)
	}

	// Execute distribution with retries
	record := DistributionRecord{
		ID:        fmt.Sprintf("dist_%d", time.Now().UnixNano()),
		Timestamp: time.Now(),
		Strategy:  strategyName,
		TotalFees: fees,
	}

	startTime := time.Now()
	result, err := dm.executeWithRetries(ctx, strategy, fees)
	record.ExecutionTime = time.Since(startTime)

	if err != nil {
		record.Success = false
		record.ErrorMessage = err.Error()
		dm.distributionHistory = append(dm.distributionHistory, record)
		return result, err
	}

	record.Success = true
	record.Result = result
	dm.distributionHistory = append(dm.distributionHistory, record)

	// Update statistics
	dm.updateStatistics(result)
	dm.lastDistribution = time.Now()

	// Emit distribution event
	dm.emitDistributionEvent(ctx, record)

	return result, nil
}

// DistributeFeesAuto distributes fees using automatic strategy selection
func (dm *DistributionManager) DistributeFeesAuto(
	ctx keepertypes.Context,
	fees keepertypes.Coins,
) (DistributionResult, error) {
	// Select optimal strategy based on current conditions
	strategyName := dm.selectOptimalStrategy(ctx, fees)
	return dm.DistributeFees(ctx, fees, strategyName)
}

// shouldSkipDistribution checks if distribution should be skipped
func (dm *DistributionManager) shouldSkipDistribution(fees keepertypes.Coins) bool {
	if dm.config.MinDistributionAmount <= 0 {
		return false
	}

	for _, coin := range fees {
		if coin.Amount >= dm.config.MinDistributionAmount {
			return false
		}
	}

	return true
}

// executeWithRetries executes distribution with retry logic
func (dm *DistributionManager) executeWithRetries(
	ctx keepertypes.Context,
	strategy DistributionStrategy,
	fees keepertypes.Coins,
) (DistributionResult, error) {
	var lastErr error

	for attempt := 0; attempt <= dm.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			time.Sleep(dm.config.RetryDelay)
		}

		result, err := dm.engine.DistributeFees(ctx, fees, strategy)
		if err == nil {
			return result, nil
		}

		lastErr = err
		ctx.Logger().Error("Distribution attempt failed",
			"attempt", attempt+1,
			"error", err,
			"retries_remaining", dm.config.MaxRetries-attempt)
	}

	return DistributionResult{}, fmt.Errorf("distribution failed after %d attempts: %w", dm.config.MaxRetries+1, lastErr)
}

// selectOptimalStrategy selects the best strategy for current conditions
func (dm *DistributionManager) selectOptimalStrategy(
	ctx keepertypes.Context,
	fees keepertypes.Coins,
) string {
	// Simple strategy selection logic
	// In practice, this could be much more sophisticated

	// Check if time-based strategy is available and it's time to distribute
	if _, exists := dm.strategies["time_based"]; exists {
		if time.Since(dm.lastDistribution) >= dm.config.DistributionInterval {
			return "time_based"
		}
	}

	// Check if threshold strategy is available
	if _, exists := dm.strategies["threshold"]; exists {
		// Check if any threshold is met (simplified check)
		for _, coin := range fees {
			if coin.Amount >= dm.config.MinDistributionAmount*10 {
				return "threshold"
			}
		}
	}

	// Check if weighted strategy should be used for large amounts
	if _, exists := dm.strategies["weighted"]; exists {
		totalValue := int64(0)
		for _, coin := range fees {
			totalValue += coin.Amount
		}
		if totalValue >= dm.config.MinDistributionAmount*100 {
			return "weighted"
		}
	}

	// Default strategy
	return dm.config.DefaultStrategy
}

// updateStatistics updates distribution statistics
func (dm *DistributionManager) updateStatistics(result DistributionResult) {
	if !dm.config.EnableMetrics {
		return
	}

	for _, coin := range result.TotalDistributed {
		dm.totalDistributed[coin.Denom] += coin.Amount
	}

	// Limit history size to prevent memory bloat
	const maxHistorySize = 1000
	if len(dm.distributionHistory) > maxHistorySize {
		// Keep only the most recent records
		copy(dm.distributionHistory, dm.distributionHistory[len(dm.distributionHistory)-maxHistorySize:])
		dm.distributionHistory = dm.distributionHistory[:maxHistorySize]
	}
}

// emitDistributionEvent emits a distribution event
func (dm *DistributionManager) emitDistributionEvent(ctx keepertypes.Context, record DistributionRecord) {
	feeStr := ""
	for i, coin := range record.TotalFees {
		if i > 0 {
			feeStr += ","
		}
		feeStr += fmt.Sprintf("%d%s", coin.Amount, coin.Denom)
	}

	event := keepertypes.Event{
		Type: types.EventTypeFeeDistribution,
		Attributes: []keepertypes.Attribute{
			{Key: "distribution_id", Value: record.ID},
			{Key: "strategy", Value: record.Strategy},
			{Key: "total_fees", Value: feeStr},
			{Key: "success", Value: fmt.Sprintf("%t", record.Success)},
			{Key: "execution_time_ms", Value: fmt.Sprintf("%d", record.ExecutionTime.Milliseconds())},
		},
	}

	ctx.EventManager().EmitEvent(event)
}

// GetDistributionStats returns distribution statistics
func (dm *DistributionManager) GetDistributionStats() DistributionManagerStats {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	stats := DistributionManagerStats{
		TotalDistributed:     make(map[string]int64),
		DistributionCount:    int64(len(dm.distributionHistory)),
		LastDistribution:     dm.lastDistribution,
		AvailableStrategies:  make([]string, 0, len(dm.strategies)),
		SuccessRate:          0.0,
	}

	// Copy total distributed
	for denom, amount := range dm.totalDistributed {
		stats.TotalDistributed[denom] = amount
	}

	// List strategies
	for name := range dm.strategies {
		stats.AvailableStrategies = append(stats.AvailableStrategies, name)
	}

	// Calculate success rate
	if len(dm.distributionHistory) > 0 {
		successCount := 0
		for _, record := range dm.distributionHistory {
			if record.Success {
				successCount++
			}
		}
		stats.SuccessRate = float64(successCount) / float64(len(dm.distributionHistory))
	}

	return stats
}

// GetDistributionHistory returns recent distribution history
func (dm *DistributionManager) GetDistributionHistory(limit int) []DistributionRecord {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	if limit <= 0 || limit > len(dm.distributionHistory) {
		limit = len(dm.distributionHistory)
	}

	// Return most recent records
	start := len(dm.distributionHistory) - limit
	history := make([]DistributionRecord, limit)
	copy(history, dm.distributionHistory[start:])

	return history
}

// UpdateConfig updates the manager configuration
func (dm *DistributionManager) UpdateConfig(config ManagerConfig) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	oldConfig := dm.config
	dm.config = config

	// Re-register strategies with new config
	dm.strategies = make(map[string]DistributionStrategy)
	dm.registerDefaultStrategies()

	// Validate that default strategy exists
	if _, exists := dm.strategies[config.DefaultStrategy]; !exists {
		// Restore old config
		dm.config = oldConfig
		dm.strategies = make(map[string]DistributionStrategy)
		dm.registerDefaultStrategies()
		return fmt.Errorf("default strategy %s not available", config.DefaultStrategy)
	}

	return nil
}

// DistributionManagerStats represents manager statistics
type DistributionManagerStats struct {
	TotalDistributed     map[string]int64 `json:"total_distributed"`
	DistributionCount    int64            `json:"distribution_count"`
	LastDistribution     time.Time        `json:"last_distribution"`
	AvailableStrategies  []string         `json:"available_strategies"`
	SuccessRate          float64          `json:"success_rate"`
	AverageExecutionTime time.Duration    `json:"average_execution_time"`
}

// DefaultManagerConfig returns default manager configuration
func DefaultManagerConfig() ManagerConfig {
	return ManagerConfig{
		DefaultStrategy:       "default",
		AutoDistribution:      true,
		DistributionInterval:  time.Hour * 24, // Daily distribution
		MinDistributionAmount: 1000,           // Minimum 1000 units
		MaxRetries:           3,
		RetryDelay:           time.Second * 5,
		EnableMetrics:        true,
		StrategyConfig:       make(map[string]interface{}),
	}
}