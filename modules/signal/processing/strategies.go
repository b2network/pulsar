package processing

import (
	"fmt"
	"time"

	keepertypes "github.com/b2network/pulsar/keeper/types"
	"github.com/b2network/pulsar/modules/signal/types"
)

// Default Processing Strategy

// DefaultProcessingStrategy implements the default signal processing strategy
type DefaultProcessingStrategy struct {
	keeper SignalKeeper
}

// NewDefaultProcessingStrategy creates a new default processing strategy
func NewDefaultProcessingStrategy(keeper SignalKeeper) *DefaultProcessingStrategy {
	return &DefaultProcessingStrategy{
		keeper: keeper,
	}
}

// ProcessSignal processes a signal using the default strategy
func (dps *DefaultProcessingStrategy) ProcessSignal(ctx keepertypes.Context, signal types.Signal) (ProcessingResult, error) {
	// Basic processing: validate and score
	if err := dps.keeper.ProcessSignal(ctx, signal.ID); err != nil {
		return ProcessingResult{
			SignalID: signal.ID,
			Status:   types.SignalStatusRejected,
		}, err
	}

	// Get updated signal with score
	updatedSignal, found := dps.keeper.GetSignal(ctx, signal.ID)
	if !found {
		return ProcessingResult{
			SignalID: signal.ID,
			Status:   types.SignalStatusRejected,
		}, types.ErrSignalNotFound
	}

	return ProcessingResult{
		SignalID: signal.ID,
		Score:    updatedSignal.Score,
		Status:   updatedSignal.Status,
	}, nil
}

// GetPriority returns the processing priority for a signal
func (dps *DefaultProcessingStrategy) GetPriority(signal types.Signal) float64 {
	return 1.0 // Default priority
}

// CanProcess checks if this strategy can process the signal
func (dps *DefaultProcessingStrategy) CanProcess(signal types.Signal) bool {
	return true // Default strategy can process any signal
}

// LLM Inference Strategy

// LLMInferenceStrategy implements processing strategy for LLM inference signals
type LLMInferenceStrategy struct {
	keeper SignalKeeper
}

// NewLLMInferenceStrategy creates a new LLM inference processing strategy
func NewLLMInferenceStrategy(keeper SignalKeeper) *LLMInferenceStrategy {
	return &LLMInferenceStrategy{
		keeper: keeper,
	}
}

// ProcessSignal processes an LLM inference signal
func (lis *LLMInferenceStrategy) ProcessSignal(ctx keepertypes.Context, signal types.Signal) (ProcessingResult, error) {
	// Validate LLM-specific requirements
	if err := lis.validateLLMRequirements(signal); err != nil {
		return ProcessingResult{
			SignalID: signal.ID,
			Status:   types.SignalStatusRejected,
		}, err
	}

	// Process with enhanced scoring for LLM work
	if err := lis.keeper.ProcessSignal(ctx, signal.ID); err != nil {
		return ProcessingResult{
			SignalID: signal.ID,
			Status:   types.SignalStatusRejected,
		}, err
	}

	// Get updated signal
	updatedSignal, found := lis.keeper.GetSignal(ctx, signal.ID)
	if !found {
		return ProcessingResult{
			SignalID: signal.ID,
			Status:   types.SignalStatusRejected,
		}, types.ErrSignalNotFound
	}

	// Apply LLM-specific bonuses
	bonusScore := lis.calculateLLMBonus(signal)

	return ProcessingResult{
		SignalID: signal.ID,
		Score:    updatedSignal.Score + bonusScore,
		Status:   updatedSignal.Status,
		Metadata: map[string]interface{}{
			"llm_bonus_score": bonusScore,
		},
	}, nil
}

