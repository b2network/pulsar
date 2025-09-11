package keeper

import (
	abcitypes "github.com/cometbft/cometbft/abci/types"
	
	keepertypes "github.com/b2network/pulsar/keeper/types"
)

// Query paths for staking module
const (
	QueryValidators     = "validators"
	QueryValidator      = "validator"
	QueryDelegations    = "delegations"
	QueryDelegation     = "delegation"
	QueryUnbonding      = "unbonding"
	QueryRedelegations  = "redelegations"
	QueryPool           = "pool"
	QueryParams         = "params"
)

// NewQuerier creates a new querier for staking module
func NewQuerier(keeper *Keeper) keepertypes.ModuleQuerier {
	return &stakingQuerier{
		keeper: keeper,
	}
}

type stakingQuerier struct {
	keeper *Keeper
}

// Query implements the Querier interface
func (q *stakingQuerier) Query(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	if len(path) == 0 {
		return keepertypes.QueryError(1, "no query specified"), nil
	}
	
	switch path[0] {
	case QueryValidators:
		return q.queryValidators(ctx, req)
	case QueryValidator:
		return q.queryValidator(ctx, req)
	case QueryDelegations:
		return q.queryDelegations(ctx, req)
	case QueryDelegation:
		return q.queryDelegation(ctx, req)
	case QueryUnbonding:
		return q.queryUnbondingDelegations(ctx, req)
	case QueryRedelegations:
		return q.queryRedelegations(ctx, req)
	case QueryPool:
		return q.queryPool(ctx, req)
	case QueryParams:
		return q.queryParams(ctx, req)
	default:
		return keepertypes.QueryError(1, "unknown staking query: %s", path[0]), nil
	}
}

// RegisterQueryRoutes returns the routes this querier handles
func (q *stakingQuerier) RegisterQueryRoutes() map[string]keepertypes.QueryHandler {
	return map[string]keepertypes.QueryHandler{
		QueryValidators:    q.handleValidators,
		QueryValidator:     q.handleValidator,
		QueryDelegations:   q.handleDelegations,
		QueryDelegation:    q.handleDelegation,
		QueryUnbonding:     q.handleUnbonding,
		QueryRedelegations: q.handleRedelegations,
		QueryPool:          q.handlePool,
		QueryParams:        q.handleParams,
	}
}

// queryValidators queries all validators
func (q *stakingQuerier) queryValidators(ctx keepertypes.Context, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	// Get all validators from keeper
	validators := q.keeper.GetAllValidators(ctx)
	
	// Create response
	resp := map[string]interface{}{
		"validators": validators,
		"pagination": map[string]interface{}{
			"total": len(validators),
		},
	}
	
	// Marshal response
	data, err := q.keeper.GetCodec().Marshal(resp)
	if err != nil {
		return keepertypes.QueryError(1, "failed to marshal response: %v", err), nil
	}
	
	return keepertypes.QuerySuccess(data), nil
}

// queryValidator queries a single validator
func (q *stakingQuerier) queryValidator(ctx keepertypes.Context, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	// Parse validator address from request data
	validatorAddr := req.Data
	if len(validatorAddr) == 0 {
		return keepertypes.QueryError(1, "validator address is required"), nil
	}
	
	// Get validator from keeper
	validator, found := q.keeper.GetValidator(ctx, validatorAddr)
	if !found {
		return keepertypes.QueryError(1, "validator not found"), nil
	}
	
	// Marshal response
	data, err := q.keeper.GetCodec().Marshal(validator)
	if err != nil {
		return keepertypes.QueryError(1, "failed to marshal response: %v", err), nil
	}
	
	return keepertypes.QuerySuccess(data), nil
}

// queryDelegations queries all delegations for a delegator
func (q *stakingQuerier) queryDelegations(ctx keepertypes.Context, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	// Parse delegator address from request data
	delegatorAddr := req.Data
	if len(delegatorAddr) == 0 {
		return keepertypes.QueryError(1, "delegator address is required"), nil
	}
	
	// Get delegations from keeper
	delegations := q.keeper.GetDelegatorDelegations(ctx, delegatorAddr)
	
	// Create response
	resp := map[string]interface{}{
		"delegations": delegations,
		"pagination": map[string]interface{}{
			"total": len(delegations),
		},
	}
	
	// Marshal response
	data, err := q.keeper.GetCodec().Marshal(resp)
	if err != nil {
		return keepertypes.QueryError(1, "failed to marshal response: %v", err), nil
	}
	
	return keepertypes.QuerySuccess(data), nil
}

