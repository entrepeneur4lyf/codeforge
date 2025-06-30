package context

import (
	"container/list"
	"sync"
	"time"
)

// CacheEntry represents a cached context entry
type CacheEntry struct {
	Key        string      `json:"key"`
	Value      interface{} `json:"value"`
	Timestamp  int64       `json:"timestamp"`
	AccessTime int64       `json:"access_time"`
	TTL        int64       `json:"ttl"`
	Size       int         `json:"size"`
}

// IsExpired checks if the cache entry has expired
func (ce *CacheEntry) IsExpired() bool {
	if ce.TTL <= 0 {
		return false // No expiration
	}
	return time.Now().Unix() > ce.Timestamp+ce.TTL
}

// ContextCache implements an LRU cache with TTL for context management
type ContextCache struct {
	maxSize    int
	ttl        int64
	cache      map[string]*list.Element
	lruList    *list.List
	mutex      sync.RWMutex
	stats      CacheStats
	onEvicted  func(key string, value interface{})
}

// CacheStats tracks cache performance metrics
type CacheStats struct {
	Hits        int64 `json:"hits"`
	Misses      int64 `json:"misses"`
	Evictions   int64 `json:"evictions"`
	Expirations int64 `json:"expirations"`
	Size        int   `json:"size"`
	MaxSize     int   `json:"max_size"`
	HitRate     float64 `json:"hit_rate"`
}

// NewContextCache creates a new context cache with LRU eviction and TTL
func NewContextCache(maxSize int, ttlSeconds int64) *ContextCache {
	return &ContextCache{
		maxSize: maxSize,
		ttl:     ttlSeconds,
		cache:   make(map[string]*list.Element),
		lruList: list.New(),
		stats: CacheStats{
			MaxSize: maxSize,
		},
	}
}

// SetEvictionCallback sets a callback function called when items are evicted
func (cc *ContextCache) SetEvictionCallback(callback func(key string, value interface{})) {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()
	cc.onEvicted = callback
}

// Get retrieves a value from the cache
func (cc *ContextCache) Get(key string) (interface{}, bool) {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	element, exists := cc.cache[key]
	if !exists {
		cc.stats.Misses++
		cc.updateHitRate()
		return nil, false
	}

	entry := element.Value.(*CacheEntry)
	
	// Check if expired
	if entry.IsExpired() {
		cc.removeElement(element)
		cc.stats.Misses++
		cc.stats.Expirations++
		cc.updateHitRate()
		return nil, false
	}

	// Move to front (most recently used)
	cc.lruList.MoveToFront(element)
	entry.AccessTime = time.Now().Unix()
	
	cc.stats.Hits++
	cc.updateHitRate()
	return entry.Value, true
}

// Set stores a value in the cache
func (cc *ContextCache) Set(key string, value interface{}, size int) {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	now := time.Now().Unix()
	
	// Check if key already exists
	if element, exists := cc.cache[key]; exists {
		// Update existing entry
		entry := element.Value.(*CacheEntry)
		entry.Value = value
		entry.Timestamp = now
		entry.AccessTime = now
		entry.Size = size
		cc.lruList.MoveToFront(element)
		return
	}

	// Create new entry
	entry := &CacheEntry{
		Key:        key,
		Value:      value,
		Timestamp:  now,
		AccessTime: now,
		TTL:        cc.ttl,
		Size:       size,
	}

	// Add to front of LRU list
	element := cc.lruList.PushFront(entry)
	cc.cache[key] = element
	cc.stats.Size++

	// Evict if necessary
	cc.evictIfNecessary()
}

// Delete removes a key from the cache
func (cc *ContextCache) Delete(key string) bool {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	element, exists := cc.cache[key]
	if !exists {
		return false
	}

	cc.removeElement(element)
	return true
}

// Clear removes all entries from the cache
func (cc *ContextCache) Clear() {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	for key, element := range cc.cache {
		entry := element.Value.(*CacheEntry)
		if cc.onEvicted != nil {
			cc.onEvicted(key, entry.Value)
		}
	}

	cc.cache = make(map[string]*list.Element)
	cc.lruList = list.New()
	cc.stats.Size = 0
}

