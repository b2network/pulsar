package keeper

import (
	"fmt"

	keepertypes "github.com/b2network/pulsar/keeper/types"
	"github.com/b2network/pulsar/modules/signal/types"
)

// StoreSignal stores a signal in the KV store
func (k Keeper) StoreSignal(ctx keepertypes.Context, signal types.Signal) error {
	store := k.GetKVStore(ctx)

	// Marshal signal
	bz, err := k.GetCodec().Marshal(&signal)
	if err != nil {
		return fmt.Errorf("failed to marshal signal: %w", err)
	}

	// Store signal by ID
	signalKey := types.GetSignalKey(signal.ID)
	store.Set(signalKey, bz)

	// Index by validator
	if signal.ValidatorAddress != "" {
		validatorSignalKey := types.GetValidatorSignalsKey(signal.ValidatorAddress, signal.ID)
		store.Set(validatorSignalKey, []byte{})
	}

	// Index by time
	timeKey := types.GetSignalByTimeKey(signal.Timestamp, signal.ID)
	store.Set(timeKey, []byte{})

	// Index by expiry time if set
	if signal.ExpiryTime > 0 {
		expiryKey := types.GetSignalExpiryKey(signal.ExpiryTime, signal.ID)
		store.Set(expiryKey, []byte{})
	}

	return nil
}

// GetSignal retrieves a signal by ID
func (k Keeper) GetSignal(ctx keepertypes.Context, signalID string) (types.Signal, bool) {
	store := k.GetKVStore(ctx)
	key := types.GetSignalKey(signalID)

	bz := store.Get(key)
	if bz == nil {
		return types.Signal{}, false
	}

	var signal types.Signal
	if err := k.GetCodec().Unmarshal(bz, &signal); err != nil {
		k.Logger(ctx).Error("failed to unmarshal signal", "error", err, "signal_id", signalID)
		return types.Signal{}, false
	}

	return signal, true
}

// DeleteSignal removes a signal from storage
func (k Keeper) DeleteSignal(ctx keepertypes.Context, signalID string) error {
	signal, found := k.GetSignal(ctx, signalID)
	if !found {
		return types.ErrSignalNotFound
	}

	store := k.GetKVStore(ctx)

	// Remove main signal entry
	signalKey := types.GetSignalKey(signalID)
	store.Delete(signalKey)

	// Remove validator index
	if signal.ValidatorAddress != "" {
		validatorSignalKey := types.GetValidatorSignalsKey(signal.ValidatorAddress, signalID)
		store.Delete(validatorSignalKey)
	}

	// Remove time index
	timeKey := types.GetSignalByTimeKey(signal.Timestamp, signalID)
	store.Delete(timeKey)

	// Remove expiry index
	if signal.ExpiryTime > 0 {
		expiryKey := types.GetSignalExpiryKey(signal.ExpiryTime, signalID)
		store.Delete(expiryKey)
	}

	return nil
}

// GetAllSignals returns all signals in the store
func (k Keeper) GetAllSignals(ctx keepertypes.Context) []types.Signal {
	store := k.GetKVStore(ctx)
	iterator := store.Iterator(types.SignalKey, append(types.SignalKey, 0xFF))
	defer iterator.Close()

	var signals []types.Signal
	for ; iterator.Valid(); iterator.Next() {
		var signal types.Signal
		if err := k.GetCodec().Unmarshal(iterator.Value(), &signal); err != nil {
			k.Logger(ctx).Error("failed to unmarshal signal", "error", err)
			continue
		}
		signals = append(signals, signal)
	}

	return signals
}

// GetValidatorSignals returns all signals for a specific validator
func (k Keeper) GetValidatorSignals(ctx keepertypes.Context, validatorAddr string) []types.Signal {
	store := k.GetKVStore(ctx)
	prefix := append(types.ValidatorSignalsKey, []byte(validatorAddr)...)
	iterator := store.Iterator(prefix, append(prefix, 0xFF))
	defer iterator.Close()

	var signals []types.Signal
	for ; iterator.Valid(); iterator.Next() {
		// Extract signal ID from key
		key := iterator.Key()
		signalID := string(key[len(prefix):])

		signal, found := k.GetSignal(ctx, signalID)
		if found {
			signals = append(signals, signal)
		}
	}

	return signals
}

// GetSignalsByTimeRange returns signals within a time range
func (k Keeper) GetSignalsByTimeRange(ctx keepertypes.Context, startTime, endTime int64) []types.Signal {
	store := k.GetKVStore(ctx)
	startKey := types.GetSignalByTimeKey(startTime, "")
	endKey := types.GetSignalByTimeKey(endTime+1, "") // +1 to make it exclusive

	iterator := store.Iterator(startKey, endKey)
	defer iterator.Close()

	var signals []types.Signal
	for ; iterator.Valid(); iterator.Next() {
		// Extract signal ID from key
		key := iterator.Key()
		timestamp := types.ParseTimestampFromKey(key, types.SignalByTimeKey)

		// The signal ID is after the timestamp (8 bytes)
		if len(key) > len(types.SignalByTimeKey)+8 {
			signalID := string(key[len(types.SignalByTimeKey)+8:])
			signal, found := k.GetSignal(ctx, signalID)
			if found && signal.Timestamp >= startTime && signal.Timestamp <= endTime {
				signals = append(signals, signal)
			}
		}
	}

	return signals
}

