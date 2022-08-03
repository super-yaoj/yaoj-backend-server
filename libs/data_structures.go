package libs

import (
	"sync"
	"time"
)

//-------memory cached map with a expiring time

type mapEntry[T any] struct {
	last_visit time.Time
	data       T
}

type MemoryCache[T any] interface {
	Set(int, T)
	Get(int) (T, bool)
	Delete(int)
}

type cacheMap[T any] struct {
	mutex           sync.RWMutex
	mp              map[int]*mapEntry[T]
	cacheExpireTime time.Duration
	maxCheckLength  int
}

func NewMemoryCache[T any](expire time.Duration, check_length int) MemoryCache[T] {
	return &cacheMap[T]{sync.RWMutex{}, make(map[int]*mapEntry[T]), expire, check_length}
}

func (cm *cacheMap[T]) Set(key int, val T) {
	cm.mutex.Lock()
	cm.mp[key] = &mapEntry[T]{time.Now(), val}
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

func (cm *cacheMap[T]) Get(key int) (T, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	a, ok := cm.mp[key]
	if !ok {
		return *new(T), false
	}
	a.last_visit = time.Now()
	return a.data, true
}

func (cm *cacheMap[T]) Delete(key int) {
	cm.mutex.Lock()
	delete(cm.mp, key)
	cm.mutex.Unlock()
}

//-------Blocked priority queue

type queueEntry[T any] struct {
	val      T
	priority int
}

type PriorityQueue[T any] interface {
	Size() int
	Push(T, int)
	Top() T
	Pop() T
}

type binaryHeap[T any] []queueEntry[T]

func NewBinaryHeap[T any]() PriorityQueue[T] {
	return &binaryHeap[T]{}
}

func (bh *binaryHeap[T]) Size() int {
	return len(*bh)
}

func (bh *binaryHeap[T]) Push(x T, p int) {
	*bh = append(*bh, queueEntry[T]{})
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
	(*bh)[n] = queueEntry[T]{x, p}
}

func (bh *binaryHeap[T]) Pop() T {
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

func (bh *binaryHeap[T]) Top() T {
	return (*bh)[0].val
}

type blockPriorityQueue[T any] struct {
	queue PriorityQueue[T]
	cond  *sync.Cond
}

func NewBlockPriorityQueue[T any]() PriorityQueue[T] {
	return &blockPriorityQueue[T]{&binaryHeap[T]{}, sync.NewCond(&sync.Mutex{})}
}

func (pq *blockPriorityQueue[T]) Size() int {
	return pq.queue.Size()
}

func (pq *blockPriorityQueue[T]) Push(x T, p int) {
	pq.cond.L.Lock()
	pq.queue.Push(x, p)
	pq.cond.Signal()
	pq.cond.L.Unlock()
}

func (pq *blockPriorityQueue[T]) Top() T {
	pq.cond.L.Lock()
	for pq.queue.Size() == 0 {
		pq.cond.Wait()
	}
	pq.cond.Signal()
	defer pq.cond.L.Unlock()
	return pq.queue.Top()
}

func (pq *blockPriorityQueue[T]) Pop() T {
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

//-------Async Promise
type SyncPromise[Resolve any, Reject any] struct {
	res      Resolve           // last resolve
	rej      Reject            // last rejected
	IsReject func(Reject) bool // judge whether T is rejected data
}

func NewSyncPromise[Res any, Rej any](isrej func(Rej) bool, callback func() (Res, Rej)) *SyncPromise[Res, Rej] {
	res, rej := callback()
	return &SyncPromise[Res, Rej]{res, rej, isrej}
}
//give a callback function that accepts data and returns next data
//data should be able to judge whether it's rejected
func (p *SyncPromise[Res, Rej]) Then(callback func(Res) (Res, Rej)) *SyncPromise[Res, Rej] {
	if p.IsReject(p.rej) {
		return p
	}
	p.res, p.rej = callback(p.res)
	return p
}

func (p *SyncPromise[Res, Rej]) Catch(callback func(Rej)) {
	if p.IsReject(p.rej) {
		callback(p.rej)
	}
}

//error promise
type ErrorPromise struct {
	*SyncPromise[struct{}, error]
}

func NewErrorPromise(callback func() error) *ErrorPromise {
	return &ErrorPromise{
		NewSyncPromise(
			func(err error) bool { return err != nil },
			func() (struct{}, error) {
				return struct{}{}, callback()
			},
		),
	}
}

func (p *ErrorPromise) Then(callback func() error) *ErrorPromise {
	p.SyncPromise.Then(func(struct{}) (struct{}, error) {
		return struct{}{}, callback()
	})
	return p
}

func (p *ErrorPromise) Catch(callback func(error)) {
	p.SyncPromise.Catch(callback)
}