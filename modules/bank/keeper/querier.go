package keeper

import (
	"fmt"
	"strings"
	
	abcitypes "github.com/cometbft/cometbft/abci/types"
	
	keepertypes "github.com/b2network/pulsar/keeper/types"
)

// Query paths for bank module
const (
	QueryBalance        = "balance"
	QueryAllBalances    = "balances"
	QueryTotalSupply    = "total-supply"
	QuerySupplyOf       = "supply-of"
	QueryDenomMetadata  = "denom-metadata"
	QueryDenomsMetadata = "denoms-metadata"
	QueryParams         = "params"
)

// NewQuerier creates a new querier for bank module
func NewQuerier(keeper *Keeper) keepertypes.ModuleQuerier {
	return &bankQuerier{
		keeper: keeper,
	}
}

type bankQuerier struct {
	keeper *Keeper
}

// Query implements the Querier interface
func (q *bankQuerier) Query(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	if len(path) == 0 {
		return keepertypes.QueryError(1, "no query specified"), nil
	}
	
	switch path[0] {
	case QueryBalance:
		return q.queryBalance(ctx, path[1:], req)
	case QueryAllBalances:
		return q.queryAllBalances(ctx, path[1:], req)
	case QueryTotalSupply:
		return q.queryTotalSupply(ctx, req)
	case QuerySupplyOf:
		return q.querySupplyOf(ctx, path[1:], req)
	case QueryDenomMetadata:
		return q.queryDenomMetadata(ctx, path[1:], req)
	case QueryDenomsMetadata:
		return q.queryDenomsMetadata(ctx, req)
	case QueryParams:
		return q.queryParams(ctx, req)
	default:
		return keepertypes.QueryError(1, "unknown bank query: %s", path[0]), nil
	}
}

// RegisterQueryRoutes returns the routes this querier handles
func (q *bankQuerier) RegisterQueryRoutes() map[string]keepertypes.QueryHandler {
	return map[string]keepertypes.QueryHandler{
		QueryBalance:        q.handleBalance,
		QueryAllBalances:    q.handleAllBalances,
		QueryTotalSupply:    q.handleTotalSupply,
		QuerySupplyOf:       q.handleSupplyOf,
		QueryDenomMetadata:  q.handleDenomMetadata,
		QueryDenomsMetadata: q.handleDenomsMetadata,
		QueryParams:         q.handleParams,
	}
}

// queryBalance queries the balance of a single coin for an account
func (q *bankQuerier) queryBalance(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	// Parse request data
	// Expected format: address and denom
	params := strings.Split(string(req.Data), ",")
	if len(params) != 2 {
		return keepertypes.QueryError(1, "invalid params, expected: address,denom"), nil
	}
	
	address := params[0]
	denom := params[1]
	
	// Get balance from keeper
	balance := q.keeper.GetBalance(ctx, []byte(address), denom)
	
	// Create response
	resp := map[string]interface{}{
		"address": address,
		"denom":   denom,
		"amount":  fmt.Sprintf("%d", balance),
	}
	
	// Marshal response
	data, err := q.keeper.GetCodec().Marshal(resp)
	if err != nil {
		return keepertypes.QueryError(1, "failed to marshal response: %v", err), nil
	}
	
	return keepertypes.QuerySuccess(data), nil
}

// queryAllBalances queries all balances for an account
func (q *bankQuerier) queryAllBalances(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	// Parse address from request data
	address := string(req.Data)
	if address == "" {
		return keepertypes.QueryError(1, "address is required"), nil
	}
	
	// Get all balances from keeper
	balances := q.keeper.GetAllBalances(ctx, []byte(address))
	
	// Create response
	resp := map[string]interface{}{
		"address":  address,
		"balances": balances,
	}
	
	// Marshal response
	data, err := q.keeper.GetCodec().Marshal(resp)
	if err != nil {
		return keepertypes.QueryError(1, "failed to marshal response: %v", err), nil
	}
	
	return keepertypes.QuerySuccess(data), nil
}

