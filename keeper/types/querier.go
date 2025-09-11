package types

import (
	"fmt"
	
	abcitypes "github.com/cometbft/cometbft/abci/types"
)

// ModuleQuerier defines the interface for module query handlers
type ModuleQuerier interface {
	// Query handles a query request for this module
	Query(ctx Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error)
	
	// RegisterQueryRoutes returns the routes this querier handles
	RegisterQueryRoutes() map[string]QueryHandler
}

// QueryHandler is a function that handles a specific query
type QueryHandler func(ctx Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error)

// QueryRouter routes queries to appropriate handlers
type QueryRouter struct {
	routes map[string]ModuleQuerier
}

// NewQueryRouter creates a new query router
func NewQueryRouter() *QueryRouter {
	return &QueryRouter{
		routes: make(map[string]ModuleQuerier),
	}
}

// AddRoute adds a new route to the router
func (qr *QueryRouter) AddRoute(module string, querier ModuleQuerier) *QueryRouter {
	if _, ok := qr.routes[module]; ok {
		panic(fmt.Sprintf("route %s already exists", module))
	}
	qr.routes[module] = querier
	return qr
}

// Route routes a query to the appropriate handler
func (qr *QueryRouter) Route(ctx Context, path string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	// Parse path: /module/query
	pathParts := parsePath(path)
	if len(pathParts) < 2 {
		return &abcitypes.ResponseQuery{
			Code: 1,
			Log:  fmt.Sprintf("invalid query path: %s", path),
		}, nil
	}
	
	module := pathParts[0]
	querier, found := qr.routes[module]
	if !found {
		return &abcitypes.ResponseQuery{
			Code: 1,
			Log:  fmt.Sprintf("no route for module: %s", module),
		}, nil
	}
	
	// Pass the remaining path to the module querier
	return querier.Query(ctx, pathParts[1:], req)
}

// HasRoute checks if a route exists
func (qr *QueryRouter) HasRoute(module string) bool {
	_, ok := qr.routes[module]
	return ok
}

// parsePath splits a path string into parts
func parsePath(path string) []string {
	// Remove leading slash if present
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}
	
	// Split by slash
	parts := []string{}
	current := ""
	for _, ch := range path {
		if ch == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	
	return parts
}

// QueryResponse is a helper to create query responses
type QueryResponse struct {
	Code  uint32
	Log   string
	Value []byte
}

// Success creates a successful query response
func QuerySuccess(value []byte) *abcitypes.ResponseQuery {
	return &abcitypes.ResponseQuery{
		Code:  0,
		Value: value,
	}
}

// Error creates an error query response
func QueryError(code uint32, format string, args ...interface{}) *abcitypes.ResponseQuery {
	return &abcitypes.ResponseQuery{
		Code: code,
		Log:  fmt.Sprintf(format, args...),
	}
}