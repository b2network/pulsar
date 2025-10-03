package types

import (
	"fmt"
)

// Signal module error codes
var (
	ErrInvalidSignal           = fmt.Errorf("invalid signal")
	ErrSignalNotFound          = fmt.Errorf("signal not found")
	ErrInvalidWorkProof        = fmt.Errorf("invalid work proof")
	ErrWorkTypeNotFound        = fmt.Errorf("work type not found")
	ErrWorkTypeDisabled        = fmt.Errorf("work type is disabled")
	ErrInvalidValidator        = fmt.Errorf("invalid validator")
	ErrSignalExpired           = fmt.Errorf("signal has expired")
	ErrSignalTooOld            = fmt.Errorf("signal is too old")
	ErrInsufficientFee         = fmt.Errorf("insufficient fee")
	ErrScoreBelowThreshold     = fmt.Errorf("score below minimum threshold")
	ErrMaxSignalsExceeded      = fmt.Errorf("maximum signals per block exceeded")
	ErrMaxValidatorSignals     = fmt.Errorf("maximum signals per validator exceeded")
	ErrCooldownNotExpired      = fmt.Errorf("submission cooldown not expired")
	ErrInvalidResources        = fmt.Errorf("invalid resource specification")
	ErrResourcesOutOfBounds    = fmt.Errorf("resources out of bounds for work type")
	ErrVerificationFailed      = fmt.Errorf("work verification failed")
	ErrInsufficientVerifiers   = fmt.Errorf("insufficient verifiers")
	ErrDuplicateSignal         = fmt.Errorf("duplicate signal")
	ErrInvalidSignature        = fmt.Errorf("invalid signal signature")
	ErrUnauthorized            = fmt.Errorf("unauthorized")
	ErrInvalidParams           = fmt.Errorf("invalid parameters")
	ErrBatchProcessingFailed   = fmt.Errorf("batch processing failed")
	ErrRewardDistributionFailed = fmt.Errorf("reward distribution failed")
	ErrSlashingFailed          = fmt.Errorf("slashing failed")
	ErrStatsUpdateFailed       = fmt.Errorf("stats update failed")
	ErrInvalidWorkType         = fmt.Errorf("invalid work type")
	ErrInvalidVerificationMethod = fmt.Errorf("invalid verification method")
	ErrSignalAlreadyProcessed  = fmt.Errorf("signal already processed")
	ErrInvalidBatchSize        = fmt.Errorf("invalid batch size")
	ErrRegistryUpdateFailed    = fmt.Errorf("registry update failed")
)

// IsSignalError checks if an error is a signal module error
func IsSignalError(err error) bool {
	if err == nil {
		return false
	}

	switch err {
	case ErrInvalidSignal,
		ErrSignalNotFound,
		ErrInvalidWorkProof,
		ErrWorkTypeNotFound,
		ErrWorkTypeDisabled,
		ErrInvalidValidator,
		ErrSignalExpired,
		ErrSignalTooOld,
		ErrInsufficientFee,
		ErrScoreBelowThreshold,
		ErrMaxSignalsExceeded,
		ErrMaxValidatorSignals,
		ErrCooldownNotExpired,
		ErrInvalidResources,
		ErrResourcesOutOfBounds,
		ErrVerificationFailed,
		ErrInsufficientVerifiers,
		ErrDuplicateSignal,
		ErrInvalidSignature,
		ErrUnauthorized,
		ErrInvalidParams,
		ErrBatchProcessingFailed,
		ErrRewardDistributionFailed,
		ErrSlashingFailed,
		ErrStatsUpdateFailed,
		ErrInvalidWorkType,
		ErrInvalidVerificationMethod,
		ErrSignalAlreadyProcessed,
		ErrInvalidBatchSize,
		ErrRegistryUpdateFailed:
		return true
	}
	return false
}

// WrapError wraps an error with additional context
func WrapError(err error, msg string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", msg, err)
}

// SignalError represents a detailed error for signal operations
type SignalError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Error implements the error interface
func (e SignalError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// NewSignalError creates a new SignalError
func NewSignalError(code, message, details string) SignalError {
	return SignalError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// Common error codes
const (
	ErrorCodeInvalidInput      = "INVALID_INPUT"
	ErrorCodeNotFound          = "NOT_FOUND"
	ErrorCodeUnauthorized      = "UNAUTHORIZED"
	ErrorCodeValidationFailed  = "VALIDATION_FAILED"
	ErrorCodeProcessingFailed  = "PROCESSING_FAILED"
	ErrorCodeResourceExhausted = "RESOURCE_EXHAUSTED"
	ErrorCodeInternal          = "INTERNAL_ERROR"
	ErrorCodeTimeout           = "TIMEOUT"
	ErrorCodeConflict          = "CONFLICT"
	ErrorCodePreconditionFailed = "PRECONDITION_FAILED"
)