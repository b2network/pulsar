package examples

import (
	"fmt"

	keepertypes "github.com/b2network/pulsar/keeper/types"
	bankkeeper "github.com/b2network/pulsar/modules/bank/keeper"
	banktypes "github.com/b2network/pulsar/modules/bank/types"
	govkeeper "github.com/b2network/pulsar/modules/gov/keeper"
	govtypes "github.com/b2network/pulsar/modules/gov/types"
	stakingkeeper "github.com/b2network/pulsar/modules/staking/keeper"
	stakingtypes "github.com/b2network/pulsar/modules/staking/types"
	storetypes "github.com/b2network/pulsar/store/types"
)

// KeeperIntegrationExample demonstrates how different keepers work together
type KeeperIntegrationExample struct {
	ctx           keepertypes.Context
	bankKeeper    *bankkeeper.Keeper
	stakingKeeper *stakingkeeper.Keeper
	govKeeper     *govkeeper.Keeper
}

// NewKeeperIntegrationExample creates a new integration example
func NewKeeperIntegrationExample() *KeeperIntegrationExample {
	// Create store keys
	bankStoreKey := storetypes.NewMemoryStoreKey("bank")
	stakingStoreKey := storetypes.NewMemoryStoreKey("staking")
	govStoreKey := storetypes.NewMemoryStoreKey("gov")

	// Create multi-store with simple memory stores
	multiStore := NewSimpleMultiStore()
	multiStore.MountStore(bankStoreKey)
	multiStore.MountStore(stakingStoreKey)
	multiStore.MountStore(govStoreKey)

	// Create context
	logger := keepertypes.NewBasicLogger("keeper-example")
	ctx := keepertypes.NewPulsarContext(multiStore, 1, "test-chain", logger)

	// Create codec
	codec := keepertypes.NewJSONCodec()

	// Create keepers
	bankKeeper := bankkeeper.NewKeeper(
		bankStoreKey,
		codec,
		nil,                       // No account keeper in this example
		make(map[string][]string), // No module permissions
	)

	stakingKeeper := stakingkeeper.NewKeeper(
		stakingStoreKey,
		codec,
		bankKeeper, // Pass bank keeper to staking keeper
		nil,        // No slashing keeper in this example
		"stake",    // Bond denomination
	)

	// Create a wrapper for staking keeper to match interface
	stakingWrapper := &StakingKeeperWrapper{stakingKeeper}

	govKeeper := govkeeper.NewKeeper(
		govStoreKey,
		codec,
		bankKeeper,     // Pass bank keeper to gov keeper
		stakingWrapper, // Pass wrapped staking keeper to gov keeper
		"gov",          // Authority
	)

	return &KeeperIntegrationExample{
		ctx:           ctx,
		bankKeeper:    bankKeeper,
		stakingKeeper: stakingKeeper,
		govKeeper:     govKeeper,
	}
}

// RunBankingOperations demonstrates basic banking operations
func (example *KeeperIntegrationExample) RunBankingOperations() error {
	fmt.Println("=== Banking Operations Example ===")

	// Create test addresses
	alice := []byte("alice")
	bob := []byte("bob")

	// Set initial balances
	aliceBalance := banktypes.Balance{
		Address: string(alice),
		Denom:   "stake",
		Amount:  1000000, // 1M tokens
	}

	bobBalance := banktypes.Balance{
		Address: string(bob),
		Denom:   "stake",
		Amount:  500000, // 500K tokens
	}

	example.bankKeeper.SetBalance(example.ctx, alice, aliceBalance)
	example.bankKeeper.SetBalance(example.ctx, bob, bobBalance)

	// Check balances
	aliceStake := example.bankKeeper.GetBalance(example.ctx, alice, "stake")
	bobStake := example.bankKeeper.GetBalance(example.ctx, bob, "stake")

	fmt.Printf("Initial balances:\n")
	fmt.Printf("Alice: %d stake\n", aliceStake)
	fmt.Printf("Bob: %d stake\n", bobStake)

	// Transfer coins from Alice to Bob
	transferAmount := []keepertypes.Coin{{Denom: "stake", Amount: 100000}}
	if err := example.bankKeeper.SendCoins(example.ctx, alice, bob, transferAmount); err != nil {
		return fmt.Errorf("failed to transfer coins: %w", err)
	}

	// Check balances after transfer
	aliceStakeAfter := example.bankKeeper.GetBalance(example.ctx, alice, "stake")
	bobStakeAfter := example.bankKeeper.GetBalance(example.ctx, bob, "stake")

	fmt.Printf("\nAfter transfer of 100000 stake:\n")
	fmt.Printf("Alice: %d stake\n", aliceStakeAfter)
	fmt.Printf("Bob: %d stake\n", bobStakeAfter)

	fmt.Println("Banking operations completed successfully!")
	return nil
}

