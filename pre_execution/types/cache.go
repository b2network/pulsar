package types

import (
	"container/list"
	"sync"
	"time"
)

// CachedExecution represents a cached pre-execution result with metadata
type CachedExecution struct {
	// Result contains the actual pre-execution result
	Result *TxPreExecResult `json:"result"`

	// CreatedAt records when this result was cached
	CreatedAt time.Time `json:"created_at"`

	// AccessedAt records the last access time (for LRU eviction)
	AccessedAt time.Time `json:"accessed_at"`

	// AccessCount tracks how often this result has been accessed (for LFU eviction)
	AccessCount uint64 `json:"access_count"`

	// Size estimates the memory size of this cached entry
	Size uint64 `json:"size"`

	// CompressedData holds compressed result data if compression is enabled
	CompressedData []byte `json:"compressed_data,omitempty"`

	// IsCompressed indicates if the result data is compressed
	IsCompressed bool `json:"is_compressed"`
}

// PreExecutionCache manages cached pre-execution results with configurable policies
type PreExecutionCache struct {
	mu sync.RWMutex

	// Configuration
	maxSize            int           // Maximum number of cached entries
	maxMemory          uint64        // Maximum memory usage in bytes
	ttl                time.Duration // Time-to-live for cached entries
	evictionPolicy     string        // Eviction policy: "LRU", "LFU", "FIFO"
	compressionEnabled bool          // Whether to compress cached data

	// Storage
	entries     map[string]*CachedExecution // Hash -> CachedExecution mapping
	orderedList *list.List                  // For LRU/FIFO eviction
	elementMap  map[string]*list.Element    // Hash -> List element mapping

	// Statistics
	totalSize uint64 // Current memory usage
	hits      uint64 // Cache hit count
	misses    uint64 // Cache miss count
	evictions uint64 // Cache eviction count

	// Cleanup
	cleanupTicker *time.Ticker  // Periodic cleanup timer
	stopCleanup   chan struct{} // Signal to stop cleanup goroutine
}

// NewPreExecutionCache creates a new pre-execution cache with the given configuration
func NewPreExecutionCache(config CacheConfig) *PreExecutionCache {
	cache := &PreExecutionCache{
		maxSize:            config.MaxSize,
		maxMemory:          config.MaxMemory,
		ttl:                config.TTL,
		evictionPolicy:     config.EvictionPolicy,
		compressionEnabled: config.CompressionEnabled,
		entries:            make(map[string]*CachedExecution),
		orderedList:        list.New(),
		elementMap:         make(map[string]*list.Element),
		stopCleanup:        make(chan struct{}),
	}

	// Start periodic cleanup if TTL is set
	if cache.ttl > 0 {
		cache.cleanupTicker = time.NewTicker(cache.ttl / 4) // Cleanup every quarter of TTL
		go cache.cleanupExpired()
	}

	return cache
}

// CacheConfig contains configuration options for the pre-execution cache
type CacheConfig struct {
	MaxSize            int           `json:"max_size"`            // Maximum number of entries
	MaxMemory          uint64        `json:"max_memory"`          // Maximum memory usage in bytes
	TTL                time.Duration `json:"ttl"`                 // Time-to-live for entries
	EvictionPolicy     string        `json:"eviction_policy"`     // "LRU", "LFU", "FIFO"
	CompressionEnabled bool          `json:"compression_enabled"` // Enable compression
	CleanupInterval    time.Duration `json:"cleanup_interval"`    // How often to clean up expired entries
}

// Get retrieves a cached pre-execution result
func (c *PreExecutionCache) Get(txHash string) (*TxPreExecResult, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.entries[txHash]
	if !exists {
		c.misses++
		return nil, false
	}

	// Check if entry has expired
	if c.ttl > 0 && time.Since(entry.CreatedAt) > c.ttl {
		c.remove(txHash)
		c.misses++
		return nil, false
	}

	// Update access tracking for eviction policies
	entry.AccessedAt = time.Now()
	entry.AccessCount++

	// Move to front for LRU policy
	if c.evictionPolicy == "LRU" {
		if elem, exists := c.elementMap[txHash]; exists {
			c.orderedList.MoveToFront(elem)
		}
	}

	c.hits++

	// Decompress if necessary
	if entry.IsCompressed {
		// TODO: Implement decompression logic
		// For now, return the uncompressed result
	}

	return entry.Result, true
}

// Set stores a pre-execution result in the cache
func (c *PreExecutionCache) Set(txHash string, result *TxPreExecResult) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Create new cache entry
	entry := &CachedExecution{
		Result:      result,
		CreatedAt:   time.Now(),
		AccessedAt:  time.Now(),
		AccessCount: 1,
		Size:        c.estimateSize(result),
	}

	// Compress if enabled
	if c.compressionEnabled {
		// TODO: Implement compression logic
		// entry.CompressedData = compress(result)
		// entry.IsCompressed = true
	}

	// Check if we need to evict entries
	for (len(c.entries) >= c.maxSize || c.totalSize+entry.Size > c.maxMemory) && len(c.entries) > 0 {
		if !c.evictOne() {
			return false // Could not evict, cache is full
		}
	}

	// Add new entry
	c.entries[txHash] = entry
	c.totalSize += entry.Size

	// Add to ordered list for eviction policies
	if c.evictionPolicy == "LRU" || c.evictionPolicy == "FIFO" {
		elem := c.orderedList.PushFront(txHash)
		c.elementMap[txHash] = elem
	}

	return true
}

