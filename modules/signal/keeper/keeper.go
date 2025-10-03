package keeper

import (
	"fmt"
	"time"

	"github.com/b2network/pulsar/keeper/base"
	keepertypes "github.com/b2network/pulsar/keeper/types"
	"github.com/b2network/pulsar/modules/signal/types"
	storetypes "github.com/b2network/pulsar/store/types"
)

// Keeper implements the signal keeper
type Keeper struct {
	*base.KVStoreKeeper

	// Dependencies
	bankKeeper         types.BankKeeper
	stakingKeeper      types.StakingKeeper
	distributionKeeper types.DistributionKeeper

	// Module permissions and parameters
	authority string // authority capable of executing gov proposals

	// Internal state
	signalSequence uint64
}

// NewKeeper creates a new signal keeper
func NewKeeper(
	storeKey storetypes.StoreKey,
	codec keepertypes.Codec,
	bankKeeper types.BankKeeper,
	stakingKeeper types.StakingKeeper,
	distributionKeeper types.DistributionKeeper,
	authority string,
) *Keeper {
	keeper := &Keeper{
		KVStoreKeeper:      base.NewKVStoreKeeper(storeKey, codec),
		bankKeeper:         bankKeeper,
		stakingKeeper:      stakingKeeper,
		distributionKeeper: distributionKeeper,
		authority:          authority,
		signalSequence:     0,
	}

	return keeper
}

// Logger returns the module logger
func (k Keeper) Logger(ctx keepertypes.Context) keepertypes.Logger {
	return ctx.Logger().With("module", types.ModuleName)
}

// GetAuthority returns the authority address
func (k Keeper) GetAuthority() string {
	return k.authority
}

// IsAuthorized checks if the given address is authorized for governance operations
func (k Keeper) IsAuthorized(address string) bool {
	return address == k.authority
}

// SetParams sets the module parameters
func (k Keeper) SetParams(ctx keepertypes.Context, params types.Params) error {
	if err := params.ValidateBasic(); err != nil {
		return err
	}

	store := k.GetKVStore(ctx)
	bz, err := k.GetCodec().Marshal(&params)
	if err != nil {
		return err
	}

	store.Set(types.ParamsKey, bz)

	// Emit parameter update event
	ctx.EventManager().EmitEvent(keepertypes.Event{
		Type: "params_updated",
		Attributes: []keepertypes.Attribute{
			{Key: "module", Value: types.ModuleName},
		},
	})

	return nil
}

// GetParams retrieves the module parameters
func (k Keeper) GetParams(ctx keepertypes.Context) types.Params {
	store := k.GetKVStore(ctx)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return types.DefaultParams()
	}

	var params types.Params
	if err := k.GetCodec().Unmarshal(bz, &params); err != nil {
		k.Logger(ctx).Error("failed to unmarshal params", "error", err)
		return types.DefaultParams()
	}

	return params
}

// SubmitSignal submits a new signal to the network
func (k Keeper) SubmitSignal(ctx keepertypes.Context, signal types.Signal) error {
	params := k.GetParams(ctx)

	// Validate signal
	if err := signal.ValidateBasic(); err != nil {
		return types.WrapError(err, "signal validation failed")
	}

	// Check if signal is within time limits
	if !params.ShouldProcessSignal(signal, ctx.BlockTime().Unix()) {
		return types.ErrSignalTooOld
	}

	// Check validator cooldown
	if err := k.checkValidatorCooldown(ctx, signal.ValidatorAddress, params); err != nil {
		return err
	}

	// Check maximum signals per validator
	if err := k.checkMaxSignalsPerValidator(ctx, signal.ValidatorAddress, params); err != nil {
		return err
	}

	// Verify work type exists and is enabled
	if err := k.verifyWorkType(ctx, signal.WorkProof.WorkType); err != nil {
		return err
	}

	// Set signal metadata
	signal.BlockHeight = ctx.BlockHeight()
	signal.ExpiryTime = signal.Timestamp + params.SignalRetentionPeriod

	// Store signal
	if err := k.StoreSignal(ctx, signal); err != nil {
		return types.WrapError(err, "failed to store signal")
	}

	// Update signal sequence
	k.incrementSignalSequence(ctx)

	// Update validator cooldown
	k.setValidatorCooldown(ctx, signal.ValidatorAddress, ctx.BlockTime().Unix())

	// Emit event
	types.EmitSignalEvent(ctx, types.NewSignalSubmittedEvent(signal))

	k.Logger(ctx).Info("signal submitted",
		"signal_id", signal.ID,
		"creator", signal.Creator,
		"validator", signal.ValidatorAddress,
		"work_type", signal.WorkProof.WorkType)

	return nil
}