// RunStakingOperations demonstrates staking operations
func (example *KeeperIntegrationExample) RunStakingOperations() error {
	fmt.Println("\n=== Staking Operations Example ===")

	// Create validator
	validatorAddr := []byte("validator1")
	validator := stakingtypes.Validator{
		OperatorAddress:   string(validatorAddr),
		ConsPubKey:        []byte("pubkey123"),
		Jailed:            false,
		Status:            stakingtypes.Bonded,
		Tokens:            0,
		DelegatorShares:   0,
		Description:       stakingtypes.Description{Moniker: "Test Validator"},
		UnbondingHeight:   0,
		UnbondingTime:     0,
		Commission:        stakingtypes.Commission{Rate: 100000}, // 10%
		MinSelfDelegation: 1000000,
	}

	example.stakingKeeper.SetValidator(example.ctx, validator)

	// Create delegator
	delegatorAddr := []byte("alice")

	// Delegate tokens
	delegateAmount := int64(500000) // 500K tokens
	if err := example.stakingKeeper.Delegate(example.ctx, delegatorAddr, validatorAddr, delegateAmount); err != nil {
		return fmt.Errorf("failed to delegate: %w", err)
	}

	// Check delegation
	delegation, found := example.stakingKeeper.GetDelegation(example.ctx, delegatorAddr, validatorAddr)
	if !found {
		return fmt.Errorf("delegation not found")
	}

	// Check validator after delegation
	validatorAfter, found := example.stakingKeeper.GetValidator(example.ctx, validatorAddr)
	if !found {
		return fmt.Errorf("validator not found")
	}

	fmt.Printf("Delegation created:\n")
	fmt.Printf("Delegator: %s\n", delegation.DelegatorAddress)
	fmt.Printf("Validator: %s\n", delegation.ValidatorAddress)
	fmt.Printf("Shares: %d\n", delegation.Shares)
	fmt.Printf("Validator tokens: %d\n", validatorAfter.Tokens)

	// Undelegate some tokens
	undelegateAmount := int64(200000) // 200K tokens
	if err := example.stakingKeeper.Undelegate(example.ctx, delegatorAddr, validatorAddr, undelegateAmount); err != nil {
		return fmt.Errorf("failed to undelegate: %w", err)
	}

	// Check delegation after undelegation
	delegationAfter, found := example.stakingKeeper.GetDelegation(example.ctx, delegatorAddr, validatorAddr)
	if found {
		fmt.Printf("\nAfter undelegation of %d tokens:\n", undelegateAmount)
		fmt.Printf("Remaining shares: %d\n", delegationAfter.Shares)
	}

	fmt.Println("Staking operations completed successfully!")
	return nil
}

