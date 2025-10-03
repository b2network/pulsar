package keeper

import (
	"fmt"

	keepertypes "github.com/b2network/pulsar/keeper/types"
	"github.com/b2network/pulsar/modules/signal/types"
)

// MsgServer implements the signal module message server
type MsgServer struct {
	Keeper
}

// NewMsgServerImpl creates a new message server implementation
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &MsgServer{Keeper: keeper}
}

var _ types.MsgServer = MsgServer{}

// SubmitSignal handles signal submission messages
func (ms MsgServer) SubmitSignal(ctx keepertypes.Context, msg *types.MsgSubmitSignal) (*types.MsgSubmitSignalResponse, error) {
	// Validate message
	if err := msg.ValidateBasic(); err != nil {
		return nil, types.WrapError(err, "invalid submit signal message")
	}

	// Check if creator has permission to submit signals
	if err := ms.validateSignalSubmissionPermission(ctx, msg.Creator); err != nil {
		return nil, err
	}

	// Create signal from message
	signal := types.NewSignal(
		msg.Creator,
		msg.SignalType,
		msg.Payload,
		msg.WorkProof,
		msg.ValidatorAddress,
	)

	// Submit signal
	if err := ms.Keeper.SubmitSignal(ctx, *signal); err != nil {
		return nil, types.WrapError(err, "failed to submit signal")
	}

	// Charge submission fee
	if err := ms.chargeSubmissionFee(ctx, msg.Creator); err != nil {
		ms.Keeper.Logger(ctx).Error("failed to charge submission fee", "error", err)
		// Don't fail the transaction for fee charging failure
	}

	ms.Keeper.Logger(ctx).Info("signal submitted successfully",
		"signal_id", signal.ID,
		"creator", msg.Creator,
		"work_type", msg.WorkProof.WorkType)

	return &types.MsgSubmitSignalResponse{
		SignalID: signal.ID,
	}, nil
}

// VerifyWork handles work verification messages
func (ms MsgServer) VerifyWork(ctx keepertypes.Context, msg *types.MsgVerifyWork) (*types.MsgVerifyWorkResponse, error) {
	// Validate message
	if err := msg.ValidateBasic(); err != nil {
		return nil, types.WrapError(err, "invalid verify work message")
	}

	// Check if verifier is authorized
	if err := ms.validateVerifierPermission(ctx, msg.Verifier); err != nil {
		return nil, err
	}

	// Check if signal exists
	signal, found := ms.Keeper.GetSignal(ctx, msg.SignalID)
	if !found {
		return nil, types.ErrSignalNotFound
	}

	// Create work verification
	verification := types.WorkVerification{
		SignalID:           msg.SignalID,
		VerifierAddress:    msg.Verifier,
		VerificationMethod: msg.VerificationMethod,
		VerificationTime:   ctx.BlockTime().Unix(),
		IsValid:            msg.IsValid,
		VerificationProof:  msg.VerificationProof,
	}

	// Calculate verification score
	verification.VerificationScore = ms.calculateVerificationScore(ctx, signal, verification)

	// Store verification
	if err := ms.Keeper.StoreWorkVerification(ctx, verification); err != nil {
		return nil, types.WrapError(err, "failed to store work verification")
	}

	// Check if we have enough verifications to finalize the signal
	if err := ms.checkAndFinalizeSignal(ctx, msg.SignalID); err != nil {
		ms.Keeper.Logger(ctx).Error("failed to finalize signal", "error", err)
		// Don't fail the verification for finalization failure
	}

	ms.Keeper.Logger(ctx).Info("work verification submitted",
		"signal_id", msg.SignalID,
		"verifier", msg.Verifier,
		"is_valid", msg.IsValid)

	return &types.MsgVerifyWorkResponse{
		VerificationScore: verification.VerificationScore,
	}, nil
}

// UpdateWorkType handles work type update messages
func (ms MsgServer) UpdateWorkType(ctx keepertypes.Context, msg *types.MsgUpdateWorkType) (*types.MsgUpdateWorkTypeResponse, error) {
	// Validate message
	if err := msg.ValidateBasic(); err != nil {
		return nil, types.WrapError(err, "invalid update work type message")
	}

	// Check authority
	if !ms.Keeper.IsAuthorized(msg.Authority) {
		return nil, types.ErrUnauthorized
	}

	// Store updated work type
	if err := ms.Keeper.StoreWorkType(ctx, msg.WorkType); err != nil {
		return nil, types.WrapError(err, "failed to update work type")
	}

	ms.Keeper.Logger(ctx).Info("work type updated",
		"work_type_id", msg.WorkType.ID,
		"authority", msg.Authority)

	return &types.MsgUpdateWorkTypeResponse{}, nil
}

