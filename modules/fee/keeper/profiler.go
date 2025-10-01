package keeper

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

// GasProfiler tracks and analyzes gas consumption patterns
type GasProfiler struct {
	keeper      *Keeper
	enabled     bool
	samples     []GasSample
	maxSamples  int
	profileData map[string]*ModuleProfile
}

// GasSample represents a single gas consumption sample
type GasSample struct {
	Timestamp    time.Time `json:"timestamp"`
	ModuleName   string    `json:"module_name"`
	MessageType  string    `json:"message_type"`
	GasUsed      uint64    `json:"gas_used"`
	GasLimit     uint64    `json:"gas_limit"`
	Success      bool      `json:"success"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
}

// ModuleProfile contains profiling data for a module
type ModuleProfile struct {
	ModuleName      string                    `json:"module_name"`
	TotalSamples    int                       `json:"total_samples"`
	AverageGas      float64                   `json:"average_gas"`
	MinGas          uint64                    `json:"min_gas"`
	MaxGas          uint64                    `json:"max_gas"`
	SuccessRate     float64                   `json:"success_rate"`
	MessageProfiles map[string]*MessageProfile `json:"message_profiles"`
	LastUpdated     time.Time                 `json:"last_updated"`
}

// MessageProfile contains profiling data for a specific message type
type MessageProfile struct {
	MessageType  string    `json:"message_type"`
	SampleCount  int       `json:"sample_count"`
	AverageGas   float64   `json:"average_gas"`
	MinGas       uint64    `json:"min_gas"`
	MaxGas       uint64    `json:"max_gas"`
	SuccessRate  float64   `json:"success_rate"`
	Percentiles  Percentiles `json:"percentiles"`
	LastSeen     time.Time `json:"last_seen"`
}

// Percentiles contains gas consumption percentile data
type Percentiles struct {
	P50 uint64 `json:"p50"`
	P75 uint64 `json:"p75"`
	P90 uint64 `json:"p90"`
	P95 uint64 `json:"p95"`
	P99 uint64 `json:"p99"`
}

// NewGasProfiler creates a new gas profiler
func NewGasProfiler(keeper *Keeper) *GasProfiler {
	return &GasProfiler{
		keeper:      keeper,
		enabled:     true,
		samples:     make([]GasSample, 0),
		maxSamples:  10000, // Keep last 10k samples
		profileData: make(map[string]*ModuleProfile),
	}
}

// Enable enables gas profiling
func (gp *GasProfiler) Enable() {
	gp.enabled = true
}

// Disable disables gas profiling
func (gp *GasProfiler) Disable() {
	gp.enabled = false
}

// IsEnabled returns whether profiling is enabled
func (gp *GasProfiler) IsEnabled() bool {
	return gp.enabled
}

// RecordGasUsage records gas usage for a transaction
func (gp *GasProfiler) RecordGasUsage(
	moduleName, messageType string,
	gasUsed, gasLimit uint64,
	success bool,
	parameters map[string]interface{},
) {
	if !gp.enabled {
		return
	}

	sample := GasSample{
		Timestamp:   time.Now(),
		ModuleName:  moduleName,
		MessageType: messageType,
		GasUsed:     gasUsed,
		GasLimit:    gasLimit,
		Success:     success,
		Parameters:  parameters,
	}

	gp.addSample(sample)
	gp.updateProfiles(sample)
}

// addSample adds a new sample to the collection
func (gp *GasProfiler) addSample(sample GasSample) {
	gp.samples = append(gp.samples, sample)

	// Keep only the most recent samples
	if len(gp.samples) > gp.maxSamples {
		// Remove oldest samples
		copy(gp.samples, gp.samples[len(gp.samples)-gp.maxSamples:])
		gp.samples = gp.samples[:gp.maxSamples]
	}
}

// updateProfiles updates the profiling data with the new sample
func (gp *GasProfiler) updateProfiles(sample GasSample) {
	// Get or create module profile
	moduleProfile, exists := gp.profileData[sample.ModuleName]
	if !exists {
		moduleProfile = &ModuleProfile{
			ModuleName:      sample.ModuleName,
			MessageProfiles: make(map[string]*MessageProfile),
			MinGas:          sample.GasUsed,
			MaxGas:          sample.GasUsed,
		}
		gp.profileData[sample.ModuleName] = moduleProfile
	}

	// Update module-level statistics
	moduleProfile.TotalSamples++
	moduleProfile.LastUpdated = time.Now()

	if sample.GasUsed < moduleProfile.MinGas {
		moduleProfile.MinGas = sample.GasUsed
	}
	if sample.GasUsed > moduleProfile.MaxGas {
		moduleProfile.MaxGas = sample.GasUsed
	}

	// Get or create message profile
	messageProfile, exists := moduleProfile.MessageProfiles[sample.MessageType]
	if !exists {
		messageProfile = &MessageProfile{
			MessageType: sample.MessageType,
			MinGas:      sample.GasUsed,
			MaxGas:      sample.GasUsed,
		}
		moduleProfile.MessageProfiles[sample.MessageType] = messageProfile
	}

	// Update message-level statistics
	messageProfile.SampleCount++
	messageProfile.LastSeen = time.Now()

	if sample.GasUsed < messageProfile.MinGas {
		messageProfile.MinGas = sample.GasUsed
	}
	if sample.GasUsed > messageProfile.MaxGas {
		messageProfile.MaxGas = sample.GasUsed
	}

	// Recalculate averages and success rates
	gp.recalculateStatistics()
}

// recalculateStatistics recalculates averages and percentiles from samples
func (gp *GasProfiler) recalculateStatistics() {
	// Group samples by module and message type
	moduleData := make(map[string][]GasSample)
	messageData := make(map[string]map[string][]GasSample)

	for _, sample := range gp.samples {
		moduleData[sample.ModuleName] = append(moduleData[sample.ModuleName], sample)

		if messageData[sample.ModuleName] == nil {
			messageData[sample.ModuleName] = make(map[string][]GasSample)
		}
		messageData[sample.ModuleName][sample.MessageType] = append(
			messageData[sample.ModuleName][sample.MessageType], sample)
	}

	// Update module profiles
	for moduleName, samples := range moduleData {
		if profile, exists := gp.profileData[moduleName]; exists {
			profile.AverageGas = calculateAverage(samples)
			profile.SuccessRate = calculateSuccessRate(samples)

			// Update message profiles
			for messageType, messageSamples := range messageData[moduleName] {
				if msgProfile, exists := profile.MessageProfiles[messageType]; exists {
					msgProfile.AverageGas = calculateAverage(messageSamples)
					msgProfile.SuccessRate = calculateSuccessRate(messageSamples)
					msgProfile.Percentiles = calculatePercentiles(messageSamples)
				}
			}
		}
	}
}

// GetModuleProfile returns the profile for a specific module
func (gp *GasProfiler) GetModuleProfile(moduleName string) (*ModuleProfile, bool) {
	profile, exists := gp.profileData[moduleName]
	return profile, exists
}

// GetMessageProfile returns the profile for a specific message type
func (gp *GasProfiler) GetMessageProfile(moduleName, messageType string) (*MessageProfile, bool) {
	if moduleProfile, exists := gp.profileData[moduleName]; exists {
		if messageProfile, exists := moduleProfile.MessageProfiles[messageType]; exists {
			return messageProfile, true
		}
	}
	return nil, false
}

// GetAllProfiles returns all profiling data
func (gp *GasProfiler) GetAllProfiles() map[string]*ModuleProfile {
	return gp.profileData
}

// GetTopGasConsumers returns the top gas consuming operations
func (gp *GasProfiler) GetTopGasConsumers(limit int) []GasConsumerStats {
	var consumers []GasConsumerStats

	for moduleName, moduleProfile := range gp.profileData {
		for messageType, messageProfile := range moduleProfile.MessageProfiles {
			consumers = append(consumers, GasConsumerStats{
				ModuleName:   moduleName,
				MessageType:  messageType,
				AverageGas:   messageProfile.AverageGas,
				MaxGas:       messageProfile.MaxGas,
				SampleCount:  messageProfile.SampleCount,
				SuccessRate:  messageProfile.SuccessRate,
			})
		}
	}

	// Sort by average gas consumption
	sort.Slice(consumers, func(i, j int) bool {
		return consumers[i].AverageGas > consumers[j].AverageGas
	})

	if limit > 0 && len(consumers) > limit {
		consumers = consumers[:limit]
	}

	return consumers
}

// GasConsumerStats represents statistics for a gas consumer
type GasConsumerStats struct {
	ModuleName   string  `json:"module_name"`
	MessageType  string  `json:"message_type"`
	AverageGas   float64 `json:"average_gas"`
	MaxGas       uint64  `json:"max_gas"`
	SampleCount  int     `json:"sample_count"`
	SuccessRate  float64 `json:"success_rate"`
}

// GenerateOptimizationReport generates recommendations for gas optimization
func (gp *GasProfiler) GenerateOptimizationReport() OptimizationReport {
	report := OptimizationReport{
		GeneratedAt:     time.Now(),
		TotalSamples:    len(gp.samples),
		Recommendations: []OptimizationRecommendation{},
	}

	// Analyze each module for optimization opportunities
	for moduleName, moduleProfile := range gp.profileData {
		for messageType, messageProfile := range moduleProfile.MessageProfiles {
			// Check for high gas consumption
			if messageProfile.AverageGas > 100000 {
				report.Recommendations = append(report.Recommendations, OptimizationRecommendation{
					Type:        "HIGH_GAS_CONSUMPTION",
					ModuleName:  moduleName,
					MessageType: messageType,
					Priority:    "HIGH",
					Description: fmt.Sprintf("Message type %s in module %s has high average gas consumption: %.0f",
						messageType, moduleName, messageProfile.AverageGas),
					SuggestedAction: "Review implementation for optimization opportunities",
				})
			}

			// Check for high variance in gas consumption
			variance := float64(messageProfile.MaxGas - messageProfile.MinGas)
			if variance > messageProfile.AverageGas {
				report.Recommendations = append(report.Recommendations, OptimizationRecommendation{
					Type:        "HIGH_VARIANCE",
					ModuleName:  moduleName,
					MessageType: messageType,
					Priority:    "MEDIUM",
					Description: fmt.Sprintf("Message type %s has high gas consumption variance: %d-%d",
						messageType, messageProfile.MinGas, messageProfile.MaxGas),
					SuggestedAction: "Consider implementing more predictable gas consumption patterns",
				})
			}

			// Check for low success rate
			if messageProfile.SuccessRate < 0.95 && messageProfile.SampleCount > 10 {
				report.Recommendations = append(report.Recommendations, OptimizationRecommendation{
					Type:        "LOW_SUCCESS_RATE",
					ModuleName:  moduleName,
					MessageType: messageType,
					Priority:    "HIGH",
					Description: fmt.Sprintf("Message type %s has low success rate: %.2f%%",
						messageType, messageProfile.SuccessRate*100),
					SuggestedAction: "Investigate causes of transaction failures",
				})
			}
		}
	}

	return report
}

// OptimizationReport contains gas optimization recommendations
type OptimizationReport struct {
	GeneratedAt     time.Time                    `json:"generated_at"`
	TotalSamples    int                          `json:"total_samples"`
	Recommendations []OptimizationRecommendation `json:"recommendations"`
}

// OptimizationRecommendation represents a single optimization recommendation
type OptimizationRecommendation struct {
	Type            string `json:"type"`
	ModuleName      string `json:"module_name"`
	MessageType     string `json:"message_type"`
	Priority        string `json:"priority"`
	Description     string `json:"description"`
	SuggestedAction string `json:"suggested_action"`
}

// ExportData exports profiling data to JSON
func (gp *GasProfiler) ExportData() ([]byte, error) {
	data := struct {
		Enabled     bool                     `json:"enabled"`
		TotalSamples int                     `json:"total_samples"`
		Profiles    map[string]*ModuleProfile `json:"profiles"`
		ExportedAt  time.Time               `json:"exported_at"`
	}{
		Enabled:      gp.enabled,
		TotalSamples: len(gp.samples),
		Profiles:     gp.profileData,
		ExportedAt:   time.Now(),
	}

	return json.MarshalIndent(data, "", "  ")
}

// ClearData clears all profiling data
func (gp *GasProfiler) ClearData() {
	gp.samples = make([]GasSample, 0)
	gp.profileData = make(map[string]*ModuleProfile)
}

// SetMaxSamples sets the maximum number of samples to keep
func (gp *GasProfiler) SetMaxSamples(maxSamples int) {
	gp.maxSamples = maxSamples

	// Trim existing samples if necessary
	if len(gp.samples) > maxSamples {
		copy(gp.samples, gp.samples[len(gp.samples)-maxSamples:])
		gp.samples = gp.samples[:maxSamples]
	}
}

// Helper functions

// calculateAverage calculates the average gas consumption from samples
func calculateAverage(samples []GasSample) float64 {
	if len(samples) == 0 {
		return 0
	}

	total := uint64(0)
	for _, sample := range samples {
		total += sample.GasUsed
	}

	return float64(total) / float64(len(samples))
}

// calculateSuccessRate calculates the success rate from samples
func calculateSuccessRate(samples []GasSample) float64 {
	if len(samples) == 0 {
		return 0
	}

	successful := 0
	for _, sample := range samples {
		if sample.Success {
			successful++
		}
	}

	return float64(successful) / float64(len(samples))
}

// calculatePercentiles calculates percentiles from gas usage samples
func calculatePercentiles(samples []GasSample) Percentiles {
	if len(samples) == 0 {
		return Percentiles{}
	}

	// Extract gas values and sort them
	gasValues := make([]uint64, len(samples))
	for i, sample := range samples {
		gasValues[i] = sample.GasUsed
	}
	sort.Slice(gasValues, func(i, j int) bool {
		return gasValues[i] < gasValues[j]
	})

	return Percentiles{
		P50: percentile(gasValues, 0.50),
		P75: percentile(gasValues, 0.75),
		P90: percentile(gasValues, 0.90),
		P95: percentile(gasValues, 0.95),
		P99: percentile(gasValues, 0.99),
	}
}

// percentile calculates the value at a given percentile
func percentile(sortedValues []uint64, p float64) uint64 {
	if len(sortedValues) == 0 {
		return 0
	}

	index := p * float64(len(sortedValues)-1)
	lower := int(index)
	upper := lower + 1

	if upper >= len(sortedValues) {
		return sortedValues[len(sortedValues)-1]
	}

	if lower == upper {
		return sortedValues[lower]
	}

	// Linear interpolation
	weight := index - float64(lower)
	return uint64(float64(sortedValues[lower])*(1-weight) + float64(sortedValues[upper])*weight)
}