// RunGovernanceOperations demonstrates governance operations
func (example *KeeperIntegrationExample) RunGovernanceOperations() error {
	fmt.Println("\n=== Governance Operations Example ===")

	// Submit a proposal
	proposer := []byte("alice")
	messages := []govtypes.ProposalMessage{
		{
			TypeURL: "/cosmos.gov.v1beta1.TextProposal",
			Value:   []byte("test proposal content"),
		},
	}

	proposalID, err := example.govKeeper.SubmitProposal(
		example.ctx,
		messages,
		"Test Proposal",
		"This is a test proposal for demonstration",
		string(proposer),
	)
	if err != nil {
		return fmt.Errorf("failed to submit proposal: %w", err)
	}

	fmt.Printf("Proposal submitted with ID: %d\n", proposalID)

	// Check proposal
	proposal, found := example.govKeeper.GetProposal(example.ctx, proposalID)
	if !found {
		return fmt.Errorf("proposal not found")
	}

	fmt.Printf("Proposal status: %s\n", proposal.Status.String())

	// Make a deposit
	depositor := []byte("alice")
	depositAmount := []keepertypes.Coin{{Denom: "stake", Amount: 10000000}} // 10M tokens

	_, err = example.govKeeper.AddDeposit(example.ctx, proposalID, depositor, depositAmount)
	if err != nil {
		return fmt.Errorf("failed to add deposit: %w", err)
	}

	// Check proposal status after deposit
	proposalAfterDeposit, found := example.govKeeper.GetProposal(example.ctx, proposalID)
	if !found {
		return fmt.Errorf("proposal not found after deposit")
	}

	fmt.Printf("Proposal status after deposit: %s\n", proposalAfterDeposit.Status.String())

	// Vote on the proposal (if it's in voting period)
	if proposalAfterDeposit.Status == govtypes.StatusVotingPeriod {
		voter := []byte("alice")
		voteOptions := []govtypes.VoteOption{govtypes.OptionYes}

		if err := example.govKeeper.AddVote(example.ctx, proposalID, voter, voteOptions, ""); err != nil {
			return fmt.Errorf("failed to vote: %w", err)
		}

		fmt.Printf("Vote cast on proposal %d\n", proposalID)

		// Tally the votes
		passes, burnDeposits, tallyResult := example.govKeeper.Tally(example.ctx, proposalAfterDeposit)

		fmt.Printf("\nTally Results:\n")
		fmt.Printf("Yes: %d, No: %d, Abstain: %d, NoWithVeto: %d\n",
			tallyResult.YesCount, tallyResult.NoCount, tallyResult.AbstainCount, tallyResult.NoWithVetoCount)
		fmt.Printf("Passes: %t, Burn Deposits: %t\n", passes, burnDeposits)
	}

	fmt.Println("Governance operations completed successfully!")
	return nil
}

