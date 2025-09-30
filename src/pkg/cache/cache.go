package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/allegro/bigcache/v3"
)

// Cache wraps bigcache with additional functionality
type Cache struct {
	cache *bigcache.BigCache
}

// Config holds cache configuration
type Config struct {
	// Shards number of cache shards (must be power of 2)
	Shards int
	// LifeWindow time after which entry can be evicted
	LifeWindow time.Duration
	// CleanWindow interval between removing expired entries
	CleanWindow time.Duration
	// MaxEntriesInWindow maximum number of entries in life window
	MaxEntriesInWindow int
	// MaxEntrySize maximum size of entry in bytes
	MaxEntrySize int
	// HardMaxCacheSize maximum cache size in MB
	HardMaxCacheSize int
	// OnRemove callback fired when entry is removed
	OnRemove func(key string, entry []byte)
	// AccessTTL TTL for user access permissions (shorter)
	AccessTTL time.Duration
}

// DefaultConfig returns a sensible default configuration for the application
func DefaultConfig() Config {
	return Config{
		Shards:             256,              // 256 shards for good distribution
		LifeWindow:         6 * time.Hour,    // 30 minutes TTL for story data
		CleanWindow:        5 * time.Minute,  // Clean every 5 minutes
		MaxEntriesInWindow: 1000,             // Max 1000 entries per window
		MaxEntrySize:       50 * 1024,        // 50KB max per entry (increased from 500 bytes)
		HardMaxCacheSize:   100,              // 100MB max cache size
		OnRemove:           nil,              // No callback by default
		AccessTTL:          15 * time.Minute, // 15 minutes TTL for access permissions
	}
}

// New creates a new cache instance with the given configuration
func New(config Config) (*Cache, error) {
	bigcacheConfig := bigcache.DefaultConfig(config.LifeWindow)
	bigcacheConfig.Shards = config.Shards
	bigcacheConfig.CleanWindow = config.CleanWindow
	bigcacheConfig.MaxEntriesInWindow = config.MaxEntriesInWindow
	bigcacheConfig.MaxEntrySize = config.MaxEntrySize
	bigcacheConfig.HardMaxCacheSize = config.HardMaxCacheSize
	bigcacheConfig.OnRemove = config.OnRemove

	cache, err := bigcache.New(context.Background(), bigcacheConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache: %w", err)
	}

	return &Cache{cache: cache}, nil
}

// Get retrieves a value from the cache
func (c *Cache) Get(key string) ([]byte, error) {
	return c.cache.Get(key)
}

// Set stores a value in the cache
func (c *Cache) Set(key string, value []byte) error {
	return c.cache.Set(key, value)
}

// Delete removes a value from the cache
func (c *Cache) Delete(key string) error {
	return c.cache.Delete(key)
}

// GetJSON retrieves and unmarshals a JSON value from the cache
func (c *Cache) GetJSON(key string, dest any) error {
	data, err := c.Get(key)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

// SetJSON marshals and stores a JSON value in the cache
func (c *Cache) SetJSON(key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}
	return c.Set(key, data)
}

// GetOrSet retrieves a value from cache, or computes and stores it if not found
func (c *Cache) GetOrSet(key string, compute func() (any, error)) ([]byte, error) {
	// Try to get from cache first
	if data, err := c.Get(key); err == nil {
		return data, nil
	}

	// Compute the value
	value, err := compute()
	if err != nil {
		return nil, err
	}

	// Handle different value types
	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		// Marshal other types to JSON
		var marshalErr error
		data, marshalErr = json.Marshal(value)
		if marshalErr != nil {
			return nil, fmt.Errorf("failed to marshal computed value: %w", marshalErr)
		}
	}

	// Store in cache (ignore error as this is best-effort)
	_ = c.Set(key, data)
	return data, nil
}

// GetOrSetJSON retrieves a JSON value from cache, or computes and stores it if not found
func (c *Cache) GetOrSetJSON(key string, dest any, compute func() (any, error)) error {
	// Try to get from cache first
	if err := c.GetJSON(key, dest); err == nil {
		// fmt.Println("DEBUG: Cache hit for key:", key)
		return nil
	}

	// Compute the value
	value, err := compute()
	if err != nil {
		return err
	}

	// Store in cache (ignore error as this is best-effort)
	_ = c.SetJSON(key, value)

	// Copy to destination
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal computed value: %w", err)
	}
	return json.Unmarshal(data, dest)
}

// Clear removes all entries from the cache
func (c *Cache) Clear() error {
	return c.cache.Reset()
}

// Len returns the number of entries in the cache
func (c *Cache) Len() int {
	return c.cache.Len()
}

// Stats returns cache statistics
func (c *Cache) Stats() bigcache.Stats {
	return c.cache.Stats()
}

// Key builders for consistent cache key generation
type KeyBuilder struct{}

// NewKeyBuilder creates a new key builder
func NewKeyBuilder() *KeyBuilder {
	return &KeyBuilder{}
}

// UserAccess builds a cache key for user access permissions
func (kb *KeyBuilder) UserAccess(userID string, storyID int) string {
	return fmt.Sprintf("access:user:%s:story:%d", userID, storyID)
}

// StoryData builds a cache key for story data (no user ID)
func (kb *KeyBuilder) StoryData(storyID int) string {
	return fmt.Sprintf("story:%d", storyID)
}

// AllStories builds a cache key for all stories (no user ID)
func (kb *KeyBuilder) AllStories(language string) string {
	return fmt.Sprintf("stories:lang:%s", language)
}

// UserVocabScores builds a cache key for user vocabulary scores
func (kb *KeyBuilder) UserVocabScores(userID string, storyID int) string {
	return fmt.Sprintf("vocab_scores:user:%s:story:%d", userID, storyID)
}

// UserGrammarScores builds a cache key for user grammar scores
func (kb *KeyBuilder) UserGrammarScores(userID string, storyID int) string {
	return fmt.Sprintf("grammar_scores:user:%s:story:%d", userID, storyID)
}

// StoryMetadata builds a cache key for story metadata
func (kb *KeyBuilder) StoryMetadata(storyID int) string {
	return fmt.Sprintf("story_metadata:%d", storyID)
}

// StoryAnnotations builds a cache key for story annotations
func (kb *KeyBuilder) StoryAnnotations(storyID int) string {
	return fmt.Sprintf("story_annotations:%d", storyID)
}

// LineAnnotations builds a cache key for line annotations
func (kb *KeyBuilder) LineAnnotations(storyID int, lineNumber int) string {
	return fmt.Sprintf("line_annotations:%d:%d", storyID, lineNumber)
}
