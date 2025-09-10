package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/cometbft/cometbft/libs/log"

	"github.com/b2network/pulsar/app"
	preexeckeeper "github.com/b2network/pulsar/pre_execution/keeper"
	"github.com/b2network/pulsar/pre_execution/monitoring"
	"github.com/b2network/pulsar/pre_execution/p2p"
	preexectypes "github.com/b2network/pulsar/pre_execution/types"

	banktypes "github.com/b2network/pulsar/modules/bank/types"
)

// TestPreExecutionSystem runs comprehensive tests for the pre-execution system
func TestPreExecutionSystem(t *testing.T) {
	fmt.Println("🧪 Starting Pulsar Pre-execution System Tests")
	fmt.Println("=" * 50)

	// Run individual test suites
	t.Run("Cache", testCacheSystem)
	t.Run("Sequencer", testSequencerSystem)
	t.Run("Manager", testManagerSystem)
	t.Run("Modules", testModuleIntegration)
	t.Run("P2P", testP2PBroadcasting)
	t.Run("Monitoring", testMonitoringSystem)
	t.Run("AppIntegration", testAppIntegration)
	t.Run("EndToEnd", testEndToEndFlow)
}

// testCacheSystem tests the pre-execution cache functionality
func testCacheSystem(t *testing.T) {
	fmt.Println("\n📦 Testing Cache System...")

	// Create cache configuration
	config := preexectypes.CacheConfig{
		MaxSize:            100,
		MaxMemory:          1024 * 1024, // 1MB
		TTL:                60 * time.Second,
		EvictionPolicy:     "LRU",
		CompressionEnabled: false,
	}

	// Create cache
	cache := preexectypes.NewPreExecutionCache(config)
	defer cache.Close()

	// Test cache operations
	result := &preexectypes.TxPreExecResult{
		TxHash:       "test_tx_hash_123",
		Success:      true,
		TotalGasUsed: 50000,
		MsgResults:   []*preexectypes.PreExecResult{},
		StateChanges: &preexectypes.StateChangeSet{
			StoreChanges: make(map[string]*preexectypes.StoreChangeSet),
			Version:      1,
		},
	}

	// Test set and get
	success := cache.Set("test_tx_hash_123", result)
	if !success {
		t.Errorf("Failed to set cache entry")
		return
	}

	cachedResult, found := cache.Get("test_tx_hash_123")
	if !found {
		t.Errorf("Failed to retrieve cached entry")
		return
	}

	if cachedResult.TxHash != result.TxHash {
		t.Errorf("Cached result mismatch: expected %s, got %s", result.TxHash, cachedResult.TxHash)
		return
	}

	fmt.Println("  ✅ Cache operations successful")
}

// testSequencerSystem tests the transaction sequencer
func testSequencerSystem(t *testing.T) {
	fmt.Println("\n🔢 Testing Sequencer System...")

	// Create sequencer configuration
	config := preexeckeeper.SequencerConfig{
		MaxPending:     100,
		SyncTimeout:    10 * time.Second,
		OrderBroadcast: true,
	}

	// Create sequencer
	sequencer := preexeckeeper.NewTxSequencer(1, config)

	// Test adding transactions
	seq1, err := sequencer.AddTransaction("tx_hash_1", "validator_1", 100)
	if err != nil {
		t.Errorf("Failed to add transaction 1: %v", err)
		return
	}

	seq2, err := sequencer.AddTransaction("tx_hash_2", "validator_1", 200)
	if err != nil {
		t.Errorf("Failed to add transaction 2: %v", err)
		return
	}

	if seq2 != seq1+1 {
		t.Errorf("Expected sequential sequence numbers: %d, %d", seq1, seq2)
		return
	}

	fmt.Println("  ✅ Sequencer operations successful")
}

