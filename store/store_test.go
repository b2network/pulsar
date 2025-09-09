package store

import (
	"github.com/b2network/pulsar/store/types"
	"os"
	"testing"
)

func TestStoreBasicOperations(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "pulsar-store-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test the example usage
	if err := ExampleUsage(tempDir); err != nil {
		t.Errorf("Example usage failed: %v", err)
	}
}

func TestPruningOptions(t *testing.T) {
	tests := []struct {
		name     string
		strategy types.PruningStrategy
		current  int64
		version  int64
		expected bool
	}{
		{
			name:     "PruningNothing should never prune",
			strategy: types.PruningNothing,
			current:  1000,
			version:  1,
			expected: false,
		},
		{
			name:     "PruningEverything should prune old versions",
			strategy: types.PruningEverything,
			current:  1000,
			version:  1,
			expected: true,
		},
		{
			name:     "PruningDefault should respect keep recent",
			strategy: types.PruningDefault,
			current:  1000,
			version:  999,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := types.NewPruningOptions(tt.strategy)
			result := options.ShouldPrune(tt.current, tt.version)
			if result != tt.expected {
				t.Errorf("ShouldPrune() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestStoreKeys(t *testing.T) {
	// Test KVStoreKey
	key := types.NewKVStoreKey("test")
	if key.Name() != "test" {
		t.Errorf("Expected name 'test', got '%s'", key.Name())
	}

	expectedStr := "KVStoreKey{test}"
	if key.String() != expectedStr {
		t.Errorf("Expected string '%s', got '%s'", expectedStr, key.String())
	}

	// Test TransientStoreKey
	transientKey := types.NewTransientStoreKey("transient")
	if transientKey.Name() != "transient" {
		t.Errorf("Expected name 'transient', got '%s'", transientKey.Name())
	}

	// Test MemoryStoreKey
	memoryKey := types.NewMemoryStoreKey("memory")
	if memoryKey.Name() != "memory" {
		t.Errorf("Expected name 'memory', got '%s'", memoryKey.Name())
	}
}

func TestCommitID(t *testing.T) {
	// Test zero CommitID
	zeroCommit := types.CommitID{}
	if !zeroCommit.IsZero() {
		t.Error("Zero CommitID should return true for IsZero()")
	}

	// Test non-zero CommitID
	nonZeroCommit := types.CommitID{
		Version: 1,
		Hash:    []byte("hash"),
	}
	if nonZeroCommit.IsZero() {
		t.Error("Non-zero CommitID should return false for IsZero()")
	}

	// Test String method
	str := nonZeroCommit.String()
	expected := "CommitID{1:68617368}"
	if str != expected {
		t.Errorf("Expected string '%s', got '%s'", expected, str)
	}
}
