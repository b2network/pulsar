package types

import (
	keepertypes "github.com/b2network/pulsar/keeper/types"
)

// RegisterCodec registers the necessary signal module types for codec
func RegisterCodec(cdc keepertypes.Codec) {
	// Register message types
	RegisterMsgTypes(cdc)

	// Register other types
	RegisterTypes(cdc)
}

// RegisterMsgTypes registers message types with the codec
func RegisterMsgTypes(cdc keepertypes.Codec) {
	// Note: In a real implementation, these would be registered with the actual codec
	// For now, we're using a simplified approach

	// Register transaction messages
	cdc.RegisterInterface((*keepertypes.Msg)(nil), nil)
	cdc.RegisterConcrete(&MsgSubmitSignal{}, "signal/MsgSubmitSignal", nil)
	cdc.RegisterConcrete(&MsgVerifyWork{}, "signal/MsgVerifyWork", nil)
	cdc.RegisterConcrete(&MsgUpdateWorkType{}, "signal/MsgUpdateWorkType", nil)
	cdc.RegisterConcrete(&MsgClaimReward{}, "signal/MsgClaimReward", nil)
	cdc.RegisterConcrete(&MsgBatchSubmit{}, "signal/MsgBatchSubmit", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "signal/MsgUpdateParams", nil)
	cdc.RegisterConcrete(&MsgRegisterWorkType{}, "signal/MsgRegisterWorkType", nil)
}

// RegisterTypes registers non-message types with the codec
func RegisterTypes(cdc keepertypes.Codec) {
	// Register core types
	cdc.RegisterConcrete(&Signal{}, "signal/Signal", nil)
	cdc.RegisterConcrete(&WorkProof{}, "signal/WorkProof", nil)
	cdc.RegisterConcrete(&Resources{}, "signal/Resources", nil)
	cdc.RegisterConcrete(&SignalBatch{}, "signal/SignalBatch", nil)

	// Register AI work types
	cdc.RegisterConcrete(&AIWorkType{}, "signal/AIWorkType", nil)
	cdc.RegisterConcrete(&AIWorkResult{}, "signal/AIWorkResult", nil)
	cdc.RegisterConcrete(&WorkVerification{}, "signal/WorkVerification", nil)
	cdc.RegisterConcrete(&AIWorkRegistry{}, "signal/AIWorkRegistry", nil)

	// Register score types
	cdc.RegisterConcrete(&SignalScore{}, "signal/SignalScore", nil)
	cdc.RegisterConcrete(&ScoreComponents{}, "signal/ScoreComponents", nil)
	cdc.RegisterConcrete(&ValidatorSignalStats{}, "signal/ValidatorSignalStats", nil)
	cdc.RegisterConcrete(&SignalScoreParams{}, "signal/SignalScoreParams", nil)
	cdc.RegisterConcrete(&SignalLeaderboard{}, "signal/SignalLeaderboard", nil)
	cdc.RegisterConcrete(&SignalLeaderboardEntry{}, "signal/SignalLeaderboardEntry", nil)
	cdc.RegisterConcrete(&ScoreDistribution{}, "signal/ScoreDistribution", nil)

	// Register module types
	cdc.RegisterConcrete(&Params{}, "signal/Params", nil)
	cdc.RegisterConcrete(&GenesisState{}, "signal/GenesisState", nil)
	cdc.RegisterConcrete(&ParamChangeProposal{}, "signal/ParamChangeProposal", nil)
	cdc.RegisterConcrete(&ParamChange{}, "signal/ParamChange", nil)
}

// ModuleCdc is the codec for the module
var ModuleCdc keepertypes.Codec

func init() {
	// In a real implementation, this would initialize the actual codec
	// For now, we'll leave it as a placeholder
}

// MarshalSignal marshals a Signal to bytes
func MarshalSignal(cdc keepertypes.Codec, signal Signal) ([]byte, error) {
	return cdc.Marshal(&signal)
}

// UnmarshalSignal unmarshals a Signal from bytes
func UnmarshalSignal(cdc keepertypes.Codec, data []byte) (Signal, error) {
	var signal Signal
	err := cdc.Unmarshal(data, &signal)
	return signal, err
}

// MarshalWorkType marshals an AIWorkType to bytes
func MarshalWorkType(cdc keepertypes.Codec, workType AIWorkType) ([]byte, error) {
	return cdc.Marshal(&workType)
}

// UnmarshalWorkType unmarshals an AIWorkType from bytes
func UnmarshalWorkType(cdc keepertypes.Codec, data []byte) (AIWorkType, error) {
	var workType AIWorkType
	err := cdc.Unmarshal(data, &workType)
	return workType, err
}

// MarshalSignalScore marshals a SignalScore to bytes
func MarshalSignalScore(cdc keepertypes.Codec, score SignalScore) ([]byte, error) {
	return cdc.Marshal(&score)
}

// UnmarshalSignalScore unmarshals a SignalScore from bytes
func UnmarshalSignalScore(cdc keepertypes.Codec, data []byte) (SignalScore, error) {
	var score SignalScore
	err := cdc.Unmarshal(data, &score)
	return score, err
}

// MarshalValidatorStats marshals ValidatorSignalStats to bytes
func MarshalValidatorStats(cdc keepertypes.Codec, stats ValidatorSignalStats) ([]byte, error) {
	return cdc.Marshal(&stats)
}

// UnmarshalValidatorStats unmarshals ValidatorSignalStats from bytes
func UnmarshalValidatorStats(cdc keepertypes.Codec, data []byte) (ValidatorSignalStats, error) {
	var stats ValidatorSignalStats
	err := cdc.Unmarshal(data, &stats)
	return stats, err
}

// MarshalParams marshals Params to bytes
func MarshalParams(cdc keepertypes.Codec, params Params) ([]byte, error) {
	return cdc.Marshal(&params)
}

// UnmarshalParams unmarshals Params from bytes
func UnmarshalParams(cdc keepertypes.Codec, data []byte) (Params, error) {
	var params Params
	err := cdc.Unmarshal(data, &params)
	return params, err
}

// MarshalGenesisState marshals GenesisState to bytes
func MarshalGenesisState(cdc keepertypes.Codec, genesis GenesisState) ([]byte, error) {
	return cdc.Marshal(&genesis)
}

// UnmarshalGenesisState unmarshals GenesisState from bytes
func UnmarshalGenesisState(cdc keepertypes.Codec, data []byte) (GenesisState, error) {
	var genesis GenesisState
	err := cdc.Unmarshal(data, &genesis)
	return genesis, err
}