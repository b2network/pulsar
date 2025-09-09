package base

import (
	"fmt"

	keepertypes "github.com/b2network/pulsar/keeper/types"
	"github.com/b2network/pulsar/store/types"
)

// BaseKeeper provides basic functionality for all keepers
type BaseKeeper struct {
	storeKey types.StoreKey
	codec    keepertypes.Codec

	// Optional authority for authorization
	authority keepertypes.Authority
}

// NewBaseKeeper creates a new BaseKeeper
func NewBaseKeeper(storeKey types.StoreKey, codec keepertypes.Codec) *BaseKeeper {
	if storeKey == nil {
		panic("store key cannot be nil")
	}
	if codec == nil {
		panic("codec cannot be nil")
	}

	return &BaseKeeper{
		storeKey: storeKey,
		codec:    codec,
	}
}

// WithAuthority sets the authority for this keeper
func (k *BaseKeeper) WithAuthority(authority keepertypes.Authority) *BaseKeeper {
	k.authority = authority
	return k
}

// GetStoreKey returns the store key
func (k *BaseKeeper) GetStoreKey() types.StoreKey {
	return k.storeKey
}

// GetCodec returns the codec
func (k *BaseKeeper) GetCodec() keepertypes.Codec {
	return k.codec
}

// GetAuthority returns the authority
func (k *BaseKeeper) GetAuthority() keepertypes.Authority {
	return k.authority
}

// KVStoreKeeper extends BaseKeeper with KV store operations
type KVStoreKeeper struct {
	*BaseKeeper
}

// NewKVStoreKeeper creates a new KVStoreKeeper
func NewKVStoreKeeper(storeKey types.StoreKey, codec keepertypes.Codec) *KVStoreKeeper {
	return &KVStoreKeeper{
		BaseKeeper: NewBaseKeeper(storeKey, codec),
	}
}

// GetKVStore returns the KVStore for this keeper
func (k *KVStoreKeeper) GetKVStore(ctx keepertypes.Context) types.KVStore {
	store := ctx.KVStore(k.storeKey)
	if store == nil {
		panic(fmt.Sprintf("store not found for key: %s", k.storeKey.Name()))
	}

	return store
}

// Get retrieves a value by key
func (k *KVStoreKeeper) Get(ctx keepertypes.Context, key []byte) []byte {
	store := k.GetKVStore(ctx)
	return store.Get(key)
}

// Set stores a key-value pair
func (k *KVStoreKeeper) Set(ctx keepertypes.Context, key []byte, value []byte) {
	store := k.GetKVStore(ctx)
	store.Set(key, value)
}

// Delete removes a key
func (k *KVStoreKeeper) Delete(ctx keepertypes.Context, key []byte) {
	store := k.GetKVStore(ctx)
	store.Delete(key)
}

// Has checks if a key exists
func (k *KVStoreKeeper) Has(ctx keepertypes.Context, key []byte) bool {
	store := k.GetKVStore(ctx)
	return store.Has(key)
}

// Iterator creates an iterator over a key range
func (k *KVStoreKeeper) Iterator(ctx keepertypes.Context, start, end []byte) types.Iterator {
	store := k.GetKVStore(ctx)
	return store.Iterator(start, end)
}

// GetObject retrieves and deserializes an object by key
func (k *KVStoreKeeper) GetObject(ctx keepertypes.Context, key []byte, obj interface{}) error {
	bz := k.Get(ctx, key)
	if bz == nil {
		return fmt.Errorf("key not found: %x", key)
	}

	return k.codec.Unmarshal(bz, obj)
}

// SetObject serializes and stores an object by key
func (k *KVStoreKeeper) SetObject(ctx keepertypes.Context, key []byte, obj interface{}) error {
	bz, err := k.codec.Marshal(obj)
	if err != nil {
		return fmt.Errorf("failed to marshal object: %w", err)
	}

	k.Set(ctx, key, bz)
	return nil
}

// IterateObjects iterates over objects in a key range
func (k *KVStoreKeeper) IterateObjects(ctx keepertypes.Context, start, end []byte, cb func(key []byte, obj interface{}) bool, objFactory func() interface{}) {
	iterator := k.Iterator(ctx, start, end)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		obj := objFactory()
		if err := k.codec.Unmarshal(iterator.Value(), obj); err != nil {
			continue // Skip invalid objects
		}

		if !cb(iterator.Key(), obj) {
			break
		}
	}
}

// ValidateAuthority checks if the caller has authority
func (k *BaseKeeper) ValidateAuthority(caller string) error {
	if k.authority == nil {
		return nil // No authority required
	}

	if !k.authority.HasAuthority(caller) {
		return fmt.Errorf("unauthorized: %s does not have authority", caller)
	}

	return nil
}

// Logger returns a logger for this keeper
func (k *BaseKeeper) Logger(ctx keepertypes.Context) keepertypes.Logger {
	return ctx.Logger()
}

// EmitEvent emits an event
func (k *BaseKeeper) EmitEvent(ctx keepertypes.Context, event keepertypes.Event) {
	ctx.EventManager().EmitEvent(event)
}

// EmitEvents emits multiple events
func (k *BaseKeeper) EmitEvents(ctx keepertypes.Context, events keepertypes.Events) {
	ctx.EventManager().EmitEvents(events)
}

// NoOpLogger is a logger that does nothing
type NoOpLogger struct{}

func (l *NoOpLogger) Debug(msg string, keyvals ...interface{}) {}
func (l *NoOpLogger) Info(msg string, keyvals ...interface{})  {}
func (l *NoOpLogger) Error(msg string, keyvals ...interface{}) {}

// NewEvent creates a new event
func NewEvent(eventType string, attributes ...keepertypes.Attribute) keepertypes.Event {
	return keepertypes.Event{
		Type:       eventType,
		Attributes: attributes,
	}
}

// NewAttribute creates a new attribute
func NewAttribute(key, value string) keepertypes.Attribute {
	return keepertypes.Attribute{
		Key:   key,
		Value: value,
	}
}

// PrefixStore returns a prefixed store for better key organization
func (k *KVStoreKeeper) PrefixStore(ctx keepertypes.Context, prefix []byte) types.KVStore {
	store := k.GetKVStore(ctx)

	// We would use the PrefixStore from our kvstore package
	// For now, return the raw store (in production, wrap with prefix)
	return store
}

// ParamStore provides parameter storage functionality
type ParamStore struct {
	*KVStoreKeeper
	paramSpace string
}

// NewParamStore creates a new parameter store
func NewParamStore(keeper *KVStoreKeeper, paramSpace string) *ParamStore {
	return &ParamStore{
		KVStoreKeeper: keeper,
		paramSpace:    paramSpace,
	}
}

// GetParam gets a parameter by key
func (ps *ParamStore) GetParam(ctx keepertypes.Context, key string, obj interface{}) error {
	paramKey := []byte(fmt.Sprintf("params/%s/%s", ps.paramSpace, key))
	return ps.GetObject(ctx, paramKey, obj)
}

// SetParam sets a parameter by key
func (ps *ParamStore) SetParam(ctx keepertypes.Context, key string, obj interface{}) error {
	paramKey := []byte(fmt.Sprintf("params/%s/%s", ps.paramSpace, key))
	return ps.SetObject(ctx, paramKey, obj)
}
