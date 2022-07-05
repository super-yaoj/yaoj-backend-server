package libs

import (
	"sync"
	"time"
)

type mapEntry struct {
	last_visit time.Time
	data       any
}

type cacheMap struct {
	mutex 			sync.RWMutex
	mp 				map[string]mapEntry
	cacheExpireTime time.Duration
	maxCheckLength 	int
}

type MemoryCache interface {
	Set(string, any)
	Get(string) (any, bool)
	Delete(string)
}

func NewMemoryCache(expire time.Duration, check_length int) *cacheMap {
	return &cacheMap{ sync.RWMutex{}, make(map[string]mapEntry), expire, check_length }
}

func (cm *cacheMap) Set(key string, val any) {
	cm.mutex.Lock()
	cm.mp[key] = mapEntry{ time.Now(), val }
	if len(cm.mp) >= cm.maxCheckLength {
		current := time.Now().Add(-cm.cacheExpireTime)
		for i := range cm.mp {
			if cm.mp[i].last_visit.Before(current) {
				delete(cm.mp, i)
			}
		}
	}
	cm.mutex.Unlock()
}

func (cm *cacheMap) Get(key string) (any, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	a := cm.mp[key]
	if a.data == nil { return nil, false }
	return a.data, true
}

func (cm *cacheMap) Delete(key string) {
	cm.mutex.Lock()
	delete(cm.mp, key)
	cm.mutex.Unlock()
}