// Remove removes a specific entry from the cache
func (c *PreExecutionCache) Remove(txHash string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.remove(txHash)
}

// remove is the internal removal function (must be called with lock held)
func (c *PreExecutionCache) remove(txHash string) {
	if entry, exists := c.entries[txHash]; exists {
		delete(c.entries, txHash)
		c.totalSize -= entry.Size

		// Remove from ordered list
		if elem, exists := c.elementMap[txHash]; exists {
			c.orderedList.Remove(elem)
			delete(c.elementMap, txHash)
		}
	}
}

// evictOne removes one entry based on the eviction policy
func (c *PreExecutionCache) evictOne() bool {
	if len(c.entries) == 0 {
		return false
	}

	var victimHash string

	switch c.evictionPolicy {
	case "LRU":
		// Evict least recently used
		if elem := c.orderedList.Back(); elem != nil {
			victimHash = elem.Value.(string)
		}

	case "LFU":
		// Evict least frequently used
		var minCount uint64 = ^uint64(0)
		for hash, entry := range c.entries {
			if entry.AccessCount < minCount {
				minCount = entry.AccessCount
				victimHash = hash
			}
		}

	case "FIFO":
		// Evict first in (oldest)
		if elem := c.orderedList.Back(); elem != nil {
			victimHash = elem.Value.(string)
		}

	default:
		// Default to LRU
		if elem := c.orderedList.Back(); elem != nil {
			victimHash = elem.Value.(string)
		}
	}

	if victimHash != "" {
		c.remove(victimHash)
		c.evictions++
		return true
	}

	return false
}

// cleanupExpired removes expired entries from the cache
func (c *PreExecutionCache) cleanupExpired() {
	ticker := c.cleanupTicker
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.mu.Lock()
			now := time.Now()
			var expiredHashes []string

			for hash, entry := range c.entries {
				if c.ttl > 0 && now.Sub(entry.CreatedAt) > c.ttl {
					expiredHashes = append(expiredHashes, hash)
				}
			}

			for _, hash := range expiredHashes {
				c.remove(hash)
			}
			c.mu.Unlock()

		case <-c.stopCleanup:
			return
		}
	}
}

// Clear removes all entries from the cache
func (c *PreExecutionCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*CachedExecution)
	c.orderedList = list.New()
	c.elementMap = make(map[string]*list.Element)
	c.totalSize = 0
}

// Close stops the cache cleanup goroutine and releases resources
func (c *PreExecutionCache) Close() {
	if c.cleanupTicker != nil {
		close(c.stopCleanup)
	}
}

// GetStats returns cache statistics
func (c *PreExecutionCache) GetStats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	hitRate := float64(0)
	if c.hits+c.misses > 0 {
		hitRate = float64(c.hits) / float64(c.hits+c.misses)
	}

	return CacheStats{
		Size:        len(c.entries),
		MaxSize:     c.maxSize,
		MemoryUsage: c.totalSize,
		MaxMemory:   c.maxMemory,
		Hits:        c.hits,
		Misses:      c.misses,
		Evictions:   c.evictions,
		HitRate:     hitRate,
	}
}

// CacheStats contains cache performance statistics
type CacheStats struct {
	Size        int     `json:"size"`         // Current number of entries
	MaxSize     int     `json:"max_size"`     // Maximum allowed entries
	MemoryUsage uint64  `json:"memory_usage"` // Current memory usage in bytes
	MaxMemory   uint64  `json:"max_memory"`   // Maximum allowed memory usage
	Hits        uint64  `json:"hits"`         // Number of cache hits
	Misses      uint64  `json:"misses"`       // Number of cache misses
	Evictions   uint64  `json:"evictions"`    // Number of evictions
	HitRate     float64 `json:"hit_rate"`     // Cache hit rate (0.0 - 1.0)
}

// estimateSize estimates the memory size of a pre-execution result
func (c *PreExecutionCache) estimateSize(result *TxPreExecResult) uint64 {
	// Simple size estimation based on serialized data size
	// In production, this could be more sophisticated
	size := uint64(len(result.TxHash))
	size += uint64(len(result.MsgResults)) * 200 // Estimate per message result

	if result.StateChanges != nil {
		for _, storeChanges := range result.StateChanges.StoreChanges {
			for k, v := range storeChanges.Sets {
				size += uint64(len(k) + len(v))
			}
			size += uint64(len(storeChanges.Deletes)) * 32 // Estimate per delete
		}
	}

	return size
}