// CleanupExpired removes all expired entries
func (cc *ContextCache) CleanupExpired() int {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	var toRemove []*list.Element
	
	// Find expired entries
	for element := cc.lruList.Back(); element != nil; element = element.Prev() {
		entry := element.Value.(*CacheEntry)
		if entry.IsExpired() {
			toRemove = append(toRemove, element)
		}
	}

	// Remove expired entries
	for _, element := range toRemove {
		cc.removeElement(element)
		cc.stats.Expirations++
	}

	return len(toRemove)
}

// GetStats returns current cache statistics
func (cc *ContextCache) GetStats() CacheStats {
	cc.mutex.RLock()
	defer cc.mutex.RUnlock()
	
	stats := cc.stats
	stats.Size = len(cc.cache)
	return stats
}

// Keys returns all keys in the cache
func (cc *ContextCache) Keys() []string {
	cc.mutex.RLock()
	defer cc.mutex.RUnlock()

	keys := make([]string, 0, len(cc.cache))
	for key := range cc.cache {
		keys = append(keys, key)
	}
	return keys
}

// Size returns the current number of entries in the cache
func (cc *ContextCache) Size() int {
	cc.mutex.RLock()
	defer cc.mutex.RUnlock()
	return len(cc.cache)
}

// evictIfNecessary removes entries if cache is over capacity
func (cc *ContextCache) evictIfNecessary() {
	for len(cc.cache) > cc.maxSize {
		// Remove least recently used item
		element := cc.lruList.Back()
		if element != nil {
			cc.removeElement(element)
			cc.stats.Evictions++
		}
	}
}

// removeElement removes an element from both the map and list
func (cc *ContextCache) removeElement(element *list.Element) {
	entry := element.Value.(*CacheEntry)
	delete(cc.cache, entry.Key)
	cc.lruList.Remove(element)
	cc.stats.Size--
	
	if cc.onEvicted != nil {
		cc.onEvicted(entry.Key, entry.Value)
	}
}

// updateHitRate calculates the current hit rate
func (cc *ContextCache) updateHitRate() {
	total := cc.stats.Hits + cc.stats.Misses
	if total > 0 {
		cc.stats.HitRate = float64(cc.stats.Hits) / float64(total)
	}
}

// ContextCacheManager manages multiple context caches
type ContextCacheManager struct {
	caches map[string]*ContextCache
	mutex  sync.RWMutex
}

// NewContextCacheManager creates a new cache manager
func NewContextCacheManager() *ContextCacheManager {
	return &ContextCacheManager{
		caches: make(map[string]*ContextCache),
	}
}

// GetCache returns a cache by name, creating it if it doesn't exist
func (ccm *ContextCacheManager) GetCache(name string, maxSize int, ttlSeconds int64) *ContextCache {
	ccm.mutex.Lock()
	defer ccm.mutex.Unlock()

	if cache, exists := ccm.caches[name]; exists {
		return cache
	}

	cache := NewContextCache(maxSize, ttlSeconds)
	ccm.caches[name] = cache
	return cache
}

// GetAllStats returns statistics for all caches
func (ccm *ContextCacheManager) GetAllStats() map[string]CacheStats {
	ccm.mutex.RLock()
	defer ccm.mutex.RUnlock()

	stats := make(map[string]CacheStats)
	for name, cache := range ccm.caches {
		stats[name] = cache.GetStats()
	}
	return stats
}

// CleanupAllExpired removes expired entries from all caches
func (ccm *ContextCacheManager) CleanupAllExpired() map[string]int {
	ccm.mutex.RLock()
	defer ccm.mutex.RUnlock()

	results := make(map[string]int)
	for name, cache := range ccm.caches {
		results[name] = cache.CleanupExpired()
	}
	return results
}

// ClearAll clears all caches
func (ccm *ContextCacheManager) ClearAll() {
	ccm.mutex.Lock()
	defer ccm.mutex.Unlock()

	for _, cache := range ccm.caches {
		cache.Clear()
	}
}

// Global cache manager instance
var globalCacheManager = NewContextCacheManager()

// GetGlobalCacheManager returns the global cache manager
func GetGlobalCacheManager() *ContextCacheManager {
	return globalCacheManager
}
