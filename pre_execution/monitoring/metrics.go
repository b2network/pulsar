package monitoring

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	preexectypes "github.com/b2network/pulsar/pre_execution/types"
)

// PreExecutionMetrics collects and aggregates performance metrics for the pre-execution system
type PreExecutionMetrics struct {
	mu sync.RWMutex

	// Execution metrics
	totalPreExecutions uint64
	successfulPreExecs uint64
	failedPreExecs     uint64
	cacheHits          uint64
	cacheMisses        uint64

	// Performance metrics
	executionTimes   []time.Duration // Recent execution times for averaging
	gasUsageHistory  []uint64        // Recent gas usage for averaging
	maxExecutionTime time.Duration   // Longest execution time recorded
	minExecutionTime time.Duration   // Shortest execution time recorded

	// Resource metrics
	memoryUsage         uint64 // Current memory usage in bytes
	maxMemoryUsage      uint64 // Peak memory usage
	activePreExecutions int    // Currently running pre-executions
	maxConcurrent       int    // Peak concurrent pre-executions

	// Module-specific metrics
	moduleMetrics map[string]*ModuleMetrics

	// Time-based metrics
	startTime     time.Time
	lastResetTime time.Time

	// Error tracking
	errorsByType map[string]uint64 // Error code -> count
	recentErrors []ErrorRecord     // Recent error details

	// Configuration
	historySize      int // How many recent samples to keep
	errorHistorySize int // How many recent errors to keep
}

// ModuleMetrics tracks metrics for a specific module
type ModuleMetrics struct {
	ModuleName      string            `json:"module_name"`
	PreExecutions   uint64            `json:"pre_executions"`
	Successes       uint64            `json:"successes"`
	Failures        uint64            `json:"failures"`
	AverageGasUsed  uint64            `json:"average_gas_used"`
	AverageExecTime time.Duration     `json:"average_execution_time"`
	CacheHits       uint64            `json:"cache_hits"`
	CacheMisses     uint64            `json:"cache_misses"`
	MsgTypeMetrics  map[string]uint64 `json:"msg_type_metrics"` // MsgType -> count
}

// ErrorRecord tracks details about execution errors
type ErrorRecord struct {
	Timestamp time.Time `json:"timestamp"`
	ErrorType string    `json:"error_type"`
	ErrorCode uint32    `json:"error_code"`
	Message   string    `json:"message"`
	TxHash    string    `json:"tx_hash,omitempty"`
	Module    string    `json:"module,omitempty"`
	MsgType   string    `json:"msg_type,omitempty"`
}

// MetricsConfig contains configuration for metrics collection
type MetricsConfig struct {
	HistorySize      int `json:"history_size"`       // Number of recent samples to keep
	ErrorHistorySize int `json:"error_history_size"` // Number of recent errors to keep
}

// NewPreExecutionMetrics creates a new metrics collector
func NewPreExecutionMetrics(config MetricsConfig) *PreExecutionMetrics {
	return &PreExecutionMetrics{
		executionTimes:   make([]time.Duration, 0, config.HistorySize),
		gasUsageHistory:  make([]uint64, 0, config.HistorySize),
		moduleMetrics:    make(map[string]*ModuleMetrics),
		errorsByType:     make(map[string]uint64),
		recentErrors:     make([]ErrorRecord, 0, config.ErrorHistorySize),
		startTime:        time.Now(),
		lastResetTime:    time.Now(),
		historySize:      config.HistorySize,
		errorHistorySize: config.ErrorHistorySize,
		minExecutionTime: time.Hour, // Initialize to very high value
	}
}

// RecordPreExecution records metrics for a completed pre-execution
func (m *PreExecutionMetrics) RecordPreExecution(moduleName, msgType string, result *preexectypes.TxPreExecResult, executionTime time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Update overall metrics
	m.totalPreExecutions++
	if result.Success {
		m.successfulPreExecs++
	} else {
		m.failedPreExecs++
	}

	// Record execution time
	m.addExecutionTime(executionTime)

	// Record gas usage
	m.addGasUsage(result.TotalGasUsed)

	// Update module metrics
	m.updateModuleMetrics(moduleName, msgType, result, executionTime)
}

// RecordCacheHit records a cache hit event
func (m *PreExecutionMetrics) RecordCacheHit(moduleName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cacheHits++

	if moduleMetrics := m.getOrCreateModuleMetrics(moduleName); moduleMetrics != nil {
		moduleMetrics.CacheHits++
	}
}

// RecordCacheMiss records a cache miss event
func (m *PreExecutionMetrics) RecordCacheMiss(moduleName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cacheMisses++

	if moduleMetrics := m.getOrCreateModuleMetrics(moduleName); moduleMetrics != nil {
		moduleMetrics.CacheMisses++
	}
}

