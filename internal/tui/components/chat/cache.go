package chat

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
)

// MessageCache caches rendered messages to avoid re-rendering
type MessageCache struct {
	mu    sync.RWMutex
	cache map[string]string
	// maxSize limits the cache size to prevent unbounded growth
	maxSize int
	// lru tracks access order for eviction
	lru []string
}

// NewMessageCache creates a new message cache with a maximum size
func NewMessageCache(maxSize int) *MessageCache {
	if maxSize <= 0 {
		maxSize = 1000 // Default to 1000 cached messages
	}
	return &MessageCache{
		cache:   make(map[string]string),
		maxSize: maxSize,
		lru:     make([]string, 0, maxSize),
	}
}

// GenerateKey creates a unique key for a message based on its content and rendering parameters
func (c *MessageCache) GenerateKey(params ...interface{}) string {
	h := sha256.New()
	for _, param := range params {
		h.Write([]byte(fmt.Sprintf("%v", param)))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// Get retrieves a cached rendered message
func (c *MessageCache) Get(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	content, exists := c.cache[key]
	if exists {
		// Move to end of LRU list (most recently used)
		c.updateLRU(key)
	}
	return content, exists
}

// Set stores a rendered message in the cache
func (c *MessageCache) Set(key string, content string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Check if we need to evict
	if len(c.cache) >= c.maxSize && c.cache[key] == "" {
		// Evict least recently used
		if len(c.lru) > 0 {
			evictKey := c.lru[0]
			delete(c.cache, evictKey)
			c.lru = c.lru[1:]
		}
	}
	
	c.cache[key] = content
	c.updateLRU(key)
}

// Clear removes all entries from the cache
func (c *MessageCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]string)
	c.lru = c.lru[:0]
}

// Size returns the current number of cached entries
func (c *MessageCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cache)
}

// updateLRU moves or adds a key to the end of the LRU list
// Must be called with lock held
func (c *MessageCache) updateLRU(key string) {
	// Remove from current position if exists
	for i, k := range c.lru {
		if k == key {
			c.lru = append(c.lru[:i], c.lru[i+1:]...)
			break
		}
	}
	// Add to end (most recently used)
	c.lru = append(c.lru, key)
}

// InvalidateMatching removes all cache entries where the key matches a predicate
func (c *MessageCache) InvalidateMatching(predicate func(key string) bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	keysToDelete := []string{}
	for key := range c.cache {
		if predicate(key) {
			keysToDelete = append(keysToDelete, key)
		}
	}
	
	for _, key := range keysToDelete {
		delete(c.cache, key)
		// Remove from LRU
		for i, k := range c.lru {
			if k == key {
				c.lru = append(c.lru[:i], c.lru[i+1:]...)
				break
			}
		}
	}
}