// queryTotalSupply queries the total supply of all coins
func (q *bankQuerier) queryTotalSupply(ctx keepertypes.Context, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	// Get total supply from keeper
	supplies := q.keeper.Supply().GetTotalSupply(ctx)
	
	// Create response
	resp := map[string]interface{}{
		"supply": supplies,
	}
	
	// Marshal response
	data, err := q.keeper.GetCodec().Marshal(resp)
	if err != nil {
		return keepertypes.QueryError(1, "failed to marshal response: %v", err), nil
	}
	
	return keepertypes.QuerySuccess(data), nil
}

// querySupplyOf queries the supply of a single coin
func (q *bankQuerier) querySupplyOf(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	// Parse denom from request data
	denom := string(req.Data)
	if denom == "" {
		return keepertypes.QueryError(1, "denom is required"), nil
	}
	
	// Get supply from keeper
	supply := q.keeper.Supply().GetSupply(ctx, denom)
	
	// Create response
	resp := map[string]interface{}{
		"denom":  denom,
		"amount": supply.Supply,
	}
	
	// Marshal response
	data, err := q.keeper.GetCodec().Marshal(resp)
	if err != nil {
		return keepertypes.QueryError(1, "failed to marshal response: %v", err), nil
	}
	
	return keepertypes.QuerySuccess(data), nil
}

// queryDenomMetadata queries the metadata of a single denomination
func (q *bankQuerier) queryDenomMetadata(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	// Parse denom from request data
	denom := string(req.Data)
	if denom == "" {
		return keepertypes.QueryError(1, "denom is required"), nil
	}
	
	// Get metadata from keeper
	metadata, found := q.keeper.DenomMetadata().GetDenomMetadata(ctx, denom)
	if !found {
		return keepertypes.QueryError(1, "denom metadata not found"), nil
	}
	
	// Marshal response
	data, err := q.keeper.GetCodec().Marshal(metadata)
	if err != nil {
		return keepertypes.QueryError(1, "failed to marshal response: %v", err), nil
	}
	
	return keepertypes.QuerySuccess(data), nil
}

// queryDenomsMetadata queries the metadata of all denominations
func (q *bankQuerier) queryDenomsMetadata(ctx keepertypes.Context, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	// Get all metadata from keeper
	metadatas := q.keeper.DenomMetadata().GetAllDenomMetadata(ctx)
	
	// Create response
	resp := map[string]interface{}{
		"metadatas": metadatas,
	}
	
	// Marshal response
	data, err := q.keeper.GetCodec().Marshal(resp)
	if err != nil {
		return keepertypes.QueryError(1, "failed to marshal response: %v", err), nil
	}
	
	return keepertypes.QuerySuccess(data), nil
}

// queryParams queries the bank module parameters
func (q *bankQuerier) queryParams(ctx keepertypes.Context, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	// Get params (placeholder for now)
	params := map[string]interface{}{
		"send_enabled":          true,
		"default_send_enabled":  true,
	}
	
	// Marshal response
	data, err := q.keeper.GetCodec().Marshal(params)
	if err != nil {
		return keepertypes.QueryError(1, "failed to marshal response: %v", err), nil
	}
	
	return keepertypes.QuerySuccess(data), nil
}

// Handler functions for direct routing
func (q *bankQuerier) handleBalance(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	return q.queryBalance(ctx, path, req)
}

func (q *bankQuerier) handleAllBalances(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	return q.queryAllBalances(ctx, path, req)
}

func (q *bankQuerier) handleTotalSupply(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	return q.queryTotalSupply(ctx, req)
}

func (q *bankQuerier) handleSupplyOf(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	return q.querySupplyOf(ctx, path, req)
}

func (q *bankQuerier) handleDenomMetadata(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	return q.queryDenomMetadata(ctx, path, req)
}

func (q *bankQuerier) handleDenomsMetadata(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	return q.queryDenomsMetadata(ctx, req)
}

func (q *bankQuerier) handleParams(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	return q.queryParams(ctx, req)
}