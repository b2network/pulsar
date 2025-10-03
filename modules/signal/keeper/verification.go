package keeper

import (
	"crypto/sha256"
	"fmt"
	"time"

	keepertypes "github.com/b2network/pulsar/keeper/types"
	"github.com/b2network/pulsar/modules/signal/types"
)

// AIWorkVerifier handles AI work verification logic
type AIWorkVerifier struct {
	keeper Keeper
}

// VerificationEngine manages the verification process
type VerificationEngine struct {
	keeper     Keeper
	verifiers  map[string]WorkVerifier
	thresholds map[string]VerificationThreshold
}

// WorkVerifier interface for different verification methods
type WorkVerifier interface {
	VerifyWork(ctx keepertypes.Context, signal types.Signal, workType types.AIWorkType) (VerificationResult, error)
	GetVerificationMethod() types.VerificationMethod
	CanVerify(workType types.AIWorkType) bool
}

// VerificationResult represents the result of work verification
type VerificationResult struct {
	IsValid           bool                   `json:"is_valid"`
	Confidence        float64                `json:"confidence"`
	VerificationScore uint64                 `json:"verification_score"`
	Method            types.VerificationMethod `json:"method"`
	Details           map[string]interface{} `json:"details"`
	Errors            []string               `json:"errors"`
	Warnings          []string               `json:"warnings"`
	VerificationTime  time.Duration          `json:"verification_time"`
}

// VerificationThreshold defines thresholds for verification acceptance
type VerificationThreshold struct {
	MinConfidence     float64 `json:"min_confidence"`
	MinVerifiers      int     `json:"min_verifiers"`
	ConsensusRequired bool    `json:"consensus_required"`
	TimeoutSeconds    int64   `json:"timeout_seconds"`
}

// NewAIWorkVerifier creates a new AI work verifier
func NewAIWorkVerifier(keeper Keeper) *AIWorkVerifier {
	return &AIWorkVerifier{
		keeper: keeper,
	}
}

// NewVerificationEngine creates a new verification engine
func NewVerificationEngine(keeper Keeper) *VerificationEngine {
	engine := &VerificationEngine{
		keeper:     keeper,
		verifiers:  make(map[string]WorkVerifier),
		thresholds: make(map[string]VerificationThreshold),
	}

	// Initialize default verifiers
	engine.initializeDefaultVerifiers()
	engine.initializeDefaultThresholds()

	return engine
}

// VerifyAIWork performs comprehensive AI work verification
func (awv *AIWorkVerifier) VerifyAIWork(ctx keepertypes.Context, signal types.Signal) (types.WorkVerification, error) {
	startTime := time.Now()

	workType, found := awv.keeper.GetWorkType(ctx, signal.WorkProof.WorkType)
	if !found {
		return types.WorkVerification{}, types.ErrWorkTypeNotFound
	}

	verification := types.WorkVerification{
		SignalID:           signal.ID,
		VerifierAddress:    "system", // System verification
		VerificationMethod: types.VerificationMethod(workType.Verifier),
		VerificationTime:   ctx.BlockTime().Unix(),
	}

	// Perform verification based on work type
	result, err := awv.performVerification(ctx, signal, workType)
	if err != nil {
		verification.IsValid = false
		verification.Comments = err.Error()
		return verification, err
	}

	verification.IsValid = result.IsValid
	verification.VerificationScore = result.VerificationScore
	verification.VerificationProof = awv.generateVerificationProof(signal, result)

	awv.keeper.Logger(ctx).Info("AI work verification completed",
		"signal_id", signal.ID,
		"work_type", signal.WorkProof.WorkType,
		"is_valid", result.IsValid,
		"confidence", result.Confidence,
		"verification_time", time.Since(startTime))

	return verification, nil
}

// performVerification performs verification based on the verification method
func (awv *AIWorkVerifier) performVerification(ctx keepertypes.Context, signal types.Signal, workType types.AIWorkType) (VerificationResult, error) {
	method := types.VerificationMethod(workType.Verifier)

	switch method {
	case types.VerificationCryptographic:
		return awv.verifyCryptographic(ctx, signal, workType)
	case types.VerificationReproducible:
		return awv.verifyReproducible(ctx, signal, workType)
	case types.VerificationStatistical:
		return awv.verifyStatistical(ctx, signal, workType)
	case types.VerificationConsensus:
		return awv.verifyConsensus(ctx, signal, workType)
	case types.VerificationHybrid:
		return awv.verifyHybrid(ctx, signal, workType)
	default:
		return VerificationResult{}, types.ErrInvalidVerificationMethod
	}
}

