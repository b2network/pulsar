package types

import (
	"fmt"
)

// AIWorkType represents a type of AI work that can be performed
type AIWorkType struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Category        string    `json:"category"`
	BaseScore       uint64    `json:"base_score"`
	ScoreMultiplier float64   `json:"score_multiplier"`
	Verifier        string    `json:"verifier"`
	MinResources    Resources `json:"min_resources"`
	MaxResources    Resources `json:"max_resources"`
	RequiredProofs  []string  `json:"required_proofs"`
	Enabled         bool      `json:"enabled"`
}

// AIWorkCategory defines categories of AI work
type AIWorkCategory string

const (
	CategoryInference      AIWorkCategory = "inference"
	CategoryTraining       AIWorkCategory = "training"
	CategoryDataProcessing AIWorkCategory = "data_processing"
	CategoryVerification   AIWorkCategory = "verification"
	CategoryOptimization   AIWorkCategory = "optimization"
	CategoryGeneration     AIWorkCategory = "generation"
)

// VerificationMethod defines how AI work is verified
type VerificationMethod string

const (
	VerificationCryptographic VerificationMethod = "cryptographic"
	VerificationConsensus     VerificationMethod = "consensus"
	VerificationReproducible  VerificationMethod = "reproducible"
	VerificationStatistical   VerificationMethod = "statistical"
	VerificationHybrid        VerificationMethod = "hybrid"
)

// AIWorkResult represents the result of AI computation
type AIWorkResult struct {
	SignalID         string            `json:"signal_id"`
	WorkType         string            `json:"work_type"`
	InputData        []byte            `json:"input_data,omitempty"`
	OutputData       []byte            `json:"output_data"`
	Metadata         map[string]string `json:"metadata"`
	VerificationData []byte            `json:"verification_data"`
	Quality          float64           `json:"quality"`
	Confidence       float64           `json:"confidence"`
}

// WorkVerification represents the verification of AI work
type WorkVerification struct {
	SignalID           string             `json:"signal_id"`
	VerifierAddress    string             `json:"verifier_address"`
	VerificationMethod VerificationMethod `json:"verification_method"`
	VerificationTime   int64              `json:"verification_time"`
	IsValid            bool               `json:"is_valid"`
	VerificationProof  []byte             `json:"verification_proof"`
	VerificationScore  uint64             `json:"verification_score"`
	Comments           string             `json:"comments,omitempty"`
}

// DefaultAIWorkTypes returns the default set of AI work types
func DefaultAIWorkTypes() []AIWorkType {
	return []AIWorkType{
		{
			ID:              "llm_inference",
			Name:            "LLM Inference",
			Description:     "Large Language Model inference computation",
			Category:        string(CategoryInference),
			BaseScore:       100,
			ScoreMultiplier: 1.5,
			Verifier:        string(VerificationStatistical),
			MinResources: Resources{
				CPUCycles:    1000000,
				MemoryMB:     512,
				GPUTimeMs:    100,
			},
			MaxResources: Resources{
				CPUCycles:    100000000,
				MemoryMB:     16384,
				GPUTimeMs:    10000,
			},
			RequiredProofs: []string{"input_hash", "output_hash", "model_hash"},
			Enabled:        true,
		},
		{
			ID:              "image_generation",
			Name:            "Image Generation",
			Description:     "AI-powered image generation from prompts",
			Category:        string(CategoryGeneration),
			BaseScore:       150,
			ScoreMultiplier: 2.0,
			Verifier:        string(VerificationConsensus),
			MinResources: Resources{
				CPUCycles:    5000000,
				MemoryMB:     1024,
				GPUTimeMs:    500,
			},
			MaxResources: Resources{
				CPUCycles:    500000000,
				MemoryMB:     32768,
				GPUTimeMs:    60000,
			},
			RequiredProofs: []string{"prompt_hash", "seed", "image_hash"},
			Enabled:        true,
		},
		{
			ID:              "model_training",
			Name:            "Model Training",
			Description:     "Training machine learning models",
			Category:        string(CategoryTraining),
			BaseScore:       500,
			ScoreMultiplier: 3.0,
			Verifier:        string(VerificationReproducible),
			MinResources: Resources{
				CPUCycles:    100000000,
				MemoryMB:     4096,
				GPUTimeMs:    10000,
			},
			MaxResources: Resources{
				CPUCycles:    10000000000,
				MemoryMB:     131072,
				GPUTimeMs:    3600000,
			},
			RequiredProofs: []string{"dataset_hash", "model_architecture", "training_metrics"},
			Enabled:        true,
		},
		{
			ID:              "data_analysis",
			Name:            "Data Analysis",
			Description:     "Complex data analysis and processing",
			Category:        string(CategoryDataProcessing),
			BaseScore:       75,
			ScoreMultiplier: 1.2,
			Verifier:        string(VerificationCryptographic),
			MinResources: Resources{
				CPUCycles:    500000,
				MemoryMB:     256,
				GPUTimeMs:    0,
			},
			MaxResources: Resources{
				CPUCycles:    50000000,
				MemoryMB:     8192,
				GPUTimeMs:    1000,
			},
			RequiredProofs: []string{"data_hash", "algorithm_id", "result_hash"},
			Enabled:        true,
		},
		{
			ID:              "proof_verification",
			Name:            "Proof Verification",
			Description:     "Verification of cryptographic proofs",
			Category:        string(CategoryVerification),
			BaseScore:       50,
			ScoreMultiplier: 1.0,
			Verifier:        string(VerificationCryptographic),
			MinResources: Resources{
				CPUCycles:    100000,
				MemoryMB:     128,
				GPUTimeMs:    0,
			},
			MaxResources: Resources{
				CPUCycles:    10000000,
				MemoryMB:     1024,
				GPUTimeMs:    0,
			},
			RequiredProofs: []string{"proof_data", "verification_key"},
			Enabled:        true,
		},
		{
			ID:              "optimization",
			Name:            "Optimization Task",
			Description:     "Complex optimization problems",
			Category:        string(CategoryOptimization),
			BaseScore:       200,
			ScoreMultiplier: 1.8,
			Verifier:        string(VerificationHybrid),
			MinResources: Resources{
				CPUCycles:    10000000,
				MemoryMB:     512,
				GPUTimeMs:    100,
			},
			MaxResources: Resources{
				CPUCycles:    1000000000,
				MemoryMB:     16384,
				GPUTimeMs:    10000,
			},
			RequiredProofs: []string{"problem_definition", "solution_hash", "objective_value"},
			Enabled:        true,
		},
	}
}

