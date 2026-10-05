package priorityqueue

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

// TestConcurrentMixedOps hammers the queue with concurrent readers and
// writers while continuously checking the capacity and uniqueness
// invariants. Run with -race to validate memory safety.
func TestConcurrentMixedOps(t *testing.T) {
	const capacity = 64
	q := mustNew(t, capacity)
	var wg sync.WaitGroup

	// Writers: push unique IDs, update, remove, pop.
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 2000; i++ {
				id := fmt.Sprintf("w%d-%d", g, i%200)
				_ = q.Push(id, (g+i)%13, "v")
				_ = q.UpdatePriority(id, (i*7)%29)
				if i%3 == 0 {
					_ = q.Remove(id)
				}
				if i%5 == 0 {
					_, _ = q.Pop()
				}
			}
		}(g)
	}

	// Readers: verify invariants hold at every observed instant.
	for r := 0; r < 4; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 2000; i++ {
				if n := q.Len(); n < 0 || n > capacity {
					t.Errorf("Len()=%d violates capacity %d", n, capacity)
					return
				}
				snap := q.Snapshot()
				if len(snap) > capacity {
					t.Errorf("Snapshot len=%d exceeds capacity %d", len(snap), capacity)
					return
				}
				seen := make(map[string]struct{}, len(snap))
				for j, it := range snap {
					if it.ID == "" {
						t.Error("Snapshot contains empty ID")
						return
					}
					if _, dup := seen[it.ID]; dup {
						t.Errorf("Snapshot contains duplicate ID %q", it.ID)
						return
					}
					seen[it.ID] = struct{}{}
					if j > 0 && snap[j-1].Priority < it.Priority {
						t.Errorf("Snapshot not ordered: %d before %d",
							snap[j-1].Priority, it.Priority)
						return
					}
				}
				_, _ = q.Peek()
				_, _ = q.Get("w0-0")
			}
		}()
	}
	wg.Wait()
}

// TestConcurrentPushUniquenessAndCapacity floods a small queue from many
// goroutines using a shared ID space, forcing duplicate-ID and
// queue-full contention. Afterwards exactly "capacity" unique elements
// must remain, and the number of successful pushes must match Len.
func TestConcurrentPushUniquenessAndCapacity(t *testing.T) {
	const (
		capacity   = 128
		goroutines = 16
		perG       = 100
		idSpace    = 300 // > capacity so the queue definitely fills up
	)
	q := mustNew(t, capacity)
	var succeeded atomic.Int64
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < perG; i++ {
				id := fmt.Sprintf("id-%d", (g*perG+i)%idSpace)
				err := q.Push(id, i, "v")
				switch {
				case err == nil:
					succeeded.Add(1)
				case errors.Is(err, ErrDuplicateID), errors.Is(err, ErrQueueFull):
					// expected rejections
				default:
					t.Errorf("Push returned unexpected error: %v", err)
				}
			}
		}(g)
	}
	wg.Wait()

	if got := q.Len(); got != capacity {
		t.Fatalf("Len()=%d, want exactly capacity %d", got, capacity)
	}
	if got := succeeded.Load(); got != int64(capacity) {
		t.Fatalf("successful pushes=%d, want %d (one per final element)", got, capacity)
	}
	snap := q.Snapshot()
	seen := make(map[string]struct{}, len(snap))
	for _, it := range snap {
		if _, dup := seen[it.ID]; dup {
			t.Fatalf("duplicate ID %q survived concurrent pushes", it.ID)
		}
		seen[it.ID] = struct{}{}
	}
}

// TestConcurrentDrain pops a full queue from many goroutines; every
// element must be dequeued exactly once with no duplicates or losses.
func TestConcurrentDrain(t *testing.T) {
	const capacity = 500
	q := mustNew(t, capacity)
	for i := 0; i < capacity; i++ {
		mustPush(t, q, fmt.Sprintf("id-%04d", i), i%17, "v")
	}

	var mu sync.Mutex
	popped := make(map[string]int) // id -> pop count
	var wg sync.WaitGroup
	for g := 0; g < 10; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				it, ok := q.Pop()
				if !ok {
					return
				}
				mu.Lock()
				popped[it.ID]++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if len(popped) != capacity {
		t.Fatalf("popped %d distinct IDs, want %d", len(popped), capacity)
	}
	for id, n := range popped {
		if n != 1 {
			t.Errorf("ID %q popped %d times, want exactly 1", id, n)
		}
	}
	if q.Len() != 0 {
		t.Fatalf("Len()=%d after drain, want 0", q.Len())
	}
}

// TestConcurrentSnapshotConsistency ensures Snapshot always reflects a
// consistent point-in-time state while updates stream in.
func TestConcurrentSnapshotConsistency(t *testing.T) {
	const capacity = 32
	q := mustNew(t, capacity)
	var wg sync.WaitGroup
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				id := fmt.Sprintf("s%d-%d", g, i%40)
				_ = q.Push(id, i%7, "v")
				_ = q.UpdatePriority(id, (i+1)%7)
				if i%2 == 0 {
					_ = q.Remove(id)
				}
			}
		}(g)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 3000; i++ {
			snap := q.Snapshot()
			for j := 1; j < len(snap); j++ {
				if snap[j-1].Priority < snap[j].Priority {
					t.Errorf("inconsistent snapshot: %d before %d",
						snap[j-1].Priority, snap[j].Priority)
					return
				}
			}
		}
	}()
	wg.Wait()
}