// validateLLMRequirements validates LLM-specific requirements
func (lis *LLMInferenceStrategy) validateLLMRequirements(signal types.Signal) error {
	if signal.WorkProof.ResourcesUsed.MemoryMB < 512 {
		return fmt.Errorf("insufficient memory for LLM inference: %d MB", signal.WorkProof.ResourcesUsed.MemoryMB)
	}
	if signal.WorkProof.ComputationTime < 100 {
		return fmt.Errorf("computation time too short for LLM inference: %d ms", signal.WorkProof.ComputationTime)
	}
	return nil
}

// calculateLLMBonus calculates bonus score for LLM inference
func (lis *LLMInferenceStrategy) calculateLLMBonus(signal types.Signal) uint64 {
	bonus := uint64(0)

	// Bonus for efficient memory usage
	if signal.WorkProof.ResourcesUsed.MemoryMB >= 1024 {
		bonus += 20
	}

	// Bonus for reasonable computation time
	if signal.WorkProof.ComputationTime >= 1000 && signal.WorkProof.ComputationTime <= 10000 {
		bonus += 15
	}

	return bonus
}

// GetPriority returns the processing priority for LLM inference signals
func (lis *LLMInferenceStrategy) GetPriority(signal types.Signal) float64 {
	return 1.2 // Higher priority for LLM inference
}

// CanProcess checks if this strategy can process the signal
func (lis *LLMInferenceStrategy) CanProcess(signal types.Signal) bool {
	return signal.WorkProof.WorkType == "llm_inference"
}

// Image Generation Strategy

// ImageGenerationStrategy implements processing strategy for image generation signals
type ImageGenerationStrategy struct {
	keeper SignalKeeper
}

// NewImageGenerationStrategy creates a new image generation processing strategy
func NewImageGenerationStrategy(keeper SignalKeeper) *ImageGenerationStrategy {
	return &ImageGenerationStrategy{
		keeper: keeper,
	}
}

// ProcessSignal processes an image generation signal
func (igs *ImageGenerationStrategy) ProcessSignal(ctx keepertypes.Context, signal types.Signal) (ProcessingResult, error) {
	// Validate image generation requirements
	if err := igs.validateImageGenRequirements(signal); err != nil {
		return ProcessingResult{
			SignalID: signal.ID,
			Status:   types.SignalStatusRejected,
		}, err
	}

	// Process signal
	if err := igs.keeper.ProcessSignal(ctx, signal.ID); err != nil {
		return ProcessingResult{
			SignalID: signal.ID,
			Status:   types.SignalStatusRejected,
		}, err
	}

	updatedSignal, found := igs.keeper.GetSignal(ctx, signal.ID)
	if !found {
		return ProcessingResult{
			SignalID: signal.ID,
			Status:   types.SignalStatusRejected,
		}, types.ErrSignalNotFound
	}

	// Apply image generation bonuses
	bonusScore := igs.calculateImageGenBonus(signal)

	return ProcessingResult{
		SignalID: signal.ID,
		Score:    updatedSignal.Score + bonusScore,
		Status:   updatedSignal.Status,
		Metadata: map[string]interface{}{
			"image_gen_bonus_score": bonusScore,
		},
	}, nil
}

// validateImageGenRequirements validates image generation requirements
func (igs *ImageGenerationStrategy) validateImageGenRequirements(signal types.Signal) error {
	if signal.WorkProof.ResourcesUsed.GPUTimeMs == 0 {
		return fmt.Errorf("GPU time required for image generation")
	}
	if signal.WorkProof.ResourcesUsed.MemoryMB < 1024 {
		return fmt.Errorf("insufficient memory for image generation: %d MB", signal.WorkProof.ResourcesUsed.MemoryMB)
	}
	return nil
}

// calculateImageGenBonus calculates bonus score for image generation
func (igs *ImageGenerationStrategy) calculateImageGenBonus(signal types.Signal) uint64 {
	bonus := uint64(0)

	// Bonus for GPU usage
	if signal.WorkProof.ResourcesUsed.GPUTimeMs > 0 {
		bonus += 30
	}

	// Bonus for high-quality generation (longer compute time)
	if signal.WorkProof.ComputationTime >= 5000 {
		bonus += 25
	}

	return bonus
}