// RunCrossModuleOperations demonstrates how modules interact with each other
func (example *KeeperIntegrationExample) RunCrossModuleOperations() error {
	fmt.Println("\n=== Cross-Module Operations Example ===")

	// Scenario: A governance proposal to change staking parameters
	// This shows how governance can interact with other modules

	// First, set up some initial state
	alice := []byte("alice")

	// Give Alice some tokens for staking and governance
	aliceBalance := banktypes.Balance{
		Address: string(alice),
		Denom:   "stake",
		Amount:  50000000, // 50M tokens
	}
	example.bankKeeper.SetBalance(example.ctx, alice, aliceBalance)

	// Create a validator for Alice to delegate to
	validatorAddr := []byte("validator1")
	validator := stakingtypes.Validator{
		OperatorAddress:   string(validatorAddr),
		ConsPubKey:        []byte("pubkey123"),
		Jailed:            false,
		Status:            stakingtypes.Bonded,
		Tokens:            0,
		DelegatorShares:   0,
		Description:       stakingtypes.Description{Moniker: "Validator One"},
		Commission:        stakingtypes.Commission{Rate: 50000}, // 5%
		MinSelfDelegation: 1000000,
	}
	example.stakingKeeper.SetValidator(example.ctx, validator)

	// Alice delegates tokens to get voting power
	delegateAmount := int64(20000000) // 20M tokens
	if err := example.stakingKeeper.Delegate(example.ctx, alice, validatorAddr, delegateAmount); err != nil {
		return fmt.Errorf("failed to delegate for voting power: %w", err)
	}

	// Submit a proposal to change staking parameters
	proposalMessages := []govtypes.ProposalMessage{
		{
			TypeURL: "/cosmos.staking.v1beta1.MsgUpdateParams",
			Value:   []byte("update staking params content"),
		},
	}

	proposalID, err := example.govKeeper.SubmitProposal(
		example.ctx,
		proposalMessages,
		"Update Staking Parameters",
		"Proposal to update the maximum number of validators",
		string(alice),
	)
	if err != nil {
		return fmt.Errorf("failed to submit staking parameter proposal: %w", err)
	}

	fmt.Printf("Cross-module proposal submitted: ID %d\n", proposalID)

	// Make deposit to activate voting
	depositAmount := []keepertypes.Coin{{Denom: "stake", Amount: 10000000}}
	activated, err := example.govKeeper.AddDeposit(example.ctx, proposalID, alice, depositAmount)
	if err != nil {
		return fmt.Errorf("failed to deposit on cross-module proposal: %w", err)
	}

	if activated {
		fmt.Printf("Proposal %d activated for voting\n", proposalID)

		// Vote on the proposal
		voteOptions := []govtypes.VoteOption{govtypes.OptionYes}
		if err := example.govKeeper.AddVote(example.ctx, proposalID, alice, voteOptions, "Supporting this change"); err != nil {
			return fmt.Errorf("failed to vote on cross-module proposal: %w", err)
		}

		fmt.Printf("Voted on cross-module proposal\n")
	}

	// Demonstrate module interoperability stats
	fmt.Printf("\nModule Interoperability Summary:\n")

	// Bank module state
	aliceStakeBalance := example.bankKeeper.GetBalance(example.ctx, alice, "stake")
	fmt.Printf("Alice's remaining stake balance: %d\n", aliceStakeBalance)

	// Staking module state
	delegation, found := example.stakingKeeper.GetDelegation(example.ctx, alice, validatorAddr)
	if found {
		fmt.Printf("Alice's delegation shares: %d\n", delegation.Shares)
	}

	// Governance module state
	proposal, found := example.govKeeper.GetProposal(example.ctx, proposalID)
	if found {
		fmt.Printf("Proposal status: %s\n", proposal.Status.String())
		fmt.Printf("Total deposits: %v\n", proposal.TotalDeposit)
	}

	votes := example.govKeeper.GetVotes(example.ctx, proposalID)
	fmt.Printf("Number of votes cast: %d\n", len(votes))

	fmt.Println("Cross-module operations completed successfully!")
	return nil
}

// RunCompleteExample runs all integration examples
func (example *KeeperIntegrationExample) RunCompleteExample() error {
	fmt.Println("Starting Keeper Integration Example...")
	fmt.Println("This example demonstrates how Pulsar's keeper pattern enables secure module interactions")
	fmt.Println("based on Cosmos SDK design principles.")

	if err := example.RunBankingOperations(); err != nil {
		return err
	}

	if err := example.RunStakingOperations(); err != nil {
		return err
	}

	if err := example.RunGovernanceOperations(); err != nil {
		return err
	}

	if err := example.RunCrossModuleOperations(); err != nil {
		return err
	}

	fmt.Println("\n=== Integration Example Complete ===")
	fmt.Println("All keeper operations completed successfully!")
	fmt.Println("The example demonstrated:")
	fmt.Println("1. Bank keeper: Balance management and transfers")
	fmt.Println("2. Staking keeper: Delegation, undelegation, and validator management")
	fmt.Println("3. Governance keeper: Proposal submission, deposits, and voting")
	fmt.Println("4. Cross-module interactions: How modules work together securely")

	return nil
}

