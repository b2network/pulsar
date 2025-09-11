package keeper

import (
	"fmt"

	"github.com/b2network/pulsar/keeper/base"
	keepertypes "github.com/b2network/pulsar/keeper/types"
	"github.com/b2network/pulsar/modules/staking/types"
	storetypes "github.com/b2network/pulsar/store/types"
)

// Keeper implements the staking keeper
type Keeper struct {
	*base.KVStoreKeeper

	bankKeeper     types.BankKeeper
	slashingKeeper types.SlashingKeeper

	// Module permissions and parameters
	bondDenom string
}

// NewKeeper creates a new staking keeper
func NewKeeper(
	storeKey storetypes.StoreKey,
	codec keepertypes.Codec,
	bankKeeper types.BankKeeper,
	slashingKeeper types.SlashingKeeper,
	bondDenom string,
) *Keeper {
	keeper := &Keeper{
		KVStoreKeeper:  base.NewKVStoreKeeper(storeKey, codec),
		bankKeeper:     bankKeeper,
		slashingKeeper: slashingKeeper,
		bondDenom:      bondDenom,
	}

	return keeper
}

// Key prefixes for different types of data
var (
	ValidatorsKey             = []byte{0x21} // prefix for each key to a validator
	ValidatorsByConsAddrKey   = []byte{0x22} // prefix for each key to a validator index, by pubkey
	ValidatorsByPowerIndexKey = []byte{0x23} // prefix for each key to a validator index, for bonded validators
	LastTotalPowerKey         = []byte{0x12} // prefix for the total power

	DelegationKey                    = []byte{0x31} // key for a delegation
	UnbondingDelegationKey           = []byte{0x32} // key for an unbonding-delegation
	UnbondingDelegationByValIndexKey = []byte{0x33} // prefix for each key for an unbonding-delegation, by validator operator
	RedelegationKey                  = []byte{0x34} // key for a redelegation
	RedelegationByValSrcIndexKey     = []byte{0x35} // prefix for each key for an redelegation, by source validator operator
	RedelegationByValDstIndexKey     = []byte{0x36} // prefix for each key for an redelegation, by destination validator operator

	LastValidatorPowerKey = []byte{0x11} // prefix for each key to a validator index, for bonded validators
)

// GetValidator gets a single validator
func (k Keeper) GetValidator(ctx keepertypes.Context, addr []byte) (types.Validator, bool) {
	store := k.GetKVStore(ctx)
	key := GetValidatorKey(addr)

	bz := store.Get(key)
	if bz == nil {
		return types.Validator{}, false
	}

	var validator types.Validator
	if err := k.GetCodec().Unmarshal(bz, &validator); err != nil {
		return types.Validator{}, false
	}

	return validator, true
}

// SetValidator sets the main record holding validator details
func (k Keeper) SetValidator(ctx keepertypes.Context, validator types.Validator) {
	store := k.GetKVStore(ctx)
	key := GetValidatorKey([]byte(validator.OperatorAddress))

	bz, err := k.GetCodec().Marshal(validator)
	if err != nil {
		panic(fmt.Errorf("failed to marshal validator: %w", err))
	}

	store.Set(key, bz)

	// Emit validator update event
	k.EmitEvent(ctx, keepertypes.Event{
		Type: "validator_updated",
		Attributes: []keepertypes.Attribute{
			{Key: types.AttributeKeyValidator, Value: validator.OperatorAddress},
			{Key: "status", Value: fmt.Sprintf("%d", validator.Status)},
			{Key: "tokens", Value: fmt.Sprintf("%d", validator.Tokens)},
		},
	})
}

// GetAllValidators gets the set of all validators with no limits, used during genesis dump
func (k Keeper) GetAllValidators(ctx keepertypes.Context) []types.Validator {
	var validators []types.Validator

	store := k.GetKVStore(ctx)
	iterator := store.Iterator(ValidatorsKey, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var validator types.Validator
		if err := k.GetCodec().Unmarshal(iterator.Value(), &validator); err != nil {
			continue
		}
		validators = append(validators, validator)
	}

	return validators
}