// GetPriority returns the processing priority
func (igs *ImageGenerationStrategy) GetPriority(signal types.Signal) float64 {
	return 1.5 // High priority for creative work
}

// CanProcess checks if this strategy can process the signal
func (igs *ImageGenerationStrategy) CanProcess(signal types.Signal) bool {
	return signal.WorkProof.WorkType == "image_generation"
}

// Model Training Strategy

// ModelTrainingStrategy implements processing strategy for model training signals
type ModelTrainingStrategy struct {
	keeper SignalKeeper
}

// NewModelTrainingStrategy creates a new model training processing strategy
func NewModelTrainingStrategy(keeper SignalKeeper) *ModelTrainingStrategy {
	return &ModelTrainingStrategy{
		keeper: keeper,
	}
}

// ProcessSignal processes a model training signal
func (mts *ModelTrainingStrategy) ProcessSignal(ctx keepertypes.Context, signal types.Signal) (ProcessingResult, error) {
	// Validate training requirements
	if err := mts.validateTrainingRequirements(signal); err != nil {
		return ProcessingResult{
			SignalID: signal.ID,
			Status:   types.SignalStatusRejected,
		}, err
	}

	// Process signal
	if err := mts.keeper.ProcessSignal(ctx, signal.ID); err != nil {
		return ProcessingResult{
			SignalID: signal.ID,
			Status:   types.SignalStatusRejected,
		}, err
	}

	updatedSignal, found := mts.keeper.GetSignal(ctx, signal.ID)
	if !found {
		return ProcessingResult{
			SignalID: signal.ID,
			Status:   types.SignalStatusRejected,
		}, types.ErrSignalNotFound
	}

	// Apply training bonuses
	bonusScore := mts.calculateTrainingBonus(signal)

	return ProcessingResult{
		SignalID: signal.ID,
		Score:    updatedSignal.Score + bonusScore,
		Status:   updatedSignal.Status,
		Metadata: map[string]interface{}{
			"training_bonus_score": bonusScore,
		},
	}, nil
}

// validateTrainingRequirements validates model training requirements
func (mts *ModelTrainingStrategy) validateTrainingRequirements(signal types.Signal) error {
	if signal.WorkProof.ComputationTime < 10000 {
		return fmt.Errorf("computation time too short for model training: %d ms", signal.WorkProof.ComputationTime)
	}
	if signal.WorkProof.ResourcesUsed.MemoryMB < 2048 {
		return fmt.Errorf("insufficient memory for model training: %d MB", signal.WorkProof.ResourcesUsed.MemoryMB)
	}
	return nil
}

// calculateTrainingBonus calculates bonus score for model training
func (mts *ModelTrainingStrategy) calculateTrainingBonus(signal types.Signal) uint64 {
	bonus := uint64(0)

	// Bonus for long training time
	if signal.WorkProof.ComputationTime >= 60000 { // 1 minute
		bonus += 50
	}

	// Bonus for high memory usage (complex models)
	if signal.WorkProof.ResourcesUsed.MemoryMB >= 8192 {
		bonus += 40
	}

	return bonus
}

// GetPriority returns the processing priority
func (mts *ModelTrainingStrategy) GetPriority(signal types.Signal) float64 {
	return 2.0 // Highest priority for training work
}

// CanProcess checks if this strategy can process the signal
func (mts *ModelTrainingStrategy) CanProcess(signal types.Signal) bool {
	return signal.WorkProof.WorkType == "model_training"
}

// Data Analysis Strategy

// DataAnalysisStrategy implements processing strategy for data analysis signals
type DataAnalysisStrategy struct {
	keeper SignalKeeper
}

// NewDataAnalysisStrategy creates a new data analysis processing strategy
func NewDataAnalysisStrategy(keeper SignalKeeper) *DataAnalysisStrategy {
	return &DataAnalysisStrategy{
		keeper: keeper,
	}
}

