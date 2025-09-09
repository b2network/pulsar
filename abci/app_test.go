package abci

import (
	"context"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	dbm "github.com/cosmos/cosmos-db"
)

func TestPulsarApp_Info(t *testing.T) {
	db := dbm.NewMemDB()
	logger := log.NewNopLogger()
	app := NewPulsarApp(db, logger)

	req := &abci.RequestInfo{}
	resp, err := app.Info(context.Background(), req)

	if err != nil {
		t.Fatalf("Info() returned error: %v", err)
	}

	if resp.Data != "Pulsar Blockchain" {
		t.Errorf("Expected Data to be 'Pulsar Blockchain', got %s", resp.Data)
	}

	if resp.Version != "0.1.0" {
		t.Errorf("Expected Version to be '0.1.0', got %s", resp.Version)
	}
}

func TestPulsarApp_InitChain(t *testing.T) {
	db := dbm.NewMemDB()
	logger := log.NewNopLogger()
	app := NewPulsarApp(db, logger)

	req := &abci.RequestInitChain{
		InitialHeight: 1,
	}
	resp, err := app.InitChain(context.Background(), req)

	if err != nil {
		t.Fatalf("InitChain() returned error: %v", err)
	}

	if resp == nil {
		t.Fatal("InitChain() returned nil response")
	}

	if app.state.Height != 1 {
		t.Errorf("Expected Height to be 1, got %d", app.state.Height)
	}
}

func TestPulsarApp_CheckTx(t *testing.T) {
	db := dbm.NewMemDB()
	logger := log.NewNopLogger()
	app := NewPulsarApp(db, logger)

	req := &abci.RequestCheckTx{
		Tx:   []byte(`{"key":"test","value":"data"}`),
		Type: abci.CheckTxType_New,
	}
	resp, err := app.CheckTx(context.Background(), req)

	if err != nil {
		t.Fatalf("CheckTx() returned error: %v", err)
	}

	if resp.Code != 0 {
		t.Errorf("Expected Code to be 0, got %d", resp.Code)
	}
}

func TestPulsarApp_FinalizeBlock(t *testing.T) {
	db := dbm.NewMemDB()
	logger := log.NewNopLogger()
	app := NewPulsarApp(db, logger)

	// Initialize first
	initReq := &abci.RequestInitChain{InitialHeight: 1}
	_, err := app.InitChain(context.Background(), initReq)
	if err != nil {
		t.Fatalf("InitChain() returned error: %v", err)
	}

	// Test finalize block with transaction
	req := &abci.RequestFinalizeBlock{
		Height: 1,
		Txs:    [][]byte{[]byte(`{"key":"test","value":"data"}`)},
	}
	resp, err := app.FinalizeBlock(context.Background(), req)

	if err != nil {
		t.Fatalf("FinalizeBlock() returned error: %v", err)
	}

	if len(resp.TxResults) != 1 {
		t.Errorf("Expected 1 transaction result, got %d", len(resp.TxResults))
	}

	if resp.TxResults[0].Code != 0 {
		t.Errorf("Expected transaction to succeed, got code %d", resp.TxResults[0].Code)
	}

	if app.state.Height != 1 {
		t.Errorf("Expected Height to be 1, got %d", app.state.Height)
	}
}

func TestPulsarApp_Query(t *testing.T) {
	db := dbm.NewMemDB()
	logger := log.NewNopLogger()
	app := NewPulsarApp(db, logger)

	// Add data first
	app.state.Data["test"] = []byte("data")

	req := &abci.RequestQuery{
		Data: []byte("test"),
	}
	resp, err := app.Query(context.Background(), req)

	if err != nil {
		t.Fatalf("Query() returned error: %v", err)
	}

	if string(resp.Value) != "data" {
		t.Errorf("Expected value to be 'data', got %s", string(resp.Value))
	}

	// Test non-existent key
	req = &abci.RequestQuery{
		Data: []byte("nonexistent"),
	}
	resp, err = app.Query(context.Background(), req)

	if err != nil {
		t.Fatalf("Query() returned error: %v", err)
	}

	if len(resp.Value) != 0 {
		t.Errorf("Expected empty value for non-existent key, got %s", string(resp.Value))
	}
}