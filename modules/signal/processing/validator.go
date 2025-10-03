package processing

import (
	"crypto/sha256"
	"fmt"
	"time"

	keepertypes "github.com/b2network/pulsar/keeper/types"
	"github.com/b2network/pulsar/modules/signal/types"
)

// SignalValidator handles signal validation logic
type SignalValidator struct {
	keeper SignalKeeper
}

// ValidationResult represents the result of signal validation
type ValidationResult struct {
	IsValid        bool                   `json:"is_valid"`
	Score          uint64                 `json:"score"`
	Errors         []string               `json:"errors"`
	Warnings       []string               `json:"warnings"`
	ValidationTime time.Duration          `json:"validation_time"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// NewSignalValidator creates a new signal validator
func NewSignalValidator(keeper SignalKeeper) *SignalValidator {
	return &SignalValidator{
		keeper: keeper,
	}
}

// ValidateSignal performs comprehensive validation of a signal
func (sv *SignalValidator) ValidateSignal(ctx keepertypes.Context, signal types.Signal) ValidationResult {
	startTime := time.Now()
	result := ValidationResult{
		IsValid:  true,
		Errors:   []string{},
		Warnings: []string{},
		Metadata: make(map[string]interface{}),
	}

	// Basic signal validation
	if err := sv.validateBasicSignal(signal); err != nil {
		result.IsValid = false
		result.Errors = append(result.Errors, err.Error())
	}

	// Work proof validation
	if err := sv.validateWorkProof(ctx, signal.WorkProof); err != nil {
		result.IsValid = false
		result.Errors = append(result.Errors, err.Error())
	}

	// Time-based validation
	if warnings := sv.validateTiming(ctx, signal); len(warnings) > 0 {
		result.Warnings = append(result.Warnings, warnings...)
	}

	// Work type specific validation
	if err := sv.validateWorkTypeSpecific(ctx, signal); err != nil {
		result.IsValid = false
		result.Errors = append(result.Errors, err.Error())
	}

	// Resource validation
	if err := sv.validateResources(ctx, signal.WorkProof.ResourcesUsed, signal.WorkProof.WorkType); err != nil {
		result.IsValid = false
		result.Errors = append(result.Errors, err.Error())
	}

	// Cryptographic validation
	if err := sv.validateCryptographic(signal); err != nil {
		result.IsValid = false
		result.Errors = append(result.Errors, err.Error())
	}

	// Calculate validation score
	if result.IsValid {
		result.Score = sv.calculateValidationScore(ctx, signal, result)
	}

	result.ValidationTime = time.Since(startTime)

	sv.keeper.Logger(ctx).Debug("signal validation completed",
		"signal_id", signal.ID,
		"is_valid", result.IsValid,
		"score", result.Score,
		"errors", len(result.Errors),
		"warnings", len(result.Warnings),
		"validation_time", result.ValidationTime)

	return result
}

// validateBasicSignal performs basic signal structure validation
func (sv *SignalValidator) validateBasicSignal(signal types.Signal) error {
	if err := signal.ValidateBasic(); err != nil {
		return fmt.Errorf("basic validation failed: %w", err)
	}

	// Additional checks
	if len(signal.PayloadHash) != 32 {
		return fmt.Errorf("invalid payload hash length: expected 32, got %d", len(signal.PayloadHash))
	}

	if signal.Timestamp <= 0 {
		return fmt.Errorf("invalid timestamp: %d", signal.Timestamp)
	}

	return nil
}

// validateWorkProof validates the work proof structure and content
func (sv *SignalValidator) validateWorkProof(ctx keepertypes.Context, workProof types.WorkProof) error {
	if err := workProof.ValidateBasic(); err != nil {
		return fmt.Errorf("work proof validation failed: %w", err)
	}

	// Validate hash formats
	if len(workProof.InputHash) != 32 {
		return fmt.Errorf("invalid input hash length: expected 32, got %d", len(workProof.InputHash))
	}

	if len(workProof.OutputHash) != 32 {
		return fmt.Errorf("invalid output hash length: expected 32, got %d", len(workProof.OutputHash))
	}

	// Validate computation time is reasonable
	if workProof.ComputationTime <= 0 {
		return fmt.Errorf("computation time must be positive")
	}

	if workProof.ComputationTime > 86400000 { // 24 hours in milliseconds
		return fmt.Errorf("computation time too long: %d ms", workProof.ComputationTime)
	}

	// Validate nonce
	if workProof.Nonce == 0 {
		return fmt.Errorf("nonce cannot be zero")
	}

	return nil
}

// validateTiming performs time-based validation
func (sv *SignalValidator) validateTiming(ctx keepertypes.Context, signal types.Signal) []string {
	var warnings []string
	currentTime := ctx.BlockTime().Unix()
	params := sv.keeper.GetParams(ctx)

	// Check if signal is too old
	age := currentTime - signal.Timestamp
	if age > params.MaxSignalAge {
		warnings = append(warnings, fmt.Sprintf("signal is %d seconds old (max: %d)", age, params.MaxSignalAge))
	}

	// Check if signal is from the future
	if signal.Timestamp > currentTime+300 { // 5 minutes tolerance
		warnings = append(warnings, "signal timestamp is in the future")
	}

	// Check computation time consistency
	if signal.WorkProof.ComputationTime > age*1000 {
		warnings = append(warnings, "computation time exceeds signal age")
	}

	return warnings
}

// validateWorkTypeSpecific performs work type specific validation
func (sv *SignalValidator) validateWorkTypeSpecific(ctx keepertypes.Context, signal types.Signal) error {
	workType, found := sv.keeper.GetWorkType(ctx, signal.WorkProof.WorkType)
	if !found {
		return types.ErrWorkTypeNotFound
	}

	if !workType.Enabled {
		return types.ErrWorkTypeDisabled
	}

	// Check required proofs based on work type
	switch signal.WorkProof.WorkType {
	case "llm_inference":
		return sv.validateLLMInference(signal.WorkProof)
	case "image_generation":
		return sv.validateImageGeneration(signal.WorkProof)
	case "model_training":
		return sv.validateModelTraining(signal.WorkProof)
	case "data_analysis":
		return sv.validateDataAnalysis(signal.WorkProof)
	case "proof_verification":
		return sv.validateProofVerification(signal.WorkProof)
	case "optimization":
		return sv.validateOptimization(signal.WorkProof)
	default:
		// Generic validation for unknown work types
		return sv.validateGenericWork(signal.WorkProof)
	}
}

// validateResources validates computational resources
func (sv *SignalValidator) validateResources(ctx keepertypes.Context, resources types.Resources, workTypeID string) error {
	workType, found := sv.keeper.GetWorkType(ctx, workTypeID)
	if !found {
		return types.ErrWorkTypeNotFound
	}

	// Check if resources are within bounds
	if !workType.CheckResourcesInBounds(resources) {
		return types.ErrResourcesOutOfBounds
	}

	// Validate resource consistency
	if err := sv.validateResourceConsistency(resources); err != nil {
		return err
	}

	return nil
}

// validateResourceConsistency checks for logical consistency in resource usage
func (sv *SignalValidator) validateResourceConsistency(resources types.Resources) error {
	// CPU and GPU time should be correlated with memory usage
	totalComputeTime := resources.CPUCycles/1000000 + resources.GPUTimeMs

	// Very rough heuristic: memory should be at least 1MB per second of compute
	minMemory := totalComputeTime
	if resources.MemoryMB < minMemory {
		return fmt.Errorf("memory usage too low for compute time: %d MB for %d ms", resources.MemoryMB, totalComputeTime)
	}

	// Network bandwidth should be reasonable for the computation
	if resources.NetworkBandwidth > resources.StorageBytes*10 {
		return fmt.Errorf("network bandwidth suspiciously high compared to storage: %d bytes", resources.NetworkBandwidth)
	}

	return nil
}

// validateCryptographic performs cryptographic validation
func (sv *SignalValidator) validateCryptographic(signal types.Signal) error {
	// Validate signature if present
	if len(signal.Signature) > 0 {
		if err := sv.validateSignature(signal); err != nil {
			return fmt.Errorf("signature validation failed: %w", err)
		}
	}

	// Validate verification key if present
	if len(signal.WorkProof.VerificationKey) > 0 {
		if err := sv.validateVerificationKey(signal.WorkProof); err != nil {
			return fmt.Errorf("verification key validation failed: %w", err)
		}
	}

	return nil
}

// validateSignature validates the signal's digital signature
func (sv *SignalValidator) validateSignature(signal types.Signal) error {
	// Simplified signature validation
	// In a real implementation, this would:
	// 1. Extract the public key from the creator address
	// 2. Reconstruct the signed message
	// 3. Verify the signature using proper cryptographic libraries

	if len(signal.Signature) < 64 {
		return fmt.Errorf("signature too short: %d bytes", len(signal.Signature))
	}

	return nil // Simplified - assume valid
}

// validateVerificationKey validates the work proof verification key
func (sv *SignalValidator) validateVerificationKey(workProof types.WorkProof) error {
	// Simplified verification key validation
	if len(workProof.VerificationKey) < 32 {
		return fmt.Errorf("verification key too short: %d bytes", len(workProof.VerificationKey))
	}

	return nil // Simplified - assume valid
}

// Work type specific validation methods

func (sv *SignalValidator) validateLLMInference(workProof types.WorkProof) error {
	// LLM inference specific validation
	if workProof.ResourcesUsed.MemoryMB < 512 {
		return fmt.Errorf("insufficient memory for LLM inference: %d MB", workProof.ResourcesUsed.MemoryMB)
	}

	if workProof.ComputationTime < 100 { // At least 100ms
		return fmt.Errorf("computation time too short for LLM inference: %d ms", workProof.ComputationTime)
	}

	return nil
}

func (sv *SignalValidator) validateImageGeneration(workProof types.WorkProof) error {
	// Image generation specific validation
	if workProof.ResourcesUsed.GPUTimeMs == 0 {
		return fmt.Errorf("GPU time required for image generation")
	}

	if workProof.ResourcesUsed.MemoryMB < 1024 {
		return fmt.Errorf("insufficient memory for image generation: %d MB", workProof.ResourcesUsed.MemoryMB)
	}

	return nil
}

func (sv *SignalValidator) validateModelTraining(workProof types.WorkProof) error {
	// Model training specific validation
	if workProof.ComputationTime < 10000 { // At least 10 seconds
		return fmt.Errorf("computation time too short for model training: %d ms", workProof.ComputationTime)
	}

	if workProof.ResourcesUsed.MemoryMB < 2048 {
		return fmt.Errorf("insufficient memory for model training: %d MB", workProof.ResourcesUsed.MemoryMB)
	}

	return nil
}

func (sv *SignalValidator) validateDataAnalysis(workProof types.WorkProof) error {
	// Data analysis specific validation
	if workProof.ResourcesUsed.StorageBytes == 0 {
		return fmt.Errorf("storage access required for data analysis")
	}

	return nil
}

func (sv *SignalValidator) validateProofVerification(workProof types.WorkProof) error {
	// Proof verification specific validation
	if len(workProof.VerificationKey) == 0 {
		return fmt.Errorf("verification key required for proof verification")
	}

	if workProof.ResourcesUsed.CPUCycles < 100000 {
		return fmt.Errorf("insufficient CPU cycles for proof verification: %d", workProof.ResourcesUsed.CPUCycles)
	}

	return nil
}

func (sv *SignalValidator) validateOptimization(workProof types.WorkProof) error {
	// Optimization specific validation
	if workProof.ComputationTime < 1000 { // At least 1 second
		return fmt.Errorf("computation time too short for optimization: %d ms", workProof.ComputationTime)
	}

	return nil
}

func (sv *SignalValidator) validateGenericWork(workProof types.WorkProof) error {
	// Generic work validation
	totalResources := workProof.ResourcesUsed.GetTotalComputationUnits()
	if totalResources == 0 {
		return fmt.Errorf("no computational resources used")
	}

	return nil
}

// calculateValidationScore calculates a score based on validation results
func (sv *SignalValidator) calculateValidationScore(ctx keepertypes.Context, signal types.Signal, result ValidationResult) uint64 {
	baseScore := uint64(100)

	// Bonus for clean validation (no warnings)
	if len(result.Warnings) == 0 {
		baseScore += 50
	}

	// Bonus for cryptographic proofs
	if len(signal.Signature) > 0 {
		baseScore += 25
	}

	if len(signal.WorkProof.VerificationKey) > 0 {
		baseScore += 25
	}

	// Bonus for resource efficiency
	workType, found := sv.keeper.GetWorkType(ctx, signal.WorkProof.WorkType)
	if found {
		efficiency := sv.calculateResourceEfficiency(signal.WorkProof.ResourcesUsed, workType)
		baseScore += uint64(efficiency * 50) // Up to 50 bonus points
	}

	// Fast validation bonus
	if result.ValidationTime < time.Millisecond*100 {
		baseScore += 10
	}

	return baseScore
}

// calculateResourceEfficiency calculates how efficiently resources were used
func (sv *SignalValidator) calculateResourceEfficiency(resources types.Resources, workType types.AIWorkType) float64 {
	// Calculate efficiency as inverse of resource usage relative to maximum
	cpuEfficiency := 1.0 - float64(resources.CPUCycles)/float64(workType.MaxResources.CPUCycles)
	memoryEfficiency := 1.0 - float64(resources.MemoryMB)/float64(workType.MaxResources.MemoryMB)
	gpuEfficiency := 1.0

	if workType.MaxResources.GPUTimeMs > 0 {
		gpuEfficiency = 1.0 - float64(resources.GPUTimeMs)/float64(workType.MaxResources.GPUTimeMs)
	}

	// Average efficiency
	totalEfficiency := (cpuEfficiency + memoryEfficiency + gpuEfficiency) / 3.0

	// Ensure efficiency is between 0 and 1
	if totalEfficiency < 0 {
		totalEfficiency = 0
	}
	if totalEfficiency > 1 {
		totalEfficiency = 1
	}

	return totalEfficiency
}

// ValidateHash validates a hash value
func (sv *SignalValidator) ValidateHash(data []byte, expectedHash []byte) bool {
	hash := sha256.Sum256(data)
	return string(hash[:]) == string(expectedHash)
}