// RecordError records an execution error
func (m *PreExecutionMetrics) RecordError(err *preexectypes.PreExecError) {
	m.mu.Lock()
	defer m.mu.Unlock()

	errorType := fmt.Sprintf("code_%d", err.Code)
	m.errorsByType[errorType]++

	// Record error details
	errorRecord := ErrorRecord{
		Timestamp: time.Now(),
		ErrorType: errorType,
		ErrorCode: err.Code,
		Message:   err.Message,
		TxHash:    err.TxHash,
		Module:    err.Module,
		MsgType:   err.MsgType,
	}

	// Add to recent errors (with size limit)
	if len(m.recentErrors) >= m.errorHistorySize {
		// Remove oldest error
		m.recentErrors = m.recentErrors[1:]
	}
	m.recentErrors = append(m.recentErrors, errorRecord)
}

// RecordMemoryUsage updates current memory usage
func (m *PreExecutionMetrics) RecordMemoryUsage(usage uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.memoryUsage = usage
	if usage > m.maxMemoryUsage {
		m.maxMemoryUsage = usage
	}
}

// RecordConcurrentExecution updates active pre-execution count
func (m *PreExecutionMetrics) RecordConcurrentExecution(active int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.activePreExecutions = active
	if active > m.maxConcurrent {
		m.maxConcurrent = active
	}
}

// addExecutionTime adds an execution time to the history
func (m *PreExecutionMetrics) addExecutionTime(duration time.Duration) {
	// Update min/max
	if duration < m.minExecutionTime {
		m.minExecutionTime = duration
	}
	if duration > m.maxExecutionTime {
		m.maxExecutionTime = duration
	}

	// Add to history (with size limit)
	if len(m.executionTimes) >= m.historySize {
		// Remove oldest entry
		m.executionTimes = m.executionTimes[1:]
	}
	m.executionTimes = append(m.executionTimes, duration)
}

// addGasUsage adds gas usage to the history
func (m *PreExecutionMetrics) addGasUsage(gasUsed uint64) {
	// Add to history (with size limit)
	if len(m.gasUsageHistory) >= m.historySize {
		// Remove oldest entry
		m.gasUsageHistory = m.gasUsageHistory[1:]
	}
	m.gasUsageHistory = append(m.gasUsageHistory, gasUsed)
}

// updateModuleMetrics updates metrics for a specific module
func (m *PreExecutionMetrics) updateModuleMetrics(moduleName, msgType string, result *preexectypes.TxPreExecResult, executionTime time.Duration) {
	moduleMetrics := m.getOrCreateModuleMetrics(moduleName)

	moduleMetrics.PreExecutions++
	if result.Success {
		moduleMetrics.Successes++
	} else {
		moduleMetrics.Failures++
	}

	// Update message type metrics
	moduleMetrics.MsgTypeMetrics[msgType]++

	// Update averages (simple moving average)
	if moduleMetrics.PreExecutions == 1 {
		// First execution
		moduleMetrics.AverageGasUsed = result.TotalGasUsed
		moduleMetrics.AverageExecTime = executionTime
	} else {
		// Update running average
		count := moduleMetrics.PreExecutions
		moduleMetrics.AverageGasUsed = (moduleMetrics.AverageGasUsed*(count-1) + result.TotalGasUsed) / count
		moduleMetrics.AverageExecTime = time.Duration(
			(int64(moduleMetrics.AverageExecTime)*(int64(count)-1) + int64(executionTime)) / int64(count),
		)
	}
}

// getOrCreateModuleMetrics gets or creates module metrics
func (m *PreExecutionMetrics) getOrCreateModuleMetrics(moduleName string) *ModuleMetrics {
	if moduleMetrics, exists := m.moduleMetrics[moduleName]; exists {
		return moduleMetrics
	}

	// Create new module metrics
	moduleMetrics := &ModuleMetrics{
		ModuleName:     moduleName,
		MsgTypeMetrics: make(map[string]uint64),
	}
	m.moduleMetrics[moduleName] = moduleMetrics
	return moduleMetrics
}

