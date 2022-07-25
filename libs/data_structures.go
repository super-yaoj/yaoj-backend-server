package libs

import (
	"sync"
	"time"
)

//-------memory cached map with a expiring time

type mapEntry struct {
	last_visit time.Time
	data       any
}

type cacheMap struct {
	mutex           sync.RWMutex
	mp              map[int]*mapEntry
	cacheExpireTime time.Duration
	maxCheckLength  int
}

type MemoryCache interface {
	Set(int, any)
	Get(int) (any, bool)
	Delete(int)
}

func NewMemoryCache(expire time.Duration, check_length int) MemoryCache {
	return &cacheMap{sync.RWMutex{}, make(map[int]*mapEntry), expire, check_length}
}

func (cm *cacheMap) Set(key int, val any) {
	cm.mutex.Lock()
	cm.mp[key] = &mapEntry{time.Now(), val}
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

func (cm *cacheMap) Get(key int) (any, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	a, ok := cm.mp[key]
	if !ok {
		return nil, false
	}
	a.last_visit = time.Now()
	return a.data, true
}

func (cm *cacheMap) Delete(key int) {
	cm.mutex.Lock()
	delete(cm.mp, key)
	cm.mutex.Unlock()
}

//-------Blocked priority queue

type queueEntry struct {
	val      any
	priority int
}

type PriorityQueue interface {
	Size() int
	Push(any, int)
	Top() any
	Pop() any
}

type binaryHeap []queueEntry

func NewBinaryHeap() *binaryHeap {
	return &binaryHeap{}
}

func (bh *binaryHeap) Size() int {
	return len(*bh)
}

func (bh *binaryHeap) Push(x any, p int) {
	*bh = append(*bh, queueEntry{})
	n := len(*bh) - 1
	for n > 0 {
		f := (n - 1) / 2
		if (*bh)[f].priority < p {
			(*bh)[n] = (*bh)[f]
		} else {
			break
		}
		n = f
	}
	(*bh)[n] = queueEntry{x, p}
}

func (bh *binaryHeap) Pop() any {
	top := (*bh)[0]
	n := len(*bh) - 1
	(*bh)[0], (*bh)[n] = (*bh)[n], (*bh)[0]
	(*bh) = (*bh)[:n]
	if n > 0 {
		temp := (*bh)[0]
		i := 0
		for {
			l := i*2 + 1
			r := l + 1
			if l >= n {
				break
			}
			if r >= n {
				if (*bh)[l].priority > temp.priority {
					(*bh)[i] = (*bh)[l]
					i = l
				} else {
					break
				}
			} else if (*bh)[l].priority < (*bh)[r].priority {
				if (*bh)[r].priority > temp.priority {
					(*bh)[i] = (*bh)[r]
					i = r
				} else {
					break
				}
			} else {
				if (*bh)[l].priority > temp.priority {
					(*bh)[i] = (*bh)[l]
					i = l
				} else {
					break
				}
			}
		}
		(*bh)[i] = temp
	}
	return top.val
}

func (bh binaryHeap) Top() any {
	return bh[0].val
}

type blockPriorityQueue struct {
	queue PriorityQueue
	cond  *sync.Cond
}

func NewBlockPriorityQueue() *blockPriorityQueue {
	ret := &blockPriorityQueue{&binaryHeap{}, sync.NewCond(&sync.Mutex{})}
	return ret
}

func (pq *blockPriorityQueue) Size() int {
	return pq.queue.Size()
}

func (pq *blockPriorityQueue) Push(x any, p int) {
	pq.cond.L.Lock()
	pq.queue.Push(x, p)
	pq.cond.Signal()
	pq.cond.L.Unlock()
}

func (pq *blockPriorityQueue) Top() any {
	pq.cond.L.Lock()
	for pq.queue.Size() == 0 {
		pq.cond.Wait()
	}
	pq.cond.Signal()
	defer pq.cond.L.Unlock()
	return pq.queue.Top()
}

func (pq *blockPriorityQueue) Pop() any {
	pq.cond.L.Lock()
	for pq.queue.Size() == 0 {
		pq.cond.Wait()
	}
	defer pq.cond.L.Unlock()
	return pq.queue.Pop()
}


//---------Multi RWlock

type MultiRWLock interface {
	Lock(int)
	Unlock(int)
	RLock(int)
	RUnlock(int)
}

type mappedMultiRWLock struct {
	lockmap sync.Map
}

func (mml *mappedMultiRWLock) getMutex(id int) *sync.RWMutex {
	lock, ok := mml.lockmap.Load(id)
	if !ok {
		lock = new(sync.RWMutex)
		mml.lockmap.Store(id, lock)
	}
	return lock.(*sync.RWMutex)
}

func (mml *mappedMultiRWLock) Lock(id int) {
	mml.getMutex(id).Lock()
}

func (mml *mappedMultiRWLock) Unlock(id int) {
	mml.getMutex(id).Unlock()
}

func (mml *mappedMultiRWLock) RLock(id int) {
	mml.getMutex(id).RLock()
}

func (mml *mappedMultiRWLock) RUnlock(id int) {
	mml.getMutex(id).RUnlock()
}

func NewMappedMultiRWMutex() MultiRWLock {
	return new(mappedMultiRWLock)
}