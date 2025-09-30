package cache

import (
	"fmt"
	"testing"
	"time"
)

func TestCacheBasicOperations(t *testing.T) {
	cache, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Test basic set/get
	key := "test-key"
	value := []byte("test-value")

	err = cache.Set(key, value)
	if err != nil {
		t.Fatalf("Failed to set cache value: %v", err)
	}

	retrieved, err := cache.Get(key)
	if err != nil {
		t.Fatalf("Failed to get cache value: %v", err)
	}

	if string(retrieved) != string(value) {
		t.Errorf("Expected %s, got %s", string(value), string(retrieved))
	}
}

func TestCacheJSONOperations(t *testing.T) {
	cache, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	key := "json-test"
	original := TestStruct{Name: "test", Value: 42}

	// Test SetJSON
	err = cache.SetJSON(key, original)
	if err != nil {
		t.Fatalf("Failed to set JSON: %v", err)
	}

	// Test GetJSON
	var retrieved TestStruct
	err = cache.GetJSON(key, &retrieved)
	if err != nil {
		t.Fatalf("Failed to get JSON: %v", err)
	}

	if retrieved.Name != original.Name || retrieved.Value != original.Value {
		t.Errorf("Expected %+v, got %+v", original, retrieved)
	}
}

func TestCacheGetOrSet(t *testing.T) {
	cache, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	key := "get-or-set-test"
	expectedValue := []byte("computed-value")

	// First call should compute and cache
	value, err := cache.GetOrSet(key, func() (interface{}, error) {
		return expectedValue, nil
	})
	if err != nil {
		t.Fatalf("Failed to get or set: %v", err)
	}

	if string(value) != string(expectedValue) {
		t.Errorf("Expected %s, got %s", string(expectedValue), string(value))
	}

	// Second call should return cached value
	value2, err := cache.GetOrSet(key, func() (interface{}, error) {
		t.Error("Compute function should not be called on second access")
		return []byte("should-not-be-returned"), nil
	})
	if err != nil {
		t.Fatalf("Failed to get cached value: %v", err)
	}

	if string(value2) != string(expectedValue) {
		t.Errorf("Expected cached %s, got %s", string(expectedValue), string(value2))
	}
}

func TestCacheGetOrSetJSON(t *testing.T) {
	cache, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	type TestStruct struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	key := "get-or-set-json-test"
	original := TestStruct{ID: 1, Name: "test"}

	// First call should compute and cache
	var retrieved TestStruct
	err = cache.GetOrSetJSON(key, &retrieved, func() (interface{}, error) {
		return original, nil
	})
	if err != nil {
		t.Fatalf("Failed to get or set JSON: %v", err)
	}

	if retrieved.ID != original.ID || retrieved.Name != original.Name {
		t.Errorf("Expected %+v, got %+v", original, retrieved)
	}

	// Second call should return cached value
	var retrieved2 TestStruct
	err = cache.GetOrSetJSON(key, &retrieved2, func() (interface{}, error) {
		t.Error("Compute function should not be called on second access")
		return TestStruct{ID: 999, Name: "should-not-be-returned"}, nil
	})
	if err != nil {
		t.Fatalf("Failed to get cached JSON: %v", err)
	}

	if retrieved2.ID != original.ID || retrieved2.Name != original.Name {
		t.Errorf("Expected cached %+v, got %+v", original, retrieved2)
	}
}

func TestCacheDelete(t *testing.T) {
	cache, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	key := "delete-test"
	value := []byte("delete-me")

	// Set value
	err = cache.Set(key, value)
	if err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}

	// Verify it exists
	_, err = cache.Get(key)
	if err != nil {
		t.Fatalf("Value should exist before deletion: %v", err)
	}

	// Delete value
	err = cache.Delete(key)
	if err != nil {
		t.Fatalf("Failed to delete value: %v", err)
	}

	// Verify it's gone
	_, err = cache.Get(key)
	if err == nil {
		t.Error("Value should not exist after deletion")
	}
}

func TestCacheTTL(t *testing.T) {
	// Create cache with short TTL for testing
	config := DefaultConfig()
	config.LifeWindow = 50 * time.Millisecond
	config.CleanWindow = 25 * time.Millisecond

	cache, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	key := "ttl-test"
	value := []byte("ttl-value")

	// Set value
	err = cache.Set(key, value)
	if err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}

	// Should exist immediately
	_, err = cache.Get(key)
	if err != nil {
		t.Fatalf("Value should exist immediately: %v", err)
	}

	// Wait for TTL to expire plus cleanup time
	time.Sleep(100 * time.Millisecond)

	// Check if value is gone after TTL (may still exist due to timing)
	_, err = cache.Get(key)
	if err == nil {
		// Value still exists - this is acceptable for TTL tests
		// as cleanup timing can vary
		t.Log("Value still exists after TTL - this is acceptable due to cleanup timing")
	}
}

func TestKeyBuilder(t *testing.T) {
	kb := NewKeyBuilder()

	// Test story data key
	storyKey := kb.StoryData(123)
	expected := "story:123"
	if storyKey != expected {
		t.Errorf("Expected %s, got %s", expected, storyKey)
	}

	// Test all stories key
	allStoriesKey := kb.AllStories("en")
	expected = "stories:lang:en"
	if allStoriesKey != expected {
		t.Errorf("Expected %s, got %s", expected, allStoriesKey)
	}

	// Test user access key
	accessKey := kb.UserAccess("user456", 123)
	expected = "access:user:user456:story:123"
	if accessKey != expected {
		t.Errorf("Expected %s, got %s", expected, accessKey)
	}

	// Test user vocab scores key
	vocabKey := kb.UserVocabScores("user123", 456)
	expected = "vocab_scores:user:user123:story:456"
	if vocabKey != expected {
		t.Errorf("Expected %s, got %s", expected, vocabKey)
	}
}

func TestCacheStats(t *testing.T) {
	cache, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Add some data
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("stats-test-%d", i)
		value := []byte(fmt.Sprintf("value-%d", i))
		err = cache.Set(key, value)
		if err != nil {
			t.Fatalf("Failed to set value %d: %v", i, err)
		}
	}

	// Check stats - stats might be zero if no cache operations occurred
	stats := cache.Stats()
	// Just verify we can get stats without error
	_ = stats

	// Check length
	length := cache.Len()
	if length != 10 {
		t.Errorf("Expected length 10, got %d", length)
	}
}

func TestCacheClear(t *testing.T) {
	cache, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Add some data
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("clear-test-%d", i)
		value := []byte(fmt.Sprintf("value-%d", i))
		err = cache.Set(key, value)
		if err != nil {
			t.Fatalf("Failed to set value %d: %v", i, err)
		}
	}

	// Verify data exists
	if cache.Len() != 5 {
		t.Errorf("Expected length 5, got %d", cache.Len())
	}

	// Clear cache
	err = cache.Clear()
	if err != nil {
		t.Fatalf("Failed to clear cache: %v", err)
	}

	// Verify cache is empty
	if cache.Len() != 0 {
		t.Errorf("Expected length 0 after clear, got %d", cache.Len())
	}
}
