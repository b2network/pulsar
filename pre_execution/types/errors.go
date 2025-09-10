package types

import (
	"fmt"
)

// Pre-execution specific errors
// These errors help identify and handle different failure scenarios in pre-execution

var (
	// ErrMsgNotPreExecutable indicates the message type doesn't support pre-execution
	ErrMsgNotPreExecutable = fmt.Errorf("message type does not support pre-execution")

	// ErrMsgTypeNotSupported indicates the message type is not supported by the module
	ErrMsgTypeNotSupported = fmt.Errorf("message type not supported for pre-execution")

	// ErrModuleNotEnabled indicates pre-execution is disabled for the module
	ErrModuleNotEnabled = fmt.Errorf("pre-execution not enabled for module")

	// ErrCacheResultExpired indicates the cached result has expired
	ErrCacheResultExpired = fmt.Errorf("cached pre-execution result has expired")

	// ErrCacheResultInvalid indicates the cached result is no longer valid
	ErrCacheResultInvalid = fmt.Errorf("cached pre-execution result is invalid")

	// ErrStateConflict indicates a state conflict was detected
	ErrStateConflict = fmt.Errorf("state conflict detected in pre-execution")

	// ErrGasLimitExceeded indicates pre-execution exceeded gas limit
	ErrGasLimitExceeded = fmt.Errorf("pre-execution gas limit exceeded")

	// ErrPreExecCacheFull indicates the pre-execution cache is at capacity
	ErrPreExecCacheFull = fmt.Errorf("pre-execution cache is full")

	// ErrSequenceOutOfOrder indicates transaction sequence is out of order
	ErrSequenceOutOfOrder = fmt.Errorf("transaction sequence is out of order")

	// ErrValidationFailed indicates pre-execution result validation failed
	ErrValidationFailed = fmt.Errorf("pre-execution result validation failed")

	// ErrSnapshotNotFound indicates state snapshot was not found
	ErrSnapshotNotFound = fmt.Errorf("state snapshot not found")

	// ErrContextNotPreExecution indicates the context is not for pre-execution
	ErrContextNotPreExecution = fmt.Errorf("context is not pre-execution context")
)

// PreExecError wraps pre-execution errors with additional context
type PreExecError struct {
	// Code is the error code for programmatic handling
	Code uint32 `json:"code"`

	// Message is the human-readable error message
	Message string `json:"message"`

	// TxHash identifies the transaction that caused the error
	TxHash string `json:"tx_hash,omitempty"`

	// Module identifies the module where the error occurred
	Module string `json:"module,omitempty"`

	// MsgType identifies the message type that caused the error
	MsgType string `json:"msg_type,omitempty"`

	// Cause is the underlying error that caused this error
	Cause error `json:"cause,omitempty"`
}

// Error implements the error interface
func (e *PreExecError) Error() string {
	if e.TxHash != "" {
		return fmt.Sprintf("pre-execution error [%s]: %s (tx: %s)", e.Module, e.Message, e.TxHash)
	}
	return fmt.Sprintf("pre-execution error [%s]: %s", e.Module, e.Message)
}

// Unwrap returns the underlying error for error wrapping
func (e *PreExecError) Unwrap() error {
	return e.Cause
}

// NewPreExecError creates a new pre-execution error with context
func NewPreExecError(code uint32, message string, txHash, module, msgType string, cause error) *PreExecError {
	return &PreExecError{
		Code:    code,
		Message: message,
		TxHash:  txHash,
		Module:  module,
		MsgType: msgType,
		Cause:   cause,
	}
}

// Predefined error codes for consistent error handling
const (
	// ErrCodeMsgNotSupported - Message type not supported for pre-execution
	ErrCodeMsgNotSupported uint32 = 1001

	// ErrCodeModuleDisabled - Pre-execution disabled for module
	ErrCodeModuleDisabled uint32 = 1002

	// ErrCodeGasExceeded - Gas limit exceeded during pre-execution
	ErrCodeGasExceeded uint32 = 1003

	// ErrCodeValidationFailed - Pre-execution validation failed
	ErrCodeValidationFailed uint32 = 1004

	// ErrCodeCacheExpired - Cached result has expired
	ErrCodeCacheExpired uint32 = 1005

	// ErrCodeCacheInvalid - Cached result is invalid
	ErrCodeCacheInvalid uint32 = 1006

	// ErrCodeStateConflict - State conflict detected
	ErrCodeStateConflict uint32 = 1007

	// ErrCodeCacheFull - Pre-execution cache is full
	ErrCodeCacheFull uint32 = 1008

	// ErrCodeSequenceError - Transaction sequence error
	ErrCodeSequenceError uint32 = 1009

	// ErrCodeSnapshotError - State snapshot error
	ErrCodeSnapshotError uint32 = 1010
)

// IsRetryableError determines if a pre-execution error is retryable
// Retryable errors can be resolved by re-attempting pre-execution later
func IsRetryableError(err error) bool {
	if preErr, ok := err.(*PreExecError); ok {
		switch preErr.Code {
		case ErrCodeGasExceeded, ErrCodeCacheFull, ErrCodeSequenceError:
			return true
		default:
			return false
		}
	}
	return false
}

// IsTemporaryError determines if a pre-execution error is temporary
// Temporary errors may resolve themselves without intervention
func IsTemporaryError(err error) bool {
	if preErr, ok := err.(*PreExecError); ok {
		switch preErr.Code {
		case ErrCodeCacheExpired, ErrCodeCacheInvalid, ErrCodeStateConflict:
			return true
		default:
			return false
		}
	}
	return false
}

// IsPermanentError determines if a pre-execution error is permanent
// Permanent errors require changes to the transaction or configuration
func IsPermanentError(err error) bool {
	if preErr, ok := err.(*PreExecError); ok {
		switch preErr.Code {
		case ErrCodeMsgNotSupported, ErrCodeModuleDisabled, ErrCodeValidationFailed:
			return true
		default:
			return false
		}
	}
	return false
}
