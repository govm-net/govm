package database

import (
	"container/list"
	"sync"
)

// CacheNode cache node
type CacheNode struct {
	Key, Value interface{}
}

// LRUCache lru cache
type LRUCache struct {
	cap      int
	dlist    *list.List
	cacheMap map[interface{}]*list.Element
	mu       sync.Mutex
}

// NewLRUCache new cache
func NewLRUCache(cap int) *LRUCache {
	return &LRUCache{
		cap:      cap,
		dlist:    list.New(),
		cacheMap: make(map[interface{}]*list.Element)}
}

// Size get length of cached
func (lru *LRUCache) Size() int {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	return lru.dlist.Len()
}

// Set add new item
func (lru *LRUCache) Set(k, v interface{}) {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	if pElement, ok := lru.cacheMap[k]; ok {
		lru.dlist.MoveToFront(pElement)
		pElement.Value.(*CacheNode).Value = v
		return
	}

	newElement := lru.dlist.PushFront(&CacheNode{k, v})
	lru.cacheMap[k] = newElement

	if lru.dlist.Len() > lru.cap {
		lastElement := lru.dlist.Back()
		if lastElement == nil {
			return
		}
		cacheNode := lastElement.Value.(*CacheNode)
		delete(lru.cacheMap, cacheNode.Key)
		lru.dlist.Remove(lastElement)
	}
	return
}

// Get get cache value
func (lru *LRUCache) Get(k interface{}) (v interface{}, ok bool) {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	if pElement, ok := lru.cacheMap[k]; ok {
		lru.dlist.MoveToFront(pElement)
		return pElement.Value.(*CacheNode).Value, true
	}
	return nil, false
}

// GetNext get next item
func (lru *LRUCache) GetNext(preKey interface{}) (k, v interface{}, ok bool) {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	if pElement, ok := lru.cacheMap[k]; ok {
		pElement = pElement.Next()
		if pElement != nil {
			node := pElement.Value.(*CacheNode)
			return node.Key, node.Value, true
		}
	}
	return nil, nil, false
}
