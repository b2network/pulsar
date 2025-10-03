package types

import (
	"fmt"

	keepertypes "github.com/b2network/pulsar/keeper/types"
)

// Message type constants
const (
	TypeMsgSubmitSignal     = "submit_signal"
	TypeMsgVerifyWork       = "verify_work"
	TypeMsgUpdateWorkType   = "update_work_type"
	TypeMsgClaimReward      = "claim_reward"
	TypeMsgBatchSubmit      = "batch_submit"
	TypeMsgUpdateParams     = "update_params"
	TypeMsgRegisterWorkType = "register_work_type"
)

// Ensure messages implement the Msg interface
var (
	_ keepertypes.Msg = &MsgSubmitSignal{}
	_ keepertypes.Msg = &MsgVerifyWork{}
	_ keepertypes.Msg = &MsgUpdateWorkType{}
	_ keepertypes.Msg = &MsgClaimReward{}
	_ keepertypes.Msg = &MsgBatchSubmit{}
	_ keepertypes.Msg = &MsgUpdateParams{}
	_ keepertypes.Msg = &MsgRegisterWorkType{}
)

// MsgSubmitSignal defines a message to submit a signal
type MsgSubmitSignal struct {
	Creator          string    `json:"creator"`
	SignalType       string    `json:"signal_type"`
	Payload          []byte    `json:"payload"`
	WorkProof        WorkProof `json:"work_proof"`
	ValidatorAddress string    `json:"validator_address"`
}

// NewMsgSubmitSignal creates a new MsgSubmitSignal
func NewMsgSubmitSignal(
	creator string,
	signalType string,
	payload []byte,
	workProof WorkProof,
	validatorAddr string,
) *MsgSubmitSignal {
	return &MsgSubmitSignal{
		Creator:          creator,
		SignalType:       signalType,
		Payload:          payload,
		WorkProof:        workProof,
		ValidatorAddress: validatorAddr,
	}
}

// Route implements Msg
func (msg *MsgSubmitSignal) Route() string {
	return RouterKey
}

// Type implements Msg
func (msg *MsgSubmitSignal) Type() string {
	return TypeMsgSubmitSignal
}

// ValidateBasic implements Msg
func (msg *MsgSubmitSignal) ValidateBasic() error {
	if msg.Creator == "" {
		return fmt.Errorf("creator cannot be empty")
	}
	if msg.SignalType == "" {
		return fmt.Errorf("signal type cannot be empty")
	}
	if len(msg.Payload) == 0 {
		return fmt.Errorf("payload cannot be empty")
	}
	if err := msg.WorkProof.ValidateBasic(); err != nil {
		return fmt.Errorf("invalid work proof: %w", err)
	}
	return nil
}

// GetSignBytes implements Msg
func (msg *MsgSubmitSignal) GetSignBytes() []byte {
	// This would typically use amino or protobuf encoding
	// Simplified for this implementation
	return []byte(fmt.Sprintf("%s:%s:%s", msg.Creator, msg.SignalType, msg.ValidatorAddress))
}

// GetSigners implements Msg
func (msg *MsgSubmitSignal) GetSigners() []string {
	return []string{msg.Creator}
}

// MsgVerifyWork defines a message to verify AI work
type MsgVerifyWork struct {
	Verifier           string             `json:"verifier"`
	SignalID           string             `json:"signal_id"`
	VerificationMethod VerificationMethod `json:"verification_method"`
	VerificationProof  []byte             `json:"verification_proof"`
	IsValid            bool               `json:"is_valid"`
}

// NewMsgVerifyWork creates a new MsgVerifyWork
func NewMsgVerifyWork(
	verifier string,
	signalID string,
	method VerificationMethod,
	proof []byte,
	isValid bool,
) *MsgVerifyWork {
	return &MsgVerifyWork{
		Verifier:           verifier,
		SignalID:           signalID,
		VerificationMethod: method,
		VerificationProof:  proof,
		IsValid:            isValid,
	}
}

// Route implements Msg
func (msg *MsgVerifyWork) Route() string {
	return RouterKey
}

// Type implements Msg
func (msg *MsgVerifyWork) Type() string {
	return TypeMsgVerifyWork
}

// ValidateBasic implements Msg
func (msg *MsgVerifyWork) ValidateBasic() error {
	if msg.Verifier == "" {
		return fmt.Errorf("verifier cannot be empty")
	}
	if msg.SignalID == "" {
		return fmt.Errorf("signal ID cannot be empty")
	}
	if msg.VerificationMethod == "" {
		return fmt.Errorf("verification method cannot be empty")
	}
	if len(msg.VerificationProof) == 0 {
		return fmt.Errorf("verification proof cannot be empty")
	}
	return nil
}

// GetSignBytes implements Msg
func (msg *MsgVerifyWork) GetSignBytes() []byte {
	return []byte(fmt.Sprintf("%s:%s:%s", msg.Verifier, msg.SignalID, msg.VerificationMethod))
}

// GetSigners implements Msg
func (msg *MsgVerifyWork) GetSigners() []string {
	return []string{msg.Verifier}
}

// MsgUpdateWorkType defines a message to update an AI work type
type MsgUpdateWorkType struct {
	Authority string     `json:"authority"`
	WorkType  AIWorkType `json:"work_type"`
}

// NewMsgUpdateWorkType creates a new MsgUpdateWorkType
func NewMsgUpdateWorkType(authority string, workType AIWorkType) *MsgUpdateWorkType {
	return &MsgUpdateWorkType{
		Authority: authority,
		WorkType:  workType,
	}
}

// Route implements Msg
func (msg *MsgUpdateWorkType) Route() string {
	return RouterKey
}

