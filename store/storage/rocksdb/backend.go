package rocksdb

import (
	"encoding/binary"
	"fmt"
	"path/filepath"

	"github.com/b2network/pulsar/store/types"
	"github.com/linxGnu/grocksdb"
)

// Backend implements a RocksDB-based storage backend with versioning
type Backend struct {
	db        *grocksdb.DB
	options   *grocksdb.Options
	writeOpts *grocksdb.WriteOptions
	readOpts  *grocksdb.ReadOptions

	// Versioning support using column families
	defaultCF *grocksdb.ColumnFamilyHandle
	versionCF *grocksdb.ColumnFamilyHandle

	// Write batch for atomic operations
	batch *grocksdb.WriteBatch

	// Latest version tracking
	latestVersion int64
}

// Config defines configuration for RocksDB backend
type Config struct {
	DataDir         string
	CreateIfMissing bool

	// Performance tuning
	MaxOpenFiles                   int
	WriteBufferSize                uint64
	MaxWriteBufferNum              int
	MinWriteBufferNum              int
	Level0FileNumCompactionTrigger int
	Level0SlowdownWritesTrigger    int
	Level0StopWritesTrigger        int
	MaxBytesForLevelBase           uint64
	MaxBytesForLevelMultiplier     float64
	TargetFileSizeBase             uint64
	TargetFileSizeMultiplier       int

	// Block cache
	BlockCacheSize uint64
}

// DefaultConfig returns default RocksDB configuration
func DefaultConfig(dataDir string) *Config {
	return &Config{
		DataDir:         dataDir,
		CreateIfMissing: true,

		// Performance settings optimized for blockchain workloads
		MaxOpenFiles:                   1000,
		WriteBufferSize:                64 << 20, // 64MB
		MaxWriteBufferNum:              3,
		MinWriteBufferNum:              2,
		Level0FileNumCompactionTrigger: 4,
		Level0SlowdownWritesTrigger:    20,
		Level0StopWritesTrigger:        36,
		MaxBytesForLevelBase:           256 << 20, // 256MB
		MaxBytesForLevelMultiplier:     10,
		TargetFileSizeBase:             64 << 20, // 64MB
		TargetFileSizeMultiplier:       2,

		BlockCacheSize: 128 << 20, // 128MB
	}
}

// NewBackend creates a new RocksDB backend
func NewBackend(config *Config) (*Backend, error) {
	// Ensure data directory exists
	dataDir := filepath.Clean(config.DataDir)

	// Configure RocksDB options
	options := grocksdb.NewDefaultOptions()
	options.SetCreateIfMissing(config.CreateIfMissing)
	options.SetCreateIfMissingColumnFamilies(true)

	// Performance tuning
	options.SetMaxOpenFiles(config.MaxOpenFiles)
	options.SetWriteBufferSize(config.WriteBufferSize)
	options.SetMaxWriteBufferNumber(config.MaxWriteBufferNum)
	options.SetMinWriteBufferNumberToMerge(config.MinWriteBufferNum)
	options.SetLevel0FileNumCompactionTrigger(config.Level0FileNumCompactionTrigger)
	options.SetLevel0SlowdownWritesTrigger(config.Level0SlowdownWritesTrigger)
	options.SetLevel0StopWritesTrigger(config.Level0StopWritesTrigger)
	options.SetMaxBytesForLevelBase(config.MaxBytesForLevelBase)
	options.SetMaxBytesForLevelMultiplier(config.MaxBytesForLevelMultiplier)
	options.SetTargetFileSizeBase(config.TargetFileSizeBase)
	options.SetTargetFileSizeMultiplier(config.TargetFileSizeMultiplier)

	// Block cache for better read performance
	blockCache := grocksdb.NewLRUCache(config.BlockCacheSize)
	blockBasedTableOptions := grocksdb.NewDefaultBlockBasedTableOptions()
	blockBasedTableOptions.SetBlockCache(blockCache)
	options.SetBlockBasedTableFactory(blockBasedTableOptions)

	// Use LZ4 compression for better performance
	options.SetCompression(grocksdb.CompressionType(grocksdb.LZ4Compression))

	// Column family setup for versioning
	cfNames := []string{"default", "versions"}
	cfOpts := []*grocksdb.Options{options, options}

	db, cfHandles, err := grocksdb.OpenDbColumnFamilies(options, dataDir, cfNames, cfOpts)
	if err != nil {
		options.Destroy()
		return nil, fmt.Errorf("failed to open RocksDB: %w", err)
	}

	if len(cfHandles) != 2 {
		db.Close()
		options.Destroy()
		return nil, fmt.Errorf("expected 2 column families, got %d", len(cfHandles))
	}

	writeOpts := grocksdb.NewDefaultWriteOptions()
	writeOpts.SetSync(false) // Async writes for better performance

	readOpts := grocksdb.NewDefaultReadOptions()

	backend := &Backend{
		db:        db,
		options:   options,
		writeOpts: writeOpts,
		readOpts:  readOpts,
		defaultCF: cfHandles[0],
		versionCF: cfHandles[1],
		batch:     grocksdb.NewWriteBatch(),
	}

	// Load latest version
	if err := backend.loadLatestVersion(); err != nil {
		backend.Close()
		return nil, fmt.Errorf("failed to load latest version: %w", err)
	}

	return backend, nil
}