// ProcessSignal processes a data analysis signal
func (das *DataAnalysisStrategy) ProcessSignal(ctx keepertypes.Context, signal types.Signal) (ProcessingResult, error) {
	// Process signal
	if err := das.keeper.ProcessSignal(ctx, signal.ID); err != nil {
		return ProcessingResult{
			SignalID: signal.ID,
			Status:   types.SignalStatusRejected,
		}, err
	}

	updatedSignal, found := das.keeper.GetSignal(ctx, signal.ID)
	if !found {
		return ProcessingResult{
			SignalID: signal.ID,
			Status:   types.SignalStatusRejected,
		}, types.ErrSignalNotFound
	}

	return ProcessingResult{
		SignalID: signal.ID,
		Score:    updatedSignal.Score,
		Status:   updatedSignal.Status,
	}, nil
}

// GetPriority returns the processing priority
func (das *DataAnalysisStrategy) GetPriority(signal types.Signal) float64 {
	return 0.8 // Lower priority for data analysis
}

// CanProcess checks if this strategy can process the signal
func (das *DataAnalysisStrategy) CanProcess(signal types.Signal) bool {
	return signal.WorkProof.WorkType == "data_analysis"
}

// Proof Verification Strategy

// ProofVerificationStrategy implements processing strategy for proof verification signals
type ProofVerificationStrategy struct {
	keeper SignalKeeper
}

// NewProofVerificationStrategy creates a new proof verification processing strategy
func NewProofVerificationStrategy(keeper SignalKeeper) *ProofVerificationStrategy {
	return &ProofVerificationStrategy{
		keeper: keeper,
	}
}

// ProcessSignal processes a proof verification signal
func (pvs *ProofVerificationStrategy) ProcessSignal(ctx keepertypes.Context, signal types.Signal) (ProcessingResult, error) {
	// Validate proof verification requirements
	if err := pvs.validateProofRequirements(signal); err != nil {
		return ProcessingResult{
			SignalID: signal.ID,
			Status:   types.SignalStatusRejected,
		}, err
	}

	// Process signal
	if err := pvs.keeper.ProcessSignal(ctx, signal.ID); err != nil {
		return ProcessingResult{
			SignalID: signal.ID,
			Status:   types.SignalStatusRejected,
		}, err
	}

	updatedSignal, found := pvs.keeper.GetSignal(ctx, signal.ID)
	if !found {
		return ProcessingResult{
			SignalID: signal.ID,
			Status:   types.SignalStatusRejected,
		}, types.ErrSignalNotFound
	}

	// Apply verification bonuses
	bonusScore := pvs.calculateVerificationBonus(signal)

	return ProcessingResult{
		SignalID: signal.ID,
		Score:    updatedSignal.Score + bonusScore,
		Status:   updatedSignal.Status,
		Metadata: map[string]interface{}{
			"verification_bonus_score": bonusScore,
		},
	}, nil
}

// validateProofRequirements validates proof verification requirements
func (pvs *ProofVerificationStrategy) validateProofRequirements(signal types.Signal) error {
	if len(signal.WorkProof.VerificationKey) == 0 {
		return fmt.Errorf("verification key required for proof verification")
	}
	return nil
}

// calculateVerificationBonus calculates bonus score for proof verification
func (pvs *ProofVerificationStrategy) calculateVerificationBonus(signal types.Signal) uint64 {
	bonus := uint64(0)

	// Bonus for having verification key
	if len(signal.WorkProof.VerificationKey) > 0 {
		bonus += 15
	}

	return bonus
}

// GetPriority returns the processing priority
func (pvs *ProofVerificationStrategy) GetPriority(signal types.Signal) float64 {
	return 1.1 // Slightly higher priority for verification work
}

// CanProcess checks if this strategy can process the signal
func (pvs *ProofVerificationStrategy) CanProcess(signal types.Signal) bool {
	return signal.WorkProof.WorkType == "proof_verification"
}

// Optimization Strategy