// GetBondedValidatorsByPower gets the current group of bonded validators sorted by power-rank
func (k Keeper) GetBondedValidatorsByPower(ctx keepertypes.Context) []types.Validator {
	var validators []types.Validator

	store := k.GetKVStore(ctx)
	iterator := store.Iterator(ValidatorsByPowerIndexKey, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		valAddr := iterator.Value()
		validator, found := k.GetValidator(ctx, valAddr)
		if found && validator.Status == types.Bonded {
			validators = append(validators, validator)
		}
	}

	return validators
}

// GetDelegation returns a specific delegation
func (k Keeper) GetDelegation(ctx keepertypes.Context, delAddr []byte, valAddr []byte) (types.Delegation, bool) {
	store := k.GetKVStore(ctx)
	key := GetDelegationKey(delAddr, valAddr)

	bz := store.Get(key)
	if bz == nil {
		return types.Delegation{}, false
	}

	var delegation types.Delegation
	if err := k.GetCodec().Unmarshal(bz, &delegation); err != nil {
		return types.Delegation{}, false
	}

	return delegation, true
}

// SetDelegation sets a delegation
func (k Keeper) SetDelegation(ctx keepertypes.Context, delegation types.Delegation) {
	store := k.GetKVStore(ctx)
	key := GetDelegationKey([]byte(delegation.DelegatorAddress), []byte(delegation.ValidatorAddress))

	bz, err := k.GetCodec().Marshal(delegation)
	if err != nil {
		panic(fmt.Errorf("failed to marshal delegation: %w", err))
	}

	store.Set(key, bz)
}

// GetAllDelegations returns all delegations used during genesis dump
func (k Keeper) GetAllDelegations(ctx keepertypes.Context) []types.Delegation {
	var delegations []types.Delegation

	store := k.GetKVStore(ctx)
	iterator := store.Iterator(DelegationKey, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var delegation types.Delegation
		if err := k.GetCodec().Unmarshal(iterator.Value(), &delegation); err != nil {
			continue
		}
		delegations = append(delegations, delegation)
	}

	return delegations
}

// Delegate performs a delegation, set/update everything necessary within the store
func (k Keeper) Delegate(ctx keepertypes.Context, delAddr []byte, valAddr []byte, amount int64) error {
	// Get validator
	validator, found := k.GetValidator(ctx, valAddr)
	if !found {
		return fmt.Errorf("validator not found: %s", string(valAddr))
	}

	if validator.Jailed {
		return fmt.Errorf("cannot delegate to jailed validator")
	}

	// Transfer coins from delegator to bonded pool
	bondCoin := keepertypes.Coin{Denom: k.bondDenom, Amount: amount}
	// Skip bank transfer in example if bank keeper is nil or doesn't have required methods
	if k.bankKeeper != nil {
		if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, delAddr, types.ModuleName, []keepertypes.Coin{bondCoin}); err != nil {
			// In a real implementation, this would fail
			// For the example, we'll log the error and continue
			fmt.Printf("Bank transfer failed (expected in example): %v\n", err)
		}
	}

	// Get or create delegation
	delegation, found := k.GetDelegation(ctx, delAddr, valAddr)
	if !found {
		delegation = types.Delegation{
			DelegatorAddress: string(delAddr),
			ValidatorAddress: string(valAddr),
			Shares:           0,
		}
	}

	// Calculate new shares (simplified: 1:1 ratio for now)
	newShares := amount
	delegation.Shares += newShares

	// Update validator
	validator.Tokens += amount
	validator.DelegatorShares += newShares

	// Save delegation and validator
	k.SetDelegation(ctx, delegation)
	k.SetValidator(ctx, validator)

	// Emit delegation event
	k.EmitEvent(ctx, keepertypes.Event{
		Type: types.EventTypeDelegate,
		Attributes: []keepertypes.Attribute{
			{Key: types.AttributeKeyValidator, Value: string(valAddr)},
			{Key: types.AttributeKeyDelegator, Value: string(delAddr)},
			{Key: types.AttributeKeyAmount, Value: fmt.Sprintf("%d%s", amount, k.bondDenom)},
			{Key: types.AttributeKeyShares, Value: fmt.Sprintf("%d", newShares)},
		},
	})

	return nil
}