// verifyCryptographic performs cryptographic verification
func (awv *AIWorkVerifier) verifyCryptographic(ctx keepertypes.Context, signal types.Signal, workType types.AIWorkType) (VerificationResult, error) {
	result := VerificationResult{
		Method:  types.VerificationCryptographic,
		Details: make(map[string]interface{}),
	}

	// Verify digital signatures
	if len(signal.Signature) > 0 {
		if valid := awv.verifySignature(signal); valid {
			result.Details["signature_valid"] = true
			result.VerificationScore += 30
		} else {
			result.Errors = append(result.Errors, "invalid digital signature")
			result.IsValid = false
			return result, fmt.Errorf("cryptographic verification failed: invalid signature")
		}
	}

	// Verify proof of work
	if valid := awv.verifyProofOfWork(signal); valid {
		result.Details["proof_of_work_valid"] = true
		result.VerificationScore += 40
	} else {
		result.Errors = append(result.Errors, "invalid proof of work")
	}

	// Verify hash chain integrity
	if valid := awv.verifyHashChain(signal); valid {
		result.Details["hash_chain_valid"] = true
		result.VerificationScore += 30
	} else {
		result.Warnings = append(result.Warnings, "hash chain verification failed")
	}

	result.IsValid = len(result.Errors) == 0
	result.Confidence = float64(result.VerificationScore) / 100.0
	if result.Confidence > 1.0 {
		result.Confidence = 1.0
	}

	return result, nil
}

// verifyReproducible performs reproducible verification
func (awv *AIWorkVerifier) verifyReproducible(ctx keepertypes.Context, signal types.Signal, workType types.AIWorkType) (VerificationResult, error) {
	result := VerificationResult{
		Method:  types.VerificationReproducible,
		Details: make(map[string]interface{}),
	}

	// Simulate re-execution of the computation
	reproduced, err := awv.reproduceComputation(ctx, signal, workType)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("reproduction failed: %s", err.Error()))
		result.IsValid = false
		return result, err
	}

	// Compare results
	if awv.compareResults(signal.WorkProof.OutputHash, reproduced.OutputHash) {
		result.Details["output_matches"] = true
		result.VerificationScore += 50
	} else {
		result.Errors = append(result.Errors, "output hash mismatch")
		result.IsValid = false
	}

	// Check computation time consistency
	if awv.checkComputationTimeConsistency(signal.WorkProof.ComputationTime, reproduced.ComputationTime) {
		result.Details["computation_time_consistent"] = true
		result.VerificationScore += 30
	} else {
		result.Warnings = append(result.Warnings, "computation time inconsistent")
	}

	// Check resource usage consistency
	if awv.checkResourceConsistency(signal.WorkProof.ResourcesUsed, reproduced.ResourcesUsed) {
		result.Details["resource_usage_consistent"] = true
		result.VerificationScore += 20
	} else {
		result.Warnings = append(result.Warnings, "resource usage inconsistent")
	}

	result.IsValid = len(result.Errors) == 0
	result.Confidence = float64(result.VerificationScore) / 100.0

	return result, nil
}

// verifyStatistical performs statistical verification
func (awv *AIWorkVerifier) verifyStatistical(ctx keepertypes.Context, signal types.Signal, workType types.AIWorkType) (VerificationResult, error) {
	result := VerificationResult{
		Method:  types.VerificationStatistical,
		Details: make(map[string]interface{}),
	}

	// Statistical analysis of work patterns
	patterns := awv.analyzeWorkPatterns(ctx, signal)
	result.Details["work_patterns"] = patterns

	// Anomaly detection
	if anomalies := awv.detectAnomalies(signal, patterns); len(anomalies) > 0 {
		result.Warnings = append(result.Warnings, anomalies...)
		result.VerificationScore -= uint64(len(anomalies) * 10)
	} else {
		result.VerificationScore += 40
	}

	// Quality assessment
	quality := awv.assessWorkQuality(signal, workType)
	result.Details["quality_score"] = quality
	result.VerificationScore += uint64(quality * 30)

	// Performance benchmarking
	performance := awv.benchmarkPerformance(signal, workType)
	result.Details["performance_score"] = performance
	result.VerificationScore += uint64(performance * 20)

	result.IsValid = result.VerificationScore >= 50 // Threshold
	result.Confidence = float64(result.VerificationScore) / 100.0

	return result, nil
}