// Type implements Msg
func (msg *MsgUpdateWorkType) Type() string {
	return TypeMsgUpdateWorkType
}

// ValidateBasic implements Msg
func (msg *MsgUpdateWorkType) ValidateBasic() error {
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if err := msg.WorkType.ValidateBasic(); err != nil {
		return fmt.Errorf("invalid work type: %w", err)
	}
	return nil
}

// GetSignBytes implements Msg
func (msg *MsgUpdateWorkType) GetSignBytes() []byte {
	return []byte(fmt.Sprintf("%s:%s", msg.Authority, msg.WorkType.ID))
}

// GetSigners implements Msg
func (msg *MsgUpdateWorkType) GetSigners() []string {
	return []string{msg.Authority}
}

// MsgClaimReward defines a message to claim signal rewards
type MsgClaimReward struct {
	Validator string `json:"validator"`
}

// NewMsgClaimReward creates a new MsgClaimReward
func NewMsgClaimReward(validator string) *MsgClaimReward {
	return &MsgClaimReward{
		Validator: validator,
	}
}

// Route implements Msg
func (msg *MsgClaimReward) Route() string {
	return RouterKey
}

// Type implements Msg
func (msg *MsgClaimReward) Type() string {
	return TypeMsgClaimReward
}

// ValidateBasic implements Msg
func (msg *MsgClaimReward) ValidateBasic() error {
	if msg.Validator == "" {
		return fmt.Errorf("validator cannot be empty")
	}
	return nil
}

// GetSignBytes implements Msg
func (msg *MsgClaimReward) GetSignBytes() []byte {
	return []byte(msg.Validator)
}

// GetSigners implements Msg
func (msg *MsgClaimReward) GetSigners() []string {
	return []string{msg.Validator}
}

// MsgBatchSubmit defines a message to submit multiple signals
type MsgBatchSubmit struct {
	Creator  string            `json:"creator"`
	Signals  []MsgSubmitSignal `json:"signals"`
}

// NewMsgBatchSubmit creates a new MsgBatchSubmit
func NewMsgBatchSubmit(creator string, signals []MsgSubmitSignal) *MsgBatchSubmit {
	return &MsgBatchSubmit{
		Creator: creator,
		Signals: signals,
	}
}

// Route implements Msg
func (msg *MsgBatchSubmit) Route() string {
	return RouterKey
}

// Type implements Msg
func (msg *MsgBatchSubmit) Type() string {
	return TypeMsgBatchSubmit
}

// ValidateBasic implements Msg
func (msg *MsgBatchSubmit) ValidateBasic() error {
	if msg.Creator == "" {
		return fmt.Errorf("creator cannot be empty")
	}
	if len(msg.Signals) == 0 {
		return fmt.Errorf("signals cannot be empty")
	}
	for i, signal := range msg.Signals {
		if err := signal.ValidateBasic(); err != nil {
			return fmt.Errorf("invalid signal at index %d: %w", i, err)
		}
	}
	return nil
}

// GetSignBytes implements Msg
func (msg *MsgBatchSubmit) GetSignBytes() []byte {
	return []byte(fmt.Sprintf("%s:%d", msg.Creator, len(msg.Signals)))
}

// GetSigners implements Msg
func (msg *MsgBatchSubmit) GetSigners() []string {
	return []string{msg.Creator}
}

// MsgUpdateParams defines a message to update module parameters
type MsgUpdateParams struct {
	Authority string `json:"authority"`
	Params    Params `json:"params"`
}

// NewMsgUpdateParams creates a new MsgUpdateParams
func NewMsgUpdateParams(authority string, params Params) *MsgUpdateParams {
	return &MsgUpdateParams{
		Authority: authority,
		Params:    params,
	}
}

// Route implements Msg
func (msg *MsgUpdateParams) Route() string {
	return RouterKey
}

// Type implements Msg
func (msg *MsgUpdateParams) Type() string {
	return TypeMsgUpdateParams
}

// ValidateBasic implements Msg
func (msg *MsgUpdateParams) ValidateBasic() error {
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if err := msg.Params.ValidateBasic(); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}
	return nil
}

// GetSignBytes implements Msg
func (msg *MsgUpdateParams) GetSignBytes() []byte {
	return []byte(msg.Authority)
}

// GetSigners implements Msg
func (msg *MsgUpdateParams) GetSigners() []string {
	return []string{msg.Authority}
}

// MsgRegisterWorkType defines a message to register a new work type
type MsgRegisterWorkType struct {
	Authority string     `json:"authority"`
	WorkType  AIWorkType `json:"work_type"`
}

// NewMsgRegisterWorkType creates a new MsgRegisterWorkType
func NewMsgRegisterWorkType(authority string, workType AIWorkType) *MsgRegisterWorkType {
	return &MsgRegisterWorkType{
		Authority: authority,
		WorkType:  workType,
	}
}

// Route implements Msg
func (msg *MsgRegisterWorkType) Route() string {
	return RouterKey
}

// Type implements Msg
func (msg *MsgRegisterWorkType) Type() string {
	return TypeMsgRegisterWorkType
}

// ValidateBasic implements Msg
func (msg *MsgRegisterWorkType) ValidateBasic() error {
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if err := msg.WorkType.ValidateBasic(); err != nil {
		return fmt.Errorf("invalid work type: %w", err)
	}
	return nil
}

// GetSignBytes implements Msg
func (msg *MsgRegisterWorkType) GetSignBytes() []byte {
	return []byte(fmt.Sprintf("%s:%s", msg.Authority, msg.WorkType.ID))
}

// GetSigners implements Msg
func (msg *MsgRegisterWorkType) GetSigners() []string {
	return []string{msg.Authority}
}