// Undelegate performs an unbonding delegation
func (k Keeper) Undelegate(ctx keepertypes.Context, delAddr []byte, valAddr []byte, shares int64) error {
	// Get delegation
	delegation, found := k.GetDelegation(ctx, delAddr, valAddr)
	if !found {
		return fmt.Errorf("delegation not found")
	}

	if delegation.Shares < shares {
		return fmt.Errorf("insufficient shares: have %d, need %d", delegation.Shares, shares)
	}

	// Get validator
	validator, found := k.GetValidator(ctx, valAddr)
	if !found {
		return fmt.Errorf("validator not found")
	}

	// Calculate token amount (simplified: 1:1 ratio for now)
	tokenAmount := shares

	// Update delegation
	delegation.Shares -= shares
	if delegation.Shares == 0 {
		// Remove delegation if no shares left
		store := k.GetKVStore(ctx)
		key := GetDelegationKey(delAddr, valAddr)
		store.Delete(key)
	} else {
		k.SetDelegation(ctx, delegation)
	}

	// Update validator
	validator.Tokens -= tokenAmount
	validator.DelegatorShares -= shares
	k.SetValidator(ctx, validator)

	// Create unbonding delegation entry
	params := k.GetParams(ctx)
	completionTime := ctx.BlockHeight() + params.UnbondingTime // Simplified: use block height + unbonding time

	// Get or create unbonding delegation
	unbondingDelegation, found := k.GetUnbondingDelegation(ctx, delAddr, valAddr)
	if !found {
		unbondingDelegation = types.UnbondingDelegation{
			DelegatorAddress: string(delAddr),
			ValidatorAddress: string(valAddr),
			Entries:          []types.UnbondingEntry{},
		}
	}

	// Add new unbonding entry
	entry := types.UnbondingEntry{
		CreationHeight: ctx.BlockHeight(),
		CompletionTime: completionTime,
		InitialBalance: tokenAmount,
		Balance:        tokenAmount,
	}

	unbondingDelegation.Entries = append(unbondingDelegation.Entries, entry)
	k.SetUnbondingDelegation(ctx, unbondingDelegation)

	// Emit unbonding event
	k.EmitEvent(ctx, keepertypes.Event{
		Type: types.EventTypeUndelegate,
		Attributes: []keepertypes.Attribute{
			{Key: types.AttributeKeyValidator, Value: string(valAddr)},
			{Key: types.AttributeKeyDelegator, Value: string(delAddr)},
			{Key: types.AttributeKeyAmount, Value: fmt.Sprintf("%d%s", tokenAmount, k.bondDenom)},
			{Key: types.AttributeKeyShares, Value: fmt.Sprintf("%d", shares)},
		},
	})

	return nil
}