// ClaimReward handles reward claiming messages
func (ms MsgServer) ClaimReward(ctx keepertypes.Context, msg *types.MsgClaimReward) (*types.MsgClaimRewardResponse, error) {
	// Validate message
	if err := msg.ValidateBasic(); err != nil {
		return nil, types.WrapError(err, "invalid claim reward message")
	}

	// Calculate and distribute rewards
	rewardAmount, err := ms.calculateAndDistributeRewards(ctx, msg.Validator)
	if err != nil {
		return nil, types.WrapError(err, "failed to claim rewards")
	}

	ms.Keeper.Logger(ctx).Info("rewards claimed",
		"validator", msg.Validator,
		"amount", rewardAmount.String())

	return &types.MsgClaimRewardResponse{
		RewardAmount: rewardAmount,
	}, nil
}

// BatchSubmit handles batch signal submission messages
func (ms MsgServer) BatchSubmit(ctx keepertypes.Context, msg *types.MsgBatchSubmit) (*types.MsgBatchSubmitResponse, error) {
	// Validate message
	if err := msg.ValidateBasic(); err != nil {
		return nil, types.WrapError(err, "invalid batch submit message")
	}

	params := ms.Keeper.GetParams(ctx)

	// Check batch size limit
	if len(msg.Signals) > int(params.MaxBatchSize) {
		return nil, types.ErrInvalidBatchSize
	}

	// Check if batch processing is enabled
	if !params.BatchProcessingEnabled {
		return nil, fmt.Errorf("batch processing is disabled")
	}

	var submittedSignalIDs []string
	var totalScore uint64

	// Process each signal in the batch
	for i, signalMsg := range msg.Signals {
		// Create signal
		signal := types.NewSignal(
			signalMsg.Creator,
			signalMsg.SignalType,
			signalMsg.Payload,
			signalMsg.WorkProof,
			signalMsg.ValidatorAddress,
		)

		// Submit signal
		if err := ms.Keeper.SubmitSignal(ctx, *signal); err != nil {
			ms.Keeper.Logger(ctx).Error("failed to submit signal in batch",
				"batch_index", i,
				"signal_id", signal.ID,
				"error", err)
			continue // Skip failed signals but continue with others
		}

		submittedSignalIDs = append(submittedSignalIDs, signal.ID)

		// Try to process signal immediately for batch
		if err := ms.Keeper.ProcessSignal(ctx, signal.ID); err != nil {
			ms.Keeper.Logger(ctx).Error("failed to process signal in batch",
				"signal_id", signal.ID,
				"error", err)
		} else {
			// Add score to total if processing succeeded
			if updatedSignal, found := ms.Keeper.GetSignal(ctx, signal.ID); found {
				totalScore += updatedSignal.Score
			}
		}
	}

	// Create batch record
	batchID := fmt.Sprintf("batch_%d_%s", ctx.BlockHeight(), msg.Creator)
	batch := types.SignalBatch{
		Signals:     []types.Signal{}, // We could populate this if needed
		BatchID:     batchID,
		ProcessedAt: ctx.BlockTime().Unix(),
		TotalScore:  totalScore,
	}

	// Store batch (optional, for record keeping)
	if err := ms.storeBatch(ctx, batch); err != nil {
		ms.Keeper.Logger(ctx).Error("failed to store batch record", "error", err)
	}

	// Emit batch processed event
	types.EmitSignalEvent(ctx, types.NewBatchProcessedEvent(
		batchID, len(submittedSignalIDs), totalScore))

	ms.Keeper.Logger(ctx).Info("batch signals submitted",
		"batch_id", batchID,
		"submitted_count", len(submittedSignalIDs),
		"total_count", len(msg.Signals),
		"total_score", totalScore)

	return &types.MsgBatchSubmitResponse{
		BatchID:          batchID,
		SubmittedSignals: submittedSignalIDs,
		TotalScore:       totalScore,
	}, nil
}

// UpdateParams handles parameter update messages
func (ms MsgServer) UpdateParams(ctx keepertypes.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	// Validate message
	if err := msg.ValidateBasic(); err != nil {
		return nil, types.WrapError(err, "invalid update params message")
	}

	// Check authority
	if !ms.Keeper.IsAuthorized(msg.Authority) {
		return nil, types.ErrUnauthorized
	}

	// Update parameters
	if err := ms.Keeper.SetParams(ctx, msg.Params); err != nil {
		return nil, types.WrapError(err, "failed to update params")
	}

	ms.Keeper.Logger(ctx).Info("module parameters updated", "authority", msg.Authority)

	return &types.MsgUpdateParamsResponse{}, nil
}

