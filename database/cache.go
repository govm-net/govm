package database

import (
	"container/list"
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
	return lru.dlist.Len()
}

// Set add new item
func (lru *LRUCache) Set(k, v interface{}) {
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
	if pElement, ok := lru.cacheMap[k]; ok {
		lru.dlist.MoveToFront(pElement)
		return pElement.Value.(*CacheNode).Value, true
	}
	return nil, false
}
