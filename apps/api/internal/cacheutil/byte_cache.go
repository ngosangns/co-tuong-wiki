package cacheutil

import "sync"

type ByteCache struct {
	mu         sync.Mutex
	maxEntries int
	entries    map[string][]byte
	order      []string
	hits       int
	misses     int
	evictions  int
}

type ByteCacheStats struct {
	Entries   int `json:"entries"`
	Hits      int `json:"hits"`
	Misses    int `json:"misses"`
	Evictions int `json:"evictions"`
}

func NewByteCache(maxEntries int) *ByteCache {
	if maxEntries < 1 {
		maxEntries = 1
	}

	return &ByteCache{
		maxEntries: maxEntries,
		entries:    map[string][]byte{},
	}
}

func (cache *ByteCache) Get(key string) ([]byte, bool) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	body, ok := cache.entries[key]
	if !ok {
		cache.misses++
		return nil, false
	}

	cache.hits++
	return append([]byte(nil), body...), true
}

func (cache *ByteCache) Set(key string, body []byte) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	if _, exists := cache.entries[key]; !exists {
		cache.order = append(cache.order, key)
	}
	cache.entries[key] = append([]byte(nil), body...)

	for len(cache.order) > cache.maxEntries {
		oldest := cache.order[0]
		cache.order = cache.order[1:]
		delete(cache.entries, oldest)
		cache.evictions++
	}
}

func (cache *ByteCache) Clear() {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	cache.entries = map[string][]byte{}
	cache.order = nil
	cache.hits = 0
	cache.misses = 0
	cache.evictions = 0
}

func (cache *ByteCache) Stats() ByteCacheStats {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	return ByteCacheStats{
		Entries:   len(cache.entries),
		Hits:      cache.hits,
		Misses:    cache.misses,
		Evictions: cache.evictions,
	}
}