// Get retrieves a value by key and version
func (b *Backend) Get(key []byte, version int64) ([]byte, error) {
	versionedKey := b.makeVersionedKey(key, version)

	value, err := b.db.GetCF(b.readOpts, b.defaultCF, versionedKey)
	if err != nil {
		return nil, err
	}
	defer value.Free()

	if !value.Exists() {
		return nil, nil
	}

	return value.Data(), nil
}

// Set stores a key-value pair in the write batch
func (b *Backend) Set(key []byte, value []byte) {
	versionedKey := b.makeVersionedKey(key, b.latestVersion+1)
	b.batch.PutCF(b.defaultCF, versionedKey, value)
}

// Delete marks a key for deletion in the write batch
func (b *Backend) Delete(key []byte) {
	versionedKey := b.makeVersionedKey(key, b.latestVersion+1)
	b.batch.DeleteCF(b.defaultCF, versionedKey)
}

// Has checks if a key exists at the given version
func (b *Backend) Has(key []byte, version int64) (bool, error) {
	versionedKey := b.makeVersionedKey(key, version)

	value, err := b.db.GetCF(b.readOpts, b.defaultCF, versionedKey)
	if err != nil {
		return false, err
	}
	defer value.Free()

	return value.Exists(), nil
}

// Iterator creates an iterator over a key range at a specific version
func (b *Backend) Iterator(start, end []byte, version int64) (types.Iterator, error) {
	startKey := b.makeVersionedKey(start, version)
	endKey := b.makeVersionedKey(end, version)

	iter := b.db.NewIteratorCF(b.readOpts, b.defaultCF)
	return newRocksDBIterator(iter, startKey, endKey, b.keyLength()), nil
}

// ReverseIterator creates a reverse iterator over a key range at a specific version
func (b *Backend) ReverseIterator(start, end []byte, version int64) (types.Iterator, error) {
	startKey := b.makeVersionedKey(start, version)
	endKey := b.makeVersionedKey(end, version)

	iter := b.db.NewIteratorCF(b.readOpts, b.defaultCF)
	return newRocksDBReverseIterator(iter, startKey, endKey, b.keyLength()), nil
}

// Flush writes all buffered changes to RocksDB
func (b *Backend) Flush(version int64) error {
	// Update latest version
	versionKey := []byte("latest_version")
	versionValue := make([]byte, 8)
	binary.BigEndian.PutUint64(versionValue, uint64(version))

	b.batch.PutCF(b.versionCF, versionKey, versionValue)

	// Write batch atomically
	if err := b.db.Write(b.writeOpts, b.batch); err != nil {
		return fmt.Errorf("failed to write batch: %w", err)
	}

	// Clear batch for next use
	b.batch.Clear()
	b.latestVersion = version

	return nil
}

// GetLatestVersion returns the latest committed version
func (b *Backend) GetLatestVersion() (int64, error) {
	return b.latestVersion, nil
}

// Close closes the RocksDB backend
func (b *Backend) Close() error {
	if b.batch != nil {
		b.batch.Destroy()
		b.batch = nil
	}

	if b.defaultCF != nil {
		b.defaultCF.Destroy()
		b.defaultCF = nil
	}

	if b.versionCF != nil {
		b.versionCF.Destroy()
		b.versionCF = nil
	}

	if b.readOpts != nil {
		b.readOpts.Destroy()
		b.readOpts = nil
	}

	if b.writeOpts != nil {
		b.writeOpts.Destroy()
		b.writeOpts = nil
	}

	if b.db != nil {
		b.db.Close()
		b.db = nil
	}

	if b.options != nil {
		b.options.Destroy()
		b.options = nil
	}

	return nil
}

func (b *Backend) loadLatestVersion() error {
	versionKey := []byte("latest_version")

	value, err := b.db.GetCF(b.readOpts, b.versionCF, versionKey)
	if err != nil {
		return err
	}
	defer value.Free()

	if !value.Exists() {
		b.latestVersion = 0
		return nil
	}

	data := value.Data()
	if len(data) != 8 {
		return fmt.Errorf("invalid version data length: %d", len(data))
	}

	b.latestVersion = int64(binary.BigEndian.Uint64(data))
	return nil
}

// makeVersionedKey creates a versioned key by appending version as timestamp
func (b *Backend) makeVersionedKey(key []byte, version int64) []byte {
	if key == nil {
		return nil
	}

	versionBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(versionBytes, uint64(version))

	versionedKey := make([]byte, len(key)+8)
	copy(versionedKey, key)
	copy(versionedKey[len(key):], versionBytes)

	return versionedKey
}

func (b *Backend) keyLength() int {
	// Returns the length of keys without version suffix
	return -8 // Negative means "remove last 8 bytes"
}
