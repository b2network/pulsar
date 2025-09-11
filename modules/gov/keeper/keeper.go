package keeper

import (
	"fmt"

	"github.com/b2network/pulsar/keeper/base"
	keepertypes "github.com/b2network/pulsar/keeper/types"
	"github.com/b2network/pulsar/modules/gov/types"
	storetypes "github.com/b2network/pulsar/store/types"
)

// Keeper implements the governance keeper
type Keeper struct {
	*base.KVStoreKeeper

	bankKeeper    types.BankKeeper
	stakingKeeper types.StakingKeeper

	// Module authority for upgrades and parameter changes
	authority string
}

// NewKeeper creates a new governance keeper
func NewKeeper(
	storeKey storetypes.StoreKey,
	codec keepertypes.Codec,
	bankKeeper types.BankKeeper,
	stakingKeeper types.StakingKeeper,
	authority string,
) *Keeper {
	keeper := &Keeper{
		KVStoreKeeper: base.NewKVStoreKeeper(storeKey, codec),
		bankKeeper:    bankKeeper,
		stakingKeeper: stakingKeeper,
		authority:     authority,
	}

	return keeper
}

// Key prefixes for different types of data
var (
	ProposalsKeyPrefix          = []byte{0x00}
	ActiveProposalQueuePrefix   = []byte{0x01}
	InactiveProposalQueuePrefix = []byte{0x02}
	ProposalIDKey               = []byte{0x03}

	DepositsKeyPrefix = []byte{0x10}
	VotesKeyPrefix    = []byte{0x20}

	ParamsKey = []byte{0x30}
)

// GetProposal gets a proposal from store by ProposalID
func (k Keeper) GetProposal(ctx keepertypes.Context, proposalID uint64) (types.Proposal, bool) {
	store := k.GetKVStore(ctx)
	key := ProposalKey(proposalID)

	bz := store.Get(key)
	if bz == nil {
		return types.Proposal{}, false
	}

	var proposal types.Proposal
	if err := k.GetCodec().Unmarshal(bz, &proposal); err != nil {
		return types.Proposal{}, false
	}

	return proposal, true
}

// SetProposal sets a proposal in the store
func (k Keeper) SetProposal(ctx keepertypes.Context, proposal types.Proposal) {
	store := k.GetKVStore(ctx)
	key := ProposalKey(proposal.ID)

	bz, err := k.GetCodec().Marshal(proposal)
	if err != nil {
		panic(fmt.Errorf("failed to marshal proposal: %w", err))
	}

	store.Set(key, bz)
}

// GetProposals gets all proposals from store
func (k Keeper) GetProposals(ctx keepertypes.Context) []types.Proposal {
	var proposals []types.Proposal

	store := k.GetKVStore(ctx)
	iterator := store.Iterator(ProposalsKeyPrefix, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var proposal types.Proposal
		if err := k.GetCodec().Unmarshal(iterator.Value(), &proposal); err != nil {
			continue
		}
		proposals = append(proposals, proposal)
	}

	return proposals
}

// GetProposalID gets the next proposal ID from the store
func (k Keeper) GetProposalID(ctx keepertypes.Context) uint64 {
	var proposalID uint64
	if err := k.GetObject(ctx, ProposalIDKey, &proposalID); err != nil {
		return 1 // Start from 1 if not found
	}
	return proposalID
}

// SetProposalID sets the next proposal ID in the store
func (k Keeper) SetProposalID(ctx keepertypes.Context, proposalID uint64) {
	if err := k.SetObject(ctx, ProposalIDKey, proposalID); err != nil {
		panic(fmt.Errorf("failed to set proposal ID: %w", err))
	}
}

// SubmitProposal creates a new proposal given a content
func (k Keeper) SubmitProposal(ctx keepertypes.Context, messages []types.ProposalMessage, title, summary, proposer string) (uint64, error) {
	proposalID := k.GetProposalID(ctx)

	submitTime := ctx.BlockHeight() // Simplified: use block height as time
	params := k.GetParams(ctx)
	depositEndTime := submitTime + params.MaxDepositPeriod

	proposal := types.Proposal{
		ID:               proposalID,
		Messages:         messages,
		Status:           types.StatusDepositPeriod,
		FinalTallyResult: types.TallyResult{},
		SubmitTime:       submitTime,
		DepositEndTime:   depositEndTime,
		TotalDeposit:     []keepertypes.Coin{},
		VotingStartTime:  0,
		VotingEndTime:    0,
		Title:            title,
		Summary:          summary,
		Proposer:         proposer,
	}

	k.SetProposal(ctx, proposal)
	k.SetProposalID(ctx, proposalID+1)

	// Emit proposal submission event
	k.EmitEvent(ctx, keepertypes.Event{
		Type: types.EventTypeSubmitProposal,
		Attributes: []keepertypes.Attribute{
			{Key: types.AttributeKeyProposalID, Value: fmt.Sprintf("%d", proposalID)},
			{Key: "title", Value: title},
			{Key: "proposer", Value: proposer},
		},
	})

	return proposalID, nil
}

