package cache

import "strings"

// DeletePrefix removes every entry whose key starts with prefix and returns
// how many were removed. It walks the whole cache, so keep it off hot read
// paths; it is meant for invalidation after admin writes.
func (c *Cache) DeletePrefix(prefix string) int {
	var keys []string
	it := c.cache.Iterator()
	for it.SetNext() {
		info, err := it.Value()
		if err != nil {
			continue
		}
		if k := info.Key(); strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}
	n := 0
	for _, k := range keys {
		if c.cache.Delete(k) == nil {
			n++
		}
	}
	return n
}