// DemonstrateKeeperSecurity shows the security features of the keeper pattern
func (example *KeeperIntegrationExample) DemonstrateKeeperSecurity() {
	fmt.Println("\n=== Keeper Security Demonstration ===")

	fmt.Println("Keeper Pattern Security Features:")
	fmt.Println("1. Object Capabilities: Each keeper only has access to its own store")
	fmt.Println("2. Interface-based Access: Modules can only access other modules through well-defined interfaces")
	fmt.Println("3. Permission Management: Module accounts have specific permissions (minter, burner, etc.)")
	fmt.Println("4. Store Isolation: Each module has its own key-value store namespace")
	fmt.Println("5. Authority Checks: Critical operations require proper authorization")

	// Show how keepers are isolated
	fmt.Printf("\nStore Key Isolation:\n")
	fmt.Printf("Bank Store Key: %s\n", example.bankKeeper.GetStoreKey().Name())
	fmt.Printf("Staking Store Key: %s\n", example.stakingKeeper.GetStoreKey().Name())
	fmt.Printf("Gov Store Key: %s\n", example.govKeeper.GetStoreKey().Name())

	fmt.Println("\nEach keeper can only access its own store, ensuring module isolation and security.")
}

// Simple memory store implementations for testing

// SimpleKVStore implements a basic in-memory KVStore for testing
type SimpleKVStore struct {
	data map[string][]byte
}

// NewSimpleKVStore creates a new simple memory store
func NewSimpleKVStore() *SimpleKVStore {
	return &SimpleKVStore{
		data: make(map[string][]byte),
	}
}

// Get implements types.KVStore
func (s *SimpleKVStore) Get(key []byte) []byte {
	if value, exists := s.data[string(key)]; exists {
		// Return a copy to avoid aliasing
		result := make([]byte, len(value))
		copy(result, value)
		return result
	}
	return nil
}

// Has implements types.KVStore
func (s *SimpleKVStore) Has(key []byte) bool {
	_, exists := s.data[string(key)]
	return exists
}

// Set implements types.KVStore
func (s *SimpleKVStore) Set(key, value []byte) {
	// Store a copy to avoid aliasing
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)
	s.data[string(key)] = valueCopy
}

// Delete implements types.KVStore
func (s *SimpleKVStore) Delete(key []byte) {
	delete(s.data, string(key))
}

// Iterator implements types.KVStore
func (s *SimpleKVStore) Iterator(start, end []byte) storetypes.Iterator {
	return &SimpleIterator{
		data:    s.data,
		start:   start,
		end:     end,
		keys:    []string{},
		current: 0,
	}
}

// ReverseIterator implements types.KVStore
func (s *SimpleKVStore) ReverseIterator(start, end []byte) storetypes.Iterator {
	return &SimpleIterator{
		data:    s.data,
		start:   start,
		end:     end,
		keys:    []string{},
		reverse: true,
		current: 0,
	}
}

// CacheWrap implements types.KVStore
func (s *SimpleKVStore) CacheWrap() storetypes.CacheWrap {
	return &SimpleCacheWrap{parent: s}
}

// GetStoreType implements types.KVStore
func (s *SimpleKVStore) GetStoreType() storetypes.StoreType {
	return storetypes.StoreTypeMemory
}

// SimpleCacheWrap is a simple cache wrapper for testing
type SimpleCacheWrap struct {
	parent  *SimpleKVStore
	cache   map[string][]byte
	deletes map[string]bool
}

// Get implements types.CacheWrap
func (cw *SimpleCacheWrap) Get(key []byte) []byte {
	if cw.cache == nil {
		cw.cache = make(map[string][]byte)
		cw.deletes = make(map[string]bool)
	}

	keyStr := string(key)
	if cw.deletes[keyStr] {
		return nil
	}

	if value, exists := cw.cache[keyStr]; exists {
		return value
	}

	return cw.parent.Get(key)
}

