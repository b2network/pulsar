package keeper

import (
	"fmt"

	keepertypes "github.com/b2network/pulsar/keeper/types"
	"github.com/b2network/pulsar/modules/signal/types"
)

// StoreWorkType stores an AI work type
func (k Keeper) StoreWorkType(ctx keepertypes.Context, workType types.AIWorkType) error {
	if err := workType.ValidateBasic(); err != nil {
		return fmt.Errorf("invalid work type: %w", err)
	}

	store := k.GetKVStore(ctx)
	key := types.GetWorkTypeKey(workType.ID)

	bz, err := k.GetCodec().Marshal(&workType)
	if err != nil {
		return fmt.Errorf("failed to marshal work type: %w", err)
	}

	store.Set(key, bz)

	// Update work registry
	if err := k.updateWorkRegistry(ctx, workType); err != nil {
		k.Logger(ctx).Error("failed to update work registry", "error", err)
		// Don't fail the whole operation for registry update failure
	}

	// Emit event
	ctx.EventManager().EmitEvent(keepertypes.Event{
		Type: types.EventTypeWorkTypeUpdated,
		Attributes: []keepertypes.Attribute{
			{Key: "work_type_id", Value: workType.ID},
			{Key: "enabled", Value: fmt.Sprintf("%t", workType.Enabled)},
		},
	})

	return nil
}

// GetWorkType retrieves an AI work type by ID
func (k Keeper) GetWorkType(ctx keepertypes.Context, workTypeID string) (types.AIWorkType, bool) {
	store := k.GetKVStore(ctx)
	key := types.GetWorkTypeKey(workTypeID)

	bz := store.Get(key)
	if bz == nil {
		return types.AIWorkType{}, false
	}

	var workType types.AIWorkType
	if err := k.GetCodec().Unmarshal(bz, &workType); err != nil {
		k.Logger(ctx).Error("failed to unmarshal work type", "error", err, "work_type_id", workTypeID)
		return types.AIWorkType{}, false
	}

	return workType, true
}

// GetAllWorkTypes returns all registered work types
func (k Keeper) GetAllWorkTypes(ctx keepertypes.Context) []types.AIWorkType {
	store := k.GetKVStore(ctx)
	iterator := store.Iterator(types.WorkTypeKey, append(types.WorkTypeKey, 0xFF))
	defer iterator.Close()

	var workTypes []types.AIWorkType
	for ; iterator.Valid(); iterator.Next() {
		var workType types.AIWorkType
		if err := k.GetCodec().Unmarshal(iterator.Value(), &workType); err != nil {
			k.Logger(ctx).Error("failed to unmarshal work type", "error", err)
			continue
		}
		workTypes = append(workTypes, workType)
	}

	return workTypes
}

// GetEnabledWorkTypes returns only enabled work types
func (k Keeper) GetEnabledWorkTypes(ctx keepertypes.Context) []types.AIWorkType {
	allWorkTypes := k.GetAllWorkTypes(ctx)
	var enabledWorkTypes []types.AIWorkType

	for _, workType := range allWorkTypes {
		if workType.Enabled {
			enabledWorkTypes = append(enabledWorkTypes, workType)
		}
	}

	return enabledWorkTypes
}

// DeleteWorkType removes a work type (admin only)
func (k Keeper) DeleteWorkType(ctx keepertypes.Context, workTypeID string) error {
	store := k.GetKVStore(ctx)
	key := types.GetWorkTypeKey(workTypeID)

	if !store.Has(key) {
		return types.ErrWorkTypeNotFound
	}

	store.Delete(key)

	// Remove from registry
	if err := k.removeFromWorkRegistry(ctx, workTypeID); err != nil {
		k.Logger(ctx).Error("failed to remove from work registry", "error", err)
	}

	return nil
}

// StoreWorkVerification stores a work verification
func (k Keeper) StoreWorkVerification(ctx keepertypes.Context, verification types.WorkVerification) error {
	store := k.GetKVStore(ctx)
	key := types.GetWorkVerificationKey(verification.SignalID, verification.VerifierAddress)

	bz, err := k.GetCodec().Marshal(&verification)
	if err != nil {
		return fmt.Errorf("failed to marshal work verification: %w", err)
	}

	store.Set(key, bz)

	// Emit verification event
	types.EmitSignalEvent(ctx, types.NewWorkVerifiedEvent(verification))

	return nil
}