// verifyConsensus performs consensus-based verification
func (awv *AIWorkVerifier) verifyConsensus(ctx keepertypes.Context, signal types.Signal, workType types.AIWorkType) (VerificationResult, error) {
	result := VerificationResult{
		Method:  types.VerificationConsensus,
		Details: make(map[string]interface{}),
	}

	// Get all verifications for this signal
	verifications := awv.keeper.GetWorkVerifications(ctx, signal.ID)
	result.Details["verification_count"] = len(verifications)

	params := awv.keeper.GetParams(ctx)

	// Check if we have minimum required verifications
	if uint32(len(verifications)) < params.VerificationThreshold {
		result.Warnings = append(result.Warnings, fmt.Sprintf("insufficient verifications: %d < %d", len(verifications), params.VerificationThreshold))
		result.IsValid = false
		return result, nil
	}

	// Calculate consensus
	validCount := 0
	totalScore := uint64(0)

	for _, verification := range verifications {
		if verification.IsValid {
			validCount++
		}
		totalScore += verification.VerificationScore
	}

	consensusRatio := float64(validCount) / float64(len(verifications))
	result.Details["consensus_ratio"] = consensusRatio
	result.Details["valid_verifications"] = validCount

	// Require majority consensus
	if consensusRatio >= 0.51 {
		result.IsValid = true
		result.VerificationScore = totalScore / uint64(len(verifications))
		result.Confidence = consensusRatio
	} else {
		result.IsValid = false
		result.Errors = append(result.Errors, "consensus not reached")
	}

	return result, nil
}

// verifyHybrid performs hybrid verification combining multiple methods
func (awv *AIWorkVerifier) verifyHybrid(ctx keepertypes.Context, signal types.Signal, workType types.AIWorkType) (VerificationResult, error) {
	result := VerificationResult{
		Method:  types.VerificationHybrid,
		Details: make(map[string]interface{}),
	}

	// Combine cryptographic and statistical verification
	cryptoResult, err := awv.verifyCryptographic(ctx, signal, workType)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("cryptographic verification failed: %s", err.Error()))
	} else {
		result.Details["cryptographic"] = cryptoResult
		result.VerificationScore += cryptoResult.VerificationScore / 2 // Weight 50%
	}

	statResult, err := awv.verifyStatistical(ctx, signal, workType)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("statistical verification failed: %s", err.Error()))
	} else {
		result.Details["statistical"] = statResult
		result.VerificationScore += statResult.VerificationScore / 2 // Weight 50%
	}

	// Both verifications must pass for hybrid verification to succeed
	result.IsValid = cryptoResult.IsValid && statResult.IsValid && len(result.Errors) == 0
	result.Confidence = (cryptoResult.Confidence + statResult.Confidence) / 2.0

	return result, nil
}

// Helper methods for verification

// verifySignature verifies the digital signature of a signal
func (awv *AIWorkVerifier) verifySignature(signal types.Signal) bool {
	// Simplified signature verification
	// In practice, this would use proper cryptographic libraries
	return len(signal.Signature) >= 64
}

// verifyProofOfWork verifies the proof of work
func (awv *AIWorkVerifier) verifyProofOfWork(signal types.Signal) bool {
	// Simplified PoW verification
	// Check if nonce produces expected hash
	data := fmt.Sprintf("%s:%d:%d", signal.ID, signal.WorkProof.Nonce, signal.WorkProof.ComputationTime)
	hash := sha256.Sum256([]byte(data))

	// Simple difficulty check (hash starts with zero)
	return hash[0] == 0
}

// verifyHashChain verifies hash chain integrity
func (awv *AIWorkVerifier) verifyHashChain(signal types.Signal) bool {
	// Verify input -> output hash relationship
	expectedOutput := sha256.Sum256(signal.WorkProof.InputHash)
	return string(expectedOutput[:]) == string(signal.WorkProof.OutputHash)
}

// reproduceComputation attempts to reproduce the computation
func (awv *AIWorkVerifier) reproduceComputation(ctx keepertypes.Context, signal types.Signal, workType types.AIWorkType) (types.WorkProof, error) {
	// Simplified reproduction simulation
	// In practice, this would actually re-run the computation
	return types.WorkProof{
		WorkType:        signal.WorkProof.WorkType,
		InputHash:       signal.WorkProof.InputHash,
		OutputHash:      signal.WorkProof.OutputHash, // Assume matches for simplicity
		ComputationTime: signal.WorkProof.ComputationTime + 100, // Slight variation
		ResourcesUsed:   signal.WorkProof.ResourcesUsed,
	}, nil
}

// compareResults compares computation results
func (awv *AIWorkVerifier) compareResults(original, reproduced []byte) bool {
	return string(original) == string(reproduced)
}

// checkComputationTimeConsistency checks if computation times are consistent
func (awv *AIWorkVerifier) checkComputationTimeConsistency(original, reproduced int64) bool {
	// Allow 20% variance
	variance := float64(abs(original-reproduced)) / float64(original)
	return variance <= 0.2
}