// RegisterWorkType handles work type registration messages
func (ms MsgServer) RegisterWorkType(ctx keepertypes.Context, msg *types.MsgRegisterWorkType) (*types.MsgRegisterWorkTypeResponse, error) {
	// Validate message
	if err := msg.ValidateBasic(); err != nil {
		return nil, types.WrapError(err, "invalid register work type message")
	}

	// Check authority
	if !ms.Keeper.IsAuthorized(msg.Authority) {
		return nil, types.ErrUnauthorized
	}

	// Check if work type already exists
	if _, found := ms.Keeper.GetWorkType(ctx, msg.WorkType.ID); found {
		return nil, fmt.Errorf("work type %s already exists", msg.WorkType.ID)
	}

	// Store new work type
	if err := ms.Keeper.StoreWorkType(ctx, msg.WorkType); err != nil {
		return nil, types.WrapError(err, "failed to register work type")
	}

	ms.Keeper.Logger(ctx).Info("work type registered",
		"work_type_id", msg.WorkType.ID,
		"authority", msg.Authority)

	return &types.MsgRegisterWorkTypeResponse{}, nil
}

// Helper methods

// validateSignalSubmissionPermission validates if creator can submit signals
func (ms MsgServer) validateSignalSubmissionPermission(ctx keepertypes.Context, creator string) error {
	// Basic validation - in a real implementation, you might check:
	// - Account balance for fees
	// - Validator status
	// - Blacklist status
	// - Rate limiting

	return nil // Simplified - allow all for now
}

// validateVerifierPermission validates if verifier can verify work
func (ms MsgServer) validateVerifierPermission(ctx keepertypes.Context, verifier string) error {
	// In a real implementation, you might check:
	// - Verifier is a registered validator
	// - Verifier has required stake
	// - Verifier is not jailed

	return nil // Simplified - allow all for now
}

// chargeSubmissionFee charges the signal submission fee
func (ms MsgServer) chargeSubmissionFee(ctx keepertypes.Context, creator string) error {
	params := ms.Keeper.GetParams(ctx)

	if len(params.SignalSubmissionFee) == 0 {
		return nil // No fee required
	}

	// In a real implementation, you would:
	// 1. Get creator's account
	// 2. Check balance
	// 3. Transfer fee to module account
	// 4. Distribute fee according to reward pool percentage

	return nil // Simplified implementation
}

// calculateVerificationScore calculates the score for a verification
func (ms MsgServer) calculateVerificationScore(ctx keepertypes.Context, signal types.Signal, verification types.WorkVerification) uint64 {
	// Simple scoring based on verification validity and method
	baseScore := uint64(10)

	if verification.IsValid {
		baseScore += 50
	}

	// Bonus for different verification methods
	switch verification.VerificationMethod {
	case types.VerificationCryptographic:
		baseScore += 20
	case types.VerificationConsensus:
		baseScore += 30
	case types.VerificationReproducible:
		baseScore += 40
	case types.VerificationHybrid:
		baseScore += 35
	}

	return baseScore
}

// checkAndFinalizeSignal checks if enough verifications exist to finalize a signal
func (ms MsgServer) checkAndFinalizeSignal(ctx keepertypes.Context, signalID string) error {
	params := ms.Keeper.GetParams(ctx)
	verifications := ms.Keeper.GetWorkVerifications(ctx, signalID)

	if uint32(len(verifications)) >= params.VerificationThreshold {
		// Count valid vs invalid verifications
		validCount := 0
		for _, verification := range verifications {
			if verification.IsValid {
				validCount++
			}
		}

		// Finalize based on majority
		if float64(validCount)/float64(len(verifications)) > 0.5 {
			return ms.Keeper.ProcessSignal(ctx, signalID)
		} else {
			// Mark as rejected
			signal, found := ms.Keeper.GetSignal(ctx, signalID)
			if found {
				signal.Status = types.SignalStatusRejected
				return ms.Keeper.StoreSignal(ctx, signal)
			}
		}
	}

	return nil
}

// calculateAndDistributeRewards calculates and distributes rewards for a validator
func (ms MsgServer) calculateAndDistributeRewards(ctx keepertypes.Context, validatorAddr string) (keepertypes.Coins, error) {
	// Get validator stats
	stats := ms.Keeper.GetValidatorStats(ctx, validatorAddr)

	// Calculate reward based on effective score
	rewardAmount := stats.EffectiveScore / 1000 // Simple conversion

	// Create reward coins
	rewards := keepertypes.Coins{
		{Denom: "stake", Amount: int64(rewardAmount)},
	}

	// In a real implementation, you would:
	// 1. Transfer from reward pool to validator
	// 2. Update reward pool balance
	// 3. Emit reward distribution event

	// Emit event
	types.EmitSignalEvent(ctx, types.NewRewardDistributedEvent(validatorAddr, rewards))

	return rewards, nil
}

// storeBatch stores a signal batch record
func (ms MsgServer) storeBatch(ctx keepertypes.Context, batch types.SignalBatch) error {
	store := ms.Keeper.GetKVStore(ctx)
	key := types.GetSignalBatchKey(batch.BatchID)

	bz, err := ms.Keeper.GetCodec().Marshal(&batch)
	if err != nil {
		return fmt.Errorf("failed to marshal batch: %w", err)
	}

	store.Set(key, bz)
	return nil
}