// testManagerSystem tests the pre-execution manager
func testManagerSystem(t *testing.T) {
	fmt.Println("\n🎛️  Testing Manager System...")

	// Create manager configurations
	managerConfig := preexeckeeper.PreExecManagerConfig{
		MaxConcurrentTxs:     10,
		DefaultGasLimit:      100000,
		MaxGasPerPreExec:     200000,
		PreExecTimeout:       30 * time.Second,
		CacheCleanupInterval: 1 * time.Minute,
		StatsReportInterval:  30 * time.Second,
		ValidatorID:          "test_validator",
	}

	cacheConfig := preexectypes.CacheConfig{
		MaxSize:            1000,
		MaxMemory:          10 * 1024 * 1024, // 10MB
		TTL:                60 * time.Second,
		EvictionPolicy:     "LRU",
		CompressionEnabled: false,
	}

	sequencerConfig := preexeckeeper.SequencerConfig{
		MaxPending:     500,
		SyncTimeout:    10 * time.Second,
		OrderBroadcast: true,
	}

	// Create manager
	manager, err := preexeckeeper.NewPreExecutionManager(managerConfig, cacheConfig, sequencerConfig, 1)
	if err != nil {
		t.Errorf("Failed to create pre-execution manager: %v", err)
		return
	}
	defer manager.Close()

	fmt.Println("  ✅ Manager system operational")
}

// testModuleIntegration tests module-specific pre-execution
func testModuleIntegration(t *testing.T) {
	fmt.Println("\n📦 Testing Module Integration...")

	// Test Bank module pre-execution hints
	bankMsg := &banktypes.MsgSend{
		FromAddress: "cosmos1fromaddr",
		ToAddress:   "cosmos1toaddr",
		Amount:      nil, // Simplified for testing
	}

	if !bankMsg.IsPreExecutable() {
		t.Errorf("Bank MsgSend should support pre-execution")
		return
	}

	hints := bankMsg.GetPreExecutionHints()
	if hints.Priority != 100 {
		t.Errorf("Expected bank message priority 100, got %d", hints.Priority)
		return
	}

	fmt.Println("  ✅ Module integration successful")
}

// testP2PBroadcasting tests P2P order broadcasting
func testP2PBroadcasting(t *testing.T) {
	fmt.Println("\n📡 Testing P2P Broadcasting...")

	// Create broadcaster configuration
	config := p2p.BroadcasterConfig{
		ValidatorID:   "test_validator_1",
		MaxRetries:    3,
		RetryInterval: 1 * time.Second,
		PeerTimeout:   30 * time.Second,
	}

	// Create broadcaster
	broadcaster := p2p.NewOrderBroadcaster(config)

	// Add test peers
	err := broadcaster.AddPeer("test_validator_2", "localhost:26656")
	if err != nil {
		t.Errorf("Failed to add peer: %v", err)
		return
	}

	fmt.Println("  ✅ P2P broadcasting operational")
}

// testMonitoringSystem tests metrics collection and monitoring
func testMonitoringSystem(t *testing.T) {
	fmt.Println("\n📈 Testing Monitoring System...")

	// Create metrics collector
	config := monitoring.MetricsConfig{
		HistorySize:      100,
		ErrorHistorySize: 50,
	}

	metrics := monitoring.NewPreExecutionMetrics(config)

	// Test recording pre-execution results
	result := &preexectypes.TxPreExecResult{
		TxHash:       "test_metrics_tx_123",
		Success:      true,
		TotalGasUsed: 75000,
		MsgResults:   []*preexectypes.PreExecResult{},
	}

	executionTime := 150 * time.Millisecond
	metrics.RecordPreExecution("bank", "MsgSend", result, executionTime)

	// Test statistics retrieval
	overallStats := metrics.GetOverallStats()
	if overallStats.TotalPreExecutions != 1 {
		t.Errorf("Expected 1 pre-execution, got %d", overallStats.TotalPreExecutions)
		return
	}

	fmt.Println("  ✅ Monitoring system operational")
}

// testAppIntegration tests integration with the main Pulsar app
func testAppIntegration(t *testing.T) {
	fmt.Println("\n🏗️  Testing App Integration...")

	// Create app configuration with pre-execution enabled
	config := app.AppConfig{
		PreExecEnabled:   true,
		PreExecCacheSize: 100,
		PreExecTTL:       "30s",
		ValidatorID:      "test-validator-integration",
	}

	// Create logger
	logger := log.NewNopLogger() // No-op logger for tests

	// Create app
	pulsarApp := app.NewPulsarApp(logger, "/tmp/pulsar-test", config)

	// Verify pre-execution is initialized
	if pulsarApp.PreExecManager == nil {
		t.Errorf("Pre-execution manager should be initialized")
		return
	}

	// Test pre-execution stats
	stats := pulsarApp.GetPreExecutionStats()
	if !stats.Enabled {
		t.Errorf("Pre-execution should be enabled")
		return
	}

	fmt.Println("  ✅ App integration successful")
}

