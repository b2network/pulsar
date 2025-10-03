package types

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	keepertypes "github.com/b2network/pulsar/keeper/types"
)

// SignalStatus represents the status of a signal
type SignalStatus int32

const (
	SignalStatusPending SignalStatus = iota
	SignalStatusValidated
	SignalStatusRejected
	SignalStatusExpired
)

// String returns the string representation of SignalStatus
func (s SignalStatus) String() string {
	switch s {
	case SignalStatusPending:
		return "pending"
	case SignalStatusValidated:
		return "validated"
	case SignalStatusRejected:
		return "rejected"
	case SignalStatusExpired:
		return "expired"
	default:
		return "unknown"
	}
}

// Signal represents an AI computation signal in the network
type Signal struct {
	ID               string       `json:"id"`
	Creator          string       `json:"creator"`
	Type             string       `json:"type"`
	PayloadHash      []byte       `json:"payload_hash"`
	WorkProof        WorkProof    `json:"work_proof"`
	Timestamp        int64        `json:"timestamp"`
	Signature        []byte       `json:"signature"`
	Score            uint64       `json:"score"`
	Status           SignalStatus `json:"status"`
	ValidatorAddress string       `json:"validator_address,omitempty"`
	BlockHeight      int64        `json:"block_height"`
	ExpiryTime       int64        `json:"expiry_time"`
}

// WorkProof represents the proof of AI work performed
type WorkProof struct {
	WorkType        string    `json:"work_type"`
	InputHash       []byte    `json:"input_hash"`
	OutputHash      []byte    `json:"output_hash"`
	ComputationTime int64     `json:"computation_time"` // in milliseconds
	ResourcesUsed   Resources `json:"resources_used"`
	VerificationKey []byte    `json:"verification_key"`
	Nonce           uint64    `json:"nonce"`
}

// Resources represents computational resources used
type Resources struct {
	CPUCycles        uint64 `json:"cpu_cycles"`
	MemoryMB         uint64 `json:"memory_mb"`
	GPUTimeMs        uint64 `json:"gpu_time_ms"`
	NetworkBandwidth uint64 `json:"network_bandwidth"` // in bytes
	StorageBytes     uint64 `json:"storage_bytes"`
}

// NewSignal creates a new signal instance
func NewSignal(
	creator string,
	signalType string,
	payload []byte,
	workProof WorkProof,
	validatorAddr string,
) *Signal {
	now := time.Now().Unix()
	id := GenerateSignalID(creator, signalType, now)

	return &Signal{
		ID:               id,
		Creator:          creator,
		Type:             signalType,
		PayloadHash:      HashPayload(payload),
		WorkProof:        workProof,
		Timestamp:        now,
		Status:           SignalStatusPending,
		ValidatorAddress: validatorAddr,
		Score:            0, // Will be calculated during validation
	}
}

// GenerateSignalID generates a unique ID for a signal
func GenerateSignalID(creator string, signalType string, timestamp int64) string {
	data := fmt.Sprintf("%s-%s-%d", creator, signalType, timestamp)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])[:16] // Use first 16 bytes for ID
}

// HashPayload generates a hash of the payload data
func HashPayload(payload []byte) []byte {
	hash := sha256.Sum256(payload)
	return hash[:]
}

// ValidateBasic performs basic validation on the signal
func (s Signal) ValidateBasic() error {
	if s.ID == "" {
		return fmt.Errorf("signal ID cannot be empty")
	}
	if s.Creator == "" {
		return fmt.Errorf("signal creator cannot be empty")
	}
	if s.Type == "" {
		return fmt.Errorf("signal type cannot be empty")
	}
	if len(s.PayloadHash) != 32 {
		return fmt.Errorf("invalid payload hash length: expected 32, got %d", len(s.PayloadHash))
	}
	if s.Timestamp <= 0 {
		return fmt.Errorf("invalid timestamp: %d", s.Timestamp)
	}
	if err := s.WorkProof.ValidateBasic(); err != nil {
		return fmt.Errorf("invalid work proof: %w", err)
	}
	return nil
}

// ValidateBasic performs basic validation on the work proof
func (wp WorkProof) ValidateBasic() error {
	if wp.WorkType == "" {
		return fmt.Errorf("work type cannot be empty")
	}
	if len(wp.InputHash) != 32 {
		return fmt.Errorf("invalid input hash length: expected 32, got %d", len(wp.InputHash))
	}
	if len(wp.OutputHash) != 32 {
		return fmt.Errorf("invalid output hash length: expected 32, got %d", len(wp.OutputHash))
	}
	if wp.ComputationTime <= 0 {
		return fmt.Errorf("computation time must be positive")
	}
	if err := wp.ResourcesUsed.ValidateBasic(); err != nil {
		return fmt.Errorf("invalid resources: %w", err)
	}
	return nil
}

// ValidateBasic performs basic validation on resources
func (r Resources) ValidateBasic() error {
	if r.CPUCycles == 0 && r.GPUTimeMs == 0 {
		return fmt.Errorf("at least one computation resource (CPU or GPU) must be used")
	}
	if r.MemoryMB == 0 {
		return fmt.Errorf("memory usage cannot be zero")
	}
	return nil
}

// IsExpired checks if the signal has expired
func (s Signal) IsExpired(currentTime int64) bool {
	return s.ExpiryTime > 0 && currentTime > s.ExpiryTime
}

// CalculateResourceScore calculates a score based on resources used
func (r Resources) CalculateResourceScore() uint64 {
	// Simple scoring formula - can be made more sophisticated
	cpuScore := r.CPUCycles / 1000000       // Normalize CPU cycles
	gpuScore := r.GPUTimeMs * 10            // GPU time is more valuable
	memoryScore := r.MemoryMB / 100         // Memory in hundreds of MB
	networkScore := r.NetworkBandwidth / 1048576 // Network in MB

	totalScore := cpuScore + gpuScore + memoryScore + networkScore
	if totalScore > ^uint64(0)/2 { // Prevent overflow
		return ^uint64(0) / 2
	}
	return totalScore
}

// GetTotalComputationUnits returns the total computation units used
func (r Resources) GetTotalComputationUnits() uint64 {
	// Weighted sum of resources
	return r.CPUCycles/1000 + r.GPUTimeMs*100 + r.MemoryMB*10
}

// SignalBatch represents a batch of signals for efficient processing
type SignalBatch struct {
	Signals      []Signal `json:"signals"`
	BatchID      string   `json:"batch_id"`
	ProcessedAt  int64    `json:"processed_at"`
	TotalScore   uint64   `json:"total_score"`
}

// SignalFilter represents filter criteria for querying signals
type SignalFilter struct {
	Creator          string       `json:"creator,omitempty"`
	ValidatorAddress string       `json:"validator_address,omitempty"`
	Type             string       `json:"type,omitempty"`
	Status           SignalStatus `json:"status,omitempty"`
	MinScore         uint64       `json:"min_score,omitempty"`
	MaxScore         uint64       `json:"max_score,omitempty"`
	StartTime        int64        `json:"start_time,omitempty"`
	EndTime          int64        `json:"end_time,omitempty"`
}