// ActivateVotingPeriod activates the voting period of a proposal
func (k Keeper) ActivateVotingPeriod(ctx keepertypes.Context, proposal types.Proposal) {
	params := k.GetParams(ctx)

	proposal.VotingStartTime = ctx.BlockHeight()
	proposal.VotingEndTime = proposal.VotingStartTime + params.VotingPeriod
	proposal.Status = types.StatusVotingPeriod

	k.SetProposal(ctx, proposal)

	// Emit active proposal event
	k.EmitEvent(ctx, keepertypes.Event{
		Type: types.EventTypeActiveProposal,
		Attributes: []keepertypes.Attribute{
			{Key: types.AttributeKeyProposalID, Value: fmt.Sprintf("%d", proposal.ID)},
		},
	})
}

// GetDeposit gets a specific deposit on a specific proposal
func (k Keeper) GetDeposit(ctx keepertypes.Context, proposalID uint64, depositorAddr []byte) (types.Deposit, bool) {
	store := k.GetKVStore(ctx)
	key := DepositKey(proposalID, depositorAddr)

	bz := store.Get(key)
	if bz == nil {
		return types.Deposit{}, false
	}

	var deposit types.Deposit
	if err := k.GetCodec().Unmarshal(bz, &deposit); err != nil {
		return types.Deposit{}, false
	}

	return deposit, true
}

// SetDeposit sets a deposit in the store
func (k Keeper) SetDeposit(ctx keepertypes.Context, deposit types.Deposit) {
	store := k.GetKVStore(ctx)
	key := DepositKey(deposit.ProposalID, []byte(deposit.Depositor))

	bz, err := k.GetCodec().Marshal(deposit)
	if err != nil {
		panic(fmt.Errorf("failed to marshal deposit: %w", err))
	}

	store.Set(key, bz)
}

// GetDeposits gets all the deposits on a specific proposal
func (k Keeper) GetDeposits(ctx keepertypes.Context, proposalID uint64) []types.Deposit {
	var deposits []types.Deposit

	store := k.GetKVStore(ctx)
	prefix := DepositsKey(proposalID)
	iterator := store.Iterator(prefix, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var deposit types.Deposit
		if err := k.GetCodec().Unmarshal(iterator.Value(), &deposit); err != nil {
			continue
		}
		deposits = append(deposits, deposit)
	}

	return deposits
}

// AddDeposit adds or updates a deposit of a specific depositor on a specific proposal
func (k Keeper) AddDeposit(ctx keepertypes.Context, proposalID uint64, depositorAddr []byte, depositAmount []keepertypes.Coin) (bool, error) {
	// Get the proposal
	proposal, found := k.GetProposal(ctx, proposalID)
	if !found {
		return false, fmt.Errorf("proposal %d not found", proposalID)
	}

	// Check if proposal is in deposit period
	if proposal.Status != types.StatusDepositPeriod {
		return false, fmt.Errorf("proposal %d (%s) not in deposit period", proposalID, proposal.Status.String())
	}

	// Transfer deposit from account to gov module
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, depositorAddr, types.ModuleName, depositAmount); err != nil {
		return false, err
	}

	// Update the proposal
	proposal.TotalDeposit = AddCoins(proposal.TotalDeposit, depositAmount)

	// Get or create deposit
	deposit, found := k.GetDeposit(ctx, proposalID, depositorAddr)
	if found {
		deposit.Amount = AddCoins(deposit.Amount, depositAmount)
	} else {
		deposit = types.Deposit{
			ProposalID: proposalID,
			Depositor:  string(depositorAddr),
			Amount:     depositAmount,
		}
	}

	k.SetDeposit(ctx, deposit)

	// Check if minimum deposit is reached
	params := k.GetParams(ctx)
	minDepositReached := IsAllGTE(proposal.TotalDeposit, params.MinDeposit)

	if minDepositReached {
		k.ActivateVotingPeriod(ctx, proposal)
	} else {
		k.SetProposal(ctx, proposal)
	}

	// Emit deposit event
	k.EmitEvent(ctx, keepertypes.Event{
		Type: types.EventTypeProposalDeposit,
		Attributes: []keepertypes.Attribute{
			{Key: types.AttributeKeyProposalID, Value: fmt.Sprintf("%d", proposalID)},
			{Key: types.AttributeKeyDepositor, Value: string(depositorAddr)},
			{Key: "amount", Value: formatCoins(depositAmount)},
		},
	})

	return minDepositReached, nil
}