// ProcessSignal processes and validates a submitted signal
func (k Keeper) ProcessSignal(ctx keepertypes.Context, signalID string) error {
	signal, found := k.GetSignal(ctx, signalID)
	if !found {
		return types.ErrSignalNotFound
	}

	if signal.Status != types.SignalStatusPending {
		return types.ErrSignalAlreadyProcessed
	}

	// Validate work proof
	if err := k.validateWorkProof(ctx, signal); err != nil {
		// Mark signal as rejected
		signal.Status = types.SignalStatusRejected
		k.StoreSignal(ctx, signal)

		types.EmitSignalEvent(ctx, types.NewSignalRejectedEvent(
			signal.ID, err.Error(), signal.ValidatorAddress))

		return err
	}

	// Calculate signal score
	score, err := k.CalculateSignalScore(ctx, signal)
	if err != nil {
		return types.WrapError(err, "failed to calculate signal score")
	}

	// Update signal with score and status
	signal.Score = score.FinalScore
	signal.Status = types.SignalStatusValidated

	// Store updated signal and score
	if err := k.StoreSignal(ctx, signal); err != nil {
		return err
	}
	if err := k.StoreSignalScore(ctx, score); err != nil {
		return err
	}

	// Update validator statistics
	if err := k.UpdateValidatorStats(ctx, signal.ValidatorAddress, signal, score.FinalScore); err != nil {
		k.Logger(ctx).Error("failed to update validator stats", "error", err)
		// Don't fail the whole process for stats update failure
	}

	// Emit events
	types.EmitSignalEvent(ctx, types.NewSignalValidatedEvent(
		signal.ID, score.FinalScore, signal.ValidatorAddress))
	types.EmitSignalEvent(ctx, types.NewScoreCalculatedEvent(score))

	k.Logger(ctx).Info("signal processed",
		"signal_id", signal.ID,
		"score", score.FinalScore,
		"status", signal.Status.String())

	return nil
}

// checkValidatorCooldown checks if validator is in cooldown period
func (k Keeper) checkValidatorCooldown(ctx keepertypes.Context, validatorAddr string, params types.Params) error {
	if params.SignalSubmissionCooldown <= 0 {
		return nil
	}

	store := k.GetKVStore(ctx)
	key := types.GetValidatorCooldownKey(validatorAddr)

	bz := store.Get(key)
	if bz == nil {
		return nil // No previous submission
	}

	var lastSubmission int64
	if err := k.GetCodec().Unmarshal(bz, &lastSubmission); err != nil {
		return nil // Corrupted data, allow submission
	}

	timeSinceLastSubmission := ctx.BlockTime().Unix() - lastSubmission
	if timeSinceLastSubmission < params.SignalSubmissionCooldown {
		return types.ErrCooldownNotExpired
	}

	return nil
}

// setValidatorCooldown sets the cooldown timestamp for a validator
func (k Keeper) setValidatorCooldown(ctx keepertypes.Context, validatorAddr string, timestamp int64) {
	store := k.GetKVStore(ctx)
	key := types.GetValidatorCooldownKey(validatorAddr)

	bz, err := k.GetCodec().Marshal(&timestamp)
	if err != nil {
		k.Logger(ctx).Error("failed to marshal cooldown timestamp", "error", err)
		return
	}

	store.Set(key, bz)
}

// checkMaxSignalsPerValidator checks if validator has exceeded max signals limit
func (k Keeper) checkMaxSignalsPerValidator(ctx keepertypes.Context, validatorAddr string, params types.Params) error {
	stats := k.GetValidatorStats(ctx, validatorAddr)
	if len(stats.ActiveSignals) >= int(params.MaxSignalsPerValidator) {
		return types.ErrMaxValidatorSignals
	}
	return nil
}

// verifyWorkType checks if work type exists and is enabled
func (k Keeper) verifyWorkType(ctx keepertypes.Context, workTypeID string) error {
	workType, found := k.GetWorkType(ctx, workTypeID)
	if !found {
		return types.ErrWorkTypeNotFound
	}
	if !workType.Enabled {
		return types.ErrWorkTypeDisabled
	}
	return nil
}

