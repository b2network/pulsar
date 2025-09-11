package keeper

import (
	"strconv"
	
	abcitypes "github.com/cometbft/cometbft/abci/types"
	
	keepertypes "github.com/b2network/pulsar/keeper/types"
)

// Query paths for gov module
const (
	QueryProposals     = "proposals"
	QueryProposal      = "proposal"
	QueryDeposits      = "deposits"
	QueryDeposit       = "deposit"
	QueryVotes         = "votes"
	QueryVote          = "vote"
	QueryTally         = "tally"
	QueryParams        = "params"
)

// NewQuerier creates a new querier for gov module
func NewQuerier(keeper *Keeper) keepertypes.ModuleQuerier {
	return &govQuerier{
		keeper: keeper,
	}
}

type govQuerier struct {
	keeper *Keeper
}

// Query implements the Querier interface
func (q *govQuerier) Query(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	if len(path) == 0 {
		return keepertypes.QueryError(1, "no query specified"), nil
	}
	
	switch path[0] {
	case QueryProposals:
		return q.queryProposals(ctx, req)
	case QueryProposal:
		return q.queryProposal(ctx, req)
	case QueryDeposits:
		return q.queryDeposits(ctx, req)
	case QueryDeposit:
		return q.queryDeposit(ctx, req)
	case QueryVotes:
		return q.queryVotes(ctx, req)
	case QueryVote:
		return q.queryVote(ctx, req)
	case QueryTally:
		return q.queryTally(ctx, req)
	case QueryParams:
		return q.queryParams(ctx, req)
	default:
		return keepertypes.QueryError(1, "unknown gov query: %s", path[0]), nil
	}
}

// RegisterQueryRoutes returns the routes this querier handles
func (q *govQuerier) RegisterQueryRoutes() map[string]keepertypes.QueryHandler {
	return map[string]keepertypes.QueryHandler{
		QueryProposals: q.handleProposals,
		QueryProposal:  q.handleProposal,
		QueryDeposits:  q.handleDeposits,
		QueryDeposit:   q.handleDeposit,
		QueryVotes:     q.handleVotes,
		QueryVote:      q.handleVote,
		QueryTally:     q.handleTally,
		QueryParams:    q.handleParams,
	}
}

// queryProposals queries all proposals
func (q *govQuerier) queryProposals(ctx keepertypes.Context, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	// Get all proposals from keeper
	proposals := q.keeper.GetAllProposals(ctx)
	
	// Create response
	resp := map[string]interface{}{
		"proposals": proposals,
		"pagination": map[string]interface{}{
			"total": len(proposals),
		},
	}
	
	// Marshal response
	data, err := q.keeper.GetCodec().Marshal(resp)
	if err != nil {
		return keepertypes.QueryError(1, "failed to marshal response: %v", err), nil
	}
	
	return keepertypes.QuerySuccess(data), nil
}

// queryProposal queries a single proposal
func (q *govQuerier) queryProposal(ctx keepertypes.Context, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	// Parse proposal ID from request data
	proposalIDStr := string(req.Data)
	proposalID, err := strconv.ParseUint(proposalIDStr, 10, 64)
	if err != nil {
		return keepertypes.QueryError(1, "invalid proposal ID: %v", err), nil
	}
	
	// Get proposal from keeper
	proposal, found := q.keeper.GetProposal(ctx, proposalID)
	if !found {
		return keepertypes.QueryError(1, "proposal not found"), nil
	}
	
	// Marshal response
	data, err := q.keeper.GetCodec().Marshal(proposal)
	if err != nil {
		return keepertypes.QueryError(1, "failed to marshal response: %v", err), nil
	}
	
	return keepertypes.QuerySuccess(data), nil
}

// queryDeposits queries all deposits for a proposal
func (q *govQuerier) queryDeposits(ctx keepertypes.Context, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	// Parse proposal ID from request data
	proposalIDStr := string(req.Data)
	proposalID, err := strconv.ParseUint(proposalIDStr, 10, 64)
	if err != nil {
		return keepertypes.QueryError(1, "invalid proposal ID: %v", err), nil
	}
	
	// Get deposits from keeper
	deposits := q.keeper.GetDeposits(ctx, proposalID)
	
	// Create response
	resp := map[string]interface{}{
		"deposits": deposits,
		"pagination": map[string]interface{}{
			"total": len(deposits),
		},
	}
	
	// Marshal response
	data, err := q.keeper.GetCodec().Marshal(resp)
	if err != nil {
		return keepertypes.QueryError(1, "failed to marshal response: %v", err), nil
	}
	
	return keepertypes.QuerySuccess(data), nil
}