// GetOverallStats returns overall system statistics
func (m *PreExecutionMetrics) GetOverallStats() OverallStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	uptime := time.Since(m.startTime)
	timeSinceReset := time.Since(m.lastResetTime)

	stats := OverallStats{
		// Execution stats
		TotalPreExecutions: m.totalPreExecutions,
		SuccessfulPreExecs: m.successfulPreExecs,
		FailedPreExecs:     m.failedPreExecs,
		SuccessRate:        m.calculateSuccessRate(),

		// Cache stats
		CacheHits:    m.cacheHits,
		CacheMisses:  m.cacheMisses,
		CacheHitRate: m.calculateCacheHitRate(),

		// Performance stats
		AverageExecutionTime: m.calculateAverageExecutionTime(),
		MinExecutionTime:     m.minExecutionTime,
		MaxExecutionTime:     m.maxExecutionTime,
		AverageGasUsed:       m.calculateAverageGasUsage(),

		// Resource stats
		CurrentMemoryUsage:  m.memoryUsage,
		MaxMemoryUsage:      m.maxMemoryUsage,
		ActivePreExecutions: m.activePreExecutions,
		MaxConcurrent:       m.maxConcurrent,

		// Time stats
		Uptime:         uptime,
		TimeSinceReset: timeSinceReset,

		// Error stats
		TotalErrors:  m.calculateTotalErrors(),
		ErrorsByType: m.copyErrorsByType(),

		// Throughput
		ExecutionsPerSecond: m.calculateExecutionsPerSecond(timeSinceReset),
	}

	return stats
}

// GetModuleStats returns statistics for all modules
func (m *PreExecutionMetrics) GetModuleStats() map[string]ModuleMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]ModuleMetrics)
	for name, metrics := range m.moduleMetrics {
		// Create a copy to avoid concurrent access issues
		stats[name] = ModuleMetrics{
			ModuleName:      metrics.ModuleName,
			PreExecutions:   metrics.PreExecutions,
			Successes:       metrics.Successes,
			Failures:        metrics.Failures,
			AverageGasUsed:  metrics.AverageGasUsed,
			AverageExecTime: metrics.AverageExecTime,
			CacheHits:       metrics.CacheHits,
			CacheMisses:     metrics.CacheMisses,
			MsgTypeMetrics:  m.copyMsgTypeMetrics(metrics.MsgTypeMetrics),
		}
	}

	return stats
}

// GetRecentErrors returns recent error records
func (m *PreExecutionMetrics) GetRecentErrors() []ErrorRecord {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy
	errors := make([]ErrorRecord, len(m.recentErrors))
	copy(errors, m.recentErrors)
	return errors
}

// Reset clears all metrics (except configuration)
func (m *PreExecutionMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalPreExecutions = 0
	m.successfulPreExecs = 0
	m.failedPreExecs = 0
	m.cacheHits = 0
	m.cacheMisses = 0

	m.executionTimes = m.executionTimes[:0]
	m.gasUsageHistory = m.gasUsageHistory[:0]
	m.maxExecutionTime = 0
	m.minExecutionTime = time.Hour

	m.memoryUsage = 0
	m.maxMemoryUsage = 0
	m.activePreExecutions = 0
	m.maxConcurrent = 0

	m.moduleMetrics = make(map[string]*ModuleMetrics)
	m.errorsByType = make(map[string]uint64)
	m.recentErrors = m.recentErrors[:0]

	m.lastResetTime = time.Now()
}

// Helper calculation methods

func (m *PreExecutionMetrics) calculateSuccessRate() float64 {
	if m.totalPreExecutions == 0 {
		return 0.0
	}
	return float64(m.successfulPreExecs) / float64(m.totalPreExecutions) * 100.0
}

func (m *PreExecutionMetrics) calculateCacheHitRate() float64 {
	total := m.cacheHits + m.cacheMisses
	if total == 0 {
		return 0.0
	}
	return float64(m.cacheHits) / float64(total) * 100.0
}

func (m *PreExecutionMetrics) calculateAverageExecutionTime() time.Duration {
	if len(m.executionTimes) == 0 {
		return 0
	}

	var total time.Duration
	for _, duration := range m.executionTimes {
		total += duration
	}

	return total / time.Duration(len(m.executionTimes))
}

func (m *PreExecutionMetrics) calculateAverageGasUsage() uint64 {
	if len(m.gasUsageHistory) == 0 {
		return 0
	}

	var total uint64
	for _, gas := range m.gasUsageHistory {
		total += gas
	}

	return total / uint64(len(m.gasUsageHistory))
}

func (m *PreExecutionMetrics) calculateTotalErrors() uint64 {
	var total uint64
	for _, count := range m.errorsByType {
		total += count
	}
	return total
}

func (m *PreExecutionMetrics) calculateExecutionsPerSecond(duration time.Duration) float64 {
	if duration == 0 {
		return 0.0
	}

	seconds := duration.Seconds()
	if seconds == 0 {
		return 0.0
	}

	return float64(m.totalPreExecutions) / seconds
}

func (m *PreExecutionMetrics) copyErrorsByType() map[string]uint64 {
	copy := make(map[string]uint64)
	for errorType, count := range m.errorsByType {
		copy[errorType] = count
	}
	return copy
}

func (m *PreExecutionMetrics) copyMsgTypeMetrics(original map[string]uint64) map[string]uint64 {
	copy := make(map[string]uint64)
	for msgType, count := range original {
		copy[msgType] = count
	}
	return copy
}