// GetWorkVerification retrieves a work verification
func (k Keeper) GetWorkVerification(ctx keepertypes.Context, signalID, verifierAddr string) (types.WorkVerification, bool) {
	store := k.GetKVStore(ctx)
	key := types.GetWorkVerificationKey(signalID, verifierAddr)

	bz := store.Get(key)
	if bz == nil {
		return types.WorkVerification{}, false
	}

	var verification types.WorkVerification
	if err := k.GetCodec().Unmarshal(bz, &verification); err != nil {
		k.Logger(ctx).Error("failed to unmarshal work verification", "error", err)
		return types.WorkVerification{}, false
	}

	return verification, true
}

// GetWorkVerifications returns all verifications for a signal
func (k Keeper) GetWorkVerifications(ctx keepertypes.Context, signalID string) []types.WorkVerification {
	store := k.GetKVStore(ctx)
	prefix := append(types.WorkVerificationKey, []byte(signalID)...)

	iterator := store.Iterator(prefix, append(prefix, 0xFF))
	defer iterator.Close()

	var verifications []types.WorkVerification
	for ; iterator.Valid(); iterator.Next() {
		var verification types.WorkVerification
		if err := k.GetCodec().Unmarshal(iterator.Value(), &verification); err != nil {
			k.Logger(ctx).Error("failed to unmarshal work verification", "error", err)
			continue
		}
		verifications = append(verifications, verification)
	}

	return verifications
}

// GetWorkRegistry retrieves the AI work registry
func (k Keeper) GetWorkRegistry(ctx keepertypes.Context) (*types.AIWorkRegistry, bool) {
	store := k.GetKVStore(ctx)
	bz := store.Get(types.WorkRegistryKey)

	if bz == nil {
		return nil, false
	}

	var registry types.AIWorkRegistry
	if err := k.GetCodec().Unmarshal(bz, &registry); err != nil {
		k.Logger(ctx).Error("failed to unmarshal work registry", "error", err)
		return nil, false
	}

	return &registry, true
}

// SetWorkRegistry stores the AI work registry
func (k Keeper) SetWorkRegistry(ctx keepertypes.Context, registry *types.AIWorkRegistry) error {
	store := k.GetKVStore(ctx)

	bz, err := k.GetCodec().Marshal(registry)
	if err != nil {
		return fmt.Errorf("failed to marshal work registry: %w", err)
	}

	store.Set(types.WorkRegistryKey, bz)
	return nil
}

// InitializeWorkRegistry initializes the work registry with default work types
func (k Keeper) InitializeWorkRegistry(ctx keepertypes.Context) error {
	// Check if registry already exists
	if _, exists := k.GetWorkRegistry(ctx); exists {
		return nil // Already initialized
	}

	// Create new registry
	registry := types.NewAIWorkRegistry()

	// Store all default work types
	for _, workType := range types.DefaultAIWorkTypes() {
		if err := k.StoreWorkType(ctx, workType); err != nil {
			return fmt.Errorf("failed to store default work type %s: %w", workType.ID, err)
		}
	}

	// Store registry
	return k.SetWorkRegistry(ctx, registry)
}

// updateWorkRegistry updates the work registry with a new/updated work type
func (k Keeper) updateWorkRegistry(ctx keepertypes.Context, workType types.AIWorkType) error {
	registry, exists := k.GetWorkRegistry(ctx)
	if !exists {
		registry = types.NewAIWorkRegistry()
	}

	// Add or update work type in registry
	if err := registry.RegisterWorkType(workType); err != nil {
		return err
	}

	return k.SetWorkRegistry(ctx, registry)
}

// removeFromWorkRegistry removes a work type from the registry
func (k Keeper) removeFromWorkRegistry(ctx keepertypes.Context, workTypeID string) error {
	registry, exists := k.GetWorkRegistry(ctx)
	if !exists {
		return nil // Nothing to remove
	}

	// Remove from registry
	delete(registry.WorkTypes, workTypeID)
	registry.Version++

	return k.SetWorkRegistry(ctx, registry)
}

