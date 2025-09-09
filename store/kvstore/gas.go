package kvstore

import (
	"github.com/b2network/pulsar/store/types"
)

// GasMeter defines an interface for measuring gas consumption
type GasMeter interface {
	ConsumeGas(amount uint64, descriptor string)
	GasConsumed() uint64
	GasLimit() uint64
	OutOfGas() bool
}

// GasConfig defines gas costs for different operations
type GasConfig struct {
	HasCost          uint64
	DeleteCost       uint64
	ReadCostFlat     uint64
	ReadCostPerByte  uint64
	WriteCostFlat    uint64
	WriteCostPerByte uint64
	IterNextCostFlat uint64
}

// DefaultGasConfig returns default gas configuration
func DefaultGasConfig() GasConfig {
	return GasConfig{
		HasCost:          1000,
		DeleteCost:       1000,
		ReadCostFlat:     1000,
		ReadCostPerByte:  3,
		WriteCostFlat:    2000,
		WriteCostPerByte: 30,
		IterNextCostFlat: 30,
	}
}

// GasKVStore wraps a KVStore and charges gas for operations
type GasKVStore struct {
	parent    types.KVStore
	gasMeter  GasMeter
	gasConfig GasConfig
}

// NewGasKVStore creates a new GasKVStore
func NewGasKVStore(parent types.KVStore, gasMeter GasMeter, gasConfig GasConfig) *GasKVStore {
	return &GasKVStore{
		parent:    parent,
		gasMeter:  gasMeter,
		gasConfig: gasConfig,
	}
}

func (s *GasKVStore) GetStoreType() types.StoreType {
	return s.parent.GetStoreType()
}

func (s *GasKVStore) CacheWrap() types.CacheWrap {
	return NewGasKVStore(s.parent.CacheWrap().(types.KVStore), s.gasMeter, s.gasConfig)
}

func (s *GasKVStore) Write() {
	if cacheWrap, ok := s.parent.(types.CacheWrap); ok {
		cacheWrap.Write()
	}
}

func (s *GasKVStore) Get(key []byte) []byte {
	s.gasMeter.ConsumeGas(s.gasConfig.ReadCostFlat, "KVStore.Get")

	value := s.parent.Get(key)
	if value != nil {
		s.gasMeter.ConsumeGas(s.gasConfig.ReadCostPerByte*uint64(len(key)), "KVStore.Get.Key")
		s.gasMeter.ConsumeGas(s.gasConfig.ReadCostPerByte*uint64(len(value)), "KVStore.Get.Value")
	}

	return value
}

func (s *GasKVStore) Has(key []byte) bool {
	s.gasMeter.ConsumeGas(s.gasConfig.HasCost, "KVStore.Has")
	s.gasMeter.ConsumeGas(s.gasConfig.ReadCostPerByte*uint64(len(key)), "KVStore.Has.Key")

	return s.parent.Has(key)
}

func (s *GasKVStore) Set(key, value []byte) {
	s.gasMeter.ConsumeGas(s.gasConfig.WriteCostFlat, "KVStore.Set")
	s.gasMeter.ConsumeGas(s.gasConfig.WriteCostPerByte*uint64(len(key)), "KVStore.Set.Key")
	s.gasMeter.ConsumeGas(s.gasConfig.WriteCostPerByte*uint64(len(value)), "KVStore.Set.Value")

	s.parent.Set(key, value)
}

func (s *GasKVStore) Delete(key []byte) {
	s.gasMeter.ConsumeGas(s.gasConfig.DeleteCost, "KVStore.Delete")
	s.gasMeter.ConsumeGas(s.gasConfig.ReadCostPerByte*uint64(len(key)), "KVStore.Delete.Key")

	s.parent.Delete(key)
}

func (s *GasKVStore) Iterator(start, end []byte) types.Iterator {
	return s.iterator(start, end, false)
}

func (s *GasKVStore) ReverseIterator(start, end []byte) types.Iterator {
	return s.iterator(start, end, true)
}

func (s *GasKVStore) iterator(start, end []byte, reverse bool) types.Iterator {
	var parentIter types.Iterator
	if reverse {
		parentIter = s.parent.ReverseIterator(start, end)
	} else {
		parentIter = s.parent.Iterator(start, end)
	}

	return newGasIterator(parentIter, s.gasMeter, s.gasConfig)
}

// gasIterator wraps an iterator and charges gas for operations
type gasIterator struct {
	parent    types.Iterator
	gasMeter  GasMeter
	gasConfig GasConfig
}

func newGasIterator(parent types.Iterator, gasMeter GasMeter, gasConfig GasConfig) types.Iterator {
	return &gasIterator{
		parent:    parent,
		gasMeter:  gasMeter,
		gasConfig: gasConfig,
	}
}

func (iter *gasIterator) Domain() (start, end []byte) {
	return iter.parent.Domain()
}

func (iter *gasIterator) Valid() bool {
	return iter.parent.Valid()
}

func (iter *gasIterator) Next() {
	iter.gasMeter.ConsumeGas(iter.gasConfig.IterNextCostFlat, "Iterator.Next")
	iter.parent.Next()
}

func (iter *gasIterator) Key() []byte {
	key := iter.parent.Key()
	if key != nil {
		iter.gasMeter.ConsumeGas(iter.gasConfig.ReadCostPerByte*uint64(len(key)), "Iterator.Key")
	}
	return key
}

func (iter *gasIterator) Value() []byte {
	value := iter.parent.Value()
	if value != nil {
		iter.gasMeter.ConsumeGas(iter.gasConfig.ReadCostPerByte*uint64(len(value)), "Iterator.Value")
	}
	return value
}

func (iter *gasIterator) Close() error {
	return iter.parent.Close()
}