// GetSignalsByStatus returns signals with a specific status
func (k Keeper) GetSignalsByStatus(ctx keepertypes.Context, status types.SignalStatus) []types.Signal {
	allSignals := k.GetAllSignals(ctx)
	var filteredSignals []types.Signal

	for _, signal := range allSignals {
		if signal.Status == status {
			filteredSignals = append(filteredSignals, signal)
		}
	}

	return filteredSignals
}

// GetSignalsByWorkType returns signals of a specific work type
func (k Keeper) GetSignalsByWorkType(ctx keepertypes.Context, workType string) []types.Signal {
	allSignals := k.GetAllSignals(ctx)
	var filteredSignals []types.Signal

	for _, signal := range allSignals {
		if signal.WorkProof.WorkType == workType {
			filteredSignals = append(filteredSignals, signal)
		}
	}

	return filteredSignals
}

// GetExpiredSignals returns signals that have expired
func (k Keeper) GetExpiredSignals(ctx keepertypes.Context, currentTime int64) []string {
	store := k.GetKVStore(ctx)
	endKey := types.GetSignalExpiryKey(currentTime, "~") // Use ~ as a high value

	iterator := store.Iterator(types.SignalExpiryKey, endKey)
	defer iterator.Close()

	var expiredSignalIDs []string
	for ; iterator.Valid(); iterator.Next() {
		key := iterator.Key()
		if len(key) > len(types.SignalExpiryKey)+8 {
			signalID := string(key[len(types.SignalExpiryKey)+8:])
			expiredSignalIDs = append(expiredSignalIDs, signalID)
		}
	}

	return expiredSignalIDs
}

// FilterSignals applies filters to signals
func (k Keeper) FilterSignals(ctx keepertypes.Context, filter types.SignalFilter) []types.Signal {
	var signals []types.Signal

	// Start with all signals or use specific indices
	if filter.ValidatorAddress != "" {
		signals = k.GetValidatorSignals(ctx, filter.ValidatorAddress)
	} else if filter.StartTime > 0 || filter.EndTime > 0 {
		startTime := filter.StartTime
		endTime := filter.EndTime
		if startTime == 0 {
			startTime = 0
		}
		if endTime == 0 {
			endTime = ctx.BlockTime().Unix()
		}
		signals = k.GetSignalsByTimeRange(ctx, startTime, endTime)
	} else {
		signals = k.GetAllSignals(ctx)
	}

	// Apply additional filters
	var filteredSignals []types.Signal
	for _, signal := range signals {
		if k.matchesFilter(signal, filter) {
			filteredSignals = append(filteredSignals, signal)
		}
	}

	return filteredSignals
}

// matchesFilter checks if a signal matches the given filter
func (k Keeper) matchesFilter(signal types.Signal, filter types.SignalFilter) bool {
	if filter.Creator != "" && signal.Creator != filter.Creator {
		return false
	}

	if filter.Type != "" && signal.Type != filter.Type {
		return false
	}

	if filter.Status != 0 && signal.Status != filter.Status {
		return false
	}

	if filter.MinScore > 0 && signal.Score < filter.MinScore {
		return false
	}

	if filter.MaxScore > 0 && signal.Score > filter.MaxScore {
		return false
	}

	if filter.StartTime > 0 && signal.Timestamp < filter.StartTime {
		return false
	}

	if filter.EndTime > 0 && signal.Timestamp > filter.EndTime {
		return false
	}

	return true
}

// GetSignalCount returns the total number of signals
func (k Keeper) GetSignalCount(ctx keepertypes.Context) uint64 {
	store := k.GetKVStore(ctx)
	iterator := store.Iterator(types.SignalKey, append(types.SignalKey, 0xFF))
	defer iterator.Close()

	count := uint64(0)
	for ; iterator.Valid(); iterator.Next() {
		count++
	}

	return count
}

// GetSignalCountByValidator returns the number of signals for a validator
func (k Keeper) GetSignalCountByValidator(ctx keepertypes.Context, validatorAddr string) uint64 {
	signals := k.GetValidatorSignals(ctx, validatorAddr)
	return uint64(len(signals))
}

// GetSignalCountByStatus returns the number of signals with a specific status
func (k Keeper) GetSignalCountByStatus(ctx keepertypes.Context, status types.SignalStatus) uint64 {
	signals := k.GetSignalsByStatus(ctx, status)
	return uint64(len(signals))
}

// removeFromActiveSignals removes a signal from validator's active signals list
func (k Keeper) removeFromActiveSignals(ctx keepertypes.Context, validatorAddr, signalID string) {
	stats := k.GetValidatorStats(ctx, validatorAddr)

	// Remove signal from active list
	for i, activeSignalID := range stats.ActiveSignals {
		if activeSignalID == signalID {
			stats.ActiveSignals = append(stats.ActiveSignals[:i], stats.ActiveSignals[i+1:]...)
			break
		}
	}

	// Update stats
	k.SetValidatorStats(ctx, stats)
}

// deleteSignalData deletes all data related to a signal
func (k Keeper) deleteSignalData(ctx keepertypes.Context, signalID string) {
	// Delete signal score
	k.DeleteSignalScore(ctx, signalID)

	// Delete work verifications
	k.deleteWorkVerifications(ctx, signalID)

	// Delete the signal itself
	k.DeleteSignal(ctx, signalID)
}

// deleteWorkVerifications deletes all verifications for a signal
func (k Keeper) deleteWorkVerifications(ctx keepertypes.Context, signalID string) {
	store := k.GetKVStore(ctx)
	prefix := append(types.WorkVerificationKey, []byte(signalID)...)

	iterator := store.Iterator(prefix, append(prefix, 0xFF))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		store.Delete(iterator.Key())
	}
}