package abci

import (
	"context"
	"encoding/json"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	dbm "github.com/cosmos/cosmos-db"
)

type PulsarApp struct {
	db           dbm.DB
	logger       log.Logger
	state        State
	lastBlockHeight int64
}

type State struct {
	Height int64             `json:"height"`
	AppHash []byte           `json:"app_hash"`
	Data    map[string][]byte `json:"data"`
}

func NewPulsarApp(db dbm.DB, logger log.Logger) *PulsarApp {
	return &PulsarApp{
		db:     db,
		logger: logger,
		state: State{
			Height: 0,
			Data:   make(map[string][]byte),
		},
	}
}

func (app *PulsarApp) Info(_ context.Context, req *abci.RequestInfo) (*abci.ResponseInfo, error) {
	return &abci.ResponseInfo{
		Data:             "Pulsar Blockchain",
		Version:          "0.1.0",
		LastBlockHeight:  app.state.Height,
		LastBlockAppHash: app.state.AppHash,
	}, nil
}

func (app *PulsarApp) Query(_ context.Context, req *abci.RequestQuery) (*abci.ResponseQuery, error) {
	if value, exists := app.state.Data[string(req.Data)]; exists {
		return &abci.ResponseQuery{
			Key:   req.Data,
			Value: value,
			Height: app.state.Height,
		}, nil
	}
	return &abci.ResponseQuery{
		Log: fmt.Sprintf("key %s not found", string(req.Data)),
	}, nil
}

func (app *PulsarApp) CheckTx(_ context.Context, req *abci.RequestCheckTx) (*abci.ResponseCheckTx, error) {
	return &abci.ResponseCheckTx{Code: 0}, nil
}

func (app *PulsarApp) InitChain(_ context.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	app.state.Height = req.InitialHeight
	return &abci.ResponseInitChain{}, nil
}

func (app *PulsarApp) PrepareProposal(_ context.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
	return &abci.ResponsePrepareProposal{
		Txs: req.Txs,
	}, nil
}

func (app *PulsarApp) ProcessProposal(_ context.Context, req *abci.RequestProcessProposal) (*abci.ResponseProcessProposal, error) {
	return &abci.ResponseProcessProposal{
		Status: abci.ResponseProcessProposal_ACCEPT,
	}, nil
}

func (app *PulsarApp) FinalizeBlock(_ context.Context, req *abci.RequestFinalizeBlock) (*abci.ResponseFinalizeBlock, error) {
	app.state.Height = req.Height
	
	txResults := make([]*abci.ExecTxResult, len(req.Txs))
	for i, tx := range req.Txs {
		var txData map[string]string
		if err := json.Unmarshal(tx, &txData); err == nil {
			if key, ok := txData["key"]; ok {
				if value, ok := txData["value"]; ok {
					app.state.Data[key] = []byte(value)
				}
			}
		}
		txResults[i] = &abci.ExecTxResult{Code: 0}
	}

	appHash := app.calculateAppHash()
	app.state.AppHash = appHash

	return &abci.ResponseFinalizeBlock{
		TxResults: txResults,
		AppHash:   appHash,
	}, nil
}

func (app *PulsarApp) Commit(_ context.Context, req *abci.RequestCommit) (*abci.ResponseCommit, error) {
	app.saveState()
	return &abci.ResponseCommit{}, nil
}

func (app *PulsarApp) ListSnapshots(_ context.Context, req *abci.RequestListSnapshots) (*abci.ResponseListSnapshots, error) {
	return &abci.ResponseListSnapshots{}, nil
}

func (app *PulsarApp) OfferSnapshot(_ context.Context, req *abci.RequestOfferSnapshot) (*abci.ResponseOfferSnapshot, error) {
	return &abci.ResponseOfferSnapshot{}, nil
}

func (app *PulsarApp) LoadSnapshotChunk(_ context.Context, req *abci.RequestLoadSnapshotChunk) (*abci.ResponseLoadSnapshotChunk, error) {
	return &abci.ResponseLoadSnapshotChunk{}, nil
}

func (app *PulsarApp) ApplySnapshotChunk(_ context.Context, req *abci.RequestApplySnapshotChunk) (*abci.ResponseApplySnapshotChunk, error) {
	return &abci.ResponseApplySnapshotChunk{}, nil
}

func (app *PulsarApp) ExtendVote(_ context.Context, req *abci.RequestExtendVote) (*abci.ResponseExtendVote, error) {
	return &abci.ResponseExtendVote{}, nil
}

func (app *PulsarApp) VerifyVoteExtension(_ context.Context, req *abci.RequestVerifyVoteExtension) (*abci.ResponseVerifyVoteExtension, error) {
	return &abci.ResponseVerifyVoteExtension{
		Status: abci.ResponseVerifyVoteExtension_ACCEPT,
	}, nil
}

func (app *PulsarApp) calculateAppHash() []byte {
	data, _ := json.Marshal(app.state)
	return data[:32]
}

func (app *PulsarApp) saveState() {
	stateBytes, _ := json.Marshal(app.state)
	app.db.Set([]byte("state"), stateBytes)
}

func (app *PulsarApp) loadState() {
	stateBytes, _ := app.db.Get([]byte("state"))
	if stateBytes != nil {
		json.Unmarshal(stateBytes, &app.state)
	}
}