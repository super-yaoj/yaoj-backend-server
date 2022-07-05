package libs

import (
	"sync"
)

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