// OptimizationStrategy implements processing strategy for optimization signals
type OptimizationStrategy struct {
	keeper SignalKeeper
}

// NewOptimizationStrategy creates a new optimization processing strategy
func NewOptimizationStrategy(keeper SignalKeeper) *OptimizationStrategy {
	return &OptimizationStrategy{
		keeper: keeper,
	}
}

// ProcessSignal processes an optimization signal
func (os *OptimizationStrategy) ProcessSignal(ctx keepertypes.Context, signal types.Signal) (ProcessingResult, error) {
	// Process signal
	if err := os.keeper.ProcessSignal(ctx, signal.ID); err != nil {
		return ProcessingResult{
			SignalID: signal.ID,
			Status:   types.SignalStatusRejected,
		}, err
	}

	updatedSignal, found := os.keeper.GetSignal(ctx, signal.ID)
	if !found {
		return ProcessingResult{
			SignalID: signal.ID,
			Status:   types.SignalStatusRejected,
		}, types.ErrSignalNotFound
	}

	// Apply optimization bonuses
	bonusScore := os.calculateOptimizationBonus(signal)

	return ProcessingResult{
		SignalID: signal.ID,
		Score:    updatedSignal.Score + bonusScore,
		Status:   updatedSignal.Status,
		Metadata: map[string]interface{}{
			"optimization_bonus_score": bonusScore,
		},
	}, nil
}

// calculateOptimizationBonus calculates bonus score for optimization work
func (os *OptimizationStrategy) calculateOptimizationBonus(signal types.Signal) uint64 {
	bonus := uint64(0)

	// Bonus for longer optimization time
	if signal.WorkProof.ComputationTime >= 5000 {
		bonus += 20
	}

	return bonus
}

// GetPriority returns the processing priority
func (os *OptimizationStrategy) GetPriority(signal types.Signal) float64 {
	return 1.3 // Higher priority for optimization work
}

// CanProcess checks if this strategy can process the signal
func (os *OptimizationStrategy) CanProcess(signal types.Signal) bool {
	return signal.WorkProof.WorkType == "optimization"
}

// Adaptive Strategy (Future Enhancement)

// AdaptiveStrategy adjusts processing based on network conditions
type AdaptiveStrategy struct {
	keeper     SignalKeeper
	strategies map[string]ProcessingStrategy
}

// NewAdaptiveStrategy creates a new adaptive processing strategy
func NewAdaptiveStrategy(keeper SignalKeeper) *AdaptiveStrategy {
	return &AdaptiveStrategy{
		keeper:     keeper,
		strategies: make(map[string]ProcessingStrategy),
	}
}

// ProcessSignal processes a signal using adaptive logic
func (as *AdaptiveStrategy) ProcessSignal(ctx keepertypes.Context, signal types.Signal) (ProcessingResult, error) {
	// Select best strategy based on current conditions
	strategy := as.selectBestStrategy(ctx, signal)
	return strategy.ProcessSignal(ctx, signal)
}

// selectBestStrategy selects the best strategy based on current conditions
func (as *AdaptiveStrategy) selectBestStrategy(ctx keepertypes.Context, signal types.Signal) ProcessingStrategy {
	// Simplified adaptive logic
	// In practice, this could consider:
	// - Network load
	// - Available resources
	// - Historical performance
	// - Current priorities

	workType := signal.WorkProof.WorkType
	if strategy, exists := as.strategies[workType]; exists {
		return strategy
	}

	// Fall back to default
	return &DefaultProcessingStrategy{keeper: as.keeper}
}

// GetPriority returns the processing priority
func (as *AdaptiveStrategy) GetPriority(signal types.Signal) float64 {
	strategy := as.selectBestStrategy(keepertypes.Context{}, signal) // Simplified
	return strategy.GetPriority(signal)
}

// CanProcess checks if this strategy can process the signal
func (as *AdaptiveStrategy) CanProcess(signal types.Signal) bool {
	return true // Adaptive strategy can handle any signal
}