// checkResourceConsistency checks if resource usage is consistent
func (awv *AIWorkVerifier) checkResourceConsistency(original, reproduced types.Resources) bool {
	// Simple consistency check
	cpuVariance := float64(abs(int64(original.CPUCycles)-int64(reproduced.CPUCycles))) / float64(original.CPUCycles)
	memVariance := float64(abs(int64(original.MemoryMB)-int64(reproduced.MemoryMB))) / float64(original.MemoryMB)

	return cpuVariance <= 0.1 && memVariance <= 0.1
}

// analyzeWorkPatterns analyzes patterns in work submission
func (awv *AIWorkVerifier) analyzeWorkPatterns(ctx keepertypes.Context, signal types.Signal) map[string]interface{} {
	// Get historical data for pattern analysis
	validatorSignals := awv.keeper.GetValidatorSignals(ctx, signal.ValidatorAddress)

	patterns := make(map[string]interface{})
	patterns["historical_count"] = len(validatorSignals)

	// Analyze submission frequency
	if len(validatorSignals) > 1 {
		intervals := make([]int64, len(validatorSignals)-1)
		for i := 1; i < len(validatorSignals); i++ {
			intervals[i-1] = validatorSignals[i].Timestamp - validatorSignals[i-1].Timestamp
		}

		// Calculate average interval
		var total int64
		for _, interval := range intervals {
			total += interval
		}
		patterns["avg_submission_interval"] = total / int64(len(intervals))
	}

	return patterns
}

// detectAnomalies detects anomalies in work patterns
func (awv *AIWorkVerifier) detectAnomalies(signal types.Signal, patterns map[string]interface{}) []string {
	var anomalies []string

	// Check for suspiciously fast computation
	if signal.WorkProof.ComputationTime < 100 {
		anomalies = append(anomalies, "computation time suspiciously fast")
	}

	// Check for resource anomalies
	if signal.WorkProof.ResourcesUsed.CPUCycles == 0 && signal.WorkProof.ResourcesUsed.GPUTimeMs == 0 {
		anomalies = append(anomalies, "no computational resources used")
	}

	return anomalies
}

// assessWorkQuality assesses the quality of work performed
func (awv *AIWorkVerifier) assessWorkQuality(signal types.Signal, workType types.AIWorkType) float64 {
	// Simplified quality assessment
	baseQuality := 0.5

	// Bonus for meeting resource requirements
	if workType.CheckResourcesInBounds(signal.WorkProof.ResourcesUsed) {
		baseQuality += 0.3
	}

	// Bonus for reasonable computation time
	if signal.WorkProof.ComputationTime >= 1000 {
		baseQuality += 0.2
	}

	if baseQuality > 1.0 {
		baseQuality = 1.0
	}

	return baseQuality
}

// benchmarkPerformance benchmarks work performance
func (awv *AIWorkVerifier) benchmarkPerformance(signal types.Signal, workType types.AIWorkType) float64 {
	// Simplified performance benchmarking
	expectedTime := float64(workType.MinResources.GetTotalComputationUnits()) / 1000.0
	actualTime := float64(signal.WorkProof.ComputationTime)

	if actualTime <= 0 {
		return 0.0
	}

	performance := expectedTime / actualTime
	if performance > 1.0 {
		performance = 1.0
	}

	return performance
}

// generateVerificationProof generates a proof of verification
func (awv *AIWorkVerifier) generateVerificationProof(signal types.Signal, result VerificationResult) []byte {
	// Generate verification proof hash
	data := fmt.Sprintf("%s:%s:%t:%f", signal.ID, result.Method, result.IsValid, result.Confidence)
	hash := sha256.Sum256([]byte(data))
	return hash[:]
}

// initializeDefaultVerifiers initializes default verification methods
func (ve *VerificationEngine) initializeDefaultVerifiers() {
	// Initialize default verifiers for different methods
	// This would be expanded with actual verifier implementations
}

// initializeDefaultThresholds initializes default verification thresholds
func (ve *VerificationEngine) initializeDefaultThresholds() {
	ve.thresholds["llm_inference"] = VerificationThreshold{
		MinConfidence:     0.7,
		MinVerifiers:      2,
		ConsensusRequired: true,
		TimeoutSeconds:    300,
	}

	ve.thresholds["image_generation"] = VerificationThreshold{
		MinConfidence:     0.8,
		MinVerifiers:      3,
		ConsensusRequired: true,
		TimeoutSeconds:    600,
	}

	ve.thresholds["model_training"] = VerificationThreshold{
		MinConfidence:     0.9,
		MinVerifiers:      5,
		ConsensusRequired: true,
		TimeoutSeconds:    1800,
	}
}

// abs returns the absolute value of an integer
func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}