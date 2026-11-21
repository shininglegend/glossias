package cache

import (
	"fmt"
	"sync/atomic"
)

// accessEpoch is part of every story-access cache key. Bumping it orphans
// every cached "allowed" at once (the old entries age out of BigCache), so a
// revocation never has to know which user/story pairs it affected.
var accessEpoch atomic.Int64

// BumpAccessEpoch invalidates every cached story-access decision.
func BumpAccessEpoch() { accessEpoch.Add(1) }

// StoryAccess builds the cache key for a user's access to a story under the
// current access epoch.
func (kb *KeyBuilder) StoryAccess(userID string, storyID int) string {
	return fmt.Sprintf("access:v%d:user:%s:story:%d", accessEpoch.Load(), userID, storyID)
}