// ValidateBasic performs basic validation on AIWorkType
func (awt AIWorkType) ValidateBasic() error {
	if awt.ID == "" {
		return fmt.Errorf("work type ID cannot be empty")
	}
	if awt.Name == "" {
		return fmt.Errorf("work type name cannot be empty")
	}
	if awt.BaseScore == 0 {
		return fmt.Errorf("base score must be positive")
	}
	if awt.ScoreMultiplier <= 0 {
		return fmt.Errorf("score multiplier must be positive")
	}
	if err := awt.MinResources.ValidateBasic(); err != nil {
		return fmt.Errorf("invalid min resources: %w", err)
	}
	if err := awt.MaxResources.ValidateBasic(); err != nil {
		return fmt.Errorf("invalid max resources: %w", err)
	}
	if !awt.ValidateResourceBounds() {
		return fmt.Errorf("min resources exceed max resources")
	}
	return nil
}

// ValidateResourceBounds checks if min resources are less than or equal to max resources
func (awt AIWorkType) ValidateResourceBounds() bool {
	return awt.MinResources.CPUCycles <= awt.MaxResources.CPUCycles &&
		awt.MinResources.MemoryMB <= awt.MaxResources.MemoryMB &&
		awt.MinResources.GPUTimeMs <= awt.MaxResources.GPUTimeMs &&
		awt.MinResources.NetworkBandwidth <= awt.MaxResources.NetworkBandwidth &&
		awt.MinResources.StorageBytes <= awt.MaxResources.StorageBytes
}

// CheckResourcesInBounds verifies if resources are within the work type bounds
func (awt AIWorkType) CheckResourcesInBounds(resources Resources) bool {
	return resources.CPUCycles >= awt.MinResources.CPUCycles &&
		resources.CPUCycles <= awt.MaxResources.CPUCycles &&
		resources.MemoryMB >= awt.MinResources.MemoryMB &&
		resources.MemoryMB <= awt.MaxResources.MemoryMB &&
		resources.GPUTimeMs >= awt.MinResources.GPUTimeMs &&
		resources.GPUTimeMs <= awt.MaxResources.GPUTimeMs
}

// CalculateWorkScore calculates the score for work based on resources and quality
func (awt AIWorkType) CalculateWorkScore(resources Resources, quality float64) uint64 {
	resourceScore := resources.CalculateResourceScore()
	baseScore := awt.BaseScore

	// Apply multiplier and quality factor
	finalScore := float64(baseScore+resourceScore) * awt.ScoreMultiplier * quality

	if finalScore > float64(^uint64(0)/2) {
		return ^uint64(0) / 2
	}
	return uint64(finalScore)
}

// AIWorkRegistry manages registered AI work types
type AIWorkRegistry struct {
	WorkTypes map[string]AIWorkType `json:"work_types"`
	Version   uint32                `json:"version"`
	UpdatedAt int64                 `json:"updated_at"`
}

// NewAIWorkRegistry creates a new work registry with default types
func NewAIWorkRegistry() *AIWorkRegistry {
	registry := &AIWorkRegistry{
		WorkTypes: make(map[string]AIWorkType),
		Version:   1,
	}

	// Add default work types
	for _, wt := range DefaultAIWorkTypes() {
		registry.WorkTypes[wt.ID] = wt
	}

	return registry
}

// GetWorkType retrieves a work type by ID
func (r *AIWorkRegistry) GetWorkType(id string) (AIWorkType, bool) {
	wt, exists := r.WorkTypes[id]
	return wt, exists
}

// RegisterWorkType registers a new work type
func (r *AIWorkRegistry) RegisterWorkType(wt AIWorkType) error {
	if err := wt.ValidateBasic(); err != nil {
		return fmt.Errorf("invalid work type: %w", err)
	}
	r.WorkTypes[wt.ID] = wt
	r.Version++
	return nil
}

// IsValidWorkType checks if a work type ID is valid and enabled
func (r *AIWorkRegistry) IsValidWorkType(id string) bool {
	wt, exists := r.WorkTypes[id]
	return exists && wt.Enabled
}