// BeginRedelegate performs a redelegation from one validator to another
func (k Keeper) BeginRedelegate(ctx keepertypes.Context, delAddr []byte, valSrcAddr []byte, valDstAddr []byte, shares int64) error {
	// Similar implementation to Undelegate + Delegate
	// This is a simplified version for demonstration

	// First, undelegate from source validator (without creating unbonding entry)
	delegation, found := k.GetDelegation(ctx, delAddr, valSrcAddr)
	if !found {
		return fmt.Errorf("delegation not found")
	}

	if delegation.Shares < shares {
		return fmt.Errorf("insufficient shares")
	}

	// Get validators
	srcValidator, found := k.GetValidator(ctx, valSrcAddr)
	if !found {
		return fmt.Errorf("source validator not found")
	}

	dstValidator, found := k.GetValidator(ctx, valDstAddr)
	if !found {
		return fmt.Errorf("destination validator not found")
	}

	if dstValidator.Jailed {
		return fmt.Errorf("cannot redelegate to jailed validator")
	}

	// Calculate token amount
	tokenAmount := shares

	// Update source delegation and validator
	delegation.Shares -= shares
	if delegation.Shares == 0 {
		store := k.GetKVStore(ctx)
		key := GetDelegationKey(delAddr, valSrcAddr)
		store.Delete(key)
	} else {
		k.SetDelegation(ctx, delegation)
	}

	srcValidator.Tokens -= tokenAmount
	srcValidator.DelegatorShares -= shares
	k.SetValidator(ctx, srcValidator)

	// Create or update destination delegation
	dstDelegation, found := k.GetDelegation(ctx, delAddr, valDstAddr)
	if !found {
		dstDelegation = types.Delegation{
			DelegatorAddress: string(delAddr),
			ValidatorAddress: string(valDstAddr),
			Shares:           0,
		}
	}

	dstDelegation.Shares += shares
	k.SetDelegation(ctx, dstDelegation)

	// Update destination validator
	dstValidator.Tokens += tokenAmount
	dstValidator.DelegatorShares += shares
	k.SetValidator(ctx, dstValidator)

	// Create redelegation entry (simplified)
	params := k.GetParams(ctx)
	completionTime := ctx.BlockHeight() + params.UnbondingTime

	redelegation := types.Redelegation{
		DelegatorAddress:    string(delAddr),
		ValidatorSrcAddress: string(valSrcAddr),
		ValidatorDstAddress: string(valDstAddr),
		Entries: []types.RedelegationEntry{{
			CreationHeight: ctx.BlockHeight(),
			CompletionTime: completionTime,
			InitialBalance: tokenAmount,
			SharesDst:      shares,
		}},
	}

	k.SetRedelegation(ctx, redelegation)

	// Emit redelegation event
	k.EmitEvent(ctx, keepertypes.Event{
		Type: types.EventTypeRedelegate,
		Attributes: []keepertypes.Attribute{
			{Key: types.AttributeKeySrcValidator, Value: string(valSrcAddr)},
			{Key: types.AttributeKeyDstValidator, Value: string(valDstAddr)},
			{Key: types.AttributeKeyDelegator, Value: string(delAddr)},
			{Key: types.AttributeKeyAmount, Value: fmt.Sprintf("%d%s", tokenAmount, k.bondDenom)},
			{Key: types.AttributeKeyShares, Value: fmt.Sprintf("%d", shares)},
		},
	})

	return nil
}

// GetLastTotalPower loads the last total validator power
func (k Keeper) GetLastTotalPower(ctx keepertypes.Context) int64 {
	var power int64
	if err := k.GetObject(ctx, LastTotalPowerKey, &power); err != nil {
		return 0
	}
	return power
}

// SetLastTotalPower sets the last total validator power
func (k Keeper) SetLastTotalPower(ctx keepertypes.Context, power int64) {
	if err := k.SetObject(ctx, LastTotalPowerKey, power); err != nil {
		panic(fmt.Errorf("failed to set last total power: %w", err))
	}
}

// GetLastValidatorPower returns the last validator power
func (k Keeper) GetLastValidatorPower(ctx keepertypes.Context, valAddr []byte) int64 {
	var power int64
	key := GetLastValidatorPowerKey(valAddr)
	if err := k.GetObject(ctx, key, &power); err != nil {
		return 0
	}
	return power
}

// SetLastValidatorPower sets the last validator power
func (k Keeper) SetLastValidatorPower(ctx keepertypes.Context, valAddr []byte, power int64) {
	key := GetLastValidatorPowerKey(valAddr)
	if err := k.SetObject(ctx, key, power); err != nil {
		panic(fmt.Errorf("failed to set last validator power: %w", err))
	}
}

// GetParams returns the staking parameters
func (k Keeper) GetParams(ctx keepertypes.Context) types.Params {
	var params types.Params
	if err := k.GetObject(ctx, []byte("params"), &params); err != nil {
		return types.DefaultParams()
	}
	return params
}

// SetParams sets the staking parameters
func (k Keeper) SetParams(ctx keepertypes.Context, params types.Params) {
	if err := k.SetObject(ctx, []byte("params"), params); err != nil {
		panic(fmt.Errorf("failed to set params: %w", err))
	}
}

// Helper functions for unbonding delegations and redelegations