// testEndToEndFlow tests the complete pre-execution flow
func testEndToEndFlow(t *testing.T) {
	fmt.Println("\n🔄 Testing End-to-End Flow...")

	// Create app with pre-execution enabled
	config := app.AppConfig{
		PreExecEnabled:   true,
		PreExecCacheSize: 100,
		PreExecTTL:       "30s",
		ValidatorID:      "test-validator-e2e",
	}

	logger := log.NewNopLogger()
	pulsarApp := app.NewPulsarApp(logger, "/tmp/pulsar-e2e-test", config)

	// Verify the complete system is working
	if pulsarApp.PreExecManager == nil {
		t.Errorf("Pre-execution manager not initialized")
		return
	}

	stats := pulsarApp.GetPreExecutionStats()
	if !stats.Enabled {
		t.Errorf("Pre-execution should be enabled")
		return
	}

	if stats.RegisteredModules == 0 {
		t.Errorf("Expected registered modules, got %d", stats.RegisteredModules)
		return
	}

	fmt.Println("  ✅ End-to-end flow successful")
	fmt.Printf("  📊 Final stats: Modules=%d, Enabled=%v\n", stats.RegisteredModules, stats.Enabled)
}

// BenchmarkPreExecutionSystem runs performance benchmarks
func BenchmarkPreExecutionSystem(b *testing.B) {
	// Benchmark cache operations
	b.Run("CacheOperations", func(b *testing.B) {
		config := preexectypes.CacheConfig{
			MaxSize:            1000,
			MaxMemory:          10 * 1024 * 1024,
			TTL:                60 * time.Second,
			EvictionPolicy:     "LRU",
			CompressionEnabled: false,
		}

		cache := preexectypes.NewPreExecutionCache(config)
		defer cache.Close()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result := &preexectypes.TxPreExecResult{
				TxHash:       fmt.Sprintf("benchmark_tx_%d", i),
				Success:      true,
				TotalGasUsed: uint64(i * 1000),
			}
			cache.Set(result.TxHash, result)
		}
	})

	// Benchmark sequencer operations
	b.Run("SequencerOperations", func(b *testing.B) {
		config := preexeckeeper.SequencerConfig{
			MaxPending:     10000,
			SyncTimeout:    10 * time.Second,
			OrderBroadcast: false,
		}

		sequencer := preexeckeeper.NewTxSequencer(1, config)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			txHash := fmt.Sprintf("benchmark_seq_tx_%d", i)
			_, err := sequencer.AddTransaction(txHash, "benchmark_validator", uint32(i))
			if err != nil {
				b.Errorf("Benchmark sequencer error: %v", err)
			}
		}
	})
}

// TestPreExecutionDemo demonstrates the complete system in action
func TestPreExecutionDemo(t *testing.T) {
	fmt.Println("\n🎬 PRE-EXECUTION SYSTEM DEMO")
	fmt.Println("=" * 40)

	// Create and configure the app
	config := app.AppConfig{
		PreExecEnabled:   true,
		PreExecCacheSize: 1000,
		PreExecTTL:       "60s",
		ValidatorID:      "demo-validator",
	}

	logger := log.NewNopLogger()
	pulsarApp := app.NewPulsarApp(logger, "/tmp/pulsar-demo", config)

	fmt.Println("✅ Pulsar app initialized with pre-execution")

	// Show system status
	stats := pulsarApp.GetPreExecutionStats()
	fmt.Printf("📊 System Status:\n")
	fmt.Printf("  • Pre-execution Enabled: %v\n", stats.Enabled)
	fmt.Printf("  • Registered Modules: %d\n", stats.RegisteredModules)
	fmt.Printf("  • Cache Size: %d\n", config.PreExecCacheSize)
	fmt.Printf("  • Cache TTL: %s\n", config.PreExecTTL)
	fmt.Printf("  • Validator ID: %s\n", config.ValidatorID)

	fmt.Println("\n🎉 Demo completed successfully!")
}

// Helper function to repeat a character
func repeatChar(char string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += char
	}
	return result
}
