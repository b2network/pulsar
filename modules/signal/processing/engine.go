package processing

import (
	"fmt"
	"sync"
	"time"

	keepertypes "github.com/b2network/pulsar/keeper/types"
	"github.com/b2network/pulsar/modules/signal/types"
)

// ProcessingEngine handles signal processing operations
type ProcessingEngine struct {
	keeper         SignalKeeper
	validator      *SignalValidator
	aggregator     *SignalAggregator
	strategies     map[string]ProcessingStrategy
	config         ProcessingConfig
	mu             sync.RWMutex
	running        bool
	stopChan       chan struct{}
	processingPool chan struct{} // Semaphore for concurrent processing
}

// SignalKeeper interface for engine dependencies
type SignalKeeper interface {
	GetSignal(ctx keepertypes.Context, signalID string) (types.Signal, bool)
	StoreSignal(ctx keepertypes.Context, signal types.Signal) error
	ProcessSignal(ctx keepertypes.Context, signalID string) error
	GetSignalsByStatus(ctx keepertypes.Context, status types.SignalStatus) []types.Signal
	GetParams(ctx keepertypes.Context) types.Params
	GetWorkType(ctx keepertypes.Context, workTypeID string) (types.AIWorkType, bool)
	UpdateValidatorStats(ctx keepertypes.Context, validatorAddr string, signal types.Signal, score uint64) error
	Logger(ctx keepertypes.Context) keepertypes.Logger
}

// ProcessingConfig defines configuration for the processing engine
type ProcessingConfig struct {
	MaxConcurrentProcessing int           `json:"max_concurrent_processing"`
	ProcessingTimeout       time.Duration `json:"processing_timeout"`
	BatchSize              int           `json:"batch_size"`
	ProcessingInterval     time.Duration `json:"processing_interval"`
	RetryAttempts          int           `json:"retry_attempts"`
	RetryDelay             time.Duration `json:"retry_delay"`
	EnableAsyncProcessing  bool          `json:"enable_async_processing"`
	PriorityWeights        map[string]float64 `json:"priority_weights"`
}

// ProcessingStrategy defines an interface for different processing strategies
type ProcessingStrategy interface {
	ProcessSignal(ctx keepertypes.Context, signal types.Signal) (ProcessingResult, error)
	GetPriority(signal types.Signal) float64
	CanProcess(signal types.Signal) bool
}

