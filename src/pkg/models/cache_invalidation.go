package models

import (
	"fmt"

	"glossias/src/pkg/cache"
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
	accessKey := keyBuilder.StoryAccess(userID, storyID)
	_ = cacheInstance.Delete(accessKey)

	// Annotations feed the lexical-form counts the readiness report checks
	InvalidateStoryContentReadiness(storyID)

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

	// Per-line annotation entries (line_annotations:{id}:{line}) hold the same
	// audio, vocabulary, grammar and footnotes.
	cacheInstance.DeletePrefix(fmt.Sprintf("line_annotations:%d:", storyID))

	// Annotations feed the lexical-form counts the readiness report checks
	InvalidateStoryContentReadiness(storyID)

	fmt.Printf("Invalidated metadata cache for story %d\n", storyID)
}

// BumpAccessEpoch forgets every cached story-access decision. Call it after
// any write that can take access away (unlinking a story, removing or
// deactivating an enrollment, expiring an entitlement, resuming payments,
// removing a course admin, turning a trial course off or deleting a course).
// Writes that only grant access need not call it: denials are never cached.
func BumpAccessEpoch() {
	cache.BumpAccessEpoch()
}

// InvalidateStoryContentReadiness drops the cached phase readiness report for a
// story. Call it after any write to target vocabulary, produce content, recall
// sentences, or annotations.
func InvalidateStoryContentReadiness(storyID int) {
	if cacheInstance == nil || keyBuilder == nil {
		return
	}
	_ = cacheInstance.Delete(keyBuilder.StoryContentReadiness(storyID))
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