// ValidateWorkProofAdvanced performs advanced validation of work proof
func (k Keeper) ValidateWorkProofAdvanced(ctx keepertypes.Context, signal types.Signal) error {
	workType, found := k.GetWorkType(ctx, signal.WorkProof.WorkType)
	if !found {
		return types.ErrWorkTypeNotFound
	}

	// Check if all required proofs are present
	if err := k.validateRequiredProofs(signal.WorkProof, workType.RequiredProofs); err != nil {
		return err
	}

	// Validate resources are within bounds
	if !workType.CheckResourcesInBounds(signal.WorkProof.ResourcesUsed) {
		return types.ErrResourcesOutOfBounds
	}

	// Validate computation time is reasonable
	if err := k.validateComputationTime(signal.WorkProof, workType); err != nil {
		return err
	}

	return nil
}

// validateRequiredProofs checks if all required proofs are present
func (k Keeper) validateRequiredProofs(workProof types.WorkProof, requiredProofs []string) error {
	// This is a simplified implementation
	// In reality, you would check specific proof fields based on requirements

	for _, requiredProof := range requiredProofs {
		switch requiredProof {
		case "input_hash":
			if len(workProof.InputHash) == 0 {
				return fmt.Errorf("missing required proof: input_hash")
			}
		case "output_hash":
			if len(workProof.OutputHash) == 0 {
				return fmt.Errorf("missing required proof: output_hash")
			}
		case "verification_key":
			if len(workProof.VerificationKey) == 0 {
				return fmt.Errorf("missing required proof: verification_key")
			}
		default:
			// For other proof types, assume they're present
			// In a real implementation, you would have specific validation logic
		}
	}

	return nil
}

// validateComputationTime validates if computation time is reasonable
func (k Keeper) validateComputationTime(workProof types.WorkProof, workType types.AIWorkType) error {
	// Simple validation: check if computation time is not suspiciously low or high
	minExpectedTime := workType.MinResources.GetTotalComputationUnits() / 1000 // ms
	maxExpectedTime := workType.MaxResources.GetTotalComputationUnits() / 100  // ms

	if workProof.ComputationTime < int64(minExpectedTime) {
		return fmt.Errorf("computation time too low: %d ms (min expected: %d ms)",
			workProof.ComputationTime, minExpectedTime)
	}

	if workProof.ComputationTime > int64(maxExpectedTime) {
		return fmt.Errorf("computation time too high: %d ms (max expected: %d ms)",
			workProof.ComputationTime, maxExpectedTime)
	}

	return nil
}

// GetWorkTypeByCategory returns work types in a specific category
func (k Keeper) GetWorkTypeByCategory(ctx keepertypes.Context, category string) []types.AIWorkType {
	allWorkTypes := k.GetAllWorkTypes(ctx)
	var filteredWorkTypes []types.AIWorkType

	for _, workType := range allWorkTypes {
		if workType.Category == category {
			filteredWorkTypes = append(filteredWorkTypes, workType)
		}
	}

	return filteredWorkTypes
}

// UpdateWorkTypeStatus enables or disables a work type
func (k Keeper) UpdateWorkTypeStatus(ctx keepertypes.Context, workTypeID string, enabled bool) error {
	workType, found := k.GetWorkType(ctx, workTypeID)
	if !found {
		return types.ErrWorkTypeNotFound
	}

	workType.Enabled = enabled
	return k.StoreWorkType(ctx, workType)
}

// GetWorkTypeStats returns statistics about work type usage
func (k Keeper) GetWorkTypeStats(ctx keepertypes.Context, workTypeID string) (map[string]interface{}, error) {
	// Get all signals of this work type
	signals := k.GetSignalsByWorkType(ctx, workTypeID)

	stats := map[string]interface{}{
		"total_signals":     len(signals),
		"validated_signals": 0,
		"rejected_signals":  0,
		"pending_signals":   0,
		"total_score":       uint64(0),
		"average_score":     uint64(0),
	}

	validatedCount := 0
	totalScore := uint64(0)

	for _, signal := range signals {
		switch signal.Status {
		case types.SignalStatusValidated:
			validatedCount++
			totalScore += signal.Score
		case types.SignalStatusRejected:
			stats["rejected_signals"] = stats["rejected_signals"].(int) + 1
		case types.SignalStatusPending:
			stats["pending_signals"] = stats["pending_signals"].(int) + 1
		}
	}

	stats["validated_signals"] = validatedCount
	stats["total_score"] = totalScore

	if validatedCount > 0 {
		stats["average_score"] = totalScore / uint64(validatedCount)
	}

	return stats, nil
}