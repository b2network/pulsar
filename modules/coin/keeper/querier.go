package keeper

import (
	abcitypes "github.com/cometbft/cometbft/abci/types"

	keepertypes "github.com/b2network/pulsar/keeper/types"
)

// Query paths for coin module
const (
	QuerySupply         = "supply"
	QueryTotalSupply    = "total_supply"
	QueryMetadata       = "metadata"
	QueryAllMetadata    = "all_metadata"
	QueryPermissions    = "permissions"
	QueryModulePerms    = "module_permissions"
)

// NewQuerier creates a new querier for coin module
func NewQuerier(keeper *Keeper) keepertypes.ModuleQuerier {
	return &coinQuerier{
		keeper: keeper,
	}
}

type coinQuerier struct {
	keeper *Keeper
}

// Query implements the ModuleQuerier interface
func (q *coinQuerier) Query(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	if len(path) == 0 {
		return keepertypes.QueryError(1, "no query specified"), nil
	}

	switch path[0] {
	case QuerySupply:
		return q.querySupply(ctx, path[1:], req)
	case QueryTotalSupply:
		return q.queryTotalSupply(ctx, req)
	case QueryMetadata:
		return q.queryMetadata(ctx, path[1:], req)
	case QueryAllMetadata:
		return q.queryAllMetadata(ctx, req)
	case QueryPermissions:
		return q.queryPermissions(ctx, req)
	case QueryModulePerms:
		return q.queryModulePermissions(ctx, path[1:], req)
	default:
		return keepertypes.QueryError(1, "unknown coin query: %s", path[0]), nil
	}
}

// RegisterQueryRoutes returns the routes this querier handles
func (q *coinQuerier) RegisterQueryRoutes() map[string]keepertypes.QueryHandler {
	return map[string]keepertypes.QueryHandler{
		QuerySupply:         q.handleSupply,
		QueryTotalSupply:    q.handleTotalSupply,
		QueryMetadata:       q.handleMetadata,
		QueryAllMetadata:    q.handleAllMetadata,
		QueryPermissions:    q.handlePermissions,
		QueryModulePerms:    q.handleModulePermissions,
	}
}

// querySupply queries the supply of a specific denomination
func (q *coinQuerier) querySupply(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	if len(path) == 0 {
		return keepertypes.QueryError(1, "denom required"), nil
	}

	denom := path[0]
	supply := q.keeper.Supply().GetSupply(ctx, denom)

	data, err := q.keeper.GetCodec().Marshal(supply)
	if err != nil {
		return keepertypes.QueryError(1, "failed to marshal supply: %v", err), nil
	}

	return keepertypes.QuerySuccess(data), nil
}

// queryTotalSupply queries all coin supplies
func (q *coinQuerier) queryTotalSupply(ctx keepertypes.Context, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	supplies := q.keeper.Supply().GetAllSupplies(ctx)

	resp := map[string]interface{}{
		"supplies": supplies,
		"pagination": map[string]interface{}{
			"total": len(supplies),
		},
	}

	data, err := q.keeper.GetCodec().Marshal(resp)
	if err != nil {
		return keepertypes.QueryError(1, "failed to marshal supplies: %v", err), nil
	}

	return keepertypes.QuerySuccess(data), nil
}

// queryMetadata queries metadata for a specific denomination
func (q *coinQuerier) queryMetadata(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	if len(path) == 0 {
		return keepertypes.QueryError(1, "denom required"), nil
	}

	denom := path[0]
	metadata, found := q.keeper.Metadata().GetMetadata(ctx, denom)
	if !found {
		return keepertypes.QueryError(1, "metadata not found for denom: %s", denom), nil
	}

	data, err := q.keeper.GetCodec().Marshal(metadata)
	if err != nil {
		return keepertypes.QueryError(1, "failed to marshal metadata: %v", err), nil
	}

	return keepertypes.QuerySuccess(data), nil
}

// queryAllMetadata queries all denomination metadata
func (q *coinQuerier) queryAllMetadata(ctx keepertypes.Context, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	metadatas := q.keeper.Metadata().GetAllMetadata(ctx)

	resp := map[string]interface{}{
		"metadatas": metadatas,
		"pagination": map[string]interface{}{
			"total": len(metadatas),
		},
	}

	data, err := q.keeper.GetCodec().Marshal(resp)
	if err != nil {
		return keepertypes.QueryError(1, "failed to marshal metadatas: %v", err), nil
	}

	return keepertypes.QuerySuccess(data), nil
}

// queryPermissions queries all module permissions
func (q *coinQuerier) queryPermissions(ctx keepertypes.Context, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	permissions := q.keeper.Permission().GetPermissionSet(ctx)

	data, err := q.keeper.GetCodec().Marshal(permissions)
	if err != nil {
		return keepertypes.QueryError(1, "failed to marshal permissions: %v", err), nil
	}

	return keepertypes.QuerySuccess(data), nil
}

// queryModulePermissions queries permissions for a specific module
func (q *coinQuerier) queryModulePermissions(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	if len(path) == 0 {
		return keepertypes.QueryError(1, "module name required"), nil
	}

	moduleName := path[0]
	permissions, found := q.keeper.Permission().GetModulePermissions(ctx, moduleName)
	if !found {
		return keepertypes.QueryError(1, "permissions not found for module: %s", moduleName), nil
	}

	data, err := q.keeper.GetCodec().Marshal(permissions)
	if err != nil {
		return keepertypes.QueryError(1, "failed to marshal module permissions: %v", err), nil
	}

	return keepertypes.QuerySuccess(data), nil
}

// Handler functions for direct routing
func (q *coinQuerier) handleSupply(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	return q.querySupply(ctx, path, req)
}

func (q *coinQuerier) handleTotalSupply(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	return q.queryTotalSupply(ctx, req)
}

func (q *coinQuerier) handleMetadata(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	return q.queryMetadata(ctx, path, req)
}

func (q *coinQuerier) handleAllMetadata(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	return q.queryAllMetadata(ctx, req)
}

func (q *coinQuerier) handlePermissions(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	return q.queryPermissions(ctx, req)
}

func (q *coinQuerier) handleModulePermissions(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	return q.queryModulePermissions(ctx, path, req)
}