// ProcessingResult represents the result of signal processing
type ProcessingResult struct {
	SignalID       string        `json:"signal_id"`
	ProcessedAt    time.Time     `json:"processed_at"`
	ProcessingTime time.Duration `json:"processing_time"`
	Score          uint64        `json:"score"`
	Status         types.SignalStatus `json:"status"`
	Errors         []string      `json:"errors,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// NewProcessingEngine creates a new signal processing engine
func NewProcessingEngine(keeper SignalKeeper, config ProcessingConfig) *ProcessingEngine {
	engine := &ProcessingEngine{
		keeper:         keeper,
		strategies:     make(map[string]ProcessingStrategy),
		config:         config,
		stopChan:       make(chan struct{}),
		processingPool: make(chan struct{}, config.MaxConcurrentProcessing),
	}

	// Initialize components
	engine.validator = NewSignalValidator(keeper)
	engine.aggregator = NewSignalAggregator(keeper)

	// Register default strategies
	engine.registerDefaultStrategies()

	return engine
}

// DefaultProcessingConfig returns default processing configuration
func DefaultProcessingConfig() ProcessingConfig {
	return ProcessingConfig{
		MaxConcurrentProcessing: 10,
		ProcessingTimeout:       time.Minute * 5,
		BatchSize:              50,
		ProcessingInterval:     time.Second * 10,
		RetryAttempts:          3,
		RetryDelay:             time.Second * 5,
		EnableAsyncProcessing:  true,
		PriorityWeights: map[string]float64{
			"llm_inference":      1.0,
			"image_generation":   1.2,
			"model_training":     1.5,
			"data_analysis":      0.8,
			"proof_verification": 0.9,
			"optimization":       1.3,
		},
	}
}

// Start starts the processing engine
func (pe *ProcessingEngine) Start(ctx keepertypes.Context) error {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	if pe.running {
		return fmt.Errorf("processing engine is already running")
	}

	pe.running = true

	if pe.config.EnableAsyncProcessing {
		go pe.processLoop(ctx)
	}

	pe.keeper.Logger(ctx).Info("signal processing engine started",
		"max_concurrent", pe.config.MaxConcurrentProcessing,
		"async_enabled", pe.config.EnableAsyncProcessing)

	return nil
}

// Stop stops the processing engine
func (pe *ProcessingEngine) Stop(ctx keepertypes.Context) error {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	if !pe.running {
		return fmt.Errorf("processing engine is not running")
	}

	close(pe.stopChan)
	pe.running = false

	pe.keeper.Logger(ctx).Info("signal processing engine stopped")

	return nil
}

// ProcessSignal processes a single signal
func (pe *ProcessingEngine) ProcessSignal(ctx keepertypes.Context, signalID string) (ProcessingResult, error) {
	startTime := time.Now()

	// Get signal
	signal, found := pe.keeper.GetSignal(ctx, signalID)
	if !found {
		return ProcessingResult{}, types.ErrSignalNotFound
	}

	// Check if signal can be processed
	if signal.Status != types.SignalStatusPending {
		return ProcessingResult{}, fmt.Errorf("signal %s is not in pending status", signalID)
	}

	// Select processing strategy
	strategy, err := pe.selectStrategy(signal)
	if err != nil {
		return ProcessingResult{}, fmt.Errorf("failed to select processing strategy: %w", err)
	}

	// Process with retry logic
	result, err := pe.processWithRetry(ctx, signal, strategy)
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
		result.Status = types.SignalStatusRejected
	}

	result.ProcessingTime = time.Since(startTime)
	result.ProcessedAt = time.Now()

	pe.keeper.Logger(ctx).Debug("signal processed",
		"signal_id", signalID,
		"status", result.Status.String(),
		"score", result.Score,
		"processing_time", result.ProcessingTime)

	return result, nil
}

// ProcessBatch processes multiple signals in batch
func (pe *ProcessingEngine) ProcessBatch(ctx keepertypes.Context, signalIDs []string) ([]ProcessingResult, error) {
	if len(signalIDs) > pe.config.BatchSize {
		return nil, fmt.Errorf("batch size %d exceeds maximum %d", len(signalIDs), pe.config.BatchSize)
	}

	results := make([]ProcessingResult, len(signalIDs))
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Process signals concurrently
	for i, signalID := range signalIDs {
		wg.Add(1)
		go func(index int, id string) {
			defer wg.Done()

			// Acquire processing slot
			pe.processingPool <- struct{}{}
			defer func() { <-pe.processingPool }()

			result, err := pe.ProcessSignal(ctx, id)
			if err != nil {
				result = ProcessingResult{
					SignalID:    id,
					ProcessedAt: time.Now(),
					Status:      types.SignalStatusRejected,
					Errors:      []string{err.Error()},
				}
			}

			mu.Lock()
			results[index] = result
			mu.Unlock()
		}(i, signalID)
	}

	wg.Wait()

	return results, nil
}

// processLoop runs the continuous processing loop
func (pe *ProcessingEngine) processLoop(ctx keepertypes.Context) {
	ticker := time.NewTicker(pe.config.ProcessingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-pe.stopChan:
			return
		case <-ticker.C:
			pe.processPendingSignals(ctx)
		}
	}
}

// processPendingSignals processes all pending signals
func (pe *ProcessingEngine) processPendingSignals(ctx keepertypes.Context) {
	pendingSignals := pe.keeper.GetSignalsByStatus(ctx, types.SignalStatusPending)

	if len(pendingSignals) == 0 {
		return
	}

	// Sort signals by priority
	prioritizedSignals := pe.prioritizeSignals(pendingSignals)

	// Process in batches
	for i := 0; i < len(prioritizedSignals); i += pe.config.BatchSize {
		end := i + pe.config.BatchSize
		if end > len(prioritizedSignals) {
			end = len(prioritizedSignals)
		}

		batch := prioritizedSignals[i:end]
		signalIDs := make([]string, len(batch))
		for j, signal := range batch {
			signalIDs[j] = signal.ID
		}

		results, err := pe.ProcessBatch(ctx, signalIDs)
		if err != nil {
			pe.keeper.Logger(ctx).Error("failed to process signal batch", "error", err)
			continue
		}

		// Log batch results
		successCount := 0
		for _, result := range results {
			if result.Status == types.SignalStatusValidated {
				successCount++
			}
		}

		pe.keeper.Logger(ctx).Info("processed signal batch",
			"batch_size", len(batch),
			"success_count", successCount,
			"failure_count", len(batch)-successCount)
	}
}

// prioritizeSignals sorts signals by processing priority
func (pe *ProcessingEngine) prioritizeSignals(signals []types.Signal) []types.Signal {
	// Simple priority based on work type and timestamp
	// In practice, this could be much more sophisticated

	signalsCopy := make([]types.Signal, len(signals))
	copy(signalsCopy, signals)

	// Sort by priority (higher first)
	for i := 0; i < len(signalsCopy)-1; i++ {
		for j := i + 1; j < len(signalsCopy); j++ {
			priority1 := pe.getSignalPriority(signalsCopy[i])
			priority2 := pe.getSignalPriority(signalsCopy[j])

			if priority2 > priority1 {
				signalsCopy[i], signalsCopy[j] = signalsCopy[j], signalsCopy[i]
			}
		}
	}

	return signalsCopy
}

// getSignalPriority calculates the processing priority for a signal
func (pe *ProcessingEngine) getSignalPriority(signal types.Signal) float64 {
	basePriority := 1.0

	// Apply work type weight
	if weight, exists := pe.config.PriorityWeights[signal.WorkProof.WorkType]; exists {
		basePriority *= weight
	}

	// Apply time factor (older signals get higher priority)
	age := time.Now().Unix() - signal.Timestamp
	timeFactor := 1.0 + float64(age)/3600.0 // +1 priority per hour

	return basePriority * timeFactor
}

// selectStrategy selects the appropriate processing strategy for a signal
func (pe *ProcessingEngine) selectStrategy(signal types.Signal) (ProcessingStrategy, error) {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	// Try to find a specific strategy for the work type
	if strategy, exists := pe.strategies[signal.WorkProof.WorkType]; exists {
		if strategy.CanProcess(signal) {
			return strategy, nil
		}
	}

	// Fall back to default strategy
	if defaultStrategy, exists := pe.strategies["default"]; exists {
		return defaultStrategy, nil
	}

	return nil, fmt.Errorf("no suitable processing strategy found for signal %s", signal.ID)
}

// processWithRetry processes a signal with retry logic
func (pe *ProcessingEngine) processWithRetry(ctx keepertypes.Context, signal types.Signal, strategy ProcessingStrategy) (ProcessingResult, error) {
	var lastErr error

	for attempt := 0; attempt <= pe.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(pe.config.RetryDelay)
		}

		result, err := strategy.ProcessSignal(ctx, signal)
		if err == nil {
			return result, nil
		}

		lastErr = err
		pe.keeper.Logger(ctx).Debug("signal processing attempt failed",
			"signal_id", signal.ID,
			"attempt", attempt+1,
			"error", err)
	}

	return ProcessingResult{
		SignalID: signal.ID,
		Status:   types.SignalStatusRejected,
	}, fmt.Errorf("processing failed after %d attempts: %w", pe.config.RetryAttempts+1, lastErr)
}

// RegisterStrategy registers a custom processing strategy
func (pe *ProcessingEngine) RegisterStrategy(name string, strategy ProcessingStrategy) error {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	if _, exists := pe.strategies[name]; exists {
		return fmt.Errorf("strategy %s already exists", name)
	}

	pe.strategies[name] = strategy
	return nil
}

// GetStrategies returns all registered strategies
func (pe *ProcessingEngine) GetStrategies() map[string]ProcessingStrategy {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	strategies := make(map[string]ProcessingStrategy)
	for name, strategy := range pe.strategies {
		strategies[name] = strategy
	}

	return strategies
}

// registerDefaultStrategies registers the default processing strategies
func (pe *ProcessingEngine) registerDefaultStrategies() {
	pe.strategies["default"] = NewDefaultProcessingStrategy(pe.keeper)
	pe.strategies["llm_inference"] = NewLLMInferenceStrategy(pe.keeper)
	pe.strategies["image_generation"] = NewImageGenerationStrategy(pe.keeper)
	pe.strategies["model_training"] = NewModelTrainingStrategy(pe.keeper)
	pe.strategies["data_analysis"] = NewDataAnalysisStrategy(pe.keeper)
	pe.strategies["proof_verification"] = NewProofVerificationStrategy(pe.keeper)
	pe.strategies["optimization"] = NewOptimizationStrategy(pe.keeper)
}

// GetProcessingStats returns processing statistics
func (pe *ProcessingEngine) GetProcessingStats() ProcessingStats {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	return ProcessingStats{
		IsRunning:            pe.running,
		MaxConcurrentSlots:   pe.config.MaxConcurrentProcessing,
		UsedConcurrentSlots:  len(pe.processingPool),
		RegisteredStrategies: len(pe.strategies),
	}
}

// ProcessingStats represents processing engine statistics
type ProcessingStats struct {
	IsRunning            bool `json:"is_running"`
	MaxConcurrentSlots   int  `json:"max_concurrent_slots"`
	UsedConcurrentSlots  int  `json:"used_concurrent_slots"`
	RegisteredStrategies int  `json:"registered_strategies"`
}