// Has implements types.CacheWrap
func (cw *SimpleCacheWrap) Has(key []byte) bool {
	return cw.Get(key) != nil
}

// Set implements types.CacheWrap
func (cw *SimpleCacheWrap) Set(key, value []byte) {
	if cw.cache == nil {
		cw.cache = make(map[string][]byte)
		cw.deletes = make(map[string]bool)
	}

	keyStr := string(key)
	cw.cache[keyStr] = value
	delete(cw.deletes, keyStr)
}

// Delete implements types.CacheWrap
func (cw *SimpleCacheWrap) Delete(key []byte) {
	if cw.cache == nil {
		cw.cache = make(map[string][]byte)
		cw.deletes = make(map[string]bool)
	}

	keyStr := string(key)
	cw.deletes[keyStr] = true
	delete(cw.cache, keyStr)
}

// Iterator implements types.CacheWrap
func (cw *SimpleCacheWrap) Iterator(start, end []byte) storetypes.Iterator {
	return cw.parent.Iterator(start, end) // Simplified
}

// ReverseIterator implements types.CacheWrap
func (cw *SimpleCacheWrap) ReverseIterator(start, end []byte) storetypes.Iterator {
	return cw.parent.ReverseIterator(start, end) // Simplified
}

// CacheWrap implements types.CacheWrap
func (cw *SimpleCacheWrap) CacheWrap() storetypes.CacheWrap {
	return cw
}

// Write implements types.CacheWrap
func (cw *SimpleCacheWrap) Write() {
	if cw.cache == nil {
		return
	}

	// Write cached changes to parent
	for keyStr, value := range cw.cache {
		cw.parent.Set([]byte(keyStr), value)
	}

	// Apply deletes
	for keyStr := range cw.deletes {
		cw.parent.Delete([]byte(keyStr))
	}

	// Clear cache
	cw.cache = make(map[string][]byte)
	cw.deletes = make(map[string]bool)
}

// SimpleIterator implements Iterator interface
type SimpleIterator struct {
	data    map[string][]byte
	start   []byte
	end     []byte
	keys    []string
	reverse bool
	current int
}

// Domain implements types.Iterator
func (i *SimpleIterator) Domain() ([]byte, []byte) {
	return i.start, i.end
}

// Valid implements types.Iterator
func (i *SimpleIterator) Valid() bool {
	return i.current < len(i.keys)
}

// Next implements types.Iterator
func (i *SimpleIterator) Next() {
	i.current++
}

// Key implements types.Iterator
func (i *SimpleIterator) Key() []byte {
	if !i.Valid() {
		return nil
	}
	return []byte(i.keys[i.current])
}

// Value implements types.Iterator
func (i *SimpleIterator) Value() []byte {
	if !i.Valid() {
		return nil
	}
	return i.data[i.keys[i.current]]
}

// Close implements types.Iterator
func (i *SimpleIterator) Close() error {
	return nil
}

// SimpleMultiStore implements a basic MultiStore for testing
type SimpleMultiStore struct {
	stores map[string]*SimpleKVStore
}

// NewSimpleMultiStore creates a new simple multi-store
func NewSimpleMultiStore() *SimpleMultiStore {
	return &SimpleMultiStore{
		stores: make(map[string]*SimpleKVStore),
	}
}

// MountStore mounts a store with the given key
func (ms *SimpleMultiStore) MountStore(key storetypes.StoreKey) {
	ms.stores[key.Name()] = NewSimpleKVStore()
}

// GetKVStore implements types.MultiStore
func (ms *SimpleMultiStore) GetKVStore(key storetypes.StoreKey) storetypes.KVStore {
	if store, exists := ms.stores[key.Name()]; exists {
		return store
	}
	// Return nil if store not found - this will be caught by the keeper
	return nil
}