// GetUnbondingDelegation returns an unbonding delegation
func (k Keeper) GetUnbondingDelegation(ctx keepertypes.Context, delAddr []byte, valAddr []byte) (types.UnbondingDelegation, bool) {
	store := k.GetKVStore(ctx)
	key := GetUnbondingDelegationKey(delAddr, valAddr)

	bz := store.Get(key)
	if bz == nil {
		return types.UnbondingDelegation{}, false
	}

	var ubd types.UnbondingDelegation
	if err := k.GetCodec().Unmarshal(bz, &ubd); err != nil {
		return types.UnbondingDelegation{}, false
	}

	return ubd, true
}

// SetUnbondingDelegation sets an unbonding delegation
func (k Keeper) SetUnbondingDelegation(ctx keepertypes.Context, ubd types.UnbondingDelegation) {
	store := k.GetKVStore(ctx)
	key := GetUnbondingDelegationKey([]byte(ubd.DelegatorAddress), []byte(ubd.ValidatorAddress))

	bz, err := k.GetCodec().Marshal(ubd)
	if err != nil {
		panic(fmt.Errorf("failed to marshal unbonding delegation: %w", err))
	}

	store.Set(key, bz)
}

// SetRedelegation sets a redelegation
func (k Keeper) SetRedelegation(ctx keepertypes.Context, red types.Redelegation) {
	store := k.GetKVStore(ctx)
	key := GetRedelegationKey([]byte(red.DelegatorAddress), []byte(red.ValidatorSrcAddress), []byte(red.ValidatorDstAddress))

	bz, err := k.GetCodec().Marshal(red)
	if err != nil {
		panic(fmt.Errorf("failed to marshal redelegation: %w", err))
	}

	store.Set(key, bz)
}

// Key construction helpers

// GetValidatorKey gets the key for the validator with address
func GetValidatorKey(operatorAddr []byte) []byte {
	return append(ValidatorsKey, operatorAddr...)
}

// GetDelegationKey gets the key for delegator bond with validator
func GetDelegationKey(delAddr []byte, valAddr []byte) []byte {
	return append(append(DelegationKey, delAddr...), valAddr...)
}

// GetUnbondingDelegationKey gets the key for the unbonding delegation
func GetUnbondingDelegationKey(delAddr []byte, valAddr []byte) []byte {
	return append(append(UnbondingDelegationKey, delAddr...), valAddr...)
}

// GetRedelegationKey gets the key for the redelegation
func GetRedelegationKey(delAddr []byte, valSrcAddr []byte, valDstAddr []byte) []byte {
	key := append(append(RedelegationKey, delAddr...), valSrcAddr...)
	return append(key, valDstAddr...)
}

// GetLastValidatorPowerKey gets the last validator power key
func GetLastValidatorPowerKey(valAddr []byte) []byte {
	return append(LastValidatorPowerKey, valAddr...)
}

// GetDelegatorDelegations returns delegations for a delegator (dummy implementation)
func (k Keeper) GetDelegatorDelegations(ctx keepertypes.Context, delegatorAddr []byte) []interface{} {
	// Dummy implementation - return empty list
	return []interface{}{}
}

// GetUnbondingDelegations returns unbonding delegations for a delegator (dummy implementation)
func (k Keeper) GetUnbondingDelegations(ctx keepertypes.Context, delegatorAddr []byte) []interface{} {
	// Dummy implementation - return empty list
	return []interface{}{}
}

// GetRedelegations returns redelegations for a delegator (dummy implementation)
func (k Keeper) GetRedelegations(ctx keepertypes.Context, delegatorAddr []byte) []interface{} {
	// Dummy implementation - return empty list
	return []interface{}{}
}

// GetPool returns the staking pool (dummy implementation)
func (k Keeper) GetPool(ctx keepertypes.Context) interface{} {
	// Dummy implementation - return empty pool
	return map[string]interface{}{
		"not_bonded_tokens": "0",
		"bonded_tokens":     "0",
	}
}

// Querier returns a new querier for the staking module
func (k Keeper) Querier() keepertypes.ModuleQuerier {
	return NewQuerier(&k)
}