// queryDeposit queries a specific deposit
func (q *govQuerier) queryDeposit(ctx keepertypes.Context, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	// Expected format: proposal_id,depositor_addr
	// For simplicity, just using proposal ID
	proposalIDStr := string(req.Data)
	proposalID, err := strconv.ParseUint(proposalIDStr, 10, 64)
	if err != nil {
		return keepertypes.QueryError(1, "invalid proposal ID: %v", err), nil
	}
	
	// Get first deposit for the proposal (simplified)
	deposits := q.keeper.GetDeposits(ctx, proposalID)
	if len(deposits) == 0 {
		return keepertypes.QueryError(1, "deposit not found"), nil
	}
	
	// Marshal response
	data, err := q.keeper.GetCodec().Marshal(deposits[0])
	if err != nil {
		return keepertypes.QueryError(1, "failed to marshal response: %v", err), nil
	}
	
	return keepertypes.QuerySuccess(data), nil
}

// queryVotes queries all votes for a proposal
func (q *govQuerier) queryVotes(ctx keepertypes.Context, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	// Parse proposal ID from request data
	proposalIDStr := string(req.Data)
	proposalID, err := strconv.ParseUint(proposalIDStr, 10, 64)
	if err != nil {
		return keepertypes.QueryError(1, "invalid proposal ID: %v", err), nil
	}
	
	// Get votes from keeper
	votes := q.keeper.GetVotes(ctx, proposalID)
	
	// Create response
	resp := map[string]interface{}{
		"votes": votes,
		"pagination": map[string]interface{}{
			"total": len(votes),
		},
	}
	
	// Marshal response
	data, err := q.keeper.GetCodec().Marshal(resp)
	if err != nil {
		return keepertypes.QueryError(1, "failed to marshal response: %v", err), nil
	}
	
	return keepertypes.QuerySuccess(data), nil
}

// queryVote queries a specific vote
func (q *govQuerier) queryVote(ctx keepertypes.Context, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	// Expected format: proposal_id,voter_addr
	// For simplicity, just using proposal ID
	proposalIDStr := string(req.Data)
	proposalID, err := strconv.ParseUint(proposalIDStr, 10, 64)
	if err != nil {
		return keepertypes.QueryError(1, "invalid proposal ID: %v", err), nil
	}
	
	// Get first vote for the proposal (simplified)
	votes := q.keeper.GetVotes(ctx, proposalID)
	if len(votes) == 0 {
		return keepertypes.QueryError(1, "vote not found"), nil
	}
	
	// Marshal response
	data, err := q.keeper.GetCodec().Marshal(votes[0])
	if err != nil {
		return keepertypes.QueryError(1, "failed to marshal response: %v", err), nil
	}
	
	return keepertypes.QuerySuccess(data), nil
}

// queryTally queries the tally result for a proposal
func (q *govQuerier) queryTally(ctx keepertypes.Context, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	// Parse proposal ID from request data
	proposalIDStr := string(req.Data)
	proposalID, err := strconv.ParseUint(proposalIDStr, 10, 64)
	if err != nil {
		return keepertypes.QueryError(1, "invalid proposal ID: %v", err), nil
	}
	
	// Get tally from keeper
	tally := q.keeper.GetTallyResult(ctx, proposalID)
	
	// Marshal response
	data, err := q.keeper.GetCodec().Marshal(tally)
	if err != nil {
		return keepertypes.QueryError(1, "failed to marshal response: %v", err), nil
	}
	
	return keepertypes.QuerySuccess(data), nil
}

// queryParams queries the gov module parameters
func (q *govQuerier) queryParams(ctx keepertypes.Context, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
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
func (q *govQuerier) handleProposals(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	return q.queryProposals(ctx, req)
}

func (q *govQuerier) handleProposal(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	return q.queryProposal(ctx, req)
}

func (q *govQuerier) handleDeposits(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	return q.queryDeposits(ctx, req)
}

func (q *govQuerier) handleDeposit(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	return q.queryDeposit(ctx, req)
}

func (q *govQuerier) handleVotes(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	return q.queryVotes(ctx, req)
}

func (q *govQuerier) handleVote(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	return q.queryVote(ctx, req)
}

func (q *govQuerier) handleTally(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	return q.queryTally(ctx, req)
}

func (q *govQuerier) handleParams(ctx keepertypes.Context, path []string, req abcitypes.RequestQuery) (*abcitypes.ResponseQuery, error) {
	return q.queryParams(ctx, req)
}