// GetVote gets a vote from a specific voter on a specific proposal
func (k Keeper) GetVote(ctx keepertypes.Context, proposalID uint64, voterAddr []byte) (types.Vote, bool) {
	store := k.GetKVStore(ctx)
	key := VoteKey(proposalID, voterAddr)

	bz := store.Get(key)
	if bz == nil {
		return types.Vote{}, false
	}

	var vote types.Vote
	if err := k.GetCodec().Unmarshal(bz, &vote); err != nil {
		return types.Vote{}, false
	}

	return vote, true
}

// SetVote sets a vote in the store
func (k Keeper) SetVote(ctx keepertypes.Context, vote types.Vote) {
	store := k.GetKVStore(ctx)
	key := VoteKey(vote.ProposalID, []byte(vote.Voter))

	bz, err := k.GetCodec().Marshal(vote)
	if err != nil {
		panic(fmt.Errorf("failed to marshal vote: %w", err))
	}

	store.Set(key, bz)
}

// GetVotes gets all the votes on a specific proposal
func (k Keeper) GetVotes(ctx keepertypes.Context, proposalID uint64) []types.Vote {
	var votes []types.Vote

	store := k.GetKVStore(ctx)
	prefix := VotesKey(proposalID)
	iterator := store.Iterator(prefix, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var vote types.Vote
		if err := k.GetCodec().Unmarshal(iterator.Value(), &vote); err != nil {
			continue
		}
		votes = append(votes, vote)
	}

	return votes
}

// AddVote adds a vote on a specific proposal
func (k Keeper) AddVote(ctx keepertypes.Context, proposalID uint64, voterAddr []byte, options []types.VoteOption, metadata string) error {
	// Get the proposal
	proposal, found := k.GetProposal(ctx, proposalID)
	if !found {
		return fmt.Errorf("proposal %d not found", proposalID)
	}

	// Check if proposal is in voting period
	if proposal.Status != types.StatusVotingPeriod {
		return fmt.Errorf("proposal %d (%s) not in voting period", proposalID, proposal.Status.String())
	}

	// Validate vote options
	if len(options) == 0 {
		return fmt.Errorf("vote options cannot be empty")
	}

	vote := types.Vote{
		ProposalID: proposalID,
		Voter:      string(voterAddr),
		Options:    options,
		Metadata:   metadata,
	}

	k.SetVote(ctx, vote)

	// Emit vote event
	k.EmitEvent(ctx, keepertypes.Event{
		Type: types.EventTypeProposalVote,
		Attributes: []keepertypes.Attribute{
			{Key: types.AttributeKeyProposalID, Value: fmt.Sprintf("%d", proposalID)},
			{Key: types.AttributeKeyVoter, Value: string(voterAddr)},
			{Key: types.AttributeKeyOption, Value: fmt.Sprintf("%v", options)},
		},
	})

	return nil
}

// Tally iterates over the votes and updates the tally of a proposal based on the voting power of the voters
func (k Keeper) Tally(ctx keepertypes.Context, proposal types.Proposal) (passes bool, burnDeposits bool, tallyResults types.TallyResult) {
	results := types.TallyResult{}

	// Get total bonded tokens
	totalBonded := k.stakingKeeper.GetLastTotalPower(ctx)
	if totalBonded <= 0 {
		return false, false, results
	}

	// Iterate through all votes
	votes := k.GetVotes(ctx, proposal.ID)

	for _, vote := range votes {
		// For simplicity, assume each vote has equal weight (1 token)
		// In a real implementation, this would be weighted by staking power
		voterWeight := int64(1)

		for _, option := range vote.Options {
			switch option {
			case types.OptionYes:
				results.YesCount += voterWeight
			case types.OptionAbstain:
				results.AbstainCount += voterWeight
			case types.OptionNo:
				results.NoCount += voterWeight
			case types.OptionNoWithVeto:
				results.NoWithVetoCount += voterWeight
			}
		}
	}

	params := k.GetParams(ctx)

	// Calculate percentages (multiply by 1000000 for precision)
	totalVotes := results.YesCount + results.AbstainCount + results.NoCount + results.NoWithVetoCount

	// Check if quorum is reached
	if totalVotes == 0 || (totalVotes*1000000/totalBonded) < params.Quorum {
		return false, false, results
	}

	// Check veto threshold
	if results.NoWithVetoCount*1000000/totalVotes >= params.VetoThreshold {
		return false, true, results
	}

	// Check if proposal passes
	if results.YesCount*1000000/(results.YesCount+results.NoCount) > params.Threshold {
		return true, false, results
	}

	return false, false, results
}