// OverallStats contains overall system performance statistics
type OverallStats struct {
	// Execution statistics
	TotalPreExecutions uint64  `json:"total_pre_executions"`
	SuccessfulPreExecs uint64  `json:"successful_pre_execs"`
	FailedPreExecs     uint64  `json:"failed_pre_execs"`
	SuccessRate        float64 `json:"success_rate"`

	// Cache statistics
	CacheHits    uint64  `json:"cache_hits"`
	CacheMisses  uint64  `json:"cache_misses"`
	CacheHitRate float64 `json:"cache_hit_rate"`

	// Performance statistics
	AverageExecutionTime time.Duration `json:"average_execution_time"`
	MinExecutionTime     time.Duration `json:"min_execution_time"`
	MaxExecutionTime     time.Duration `json:"max_execution_time"`
	AverageGasUsed       uint64        `json:"average_gas_used"`

	// Resource statistics
	CurrentMemoryUsage  uint64 `json:"current_memory_usage"`
	MaxMemoryUsage      uint64 `json:"max_memory_usage"`
	ActivePreExecutions int    `json:"active_pre_executions"`
	MaxConcurrent       int    `json:"max_concurrent"`

	// Time statistics
	Uptime         time.Duration `json:"uptime"`
	TimeSinceReset time.Duration `json:"time_since_reset"`

	// Error statistics
	TotalErrors  uint64            `json:"total_errors"`
	ErrorsByType map[string]uint64 `json:"errors_by_type"`

	// Throughput
	ExecutionsPerSecond float64 `json:"executions_per_second"`
}

// ToJSON serializes the overall stats to JSON
func (s *OverallStats) ToJSON() ([]byte, error) {
	return json.MarshalIndent(s, "", "  ")
}

// GetFormattedReport returns a human-readable metrics report
func (m *PreExecutionMetrics) GetFormattedReport() string {
	stats := m.GetOverallStats()
	moduleStats := m.GetModuleStats()
	recentErrors := m.GetRecentErrors()

	report := fmt.Sprintf(`
📊 PRE-EXECUTION METRICS REPORT
================================

🔄 EXECUTION STATISTICS:
  • Total Pre-executions: %d
  • Successful: %d (%.1f%%)
  • Failed: %d (%.1f%%)
  • Executions/sec: %.2f

💾 CACHE STATISTICS:
  • Cache Hits: %d
  • Cache Misses: %d
  • Hit Rate: %.1f%%

⚡ PERFORMANCE STATISTICS:
  • Average Execution Time: %v
  • Min Execution Time: %v
  • Max Execution Time: %v
  • Average Gas Used: %d

🖥️  RESOURCE STATISTICS:
  • Current Memory Usage: %d bytes
  • Peak Memory Usage: %d bytes
  • Active Pre-executions: %d
  • Max Concurrent: %d

⏱️  UPTIME STATISTICS:
  • System Uptime: %v
  • Time Since Reset: %v
`,
		stats.TotalPreExecutions,
		stats.SuccessfulPreExecs, stats.SuccessRate,
		stats.FailedPreExecs, 100.0-stats.SuccessRate,
		stats.ExecutionsPerSecond,

		stats.CacheHits,
		stats.CacheMisses,
		stats.CacheHitRate,

		stats.AverageExecutionTime,
		stats.MinExecutionTime,
		stats.MaxExecutionTime,
		stats.AverageGasUsed,

		stats.CurrentMemoryUsage,
		stats.MaxMemoryUsage,
		stats.ActivePreExecutions,
		stats.MaxConcurrent,

		stats.Uptime,
		stats.TimeSinceReset,
	)

	// Add module statistics
	if len(moduleStats) > 0 {
		report += "\n📦 MODULE STATISTICS:\n"
		for moduleName, modStats := range moduleStats {
			report += fmt.Sprintf(`  • %s:
    - Pre-executions: %d (Success: %d, Failed: %d)
    - Average Gas: %d
    - Average Time: %v
    - Cache Hits/Misses: %d/%d
`,
				moduleName,
				modStats.PreExecutions, modStats.Successes, modStats.Failures,
				modStats.AverageGasUsed,
				modStats.AverageExecTime,
				modStats.CacheHits, modStats.CacheMisses,
			)
		}
	}

	// Add recent errors
	if len(recentErrors) > 0 {
		report += fmt.Sprintf("\n❌ RECENT ERRORS (%d):\n", len(recentErrors))
		for i, err := range recentErrors {
			if i >= 5 { // Limit to 5 most recent errors
				break
			}
			report += fmt.Sprintf("  • [%s] %s: %s\n",
				err.Timestamp.Format("15:04:05"),
				err.ErrorType,
				err.Message,
			)
		}
	}

	return report
}