// validateWorkProof validates the work proof for a signal
func (k Keeper) validateWorkProof(ctx keepertypes.Context, signal types.Signal) error {
	workType, found := k.GetWorkType(ctx, signal.WorkProof.WorkType)
	if !found {
		return types.ErrWorkTypeNotFound
	}

	// Check if resources are within bounds
	if !workType.CheckResourcesInBounds(signal.WorkProof.ResourcesUsed) {
		return types.ErrResourcesOutOfBounds
	}

	// Additional verification based on work type
	switch workType.Verifier {
	case string(types.VerificationCryptographic):
		return k.verifyCryptographicProof(ctx, signal)
	case string(types.VerificationReproducible):
		return k.verifyReproducibleWork(ctx, signal)
	case string(types.VerificationStatistical):
		return k.verifyStatisticalWork(ctx, signal)
	case string(types.VerificationConsensus):
		return k.verifyConsensusWork(ctx, signal)
	case string(types.VerificationHybrid):
		return k.verifyHybridWork(ctx, signal)
	default:
		return types.ErrInvalidVerificationMethod
	}
}

// incrementSignalSequence increments the signal sequence counter
func (k Keeper) incrementSignalSequence(ctx keepertypes.Context) {
	store := k.GetKVStore(ctx)

	// Get current sequence
	bz := store.Get(types.SignalSequenceKey)
	var sequence uint64 = 0
	if bz != nil {
		if err := k.GetCodec().Unmarshal(bz, &sequence); err != nil {
			k.Logger(ctx).Error("failed to unmarshal signal sequence", "error", err)
		}
	}

	// Increment and store
	sequence++
	bz, err := k.GetCodec().Marshal(&sequence)
	if err != nil {
		k.Logger(ctx).Error("failed to marshal signal sequence", "error", err)
		return
	}

	store.Set(types.SignalSequenceKey, bz)
	k.signalSequence = sequence
}

// GetSignalSequence returns the current signal sequence
func (k Keeper) GetSignalSequence(ctx keepertypes.Context) uint64 {
	store := k.GetKVStore(ctx)
	bz := store.Get(types.SignalSequenceKey)

	if bz == nil {
		return 0
	}

	var sequence uint64
	if err := k.GetCodec().Unmarshal(bz, &sequence); err != nil {
		k.Logger(ctx).Error("failed to unmarshal signal sequence", "error", err)
		return 0
	}

	return sequence
}

// CleanupExpiredSignals removes expired signals from storage
func (k Keeper) CleanupExpiredSignals(ctx keepertypes.Context) error {
	currentTime := ctx.BlockTime().Unix()
	params := k.GetParams(ctx)

	// Get expired signals
	expiredSignals := k.GetExpiredSignals(ctx, currentTime)

	for _, signalID := range expiredSignals {
		signal, found := k.GetSignal(ctx, signalID)
		if !found {
			continue
		}

		// Mark as expired
		signal.Status = types.SignalStatusExpired
		k.StoreSignal(ctx, signal)

		// Remove from active signals
		k.removeFromActiveSignals(ctx, signal.ValidatorAddress, signalID)

		// Clean up related data if beyond retention period
		if currentTime-signal.Timestamp > params.SignalRetentionPeriod {
			k.deleteSignalData(ctx, signalID)
		}
	}

	return nil
}

// Helper methods for different verification types
func (k Keeper) verifyCryptographicProof(ctx keepertypes.Context, signal types.Signal) error {
	// Implement cryptographic proof verification
	// This would verify digital signatures, zero-knowledge proofs, etc.
	return nil // Simplified implementation
}

func (k Keeper) verifyReproducibleWork(ctx keepertypes.Context, signal types.Signal) error {
	// Implement reproducible work verification
	// This would re-run the computation to verify results
	return nil // Simplified implementation
}

func (k Keeper) verifyStatisticalWork(ctx keepertypes.Context, signal types.Signal) error {
	// Implement statistical verification
	// This would use statistical methods to verify work quality
	return nil // Simplified implementation
}

func (k Keeper) verifyConsensusWork(ctx keepertypes.Context, signal types.Signal) error {
	// Implement consensus-based verification
	// This would require multiple validators to verify the work
	return nil // Simplified implementation
}

func (k Keeper) verifyHybridWork(ctx keepertypes.Context, signal types.Signal) error {
	// Implement hybrid verification combining multiple methods
	return nil // Simplified implementation
}