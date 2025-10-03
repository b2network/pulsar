package types

import (
	"encoding/binary"
)

const (
	// ModuleName defines the module name
	ModuleName = "signal"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName

	// QuerierRoute defines the module's query routing key
	QuerierRoute = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_signal"
)

// Store key prefixes
var (
	// SignalKey is the prefix for signal storage
	SignalKey = []byte{0x01}

	// ValidatorSignalsKey is the prefix for validator signals index
	ValidatorSignalsKey = []byte{0x02}

	// SignalScoreKey is the prefix for signal scores
	SignalScoreKey = []byte{0x03}

	// WorkTypeKey is the prefix for AI work types
	WorkTypeKey = []byte{0x04}

	// SignalByTimeKey is the prefix for time-based signal index
	SignalByTimeKey = []byte{0x05}

	// ValidatorStatsKey is the prefix for validator statistics
	ValidatorStatsKey = []byte{0x06}

	// WorkVerificationKey is the prefix for work verifications
	WorkVerificationKey = []byte{0x07}

	// SignalBatchKey is the prefix for signal batches
	SignalBatchKey = []byte{0x08}

	// ParamsKey is the prefix for module parameters
	ParamsKey = []byte{0x09}

	// SignalSequenceKey is the prefix for signal sequence counter
	SignalSequenceKey = []byte{0x0A}

	// RewardPoolKey is the prefix for reward pool
	RewardPoolKey = []byte{0x0B}

	// LeaderboardKey is the prefix for signal leaderboard
	LeaderboardKey = []byte{0x0C}

	// WorkRegistryKey is the prefix for AI work registry
	WorkRegistryKey = []byte{0x0D}

	// SignalExpiryKey is the prefix for signal expiry index
	SignalExpiryKey = []byte{0x0E}

	// ValidatorCooldownKey is the prefix for validator cooldown tracking
	ValidatorCooldownKey = []byte{0x0F}
)

// GetSignalKey returns the store key for a signal
func GetSignalKey(signalID string) []byte {
	return append(SignalKey, []byte(signalID)...)
}

// GetValidatorSignalsKey returns the store key for validator signals
func GetValidatorSignalsKey(validatorAddr string, signalID string) []byte {
	return append(append(ValidatorSignalsKey, []byte(validatorAddr)...), []byte(signalID)...)
}

// GetSignalScoreKey returns the store key for a signal score
func GetSignalScoreKey(signalID string) []byte {
	return append(SignalScoreKey, []byte(signalID)...)
}

// GetWorkTypeKey returns the store key for a work type
func GetWorkTypeKey(workTypeID string) []byte {
	return append(WorkTypeKey, []byte(workTypeID)...)
}

// GetSignalByTimeKey returns the store key for time-based signal index
func GetSignalByTimeKey(timestamp int64, signalID string) []byte {
	timeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timeBytes, uint64(timestamp))
	return append(append(SignalByTimeKey, timeBytes...), []byte(signalID)...)
}

// GetValidatorStatsKey returns the store key for validator statistics
func GetValidatorStatsKey(validatorAddr string) []byte {
	return append(ValidatorStatsKey, []byte(validatorAddr)...)
}

// GetWorkVerificationKey returns the store key for work verification
func GetWorkVerificationKey(signalID string, verifierAddr string) []byte {
	return append(append(WorkVerificationKey, []byte(signalID)...), []byte(verifierAddr)...)
}

// GetSignalBatchKey returns the store key for a signal batch
func GetSignalBatchKey(batchID string) []byte {
	return append(SignalBatchKey, []byte(batchID)...)
}

// GetSignalExpiryKey returns the store key for signal expiry index
func GetSignalExpiryKey(expiryTime int64, signalID string) []byte {
	timeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timeBytes, uint64(expiryTime))
	return append(append(SignalExpiryKey, timeBytes...), []byte(signalID)...)
}

// GetValidatorCooldownKey returns the store key for validator cooldown
func GetValidatorCooldownKey(validatorAddr string) []byte {
	return append(ValidatorCooldownKey, []byte(validatorAddr)...)
}

// ParseSignalIDFromKey extracts signal ID from a composite key
func ParseSignalIDFromKey(key []byte, prefix []byte) string {
	if len(key) <= len(prefix) {
		return ""
	}
	return string(key[len(prefix):])
}

// ParseValidatorAddrFromKey extracts validator address from a composite key
func ParseValidatorAddrFromKey(key []byte, prefix []byte) string {
	if len(key) <= len(prefix) {
		return ""
	}
	// Assuming validator address is fixed length or delimited
	// This is a simplified version - adjust based on actual address format
	return string(key[len(prefix):])
}

// ParseTimestampFromKey extracts timestamp from a time-indexed key
func ParseTimestampFromKey(key []byte, prefix []byte) int64 {
	if len(key) < len(prefix)+8 {
		return 0
	}
	return int64(binary.BigEndian.Uint64(key[len(prefix) : len(prefix)+8]))
}