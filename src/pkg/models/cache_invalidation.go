package models

import (
	"fmt"
)

// InvalidateStoryCache removes cached data for a specific story
func InvalidateStoryCache(storyID int, userID string) {
	if cacheInstance == nil || keyBuilder == nil {
		return
	}

	// Invalidate story data (no user ID in key now)
	cacheKey := keyBuilder.StoryData(storyID)
	_ = cacheInstance.Delete(cacheKey)

	// Invalidate story metadata
	metadataKey := keyBuilder.StoryMetadata(storyID)
	_ = cacheInstance.Delete(metadataKey)

	// Invalidate story annotations
	annotationsKey := keyBuilder.StoryAnnotations(storyID)
	_ = cacheInstance.Delete(annotationsKey)

	// Invalidate story vocab count
	vocabCountKey := keyBuilder.StoryVocabCount(storyID)
	_ = cacheInstance.Delete(vocabCountKey)

	// Invalidate user access cache
	accessKey := keyBuilder.UserAccess(userID, storyID)
	_ = cacheInstance.Delete(accessKey)

	fmt.Printf("Invalidated cache for story %d, user %s\n", storyID, userID)
}

// InvalidateStoryMetadata removes cached metadata for a specific story (affects all users)
func InvalidateStoryMetadata(storyID int) {
	if cacheInstance == nil || keyBuilder == nil {
		return
	}

	// Invalidate story data
	storyKey := keyBuilder.StoryData(storyID)
	_ = cacheInstance.Delete(storyKey)

	// Invalidate story metadata
	metadataKey := keyBuilder.StoryMetadata(storyID)
	_ = cacheInstance.Delete(metadataKey)

	// Invalidate story annotations
	annotationsKey := keyBuilder.StoryAnnotations(storyID)
	_ = cacheInstance.Delete(annotationsKey)

	// Invalidate story vocab count
	vocabCountKey := keyBuilder.StoryVocabCount(storyID)
	_ = cacheInstance.Delete(vocabCountKey)

	fmt.Printf("Invalidated metadata cache for story %d\n", storyID)
}

// InvalidateUserStoryCache removes all cached data for a user's story interactions
func InvalidateUserStoryCache(userID string, storyID int) {
	if cacheInstance == nil || keyBuilder == nil {
		return
	}

	// Invalidate user-specific story data
	InvalidateStoryCache(storyID, userID)

	// Invalidate user scores
	vocabKey := keyBuilder.UserVocabScores(userID, storyID)
	_ = cacheInstance.Delete(vocabKey)

	grammarKey := keyBuilder.UserGrammarScores(userID, storyID)
	_ = cacheInstance.Delete(grammarKey)

	fmt.Printf("Invalidated all user cache for user %s, story %d\n", userID, storyID)
}

// InvalidateAllStoriesCache - No longer needed since we don't cache story lists
// Story lists are user-specific due to access controls, so caching would be a security risk
func InvalidateAllStoriesCache(language string) {
	// No-op: We don't cache story lists for security reasons
	fmt.Printf("Story list cache invalidation skipped for language %s (not cached for security)\n", language)
}

// ClearAllCache removes all cached data
func ClearAllCache() error {
	if cacheInstance == nil {
		return nil
	}

	err := cacheInstance.Clear()
	if err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}

	fmt.Println("Cleared all cache data")
	return nil
}
