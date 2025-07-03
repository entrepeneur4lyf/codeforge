package chat

import (
	"strings"
	"sync"
	"testing"
)

func TestMessageCache(t *testing.T) {
	t.Run("basic get and set", func(t *testing.T) {
		cache := NewMessageCache(10)
		
		key := cache.GenerateKey("msg1", "user", "Hello world")
		content := "Rendered message"
		
		// Should not exist initially
		if _, found := cache.Get(key); found {
			t.Error("Expected key to not exist initially")
		}
		
		// Set and get
		cache.Set(key, content)
		if got, found := cache.Get(key); !found || got != content {
			t.Errorf("Expected to get %q, got %q (found: %v)", content, got, found)
		}
	})
	
	t.Run("cache eviction", func(t *testing.T) {
		cache := NewMessageCache(3)
		
		// Fill cache
		for i := 0; i < 4; i++ {
			key := cache.GenerateKey("msg", i)
			cache.Set(key, string(rune('A'+i)))
		}
		
		// First item should be evicted
		if _, found := cache.Get(cache.GenerateKey("msg", 0)); found {
			t.Error("Expected first item to be evicted")
		}
		
		// Other items should still exist
		for i := 1; i < 4; i++ {
			key := cache.GenerateKey("msg", i)
			if _, found := cache.Get(key); !found {
				t.Errorf("Expected item %d to still exist", i)
			}
		}
	})
	
	t.Run("LRU ordering", func(t *testing.T) {
		cache := NewMessageCache(3)
		
		// Add 3 items
		for i := 0; i < 3; i++ {
			cache.Set(cache.GenerateKey("msg", i), string(rune('A'+i)))
		}
		
		// Access first item (makes it most recently used)
		cache.Get(cache.GenerateKey("msg", 0))
		
		// Add new item - should evict item 1, not 0
		cache.Set(cache.GenerateKey("msg", 3), "D")
		
		// Item 0 should still exist (was accessed recently)
		if _, found := cache.Get(cache.GenerateKey("msg", 0)); !found {
			t.Error("Expected item 0 to still exist after recent access")
		}
		
		// Item 1 should be evicted
		if _, found := cache.Get(cache.GenerateKey("msg", 1)); found {
			t.Error("Expected item 1 to be evicted")
		}
	})
	
	t.Run("clear", func(t *testing.T) {
		cache := NewMessageCache(10)
		
		// Add some items
		for i := 0; i < 5; i++ {
			cache.Set(cache.GenerateKey("msg", i), "content")
		}
		
		if cache.Size() != 5 {
			t.Errorf("Expected size 5, got %d", cache.Size())
		}
		
		// Clear
		cache.Clear()
		
		if cache.Size() != 0 {
			t.Errorf("Expected size 0 after clear, got %d", cache.Size())
		}
		
		// Should not find any items
		for i := 0; i < 5; i++ {
			if _, found := cache.Get(cache.GenerateKey("msg", i)); found {
				t.Errorf("Expected item %d to be cleared", i)
			}
		}
	})
	
	t.Run("invalidate matching", func(t *testing.T) {
		cache := NewMessageCache(10)
		
		// Add items with different prefixes
		cache.Set("user:1", "content1")
		cache.Set("user:2", "content2")
		cache.Set("assistant:1", "content3")
		cache.Set("assistant:2", "content4")
		
		// Invalidate all user messages
		cache.InvalidateMatching(func(key string) bool {
			return strings.HasPrefix(key, "user:")
		})
		
		// User messages should be gone
		if _, found := cache.Get("user:1"); found {
			t.Error("Expected user:1 to be invalidated")
		}
		if _, found := cache.Get("user:2"); found {
			t.Error("Expected user:2 to be invalidated")
		}
		
		// Assistant messages should remain
		if _, found := cache.Get("assistant:1"); !found {
			t.Error("Expected assistant:1 to remain")
		}
		if _, found := cache.Get("assistant:2"); !found {
			t.Error("Expected assistant:2 to remain")
		}
	})
	
	t.Run("concurrent access", func(t *testing.T) {
		cache := NewMessageCache(100)
		var wg sync.WaitGroup
		
		// Concurrent writers
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < 10; j++ {
					key := cache.GenerateKey("writer", id, j)
					cache.Set(key, "content")
				}
			}(i)
		}
		
		// Concurrent readers
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < 10; j++ {
					key := cache.GenerateKey("writer", id%10, j)
					cache.Get(key)
				}
			}(i)
		}
		
		wg.Wait()
		
		// Should not panic and size should be reasonable
		size := cache.Size()
		if size < 50 || size > 100 {
			t.Errorf("Unexpected cache size: %d", size)
		}
	})
}

func TestGenerateKey(t *testing.T) {
	cache := NewMessageCache(1)
	
	// Same inputs should generate same key
	key1 := cache.GenerateKey("msg1", "user", "Hello", 100)
	key2 := cache.GenerateKey("msg1", "user", "Hello", 100)
	
	if key1 != key2 {
		t.Error("Expected same inputs to generate same key")
	}
	
	// Different inputs should generate different keys
	key3 := cache.GenerateKey("msg2", "user", "Hello", 100)
	if key1 == key3 {
		t.Error("Expected different inputs to generate different keys")
	}
	
	// Order matters
	key4 := cache.GenerateKey("user", "msg1", "Hello", 100)
	if key1 == key4 {
		t.Error("Expected different order to generate different keys")
	}
}