// Package priorityqueue provides a concurrent, bounded priority queue.
//
// Elements are dequeued by priority (highest first); elements with equal
// priority are dequeued in the FIFO order of their original successful
// Push. The queue is safe for concurrent use by multiple goroutines.
package priorityqueue

import (
	"container/heap"
	"errors"
	"sort"
	"sync"
)

// Sentinel errors returned by Queue operations.
var (
	// ErrInvalidCapacity is returned by New when capacity is not positive.
	ErrInvalidCapacity = errors.New("priorityqueue: capacity must be positive")
	// ErrEmptyID is returned by Push when the element ID is empty.
	ErrEmptyID = errors.New("priorityqueue: element ID must not be empty")
	// ErrDuplicateID is returned by Push when the ID already exists.
	ErrDuplicateID = errors.New("priorityqueue: element ID already exists")
	// ErrQueueFull is returned by Push when the queue is at capacity.
	ErrQueueFull = errors.New("priorityqueue: queue is full")
)

// Item is the user-facing view of a queue element. It is a plain value
// type, so every Item handed out by the Queue is an independent copy.
type Item struct {
	ID       string
	Priority int
	Value    string
}

// Queue is a concurrent bounded priority queue. The zero value is not
// usable; construct one with New.
type Queue struct {
	mu       sync.Mutex
	capacity int
	seq      uint64
	h        entryHeap
	index    map[string]*entry
}

// New creates a Queue that holds at most capacity elements.
// It returns ErrInvalidCapacity if capacity <= 0.
func New(capacity int) (*Queue, error) {
	if capacity <= 0 {
		return nil, ErrInvalidCapacity
	}
	return &Queue{
		capacity: capacity,
		index:    make(map[string]*entry),
	}, nil
}

// Push inserts a new element.
//
// It returns ErrEmptyID if id is empty, ErrDuplicateID if id already
// exists, and ErrQueueFull if the queue is at capacity (only the new
// element is rejected; existing elements are untouched).
func (q *Queue) Push(id string, priority int, value string) error {
	if id == "" {
		return ErrEmptyID
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if _, ok := q.index[id]; ok {
		return ErrDuplicateID
	}
	if len(q.h) >= q.capacity {
		return ErrQueueFull
	}
	e := &entry{
		item: Item{ID: id, Priority: priority, Value: value},
		seq:  q.seq,
	}
	q.seq++
	heap.Push(&q.h, e)
	q.index[id] = e
	return nil
}

// Get returns a copy of the element with the given ID.
// The second return value is false if no such element exists.
func (q *Queue) Get(id string) (Item, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	e, ok := q.index[id]
	if !ok {
		return Item{}, false
	}
	return e.item, true
}

// UpdatePriority changes the priority of an existing element.
// It does not affect FIFO order among equal-priority elements.
// It returns false if no element with the given ID exists.
func (q *Queue) UpdatePriority(id string, priority int) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	e, ok := q.index[id]
	if !ok {
		return false
	}
	e.item.Priority = priority
	heap.Fix(&q.h, e.index)
	return true
}

// Remove deletes the element with the given ID.
// It returns false if no such element exists.
func (q *Queue) Remove(id string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	e, ok := q.index[id]
	if !ok {
		return false
	}
	heap.Remove(&q.h, e.index)
	delete(q.index, id)
	return true
}

// Peek returns a copy of the highest-priority element without removing it.
// The second return value is false if the queue is empty.
func (q *Queue) Peek() (Item, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.h) == 0 {
		return Item{}, false
	}
	return q.h[0].item, true
}

// Pop removes and returns the highest-priority element.
// The second return value is false if the queue is empty.
func (q *Queue) Pop() (Item, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.h) == 0 {
		return Item{}, false
	}
	e := heap.Pop(&q.h).(*entry)
	delete(q.index, e.item.ID)
	return e.item, true
}

// Snapshot returns copies of all current elements in dequeue order
// (priority descending, ties broken by original Push order). The returned
// slice and its Items are independent of the queue's internal state.
func (q *Queue) Snapshot() []Item {
	q.mu.Lock()
	defer q.mu.Unlock()
	// Heap order is only partial, so sort the copy into full dequeue order.
	entries := make([]*entry, len(q.h))
	copy(entries, q.h)
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].item.Priority != entries[j].item.Priority {
			return entries[i].item.Priority > entries[j].item.Priority
		}
		return entries[i].seq < entries[j].seq
	})
	items := make([]Item, len(entries))
	for i, e := range entries {
		items[i] = e.item
	}
	return items
}

// Len returns the current number of elements.
func (q *Queue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.h)
}

// Capacity returns the maximum number of elements the queue can hold.
func (q *Queue) Capacity() int {
	return q.capacity
}
