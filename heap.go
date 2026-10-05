package priorityqueue

// entry is the internal heap node. It wraps the user-facing Item with
// bookkeeping fields that must never leak to callers.
type entry struct {
	item Item
	// seq is a monotonically increasing sequence number assigned at Push
	// time. It provides FIFO ordering among items with equal priority and
	// is intentionally NOT changed by UpdatePriority.
	seq uint64
	// index is the entry's current position in the heap slice, maintained
	// by the container/heap callbacks so Remove/Fix stay O(log n).
	index int
}

// entryHeap implements heap.Interface as a max-heap ordered by
// (priority desc, seq asc): the next element to dequeue is always at [0].
type entryHeap []*entry

func (h entryHeap) Len() int { return len(h) }

func (h entryHeap) Less(i, j int) bool {
	if h[i].item.Priority != h[j].item.Priority {
		return h[i].item.Priority > h[j].item.Priority
	}
	return h[i].seq < h[j].seq
}

func (h entryHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *entryHeap) Push(x any) {
	e := x.(*entry)
	e.index = len(*h)
	*h = append(*h, e)
}

func (h *entryHeap) Pop() any {
	old := *h
	n := len(old)
	e := old[n-1]
	old[n-1] = nil // avoid memory leak
	e.index = -1
	*h = old[:n-1]
	return e
}