// CacheMultiStore implements types.MultiStore
func (ms *SimpleMultiStore) CacheMultiStore() storetypes.CacheMultiStore {
	// Return a simple implementation for testing
	return &SimpleCacheMultiStore{parent: ms}
}

// CacheWrap implements types.CacheWrapper
func (ms *SimpleMultiStore) CacheWrap() storetypes.CacheWrap {
	return ms.CacheMultiStore()
}

// GetStoreType implements types.Store
func (ms *SimpleMultiStore) GetStoreType() storetypes.StoreType {
	return storetypes.StoreTypeMulti
}

// GetStore implements types.MultiStore
func (ms *SimpleMultiStore) GetStore(key storetypes.StoreKey) storetypes.Store {
	return ms.GetKVStore(key)
}

// TracingEnabled implements types.MultiStore
func (ms *SimpleMultiStore) TracingEnabled() bool {
	return false
}

// SetTracer implements types.MultiStore
func (ms *SimpleMultiStore) SetTracer(w storetypes.TraceWriter) storetypes.MultiStore {
	return ms
}

// SimpleCacheMultiStore is a simple cache implementation for testing
type SimpleCacheMultiStore struct {
	parent *SimpleMultiStore
	cache  map[string]*SimpleKVStore
}

// GetKVStore implements types.CacheMultiStore
func (cms *SimpleCacheMultiStore) GetKVStore(key storetypes.StoreKey) storetypes.KVStore {
	if cms.cache == nil {
		cms.cache = make(map[string]*SimpleKVStore)
	}

	// Return cached store if exists
	if store, exists := cms.cache[key.Name()]; exists {
		return store
	}

	// Create cached copy
	parentStore := cms.parent.GetKVStore(key)
	if parentStore == nil {
		return nil
	}

	// For simplicity, just return the parent store
	// In a real implementation, this would be a cached overlay
	return parentStore
}

// Write implements types.CacheMultiStore
func (cms *SimpleCacheMultiStore) Write() {
	// In a real implementation, this would write cached changes to parent
	// For testing, we do nothing
}

// CacheMultiStore implements types.CacheMultiStore
func (cms *SimpleCacheMultiStore) CacheMultiStore() storetypes.CacheMultiStore {
	return cms
}

// CacheWrap implements types.CacheWrapper
func (cms *SimpleCacheMultiStore) CacheWrap() storetypes.CacheWrap {
	return cms
}

// GetStoreType implements types.Store
func (cms *SimpleCacheMultiStore) GetStoreType() storetypes.StoreType {
	return storetypes.StoreTypeMulti
}

// GetStore implements types.MultiStore
func (cms *SimpleCacheMultiStore) GetStore(key storetypes.StoreKey) storetypes.Store {
	return cms.GetKVStore(key)
}

// TracingEnabled implements types.MultiStore
func (cms *SimpleCacheMultiStore) TracingEnabled() bool {
	return false
}

// SetTracer implements types.MultiStore
func (cms *SimpleCacheMultiStore) SetTracer(w storetypes.TraceWriter) storetypes.MultiStore {
	return cms
}

// StakingKeeperWrapper wraps the staking keeper to match the governance interface
type StakingKeeperWrapper struct {
	keeper *stakingkeeper.Keeper
}

// GetBondedValidatorsByPower implements govtypes.StakingKeeper
func (w *StakingKeeperWrapper) GetBondedValidatorsByPower(ctx keepertypes.Context) []interface{} {
	validators := w.keeper.GetBondedValidatorsByPower(ctx)
	result := make([]interface{}, len(validators))
	for i, v := range validators {
		result[i] = v
	}
	return result
}

// GetLastTotalPower implements govtypes.StakingKeeper
func (w *StakingKeeperWrapper) GetLastTotalPower(ctx keepertypes.Context) int64 {
	return w.keeper.GetLastTotalPower(ctx)
}