// queryDelegation queries a specific delegation
func (q *stakingQuerier) queryDelegation(ctx keepertypes.Context, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	// Expected format: delegator_addr,validator_addr
	// For simplicity, using the entire data as delegator address
	delegatorAddr := req.Data
	
	// Get delegation from keeper (simplified - would need both addresses)
	delegations := q.keeper.GetDelegatorDelegations(ctx, delegatorAddr)
	if len(delegations) == 0 {
		return keepertypes.QueryError(1, "delegation not found"), nil
	}
	
	// Return first delegation for now
	data, err := q.keeper.GetCodec().Marshal(delegations[0])
	if err != nil {
		return keepertypes.QueryError(1, "failed to marshal response: %v", err), nil
	}
	
	return keepertypes.QuerySuccess(data), nil
}

// queryUnbondingDelegations queries unbonding delegations
func (q *stakingQuerier) queryUnbondingDelegations(ctx keepertypes.Context, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	// Parse delegator address from request data
	delegatorAddr := req.Data
	if len(delegatorAddr) == 0 {
		return keepertypes.QueryError(1, "delegator address is required"), nil
	}
	
	// Get unbonding delegations from keeper
	unbondingDelegations := q.keeper.GetUnbondingDelegations(ctx, delegatorAddr)
	
	// Create response
	resp := map[string]interface{}{
		"unbonding_delegations": unbondingDelegations,
		"pagination": map[string]interface{}{
			"total": len(unbondingDelegations),
		},
	}
	
	// Marshal response
	data, err := q.keeper.GetCodec().Marshal(resp)
	if err != nil {
		return keepertypes.QueryError(1, "failed to marshal response: %v", err), nil
	}
	
	return keepertypes.QuerySuccess(data), nil
}

// queryRedelegations queries redelegations
func (q *stakingQuerier) queryRedelegations(ctx keepertypes.Context, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	// Parse delegator address from request data
	delegatorAddr := req.Data
	
	// Get redelegations from keeper
	redelegations := q.keeper.GetRedelegations(ctx, delegatorAddr)
	
	// Create response
	resp := map[string]interface{}{
		"redelegations": redelegations,
		"pagination": map[string]interface{}{
			"total": len(redelegations),
		},
	}
	
	// Marshal response
	data, err := q.keeper.GetCodec().Marshal(resp)
	if err != nil {
		return keepertypes.QueryError(1, "failed to marshal response: %v", err), nil
	}
	
	return keepertypes.QuerySuccess(data), nil
}

// queryPool queries the staking pool
func (q *stakingQuerier) queryPool(ctx keepertypes.Context, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	// Get pool info from keeper
	pool := q.keeper.GetPool(ctx)
	
	// Marshal response
	data, err := q.keeper.GetCodec().Marshal(pool)
	if err != nil {
		return keepertypes.QueryError(1, "failed to marshal response: %v", err), nil
	}
	
	return keepertypes.QuerySuccess(data), nil
}

// queryParams queries the staking module parameters
func (q *stakingQuerier) queryParams(ctx keepertypes.Context, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	// Get params from keeper
	params := q.keeper.GetParams(ctx)
	
	// Marshal response
	data, err := q.keeper.GetCodec().Marshal(params)
	if err != nil {
		return keepertypes.QueryError(1, "failed to marshal response: %v", err), nil
	}
	
	return keepertypes.QuerySuccess(data), nil
}

// Handler functions for direct routing
func (q *stakingQuerier) handleValidators(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	return q.queryValidators(ctx, req)
}

func (q *stakingQuerier) handleValidator(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	return q.queryValidator(ctx, req)
}

func (q *stakingQuerier) handleDelegations(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	return q.queryDelegations(ctx, req)
}

func (q *stakingQuerier) handleDelegation(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	return q.queryDelegation(ctx, req)
}

func (q *stakingQuerier) handleUnbonding(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	return q.queryUnbondingDelegations(ctx, req)
}

func (q *stakingQuerier) handleRedelegations(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	return q.queryRedelegations(ctx, req)
}

func (q *stakingQuerier) handlePool(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	return q.queryPool(ctx, req)
}

func (q *stakingQuerier) handleParams(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	return q.queryParams(ctx, req)
}