// GetParams returns the governance parameters
func (k Keeper) GetParams(ctx keepertypes.Context) types.Params {
	var params types.Params
	if err := k.GetObject(ctx, ParamsKey, &params); err != nil {
		return types.DefaultParams()
	}
	return params
}

// SetParams sets the governance parameters
func (k Keeper) SetParams(ctx keepertypes.Context, params types.Params) {
	if err := k.SetObject(ctx, ParamsKey, params); err != nil {
		panic(fmt.Errorf("failed to set params: %w", err))
	}
}

// Key construction helpers

// ProposalKey gets the key for a proposal from its proposalID
func ProposalKey(proposalID uint64) []byte {
	return append(ProposalsKeyPrefix, Uint64ToBigEndian(proposalID)...)
}

// DepositKey gets the key for a deposit from its proposalID and depositorAddr
func DepositKey(proposalID uint64, depositorAddr []byte) []byte {
	return append(DepositsKey(proposalID), depositorAddr...)
}

// DepositsKey gets the key prefix for deposits on a proposal
func DepositsKey(proposalID uint64) []byte {
	return append(DepositsKeyPrefix, Uint64ToBigEndian(proposalID)...)
}

// VoteKey gets the key for a vote from its proposalID and voterAddr
func VoteKey(proposalID uint64, voterAddr []byte) []byte {
	return append(VotesKey(proposalID), voterAddr...)
}

// VotesKey gets the key prefix for votes on a proposal
func VotesKey(proposalID uint64) []byte {
	return append(VotesKeyPrefix, Uint64ToBigEndian(proposalID)...)
}

// Helper functions

// Uint64ToBigEndian converts uint64 to big endian bytes
func Uint64ToBigEndian(i uint64) []byte {
	b := make([]byte, 8)
	b[0] = byte(i >> 56)
	b[1] = byte(i >> 48)
	b[2] = byte(i >> 40)
	b[3] = byte(i >> 32)
	b[4] = byte(i >> 24)
	b[5] = byte(i >> 16)
	b[6] = byte(i >> 8)
	b[7] = byte(i)
	return b
}

// AddCoins adds two coin slices
func AddCoins(coins1, coins2 []keepertypes.Coin) []keepertypes.Coin {
	// Simplified implementation - just append
	// In real implementation, this would merge coins of same denomination
	return append(coins1, coins2...)
}

// IsAllGTE returns true if coins1 >= coins2 for all denominations
func IsAllGTE(coins1, coins2 []keepertypes.Coin) bool {
	// Simplified implementation - just check if lengths match
	// In real implementation, this would check amounts per denomination
	return len(coins1) >= len(coins2)
}

// formatCoins formats coins for events
func formatCoins(coins []keepertypes.Coin) string {
	result := ""
	for i, coin := range coins {
		if i > 0 {
			result += ","
		}
		result += fmt.Sprintf("%d%s", coin.Amount, coin.Denom)
	}
	return result
}

// GetAllProposals returns all proposals (dummy implementation)
func (k Keeper) GetAllProposals(ctx keepertypes.Context) []interface{} {
	// Dummy implementation - return empty list
	return []interface{}{}
}

// GetTallyResult returns tally result for a proposal (dummy implementation)
func (k Keeper) GetTallyResult(ctx keepertypes.Context, proposalID uint64) interface{} {
	// Dummy implementation - return empty tally
	return map[string]interface{}{
		"yes":        "0",
		"abstain":    "0", 
		"no":         "0",
		"no_with_veto": "0",
	}
}

// Querier returns a new querier for the gov module
func (k Keeper) Querier() keepertypes.ModuleQuerier {
	return NewQuerier(&k)
}
